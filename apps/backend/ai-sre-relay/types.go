package main

// Alert is one firing Alertmanager alert. Fields are extended when the webhook handler is implemented.
type Alert struct {
	Fingerprint string `json:"fingerprint"`
}
