package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"libs/backend/shared/auth/authtest"
	"libs/backend/shared/servicehttp"
)

// The end-to-end half of the parity suite: the same handlers, router, CORS and
// authentication as production, over the deployed schema in a container. The
// stub-backed suite pins authorization; this one pins what the service actually
// answers once real rows are involved.

type liveService struct {
	handler http.Handler
	store   *PostgresStore
	adminID int64
}

func newLiveService(t *testing.T) *liveService {
	t.Helper()
	store, pool := newTestStore(t)
	server, err := NewServer(ServerConfig{
		Store: store, Verifier: parityVerifier(t), CORS: springShapedCORS(),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}
	return &liveService{
		handler: server.Handler(),
		store:   store,
		adminID: seedUser(t, pool, fmt.Sprintf("admin-%s@jdw.com", t.Name())),
	}
}

func (s *liveService) adminToken(t *testing.T) string {
	t.Helper()
	return mint(t, authtest.Claims{
		Subject: "admin@jdw.com", Roles: []string{"ADMIN"}, UserID: &s.adminID,
	})
}

func (s *liveService) do(t *testing.T, method, path, token string, body []byte, contentType string) *httptest.ResponseRecorder {
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

func TestAProfileTravelsTheWireInTheShapeTheFrontendsParse(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	ownerID := seedUser(t, pool, "wire@jdw.com")

	created := service.do(t, http.MethodPost, "/api/profiles", token,
		[]byte(fmt.Sprintf(
			`{"firstName":"Ada","middleName":null,"lastName":"Lovelace","birthdate":"1815-12-10","userId":%d}`,
			ownerID)), "application/json")

	if created.Code != http.StatusCreated {
		t.Fatalf("status = %d, want %d (body %q)", created.Code, http.StatusCreated, created.Body.String())
	}
	if got, want := created.Header().Get("Content-Type"), contentTypeJSON; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	var body map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %v", err)
	}
	if body["birthdate"] != "1815-12-10" {
		t.Errorf("birthdate = %v, want the plain calendar date", body["birthdate"])
	}
	if body["middleName"] != nil {
		t.Errorf("middleName = %v, want null", body["middleName"])
	}
	if body["icon"] != nil {
		t.Errorf("icon = %v, want null", body["icon"])
	}
	addresses, ok := body["addresses"].([]any)
	if !ok || len(addresses) != 0 {
		t.Errorf("addresses = %v, want an empty array", body["addresses"])
	}
	createdTime, _ := body["createdTime"].(string)
	if !strings.HasSuffix(createdTime, "Z") || !strings.Contains(createdTime, "T") {
		t.Errorf("createdTime = %q, want an ISO-8601 stamp at the zero offset Jackson writes", createdTime)
	}
}

func TestCreatingAProfileForAMissingUserAnswersTheExceptionMessage(t *testing.T) {
	service := newLiveService(t)

	response := service.do(t, http.MethodPost, "/api/profiles", service.adminToken(t),
		[]byte(`{"firstName":"Ada","lastName":"Lovelace","birthdate":"1815-12-10","userId":987654}`),
		"application/json")

	assertStatusAndText(t, response, http.StatusNotFound, "User not found with user id 987654")
}

func TestCreatingASecondProfileAnswersTheConflictMessage(t *testing.T) {
	service := newLiveService(t)
	_, pool := newTestStore(t)
	ownerID := seedUser(t, pool, "conflict@jdw.com")
	body := []byte(fmt.Sprintf(
		`{"firstName":"Ada","lastName":"Lovelace","birthdate":"1815-12-10","userId":%d}`, ownerID))
	if first := service.do(t, http.MethodPost, "/api/profiles", service.adminToken(t), body,
		"application/json"); first.Code != http.StatusCreated {
		t.Fatalf("the first create failed: %d %s", first.Code, first.Body.String())
	}

	response := service.do(t, http.MethodPost, "/api/profiles", service.adminToken(t), body, "application/json")

	assertStatusAndText(t, response, http.StatusConflict,
		fmt.Sprintf("Profile already exists for user with id %d", ownerID))
}

func TestAnInvalidCreateBodyIsRefusedFieldByField(t *testing.T) {
	service := newLiveService(t)

	response := service.do(t, http.MethodPost, "/api/profiles", service.adminToken(t),
		[]byte(`{"firstName":"  ","lastName":"Lovelace","birthdate":"3000-01-01"}`), "application/json")

	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	var fields map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &fields); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	for field, want := range map[string]string{
		"firstName": "firstName is mandatory",
		"birthdate": "birthdate must be a past date",
		"userId":    "userId is mandatory",
	} {
		if fields[field] != want {
			t.Errorf("errors[%s] = %q, want %q", field, fields[field], want)
		}
	}
}

