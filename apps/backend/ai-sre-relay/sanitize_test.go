package main

import (
	"errors"
	"strings"
	"testing"
)

// Modeled on a filed ticket whose body opened with model narration before the
// actual analysis.
const preambleAnalysis = "Now I have a complete picture. Let me summarize the investigation findings.\n\n" +
	"## Root Cause Analysis: KubeAggregatedAPIDown Alert\n\n" +
	"### Root Cause Chain\n\n" +
	"etcd on 192.168.1.125 degraded under OOM pressure."

// Modeled on a filed ticket whose entire body was an un-executed tool call
// emitted as text.
const toolCallOnly = "<tool_call>\n<function=execute_prometheus_instant_query>\n" +
	"<parameter=description>\nCheck target scrape pool target health\n</parameter>\n" +
	"<parameter=query>\nprometheus_target_scrape_pool_target_health\n</parameter>\n" +
	"<parameter=timeout>\n20\n</parameter>\n</function>\n</tool_call>\n"

func TestSanitizeAnalysisStripsConversationalPreamble(t *testing.T) {
	got, err := sanitizeAnalysis(preambleAnalysis)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got, "## Root Cause Analysis") {
		t.Fatalf("preamble not stripped, got: %q", got)
	}
	if !strings.Contains(got, "etcd on 192.168.1.125") {
		t.Fatalf("substantive content lost: %q", got)
	}
}

func TestSanitizeAnalysisStripsTrailingChatOffer(t *testing.T) {
	in := "## Root Cause\n\nThe node is under memory pressure.\n\nLet me know if you need anything else!"
	got, err := sanitizeAnalysis(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "Let me know") {
		t.Fatalf("trailing chat offer not stripped: %q", got)
	}
	if !strings.Contains(got, "memory pressure") {
		t.Fatalf("substantive content lost: %q", got)
	}
}

func TestSanitizeAnalysisPassesCleanAnalysisThrough(t *testing.T) {
	in := "## Root Cause Analysis: TargetDown\n\n" +
		"### Summary\n\nThe kubelet on talos-fow-vbk is unresponsive.\n\n" +
		"1. Check node memory\n2. Add limits to kube-proxy"
	got, err := sanitizeAnalysis(in)
	if err != nil {
		t.Fatal(err)
	}
	if got != in {
		t.Fatalf("clean analysis must pass through unchanged:\nwant %q\ngot  %q", in, got)
	}
}

func TestSanitizeAnalysisRejectsToolCallMarkup(t *testing.T) {
	if _, err := sanitizeAnalysis(toolCallOnly); !errors.Is(err, errAnalysisToolCall) {
		t.Fatalf("want errAnalysisToolCall, got %v", err)
	}
}

func TestSanitizeAnalysisRejectsUnterminatedToolCall(t *testing.T) {
	in := "<tool_call>\n<function=fetch_logs>\n<parameter=pod>api-0"
	if _, err := sanitizeAnalysis(in); !errors.Is(err, errAnalysisToolCall) {
		t.Fatalf("want errAnalysisToolCall, got %v", err)
	}
}

func TestSanitizeAnalysisRejectsJSONToolCall(t *testing.T) {
	in := `{"name":"execute_prometheus_instant_query","arguments":{"query":"up == 0"}}`
	if _, err := sanitizeAnalysis(in); !errors.Is(err, errAnalysisToolCall) {
		t.Fatalf("want errAnalysisToolCall, got %v", err)
	}
	in = `{"tool_calls":[{"function":{"name":"kubectl_get","arguments":"{}"}}]}`
	if _, err := sanitizeAnalysis(in); !errors.Is(err, errAnalysisToolCall) {
		t.Fatalf("want errAnalysisToolCall, got %v", err)
	}
}

func TestSanitizeAnalysisRejectsEmpty(t *testing.T) {
	for _, in := range []string{"", "   \n\t\n"} {
		if _, err := sanitizeAnalysis(in); !errors.Is(err, errAnalysisEmpty) {
			t.Fatalf("input %q: want errAnalysisEmpty, got %v", in, err)
		}
	}
}

func TestSanitizeAnalysisRejectsAllFillerContent(t *testing.T) {
	in := "Let me investigate this alert for you.\n\nI'll start by checking the pod logs."
	if _, err := sanitizeAnalysis(in); err == nil {
		t.Fatal("pure narration with no analysis must be rejected")
	}
}

func TestSanitizeAnalysisStripsEmbeddedToolCallKeepsProse(t *testing.T) {
	in := "## Root Cause\n\nDisk pressure on node pve5.\n\n" + toolCallOnly
	got, err := sanitizeAnalysis(in)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "<tool_call>") || strings.Contains(got, "<function=") {
		t.Fatalf("tool-call markup leaked through: %q", got)
	}
	if !strings.Contains(got, "Disk pressure on node pve5.") {
		t.Fatalf("substantive content lost: %q", got)
	}
}
