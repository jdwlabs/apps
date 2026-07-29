package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

type JiraClient struct {
	baseURL    string
	email      string
	token      string
	projectKey string
	issueType  string
	hc         *http.Client

	// mu serializes upserts and guards known. Jira offers no unique-constraint
	// primitive and its JQL index is eventually consistent — a search issued
	// right after a create can miss the fresh issue — so the correctness of
	// the search-then-create sequence depends on upserts running one at a time
	// with their results remembered in-process.
	mu    sync.Mutex
	known map[string]IssueKey // dedup label -> issue key created or matched by this process
}

func NewJiraClient(baseURL, email, token, projectKey, issueType string, hc *http.Client) *JiraClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &JiraClient{
		baseURL: baseURL, email: email, token: token,
		projectKey: projectKey, issueType: issueType, hc: hc,
		known: map[string]IssueKey{},
	}
}

func (j *JiraClient) label(a Alert) string { return "amfp-" + a.Fingerprint }

// alertLabel keys dedup at alert-name granularity. Alertmanager fingerprints
// hash the full labelset, so one incident commonly fans out into several
// fingerprints of the same alertname (e.g. TargetDown once per job); a
// fingerprint-only key then files a sibling ticket per labelset. An open
// ticket for the same alertname absorbs those refires as comments instead.
func (j *JiraClient) alertLabel(a Alert) string {
	name := strings.ToLower(strings.TrimSpace(a.Name()))
	if name == "" {
		return ""
	}
	return "amalert-" + strings.ReplaceAll(name, " ", "-")
}

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

// Upsert routes the alert to exactly one issue, in strength order:
//
//  1. an issue already known for this fingerprint (in-process memory first,
//     then a label search) — commented on, reopened first if Done;
//  2. an open issue for the same alertname under a different fingerprint —
//     commented on, so one incident fanning out across labelsets stays on one
//     ticket. Done issues are excluded here: a re-fire under a new labelset
//     after the incident was closed is new work;
//  3. otherwise a fresh issue carrying both dedup labels.
//
// Alertmanager fingerprints are stable per labelset, so the same underlying
// condition re-firing after a human closed the ticket means "resolved for
// now", not "never contact again": a Done fingerprint match is reopened
// rather than duplicated.
func (j *JiraClient) Upsert(ctx context.Context, a Alert, an Analysis) (IssueKey, error) {
	j.mu.Lock()
	defer j.mu.Unlock()

	if key, ok := j.known[j.label(a)]; ok {
		done, err := j.isDone(ctx, key)
		if err != nil {
			return "", err
		}
		return j.updateIssue(ctx, key, an, done)
	}
	hit, err := j.searchOne(ctx, fmt.Sprintf(`project = %s AND labels = %q ORDER BY created DESC`, j.projectKey, j.label(a)))
	if err != nil {
		return "", err
	}
	if hit != nil {
		key := IssueKey(hit.Key)
		j.known[j.label(a)] = key
		return j.updateIssue(ctx, key, an, hit.done())
	}

	if al := j.alertLabel(a); al != "" {
		if key, ok := j.known[al]; ok {
			done, err := j.isDone(ctx, key)
			if err != nil {
				return "", err
			}
			if !done {
				return j.groupInto(ctx, key, an)
			}
			// Closed since it was cached; forget it so this path converges
			// with the open-only remote search below.
			delete(j.known, al)
		}
		hit, err := j.searchOne(ctx, fmt.Sprintf(`project = %s AND labels = %q AND statusCategory != Done ORDER BY created DESC`, j.projectKey, al))
		if err != nil {
			return "", err
		}
		if hit != nil {
			key := IssueKey(hit.Key)
			j.known[al] = key
			return j.groupInto(ctx, key, an)
		}
	}

	key, err := j.create(ctx, a, an)
	if err != nil {
		return "", err
	}
	j.known[j.label(a)] = key
	if al := j.alertLabel(a); al != "" {
		j.known[al] = key
	}
	return key, nil
}

