package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// schemaPath is the deployed schema itself rather than a copy, so a column this
// service reads cannot drift from the one the database has.
const schemaPath = "../../../apps/database/authdb/src/00_schema.sql"

// The container is started once for the package and shared: each test seeds its
// own user and works within it, so they do not need a database each. Starting
// one per test cost six minutes of wall clock for this suite.
var (
	sharedPostgres    sync.Once
	sharedPool        *pgxpool.Pool
	sharedPostgresErr error
	terminatePostgres func()
)

func TestMain(m *testing.M) {
	code := m.Run()
	if terminatePostgres != nil {
		terminatePostgres()
	}
	os.Exit(code)
}

// startPostgres brings up the deployed schema in the shared container.
func startPostgres(t *testing.T) *pgxpool.Pool {
	t.Helper()
	if testing.Short() {
		t.Skip("needs Docker for a Postgres container")
	}

	sharedPostgres.Do(func() { sharedPool, sharedPostgresErr = runPostgres(mustAbs(t, schemaPath)) })
	if sharedPostgresErr != nil {
		t.Fatalf("start postgres: %v", sharedPostgresErr)
	}
	return sharedPool
}

func runPostgres(schema string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("jdw"),
		postgres.WithUsername("jdw"),
		postgres.WithPassword("jdw"),
		postgres.WithInitScripts(schema),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(2*time.Minute)),
	)
	if err != nil {
		return nil, fmt.Errorf("run container: %w", err)
	}
	terminatePostgres = func() {
		if err := testcontainers.TerminateContainer(container); err != nil {
			log.Printf("terminate postgres: %v", err)
		}
	}

	dsn, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("connection string: %w", err)
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("open pool: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("ping: %w", err)
	}
	return pool, nil
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	absolute, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve %s: %v", path, err)
	}
	if _, err := os.Stat(absolute); err != nil {
		t.Fatalf("stat %s: %v", absolute, err)
	}
	return absolute
}

