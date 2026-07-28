package validate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// PLAN-SPEC-051 TASK-001 — corpus-assertion harness. These tests consume
// SPEC-050's public resolution + version-log functions over the REAL project
// corpus to prove the POST-SWEEP end state (deprecations, version stamps, pins,
// clean resolution). They assert the reconciled corpus, so they stay RED until
// SPEC-051's backfill sweep co-lands with SPEC-050's enforcement flip — that red
// is the co-land signal, not a defect in this file.

func reconcileRepoRoot() string { return filepath.Join("..", "..") }

// loadCorpusDir parses every artifact under <repo>/<sub> whose name ends in
// suffix.
func loadCorpusDir(t *testing.T, sub, suffix string) []*artifact.ParsedArtifact {
	t.Helper()
	root := filepath.Join(reconcileRepoRoot(), sub)
	var arts []*artifact.ParsedArtifact
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(info.Name(), suffix) {
			return nil
		}
		art, perr := artifact.ParseFile(path)
		if perr != nil {
			t.Fatalf("parsing %s: %v", path, perr)
		}
		arts = append(arts, art)
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	return arts
}

func loadBundles(t *testing.T) []*artifact.ParsedArtifact {
	return loadCorpusDir(t, "bundles", ".bundle.md")
}
func loadSpecs(t *testing.T) []*artifact.ParsedArtifact {
	return loadCorpusDir(t, "specs", ".spec.md")
}
func loadIssues(t *testing.T) []*artifact.ParsedArtifact {
	return loadCorpusDir(t, "issues", ".issue.md")
}

func findBundleByName(arts []*artifact.ParsedArtifact, name string) *artifact.ParsedArtifact {
	for _, a := range arts {
		if bundleName(a) == name {
			return a
		}
	}
	return nil
}

func findSpecByNumber(arts []*artifact.ParsedArtifact, number string) *artifact.ParsedArtifact {
	for _, a := range arts {
		if a.Metadata["number"] == number {
			return a
		}
	}
	return nil
}

