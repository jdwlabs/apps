package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
)

type PatchGenerator struct {
	baseURL       string
	apiKey        string
	model         string
	minConfidence float64
	// targets are the repositories a remediation may land in, in the order the
	// prompt presents them. Which repository an alert remediates is a
	// deployment fact the relay already holds, so the single-target case is
	// injected rather than asked for: a model told to fill in "owner/name" from
	// alert text answers with a plausible invention, and the boundary check
	// then throws the whole proposal away.
	targets []string
	hc      *http.Client
	log     *slog.Logger
}

func NewPatchGenerator(baseURL, apiKey, model string, minConfidence float64, targets []string, log *slog.Logger, hc *http.Client) *PatchGenerator {
	if hc == nil {
		hc = http.DefaultClient
	}
	if log == nil {
		log = slog.Default()
	}
	return &PatchGenerator{
		baseURL: baseURL, apiKey: apiKey, model: model,
		minConfidence: minConfidence, targets: slices.Clone(targets), log: log, hc: hc,
	}
}

const (
	patchPromptHead = `You are an SRE assistant. Given a root-cause analysis, propose AT MOST ONE single-file GitOps change that fixes it. Respond with ONLY a JSON object, no prose, matching:
`
	// patchPromptLayout tells the model what the repository actually looks
	// like. Without it the model has no view of the tree and answers with a
	// layout it has seen elsewhere (manifests/, cluster/, clusters/…), which
	// ArgoCD never reads. The relay re-checks the path against the same
	// layout before writing, so this is guidance, not the boundary.
	patchPromptLayout = `
The repository is an ArgoCD GitOps tree. The ONLY files that are reconciled are:
- tenants/<tenant>/tenant.yaml — the tenant's service list (each entry is a Helm release);
- tenants/<tenant>/services/<release>/values.yaml — that release's Helm values;
- tenants/<tenant>/services/<release>/postInstall/<name>.yaml — raw manifests applied after the release.
"file_path" MUST be one of those existing files, edited in place with its full new contents. NEVER invent another layout, NEVER create a new file or a new top-level directory: a file anywhere else is not watched and changes nothing. Find the release that owns the failing resource from the tenant.yaml service names and the namespace in the analysis. If you are not certain which existing file owns the resource, respond with {"confidence":0} instead of guessing.`
	patchPromptTail = `
If no safe single-file change exists, respond with exactly: {"confidence":0}`
)

// patchSystemPrompt renders the instruction for one configured target set. The
// destination is never left to the model to infer from alert text: with a
// single target it is stated as a fact and omitted from the response contract
// entirely, and only a real choice between several is put to the model — as a
// closed enumeration, never as free text.
func patchSystemPrompt(targets []string) string {
	if len(targets) > 1 {
		return patchPromptHead +
			`{"repo":"...","file_path":"...","new_content":"<full file>","rationale":"...","confidence":0.0-1.0}` + "\n" +
			`"repo" MUST be copied verbatim from this list and may be nothing else: ` + strings.Join(targets, ", ") + ".\n" +
			`"file_path" is relative to the root of that repository.` +
			patchPromptLayout +
			patchPromptTail
	}
	// Zero targets means the PR arm is switched off, so a repository name would
	// be noise the boundary refuses anyway — the shape is the same either way.
	where := "the target repository"
	if len(targets) == 1 {
		where = "the " + targets[0] + " repository"
	}
	return patchPromptHead +
		`{"file_path":"...","new_content":"<full file>","rationale":"...","confidence":0.0-1.0}` + "\n" +
		"The change will be applied to " + where + `. Do not name a repository; "file_path" is relative to its root.` +
		patchPromptLayout +
		patchPromptTail
}

type openAIRequest struct {
	Model    string          `json:"model"`
	Messages []openAIMessage `json:"messages"`
}
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
type openAIResponse struct {
	Choices []struct {
		Message openAIMessage `json:"message"`
	} `json:"choices"`
}

// Generate returns (nil, nil) for the common "no confident patch" case; an
// error only for transport/decode failures of the LiteLLM call itself.
func (g *PatchGenerator) Generate(ctx context.Context, an Analysis) (*Patch, error) {
	reqBody, _ := json.Marshal(openAIRequest{
		Model: g.model,
		Messages: []openAIMessage{
			{Role: "system", Content: patchSystemPrompt(g.targets)},
			{Role: "user", Content: an.RootCause},
		},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, g.baseURL+"/chat/completions", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+g.apiKey)
	resp, err := g.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("litellm: status %d", resp.StatusCode)
	}
	var or openAIResponse
	if err := json.NewDecoder(resp.Body).Decode(&or); err != nil {
		return nil, fmt.Errorf("litellm: decode: %w", err)
	}
	if len(or.Choices) == 0 {
		return nil, nil
	}
	content := strings.TrimSpace(or.Choices[0].Message.Content)

	// Malformed / refusal → no patch (not an error). Only accept strict JSON.
	var p Patch
	if err := json.Unmarshal([]byte(content), &p); err != nil {
		return nil, nil
	}
	// Repo is deliberately absent from this gate: it is the relay's to set, not
	// the model's to supply.
	if p.Confidence < g.minConfidence || strings.TrimSpace(p.FilePath) == "" || strings.TrimSpace(p.NewContent) == "" {
		return nil, nil
	}
	if !g.resolveRepo(&p) {
		return nil, nil
	}
	return &p, nil
}

// resolveRepo sets the destination on a proposal and reports whether it is
// still usable. With one configured target the model's answer is overwritten
// rather than checked, so a repository it named unasked costs a log line
// instead of the entire remediation; with several, naming one outside the set
// is fatal to the proposal and says so loudly enough to alert on.
func (g *PatchGenerator) resolveRepo(p *Patch) bool {
	named := strings.TrimSpace(p.Repo)
	if len(g.targets) > 1 {
		if !slices.Contains(g.targets, named) {
			g.log.Error("discarding remediation: proposed repository is outside the configured target set",
				"proposed_repo", named, "allowed_repos", strings.Join(g.targets, ","),
				"file_path", p.FilePath)
			return false
		}
		p.Repo = named
		return true
	}
	if len(g.targets) == 1 {
		if named != "" && named != g.targets[0] {
			g.log.Warn("model named a repository it was not asked for; overriding with the configured target",
				"proposed_repo", named, "target_repo", g.targets[0], "file_path", p.FilePath)
		}
		p.Repo = g.targets[0]
		return true
	}
	// No target configured: the PR arm is off. The proposal still reaches
	// Discord, and the boundary check refuses to write it anywhere.
	p.Repo = named
	return true
}
