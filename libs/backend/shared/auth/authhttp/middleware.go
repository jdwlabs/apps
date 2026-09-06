// Package authhttp wires the verifier and the authorization rules into
// net/http, and refuses requests the way the Spring application refuses them:
// the same status, the same Access-Denied-Reason header, and the same empty
// body.
//
// The empty body is measured, not assumed. Both Spring handlers call
// sendError with a message, but server.error.include-message is unset, so
// Boot's default of "never" applies and the deployed service answers 401 and
// 403 with Content-Length 0 and no Content-Type, whatever the request's Accept
// header. The frozen contract documents a ContainerError JSON body for those
// statuses; the running application does not produce one. Reproducing what the
// application does is what keeps the cutover invisible, and the contract itself
// records that clients key off the status and the header rather than the body.
package authhttp

import (
	"context"
	"errors"
	"net/http"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authz"
)

// HeaderAccessDeniedReason is the header both Spring handlers add, and the one
// the contracts mark required on every 401 and 403.
const HeaderAccessDeniedReason = "Access-Denied-Reason"

// The two values that header takes, from CustomAuthenticationEntryPoint and
// CustomAccessDeniedHandler.
const (
	ReasonAuthenticationRequired = "Authentication Required"
	ReasonNotAuthorized          = "Not Authorized"
)

type contextKey struct{}

var principalKey contextKey

// ErrNoVerifier is returned by NewMiddleware rather than left to surface as a
// nil dereference on the first request, which would take a service down after
// it had already reported itself healthy.
var ErrNoVerifier = errors.New("middleware needs a verifier")

// Middleware verifies the bearer token on each request and puts the principal
// in the request context. Build it with NewMiddleware.
type Middleware struct {
	verifier *auth.Verifier
	// Public reports whether a request bypasses the authentication requirement,
	// standing in for SecurityConfig's permitAll matchers. A public request
	// whose token verifies still carries its principal, because the JVM filter
	// runs ahead of those matchers and some public operations read the caller.
	// Nil makes every route require a token.
	Public func(*http.Request) bool
	// OnError observes a token that was presented and failed to verify. It is
	// not called for a request that carried no bearer token at all: the JVM
	// filter returns early on those without logging, and treating every
	// anonymous request as an error would bury the real failures.
	OnError func(*http.Request, error)
	// Preflight answers CORS preflight requests, which carry no Authorization
	// header and so cannot be authenticated. Leaving it nil authenticates them
	// like anything else, which is correct when the CORS layer is mounted
	// outside this middleware — the ordering Spring uses, where CorsFilter runs
	// ahead of the JWT filter and answers the preflight before it arrives.
	//
	// Set it only when this middleware is the outer layer. The preflight is
	// handed to this handler and never to the wrapped one, so a request shaped
	// like a preflight cannot reach a business handler without a principal.
	Preflight http.Handler
}

// NewMiddleware returns a Middleware, refusing a nil verifier.
func NewMiddleware(verifier *auth.Verifier) (*Middleware, error) {
	if verifier == nil {
		return nil, ErrNoVerifier
	}
	return &Middleware{verifier: verifier}, nil
}

// Handler wraps next with authentication.
func (m *Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// A CORS preflight carries no Authorization header — browsers never put
		// one on it — so authenticating it would refuse every cross-origin
		// request the frontends make. It goes to the configured CORS handler
		// rather than to next: an unauthenticated request must not reach a
		// handler just because it is shaped like a preflight.
		if m.Preflight != nil && isCorsPreflight(r) {
			m.Preflight.ServeHTTP(w, r)
			return
		}

		public := m.Public != nil && m.Public(r)

		principal, err := m.verifier.VerifyAuthorizationHeader(r.Header.Get("Authorization"))
		if err != nil {
			if m.OnError != nil && !errors.Is(err, auth.ErrMissingBearerToken) {
				m.OnError(r, err)
			}
			if !public {
				WriteUnauthorized(w, r)
				return
			}
			next.ServeHTTP(w, r)
			return
		}

		next.ServeHTTP(w, r.WithContext(WithPrincipal(r.Context(), principal)))
	})
}

// isCorsPreflight matches the three conditions Spring's
// CorsUtils.isPreFlightRequest requires, so the two agree on what a preflight
// is: the OPTIONS method, an Origin, and the request-method header a browser
// sends only on a preflight.
func isCorsPreflight(r *http.Request) bool {
	return r.Method == http.MethodOptions &&
		r.Header.Get("Origin") != "" &&
		r.Header.Get("Access-Control-Request-Method") != ""
}

// WithPrincipal stores a verified principal in a context. Exported so a test
// or a handler composed outside this middleware can build the same context.
func WithPrincipal(ctx context.Context, p *auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey, p)
}

// PrincipalFrom returns the verified principal a request carries.
func PrincipalFrom(ctx context.Context) (*auth.Principal, bool) {
	principal, ok := ctx.Value(principalKey).(*auth.Principal)
	return principal, ok && principal != nil
}

// Authorize decides one rule for the principal on the request and writes the
// refusal itself when it denies, so a handler is one `if !Authorize(...) {
// return }` rather than a copy of the error shape.
//
// A rule that cannot be decided — a failed profile lookup, say — answers 500
// rather than 403: an infrastructure failure is not a statement about the
// caller's rights, and reporting it as one hides an outage behind a plausible
// refusal.
func Authorize(w http.ResponseWriter, r *http.Request, a authz.Authorizer, rule authz.Rule, subject authz.Subject) bool {
	principal, _ := PrincipalFrom(r.Context())
	allowed, err := a.Allow(r.Context(), rule, principal, subject)
	switch {
	case err != nil:
		w.WriteHeader(http.StatusInternalServerError)
		return false
	case allowed:
		return true
	case principal == nil:
		WriteUnauthorized(w, r)
		return false
	default:
		WriteForbidden(w, r)
		return false
	}
}

// WriteUnauthorized answers as CustomAuthenticationEntryPoint does.
func WriteUnauthorized(w http.ResponseWriter, _ *http.Request) {
	writeRefusal(w, http.StatusUnauthorized, ReasonAuthenticationRequired)
}

// WriteForbidden answers as CustomAccessDeniedHandler does.
func WriteForbidden(w http.ResponseWriter, _ *http.Request) {
	writeRefusal(w, http.StatusForbidden, ReasonNotAuthorized)
}

// writeRefusal sends the status and the reason header with no body.
//
// The deployed service sends the header twice, because the filter chain also
// runs on the error dispatch and the entry point commences a second time. One
// copy is sent here: a client reads the first value either way, and reproducing
// a duplicate would be copying an accident rather than a behaviour.
func writeRefusal(w http.ResponseWriter, status int, reason string) {
	w.Header().Set(HeaderAccessDeniedReason, reason)
	w.WriteHeader(status)
}
