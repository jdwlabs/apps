package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func springShapedCORS() CORS {
	// The three lists SecurityConfig's CorsConfigurationSource registers for
	// "/**", transcribed verbatim.
	return CORS{
		AllowedOriginPatterns: []string{"http://*:[*]", "https://*:[*]"},
		AllowedMethods:        []string{"GET", "POST", "PUT", "DELETE", "HEAD", "PATCH", "OPTIONS"},
		AllowedHeaders:        []string{"Authorization", "Content-Type"},
	}
}

func corsHandler(reached *bool) http.Handler {
	return springShapedCORS().Handler(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		*reached = true
		w.WriteHeader(http.StatusTeapot)
	}))
}

func TestAPreflightIsAnsweredWithoutReachingTheWrappedHandler(t *testing.T) {
	// The deployed service answers OPTIONS through CorsFilter before the JWT
	// filter runs. A browser never puts an Authorization header on a preflight,
	// so anything that authenticated it would refuse every cross-origin call.
	reached := false
	request := httptest.NewRequest(http.MethodOptions, "/api/profiles/1", nil)
	request.Header.Set("Origin", "http://localhost:4200")
	request.Header.Set("Access-Control-Request-Method", "GET")
	request.Header.Set("Access-Control-Request-Headers", "authorization,content-type")
	response := httptest.NewRecorder()

	corsHandler(&reached).ServeHTTP(response, request)

	if reached {
		t.Error("the preflight reached the wrapped handler; the CORS layer must answer it")
	}
	if response.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", response.Code, http.StatusOK)
	}
	if body := response.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	if got, want := response.Header().Get("Access-Control-Allow-Origin"), "http://localhost:4200"; got != want {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Access-Control-Allow-Methods"),
		"GET, POST, PUT, DELETE, HEAD, PATCH, OPTIONS"; got != want {
		t.Errorf("Access-Control-Allow-Methods = %q, want %q", got, want)
	}
	if got, want := response.Header().Get("Access-Control-Allow-Headers"), "authorization, content-type"; got != want {
		t.Errorf("Access-Control-Allow-Headers = %q, want %q", got, want)
	}
	if got := response.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Errorf("Access-Control-Allow-Credentials = %q; the JVM configures no credentials", got)
	}
	// No max age either, so a browser repeats the preflight on the next
	// cross-origin request rather than caching this answer.
	if got := response.Header().Get("Access-Control-Max-Age"); got != "" {
		t.Errorf("Access-Control-Max-Age = %q; the JVM sets none", got)
	}
}

func TestEveryResponseVariesOnTheCorsRequestHeaders(t *testing.T) {
	// Spring adds all three unconditionally, so a shared cache cannot serve one
	// origin's response to another.
	reached := false
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/1", nil)
	response := httptest.NewRecorder()

	corsHandler(&reached).ServeHTTP(response, request)

	vary := response.Header().Values("Vary")
	for _, want := range []string{"Origin", "Access-Control-Request-Method", "Access-Control-Request-Headers"} {
		if !contains(vary, want) {
			t.Errorf("Vary = %v, want it to include %q", vary, want)
		}
	}
}

func TestARequestWithNoOriginPassesThroughUntouched(t *testing.T) {
	reached := false
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/1", nil)
	response := httptest.NewRecorder()

	corsHandler(&reached).ServeHTTP(response, request)

	if !reached {
		t.Fatal("a same-origin request did not reach the wrapped handler")
	}
	if got := response.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Errorf("Access-Control-Allow-Origin = %q, want unset for a non-CORS request", got)
	}
}

func TestAnActualCrossOriginRequestIsAllowedAndStillServed(t *testing.T) {
	reached := false
	request := httptest.NewRequest(http.MethodGet, "/api/profiles/1", nil)
	request.Header.Set("Origin", "https://app.example.com:8443")
	response := httptest.NewRecorder()

	corsHandler(&reached).ServeHTTP(response, request)

	if !reached {
		t.Fatal("an allowed cross-origin request did not reach the wrapped handler")
	}
	if response.Code != http.StatusTeapot {
		t.Errorf("status = %d, want the wrapped handler's %d", response.Code, http.StatusTeapot)
	}
	if got, want := response.Header().Get("Access-Control-Allow-Origin"), "https://app.example.com:8443"; got != want {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, want)
	}
}

func TestACorsRequestThatFailsACheckIsRefused(t *testing.T) {
	cases := []struct {
		name    string
		method  string
		headers map[string]string
	}{
		{
			name:   "an origin outside the allowed patterns",
			method: http.MethodOptions,
			headers: map[string]string{
				"Origin":                        "file://local",
				"Access-Control-Request-Method": "GET",
			},
		},
		{
			name:   "a method outside the allowed list",
			method: http.MethodOptions,
			headers: map[string]string{
				"Origin":                        "http://localhost:4200",
				"Access-Control-Request-Method": "TRACE",
			},
		},
		{
			name:   "a request header outside the allowed list",
			method: http.MethodOptions,
			headers: map[string]string{
				"Origin":                         "http://localhost:4200",
				"Access-Control-Request-Method":  "GET",
				"Access-Control-Request-Headers": "X-Smuggled",
			},
		},
		{
			name:    "an actual request from an origin outside the patterns",
			method:  http.MethodGet,
			headers: map[string]string{"Origin": "file://local"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reached := false
			request := httptest.NewRequest(tc.method, "/api/profiles/1", nil)
			for name, value := range tc.headers {
				request.Header.Set(name, value)
			}
			response := httptest.NewRecorder()

			corsHandler(&reached).ServeHTTP(response, request)

			if reached {
				t.Error("a refused request reached the wrapped handler")
			}
			if response.Code != http.StatusForbidden {
				t.Errorf("status = %d, want %d", response.Code, http.StatusForbidden)
			}
			if got, want := response.Body.String(), "Invalid CORS request"; got != want {
				t.Errorf("body = %q, want %q", got, want)
			}
		})
	}
}

func TestTheOriginPatternSyntaxMatchesWhatSpringMatches(t *testing.T) {
	cases := []struct {
		pattern string
		origin  string
		want    bool
	}{
		{pattern: "http://*:[*]", origin: "http://localhost:4200", want: true},
		{pattern: "http://*:[*]", origin: "http://localhost", want: true},
		{pattern: "http://*:[*]", origin: "https://localhost:4200", want: false},
		{pattern: "https://*:[*]", origin: "https://app.example.com:8443", want: true},
		{pattern: "http://*:[*]", origin: "file://local", want: false},
		{pattern: "https://app.example.com:[443]", origin: "https://app.example.com:443", want: true},
		{pattern: "https://app.example.com:[443]", origin: "https://app.example.com:8443", want: false},
		{pattern: "https://app.example.com", origin: "https://app.example.com", want: true},
		{pattern: "https://app.example.com", origin: "https://app.example.com:443", want: false},
	}

	for _, tc := range cases {
		t.Run(tc.pattern+" vs "+tc.origin, func(t *testing.T) {
			got := matchesOriginPattern(tc.pattern, tc.origin)

			if got != tc.want {
				t.Errorf("matchesOriginPattern(%q, %q) = %v, want %v", tc.pattern, tc.origin, got, tc.want)
			}
		})
	}
}

func contains(values []string, want string) bool {
	for _, value := range values {
		for _, entry := range strings.Split(value, ",") {
			if strings.TrimSpace(entry) == want {
				return true
			}
		}
	}
	return false
}
