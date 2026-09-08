package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// timestampLayout is StdDateFormat's ISO-8601 form, which is what Jackson's
// DateSerializer writes for a java.sql.Timestamp once Spring Boot has turned
// WRITE_DATES_AS_TIMESTAMPS off. Milliseconds are always present and the offset
// always carries a colon.
const timestampLayout = "2006-01-02T15:04:05.000-07:00"

// statusActive is the only value Status declares, and what every row this
// service creates carries.
const statusActive = "ACTIVE"

// Timestamp is an audit stamp. It always renders in UTC because Jackson's
// default time zone is UTC, so the JVM's own zone never reaches the wire.
type Timestamp struct{ time.Time }

func (t Timestamp) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.UTC().Format(timestampLayout))
}

func (t *Timestamp) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("a timestamp must be a string: %w", err)
	}
	parsed, err := time.Parse(timestampLayout, text)
	if err != nil {
		parsed, err = time.Parse(time.RFC3339, text)
	}
	if err != nil {
		return fmt.Errorf("%q is not a date-time", text)
	}
	t.Time = parsed
	return nil
}

// User is the identity payload. It carries no password field at all — not an
// ignored one — so no future edit to a struct tag can put the bcrypt hash back
// on the wire, which is how it reached clients before. The hash lives on
// Credential, which nothing marshals.
//
// It carries profileId rather than the embedded profile aggregate: keeping the
// aggregate would make every user read a synchronous call into profile-service.
type User struct {
	ID           int64      `json:"id"`
	EmailAddress string     `json:"emailAddress"`
	Status       string     `json:"status"`
	Roles        []UserRole `json:"roles"`
	// ProfileID is nullable: a user with no profile row has none.
	ProfileID        *int64    `json:"profileId"`
	CreatedByUserID  int64     `json:"createdByUserId"`
	CreatedTime      Timestamp `json:"createdTime"`
	ModifiedByUserID int64     `json:"modifiedByUserId"`
	ModifiedTime     Timestamp `json:"modifiedTime"`
}

// MarshalJSON keeps an unpopulated grant set an empty array. The JVM builds a
// HashSet even for a user with no grants, so null never reaches a client that
// types the field as an array.
func (u User) MarshalJSON() ([]byte, error) {
	type wire User
	copied := wire(u)
	if copied.Roles == nil {
		copied.Roles = []UserRole{}
	}
	return json.Marshal(copied)
}

// UserRole is one row of auth.users_roles, written from both the user and the
// role side.
type UserRole struct {
	UserID          int64     `json:"userId"`
	RoleID          int64     `json:"roleId"`
	CreatedByUserID int64     `json:"createdByUserId"`
	CreatedTime     Timestamp `json:"createdTime"`
}

