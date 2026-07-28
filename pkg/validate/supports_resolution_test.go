package validate_test

import (
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/validate"
)

// resBundle builds a bundle artifact carrying the given name and requirements[]
// for BuildBundleReqCatalog to index.
func resBundle(name string, reqs []interface{}) *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: name + ".bundle.md",
		Frontmatter: map[string]interface{}{
			"bundle": map[string]interface{}{
				"name": name,
			},
			"requirements": reqs,
		},
	}
}

// resSpecCiter builds a spec artifact at the given status citing one supports ref.
func resSpecCiter(status, supports string) *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "SPEC-001-example.spec.md",
		Metadata: map[string]string{"status": status},
		Frontmatter: map[string]interface{}{
			"status": status,
			"requirements": []interface{}{
				map[string]interface{}{
					"id":       "REQ-001",
					"text":     "cites a bundle req",
					"supports": supports,
				},
			},
		},
	}
}

// resIssueCiter builds an issue artifact at the given status citing one supports ref.
func resIssueCiter(status, supports string) *artifact.ParsedArtifact {
	return &artifact.ParsedArtifact{
		Filename: "ISSUE-001-example.issue.md",
		Frontmatter: map[string]interface{}{
			"issue": map[string]interface{}{
				"id":     "ISSUE-001",
				"status": status,
			},
			"requirements": []interface{}{
				map[string]interface{}{
					"id":       "REQ-001",
					"text":     "cites a bundle req",
					"supports": supports,
				},
			},
		},
	}
}

func singleVersionReq(id, version string) map[string]interface{} {
	return map[string]interface{}{"id": id, "text": "req text", "version": version}
}

func resolveHasRule(vs []validate.Violation, rule string) bool {
	for _, v := range vs {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

func TestResolveSupports_DeclaredReqResolves(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected a declared+logged ref to resolve clean, got: %v", violations)
	}
}

func TestResolveSupports_MissingBundleErrors(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "other-bundle:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if !resolveHasRule(violations, "supports/missing-bundle") {
		t.Errorf("expected supports/missing-bundle, got: %v", violations)
	}
}

func TestResolveSupports_UndeclaredReqErrors(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-099@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if !resolveHasRule(violations, "supports/undeclared-req") {
		t.Errorf("expected supports/undeclared-req, got: %v", violations)
	}
}

func TestResolveSupports_DraftCitingStatusIndependent(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	// A draft spec citing a real, declared, logged REQ resolves clean — resolution
	// keys off the ref target, never the citing artifact's live status.
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected a draft citer's valid ref to resolve clean, got: %v", violations)
	}
}

func TestResolveSupports_IssueRefMissingBundleErrors(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resIssueCiter("open", "other-bundle:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if !resolveHasRule(violations, "supports/missing-bundle") {
		t.Errorf("expected an issue ref to be resolution-checked (missing-bundle), got: %v", violations)
	}
}

func TestResolveSupports_PinMatchesLogEntry(t *testing.T) {
	reqWithLog := map[string]interface{}{
		"id":      "REQ-001",
		"text":    "Second",
		"version": "2.0.0",
		"versions": []interface{}{
			map[string]interface{}{"version": "1.0.0", "text": "First"},
			map[string]interface{}{"version": "2.0.0", "text": "Second"},
		},
	}
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{reqWithLog}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-001@2.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected a pin matching a log entry to resolve clean, got: %v", violations)
	}
}

func TestResolveSupports_FabricatedVersionErrors(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{singleVersionReq("REQ-001", "1.0.0")}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-001@9.9.9"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if !resolveHasRule(violations, "supports/version-unlogged") {
		t.Errorf("expected supports/version-unlogged for a fabricated version, got: %v", violations)
	}
}

func TestResolveSupports_OlderLoggedVersionResolves(t *testing.T) {
	reqWithLog := map[string]interface{}{
		"id":      "REQ-001",
		"text":    "Second",
		"version": "2.0.0",
		"versions": []interface{}{
			map[string]interface{}{"version": "1.0.0", "text": "First"},
			map[string]interface{}{"version": "2.0.0", "text": "Second"},
		},
	}
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		resBundle("my-bundle", []interface{}{reqWithLog}),
	})
	// Pin to the OLDER logged version — historical pins stay resolvable.
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "my-bundle:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected an older logged version to resolve clean, got: %v", violations)
	}
}

func TestBuildBundleReqCatalog_MalformedBundleSkippedGracefully(t *testing.T) {
	// A bundle whose requirements[] is a non-list must be skipped without panic;
	// resolution still surfaces the ref violation.
	malformed := &artifact.ParsedArtifact{
		Filename: "broken.bundle.md",
		Frontmatter: map[string]interface{}{
			"bundle":       map[string]interface{}{"name": "broken"},
			"requirements": "not-a-list",
		},
	}
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{malformed})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "broken:REQ-001@1.0.0"),
	})
	violations := validate.ResolveSupports(catalog, refs)
	if !resolveHasRule(violations, "supports/missing-bundle") {
		t.Errorf("expected the malformed bundle to be skipped and the ref to surface missing-bundle, got: %v", violations)
	}
}

