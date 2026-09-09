package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"time"
)

// timestampLayout is the ISO-8601 form Jackson's DateSerializer writes for a
// java.sql.Timestamp once Spring Boot has turned WRITE_DATES_AS_TIMESTAMPS off.
// Milliseconds are always present, and the zero offset is written "Z" rather
// than "+00:00": that is Jackson 3's output, which Boot 4.1 resolves, where
// Jackson 2 wrote the numeric form. Measured off a booted usersrole with the
// JVM's default zone forced away from UTC, since Jackson serializes in UTC
// regardless of it.
const timestampLayout = "2006-01-02T15:04:05.000Z07:00"

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
// claims a token is minted from. It is its own type, separate from User, so the
// hash has one call site rather than riding along on the payload every read
// returns.
//
// The two hooks below are what keep it out of a log line. Nothing in the type
// keeps it out of a response body: the fields are exported, and an untagged
// exported field marshals under its Go name, so writeJSON on a Credential would
// send PasswordHash. What holds that line is that no handler marshals this type,
// asserted by driving every operation and reading the responses.
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

// LogValue is what actually holds the redaction. The service installs a JSON
// slog handler, and that handler marshals an Any value with encoding/json — it
// never consults Stringer, so String alone would let slog.Any("credential", c)
// write the hash out in full. Every handler resolves a LogValuer.
func (c Credential) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Int64("userId", c.UserID),
		slog.String("emailAddress", c.EmailAddress),
		slog.String("passwordHash", "***"),
	)
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
	return "UserRequest[emailAddress=" + r.emailAddress() + ", password=***]"
}

// LogValue redacts for the JSON handler, which reaches for encoding/json rather
// than Stringer and would otherwise write the cleartext password out under the
// field name this type's own json tag supplies.
func (r UserRequest) LogValue() slog.Value {
	return slog.GroupValue(
		slog.String("emailAddress", r.emailAddress()),
		slog.String("password", "***"),
	)
}

func (r UserRequest) emailAddress() string {
	if r.EmailAddress == nil {
		return ""
	}
	return *r.EmailAddress
}

// emailPattern is the regexp the DTO's @Email declares. It is half the
// constraint, not the whole of it: Hibernate Validator's EmailValidator runs the
// shared address checks below first and applies this pattern only to an address
// that already passed them. Matching the pattern alone accepted a..b@x.co, a
// 65-character local part, x@-b.co and x@b..c.co — all of which the JVM answers
// 400 for, on all four operations that take a UserRequest.
var emailPattern = regexp.MustCompile(`^[\w.-]+@[\w.-]+\.[a-zA-Z]{2,}$`)

// The bounds AbstractEmailValidator and DomainNameUtil apply. The domain label
// bound is java.net.IDN's rather than either pattern's: DomainNameUtil converts
// the domain before matching it, and IDN refuses a label outside 1 to 63
// characters.
const (
	maximumLocalPartLength   = 64
	maximumDomainLength      = 255
	maximumDomainLabelLength = 63
)

// emailIsValid decides @Email the way the JVM decides it: the shared address
// checks, then the declared pattern.
//
// The shared checks are transcribed only as far as the pattern leaves room for.
// An address that matches it is ASCII, carries exactly one @ and holds nothing
// outside [A-Za-z0-9_.-], so the quoted local part, the bracketed IP-literal
// domain and the non-ASCII ranges Hibernate's own patterns also admit are all
// unreachable here and are not written out. What survives is the length bounds
// and the shape of the dot-separated parts.
func emailIsValid(address string) bool {
	if !emailPattern.MatchString(address) {
		return false
	}
	at := strings.LastIndex(address, "@")
	return localPartIsValid(address[:at]) && domainIsValid(address[at+1:])
}

// localPartIsValid applies LOCAL_PART_PATTERN, which is one or more
// dot-separated atoms. Every character the pattern above admits is an atom
// character, so the only way to fail it is to leave an atom empty: a leading
// dot, a trailing dot or a doubled one. A hyphen at either end is fine, which is
// where this differs from the domain rule below.
func localPartIsValid(localPart string) bool {
	if len(localPart) > maximumLocalPartLength {
		return false
	}
	for atom := range strings.SplitSeq(localPart, ".") {
		if atom == "" {
			return false
		}
	}
	return true
}

// domainIsValid applies DOMAIN_LABEL: a label is a run of domain characters that
// may hold hyphens inside it but may not begin or end with one, and no label may
// be empty.
func domainIsValid(domain string) bool {
	if len(domain) > maximumDomainLength {
		return false
	}
	for label := range strings.SplitSeq(domain, ".") {
		if label == "" || len(label) > maximumDomainLabelLength {
			return false
		}
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}
	return true
}

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
	case !emailIsValid(*r.EmailAddress):
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
// The contract gives both list schemas minItems: 1 and lists 400 in the response
// set, and 400 is what the JVM answers: a constraint annotation sitting directly
// on a controller parameter triggers built-in method validation, which raises
// HandlerMethodValidationException before the handler body runs. The empty list
// never reaches the service.
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