type Role struct {
	ID          int64      `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	Users       []UserRole `json:"users"`

	CreatedByUserID  int64     `json:"createdByUserId"`
	CreatedTime      Timestamp `json:"createdTime"`
	ModifiedByUserID int64     `json:"modifiedByUserId"`
	ModifiedTime     Timestamp `json:"modifiedTime"`
}

func (r Role) MarshalJSON() ([]byte, error) {
	type wire Role
	copied := wire(r)
	if copied.Users == nil {
		copied.Users = []UserRole{}
	}
	return json.Marshal(copied)
}

// Credential is the row authentication reads: the stored hash together with the
// claims a token is minted from. It exists as its own type, with no JSON tags
// and a redacting String, so the hash has no path to a response body or a log
// line.
type Credential struct {
	UserID       int64
	EmailAddress string
	PasswordHash string
	Roles        []string
	ProfileID    *int64
}

func (c Credential) String() string {
	return fmt.Sprintf("Credential[userId=%d, emailAddress=%s, passwordHash=***]", c.UserID, c.EmailAddress)
}

// UserRequest is the registration, creation and update body. Both fields are
// required on all three. Each is a pointer so an absent field is
// distinguishable from an empty one, which is what lets the validation below
// reproduce @NotNull rather than guess at it.
type UserRequest struct {
	EmailAddress *string `json:"emailAddress"`
	Password     *string `json:"password"`
}

// String redacts the password. The JVM record overrides toString for the same
// reason: a cleartext password reached the request log through a formatted DTO
// once already, and a struct printed with %v or logged as a value would put it
// back.
func (r UserRequest) String() string {
	email := ""
	if r.EmailAddress != nil {
		email = *r.EmailAddress
	}
	return "UserRequest[emailAddress=" + email + ", password=***]"
}

// emailPattern is the @Email regexp the DTO declares, which is looser than
// RFC 5322. Transcribed rather than idealised: a Go service that validated more
// strictly would reject addresses the JVM accepts, and existing users could not
// sign in.
var emailPattern = regexp.MustCompile(`^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$`)

// passwordSymbols is the character class the @Pattern constraint allows outside
// letters and digits, and the set at least one character must come from.
const passwordSymbols = "!@#$%^&*()_+-=[]{};':\"\\|,.<>/?~`"

// passwordRequirements is the constraint's own message, kept exactly, because a
// client renders it to whoever typed the password.
const passwordRequirements = "Password must be at least 8 characters long, contain at least one uppercase " +
	"letter, one lowercase letter, one number, and one special character. Special characters include: " +
	"! @ # $ % ^ & * ( ) _ + - = [ ] { } ; ' : \" \\ | , . < > / ? ~ `"

const minimumPasswordLength = 8

func (r UserRequest) Validate() map[string]string {
	errors := map[string]string{}
	switch {
	case isBlank(r.EmailAddress):
		// @NotNull and @NotBlank carry the same message on this field, and the
		// @Email constraint would fire on a whitespace-only value as well. One
		// entry rather than a race between three.
		errors["emailAddress"] = "emailAddress is mandatory"
	case !emailPattern.MatchString(*r.EmailAddress):
		errors["emailAddress"] = "emailAddress is not valid"
	}
	switch {
	case isBlank(r.Password):
		errors["password"] = "password is mandatory"
	case !passwordMeetsPolicy(*r.Password):
		errors["password"] = passwordRequirements
	}
	return errors
}

// passwordMeetsPolicy decides the @Pattern constraint. It is written out rather
// than compiled because the constraint is four lookaheads over one character
// class, and Go's RE2 has no lookahead at all — a regexp here would either be
// wrong or would silently accept everything.
func passwordMeetsPolicy(password string) bool {
	if len(password) < minimumPasswordLength {
		return false
	}
	upper, lower, digit, symbol := false, false, false, false
	for _, character := range password {
		switch {
		case character >= 'A' && character <= 'Z':
			upper = true
		case character >= 'a' && character <= 'z':
			lower = true
		case character >= '0' && character <= '9':
			digit = true
		case strings.ContainsRune(passwordSymbols, character):
			symbol = true
		default:
			// The class is closed: anything outside it fails the whole match,
			// so a space or an accented letter is refused as the JVM refuses it.
			return false
		}
	}
	return upper && lower && digit && symbol
}

// RoleRequest is the create and update body for a role.
type RoleRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

func (r RoleRequest) Validate() map[string]string {
	errors := map[string]string{}
	if isBlank(r.Name) {
		errors["name"] = "name is mandatory"
	}
	if isBlank(r.Description) {
		errors["description"] = "description is mandatory"
	}
	return errors
}

// RoleIDList is the body of the two user-side grant and revoke operations: a
// bare JSON array, wrapped so it can carry the same Validate method every other
// body has.
type RoleIDList struct{ IDs []int64 }

func (l *RoleIDList) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &l.IDs) }

// Validate reproduces the @NotEmpty the controllers declare on these bodies.
// The contract gives both list schemas minItems: 1 and lists 400 in the
// response set, so an empty array is refused here rather than reaching the
// service, where the JVM reads the first element of the list it built and
// answers 500 for an empty one.
func (l RoleIDList) Validate() map[string]string {
	if len(l.IDs) == 0 {
		return map[string]string{"roleIds": "must not be empty"}
	}
	return nil
}

// UserIDList is the same body from the role side, named for the parameter that
// controller declares so a client sees the field it sent.
type UserIDList struct{ IDs []int64 }

func (l *UserIDList) UnmarshalJSON(data []byte) error { return json.Unmarshal(data, &l.IDs) }

func (l UserIDList) Validate() map[string]string {
	if len(l.IDs) == 0 {
		return map[string]string{"userIds": "must not be empty"}
	}
	return nil
}

// AuthResponse is the only body /auth/authenticate answers with.
type AuthResponse struct {
	JWTToken string `json:"jwtToken"`
}

// isBlank folds @NotNull and @NotBlank together. Both carry the same message on
// every field that declares them, so a null and a whitespace-only value produce
// one entry rather than two.
func isBlank(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}
