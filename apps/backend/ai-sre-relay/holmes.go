package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type HolmesClient struct {
	baseURL string
	model   string
	hc      *http.Client
}

func NewHolmesClient(baseURL, model string, hc *http.Client) *HolmesClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &HolmesClient{baseURL: baseURL, model: model, hc: hc}
}

// chatRequest matches Holmes 0.34.0 /api/chat — the only investigation
// endpoint the server exposes (there is no /api/investigate). Only `ask`
// is required; model must name a key in Holmes' modelList.
type chatRequest struct {
	Ask    string `json:"ask"`
	Model  string `json:"model,omitempty"`
	Stream bool   `json:"stream"`
}

type chatResponse struct {
	Analysis  string         `json:"analysis"`
	ToolCalls []chatToolCall `json:"tool_calls"`
}

// chatToolCall is Holmes' ToolCallResult. result is its StructuredToolResult,
// whose data is usually the tool's text output but may be any JSON value.
type chatToolCall struct {
	ID          string `json:"tool_call_id"`
	Name        string `json:"tool_name"`
	Description string `json:"description"`
	Result      struct {
		Status string          `json:"status"`
		Data   json.RawMessage `json:"data"`
	} `json:"result"`
}

// toolEvidence keeps the reads that succeeded and returned something, which
// are the only ones a remediation can cite. Output is bounded per read and
// the number of reads is capped; see maxEvidenceCalls.
func toolEvidence(calls []chatToolCall) []ToolEvidence {
	var out []ToolEvidence
	for _, c := range calls {
		if len(out) == maxEvidenceCalls {
			break
		}
		if c.ID == "" || c.Result.Status != "success" {
			continue
		}
		var data string
		if err := json.Unmarshal(c.Result.Data, &data); err != nil {
			data = string(c.Result.Data)
		}
		if data = strings.TrimSpace(data); data == "" || data == "null" {
			continue
		}
		out = append(out, ToolEvidence{
			ID: c.ID, Tool: c.Name, Description: c.Description,
			Output: truncateUTF8(data, maxEvidenceOutputBytes),
		})
	}
	return out
}

// buildAsk flattens the alert into a single investigation prompt, since
// /api/chat takes free text rather than a structured subject.
func buildAsk(a Alert) string {
	var b strings.Builder
	b.WriteString("Investigate this firing Prometheus alert and identify the root cause.\n")
	fmt.Fprintf(&b, "Alert: %s\n", a.Name())
	fmt.Fprintf(&b, "Severity: %s\n", a.Severity())
	fmt.Fprintf(&b, "Namespace: %s\n", a.Namespace())
	if pod := a.Labels["pod"]; pod != "" {
		fmt.Fprintf(&b, "Pod: %s\n", pod)
	}
	if kind := a.Labels["kind"]; kind != "" {
		fmt.Fprintf(&b, "Kind: %s\n", kind)
	}
	if desc := a.Annotations["description"]; desc != "" {
		fmt.Fprintf(&b, "Description: %s\n", desc)
	}
	fmt.Fprintf(&b, "Fingerprint: %s\n", a.Fingerprint)
	return b.String()
}

// investigateAttempts bounds re-asks when the model returns unusable output
// (empty, tool-call markup, all conversational filler). One retry usually
// clears a bad sample; more would burn the per-alert deadline.
const investigateAttempts = 2

// Investigate runs the Holmes chat and sanitizes the result. Unusable output
// is re-asked up to investigateAttempts times and then surfaced as an error,
// so downstream outputs (Jira, Discord) never publish a raw model artifact.
func (c *HolmesClient) Investigate(ctx context.Context, a Alert) (Analysis, error) {
	var lastErr error
	for range investigateAttempts {
		cr, err := c.chat(ctx, a)
		if err != nil {
			return Analysis{}, err
		}
		clean, serr := sanitizeAnalysis(cr.Analysis)
		if serr == nil {
			return Analysis{RootCause: clean, Evidence: toolEvidence(cr.ToolCalls)}, nil
		}
		lastErr = serr
	}
	return Analysis{}, fmt.Errorf("holmes: unusable analysis after %d attempts: %w", investigateAttempts, lastErr)
}

func (c *HolmesClient) chat(ctx context.Context, a Alert) (chatResponse, error) {
	body, _ := json.Marshal(chatRequest{Ask: buildAsk(a), Model: c.model, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return chatResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return chatResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return chatResponse{}, fmt.Errorf("holmes: status %d", resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return chatResponse{}, fmt.Errorf("holmes: decode: %w", err)
	}
	return cr, nil
}