// seedUser inserts a user and returns its generated id. The email is unique per
// call so tests sharing one container cannot collide.
func seedUser(t *testing.T, pool *pgxpool.Pool, email string) int64 {
	t.Helper()
	var id int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO auth.users (email_address, password, status, created_by_user_id, created_time, modified_by_user_id, modified_time)
		VALUES ($1, 'x', 'ACTIVE', 1, now(), 1, now())
		RETURNING user_id`, email).Scan(&id)
	if err != nil {
		t.Fatalf("seed user %s: %v", email, err)
	}
	return id
}

func newTestStore(t *testing.T) (*PostgresStore, *pgxpool.Pool) {
	t.Helper()
	pool := startPostgres(t)
	return NewPostgresStore(pool), pool
}

func createProfileFor(t *testing.T, store *PostgresStore, userID int64) Profile {
	t.Helper()
	profile, err := store.CreateProfile(context.Background(), ProfileCreateRequest{
		FirstName: ptr("Ada"),
		LastName:  ptr("Lovelace"),
		Birthdate: &Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)},
		UserID:    &userID,
	}, userID)
	if err != nil {
		t.Fatalf("CreateProfile: %v", err)
	}
	return profile
}

func TestCreatingAProfile(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "create@jdw.com")

	profile := createProfileFor(t, store, userID)

	if profile.ID == 0 {
		t.Error("the created profile carries no id")
	}
	if profile.UserID != userID {
		t.Errorf("userId = %d, want %d", profile.UserID, userID)
	}
	if profile.FirstName != "Ada" || profile.LastName != "Lovelace" {
		t.Errorf("names = %q %q, want Ada Lovelace", profile.FirstName, profile.LastName)
	}
	if got := profile.Birthdate.Format("2006-01-02"); got != "1815-12-10" {
		t.Errorf("birthdate = %s, want 1815-12-10", got)
	}
	if profile.Addresses == nil || len(profile.Addresses) != 0 {
		t.Errorf("addresses = %v, want an empty set", profile.Addresses)
	}
	if profile.Icon != nil {
		t.Errorf("icon = %v, want none", profile.Icon)
	}
	if profile.CreatedByUserID != userID || profile.ModifiedByUserID != userID {
		t.Errorf("audit ids = %d/%d, want %d", profile.CreatedByUserID, profile.ModifiedByUserID, userID)
	}
	if profile.CreatedTime.IsZero() || profile.ModifiedTime.IsZero() {
		t.Error("the audit stamps are unset")
	}

	if _, err := store.ProfileByID(ctx, profile.ID); err != nil {
		t.Errorf("the created profile does not read back: %v", err)
	}
}

func TestCreatingAProfileForAUserThatDoesNotExist(t *testing.T) {
	// The JVM proves the user exists with a read before inserting. The foreign
	// key on auth.profiles.user_id does it instead, which is stronger and free,
	// but the caller still has to see the same 404 rather than a 500.
	store, _ := newTestStore(t)
	missing := int64(987654)

	_, err := store.CreateProfile(context.Background(), ProfileCreateRequest{
		FirstName: ptr("Ada"), LastName: ptr("Lovelace"),
		Birthdate: &Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)}, UserID: &missing,
	}, 1)

	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestCreatingASecondProfileForTheSameUser(t *testing.T) {
	store, pool := newTestStore(t)
	userID := seedUser(t, pool, "second@jdw.com")
	createProfileFor(t, store, userID)

	_, err := store.CreateProfile(context.Background(), ProfileCreateRequest{
		FirstName: ptr("Ada"), LastName: ptr("Lovelace"),
		Birthdate: &Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)}, UserID: &userID,
	}, userID)

	if !errors.Is(err, ErrProfileExists) {
		t.Errorf("error = %v, want %v", err, ErrProfileExists)
	}
}

func TestReadingAProfileThatIsNotThere(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	if _, err := store.ProfileByID(ctx, 987654); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ProfileByID error = %v, want %v", err, ErrProfileNotFound)
	}
	if _, err := store.ProfileByUserID(ctx, 987654); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ProfileByUserID error = %v, want %v", err, ErrProfileNotFound)
	}
}

func TestResolvingAProfileIdFromAUserId(t *testing.T) {
	// The fallback the split needs: a caller whose token predates their profile
	// carries no profile_id claim, and this lookup is what keeps them from being
	// locked out of the profile they just created.
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "fallback@jdw.com")
	profile := createProfileFor(t, store, userID)

	resolved, found, err := store.ProfileIDForUser(ctx, userID)

	if err != nil {
		t.Fatalf("ProfileIDForUser: %v", err)
	}
	if !found || resolved != profile.ID {
		t.Errorf("resolved = %d/%v, want %d/true", resolved, found, profile.ID)
	}

	if _, found, err := store.ProfileIDForUser(ctx, seedUser(t, pool, "profileless@jdw.com")); err != nil || found {
		t.Errorf("a user with no profile resolved to found=%v, err=%v", found, err)
	}
}

func TestUpdatingAProfile(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "update@jdw.com")
	actorID := seedUser(t, pool, "update-actor@jdw.com")
	profile := createProfileFor(t, store, userID)

	updated, err := store.UpdateProfileByID(ctx, profile.ID, ProfileUpdateRequest{
		FirstName: ptr("Augusta"), MiddleName: ptr("Ada"), LastName: ptr("King"),
		Birthdate: &Date{time.Date(1815, 12, 11, 0, 0, 0, 0, time.UTC)},
	}, actorID)

	if err != nil {
		t.Fatalf("UpdateProfileByID: %v", err)
	}
	if updated.FirstName != "Augusta" || updated.LastName != "King" {
		t.Errorf("names = %q %q, want Augusta King", updated.FirstName, updated.LastName)
	}
	if updated.MiddleName == nil || *updated.MiddleName != "Ada" {
		t.Errorf("middleName = %v, want Ada", updated.MiddleName)
	}
	if updated.UserID != userID {
		t.Errorf("userId = %d, want %d; an update must not move a profile between users", updated.UserID, userID)
	}
	if updated.ModifiedByUserID != actorID {
		t.Errorf("modifiedByUserId = %d, want the acting user %d", updated.ModifiedByUserID, actorID)
	}
	if updated.CreatedByUserID != userID {
		t.Errorf("createdByUserId = %d, want the original %d", updated.CreatedByUserID, userID)
	}
}

func TestUpdatingAProfileByUserId(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "update-by-user@jdw.com")
	profile := createProfileFor(t, store, userID)

	updated, err := store.UpdateProfileByUserID(ctx, userID, ProfileUpdateRequest{
		FirstName: ptr("Augusta"), LastName: ptr("King"),
		Birthdate: &Date{time.Date(1815, 12, 11, 0, 0, 0, 0, time.UTC)},
	}, userID)

	if err != nil {
		t.Fatalf("UpdateProfileByUserID: %v", err)
	}
	if updated.ID != profile.ID {
		t.Errorf("id = %d, want %d", updated.ID, profile.ID)
	}
	if updated.FirstName != "Augusta" {
		t.Errorf("firstName = %q, want Augusta", updated.FirstName)
	}

	if _, err := store.UpdateProfileByUserID(ctx, 987654, ProfileUpdateRequest{
		FirstName: ptr("A"), LastName: ptr("B"), Birthdate: &Date{time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)},
	}, 1); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("error for a user with no profile = %v, want %v", err, ErrProfileNotFound)
	}
}

func TestDeletingAProfileTakesItsAddressesAndIconWithIt(t *testing.T) {
	// auth.addresses and auth.profile_icons reference auth.profiles with no
	// cascade, so the delete has to clear them in order or the profile row
	// cannot go.
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "delete-cascade@jdw.com")
	profile := createProfileFor(t, store, userID)
	if _, err := store.AddAddress(ctx, profile.ID, completeAddress(), userID); err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	if _, err := store.AddIcon(ctx, profile.ID, []byte("icon"), userID); err != nil {
		t.Fatalf("AddIcon: %v", err)
	}

	if err := store.DeleteProfileByID(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteProfileByID: %v", err)
	}

	if _, err := store.ProfileByID(ctx, profile.ID); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("the profile survived the delete: %v", err)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE profile_id = $1", profile.ID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profile_icons WHERE profile_id = $1", profile.ID, 0)
}

func TestDeletingAProfileThatIsNotThereIsNotAnError(t *testing.T) {
	// ProfileRepositoryImpl.deleteById has no existence check, which is why the
	// operation has no 404 in its response set at all.
	store, pool := newTestStore(t)
	ctx := context.Background()

	if err := store.DeleteProfileByID(ctx, 987654); err != nil {
		t.Errorf("DeleteProfileByID on a missing id = %v, want no error", err)
	}
	if err := store.DeleteProfileByUserID(ctx, seedUser(t, pool, "no-profile-delete@jdw.com")); err != nil {
		t.Errorf("DeleteProfileByUserID for a user with no profile = %v, want no error", err)
	}
}

func TestDeletingAProfileByUserIdTakesItsSubresourcesWithIt(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "delete-by-user@jdw.com")
	profile := createProfileFor(t, store, userID)
	if _, err := store.AddIcon(ctx, profile.ID, []byte("icon"), userID); err != nil {
		t.Fatalf("AddIcon: %v", err)
	}

	if err := store.DeleteProfileByUserID(ctx, userID); err != nil {
		t.Fatalf("DeleteProfileByUserID: %v", err)
	}

	assertRowCount(t, pool, "SELECT count(*) FROM auth.profiles WHERE user_id = $1", userID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profile_icons WHERE profile_id = $1", profile.ID, 0)
}

func completeAddress() AddressRequest {
	return AddressRequest{
		AddressLine1: ptr("12 Noel Street"), City: ptr("London"),
		StateProvince: ptr("Greater London"), PostalCode: ptr("W1F 8GQ"), Country: ptr("GB"),
	}
}

func TestAddingAnAddressAnswersWithTheWholeProfile(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "address-add@jdw.com")
	profile := createProfileFor(t, store, userID)

	withAddress, err := store.AddAddress(ctx, profile.ID, completeAddress(), userID)

	if err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	if len(withAddress.Addresses) != 1 {
		t.Fatalf("addresses = %v, want one", withAddress.Addresses)
	}
	added := withAddress.Addresses[0]
	if added.ID == 0 {
		t.Error("the created address carries no id")
	}
	if added.ProfileID != profile.ID {
		t.Errorf("profileId = %d, want %d", added.ProfileID, profile.ID)
	}
	if added.AddressLine1 != "12 Noel Street" || added.Country != "GB" {
		t.Errorf("address = %+v, want the one supplied", added)
	}
	if added.AddressLine2 != nil {
		t.Errorf("addressLine2 = %v, want null", added.AddressLine2)
	}
}

func TestAddingAnAddressToAProfileThatIsNotThere(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.AddAddress(context.Background(), 987654, completeAddress(), 1)

	if !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("error = %v, want %v", err, ErrProfileNotFound)
	}
}

func TestUpdatingAnAddressIsScopedToItsProfile(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	ownerID := seedUser(t, pool, "address-owner@jdw.com")
	strangerID := seedUser(t, pool, "address-stranger@jdw.com")
	owned := createProfileFor(t, store, ownerID)
	stranger := createProfileFor(t, store, strangerID)
	withAddress, err := store.AddAddress(ctx, owned.ID, completeAddress(), ownerID)
	if err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	addressID := withAddress.Addresses[0].ID

	changed := completeAddress()
	changed.City = ptr("Nottingham")
	updated, err := store.UpdateAddress(ctx, owned.ID, addressID, changed, ownerID)
	if err != nil {
		t.Fatalf("UpdateAddress: %v", err)
	}
	if updated.Addresses[0].City != "Nottingham" {
		t.Errorf("city = %q, want Nottingham", updated.Addresses[0].City)
	}

	// The same address id under a profile that does not own it.
	if _, err := store.UpdateAddress(ctx, stranger.ID, addressID, changed, strangerID); !errors.Is(err, ErrAddressNotFound) {
		t.Errorf("error = %v, want %v", err, ErrAddressNotFound)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE address_id = $1 AND city = 'Nottingham'", addressID, 1)
}

func TestDeletingAnAddressIsScopedToItsProfile(t *testing.T) {
	// The scoping is the whole point: before AddressDaoPostgres took the profile
	// id, any principal with a profile could delete any address in the table by
	// guessing a sequential id.
	store, pool := newTestStore(t)
	ctx := context.Background()
	ownerID := seedUser(t, pool, "address-delete-owner@jdw.com")
	strangerID := seedUser(t, pool, "address-delete-stranger@jdw.com")
	owned := createProfileFor(t, store, ownerID)
	stranger := createProfileFor(t, store, strangerID)
	withAddress, err := store.AddAddress(ctx, owned.ID, completeAddress(), ownerID)
	if err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	addressID := withAddress.Addresses[0].ID

	if err := store.DeleteAddress(ctx, stranger.ID, addressID); !errors.Is(err, ErrAddressNotFound) {
		t.Errorf("a delete from another profile = %v, want %v", err, ErrAddressNotFound)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE address_id = $1", addressID, 1)

	if err := store.DeleteAddress(ctx, owned.ID, addressID); err != nil {
		t.Errorf("the owner's delete = %v, want no error", err)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE address_id = $1", addressID, 0)
}

func TestAddressesComeBackInAStableOrder(t *testing.T) {
	// The JVM builds a HashSet, so its order is whatever the hashes give. A
	// deterministic order is a strict improvement and keeps a diff of two reads
	// meaningful.
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "address-order@jdw.com")
	profile := createProfileFor(t, store, userID)
	for _, city := range []string{"London", "Nottingham", "Bath"} {
		address := completeAddress()
		address.City = ptr(city)
		if _, err := store.AddAddress(ctx, profile.ID, address, userID); err != nil {
			t.Fatalf("AddAddress %s: %v", city, err)
		}
	}

	read, err := store.ProfileByID(ctx, profile.ID)
	if err != nil {
		t.Fatalf("ProfileByID: %v", err)
	}

	if len(read.Addresses) != 3 {
		t.Fatalf("addresses = %v, want three", read.Addresses)
	}
	for i := 1; i < len(read.Addresses); i++ {
		if read.Addresses[i-1].ID >= read.Addresses[i].ID {
			t.Errorf("addresses are not ordered by id: %d then %d", read.Addresses[i-1].ID, read.Addresses[i].ID)
		}
	}
}

func TestTheIconLifecycle(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "icon@jdw.com")
	actorID := seedUser(t, pool, "icon-actor@jdw.com")
	profile := createProfileFor(t, store, userID)

	if _, err := store.Icon(ctx, profile.ID); !errors.Is(err, ErrIconNotFound) {
		t.Errorf("reading a missing icon = %v, want %v", err, ErrIconNotFound)
	}

	added, err := store.AddIcon(ctx, profile.ID, []byte("first"), userID)
	if err != nil {
		t.Fatalf("AddIcon: %v", err)
	}
	if added.Icon == nil || string(added.Icon.Icon) != "first" {
		t.Fatalf("icon = %v, want the uploaded bytes", added.Icon)
	}
	if added.Icon.ProfileID != profile.ID {
		t.Errorf("icon profileId = %d, want %d", added.Icon.ProfileID, profile.ID)
	}

	if _, err := store.AddIcon(ctx, profile.ID, []byte("second"), userID); !errors.Is(err, ErrIconExists) {
		t.Errorf("a second upload = %v, want %v", err, ErrIconExists)
	}

	replaced, err := store.ReplaceIcon(ctx, profile.ID, []byte("replacement"), actorID)
	if err != nil {
		t.Fatalf("ReplaceIcon: %v", err)
	}
	if string(replaced.Icon.Icon) != "replacement" {
		t.Errorf("icon = %q, want the replacement", replaced.Icon.Icon)
	}
	if replaced.Icon.ID != added.Icon.ID {
		t.Errorf("icon id = %d, want the original %d; a replacement keeps the row", replaced.Icon.ID, added.Icon.ID)
	}
	if replaced.Icon.CreatedByUserID != userID || replaced.Icon.ModifiedByUserID != actorID {
		t.Errorf("icon audit = %d/%d, want %d/%d",
			replaced.Icon.CreatedByUserID, replaced.Icon.ModifiedByUserID, userID, actorID)
	}

	read, err := store.Icon(ctx, profile.ID)
	if err != nil {
		t.Fatalf("Icon: %v", err)
	}
	if string(read.Icon) != "replacement" {
		t.Errorf("stored icon = %q, want the replacement", read.Icon)
	}

	if err := store.DeleteIcon(ctx, profile.ID); err != nil {
		t.Fatalf("DeleteIcon: %v", err)
	}
	if _, err := store.Icon(ctx, profile.ID); !errors.Is(err, ErrIconNotFound) {
		t.Errorf("the icon survived the delete: %v", err)
	}
	if err := store.DeleteIcon(ctx, profile.ID); err != nil {
		t.Errorf("deleting an absent icon = %v, want no error", err)
	}
}

func TestReplacingAnIconAProfileDoesNotHave(t *testing.T) {
	// The frozen defect: the JVM reads the current icon's id to carry onto the
	// replacement and dereferences null, answering 500 rather than 404. Named
	// here so the status is a decision rather than a crash.
	store, pool := newTestStore(t)
	userID := seedUser(t, pool, "icon-replace-missing@jdw.com")
	profile := createProfileFor(t, store, userID)

	_, err := store.ReplaceIcon(context.Background(), profile.ID, []byte("x"), userID)

	if !errors.Is(err, ErrNoIconToReplace) {
		t.Errorf("error = %v, want %v", err, ErrNoIconToReplace)
	}
}

func TestIconOperationsOnAProfileThatIsNotThere(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	if _, err := store.AddIcon(ctx, 987654, []byte("x"), 1); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("AddIcon = %v, want %v", err, ErrProfileNotFound)
	}
	if _, err := store.ReplaceIcon(ctx, 987654, []byte("x"), 1); !errors.Is(err, ErrProfileNotFound) {
		t.Errorf("ReplaceIcon = %v, want %v", err, ErrProfileNotFound)
	}
	if _, err := store.Icon(ctx, 987654); !errors.Is(err, ErrIconNotFound) {
		t.Errorf("Icon = %v, want %v", err, ErrIconNotFound)
	}
}

func TestListingProfilesIsPagedAndOrderedById(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	created := make([]int64, 0, 3)
	for _, email := range []string{"list-1@jdw.com", "list-2@jdw.com", "list-3@jdw.com"} {
		created = append(created, createProfileFor(t, store, seedUser(t, pool, email)).ID)
	}

	all, err := store.ListProfiles(ctx, 500, 0)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatalf("profiles are not ordered by id: %d then %d", all[i-1].ID, all[i].ID)
		}
	}

	// A page of one, offset onto the second profile this test created.
	position := 0
	for i, profile := range all {
		if profile.ID == created[0] {
			position = i
			break
		}
	}
	page, err := store.ListProfiles(ctx, 1, position+1)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}
	if len(page) != 1 || page[0].ID != created[1] {
		t.Errorf("page = %v, want the profile with id %d alone", page, created[1])
	}
}

func TestAListedProfileCarriesItsAddressesAndIcon(t *testing.T) {
	// ProfileRepositoryImpl.findAll issues two more queries per row. The set-based
	// read here has to produce the same aggregate without that fan-out.
	store, pool := newTestStore(t)
	ctx := context.Background()
	userID := seedUser(t, pool, "list-aggregate@jdw.com")
	profile := createProfileFor(t, store, userID)
	if _, err := store.AddAddress(ctx, profile.ID, completeAddress(), userID); err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	if _, err := store.AddIcon(ctx, profile.ID, []byte("icon"), userID); err != nil {
		t.Fatalf("AddIcon: %v", err)
	}

	all, err := store.ListProfiles(ctx, 500, 0)
	if err != nil {
		t.Fatalf("ListProfiles: %v", err)
	}

	var listed *Profile
	for i := range all {
		if all[i].ID == profile.ID {
			listed = &all[i]
			break
		}
	}
	if listed == nil {
		t.Fatalf("the profile with id %d is not in the listing", profile.ID)
	}
	if len(listed.Addresses) != 1 {
		t.Errorf("addresses = %v, want one", listed.Addresses)
	}
	if listed.Icon == nil || string(listed.Icon.Icon) != "icon" {
		t.Errorf("icon = %v, want the stored bytes", listed.Icon)
	}
}

func assertRowCount(t *testing.T, pool *pgxpool.Pool, query string, argument any, want int) {
	t.Helper()
	var count int
	if err := pool.QueryRow(context.Background(), query, argument).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != want {
		t.Errorf("row count = %d, want %d", count, want)
	}
}
