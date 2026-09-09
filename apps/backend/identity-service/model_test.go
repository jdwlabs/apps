package main

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestNoTypeThisServiceSerializesCarriesAPasswordField(t *testing.T) {
	// The defect this is here to prevent came back once already: every user
	// response serialized the bcrypt hash, the public registration one
	// included. Asserted over the types themselves rather than over one
	// response, so a field added tomorrow fails before it is ever served.
	for _, wire := range []any{User{}, UserRole{}, Role{}, AuthResponse{}} {
		t.Run(reflect.TypeOf(wire).Name(), func(t *testing.T) {
			for _, field := range reflect.VisibleFields(reflect.TypeOf(wire)) {
				name := strings.ToLower(field.Name + " " + field.Tag.Get("json"))
				for _, banned := range []string{"password", "secret", "credential", "hash"} {
					if strings.Contains(name, banned) {
						t.Errorf("%s carries %s, which must never reach a response", field.Name, banned)
					}
				}
			}
		})
	}
}

func TestTheTypesThatHoldACredentialRedactItWhenFormatted(t *testing.T) {
	// The other half of the same regression: the hash stayed out of the JSON and
	// went into a log line instead, through a value formatted with its default
	// representation.
	credential := Credential{UserID: 1, EmailAddress: "self@jdw.com", PasswordHash: "$2a$10$notarealhash"}
	request := UserRequest{EmailAddress: ptr("self@jdw.com"), Password: ptr("Password1!")}

	for _, formatted := range []string{
		credential.String(),
		request.String(),
	} {
		if strings.Contains(formatted, "notarealhash") || strings.Contains(formatted, "Password1!") {
			t.Errorf("a formatted value disclosed the secret: %s", formatted)
		}
		if !strings.Contains(formatted, "***") {
			t.Errorf("a formatted value did not mark the redaction: %s", formatted)
		}
	}
}

func TestTheJsonLogHandlerThisServiceInstallsAlsoRedactsThem(t *testing.T) {
	// String is not enough on its own. slog's JSON handler marshals an Any value
	// with encoding/json and never consults Stringer, so a debug line written as
	// slog.Any("request", request) would print the cleartext under the field name
	// the struct tag supplies. Driven through the same handler main installs,
	// because that is the one whose behaviour matters.
	var written bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&written, nil))

	logger.Info("a line somebody adds later",
		"credential", Credential{UserID: 1, EmailAddress: "self@jdw.com", PasswordHash: "$2a$10$notarealhash"},
		"request", UserRequest{EmailAddress: ptr("self@jdw.com"), Password: ptr("Password1!")})

	line := written.String()
	for _, secret := range []string{"notarealhash", "Password1!"} {
		if strings.Contains(line, secret) {
			t.Errorf("the log line disclosed %q: %s", secret, line)
		}
	}
	if !strings.Contains(line, "***") {
		t.Errorf("the log line did not mark the redaction: %s", line)
	}
	if !strings.Contains(line, "self@jdw.com") {
		t.Errorf("the log line dropped the field worth logging: %s", line)
	}
}

func TestAUserWithNoGrantsSerializesAnEmptyArray(t *testing.T) {
	// The JVM builds a HashSet even for a user with no grants, so a client that
	// types the field as an array never receives null.
	encoded, err := json.Marshal(User{ID: 1, EmailAddress: "self@jdw.com"})

	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	roles, ok := decoded["roles"].([]any)
	if !ok || len(roles) != 0 {
		t.Errorf("roles = %v, want an empty array", decoded["roles"])
	}
	if _, present := decoded["profileId"]; !present {
		t.Error("profileId is absent; the contract types it as nullable, not optional")
	}
	if decoded["profileId"] != nil {
		t.Errorf("profileId = %v, want null for a user with no profile", decoded["profileId"])
	}
}

func TestARoleWithNoGrantsSerializesAnEmptyArray(t *testing.T) {
	encoded, err := json.Marshal(Role{ID: 1, Name: "ADMIN"})

	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	users, ok := decoded["users"].([]any)
	if !ok || len(users) != 0 {
		t.Errorf("users = %v, want an empty array", decoded["users"])
	}
}

func TestATimestampRendersInTheShapeJacksonWrites(t *testing.T) {
	// Milliseconds always present, the zero offset written "Z", and UTC whatever
	// the host's zone: Jackson's default time zone is UTC, so the JVM's own zone
	// never reaches the wire. Measured off a booted usersrole with its default
	// zone forced to Asia/Kolkata, which changed neither the offset nor the
	// instant.
	zone := time.FixedZone("UTC+5", 5*60*60)
	stamp := Timestamp{time.Date(2026, 9, 8, 9, 30, 0, 123_000_000, zone)}

	encoded, err := json.Marshal(stamp)

	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(encoded), `"2026-09-08T04:30:00.123Z"`; got != want {
		t.Errorf("timestamp = %s, want %s", got, want)
	}
}

