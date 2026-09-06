package authhttp_test

import (
	"context"
	"errors"
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

func middleware(t *testing.T) *authhttp.Middleware {
	t.Helper()
	m, err := authhttp.NewMiddleware(verifier(t))
	if err != nil {
		t.Fatalf("NewMiddleware: %v", err)
	}
	return m
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

// assertRefusalShape pins what the deployed service actually sends: the status,
// the reason header, and nothing else. Measured against a booted usersrole on a
// real port, across Accept values of application/json, */* and text/html.
func assertRefusalShape(t *testing.T, recorder *httptest.ResponseRecorder, status int, reason string) {
	t.Helper()
	if recorder.Code != status {
		t.Errorf("status = %d, want %d", recorder.Code, status)
	}
	if got := recorder.Header().Get(authhttp.HeaderAccessDeniedReason); got != reason {
		t.Errorf("%s = %q, want %q", authhttp.HeaderAccessDeniedReason, got, reason)
	}
	if body := recorder.Body.String(); body != "" {
		t.Errorf("body = %q, want empty; the JVM refuses with Content-Length 0", body)
	}
	if got := recorder.Header().Get("Content-Type"); got != "" {
		t.Errorf("Content-Type = %q, want none", got)
	}
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

	middleware(t).Handler(next).ServeHTTP(recorder, request)

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

			middleware(t).Handler(next).ServeHTTP(recorder, request)

			if reached {
				t.Error("the handler ran for an unauthenticated request")
			}
			assertRefusalShape(t, recorder, http.StatusUnauthorized, "Authentication Required")
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

	m := middleware(t)
	m.Public = func(r *http.Request) bool { return r.URL.Path == "/auth/authenticate" }
	m.Handler(next).ServeHTTP(recorder, request)

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

	m := middleware(t)
	m.Public = func(*http.Request) bool { return true }
	m.Handler(next).ServeHTTP(recorder, request)

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

	m := middleware(t)
	m.Public = func(*http.Request) bool { return true }
	m.Handler(next).ServeHTTP(recorder, request)

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

	m := middleware(t)
	m.OnError = func(_ *http.Request, err error) { reported = err }
	m.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(recorder, request)

	if reported == nil {
		t.Fatal("OnError was not called")
	}
}

// The JVM filter returns early on a request with no bearer token, without
// logging: an anonymous request is ordinary traffic, not a failure. Reporting
// them would bury the tokens that actually failed to verify.
func TestMiddlewareDoesNotReportRequestsThatCarriedNoToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
	}{
		{"no authorization header", ""},
		{"non-bearer scheme", "Basic Zm9vOmJhcg=="},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			called := false
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/api/users/42", nil)
			if tc.header != "" {
				request.Header.Set("Authorization", tc.header)
			}

			m := middleware(t)
			m.OnError = func(*http.Request, error) { called = true }
			m.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(recorder, request)

			if called {
				t.Error("OnError was called for a request that presented no token")
			}
			if recorder.Code != http.StatusUnauthorized {
				t.Errorf("status = %d, want 401", recorder.Code)
			}
		})
	}
}

func preflightRequest() *http.Request {
	request := httptest.NewRequest(http.MethodOptions, "/api/users/42", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	request.Header.Set("Access-Control-Request-Method", "GET")
	return request
}

// A browser never attaches Authorization to a preflight, and Spring's CorsFilter
// answers it ahead of the JWT filter. Measured on a booted usersrole: an
// unauthenticated preflight returns 200. Refusing it would break every
// cross-origin call the frontends make at cutover.
func TestMiddlewareHandsCorsPreflightToTheCorsHandler(t *testing.T) {
	handledPreflight := false
	reachedNext := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reachedNext = true })
	recorder := httptest.NewRecorder()

	m := middleware(t)
	m.Preflight = http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		handledPreflight = true
		w.WriteHeader(http.StatusOK)
	})
	m.Handler(next).ServeHTTP(recorder, preflightRequest())

	if !handledPreflight {
		t.Error("the preflight did not reach the CORS handler")
	}
	if reachedNext {
		t.Error("the preflight reached the wrapped handler; an unauthenticated request must not get there")
	}
	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", recorder.Code)
	}
}

// The exemption is opt-in. Without a CORS handler the middleware is the inner
// layer, the CORS layer outside it has already answered the preflight, and
// anything still arriving here is authenticated like any other request.
func TestMiddlewareAuthenticatesPreflightWhenNoCorsHandlerIsSet(t *testing.T) {
	reached := false
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reached = true })
	recorder := httptest.NewRecorder()

	middleware(t).Handler(next).ServeHTTP(recorder, preflightRequest())

	if reached {
		t.Error("a preflight reached the handler with no principal and no CORS handler configured")
	}
	assertRefusalShape(t, recorder, http.StatusUnauthorized, "Authentication Required")
}

// Spring's CorsUtils.isPreFlightRequest needs all three conditions. A request
// missing any of them is not a preflight, and must not be able to use the
// exemption to reach a handler without a token.
func TestMiddlewareOnlyExemptsARealPreflight(t *testing.T) {
	tests := []struct {
		name    string
		method  string
		headers map[string]string
	}{
		{"plain options", http.MethodOptions, nil},
		{"options without origin", http.MethodOptions, map[string]string{"Access-Control-Request-Method": "GET"}},
		{"options without the request-method header", http.MethodOptions, map[string]string{"Origin": "http://localhost:4200"}},
		{"get carrying preflight headers", http.MethodGet, map[string]string{
			"Origin": "http://localhost:4200", "Access-Control-Request-Method": "GET",
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			reachedNext, reachedPreflight := false, false
			next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reachedNext = true })
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tc.method, "/api/users/42", nil)
			for name, value := range tc.headers {
				request.Header.Set(name, value)
			}

			m := middleware(t)
			m.Preflight = http.HandlerFunc(func(http.ResponseWriter, *http.Request) { reachedPreflight = true })
			m.Handler(next).ServeHTTP(recorder, request)

			if reachedNext || reachedPreflight {
				t.Error("the request was treated as a preflight and skipped authentication")
			}
			assertRefusalShape(t, recorder, http.StatusUnauthorized, "Authentication Required")
		})
	}
}

func TestNewMiddlewareRejectsANilVerifier(t *testing.T) {
	_, err := authhttp.NewMiddleware(nil)

	if !errors.Is(err, authhttp.ErrNoVerifier) {
		t.Errorf("error = %v, want %v", err, authhttp.ErrNoVerifier)
	}
}

func TestWriteForbiddenMatchesTheAccessDeniedHandler(t *testing.T) {
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodDelete, "/api/profiles/7", nil)

	authhttp.WriteForbidden(recorder, request)

	assertRefusalShape(t, recorder, http.StatusForbidden, "Not Authorized")
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
	assertRefusalShape(t, recorder, http.StatusForbidden, "Not Authorized")
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
	assertRefusalShape(t, recorder, http.StatusUnauthorized, "Authentication Required")
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
}
