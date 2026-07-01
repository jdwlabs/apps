package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type HolmesClient struct {
	baseURL string
	hc      *http.Client
}

func NewHolmesClient(baseURL string, hc *http.Client) *HolmesClient {
	if hc == nil {
		hc = http.DefaultClient
	}
	return &HolmesClient{baseURL: baseURL, hc: hc}
}

// holmesRequest matches Holmes 0.34.0 /api/investigate. subject carries the
// k8s object hints; context is reserved for extra grounding data.
type holmesRequest struct {
	Source      string            `json:"source"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Subject     map[string]string `json:"subject"`
	Context     map[string]string `json:"context"`
}

type holmesResponse struct {
	Analysis string `json:"analysis"`
}

func (c *HolmesClient) Investigate(ctx context.Context, a Alert) (Analysis, error) {
	body, _ := json.Marshal(holmesRequest{
		Source:      "prometheus",
		Title:       a.Name(),
		Description: a.Annotations["description"],
		Subject:     map[string]string{"namespace": a.Namespace(), "name": a.Labels["pod"], "kind": a.Labels["kind"]},
		Context:     map[string]string{"severity": a.Severity(), "fingerprint": a.Fingerprint},
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/investigate", bytes.NewReader(body))
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
	var hr holmesResponse
	if err := json.NewDecoder(resp.Body).Decode(&hr); err != nil {
		return Analysis{}, fmt.Errorf("holmes: decode: %w", err)
	}
	return Analysis{RootCause: hr.Analysis}, nil
}
