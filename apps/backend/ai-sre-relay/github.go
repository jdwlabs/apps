package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"
)

const (
	// baseBranch is the only branch the arm reads a SHA from and targets with a
	// PR. It is never written to; the patch always lands on a derived branch.
	baseBranch = "main"

	// branchPrefix namespaces every relay-authored branch, and doubles as the
	// prefix the org's branch-naming rule requires.
	branchPrefix = "fix/ai-sre/"

	// maxPatchBytes caps a model-authored file body. A remediation is a small
	// GitOps edit; anything larger is a runaway generation rather than a fix.
	maxPatchBytes = 64 << 10

	// maxRationaleRunes bounds the model text interpolated into a commit
	// message and PR title, which are single-line by contract.
	maxRationaleRunes = 120

	// maxRationaleBodyRunes bounds the same text in the PR body, where more
	// context is useful but unbounded model output still is not.
	maxRationaleBodyRunes = 2000
)

// tagLike matches an HTML/XML-ish token, which is also the shape of the
// tool-call markup a model emits as text. A non-space is required directly
// after '<' so an ordinary comparison such as "memory < 512Mi" is left alone.
var tagLike = regexp.MustCompile(`<[^\s<>][^<>]*>`)

// remediationBranch derives the target branch from the patch rather than taking
// the model's suggestion. Two properties matter: it can never name a branch a
// human relies on — least of all baseBranch — and it is deterministic, so
// re-running the same patch reuses one branch instead of littering the repo.
func remediationBranch(p Patch, issue IssueKey) string {
	if slug := branchSlug(string(issue)); slug != "" {
		return branchPrefix + slug
	}
	// No issue key (the Jira arm failed): fall back to the patch's own identity
	// so dedup still holds.
	sum := sha256.Sum256([]byte(p.Repo + "\x00" + p.FilePath + "\x00" + p.NewContent))
	return branchPrefix + hex.EncodeToString(sum[:6])
}

// branchSlug reduces an issue key to lowercase alphanumerics and dashes, so the
// result is a valid git ref with no room for path or option injection.
func branchSlug(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-', r == '_', r == '.':
			b.WriteRune('-')
		}
	}
	return strings.Trim(b.String(), "-")
}

// safeSummary flattens model-authored text for embedding in a commit message,
// PR title or PR body. The rationale is derived from alert annotations and
// Holmes output, so it is untrusted: markup is dropped, newlines and control
// characters collapse to spaces, and the result is length-capped so it cannot
// reshape the message that carries it.
func safeSummary(s string, maxRunes int) string {
	s = tagLike.ReplaceAllString(s, " ")
	var b strings.Builder
	for _, r := range s {
		if unicode.IsControl(r) {
			b.WriteRune(' ')
			continue
		}
		b.WriteRune(r)
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if out == "" {
		return "no rationale provided"
	}
	if rs := []rune(out); len(rs) > maxRunes {
		return strings.TrimSpace(string(rs[:maxRunes])) + "…"
	}
	return out
}

// GitHubTokenSource supplies the bearer token GitHubClient presents on every
// request. It is consulted per-request rather than fixed at construction, so
// an installation token can be minted fresh as it nears expiry instead of
// being read once at startup and going stale under the client's feet.
type GitHubTokenSource interface {
	Token(ctx context.Context) (string, error)
}

// StaticGitHubToken is a GitHubTokenSource that always returns the same
// value. It is what the tests use, and the fallback for running against a
// plain PAT when no GitHub App identity is configured.
type StaticGitHubToken string

func (s StaticGitHubToken) Token(context.Context) (string, error) { return string(s), nil }

type GitHubClient struct {
	apiBase string // https://api.github.com (override in tests)
	tokens  GitHubTokenSource
	// allowed bounds which repositories a patch may target. Repo, file path,
	// and file body all originate from a model prompted with alert
	// annotations and Holmes output — untrusted text — so without this the
	// only limit on where the relay writes is the token's own scope.
	allowed map[string]struct{}
	// allowedPaths are the glob patterns (path.Match syntax, one segment per
	// star) a patched file must match. The model has no view of the
	// repository tree and invents directories that look like GitOps but that
	// no Application reads; the patterns pin writes to the paths that are.
	allowedPaths []string
	hc           *http.Client
}

func NewGitHubClient(apiBase string, tokens GitHubTokenSource, allowedRepos, allowedPaths []string, hc *http.Client) *GitHubClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	allowed := make(map[string]struct{}, len(allowedRepos))
	for _, r := range allowedRepos {
		if r = strings.TrimSpace(r); r != "" {
			allowed[r] = struct{}{}
		}
	}
	var paths []string
	for _, p := range allowedPaths {
		if p = strings.TrimSpace(p); p != "" {
			paths = append(paths, p)
		}
	}
	return &GitHubClient{apiBase: apiBase, tokens: tokens, allowed: allowed, allowedPaths: paths, hc: hc}
}

