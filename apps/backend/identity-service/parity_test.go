package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authtest"
	"libs/backend/shared/auth/authz"
)

// A published test key, not a credential: it signs nothing outside this suite.
const paritySecret = "dGVzdC1zZWNyZXQtd2l0aC10aGlydHktdHdvLWJ5dGVzISE=" // gitleaks:allow

const parityIssuerOrigin = "http://localhost:8080"

// The fixture the whole suite is scoped to: four users and two roles, with the
// elevated role holding the id the service's guard names.
const (
	selfUserID       = int64(1)
	adminUserID      = int64(2)
	managerUserID    = int64(3)
	strangerUserID   = int64(4)
	ordinaryRoleID   = int64(2)
	selfEmail        = "self@jdw.com"
	adminEmail       = "admin@jdw.com"
	strangerEmail    = "stranger@jdw.com"
	fixturePassword  = "Password1!" // gitleaks:allow
	fixtureProfileID = int64(11)
)

// fixturePasswordHash is encoded once: bcrypt is deliberately slow, and the
// parity suite drives the sign-in operation dozens of times.
var fixturePasswordHash = mustHash(fixturePassword)

func mustHash(password string) string {
	hash, err := hashPassword(password)
	if err != nil {
		panic("the fixture password would not hash: " + err.Error())
	}
	return hash
}

// stubStore answers every read with the fixture rows and every write with
// success, so an operation's status depends on nothing but its authorization
// outcome. Storage behaviour is covered against a real Postgres in the
// integration suites.
type stubStore struct{}

func fixtureUser() User {
	return User{
		ID: selfUserID, EmailAddress: selfEmail, Status: statusActive,
		Roles:       []UserRole{{UserID: selfUserID, RoleID: ordinaryRoleID}},
		ProfileID:   ptr(fixtureProfileID),
		CreatedTime: Timestamp{time.Unix(0, 0).UTC()}, ModifiedTime: Timestamp{time.Unix(0, 0).UTC()},
	}
}

func fixtureRole() Role {
	return Role{
		ID: ordinaryRoleID, Name: "MANAGER", Description: "Manager role with limited access.",
		Status: statusActive, Users: []UserRole{{UserID: selfUserID, RoleID: ordinaryRoleID}},
	}
}

func (stubStore) ListUsers(context.Context, int, int) ([]User, error) {
	return []User{fixtureUser()}, nil
}
func (stubStore) UserByID(context.Context, int64) (User, error) { return fixtureUser(), nil }
func (stubStore) UserByEmailAddress(context.Context, string) (User, error) {
	return fixtureUser(), nil
}

func (stubStore) CredentialByEmailAddress(_ context.Context, emailAddress string) (Credential, error) {
	if emailAddress != selfEmail {
		return Credential{}, ErrUserNotFound
	}
	return Credential{
		UserID: selfUserID, EmailAddress: selfEmail, PasswordHash: fixturePasswordHash,
		Roles: []string{"USER"}, ProfileID: ptr(fixtureProfileID),
	}, nil
}

func (stubStore) CreateUser(context.Context, string, string, int64) (User, error) {
	return fixtureUser(), nil
}
func (stubStore) UpdateUser(context.Context, int64, string, string, int64) (User, error) {
	return fixtureUser(), nil
}
func (stubStore) DeleteUser(context.Context, int64) error { return nil }
func (stubStore) UserHasAnyRole(_ context.Context, userID int64, _ []int64) (bool, error) {
	return userID == adminUserID, nil
}
func (stubStore) GrantRolesToUser(context.Context, int64, []int64, int64) (User, error) {
	return fixtureUser(), nil
}
func (stubStore) RevokeRolesFromUser(context.Context, int64, []int64, int64) (User, error) {
	return fixtureUser(), nil
}
func (stubStore) ListRoles(context.Context, int, int) ([]Role, error) {
	return []Role{fixtureRole()}, nil
}
func (stubStore) RoleByID(context.Context, int64) (Role, error)    { return fixtureRole(), nil }
func (stubStore) RoleByName(context.Context, string) (Role, error) { return fixtureRole(), nil }
func (stubStore) CreateRole(context.Context, string, string, int64) (Role, error) {
	return fixtureRole(), nil
}
func (stubStore) UpdateRole(context.Context, int64, string, string, int64) (Role, error) {
	return fixtureRole(), nil
}
func (stubStore) DeleteRole(context.Context, int64) error { return nil }
func (stubStore) GrantUsersToRole(context.Context, int64, []int64, int64) (Role, error) {
	return fixtureRole(), nil
}
func (stubStore) RevokeUsersFromRole(context.Context, int64, []int64, int64) (Role, error) {
	return fixtureRole(), nil
}

