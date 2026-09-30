package servicehttp

import (
	"encoding/json"
	"net/http"
)

// BuildInfo serves the running version in the shape Spring's /actuator/info
// takes with build-info enabled, so a caller waiting for a rollout reads the
// JVM and the Go services alike. An empty version means the binary was built
// outside the image pipeline and says so rather than claiming a release.
func BuildInfo(version string) http.Handler {
	if version == "" {
		version = "unknown"
	}
	body, _ := json.Marshal(map[string]map[string]string{"build": {"version": version}})
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write(body)
	})
}
