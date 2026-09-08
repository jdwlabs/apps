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

// Store is the identity half's storage: auth.users, auth.roles and the
// auth.users_roles table both sides write. Every method reports its outcome
// through the sentinels in errors.go, so a handler maps an outcome to a status
// without knowing any SQL.
type Store interface {
	ListUsers(ctx context.Context, limit, offset int) ([]User, error)
	UserByID(ctx context.Context, userID int64) (User, error)
	UserByEmailAddress(ctx context.Context, emailAddress string) (User, error)
	CredentialByEmailAddress(ctx context.Context, emailAddress string) (Credential, error)
	CreateUser(ctx context.Context, emailAddress, passwordHash string, actorUserID int64) (User, error)
	UpdateUser(ctx context.Context, userID int64, emailAddress, passwordHash string, actorUserID int64) (User, error)
	DeleteUser(ctx context.Context, userID int64) error
	UserHasAnyRole(ctx context.Context, userID int64, roleIDs []int64) (bool, error)
	GrantRolesToUser(ctx context.Context, userID int64, roleIDs []int64, actorUserID int64) (User, error)
	RevokeRolesFromUser(ctx context.Context, userID int64, roleIDs []int64, actorUserID int64) (User, error)

	ListRoles(ctx context.Context, limit, offset int) ([]Role, error)
	RoleByID(ctx context.Context, roleID int64) (Role, error)
	RoleByName(ctx context.Context, name string) (Role, error)
	CreateRole(ctx context.Context, name, description string, actorUserID int64) (Role, error)
	UpdateRole(ctx context.Context, roleID int64, name, description string, actorUserID int64) (Role, error)
	DeleteRole(ctx context.Context, roleID int64) error
	GrantUsersToRole(ctx context.Context, roleID int64, userIDs []int64, actorUserID int64) (Role, error)
	RevokeUsersFromRole(ctx context.Context, roleID int64, userIDs []int64, actorUserID int64) (Role, error)
}

// uniqueViolation is Postgres' SQLSTATE for a broken unique constraint, which
// is how an update onto an email address or role name another row holds reports
// itself. No handler in the JVM maps it, which is why it is a 500 there and
// here.
const uniqueViolation = "23505"

// PostgresStore reads and writes the three tables the identity half owns, and
// reads auth.profiles for the profile id the user payload and the token claim
// both carry.
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

const userColumns = `user_id, email_address, status, created_by_user_id, created_time,
	modified_by_user_id, modified_time`

const roleColumns = `role_id, role_name, role_description, status, created_by_user_id, created_time,
	modified_by_user_id, modified_time`

const grantColumns = `user_id, role_id, created_by_user_id, created_time`

