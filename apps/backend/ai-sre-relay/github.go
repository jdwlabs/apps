package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strings"
)

type GitHubClient struct {
	apiBase string // https://api.github.com (override in tests)
	token   string
	// allowed bounds which repositories a patch may target. Repo, file path,
	// and file body all originate from a model prompted with alert
	// annotations and Holmes output — untrusted text — so without this the
	// only limit on where the relay writes is the token's own scope.
	allowed map[string]struct{}
	hc      *http.Client
}

func NewGitHubClient(apiBase, token string, allowedRepos []string, hc *http.Client) *GitHubClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	allowed := make(map[string]struct{}, len(allowedRepos))
	for _, r := range allowedRepos {
		if r = strings.TrimSpace(r); r != "" {
			allowed[r] = struct{}{}
		}
	}
	return &GitHubClient{apiBase: apiBase, token: token, allowed: allowed, hc: hc}
}

// checkRepo fails closed: an empty allowlist disables the PR arm entirely
// rather than falling back to whatever the model named.
func (g *GitHubClient) checkRepo(repo string) error {
	if len(g.allowed) == 0 {
		return fmt.Errorf("github: no repo allowlist configured; refusing to open a PR")
	}
	if _, ok := g.allowed[repo]; !ok {
		return fmt.Errorf("github: repo %q is not allowlisted", repo)
	}
	return nil
}

// safeFilePath rejects a model-supplied path that is absolute or climbs out of
// the repository root, and escapes each segment so it cannot inject further
// path or query structure into the Contents API URL.
func safeFilePath(p string) (string, error) {
	if strings.TrimSpace(p) == "" {
		return "", fmt.Errorf("github: empty file path")
	}
	if strings.HasPrefix(p, "/") {
		return "", fmt.Errorf("github: absolute file path %q", p)
	}
	clean := path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", fmt.Errorf("github: file path %q escapes the repository root", p)
	}
	segs := strings.Split(clean, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return strings.Join(segs, "/"), nil
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
	if err := g.checkRepo(p.Repo); err != nil {
		return "", err
	}
	filePath, err := safeFilePath(p.FilePath)
	if err != nil {
		return "", err
	}
	repoPath := "/repos/" + p.Repo

	// 1. base ref SHA
	refResp, err := g.req(ctx, http.MethodGet, repoPath+"/git/ref/heads/main", nil)
	if err != nil {
		return "", err
	}
	// The status is what distinguishes "repo does not exist" from "token
	// cannot see it" from a genuine empty ref. Decoding a non-2xx body yields
	// an empty SHA, which previously surfaced as a generic error that hid the
	// real cause.
	if refResp.StatusCode != http.StatusOK {
		refResp.Body.Close()
		return "", fmt.Errorf("github base ref: status %d (repo %s)", refResp.StatusCode, p.Repo)
	}
	var ref struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	_ = json.NewDecoder(refResp.Body).Decode(&ref)
	refResp.Body.Close()
	if ref.Object.SHA == "" {
		return "", fmt.Errorf("github: empty base sha (repo %s)", p.Repo)
	}

	// 2. create branch; 422 means the branch already exists and is fine (idempotent
	// re-run of the same patch), other non-2xx statuses are real failures.
	brResp, err := g.req(ctx, http.MethodPost, repoPath+"/git/refs", map[string]string{
		"ref": "refs/heads/" + p.Branch, "sha": ref.Object.SHA,
	})
	if err != nil {
		return "", err
	}
	if brResp.StatusCode >= 300 && brResp.StatusCode != http.StatusUnprocessableEntity {
		brResp.Body.Close()
		return "", fmt.Errorf("github create branch: status %d", brResp.StatusCode)
	}
	brResp.Body.Close()

	// 3. existing file SHA (needed to update; absent => new file)
	var existingSHA string
	fResp, err := g.req(ctx, http.MethodGet, repoPath+"/contents/"+filePath+"?ref="+url.QueryEscape(p.Branch), nil)
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

	put := map[string]any{
		"message": fmt.Sprintf("fix(ai-sre): %s (%s)", p.Rationale, issue),
		"content": base64.StdEncoding.EncodeToString([]byte(p.NewContent)),
		"branch":  p.Branch,
	}
	if existingSHA != "" {
		put["sha"] = existingSHA
	}
	pResp, err := g.req(ctx, http.MethodPut, repoPath+"/contents/"+filePath, put)
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
