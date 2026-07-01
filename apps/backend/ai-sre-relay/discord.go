package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type DiscordNotifier struct {
	webhookURL string
	hc         *http.Client
}

func NewDiscordNotifier(webhookURL string, hc *http.Client) *DiscordNotifier {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &DiscordNotifier{webhookURL: webhookURL, hc: hc}
}

func (d *DiscordNotifier) Notify(ctx context.Context, a Alert, an Analysis, issue IssueKey, pr *PRLink) error {
	msg := fmt.Sprintf("**AI-SRE — %s**\n%s", a.Name(), truncate(an.RootCause, 1500))
	if issue != "" {
		msg += fmt.Sprintf("\nJira: %s", issue)
	}
	if pr != nil {
		msg += fmt.Sprintf("\nPR: %s", *pr)
	}
	body, _ := json.Marshal(map[string]string{"content": msg})
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

// truncate keeps Discord's 2000-char content ceiling safe.
func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
