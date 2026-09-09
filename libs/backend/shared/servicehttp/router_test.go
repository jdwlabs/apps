package servicehttp

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authhttp"
)

// The route table is deliberately no service's. Each service pins its own
// measured routing against its own surface; what belongs here is the resolution
// the two of them share, on a shape that exercises every branch of it: a
// literal that competes with a capture, and an operation that promises a media
// type.
func recordingHandler(name string, seen *string) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*seen = name
		for _, capture := range []string{"resourceId", "name"} {
			if value := r.PathValue(capture); value != "" {
				*seen += ":" + capture + "=" + value
			}
		}
	})
}

func shapedRouter(t *testing.T, seen *string) *Router {
	t.Helper()
	router, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/resources", Handler: recordingHandler("list", seen)},
		{Method: http.MethodPost, Pattern: "/api/resources", Handler: recordingHandler("create", seen)},
		{Method: http.MethodGet, Pattern: "/api/resources/{resourceId}", Handler: recordingHandler("byId", seen)},
		{Method: http.MethodPut, Pattern: "/api/resources/{resourceId}", Handler: recordingHandler("update", seen)},
		{Method: http.MethodGet, Pattern: "/api/resources/by-name/{name}", Handler: recordingHandler("byName", seen)},
		{
			Method:   http.MethodGet,
			Pattern:  "/api/resources/{resourceId}/icon",
			Produces: "image/png",
			Handler:  recordingHandler("icon", seen),
		},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return router
}

// authenticatedRequest carries a verified principal, which is what decides the
// shape of a status the container sets: in the JVM the forward to /error either
// re-authenticates with the caller's token or is refused a second time.
func authenticatedRequest(method, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	principal := &auth.Principal{Subject: "admin@jdw.com", Roles: []string{"ADMIN"}}
	return request.WithContext(authhttp.WithPrincipal(request.Context(), principal))
}

func TestRouterResolvesAPathTheWaySpringsComparatorDoes(t *testing.T) {
	cases := []struct {
		name   string
		method string
		path   string
		accept string
		want   string
		status int
	}{
		{name: "the listing", method: http.MethodGet, path: "/api/resources", want: "list", status: http.StatusOK},
		{
			name: "a capture takes the id", method: http.MethodGet, path: "/api/resources/42",
			want: "byId:resourceId=42", status: http.StatusOK,
		},
		{
			name:   "a literal segment outranks the capture that would also match it",
			method: http.MethodGet, path: "/api/resources/by-name/widget",
			want: "byName:name=widget", status: http.StatusOK,
		},
		{
			name:   "the longer pattern wins where captures tie",
			method: http.MethodGet, path: "/api/resources/by-name/icon", accept: "image/png",
			want: "byName:name=icon", status: http.StatusOK,
		},
		{
			name:   "the bare literal still falls to the capture, as it does in Spring",
			method: http.MethodGet, path: "/api/resources/by-name",
			want: "byId:resourceId=by-name", status: http.StatusOK,
		},
		{
			name: "an unmapped path is 404", method: http.MethodGet, path: "/api/nothing",
			want: "", status: http.StatusNotFound,
		},
		{
			name: "a mapped path with an unmapped method is 405", method: http.MethodPatch,
			path: "/api/resources", want: "", status: http.StatusMethodNotAllowed,
		},
		{
			name: "a mapping dropped only by its produces condition is 406", method: http.MethodGet,
			path: "/api/resources/7/icon", accept: "application/json", want: "", status: http.StatusNotAcceptable,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			router := shapedRouter(t, &seen)
			// Authenticated, because a refusal's status depends on it: the
			// router sits inside the authentication middleware in the served
			// handler, and an anonymous refusal is a 401 rather than the status
			// resolved.
			request := authenticatedRequest(tc.method, tc.path)
			if tc.accept != "" {
				request.Header.Set("Accept", tc.accept)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != tc.status {
				t.Errorf("status = %d, want %d", response.Code, tc.status)
			}
			if seen != tc.want {
				t.Errorf("handler = %q, want %q", seen, tc.want)
			}
		})
	}
}

func TestRouterServesAnOperationForEveryAcceptThatPermitsItsMediaType(t *testing.T) {
	for _, accept := range []string{"", "*/*", "image/*", "image/png", "image/png;q=0.8", "text/html, image/png"} {
		t.Run(accept, func(t *testing.T) {
			var seen string
			router := shapedRouter(t, &seen)
			request := authenticatedRequest(http.MethodGet, "/api/resources/7/icon")
			if accept != "" {
				request.Header.Set("Accept", accept)
			}
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			if response.Code != http.StatusOK {
				t.Errorf("status = %d, want 200", response.Code)
			}
			if seen != "icon:resourceId=7" {
				t.Errorf("handler = %q, want the icon handler", seen)
			}
		})
	}
}

func TestNewRouterRefusesRoutesAtStartupWhenNothingSeparatesThem(t *testing.T) {
	// Spring discovers such a pair at request time, throwing Ambiguous handler
	// methods mapped and answering 500 to whoever asked. Refusing at
	// construction turns that into a failure to start, which a deployment
	// notices and a user never sees. Both shapes below were live in a service
	// before its route table was reshaped to separate them.
	for _, tc := range []struct {
		name  string
		left  string
		right string
	}{
		{
			name: "captures in different positions, same length",
			left: "/api/users/{userId}/roles/grant", right: "/api/users/email/{emailAddress}/grant",
		},
		{
			name: "a literal that a capture also matches, same length",
			left: "/api/profiles/user/{userId}", right: "/api/profiles/{profileId}/icon",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewRouter([]Route{
				{Method: http.MethodPut, Pattern: tc.left, Handler: http.NotFoundHandler()},
				{Method: http.MethodPut, Pattern: tc.right, Handler: http.NotFoundHandler()},
			})

			if err == nil {
				t.Fatal("NewRouter accepted two patterns nothing separates; it must refuse them")
			}
		})
	}
}

