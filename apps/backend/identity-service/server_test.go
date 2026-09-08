package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTheHealthEndpointsAnswerInTheShapeABootProbeReads(t *testing.T) {
	// The chart's probes are written against the actuator, which reports its
	// verdict in the body rather than only in the status. A handler that
	// answered 200 with {} would satisfy every status assertion and fail every
	// probe keyed on .status == "UP" the moment the cutover happened.
	server := parityServer(t, stubStore{})

	for _, path := range []string{healthPath, actuatorHealthPath} {
		t.Run(path, func(t *testing.T) {
			response := httptest.NewRecorder()

			server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))

			if response.Code != http.StatusOK {
				t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
			}
			if got := response.Header().Get("Content-Type"); got != contentTypeJSON {
				t.Errorf("Content-Type = %q, want %q", got, contentTypeJSON)
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("the body is not JSON: %v (%q)", err, response.Body.String())
			}
			if body["status"] != "UP" {
				t.Errorf("status field = %v, want %q", body["status"], "UP")
			}
		})
	}
}
