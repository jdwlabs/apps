package main

import (
	"fmt"
	"io"
	"sync/atomic"
)

// counters is the relay's own operational signal. Skipping a repeat is
// indistinguishable from a stalled pipeline unless both outcomes are counted,
// so the ratio between the run and skip counts is the thing worth alerting on.
//
// Written by hand in the Prometheus text exposition format: the relay carries
// no metrics dependency today, and this handful of counters does not justify
// adding one to a 32Mi replica.
type counters struct {
	investigationsRun atomic.Int64
	repeatsSkipped    atomic.Int64
	// reposRejected counts remediations produced and then dropped at the
	// allowlist. It is the difference between "no alert ever warranted a PR"
	// and "every PR this arm proposed was aimed somewhere it may not write" —
	// two states that look identical from the outside as a PR count of zero.
	reposRejected atomic.Int64
	// pathsRejected is the same signal one level down: the repository was
	// right and the file was not — unwatched by ArgoCD or nonexistent. Every
	// unusable PR before this gate existed would have counted here.
	pathsRejected atomic.Int64
	// branchesSkipped counts proposals dropped because the ticket's PR branch
	// already existed. Expected to tick on refires; a burst means a ticket is
	// generating a new proposal every cycle and its PR should be looked at.
	branchesSkipped atomic.Int64
	// branchesOrphaned counts proposals dropped because the ticket's PR
	// branch already existed but carried no pull request in any state — a
	// previous run failed after creating the branch and never opened one.
	// Unlike branchesSkipped this is never expected: any nonzero rate means a
	// commit is sitting unseen and wants a human to go look.
	branchesOrphaned atomic.Int64
	// ticketsAutoClosed counts tickets the relay closed itself after their
	// alert stayed resolved for the grace period. Read against the rate of
	// investigations: a zero here while investigations keep landing means
	// resolve notifications are not arriving (send_resolved off on the
	// receiver, most likely) and every ticket is again waiting on a human.
	ticketsAutoClosed atomic.Int64
	// alertsRejected counts refusal responses, not distinct alerts: a sender
	// retrying one alert increments it once per attempt, so during a storm it
	// runs an order of magnitude above the number of alerts actually affected.
	// Read it as refusal pressure and alert on its rate, never as a count of
	// lost coverage. It is still the only signal there is — every other counter
	// here is incremented from inside the pipeline, which a refused alert never
	// reaches, so without it an overflow is invisible to Prometheus and only a
	// burst of unread ERROR lines marks the gap.
	alertsRejected atomic.Int64
}

func (c *counters) writeTo(w io.Writer) {
	fmt.Fprint(w, "# HELP ai_sre_relay_investigations_run_total Alert investigations executed.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_investigations_run_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_investigations_run_total %d\n", c.investigationsRun.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_repeats_skipped_total Repeat notifications not investigated because an open ticket already covers the firing episode.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_repeats_skipped_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_repeats_skipped_total %d\n", c.repeatsSkipped.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_repo_rejections_total Remediations discarded because the proposed repository was not allowlisted.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_repo_rejections_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_repo_rejections_total %d\n", c.reposRejected.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_path_rejections_total Remediations discarded because the proposed file was not a watched, existing manifest.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_path_rejections_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_path_rejections_total %d\n", c.pathsRejected.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_branches_skipped_total Remediations dropped because a PR branch for the ticket already existed.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_branches_skipped_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_branches_skipped_total %d\n", c.branchesSkipped.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_branches_orphaned_total Remediations dropped because the ticket's PR branch existed with no pull request in any state, the remnant of a run that failed after creating it.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_branches_orphaned_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_branches_orphaned_total %d\n", c.branchesOrphaned.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_tickets_auto_closed_total Tickets the relay transitioned to Done after their alert stayed resolved for the close grace period.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_tickets_auto_closed_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_tickets_auto_closed_total %d\n", c.ticketsAutoClosed.Load())
	fmt.Fprint(w, "# HELP ai_sre_relay_alerts_rejected_total Refusal responses returned because the investigation queue was full, including repeated sender retries of the same alert; not a count of distinct alerts.\n")
	fmt.Fprint(w, "# TYPE ai_sre_relay_alerts_rejected_total counter\n")
	fmt.Fprintf(w, "ai_sre_relay_alerts_rejected_total %d\n", c.alertsRejected.Load())
}
