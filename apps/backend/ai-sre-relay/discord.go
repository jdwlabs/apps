package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"
)

type DiscordNotifier struct {
	webhookURL  string
	jiraBaseURL string
	hc          *http.Client
}

func NewDiscordNotifier(webhookURL, jiraBaseURL string, hc *http.Client) *DiscordNotifier {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &DiscordNotifier{webhookURL: webhookURL, jiraBaseURL: jiraBaseURL, hc: hc}
}

type discordEmbed struct {
	Title       string              `json:"title"`
	URL         string              `json:"url,omitempty"`
	Description string              `json:"description"`
	Color       int                 `json:"color"`
	Fields      []discordEmbedField `json:"fields,omitempty"`
	Timestamp   string              `json:"timestamp,omitempty"`
}

type discordEmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline bool   `json:"inline"`
}

const (
	colorRed    = 0xE74C3C
	colorOrange = 0xF39C12
	colorGray   = 0x95A5A6
	colorGreen  = 0x2ECC71
)

// Discord's embed limits, counted in characters. The total covers title,
// description and every field name and value together.
const (
	maxEmbedTitle       = 256
	maxEmbedDescription = 4096
	maxEmbedFieldValue  = 1024
	maxEmbedTotal       = 6000
)

const codeFence = "```"

// embedColor picks red/orange/gray by severity; a resolved alert is always
// green regardless of severity, since it's good news rather than still-bad.
func embedColor(a Alert) int {
	if a.Status == statusResolved {
		return colorGreen
	}
	switch a.Severity() {
	case "critical":
		return colorRed
	case "warning":
		return colorOrange
	default:
		return colorGray
	}
}

// embedTitle leads with the alert state so a feed of embeds reads firing
// versus cleared without opening each one.
func embedTitle(a Alert) string {
	prefix := "🔥 FIRING"
	if a.Status == statusResolved {
		prefix = "✅ RESOLVED"
	}
	return truncate(prefix+" · "+a.Name(), maxEmbedTitle)
}

func (d *DiscordNotifier) buildEmbed(a Alert, an Analysis, issue IssueKey, pr *PRLink, patch *Patch) discordEmbed {
	e := discordEmbed{
		Title: embedTitle(a),
		Color: embedColor(a),
	}
	if issue != "" {
		e.URL = fmt.Sprintf("%s/browse/%s", d.jiraBaseURL, issue)
	}
	if ns := a.Namespace(); ns != "" {
		e.Fields = append(e.Fields, discordEmbedField{Name: "Namespace", Value: truncate(ns, maxEmbedFieldValue), Inline: true})
	}
	if sev := a.Severity(); sev != "" {
		e.Fields = append(e.Fields, discordEmbedField{Name: "Severity", Value: truncate(sev, maxEmbedFieldValue), Inline: true})
	}
	if issue != "" {
		e.Fields = append(e.Fields, discordEmbedField{
			Name:  "Jira",
			Value: truncate(fmt.Sprintf("[%s](%s/browse/%s)", issue, d.jiraBaseURL, issue), maxEmbedFieldValue),
		})
	}
	if pr != nil {
		e.Fields = append(e.Fields, discordEmbedField{Name: "PR", Value: truncate(string(*pr), maxEmbedFieldValue)})
	}
	if patch != nil {
		e.Fields = append(e.Fields, discordEmbedField{
			Name:  "Patch",
			Value: truncate(fmt.Sprintf("%.0f%% confidence — %s", patch.Confidence*100, truncate(patch.Rationale, 900)), maxEmbedFieldValue),
		})
	}
	if _, err := time.Parse(time.RFC3339, a.StartsAt); err == nil {
		e.Timestamp = a.StartsAt
	}

	// The description gets whatever the 6000-character total leaves after the
	// title and fields, never more than its own cap.
	used := utf8.RuneCountInString(e.Title)
	for _, f := range e.Fields {
		used += utf8.RuneCountInString(f.Name) + utf8.RuneCountInString(f.Value)
	}
	budget := min(maxEmbedDescription, maxEmbedTotal-used)
	e.Description = truncateMarkdown(tablesToCodeBlocks(an.RootCause), budget, issue)
	return e
}

func (d *DiscordNotifier) Notify(ctx context.Context, a Alert, an Analysis, issue IssueKey, pr *PRLink, patch *Patch) error {
	embed := d.buildEmbed(a, an, issue, pr, patch)
	body, _ := json.Marshal(map[string]any{"embeds": []discordEmbed{embed}})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := d.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("discord: status %d", resp.StatusCode)
	}
	return nil
}

