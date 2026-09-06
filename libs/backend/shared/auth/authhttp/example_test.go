package authhttp_test

import (
	"fmt"
	"net/http"
	"strings"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authhttp"
	"libs/backend/shared/auth/authz"
)

// The wiring the README documents, compiled and run by `go test`. It lives here
// so the snippet cannot drift from the API: a rename or a signature change
// fails the build rather than leaving a README that no longer works.
func Example() {
	secret := paritySecret // in a service: auth.SecretKeyFromEnv()

	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:  secret,
		ExpectedIssuer:   issuerOrigin + "/auth/authenticate",
		ExpectedAudience: issuerOrigin,
	})
	if err != nil {
		panic(err)
	}

	middleware, err := authhttp.NewMiddleware(verifier)
	if err != nil {
		panic(err)
	}
	// The operations the contract freezes as PUBLIC, reachable without a token.
	middleware.Public = func(r *http.Request) bool {
		return strings.HasPrefix(r.URL.Path, "/auth/")
	}

	authorizer := authz.Authorizer{}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/users/{userId}", func(w http.ResponseWriter, r *http.Request) {
		var userID int64 = 42 // parsed from r.PathValue("userId")

		if !authhttp.Authorize(w, r, authorizer, authz.RuleAdminOrSelfByUserID,
			authz.Subject{UserID: &userID}) {
			return
		}

		principal, _ := authhttp.PrincipalFrom(r.Context())
		// A failed write means the client is gone; there is nobody left to tell.
		_, _ = fmt.Fprintf(w, "hello %s", principal.EmailAddress())
	})

	handler := middleware.Handler(mux)

	// Serving it would block, so the example only shows that it is a handler.
	fmt.Println(handler != nil)
	// Output: true
}
