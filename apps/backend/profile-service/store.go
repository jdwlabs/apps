package main

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Store is the profile aggregate's storage. Every method reports its outcome
// through the sentinels in errors.go, so a handler maps an outcome to a status
// without knowing any SQL.
type Store interface {
	ListProfiles(ctx context.Context, limit, offset int) ([]Profile, error)
	ProfileByID(ctx context.Context, profileID int64) (Profile, error)
	ProfileByUserID(ctx context.Context, userID int64) (Profile, error)
	ProfileIDForUser(ctx context.Context, userID int64) (int64, bool, error)
	CreateProfile(ctx context.Context, request ProfileCreateRequest, actorUserID int64) (Profile, error)
	UpdateProfileByID(ctx context.Context, profileID int64, request ProfileUpdateRequest, actorUserID int64) (Profile, error)
	UpdateProfileByUserID(ctx context.Context, userID int64, request ProfileUpdateRequest, actorUserID int64) (Profile, error)
	DeleteProfileByID(ctx context.Context, profileID int64) error
	DeleteProfileByUserID(ctx context.Context, userID int64) error
	AddAddress(ctx context.Context, profileID int64, request AddressRequest, actorUserID int64) (Profile, error)
	UpdateAddress(ctx context.Context, profileID, addressID int64, request AddressRequest, actorUserID int64) (Profile, error)
	DeleteAddress(ctx context.Context, profileID, addressID int64) error
	Icon(ctx context.Context, profileID int64) (ProfileIcon, error)
	AddIcon(ctx context.Context, profileID int64, icon []byte, actorUserID int64) (Profile, error)
	ReplaceIcon(ctx context.Context, profileID int64, icon []byte, actorUserID int64) (Profile, error)
	DeleteIcon(ctx context.Context, profileID int64) error
}

// foreignKeyViolation is Postgres' SQLSTATE for a failed foreign key, which is
// how a create for a user that does not exist reports itself.
const foreignKeyViolation = "23503"

// PostgresStore reads and writes auth.profiles, auth.addresses and
// auth.profile_icons — the three tables the profile half of the split owns.
type PostgresStore struct {
	pool *pgxpool.Pool
}

func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{pool: pool}
}

// querier is the subset of pgx both a pool and a transaction offer, so a read
// can run inside a write's transaction or on its own.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

const profileColumns = `profile_id, user_id, first_name, middle_name, last_name, birthdate,
	created_by_user_id, created_time, modified_by_user_id, modified_time`

const addressColumns = `address_id, profile_id, address_line_1, address_line_2, city, state_province,
	postal_code, country, created_by_user_id, created_time, modified_by_user_id, modified_time`

const iconColumns = `icon_id, profile_id, icon, created_by_user_id, created_time,
	modified_by_user_id, modified_time`

