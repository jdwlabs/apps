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
