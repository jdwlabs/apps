package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"libs/backend/shared/auth/authtest"
)

// The end-to-end half of the parity suite: the same handlers, router, CORS,
// authentication and minter as production, over the deployed schema in a
// container. The stub-backed suite pins authorization; this one pins what the
// service actually answers once real rows are involved.

type liveService struct {
	handler http.Handler
	store   *PostgresStore
	admin   User
}

func newLiveService(t *testing.T) *liveService {
	t.Helper()
	store, _ := newTestStore(t)
	server, err := NewServer(ServerConfig{
		Store: store, Verifier: parityVerifier(t), Minter: parityMinter(t), CORS: springShapedCORS(),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	admin := createUserFor(t, store, fmt.Sprintf("admin-%s@jdw.com", t.Name()))
	// Granted in the table as well as claimed in the token. The two are
	// different sources: the operation's own rule reads the claim, and the
	// elevated-role guard reads the table, so an administrator who exists only
	// in the token cannot hand the elevated role out.
	if _, err := store.GrantRolesToUser(t.Context(), admin.ID, []int64{elevatedRoleID}, admin.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}
	return &liveService{handler: server.Handler(), store: store, admin: admin}
}

// adminToken mints for a real row, so the acting user the audit columns record
// exists. The roles come from the claim, as they do in production.
func (s *liveService) adminToken(t *testing.T) string {
	t.Helper()
	return mint(t, authtest.Claims{
		Subject: s.admin.EmailAddress, Roles: []string{"ADMIN"}, UserID: &s.admin.ID,
	})
}

func (s *liveService) do(
	t *testing.T, method, path, token string, body []byte, contentType string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	response := httptest.NewRecorder()
	s.handler.ServeHTTP(response, request)
	return response
}

func (s *liveService) getJSON(t *testing.T, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	return s.do(t, http.MethodGet, path, token, nil, "")
}

func (s *liveService) postJSON(t *testing.T, path, token, body string) *httptest.ResponseRecorder {
	t.Helper()
	return s.do(t, http.MethodPost, path, token, []byte(body), "application/json")
}

func assertStatusAndText(t *testing.T, response *httptest.ResponseRecorder, status int, body string) {
	t.Helper()
	if response.Code != status {
		t.Errorf("status = %d, want %d (body %q)", response.Code, status, response.Body.String())
	}
	if got := response.Body.String(); got != body {
		t.Errorf("body = %q, want %q", got, body)
	}
	if got, want := response.Header().Get("Content-Type"), contentTypeText; body != "" && got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
}

func credentialsBody(email string) string {
	return fmt.Sprintf(`{"emailAddress":%q,"password":%q}`, email, fixturePassword)
}

func TestRegisteringAndThenSigningInProducesATokenTheServiceItselfAccepts(t *testing.T) {
	// The whole point of this service, end to end: a password goes in, a token
	// comes out, and that token is what the authenticated surface accepts. If
	// the minter and the verifier ever disagreed, nothing else would show it.
	service := newLiveService(t)
	email := "round-trip@jdw.com"

	registered := service.postJSON(t, "/auth/user", "", credentialsBody(email))
	if registered.Code != http.StatusCreated {
		t.Fatalf("registration = %d %s", registered.Code, registered.Body.String())
	}
	var created User
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("the registration body is not a user: %v", err)
	}

	signedIn := service.postJSON(t, "/auth/authenticate", "", credentialsBody(email))
	if signedIn.Code != http.StatusOK {
		t.Fatalf("sign-in = %d %s", signedIn.Code, signedIn.Body.String())
	}
	if got, want := signedIn.Header().Get("Content-Type"), contentTypeJSON; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	var issued AuthResponse
	if err := json.Unmarshal(signedIn.Body.Bytes(), &issued); err != nil {
		t.Fatalf("the sign-in body is not an auth response: %v", err)
	}
	if issued.JWTToken == "" {
		t.Fatal("the sign-in answered with no token")
	}

	own := service.getJSON(t, fmt.Sprintf("/api/users/%d", created.ID), issued.JWTToken)

	if own.Code != http.StatusOK {
		t.Errorf("reading their own record with the issued token = %d %s", own.Code, own.Body.String())
	}
	// The rule that admitted them is the self check, so the claim the minter
	// wrote is the one the verifier read back.
	other := service.getJSON(t, fmt.Sprintf("/api/users/%d", service.admin.ID), issued.JWTToken)
	if other.Code != http.StatusForbidden {
		t.Errorf("reading somebody else's record = %d, want %d", other.Code, http.StatusForbidden)
	}
}

func TestAnIssuedTokenCarriesTheGrantsTheDatabaseHolds(t *testing.T) {
	// Role decisions move from a per-request database read to the token. What
	// has to hold is that the token is minted from the grants at sign-in time.
	service := newLiveService(t)
	email := "granted@jdw.com"
	registered := service.postJSON(t, "/auth/user", "", credentialsBody(email))
	if registered.Code != http.StatusCreated {
		t.Fatalf("registration = %d %s", registered.Code, registered.Body.String())
	}
	var created User
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("body: %v", err)
	}

	before := service.postJSON(t, "/auth/authenticate", "", credentialsBody(email))
	var withoutGrant AuthResponse
	if err := json.Unmarshal(before.Body.Bytes(), &withoutGrant); err != nil {
		t.Fatalf("body: %v", err)
	}
	if listed := service.getJSON(t, "/api/users", withoutGrant.JWTToken); listed.Code != http.StatusForbidden {
		t.Errorf("an ungranted user listing users = %d, want %d", listed.Code, http.StatusForbidden)
	}

	grant := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/users/%d/roles/grant", created.ID), service.adminToken(t),
		[]byte(fmt.Sprintf("[%d]", roleIDByName(t, service.store, "ADMIN"))), "application/json")
	if grant.Code != http.StatusOK {
		t.Fatalf("granting = %d %s", grant.Code, grant.Body.String())
	}

	after := service.postJSON(t, "/auth/authenticate", "", credentialsBody(email))
	var withGrant AuthResponse
	if err := json.Unmarshal(after.Body.Bytes(), &withGrant); err != nil {
		t.Fatalf("body: %v", err)
	}

	if listed := service.getJSON(t, "/api/users", withGrant.JWTToken); listed.Code != http.StatusOK {
		t.Errorf("a granted ADMIN listing users = %d %s", listed.Code, listed.Body.String())
	}
	// The token minted before the grant still carries the old authority, which
	// is the revocation latency the split introduces and the contract records.
	if stale := service.getJSON(t, "/api/users", withoutGrant.JWTToken); stale.Code != http.StatusForbidden {
		t.Errorf("the pre-grant token = %d, want %d; authority comes from the token, not a fresh read",
			stale.Code, http.StatusForbidden)
	}
}