func TestAnUnparseableBodyIsRefusedWithTheFixedMessage(t *testing.T) {
	service := newLiveService(t)

	response := service.do(t, http.MethodPost, "/api/profiles", service.adminToken(t),
		[]byte(`{"firstName":`), "application/json")

	assertStatusAndText(t, response, http.StatusBadRequest,
		"Request body is invalid. Please check the format and try again.")
}

func TestAProfileIdThatIsNotANumberIsRefusedBeforeAnythingElse(t *testing.T) {
	// Over the wire, through the real router and a real token: the refusal is a
	// 400 the container sets rather than a body the handler composed, so it
	// carries what BasicErrorController renders on the forward the caller's own
	// token authenticates. GET /api/profiles/abc against a booted usersrole
	// answers exactly this.
	service := newLiveService(t)

	response := service.getJSON(t, "/api/profiles/not-a-number", service.adminToken(t))

	assertContainerErrorBody(t, response, http.StatusBadRequest, "/api/profiles/not-a-number")
}

func TestARequestThatRoutesToNothingAnswersAsTheContainerDoes(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)

	unmapped := service.getJSON(t, "/api/nothing", token)
	assertContainerErrorBody(t, unmapped, http.StatusNotFound, "/api/nothing")

	wrongMethod := service.do(t, http.MethodPatch, "/api/profiles", token, nil, "")
	assertContainerErrorBody(t, wrongMethod, http.StatusMethodNotAllowed, "/api/profiles")
	if got := wrongMethod.Header().Get("Allow"); got == "" {
		t.Error("a 405 carries no Allow header")
	}
}

func TestAnAcceptHeaderTheIconCannotSatisfyAnswersAsTheContainerDoes(t *testing.T) {
	// The only produces condition in either service, and so the only 406 either
	// can answer. Measured on a booted usersrole: GET /api/profiles/1/icon with
	// Accept: application/json is refused before the handler runs, and the body
	// is the container's — the caller asked for JSON and JSON is what /error
	// renders.
	service := newLiveService(t)
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/1/icon", nil)
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Authorization", "Bearer "+service.adminToken(t))
	response := httptest.NewRecorder()

	service.handler.ServeHTTP(response, request)

	assertContainerErrorBody(t, response, http.StatusNotAcceptable, "/api/profiles/1/icon")
}

func TestAnUnroutableRequestWithNoTokenAnswers401(t *testing.T) {
	// Only a public path reaches the router without a token, and the JVM
	// answers those 401 rather than 404: its forward to /error is refused a
	// second time, and the entry point's status replaces the router's. Measured
	// on a booted usersrole for /actuator/nope, which sits inside a permitAll
	// matcher and still answers 401.
	service := newLiveService(t)

	response := service.getJSON(t, "/actuator/nothing", "")

	assertUnauthorizedShape(t, response)
}

func TestReadingAMissingProfileAnswersTheExceptionMessage(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)

	assertStatusAndText(t, service.getJSON(t, "/api/profiles/987654", token),
		http.StatusNotFound, "Profile not found with id 987654")
	assertStatusAndText(t, service.getJSON(t, "/api/profiles/by-user/987654", token),
		http.StatusNotFound, "Profile not found with user id 987654")
}

func TestTheListingClampsOutOfRangePagingRatherThanRejectingIt(t *testing.T) {
	// A negative page is served as page 0 and an enormous size as 500, both with
	// a 200. The bound exists so the query is always bounded whatever a caller
	// asks for.
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	for _, email := range []string{"page-a@jdw.com", "page-b@jdw.com"} {
		createProfileFor(t, service.store, seedUser(t, pool, email))
	}

	for _, query := range []string{"", "?page=-5", "?size=0", "?size=100000", "?page=0&size=1"} {
		t.Run("profiles"+query, func(t *testing.T) {
			response := service.getJSON(t, "/api/profiles"+query, token)

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d (body %q)", response.Code, http.StatusOK, response.Body.String())
			}
			var profiles []map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &profiles); err != nil {
				t.Fatalf("body is not a JSON array: %v", err)
			}
			if len(profiles) == 0 {
				t.Error("the page is empty; the clamp served no rows")
			}
			if query == "?page=0&size=1" && len(profiles) != 1 {
				t.Errorf("page of one returned %d rows", len(profiles))
			}
			if query == "?size=100000" && len(profiles) > maximumSize {
				t.Errorf("page returned %d rows, want at most %d", len(profiles), maximumSize)
			}
		})
	}
}

