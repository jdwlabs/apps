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

// The deployed schema and its seed data, rather than copies, so a column this
// service reads cannot drift from the one the database has. The seed matters
// here in a way it does not for the sibling service: it is what populates the
// role catalogue, and the elevated-role guard names a role by id.
const (
	schemaPath = "../../../apps/database/authdb/src/00_schema.sql"
	seedPath   = "../../../apps/database/authdb/src/01_data.sql"
)

// The container is started once for the package and shared: each test seeds its
// own rows and works within them, so they do not need a database each. Starting
// one per test cost minutes of wall clock for this suite.
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

	sharedPostgres.Do(func() {
		sharedPool, sharedPostgresErr = runPostgres(mustAbs(t, schemaPath), mustAbs(t, seedPath))
	})
	if sharedPostgresErr != nil {
		t.Fatalf("start postgres: %v", sharedPostgresErr)
	}
	return sharedPool
}

func runPostgres(scripts ...string) (*pgxpool.Pool, error) {
	ctx := context.Background()
	container, err := postgres.Run(ctx, "postgres:18-alpine",
		postgres.WithDatabase("jdw"),
		postgres.WithUsername("jdw"),
		postgres.WithPassword("jdw"),
		postgres.WithInitScripts(scripts...),
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

func newTestStore(t *testing.T) (*PostgresStore, *pgxpool.Pool) {
	t.Helper()
	pool := startPostgres(t)
	return NewPostgresStore(pool), pool
}

// createUserFor registers a user through the store, so every test works with
// rows the service itself wrote. The email is unique per call so tests sharing
// one container cannot collide.
func createUserFor(t *testing.T, store *PostgresStore, email string) User {
	t.Helper()
	user, err := store.CreateUser(context.Background(), email, mustHash(fixturePassword), registrationRequesterID)
	if err != nil {
		t.Fatalf("CreateUser %s: %v", email, err)
	}
	return user
}

func roleIDByName(t *testing.T, store *PostgresStore, name string) int64 {
	t.Helper()
	role, err := store.RoleByName(context.Background(), name)
	if err != nil {
		t.Fatalf("RoleByName %s: %v", name, err)
	}
	return role.ID
}

func TestTheSeededCatalogueGivesTheElevatedRoleTheIdTheGuardNames(t *testing.T) {
	// UserService.validateAdminRoleRequest hard-codes the id, while the seed
	// resolves ADMIN by natural key. They agree on the deployed data, and if
	// they ever stop agreeing the guard protects the wrong role — which is a
	// silent privilege change, so it is asserted rather than assumed.
	store, _ := newTestStore(t)

	if got := roleIDByName(t, store, "ADMIN"); got != elevatedRoleID {
		t.Errorf("ADMIN is role %d and the elevated-role guard names %d", got, elevatedRoleID)
	}
}

func TestCreatingAUser(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	user := createUserFor(t, store, "create@jdw.com")

	if user.ID == 0 {
		t.Error("the created user carries no id")
	}
	if user.EmailAddress != "create@jdw.com" {
		t.Errorf("emailAddress = %q, want the one supplied", user.EmailAddress)
	}
	if user.Status != statusActive {
		t.Errorf("status = %q, want %q", user.Status, statusActive)
	}
	if user.Roles == nil || len(user.Roles) != 0 {
		t.Errorf("roles = %v, want an empty set", user.Roles)
	}
	if user.ProfileID != nil {
		t.Errorf("profileId = %v, want null for a user with no profile", user.ProfileID)
	}
	if user.CreatedByUserID != registrationRequesterID || user.ModifiedByUserID != registrationRequesterID {
		t.Errorf("audit ids = %d/%d, want %d",
			user.CreatedByUserID, user.ModifiedByUserID, registrationRequesterID)
	}
	if user.CreatedTime.IsZero() || user.ModifiedTime.IsZero() {
		t.Error("the audit stamps are unset")
	}

	if _, err := store.UserByID(ctx, user.ID); err != nil {
		t.Errorf("the created user does not read back: %v", err)
	}
}

func TestCreatingAUserWithAnEmailAddressAlreadyTaken(t *testing.T) {
	store, _ := newTestStore(t)
	createUserFor(t, store, "duplicate@jdw.com")

	_, err := store.CreateUser(context.Background(), "duplicate@jdw.com", mustHash(fixturePassword), 1)

	if !errors.Is(err, ErrUserExists) {
		t.Errorf("error = %v, want %v", err, ErrUserExists)
	}
}

func TestReadingWhetherAnEmailAddressIsTaken(t *testing.T) {
	// The pre-check the registration makes before it encodes a password.
	store, _ := newTestStore(t)
	createUserFor(t, store, "occupied@jdw.com")
	ctx := context.Background()

	taken, err := store.UserExists(ctx, "occupied@jdw.com")
	if err != nil {
		t.Fatalf("UserExists: %v", err)
	}
	if !taken {
		t.Error("an address a user holds was reported free")
	}

	free, err := store.UserExists(ctx, "unoccupied@jdw.com")
	if err != nil {
		t.Fatalf("UserExists: %v", err)
	}
	if free {
		t.Error("an address nobody holds was reported taken")
	}
}

func TestReadingAUserThatIsNotThere(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()

	if _, err := store.UserByID(ctx, 987654); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByID error = %v, want %v", err, ErrUserNotFound)
	}
	if _, err := store.UserByEmailAddress(ctx, "nobody@jdw.com"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("UserByEmailAddress error = %v, want %v", err, ErrUserNotFound)
	}
	if _, err := store.CredentialByEmailAddress(ctx, "nobody@jdw.com"); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("CredentialByEmailAddress error = %v, want %v", err, ErrUserNotFound)
	}
}

