package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type JiraClient struct {
	baseURL    string
	email      string
	token      string
	projectKey string
	issueType  string
	hc         *http.Client
}

func NewJiraClient(baseURL, email, token, projectKey, issueType string, hc *http.Client) *JiraClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &JiraClient{baseURL: baseURL, email: email, token: token, projectKey: projectKey, issueType: issueType, hc: hc}
}

func (j *JiraClient) label(a Alert) string { return "amfp-" + a.Fingerprint }

func (j *JiraClient) do(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var r *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		r = bytes.NewReader(b)
	} else {
		r = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, j.baseURL+path, r)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(j.email, j.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	return j.hc.Do(req)
}

// Upsert creates a new issue for the alert, comments on the existing open
// issue that carries the fingerprint dedup label, or — when the most recent
// issue for that fingerprint is Done — reopens it before commenting.
//
// Alertmanager fingerprints are stable per labelset, so the same underlying
// condition re-firing after a human closed the ticket means "resolved for
// now", not "never contact again". Excluding Done issues from the dedup
// search (as the original implementation did) made the relay create a fresh
// duplicate on every refire once a ticket was closed (JDWLABS-126).
func (j *JiraClient) Upsert(ctx context.Context, a Alert, an Analysis) (IssueKey, error) {
	// /rest/api/3/search was removed by Atlassian in 2025; its /search/jql
	// replacement returns bare issue IDs unless fields are requested. status
	// is requested too so we can tell a Done match from an open one without
	// a second round-trip; ORDER BY + maxResults=1 pins it to the most
	// recently touched issue for this fingerprint.
	jql := fmt.Sprintf(`project = %s AND labels = "%s" ORDER BY created DESC`, j.projectKey, j.label(a))
	resp, err := j.do(ctx, http.MethodGet, "/rest/api/3/search/jql?fields=key,status&maxResults=1&jql="+url.QueryEscape(jql), nil)
	if err != nil {
		return "", err
	}
	// A non-2xx search response (auth failure, rate-limit, server error) returns
	// a JSON error body that decodes fine with Issues == nil, which would
	// incorrectly pass the dedup check and create a duplicate issue. Fail-safe:
	// surface the error instead.
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return "", fmt.Errorf("jira search: status %d", resp.StatusCode)
	}
	var search struct {
		Issues []struct {
			Key    string `json:"key"`
			Fields struct {
				Status struct {
					StatusCategory struct {
						Key string `json:"key"`
					} `json:"statusCategory"`
				} `json:"status"`
			} `json:"fields"`
		} `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&search); err != nil {
		resp.Body.Close()
		return "", err
	}
	resp.Body.Close()

	if len(search.Issues) == 0 {
		return j.create(ctx, a, an)
	}

	issue := search.Issues[0]
	key := IssueKey(issue.Key)
	text := an.RootCause
	if issue.Fields.Status.StatusCategory.Key == "done" {
		if err := j.reopen(ctx, key); err != nil {
			return "", err
		}
		text = "🔁 Alert re-fired after this ticket was closed; reopening.\n\n" + text
	}
	return j.commentOn(ctx, key, text)
}

func (j *JiraClient) commentOn(ctx context.Context, key IssueKey, text string) (IssueKey, error) {
	cResp, err := j.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/comment", key), adf(text))
	if err != nil {
		return "", err
	}
	defer cResp.Body.Close()
	if cResp.StatusCode >= 300 {
		return "", fmt.Errorf("jira comment: status %d", cResp.StatusCode)
	}
	return key, nil
}

func (j *JiraClient) create(ctx context.Context, a Alert, an Analysis) (IssueKey, error) {
	create := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": j.projectKey},
			"issuetype":   map[string]string{"name": j.issueType},
			"summary":     "AI-SRE: " + a.Name(),
			"description": adfDoc(an.RootCause),
			"labels":      []string{j.label(a), "ai-sre"},
		},
	}
	cResp, err := j.do(ctx, http.MethodPost, "/rest/api/3/issue", create)
	if err != nil {
		return "", err
	}
	defer cResp.Body.Close()
	if cResp.StatusCode >= 300 {
		return "", fmt.Errorf("jira create: status %d", cResp.StatusCode)
	}
	var created struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(cResp.Body).Decode(&created); err != nil {
		return "", err
	}
	return IssueKey(created.Key), nil
}

// reopen transitions a Done issue back to an open status so a refiring
// alert updates the original ticket instead of spawning a duplicate. It
// picks the first available transition whose target status sits outside
// the Done category; workflows name this transition differently ("Reopen",
// "Backlog", "To Do", ...), so the target status category is what matters,
// not the transition name.
func (j *JiraClient) reopen(ctx context.Context, key IssueKey) error {
	resp, err := j.do(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s/transitions", key), nil)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("jira transitions: status %d", resp.StatusCode)
	}
	var t struct {
		Transitions []struct {
			ID string `json:"id"`
			To struct {
				StatusCategory struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"to"`
		} `json:"transitions"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&t); err != nil {
		resp.Body.Close()
		return err
	}
	resp.Body.Close()

	var transitionID string
	for _, tr := range t.Transitions {
		if tr.To.StatusCategory.Key != "done" {
			transitionID = tr.ID
			break
		}
	}
	if transitionID == "" {
		return fmt.Errorf("jira reopen %s: no non-Done transition available", key)
	}

	tResp, err := j.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/transitions", key),
		map[string]any{"transition": map[string]string{"id": transitionID}})
	if err != nil {
		return err
	}
	defer tResp.Body.Close()
	if tResp.StatusCode >= 300 {
		return fmt.Errorf("jira transition %s: status %d", key, tResp.StatusCode)
	}
	return nil
}

// adf wraps text as an Atlassian Document Format comment body.
func adf(text string) map[string]any {
	return map[string]any{"body": adfDoc(text)}
}

func adfDoc(text string) map[string]any {
	return map[string]any{
		"type":    "doc",
		"version": 1,
		"content": []any{map[string]any{
			"type":    "paragraph",
			"content": []any{map[string]any{"type": "text", "text": text}},
		}},
	}
}
