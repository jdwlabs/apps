package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"libs/backend/shared/auth/authhttp"
	"libs/backend/shared/auth/authz"
)

// The page bounds ProfileService applies, independent of what a caller asks
// for, so the listing always issues a bounded query.
const (
	defaultPage    = 0
	defaultSize    = 100
	minimumSize    = 1
	maximumSize    = 500
	maxUploadBytes = 2 << 20
)

// The multipart part name is fixed by the handler's @RequestParam("icon"); a
// different name is a missing part.
const iconPartName = "icon"

type handlers struct {
	store      Store
	authorizer authz.Authorizer
}

// Operation is one served operation together with the contract rule that
// authorizes it, so the rule is declared beside the route rather than buried in
// the handler that enforces it.
type Operation struct {
	Method   string
	Pattern  string
	Rule     authz.Rule
	Produces string
	Handler  http.HandlerFunc
}

func (h *handlers) operations() []Operation {
	return []Operation{
		{
			Method: http.MethodGet, Pattern: "/api/profiles",
			Rule: authz.RuleAdmin, Handler: h.getProfiles,
		},
		{
			Method: http.MethodPost, Pattern: "/api/profiles",
			Rule: authz.RuleAdminOrSelfByBodyUserID, Handler: h.createProfile,
		},
		{
			Method: http.MethodGet, Pattern: "/api/profiles/{profileId}",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.getProfileByID,
		},
		{
			Method: http.MethodPut, Pattern: "/api/profiles/{profileId}",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.updateProfileByID,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/profiles/{profileId}",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.deleteProfileByID,
		},
		{
			Method: http.MethodGet, Pattern: "/api/profiles/by-user/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.getProfileByUserID,
		},
		{
			Method: http.MethodPut, Pattern: "/api/profiles/by-user/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.updateProfileByUserID,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/profiles/by-user/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.deleteProfileByUserID,
		},
		{
			Method: http.MethodPost, Pattern: "/api/profiles/{profileId}/address",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.addAddress,
		},
		{
			Method: http.MethodPut, Pattern: "/api/profiles/{profileId}/address/{addressId}",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.updateAddress,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/profiles/{profileId}/address/{addressId}",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.deleteAddress,
		},
		{
			Method: http.MethodGet, Pattern: "/api/profiles/{profileId}/icon",
			Rule: authz.RuleAdminOrSelfByProfileID, Produces: "image/png", Handler: h.getIcon,
		},
		{
			Method: http.MethodPost, Pattern: "/api/profiles/{profileId}/icon",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.addIcon,
		},
		{
			Method: http.MethodPut, Pattern: "/api/profiles/{profileId}/icon",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.updateIcon,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/profiles/{profileId}/icon",
			Rule: authz.RuleAdminOrSelfByProfileID, Handler: h.deleteIcon,
		},
	}
}