func (s *PostgresStore) ListUsers(ctx context.Context, limit, offset int) ([]User, error) {
	// ORDER BY makes LIMIT/OFFSET deterministic across pages; Postgres gives no
	// ordering guarantee otherwise.
	rows, err := s.pool.Query(ctx,
		`SELECT `+userColumns+` FROM auth.users ORDER BY user_id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	users, err := scanUsers(rows)
	if err != nil {
		return nil, err
	}
	if err := attachUserSubresources(ctx, s.pool, users); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *PostgresStore) UserByID(ctx context.Context, userID int64) (User, error) {
	return loadUser(ctx, s.pool, `SELECT `+userColumns+` FROM auth.users WHERE user_id = $1`, userID)
}

func (s *PostgresStore) UserByEmailAddress(ctx context.Context, emailAddress string) (User, error) {
	return loadUser(ctx, s.pool,
		`SELECT `+userColumns+` FROM auth.users WHERE email_address = $1`, emailAddress)
}

// CredentialByEmailAddress reads the one row authentication needs. It is a
// separate method from UserByEmailAddress, returning a separate type, so the
// stored hash has exactly one call site and no path into a response.
func (s *PostgresStore) CredentialByEmailAddress(ctx context.Context, emailAddress string) (Credential, error) {
	var credential Credential
	err := s.pool.QueryRow(ctx,
		`SELECT user_id, email_address, password FROM auth.users WHERE email_address = $1`, emailAddress).
		Scan(&credential.UserID, &credential.EmailAddress, &credential.PasswordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credential{}, ErrUserNotFound
	}
	if err != nil {
		return Credential{}, fmt.Errorf("read credential: %w", err)
	}

	// The claims a token is minted from, resolved the way SecurityUser resolves
	// them: role names through the grant table, and the profile id from the
	// profile row if there is one.
	rows, err := s.pool.Query(ctx, `
		SELECT r.role_name FROM auth.users_roles ur
		JOIN auth.roles r ON r.role_id = ur.role_id
		WHERE ur.user_id = $1
		ORDER BY r.role_id`, credential.UserID)
	if err != nil {
		return Credential{}, fmt.Errorf("read role names: %w", err)
	}
	credential.Roles, err = scanStrings(rows)
	if err != nil {
		return Credential{}, err
	}

	var profileID int64
	err = s.pool.QueryRow(ctx, `SELECT profile_id FROM auth.profiles WHERE user_id = $1`, credential.UserID).
		Scan(&profileID)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		// A null claim rather than an absent user: profile-service falls back to
		// a user_id lookup when it sees one.
	case err != nil:
		return Credential{}, fmt.Errorf("read profile id: %w", err)
	default:
		credential.ProfileID = &profileID
	}
	return credential, nil
}

func (s *PostgresStore) CreateUser(
	ctx context.Context, emailAddress, passwordHash string, actorUserID int64,
) (User, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (User, error) {
		exists, err := rowExists(ctx, tx, `SELECT 1 FROM auth.users WHERE email_address = $1`, emailAddress)
		if err != nil {
			return User{}, err
		}
		if exists {
			return User{}, ErrUserExists
		}
		now := time.Now().UTC()
		var userID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO auth.users (email_address, password, status, created_by_user_id, created_time,
				modified_by_user_id, modified_time)
			VALUES ($1, $2, $3, $4, $5, $4, $5)
			RETURNING user_id`, emailAddress, passwordHash, statusActive, actorUserID, now).Scan(&userID)
		if isUniqueViolation(err) {
			// The pre-check above lost a race with a concurrent registration.
			// The database is the one that actually holds the rule.
			return User{}, ErrUserExists
		}
		if err != nil {
			return User{}, fmt.Errorf("create user: %w", err)
		}
		return loadUser(ctx, tx, `SELECT `+userColumns+` FROM auth.users WHERE user_id = $1`, userID)
	})
}

// UpdateUser rewrites the email address and the password hash together. The
// request body carries both and both are required, so an update that means to
// change only the address still re-encodes the password it was sent — frozen,
// because a client that stopped sending one would start failing validation.
func (s *PostgresStore) UpdateUser(
	ctx context.Context, userID int64, emailAddress, passwordHash string, actorUserID int64,
) (User, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (User, error) {
		tag, err := tx.Exec(ctx, `
			UPDATE auth.users
			SET email_address = $1, password = $2, modified_by_user_id = $3, modified_time = $4
			WHERE user_id = $5`, emailAddress, passwordHash, actorUserID, time.Now().UTC(), userID)
		if isUniqueViolation(err) {
			return User{}, ErrNameTaken
		}
		if err != nil {
			return User{}, fmt.Errorf("update user %d: %w", userID, err)
		}
		if tag.RowsAffected() == 0 {
			return User{}, ErrUserNotFound
		}
		return loadUser(ctx, tx, `SELECT `+userColumns+` FROM auth.users WHERE user_id = $1`, userID)
	})
}

// DeleteUser is the one write that crosses the service boundary: three of the
// five tables belong to profile-service. It stays a single local transaction
// rather than a call into that service, which would make user deletion fail
// whenever the other service is down and leave a half-deleted user when it
// failed midway.
//
// The order is load-bearing application logic, not a database behaviour: no
// foreign key in the auth schema declares ON DELETE CASCADE, so a parent
// deleted before its children fails the constraint.
func (s *PostgresStore) DeleteUser(ctx context.Context, userID int64) error {
	_, err := inTransaction(ctx, s.pool, func(tx pgx.Tx) (User, error) {
		// Every profile row, not the first: one profile per user is an
		// application rule that profile-service enforces with a read before the
		// insert, and auth.profiles.user_id carries a plain index rather than a
		// unique one. Clearing only the first would leave the second's children
		// behind, the profile delete below would then break its foreign key, and
		// the user would be undeletable behind a 500 from then on.
		rows, err := tx.Query(ctx, `SELECT profile_id FROM auth.profiles WHERE user_id = $1`, userID)
		if err != nil {
			return User{}, fmt.Errorf("find profiles of user %d: %w", userID, err)
		}
		profileIDs, err := scanIDs(rows)
		if err != nil {
			return User{}, err
		}
		if len(profileIDs) > 0 {
			for _, statement := range []string{
				`DELETE FROM auth.addresses WHERE profile_id = ANY($1)`,
				`DELETE FROM auth.profile_icons WHERE profile_id = ANY($1)`,
			} {
				if _, err := tx.Exec(ctx, statement, profileIDs); err != nil {
					return User{}, fmt.Errorf("delete subresources of user %d: %w", userID, err)
				}
			}
		}
		for _, statement := range []string{
			`DELETE FROM auth.profiles WHERE user_id = $1`,
			`DELETE FROM auth.users_roles WHERE user_id = $1`,
			`DELETE FROM auth.users WHERE user_id = $1`,
		} {
			if _, err := tx.Exec(ctx, statement, userID); err != nil {
				return User{}, fmt.Errorf("delete user %d: %w", userID, err)
			}
		}
		return User{}, nil
	})
	return err
}