// ErrRepoNotAllowed marks the allowlist refusal specifically, so callers can
// report a discarded remediation as its own outcome instead of folding it into
// the generic "GitHub call failed" bucket where it reads as transient.
var ErrRepoNotAllowed = errors.New("github: repo is not allowlisted")

// ErrPathNotAllowed marks a proposal whose file is outside the paths the
// GitOps controller reads, or that does not exist on the base branch. Every
// unusable remediation so far was one of these: a plausible-looking manifest
// under a top-level directory nothing watches, so it could never take effect.
var ErrPathNotAllowed = errors.New("github: file path is not a watched, existing manifest")

// ErrBranchExists marks a refusal to write onto a remediation branch that a
// previous run already created. Whatever is on that branch is under human
// review; a second proposal for the same ticket must not land on top of it.
var ErrBranchExists = errors.New("github: remediation branch already exists")

// allowedList renders the allowed set for an operator-facing message. Sorted,
// because a map's order would make the same refusal look different each time.
func (g *GitHubClient) allowedList() string {
	names := make([]string, 0, len(g.allowed))
	for r := range g.allowed {
		names = append(names, r)
	}
	sort.Strings(names)
	return strings.Join(names, ",")
}

// checkRepo fails closed: an empty allowlist disables the PR arm entirely
// rather than falling back to whatever the model named. The refusal names both
// sides — what was proposed and what was permitted — because the difference is
// the whole diagnosis, and without it a systematic mismatch is indistinguishable
// from a pipeline that simply never had work to do.
func (g *GitHubClient) checkRepo(repo string) error {
	if len(g.allowed) == 0 {
		return fmt.Errorf("github: no repo allowlist configured; refusing to open a PR")
	}
	if _, ok := g.allowed[repo]; !ok {
		return fmt.Errorf("%w: proposed %q, allowed [%s]", ErrRepoNotAllowed, repo, g.allowedList())
	}
	return nil
}

// checkPath fails closed like checkRepo: with no patterns configured the arm
// writes nowhere. The path is matched after cleaning so "./tenants/x" and
// "tenants/x" are one case, and the refusal names the patterns because the
// gap between what was proposed and what is watched is the diagnosis.
func (g *GitHubClient) checkPath(clean string) error {
	if len(g.allowedPaths) == 0 {
		return fmt.Errorf("github: no path allowlist configured; refusing to open a PR")
	}
	for _, pat := range g.allowedPaths {
		if ok, err := filepath.Match(pat, clean); err == nil && ok {
			return nil
		}
	}
	return fmt.Errorf("%w: proposed %q, allowed [%s]", ErrPathNotAllowed, clean, strings.Join(g.allowedPaths, ","))
}