func (s *PostgresStore) ListProfiles(ctx context.Context, limit, offset int) ([]Profile, error) {
	// ORDER BY makes LIMIT/OFFSET deterministic across pages; Postgres gives no
	// ordering guarantee otherwise.
	rows, err := s.pool.Query(ctx,
		`SELECT `+profileColumns+` FROM auth.profiles ORDER BY profile_id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list profiles: %w", err)
	}
	profiles, err := scanProfiles(rows)
	if err != nil {
		return nil, err
	}
	if err := attachSubresources(ctx, s.pool, profiles); err != nil {
		return nil, err
	}
	return profiles, nil
}

func (s *PostgresStore) ProfileByID(ctx context.Context, profileID int64) (Profile, error) {
	return loadOne(ctx, s.pool, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
}

func (s *PostgresStore) ProfileByUserID(ctx context.Context, userID int64) (Profile, error) {
	return loadOne(ctx, s.pool, `SELECT `+profileColumns+` FROM auth.profiles WHERE user_id = $1`, userID)
}

// ProfileIDForUser backs the fallback for a token minted before its owner had a
// profile. It keys on the user_id claim alone, never on anything the request
// carries, so falling back cannot widen a caller's own authorization.
func (s *PostgresStore) ProfileIDForUser(ctx context.Context, userID int64) (int64, bool, error) {
	var profileID int64
	err := s.pool.QueryRow(ctx, `SELECT profile_id FROM auth.profiles WHERE user_id = $1`, userID).Scan(&profileID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("resolve profile for user %d: %w", userID, err)
	}
	return profileID, true, nil
}

func (s *PostgresStore) CreateProfile(
	ctx context.Context, request ProfileCreateRequest, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		// One profile per user is an application rule: auth.profiles carries a
		// plain index on user_id, not a unique one, so the database will not
		// refuse the second row.
		exists, err := rowExists(ctx, tx, `SELECT 1 FROM auth.profiles WHERE user_id = $1`, *request.UserID)
		if err != nil {
			return Profile{}, err
		}
		if exists {
			return Profile{}, ErrProfileExists
		}

		now := time.Now().UTC()
		var profileID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO auth.profiles (user_id, first_name, middle_name, last_name, birthdate,
				created_by_user_id, created_time, modified_by_user_id, modified_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $6, $7)
			RETURNING profile_id`,
			*request.UserID, *request.FirstName, request.MiddleName, *request.LastName,
			request.Birthdate.Time, actorUserID, now).Scan(&profileID)
		if isForeignKeyViolation(err) {
			// The existence check the JVM makes with a read into auth.users,
			// made by the database instead: stronger, and free on the request
			// path. The caller still sees a missing user rather than a 500.
			return Profile{}, ErrUserNotFound
		}
		if err != nil {
			return Profile{}, fmt.Errorf("create profile: %w", err)
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

func (s *PostgresStore) UpdateProfileByID(
	ctx context.Context, profileID int64, request ProfileUpdateRequest, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		tag, err := tx.Exec(ctx, `
			UPDATE auth.profiles
			SET first_name = $1, middle_name = $2, last_name = $3, birthdate = $4,
				modified_by_user_id = $5, modified_time = $6
			WHERE profile_id = $7`,
			*request.FirstName, request.MiddleName, *request.LastName, request.Birthdate.Time,
			actorUserID, time.Now().UTC(), profileID)
		if err != nil {
			return Profile{}, fmt.Errorf("update profile %d: %w", profileID, err)
		}
		if tag.RowsAffected() == 0 {
			return Profile{}, ErrProfileNotFound
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

func (s *PostgresStore) UpdateProfileByUserID(
	ctx context.Context, userID int64, request ProfileUpdateRequest, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		var profileID int64
		err := tx.QueryRow(ctx, `
			UPDATE auth.profiles
			SET first_name = $1, middle_name = $2, last_name = $3, birthdate = $4,
				modified_by_user_id = $5, modified_time = $6
			WHERE user_id = $7
			RETURNING profile_id`,
			*request.FirstName, request.MiddleName, *request.LastName, request.Birthdate.Time,
			actorUserID, time.Now().UTC(), userID).Scan(&profileID)
		if errors.Is(err, pgx.ErrNoRows) {
			return Profile{}, ErrProfileNotFound
		}
		if err != nil {
			return Profile{}, fmt.Errorf("update profile for user %d: %w", userID, err)
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

// DeleteProfileByID clears the subresources before the profile itself.
// auth.addresses and auth.profile_icons reference auth.profiles with no cascade,
// so the order is what makes the delete possible at all.
func (s *PostgresStore) DeleteProfileByID(ctx context.Context, profileID int64) error {
	_, err := s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		return Profile{}, deleteProfileTree(ctx, tx, profileID,
			`DELETE FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
	return err
}

func (s *PostgresStore) DeleteProfileByUserID(ctx context.Context, userID int64) error {
	_, err := s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		var profileID int64
		err := tx.QueryRow(ctx, `SELECT profile_id FROM auth.profiles WHERE user_id = $1`, userID).Scan(&profileID)
		if errors.Is(err, pgx.ErrNoRows) {
			// A no-op for a user with no profile, and still a success: the
			// operation has no 404 in its response set.
			return Profile{}, nil
		}
		if err != nil {
			return Profile{}, fmt.Errorf("find profile for user %d: %w", userID, err)
		}
		return Profile{}, deleteProfileTree(ctx, tx, profileID,
			`DELETE FROM auth.profiles WHERE user_id = $1`, userID)
	})
	return err
}

func deleteProfileTree(ctx context.Context, tx pgx.Tx, profileID int64, profileDelete string, key int64) error {
	for _, statement := range []string{
		`DELETE FROM auth.addresses WHERE profile_id = $1`,
		`DELETE FROM auth.profile_icons WHERE profile_id = $1`,
	} {
		if _, err := tx.Exec(ctx, statement, profileID); err != nil {
			return fmt.Errorf("delete subresources of profile %d: %w", profileID, err)
		}
	}
	if _, err := tx.Exec(ctx, profileDelete, key); err != nil {
		return fmt.Errorf("delete profile %d: %w", profileID, err)
	}
	return nil
}

func (s *PostgresStore) AddAddress(
	ctx context.Context, profileID int64, request AddressRequest, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		if err := requireProfile(ctx, tx, profileID); err != nil {
			return Profile{}, err
		}
		now := time.Now().UTC()
		_, err := tx.Exec(ctx, `
			INSERT INTO auth.addresses (profile_id, address_line_1, address_line_2, city, state_province,
				postal_code, country, created_by_user_id, created_time, modified_by_user_id, modified_time)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $8, $9)`,
			profileID, *request.AddressLine1, request.AddressLine2, *request.City, *request.StateProvince,
			*request.PostalCode, *request.Country, actorUserID, now)
		if err != nil {
			return Profile{}, fmt.Errorf("add address to profile %d: %w", profileID, err)
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

// UpdateAddress is scoped to the profile in the path, so an address id
// belonging to another profile is a miss rather than an edit.
func (s *PostgresStore) UpdateAddress(
	ctx context.Context, profileID, addressID int64, request AddressRequest, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		if err := requireProfile(ctx, tx, profileID); err != nil {
			return Profile{}, err
		}
		tag, err := tx.Exec(ctx, `
			UPDATE auth.addresses
			SET address_line_1 = $1, address_line_2 = $2, city = $3, state_province = $4,
				postal_code = $5, country = $6, modified_by_user_id = $7, modified_time = $8
			WHERE address_id = $9 AND profile_id = $10`,
			*request.AddressLine1, request.AddressLine2, *request.City, *request.StateProvince,
			*request.PostalCode, *request.Country, actorUserID, time.Now().UTC(), addressID, profileID)
		if err != nil {
			return Profile{}, fmt.Errorf("update address %d: %w", addressID, err)
		}
		if tag.RowsAffected() == 0 {
			return Profile{}, ErrAddressNotFound
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

func (s *PostgresStore) DeleteAddress(ctx context.Context, profileID, addressID int64) error {
	tag, err := s.pool.Exec(ctx,
		`DELETE FROM auth.addresses WHERE address_id = $1 AND profile_id = $2`, addressID, profileID)
	if err != nil {
		return fmt.Errorf("delete address %d: %w", addressID, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrAddressNotFound
	}
	return nil
}

func (s *PostgresStore) Icon(ctx context.Context, profileID int64) (ProfileIcon, error) {
	icon, err := loadIcon(ctx, s.pool, profileID)
	if err != nil {
		return ProfileIcon{}, err
	}
	if icon == nil {
		return ProfileIcon{}, ErrIconNotFound
	}
	return *icon, nil
}

func (s *PostgresStore) AddIcon(
	ctx context.Context, profileID int64, icon []byte, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		if err := requireProfile(ctx, tx, profileID); err != nil {
			return Profile{}, err
		}
		// "At most one icon per profile" is not a database constraint:
		// profile_icons_profile_id_idx is a plain index. Until it is unique the
		// check has to live here, as it does in the JVM.
		exists, err := rowExists(ctx, tx, `SELECT 1 FROM auth.profile_icons WHERE profile_id = $1`, profileID)
		if err != nil {
			return Profile{}, err
		}
		if exists {
			return Profile{}, ErrIconExists
		}
		now := time.Now().UTC()
		if _, err := tx.Exec(ctx, `
			INSERT INTO auth.profile_icons (profile_id, icon, created_by_user_id, created_time,
				modified_by_user_id, modified_time)
			VALUES ($1, $2, $3, $4, $3, $4)`, profileID, icon, actorUserID, now); err != nil {
			return Profile{}, fmt.Errorf("add icon to profile %d: %w", profileID, err)
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

func (s *PostgresStore) ReplaceIcon(
	ctx context.Context, profileID int64, icon []byte, actorUserID int64,
) (Profile, error) {
	return s.inTransaction(ctx, func(tx pgx.Tx) (Profile, error) {
		if err := requireProfile(ctx, tx, profileID); err != nil {
			return Profile{}, err
		}
		current, err := loadIcon(ctx, tx, profileID)
		if err != nil {
			return Profile{}, err
		}
		if current == nil {
			return Profile{}, ErrNoIconToReplace
		}
		// Filtered on icon_id, as the JVM filters, so a profile that somehow
		// carries two rows has exactly one of them replaced rather than both.
		//
		// modified_by_user_id takes the acting user. The JVM writes the icon's
		// original creator here instead, passing createdByUserId where the
		// service supplied the actor; that stamps a replacement with the wrong
		// author, and reproducing it would mean writing a knowingly false audit
		// record into a column this service's own response exposes.
		if _, err := tx.Exec(ctx, `
			UPDATE auth.profile_icons
			SET icon = $1, modified_by_user_id = $2, modified_time = $3
			WHERE icon_id = $4`, icon, actorUserID, time.Now().UTC(), current.ID); err != nil {
			return Profile{}, fmt.Errorf("replace icon of profile %d: %w", profileID, err)
		}
		return loadOne(ctx, tx, `SELECT `+profileColumns+` FROM auth.profiles WHERE profile_id = $1`, profileID)
	})
}

// DeleteIcon is keyed on profile_id: no route accepts an icon id, and a profile
// carries at most one icon.
func (s *PostgresStore) DeleteIcon(ctx context.Context, profileID int64) error {
	if _, err := s.pool.Exec(ctx, `DELETE FROM auth.profile_icons WHERE profile_id = $1`, profileID); err != nil {
		return fmt.Errorf("delete icon of profile %d: %w", profileID, err)
	}
	return nil
}

// inTransaction runs body in a transaction, rolling back on any error. The
// rollback after a commit is a no-op, which is why it can be deferred
// unconditionally.
func (s *PostgresStore) inTransaction(ctx context.Context, body func(pgx.Tx) (Profile, error)) (Profile, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Profile{}, fmt.Errorf("begin: %w", err)
	}
	defer func() {
		if err := tx.Rollback(ctx); err != nil && !errors.Is(err, pgx.ErrTxClosed) {
			// Losing the rollback leaks a connection's transaction state, which
			// is worth a line in the log even though the caller already has the
			// failure that caused it.
			logRollbackFailure(err)
		}
	}()

	result, err := body(tx)
	if err != nil {
		return Profile{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Profile{}, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

func requireProfile(ctx context.Context, q querier, profileID int64) error {
	exists, err := rowExists(ctx, q, `SELECT 1 FROM auth.profiles WHERE profile_id = $1`, profileID)
	if err != nil {
		return err
	}
	if !exists {
		return ErrProfileNotFound
	}
	return nil
}

func rowExists(ctx context.Context, q querier, query string, argument int64) (bool, error) {
	var one int
	err := q.QueryRow(ctx, query, argument).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("existence check: %w", err)
	}
	return true, nil
}

func loadOne(ctx context.Context, q querier, query string, argument int64) (Profile, error) {
	rows, err := q.Query(ctx, query, argument)
	if err != nil {
		return Profile{}, fmt.Errorf("read profile: %w", err)
	}
	profiles, err := scanProfiles(rows)
	if err != nil {
		return Profile{}, err
	}
	if len(profiles) == 0 {
		return Profile{}, ErrProfileNotFound
	}
	if err := attachSubresources(ctx, q, profiles); err != nil {
		return Profile{}, err
	}
	return profiles[0], nil
}

func scanProfiles(rows pgx.Rows) ([]Profile, error) {
	defer rows.Close()
	profiles := []Profile{}
	for rows.Next() {
		var profile Profile
		var birthdate, createdTime, modifiedTime time.Time
		if err := rows.Scan(&profile.ID, &profile.UserID, &profile.FirstName, &profile.MiddleName,
			&profile.LastName, &birthdate, &profile.CreatedByUserID, &createdTime,
			&profile.ModifiedByUserID, &modifiedTime); err != nil {
			return nil, fmt.Errorf("scan profile: %w", err)
		}
		profile.Birthdate = Date{birthdate}
		profile.CreatedTime = Timestamp{createdTime}
		profile.ModifiedTime = Timestamp{modifiedTime}
		profile.Addresses = []Address{}
		profiles = append(profiles, profile)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read profiles: %w", err)
	}
	return profiles, nil
}

// attachSubresources fills in the addresses and icon of every profile in one
// query each, whatever the page size. ProfileRepositoryImpl issues two queries
// per row instead, so a hundred-row page costs it two hundred round trips.
func attachSubresources(ctx context.Context, q querier, profiles []Profile) error {
	if len(profiles) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(profiles))
	at := make(map[int64]*Profile, len(profiles))
	for i := range profiles {
		ids = append(ids, profiles[i].ID)
		at[profiles[i].ID] = &profiles[i]
	}

	addresses, err := q.Query(ctx,
		`SELECT `+addressColumns+` FROM auth.addresses WHERE profile_id = ANY($1) ORDER BY address_id`, ids)
	if err != nil {
		return fmt.Errorf("read addresses: %w", err)
	}
	if err := func() error {
		defer addresses.Close()
		for addresses.Next() {
			address, err := scanAddress(addresses)
			if err != nil {
				return err
			}
			if profile, known := at[address.ProfileID]; known {
				profile.Addresses = append(profile.Addresses, address)
			}
		}
		return addresses.Err()
	}(); err != nil {
		return fmt.Errorf("read addresses: %w", err)
	}

	icons, err := q.Query(ctx, `SELECT `+iconColumns+` FROM auth.profile_icons WHERE profile_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("read icons: %w", err)
	}
	defer icons.Close()
	for icons.Next() {
		icon, err := scanIcon(icons)
		if err != nil {
			return err
		}
		if profile, known := at[icon.ProfileID]; known {
			stored := icon
			profile.Icon = &stored
		}
	}
	if err := icons.Err(); err != nil {
		return fmt.Errorf("read icons: %w", err)
	}
	return nil
}

func loadIcon(ctx context.Context, q querier, profileID int64) (*ProfileIcon, error) {
	rows, err := q.Query(ctx, `SELECT `+iconColumns+` FROM auth.profile_icons WHERE profile_id = $1`, profileID)
	if err != nil {
		return nil, fmt.Errorf("read icon: %w", err)
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return nil, fmt.Errorf("read icon: %w", err)
		}
		return nil, nil
	}
	icon, err := scanIcon(rows)
	if err != nil {
		return nil, err
	}
	return &icon, nil
}

func scanAddress(rows pgx.Rows) (Address, error) {
	var address Address
	var createdTime, modifiedTime time.Time
	if err := rows.Scan(&address.ID, &address.ProfileID, &address.AddressLine1, &address.AddressLine2,
		&address.City, &address.StateProvince, &address.PostalCode, &address.Country,
		&address.CreatedByUserID, &createdTime, &address.ModifiedByUserID, &modifiedTime); err != nil {
		return Address{}, fmt.Errorf("scan address: %w", err)
	}
	address.CreatedTime = Timestamp{createdTime}
	address.ModifiedTime = Timestamp{modifiedTime}
	return address, nil
}

func scanIcon(rows pgx.Rows) (ProfileIcon, error) {
	var icon ProfileIcon
	var createdTime, modifiedTime time.Time
	if err := rows.Scan(&icon.ID, &icon.ProfileID, &icon.Icon, &icon.CreatedByUserID, &createdTime,
		&icon.ModifiedByUserID, &modifiedTime); err != nil {
		return ProfileIcon{}, fmt.Errorf("scan icon: %w", err)
	}
	icon.CreatedTime = Timestamp{createdTime}
	icon.ModifiedTime = Timestamp{modifiedTime}
	return icon, nil
}

func isForeignKeyViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == foreignKeyViolation
}