// truncate shortens s to at most n characters, appending an ellipsis. It cuts
// on rune boundaries: slicing bytes can split a multibyte character, and
// Discord rejects the whole request for invalid UTF-8.
func truncate(s string, n int) string {
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	return string([]rune(s)[:max(n-1, 0)]) + "…"
}

// truncateMarkdown fits markdown into n characters, cutting at a paragraph
// (else line) boundary so a section is not sliced mid-sentence, and says where
// the rest lives. A code fence left open by the cut is closed so the pointer
// does not render as code.
func truncateMarkdown(s string, n int, issue IssueKey) string {
	if n <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= n {
		return s
	}
	pointer := "\n\n…truncated"
	if issue != "" {
		pointer = fmt.Sprintf("\n\n…continued in Jira %s", issue)
	}
	closer := "\n" + codeFence
	avail := n - utf8.RuneCountInString(pointer) - utf8.RuneCountInString(closer)
	if avail <= 0 {
		return truncate(s, n)
	}
	cut := string([]rune(s)[:avail])
	// A boundary near the start would throw away most of the budget to save a
	// few characters of an unbroken line, so only honour one past a third.
	floor := len(cut) / 3
	if i := strings.LastIndex(cut, "\n\n"); i >= floor {
		cut = cut[:i]
	} else if i := strings.LastIndex(cut, "\n"); i >= floor {
		cut = cut[:i]
	}
	cut = strings.TrimRight(cut, " \t\n")
	if fenceOpen(cut) {
		cut += closer
	}
	return cut + pointer
}

// fenceOpen reports whether s ends inside a code fence.
func fenceOpen(s string) bool {
	open := false
	for _, line := range strings.Split(s, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), codeFence) {
			open = !open
		}
	}
	return open
}

// tablesToCodeBlocks rewrites markdown tables as aligned monospace fences,
// because embeds render pipes literally. Tables inside existing code fences are
// left alone.
func tablesToCodeBlocks(s string) string {
	lines := strings.Split(s, "\n")
	out := make([]string, 0, len(lines))
	inFence := false
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if strings.HasPrefix(strings.TrimSpace(line), codeFence) {
			inFence = !inFence
		}
		if inFence || !isTableRow(line) || i+1 >= len(lines) || !isTableSeparator(lines[i+1]) {
			out = append(out, line)
			continue
		}
		rows := [][]string{splitTableRow(line)}
		i += 2
		for ; i < len(lines) && isTableRow(lines[i]); i++ {
			rows = append(rows, splitTableRow(lines[i]))
		}
		i--
		out = append(out, renderTable(rows))
	}
	return strings.Join(out, "\n")
}

func isTableRow(line string) bool {
	t := strings.TrimSpace(line)
	return strings.HasPrefix(t, "|") && strings.Count(t, "|") >= 2
}

func isTableSeparator(line string) bool {
	if !isTableRow(line) {
		return false
	}
	for _, c := range splitTableRow(line) {
		if c == "" || strings.Trim(c, "-: ") != "" {
			return false
		}
	}
	return true
}

func splitTableRow(line string) []string {
	t := strings.TrimSpace(line)
	t = strings.TrimSuffix(strings.TrimPrefix(t, "|"), "|")
	cells := strings.Split(t, "|")
	// Emphasis and code markers would show literally inside a fence.
	strip := strings.NewReplacer("**", "", "__", "", "`", "")
	for i, c := range cells {
		cells[i] = strip.Replace(strings.TrimSpace(c))
	}
	return cells
}

// maxTableCell keeps one long cell from forcing a table wider than a phone
// screen, where a fence scrolls rather than wraps.
const maxTableCell = 48

func renderTable(rows [][]string) string {
	cols := 0
	for _, r := range rows {
		cols = max(cols, len(r))
	}
	widths := make([]int, cols)
	for ri, r := range rows {
		for ci := range r {
			rows[ri][ci] = truncate(r[ci], maxTableCell)
			widths[ci] = max(widths[ci], utf8.RuneCountInString(rows[ri][ci]))
		}
	}
	ruleWidth := 2 * max(cols-1, 0)
	for _, w := range widths {
		ruleWidth += w
	}
	var b strings.Builder
	b.WriteString(codeFence + "\n")
	for ri, r := range rows {
		var line strings.Builder
		for ci := range cols {
			cell := ""
			if ci < len(r) {
				cell = r[ci]
			}
			line.WriteString(cell)
			if ci < cols-1 {
				line.WriteString(strings.Repeat(" ", widths[ci]-utf8.RuneCountInString(cell)) + "  ")
			}
		}
		b.WriteString(strings.TrimRight(line.String(), " ") + "\n")
		if ri == 0 {
			b.WriteString(strings.Repeat("-", ruleWidth) + "\n")
		}
	}
	b.WriteString(codeFence)
	return b.String()
}