// UserHasAnyRole backs the elevated-role guard. It reads the grant table rather
// than the roles claim on the caller's token because the guard is written in
// terms of role ids and the claim carries names; a name-to-id mapping resolved
// here would be a second source of truth for which role is elevated.
func (s *PostgresStore) UserHasAnyRole(ctx context.Context, userID int64, roleIDs []int64) (bool, error) {
	return rowExists(ctx, s.pool,
		`SELECT 1 FROM auth.users_roles WHERE user_id = $1 AND role_id = ANY($2)`, userID, roleIDs)
}

func (s *PostgresStore) GrantRolesToUser(
	ctx context.Context, userID int64, roleIDs []int64, actorUserID int64,
) (User, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (User, error) {
		if err := requireUsers(ctx, tx, []int64{userID}); err != nil {
			return User{}, err
		}
		if err := requireRoles(ctx, tx, roleIDs); err != nil {
			return User{}, err
		}
		if err := grantRoles(ctx, tx, userID, roleIDs, actorUserID); err != nil {
			return User{}, err
		}
		return loadUser(ctx, tx, `SELECT `+userColumns+` FROM auth.users WHERE user_id = $1`, userID)
	})
}

func (s *PostgresStore) RevokeRolesFromUser(
	ctx context.Context, userID int64, roleIDs []int64, _ int64,
) (User, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (User, error) {
		if err := requireUsers(ctx, tx, []int64{userID}); err != nil {
			return User{}, err
		}
		if err := requireRoles(ctx, tx, roleIDs); err != nil {
			return User{}, err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM auth.users_roles WHERE user_id = $1 AND role_id = ANY($2)`, userID, roleIDs); err != nil {
			return User{}, fmt.Errorf("revoke roles from user %d: %w", userID, err)
		}
		return loadUser(ctx, tx, `SELECT `+userColumns+` FROM auth.users WHERE user_id = $1`, userID)
	})
}

func (s *PostgresStore) ListRoles(ctx context.Context, limit, offset int) ([]Role, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT `+roleColumns+` FROM auth.roles ORDER BY role_id LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list roles: %w", err)
	}
	roles, err := scanRoles(rows)
	if err != nil {
		return nil, err
	}
	if err := attachRoleGrants(ctx, s.pool, roles); err != nil {
		return nil, err
	}
	return roles, nil
}

func (s *PostgresStore) RoleByID(ctx context.Context, roleID int64) (Role, error) {
	return loadRole(ctx, s.pool, `SELECT `+roleColumns+` FROM auth.roles WHERE role_id = $1`, roleID)
}

func (s *PostgresStore) RoleByName(ctx context.Context, name string) (Role, error) {
	return loadRole(ctx, s.pool, `SELECT `+roleColumns+` FROM auth.roles WHERE role_name = $1`, name)
}

func (s *PostgresStore) CreateRole(
	ctx context.Context, name, description string, actorUserID int64,
) (Role, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (Role, error) {
		exists, err := rowExists(ctx, tx, `SELECT 1 FROM auth.roles WHERE role_name = $1`, name)
		if err != nil {
			return Role{}, err
		}
		if exists {
			return Role{}, ErrRoleExists
		}
		now := time.Now().UTC()
		var roleID int64
		err = tx.QueryRow(ctx, `
			INSERT INTO auth.roles (role_name, role_description, status, created_by_user_id, created_time,
				modified_by_user_id, modified_time)
			VALUES ($1, $2, $3, $4, $5, $4, $5)
			RETURNING role_id`, name, description, statusActive, actorUserID, now).Scan(&roleID)
		if isUniqueViolation(err) {
			return Role{}, ErrRoleExists
		}
		if err != nil {
			return Role{}, fmt.Errorf("create role: %w", err)
		}
		return loadRole(ctx, tx, `SELECT `+roleColumns+` FROM auth.roles WHERE role_id = $1`, roleID)
	})
}

// UpdateRole has no pre-check on the new name, unlike CreateRole. The asymmetry
// is the JVM's: renaming onto a name another role holds reaches Postgres as a
// unique violation and surfaces as 500 rather than the create's 409.
func (s *PostgresStore) UpdateRole(
	ctx context.Context, roleID int64, name, description string, actorUserID int64,
) (Role, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (Role, error) {
		tag, err := tx.Exec(ctx, `
			UPDATE auth.roles
			SET role_name = $1, role_description = $2, modified_by_user_id = $3, modified_time = $4
			WHERE role_id = $5`, name, description, actorUserID, time.Now().UTC(), roleID)
		if isUniqueViolation(err) {
			return Role{}, ErrNameTaken
		}
		if err != nil {
			return Role{}, fmt.Errorf("update role %d: %w", roleID, err)
		}
		if tag.RowsAffected() == 0 {
			return Role{}, ErrRoleNotFound
		}
		return loadRole(ctx, tx, `SELECT `+roleColumns+` FROM auth.roles WHERE role_id = $1`, roleID)
	})
}

// DeleteRole clears the grants first: auth.users_roles references auth.roles
// with no cascade, so the order is what makes the delete possible at all.
func (s *PostgresStore) DeleteRole(ctx context.Context, roleID int64) error {
	_, err := inTransaction(ctx, s.pool, func(tx pgx.Tx) (Role, error) {
		for _, statement := range []string{
			`DELETE FROM auth.users_roles WHERE role_id = $1`,
			`DELETE FROM auth.roles WHERE role_id = $1`,
		} {
			if _, err := tx.Exec(ctx, statement, roleID); err != nil {
				return Role{}, fmt.Errorf("delete role %d: %w", roleID, err)
			}
		}
		return Role{}, nil
	})
	return err
}

func (s *PostgresStore) GrantUsersToRole(
	ctx context.Context, roleID int64, userIDs []int64, actorUserID int64,
) (Role, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (Role, error) {
		if err := requireRoles(ctx, tx, []int64{roleID}); err != nil {
			return Role{}, err
		}
		if err := requireUsers(ctx, tx, userIDs); err != nil {
			return Role{}, err
		}
		if err := grantUsers(ctx, tx, roleID, userIDs, actorUserID); err != nil {
			return Role{}, err
		}
		return loadRole(ctx, tx, `SELECT `+roleColumns+` FROM auth.roles WHERE role_id = $1`, roleID)
	})
}

func (s *PostgresStore) RevokeUsersFromRole(
	ctx context.Context, roleID int64, userIDs []int64, _ int64,
) (Role, error) {
	return inTransaction(ctx, s.pool, func(tx pgx.Tx) (Role, error) {
		if err := requireRoles(ctx, tx, []int64{roleID}); err != nil {
			return Role{}, err
		}
		if err := requireUsers(ctx, tx, userIDs); err != nil {
			return Role{}, err
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM auth.users_roles WHERE role_id = $1 AND user_id = ANY($2)`, roleID, userIDs); err != nil {
			return Role{}, fmt.Errorf("revoke users from role %d: %w", roleID, err)
		}
		return loadRole(ctx, tx, `SELECT `+roleColumns+` FROM auth.roles WHERE role_id = $1`, roleID)
	})
}

