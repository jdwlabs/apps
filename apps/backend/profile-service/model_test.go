package main

import (
	"encoding/json"
	"testing"
	"time"
)

func ptr[T any](value T) *T { return &value }

func TestProfileMarshalsTheShapeTheFrontendsAlreadyParse(t *testing.T) {
	// The wire shape is pinned by libs/frontend/shared/util's Profile type and
	// the fixtures in profiles.service.spec.ts: birthdate is a plain date,
	// the audit stamps are date-times, and middleName and icon are nullable.
	profile := Profile{
		ID:        2,
		FirstName: "Ada",
		LastName:  "Lovelace",
		Birthdate: Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)},
		UserID:    1,
		Addresses: []Address{},
		Icon: &ProfileIcon{
			ID:               5,
			ProfileID:        2,
			Icon:             []byte{0x89, 0x50, 0x4e, 0x47},
			CreatedByUserID:  1,
			CreatedTime:      Timestamp{time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)},
			ModifiedByUserID: 1,
			ModifiedTime:     Timestamp{time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)},
		},
		CreatedByUserID:  1,
		CreatedTime:      Timestamp{time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)},
		ModifiedByUserID: 1,
		ModifiedTime:     Timestamp{time.Date(2024, 12, 1, 0, 0, 0, 0, time.UTC)},
	}

	encoded, err := json.Marshal(profile)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	want := `{"id":2,"firstName":"Ada","middleName":null,"lastName":"Lovelace",` +
		`"birthdate":"1815-12-10","userId":1,"addresses":[],` +
		`"icon":{"id":5,"profileId":2,"icon":"iVBORw==","createdByUserId":1,` +
		`"createdTime":"2024-12-01T00:00:00.000+00:00","modifiedByUserId":1,` +
		`"modifiedTime":"2024-12-01T00:00:00.000+00:00"},` +
		`"createdByUserId":1,"createdTime":"2024-12-01T00:00:00.000+00:00",` +
		`"modifiedByUserId":1,"modifiedTime":"2024-12-01T00:00:00.000+00:00"}`
	if got := string(encoded); got != want {
		t.Errorf("profile JSON\ngot  %s\nwant %s", got, want)
	}
}

func TestAnEmptyAddressSetMarshalsAsAnArrayRatherThanNull(t *testing.T) {
	// A nil slice would marshal to null, which the frontends' Address[] type
	// cannot iterate.
	encoded, err := json.Marshal(Profile{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	addresses, ok := decoded["addresses"].([]any)
	if !ok {
		t.Fatalf("addresses = %#v, want an array", decoded["addresses"])
	}
	if len(addresses) != 0 {
		t.Errorf("addresses = %v, want empty", addresses)
	}
}

func TestATimestampMarshalsInUtcWhateverZoneItCarries(t *testing.T) {
	// Jackson's default time zone is UTC, so the JVM renders every stamp with a
	// +00:00 offset regardless of the JVM's own zone. A Go service that let the
	// host zone through would change every audit stamp on the wire.
	zone := time.FixedZone("UTC+5", 5*60*60)

	encoded, err := json.Marshal(Timestamp{time.Date(2024, 12, 1, 5, 0, 0, 0, zone)})

	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if got, want := string(encoded), `"2024-12-01T00:00:00.000+00:00"`; got != want {
		t.Errorf("timestamp = %s, want %s", got, want)
	}
}

func TestADateAcceptsBothFormsAClientSends(t *testing.T) {
	// The form control posts a plain date; a JavaScript Date that reached
	// JSON.stringify posts an instant. Jackson's java.sql.Date deserializer
	// reads both, so this has to as well.
	cases := map[string]string{
		`"1990-01-01"`:                "1990-01-01",
		`"1990-01-01T00:00:00.000Z"`:  "1990-01-01",
		`"1990-01-01T23:30:00+00:00"`: "1990-01-01",
	}

	for encoded, want := range cases {
		t.Run(encoded, func(t *testing.T) {
			var date Date

			if err := json.Unmarshal([]byte(encoded), &date); err != nil {
				t.Fatalf("Unmarshal: %v", err)
			}
			if got := date.Format("2006-01-02"); got != want {
				t.Errorf("date = %s, want %s", got, want)
			}
		})
	}
}

func TestADateRejectsTextThatIsNotADate(t *testing.T) {
	var date Date

	err := json.Unmarshal([]byte(`"not-a-date"`), &date)

	if err == nil {
		t.Fatal("Unmarshal accepted text that is not a date")
	}
}

func TestProfileCreateRequestReportsTheValidationMessagesTheJvmReports(t *testing.T) {
	cases := []struct {
		name    string
		request ProfileCreateRequest
		want    map[string]string
	}{
		{
			name: "every field missing",
			want: map[string]string{
				"firstName": "firstName is mandatory",
				"lastName":  "lastName is mandatory",
				"birthdate": "birthdate is mandatory",
				"userId":    "userId is mandatory",
			},
		},
		{
			name: "blank names are as bad as absent ones",
			request: ProfileCreateRequest{
				FirstName: ptr("   "), LastName: ptr(""),
				Birthdate: &Date{time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)}, UserID: ptr(int64(1)),
			},
			want: map[string]string{
				"firstName": "firstName is mandatory",
				"lastName":  "lastName is mandatory",
			},
		},
		{
			name: "a birthdate in the future",
			request: ProfileCreateRequest{
				FirstName: ptr("Ada"), LastName: ptr("Lovelace"),
				Birthdate: &Date{time.Now().AddDate(1, 0, 0)}, UserID: ptr(int64(1)),
			},
			want: map[string]string{"birthdate": "birthdate must be a past date"},
		},
		{
			name: "a complete request",
			request: ProfileCreateRequest{
				FirstName: ptr("Ada"), MiddleName: ptr("Byron"), LastName: ptr("Lovelace"),
				Birthdate: &Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)}, UserID: ptr(int64(1)),
			},
			want: map[string]string{},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.request.Validate()

			if len(got) != len(tc.want) {
				t.Fatalf("errors = %v, want %v", got, tc.want)
			}
			for field, message := range tc.want {
				if got[field] != message {
					t.Errorf("errors[%s] = %q, want %q", field, got[field], message)
				}
			}
		})
	}
}

func TestAddressRequestReportsTheValidationMessagesTheJvmReports(t *testing.T) {
	got := AddressRequest{AddressLine2: ptr("Flat 2")}.Validate()

	want := map[string]string{
		"addressLine1":  "addressLine1 is mandatory",
		"city":          "city is mandatory",
		"stateProvince": "stateProvince is mandatory",
		"postalCode":    "postalCode is mandatory",
		"country":       "country is mandatory",
	}
	if len(got) != len(want) {
		t.Fatalf("errors = %v, want %v", got, want)
	}
	for field, message := range want {
		if got[field] != message {
			t.Errorf("errors[%s] = %q, want %q", field, got[field], message)
		}
	}
}

func TestProfileUpdateRequestDoesNotCarryAUserId(t *testing.T) {
	// A profile cannot be moved between users, so the update body has no userId
	// and an update request that supplies one is not rejected for lacking it.
	got := ProfileUpdateRequest{
		FirstName: ptr("Ada"), LastName: ptr("Lovelace"),
		Birthdate: &Date{time.Date(1815, 12, 10, 0, 0, 0, 0, time.UTC)},
	}.Validate()

	if len(got) != 0 {
		t.Errorf("errors = %v, want none", got)
	}
}
