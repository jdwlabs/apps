package main

import (
	"fmt"
	"io"
	"sort"
	"sync"
	"sync/atomic"
)

// repoWorkflowKey identifies one (repo, workflow) series. repo carries the
// short name (platform, apps, infrastructure, deployments) to match the
// metric surface in the originating investigation, not the owner/repo form
// the GitHub API takes as input.
type repoWorkflowKey struct {
	repo     string
	workflow string
}

// ciHealthGauges holds the exporter's whole current view of CI health,
// replaced wholesale by each refresh rather than mutated key-by-key — a
// workflow that stops running on main (renamed, deleted, branch protection
// changed) must disappear from /metrics on the next refresh instead of
// serving a last-known value forever, and a wholesale swap is the only way
// that happens without tracking staleness per key.
type ciHealthGauges struct {
	mu    sync.RWMutex
	stats map[repoWorkflowKey]ciStat

	// Exporter's own operational counters, written by hand in the Prometheus
	// text exposition format like ai-sre-relay's — this handful of gauges and
	// counters does not justify a metrics client dependency.
	refreshesOK     atomic.Int64
	refreshesFailed atomic.Int64
	lastRefreshUnix atomic.Int64
}

func newCIHealthGauges() *ciHealthGauges {
	return &ciHealthGauges{stats: map[repoWorkflowKey]ciStat{}}
}

// set replaces the stats for one repo wholesale (every workflow that repo's
// refresh observed this cycle) — the unit a single refreshOne call produces —
// so a repo whose own fetch failed this cycle keeps its prior values instead
// of a partial map wiping series a different repo's success still supports.
func (g *ciHealthGauges) set(repo string, byWorkflow map[string]ciStat) {
	g.mu.Lock()
	defer g.mu.Unlock()
	for k := range g.stats {
		if k.repo == repo {
			delete(g.stats, k)
		}
	}
	for workflow, stat := range byWorkflow {
		g.stats[repoWorkflowKey{repo: repo, workflow: workflow}] = stat
	}
}

func (g *ciHealthGauges) writeTo(w io.Writer) {
	g.mu.RLock()
	keys := make([]repoWorkflowKey, 0, len(g.stats))
	for k := range g.stats {
		keys = append(keys, k)
	}
	stats := g.stats
	g.mu.RUnlock()

	// Sorted so two consecutive scrapes of an unchanged state produce
	// byte-identical output — Prometheus doesn't require this, but a stable
	// diff is worth having when comparing scrapes by hand.
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].repo != keys[j].repo {
			return keys[i].repo < keys[j].repo
		}
		return keys[i].workflow < keys[j].workflow
	})

	fmt.Fprint(w, "# HELP jdwlabs_repo_ci_main_failure_ratio Fraction of the most recent main-branch Actions runs (per workflow) that concluded failure.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_repo_ci_main_failure_ratio gauge\n")
	for _, k := range keys {
		fmt.Fprintf(w, "jdwlabs_repo_ci_main_failure_ratio{repo=%q,workflow=%q} %g\n", k.repo, k.workflow, stats[k].FailureRatio)
	}

	fmt.Fprint(w, "# HELP jdwlabs_repo_ci_consecutive_failures Count of the most recent main-branch Actions runs (per workflow), newest first, that concluded failure before the first non-failure.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_repo_ci_consecutive_failures gauge\n")
	for _, k := range keys {
		fmt.Fprintf(w, "jdwlabs_repo_ci_consecutive_failures{repo=%q,workflow=%q} %d\n", k.repo, k.workflow, stats[k].ConsecutiveFailures)
	}

	fmt.Fprint(w, "# HELP jdwlabs_repo_health_exporter_sample_size Completed main-branch runs the current value for a (repo, workflow) pair was computed from.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_repo_health_exporter_sample_size gauge\n")
	for _, k := range keys {
		fmt.Fprintf(w, "jdwlabs_repo_health_exporter_sample_size{repo=%q,workflow=%q} %d\n", k.repo, k.workflow, stats[k].SampleSize)
	}

	fmt.Fprint(w, "# HELP jdwlabs_repo_health_exporter_refreshes_total GitHub polling cycles, by outcome.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_repo_health_exporter_refreshes_total counter\n")
	fmt.Fprintf(w, "jdwlabs_repo_health_exporter_refreshes_total{outcome=\"ok\"} %d\n", g.refreshesOK.Load())
	fmt.Fprintf(w, "jdwlabs_repo_health_exporter_refreshes_total{outcome=\"failed\"} %d\n", g.refreshesFailed.Load())

	fmt.Fprint(w, "# HELP jdwlabs_repo_health_exporter_last_refresh_timestamp_seconds Unix time of the last refresh cycle that updated at least one repo's stats, regardless of whether every repo in it succeeded.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_repo_health_exporter_last_refresh_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "jdwlabs_repo_health_exporter_last_refresh_timestamp_seconds %d\n", g.lastRefreshUnix.Load())
}
