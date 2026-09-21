package main

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ToolEvidence is one live-state read Holmes made during the investigation:
// a kubectl, Prometheus or log query and what it returned. It is the only
// ground truth the relay holds about the cluster, since the relay itself has
// no read access to it.
type ToolEvidence struct {
	ID          string
	Tool        string
	Description string
	Output      string
}

// Verification is the model's citation of the live read that shows the defect
// its patch fixes, and the excerpt of that read's output that shows it.
type Verification struct {
	ToolCallID string `json:"tool_call_id"`
	Observed   string `json:"observed"`
}

// ErrUnverified marks a proposal that does not tie the defect it asserts to a
// live read the investigation actually made. A remediation that cannot quote
// the cluster showing the problem is a guess about the problem, and guesses
// have been the whole failure mode: a store "missing its version" whose
// version was set, a Secret "missing" that an ExternalSecret already owned.
var ErrUnverified = errors.New("github: asserted defect has no live-state verification")

const (
	// maxEvidenceCalls and maxEvidenceOutputBytes bound what one investigation
	// can make the relay hold. The replica has a 32Mi limit and several
	// investigations can be in flight; the tail of a long log read is not
	// what a citation needs.
	maxEvidenceCalls       = 24
	maxEvidenceOutputBytes = 16 << 10

	// minObservedRunes stops a citation made of a token that appears in every
	// read ("True", a namespace name) from passing as evidence of anything.
	minObservedRunes = 12
	// maxObservedRunes bounds the excerpt quoted into the PR body.
	maxObservedRunes = 1500
)

// secretRead matches a tool call that read Secret objects. Its output must
// never be quoted into a pull request, so it cannot serve as the citation
// even when it is the read that shows the defect. Word boundaries keep an
// ExternalSecret or ClusterSecretStore read citable.
var secretRead = regexp.MustCompile(`(?i)\bsecrets?\b`)

// truncateUTF8 cuts s to at most n bytes without splitting a rune.
func truncateUTF8(s string, n int) string {
	if len(s) <= n {
		return s
	}
	s = s[:n]
	for len(s) > 0 && !utf8.ValidString(s) {
		s = s[:len(s)-1]
	}
	return s
}

// normalizeSpace collapses every whitespace run to one space so a citation
// survives the model re-wrapping or re-indenting the lines it copied.
func normalizeSpace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// vetVerification checks the citation deterministically: it must name a read
// that actually ran and returned output, and the excerpt must appear in that
// output verbatim, whitespace aside. That does not prove the model reasoned
// correctly about what it quoted — the PR body puts the quote beside the
// claim for the reviewer to judge — but it does make a defect invented
// without any live read unrepresentable.
func vetVerification(p Patch) (ToolEvidence, error) {
	v := p.Verification
	if v == nil || strings.TrimSpace(v.ToolCallID) == "" || strings.TrimSpace(v.Observed) == "" {
		return ToolEvidence{}, fmt.Errorf("%w: the proposal cites no live read", ErrUnverified)
	}
	if len(p.Evidence) == 0 {
		return ToolEvidence{}, fmt.Errorf("%w: the investigation made no live reads to cite", ErrUnverified)
	}
	var ev *ToolEvidence
	for i := range p.Evidence {
		if p.Evidence[i].ID == strings.TrimSpace(v.ToolCallID) {
			ev = &p.Evidence[i]
			break
		}
	}
	if ev == nil {
		return ToolEvidence{}, fmt.Errorf("%w: cited read %q was not made by the investigation", ErrUnverified, v.ToolCallID)
	}
	if secretRead.MatchString(ev.Tool + " " + ev.Description) {
		return ToolEvidence{}, fmt.Errorf("%w: cited read %q reads Secret objects and cannot be quoted", ErrUnverified, v.ToolCallID)
	}
	observed := normalizeSpace(v.Observed)
	if n := utf8.RuneCountInString(observed); n < minObservedRunes || n > maxObservedRunes {
		return ToolEvidence{}, fmt.Errorf("%w: cited excerpt is %d characters, want %d..%d", ErrUnverified, n, minObservedRunes, maxObservedRunes)
	}
	if !strings.Contains(normalizeSpace(ev.Output), observed) {
		return ToolEvidence{}, fmt.Errorf("%w: cited excerpt does not appear in the output of read %q", ErrUnverified, v.ToolCallID)
	}
	return *ev, nil
}

// renderVerification is the PR body section a reviewer checks the claim
// against. The excerpt is fenced with a run of tildes longer than any in the
// excerpt so cluster output cannot close the fence and inject markdown.
func renderVerification(ev ToolEvidence, observed string) string {
	fence := "~~~"
	for strings.Contains(observed, fence) {
		fence += "~"
	}
	return fmt.Sprintf("### Live-state verification\n\nRead `%s` during the investigation: %s\n\nObserved:\n\n%s\n%s\n%s\n",
		safeSummary(strings.ReplaceAll(ev.Tool, "`", ""), 80), safeSummary(ev.Description, 300), fence, stripControl(strings.TrimSpace(observed)), fence)
}

// stripControl drops control characters other than newline and tab, which
// could otherwise rewrite how the quoted cluster output renders.
func stripControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' || !unicode.IsControl(r) {
			return r
		}
		return -1
	}, s)
}
