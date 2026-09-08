package main

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"libs/backend/shared/auth/authtest"
)

// bcryptPrefixes are the three 2-family prefixes Spring's encoder emits and this
// service accepts, so a hash that reached a log line is recognisable whichever
// one produced it.
var bcryptPrefixes = []string{"$2a$", "$2b$", "$2y$"}

// captureTheServiceLog redirects the process-wide default logger into a buffer,
// through the same JSON handler main installs.
//
// The default is swapped rather than a logger threaded through ServerConfig
// because of what this guards against: a bare slog.Warn added to a handler,
// dereferencing UserRequest.Password past the redaction its type carries. A
// logger the server held would not see that call, and the guard would be
// decorative again. Debug level, because a line written below the default
// threshold still reaches a deployment that lowers it.
func captureTheServiceLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var written bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&written, &slog.HandlerOptions{Level: slog.LevelDebug})))
	t.Cleanup(func() { slog.SetDefault(previous) })
	return &written
}

// errStoreUnavailable stands in for any storage failure, so the handlers' 500
// paths are driven with the same bodies as their success paths. A store error
// that quoted the value it was handed would put a bcrypt hash into the line
// fail writes.
var errStoreUnavailable = errors.New("the store is unavailable")

type brokenStore struct{}

func (brokenStore) ListUsers(context.Context, int, int) ([]User, error) {
	return nil, errStoreUnavailable
}

func (brokenStore) UserByID(context.Context, int64) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) UserByEmailAddress(context.Context, string) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) UserExists(context.Context, string) (bool, error) {
	return false, errStoreUnavailable
}

func (brokenStore) CredentialByEmailAddress(context.Context, string) (Credential, error) {
	return Credential{}, errStoreUnavailable
}

func (brokenStore) CreateUser(context.Context, string, string, int64) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) UpdateUser(context.Context, int64, string, string, int64) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) DeleteUser(context.Context, int64) error { return errStoreUnavailable }

func (brokenStore) UserHasAnyRole(context.Context, int64, []int64) (bool, error) {
	return false, errStoreUnavailable
}

func (brokenStore) GrantRolesToUser(context.Context, int64, []int64, int64) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) RevokeRolesFromUser(context.Context, int64, []int64, int64) (User, error) {
	return User{}, errStoreUnavailable
}

func (brokenStore) ListRoles(context.Context, int, int) ([]Role, error) {
	return nil, errStoreUnavailable
}

func (brokenStore) RoleByID(context.Context, int64) (Role, error) {
	return Role{}, errStoreUnavailable
}

func (brokenStore) RoleByName(context.Context, string) (Role, error) {
	return Role{}, errStoreUnavailable
}

func (brokenStore) CreateRole(context.Context, string, string, int64) (Role, error) {
	return Role{}, errStoreUnavailable
}

func (brokenStore) UpdateRole(context.Context, int64, string, string, int64) (Role, error) {
	return Role{}, errStoreUnavailable
}

func (brokenStore) DeleteRole(context.Context, int64) error { return errStoreUnavailable }

func (brokenStore) GrantUsersToRole(context.Context, int64, []int64, int64) (Role, error) {
	return Role{}, errStoreUnavailable
}

func (brokenStore) RevokeUsersFromRole(context.Context, int64, []int64, int64) (Role, error) {
	return Role{}, errStoreUnavailable
}

// mismatchedPassword is the wrong password for every address the fixture knows.
// It shares no substring with the fixture password, so a line carrying one
// cannot be mistaken for a line carrying the other.
const mismatchedPassword = "Nemesis7?" // gitleaks:allow

// drivenPasswords is every cleartext this suite sends. Asserting on the set
// rather than on one value means a handler that logs whichever password it was
// handed is caught wherever in the surface it does it.
var drivenPasswords = []string{fixturePassword, mismatchedPassword}

// credentialBearingBody carries the fixture password under the field name the
// request DTO declares, so a handler that logged the decoded body — or the raw
// one — would disclose it. It is deliberately not a valid request: it drives the
// validation refusal with a real password in hand.
const credentialBearingBody = `{"emailAddress":"not-an-address","password":"` + fixturePassword + `"}`

