package main

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// committedPlaceholderSecret is the body a remediation PR actually committed,
// verbatim: a Secret an ExternalSecret already owned, with literal
// placeholder values. It is the regression this guard exists for.
const committedPlaceholderSecret = `apiVersion: v1
kind: Secret
metadata:
  name: minecraft-fwb-console-bridge
  namespace: jdwillmsen-prd
stringData:
  console_websocket_password: "change-me"
  console_bridge_token: "change-me"
`

func TestVetManifestRefusesSecretManifests(t *testing.T) {
	for name, body := range map[string]string{
		"committed regression": committedPlaceholderSecret,
		"real-looking values": `apiVersion: v1
kind: Secret
metadata: {name: s}
data:
  password: aHVudGVyMg==
`,
		"flow style":     `{apiVersion: v1, kind: Secret, metadata: {name: s}, data: {a: Yg==}}`,
		"quoted kind":    "apiVersion: v1\nkind: \"Secret\"\nmetadata:\n  name: s\n",
		"lowercase kind": "apiVersion: v1\nkind: secret\nmetadata:\n  name: s\n",
		"second document": `apiVersion: v1
kind: ConfigMap
metadata: {name: c}
---
apiVersion: v1
kind: Secret
metadata: {name: s}
type: Opaque
`,
		"inside a List": `apiVersion: v1
kind: List
items:
  - apiVersion: v1
    kind: Secret
    metadata: {name: s}
`,
		"values extraObjects": `replicas: 2
extraObjects:
  - apiVersion: v1
    kind: Secret
    metadata: {name: s}
    stringData: {k: v}
`,
		"block scalar manifest": `extraManifests: |
  apiVersion: v1
  kind: Secret
  metadata:
    name: s
`,
		"embedded JSON": "extra: \"{\\\"apiVersion\\\":\\\"v1\\\",\\n\\\"kind\\\":\\\"Secret\\\",\\\"metadata\\\":{\\\"name\\\":\\\"s\\\"}}\"\n",
		"behind an alias": `base: &secretish
  apiVersion: v1
  kind: Secret
  metadata: {name: s}
copy: *secretish
`,
	} {
		t.Run(name, func(t *testing.T) {
			if err := vetManifest(body); !errors.Is(err, ErrContentRefused) {
				t.Fatalf("want ErrContentRefused, got %v", err)
			}
		})
	}
}

func TestVetManifestRefusesPlaceholderCredentials(t *testing.T) {
	for name, body := range map[string]string{
		"change-me in a ConfigMap": "kind: ConfigMap\ndata:\n  DB_PASSWORD: change-me\n",
		"angle-bracket token":      "auth:\n  token: \"<your-token>\"\n",
		"replace me api key":       "apiKey: REPLACE_ME\n",
		"x-run password":           "password: xxxxxxxx\n",
		"longer change-me":         "admin_password: change-me-before-prod\n",
		"example secret":           "webhookSecret: example\n",
	} {
		t.Run(name, func(t *testing.T) {
			if err := vetManifest(body); !errors.Is(err, ErrContentRefused) {
				t.Fatalf("want ErrContentRefused, got %v", err)
			}
		})
	}
}

// What the guard must leave alone: references to a Secret by name, the
// ExternalSecret that is the sanctioned way to materialize one, and ordinary
// values. The first is taken from a values file the arm may edit today.
func TestVetManifestAllowsReferencesAndOrdinaryValues(t *testing.T) {
	for name, body := range map[string]string{
		"gateway certificateRef": `gateway:
  listeners:
    - name: https
      tls:
        certificateRefs:
          - kind: Secret
            name: wildcard-jdwlabs-tls
`,
		"external secret": `apiVersion: external-secrets.io/v1
kind: ExternalSecret
metadata: {name: bridge, namespace: jdwillmsen-prd}
spec:
  secretStoreRef: {kind: ClusterSecretStore, name: vault}
  target: {name: bridge}
  data:
    - secretKey: console_bridge_token
      remoteRef: {key: jdwillmsen/bridge, property: token}
`,
		"secret references": `existingSecret: grafana-admin
secretName: tls
tokenSecretRef: {name: vault-token, key: token}
auth: none
`,
		// Resolved elsewhere, not a stand-in: ExternalSecret templates and
		// chart env lookups, as the watched tree uses them today.
		"template references": `OIDC_CLIENT_SECRET: "{{ .clientSecret }}"
auth: "{{ .auth }}"
api_key: "{{env.OPENAI_API_KEY}}"
clientSecret: ${CLIENT_SECRET}
`,
		"plain values":    "resources:\n  limits:\n    memory: 512Mi\n",
		"empty password":  "password: \"\"\n",
		"scalar document": "x",
	} {
		t.Run(name, func(t *testing.T) {
			if err := vetManifest(body); err != nil {
				t.Fatalf("refused an acceptable body: %v", err)
			}
		})
	}
}

func TestVetManifestRefusesUnparseableBody(t *testing.T) {
	if err := vetManifest("key: [unclosed\n"); !errors.Is(err, ErrContentRefused) {
		t.Fatalf("want ErrContentRefused, got %v", err)
	}
}

// End to end at the commit path: the committed Secret, aimed at an existing,
// reconciled postInstall file with a valid citation, still never reaches the
// GitHub API. Nothing about the path or the evidence can make it acceptable.
func TestGitHubOpenPRNeverCommitsASecret(t *testing.T) {
	var called bool
	srv := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { called = true }))
	defer srv.Close()

	p := verified(Patch{Repo: "jdwlabs/platform", FilePath: "tenants/platform/services/vault/postInstall/externalsecret.yaml",
		NewContent: committedPlaceholderSecret, Rationale: "creates the missing secret", Confidence: 0.95})
	_, err := constrainedClient(srv).OpenPR(context.Background(), p, "JDWLABS-500")
	if !errors.Is(err, ErrContentRefused) {
		t.Fatalf("want ErrContentRefused, got %v", err)
	}
	if called {
		t.Fatal("a Secret manifest reached the GitHub API")
	}
}
