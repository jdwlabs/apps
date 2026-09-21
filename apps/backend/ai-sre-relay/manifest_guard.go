package main

import (
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

// ErrContentRefused marks a proposal whose file body the relay will not commit
// whatever the alert says: a Secret manifest, a placeholder credential, or a
// body that is not YAML at all. Unlike a path refusal there is no better file
// the model could have picked; the change itself is the defect.
var ErrContentRefused = errors.New("github: patch content refused")

// maxEmbeddedDepth bounds how far YAML nested inside string values is
// re-parsed. Real values files embed a manifest one level down at most; the
// bound exists so a crafted body cannot make the walk recurse without end.
const maxEmbeddedDepth = 3

// secretManifestFields are the keys that make a `kind: Secret` mapping an
// object rather than a reference to one. A Gateway certificateRef or a
// secretKeyRef carries kind and name only; anything that would create or
// overwrite a Secret needs at least one of these.
var secretManifestFields = []string{"apiVersion", "metadata", "data", "stringData", "type", "immutable"}

// credentialKey matches keys that hold a credential by value. Keys that name
// or reference a credential (existingSecret, secretName, tokenSecretRef) are
// excluded by requiring the credential word to end the key, so only the
// value-bearing shape reaches the placeholder test.
var credentialKey = regexp.MustCompile(`(?i)(password|passwd|passphrase|pass|pwd|token|secret|api[_-]?key|access[_-]?key|private[_-]?key|credentials?|auth)$`)

// placeholderValue matches the stand-in values a model writes when it has no
// real credential to put in the field. Committed, any of them is either a
// working weak credential or a broken one; both are worse than no change.
// Template references ({{ .key }}, {{env.X}}, ${X}) are deliberately absent:
// ExternalSecret templates and chart env lookups use them to name a value
// that is resolved elsewhere, which is the sanctioned shape.
var placeholderValue = regexp.MustCompile(`(?i)^\s*(` +
	`change[-_ .]?me|change[-_ .]?it|replace[-_ .]?me|replace[-_ .]?this|placeholder|` +
	`todo|tbd|fixme|dummy|example|sample|redacted|` +
	`secret|password|passw0rd|p@ssw0rd|token|api[-_ ]?key|` +
	`x{3,}|\*{3,}|\.{3,}|` +
	`<[^>]*>|` +
	`your[-_ ].*|insert[-_ ].*|` +
	`12345\d*|0{4,}` +
	`)\s*$`)

// placeholderFragment catches the same intent inside a longer value, such as
// "change-me-please" or "REPLACE_WITH_REAL_TOKEN".
var placeholderFragment = regexp.MustCompile(`(?i)(change[-_ .]?me|replace[-_ .]?(me|with)|placeholder|dummy[-_ ]?(secret|password|token|value))`)

// vetManifest refuses a file body the relay must never commit. It parses every
// YAML document rather than grepping, so a Secret written in flow style,
// with a quoted kind, behind an anchor, or inside a values file's list of
// extra objects is found the same way as a plain top-level one. YAML embedded
// in a string value — a block scalar carrying a raw manifest, which several
// charts accept — is parsed and walked too.
func vetManifest(content string) error {
	return vetYAML(content, 0)
}

func vetYAML(content string, depth int) error {
	dec := yaml.NewDecoder(strings.NewReader(content))
	for {
		var doc yaml.Node
		err := dec.Decode(&doc)
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			if depth > 0 {
				// An embedded string that merely looked like YAML is not a
				// manifest; only the file itself has to parse.
				return nil
			}
			return fmt.Errorf("%w: body is not parseable YAML: %v", ErrContentRefused, err)
		}
		if err := walkNode(&doc, depth, map[*yaml.Node]bool{}); err != nil {
			return err
		}
	}
}

func walkNode(n *yaml.Node, depth int, seen map[*yaml.Node]bool) error {
	if n == nil || seen[n] {
		return nil
	}
	seen[n] = true
	switch n.Kind {
	case yaml.AliasNode:
		return walkNode(n.Alias, depth, seen)
	case yaml.MappingNode:
		if err := vetMapping(n); err != nil {
			return err
		}
	case yaml.ScalarNode:
		if depth < maxEmbeddedDepth && strings.Contains(n.Value, "\n") && strings.Contains(n.Value, "kind") {
			return vetYAML(n.Value, depth+1)
		}
		return nil
	}
	for _, c := range n.Content {
		if err := walkNode(c, depth, seen); err != nil {
			return err
		}
	}
	return nil
}

// vetMapping applies both refusals to one mapping. Merge keys and aliases are
// resolved by the caller's walk, so a value hidden behind an anchor is still
// seen here once the walk reaches the anchored node.
func vetMapping(n *yaml.Node) error {
	fields := map[string]*yaml.Node{}
	for i := 0; i+1 < len(n.Content); i += 2 {
		k, v := n.Content[i], resolveAlias(n.Content[i+1])
		if k.Kind != yaml.ScalarNode {
			continue
		}
		fields[k.Value] = v
		if v != nil && v.Kind == yaml.ScalarNode && credentialKey.MatchString(k.Value) && isPlaceholder(v.Value) {
			return fmt.Errorf("%w: %q carries a placeholder credential value (line %d)", ErrContentRefused, k.Value, k.Line)
		}
	}
	kind, ok := fields["kind"]
	if !ok || kind == nil || kind.Kind != yaml.ScalarNode || !strings.EqualFold(strings.TrimSpace(kind.Value), "Secret") {
		return nil
	}
	for _, f := range secretManifestFields {
		if _, present := fields[f]; present {
			return fmt.Errorf("%w: a kind: Secret manifest (line %d); secret material belongs in Vault behind an ExternalSecret, never in git", ErrContentRefused, kind.Line)
		}
	}
	return nil
}

func resolveAlias(n *yaml.Node) *yaml.Node {
	for i := 0; n != nil && n.Kind == yaml.AliasNode && i < 8; i++ {
		n = n.Alias
	}
	return n
}

func isPlaceholder(v string) bool {
	if strings.TrimSpace(v) == "" {
		return false
	}
	return placeholderValue.MatchString(v) || placeholderFragment.MatchString(v)
}
