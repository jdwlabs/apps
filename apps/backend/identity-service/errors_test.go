package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"testing"

	"libs/backend/shared/auth"
	"libs/backend/shared/auth/authhttp"
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
			write:       func(w http.ResponseWriter) { writeNotFound(w, "User not found with id 42") },
			status:      http.StatusNotFound,
			contentType: "text/plain;charset=UTF-8",
			body:        "User not found with id 42",
		},
		{
			name:        "an existing resource is the exception message as text",
			write:       func(w http.ResponseWriter) { writeConflict(w, "Role already exists with name ADMIN") },
			status:      http.StatusConflict,
			contentType: "text/plain;charset=UTF-8",
			body:        "Role already exists with name ADMIN",
		},
		{
			name:        "an unreadable body is a fixed string",
			write:       writeUnreadableBody,
			status:      http.StatusBadRequest,
			contentType: "text/plain;charset=UTF-8",
			body:        "Request body is invalid. Please check the format and try again.",
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

	writeValidationErrors(response, map[string]string{"emailAddress": "emailAddress is mandatory"})

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
	if decoded["emailAddress"] != "emailAddress is mandatory" {
		t.Errorf("errors = %v, want emailAddress mandatory", decoded)
	}
}

func TestAnUnconvertableParameterCarriesTheContainerErrorBody(t *testing.T) {
	// Spring's type conversion fails before the handler runs and nothing in
	// GlobalExceptionHandler catches it, so this is a status the container sets
	// through sendError — and the forward to /error that sendError triggers
	// re-authenticates with the caller's own token, so BasicErrorController
	// renders its body. Measured on a booted usersrole: GET /api/users/abc with a
	// valid token answers 400 application/json, not 400 with nothing.
	response := httptest.NewRecorder()
	request := authenticatedRequest(http.MethodGet, "/api/users/abc")

	writeUnconvertableParameter(response, request)

	assertContainerErrorBody(t, response, http.StatusBadRequest, "/api/users/abc")
}

func TestAnUnconvertableParameterAnswers401WithoutAToken(t *testing.T) {
	// The same forward, refused a second time. No operation taking a numeric
	// parameter is reachable anonymously today, so the writer decides the shape
	// from the request rather than from the caller's promise — which is the
	// property worth pinning.
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/users/abc", nil)

	writeUnconvertableParameter(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
	if body := response.Body.String(); body != "" {
		t.Errorf("body = %q, want empty", body)
	}
}

func TestTheContainerErrorBodyCarriesTheStatusAndPath(t *testing.T) {
	// Every status this service does not compose a body for reaches the wire
	// through this writer. The named one is the frozen 500: an update onto an
	// email address or a role name another row already holds.
	response := httptest.NewRecorder()

	writeContainerError(response, authenticatedRequest(http.MethodPut, "/api/users/7"), http.StatusInternalServerError)

	assertContainerErrorBody(t, response, http.StatusInternalServerError, "/api/users/7")
}

func TestAMissingRowCarriesTheIdTheMessageHasToName(t *testing.T) {
	// The grant and revoke operations check several ids in one call, so the
	// handler cannot re-derive which one was absent from the request alone.
	err := &notFound{sentinel: ErrRoleNotFound, id: 42}

	if !errors.Is(err, ErrRoleNotFound) {
		t.Error("the wrapped sentinel does not match; the handler would answer 500 instead of 404")
	}
	if got := missingID(err, 7); got != 42 {
		t.Errorf("missingID = %d, want the id the error carried", got)
	}
	if got := missingID(ErrRoleNotFound, 7); got != 7 {
		t.Errorf("missingID = %d, want the caller's fallback for a bare sentinel", got)
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

// containerErrorKeys is the exact key set and order BasicErrorController writes
// with server.error.include-message left at its default of never. message is
// omitted entirely rather than present and blank.
var containerErrorKeys = []string{"timestamp", "status", "error", "path"}

// bootTimestampPattern matches the millisecond-precision, zero-offset stamp
// Jackson renders for the java.util.Date DefaultErrorAttributes writes. The
// instant is never asserted, only the shape: two implementations answering the
// same request do so at different moments.
var bootTimestampPattern = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

// authenticatedRequest carries a verified principal, which is what decides the
// shape of every container-set status: in the JVM the forward to /error either
// re-authenticates with the caller's token or is refused a second time, and the
// principal on the context is this service's equivalent of holding one.
func authenticatedRequest(method, path string) *http.Request {
	request := httptest.NewRequest(method, path, nil)
	principal := &auth.Principal{Subject: "admin@jdw.com", Roles: []string{"ADMIN"}}
	return request.WithContext(authhttp.WithPrincipal(request.Context(), principal))
}

// assertContainerErrorBody pins the body against what a booted usersrole
// answers an authenticated caller for a status the container set. status and
// error are asserted by value; timestamp and path vary per request, so they are
// asserted by shape.
func assertContainerErrorBody(t *testing.T, response *httptest.ResponseRecorder, status int, path string) {
	t.Helper()
	if response.Code != status {
		t.Errorf("status = %d, want %d", response.Code, status)
	}
	if got, want := response.Header().Get("Content-Type"), contentTypeJSON; got != want {
		t.Errorf("Content-Type = %q, want %q", got, want)
	}

	body := response.Body.Bytes()
	if keys := jsonKeysInOrder(t, body); !slices.Equal(keys, containerErrorKeys) {
		t.Fatalf("body keys = %v, want %v in that order (message must be absent: include-message is never)",
			keys, containerErrorKeys)
	}

	var fields map[string]any
	if err := json.Unmarshal(body, &fields); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	if fields["status"] != float64(status) {
		t.Errorf("status field = %v, want %d", fields["status"], status)
	}
	if fields["error"] != http.StatusText(status) {
		t.Errorf("error field = %v, want %q", fields["error"], http.StatusText(status))
	}
	if fields["path"] != path {
		t.Errorf("path field = %v, want %q", fields["path"], path)
	}
	timestamp, ok := fields["timestamp"].(string)
	if !ok || !bootTimestampPattern.MatchString(timestamp) {
		t.Errorf("timestamp field = %v, want millisecond precision and a Z offset", fields["timestamp"])
	}
}

// jsonKeysInOrder returns the top-level object keys of body in the order they
// appear on the wire. Decoding into a map loses that order, and the order is
// part of what is pinned.
func jsonKeysInOrder(t *testing.T, body []byte) []string {
	t.Helper()
	decoder := json.NewDecoder(bytes.NewReader(body))
	if _, err := decoder.Token(); err != nil {
		t.Fatalf("body is not a JSON object: %v", err)
	}
	keys := []string{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			t.Fatalf("reading the body's keys: %v", err)
		}
		keys = append(keys, key.(string))
		var discard json.RawMessage
		if err := decoder.Decode(&discard); err != nil {
			t.Fatalf("reading the body's values: %v", err)
		}
	}
	return keys
}
