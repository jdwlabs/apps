package main

import (
	"strings"
	"testing"
	"time"
)

var (
	viewStart = time.Date(2026, 10, 15, 0, 0, 0, 0, time.UTC)
	viewEnd   = time.Date(2026, 11, 15, 0, 0, 0, 0, time.UTC)
)

func render(v *spendView) string {
	var b strings.Builder
	v.writeTo(&b)
	return b.String()
}

func TestSpendView_WriteTo_FormatAndDeterministicOrder(t *testing.T) {
	v := newSpendView(100)
	v.replace(map[costSeries]float64{
		{"sre", "claude-haiku-5-5", "standard", "tokens"}:     0.5,
		{"agents", "claude-sonnet-5-5", "standard", "tokens"}: 12.34,
		{"agents", "claude-sonnet-5-5", "batch", "tokens"}:    3,
		{"default", "", "", "web_search"}:                     0.01,
	}, viewStart, viewEnd)
	v.refreshesOK.Add(3)
	v.refreshesFailed.Add(1)
	v.lastRefreshUnix.Store(1791000000)

	out := render(v)
	wantLines := []string{
		`# TYPE jdwlabs_anthropic_cycle_cost_usd gauge`,
		`jdwlabs_anthropic_cycle_cost_usd{workspace="agents",model="claude-sonnet-5-5",service_tier="batch",cost_type="tokens"} 3`,
		`jdwlabs_anthropic_cycle_cost_usd{workspace="agents",model="claude-sonnet-5-5",service_tier="standard",cost_type="tokens"} 12.34`,
		`jdwlabs_anthropic_cycle_cost_usd{workspace="default",model="",service_tier="",cost_type="web_search"} 0.01`,
		`jdwlabs_anthropic_cycle_cost_usd{workspace="sre",model="claude-haiku-5-5",service_tier="standard",cost_type="tokens"} 0.5`,
		`jdwlabs_anthropic_credit_budget_usd 100`,
		`jdwlabs_anthropic_cycle_start_timestamp_seconds 1792022400`,
		`jdwlabs_anthropic_cycle_end_timestamp_seconds 1794700800`,
		`jdwlabs_anthropic_spend_exporter_refreshes_total{outcome="ok"} 3`,
		`jdwlabs_anthropic_spend_exporter_refreshes_total{outcome="failed"} 1`,
		`jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds 1791000000`,
	}
	last := -1
	for _, want := range wantLines {
		at := strings.Index(out, want+"\n")
		if at < 0 {
			t.Fatalf("output missing line %q\nfull output:\n%s", want, out)
		}
		// wantLines is in the order the exposition must use, so this also
		// pins the sort: agents before default before sre, batch before
		// standard.
		if at < last {
			t.Errorf("line %q is out of order\nfull output:\n%s", want, out)
		}
		last = at
	}
	if again := render(v); again != out {
		t.Error("two renders of an unchanged view differ")
	}
}

func TestSpendView_ReplaceDropsSeriesThatLeftTheReport(t *testing.T) {
	v := newSpendView(100)
	v.replace(map[costSeries]float64{{"old", "m", "standard", "tokens"}: 5}, viewStart, viewEnd)
	v.replace(map[costSeries]float64{{"new", "m", "standard", "tokens"}: 1}, viewStart, viewEnd)

	out := render(v)
	if strings.Contains(out, `workspace="old"`) {
		t.Error("a series absent from the newest report is still exported")
	}
	if !strings.Contains(out, `workspace="new"`) {
		t.Error("the newest report's series is missing")
	}
}

func TestSpendView_BeforeFirstRefresh(t *testing.T) {
	out := render(newSpendView(100))
	if strings.Contains(out, "jdwlabs_anthropic_cycle_cost_usd{") {
		t.Error("cost series exported before any refresh")
	}
	if strings.Contains(out, "jdwlabs_anthropic_cycle_end_timestamp_seconds ") {
		t.Error("cycle window exported before any refresh")
	}
	for _, want := range []string{
		"jdwlabs_anthropic_credit_budget_usd 100\n",
		"jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds 0\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\nfull output:\n%s", want, out)
		}
	}
}

func TestSpendView_EscapesLabelValues(t *testing.T) {
	v := newSpendView(100)
	v.replace(map[costSeries]float64{
		{"team \"a\"\\b\nc", "m", "standard", "tokens"}: 1,
	}, viewStart, viewEnd)

	want := `workspace="team \"a\"\\b\nc"`
	if out := render(v); !strings.Contains(out, want) {
		t.Errorf("label not escaped to the exposition format; want %s in:\n%s", want, out)
	}
}
