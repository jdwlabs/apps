package main

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
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

// The fixture the whole suite is scoped to: one profile, id 1, owned by user 1.
const (
	selfUserID     = int64(1)
	selfProfileID  = int64(1)
	otherUserID    = int64(2)
	otherProfileID = int64(2)
	fixtureAddress = int64(1)
)

// stubStore answers every read with the fixture profile and every write with
// success, so an operation's status depends on nothing but its authorization
// outcome. Storage behaviour is covered against a real Postgres in the
// integration suites.
type stubStore struct{}

func fixtureProfile() Profile {
	return Profile{
		ID: selfProfileID, FirstName: "Ada", LastName: "Lovelace",
		Birthdate: Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)},
		UserID:    selfUserID,
		Addresses: []Address{{ID: fixtureAddress, ProfileID: selfProfileID, AddressLine1: "12 Noel Street"}},
	}
}

func (stubStore) ListProfiles(context.Context, int, int) ([]Profile, error) {
	return []Profile{fixtureProfile()}, nil
}
func (stubStore) ProfileByID(context.Context, int64) (Profile, error) { return fixtureProfile(), nil }
func (stubStore) ProfileByUserID(context.Context, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) ProfileIDForUser(_ context.Context, userID int64) (int64, bool, error) {
	if userID == selfUserID {
		return selfProfileID, true, nil
	}
	return 0, false, nil
}
func (stubStore) CreateProfile(context.Context, ProfileCreateRequest, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) UpdateProfileByID(context.Context, int64, ProfileUpdateRequest, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) UpdateProfileByUserID(context.Context, int64, ProfileUpdateRequest, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) DeleteProfileByID(context.Context, int64) error     { return nil }
func (stubStore) DeleteProfileByUserID(context.Context, int64) error { return nil }
func (stubStore) AddAddress(context.Context, int64, AddressRequest, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) UpdateAddress(context.Context, int64, int64, AddressRequest, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) DeleteAddress(context.Context, int64, int64) error { return nil }
func (stubStore) Icon(context.Context, int64) (ProfileIcon, error) {
	return ProfileIcon{ID: 1, ProfileID: selfProfileID, Icon: []byte("png")}, nil
}
func (stubStore) AddIcon(context.Context, int64, []byte, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) ReplaceIcon(context.Context, int64, []byte, int64) (Profile, error) {
	return fixtureProfile(), nil
}
func (stubStore) DeleteIcon(context.Context, int64) error { return nil }

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