func TestACredentialCarriesTheClaimsATokenIsMintedFrom(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "credential@jdw.com")
	adminRole := roleIDByName(t, store, "ADMIN")
	if _, err := store.GrantRolesToUser(ctx, user.ID, []int64{adminRole}, user.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}

	credential, err := store.CredentialByEmailAddress(ctx, "credential@jdw.com")

	if err != nil {
		t.Fatalf("CredentialByEmailAddress: %v", err)
	}
	if credential.UserID != user.ID {
		t.Errorf("userId = %d, want %d", credential.UserID, user.ID)
	}
	if !passwordMatches(credential.PasswordHash, fixturePassword) {
		t.Error("the stored hash does not verify the password it was written from")
	}
	if len(credential.Roles) != 1 || credential.Roles[0] != "ADMIN" {
		t.Errorf("roles = %v, want the granted role's name", credential.Roles)
	}
	if credential.ProfileID != nil {
		t.Errorf("profileId = %v, want null before a profile exists", credential.ProfileID)
	}

	profileID := seedProfile(t, pool, user.ID)
	withProfile, err := store.CredentialByEmailAddress(ctx, "credential@jdw.com")
	if err != nil {
		t.Fatalf("CredentialByEmailAddress: %v", err)
	}
	if withProfile.ProfileID == nil || *withProfile.ProfileID != profileID {
		t.Errorf("profileId = %v, want %d once the profile exists", withProfile.ProfileID, profileID)
	}
}

func TestUpdatingAUser(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "update@jdw.com")
	actor := createUserFor(t, store, "update-actor@jdw.com")

	updated, err := store.UpdateUser(ctx, user.ID, "renamed@jdw.com", mustHash("Different1!"), actor.ID)

	if err != nil {
		t.Fatalf("UpdateUser: %v", err)
	}
	if updated.EmailAddress != "renamed@jdw.com" {
		t.Errorf("emailAddress = %q, want the new one", updated.EmailAddress)
	}
	if updated.ModifiedByUserID != actor.ID {
		t.Errorf("modifiedByUserId = %d, want the acting user %d", updated.ModifiedByUserID, actor.ID)
	}
	if updated.CreatedByUserID != user.CreatedByUserID {
		t.Errorf("createdByUserId = %d, want the original %d", updated.CreatedByUserID, user.CreatedByUserID)
	}
	if !updated.CreatedTime.Equal(user.CreatedTime.Time) {
		t.Errorf("createdTime = %v, want the original %v", updated.CreatedTime, user.CreatedTime)
	}

	credential, err := store.CredentialByEmailAddress(ctx, "renamed@jdw.com")
	if err != nil {
		t.Fatalf("CredentialByEmailAddress: %v", err)
	}
	if !passwordMatches(credential.PasswordHash, "Different1!") {
		t.Error("the password was not rewritten; every update rewrites the hash")
	}
}

