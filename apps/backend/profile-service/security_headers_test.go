package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEveryResponseCarriesTheHeadersTheJVMSends(t *testing.T) {
	// The JVM's security filter chain sends both on every response, so moving
	// a path group to this service must not drop them, least of all from the
	// refusals, which are the responses no handler writes.
	server := parityServer(t, stubStore{})
	withToken := func(method, path string) *http.Request {
		request := httptest.NewRequest(method, path, nil)
		request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))
		return request
	}
	disallowedOrigin := httptest.NewRequest(http.MethodGet, "/api/profiles", nil)
	disallowedOrigin.Header.Set("Origin", "file://nowhere")

	cases := []struct {
		name    string
		request *http.Request
		status  int
	}{
		{name: "an authorized call", request: withToken(http.MethodGet, "/api/profiles"), status: http.StatusOK},
		{name: "a call with no token", request: httptest.NewRequest(http.MethodGet, "/api/profiles", nil), status: http.StatusUnauthorized},
		{name: "an unrouted path", request: withToken(http.MethodGet, "/api/nothing-here"), status: http.StatusNotFound},
		{name: "a refused origin", request: disallowedOrigin, status: http.StatusForbidden},
		{name: "the health probe", request: httptest.NewRequest(http.MethodGet, actuatorHealthPath, nil), status: http.StatusOK},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			server.ServeHTTP(response, tc.request)

			if response.Code != tc.status {
				t.Fatalf("status = %d, want %d", response.Code, tc.status)
			}
			if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
				t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
			}
			if got := response.Header().Get("X-Frame-Options"); got != "DENY" {
				t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
			}
		})
	}
}
