package main

import "encoding/base64"

// testTenantYAML is the tenant.yaml the fake GitHub serves for every tenant.
// vault reads values.yaml and postInstall/; raw-only reads postInstall/ only;
// values-only reads values.yaml only. Anything else is unlisted.
const testTenantYAML = `name: platform
services:
  - name: vault
    postInstall: true
  - name: raw-only
    rawManifests: true
  - name: values-only
    postInstall: false
  # - name: parked
  #   postInstall: true
`

// contentsStub is the Contents API answer the fake GitHub gives for any file:
// the blob SHA the existing-file check reads, and a body that is the test
// tenant.yaml, which is what the reconciliation check decodes.
func contentsStub() map[string]string {
	return map[string]string{
		"sha": "filesha", "type": "file", "encoding": "base64",
		"content": base64.StdEncoding.EncodeToString([]byte(testTenantYAML)),
	}
}

// testEvidence is one live read shaped like Holmes' kubectl output.
var testEvidence = []ToolEvidence{{
	ID:          "call_1",
	Tool:        "kubectl_describe",
	Description: "kubectl describe statefulset vault -n vault",
	Output:      "Name: vault\nReplicas:  1 desired | 0 total\nEvents:\n  Warning  FailedCreate  pod vault-0 exceeded quota: memory limit 256Mi",
}}

// verified attaches the evidence and a citation that passes vetVerification,
// for tests whose subject is something other than the citation.
func verified(p Patch) Patch {
	p.Evidence = testEvidence
	p.Verification = &Verification{ToolCallID: "call_1", Observed: "pod vault-0 exceeded quota: memory limit 256Mi"}
	return p
}