func TestNewRouterAllowsAProducesConditionToSeparateAnOverlappingPair(t *testing.T) {
	// Spring lets the produces condition settle a tie the path cannot, so a
	// refusal here would reject a registration the JVM serves.
	_, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/a/{id}/x", Handler: http.NotFoundHandler()},
		{Method: http.MethodGet, Pattern: "/api/a/x/{id}", Produces: "image/png", Handler: http.NotFoundHandler()},
	})

	if err != nil {
		t.Fatalf("NewRouter refused a pair their produces conditions separate: %v", err)
	}
}

func TestNewRouterRefusesTwoRegistrationsOfTheSameOperation(t *testing.T) {
	_, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/resources", Handler: http.NotFoundHandler()},
		{Method: http.MethodGet, Pattern: "/api/resources", Handler: http.NotFoundHandler()},
	})

	if err == nil {
		t.Fatal("NewRouter accepted the same method and pattern twice")
	}
}

func TestNewRouterRefusesARouteItCannotCompile(t *testing.T) {
	for _, tc := range []struct {
		name  string
		route Route
	}{
		{
			name:  "a pattern that is not rooted",
			route: Route{Method: http.MethodGet, Pattern: "api/resources", Handler: http.NotFoundHandler()},
		},
		{
			name:  "no handler",
			route: Route{Method: http.MethodGet, Pattern: "/api/resources"},
		},
		{
			name:  "an unnamed capture",
			route: Route{Method: http.MethodGet, Pattern: "/api/resources/{}", Handler: http.NotFoundHandler()},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := NewRouter([]Route{tc.route}); err == nil {
				t.Fatal("NewRouter accepted a route it cannot serve")
			}
		})
	}
}

func TestRouterAnswersARefusalTheWayTheContainerDoes(t *testing.T) {
	// A routing refusal is a status the container sets, not one a handler
	// composed, so it carries whatever the forward to /error renders. With a
	// verified token that is Boot's error body; the router sits inside the
	// authentication middleware, so the principal is on the request by the time
	// it resolves. The body's own shape is pinned in authhttp, which writes it.
	var seen string
	router := shapedRouter(t, &seen)

	for _, tc := range []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "not found", method: http.MethodGet, path: "/api/nothing", status: http.StatusNotFound},
		{
			name: "method not allowed", method: http.MethodPatch, path: "/api/resources",
			status: http.StatusMethodNotAllowed,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			router.ServeHTTP(response, authenticatedRequest(tc.method, tc.path))

			if response.Code != tc.status {
				t.Errorf("status = %d, want %d", response.Code, tc.status)
			}
			if got, want := response.Header().Get("Content-Type"), "application/json"; got != want {
				t.Errorf("Content-Type = %q, want %q", got, want)
			}
			if response.Body.Len() == 0 {
				t.Error("body is empty; an authenticated caller reads the container's error body")
			}
		})
	}
}

func TestRouterAnswers401ToARefusalWithNoToken(t *testing.T) {
	// The other half of the same rule, and the one that changes a status rather
	// than a body: an anonymous request that reaches the router at all is one on
	// a public path, and the JVM answers it 401 — its forward to /error is
	// refused a second time, and the entry point's status replaces the 404.
	var seen string
	router := shapedRouter(t, &seen)
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
	router := shapedRouter(t, &seen)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodDelete, "/api/resources/9", nil))

	if got, want := response.Header().Get("Allow"), "GET, PUT"; got != want {
		t.Errorf("Allow = %q, want %q", got, want)
	}
}

func TestRouterExposesTheMatchedPatternForMetrics(t *testing.T) {
	// The metrics layer labels by route pattern rather than by request path, so
	// /api/resources/1 and /api/resources/2 stay one series.
	var seen string
	router := shapedRouter(t, &seen)

	matched, ok := router.Match(httptest.NewRequest(http.MethodGet, "/api/resources/42", nil))

	if !ok {
		t.Fatal("Match reported no route for a mapped request")
	}
	if matched != "/api/resources/{resourceId}" {
		t.Errorf("matched pattern = %q, want %q", matched, "/api/resources/{resourceId}")
	}
	if _, ok := router.Match(httptest.NewRequest(http.MethodGet, "/api/nothing", nil)); ok {
		t.Error("Match reported a route for an unmapped request")
	}
}

func TestRouterListsTheOperationsItServes(t *testing.T) {
	// The contract suites compare this against the frozen operation set, so a
	// route added without a contract entry is caught rather than served.
	var seen string
	router := shapedRouter(t, &seen)

	want := []string{
		"GET /api/resources",
		"GET /api/resources/by-name/{name}",
		"GET /api/resources/{resourceId}",
		"GET /api/resources/{resourceId}/icon",
		"POST /api/resources",
		"PUT /api/resources/{resourceId}",
	}
	if got := router.Patterns(); !slices.Equal(got, want) {
		t.Errorf("Patterns() = %v, want %v", got, want)
	}
}