func (h *handlers) getProfiles(w http.ResponseWriter, r *http.Request) {
	page, ok := queryInt(w, r, "page", defaultPage)
	if !ok {
		return
	}
	size, ok := queryInt(w, r, "size", defaultSize)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdmin, authz.Subject{}) {
		return
	}

	// Out-of-range input is clamped rather than rejected: a negative page is
	// served as page 0 and an enormous size as 500, both with a 200.
	page = max(page, defaultPage)
	size = min(max(size, minimumSize), maximumSize)

	profiles, err := h.store.ListProfiles(r.Context(), size, page*size)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (h *handlers) createProfile(w http.ResponseWriter, r *http.Request) {
	request, ok := decode[ProfileCreateRequest](w, r)
	if !ok {
		return
	}
	// The only rule in either service that reads the request body. It is passed
	// as BodyUserID rather than UserID so that a body cannot satisfy a
	// path-scoped rule.
	if !h.authorize(w, r, authz.RuleAdminOrSelfByBodyUserID, authz.Subject{BodyUserID: request.UserID}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.CreateProfile(r.Context(), request, actor)
	switch {
	case errors.Is(err, ErrProfileExists):
		writeConflict(w, fmt.Sprintf("Profile already exists for user with id %d", *request.UserID))
	case errors.Is(err, ErrUserNotFound):
		writeNotFound(w, fmt.Sprintf("User not found with user id %d", *request.UserID))
	case err != nil:
		h.fail(w, r, err)
	default:
		writeJSON(w, http.StatusCreated, profile)
	}
}

func (h *handlers) getProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}

	profile, err := h.store.ProfileByID(r.Context(), profileID)
	if err != nil {
		h.failProfile(w, r, err, profileID)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) updateProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	request, ok := decode[ProfileUpdateRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.UpdateProfileByID(r.Context(), profileID, request, actor)
	if err != nil {
		h.failProfile(w, r, err, profileID)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) deleteProfileByID(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	if _, ok := actingUser(w, r); !ok {
		return
	}

	// A no-op for an id that does not exist, and still 204: the repository
	// deletes without reading first, which is why this operation has no 404 in
	// its response set at all.
	if err := h.store.DeleteProfileByID(r.Context(), profileID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) getProfileByUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: &userID}) {
		return
	}

	profile, err := h.store.ProfileByUserID(r.Context(), userID)
	if errors.Is(err, ErrProfileNotFound) {
		writeNotFound(w, fmt.Sprintf("Profile not found with user id %d", userID))
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) updateProfileByUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	request, ok := decode[ProfileUpdateRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: &userID}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.UpdateProfileByUserID(r.Context(), userID, request, actor)
	if errors.Is(err, ErrProfileNotFound) {
		writeNotFound(w, fmt.Sprintf("Profile not found with user id %d", userID))
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) deleteProfileByUserID(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: &userID}) {
		return
	}
	if _, ok := actingUser(w, r); !ok {
		return
	}

	if err := h.store.DeleteProfileByUserID(r.Context(), userID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) addAddress(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	request, ok := decode[AddressRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.AddAddress(r.Context(), profileID, request, actor)
	if err != nil {
		h.failProfile(w, r, err, profileID)
		return
	}
	// 200, not 201, unlike every other create in either service: the handler
	// builds ResponseEntity.ok, and the body is the parent profile rather than
	// the created address.
	writeJSON(w, http.StatusOK, profile)
}

func (h *handlers) updateAddress(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	addressID, ok := pathID(w, r, "addressId")
	if !ok {
		return
	}
	request, ok := decode[AddressRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.UpdateAddress(r.Context(), profileID, addressID, request, actor)
	switch {
	case errors.Is(err, ErrAddressNotFound):
		writeNotFound(w, fmt.Sprintf("Address not found with id %d", addressID))
	case err != nil:
		h.failProfile(w, r, err, profileID)
	default:
		writeJSON(w, http.StatusOK, profile)
	}
}

func (h *handlers) deleteAddress(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	addressID, ok := pathID(w, r, "addressId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	if _, ok := actingUser(w, r); !ok {
		return
	}

	// Scoped to the profile in the path, so an address belonging to another
	// profile is a miss rather than a deletion.
	err := h.store.DeleteAddress(r.Context(), profileID, addressID)
	switch {
	case errors.Is(err, ErrAddressNotFound):
		writeNotFound(w, fmt.Sprintf("Address not found with id %d for profile with id %d", addressID, profileID))
	case err != nil:
		h.fail(w, r, err)
	default:
		w.WriteHeader(http.StatusNoContent)
	}
}

func (h *handlers) getIcon(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}

	icon, err := h.store.Icon(r.Context(), profileID)
	if errors.Is(err, ErrIconNotFound) {
		writeNotFound(w, fmt.Sprintf("Profile icon not found with id %d", profileID))
		return
	}
	if err != nil {
		h.fail(w, r, err)
		return
	}
	// Declared image/png by the handler's produces. Nothing validates that the
	// stored bytes are a PNG, and sniffing them here would change the type of
	// every row uploaded before anyone thought to check.
	w.Header().Set("Content-Type", "image/png")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(icon.Icon); err != nil {
		slog.Error("could not write the icon bytes", "error", err, "profileId", profileID)
	}
}

func (h *handlers) addIcon(w http.ResponseWriter, r *http.Request) {
	profileID, icon, ok := h.acceptIcon(w, r)
	if !ok {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.AddIcon(r.Context(), profileID, icon, actor)
	switch {
	case errors.Is(err, ErrIconExists):
		// The message carries a colon before the id, unlike every other
		// conflict message; transcribed rather than tidied.
		writeConflict(w, fmt.Sprintf("Icon already exists for profile with id: %d", profileID))
	case err != nil:
		h.failProfile(w, r, err, profileID)
	default:
		writeJSON(w, http.StatusOK, profile)
	}
}

func (h *handlers) updateIcon(w http.ResponseWriter, r *http.Request) {
	profileID, icon, ok := h.acceptIcon(w, r)
	if !ok {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	profile, err := h.store.ReplaceIcon(r.Context(), profileID, icon, actor)
	switch {
	case errors.Is(err, ErrNoIconToReplace):
		// Replacing an icon on a profile that has none is 500, not 404: the
		// JVM dereferences the absent icon while carrying its id onto the
		// replacement. Frozen, because the frontends' error text is keyed on
		// the status.
		writeContainerError(w, r, http.StatusInternalServerError)
	case err != nil:
		h.failProfile(w, r, err, profileID)
	default:
		writeJSON(w, http.StatusOK, profile)
	}
}

func (h *handlers) deleteIcon(w http.ResponseWriter, r *http.Request) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return
	}
	if _, ok := actingUser(w, r); !ok {
		return
	}

	if err := h.store.DeleteIcon(r.Context(), profileID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// acceptIcon does the work both upload handlers share: read the path variable,
// read the part, and authorize. Multipart resolution precedes the authorization
// check because Spring resolves the request before the method interceptor runs,
// so an oversized upload is 400 for a caller the rule would have refused.
func (h *handlers) acceptIcon(w http.ResponseWriter, r *http.Request) (int64, []byte, bool) {
	profileID, ok := pathID(w, r, "profileId")
	if !ok {
		return 0, nil, false
	}
	icon, ok := readIconPart(w, r)
	if !ok {
		return 0, nil, false
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByProfileID, authz.Subject{ProfileID: &profileID}) {
		return 0, nil, false
	}
	return profileID, icon, true
}

// readIconPart reads the single part the handlers consume, under the same 2 MB
// caps spring.servlet.multipart applies to the file and to the whole request.
func readIconPart(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || !strings.HasPrefix(mediaType, "multipart/") {
		writeMultipartRefused(w, "Current request is not a multipart request")
		return nil, false
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes)
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		var toLarge *http.MaxBytesError
		if errors.As(err, &toLarge) {
			writeMultipartRefused(w, "Maximum upload size exceeded")
			return nil, false
		}
		writeMultipartRefused(w, "Failed to parse multipart servlet request")
		return nil, false
	}
	defer func() {
		if err := r.MultipartForm.RemoveAll(); err != nil {
			slog.Error("could not clear the multipart temporary files", "error", err)
		}
	}()

	part, _, err := r.FormFile(iconPartName)
	if errors.Is(err, http.ErrMissingFile) {
		writeMultipartRefused(w, "Required part '"+iconPartName+"' is not present.")
		return nil, false
	}
	if err != nil {
		writeIconUploadFailed(w, "Failed to add icon")
		return nil, false
	}
	defer func() {
		if err := part.Close(); err != nil {
			slog.Error("could not close the uploaded part", "error", err)
		}
	}()

	icon, err := io.ReadAll(part)
	if err != nil {
		writeIconUploadFailed(w, "Failed to add icon")
		return nil, false
	}
	return icon, true
}

func (h *handlers) authorize(w http.ResponseWriter, r *http.Request, rule authz.Rule, subject authz.Subject) bool {
	return authhttp.Authorize(w, r, h.authorizer, rule, subject)
}

// fail answers a storage failure. It is a 500 with no body, and the cause goes
// to the log rather than to the caller: an error message from a database is a
// disclosure, not a diagnosis a client can act on.
func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("the request could not be served", "error", err, "method", r.Method, "path", r.URL.Path)
	w.WriteHeader(http.StatusInternalServerError)
}

// failProfile answers the outcome every profile-scoped operation shares before
// falling back to a plain failure.
func (h *handlers) failProfile(w http.ResponseWriter, r *http.Request, err error, profileID int64) {
	if errors.Is(err, ErrProfileNotFound) {
		writeNotFound(w, fmt.Sprintf("Profile not found with id %d", profileID))
		return
	}
	h.fail(w, r, err)
}

// actingUser reads the user the audit columns record. The JVM resolves it by
// reading auth.users for the token's subject; here it is the verified user_id
// claim, so the read disappears from the request path. A token carrying no
// user_id names nobody this service can attribute a write to, and answers as
// the JVM answers a subject with no user row.
func actingUser(w http.ResponseWriter, r *http.Request) (int64, bool) {
	principal, present := authhttp.PrincipalFrom(r.Context())
	if !present || principal.UserID == nil {
		subject := ""
		if principal != nil {
			subject = principal.Subject
		}
		writeNotFound(w, "User not found with email address "+subject)
		return 0, false
	}
	return *principal.UserID, true
}

func pathID(w http.ResponseWriter, r *http.Request, name string) (int64, bool) {
	value, ok := parseID(r.PathValue(name))
	if !ok {
		writeUnconvertablePathVariable(w)
		return 0, false
	}
	return value, true
}

// queryInt reads a numeric query parameter the way Spring converts one, and
// refuses text that is not a number before the handler runs.
func queryInt(w http.ResponseWriter, r *http.Request, name string, fallback int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		writeUnconvertablePathVariable(w)
		return 0, false
	}
	return value, true
}

// decode reads and validates a JSON body, answering as the two handlers that
// cover those failures answer. Both run during Spring's argument resolution,
// ahead of the method interceptor, so a malformed body is 400 even for a caller
// the operation's rule would refuse.
func decode[T interface{ Validate() map[string]string }](w http.ResponseWriter, r *http.Request) (T, bool) {
	var request T
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&request); err != nil {
		writeUnreadableBody(w)
		return request, false
	}
	if fields := request.Validate(); len(fields) > 0 {
		writeValidationErrors(w, fields)
		return request, false
	}
	return request, true
}