func parityVerifier(t *testing.T) *auth.Verifier {
	t.Helper()
	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecret,
		ExpectedIssuer:   parityIssuerOrigin + "/auth/authenticate",
		ExpectedAudience: parityIssuerOrigin,
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return verifier
}

func parityMinter(t *testing.T) *minter {
	t.Helper()
	made, err := newMinter(paritySecret, parityIssuerOrigin, defaultTokenTTL)
	if err != nil {
		t.Fatalf("newMinter: %v", err)
	}
	return made
}

func parityServer(t *testing.T, store Store) http.Handler {
	t.Helper()
	server, err := NewServer(ServerConfig{
		Store:    store,
		Verifier: parityVerifier(t),
		Minter:   parityMinter(t),
		CORS:     springShapedCORS(),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return server.Handler()
}

func mint(t *testing.T, claims authtest.Claims) string {
	t.Helper()
	token, err := authtest.Minter{SecretKeyBase64: paritySecret, IssuerOrigin: parityIssuerOrigin}.Mint(claims)
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	return token
}

// principal names the token each authorization outcome needs, in terms a
// contract rule can be read against.
type principal struct {
	name   string
	claims authtest.Claims
}

func admin() principal {
	return principal{
		name: "an ADMIN",
		claims: authtest.Claims{
			Subject: adminEmail, Roles: []string{"ADMIN"}, UserID: ptr(adminUserID),
		},
	}
}

func manager() principal {
	return principal{
		name: "a MANAGER",
		claims: authtest.Claims{
			Subject: "manager@jdw.com", Roles: []string{"MANAGER"}, UserID: ptr(managerUserID),
		},
	}
}

func self() principal {
	return principal{
		name: "the subject of the record",
		claims: authtest.Claims{
			Subject: selfEmail, Roles: []string{"USER"}, UserID: ptr(selfUserID),
		},
	}
}

func stranger() principal {
	return principal{
		name: "a stranger",
		claims: authtest.Claims{
			Subject: strangerEmail, Roles: []string{"USER"}, UserID: ptr(strangerUserID),
		},
	}
}

// allowedBy and deniedBy turn a contract rule into the principals that must
// pass and fail it, so a rule change in the contract changes which tokens this
// suite drives rather than being silently untested.
func allowedBy(rule authz.Rule) []principal {
	switch rule {
	case authz.RulePublic:
		// Named for the assertion it drives: a public operation is reached with
		// no Authorization header at all, which is what the empty claims give.
		return []principal{{name: "an anonymous caller"}}
	case authz.RuleAuthenticated:
		return []principal{admin(), manager(), self(), stranger()}
	case authz.RuleAdmin:
		return []principal{admin()}
	case authz.RuleAdminOrManager:
		return []principal{admin(), manager()}
	case authz.RuleAdminOrSelfByUserID, authz.RuleAdminOrSelfByEmail:
		return []principal{admin(), self()}
	default:
		return nil
	}
}

func deniedBy(rule authz.Rule) []principal {
	switch rule {
	case authz.RuleAdmin:
		return []principal{manager(), self(), stranger()}
	case authz.RuleAdminOrManager:
		return []principal{self(), stranger()}
	case authz.RuleAdminOrSelfByUserID, authz.RuleAdminOrSelfByEmail:
		return []principal{manager(), stranger()}
	default:
		// PUBLIC and AUTHENTICATED discriminate between no principals at all.
		// Which rules may answer nil here is asserted in the contract suite, so
		// a new rule cannot join them by accident.
		return nil
	}
}

// undiscriminatingRules are the two rules that refuse no verified principal.
// They are listed rather than inferred so that adding a rule with no deny case
// fails the contract suite instead of passing it vacuously.
var undiscriminatingRules = map[authz.Rule]bool{
	authz.RulePublic:        true,
	authz.RuleAuthenticated: true,
}

// parityCase is one operation, with the request that exercises it and the
// status it answers to a caller the rule allows.
type parityCase struct {
	operation     string
	method        string
	path          string
	rule          authz.Rule
	successStatus int
	body          func() (payload []byte, contentType string)
}

func jsonBody(text string) func() ([]byte, string) {
	return func() ([]byte, string) { return []byte(text), "application/json" }
}

const validCredentialsBody = `{"emailAddress":"` + selfEmail + `","password":"` + fixturePassword + `"}`

const validUserBody = `{"emailAddress":"new@jdw.com","password":"` + fixturePassword + `"}`

const validRoleBody = `{"name":"AUDITOR","description":"Reads everything, writes nothing."}`

// ordinaryRoleIDList and ordinaryUserIDList stay clear of the elevated role, so
// these cases measure the operation's own rule rather than the extra guard the
// service applies on top of it. That guard has its own tests below.
const ordinaryRoleIDList = `[2]`

const ordinaryUserIDList = `[1]`

// parityCases is the whole served surface, one entry per operation. The
// contract suite asserts that these are exactly the operations the frozen
// document describes and that each rule here is the rule it names.
func parityCases() []parityCase {
	return []parityCase{
		{
			operation: "authenticate", method: http.MethodPost, path: "/auth/authenticate",
			rule: authz.RulePublic, successStatus: http.StatusOK, body: jsonBody(validCredentialsBody),
		},
		{
			operation: "registerUser", method: http.MethodPost, path: "/auth/user",
			rule: authz.RulePublic, successStatus: http.StatusCreated, body: jsonBody(validUserBody),
		},
		{
			operation: "getAllUsers", method: http.MethodGet, path: "/api/users",
			rule: authz.RuleAdmin, successStatus: http.StatusOK,
		},
		{
			operation: "createUser", method: http.MethodPost, path: "/api/users",
			rule: authz.RuleAuthenticated, successStatus: http.StatusCreated, body: jsonBody(validUserBody),
		},
		{
			operation: "getUserById", method: http.MethodGet, path: "/api/users/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusOK,
		},
		{
			operation: "updateUser", method: http.MethodPut, path: "/api/users/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusOK, body: jsonBody(validUserBody),
		},
		{
			operation: "deleteUser", method: http.MethodDelete, path: "/api/users/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusNoContent,
		},
		{
			operation: "getUserByEmailAddress", method: http.MethodGet, path: "/api/users/email/" + selfEmail,
			rule: authz.RuleAdminOrSelfByEmail, successStatus: http.StatusOK,
		},
		{
			operation: "grantRolesToUser", method: http.MethodPut, path: "/api/users/1/roles/grant",
			rule: authz.RuleAdminOrManager, successStatus: http.StatusOK, body: jsonBody(ordinaryRoleIDList),
		},
		{
			operation: "revokeRolesFromUser", method: http.MethodPut, path: "/api/users/1/roles/revoke",
			rule: authz.RuleAdminOrManager, successStatus: http.StatusOK, body: jsonBody(ordinaryRoleIDList),
		},
		{
			operation: "getAllRoles", method: http.MethodGet, path: "/api/roles",
			rule: authz.RuleAuthenticated, successStatus: http.StatusOK,
		},
		{
			operation: "createRole", method: http.MethodPost, path: "/api/roles",
			rule: authz.RuleAdmin, successStatus: http.StatusCreated, body: jsonBody(validRoleBody),
		},
		{
			operation: "getRoleById", method: http.MethodGet, path: "/api/roles/2",
			rule: authz.RuleAuthenticated, successStatus: http.StatusOK,
		},
		{
			operation: "updateRole", method: http.MethodPut, path: "/api/roles/2",
			rule: authz.RuleAdmin, successStatus: http.StatusOK, body: jsonBody(validRoleBody),
		},
		{
			operation: "deleteRole", method: http.MethodDelete, path: "/api/roles/2",
			rule: authz.RuleAdmin, successStatus: http.StatusNoContent,
		},
		{
			operation: "getRoleByName", method: http.MethodGet, path: "/api/roles/name/MANAGER",
			rule: authz.RuleAuthenticated, successStatus: http.StatusOK,
		},
		{
			operation: "grantUsersToRole", method: http.MethodPut, path: "/api/roles/2/users/grant",
			rule: authz.RuleAdminOrManager, successStatus: http.StatusOK, body: jsonBody(ordinaryUserIDList),
		},
		{
			operation: "revokeUsersFromRole", method: http.MethodPut, path: "/api/roles/2/users/revoke",
			rule: authz.RuleAdminOrManager, successStatus: http.StatusOK, body: jsonBody(ordinaryUserIDList),
		},
	}
}

func (tc parityCase) request(t *testing.T, token string) *http.Request {
	t.Helper()
	var payload []byte
	contentType := ""
	if tc.body != nil {
		payload, contentType = tc.body()
	}
	request := httptest.NewRequest(tc.method, tc.path, bytes.NewReader(payload))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	return request
}

func TestEveryOperationAllowsThePrincipalsItsRuleAllows(t *testing.T) {
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
		for _, who := range allowedBy(tc.rule) {
			t.Run(tc.operation+"/"+who.name, func(t *testing.T) {
				token := ""
				if who.claims.Subject != "" {
					token = mint(t, who.claims)
				}
				response := httptest.NewRecorder()

				server.ServeHTTP(response, tc.request(t, token))

				if response.Code != tc.successStatus {
					t.Errorf("status = %d, want %d (body %q)",
						response.Code, tc.successStatus, response.Body.String())
				}
			})
		}
	}
}

func TestEveryOperationDeniesThePrincipalsItsRuleDenies(t *testing.T) {
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
		for _, who := range deniedBy(tc.rule) {
			t.Run(tc.operation+"/"+who.name, func(t *testing.T) {
				response := httptest.NewRecorder()

				server.ServeHTTP(response, tc.request(t, mint(t, who.claims)))

				if response.Code != http.StatusForbidden {
					t.Errorf("status = %d, want %d (body %q)",
						response.Code, http.StatusForbidden, response.Body.String())
				}
				assertForbiddenShape(t, response)
			})
		}
	}
}

