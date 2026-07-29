package main

import (
	"fmt"
	"io"
	"sync/atomic"
)

// counters is the relay's own operational signal. Skipping a repeat is
// indistinguishable from a stalled pipeline unless both outcomes are counted,
// so the ratio between these two is the thing worth alerting on.
//
// Written by hand in the Prometheus text exposition format: the relay carries
// no metrics dependency today, and two counters do not justify adding one to a
// 32Mi replica.
type counters struct {
	investigationsRun atomic.Int64
	repeatsSkipped    atomic.Int64
}

func (c *counters) writeTo(w io.Writer) {
	fmt.Fprint(w, "# HELP ai_sre_relay_investigations_run_total Alert investigations executed.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_investigations_run_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_investigations_run_total %d\n", c.investigationsRun.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_repeats_skipped_total Repeat notifications not investigated because an open ticket already covers the firing episode.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_repeats_skipped_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_repeats_skipped_total %d\n", c.repeatsSkipped.Load())
}
