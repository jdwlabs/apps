package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"

	"libs/backend/shared/auth/authhttp"
	"libs/backend/shared/auth/authz"
)

// The page bounds the services apply, independent of what a caller asks for, so
// a listing always issues a bounded query.
const (
	defaultPage = 0
	defaultSize = 100
	minimumSize = 1
	maximumSize = 500
	// maxRequestBytes bounds a JSON body. It is generous next to the largest
	// body any operation takes and small next to what an anonymous caller could
	// otherwise make this process allocate.
	maxRequestBytes = 1 << 20
)

// registrationRequesterID is the requester the public registration audits its
// row as. UserService.createUser(dto) hard-codes it, and nothing in the schema
// constrains created_by_user_id to an existing user, so the value stands even
// on an empty database.
const registrationRequesterID = int64(1)

// elevatedRoleID is the role a caller must already hold to grant or revoke it.
// UserService.validateAdminRoleRequest hard-codes the id rather than the name;
// transcribed as an id, because the seed data resolves ADMIN by natural key and
// no longer guarantees that ADMIN is the row this names.
const elevatedRoleID = int64(1)

type handlers struct {
	store      Store
	minter     *minter
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
			Method: http.MethodPost, Pattern: "/auth/authenticate",
			Rule: authz.RulePublic, Handler: h.authenticate,
		},
		{
			Method: http.MethodPost, Pattern: "/auth/user",
			Rule: authz.RulePublic, Handler: h.registerUser,
		},
		{
			Method: http.MethodGet, Pattern: "/api/users",
			Rule: authz.RuleAdmin, Handler: h.getAllUsers,
		},
		{
			Method: http.MethodPost, Pattern: "/api/users",
			Rule: authz.RuleAuthenticated, Handler: h.createUser,
		},
		{
			Method: http.MethodGet, Pattern: "/api/users/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.getUserByID,
		},
		{
			Method: http.MethodPut, Pattern: "/api/users/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.updateUser,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/users/{userId}",
			Rule: authz.RuleAdminOrSelfByUserID, Handler: h.deleteUser,
		},
		{
			Method: http.MethodGet, Pattern: "/api/users/email/{emailAddress}",
			Rule: authz.RuleAdminOrSelfByEmail, Handler: h.getUserByEmailAddress,
		},
		{
			Method: http.MethodPut, Pattern: "/api/users/{userId}/roles/grant",
			Rule: authz.RuleAdminOrManager, Handler: h.grantRolesToUser,
		},
		{
			Method: http.MethodPut, Pattern: "/api/users/{userId}/roles/revoke",
			Rule: authz.RuleAdminOrManager, Handler: h.revokeRolesFromUser,
		},
		{
			Method: http.MethodGet, Pattern: "/api/roles",
			Rule: authz.RuleAuthenticated, Handler: h.getAllRoles,
		},
		{
			Method: http.MethodPost, Pattern: "/api/roles",
			Rule: authz.RuleAdmin, Handler: h.createRole,
		},
		{
			Method: http.MethodGet, Pattern: "/api/roles/{roleId}",
			Rule: authz.RuleAuthenticated, Handler: h.getRoleByID,
		},
		{
			Method: http.MethodPut, Pattern: "/api/roles/{roleId}",
			Rule: authz.RuleAdmin, Handler: h.updateRole,
		},
		{
			Method: http.MethodDelete, Pattern: "/api/roles/{roleId}",
			Rule: authz.RuleAdmin, Handler: h.deleteRole,
		},
		{
			Method: http.MethodGet, Pattern: "/api/roles/name/{roleName}",
			Rule: authz.RuleAuthenticated, Handler: h.getRoleByName,
		},
		{
			Method: http.MethodPut, Pattern: "/api/roles/{roleId}/users/grant",
			Rule: authz.RuleAdminOrManager, Handler: h.grantUsersToRole,
		},
		{
			Method: http.MethodPut, Pattern: "/api/roles/{roleId}/users/revoke",
			Rule: authz.RuleAdminOrManager, Handler: h.revokeUsersFromRole,
		},
	}
}