func TestEveryAuthenticatedOperationRefusesARequestWithNoToken(t *testing.T) {
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
		if tc.rule == authz.RulePublic {
			continue
		}
		t.Run(tc.operation, func(t *testing.T) {
			response := httptest.NewRecorder()

			server.ServeHTTP(response, tc.request(t, ""))

			if response.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
			assertUnauthorizedShape(t, response)
		})
	}
}

func TestEveryPublicOperationIsReachedWithNoToken(t *testing.T) {
	// The half the @PreAuthorize annotations cannot state: PUBLIC and
	// AUTHENTICATED both carry no predicate, and only the filter chain's
	// permitAll matchers separate them. Flipping that matcher list would turn
	// sign-in and registration private with every other check still green.
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
		if tc.rule != authz.RulePublic {
			continue
		}
		t.Run(tc.operation, func(t *testing.T) {
			response := httptest.NewRecorder()

			server.ServeHTTP(response, tc.request(t, ""))

			if response.Code == http.StatusUnauthorized {
				t.Errorf("status = %d; a public operation must be reachable without a token", response.Code)
			}
		})
	}
}

func TestEveryAuthenticatedOperationRefusesATokenThatDoesNotVerify(t *testing.T) {
	server := parityServer(t, stubStore{})
	tampered := authtest.TamperSignature(mint(t, admin().claims))

	for _, tc := range parityCases() {
		if tc.rule == authz.RulePublic {
			continue
		}
		t.Run(tc.operation, func(t *testing.T) {
			response := httptest.NewRecorder()

			server.ServeHTTP(response, tc.request(t, tampered))

			if response.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d", response.Code, http.StatusUnauthorized)
			}
		})
	}
}