func parityServer(t *testing.T, store Store) http.Handler {
	t.Helper()
	server, err := NewServer(ServerConfig{
		Store:    store,
		Verifier: parityVerifier(t),
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

// principals names the token each authorization outcome needs, in terms a
// contract rule can be read against.
type principal struct {
	name   string
	claims authtest.Claims
}

func admin() principal {
	return principal{
		name: "an ADMIN",
		claims: authtest.Claims{
			Subject: "admin@jdw.com", Roles: []string{"ADMIN"},
			UserID: ptr(otherUserID), ProfileID: ptr(otherProfileID),
		},
	}
}

func self() principal {
	return principal{
		name: "the owner",
		claims: authtest.Claims{
			Subject: "self@jdw.com", Roles: []string{"USER"},
			UserID: ptr(selfUserID), ProfileID: ptr(selfProfileID),
		},
	}
}

func selfWithNoProfileClaim() principal {
	return principal{
		name: "the owner whose token predates the profile",
		claims: authtest.Claims{
			Subject: "self@jdw.com", Roles: []string{"USER"}, UserID: ptr(selfUserID),
		},
	}
}

func stranger() principal {
	return principal{
		name: "a stranger",
		claims: authtest.Claims{
			Subject: "stranger@jdw.com", Roles: []string{"USER"},
			UserID: ptr(otherUserID), ProfileID: ptr(otherProfileID),
		},
	}
}

// allowedBy and deniedBy turn a contract rule into the principals that must
// pass and fail it, so a rule change in the contract changes which tokens this
// suite drives rather than being silently untested.
func allowedBy(rule authz.Rule) []principal {
	switch rule {
	case authz.RuleAdmin:
		return []principal{admin()}
	case authz.RuleAdminOrSelfByProfileID:
		return []principal{admin(), self(), selfWithNoProfileClaim()}
	case authz.RuleAdminOrSelfByUserID, authz.RuleAdminOrSelfByBodyUserID:
		return []principal{admin(), self()}
	default:
		return nil
	}
}

func deniedBy(rule authz.Rule) []principal {
	switch rule {
	case authz.RuleAdmin:
		return []principal{self(), stranger()}
	case authz.RuleAdminOrSelfByProfileID, authz.RuleAdminOrSelfByUserID, authz.RuleAdminOrSelfByBodyUserID:
		return []principal{stranger()}
	default:
		return nil
	}
}

// parityCase is one operation, with the request that exercises it and the
// status it answers to a caller the rule allows.
type parityCase struct {
	operation     string
	method        string
	path          string
	rule          authz.Rule
	successStatus int
	body          func() (io []byte, contentType string)
	accept        string
}

func jsonBody(text string) func() ([]byte, string) {
	return func() ([]byte, string) { return []byte(text), "application/json" }
}

func iconUpload() ([]byte, string) {
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile("icon", "icon.png")
	if err != nil {
		panic("multipart writer refused a part: " + err.Error())
	}
	if _, err := part.Write([]byte("\x89PNG\r\n\x1a\n")); err != nil {
		panic("multipart writer refused the bytes: " + err.Error())
	}
	if err := writer.Close(); err != nil {
		panic("multipart writer refused to close: " + err.Error())
	}
	return buffer.Bytes(), writer.FormDataContentType()
}

const validProfileBody = `{"firstName":"Ada","lastName":"Lovelace","birthdate":"1815-12-10","userId":1}`

const validProfileUpdateBody = `{"firstName":"Ada","lastName":"Lovelace","birthdate":"1815-12-10"}`

const validAddressBody = `{"addressLine1":"12 Noel Street","city":"London","stateProvince":"Greater London",` +
	`"postalCode":"W1F 8GQ","country":"GB"}`

// parityCases is the whole served surface, one entry per operation. The
// contract suite asserts that these are exactly the operations the frozen
// document describes and that each rule here is the rule it names.
func parityCases() []parityCase {
	return []parityCase{
		{
			operation: "getProfiles", method: http.MethodGet, path: "/api/profiles",
			rule: authz.RuleAdmin, successStatus: http.StatusOK,
		},
		{
			operation: "createProfile", method: http.MethodPost, path: "/api/profiles",
			rule: authz.RuleAdminOrSelfByBodyUserID, successStatus: http.StatusCreated,
			body: jsonBody(validProfileBody),
		},
		{
			operation: "getProfileById", method: http.MethodGet, path: "/api/profiles/1",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK,
		},
		{
			operation: "updateProfileById", method: http.MethodPut, path: "/api/profiles/1",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK,
			body: jsonBody(validProfileUpdateBody),
		},
		{
			operation: "deleteProfileById", method: http.MethodDelete, path: "/api/profiles/1",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusNoContent,
		},
		{
			operation: "getProfileByUserId", method: http.MethodGet, path: "/api/profiles/by-user/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusOK,
		},
		{
			operation: "updateProfileByUserId", method: http.MethodPut, path: "/api/profiles/by-user/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusOK,
			body: jsonBody(validProfileUpdateBody),
		},
		{
			operation: "deleteProfileByUserId", method: http.MethodDelete, path: "/api/profiles/by-user/1",
			rule: authz.RuleAdminOrSelfByUserID, successStatus: http.StatusNoContent,
		},
		{
			operation: "addAddress", method: http.MethodPost, path: "/api/profiles/1/address",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK,
			body: jsonBody(validAddressBody),
		},
		{
			operation: "updateAddress", method: http.MethodPut, path: "/api/profiles/1/address/1",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK,
			body: jsonBody(validAddressBody),
		},
		{
			operation: "deleteAddress", method: http.MethodDelete, path: "/api/profiles/1/address/1",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusNoContent,
		},
		{
			operation: "getProfileIcon", method: http.MethodGet, path: "/api/profiles/1/icon",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK, accept: "image/png",
		},
		{
			operation: "addIcon", method: http.MethodPost, path: "/api/profiles/1/icon",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK, body: iconUpload,
		},
		{
			operation: "updateIcon", method: http.MethodPut, path: "/api/profiles/1/icon",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusOK, body: iconUpload,
		},
		{
			operation: "deleteIcon", method: http.MethodDelete, path: "/api/profiles/1/icon",
			rule: authz.RuleAdminOrSelfByProfileID, successStatus: http.StatusNoContent,
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
	if tc.accept != "" {
		request.Header.Set("Accept", tc.accept)
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
				response := httptest.NewRecorder()

				server.ServeHTTP(response, tc.request(t, mint(t, who.claims)))

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

func TestEveryOperationRefusesARequestWithNoToken(t *testing.T) {
	server := parityServer(t, stubStore{})

	for _, tc := range parityCases() {
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

func TestEveryOperationRefusesATokenThatDoesNotVerify(t *testing.T) {
	server := parityServer(t, stubStore{})
	tampered := authtest.TamperSignature(mint(t, admin().claims))

	for _, tc := range parityCases() {
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
// renders nothing. Every refusal on this service goes through the shared
// library's writer, so what this asserts is that the writer is reached, not that
// this service composed a body.
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

func TestTheProfileFallbackIsKeyedOnTheClaimAndNotOnThePath(t *testing.T) {
	// Falling back is not the same as trusting the request: the lookup keys on
	// the user_id claim, so a caller with no profile_id claim still cannot reach
	// somebody else's profile by asking for its id.
	server := parityServer(t, stubStore{})
	token := mint(t, selfWithNoProfileClaim().claims)
	response := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/api/profiles/2", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestACallerWithNoProfileAtAllIsDeniedRatherThanAdmitted(t *testing.T) {
	server := parityServer(t, stubStore{})
	token := mint(t, authtest.Claims{
		Subject: "profileless@jdw.com", Roles: []string{"USER"}, UserID: ptr(int64(99)),
	})
	response := httptest.NewRecorder()

	request := httptest.NewRequest(http.MethodGet, "/api/profiles/1", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	server.ServeHTTP(response, request)

	if response.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}
