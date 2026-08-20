package main

import (
	"strings"
	"testing"
)

func TestCIHealthGauges_SetReplacesOnlyThatRepo(t *testing.T) {
	g := newCIHealthGauges()
	g.set("platform", map[string]ciStat{"CI": {FailureRatio: 0.5, ConsecutiveFailures: 1, SampleSize: 2}})
	g.set("apps", map[string]ciStat{"CI": {FailureRatio: 0, ConsecutiveFailures: 0, SampleSize: 5}})

	// Re-set platform with a different workflow set: apps' series must
	// survive untouched, and platform's old "CI" key must be gone, not
	// merged with the new set.
	g.set("platform", map[string]ciStat{"Lint": {FailureRatio: 1, ConsecutiveFailures: 3, SampleSize: 3}})

	var buf strings.Builder
	g.writeTo(&buf)
	out := buf.String()

	if strings.Contains(out, `repo="platform",workflow="CI"`) {
		t.Error("stale platform/CI series survived a re-set")
	}
	if !strings.Contains(out, `repo="platform",workflow="Lint"`) {
		t.Error("new platform/Lint series missing")
	}
	if !strings.Contains(out, `repo="apps",workflow="CI"`) {
		t.Error("unrelated apps/CI series was clobbered by platform's re-set")
	}
}

func TestCIHealthGauges_WriteTo_FormatAndDeterministicOrder(t *testing.T) {
	g := newCIHealthGauges()
	g.set("platform", map[string]ciStat{
		"Zebra": {FailureRatio: 1, ConsecutiveFailures: 5, SampleSize: 5},
		"Alpha": {FailureRatio: 0, ConsecutiveFailures: 0, SampleSize: 5},
	})
	g.refreshesOK.Add(3)
	g.refreshesFailed.Add(1)

	var buf strings.Builder
	g.writeTo(&buf)
	out := buf.String()

	wantLines := []string{
		`# TYPE jdwlabs_repo_ci_main_failure_ratio gauge`,
		`jdwlabs_repo_ci_main_failure_ratio{repo="platform",workflow="Alpha"} 0`,
		`jdwlabs_repo_ci_main_failure_ratio{repo="platform",workflow="Zebra"} 1`,
		`jdwlabs_repo_ci_consecutive_failures{repo="platform",workflow="Zebra"} 5`,
		`jdwlabs_repo_health_exporter_refreshes_total{outcome="ok"} 3`,
		`jdwlabs_repo_health_exporter_refreshes_total{outcome="failed"} 1`,
	}
	for _, want := range wantLines {
		if !strings.Contains(out, want) {
			t.Errorf("output missing line %q\nfull output:\n%s", want, out)
		}
	}

	// Alpha sorts before Zebra: assert relative order, not just presence, so
	// this test would fail if the deliberate sort in writeTo were dropped.
	if strings.Index(out, `workflow="Alpha"`) > strings.Index(out, `workflow="Zebra"`) {
		t.Error("series are not sorted: Alpha should appear before Zebra")
	}
}

func TestCIHealthGauges_EmptyWritesValidHeadersOnly(t *testing.T) {
	g := newCIHealthGauges()
	var buf strings.Builder
	g.writeTo(&buf)
	out := buf.String()
	if !strings.Contains(out, "# TYPE jdwlabs_repo_ci_main_failure_ratio gauge") {
		t.Error("HELP/TYPE headers must be present even with zero series")
	}
	if strings.Contains(out, `repo=`) {
		t.Error("no series expected with an empty gauges set")
	}
}