func TestUpdatingAUserOntoAnEmailAddressAnotherUserHolds(t *testing.T) {
	// A unique-constraint violation that no handler maps, which is why the
	// operation answers 500 rather than the 409 the create gives.
	store, _ := newTestStore(t)
	held := createUserFor(t, store, "held@jdw.com")
	mover := createUserFor(t, store, "mover@jdw.com")

	_, err := store.UpdateUser(context.Background(), mover.ID, held.EmailAddress, mustHash(fixturePassword), 1)

	if !errors.Is(err, ErrNameTaken) {
		t.Errorf("error = %v, want %v", err, ErrNameTaken)
	}
}

func TestUpdatingAUserThatIsNotThere(t *testing.T) {
	store, _ := newTestStore(t)

	_, err := store.UpdateUser(context.Background(), 987654, "ghost@jdw.com", mustHash(fixturePassword), 1)

	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want %v", err, ErrUserNotFound)
	}
}

// seedProfile writes the profile row the identity half only ever reads, plus an
// address and an icon under it, so the delete cascade has something to clear.
func seedProfile(t *testing.T, pool *pgxpool.Pool, userID int64) int64 {
	t.Helper()
	var profileID int64
	err := pool.QueryRow(context.Background(), `
		INSERT INTO auth.profiles (user_id, first_name, last_name, birthdate,
			created_by_user_id, created_time, modified_by_user_id, modified_time)
		VALUES ($1, 'Ada', 'Lovelace', DATE '1815-12-10', $1, now(), $1, now())
		RETURNING profile_id`, userID).Scan(&profileID)
	if err != nil {
		t.Fatalf("seed profile for user %d: %v", userID, err)
	}
	return profileID
}

