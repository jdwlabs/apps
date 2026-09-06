package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// dateLayout is how Jackson renders java.sql.Date once Spring Boot has turned
// WRITE_DATES_AS_TIMESTAMPS off: SqlDateSerializer falls through to
// Date.toString(), which is the ISO calendar date alone.
const dateLayout = "2006-01-02"

// timestampLayout is StdDateFormat's ISO-8601 form, which is what Jackson's
// DateSerializer writes for a java.sql.Timestamp under the same setting.
// Milliseconds are always present and the offset always carries a colon.
const timestampLayout = "2006-01-02T15:04:05.000-07:00"

// Date is a calendar date on the wire and in Postgres, with no zone of its own.
type Date struct{ time.Time }

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.Format(dateLayout))
}

// UnmarshalJSON accepts both forms a client sends, as Jackson's java.sql.Date
// deserializer does: a plain calendar date, and an instant that a JavaScript
// Date produced on its way through JSON.stringify.
func (d *Date) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return fmt.Errorf("a date must be a string: %w", err)
	}
	if parsed, err := time.Parse(dateLayout, text); err == nil {
		d.Time = parsed
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		return fmt.Errorf("%q is neither a date nor a date-time", text)
	}
	// The instant's own zone decides which calendar day it names, matching the
	// JVM, which converts to a millisecond value before truncating.
	d.Time = time.Date(parsed.Year(), parsed.Month(), parsed.Day(), 0, 0, 0, 0, time.UTC)
	return nil
}

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

// Profile is the aggregate every operation but the icon download and the
// deletes answers with.
type Profile struct {
	ID               int64        `json:"id"`
	FirstName        string       `json:"firstName"`
	MiddleName       *string      `json:"middleName"`
	LastName         string       `json:"lastName"`
	Birthdate        Date         `json:"birthdate"`
	UserID           int64        `json:"userId"`
	Addresses        []Address    `json:"addresses"`
	Icon             *ProfileIcon `json:"icon"`
	CreatedByUserID  int64        `json:"createdByUserId"`
	CreatedTime      Timestamp    `json:"createdTime"`
	ModifiedByUserID int64        `json:"modifiedByUserId"`
	ModifiedTime     Timestamp    `json:"modifiedTime"`
}

// MarshalJSON keeps an unpopulated address set an empty array. The JVM builds a
// HashSet even for a profile with no addresses, so null never reaches a client
// that types the field as an array.
func (p Profile) MarshalJSON() ([]byte, error) {
	type wire Profile
	copied := wire(p)
	if copied.Addresses == nil {
		copied.Addresses = []Address{}
	}
	return json.Marshal(copied)
}

type Address struct {
	ID               int64     `json:"id"`
	AddressLine1     string    `json:"addressLine1"`
	AddressLine2     *string   `json:"addressLine2"`
	City             string    `json:"city"`
	StateProvince    string    `json:"stateProvince"`
	PostalCode       string    `json:"postalCode"`
	Country          string    `json:"country"`
	ProfileID        int64     `json:"profileId"`
	CreatedByUserID  int64     `json:"createdByUserId"`
	CreatedTime      Timestamp `json:"createdTime"`
	ModifiedByUserID int64     `json:"modifiedByUserId"`
	ModifiedTime     Timestamp `json:"modifiedTime"`
}

// ProfileIcon carries the stored bytes as well as the metadata. ID is a
// surrogate no route accepts and no lookup uses; the icon's identifier is
// ProfileID.
type ProfileIcon struct {
	ID               int64     `json:"id"`
	ProfileID        int64     `json:"profileId"`
	Icon             []byte    `json:"icon"`
	CreatedByUserID  int64     `json:"createdByUserId"`
	CreatedTime      Timestamp `json:"createdTime"`
	ModifiedByUserID int64     `json:"modifiedByUserId"`
	ModifiedTime     Timestamp `json:"modifiedTime"`
}

// ProfileCreateRequest is the create body. Every field is a pointer so that an
// absent field is distinguishable from a zero one, which is what lets the
// validation below reproduce @NotNull rather than guess at it.
type ProfileCreateRequest struct {
	FirstName  *string `json:"firstName"`
	MiddleName *string `json:"middleName"`
	LastName   *string `json:"lastName"`
	Birthdate  *Date   `json:"birthdate"`
	UserID     *int64  `json:"userId"`
}

func (r ProfileCreateRequest) Validate() map[string]string {
	errors := validateNames(r.FirstName, r.LastName, r.Birthdate)
	if r.UserID == nil {
		errors["userId"] = "userId is mandatory"
	}
	return errors
}

// ProfileUpdateRequest is the create body without userId: a profile cannot be
// moved between users.
type ProfileUpdateRequest struct {
	FirstName  *string `json:"firstName"`
	MiddleName *string `json:"middleName"`
	LastName   *string `json:"lastName"`
	Birthdate  *Date   `json:"birthdate"`
}

func (r ProfileUpdateRequest) Validate() map[string]string {
	return validateNames(r.FirstName, r.LastName, r.Birthdate)
}

// validateNames applies the three constraints both profile bodies share. The
// messages are the ones the DTO records declare, because a client that shows
// them to a user would otherwise start showing different text at cutover.
func validateNames(firstName, lastName *string, birthdate *Date) map[string]string {
	errors := map[string]string{}
	if isBlank(firstName) {
		errors["firstName"] = "firstName is mandatory"
	}
	if isBlank(lastName) {
		errors["lastName"] = "lastName is mandatory"
	}
	switch {
	case birthdate == nil:
		errors["birthdate"] = "birthdate is mandatory"
	case !birthdate.Before(time.Now()):
		errors["birthdate"] = "birthdate must be a past date"
	}
	return errors
}

type AddressRequest struct {
	AddressLine1  *string `json:"addressLine1"`
	AddressLine2  *string `json:"addressLine2"`
	City          *string `json:"city"`
	StateProvince *string `json:"stateProvince"`
	PostalCode    *string `json:"postalCode"`
	Country       *string `json:"country"`
}

func (r AddressRequest) Validate() map[string]string {
	errors := map[string]string{}
	for _, field := range []struct {
		name  string
		value *string
	}{
		{"addressLine1", r.AddressLine1},
		{"city", r.City},
		{"stateProvince", r.StateProvince},
		{"postalCode", r.PostalCode},
		{"country", r.Country},
	} {
		if isBlank(field.value) {
			errors[field.name] = field.name + " is mandatory"
		}
	}
	return errors
}

// isBlank folds @NotNull and @NotBlank together. Both carry the same message on
// every field that declares them, so a null and a whitespace-only value produce
// one entry rather than two.
func isBlank(value *string) bool {
	return value == nil || strings.TrimSpace(*value) == ""
}
