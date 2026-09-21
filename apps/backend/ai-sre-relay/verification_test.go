package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// storeRead is what one read of the store showed while a remediation claimed
// the store was missing its version: the version was already set.
var storeRead = ToolEvidence{
	ID:          "call_store",
	Tool:        "kubectl_get_by_name",
	Description: "kubectl get clustersecretstore vault -o yaml",
	Output: `spec:
  provider:
    vault:
      path: kv
      version: v2
status:
  conditions:
  - type: Ready
    status: "True"`,
}

func TestVetVerificationRefusesUngroundedClaims(t *testing.T) {
	for name, p := range map[string]Patch{
		"no citation":          {Evidence: []ToolEvidence{storeRead}},
		"no live reads at all": {Verification: &Verification{ToolCallID: "call_store", Observed: "provider: vault: path: kv"}},
		"read that never ran": {Evidence: []ToolEvidence{storeRead},
			Verification: &Verification{ToolCallID: "call_invented", Observed: "provider: vault: path: kv"}},
		// The claim behind the closed store PR: quoted output the read did not produce.
		"excerpt not in the read": {Evidence: []ToolEvidence{storeRead},
			Verification: &Verification{ToolCallID: "call_store", Observed: "version: <unset> (defaults to v1)"}},
		"trivially short excerpt": {Evidence: []ToolEvidence{storeRead},
			Verification: &Verification{ToolCallID: "call_store", Observed: "True"}},
		"secret read": {Evidence: []ToolEvidence{{ID: "s", Tool: "kubectl_get_by_name", Description: "kubectl get secret bridge -n jdwillmsen-prd -o yaml", Output: "type: Opaque\ndata: 2 keys, both populated"}},
			Verification: &Verification{ToolCallID: "s", Observed: "type: Opaque data: 2 keys, both populated"}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := vetVerification(p); !errors.Is(err, ErrUnverified) {
				t.Fatalf("want ErrUnverified, got %v", err)
			}
		})
	}
}

// A citation survives the model re-indenting what it copied; the words and
// their order must match, the layout need not.
func TestVetVerificationAcceptsReflowedExcerpt(t *testing.T) {
	p := Patch{Evidence: []ToolEvidence{storeRead},
		Verification: &Verification{ToolCallID: "call_store", Observed: "vault:\n path: kv\n version: v2"}}
	ev, err := vetVerification(p)
	if err != nil {
		t.Fatal(err)
	}
	if ev.ID != "call_store" {
		t.Fatalf("cited %q", ev.ID)
	}
}

// An ExternalSecret or ClusterSecretStore read is not a Secret read.
func TestVetVerificationAllowsSecretAdjacentReads(t *testing.T) {
	ev := ToolEvidence{ID: "es", Tool: "kubectl_describe", Description: "kubectl describe externalsecret bridge -n jdwillmsen-prd",
		Output: "Status: SecretSyncedError could not get secret data from provider"}
	p := Patch{Evidence: []ToolEvidence{ev}, Verification: &Verification{ToolCallID: "es", Observed: "SecretSyncedError could not get secret data"}}
	if _, err := vetVerification(p); err != nil {
		t.Fatal(err)
	}
}

func TestGitHubOpenPRRefusesUnverifiedBeforeAnyCall(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	p := Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x: 1\n", Rationale: "r", Confidence: 0.9,
		Evidence: []ToolEvidence{storeRead}, Verification: &Verification{ToolCallID: "call_store", Observed: "version: <unset> (defaults to v1)"}}
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrUnverified) {
		t.Fatalf("want ErrUnverified, got %v", err)
	}
	if called {
		t.Fatal("an unverified proposal reached the GitHub API")
	}
}

// The PR carries the read and the quoted output, so a reviewer checks the
// claim against the cluster rather than against the model's prose.
func TestGitHubOpenPRBodyCarriesTheVerification(t *testing.T) {
	srv, calls := recordingGitHub(t)
	defer srv.Close()

	p := verified(Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/values.yaml", NewContent: "x: 1\n", Rationale: "raise limit", Confidence: 0.9})
	if _, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500"); err != nil {
		t.Fatal(err)
	}
	var body string
	for _, c := range *calls {
		if c.method == http.MethodPost && strings.HasSuffix(c.path, "/pulls") {
			body, _ = c.body["body"].(string)
		}
	}
	for _, want := range []string{"Live-state verification", "kubectl describe statefulset vault -n vault", "pod vault-0 exceeded quota: memory limit 256Mi"} {
		if !strings.Contains(body, want) {
			t.Fatalf("PR body omits %q:\n%s", want, body)
		}
	}
}

// Cluster output is untrusted text: it cannot close the fence it is quoted in.
func TestRenderVerificationFenceCannotBeClosedByOutput(t *testing.T) {
	out := renderVerification(ToolEvidence{Tool: "t", Description: "d"}, "a ~~~ b\n~~~~\n# heading")
	lines := strings.Split(out, "\n")
	var fence string
	for _, l := range lines {
		if strings.HasPrefix(l, "~~~") && !strings.Contains(l, " ") {
			fence = l
			break
		}
	}
	if len(fence) <= 4 {
		t.Fatalf("fence %q is not longer than the tildes in the output:\n%s", fence, out)
	}
}
