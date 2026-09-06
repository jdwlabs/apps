package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func recordingHandler(name string, seen *string) http.Handler {
	return http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		*seen = name
		if id := r.PathValue("profileId"); id != "" {
			*seen += ":profileId=" + id
		}
		if id := r.PathValue("userId"); id != "" {
			*seen += ":userId=" + id
		}
	})
}

func profileShapedRouter(t *testing.T, seen *string) *Router {
	t.Helper()
	router, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/profiles", Handler: recordingHandler("getProfiles", seen)},
		{Method: http.MethodGet, Pattern: "/api/profiles/{profileId}", Handler: recordingHandler("getProfileById", seen)},
		{Method: http.MethodGet, Pattern: "/api/profiles/by-user/{userId}", Handler: recordingHandler("getProfileByUserId", seen)},
		{Method: http.MethodPut, Pattern: "/api/profiles/by-user/{userId}", Handler: recordingHandler("updateProfileByUserId", seen)},
		{Method: http.MethodDelete, Pattern: "/api/profiles/by-user/{userId}", Handler: recordingHandler("deleteProfileByUserId", seen)},
		{Method: http.MethodPost, Pattern: "/api/profiles/{profileId}/address", Handler: recordingHandler("addAddress", seen)},
		{
			Method:   http.MethodGet,
			Pattern:  "/api/profiles/{profileId}/icon",
			Produces: "image/png",
			Handler:  recordingHandler("getProfileIcon", seen),
		},
		{Method: http.MethodPut, Pattern: "/api/profiles/{profileId}/icon", Handler: recordingHandler("updateIcon", seen)},
		{Method: http.MethodDelete, Pattern: "/api/profiles/{profileId}/icon", Handler: recordingHandler("deleteIcon", seen)},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	return router
}