// authenticate exchanges credentials for a token.
//
// The body is validated before the credentials are read, as @Valid runs during
// argument resolution: a password that does not meet the format policy is a 400
// rather than a 401, whether or not it is the right one.
func (h *handlers) authenticate(w http.ResponseWriter, r *http.Request) {
	request, ok := decode[UserRequest](w, r)
	if !ok {
		return
	}

	credential, err := h.store.CredentialByEmailAddress(r.Context(), *request.EmailAddress)
	switch {
	case errors.Is(err, ErrUserNotFound):
		// One shape for both refusals, after the same work. The JVM has no
		// choice about the shape either: DaoAuthenticationProvider hides an
		// unknown user behind BadCredentialsException by default, so both
		// answers come from the entry point. The decoy comparison is what stops
		// the duration saying what the response does not — returning in
		// microseconds where a known address costs a bcrypt round enumerates
		// the registered addresses on its own.
		spendAComparison(*request.Password)
		h.refuseCredentials(w, r, *request.EmailAddress)
		return
	case err != nil:
		h.fail(w, r, err)
		return
	}
	if !passwordMatches(credential.PasswordHash, *request.Password) {
		h.refuseCredentials(w, r, *request.EmailAddress)
		return
	}

	token, err := h.minter.Mint(credential)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, AuthResponse{JWTToken: token})
}

// refuseCredentials answers as the authentication entry point does: the header
// and the status, with no body. The email address is logged and never returned,
// so an operator can see the attempt without the response distinguishing an
// unknown address from a wrong password.
func (h *handlers) refuseCredentials(w http.ResponseWriter, r *http.Request, emailAddress string) {
	slog.Warn("a sign-in was refused", "emailAddress", emailAddress, "path", r.URL.Path)
	authhttp.WriteUnauthorized(w, r)
}

// registerUser is the public self-registration. The created row is audited as
// created by the hard-coded requester id rather than by the caller, who has no
// identity yet.
func (h *handlers) registerUser(w http.ResponseWriter, r *http.Request) {
	request, ok := decode[UserRequest](w, r)
	if !ok {
		return
	}
	h.create(w, r, request, registrationRequesterID)
}

// createUser is the same write for an authenticated caller, audited to them.
func (h *handlers) createUser(w http.ResponseWriter, r *http.Request) {
	request, ok := decode[UserRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAuthenticated, authz.Subject{}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}
	h.create(w, r, request, actor)
}

func (h *handlers) create(w http.ResponseWriter, r *http.Request, request UserRequest, actorUserID int64) {
	// The existence check comes first, as UserService.createUser makes it before
	// it encodes. Reversed, every attempt at an address already registered costs
	// a full bcrypt round before anything refuses it — and /auth/user takes no
	// token, so an anonymous caller sets that cost.
	//
	// It narrows the window rather than closing it. The write below re-checks
	// inside its transaction and the unique constraint holds the rule whatever
	// either check saw.
	taken, err := h.store.UserExists(r.Context(), *request.EmailAddress)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	if taken {
		writeConflict(w, "User already exists with email address "+*request.EmailAddress)
		return
	}

	hash, err := hashPassword(*request.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	user, err := h.store.CreateUser(r.Context(), *request.EmailAddress, hash, actorUserID)
	switch {
	case errors.Is(err, ErrUserExists):
		writeConflict(w, "User already exists with email address "+*request.EmailAddress)
	case err != nil:
		h.fail(w, r, err)
	default:
		writeJSON(w, http.StatusCreated, user)
	}
}

func (h *handlers) getAllUsers(w http.ResponseWriter, r *http.Request) {
	page, size, ok := pageBounds(w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdmin, authz.Subject{}) {
		return
	}

	users, err := h.store.ListUsers(r.Context(), size, page*size)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (h *handlers) getUserByID(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: &userID}) {
		return
	}

	user, err := h.store.UserByID(r.Context(), userID)
	if err != nil {
		h.failUser(w, r, err, userID)
		return
	}
	writeJSON(w, http.StatusOK, user)
}

