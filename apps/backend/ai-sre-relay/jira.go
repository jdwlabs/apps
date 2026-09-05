package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// maxTrackedDedupKeys bounds known the same way pipeline.go's maxTrackedFirings
// bounds active. Beyond it, new dedup keys are simply not remembered
// in-process and fall back to the label search below on the next Upsert —
// the memory ceiling matters more than caching every possible dedup key.
const maxTrackedDedupKeys = 1024

// defaultDoneStatus is the status name a close aims for before falling back to
// any Done-category transition.
const defaultDoneStatus = "Done"

type JiraClient struct {
	baseURL    string
	email      string
	token      string
	projectKey string
	issueType  string
	// doneStatus names the status a close aims for. Workflows commonly expose
	// several Done-category transitions — Done, Won't Do, Duplicate — and
	// picking whichever comes first mislabels why the work ended.
	doneStatus string
	hc         *http.Client

	// mu serializes upserts and guards known. Jira offers no unique-constraint
	// primitive and its JQL index is eventually consistent — a search issued
	// right after a create can miss the fresh issue — so the correctness of
	// the search-then-create sequence depends on upserts running one at a time
	// with their results remembered in-process.
	mu    sync.Mutex
	known map[string]IssueKey // dedup label -> issue key created or matched by this process
}

type jiraOption func(*JiraClient)

func withDoneStatus(name string) jiraOption {
	return func(j *JiraClient) {
		if name != "" {
			j.doneStatus = name
		}
	}
}

func NewJiraClient(baseURL, email, token, projectKey, issueType string, hc *http.Client, opts ...jiraOption) *JiraClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	j := &JiraClient{
		baseURL: baseURL, email: email, token: token,
		projectKey: projectKey, issueType: issueType, doneStatus: defaultDoneStatus, hc: hc,
		known: map[string]IssueKey{},
	}
	for _, opt := range opts {
		opt(j)
	}
	return j
}

func (j *JiraClient) label(a Alert) string { return "amfp-" + a.Fingerprint }

