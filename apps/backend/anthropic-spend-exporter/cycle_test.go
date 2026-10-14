package main

import (
	"testing"
	"time"
)

func TestCycleWindow(t *testing.T) {
	utc := func(s string) time.Time {
		t.Helper()
		v, err := time.Parse(time.RFC3339, s)
		if err != nil {
			t.Fatalf("bad test time %q: %v", s, err)
		}
		return v
	}
	cases := []struct {
		name      string
		now       string
		day       int
		wantStart string
		wantEnd   string
	}{
		{"mid cycle", "2026-10-13T12:00:00Z", 15, "2026-09-15T00:00:00Z", "2026-10-15T00:00:00Z"},
		{"first instant of the cycle day", "2026-10-15T00:00:00Z", 15, "2026-10-15T00:00:00Z", "2026-11-15T00:00:00Z"},
		{"last instant before the cycle day", "2026-10-14T23:59:59Z", 15, "2026-09-15T00:00:00Z", "2026-10-15T00:00:00Z"},
		{"day 31 clamps to the 30th in April", "2026-04-30T00:00:00Z", 31, "2026-04-30T00:00:00Z", "2026-05-31T00:00:00Z"},
		{"day 31 before the clamped April boundary", "2026-04-29T10:00:00Z", 31, "2026-03-31T00:00:00Z", "2026-04-30T00:00:00Z"},
		{"day 31 in a non-leap February", "2026-02-10T00:00:00Z", 31, "2026-01-31T00:00:00Z", "2026-02-28T00:00:00Z"},
		{"day 30 in a leap February", "2028-02-29T05:00:00Z", 30, "2028-02-29T00:00:00Z", "2028-03-30T00:00:00Z"},
		{"cycle spanning new year, in December", "2026-12-25T00:00:00Z", 20, "2026-12-20T00:00:00Z", "2027-01-20T00:00:00Z"},
		{"cycle spanning new year, in January", "2027-01-05T00:00:00Z", 20, "2026-12-20T00:00:00Z", "2027-01-20T00:00:00Z"},
		{"day 1", "2026-10-13T00:00:00Z", 1, "2026-10-01T00:00:00Z", "2026-11-01T00:00:00Z"},
		// 23:30 on the 14th at UTC-5 is already the 15th in UTC.
		{"non-UTC input is converted before comparing", "2026-10-14T23:30:00-05:00", 15, "2026-10-15T00:00:00Z", "2026-11-15T00:00:00Z"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			start, end := cycleWindow(utc(tc.now), tc.day)
			if !start.Equal(utc(tc.wantStart)) || !end.Equal(utc(tc.wantEnd)) {
				t.Errorf("cycleWindow(%s, %d) = [%s, %s), want [%s, %s)",
					tc.now, tc.day, start.Format(time.RFC3339), end.Format(time.RFC3339), tc.wantStart, tc.wantEnd)
			}
		})
	}
}