func (h *handlers) getUserByEmailAddress(w http.ResponseWriter, r *http.Request) {
	emailAddress := r.PathValue("emailAddress")
	if !h.authorize(w, r, authz.RuleAdminOrSelfByEmail, authz.Subject{EmailAddress: &emailAddress}) {
		return
	}

	user, err := h.store.UserByEmailAddress(r.Context(), emailAddress)
	switch {
	case errors.Is(err, ErrUserNotFound):
		writeNotFound(w, "User not found with email address "+emailAddress)
	case err != nil:
		h.fail(w, r, err)
	default:
		writeJSON(w, http.StatusOK, user)
	}
}

func (h *handlers) updateUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	request, ok := decode[UserRequest](w, r)
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
	hash, err := hashPassword(*request.Password)
	if err != nil {
		h.fail(w, r, err)
		return
	}

	user, err := h.store.UpdateUser(r.Context(), userID, *request.EmailAddress, hash, actor)
	switch {
	case errors.Is(err, ErrNameTaken):
		// The new address is already held by another user. A unique-constraint
		// violation with no handler mapping it, so it is a 500 rather than the
		// 409 the create would give. Frozen: clients key their messages off the
		// status.
		writeContainerError(w, r, http.StatusInternalServerError)
	case err != nil:
		h.failUser(w, r, err, userID)
	default:
		writeJSON(w, http.StatusOK, user)
	}
}

func (h *handlers) deleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrSelfByUserID, authz.Subject{UserID: &userID}) {
		return
	}
	// No acting user is resolved here, unlike every other write: the delete
	// records nothing, so UserService.deleteUser takes the requester's address
	// only to log it. Looking one up would add a 404 the contract does not list.

	// A no-op for an id that does not exist, and still 204: the repository
	// deletes without reading first, which is why this operation has no 404 in
	// its response set at all.
	if err := h.store.DeleteUser(r.Context(), userID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *handlers) grantRolesToUser(w http.ResponseWriter, r *http.Request) {
	h.changeUserRoles(w, r, h.store.GrantRolesToUser)
}

func (h *handlers) revokeRolesFromUser(w http.ResponseWriter, r *http.Request) {
	h.changeUserRoles(w, r, h.store.RevokeRolesFromUser)
}

// changeUserRoles does what the grant and revoke operations share: the same
// path variable, the same body, the same rule, the same elevated-role guard and
// the same outcomes. Only the write differs.
func (h *handlers) changeUserRoles(
	w http.ResponseWriter, r *http.Request,
	write func(ctx context.Context, userID int64, roleIDs []int64, actorUserID int64) (User, error),
) {
	userID, ok := pathID(w, r, "userId")
	if !ok {
		return
	}
	request, ok := decode[RoleIDList](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrManager, authz.Subject{}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}
	if !h.allowElevatedRoleChange(w, r, request.IDs, actor) {
		return
	}

	user, err := write(r.Context(), userID, request.IDs, actor)
	switch {
	case errors.Is(err, ErrRoleNotFound):
		writeNotFound(w, fmt.Sprintf("Role not found with id %d", missingID(err, 0)))
	case err != nil:
		h.failUser(w, r, err, missingID(err, userID))
	default:
		writeJSON(w, http.StatusOK, user)
	}
}

func (h *handlers) grantUsersToRole(w http.ResponseWriter, r *http.Request) {
	h.changeRoleUsers(w, r, h.store.GrantUsersToRole)
}

func (h *handlers) revokeUsersFromRole(w http.ResponseWriter, r *http.Request) {
	h.changeRoleUsers(w, r, h.store.RevokeUsersFromRole)
}

// changeRoleUsers is the same write from the role side. The elevated-role guard
// applies to the role in the path rather than to a list in the body, which is
// how RoleService calls it.
func (h *handlers) changeRoleUsers(
	w http.ResponseWriter, r *http.Request,
	write func(ctx context.Context, roleID int64, userIDs []int64, actorUserID int64) (Role, error),
) {
	roleID, ok := pathID(w, r, "roleId")
	if !ok {
		return
	}
	request, ok := decode[UserIDList](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdminOrManager, authz.Subject{}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}
	if !h.allowElevatedRoleChange(w, r, []int64{roleID}, actor) {
		return
	}

	role, err := write(r.Context(), roleID, request.IDs, actor)
	switch {
	case errors.Is(err, ErrUserNotFound):
		writeNotFound(w, fmt.Sprintf("User not found with id %d", missingID(err, 0)))
	case err != nil:
		h.failRole(w, r, err, roleID)
	default:
		writeJSON(w, http.StatusOK, role)
	}
}

