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
	hc         *http.Client
}

func NewJiraClient(baseURL, email, token, projectKey string, hc *http.Client) *JiraClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &JiraClient{baseURL: baseURL, email: email, token: token, projectKey: projectKey, hc: hc}
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

// Upsert creates a new issue for the alert, or comments on the existing open
// issue that carries the fingerprint dedup label.
func (j *JiraClient) Upsert(ctx context.Context, a Alert, an Analysis) (IssueKey, error) {
	jql := fmt.Sprintf(`project = %s AND labels = "%s" AND statusCategory != Done`, j.projectKey, j.label(a))
	// /rest/api/3/search was removed by Atlassian in 2025; its /search/jql
	// replacement returns bare issue IDs unless fields are requested.
	resp, err := j.do(ctx, http.MethodGet, "/rest/api/3/search/jql?fields=key&jql="+url.QueryEscape(jql), nil)
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
			Key string `json:"key"`
		} `json:"issues"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&search); err != nil {
		resp.Body.Close()
		return "", err
	}
	resp.Body.Close()

	if len(search.Issues) > 0 {
		key := search.Issues[0].Key
		cResp, err := j.do(ctx, http.MethodPost, fmt.Sprintf("/rest/api/3/issue/%s/comment", key), adf(an.RootCause))
		if err != nil {
			return "", err
		}
		defer cResp.Body.Close()
		if cResp.StatusCode >= 300 {
			return "", fmt.Errorf("jira comment: status %d", cResp.StatusCode)
		}
		return IssueKey(key), nil
	}

	create := map[string]any{
		"fields": map[string]any{
			"project":     map[string]string{"key": j.projectKey},
			"issuetype":   map[string]string{"name": "Bug"},
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