func hasRuleViolation(vs []Violation, rule string) bool {
	for _, v := range vs {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// specSupportRefs harvests the supports refs from a single spec's requirements
// (bypassing the terminal-citer skip so terminal-spec assertions can inspect the
// raw refs directly).
func specSupportRefs(art *artifact.ParsedArtifact) []SupportRef {
	var refs []SupportRef
	reqsVal, ok := art.Frontmatter["requirements"]
	if !ok {
		return refs
	}
	reqs, ok := reqsVal.([]interface{})
	if !ok {
		return refs
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
		for _, s := range supportsValues(supVal) {
			if strings.TrimSpace(s) == "" {
				continue
			}
			refs = append(refs, parseSupportRef(s, art.Filename, "requirements["+itoa(i)+"]"))
		}
	}
	return refs
}

func itoa(i int) string {
	if i == 0 {
		return "0"
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	return string(b)
}

// --- CLM-001..004: agent-definitions retirement ---

func TestReconcile_AgentDefsDeprecated(t *testing.T) {
	b := findBundleByName(loadBundles(t), "agent-definitions")
	if b == nil {
		t.Fatal("agent-definitions bundle not found")
	}
	if got := extractMaturity(b); got != "deprecated" {
		t.Errorf("agent-definitions maturity = %q, want deprecated", got)
	}
}

func TestReconcile_AgentDefsTerminalExemptFromRequirementsGate(t *testing.T) {
	b := findBundleByName(loadBundles(t), "agent-definitions")
	if b == nil {
		t.Fatal("agent-definitions bundle not found")
	}
	res := Bundle(b, &schema.Schema{ArtifactType: "bundle"})
	if hasRuleViolation(res.Violations, "bundle/requirements-required") {
		t.Error("deprecated agent-definitions must be terminal-exempt from the requirements-required gate")
	}
}

func TestReconcile_AgentDefsNoBackfilledRequirements(t *testing.T) {
	b := findBundleByName(loadBundles(t), "agent-definitions")
	if b == nil {
		t.Fatal("agent-definitions bundle not found")
	}
	if reqs, ok := b.Frontmatter["requirements"]; ok {
		if list, ok := reqs.([]interface{}); ok && len(list) > 0 {
			t.Errorf("agent-definitions must carry no backfilled requirements[], found %d", len(list))
		}
	}
}

func TestReconcile_AgentDefsTerminalOutsideCoverageEval(t *testing.T) {
	b := findBundleByName(loadBundles(t), "agent-definitions")
	if b == nil {
		t.Fatal("agent-definitions bundle not found")
	}
	if !isTerminalStatus(extractMaturity(b)) {
		t.Error("agent-definitions must be terminal so it is outside the SPEC-052 delivered-coverage evaluation")
	}
}

// --- CLM-005..007: BUNDLE-011 re-home to implemented specs ---

func TestReconcile_Bundle011Req004CoveredBySpec040Req001(t *testing.T) {
	specs := loadSpecs(t)
	spec040 := findSpecByNumber(specs, "SPEC-040")
	if spec040 == nil {
		t.Fatal("SPEC-040 not found")
	}
	catalog := BuildBundleReqCatalog(loadBundles(t))
	ref := findRef(specSupportRefs(spec040), "collapse-legacy-codecheck-into-packs", "REQ-004")
	if ref == nil {
		t.Fatal("SPEC-040 does not cite collapse-legacy-codecheck-into-packs:REQ-004")
	}
	if v := ResolveSupports(catalog, []SupportRef{*ref}); len(v) != 0 {
		t.Errorf("SPEC-040 REQ-004 ref must resolve clean, got: %v", v)
	}
}

func TestReconcile_Bundle011Req007CoveredBySpec053(t *testing.T) {
	assertReqReHomed(t, "REQ-007")
}

func TestReconcile_Bundle011Req010CoveredBySpec053(t *testing.T) {
	assertReqReHomed(t, "REQ-010")
}

// assertReqReHomed asserts collapse-legacy-codecheck-into-packs:<reqID> is cited
// by the implemented SPEC-053 (resolving clean) and NOT by SPEC-040 or SPEC-039.
func assertReqReHomed(t *testing.T, reqID string) {
	t.Helper()
	specs := loadSpecs(t)
	catalog := BuildBundleReqCatalog(loadBundles(t))
	spec053 := findSpecByNumber(specs, "SPEC-053")
	if spec053 == nil {
		t.Fatal("SPEC-053 not found")
	}
	ref := findRef(specSupportRefs(spec053), "collapse-legacy-codecheck-into-packs", reqID)
	if ref == nil {
		t.Fatalf("SPEC-053 does not cite collapse-legacy-codecheck-into-packs:%s", reqID)
	}
	if v := ResolveSupports(catalog, []SupportRef{*ref}); len(v) != 0 {
		t.Errorf("SPEC-053 %s ref must resolve clean, got: %v", reqID, v)
	}
	if spec040 := findSpecByNumber(specs, "SPEC-040"); spec040 != nil {
		if findRef(specSupportRefs(spec040), "collapse-legacy-codecheck-into-packs", reqID) != nil {
			t.Errorf("SPEC-040 must NOT cite collapse-legacy-codecheck-into-packs:%s (re-homed to SPEC-053)", reqID)
		}
	}
	// SPEC-039 is `replaced` and left untouched (CLM-008): it may still TEXTUALLY
	// carry the ref, but as a terminal citer it is skipped by CollectSupportRefs,
	// so it is not the LIVE coverer. Assert its terminal status rather than the
	// ref's absence.
	if spec039 := findSpecByNumber(specs, "SPEC-039"); spec039 != nil {
		if !isTerminalStatus(spec039.Metadata["status"]) {
			t.Errorf("SPEC-039 must be terminal (replaced) so it is not the live coverer of %s", reqID)
		}
		if refs := CollectSupportRefs([]*artifact.ParsedArtifact{spec039}); len(refs) != 0 {
			t.Errorf("terminal SPEC-039 must be skipped by the harvest, so it cannot cover %s (got %d refs)", reqID, len(refs))
		}
	}
}

func findRef(refs []SupportRef, bundle, req string) *SupportRef {
	for i := range refs {
		if refs[i].BundleName == bundle && refs[i].ReqID == req {
			return &refs[i]
		}
	}
	return nil
}

func TestReconcile_Spec039LeftUntouched(t *testing.T) {
	spec039 := findSpecByNumber(loadSpecs(t), "SPEC-039")
	if spec039 == nil {
		t.Fatal("SPEC-039 not found")
	}
	if spec039.Metadata["status"] != "replaced" {
		t.Errorf("SPEC-039 status = %q, want replaced (left untouched)", spec039.Metadata["status"])
	}
	for _, ref := range specSupportRefs(spec039) {
		if ref.Pinned {
			t.Errorf("SPEC-039 ref %q must remain unpinned (left untouched)", ref.Raw)
		}
	}
}

// --- CLM-009..013: SPEC-002/003/004 retirement ---

func TestReconcile_Spec002Deprecated(t *testing.T) { assertSpecDeprecated(t, "SPEC-002") }
func TestReconcile_Spec003Deprecated(t *testing.T) { assertSpecDeprecated(t, "SPEC-003") }
func TestReconcile_Spec004Deprecated(t *testing.T) { assertSpecDeprecated(t, "SPEC-004") }

func assertSpecDeprecated(t *testing.T, number string) {
	t.Helper()
	s := findSpecByNumber(loadSpecs(t), number)
	if s == nil {
		t.Fatalf("%s not found", number)
	}
	if s.Metadata["status"] != "deprecated" {
		t.Errorf("%s status = %q, want deprecated", number, s.Metadata["status"])
	}
}

func TestReconcile_RetiredDraftsExemptFromDanglingRefs(t *testing.T) {
	specs := loadSpecs(t)
	for _, number := range []string{"SPEC-002", "SPEC-003", "SPEC-004"} {
		s := findSpecByNumber(specs, number)
		if s == nil {
			t.Fatalf("%s not found", number)
		}
		res := Spec(s, &schema.Schema{ArtifactType: "spec"})
		if hasRuleViolation(res.Violations, "spec/requirement-supports-format") {
			t.Errorf("%s (retired) must be terminal-exempt from supports-format on its dangling refs", number)
		}
	}
}

func TestReconcile_RetiredDraftsNotImplemented(t *testing.T) {
	specs := loadSpecs(t)
	for _, number := range []string{"SPEC-002", "SPEC-003", "SPEC-004"} {
		s := findSpecByNumber(specs, number)
		if s == nil {
			t.Fatalf("%s not found", number)
		}
		if s.Metadata["status"] == "implemented" {
			t.Errorf("%s must be retired, not implemented", number)
		}
	}
}

// --- CLM-014..015: bundle version + version-log backfill ---

func TestReconcile_AllNonTerminalBundleReqsVersioned(t *testing.T) {
	for _, b := range loadBundles(t) {
		if isTerminalStatus(extractMaturity(b)) {
			continue
		}
		reqsVal, ok := b.Frontmatter["requirements"]
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
			v, has := getStringField(req, "version")
			if !has || !semverRe.MatchString(v) {
				t.Errorf("%s requirements[%d] version = %q, want well-formed semver", b.Filename, i, v)
			}
		}
	}
}

func TestReconcile_NoBundleVersionLogViolationsCorpusWide(t *testing.T) {
	versionRules := map[string]bool{
		"bundle/requirement-version-required":      true,
		"bundle/requirement-version-format":        true,
		"bundle/requirement-versions-empty":        true,
		"bundle/requirement-versions-format":       true,
		"bundle/requirement-versions-entry-format": true,
		"bundle/requirement-versions-entry-text":   true,
		"bundle/requirement-versions-nonmonotonic": true,
		"bundle/requirement-versions-duplicate":    true,
		"bundle/requirement-version-not-newest":    true,
		"bundle/requirement-text-not-newest":       true,
	}
	for _, b := range loadBundles(t) {
		res := Bundle(b, &schema.Schema{ArtifactType: "bundle"})
		for _, v := range res.Violations {
			if versionRules[v.Rule] {
				t.Errorf("%s: unexpected version-log violation [%s] %s", b.Filename, v.Rule, v.Message)
			}
		}
	}
}

// --- CLM-016..018: pin backfill on live specs ---

func TestReconcile_AllLiveSupportsRefsPinned(t *testing.T) {
	for _, s := range loadSpecs(t) {
		if isTerminalStatus(s.Metadata["status"]) {
			continue
		}
		for _, ref := range specSupportRefs(s) {
			if !ref.Pinned {
				t.Errorf("%s supports ref %q must carry an @MAJOR.MINOR.PATCH pin", s.Filename, ref.Raw)
			}
		}
	}
}

func TestReconcile_NoUnpinnedSupportsRefViolations(t *testing.T) {
	for _, s := range loadSpecs(t) {
		if isTerminalStatus(s.Metadata["status"]) {
			continue
		}
		res := Spec(s, &schema.Schema{ArtifactType: "spec"})
		if hasRuleViolation(res.Violations, "spec/requirement-supports-format") {
			t.Errorf("%s: tightened supportsRe raised a pin-format violation on a live spec", s.Filename)
		}
	}
}

func TestReconcile_SeedSpecsSelfPinnedAt100(t *testing.T) {
	specs := loadSpecs(t)
	for _, number := range []string{"SPEC-050", "SPEC-051", "SPEC-052"} {
		s := findSpecByNumber(specs, number)
		if s == nil {
			t.Fatalf("%s not found", number)
		}
		refs := specSupportRefs(s)
		if len(refs) == 0 {
			t.Errorf("%s should carry self-referential supports refs", number)
		}
		// Mixed-pin rule (CLM-018, amended v1.3.1): each self-ref is pinned at the
		// CURRENT version of the bundle REQ it implements — @1.0.0 for
		// requirement-traceability:REQ-001..014, @1.1.0 for REQ-015 (the RDQ-2
		// amendment; an in-flight bundle's citers rev to the latest on a minor bump).
		for _, ref := range refs {
			want := "1.0.0"
			if ref.ReqID == "REQ-015" {
				want = "1.1.0"
			}
			if !ref.Pinned || ref.Version != want {
				t.Errorf("%s ref %q must be self-pinned at @%s (current version of %s)", number, ref.Raw, want, ref.ReqID)
			}
		}
	}
}

// --- CLM-019..020: corpus resolves green; enforcement genuinely on ---

func TestReconcile_CorpusResolvesGreenUnderEnforcement(t *testing.T) {
	catalog := BuildBundleReqCatalog(loadBundles(t))
	citers := append(loadSpecs(t), loadIssues(t)...)
	refs := CollectSupportRefs(citers)
	if v := ResolveSupports(catalog, refs); len(v) != 0 {
		t.Errorf("corpus must resolve green under enforcement, got %d violations:", len(v))
		for _, vv := range v {
			t.Errorf("  [%s] %s (%s)", vv.Rule, vv.Message, vv.File)
		}
	}
}

func TestReconcile_UnpinnedAndFabricatedRefsStillRedUnderEnforcement(t *testing.T) {
	catalog := BuildBundleReqCatalog(loadBundles(t))
	bundle, req := sampleCatalogReq(catalog)
	if bundle == "" {
		t.Fatal("no bundle REQ available in the catalog to build the enforcement probes")
	}

	// A still-unpinned ref fails the tightened format check under enforcement.
	if supportsRe.MatchString(bundle + ":" + req) {
		t.Errorf("an unpinned ref %s:%s must still fail supportsRe under enforcement", bundle, req)
	}

	// A fabricated-version pin fails resolution under enforcement.
	fabricated := parseSupportRef(bundle+":"+req+"@99.99.99", "probe.spec.md", "requirements[0]")
	if v := ResolveSupports(catalog, []SupportRef{fabricated}); !hasRuleViolation(v, "supports/version-unlogged") {
		t.Errorf("a fabricated-version ref must fail resolution (version-unlogged), got: %v", v)
	}
}

// sampleCatalogReq returns any (bundle, req) pair present in the catalog.
func sampleCatalogReq(c *BundleReqCatalog) (string, string) {
	for name, reqs := range c.bundles {
		for req := range reqs {
			return name, req
		}
	}
	return "", ""
}