func TestAPageParameterThatIsNotANumberIsRefused(t *testing.T) {
	// The query half of the same conversion failure the path variable above
	// answers. It shares a writer because the JVM shares a mechanism: both fail
	// argument resolution, neither is caught, and the container answers both.
	service := newLiveService(t)

	response := service.getJSON(t, "/api/profiles?page=first", service.adminToken(t))

	// The body names the path without the query string, as the JVM's does.
	assertContainerErrorBody(t, response, http.StatusBadRequest, "/api/profiles")
}

func TestAnEmptyListingIsAnArrayRatherThanNull(t *testing.T) {
	service := newLiveService(t)

	response := service.getJSON(t, "/api/profiles?page=9999", service.adminToken(t))

	if got := strings.TrimSpace(response.Body.String()); got != "[]" {
		t.Errorf("body = %q, want []", got)
	}
}

func TestDeletingAnAddressFromAnotherProfileAnswersTheScopedMessage(t *testing.T) {
	// The insecure direct object reference this scoping closed: without the
	// profile in the WHERE clause, any authenticated principal with a profile
	// could delete any address by guessing a sequential id.
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	owner := createProfileFor(t, service.store, seedUser(t, pool, "scoped-owner@jdw.com"))
	stranger := createProfileFor(t, service.store, seedUser(t, pool, "scoped-stranger@jdw.com"))

	added := service.do(t, http.MethodPost, fmt.Sprintf("/api/profiles/%d/address", owner.ID), token,
		[]byte(validAddressBody), "application/json")
	if added.Code != http.StatusOK {
		t.Fatalf("adding an address: %d %s", added.Code, added.Body.String())
	}
	var withAddress Profile
	if err := json.Unmarshal(added.Body.Bytes(), &withAddress); err != nil {
		t.Fatalf("body is not a profile: %v", err)
	}
	addressID := withAddress.Addresses[0].ID

	refused := service.do(t, http.MethodDelete,
		fmt.Sprintf("/api/profiles/%d/address/%d", stranger.ID, addressID), token, nil, "")

	assertStatusAndText(t, refused, http.StatusNotFound,
		fmt.Sprintf("Address not found with id %d for profile with id %d", addressID, stranger.ID))

	deleted := service.do(t, http.MethodDelete,
		fmt.Sprintf("/api/profiles/%d/address/%d", owner.ID, addressID), token, nil, "")
	if deleted.Code != http.StatusNoContent {
		t.Errorf("the owner's delete = %d, want %d", deleted.Code, http.StatusNoContent)
	}
	if body := deleted.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestUpdatingAnAddressFromAnotherProfileAnswersTheUnscopedMessage(t *testing.T) {
	// The update path reports the address alone, without the profile. The two
	// messages differ in the JVM and are transcribed rather than harmonised.
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	owner := createProfileFor(t, service.store, seedUser(t, pool, "update-scope-owner@jdw.com"))
	stranger := createProfileFor(t, service.store, seedUser(t, pool, "update-scope-stranger@jdw.com"))
	added, err := service.store.AddAddress(context.Background(), owner.ID, completeAddress(), owner.UserID)
	if err != nil {
		t.Fatalf("AddAddress: %v", err)
	}
	addressID := added.Addresses[0].ID

	response := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/profiles/%d/address/%d", stranger.ID, addressID), token,
		[]byte(validAddressBody), "application/json")

	assertStatusAndText(t, response, http.StatusNotFound, fmt.Sprintf("Address not found with id %d", addressID))
}

