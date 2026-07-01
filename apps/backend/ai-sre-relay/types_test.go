package main

import (
	"encoding/json"
	"testing"
)

func TestAlertDecodesFingerprintAndLabels(t *testing.T) {
	const payload = `{
	  "fingerprint": "abc123",
	  "status": "firing",
	  "labels": {"alertname": "KubePodCrashLooping", "namespace": "prod", "severity": "warning"},
	  "annotations": {"description": "pod restarting"},
	  "startsAt": "2026-06-30T00:00:00Z"
	}`
	var a Alert
	if err := json.Unmarshal([]byte(payload), &a); err != nil {
		t.Fatal(err)
	}
	if a.Fingerprint != "abc123" || a.Labels["alertname"] != "KubePodCrashLooping" {
		t.Fatalf("bad decode: %+v", a)
	}
	if a.Name() != "KubePodCrashLooping" || a.Namespace() != "prod" || a.Severity() != "warning" {
		t.Fatalf("accessor mismatch: %+v", a)
	}
}
