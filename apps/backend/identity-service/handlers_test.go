package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestThePageBoundsClampRatherThanRefuse(t *testing.T) {
	// The 500-row ceiling is a frozen contract clause, and no request against a
	// live database can prove it: asserting it end to end needs 500 rows in the
	// table before the ceiling has anything to cut. Driven here instead, where
	// removing min(…, maximumSize) fails rather than passes.
	for _, tc := range []struct {
		query string
		page  int
		size  int
	}{
		{query: "", page: defaultPage, size: defaultSize},
		{query: "?size=100000", page: defaultPage, size: maximumSize},
		{query: "?size=501", page: defaultPage, size: maximumSize},
		{query: "?size=500", page: defaultPage, size: maximumSize},
		{query: "?size=0", page: defaultPage, size: minimumSize},
		{query: "?size=-1", page: defaultPage, size: minimumSize},
		{query: "?page=-5", page: defaultPage, size: defaultSize},
		{query: "?page=3&size=25", page: 3, size: 25},
		// Spring hands an empty value to the default rather than to the
		// converter, so it is the absent case rather than a refusal.
		{query: "?page=&size=", page: defaultPage, size: defaultSize},
	} {
		t.Run(tc.query, func(t *testing.T) {
			response := httptest.NewRecorder()

			page, size, ok := pageBounds(response, httptest.NewRequest(http.MethodGet, "/api/users"+tc.query, nil))

			if !ok {
				t.Fatalf("the bounds were refused with %d; out-of-range input is clamped, not rejected",
					response.Code)
			}
			if page != tc.page {
				t.Errorf("page = %d, want %d", page, tc.page)
			}
			if size != tc.size {
				t.Errorf("size = %d, want %d", size, tc.size)
			}
		})
	}
}

func TestAPagingParameterThatIsNotANumberIsRefused(t *testing.T) {
	// Both listings convert `page` and `size` as an int32, so a value that is
	// not one fails before the handler runs. On /api/roles that is part of the
	// behaviour change the contract records: the JVM handler declares no
	// arguments and never looks at the query string, so it serves 200 for the
	// same request. Serving the first hundred rows while ignoring the paging a
	// caller asked for is the worse half of that pair.
	server := parityServer(t, stubStore{})

	for _, surface := range []string{"/api/users", "/api/roles"} {
		for _, query := range []string{"?page=abc", "?size=abc", "?page=1.5", "?size=1e3"} {
			t.Run(surface+query, func(t *testing.T) {
				request := httptest.NewRequest(http.MethodGet, surface+query, nil)
				request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))
				response := httptest.NewRecorder()

				server.ServeHTTP(response, request)

				if response.Code != http.StatusBadRequest {
					t.Errorf("status = %d, want %d (body %q)",
						response.Code, http.StatusBadRequest, response.Body.String())
				}
			})
		}
	}
}

// occupiedStore reports every address as taken, so the registration refuses
// before it reaches the write.
type occupiedStore struct{ stubStore }

func (occupiedStore) UserExists(context.Context, string) (bool, error) { return true, nil }

func TestARegistrationForATakenAddressRefusesBeforeItEncodes(t *testing.T) {
	// /auth/user takes no token, so an anonymous caller sets how often this path
	// runs. Encoding first makes every attempt at an address already registered
	// cost a full bcrypt round before anything refuses it, which is a flood
	// amplifier a public endpoint should not carry. UserService checks first and
	// only then encodes.
	//
	// Asserted by cost rather than by call order, because the cost is the point:
	// the floor is far below one bcrypt round at the cost this service encodes
	// at and far above what a refusal without one takes, and the shortest of
	// several runs keeps a shared runner's noise out of it.
	const bcryptRoundFloor = 5 * time.Millisecond
	server := parityServer(t, occupiedStore{})

	shortest := time.Hour
	for range 3 {
		request := httptest.NewRequest(http.MethodPost, "/auth/user",
			strings.NewReader(`{"emailAddress":"taken@jdw.com","password":"`+fixturePassword+`"}`))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		start := time.Now()
		server.ServeHTTP(response, request)
		if elapsed := time.Since(start); elapsed < shortest {
			shortest = elapsed
		}

		if response.Code != http.StatusConflict {
			t.Fatalf("status = %d, want %d (body %q)",
				response.Code, http.StatusConflict, response.Body.String())
		}
		if got := response.Body.String(); got != "User already exists with email address taken@jdw.com" {
			t.Errorf("body = %q, want the message the JVM composes", got)
		}
	}

	if shortest >= bcryptRoundFloor {
		t.Errorf("a taken address was refused in %s, which is a bcrypt round; "+
			"the encode is running before the check", shortest)
	}
}

// missingUserStore reports the id in the path as absent from both the lookup
// and the write, so the refusal is the same whichever of the two reaches it
// first and only the cost separates them.
type missingUserStore struct{ stubStore }

func (missingUserStore) UserByID(context.Context, int64) (User, error) {
	return User{}, ErrUserNotFound
}

func (missingUserStore) UpdateUser(context.Context, int64, string, string, int64) (User, error) {
	return User{}, ErrUserNotFound
}

func TestAnUpdateToAMissingUserRefusesBeforeItEncodes(t *testing.T) {
	// UserService.updateUser reads the row before it encodes, and the order is
	// the whole difference: encoding first spends a bcrypt round on every update
	// naming an id that is not there, for a 404 that was settled before the
	// request arrived. The rule gates this path, so it is waste rather than the
	// flood /auth/user would carry — but it is the same asymmetry the create
	// already had fixed.
	//
	// Asserted by cost for the same reason as the registration above: the floor
	// sits far below one bcrypt round at the cost this service encodes at and
	// far above a refusal without one.
	const bcryptRoundFloor = 5 * time.Millisecond
	server := parityServer(t, missingUserStore{})
	body := `{"emailAddress":"nobody@jdw.com","password":"` + fixturePassword + `"}`

	shortest := time.Hour
	for range 3 {
		request := httptest.NewRequest(http.MethodPut, "/api/users/987654", strings.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("Authorization", "Bearer "+mint(t, admin().claims))
		response := httptest.NewRecorder()

		start := time.Now()
		server.ServeHTTP(response, request)
		if elapsed := time.Since(start); elapsed < shortest {
			shortest = elapsed
		}

		if response.Code != http.StatusNotFound {
			t.Fatalf("status = %d, want %d (body %q)",
				response.Code, http.StatusNotFound, response.Body.String())
		}
		if got := response.Body.String(); got != "User not found with id 987654" {
			t.Errorf("body = %q, want the message the JVM composes", got)
		}
	}

	if shortest >= bcryptRoundFloor {
		t.Errorf("an update to a missing user was refused in %s, which is a bcrypt round; "+
			"the encode is running before the lookup", shortest)
	}
}