func TestResolveSupports_DeprecatedCiterExempt(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog(nil)
	// A deprecated (terminal) spec's dangling ref must NOT be harvested.
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("deprecated", "gone-bundle:REQ-001@1.0.0"),
	})
	if len(refs) != 0 {
		t.Errorf("expected a deprecated citer's refs to be skipped by the harvest, got: %v", refs)
	}
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected a deprecated citer's dangling ref to resolve clean, got: %v", violations)
	}
}

func TestResolveSupports_ReplacedCiterExempt(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog(nil)
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("replaced", "gone-bundle:REQ-001@1.0.0"),
	})
	if len(refs) != 0 {
		t.Errorf("expected a replaced citer's refs to be skipped by the harvest, got: %v", refs)
	}
	violations := validate.ResolveSupports(catalog, refs)
	if len(violations) != 0 {
		t.Errorf("expected a replaced citer's ref to resolve clean, got: %v", violations)
	}
}

// TestBuildBundleReqCatalog_SkipsNilNamelessAndMalformedEntries exercises the
// graceful-skip edge paths: a nil bundle, a bundle with no name, a bundle whose
// `bundle` block is not a map, a non-map requirement entry, and a requirement with
// no id — none contribute, none panic, and a real REQ alongside them still lands.
func TestBuildBundleReqCatalog_SkipsNilNamelessAndMalformedEntries(t *testing.T) {
	catalog := validate.BuildBundleReqCatalog([]*artifact.ParsedArtifact{
		nil,
		{Filename: "noname.bundle.md", Frontmatter: map[string]interface{}{"bundle": map[string]interface{}{}}},
		{Filename: "notmap.bundle.md", Frontmatter: map[string]interface{}{"bundle": "not-a-map", "requirements": []interface{}{}}},
		resBundle("real", []interface{}{
			"not-a-map-req",
			map[string]interface{}{"text": "no id here"},
			singleVersionReq("REQ-001", "1.0.0"),
		}),
	})
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "real:REQ-001@1.0.0"),
	})
	if v := validate.ResolveSupports(catalog, refs); len(v) != 0 {
		t.Errorf("real:REQ-001 should resolve clean alongside skipped malformed bundles, got: %v", v)
	}
	// A ref to the nameless/malformed bundles resolves as missing-bundle (they
	// contributed nothing), proving no panic and the skip.
	missingRefs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		resSpecCiter("draft", "noname:REQ-001@1.0.0"),
	})
	if v := validate.ResolveSupports(catalog, missingRefs); !resolveHasRule(v, "supports/missing-bundle") {
		t.Errorf("a skipped bundle must surface missing-bundle, got: %v", v)
	}
}

// TestCollectSupportRefs_SkipsNilMalformedAndEmpty exercises the harvest edge
// paths: a nil citer, a citer whose requirements[] is not a list, a non-map
// requirement, an empty supports value, an unpinned/colon-less ref, and a supports
// LIST carrying a non-string element.
func TestCollectSupportRefs_SkipsNilMalformedAndEmpty(t *testing.T) {
	specWith := func(name string, supports interface{}) *artifact.ParsedArtifact {
		return &artifact.ParsedArtifact{
			Filename: name + ".spec.md",
			Metadata: map[string]string{"status": "draft"},
			Frontmatter: map[string]interface{}{
				"status": "draft",
				"requirements": []interface{}{
					map[string]interface{}{"id": "REQ-001", "text": "t", "supports": supports},
				},
			},
		}
	}
	refs := validate.CollectSupportRefs([]*artifact.ParsedArtifact{
		nil,
		{Filename: "notlist.spec.md", Metadata: map[string]string{"status": "draft"}, Frontmatter: map[string]interface{}{"status": "draft", "requirements": "not-a-list"}},
		{Filename: "nonmap.spec.md", Metadata: map[string]string{"status": "draft"}, Frontmatter: map[string]interface{}{"status": "draft", "requirements": []interface{}{"not-a-map"}}},
		specWith("empty", ""),
		specWith("nocolon", "noColonRef"),
		specWith("list", []interface{}{"real:REQ-001@1.0.0", 42}),
	})
	// empty supports contributes nothing; nocolon yields one ref (BundleName ""),
	// list yields one ref (the int is dropped). So exactly two refs harvested.
	if len(refs) != 2 {
		t.Fatalf("expected 2 harvested refs (nocolon + list), got %d: %v", len(refs), refs)
	}
	var sawNoColon, sawListReal bool
	for _, r := range refs {
		if r.Raw == "noColonRef" && r.BundleName == "" && r.ReqID == "noColonRef" {
			sawNoColon = true
		}
		if r.Raw == "real:REQ-001@1.0.0" && r.BundleName == "real" && r.Version == "1.0.0" {
			sawListReal = true
		}
	}
	if !sawNoColon {
		t.Error("expected the colon-less ref parsed with empty bundle name")
	}
	if !sawListReal {
		t.Error("expected the string element of the supports list harvested, the non-string dropped")
	}
}
