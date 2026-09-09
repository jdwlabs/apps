package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"libs/backend/shared/auth/authhttp"
)

// The sentinels the store reports, one per outcome GlobalExceptionHandler maps
// to a distinct status. They carry no message: the wording depends on which
// operation asked, so each handler composes its own.
var (
	ErrUserNotFound = errors.New("user not found")
	ErrUserExists   = errors.New("user exists with email address")
	ErrRoleNotFound = errors.New("role not found")
	ErrRoleExists   = errors.New("role exists with name")
	// ErrNameTaken is the frozen 500: auth.users.email_address and
	// auth.roles.role_name are unique, neither update pre-checks the new value,
	// and no handler maps the resulting constraint violation. It is a named
	// outcome rather than a raw driver error so the status is a decision a
	// reviewer can see and the parity suite can assert it.
	ErrNameTaken = errors.New("another row already holds that unique value")
	// ErrBadCredentials covers both authentication failures — an unknown email
	// and a wrong password — because answering them differently tells an
	// anonymous caller which addresses are registered.
	ErrBadCredentials = errors.New("credentials are not valid")
)

// notFound names which row a lookup did not find. The grant and revoke
// operations check several ids in one call and the message the caller sees
// carries the offending one, so the id travels with the error rather than being
// re-derived by the handler.
type notFound struct {
	sentinel error
	id       int64
}

func (e *notFound) Error() string { return fmt.Sprintf("%v: %d", e.sentinel, e.id) }

func (e *notFound) Unwrap() error { return e.sentinel }

// missingID reports the id a notFound carried, falling back to the one the
// caller already knows for a sentinel raised without one.
func missingID(err error, fallback int64) int64 {
	var missing *notFound
	if errors.As(err, &missing) {
		return missing.id
	}
	return fallback
}

// contentTypeText is what Spring's StringHttpMessageConverter writes, charset
// and all, for every handler that answers with an exception message.
const contentTypeText = "text/plain;charset=UTF-8"

const contentTypeJSON = "application/json"

func writeText(w http.ResponseWriter, status int, body string) {
	w.Header().Set("Content-Type", contentTypeText)
	w.WriteHeader(status)
	if _, err := w.Write([]byte(body)); err != nil {
		slog.Error("could not write the response body", "error", err)
	}
}

// writeNotFound answers as the ResourceNotFoundException handler does.
func writeNotFound(w http.ResponseWriter, message string) {
	writeText(w, http.StatusNotFound, message)
}

// writeConflict answers as the ResourceExistsException handler does.
func writeConflict(w http.ResponseWriter, message string) {
	writeText(w, http.StatusConflict, message)
}

// writeUnreadableBody answers as the HttpMessageNotReadableException handler
// does, with its fixed string rather than the parse error.
func writeUnreadableBody(w http.ResponseWriter) {
	writeText(w, http.StatusBadRequest, "Request body is invalid. Please check the format and try again.")
}

// writeValidationErrors answers as the MethodArgumentNotValidException handler
// does: one entry per rejected field.
func writeValidationErrors(w http.ResponseWriter, fields map[string]string) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(fields); err != nil {
		slog.Error("could not write the validation errors", "error", err)
	}
}

// writeUnconvertableParameter answers a path variable or a query parameter that
// is not a number. Spring's argument resolution fails before the handler is
// invoked and nothing in GlobalExceptionHandler catches the resulting
// exception, so the status is one the container sets rather than a handler
// composes — and carries the body the container renders with it.
func writeUnconvertableParameter(w http.ResponseWriter, r *http.Request) {
	writeContainerError(w, r, http.StatusBadRequest)
}

// writeContainerError writes Boot's error representation, which the JVM
// produces for every status it reaches by throwing or by sendError rather than
// by composing a response: an argument that would not convert, a storage
// failure, an authorization rule that could not be decided. The shared writer
// decides between that body and the empty 401, because the JVM decides it from
// the same thing — whether the caller's token survives the internal forward to
// /error. The shared router answers its own refusals through the same writer,
// so a 404 and a storage failure carry the same shape.
func writeContainerError(w http.ResponseWriter, r *http.Request, status int) {
	authhttp.WriteContainerError(w, r, status)
}

// writeJSON answers with a composed body, as every successful operation does.
func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Error("could not write the response body", "error", err)
	}
}

// parseID converts a path variable the way Spring converts a Long, which
// accepts a signed decimal and nothing else — no surrounding space, no decimal
// point, nothing outside the type's range.
func parseID(text string) (int64, bool) {
	value, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// logRollbackFailure records a rollback that did not complete. It is separate
// from the caller so the store file carries no logging concern of its own.
func logRollbackFailure(err error) {
	slog.Error("could not roll back a transaction", "error", err)
}
