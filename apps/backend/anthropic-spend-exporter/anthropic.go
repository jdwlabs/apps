package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const (
	anthropicVersion = "2023-06-01"
	userAgent        = "jdwlabs-anthropic-spend-exporter/1 (https://github.com/jdwlabs/apps)"

	// Both list endpoints page by a cursor the server chooses. A cycle is at
	// most 31 daily buckets and the org has a handful of workspaces, so a
	// well-behaved server never gets near this; the cap is what stops a
	// repeated cursor from spinning one refresh forever.
	maxPages = 50

	// Bounds what one response can make the exporter hold in memory.
	maxBodyBytes = 8 << 20
)

// costSeries identifies one exported cost series. Out of the client,
// workspace holds the workspace ID ("" for the default workspace); the
// refresher swaps it for a display name before the series is published.
type costSeries struct {
	workspace   string
	model       string
	serviceTier string
	costType    string
}

type AnthropicClient struct {
	base     string
	adminKey string
	hc       *http.Client
}

func NewAnthropicClient(base, adminKey string, hc *http.Client) *AnthropicClient {
	return &AnthropicClient{base: base, adminKey: adminKey, hc: hc}
}

func (c *AnthropicClient) get(ctx context.Context, op, path string, q url.Values, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base+path+"?"+q.Encode(), nil)
	if err != nil {
		return fmt.Errorf("anthropic: %s: %w", op, err)
	}
	req.Header.Set("x-api-key", c.adminKey)
	req.Header.Set("anthropic-version", anthropicVersion)
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.hc.Do(req)
	if err != nil {
		return fmt.Errorf("anthropic: %s: %w", op, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusUnauthorized, http.StatusForbidden:
		return fmt.Errorf("anthropic: %s: status %d: ANTHROPIC_ADMIN_KEY must be an Admin API key, and the Admin API is unavailable for individual accounts", op, resp.StatusCode)
	default:
		return fmt.Errorf("anthropic: %s: status %d", op, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBodyBytes+1))
	if err != nil {
		return fmt.Errorf("anthropic: %s: reading response: %w", op, err)
	}
	if len(body) > maxBodyBytes {
		return fmt.Errorf("anthropic: %s: response larger than %d bytes", op, maxBodyBytes)
	}
	if err := json.Unmarshal(body, out); err != nil {
		return fmt.Errorf("anthropic: %s: decoding response: %w", op, err)
	}
	return nil
}

type costReportPage struct {
	Data []struct {
		Results []struct {
			Amount      string  `json:"amount"`
			Currency    string  `json:"currency"`
			CostType    *string `json:"cost_type"`
			Model       *string `json:"model"`
			ServiceTier *string `json:"service_tier"`
			WorkspaceID *string `json:"workspace_id"`
		} `json:"results"`
	} `json:"data"`
	HasMore  bool    `json:"has_more"`
	NextPage *string `json:"next_page"`
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// CycleCosts sums the cost report from start to now, in USD, keyed by
// workspace ID. It returns either the whole cycle or an error: a total built
// from the pages that happened to arrive would be an undercount that reads
// exactly like a real one.
func (c *AnthropicClient) CycleCosts(ctx context.Context, start time.Time) (map[costSeries]float64, error) {
	const op = "cost report"
	q := url.Values{}
	q.Set("starting_at", start.UTC().Format(time.RFC3339))
	q.Set("bucket_width", "1d")
	q.Set("limit", "31")
	q.Add("group_by[]", "workspace_id")
	q.Add("group_by[]", "description")

	costs := map[costSeries]float64{}
	for range maxPages {
		var page costReportPage
		if err := c.get(ctx, op, "/v1/organizations/cost_report", q, &page); err != nil {
			return nil, err
		}
		for _, bucket := range page.Data {
			for _, item := range bucket.Results {
				if item.Currency != "USD" {
					return nil, fmt.Errorf("anthropic: %s: unexpected currency %q", op, item.Currency)
				}
				// ParseFloat accepts "NaN" and "Inf" without error, and one
				// of either would poison every sum built on this map.
				cents, err := strconv.ParseFloat(item.Amount, 64)
				if err != nil || math.IsNaN(cents) || math.IsInf(cents, 0) {
					return nil, fmt.Errorf("anthropic: %s: unparseable amount %q", op, item.Amount)
				}
				key := costSeries{
					workspace:   deref(item.WorkspaceID),
					model:       deref(item.Model),
					serviceTier: deref(item.ServiceTier),
					costType:    deref(item.CostType),
				}
				costs[key] += cents / 100
			}
		}
		if !page.HasMore {
			return costs, nil
		}
		if deref(page.NextPage) == "" {
			return nil, fmt.Errorf("anthropic: %s: has_more without next_page", op)
		}
		q.Set("page", *page.NextPage)
	}
	return nil, fmt.Errorf("anthropic: %s: more than %d pages", op, maxPages)
}

type workspacesPage struct {
	Data []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"data"`
	HasMore bool    `json:"has_more"`
	LastID  *string `json:"last_id"`
}

// WorkspaceNames maps workspace ID to name. Archived workspaces are
// included because one archived mid-cycle still has cost in that cycle.
func (c *AnthropicClient) WorkspaceNames(ctx context.Context) (map[string]string, error) {
	const op = "listing workspaces"
	q := url.Values{}
	q.Set("limit", "1000")
	q.Set("include_archived", "true")

	names := map[string]string{}
	for range maxPages {
		var page workspacesPage
		if err := c.get(ctx, op, "/v1/organizations/workspaces", q, &page); err != nil {
			return nil, err
		}
		for _, ws := range page.Data {
			names[ws.ID] = ws.Name
		}
		if !page.HasMore {
			return names, nil
		}
		if deref(page.LastID) == "" {
			return nil, errors.New("anthropic: " + op + ": has_more without last_id")
		}
		q.Set("after_id", *page.LastID)
	}
	return nil, fmt.Errorf("anthropic: %s: more than %d pages", op, maxPages)
}
