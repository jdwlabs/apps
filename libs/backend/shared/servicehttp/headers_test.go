package servicehttp

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestSecurityHeadersAreOnEveryResponseWhateverTheStatus(t *testing.T) {
	// The headers are set before the wrapped handler runs, so a handler that
	// writes an error and returns early still sends them.
	for _, status := range []int{http.StatusOK, http.StatusUnauthorized, http.StatusNotFound, http.StatusInternalServerError} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				http.Error(w, "refused", status)
			}))
			response := httptest.NewRecorder()

			handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/api/resources", nil))

			if response.Code != status {
				t.Fatalf("status = %d, want %d", response.Code, status)
			}
			assertSecurityHeaders(t, response.Header())
		})
	}
}

func assertSecurityHeaders(t *testing.T, header http.Header) {
	t.Helper()
	if got := header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options = %q, want %q", got, "nosniff")
	}
	if got := header.Get("X-Frame-Options"); got != "DENY" {
		t.Errorf("X-Frame-Options = %q, want %q", got, "DENY")
	}
}
