package servicehttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestBuildInfoAnswersInTheActuatorShape(t *testing.T) {
	// The rollout wait reads the JVM's /actuator/info and these services the
	// same way, so the Go body has to match Spring's build-info shape.
	for _, tc := range []struct{ in, want string }{{"1.4.0", "1.4.0"}, {"", "unknown"}} {
		response := httptest.NewRecorder()
		BuildInfo(tc.in).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/actuator/info", nil))

		if response.Code != http.StatusOK {
			t.Fatalf("status = %d, want 200", response.Code)
		}
		if got := response.Header().Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q", got)
		}
		var body struct {
			Build struct {
				Version string `json:"version"`
			} `json:"build"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
			t.Fatalf("not JSON: %v", err)
		}
		if body.Build.Version != tc.want {
			t.Errorf("build.version = %q, want %q", body.Build.Version, tc.want)
		}
	}
}
