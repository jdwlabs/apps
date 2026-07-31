package main

// Alert is one firing alert from an Alertmanager v4 webhook.
type Alert struct {
	Fingerprint string            `json:"fingerprint"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	StartsAt    string            `json:"startsAt"`
}

func (a Alert) Name() string      { return a.Labels["alertname"] }
func (a Alert) Namespace() string { return a.Labels["namespace"] }
func (a Alert) Severity() string  { return a.Labels["severity"] }

// Analysis is Holmes' prose root-cause output.
type Analysis struct {
	RootCause string // full markdown analysis
}

// Patch is a machine-readable, single-file remediation proposal.
type Patch struct {
	Repo       string  `json:"repo"`        // "owner/name"
	FilePath   string  `json:"file_path"`   // path within repo
	NewContent string  `json:"new_content"` // full new file contents
	Rationale  string  `json:"rationale"`
	Confidence float64 `json:"confidence"` // 0..1
}

// IssueKey is a Jira issue identifier, e.g. "ABC-123".
type IssueKey string

// PRLink is a rendered GitHub pull request URL.
type PRLink string