func TestSigningInWithTheWrongPasswordAndWithAnUnknownAddressAnswerAlike(t *testing.T) {
	// Answering them differently would let an anonymous caller enumerate which
	// addresses are registered, one request at a time.
	service := newLiveService(t)
	email := "enumerate@jdw.com"
	if registered := service.postJSON(t, "/auth/user", "", credentialsBody(email)); registered.Code != http.StatusCreated {
		t.Fatalf("registration = %d %s", registered.Code, registered.Body.String())
	}

	wrongPassword := service.postJSON(t, "/auth/authenticate", "",
		fmt.Sprintf(`{"emailAddress":%q,"password":"Different1!"}`, email))
	unknownAddress := service.postJSON(t, "/auth/authenticate", "", credentialsBody("nobody@jdw.com"))

	for name, response := range map[string]*httptest.ResponseRecorder{
		"a wrong password": wrongPassword, "an unknown address": unknownAddress,
	} {
		t.Run(name, func(t *testing.T) {
			if response.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want %d (body %q)",
					response.Code, http.StatusUnauthorized, response.Body.String())
			}
			assertUnauthorizedShape(t, response)
		})
	}
}

func TestASignInBodyThatFailsTheFormatRulesIsRefusedBeforeTheCredentialsAreRead(t *testing.T) {
	// @Valid runs during argument resolution, ahead of anything the handler
	// does, so a password that does not meet the policy is a 400 whether or not
	// it happens to be the right one.
	service := newLiveService(t)

	response := service.postJSON(t, "/auth/authenticate", "",
		`{"emailAddress":"someone@jdw.com","password":"short"}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d (body %q)", response.Code, http.StatusBadRequest, response.Body.String())
	}
	var fields map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("body is not a validation-error object: %v", err)
	}
	if fields["password"] != passwordRequirements {
		t.Errorf("password message = %q, want the constraint's own text", fields["password"])
	}
}

func TestNoResponseOnTheLiveSurfaceCarriesTheStoredHash(t *testing.T) {
	// Against real rows rather than a stub, because the hash only exists here:
	// the stub has one, the database has the one that matters.
	service := newLiveService(t)
	email := "hash-leak@jdw.com"
	registered := service.postJSON(t, "/auth/user", "", credentialsBody(email))
	if registered.Code != http.StatusCreated {
		t.Fatalf("registration = %d %s", registered.Code, registered.Body.String())
	}
	var created User
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("body: %v", err)
	}
	token := service.adminToken(t)

	for _, response := range []*httptest.ResponseRecorder{
		registered,
		service.getJSON(t, "/api/users", token),
		service.getJSON(t, fmt.Sprintf("/api/users/%d", created.ID), token),
		service.getJSON(t, "/api/users/email/"+email, token),
		service.do(t, http.MethodPut, fmt.Sprintf("/api/users/%d", created.ID), token,
			[]byte(credentialsBody(email)), "application/json"),
	} {
		body := response.Body.String()
		for _, forbidden := range []string{"password", "$2a$", "$2b$", "$2y$", fixturePassword} {
			if strings.Contains(body, forbidden) {
				t.Errorf("a response carried %q: %s", forbidden, body)
			}
		}
	}
}

func TestRegisteringAnAddressAlreadyTakenAnswersTheConflictMessage(t *testing.T) {
	service := newLiveService(t)
	email := "conflict@jdw.com"
	if first := service.postJSON(t, "/auth/user", "", credentialsBody(email)); first.Code != http.StatusCreated {
		t.Fatalf("the first registration failed: %d %s", first.Code, first.Body.String())
	}

	response := service.postJSON(t, "/auth/user", "", credentialsBody(email))

	assertStatusAndText(t, response, http.StatusConflict, "User already exists with email address "+email)
}

func TestARegistrationIsAuditedToTheHardCodedRequester(t *testing.T) {
	// UserService.createUser(dto) hard-codes the requester, because the caller
	// has no identity yet. Transcribed rather than tidied.
	service := newLiveService(t)

	registered := service.postJSON(t, "/auth/user", "", credentialsBody("audited@jdw.com"))

	if registered.Code != http.StatusCreated {
		t.Fatalf("status = %d %s", registered.Code, registered.Body.String())
	}
	var created User
	if err := json.Unmarshal(registered.Body.Bytes(), &created); err != nil {
		t.Fatalf("body: %v", err)
	}
	if created.CreatedByUserID != registrationRequesterID {
		t.Errorf("createdByUserId = %d, want %d", created.CreatedByUserID, registrationRequesterID)
	}
}

func TestAnAuthenticatedCreateIsAuditedToTheCaller(t *testing.T) {
	service := newLiveService(t)

	created := service.postJSON(t, "/api/users", service.adminToken(t), credentialsBody("audited-by@jdw.com"))

	if created.Code != http.StatusCreated {
		t.Fatalf("status = %d %s", created.Code, created.Body.String())
	}
	var user User
	if err := json.Unmarshal(created.Body.Bytes(), &user); err != nil {
		t.Fatalf("body: %v", err)
	}
	if user.CreatedByUserID != service.admin.ID {
		t.Errorf("createdByUserId = %d, want the caller %d", user.CreatedByUserID, service.admin.ID)
	}
}

func TestAnInvalidRegistrationBodyIsRefusedFieldByField(t *testing.T) {
	service := newLiveService(t)

	response := service.postJSON(t, "/auth/user", "", `{"emailAddress":"not-an-address","password":"  "}`)

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	var fields map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	for field, want := range map[string]string{
		"emailAddress": "emailAddress is not valid",
		"password":     "password is mandatory",
	} {
		if fields[field] != want {
			t.Errorf("errors[%s] = %q, want %q", field, fields[field], want)
		}
	}
}

func TestAnUnparseableBodyIsRefusedWithTheFixedMessage(t *testing.T) {
	service := newLiveService(t)

	response := service.postJSON(t, "/auth/user", "", `{"emailAddress":`)

	assertStatusAndText(t, response, http.StatusBadRequest,
		"Request body is invalid. Please check the format and try again.")
}

func TestAnIdThatIsNotANumberIsRefusedBeforeAnythingElse(t *testing.T) {
	// Over the wire, through the real router and a real token: the refusal is a
	// 400 the container sets rather than a body the handler composed, so it
	// carries what BasicErrorController renders on the forward the caller's own
	// token authenticates. GET /api/users/abc against a booted usersrole
	// answers exactly this.
	service := newLiveService(t)
	token := service.adminToken(t)

	for _, path := range []string{"/api/users/not-a-number", "/api/roles/not-a-number"} {
		t.Run(path, func(t *testing.T) {
			assertContainerErrorBody(t, service.getJSON(t, path, token), http.StatusBadRequest, path)
		})
	}
}

func TestAPagingParameterThatIsNotANumberIsRefusedWithTheSameBody(t *testing.T) {
	// The query half of the same conversion failure. It shares a writer with the
	// path half because the JVM shares a mechanism: both fail argument
	// resolution, neither is caught, and the container answers both.
	service := newLiveService(t)
	token := service.adminToken(t)

	for _, path := range []string{"/api/users?page=abc", "/api/roles?size=abc"} {
		t.Run(path, func(t *testing.T) {
			response := service.getJSON(t, path, token)

			// The body names the path without the query string, as the JVM's does.
			assertContainerErrorBody(t, response, http.StatusBadRequest, strings.Split(path, "?")[0])
		})
	}
}

func TestARequestThatRoutesToNothingAnswersAsTheContainerDoes(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)

	unmapped := service.getJSON(t, "/api/nothing", token)
	assertContainerErrorBody(t, unmapped, http.StatusNotFound, "/api/nothing")

	wrongMethod := service.do(t, http.MethodPatch, "/api/users", token, nil, "")
	assertContainerErrorBody(t, wrongMethod, http.StatusMethodNotAllowed, "/api/users")
	if got := wrongMethod.Header().Get("Allow"); got == "" {
		t.Error("a 405 carries no Allow header")
	}
}

func TestAnUnroutableRequestWithNoTokenAnswers401(t *testing.T) {
	// Only a public path reaches the router without a token, and the JVM
	// answers those 401 rather than 404: its forward to /error is refused a
	// second time, and the entry point's status replaces the router's. Measured
	// on a booted usersrole for /auth/nope and /actuator/nope alike, both of
	// which sit inside a permitAll matcher.
	service := newLiveService(t)

	for _, path := range []string{"/auth/nothing", "/actuator/nothing"} {
		t.Run(path, func(t *testing.T) {
			response := service.getJSON(t, path, "")

			assertUnauthorizedShape(t, response)
		})
	}
}

func TestReadingAMissingRecordAnswersTheExceptionMessage(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)

	assertStatusAndText(t, service.getJSON(t, "/api/users/987654", token),
		http.StatusNotFound, "User not found with id 987654")
	assertStatusAndText(t, service.getJSON(t, "/api/users/email/nobody@jdw.com", token),
		http.StatusNotFound, "User not found with email address nobody@jdw.com")
	assertStatusAndText(t, service.getJSON(t, "/api/roles/987654", token),
		http.StatusNotFound, "Role not found with id 987654")
	assertStatusAndText(t, service.getJSON(t, "/api/roles/name/NO-SUCH-ROLE", token),
		http.StatusNotFound, "Role not found with name NO-SUCH-ROLE")
}

func TestGrantingNamesTheMissingIdInTheMessage(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)

	missingRole := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/users/%d/roles/grant", service.admin.ID), token,
		[]byte("[987654]"), "application/json")
	assertStatusAndText(t, missingRole, http.StatusNotFound, "Role not found with id 987654")

	missingUser := service.do(t, http.MethodPut, "/api/users/987654/roles/grant", token,
		[]byte(fmt.Sprintf("[%d]", roleIDByName(t, service.store, "USER"))), "application/json")
	assertStatusAndText(t, missingUser, http.StatusNotFound, "User not found with id 987654")

	missingFromRoleSide := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/roles/%d/users/grant", roleIDByName(t, service.store, "USER")), token,
		[]byte("[987654]"), "application/json")
	assertStatusAndText(t, missingFromRoleSide, http.StatusNotFound, "User not found with id 987654")
}

func TestUpdatingOntoAValueAlreadyHeldIsTheFrozenFiveHundred(t *testing.T) {
	// A unique-constraint violation that no handler maps. Frozen because the
	// frontends' error text is keyed on the status code.
	service := newLiveService(t)
	token := service.adminToken(t)
	held := service.postJSON(t, "/api/users", token, credentialsBody("taken@jdw.com"))
	if held.Code != http.StatusCreated {
		t.Fatalf("creating the holder failed: %d %s", held.Code, held.Body.String())
	}
	mover := service.postJSON(t, "/api/users", token, credentialsBody("moving@jdw.com"))
	if mover.Code != http.StatusCreated {
		t.Fatalf("creating the mover failed: %d %s", mover.Code, mover.Body.String())
	}
	var moving User
	if err := json.Unmarshal(mover.Body.Bytes(), &moving); err != nil {
		t.Fatalf("body: %v", err)
	}

	response := service.do(t, http.MethodPut, fmt.Sprintf("/api/users/%d", moving.ID), token,
		[]byte(credentialsBody("taken@jdw.com")), "application/json")

	if response.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d (body %q)",
			response.Code, http.StatusInternalServerError, response.Body.String())
	}
	var container map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &container); err != nil {
		t.Fatalf("body is not the container error representation: %v", err)
	}
	if container["error"] != "Internal Server Error" {
		t.Errorf("error = %v, want Internal Server Error", container["error"])
	}
}

func TestCreatingARoleAnswersTheCreatedStatusAndTheConflictMessage(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	body := `{"name":"LIVE-AUDITOR","description":"Reads everything, writes nothing."}`

	created := service.postJSON(t, "/api/roles", token, body)

	if created.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %q)", created.Code, http.StatusCreated, created.Body.String())
	}
	var role Role
	if err := json.Unmarshal(created.Body.Bytes(), &role); err != nil {
		t.Fatalf("body is not a role: %v", err)
	}
	if role.CreatedByUserID != service.admin.ID {
		t.Errorf("createdByUserId = %d, want the caller %d", role.CreatedByUserID, service.admin.ID)
	}

	assertStatusAndText(t, service.postJSON(t, "/api/roles", token, body),
		http.StatusConflict, "Role already exists with name LIVE-AUDITOR")
}

func TestDeletingAUserOverTheWireTakesItsProfileTreeWithIt(t *testing.T) {
	service := newLiveService(t)
	_, pool := newTestStore(t)
	user := createUserFor(t, service.store, "wire-delete@jdw.com")
	profileID := seedProfile(t, pool, user.ID)
	seedProfileSubresources(t, pool, profileID, user.ID)

	deleted := service.do(t, http.MethodDelete, fmt.Sprintf("/api/users/%d", user.ID), service.adminToken(t), nil, "")

	if deleted.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want %d (body %q)", deleted.Code, http.StatusNoContent, deleted.Body.String())
	}
	if body := deleted.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	assertRowCount(t, pool, "SELECT count(*) FROM auth.addresses WHERE profile_id = $1", profileID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profile_icons WHERE profile_id = $1", profileID, 0)
	assertRowCount(t, pool, "SELECT count(*) FROM auth.profiles WHERE user_id = $1", user.ID, 0)

	// A no-op for an id that does not exist, and still 204.
	repeated := service.do(t, http.MethodDelete, fmt.Sprintf("/api/users/%d", user.ID),
		service.adminToken(t), nil, "")
	if repeated.Code != http.StatusNoContent {
		t.Errorf("deleting again = %d, want %d", repeated.Code, http.StatusNoContent)
	}
}

func TestTheListingsClampOutOfRangePagingRatherThanRejectingIt(t *testing.T) {
	// A negative page is served as page 0 and an enormous size as 500, both with
	// a 200. The bound exists so the query is always bounded whatever a caller
	// asks for.
	service := newLiveService(t)
	token := service.adminToken(t)
	createUserFor(t, service.store, "page-a@jdw.com")
	createUserFor(t, service.store, "page-b@jdw.com")

	for _, surface := range []string{"/api/users", "/api/roles"} {
		for _, query := range []string{"", "?page=-5", "?size=0", "?size=100000", "?page=0&size=1"} {
			t.Run(surface+query, func(t *testing.T) {
				response := service.getJSON(t, surface+query, token)

				if response.Code != http.StatusOK {
					t.Fatalf("status = %d, want %d (body %q)",
						response.Code, http.StatusOK, response.Body.String())
				}
				var rows []map[string]any
				if err := json.Unmarshal(response.Body.Bytes(), &rows); err != nil {
					t.Fatalf("body is not a JSON array: %v", err)
				}
				if len(rows) == 0 {
					t.Error("the page is empty; the clamp served no rows")
				}
				if query == "?page=0&size=1" && len(rows) != 1 {
					t.Errorf("page of one returned %d rows", len(rows))
				}
				if query == "?size=100000" && len(rows) > maximumSize {
					t.Errorf("page returned %d rows, want at most %d", len(rows), maximumSize)
				}
			})
		}
	}
}

func TestAPageIndexWiderThanSpringsIntIsRefusedRatherThanOverflowed(t *testing.T) {
	// The clamp only raises a floor, so a page index that survives it still gets
	// multiplied by the page size. Read at 64 bits that product wraps negative,
	// Postgres refuses the OFFSET and the caller reads a 500 for what the
	// contract calls a 400. Both listings, because /api/roles is reachable by any
	// authenticated principal.
	service := newLiveService(t)
	token := service.adminToken(t)

	for _, surface := range []string{"/api/users", "/api/roles"} {
		for _, query := range []string{"?page=9223372036854775807", "?page=2147483648", "?size=9223372036854775807"} {
			t.Run(surface+query, func(t *testing.T) {
				response := service.getJSON(t, surface+query, token)

				if response.Code != http.StatusBadRequest {
					t.Errorf("status = %d, want %d (body %q)",
						response.Code, http.StatusBadRequest, response.Body.String())
				}
			})
		}
	}
}

func TestAPageParameterThatIsNotANumberIsRefused(t *testing.T) {
	service := newLiveService(t)

	response := service.getJSON(t, "/api/users?page=first", service.adminToken(t))

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestAnEmptyListingIsAnArrayRatherThanNull(t *testing.T) {
	service := newLiveService(t)

	response := service.getJSON(t, "/api/users?page=9999", service.adminToken(t))

	if got := strings.TrimSpace(response.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestTheRoleCatalogueIsReadableByAnyAuthenticatedPrincipal(t *testing.T) {
	// Frozen deliberately rather than tightened: the role screens are reachable
	// by every authenticated user precisely because these three reads answer
	// them, and tightening turns them into 403s at cutover with no client change
	// to compensate.
	service := newLiveService(t)
	ordinary := createUserFor(t, service.store, "catalogue@jdw.com")
	token := mint(t, authtest.Claims{
		Subject: ordinary.EmailAddress, Roles: []string{"USER"}, UserID: &ordinary.ID,
	})

	for _, path := range []string{
		"/api/roles",
		fmt.Sprintf("/api/roles/%d", roleIDByName(t, service.store, "USER")),
		"/api/roles/name/USER",
	} {
		t.Run(path, func(t *testing.T) {
			response := service.getJSON(t, path, token)

			if response.Code != http.StatusOK {
				t.Errorf("status = %d, want %d (body %q)", response.Code, http.StatusOK, response.Body.String())
			}
		})
	}
}

func TestTheElevatedRoleGuardReadsTheGrantTableRatherThanTheToken(t *testing.T) {
	// A caller whose token claims MANAGER, and whose grants say nothing, still
	// cannot hand out the elevated role. The claim decides the operation's own
	// rule; the table decides this one.
	service := newLiveService(t)
	claimant := createUserFor(t, service.store, "claims-manager@jdw.com")
	token := mint(t, authtest.Claims{
		Subject: claimant.EmailAddress, Roles: []string{"MANAGER"}, UserID: &claimant.ID,
	})
	body := []byte(fmt.Sprintf("[%d]", elevatedRoleID))

	refused := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/users/%d/roles/grant", claimant.ID), token, body, "application/json")

	if refused.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %q)", refused.Code, http.StatusForbidden, refused.Body.String())
	}
	assertForbiddenShape(t, refused)

	// Granted the elevated role in the table, the same request goes through.
	if _, err := service.store.GrantRolesToUser(t.Context(),
		claimant.ID, []int64{elevatedRoleID}, claimant.ID); err != nil {
		t.Fatalf("GrantRolesToUser: %v", err)
	}
	allowed := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/users/%d/roles/grant", claimant.ID), token, body, "application/json")
	if allowed.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body %q)", allowed.Code, http.StatusOK, allowed.Body.String())
	}
}

func TestTheOperationalEndpointsAreServedWithoutAToken(t *testing.T) {
	// SecurityConfig permits /actuator/** without authentication, and the
	// chart's probes and the scrape configuration depend on it staying that way.
	service := newLiveService(t)

	for _, path := range []string{healthPath, actuatorHealthPath, actuatorMetricsPath} {
		t.Run(path, func(t *testing.T) {
			response := service.getJSON(t, path, "")

			if response.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
			}
		})
	}
}

func TestTheScrapeEndpointCarriesTheRequestDurationHistogram(t *testing.T) {
	service := newLiveService(t)
	service.getJSON(t, "/api/users/987654", service.adminToken(t))

	response := service.getJSON(t, actuatorMetricsPath, "")

	body := response.Body.String()
	if !strings.Contains(body, requestDurationName+"_bucket") {
		t.Error("the scrape carries no histogram buckets; the percentile panels would be unfillable")
	}
	if !strings.Contains(body, `uri="/api/users/{userId}"`) {
		t.Error("the uri label is not the route pattern; every id would open its own series")
	}
	if !strings.Contains(body, `outcome="CLIENT_ERROR"`) {
		t.Error("the outcome label is missing; a query written against the JVM series would not select")
	}
}

func TestAPreflightIsAnsweredAheadOfAuthenticationOnALiveRoute(t *testing.T) {
	service := newLiveService(t)
	request := httptest.NewRequest(http.MethodOptions, "/api/users/1", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	request.Header.Set("Access-Control-Request-Method", "GET")
	request.Header.Set("Access-Control-Request-Headers", "authorization")
	response := httptest.NewRecorder()

	service.handler.ServeHTTP(response, request)

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d; a preflight carries no token and must not be authenticated",
			response.Code, http.StatusOK)
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "http://localhost:4200" {
		t.Errorf("Access-Control-Allow-Origin = %q, want the requesting origin", got)
	}
	if got := response.Header().Get("Access-Denied-Reason"); got != "" {
		t.Errorf("Access-Denied-Reason = %q; the preflight reached the authentication layer", got)
	}
}

func TestARefusedCallerSeesTheContainerErrorBodyForTheRequestedPath(t *testing.T) {
	// The 403 body comes from the shared library's writer, not from here. What
	// this pins is that the path it reports is the one the caller asked for,
	// which is the field a support ticket is read against.
	service := newLiveService(t)
	stranger := createUserFor(t, service.store, "forbidden-path@jdw.com")
	token := mint(t, authtest.Claims{
		Subject: stranger.EmailAddress, Roles: []string{"USER"}, UserID: &stranger.ID,
	})

	response := service.getJSON(t, "/api/users/424242", token)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %q)", response.Code, http.StatusForbidden, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("the 403 body is not JSON: %v", err)
	}
	if body["path"] != "/api/users/424242" {
		t.Errorf("path = %v, want the requested path", body["path"])
	}
}

// unreachableStore is the production store over a pool that can never connect,
// so the failure the handlers see is a real driver error rather than a
// hand-written sentinel. pgxpool connects lazily, so building it costs nothing
// and every statement fails the same way a database that has gone away does.
func unreachableStore(t *testing.T) *PostgresStore {
	t.Helper()
	pool, err := pgxpool.New(t.Context(), "postgres://nobody:nobody@127.0.0.1:1/nothing")
	if err != nil {
		t.Fatalf("open a pool that cannot connect: %v", err)
	}
	t.Cleanup(pool.Close)
	return NewPostgresStore(pool)
}

func TestAStorageFailureCarriesTheContainerErrorBody(t *testing.T) {
	// Nothing composes this response: the handler logs the cause and hands the
	// status to the container, which renders its own body for a caller whose
	// token survives the forward to /error. Measured on a booted usersrole by
	// making the repository throw — 500 application/json, not 500 with nothing.
	//
	// Driven twice: through brokenStore, so every operation's failure path is
	// the same shape, and through the production store over a dead pool, so the
	// error that reaches the writer is one pgx actually produced.
	for name, store := range map[string]Store{
		"a store that reports failure": brokenStore{},
		"the real store, unreachable":  unreachableStore(t),
	} {
		t.Run(name, func(t *testing.T) {
			server := parityServer(t, store)
			request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
			request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertContainerErrorBody(t, response, http.StatusInternalServerError, "/api/users/42")
		})
	}
}

func TestAStorageFailureOnAPublicOperationAnswers401(t *testing.T) {
	// The two /auth operations take no token, so their forward to /error carries
	// none either and is refused a second time — the entry point's 401 replaces
	// the 500 the container had set. Measured on a booted usersrole: a
	// repository that throws under POST /auth/user answers 401 with
	// Content-Length 0, never 500.
	//
	// It reads as an odd answer to an outage, and it is what the deployed
	// service answers. Diverging here would move a status the frontends key
	// their message off.
	server := parityServer(t, brokenStore{})
	request := httptest.NewRequest(http.MethodPost, "/auth/user",
		strings.NewReader(credentialsBody("outage@jdw.com")))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	assertUnauthorizedShape(t, response)
}