func iconMultipart(t *testing.T, partName string, size int) ([]byte, string) {
	t.Helper()
	var buffer bytes.Buffer
	writer := multipart.NewWriter(&buffer)
	part, err := writer.CreateFormFile(partName, "icon.png")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}
	if _, err := part.Write(bytes.Repeat([]byte{0x89}, size)); err != nil {
		t.Fatalf("write part: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	return buffer.Bytes(), writer.FormDataContentType()
}

func TestTheIconRoundTripsAsImagePng(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	profile := createProfileFor(t, service.store, seedUser(t, pool, "icon-http@jdw.com"))
	path := fmt.Sprintf("/api/profiles/%d/icon", profile.ID)

	missing := service.getJSON(t, path, token)
	assertStatusAndText(t, missing, http.StatusNotFound,
		fmt.Sprintf("Profile icon not found with id %d", profile.ID))

	body, contentType := iconMultipart(t, "icon", 64)
	uploaded := service.do(t, http.MethodPost, path, token, body, contentType)
	if uploaded.Code != http.StatusOK {
		t.Fatalf("upload = %d %s", uploaded.Code, uploaded.Body.String())
	}

	downloaded := service.getJSON(t, path, token)
	if downloaded.Code != http.StatusOK {
		t.Fatalf("download = %d %s", downloaded.Code, downloaded.Body.String())
	}
	if got, want := downloaded.Header().Get("Content-Type"), "image/png"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	if got := downloaded.Body.Len(); got != 64 {
		t.Errorf("body = %d bytes, want 64", got)
	}

	conflict := service.do(t, http.MethodPost, path, token, body, contentType)
	assertStatusAndText(t, conflict, http.StatusConflict,
		fmt.Sprintf("Icon already exists for profile with id: %d", profile.ID))

	deleted := service.do(t, http.MethodDelete, path, token, nil, "")
	if deleted.Code != http.StatusNoContent {
		t.Errorf("delete = %d, want %d", deleted.Code, http.StatusNoContent)
	}
}

func TestAnUploadOverTheMultipartCapIsRefused(t *testing.T) {
	// The cap applies to the whole request as well as the file, so a 2 MB file
	// plus its part headers is already over.
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	profile := createProfileFor(t, service.store, seedUser(t, pool, "icon-oversize@jdw.com"))
	body, contentType := iconMultipart(t, "icon", maxUploadBytes)

	response := service.do(t, http.MethodPost,
		fmt.Sprintf("/api/profiles/%d/icon", profile.ID), token, body, contentType)

	assertStatusAndText(t, response, http.StatusBadRequest, "Maximum upload size exceeded")
}

func TestAnUploadJustUnderTheCapIsAccepted(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	profile := createProfileFor(t, service.store, seedUser(t, pool, "icon-just-under@jdw.com"))
	// Room for the part headers and the closing boundary inside the same cap.
	body, contentType := iconMultipart(t, "icon", maxUploadBytes-1024)

	response := service.do(t, http.MethodPost,
		fmt.Sprintf("/api/profiles/%d/icon", profile.ID), token, body, contentType)

	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d (body %q)", response.Code, http.StatusOK, response.Body.String())
	}
}

func TestAnUploadUnderTheWrongPartNameIsRefused(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	profile := createProfileFor(t, service.store, seedUser(t, pool, "icon-wrong-part@jdw.com"))
	body, contentType := iconMultipart(t, "avatar", 64)

	response := service.do(t, http.MethodPost,
		fmt.Sprintf("/api/profiles/%d/icon", profile.ID), token, body, contentType)

	assertStatusAndText(t, response, http.StatusBadRequest, "Required part 'icon' is not present.")
}

func TestReplacingAnIconOnAProfileWithNoneIsTheFrozenFiveHundred(t *testing.T) {
	service := newLiveService(t)
	token := service.adminToken(t)
	_, pool := newTestStore(t)
	profile := createProfileFor(t, service.store, seedUser(t, pool, "icon-replace-none@jdw.com"))
	body, contentType := iconMultipart(t, "icon", 64)

	response := service.do(t, http.MethodPut,
		fmt.Sprintf("/api/profiles/%d/icon", profile.ID), token, body, contentType)

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

func TestTheProfileFallbackAdmitsAnOwnerWhoseTokenPredatesTheirProfile(t *testing.T) {
	// Against real storage rather than a stub: the lookup this depends on is one
	// indexed read on auth.profiles.user_id.
	service := newLiveService(t)
	_, pool := newTestStore(t)
	ownerID := seedUser(t, pool, "fallback-http@jdw.com")
	profile := createProfileFor(t, service.store, ownerID)
	token := mint(t, authtest.Claims{
		Subject: "fallback@jdw.com", Roles: []string{"USER"}, UserID: &ownerID,
	})

	allowed := service.getJSON(t, fmt.Sprintf("/api/profiles/%d", profile.ID), token)
	if allowed.Code != http.StatusOK {
		t.Errorf("the owner was refused their own profile: %d %s", allowed.Code, allowed.Body.String())
	}

	// The fallback resolves the caller's own profile, so a stranger's id is
	// still refused rather than served.
	stranger := createProfileFor(t, service.store, seedUser(t, pool, "fallback-stranger@jdw.com"))
	refused := service.getJSON(t, fmt.Sprintf("/api/profiles/%d", stranger.ID), token)
	if refused.Code != http.StatusForbidden {
		t.Errorf("a stranger's profile answered %d, want %d", refused.Code, http.StatusForbidden)
	}
}

func TestTheOperationalEndpointsAreServedWithoutAToken(t *testing.T) {
	// SecurityConfig permits /actuator/** without authentication, and the chart's
	// probes and the scrape configuration depend on it staying that way.
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
	service.getJSON(t, "/api/profiles/987654", service.adminToken(t))

	response := service.getJSON(t, actuatorMetricsPath, "")

	body := response.Body.String()
	if !strings.Contains(body, servicehttp.RequestDurationName+"_bucket") {
		t.Error("the scrape carries no histogram buckets; the percentile panels would be unfillable")
	}
	if !strings.Contains(body, `uri="/api/profiles/{profileId}"`) {
		t.Error("the uri label is not the route pattern; every id would open its own series")
	}
	if !strings.Contains(body, `outcome="CLIENT_ERROR"`) {
		t.Error("the outcome label is missing; a query written against the JVM series would not select")
	}
}

func TestAPreflightIsAnsweredAheadOfAuthenticationOnALiveRoute(t *testing.T) {
	service := newLiveService(t)
	request := httptest.NewRequest(http.MethodOptions, "/api/profiles/1", nil)
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
	_, pool := newTestStore(t)
	stranger := seedUser(t, pool, "forbidden-path@jdw.com")
	token := mint(t, authtest.Claims{
		Subject: "stranger@jdw.com", Roles: []string{"USER"},
		UserID: &stranger, ProfileID: &stranger,
	})

	response := service.getJSON(t, "/api/profiles/424242", token)

	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d (body %q)", response.Code, http.StatusForbidden, response.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("the 403 body is not JSON: %v", err)
	}
	if body["path"] != "/api/profiles/424242" {
		t.Errorf("path = %v, want the requested path", body["path"])
	}
}

