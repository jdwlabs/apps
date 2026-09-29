package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTheInfoEndpointNeedsNoTokenAndCarriesTheImageVersion(t *testing.T) {
	server := parityServerWithVersion(t, stubStore{}, "9.9.9")
	response := httptest.NewRecorder()

	server.ServeHTTP(response, httptest.NewRequest(http.MethodGet, actuatorInfoPath, nil))

	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusOK)
	}
	var body struct {
		Build struct {
			Version string `json:"version"`
		} `json:"build"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
		t.Fatalf("not JSON: %v", err)
	}
	if body.Build.Version != "9.9.9" {
		t.Errorf("build.version = %q, want 9.9.9", body.Build.Version)
	}
}
