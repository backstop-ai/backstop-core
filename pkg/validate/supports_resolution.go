package validate

import (
	"fmt"
	"strings"

	"github.com/bmanson/backstop-core/pkg/artifact"
)

// SupportRef is one parsed supports pin harvested from a citing spec or issue
// requirement. Version is the @X.Y.Z pin; Pinned is false when no pin is present
// (an unpinned ref is a format defect caught per-artifact, but resolution still
// parses it). File and Label locate the citing requirement for violation
// attribution.
type SupportRef struct {
	Raw        string
	BundleName string
	ReqID      string
	Version    string
	Pinned     bool
	File       string
	Label      string
}

// BundleReqCatalog is a corpus index of every bundle REQ's effective version
// log: bundle name -> REQ id -> set of logged versions. It is the resolution
// target for both-direction resolution (REQ-001) and version-log match (REQ-003).
type BundleReqCatalog struct {
	// bundles maps bundle name -> REQ id -> the set of versions in that REQ's
	// effective log (membership is all resolution needs).
	bundles map[string]map[string]map[string]bool
}

// hasBundle reports whether the named bundle contributed any REQ to the catalog.
func (c *BundleReqCatalog) hasBundle(name string) bool {
	_, ok := c.bundles[name]
	return ok
}

// versionsFor returns the logged version set for a bundle REQ and whether the
// REQ was declared at all.
func (c *BundleReqCatalog) versionsFor(bundle, req string) (map[string]bool, bool) {
	reqs, ok := c.bundles[bundle]
	if !ok {
		return nil, false
	}
	versions, ok := reqs[req]
	return versions, ok
}

// BuildBundleReqCatalog extracts each bundle REQ's effective version log into the
// catalog. When a REQ's `versions:` list is absent the effective log is the
// single implicit entry {version}; when present, the list's versions are used.
// A bundle whose requirements[] is absent, non-list, or otherwise malformed is
// GRACEFULLY skipped (its own shape violations are reported by
// validateBundleRequirements) — the builder never panics (CLM-034). Well-formedness
// is NOT re-checked here; the builder reads what is present. The bundle set must
// be the FULL corpus regardless of any type-scoping filter (REQ-001).
func BuildBundleReqCatalog(bundles []*artifact.ParsedArtifact) *BundleReqCatalog {
	catalog := &BundleReqCatalog{bundles: make(map[string]map[string]map[string]bool)}

	for _, b := range bundles {
		if b == nil {
			continue
		}
		name := bundleName(b)
		if name == "" {
			continue
		}
		reqsVal, ok := b.Frontmatter["requirements"]
		if !ok {
			continue
		}
		reqs, ok := reqsVal.([]interface{})
		if !ok {
			continue // malformed requirements[] — skip this bundle gracefully
		}

		reqMap := make(map[string]map[string]bool)
		for _, item := range reqs {
			req, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			id, ok := getStringField(req, "id")
			if !ok {
				continue
			}
			reqMap[id] = effectiveVersionSet(req)
		}
		if len(reqMap) > 0 {
			catalog.bundles[name] = reqMap
		}
	}

	return catalog
}

// effectiveVersionSet returns the set of versions in a bundle REQ's effective
// log: the explicit `versions:` entries when present, else the single implicit
// entry from the REQ's own `version:`.
func effectiveVersionSet(req map[string]interface{}) map[string]bool {
	set := make(map[string]bool)
	if versionsVal, ok := req["versions"]; ok {
		if versions, ok := versionsVal.([]interface{}); ok {
			for _, item := range versions {
				entry, ok := item.(map[string]interface{})
				if !ok {
					continue
				}
				if v, ok := getStringField(entry, "version"); ok {
					set[v] = true
				}
			}
			if len(set) > 0 {
				return set
			}
		}
	}
	if v, ok := getStringField(req, "version"); ok {
		set[v] = true
	}
	return set
}