// errStoreUnavailable stands in for any storage failure, so the handlers' 500
// paths are driven with the same bodies as their success paths.
var errStoreUnavailable = errors.New("the store is unavailable")

// brokenStore reports failure from every method, mirroring identity-service's
// double of the same name: the two services answer a storage failure
// identically, so they are driven against the same shape of store.
type brokenStore struct{}

func (brokenStore) ListProfiles(context.Context, int, int) ([]Profile, error) {
	return nil, errStoreUnavailable
}

func (brokenStore) ProfileByID(context.Context, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) ProfileByUserID(context.Context, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) ProfileIDForUser(context.Context, int64) (int64, bool, error) {
	return 0, false, errStoreUnavailable
}

func (brokenStore) CreateProfile(context.Context, ProfileCreateRequest, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) UpdateProfileByID(context.Context, int64, ProfileUpdateRequest, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) UpdateProfileByUserID(context.Context, int64, ProfileUpdateRequest, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) DeleteProfileByID(context.Context, int64) error     { return errStoreUnavailable }
func (brokenStore) DeleteProfileByUserID(context.Context, int64) error { return errStoreUnavailable }

func (brokenStore) AddAddress(context.Context, int64, AddressRequest, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) UpdateAddress(context.Context, int64, int64, AddressRequest, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) DeleteAddress(context.Context, int64, int64) error { return errStoreUnavailable }

func (brokenStore) Icon(context.Context, int64) (ProfileIcon, error) {
	return ProfileIcon{}, errStoreUnavailable
}

func (brokenStore) AddIcon(context.Context, int64, []byte, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) ReplaceIcon(context.Context, int64, []byte, int64) (Profile, error) {
	return Profile{}, errStoreUnavailable
}

func (brokenStore) DeleteIcon(context.Context, int64) error { return errStoreUnavailable }

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
			request := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
			request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))
			response := httptest.NewRecorder()

			server.ServeHTTP(response, request)

			assertContainerErrorBody(t, response, http.StatusInternalServerError, "/api/profiles")
		})
	}
}

func TestARuleThatCannotBeDecidedCarriesTheContainerErrorBody(t *testing.T) {
	// The profile fallback reads storage to decide a rule, so it is the one
	// authorization decision an outage can take away. It answers 500 rather
	// than 403 — a failed lookup says nothing about the caller's rights — and
	// that 500 is a status the container sets, so it carries the same body as
	// any other. The caller here holds a verified token with no profile_id
	// claim, which is what sends the rule to the fallback.
	server := parityServer(t, brokenStore{})
	userID := int64(4242)
	token := mint(t, authtest.Claims{Subject: "no-claim@jdw.com", Roles: []string{"USER"}, UserID: &userID})
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/7", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()

	server.ServeHTTP(response, request)

	assertContainerErrorBody(t, response, http.StatusInternalServerError, "/api/profiles/7")
	if got := response.Header().Get("Access-Denied-Reason"); got != "" {
		t.Errorf("Access-Denied-Reason = %q; a lookup failure is not a refusal", got)
	}
}
