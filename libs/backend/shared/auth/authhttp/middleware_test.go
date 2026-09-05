package authhttp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authhttp"
	"libs/backend/shared/auth/authtest"
	"libs/backend/shared/auth/authz"
)

const paritySecret = "bXl0dGVzdHNlY3JldGtleWZvcmpzb253d2VidG9rZW4xMjM0NTY3ODkwIC1uCg=="

const (
	issuerOrigin = "http://localhost:8080"
	issuerClaim  = issuerOrigin + "/auth/authenticate"
)

var mintedAt = time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)

func ptr(v int64) *int64 { return &v }

func minter() authtest.Minter {
	return authtest.Minter{
		SecretKeyBase64: paritySecret,
		IssuerOrigin:    issuerOrigin,
		Now:             func() time.Time { return mintedAt },
	}
}

func verifier(t *testing.T) *auth.Verifier {
	t.Helper()
	v, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  paritySecret,
		ExpectedIssuer:   issuerClaim,
		ExpectedAudience: issuerOrigin,
		Now:              func() time.Time { return mintedAt.Add(time.Minute) },
	})
	if err != nil {
		t.Fatalf("NewVerifier: %v", err)
	}
	return v
}

func validToken(t *testing.T) string {
	t.Helper()
	token, err := minter().Mint(authtest.Claims{
		Subject: "user@jdw.com", Roles: []string{"ADMIN"}, UserID: ptr(42), ProfileID: ptr(7),
	})
	if err != nil {
		t.Fatalf("Mint: %v", err)
	}
	return token
}