// grantRoles inserts every pair the request names that is not there already, in
// one statement. ON CONFLICT DO NOTHING does what the JVM does with a read
// before each insert, and closes the race between the two.
func grantRoles(ctx context.Context, q querier, userID int64, roleIDs []int64, actorUserID int64) error {
	_, err := q.Exec(ctx, `
		INSERT INTO auth.users_roles (user_id, role_id, created_by_user_id, created_time)
		SELECT $1, role_id, $3, $4 FROM unnest($2::bigint[]) AS role_id
		ON CONFLICT (user_id, role_id) DO NOTHING`,
		userID, roleIDs, actorUserID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("grant roles to user %d: %w", userID, err)
	}
	return nil
}

// grantUsers is the same insert from the role side.
func grantUsers(ctx context.Context, q querier, roleID int64, userIDs []int64, actorUserID int64) error {
	_, err := q.Exec(ctx, `
		INSERT INTO auth.users_roles (user_id, role_id, created_by_user_id, created_time)
		SELECT user_id, $1, $3, $4 FROM unnest($2::bigint[]) AS user_id
		ON CONFLICT (user_id, role_id) DO NOTHING`,
		roleID, userIDs, actorUserID, time.Now().UTC())
	if err != nil {
		return fmt.Errorf("grant role %d to users: %w", roleID, err)
	}
	return nil
}

