package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
)

type GitHubClient struct {
	apiBase string // https://api.github.com (override in tests)
	token   string
	hc      *http.Client
}

func NewGitHubClient(apiBase, token string, hc *http.Client) *GitHubClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &GitHubClient{apiBase: apiBase, token: token, hc: hc}
}

func (g *GitHubClient) req(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var buf *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		buf = bytes.NewReader(b)
	} else {
		buf = bytes.NewReader(nil)
	}
	req, err := http.NewRequestWithContext(ctx, method, g.apiBase+path, buf)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+g.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return g.hc.Do(req)
}

// OpenPR creates a branch off main, writes the single patched file, and opens a
// PR for human review. The branch name comes from the patch so callers control
// dedup (re-running the same patch lands on the same branch).
func (g *GitHubClient) OpenPR(ctx context.Context, p Patch, issue IssueKey) (PRLink, error) {
	repoPath := "/repos/" + p.Repo

	// 1. base ref SHA
	refResp, err := g.req(ctx, http.MethodGet, repoPath+"/git/ref/heads/main", nil)
	if err != nil {
		return "", err
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	_ = json.NewDecoder(refResp.Body).Decode(&ref)
	refResp.Body.Close()
	if ref.Object.SHA == "" {
		return "", fmt.Errorf("github: empty base sha")
	}

	// 2. create branch (ignore 422 already-exists)
	brResp, err := g.req(ctx, http.MethodPost, repoPath+"/git/refs", map[string]string{
		"ref": "refs/heads/" + p.Branch, "sha": ref.Object.SHA,
	})
	if err != nil {
		return "", err
	}
	brResp.Body.Close()

	// 3. existing file SHA (needed to update; absent => new file)
	var existingSHA string
	fResp, err := g.req(ctx, http.MethodGet, repoPath+"/contents/"+p.FilePath+"?ref="+p.Branch, nil)
	if err != nil {
		return "", err
	}
	if fResp.StatusCode == http.StatusOK {
		var f struct {
			SHA string `json:"sha"`
		}
		_ = json.NewDecoder(fResp.Body).Decode(&f)
		existingSHA = f.SHA
	}
	fResp.Body.Close()

	// 4. put contents
	put := map[string]any{
		"message": fmt.Sprintf("fix(ai-sre): %s (%s)", p.Rationale, issue),
		"content": base64.StdEncoding.EncodeToString([]byte(p.NewContent)),
		"branch":  p.Branch,
	}
	if existingSHA != "" {
		put["sha"] = existingSHA
	}
	pResp, err := g.req(ctx, http.MethodPut, repoPath+"/contents/"+p.FilePath, put)
	if err != nil {
		return "", err
	}
	pResp.Body.Close()
	if pResp.StatusCode >= 300 {
		return "", fmt.Errorf("github put contents: status %d", pResp.StatusCode)
	}

	// 5. open PR
	prBody := fmt.Sprintf("Automated AI-SRE remediation for %s.\n\n%s\n\n**Human review required — do not auto-merge.**", issue, p.Rationale)
	prResp, err := g.req(ctx, http.MethodPost, repoPath+"/pulls", map[string]string{
		"title": fmt.Sprintf("fix(ai-sre): %s [%s]", p.Rationale, issue),
		"head":  p.Branch,
		"base":  "main",
		"body":  prBody,
	})
	if err != nil {
		return "", err
	}
	defer prResp.Body.Close()
	if prResp.StatusCode >= 300 {
		return "", fmt.Errorf("github create pr: status %d", prResp.StatusCode)
	}
	var pr struct {
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(prResp.Body).Decode(&pr); err != nil {
		return "", err
	}
	return PRLink(pr.HTMLURL), nil
}