type containerError struct {
	Timestamp string `json:"timestamp"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

func decodeBody(t *testing.T, res *http.Response) containerError {
	t.Helper()
	var body containerError
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return body
}

func TestMiddlewarePassesAVerifiedPrincipalToTheHandler(t *testing.T) {
	var got *auth.Principal
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		principal, ok := authhttp.PrincipalFrom(r.Context())
		if !ok {
			t.Error("no principal in the request context")
		}
		got = principal
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))

	authhttp.Middleware{Verifier: verifier(t)}.Handler(next).ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got == nil {
		t.Fatal("handler saw no principal")
	}
	if got.Subject != "user@jdw.com" {
		t.Errorf("Subject = %q, want user@jdw.com", got.Subject)
	}
	if got.UserID == nil || *got.UserID != 42 {
		t.Errorf("UserID = %v, want 42", got.UserID)
	}
}

func TestMiddlewareAnswersTheEntryPointShapeOnEveryAuthenticationFailure(t *testing.T) {
	tests := []struct {
		name   string
		header func(t *testing.T) string
	}{
		{"no authorization header", func(*testing.T) string { return "" }},
		{"wrong scheme", func(t *testing.T) string { return "Basic " + validToken(t) }},
		{"garbage token", func(*testing.T) string { return "Bearer not-a-token" }},
		{"tampered signature", func(t *testing.T) string { return "Bearer " + authtest.TamperSignature(validToken(t)) }},
		{"expired", func(t *testing.T) string {
			token, err := minter().MintRaw("HS256", map[string]any{
				"sub": "user@jdw.com", "roles": []string{}, "user_id": 42, "profile_id": nil,
				"iss": issuerClaim, "aud": issuerOrigin, "jti": "id",
				"nbf": mintedAt.Unix(), "exp": mintedAt.Add(-time.Second).Unix(),
			})
			if err != nil {
				t.Fatalf("MintRaw: %v", err)
			}
			return "Bearer " + token
		}},
		{"wrong audience", func(t *testing.T) string {
			other := authtest.Minter{SecretKeyBase64: paritySecret, IssuerOrigin: "http://evil.example", Now: func() time.Time { return mintedAt }}
			token, err := other.Mint(authtest.Claims{Subject: "user@jdw.com", UserID: ptr(42)})
			if err != nil {
				t.Fatalf("Mint: %v", err)
			}
			return "Bearer " + token
		}},
		{"algorithm none", func(t *testing.T) string {
			token, err := minter().MintRaw("none", map[string]any{
				"sub": "user@jdw.com", "roles": []string{"ADMIN"}, "user_id": 42,
				"iss": issuerClaim, "aud": issuerOrigin, "jti": "id",
				"nbf": mintedAt.Unix(), "exp": mintedAt.Add(time.Hour).Unix(),
			})
			if err != nil {
				t.Fatalf("MintRaw: %v", err)
			}
			return "Bearer " + token
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
			if header := tc.header(t); header != "" {
				request.Header.Set("Authorization", header)
			}

			authhttp.Middleware{Verifier: verifier(t)}.Handler(next).ServeHTTP(recorder, request)

			if reached {
				t.Error("the handler ran for an unauthenticated request")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", recorder.Code)
			}
			if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != "Authentication Required" {
				t.Errorf("%s = %q, want %q", authhttp.HeaderAccessDeniedReason, got, "Authentication Required")
			}
			if got := recorder.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q, want application/json", got)
			}
			body := decodeBody(t, recorder.Result())
			if body.Status != http.StatusUnauthorized {
				t.Errorf("body status = %d, want 401", body.Status)
			}
			if body.Error != "Unauthorized" {
				t.Errorf("body error = %q, want Unauthorized", body.Error)
			}
			if body.Message != "Access Denied Full authentication is required to access this resource" {
				t.Errorf("body message = %q", body.Message)
			}
			if body.Path != "/api/users/42" {
				t.Errorf("body path = %q, want /api/users/42", body.Path)
			}
			if _, err := time.Parse("2006-01-02T15:04:05.000-07:00", body.Timestamp); err != nil {
				t.Errorf("body timestamp %q is not the shape the container writes: %v", body.Timestamp, err)
			}
		})
	}
}

func TestMiddlewareLeavesPublicRoutesAlone(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached = true
		if _, ok := authhttp.PrincipalFrom(r.Context()); ok {
			t.Error("a principal reached a public route without a token")
		}
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/authenticate", nil)

	middleware := authhttp.Middleware{
		Verifier: verifier(t),
		Public:   func(r *http.Request) bool { return r.URL.Path == "/auth/authenticate" },
	}
	middleware.Handler(next).ServeHTTP(recorder, request)

	if !reached {
		t.Error("the handler did not run for a public route")
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
}

func TestMiddlewareStillVerifiesATokenOnAPublicRoute(t *testing.T) {
	// The JVM filter runs ahead of the permitAll matchers, so a public
	// operation that reads the caller (user creation audits by token) sees the
	// same principal it would on a guarded route.
	var got *auth.Principal
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got, _ = authhttp.PrincipalFrom(r.Context())
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/authenticate", nil)
	request.Header.Set("Authorization", "Bearer "+validToken(t))

	middleware := authhttp.Middleware{Verifier: verifier(t), Public: func(*http.Request) bool { return true }}
	middleware.Handler(next).ServeHTTP(recorder, request)

	if got == nil {
		t.Fatal("a valid token on a public route produced no principal")
	}
	if got.Subject != "user@jdw.com" {
		t.Errorf("Subject = %q, want user@jdw.com", got.Subject)
	}
}

func TestMiddlewareLetsAnInvalidTokenThroughOnAPublicRoute(t *testing.T) {
	// Matching the filter: a token it cannot verify leaves the context
	// unauthenticated, and the permitAll matcher then decides the request.
	reached := false
	next := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		reached = true
		if _, ok := authhttp.PrincipalFrom(r.Context()); ok {
			t.Error("an unverifiable token produced a principal")
		}
	})
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/auth/authenticate", nil)
	request.Header.Set("Authorization", "Bearer not-a-token")

	middleware := authhttp.Middleware{Verifier: verifier(t), Public: func(*http.Request) bool { return true }}
	middleware.Handler(next).ServeHTTP(recorder, request)

	if !reached {
		t.Error("the handler did not run for a public route")
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
}

func TestMiddlewareReportsVerificationFailures(t *testing.T) {
	var reported error
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	request.Header.Set("Authorization", "Bearer "+authtest.TamperSignature(validToken(t)))

	middleware := authhttp.Middleware{
		Verifier: verifier(t),
		OnError:  func(_ *http.Request, err error) { reported = err },
	}
	middleware.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(recorder, request)

	if reported == nil {
		t.Fatal("OnError was not called")
	}
}

func TestWriteForbiddenMatchesTheAccessDeniedHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/profiles/7", nil)

	authhttp.WriteForbidden(recorder, request)

	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", recorder.Code)
	}
	if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != "Not Authorized" {
		t.Errorf("%s = %q, want %q", authhttp.HeaderAccessDeniedReason, got, "Not Authorized")
	}
	body := decodeBody(t, recorder.Result())
	if body.Status != http.StatusForbidden {
		t.Errorf("body status = %d, want 403", body.Status)
	}
	if body.Error != "Forbidden" {
		t.Errorf("body error = %q, want Forbidden", body.Error)
	}
	if body.Message != "Access Denied Access is denied" {
		t.Errorf("body message = %q", body.Message)
	}
	if body.Path != "/api/profiles/7" {
		t.Errorf("body path = %q, want /api/profiles/7", body.Path)
	}
}

func TestPrincipalFromReportsAnEmptyContext(t *testing.T) {
	if _, ok := authhttp.PrincipalFrom(context.Background()); ok {
		t.Error("PrincipalFrom found a principal in an empty context")
	}
}

func TestAuthorizeWritesTheAccessDeniedShapeOnDenial(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/99", nil)
	principal := &auth.Principal{Subject: "user@jdw.com", Roles: []string{"USER"}, UserID: ptr(42)}
	request = request.WithContext(authhttp.WithPrincipal(request.Context(), principal))

	allowed := authhttp.Authorize(recorder, request, authz.Authorizer{}, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: ptr(99)})

	if allowed {
		t.Fatal("Authorize allowed a caller reading another user")
	}
	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", recorder.Code)
	}
	if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != "Not Authorized" {
		t.Errorf("%s = %q, want Not Authorized", authhttp.HeaderAccessDeniedReason, got)
	}
}

func TestAuthorizeWritesNothingWhenItAllows(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
	principal := &auth.Principal{Subject: "user@jdw.com", Roles: []string{"USER"}, UserID: ptr(42)}
	request = request.WithContext(authhttp.WithPrincipal(request.Context(), principal))

	allowed := authhttp.Authorize(recorder, request, authz.Authorizer{}, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: ptr(42)})

	if !allowed {
		t.Fatal("Authorize denied a caller reading themselves")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 {
		t.Errorf("Authorize wrote a response: status %d, body %q", recorder.Code, recorder.Body.String())
	}
}

func TestAuthorizeAnswersUnauthorizedWithoutAPrincipal(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)

	allowed := authhttp.Authorize(recorder, request, authz.Authorizer{}, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: ptr(42)})

	if allowed {
		t.Fatal("Authorize allowed an anonymous caller")
	}
	if recorder.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", recorder.Code)
	}
	if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != "Authentication Required" {
		t.Errorf("%s = %q, want Authentication Required", authhttp.HeaderAccessDeniedReason, got)
	}
}

func TestAuthorizeAnswersServerErrorWhenTheRuleCannotBeDecided(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/7", nil)
	principal := &auth.Principal{Subject: "user@jdw.com", Roles: []string{"USER"}, UserID: ptr(42)}
	request = request.WithContext(authhttp.WithPrincipal(request.Context(), principal))
	failing := authz.Authorizer{ProfileIDForUser: func(context.Context, int64) (int64, bool, error) {
		return 0, false, context.DeadlineExceeded
	}}

	allowed := authhttp.Authorize(recorder, request, failing, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: ptr(7)})

	if allowed {
		t.Fatal("Authorize allowed a request whose rule could not be decided")
	}
	if recorder.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500; an undecidable rule is not a denial", recorder.Code)
	}
	if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != "" {
		t.Errorf("%s = %q; a lookup failure says nothing about the caller's rights", authhttp.HeaderAccessDeniedReason, got)
	}
	body := decodeBody(t, recorder.Result())
	if body.Status != http.StatusInternalServerError || body.Error != "Internal Server Error" {
		t.Errorf("body = %+v, want the container error shape for 500", body)
	}
	if body.Path != "/api/profiles/7" {
		t.Errorf("body path = %q, want /api/profiles/7", body.Path)
	}
}
