package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
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

// embedColor picks red/orange/gray by severity; a resolved alert is always
// green regardless of severity, since it's good news rather than still-bad.
func embedColor(a Alert) int {
	if a.Status == "resolved" {
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

func (d *DiscordNotifier) buildEmbed(a Alert, an Analysis, issue IssueKey, pr *PRLink, patch *Patch) discordEmbed {
	e := discordEmbed{
		Title:       a.Name(),
		Description: truncate(an.RootCause, 2000),
		Color:       embedColor(a),
	}
	if ns := a.Namespace(); ns != "" {
		e.Fields = append(e.Fields, discordEmbedField{Name: "Namespace", Value: ns, Inline: true})
	}
	if sev := a.Severity(); sev != "" {
		e.Fields = append(e.Fields, discordEmbedField{Name: "Severity", Value: sev, Inline: true})
	}
	if issue != "" {
		e.Fields = append(e.Fields, discordEmbedField{
			Name:  "Jira",
			Value: fmt.Sprintf("[%s](%s/browse/%s)", issue, d.jiraBaseURL, issue),
		})
	}
	if pr != nil {
		e.Fields = append(e.Fields, discordEmbedField{Name: "PR", Value: string(*pr)})
	}
	if patch != nil {
		e.Fields = append(e.Fields, discordEmbedField{
			Name:  "Patch",
			Value: fmt.Sprintf("%.0f%% confidence — %s", patch.Confidence*100, truncate(patch.Rationale, 900)),
		})
	}
	if _, err := time.Parse(time.RFC3339, a.StartsAt); err == nil {
		e.Timestamp = a.StartsAt
	}
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

// truncate keeps the embed description within a readable budget well under
// Discord's 4096-char description ceiling.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
