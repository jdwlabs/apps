package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

// A data-loss alert reports something a resolve does not undo: a lost chunk
// stays lost after the evidence window that raised the alert has rolled off.
// Treating its resolve like a transient fault's closed a ticket describing a
// permanent loss, so these alerts are never closed by the relay and a growing
// value is never dismissed as a repeat.

// defaultDataLossAlerts names the world-integrity alerts of the Minecraft
// charts. Names match as a suffix so a chart's release prefix does not matter.
var defaultDataLossAlerts = []string{
	"WorldChunksLost",
	"WorldChunkCountDropped",
	"WorldCorruptionReported",
	"WorldCorruptionLogged",
	"WorldArchiveShrank",
	"WorldGuardHeld",
	"WorldGuardWouldHold",
}

const (
	defaultDataLossValueAnnotation = "summary"
	// defaultDataLossGrowthPct is how much a value must grow over the last
	// recorded one to count as material; a one-chunk wobble is not news.
	defaultDataLossGrowthPct = 10
)

const (
	datalossLabel      = "dataloss"
	needsHumanLabel    = "needs-human"
	datalossPriority   = "High"
	datalossOptInValue = "true"
)

var numberRe = regexp.MustCompile(`-?\d[\d,]*(?:\.\d+)?(?:[eE][+-]?\d+)?`)

// datalossPolicy decides which alerts are data-loss alerts and when a changed
// value is worth re-investigating.
type datalossPolicy struct {
	names           []string // lowercased suffixes
	valueAnnotation string
	growthPct       float64
}

func newDatalossPolicy(names []string, valueAnnotation string, growthPct float64) datalossPolicy {
	if len(names) == 0 {
		names = defaultDataLossAlerts
	}
	if valueAnnotation == "" {
		valueAnnotation = defaultDataLossValueAnnotation
	}
	if growthPct <= 0 {
		growthPct = defaultDataLossGrowthPct
	}
	p := datalossPolicy{valueAnnotation: valueAnnotation, growthPct: growthPct}
	for _, n := range names {
		if n = strings.ToLower(strings.TrimSpace(n)); n != "" {
			p.names = append(p.names, n)
		}
	}
	return p
}

func defaultDatalossPolicy() datalossPolicy { return newDatalossPolicy(nil, "", 0) }

// isDataLoss reports whether the alert is in the class: opted in by a
// `dataloss: "true"` label or annotation, or named in the configured list.
func (p datalossPolicy) isDataLoss(a Alert) bool {
	if strings.EqualFold(a.Labels[datalossLabel], datalossOptInValue) ||
		strings.EqualFold(a.Annotations[datalossLabel], datalossOptInValue) {
		return true
	}
	name := strings.ToLower(a.Name())
	if name == "" {
		return false
	}
	for _, n := range p.names {
		if strings.HasSuffix(name, n) {
			return true
		}
	}
	return false
}

// value extracts the alert's headline number, such as the lost-chunk count
// from "has lost 14105 world chunks". ok is false when there is none.
func (p datalossPolicy) value(a Alert) (float64, bool) {
	for _, text := range []string{a.Annotations[p.valueAnnotation], a.Annotations["value"]} {
		m := numberRe.FindString(text)
		if m == "" {
			continue
		}
		v, err := strconv.ParseFloat(strings.ReplaceAll(m, ",", ""), 64)
		if err == nil && !math.IsNaN(v) && !math.IsInf(v, 0) {
			return v, true
		}
	}
	return 0, false
}

// grew reports whether next exceeds prev by at least the configured share.
func (p datalossPolicy) grew(prev, next float64) bool {
	return next > prev && next-prev >= math.Abs(prev)*p.growthPct/100
}

func growthNote(prev, next float64) string {
	return fmt.Sprintf("📈 Data-loss value grew from %s to %s since the last investigation of this alert; re-investigated.",
		formatValue(prev), formatValue(next))
}

func formatValue(v float64) string {
	if v == math.Trunc(v) && math.Abs(v) < 1e15 {
		s := strconv.FormatInt(int64(v), 10)
		neg := strings.HasPrefix(s, "-")
		s = strings.TrimPrefix(s, "-")
		for i := len(s) - 3; i > 0; i -= 3 {
			s = s[:i] + "," + s[i:]
		}
		if neg {
			s = "-" + s
		}
		return s
	}
	return strconv.FormatFloat(v, 'f', -1, 64)
}

func datalossResolveNotice(issue IssueKey, at string) string {
	return fmt.Sprintf("⚠️ Data-loss alert resolved at %s. %s stays open: a human must confirm the data was restored.", at, issue)
}
