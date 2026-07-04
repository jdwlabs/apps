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
	Analysis string `json:"analysis"`
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

func (c *HolmesClient) Investigate(ctx context.Context, a Alert) (Analysis, error) {
	body, _ := json.Marshal(chatRequest{Ask: buildAsk(a), Model: c.model, Stream: false})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/chat", bytes.NewReader(body))
	if err != nil {
		return Analysis{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.hc.Do(req)
	if err != nil {
		return Analysis{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return Analysis{}, fmt.Errorf("holmes: status %d", resp.StatusCode)
	}
	var cr chatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return Analysis{}, fmt.Errorf("holmes: decode: %w", err)
	}
	return Analysis{RootCause: cr.Analysis}, nil
}
