package main

import "testing"

func TestComputeCIHealth(t *testing.T) {
	tests := []struct {
		name string
		runs []WorkflowRun // newest first, as the API returns them
		want map[string]ciStat
	}{
		{
			name: "empty input yields no series",
			runs: nil,
			want: map[string]ciStat{},
		},
		{
			name: "the motivating case: 13 of 15 failed, most recent 3 failing",
			runs: mkRuns("prd Drift",
				"failure", "failure", "failure", "success", "failure",
				"failure", "failure", "failure", "failure", "failure",
				"failure", "failure", "failure", "failure", "success"),
			want: map[string]ciStat{
				"prd Drift": {FailureRatio: 13.0 / 15.0, ConsecutiveFailures: 3, SampleSize: 15},
			},
		},
		{
			name: "all success: zero ratio, zero streak",
			runs: mkRuns("CI", "success", "success", "success"),
			want: map[string]ciStat{
				"CI": {FailureRatio: 0, ConsecutiveFailures: 0, SampleSize: 3},
			},
		},
		{
			name: "all failure: ratio 1, streak equals sample size",
			runs: mkRuns("CI", "failure", "failure"),
			want: map[string]ciStat{
				"CI": {FailureRatio: 1, ConsecutiveFailures: 2, SampleSize: 2},
			},
		},
		{
			name: "newest run cancelled breaks the streak without counting as a failure in the streak",
			runs: mkRuns("CI", "cancelled", "failure", "failure"),
			want: map[string]ciStat{
				// ratio still counts every non-success-non-cancelled-as-failure
				// conclusion literally: only "failure" counts, so cancelled
				// contributes to neither numerator.
				"CI": {FailureRatio: 2.0 / 3.0, ConsecutiveFailures: 0, SampleSize: 3},
			},
		},
		{
			name: "two workflows on the same repo are independent series",
			runs: append(mkRuns("A", "failure"), mkRuns("B", "success")...),
			want: map[string]ciStat{
				"A": {FailureRatio: 1, ConsecutiveFailures: 1, SampleSize: 1},
				"B": {FailureRatio: 0, ConsecutiveFailures: 0, SampleSize: 1},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := computeCIHealth(tc.runs)
			if len(got) != len(tc.want) {
				t.Fatalf("got %d series, want %d: %+v", len(got), len(tc.want), got)
			}
			for name, want := range tc.want {
				gotStat, ok := got[name]
				if !ok {
					t.Fatalf("missing series %q in %+v", name, got)
				}
				if gotStat != want {
					t.Errorf("series %q = %+v, want %+v", name, gotStat, want)
				}
			}
		})
	}
}

// mkRuns builds WorkflowRun fixtures for one workflow name from a
// newest-first list of conclusions; status is always "completed" since
// ListMainRuns has already filtered to that by the time computeCIHealth sees
// runs, and computeCIHealth does not re-check it.
func mkRuns(name string, conclusions ...string) []WorkflowRun {
	runs := make([]WorkflowRun, len(conclusions))
	for i, c := range conclusions {
		runs[i] = WorkflowRun{Name: name, Status: "completed", Conclusion: c}
	}
	return runs
}