type searchHit struct {
	Key            string
	StatusCategory string
}

func (h *searchHit) done() bool { return h.StatusCategory == "done" }

// searchOne returns the most recent matching issue, or nil when none match.
func (j *JiraClient) searchOne(ctx context.Context, jql string) (*searchHit, error) {
	// /rest/api/3/search was removed by Atlassian in 2025; its /search/jql
	// replacement returns bare issue IDs unless fields are requested. status
	// is requested too so we can tell a Done match from an open one without
	// a second round-trip; ORDER BY + maxResults=1 pins it to the most
	// recently touched issue.
	resp, err := j.do(ctx, http.MethodGet, "/rest/api/3/search/jql?fields=key,status&maxResults=1&jql="+url.QueryEscape(jql), nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	// A non-2xx search response (auth failure, rate-limit, server error) returns
	// a JSON error body that decodes fine with Issues == nil, which would
	// incorrectly pass the dedup check and create a duplicate issue. Fail-safe:
	// surface the error instead.
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("jira search: status %d", resp.StatusCode)
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
		return nil, err
	}
	if len(search.Issues) == 0 {
		return nil, nil
	}
	return &searchHit{Key: search.Issues[0].Key, StatusCategory: search.Issues[0].Fields.Status.StatusCategory.Key}, nil
}

// isDone reads the issue status directly by key — unlike a JQL search this is
// strongly consistent, so it is safe immediately after a create.
func (j *JiraClient) isDone(ctx context.Context, key IssueKey) (bool, error) {
	resp, err := j.do(ctx, http.MethodGet, fmt.Sprintf("/rest/api/3/issue/%s?fields=status", key), nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return false, fmt.Errorf("jira issue status: status %d", resp.StatusCode)
	}
	var out struct {
		Fields struct {
			Status struct {
				StatusCategory struct {
					Key string `json:"key"`
				} `json:"statusCategory"`
			} `json:"status"`
		} `json:"fields"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return false, err
	}
	return out.Fields.Status.StatusCategory.Key == "done", nil
}

func (j *JiraClient) updateIssue(ctx context.Context, key IssueKey, an Analysis, done bool) (IssueKey, error) {
	text := an.RootCause
	if done {
		if err := j.reopen(ctx, key); err != nil {
			return "", err
		}
		text = "🔁 Alert re-fired after this ticket was closed; reopening.\n\n" + text
	}
	return j.commentOn(ctx, key, text)
}

// IsOpen reports whether an issue is still actionable. It reads the issue
// directly by key, which is strongly consistent, so a human closing a ticket
// takes effect on the very next refire.
func (j *JiraClient) IsOpen(ctx context.Context, key IssueKey) (bool, error) {
	done, err := j.isDone(ctx, key)
	if err != nil {
		return false, err
	}
	return !done, nil
}

// NoteRefire records a repeat notification that was deliberately not
// investigated, so the skip is auditable from the ticket alone.
func (j *JiraClient) NoteRefire(ctx context.Context, key IssueKey, count int) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	_, err := j.commentOn(ctx, key, fmt.Sprintf(
		"🔁 Still firing — repeat notification %d. Investigation skipped: the alert has not resolved since the analysis above, so re-running it would restate the same root cause.", count))
	return err
}

func (j *JiraClient) groupInto(ctx context.Context, key IssueKey, an Analysis) (IssueKey, error) {
	return j.commentOn(ctx, key, "🔗 Same alert re-fired with a new fingerprint (different labelset); grouping into this open ticket.\n\n"+an.RootCause)
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
	labels := []string{j.label(a), "ai-sre"}
	if al := j.alertLabel(a); al != "" {
		labels = append(labels, al)
	}
	create := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": j.projectKey},
			"issuetype":   map[string]string{"name": j.issueType},
			"summary":     "AI-SRE: " + a.Name(),
			"description": adfDoc(an.RootCause),
			"labels":      labels,
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
