package util

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestGetEnvOrDefault(t *testing.T) {
	// Test with an existing environment variable
	_ = os.Setenv("EXISTING_VAR", "value")
	result := GetEnvOrDefault("EXISTING_VAR", "fallback")
	if result != "value" {
		t.Errorf("Expected 'value', got '%s'", result)
	}

	// Test with a non-existing environment variable
	result = GetEnvOrDefault("NON_EXISTING_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", result)
	}

	// Test with an existing but empty environment variable
	_ = os.Setenv("EMPTY_VAR", "")
	result = GetEnvOrDefault("EMPTY_VAR", "fallback")
	if result != "fallback" {
		t.Errorf("Expected 'fallback', got '%s'", result)
	}

	// Clean up environment variables
	_ = os.Unsetenv("EXISTING_VAR")
	_ = os.Unsetenv("EMPTY_VAR")
}

func TestLoggingDifferentMethods(t *testing.T) {
	methods := []string{"GET", "POST", "PUT", "DELETE"}
	for _, method := range methods {
		req := httptest.NewRequest(method, "https://example.com/foo", nil)
		w := httptest.NewRecorder()

		handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		handler.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status code %d for method %s, got %d", http.StatusOK, method, w.Code)
		}
	}
}

func TestLoggingDifferentStatusCode(t *testing.T) {
	statusCodes := []int{http.StatusOK, http.StatusNotFound, http.StatusInternalServerError}
	for _, code := range statusCodes {
		req := httptest.NewRequest("GET", "https://example.com/foo", nil)
		w := httptest.NewRecorder()

		handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(code)
		}))

		handler.ServeHTTP(w, req)

		if w.Code != code {
			t.Errorf("Expected status code %d, got %d", code, w.Code)
		}
	}
}

func TestLoggingMissingHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggingClientIP(t *testing.T) {
	// Test with X-Forwarded-For header
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Test with X-Real-IP header
	req = httptest.NewRequest("GET", "https://example.com/foo", nil)
	req.Header.Set("X-Real-IP", "198.51.100.1")
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Test with RemoteAddr fallback
	req = httptest.NewRequest("GET", "https://example.com/foo", nil)
	req.RemoteAddr = "192.0.2.1:12345"
	w = httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggingRequestTiming(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond) // Simulate a delay
		w.WriteHeader(http.StatusOK)
	}))

	start := time.Now()
	handler.ServeHTTP(w, req)
	duration := time.Since(start)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}

	// Check that the duration is at least 100ms
	if duration < 100*time.Millisecond {
		t.Errorf("Expected duration to be at least 100ms, got %v", duration)
	}
}

func TestLoggingUserAgent(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	req.Header.Set("User-Agent", "Go-http-client/1.1")
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggingCustomHeaders(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	req.Header.Set("X-Custom-Header", "CustomValue")
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code %d, got %d", http.StatusOK, w.Code)
	}
}

func TestLoggingErrorScenario(t *testing.T) {
	req := httptest.NewRequest("GET", "https://example.com/foo", nil)
	w := httptest.NewRecorder()

	handler := Logging(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status code %d, got %d", http.StatusInternalServerError, w.Code)
	}
}