// allowElevatedRoleChange reproduces the second check the service makes, after
// the operation's own rule has already passed: changing the elevated role
// requires already holding it, so a MANAGER cannot promote anybody to it.
//
// It answers 403 through the same writer the rule uses, because the JVM raises
// AccessDeniedException here and the access-denied handler renders both.
func (h *handlers) allowElevatedRoleChange(
	w http.ResponseWriter, r *http.Request, roleIDs []int64, actorUserID int64,
) bool {
	elevated := false
	for _, roleID := range roleIDs {
		if roleID == elevatedRoleID {
			elevated = true
			break
		}
	}
	if !elevated {
		return true
	}

	// Read from the grant table rather than from the caller's roles claim: the
	// guard is written in terms of an id and the claim carries names.
	holds, err := h.store.UserHasAnyRole(r.Context(), actorUserID, []int64{elevatedRoleID})
	if err != nil {
		// A lookup that failed is an outage, not a statement about the caller's
		// rights; reporting it as a refusal would hide one behind the other.
		h.fail(w, r, err)
		return false
	}
	if !holds {
		authhttp.WriteForbidden(w, r)
		return false
	}
	return true
}

func (h *handlers) getAllRoles(w http.ResponseWriter, r *http.Request) {
	page, size, ok := pageBounds(w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAuthenticated, authz.Subject{}) {
		return
	}

	// Paginated and ordered, where the JVM reads the whole table unordered. The
	// contract records the change: pagination without a total order returns
	// overlapping pages, and the three role-list callers pass no parameters, so
	// they receive the first hundred rows. Declaring the parameters is also what
	// brings the 400 above — the JVM's handler takes no arguments and so never
	// looks at the query string, where this one converts it as the user listing
	// has always converted its own.
	roles, err := h.store.ListRoles(r.Context(), size, page*size)
	if err != nil {
		h.fail(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, roles)
}

func (h *handlers) getRoleByID(w http.ResponseWriter, r *http.Request) {
	roleID, ok := pathID(w, r, "roleId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAuthenticated, authz.Subject{}) {
		return
	}

	role, err := h.store.RoleByID(r.Context(), roleID)
	if err != nil {
		h.failRole(w, r, err, roleID)
		return
	}
	writeJSON(w, http.StatusOK, role)
}

func (h *handlers) getRoleByName(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("roleName")
	if !h.authorize(w, r, authz.RuleAuthenticated, authz.Subject{}) {
		return
	}

	role, err := h.store.RoleByName(r.Context(), name)
	switch {
	case errors.Is(err, ErrRoleNotFound):
		writeNotFound(w, "Role not found with name "+name)
	case err != nil:
		h.fail(w, r, err)
	default:
		writeJSON(w, http.StatusOK, role)
	}
}

func (h *handlers) createRole(w http.ResponseWriter, r *http.Request) {
	request, ok := decode[RoleRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdmin, authz.Subject{}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	role, err := h.store.CreateRole(r.Context(), *request.Name, *request.Description, actor)
	switch {
	case errors.Is(err, ErrRoleExists):
		writeConflict(w, "Role already exists with name "+*request.Name)
	case err != nil:
		h.fail(w, r, err)
	default:
		writeJSON(w, http.StatusCreated, role)
	}
}

func (h *handlers) updateRole(w http.ResponseWriter, r *http.Request) {
	roleID, ok := pathID(w, r, "roleId")
	if !ok {
		return
	}
	request, ok := decode[RoleRequest](w, r)
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdmin, authz.Subject{}) {
		return
	}
	actor, ok := actingUser(w, r)
	if !ok {
		return
	}

	role, err := h.store.UpdateRole(r.Context(), roleID, *request.Name, *request.Description, actor)
	switch {
	case errors.Is(err, ErrNameTaken):
		// The update has no pre-check on the name, unlike the create. Frozen:
		// the asymmetry is real and the status is what clients key on.
		writeContainerError(w, r, http.StatusInternalServerError)
	case err != nil:
		h.failRole(w, r, err, roleID)
	default:
		writeJSON(w, http.StatusOK, role)
	}
}

func (h *handlers) deleteRole(w http.ResponseWriter, r *http.Request) {
	roleID, ok := pathID(w, r, "roleId")
	if !ok {
		return
	}
	if !h.authorize(w, r, authz.RuleAdmin, authz.Subject{}) {
		return
	}
	// As with the user delete: nothing is audited, so nothing is resolved.

	if err := h.store.DeleteRole(r.Context(), roleID); err != nil {
		h.fail(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// pageBounds reads the two paging parameters and clamps them. Out-of-range
// input is clamped rather than rejected: a negative page is served as page 0
// and an enormous size as 500, both with a 200.
func pageBounds(w http.ResponseWriter, r *http.Request) (page, size int, ok bool) {
	page, ok = queryInt(w, r, "page", defaultPage)
	if !ok {
		return 0, 0, false
	}
	size, ok = queryInt(w, r, "size", defaultSize)
	if !ok {
		return 0, 0, false
	}
	return max(page, defaultPage), min(max(size, minimumSize), maximumSize), true
}

func (h *handlers) authorize(w http.ResponseWriter, r *http.Request, rule authz.Rule, subject authz.Subject) bool {
	return authhttp.Authorize(w, r, h.authorizer, rule, subject)
}

// fail answers a storage failure. The cause goes to the log rather than to the
// caller: an error message from a database is a disclosure, not a diagnosis a
// client can act on, and the container's own body names no cause either.
func (h *handlers) fail(w http.ResponseWriter, r *http.Request, err error) {
	slog.Error("the request could not be served", "error", err, "method", r.Method, "path", r.URL.Path)
	writeContainerError(w, r, http.StatusInternalServerError)
}

func (h *handlers) failUser(w http.ResponseWriter, r *http.Request, err error, userID int64) {
	if errors.Is(err, ErrUserNotFound) {
		writeNotFound(w, fmt.Sprintf("User not found with id %d", userID))
		return
	}
	h.fail(w, r, err)
}

func (h *handlers) failRole(w http.ResponseWriter, r *http.Request, err error, roleID int64) {
	if errors.Is(err, ErrRoleNotFound) {
		writeNotFound(w, fmt.Sprintf("Role not found with id %d", roleID))
		return
	}
	h.fail(w, r, err)
}

// actingUser reads the user the audit columns record. The JVM parses the
// Authorization header itself and reads auth.users for the subject it carries;
// here it is the verified user_id claim, so the read disappears from the
// request path. A token carrying no user_id names nobody this service can
// attribute a write to, and answers as the JVM answers a subject with no user
// row.
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
		writeUnconvertableParameter(w, r)
		return 0, false
	}
	return value, true
}

// queryInt reads a numeric query parameter the way Spring converts one, and
// refuses text that is not a number before the handler runs.
//
// The width is the one Spring's int has, not the host's. Parsing at 64 bits
// would accept a page index that survives the clamp — the clamp only raises a
// floor — and then overflow when multiplied by the page size, leaving Postgres
// to refuse a negative OFFSET and the caller to read a 500 for what the contract
// says is a 400.
func queryInt(w http.ResponseWriter, r *http.Request, name string, fallback int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback, true
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil {
		writeUnconvertableParameter(w, r)
		return 0, false
	}
	return int(value), true
}

// decode reads and validates a JSON body, answering as the two handlers that
// cover those failures answer. Both run during Spring's argument resolution,
// ahead of the method interceptor, so a malformed body is 400 even for a caller
// the operation's rule would refuse.
func decode[T interface{ Validate() map[string]string }](w http.ResponseWriter, r *http.Request) (T, bool) {
	var request T
	// Bounded because two of these operations are reachable without a token, and
	// nothing else on the path caps a request body. Every body this service
	// accepts is a handful of fields or a list of ids, so the cap is far above
	// anything a client sends and an oversized one is refused as unreadable —
	// the same status a malformed one gets.
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxRequestBytes))
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