// inTransaction runs body in a transaction, rolling back on any error. The
// rollback after a commit is a no-op, which is why it can be deferred
// unconditionally.
func inTransaction[T any](ctx context.Context, pool *pgxpool.Pool, body func(pgx.Tx) (T, error)) (T, error) {
	var zero T
	tx, err := pool.Begin(ctx)
	if err != nil {
		return zero, fmt.Errorf("begin: %w", err)
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
		return zero, err
	}
	if err := tx.Commit(ctx); err != nil {
		return zero, fmt.Errorf("commit: %w", err)
	}
	return result, nil
}

// requireUsers and requireRoles check every id a request named in one query and
// report the first that is absent, because the message the caller reads carries
// that id. One query rather than one per id: a grant body carries no declared
// upper bound, so a per-id check would let a caller choose how many round trips
// a single request costs.
func requireUsers(ctx context.Context, q querier, userIDs []int64) error {
	return requireAll(ctx, q, `SELECT user_id FROM auth.users WHERE user_id = ANY($1)`, userIDs, ErrUserNotFound)
}

func requireRoles(ctx context.Context, q querier, roleIDs []int64) error {
	return requireAll(ctx, q, `SELECT role_id FROM auth.roles WHERE role_id = ANY($1)`, roleIDs, ErrRoleNotFound)
}

func requireAll(ctx context.Context, q querier, query string, ids []int64, sentinel error) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := q.Query(ctx, query, ids)
	if err != nil {
		return fmt.Errorf("existence check: %w", err)
	}
	present, err := scanIDSet(rows)
	if err != nil {
		return err
	}
	// Reported in the order the request named them, so the id in the message is
	// the one a caller reading their own payload would expect.
	for _, id := range ids {
		if !present[id] {
			return &notFound{sentinel: sentinel, id: id}
		}
	}
	return nil
}

func scanIDs(rows pgx.Rows) ([]int64, error) {
	defer rows.Close()
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read ids: %w", err)
	}
	return ids, nil
}

func scanIDSet(rows pgx.Rows) (map[int64]bool, error) {
	defer rows.Close()
	present := map[int64]bool{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan id: %w", err)
		}
		present[id] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("existence check: %w", err)
	}
	return present, nil
}

func rowExists(ctx context.Context, q querier, query string, arguments ...any) (bool, error) {
	var one int
	err := q.QueryRow(ctx, query, arguments...).Scan(&one)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("existence check: %w", err)
	}
	return true, nil
}

func loadUser(ctx context.Context, q querier, query string, argument any) (User, error) {
	rows, err := q.Query(ctx, query, argument)
	if err != nil {
		return User{}, fmt.Errorf("read user: %w", err)
	}
	users, err := scanUsers(rows)
	if err != nil {
		return User{}, err
	}
	if len(users) == 0 {
		return User{}, ErrUserNotFound
	}
	if err := attachUserSubresources(ctx, q, users); err != nil {
		return User{}, err
	}
	return users[0], nil
}

func loadRole(ctx context.Context, q querier, query string, argument any) (Role, error) {
	rows, err := q.Query(ctx, query, argument)
	if err != nil {
		return Role{}, fmt.Errorf("read role: %w", err)
	}
	roles, err := scanRoles(rows)
	if err != nil {
		return Role{}, err
	}
	if len(roles) == 0 {
		return Role{}, ErrRoleNotFound
	}
	if err := attachRoleGrants(ctx, q, roles); err != nil {
		return Role{}, err
	}
	return roles[0], nil
}

