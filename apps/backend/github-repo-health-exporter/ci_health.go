package main

// This file computes the two CI-health metrics this exporter's first slice
// carries: jdwlabs_repo_ci_main_failure_ratio and
// jdwlabs_repo_ci_consecutive_failures. Both are read off `main`-branch
// Actions run history only — the branch every one of this org's rulesets
// protects and the one a broken run on it (uncaught) has already cost real
// incident time to notice (the "prd Drift" workflow: 13 of 15 daily runs
// failed over two weeks with nobody alerted). The remaining metrics in the
// originating investigation's proposed surface — PR review bypass, ruleset
// drift, missing required checks, open security alerts, release staleness —
// are a separate slice, deferred out of this first cut.

// ciStat is one (repo, workflow) pair's computed CI health.
type ciStat struct {
	FailureRatio        float64
	ConsecutiveFailures int
	SampleSize          int
}

// computeCIHealth groups runs (as returned by GitHubClient.ListMainRuns,
// newest first) by workflow name and reduces each group to a ciStat.
//
// failureRatio is failed/total over every completed run in the window
// (bounded by runsWindow at the fetch site, not re-bounded here) — matching
// how the motivating incident was itself measured ("13 of 15 runs"), not a
// time-windowed rate.
//
// consecutiveFailures counts back from the newest run while conclusion ==
// "failure", stopping at the first run that is not a failure (success,
// cancelled, or any other conclusion). A cancelled run breaks the streak
// deliberately: it says a human or a newer commit interrupted the run, not
// that the pipeline itself is healthy, but counting it as a failure would
// conflate "someone cancelled this" with "this keeps failing" — two findings
// this metric exists to tell apart.
func computeCIHealth(runs []WorkflowRun) map[string]ciStat {
	byWorkflow := make(map[string][]WorkflowRun)
	for _, r := range runs {
		byWorkflow[r.Name] = append(byWorkflow[r.Name], r)
	}

	out := make(map[string]ciStat, len(byWorkflow))
	for name, group := range byWorkflow {
		var failures int
		for _, r := range group {
			if r.Conclusion == "failure" {
				failures++
			}
		}

		streak := 0
		for _, r := range group {
			if r.Conclusion != "failure" {
				break
			}
			streak++
		}

		out[name] = ciStat{
			FailureRatio:        float64(failures) / float64(len(group)),
			ConsecutiveFailures: streak,
			SampleSize:          len(group),
		}
	}
	return out
}