// CollectSupportRefs harvests every supports ref from the requirements[] of the
// given spec/issue artifacts, parsing bundle name, REQ id, and pin into a
// SupportRef. Terminal-status citers (replaced/canceled/deprecated/obsoleted) are
// SKIPPED so a retired artifact's stale ref is not resolution-checked, mirroring
// the per-artifact terminal exemption at spec.go / issue.go (CLM-035/CLM-036).
func CollectSupportRefs(citing []*artifact.ParsedArtifact) []SupportRef {
	var refs []SupportRef

	for _, art := range citing {
		if art == nil {
			continue
		}
		if isTerminalStatus(citingStatus(art)) {
			continue
		}
		reqsVal, ok := art.Frontmatter["requirements"]
		if !ok {
			continue
		}
		reqs, ok := reqsVal.([]interface{})
		if !ok {
			continue
		}
		for i, item := range reqs {
			req, ok := item.(map[string]interface{})
			if !ok {
				continue
			}
			supVal, ok := req["supports"]
			if !ok {
				continue
			}
			label := fmt.Sprintf("requirements[%d]", i)
			for _, sup := range supportsValues(supVal) {
				if strings.TrimSpace(sup) == "" {
					continue
				}
				refs = append(refs, parseSupportRef(sup, art.Filename, label))
			}
		}
	}

	return refs
}

// ResolveSupports checks each ref against the catalog and emits error-severity
// Violations for a missing bundle (REQ-001), an undeclared REQ (REQ-001), and a
// pin absent from the REQ's logged version set (REQ-003), at ANY citing status. A
// declared REQ with its pin present in the log resolves clean (CLM-001/012/014).
func ResolveSupports(catalog *BundleReqCatalog, refs []SupportRef) []Violation {
	var violations []Violation

	for _, ref := range refs {
		if !catalog.hasBundle(ref.BundleName) {
			violations = append(violations, Violation{
				Rule:     "supports/missing-bundle",
				File:     ref.File,
				Message:  fmt.Sprintf("%s supports '%s' names bundle '%s' which does not exist in the corpus", ref.Label, ref.Raw, ref.BundleName),
				Severity: "error",
			})
			continue
		}
		versions, declared := catalog.versionsFor(ref.BundleName, ref.ReqID)
		if !declared {
			violations = append(violations, Violation{
				Rule:     "supports/undeclared-req",
				File:     ref.File,
				Message:  fmt.Sprintf("%s supports '%s' cites '%s' which bundle '%s' never declares", ref.Label, ref.Raw, ref.ReqID, ref.BundleName),
				Severity: "error",
			})
			continue
		}
		if ref.Pinned && !versions[ref.Version] {
			violations = append(violations, Violation{
				Rule:     "supports/version-unlogged",
				File:     ref.File,
				Message:  fmt.Sprintf("%s supports '%s' pins version '%s' which is absent from '%s:%s' version log", ref.Label, ref.Raw, ref.Version, ref.BundleName, ref.ReqID),
				Severity: "error",
			})
		}
	}

	return violations
}

// parseSupportRef parses a raw supports value `bundle-name:REQ-NNN@X.Y.Z` (the
// pin is optional at parse time) into a SupportRef.
func parseSupportRef(raw, file, label string) SupportRef {
	ref := SupportRef{Raw: raw, File: file, Label: label}
	rest := raw
	if at := strings.LastIndex(rest, "@"); at >= 0 {
		ref.Version = rest[at+1:]
		ref.Pinned = true
		rest = rest[:at]
	}
	if colon := strings.Index(rest, ":"); colon >= 0 {
		ref.BundleName = rest[:colon]
		ref.ReqID = rest[colon+1:]
	} else {
		ref.ReqID = rest
	}
	return ref
}

// supportsValues normalizes a supports frontmatter value (a string or a list of
// strings) to a slice of strings.
func supportsValues(val interface{}) []string {
	switch v := val.(type) {
	case string:
		return []string{v}
	case []interface{}:
		out := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

// citingStatus resolves a citing artifact's status for the terminal exemption:
// issues carry it in the issue.* block, every other type in top-level metadata.
func citingStatus(art *artifact.ParsedArtifact) string {
	if strings.HasSuffix(art.Filename, ".issue.md") {
		if issueVal, ok := art.Frontmatter["issue"].(map[string]interface{}); ok {
			if s, ok := getStringField(issueVal, "status"); ok {
				return s
			}
		}
		return ""
	}
	return art.Metadata["status"]
}

// bundleName returns bundle.name from a parsed bundle, or "" when absent.
func bundleName(art *artifact.ParsedArtifact) string {
	bundleVal, ok := art.Frontmatter["bundle"].(map[string]interface{})
	if !ok {
		return ""
	}
	return stringField(bundleVal, "name")
}
