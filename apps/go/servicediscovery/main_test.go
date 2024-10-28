package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// Mock envGetter to simulate environment variables in tests
type mockEnvGetter struct {
	envs map[string]string
}

func (m mockEnvGetter) getEnvOrDefault(key, fallback string) string {
	if val, ok := m.envs[key]; ok {
		return val
	}
	return fallback
}

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()

	rootHandler(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("rootHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := "Service Discovery is up and running"
	if strings.TrimSpace(w.Body.String()) != expected {
		t.Errorf("rootHandler returned unexpected body: got %v want %v", w.Body.String(), expected)
	}
}

func TestHealthHandler(t *testing.T) {
	// Create a new request to the /health endpoint
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()

	// Call the handler function
	healthHandler(w, req)

	// Check the status code is 200 OK
	if status := w.Code; status != http.StatusOK {
		t.Errorf("healthHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Check the response body
	expected := "Service is healthy"
	if strings.TrimSpace(w.Body.String()) != expected {
		t.Errorf("healthHandler returned unexpected body: got %v want %v", w.Body.String(), expected)
	}
}

func TestRemotesHandler(t *testing.T) {
	// Setup default config for the test
	config.Remotes = map[string]string{"key": "value"}

	req := httptest.NewRequest(http.MethodGet, "/api/remotes", nil)
	w := httptest.NewRecorder()

	remotesHandler(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("remotesHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("remotesHandler returned invalid JSON: %v", err)
	}

	expected := map[string]string{"key": "value"}
	if resp["key"] != expected["key"] {
		t.Errorf("remotesHandler returned unexpected body: got %v want %v", resp, expected)
	}
}

func TestRemotesHandlerEmpty(t *testing.T) {
	// Test with empty remotes
	config.Remotes = map[string]string{}

	req := httptest.NewRequest(http.MethodGet, "/api/remotes", nil)
	w := httptest.NewRecorder()

	remotesHandler(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("remotesHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("remotesHandler returned invalid JSON: %v", err)
	}

	if len(resp) != 0 {
		t.Errorf("remotesHandler returned unexpected body: got %v want empty", resp)
	}
}

func TestMicroFrontendsHandler(t *testing.T) {
	// Setup default config for the test
	config.MicroFrontends = []MicroFrontend{
		{
			Name:        "example",
			Path:        "/example",
			RemoteName:  "exampleRemote",
			ModuleName:  "ExampleModule",
			URL:         "http://example.com",
			Icon:        "icon.png",
			Title:       "Example",
			Description: "An example micro frontend",
		},
	}

	req := httptest.NewRequest(http.MethodGet, "/api/micro-frontends", nil)
	w := httptest.NewRecorder()

	microFrontendsHandler(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("microFrontendsHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	var resp []MicroFrontend
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Errorf("microFrontendsHandler returned invalid JSON: %v", err)
	}

	if len(resp) != 1 || resp[0].Name != "example" {
		t.Errorf("microFrontendsHandler returned unexpected body: got %v want %v", resp, []MicroFrontend{{Name: "example"}})
	}
}

func TestVersionHandler(t *testing.T) {
	// Set up a mock environment with the SD_VERSION variable
	mockEnv := mockEnvGetter{envs: map[string]string{"SD_VERSION": "1.2.3"}}

	// Create a new request to the /version endpoint
	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	// Call the version handler with the mock environment
	versionHandler(mockEnv)(w, req)

	// Check the status code is 200 OK
	if status := w.Code; status != http.StatusOK {
		t.Errorf("versionHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	// Verify the response contains the expected version
	expected := `{"version": "1.2.3"}`
	if strings.TrimSpace(w.Body.String()) != expected {
		t.Errorf("versionHandler returned unexpected body: got %v want %v", w.Body.String(), expected)
	}
}

func TestVersionHandlerDefault(t *testing.T) {
	// Set up a mock environment with no SD_VERSION variable
	mockEnv := mockEnvGetter{envs: map[string]string{}}

	req := httptest.NewRequest(http.MethodGet, "/version", nil)
	w := httptest.NewRecorder()

	versionHandler(mockEnv)(w, req)

	if status := w.Code; status != http.StatusOK {
		t.Errorf("versionHandler returned wrong status code: got %v want %v", status, http.StatusOK)
	}

	expected := `{"version": "local"}`
	if strings.TrimSpace(w.Body.String()) != expected {
		t.Errorf("versionHandler returned unexpected body: got %v want %v", w.Body.String(), expected)
	}
}

func TestServerRoutes(t *testing.T) {
	// Use a mock environment getter
	mockEnv := mockEnvGetter{
		envs: map[string]string{"SD_PORT": "9090"},
	}

	// Initialize the server
	server := initServer(mockEnv)

	// Ensure the server is set to the correct port
	if server.Addr != ":9090" {
		t.Errorf("server initialized with wrong port: got %v want %v", server.Addr, ":9090")
	}
}