// rememberKnown records a dedup key -> issue mapping, bounded like
// Pipeline.remember: once the cap is hit, new keys are simply not tracked.
// Callers always hold mu already (Upsert holds it for its whole body), so
// this does not lock itself.
func (j *JiraClient) rememberKnown(key string, issue IssueKey) {
	if _, known := j.known[key]; !known && len(j.known) >= maxTrackedDedupKeys {
		return
	}
	j.known[key] = issue
}

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
	hit, err := j.fingerprintHit(ctx, a)
	if err != nil {
		return "", err
	}
	if hit != nil {
		key := IssueKey(hit.Key)
		j.rememberKnown(j.label(a), key)
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
			j.rememberKnown(al, key)
			return j.groupInto(ctx, key, an)
		}
	}

	key, err := j.create(ctx, a, an)
	if err != nil {
		return "", err
	}
	j.rememberKnown(j.label(a), key)
	if al := j.alertLabel(a); al != "" {
		j.rememberKnown(al, key)
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
	// a second round-trip; ORDER BY pins the answer to the most recently
	// created issue. Two are fetched to use one: a second match means the
	// dedup label is on more than one ticket, which is worth saying out loud
	// because from then on the choice between them is just ordering.
	resp, err := j.do(ctx, http.MethodGet, "/rest/api/3/search/jql?fields=key,status&maxResults=2&jql="+url.QueryEscape(jql), nil)
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
	if len(search.Issues) > 1 {
		slog.Warn("jira search matched more than one issue; using the most recent",
			"jql", jql, "using", search.Issues[0].Key, "also", search.Issues[1].Key)
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

// NoteResolved records that Alertmanager reported the alert resolved, so the
// pending close is visible on the ticket before it happens.
func (j *JiraClient) NoteResolved(ctx context.Context, key IssueKey, resolvedAt time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	_, err := j.commentOn(ctx, key, fmt.Sprintf(
		"✅ Alertmanager reported this alert resolved at %s. This ticket closes automatically if the alert stays resolved; a re-fire before then cancels the close.",
		stamp(resolvedAt)))
	return err
}

// ErrClosedWithoutNote reports a ticket that reached Done but whose closing
// comment did not post. The distinction matters to the caller: the close is
// finished and must not be retried, but nothing on the ticket says why it
// closed.
var ErrClosedWithoutNote = errors.New("ticket closed without its closing comment")

// Close transitions the issue to Done after its alert has stayed resolved,
// leaving the resolve time on the ticket so the close is auditable without
// cross-referencing Alertmanager.
//
// The transition goes first and the comment second, because the caller retries
// a failed close on every sweep: commenting first turns a workflow that cannot
// reach Done into a ticket collecting one closing comment per sweep, forever.
func (j *JiraClient) Close(ctx context.Context, key IssueKey, resolvedAt time.Time) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	if err := j.transition(ctx, key, true); err != nil {
		return err
	}
	if _, err := j.commentOn(ctx, key, fmt.Sprintf(
		"🟢 Closed automatically: the alert resolved at %s and stayed resolved for the grace period. If it fires again this ticket is reopened.",
		stamp(resolvedAt))); err != nil {
		return fmt.Errorf("%w: %s: %w", ErrClosedWithoutNote, key, err)
	}
	return nil
}

// Reopen returns a ticket to an open status after a close raced a re-fire, so
// a ticket is never left Done while its alert is firing.
func (j *JiraClient) Reopen(ctx context.Context, key IssueKey) error {
	j.mu.Lock()
	defer j.mu.Unlock()
	// The close this undoes may never have landed, or a human may have
	// reopened it first. Moving an already-open ticket would push it to an
	// arbitrary status and comment about a reopen that did not happen.
	done, err := j.isDone(ctx, key)
	if err != nil {
		return err
	}
	if !done {
		return nil
	}
	if err := j.transition(ctx, key, false); err != nil {
		return err
	}
	_, cerr := j.commentOn(ctx, key,
		"🔁 Alert re-fired while this ticket was being closed automatically; reopening.")
	return cerr
}

// fingerprintHit is the one fingerprint-label search both the upsert and the
// resolve path use. It deliberately does not filter Done: the upsert has to
// see a closed ticket in order to reopen it. Filtering on one side only would
// let the two paths select different issues whenever a fingerprint has both a
// newer Done ticket and an older open one.
func (j *JiraClient) fingerprintHit(ctx context.Context, a Alert) (*searchHit, error) {
	return j.searchOne(ctx, fmt.Sprintf(`project = %s AND labels = %q ORDER BY created DESC`,
		j.projectKey, j.label(a)))
}

// FindOpenByFingerprint recovers the open ticket already filed for an alert's
// fingerprint. The relay's own tracking is in-process, so this is how a
// resolve arriving after a restart finds the ticket it belongs to. A Done
// match reports nothing: there is nothing left to close.
func (j *JiraClient) FindOpenByFingerprint(ctx context.Context, a Alert) (IssueKey, error) {
	if a.Fingerprint == "" {
		return "", nil
	}
	j.mu.Lock()
	defer j.mu.Unlock()
	hit, err := j.fingerprintHit(ctx, a)
	if err != nil || hit == nil || hit.done() {
		return "", err
	}
	key := IssueKey(hit.Key)
	j.rememberKnown(j.label(a), key)
	return key, nil
}

// stamp renders a resolve time for a ticket comment. UTC, because the relay,
// Alertmanager and the reader are rarely in the same zone.
func stamp(t time.Time) string { return t.UTC().Format(time.RFC3339) }

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

// reopen moves a Done issue back to an open status so a refiring alert updates
// the original ticket instead of spawning a duplicate.
func (j *JiraClient) reopen(ctx context.Context, key IssueKey) error {
	return j.transition(ctx, key, false)
}

// transition moves an issue into or out of the Done status category. It picks
// the first available transition whose target status sits on the wanted side;
// workflows name these differently ("Reopen", "Backlog", "Ship It", ...) and
// their ids differ per project, so the target status category is what matters.
func (j *JiraClient) transition(ctx context.Context, key IssueKey, toDone bool) error {
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
				Name           string `json:"name"`
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
	// "Won't Do" and "Duplicate" usually sit in the Done category too, and
	// closing an incident as a duplicate says something the relay does not
	// mean. The configured status name wins; the category is the fallback for
	// workflows that name it something else entirely.
	if toDone {
		for _, tr := range t.Transitions {
			if strings.EqualFold(tr.To.Name, j.doneStatus) && tr.To.StatusCategory.Key == "done" {
				transitionID = tr.ID
				break
			}
		}
	}
	for _, tr := range t.Transitions {
		if transitionID != "" {
			break
		}
		if (tr.To.StatusCategory.Key == "done") == toDone {
			transitionID = tr.ID
		}
	}
	if transitionID == "" {
		want := "non-Done"
		if toDone {
			want = "Done"
		}
		return fmt.Errorf("jira transition %s: no %s transition available", key, want)
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
