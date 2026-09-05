// Package authhttp wires the verifier and the authorization rules into
// net/http, and answers refusals in the exact shape the Spring application
// answers them: the same status, the same Access-Denied-Reason header, and the
// container error body the frozen contracts pin.
//
// The bodies matter less than the statuses — no frontend reads them — but
// reproducing them keeps a client that logs a response from seeing the split.
package authhttp

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

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

// Both handlers build the body as "Access Denied " plus the exception message,
// so the prefix appears twice in the rendered message. Reproduced rather than
// tidied: a client that matches on the string must not see the split.
const (
	messagePrefix          = "Access Denied "
	authenticationRequired = "Full authentication is required to access this resource"
	accessIsDenied         = "Access is denied"
)

// Spring Boot serializes the error timestamp with Jackson's default date
// format, not as an epoch number.
const containerTimestampLayout = "2006-01-02T15:04:05.000-07:00"

type contextKey struct{}

var principalKey contextKey

// containerError is the servlet container's error body. Field order is the
// order Spring Boot's DefaultErrorAttributes inserts them.
type containerError struct {
	Timestamp string `json:"timestamp"`
	Status    int    `json:"status"`
	Error     string `json:"error"`
	Message   string `json:"message"`
	Path      string `json:"path"`
}

// Middleware verifies the bearer token on each request and puts the principal
// in the request context.
type Middleware struct {
	Verifier *auth.Verifier
	// Public reports whether a request bypasses the authentication requirement,
	// standing in for SecurityConfig's permitAll matchers. A public request
	// whose token verifies still carries its principal, because the JVM filter
	// runs ahead of those matchers and some public operations read the caller.
	// Nil makes every route require a token.
	Public func(*http.Request) bool
	// OnError observes verification failures. The JVM filter logs them at warn
	// and answers 401 either way; nothing here depends on a logger.
	OnError func(*http.Request, error)
}

// Handler wraps next with authentication.
func (m Middleware) Handler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		public := m.Public != nil && m.Public(r)

		principal, err := m.Verifier.VerifyAuthorizationHeader(r.Header.Get("Authorization"))
		if err != nil {
			if m.OnError != nil {
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
		writeError(w, r, http.StatusInternalServerError, "", http.StatusText(http.StatusInternalServerError))
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
func WriteUnauthorized(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusUnauthorized, ReasonAuthenticationRequired, messagePrefix+authenticationRequired)
}

// WriteForbidden answers as CustomAccessDeniedHandler does.
func WriteForbidden(w http.ResponseWriter, r *http.Request) {
	writeError(w, r, http.StatusForbidden, ReasonNotAuthorized, messagePrefix+accessIsDenied)
}

// writeError renders the container error body. An empty reason omits the
// header: only the two refusal handlers add it, and a 500 is not a refusal.
func writeError(w http.ResponseWriter, r *http.Request, status int, reason, message string) {
	if reason != "" {
		w.Header().Set(HeaderAccessDeniedReason, reason)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	// A write failure here means the client is gone; there is no second
	// response to send and no caller left to tell.
	_ = json.NewEncoder(w).Encode(containerError{
		Timestamp: time.Now().UTC().Format(containerTimestampLayout),
		Status:    status,
		Error:     http.StatusText(status),
		Message:   message,
		Path:      r.URL.Path,
	})
}
