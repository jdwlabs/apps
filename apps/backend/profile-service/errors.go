package main

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"libs/backend/shared/auth/authhttp"
)

// The sentinels the store reports, one per outcome GlobalExceptionHandler maps
// to a distinct status. They carry no message: the wording depends on which
// operation asked, so each handler composes its own.
var (
	ErrProfileNotFound = errors.New("profile not found")
	ErrUserNotFound    = errors.New("user not found")
	ErrProfileExists   = errors.New("profile exists for user")
	ErrAddressNotFound = errors.New("address not found")
	ErrIconNotFound    = errors.New("profile icon not found")
	ErrIconExists      = errors.New("icon exists for profile")
	// ErrNoIconToReplace is the frozen defect in updateIcon: the JVM reads the
	// current icon's id to carry onto the replacement and dereferences null when
	// there is none, answering 500 rather than 404. It is a named outcome here
	// rather than a nil dereference so that the status is a decision a reviewer
	// can see, and so the parity suite can assert it.
	ErrNoIconToReplace = errors.New("profile has no icon to replace")
)

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

// writeIconUploadFailed answers as the IconUploadException handler does.
func writeIconUploadFailed(w http.ResponseWriter, message string) {
	writeText(w, http.StatusInternalServerError, message)
}

// writeMultipartRefused answers as the MultipartException and
// MissingServletRequestPartException handlers do: 400 with the exception's own
// message.
func writeMultipartRefused(w http.ResponseWriter, message string) {
	writeText(w, http.StatusBadRequest, message)
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