// assertUnauthorizedShape pins the measured shape of a 401 from the deployed
// service: the reason header, an empty body and no Content-Type. sendError
// forwards to /error, and with no token that forward is refused again before
// Boot's error controller can render anything.
func assertUnauthorizedShape(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got, want := response.Header().Get("Access-Denied-Reason"), "Authentication Required"; got != want {
		t.Errorf("Access-Denied-Reason = %q, want %q", got, want)
	}
	if body := response.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Content-Type = %q, want unset", contentType)
	}
}

// assertForbiddenShape pins the measured shape of a 403, which is not the shape
// of a 401. The caller here already holds a verified token, so the forward to
// /error that sendError triggers re-authenticates with it and reaches Boot's
// error controller; an unauthenticated caller's forward is refused again and
// renders nothing.
func assertForbiddenShape(t *testing.T, response *httptest.ResponseRecorder) {
	t.Helper()
	if got, want := response.Header().Get("Access-Denied-Reason"), "Not Authorized"; got != want {
		t.Errorf("Access-Denied-Reason = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Content-Type"), contentTypeJSON; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	var body struct {
		Timestamp string  `json:"timestamp"`
		Status    int     `json:"status"`
		Error     string  `json:"error"`
		Path      string  `json:"path"`
		Message   *string `json:"message"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("the 403 body is not the container error representation: %v", err)
	}
	if body.Status != http.StatusForbidden {
		t.Errorf("status field = %d, want %d", body.Status, http.StatusForbidden)
	}
	if body.Error != "Forbidden" {
		t.Errorf("error field = %q, want %q", body.Error, "Forbidden")
	}
	if body.Timestamp == "" {
		t.Error("timestamp field is empty")
	}
	// server.error.include-message is never, so the key is absent rather than
	// present and empty.
	if body.Message != nil {
		t.Errorf("message field = %q, want it absent", *body.Message)
	}
}

func TestTheEmailRuleComparesTheSubjectExactly(t *testing.T) {
	// hasAuthority('ADMIN') or #emailAddress == principal.getUsername() is a
	// case-sensitive string comparison, so a caller authenticated as one casing
	// cannot read another.
	server := parityServer(t, stubStore{})
	token := mint(t, self().claims)
	response := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/api/users/email/Self@jdw.com", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestGrantingTheElevatedRoleNeedsTheCallerToHoldIt(t *testing.T) {
	// The second check, applied after the operation's own rule has passed: a
	// MANAGER may grant roles, but not the one that would make somebody their
	// equal. The JVM raises AccessDeniedException here, which the access-denied
	// handler renders as the same 403 the predicate produces.
	server := parityServer(t, stubStore{})

	for _, tc := range []struct {
		name string
		path string
		body string
	}{
		{name: "granting it to a user", path: "/api/users/1/roles/grant", body: "[1]"},
		{name: "revoking it from a user", path: "/api/users/1/roles/revoke", body: "[1]"},
		{name: "granting users to it", path: "/api/roles/1/users/grant", body: "[1]"},
		{name: "revoking users from it", path: "/api/roles/1/users/revoke", body: "[1]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			refused := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewReader([]byte(tc.body)))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+mint(t, manager().claims))

			server.ServeHTTP(refused, request)

			if refused.Code != http.StatusForbidden {
				t.Errorf("a MANAGER got %d, want %d (body %q)",
					refused.Code, http.StatusForbidden, refused.Body.String())
			}
			assertForbiddenShape(t, refused)

			allowed := httptest.NewRecorder()
			admitted := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewReader([]byte(tc.body)))
			admitted.Header.Set("Content-Type", "application/json")
			admitted.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))

			server.ServeHTTP(allowed, admitted)

			if allowed.Code != http.StatusOK {
				t.Errorf("a holder of the elevated role got %d, want %d (body %q)",
					allowed.Code, http.StatusOK, allowed.Body.String())
			}
		})
	}
}

func TestAnEmptyGrantListIsRefusedRatherThanServed(t *testing.T) {
	// The contract gives both list bodies minItems: 1. The JVM reads the first
	// element of the list it built from them and answers 500 for an empty one.
	server := parityServer(t, stubStore{})

	for _, tc := range []struct {
		path  string
		field string
	}{
		{path: "/api/users/1/roles/grant", field: "roleIds"},
		{path: "/api/users/1/roles/revoke", field: "roleIds"},
		{path: "/api/roles/2/users/grant", field: "userIds"},
		{path: "/api/roles/2/users/revoke", field: "userIds"},
	} {
		t.Run(tc.path, func(t *testing.T) {
			response := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPut, tc.path, bytes.NewReader([]byte("[]")))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))

			server.ServeHTTP(response, request)

			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d (body %q)",
					response.Code, http.StatusBadRequest, response.Body.String())
			}
			var fields map[string]string
			if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
				t.Fatalf("body is not a validation-error object: %v", err)
			}
			if fields[tc.field] == "" {
				t.Errorf("errors = %v, want an entry for %s", fields, tc.field)
			}
		})
	}
}

func TestNoResponseBodyCarriesAPassword(t *testing.T) {
	// The regression this service exists to not repeat: every user response
	// once serialized the bcrypt hash, the public registration one included.
	// Driven over the whole surface rather than over the model, so a handler
	// that composed its own body would be caught too.
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
		t.Run(tc.operation, func(t *testing.T) {
			who := allowedBy(tc.rule)[0]
			token := ""
			if who.claims.Subject != "" {
				token = mint(t, who.claims)
			}
			response := httptest.NewRecorder()

			server.ServeHTTP(response, tc.request(t, token))

			body := response.Body.String()
			for _, forbidden := range []string{"password", "Password", "$2a$", "$2b$", "$2y$", fixturePassword} {
				if bytes.Contains([]byte(body), []byte(forbidden)) {
					t.Errorf("the response body carries %q: %s", forbidden, body)
				}
			}
		})
	}
}

func ptr[T any](value T) *T { return &value }

func TestASignInForAnUnknownAddressStillSpendsAComparison(t *testing.T) {
	// The refusal shapes are identical; without the decoy comparison the two
	// durations would not be, and an anonymous caller could enumerate registered
	// addresses by timing alone. The floor is far below one bcrypt round at the
	// cost this service encodes at and far above the microseconds a bare lookup
	// takes, so a shared runner's noise cannot move it either way.
	const bcryptRoundFloor = 5 * time.Millisecond
	server := parityServer(t, stubStore{})

	shortest := time.Hour
	for range 3 {
		request := httptest.NewRequest(http.MethodPost, "/auth/authenticate",
			bytes.NewReader([]byte(`{"emailAddress":"nobody@jdw.com","password":"`+fixturePassword+`"}`)))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		start := time.Now()
		server.ServeHTTP(response, request)
		if elapsed := time.Since(start); elapsed < shortest {
			shortest = elapsed
		}

		if response.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
		}
	}

	if shortest < bcryptRoundFloor {
		t.Errorf("an unknown address was refused in %s, faster than a password check costs; "+
			"the two answers are distinguishable by timing", shortest)
	}
}
