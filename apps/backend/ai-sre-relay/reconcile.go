package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"go.yaml.in/yaml/v3"
)

// tenantManifest is the part of tenants/<tenant>/tenant.yaml that decides
// which files under services/ ArgoCD reads. The governance ApplicationSet
// renders one tenant envelope per tenant.yaml, and the envelope's services
// ApplicationSet sources each listed release exactly as releaseReads states.
type tenantManifest struct {
	Services []struct {
		Name         string `yaml:"name"`
		PostInstall  bool   `yaml:"postInstall"`
		RawManifests bool   `yaml:"rawManifests"`
	} `yaml:"services"`
}

// releaseReads mirrors the services ApplicationSet's branches, and the
// platform repository's orphaned-manifest CI check that restates them:
// a rawManifests release reads only postInstall/*.yaml; any other release
// reads values.yaml, plus postInstall/*.yaml only when postInstall is true.
func releaseReads(t tenantManifest, release string) (values, postInstall, listed bool) {
	for _, s := range t.Services {
		if s.Name != release {
			continue
		}
		if s.RawManifests {
			return false, true, true
		}
		return true, s.PostInstall, true
	}
	return false, false, false
}

// splitServicePath breaks a cleaned path into its tenant, release and the file
// under the release, or reports that it is not a service file at all.
func splitServicePath(clean string) (tenant, release string, tail []string, ok bool) {
	parts := strings.Split(clean, "/")
	if len(parts) < 5 || parts[0] != "tenants" || parts[2] != "services" {
		return "", "", nil, false
	}
	return parts[1], parts[3], parts[4:], true
}

// checkReconciled resolves whether ArgoCD reads the file, from the tenant.yaml
// on the base branch rather than from a pattern. The path globs pass a
// directory that looks like a release but that no tenant lists — a release
// commented out, removed, or never added — and a postInstall file of a
// release that does not read postInstall. Either is a change that merges and
// does nothing. Reading tenant.yaml fails closed: an unreadable or
// unparseable one refuses the patch.
func (g *GitHubClient) checkReconciled(ctx context.Context, repo, clean string) error {
	tenant, release, tail, ok := splitServicePath(clean)
	if !ok {
		return fmt.Errorf("%w: %q is not under tenants/<tenant>/services/<release>/", ErrPathNotAllowed, clean)
	}
	tenantFile := "tenants/" + url.PathEscape(tenant) + "/tenant.yaml"
	resp, err := g.req(ctx, http.MethodGet, "/repos/"+repo+"/contents/"+tenantFile+"?ref="+baseBranch, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: cannot read %s on %s to resolve what ArgoCD reconciles (status %d)", ErrPathNotAllowed, tenantFile, baseBranch, resp.StatusCode)
	}
	var file struct {
		Content  string `json:"content"`
		Encoding string `json:"encoding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&file); err != nil || file.Encoding != "base64" {
		return fmt.Errorf("%w: %s on %s is unreadable", ErrPathNotAllowed, tenantFile, baseBranch)
	}
	raw, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(file.Content, "\n", ""))
	if err != nil {
		return fmt.Errorf("%w: %s on %s is not valid base64", ErrPathNotAllowed, tenantFile, baseBranch)
	}
	var t tenantManifest
	if err := yaml.Unmarshal(raw, &t); err != nil {
		return fmt.Errorf("%w: %s on %s does not parse: %v", ErrPathNotAllowed, tenantFile, baseBranch, err)
	}
	reads, readsPost, listed := releaseReads(t, release)
	switch {
	case !listed:
		return fmt.Errorf("%w: release %q is not listed in %s, so nothing reconciles %q", ErrPathNotAllowed, release, tenantFile, clean)
	case len(tail) == 1 && tail[0] == "values.yaml":
		if !reads {
			return fmt.Errorf("%w: release %q is rawManifests; its values.yaml is never read", ErrPathNotAllowed, release)
		}
	case len(tail) == 2 && tail[0] == "postInstall":
		if !readsPost {
			return fmt.Errorf("%w: release %q does not read postInstall/", ErrPathNotAllowed, release)
		}
	default:
		return fmt.Errorf("%w: only values.yaml and postInstall/*.yaml are read under services/%s/", ErrPathNotAllowed, release)
	}
	return nil
}
