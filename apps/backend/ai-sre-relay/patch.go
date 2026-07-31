package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type PatchGenerator struct {
	baseURL       string
	apiKey        string
	model         string
	minConfidence float64
	hc            *http.Client
}

func NewPatchGenerator(baseURL, apiKey, model string, minConfidence float64, hc *http.Client) *PatchGenerator {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &PatchGenerator{baseURL: baseURL, apiKey: apiKey, model: model, minConfidence: minConfidence, hc: hc}
}

const patchSystemPrompt = `You are an SRE assistant. Given a root-cause analysis, propose AT MOST ONE single-file GitOps change that fixes it. Respond with ONLY a JSON object, no prose, matching:
{"repo":"owner/name","file_path":"...","new_content":"<full file>","rationale":"...","confidence":0.0-1.0}
If no safe single-file change exists, respond with exactly: {"confidence":0}`

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
			{Role: "system", Content: patchSystemPrompt},
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
	if p.Confidence < g.minConfidence || strings.TrimSpace(p.Repo) == "" || strings.TrimSpace(p.FilePath) == "" || strings.TrimSpace(p.NewContent) == "" {
		return nil, nil
	}
	return &p, nil
}
