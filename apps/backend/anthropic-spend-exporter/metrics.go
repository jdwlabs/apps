package main

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// spendView is the exporter's whole current picture of the billing cycle.
// The costs, the window they belong to and the refresh that produced them
// are swapped together: a total from one cycle shown against another
// cycle's dates, or a new total stamped with the previous refresh time,
// would look like a real reading and be wrong.
type spendView struct {
	mu              sync.RWMutex
	costs           map[costSeries]float64
	cycleStart      time.Time
	cycleEnd        time.Time
	lastRefreshUnix int64

	budgetUSD float64

	// Written by hand in the Prometheus text exposition format, like the
	// other in-house exporters: this handful of series does not justify a
	// metrics client dependency.
	refreshesOK     atomic.Int64
	refreshesFailed atomic.Int64
}

func newSpendView(budgetUSD float64) *spendView {
	return &spendView{costs: map[costSeries]float64{}, budgetUSD: budgetUSD}
}

// replace takes ownership of costs. Replacing wholesale is what makes a
// workspace or model that drops out of the report disappear from /metrics
// instead of serving its last value forever.
func (v *spendView) replace(costs map[costSeries]float64, start, end, refreshedAt time.Time) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.costs = costs
	v.cycleStart = start
	v.cycleEnd = end
	v.lastRefreshUnix = refreshedAt.Unix()
}

// Workspace names are free text chosen in the Console, so they are escaped
// to the exposition format's own rules. Go's %q is not a substitute: it
// writes escapes such as \x00 that the format does not define.
var labelEscaper = strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`)

func (v *spendView) writeTo(w io.Writer) {
	v.mu.RLock()
	costs := v.costs
	start, end := v.cycleStart, v.cycleEnd
	lastRefreshUnix := v.lastRefreshUnix
	v.mu.RUnlock()

	keys := make([]costSeries, 0, len(costs))
	for k := range costs {
		keys = append(keys, k)
	}
	// Sorted so two scrapes of an unchanged view are byte-identical, which
	// keeps a by-hand diff of two scrapes meaningful.
	sort.Slice(keys, func(i, j int) bool {
		a, b := keys[i], keys[j]
		if a.workspace != b.workspace {
			return a.workspace < b.workspace
		}
		if a.model != b.model {
			return a.model < b.model
		}
		if a.serviceTier != b.serviceTier {
			return a.serviceTier < b.serviceTier
		}
		return a.costType < b.costType
	})

	fmt.Fprint(w, "# HELP jdwlabs_anthropic_cycle_cost_usd Anthropic API cost since the start of the current billing cycle, in USD.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_anthropic_cycle_cost_usd gauge\n")
	for _, k := range keys {
		fmt.Fprintf(w, "jdwlabs_anthropic_cycle_cost_usd{workspace=\"%s\",model=\"%s\",service_tier=\"%s\",cost_type=\"%s\"} %g\n",
			labelEscaper.Replace(k.workspace), labelEscaper.Replace(k.model),
			labelEscaper.Replace(k.serviceTier), labelEscaper.Replace(k.costType), costs[k])
	}

	fmt.Fprint(w, "# HELP jdwlabs_anthropic_credit_budget_usd Configured API credit per billing cycle, in USD.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_anthropic_credit_budget_usd gauge\n")
	fmt.Fprintf(w, "jdwlabs_anthropic_credit_budget_usd %g\n", v.budgetUSD)

	// The window is absent until the first refresh succeeds. Publishing the
	// configured dates before any cost has been read would let the
	// expiring-unused alert fire on a pod that has never reached Anthropic.
	if !start.IsZero() {
		fmt.Fprint(w, "# HELP jdwlabs_anthropic_cycle_start_timestamp_seconds Unix time the current billing cycle started, from configuration.\n")
		fmt.Fprint(w, "# TYPE jdwlabs_anthropic_cycle_start_timestamp_seconds gauge\n")
		fmt.Fprintf(w, "jdwlabs_anthropic_cycle_start_timestamp_seconds %d\n", start.Unix())
		fmt.Fprint(w, "# HELP jdwlabs_anthropic_cycle_end_timestamp_seconds Unix time the current billing cycle ends, from configuration.\n")
		fmt.Fprint(w, "# TYPE jdwlabs_anthropic_cycle_end_timestamp_seconds gauge\n")
		fmt.Fprintf(w, "jdwlabs_anthropic_cycle_end_timestamp_seconds %d\n", end.Unix())
	}

	fmt.Fprint(w, "# HELP jdwlabs_anthropic_spend_exporter_refreshes_total Anthropic polling cycles, by outcome.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_anthropic_spend_exporter_refreshes_total counter\n")
	fmt.Fprintf(w, "jdwlabs_anthropic_spend_exporter_refreshes_total{outcome=\"ok\"} %d\n", v.refreshesOK.Load())
	fmt.Fprintf(w, "jdwlabs_anthropic_spend_exporter_refreshes_total{outcome=\"failed\"} %d\n", v.refreshesFailed.Load())

	fmt.Fprint(w, "# HELP jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds Unix time of the last refresh that replaced the cost view; 0 until the first.\n")
	fmt.Fprint(w, "# TYPE jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds gauge\n")
	fmt.Fprintf(w, "jdwlabs_anthropic_spend_exporter_last_refresh_timestamp_seconds %d\n", lastRefreshUnix)
}