func scanUsers(rows pgx.Rows) ([]User, error) {
	defer rows.Close()
	users := []User{}
	for rows.Next() {
		var user User
		var createdTime, modifiedTime time.Time
		if err := rows.Scan(&user.ID, &user.EmailAddress, &user.Status, &user.CreatedByUserID,
			&createdTime, &user.ModifiedByUserID, &modifiedTime); err != nil {
			return nil, fmt.Errorf("scan user: %w", err)
		}
		user.CreatedTime = Timestamp{createdTime}
		user.ModifiedTime = Timestamp{modifiedTime}
		user.Roles = []UserRole{}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read users: %w", err)
	}
	return users, nil
}

func scanRoles(rows pgx.Rows) ([]Role, error) {
	defer rows.Close()
	roles := []Role{}
	for rows.Next() {
		var role Role
		var createdTime, modifiedTime time.Time
		if err := rows.Scan(&role.ID, &role.Name, &role.Description, &role.Status, &role.CreatedByUserID,
			&createdTime, &role.ModifiedByUserID, &modifiedTime); err != nil {
			return nil, fmt.Errorf("scan role: %w", err)
		}
		role.CreatedTime = Timestamp{createdTime}
		role.ModifiedTime = Timestamp{modifiedTime}
		role.Users = []UserRole{}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read roles: %w", err)
	}
	return roles, nil
}

func scanStrings(rows pgx.Rows) ([]string, error) {
	defer rows.Close()
	values := []string{}
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("scan: %w", err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read: %w", err)
	}
	return values, nil
}

// attachUserSubresources fills in the grants and the profile id of every user in
// one query each, whatever the page size. UserRepositoryImpl issues four queries
// per row instead, so a hundred-row page costs it four hundred round trips.
func attachUserSubresources(ctx context.Context, q querier, users []User) error {
	if len(users) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(users))
	at := make(map[int64]*User, len(users))
	for i := range users {
		ids = append(ids, users[i].ID)
		at[users[i].ID] = &users[i]
	}

	grants, err := q.Query(ctx,
		`SELECT `+grantColumns+` FROM auth.users_roles WHERE user_id = ANY($1) ORDER BY user_id, role_id`, ids)
	if err != nil {
		return fmt.Errorf("read grants: %w", err)
	}
	if err := func() error {
		defer grants.Close()
		for grants.Next() {
			grant, err := scanGrant(grants)
			if err != nil {
				return err
			}
			if user, known := at[grant.UserID]; known {
				user.Roles = append(user.Roles, grant)
			}
		}
		return grants.Err()
	}(); err != nil {
		return fmt.Errorf("read grants: %w", err)
	}

	// A read across the boundary: auth.profiles belongs to profile-service and
	// this is the only place the identity half touches it for a read.
	profiles, err := q.Query(ctx, `SELECT user_id, profile_id FROM auth.profiles WHERE user_id = ANY($1)`, ids)
	if err != nil {
		return fmt.Errorf("read profile ids: %w", err)
	}
	defer profiles.Close()
	for profiles.Next() {
		var userID, profileID int64
		if err := profiles.Scan(&userID, &profileID); err != nil {
			return fmt.Errorf("scan profile id: %w", err)
		}
		if user, known := at[userID]; known {
			stored := profileID
			user.ProfileID = &stored
		}
	}
	if err := profiles.Err(); err != nil {
		return fmt.Errorf("read profile ids: %w", err)
	}
	return nil
}

func attachRoleGrants(ctx context.Context, q querier, roles []Role) error {
	if len(roles) == 0 {
		return nil
	}
	ids := make([]int64, 0, len(roles))
	at := make(map[int64]*Role, len(roles))
	for i := range roles {
		ids = append(ids, roles[i].ID)
		at[roles[i].ID] = &roles[i]
	}

	rows, err := q.Query(ctx,
		`SELECT `+grantColumns+` FROM auth.users_roles WHERE role_id = ANY($1) ORDER BY role_id, user_id`, ids)
	if err != nil {
		return fmt.Errorf("read grants: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		grant, err := scanGrant(rows)
		if err != nil {
			return err
		}
		if role, known := at[grant.RoleID]; known {
			role.Users = append(role.Users, grant)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read grants: %w", err)
	}
	return nil
}

func scanGrant(rows pgx.Rows) (UserRole, error) {
	var grant UserRole
	var createdTime time.Time
	if err := rows.Scan(&grant.UserID, &grant.RoleID, &grant.CreatedByUserID, &createdTime); err != nil {
		return UserRole{}, fmt.Errorf("scan grant: %w", err)
	}
	grant.CreatedTime = Timestamp{createdTime}
	return grant, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == uniqueViolation
}