func TestRouterReproducesTheMeasuredSpringRouting(t *testing.T) {
	// Every expectation here is transcribed from x-path-precedence.measuredRouting
	// in the frozen contract, which resolved them against the running
	// RequestMappingHandlerMapping rather than reasoning from the comparator.
	cases := []struct {
		name   string
		method string
		path   string
		accept string
		want   string
		status int
	}{
		{name: "list", method: http.MethodGet, path: "/api/profiles", want: "getProfiles", status: http.StatusOK},
		{
			name: "numeric id reaches the profile handler", method: http.MethodGet, path: "/api/profiles/42",
			want: "getProfileById:profileId=42", status: http.StatusOK,
		},
		{
			name: "by-user with a numeric id", method: http.MethodGet, path: "/api/profiles/by-user/7",
			want: "getProfileByUserId:userId=7", status: http.StatusOK,
		},
		{
			name:   "by-user/icon GET goes to the by-user handler, even asking for png",
			method: http.MethodGet, path: "/api/profiles/by-user/icon", accept: "image/png",
			want: "getProfileByUserId:userId=icon", status: http.StatusOK,
		},
		{
			name: "by-user/icon PUT goes to the by-user handler", method: http.MethodPut,
			path: "/api/profiles/by-user/icon", want: "updateProfileByUserId:userId=icon", status: http.StatusOK,
		},
		{
			name: "by-user/icon DELETE goes to the by-user handler", method: http.MethodDelete,
			path: "/api/profiles/by-user/icon", want: "deleteProfileByUserId:userId=icon", status: http.StatusOK,
		},
		{
			name: "user/icon GET reaches the icon handler with Accept unset", method: http.MethodGet,
			path: "/api/profiles/user/icon", want: "getProfileIcon:profileId=user", status: http.StatusOK,
		},
		{
			name: "user/icon GET reaches the icon handler for */*", method: http.MethodGet,
			path: "/api/profiles/user/icon", accept: "*/*", want: "getProfileIcon:profileId=user", status: http.StatusOK,
		},
		{
			name: "user/icon GET reaches the icon handler for image/png", method: http.MethodGet,
			path: "/api/profiles/user/icon", accept: "image/png", want: "getProfileIcon:profileId=user", status: http.StatusOK,
		},
		{
			name: "user/icon GET is refused for application/json", method: http.MethodGet,
			path: "/api/profiles/user/icon", accept: "application/json", want: "", status: http.StatusNotAcceptable,
		},
		{
			name: "user/icon PUT reaches the icon handler", method: http.MethodPut,
			path: "/api/profiles/user/icon", want: "updateIcon:profileId=user", status: http.StatusOK,
		},
		{
			name:   "by-user/address still reaches the address handler because only POST maps it",
			method: http.MethodPost, path: "/api/profiles/by-user/address",
			want: "addAddress:profileId=by-user", status: http.StatusOK,
		},
		{
			name: "an unmapped path is 404", method: http.MethodGet, path: "/api/nothing",
			want: "", status: http.StatusNotFound,
		},
		{
			name: "a mapped path with an unmapped method is 405", method: http.MethodPatch,
			path: "/api/profiles", want: "", status: http.StatusMethodNotAllowed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var seen string
			router := profileShapedRouter(t, &seen)
			request := httptest.NewRequest(tc.method, tc.path, nil)
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

func TestRouterRefusesRoutesAtStartupWhenNothingSeparatesThem(t *testing.T) {
	// The tie the by-user move resolved: two patterns of equal capture count and
	// equal normalized length, overlapping on /api/profiles/user/icon. Spring
	// answered 500 at request time for this; a startup refusal turns the same
	// defect into a build failure.
	_, err := NewRouter([]Route{
		{Method: http.MethodPut, Pattern: "/api/profiles/user/{userId}", Handler: http.NotFoundHandler()},
		{Method: http.MethodPut, Pattern: "/api/profiles/{profileId}/icon", Handler: http.NotFoundHandler()},
	})

	if err == nil {
		t.Fatal("NewRouter accepted two patterns nothing separates; it must refuse them")
	}
}

func TestRouterRefusesTwoRegistrationsOfTheSameOperation(t *testing.T) {
	_, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/profiles", Handler: http.NotFoundHandler()},
		{Method: http.MethodGet, Pattern: "/api/profiles", Handler: http.NotFoundHandler()},
	})

	if err == nil {
		t.Fatal("NewRouter accepted the same method and pattern twice")
	}
}

func TestRouterSendsNoBodyOnARefusal(t *testing.T) {
	// The deployed service writes every routing refusal through sendError with
	// server.error.include-message unset, so the body is empty.
	var seen string
	router := profileShapedRouter(t, &seen)

	for _, tc := range []struct {
		name   string
		method string
		path   string
	}{
		{name: "not found", method: http.MethodGet, path: "/api/nothing"},
		{name: "method not allowed", method: http.MethodPatch, path: "/api/profiles"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			router.ServeHTTP(response, httptest.NewRequest(tc.method, tc.path, nil))

			if body := response.Body.String(); body != "" {
				t.Errorf("body = %q, want empty", body)
			}
			if contentType := response.Header().Get("Content-Type"); contentType != "" {
				t.Errorf("Content-Type = %q, want unset", contentType)
			}
		})
	}
}

func TestRouterAdvertisesTheMethodsAPathAccepts(t *testing.T) {
	var seen string
	router := profileShapedRouter(t, &seen)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, httptest.NewRequest(http.MethodPatch, "/api/profiles/by-user/9", nil))

	if got, want := response.Header().Get("Allow"), "DELETE, GET, PUT"; got != want {
		t.Errorf("Allow = %q, want %q", got, want)
	}
}

func TestRouterExposesTheMatchedPatternForMetrics(t *testing.T) {
	// The metrics layer labels by route pattern rather than by request path, so
	// /api/profiles/1 and /api/profiles/2 stay one series.
	var seen string
	router := profileShapedRouter(t, &seen)
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/42", nil)

	matched, ok := router.Match(request)

	if !ok {
		t.Fatal("Match reported no route for a mapped request")
	}
	if matched != "/api/profiles/{profileId}" {
		t.Errorf("matched pattern = %q, want %q", matched, "/api/profiles/{profileId}")
	}
}