// safeFilePath rejects a model-supplied path that is absolute or climbs out of
// the repository root, and returns the cleaned path alongside a URL-safe form
// whose segments are escaped so they cannot inject further path or query
// structure into the Contents API URL.
func safeFilePath(p string) (clean, escaped string, err error) {
	if strings.TrimSpace(p) == "" {
		return "", "", fmt.Errorf("github: empty file path")
	}
	if strings.HasPrefix(p, "/") {
		return "", "", fmt.Errorf("github: absolute file path %q", p)
	}
	clean = path.Clean(p)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", "", fmt.Errorf("github: file path %q escapes the repository root", p)
	}
	segs := strings.Split(clean, "/")
	for i, s := range segs {
		segs[i] = url.PathEscape(s)
	}
	return clean, strings.Join(segs, "/"), nil
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
	token, err := g.tokens.Token(ctx)
	if err != nil {
		return nil, fmt.Errorf("github: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	return g.hc.Do(req)
}

// OpenPR creates a derived branch off baseBranch, writes the single patched
// file to it, and opens a PR for human review. It never pushes to baseBranch,
// never force-updates a ref, never writes to a branch it did not just create,
// and never merges: every write is confined to the branch remediationBranch
// derives, and the merge decision stays with a human. The arm edits files
// that already exist on baseBranch and matches the path allowlist; it never
// creates one, because a new file is exactly what a model with no view of
// the tree produces when it guesses.
func (g *GitHubClient) OpenPR(ctx context.Context, p Patch, issue IssueKey) (PRLink, error) {
	if err := g.checkRepo(p.Repo); err != nil {
		return "", err
	}
	cleanPath, filePath, err := safeFilePath(p.FilePath)
	if err != nil {
		return "", err
	}
	if err := g.checkPath(cleanPath); err != nil {
		return "", err
	}
	if n := len(p.NewContent); n == 0 || n > maxPatchBytes {
		return "", fmt.Errorf("github: patch body is %d bytes, want 1..%d", n, maxPatchBytes)
	}
	branch := remediationBranch(p, issue)
	// An invariant, not a possibility. Asserted rather than assumed so that
	// reintroducing a caller- or model-supplied branch cannot silently turn this
	// back into a write primitive against a protected branch.
	if branch == baseBranch || !strings.HasPrefix(branch, branchPrefix) {
		return "", fmt.Errorf("github: refusing to write to branch %q", branch)
	}
	summary := safeSummary(p.Rationale, maxRationaleRunes)
	repoPath := "/repos/" + p.Repo

	// 1. base ref SHA
	refResp, err := g.req(ctx, http.MethodGet, repoPath+"/git/ref/heads/"+baseBranch, nil)
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

	// 2. the file must already exist on the base branch. Read before the
	// branch is created so a refusal leaves nothing behind. The blob SHA is
	// also what the Contents API needs to update rather than create.
	fResp, err := g.req(ctx, http.MethodGet, repoPath+"/contents/"+filePath+"?ref="+baseBranch, nil)
	if err != nil {
		return "", err
	}
	var existing struct {
		SHA  string `json:"sha"`
		Type string `json:"type"`
	}
	if fResp.StatusCode == http.StatusOK {
		_ = json.NewDecoder(fResp.Body).Decode(&existing)
	}
	fResp.Body.Close()
	if fResp.StatusCode != http.StatusOK || existing.SHA == "" || (existing.Type != "" && existing.Type != "file") {
		return "", fmt.Errorf("%w: %q is not an existing file on %s (status %d)", ErrPathNotAllowed, cleanPath, baseBranch, fResp.StatusCode)
	}

	// 3. create branch. 422 means it already exists: a previous run's PR is
	// (or was) open from it, and this proposal is a separate, unrelated
	// patch that must not be appended to it. Refusing here is what keeps one
	// PR to one commit.
	brResp, err := g.req(ctx, http.MethodPost, repoPath+"/git/refs", map[string]string{
		"ref": "refs/heads/" + branch, "sha": ref.Object.SHA,
	})
	if err != nil {
		return "", err
	}
	brResp.Body.Close()
	if brResp.StatusCode == http.StatusUnprocessableEntity {
		return "", fmt.Errorf("%w: %s (repo %s)", ErrBranchExists, branch, p.Repo)
	}
	if brResp.StatusCode >= 300 {
		return "", fmt.Errorf("github create branch: status %d", brResp.StatusCode)
	}

	put := map[string]any{
		"message": fmt.Sprintf("fix(ai-sre): %s (%s)", summary, issue),
		"content": base64.StdEncoding.EncodeToString([]byte(p.NewContent)),
		"branch":  branch,
		"sha":     existing.SHA,
	}
	pResp, err := g.req(ctx, http.MethodPut, repoPath+"/contents/"+filePath, put)
	if err != nil {
		return "", err
	}
	pResp.Body.Close()
	if pResp.StatusCode >= 300 {
		return "", fmt.Errorf("github put contents: status %d", pResp.StatusCode)
	}

	// 4. open PR
	prBody := fmt.Sprintf(
		"Automated AI-SRE remediation for %s.\n\nSingle file changed: `%s`\n\n%s\n\n**Human review required — do not auto-merge.**",
		issue, filePath, safeSummary(p.Rationale, maxRationaleBodyRunes))
	prResp, err := g.req(ctx, http.MethodPost, repoPath+"/pulls", map[string]string{
		"title": fmt.Sprintf("fix(ai-sre): %s [%s]", summary, issue),
		"head":  branch,
		"base":  baseBranch,
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
