package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestErrorWritersReproduceTheStatusAndMediaTypeEachHandlerBuilds(t *testing.T) {
	cases := []struct {
		name        string
		write       func(http.ResponseWriter)
		status      int
		contentType string
		body        string
	}{
		{
			name:        "a missing resource is the exception message as text",
			write:       func(w http.ResponseWriter) { writeNotFound(w, "Profile not found with id 42") },
			status:      http.StatusNotFound,
			contentType: "text/plain;charset=UTF-8",
			body:        "Profile not found with id 42",
		},
		{
			name:        "an existing resource is the exception message as text",
			write:       func(w http.ResponseWriter) { writeConflict(w, "Profile already exists for user with id 7") },
			status:      http.StatusConflict,
			contentType: "text/plain;charset=UTF-8",
			body:        "Profile already exists for user with id 7",
		},
		{
			name:        "an unreadable body is a fixed string",
			write:       writeUnreadableBody,
			status:      http.StatusBadRequest,
			contentType: "text/plain;charset=UTF-8",
			body:        "Request body is invalid. Please check the format and try again.",
		},
		{
			name:        "a failed icon read is the exception message as text",
			write:       func(w http.ResponseWriter) { writeIconUploadFailed(w, "Failed to add icon") },
			status:      http.StatusInternalServerError,
			contentType: "text/plain;charset=UTF-8",
			body:        "Failed to add icon",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()

			tc.write(response)

			if response.Code != tc.status {
				t.Errorf("status = %d, want %d", response.Code, tc.status)
			}
			if got := response.Header().Get("Content-Type"); got != tc.contentType {
				t.Errorf("Content-Type = %q, want %q", got, tc.contentType)
			}
			if got := response.Body.String(); got != tc.body {
				t.Errorf("body = %q, want %q", got, tc.body)
			}
		})
	}
}

func TestValidationErrorsAreAJsonObjectOfFieldToMessage(t *testing.T) {
	response := httptest.NewRecorder()

	writeValidationErrors(response, map[string]string{"firstName": "firstName is mandatory"})

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if got, want := response.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	var decoded map[string]string
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	if decoded["firstName"] != "firstName is mandatory" {
		t.Errorf("errors = %v, want firstName mandatory", decoded)
	}
}

func TestAPathVariableThatIsNotANumberIsRefusedWithNoBody(t *testing.T) {
	// Spring's type conversion fails before the handler runs, and nothing in
	// GlobalExceptionHandler catches it, so the container writes the status
	// with no message.
	response := httptest.NewRecorder()

	writeUnconvertablePathVariable(response)

	if response.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
	if body := response.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
	if contentType := response.Header().Get("Content-Type"); contentType != "" {
		t.Errorf("Content-Type = %q, want unset", contentType)
	}
}

func TestTheContainerErrorBodyCarriesTheStatusAndPath(t *testing.T) {
	// The one status in this service that answers with the container's own
	// error representation rather than a composed body.
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPut, "/api/profiles/7/icon", nil)

	writeContainerError(response, request, http.StatusInternalServerError)

	if response.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", response.Code, http.StatusInternalServerError)
	}
	if got, want := response.Header().Get("Content-Type"), "application/json"; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}
	var decoded struct {
		Timestamp string `json:"timestamp"`
		Status    int    `json:"status"`
		Error     string `json:"error"`
		Path      string `json:"path"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	if decoded.Status != http.StatusInternalServerError {
		t.Errorf("status field = %d, want 500", decoded.Status)
	}
	if decoded.Error != "Internal Server Error" {
		t.Errorf("error field = %q, want %q", decoded.Error, "Internal Server Error")
	}
	if decoded.Path != "/api/profiles/7/icon" {
		t.Errorf("path field = %q, want %q", decoded.Path, "/api/profiles/7/icon")
	}
	if decoded.Timestamp == "" {
		t.Error("timestamp field is empty")
	}
}

func TestParsingAPathVariable(t *testing.T) {
	cases := map[string]struct {
		text  string
		value int64
		ok    bool
	}{
		"a positive id":      {text: "42", value: 42, ok: true},
		"a negative id":      {text: "-1", value: -1, ok: true},
		"a word":             {text: "user", ok: false},
		"empty":              {text: "", ok: false},
		"a decimal":          {text: "4.2", ok: false},
		"beyond int64":       {text: "9223372036854775808", ok: false},
		"leading whitespace": {text: " 42", ok: false},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			value, ok := parseID(tc.text)

			if ok != tc.ok {
				t.Fatalf("ok = %v, want %v", ok, tc.ok)
			}
			if ok && value != tc.value {
				t.Errorf("value = %d, want %d", value, tc.value)
			}
		})
	}
}
