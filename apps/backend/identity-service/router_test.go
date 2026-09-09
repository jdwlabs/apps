package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func recordingHandler(name string, seen *string) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*seen = name
		for _, capture := range []string{"userId", "roleId", "emailAddress", "roleName"} {
			if value := r.PathValue(capture); value != "" {
				*seen += ":" + capture + "=" + value
			}
		}
	})
}

func identityShapedRouter(t *testing.T, seen *string) *Router {
	t.Helper()
	router, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/users", Handler: recordingHandler("getAllUsers", seen)},
		{Method: http.MethodPost, Pattern: "/api/users", Handler: recordingHandler("createUser", seen)},
		{Method: http.MethodGet, Pattern: "/api/users/{userId}", Handler: recordingHandler("getUserById", seen)},
		{Method: http.MethodPut, Pattern: "/api/users/{userId}", Handler: recordingHandler("updateUser", seen)},
		{
			Method:  http.MethodGet,
			Pattern: "/api/users/email/{emailAddress}",
			Handler: recordingHandler("getUserByEmailAddress", seen),
		},
		{
			Method:  http.MethodPut,
			Pattern: "/api/users/{userId}/roles/grant",
			Handler: recordingHandler("grantRolesToUser", seen),
		},
		{Method: http.MethodGet, Pattern: "/api/roles/{roleId}", Handler: recordingHandler("getRoleById", seen)},
		{
			Method:  http.MethodGet,
			Pattern: "/api/roles/name/{roleName}",
			Handler: recordingHandler("getRoleByName", seen),
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return router
}

func TestRouterResolvesTheIdentitySurfaceAsSpringResolvesIt(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		want   string
		status int
	}{
		{
			name: "the listing", method: http.MethodGet, path: "/api/users",
			want: "getAllUsers", status: http.StatusOK,
		},
		{
			name: "a numeric id reaches the user handler", method: http.MethodGet, path: "/api/users/42",
			want: "getUserById:userId=42", status: http.StatusOK,
		},
		{
			name:   "a literal segment outranks the capture that would also match it",
			method: http.MethodGet, path: "/api/users/email/a@b.co",
			want: "getUserByEmailAddress:emailAddress=a@b.co", status: http.StatusOK,
		},
		{
			name:   "the bare literal still falls to the capture, as it does in Spring",
			method: http.MethodGet, path: "/api/users/email",
			want: "getUserById:userId=email", status: http.StatusOK,
		},
		{
			name: "a role name that looks like an id", method: http.MethodGet, path: "/api/roles/name/42",
			want: "getRoleByName:roleName=42", status: http.StatusOK,
		},
		{
			name: "a grant path keeps its captures", method: http.MethodPut, path: "/api/users/7/roles/grant",
			want: "grantRolesToUser:userId=7", status: http.StatusOK,
		},
		{
			name: "an unmapped path is 404", method: http.MethodGet, path: "/api/nothing",
			want: "", status: http.StatusNotFound,
		},
		{
			name: "a mapped path with an unmapped method is 405", method: http.MethodPatch,
			path: "/api/users", want: "", status: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			router := identityShapedRouter(t, &seen)
			response := httptest.NewRecorder()

			// Authenticated, because a refusal's status now depends on it:
			// the router sits inside the middleware in the served handler, and
			// an anonymous refusal is a 401 rather than the status resolved.
			router.ServeHTTP(response, authenticatedRequest(tc.method, tc.path))

			if response.Code != tc.status {
				t.Errorf("status = %d, want %d", response.Code, tc.status)
			}
			if seen != tc.want {
				t.Errorf("handler = %q, want %q", seen, tc.want)
			}
		})
	}
}

func TestRouterRefusesRoutesAtStartupWhenNothingSeparatesThem(t *testing.T) {
	// Spring discovers such a pair at request time, throwing Ambiguous handler
	// methods mapped and answering 500 to whoever asked. Refusing at
	// construction turns that into a failure to start, which a deployment
	// notices and a user never sees.
	//
	// These two carry one capture each and normalize to the same length, so
	// neither outranks the other, and /api/users/email/roles/grant matches both.
	_, err := NewRouter([]Route{
		{Method: http.MethodPut, Pattern: "/api/users/{userId}/roles/grant", Handler: http.NotFoundHandler()},
		{Method: http.MethodPut, Pattern: "/api/users/email/{emailAddress}/grant", Handler: http.NotFoundHandler()},
	})

	if err == nil {
		t.Fatal("NewRouter accepted two patterns nothing separates; it must refuse them")
	}
}

func TestRouterRefusesTwoRegistrationsOfTheSameOperation(t *testing.T) {
	_, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/users", Handler: http.NotFoundHandler()},
		{Method: http.MethodGet, Pattern: "/api/users", Handler: http.NotFoundHandler()},
	})

	if err == nil {
		t.Fatal("NewRouter accepted the same method and pattern twice")
	}
}

func TestRouterAnswersARefusalTheWayTheContainerDoes(t *testing.T) {
	// A routing refusal is a status the container sets, not one a handler
	// composed, so it carries whatever the forward to /error renders. With a
	// verified token that is Boot's error body; the router sits inside the
	// authentication middleware, so the principal is on the request by the time
	// it resolves. Measured on a booted usersrole for 404 and 405 alike.
	var seen string
	router := identityShapedRouter(t, &seen)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "not found", method: http.MethodGet, path: "/api/nothing", status: http.StatusNotFound},
		{
			name: "method not allowed", method: http.MethodPatch, path: "/api/users",
			status: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			router.ServeHTTP(response, authenticatedRequest(tc.method, tc.path))

			assertContainerErrorBody(t, response, tc.status, tc.path)
		})
	}
}

func TestRouterAnswers401ToARefusalWithNoToken(t *testing.T) {
	// The other half of the same rule, and the one that changes a status rather
	// than a body: an anonymous request that reaches the router at all is one on
	// a public path, and the JVM answers it 401 — its forward to /error is
	// refused a second time, and the entry point's status replaces the 404.
	// Measured on a booted usersrole for /auth/nope and /actuator/nope alike.
	var seen string
	router := identityShapedRouter(t, &seen)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/nothing", nil))

	if response.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if body := response.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestRouterAdvertisesTheMethodsAPathAccepts(t *testing.T) {
	var seen string
	router := identityShapedRouter(t, &seen)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/users/9", nil))

	if got, want := response.Header().Get("Allow"), "GET, PUT"; got != want {
		t.Errorf("Allow = %q, want %q", got, want)
	}
}

func TestRouterExposesTheMatchedPatternForMetrics(t *testing.T) {
	// The metrics layer labels by route pattern rather than by request path, so
	// /api/users/1 and /api/users/2 stay one series.
	var seen string
	router := identityShapedRouter(t, &seen)

	matched, ok := router.Match(httptest.NewRequest(http.MethodGet, "/api/users/42", nil))

	if !ok {
		t.Fatal("Match reported no route for a mapped request")
	}
	if matched != "/api/users/{userId}" {
		t.Errorf("matched pattern = %q, want %q", matched, "/api/users/{userId}")
	}
}

func TestTheServedRoutesCarryNoPairSpringWouldFindAmbiguous(t *testing.T) {
	// The real registration rather than a shaped one: NewServer refuses an
	// ambiguous pair, so a route added to the operation list that collides with
	// an existing one fails here rather than at the first request that hits it.
	if _, err := NewServer(ServerConfig{
		Store: stubStore{}, Verifier: parityVerifier(t), Minter: parityMinter(t), CORS: springShapedCORS(),
	}); err != nil {
		t.Fatalf("the served route set does not compile into a router: %v", err)
	}
}
