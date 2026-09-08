package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
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