func seedProfileSubresources(t *testing.T, pool *pgxpool.Pool, profileID, userID int64) {
	t.Helper()
	ctx := context.Background()
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.addresses (profile_id, address_line_1, city, state_province, postal_code, country,
			created_by_user_id, created_time, modified_by_user_id, modified_time)
		VALUES ($1, '12 Noel Street', 'London', 'Greater London', 'W1F 8GQ', 'GB', $2, now(), $2, now())`,
		profileID, userID); err != nil {
		t.Fatalf("seed address: %v", err)
	}
	if _, err := pool.Exec(ctx, `
		INSERT INTO auth.profile_icons (profile_id, icon, created_by_user_id, created_time,
			modified_by_user_id, modified_time)
		VALUES ($1, '\x89504e47', $2, now(), $2, now())`, profileID, userID); err != nil {
		t.Fatalf("seed icon: %v", err)
	}
}

func TestDeletingAUserTakesItsGrantsAndItsWholeProfileTreeWithIt(t *testing.T) {
	// The one write that crosses the service boundary. No foreign key in the
	// schema cascades, so the order the store deletes in is what makes this
	// possible at all — a parent removed before its children fails the
	// constraint and the whole transaction with it.
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "delete-cascade@jdw.com")
	profileID := seedProfile(t, pool, user.ID)
	seedProfileSubresources(t, pool, profileID, user.ID)
	if _, err := store.GrantRolesToUser(ctx, user.ID, []int64{roleIDByName(t, store, "USER")}, user.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}

	if err := store.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	if _, err := store.UserByID(ctx, user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("the user survived the delete: %v", err)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE profile_id = $1", profileID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profile_icons WHERE profile_id = $1", profileID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profiles WHERE user_id = $1", user.ID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.users_roles WHERE user_id = $1", user.ID, 0)
}

func TestDeletingAUserWithMoreThanOneProfileRowStillSucceeds(t *testing.T) {
	// One profile per user is an application rule the sibling service enforces
	// with a read before the insert, and auth.profiles.user_id carries a plain
	// index rather than a unique one — so a lost race leaves two rows. Clearing
	// only the first would break the profile delete's foreign key and leave the
	// user undeletable behind a 500 from then on.
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "two-profiles@jdw.com")
	for range 2 {
		profileID := seedProfile(t, pool, user.ID)
		seedProfileSubresources(t, pool, profileID, user.ID)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profiles WHERE user_id = $1", user.ID, 2)

	if err := store.DeleteUser(ctx, user.ID); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}

	assertRowCount(t, pool, "SELECT count(*) FROM auth.profiles WHERE user_id = $1", user.ID, 0)
	if _, err := store.UserByID(ctx, user.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("the user survived the delete: %v", err)
	}
}

func TestDeletingAUserThatIsNotThereIsNotAnError(t *testing.T) {
	// The handler issues the deletes without checking for the row first, which
	// is why the operation has no 404 in its response set at all.
	store, _ := newTestStore(t)

	if err := store.DeleteUser(context.Background(), 987654); err != nil {
		t.Errorf("DeleteUser on a missing id = %v, want no error", err)
	}
}

func TestGrantingAndRevokingRolesFromTheUserSide(t *testing.T) {
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "grants@jdw.com")
	manager := roleIDByName(t, store, "MANAGER")
	ordinary := roleIDByName(t, store, "USER")

	granted, err := store.GrantRolesToUser(ctx, user.ID, []int64{manager, ordinary}, user.ID)

	if err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}
	if len(granted.Roles) != 2 {
		t.Fatalf("roles = %v, want both", granted.Roles)
	}
	for i := 1; i < len(granted.Roles); i++ {
		if granted.Roles[i-1].RoleID >= granted.Roles[i].RoleID {
			t.Errorf("grants are not ordered by role id: %d then %d",
				granted.Roles[i-1].RoleID, granted.Roles[i].RoleID)
		}
	}

	// Granting again is a no-op rather than a duplicate-key failure: the JVM
	// reads before each insert, and the conflict clause does the same job
	// without the race.
	if _, err := store.GrantRolesToUser(ctx, user.ID, []int64{manager}, user.ID); err != nil {
		t.Errorf("a repeated grant = %v, want no error", err)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.users_roles WHERE user_id = $1", user.ID, 2)

	revoked, err := store.RevokeRolesFromUser(ctx, user.ID, []int64{manager}, user.ID)
	if err != nil {
		t.Fatalf("RevokeRolesFromUser: %v", err)
	}
	if len(revoked.Roles) != 1 || revoked.Roles[0].RoleID != ordinary {
		t.Errorf("roles = %v, want only the one that was not revoked", revoked.Roles)
	}
	if _, err := store.RevokeRolesFromUser(ctx, user.ID, []int64{manager}, user.ID); err != nil {
		t.Errorf("a repeated revoke = %v, want no error", err)
	}
}

func TestGrantingReportsWhichIdWasMissing(t *testing.T) {
	// The message the caller reads names the offending id, and a grant can name
	// several, so the id has to survive the error rather than be re-derived.
	store, _ := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "missing-ids@jdw.com")
	ordinary := roleIDByName(t, store, "USER")

	_, err := store.GrantRolesToUser(ctx, user.ID, []int64{ordinary, 987654}, user.ID)
	if !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("error = %v, want %v", err, ErrRoleNotFound)
	}
	if got := missingID(err, 0); got != 987654 {
		t.Errorf("missing id = %d, want the role that is not there", got)
	}

	_, err = store.GrantRolesToUser(ctx, 987654, []int64{ordinary}, user.ID)
	if !errors.Is(err, ErrUserNotFound) {
		t.Errorf("error = %v, want %v", err, ErrUserNotFound)
	}
	if got := missingID(err, 0); got != 987654 {
		t.Errorf("missing id = %d, want the user that is not there", got)
	}
}

func TestAFailedGrantLeavesNoPartialWrite(t *testing.T) {
	// Every id is checked before any is written, so a request naming one good
	// role and one that does not exist grants neither.
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "atomic-grant@jdw.com")

	if _, err := store.GrantRolesToUser(ctx, user.ID,
		[]int64{roleIDByName(t, store, "USER"), 987654}, user.ID); err == nil {
		t.Fatal("the grant succeeded with an id that does not exist")
	}

	assertRowCount(t, pool, "SELECT count(*) FROM auth.users_roles WHERE user_id = $1", user.ID, 0)
}

func TestReadingWhetherAUserHoldsARole(t *testing.T) {
	// What the elevated-role guard consults. It reads the grant table rather
	// than the caller's token, because the guard is written in terms of ids.
	store, _ := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "has-role@jdw.com")
	adminRole := roleIDByName(t, store, "ADMIN")

	holds, err := store.UserHasAnyRole(ctx, user.ID, []int64{adminRole})
	if err != nil {
		t.Fatalf("UserHasAnyRole: %v", err)
	}
	if holds {
		t.Error("a user with no grants was reported as holding one")
	}

	if _, err := store.GrantRolesToUser(ctx, user.ID, []int64{adminRole}, user.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}
	if holds, err = store.UserHasAnyRole(ctx, user.ID, []int64{adminRole}); err != nil || !holds {
		t.Errorf("holds = %v, err = %v; want the granted role to be seen", holds, err)
	}
}

func TestGrantingAndRevokingUsersFromTheRoleSide(t *testing.T) {
	// The same table written from the other direction. That two-writer table is
	// why users and roles stay in one service.
	store, _ := newTestStore(t)
	ctx := context.Background()
	first := createUserFor(t, store, "role-side-a@jdw.com")
	second := createUserFor(t, store, "role-side-b@jdw.com")
	role, err := store.CreateRole(ctx, "ROLE-SIDE", "Granted from the role side.", first.ID)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	granted, err := store.GrantUsersToRole(ctx, role.ID, []int64{first.ID, second.ID}, first.ID)

	if err != nil {
		t.Fatalf("GrantUsersToRole: %v", err)
	}
	if len(granted.Users) != 2 {
		t.Fatalf("users = %v, want both", granted.Users)
	}

	revoked, err := store.RevokeUsersFromRole(ctx, role.ID, []int64{first.ID}, first.ID)
	if err != nil {
		t.Fatalf("RevokeUsersFromRole: %v", err)
	}
	if len(revoked.Users) != 1 || revoked.Users[0].UserID != second.ID {
		t.Errorf("users = %v, want only the one that was not revoked", revoked.Users)
	}

	if _, err := store.GrantUsersToRole(ctx, 987654, []int64{first.ID}, first.ID); !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("granting to a missing role = %v, want %v", err, ErrRoleNotFound)
	}
	if _, err := store.GrantUsersToRole(ctx, role.ID, []int64{987654}, first.ID); !errors.Is(err, ErrUserNotFound) {
		t.Errorf("granting a missing user = %v, want %v", err, ErrUserNotFound)
	}
}

func TestCreatingARole(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	actor := createUserFor(t, store, "role-create@jdw.com")

	role, err := store.CreateRole(ctx, "AUDITOR", "Reads everything, writes nothing.", actor.ID)

	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if role.ID == 0 {
		t.Error("the created role carries no id")
	}
	if role.Name != "AUDITOR" || role.Description != "Reads everything, writes nothing." {
		t.Errorf("role = %+v, want the one supplied", role)
	}
	if role.Status != statusActive {
		t.Errorf("status = %q, want %q", role.Status, statusActive)
	}
	if role.Users == nil || len(role.Users) != 0 {
		t.Errorf("users = %v, want an empty set", role.Users)
	}

	if _, err := store.CreateRole(ctx, "AUDITOR", "A second one.", actor.ID); !errors.Is(err, ErrRoleExists) {
		t.Errorf("a duplicate name = %v, want %v", err, ErrRoleExists)
	}
}

func TestUpdatingARoleOntoANameAnotherRoleHolds(t *testing.T) {
	// The update has no pre-check on the name, unlike the create. The asymmetry
	// is the JVM's and is frozen: a rename onto a taken name is a 500, not a 409.
	store, _ := newTestStore(t)
	ctx := context.Background()
	actor := createUserFor(t, store, "role-rename@jdw.com")
	if _, err := store.CreateRole(ctx, "HELD", "Holds the name.", actor.ID); err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	mover, err := store.CreateRole(ctx, "MOVER", "Wants the name.", actor.ID)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}

	if _, err := store.UpdateRole(ctx, mover.ID, "HELD", "Wants the name.", actor.ID); !errors.Is(err, ErrNameTaken) {
		t.Errorf("error = %v, want %v", err, ErrNameTaken)
	}

	renamed, err := store.UpdateRole(ctx, mover.ID, "MOVED", "Got a free name.", actor.ID)
	if err != nil {
		t.Fatalf("UpdateRole: %v", err)
	}
	if renamed.Name != "MOVED" {
		t.Errorf("name = %q, want the new one", renamed.Name)
	}
	if _, err := store.UpdateRole(ctx, 987654, "GHOST", "Nothing to update.", actor.ID); !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("updating a missing role = %v, want %v", err, ErrRoleNotFound)
	}
}

func TestDeletingARoleClearsItsGrantsFirst(t *testing.T) {
	// auth.users_roles references auth.roles with no cascade, so the order is
	// what makes the delete possible at all.
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "role-delete@jdw.com")
	role, err := store.CreateRole(ctx, "TEMPORARY", "Will not last.", user.ID)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := store.GrantUsersToRole(ctx, role.ID, []int64{user.ID}, user.ID); err != nil {
		t.Fatalf("GrantUsersToRole: %v", err)
	}

	if err := store.DeleteRole(ctx, role.ID); err != nil {
		t.Fatalf("DeleteRole: %v", err)
	}

	if _, err := store.RoleByID(ctx, role.ID); !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("the role survived the delete: %v", err)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.users_roles WHERE role_id = $1", role.ID, 0)
	if err := store.DeleteRole(ctx, 987654); err != nil {
		t.Errorf("deleting a role that is not there = %v, want no error", err)
	}
}

func TestListingUsersIsPagedAndOrderedById(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	created := make([]int64, 0, 3)
	for _, email := range []string{"list-1@jdw.com", "list-2@jdw.com", "list-3@jdw.com"} {
		created = append(created, createUserFor(t, store, email).ID)
	}

	all, err := store.ListUsers(ctx, 500, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatalf("users are not ordered by id: %d then %d", all[i-1].ID, all[i].ID)
		}
	}

	// A page of one, offset onto the second user this test created.
	position := 0
	for i, user := range all {
		if user.ID == created[0] {
			position = i
			break
		}
	}
	page, err := store.ListUsers(ctx, 1, position+1)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(page) != 1 || page[0].ID != created[1] {
		t.Errorf("page = %v, want the user with id %d alone", page, created[1])
	}
}

func TestAListedUserCarriesItsGrantsAndItsProfileId(t *testing.T) {
	// UserRepositoryImpl issues four more queries per row. The set-based read
	// here has to produce the same payload without that fan-out.
	store, pool := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "list-aggregate@jdw.com")
	profileID := seedProfile(t, pool, user.ID)
	if _, err := store.GrantRolesToUser(ctx, user.ID, []int64{roleIDByName(t, store, "USER")}, user.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}

	all, err := store.ListUsers(ctx, 500, 0)
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}

	var listed *User
	for i := range all {
		if all[i].ID == user.ID {
			listed = &all[i]
			break
		}
	}
	if listed == nil {
		t.Fatalf("the user with id %d is not in the listing", user.ID)
	}
	if len(listed.Roles) != 1 {
		t.Errorf("roles = %v, want one", listed.Roles)
	}
	if listed.ProfileID == nil || *listed.ProfileID != profileID {
		t.Errorf("profileId = %v, want %d", listed.ProfileID, profileID)
	}
}

func TestListingRolesIsPagedAndOrderedById(t *testing.T) {
	// Paginated here and unpaginated in the JVM. The ordering is part of the
	// change: pagination without a total order returns overlapping pages.
	store, _ := newTestStore(t)
	ctx := context.Background()

	all, err := store.ListRoles(ctx, 500, 0)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(all) < 3 {
		t.Fatalf("roles = %v, want at least the three the seed data holds", all)
	}
	for i := 1; i < len(all); i++ {
		if all[i-1].ID >= all[i].ID {
			t.Fatalf("roles are not ordered by id: %d then %d", all[i-1].ID, all[i].ID)
		}
	}

	page, err := store.ListRoles(ctx, 1, 1)
	if err != nil {
		t.Fatalf("ListRoles: %v", err)
	}
	if len(page) != 1 || page[0].ID != all[1].ID {
		t.Errorf("page = %v, want the second role alone", page)
	}
}

func TestAListedRoleCarriesItsGrants(t *testing.T) {
	store, _ := newTestStore(t)
	ctx := context.Background()
	user := createUserFor(t, store, "role-grants@jdw.com")
	role, err := store.CreateRole(ctx, "LISTED", "Carries its grants.", user.ID)
	if err != nil {
		t.Fatalf("CreateRole: %v", err)
	}
	if _, err := store.GrantUsersToRole(ctx, role.ID, []int64{user.ID}, user.ID); err != nil {
		t.Fatalf("GrantUsersToRole: %v", err)
	}

	read, err := store.RoleByName(ctx, "LISTED")

	if err != nil {
		t.Fatalf("RoleByName: %v", err)
	}
	if len(read.Users) != 1 || read.Users[0].UserID != user.ID {
		t.Errorf("users = %v, want the one grant", read.Users)
	}
	if _, err := store.RoleByName(ctx, "NO-SUCH-ROLE"); !errors.Is(err, ErrRoleNotFound) {
		t.Errorf("RoleByName for a missing name = %v, want %v", err, ErrRoleNotFound)
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
