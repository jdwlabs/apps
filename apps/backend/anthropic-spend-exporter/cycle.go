package main

import "time"

// cycleStartIn is the cycle boundary inside one calendar month. A cycle day
// the month does not have clamps to its last day, so a plan billed on the
// 31st renews on the 30th in April rather than skipping the month.
func cycleStartIn(year int, month time.Month, day int) time.Time {
	lastDay := time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day > lastDay {
		day = lastDay
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC)
}

// cycleWindow returns the billing cycle containing now as [start, end) in
// UTC. The boundary is configuration, not something Anthropic reports, so it
// can sit hours away from when the credit actually refreshes.
func cycleWindow(now time.Time, day int) (start, end time.Time) {
	now = now.UTC()
	year, month, _ := now.Date()
	start = cycleStartIn(year, month, day)
	if now.Before(start) {
		return cycleStartIn(year, month-1, day), start
	}
	return start, cycleStartIn(year, month+1, day)
}