func TestNoLineTheServiceLogsCarriesACredential(t *testing.T) {
	// The redactions on Credential and UserRequest protect those types, and
	// nothing forces a handler to log either one: UserRequest.Password is a
	// *string, and dereferencing it into a log call walks straight past both
	// LogValue and String. So the assertion is over what the service actually
	// writes while every operation is driven, not over a hand-built struct.
	written := captureTheServiceLog(t)

	driveEveryOperation(t, parityServer(t, stubStore{}))
	driveEveryOperation(t, parityServer(t, brokenStore{}))
	driveTheSignInRefusals(t, parityServer(t, stubStore{}))

	logged := written.String()
	// Without this the assertions below would pass on an empty buffer, which is
	// exactly what a capture that stopped working would produce.
	if !strings.Contains(logged, "a sign-in was refused") {
		t.Fatalf("the capture holds no line the service is known to write: %s", logged)
	}
	if !strings.Contains(logged, "the request could not be served") {
		t.Fatalf("the capture holds no failure line, so no error path was driven: %s", logged)
	}
	assertNoCredentialIn(t, logged)
}

// assertNoCredentialIn reports the offending line rather than the whole capture,
// which runs to hundreds of lines and would bury the one that matters.
func assertNoCredentialIn(t *testing.T, logged string) {
	t.Helper()
	for line := range strings.SplitSeq(strings.TrimSpace(logged), "\n") {
		for _, password := range drivenPasswords {
			if strings.Contains(line, password) {
				t.Errorf("a log line carries the cleartext password %q: %s", password, line)
			}
		}
		for _, prefix := range bcryptPrefixes {
			if strings.Contains(line, prefix) {
				t.Errorf("a log line carries a bcrypt hash (%s): %s", prefix, line)
			}
		}
	}
}

// driveEveryOperation walks the whole served surface through every outcome an
// operation has: served, refused for the rule, refused for want of a token,
// refused for a token that does not verify, and refused for a body that does not
// validate. Each is a place a handler could log what it was sent.
func driveEveryOperation(t *testing.T, server http.Handler) {
	t.Helper()
	tampered := authtest.TamperSignature(mint(t, admin().claims))

	for _, tc := range parityCases() {
		for _, who := range allowedBy(tc.rule) {
			exchange(t, server, tc, tokenFor(t, who))
		}
		for _, who := range deniedBy(tc.rule) {
			exchange(t, server, tc, tokenFor(t, who))
		}
		exchange(t, server, tc, "")
		exchange(t, server, tc, tampered)

		if tc.body == nil {
			continue
		}
		token := tokenFor(t, allowedBy(tc.rule)[0])
		post(t, server, tc.method, tc.path, token, credentialBearingBody)
		post(t, server, tc.method, tc.path, token, `{"emailAddress":`)
	}
}

// driveTheSignInRefusals covers the two outcomes the parity cases cannot reach,
// because both need credentials the fixture rejects. They are the paths most
// likely to grow a diagnostic log line later.
func driveTheSignInRefusals(t *testing.T, server http.Handler) {
	t.Helper()
	post(t, server, http.MethodPost, "/auth/authenticate", "",
		`{"emailAddress":"`+selfEmail+`","password":"`+mismatchedPassword+`"}`)
	post(t, server, http.MethodPost, "/auth/authenticate", "",
		`{"emailAddress":"nobody@jdw.com","password":"`+fixturePassword+`"}`)
}

func tokenFor(t *testing.T, who principal) string {
	t.Helper()
	if who.claims.Subject == "" {
		return ""
	}
	return mint(t, who.claims)
}

func exchange(t *testing.T, server http.Handler, tc parityCase, token string) {
	t.Helper()
	server.ServeHTTP(httptest.NewRecorder(), tc.request(t, token))
}

func post(t *testing.T, server http.Handler, method, path, token, body string) {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	server.ServeHTTP(httptest.NewRecorder(), request)
}