func TestTheEmailConstraintIsTheOneTheDtoDeclares(t *testing.T) {
	// @Email is one constraint with two halves: EmailValidator runs Hibernate's
	// shared address checks and applies the DTO's regexp only to an address that
	// already passed them. Transcribing the regexp alone was looser than the JVM,
	// which is the direction that matters — it would let this service create rows
	// the system it replaces refuses.
	//
	// Every expectation below is measured against hibernate-validator 9.1.3, the
	// version Boot 4.1.1 resolves, driving the DTO's own annotation.
	accepted := []string{
		"a@b.co", "first.last@example.com", "a-b@sub.domain.example", "_x@y.info",
		// A hyphen bounds a domain label and not a local-part atom, and an
		// underscore is a domain character.
		"a-@b.co", "-a@b.co", "a@_b.co", "a@ex--ample.com",
		"A@B.CO",
		strings.Repeat("a", 64) + "@example.com",
		"a@" + strings.Repeat("b", 63) + ".com",
	}
	refused := []string{
		"nobody", "a@b", "a@b.c", "a b@c.com", "@b.com", "a@.com",
		// The four the regexp alone accepted.
		"a..b@example.com", strings.Repeat("a", 65) + "@example.com",
		"a@-example.com", "a@b..c.com",
		".a@example.com", "a.@example.com", "a@example-.com",
		"a@" + strings.Repeat("b", 64) + ".com",
		"a@" + strings.Repeat(strings.Repeat("x", 63)+".", 4) + "com",
	}

	for _, address := range accepted {
		if got := (UserRequest{EmailAddress: &address, Password: ptr("Password1!")}).Validate(); got["emailAddress"] != "" {
			t.Errorf("%q was refused: %s", address, got["emailAddress"])
		}
	}
	for _, address := range refused {
		if got := (UserRequest{EmailAddress: &address, Password: ptr("Password1!")}).Validate(); got["emailAddress"] == "" {
			t.Errorf("%q was accepted", address)
		}
	}
}

func TestThePasswordPolicyIsTheOneTheDtoDeclares(t *testing.T) {
	// Written out rather than compiled: the constraint is four lookaheads over
	// one character class, and Go's regexp engine has no lookahead at all.
	cases := map[string]bool{
		"Password1!":  true,
		"aB3$aB3$":    true,
		"Aa1~....":    true,
		"Passwor1!":   true,
		"password1!":  false, // no upper case
		"PASSWORD1!":  false, // no lower case
		"Password!!":  false, // no digit
		"Password11":  false, // no symbol
		"Pa1!":        false, // too short
		"Password 1!": false, // a space is outside the class
		"Pässword1!":  false, // an accented letter is outside the class
	}

	for password, want := range cases {
		t.Run(password, func(t *testing.T) {
			got := (UserRequest{EmailAddress: ptr("a@b.co"), Password: &password}).Validate()

			if accepted := got["password"] == ""; accepted != want {
				t.Errorf("accepted = %v, want %v (message %q)", accepted, want, got["password"])
			}
			if !want && got["password"] != passwordRequirements {
				t.Errorf("message = %q, want the constraint's own text", got["password"])
			}
		})
	}
}

func TestAnAbsentFieldAndABlankOneCarryTheSameMandatoryMessage(t *testing.T) {
	// @NotNull and @NotBlank declare the same message on every field that has
	// both, so a null and a whitespace-only value produce one entry rather than
	// two racing to overwrite each other in the map.
	for _, request := range []UserRequest{
		{},
		{EmailAddress: ptr("   "), Password: ptr("  ")},
	} {
		got := request.Validate()

		if got["emailAddress"] != "emailAddress is mandatory" {
			t.Errorf("emailAddress = %q, want the mandatory message", got["emailAddress"])
		}
		if got["password"] != "password is mandatory" {
			t.Errorf("password = %q, want the mandatory message", got["password"])
		}
	}
}

func TestARoleRequestNeedsBothFields(t *testing.T) {
	got := RoleRequest{}.Validate()

	if got["name"] != "name is mandatory" || got["description"] != "description is mandatory" {
		t.Errorf("errors = %v, want both fields mandatory", got)
	}
	if valid := (RoleRequest{Name: ptr("AUDITOR"), Description: ptr("Reads only.")}).Validate(); len(valid) != 0 {
		t.Errorf("errors = %v, want none", valid)
	}
}

func TestAGrantListRefusesAnEmptyArrayAndAcceptsAPopulatedOne(t *testing.T) {
	var roles RoleIDList
	if err := json.Unmarshal([]byte(`[1,2]`), &roles); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if len(roles.Validate()) != 0 {
		t.Errorf("errors = %v, want none", roles.Validate())
	}

	var empty UserIDList
	if err := json.Unmarshal([]byte(`[]`), &empty); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if empty.Validate()["userIds"] == "" {
		t.Errorf("errors = %v, want an entry for userIds", empty.Validate())
	}
}
