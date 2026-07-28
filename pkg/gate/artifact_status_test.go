package gate_test

// Phase-1 substrate tests (ISSUE-042 TASK-001, CLM-001/002/003/013/014). This file
// is DELIBERATELY the external `gate_test` package: it may touch ONLY the exported
// resolver surface (ResolveArtifactStatus / ArtifactStatusRecord / ClassifyArtifactStatus
// / the Kind* + Class* constants / BackingPlan), which mechanically proves ISSUE-043 can
// reuse the status+mandated-test resolution without reaching into step internals
// (CLM-013). It reuses ExtractMandatedTests machinery indirectly through the resolver.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// writeEdgeFile writes a fixture file (creating parent dirs) for the edge-path tests.
func writeEdgeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// fixtureRoot is the cross-type artifact fixture tree. A function (not a package-level
// var/const) to keep the test file free of package-level mutable state.
func fixtureRoot() string { return "testdata/artifact_status" }

// recordByID finds the resolved record for an artifact id, or fails the test.
func recordByID(t *testing.T, res *gate.ArtifactStatusResolution, id string) gate.ArtifactStatusRecord {
	t.Helper()
	for _, r := range res.Records {
		if r.ID == id {
			return r
		}
	}
	t.Fatalf("no resolved record for %q (have %d records)", id, len(res.Records))
	return gate.ArtifactStatusRecord{}
}

// testNames flattens a record's mandated tests to their function names.
func testNames(r gate.ArtifactStatusRecord) []string {
	out := make([]string, 0, len(r.MandatedTests))
	for _, mt := range r.MandatedTests {
		out = append(out, mt.FuncName)
	}
	return out
}

func containsStr(hay []string, needle string) bool {
	for _, h := range hay {
		if h == needle {
			return true
		}
	}
	return false
}

// TestResolveArtifactStatus_ParsesAllArtifactKinds (CLM-001): the resolver walks
// issues/ specs/ directives/ plans/ under a fixture root and returns one record per
// artifact carrying (id, kind, declared status, mandated test names). An issue's and a
// spec's claims->tests are both surfaced (via the reused ExtractMandatedTests
// machinery for specs, the same claim->tests walk for issues); a directive/plan record
// carries its declared status even with no own tests.
func TestResolveArtifactStatus_ParsesAllArtifactKinds(t *testing.T) {
	res, err := gate.ResolveArtifactStatus(fixtureRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	if len(res.Records) == 0 {
		t.Fatal("expected resolved records across all artifact kinds, got none")
	}

	// Issue: open, non-terminal, claims->tests surfaced.
	iss := recordByID(t, res, "ISSUE-900")
	if iss.Kind != gate.KindIssue {
		t.Errorf("ISSUE-900 kind = %q, want %q", iss.Kind, gate.KindIssue)
	}
	if iss.Status != "open" {
		t.Errorf("ISSUE-900 status = %q, want open", iss.Status)
	}
	if !containsStr(testNames(iss), "TestFixturePresentAlpha") {
		t.Errorf("ISSUE-900 mandated tests = %v, want to contain TestFixturePresentAlpha", testNames(iss))
	}

	// Spec: implemented, tests surfaced via ExtractMandatedTests machinery.
	spec := recordByID(t, res, "SPEC-900")
	if spec.Kind != gate.KindSpec {
		t.Errorf("SPEC-900 kind = %q, want %q", spec.Kind, gate.KindSpec)
	}
	if spec.Status != "implemented" {
		t.Errorf("SPEC-900 status = %q, want implemented", spec.Status)
	}
	if !containsStr(testNames(spec), "TestFixtureSpecGamma") {
		t.Errorf("SPEC-900 mandated tests = %v, want to contain TestFixtureSpecGamma", testNames(spec))
	}

	// Plan: carries declared status even with no own mandated tests.
	plan := recordByID(t, res, "PLAN-ISSUE-900")
	if plan.Kind != gate.KindPlan {
		t.Errorf("PLAN-ISSUE-900 kind = %q, want %q", plan.Kind, gate.KindPlan)
	}
	if plan.Status != "completed" {
		t.Errorf("PLAN-ISSUE-900 status = %q, want completed", plan.Status)
	}
	if len(plan.MandatedTests) != 0 {
		t.Errorf("PLAN-ISSUE-900 mandated tests = %v, want none", testNames(plan))
	}

	// Directive: carries declared status even with no own mandated tests.
	dir := recordByID(t, res, "DIR-900")
	if dir.Kind != gate.KindDirective {
		t.Errorf("DIR-900 kind = %q, want %q", dir.Kind, gate.KindDirective)
	}
	if dir.Status != "done" {
		t.Errorf("DIR-900 status = %q, want done", dir.Status)
	}
	if len(dir.MandatedTests) != 0 {
		t.Errorf("DIR-900 mandated tests = %v, want none", testNames(dir))
	}
}

// TestClassifyArtifactStatus_ThreeClassesPerType (CLM-002): every status enum value for
// each artifact type maps to exactly one of {non-terminal, success-terminal,
// retired-terminal}. Includes the explicit negative: spec `implemented` (NOT the phantom
// `active`, which is a DIRECTIVE status) is success-terminal.
func TestClassifyArtifactStatus_ThreeClassesPerType(t *testing.T) {
	cases := []struct {
		kind   gate.ArtifactKind
		status string
		want   gate.StatusClass
	}{
		// issue: open/ready/in-progress/blocked -> non-terminal; closed -> success;
		// replaced/canceled -> retired.
		{gate.KindIssue, "open", gate.ClassNonTerminal},
		{gate.KindIssue, "ready", gate.ClassNonTerminal},
		{gate.KindIssue, "in-progress", gate.ClassNonTerminal},
		{gate.KindIssue, "blocked", gate.ClassNonTerminal},
		{gate.KindIssue, "closed", gate.ClassSuccessTerminal},
		{gate.KindIssue, "replaced", gate.ClassRetiredTerminal},
		{gate.KindIssue, "canceled", gate.ClassRetiredTerminal},
		// spec: draft/ready-for-implementation -> non-terminal; implemented -> success;
		// replaced/canceled/deprecated -> retired. There is NO spec `active`.
		{gate.KindSpec, "draft", gate.ClassNonTerminal},
		{gate.KindSpec, "ready-for-implementation", gate.ClassNonTerminal},
		{gate.KindSpec, "implemented", gate.ClassSuccessTerminal},
		{gate.KindSpec, "replaced", gate.ClassRetiredTerminal},
		{gate.KindSpec, "canceled", gate.ClassRetiredTerminal},
		{gate.KindSpec, "deprecated", gate.ClassRetiredTerminal},
		// directive: queued/active/specced -> non-terminal; done -> success;
		// replaced/canceled -> retired.
		{gate.KindDirective, "queued", gate.ClassNonTerminal},
		{gate.KindDirective, "active", gate.ClassNonTerminal},
		{gate.KindDirective, "specced", gate.ClassNonTerminal},
		{gate.KindDirective, "done", gate.ClassSuccessTerminal},
		{gate.KindDirective, "replaced", gate.ClassRetiredTerminal},
		{gate.KindDirective, "canceled", gate.ClassRetiredTerminal},
		// plan: draft/ready/implementing -> non-terminal; completed -> success;
		// replaced/canceled -> retired.
		{gate.KindPlan, "draft", gate.ClassNonTerminal},
		{gate.KindPlan, "ready", gate.ClassNonTerminal},
		{gate.KindPlan, "implementing", gate.ClassNonTerminal},
		{gate.KindPlan, "completed", gate.ClassSuccessTerminal},
		{gate.KindPlan, "replaced", gate.ClassRetiredTerminal},
		{gate.KindPlan, "canceled", gate.ClassRetiredTerminal},
	}
	for _, c := range cases {
		got := gate.ClassifyArtifactStatus(c.kind, c.status)
		if got != c.want {
			t.Errorf("ClassifyArtifactStatus(%q, %q) = %v, want %v", c.kind, c.status, got, c.want)
		}
	}

	// Explicit negative: spec `implemented` is success-terminal, NOT misread as the
	// phantom spec `active` (which does not exist for specs).
	if gate.ClassifyArtifactStatus(gate.KindSpec, "implemented") != gate.ClassSuccessTerminal {
		t.Error("spec `implemented` must classify success-terminal (there is no spec `active`)")
	}
	// And `active` is a DIRECTIVE non-terminal, never a spec success.
	if gate.ClassifyArtifactStatus(gate.KindDirective, "active") != gate.ClassNonTerminal {
		t.Error("directive `active` must classify non-terminal")
	}
}

// TestResolveArtifactStatus_RetiredStaysExcludable (CLM-003): replaced/canceled/deprecated
// classify retired-terminal per TYPE, so the drift step can exclude them — WITHOUT the
// classifier being the folded isTerminalSpecStatus (it is separate and type-aware, and it
// also distinguishes success-terminal, which isTerminalSpecStatus never does).
func TestResolveArtifactStatus_RetiredStaysExcludable(t *testing.T) {
	retired := []struct {
		kind   gate.ArtifactKind
		status string
	}{
		{gate.KindIssue, "replaced"},
		{gate.KindIssue, "canceled"},
		{gate.KindSpec, "replaced"},
		{gate.KindSpec, "canceled"},
		{gate.KindSpec, "deprecated"},
		{gate.KindPlan, "replaced"},
		{gate.KindDirective, "canceled"},
	}
	for _, r := range retired {
		if got := gate.ClassifyArtifactStatus(r.kind, r.status); got != gate.ClassRetiredTerminal {
			t.Errorf("ClassifyArtifactStatus(%q, %q) = %v, want retired-terminal", r.kind, r.status, got)
		}
	}

	// Type-aware, NOT folded: unlike isTerminalSpecStatus (which folds ALL terminals
	// together and never names success), this classifier separates success-terminal
	// from retired-terminal — proving it is a distinct classifier, not a reuse of the
	// folded exclusion.
	if gate.ClassifyArtifactStatus(gate.KindSpec, "implemented") == gate.ClassRetiredTerminal {
		t.Error("spec `implemented` must NOT fold into retired-terminal — success-terminal is a distinct class")
	}
	if gate.ClassifyArtifactStatus(gate.KindIssue, "closed") != gate.ClassSuccessTerminal {
		t.Error("issue `closed` must classify success-terminal, distinct from retired")
	}

	// The resolver surfaces the retired fixture spec's class as retired-terminal so the
	// drift step can exclude it.
	res, err := gate.ResolveArtifactStatus(fixtureRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	rec := recordByID(t, res, "SPEC-902")
	if rec.Class != gate.ClassRetiredTerminal {
		t.Errorf("SPEC-902 (replaced) resolved class = %v, want retired-terminal", rec.Class)
	}
}

// TestResolveArtifactStatus_ExportedForReuse (CLM-013): the resolver entrypoint and
// record type are exported and callable as an external package consumer (this whole file
// is package gate_test) — proving ISSUE-043 can reuse the status+mandated-test resolution
// without touching step internals.
func TestResolveArtifactStatus_ExportedForReuse(t *testing.T) {
	// Exercise the exported surface exactly as ISSUE-043 would: resolve, read records,
	// classify, and resolve backing plans — all via exported identifiers. The results are
	// bound to explicitly-typed vars so a rename/unexport of either type would break this
	// external consumer at compile time (the reuse guarantee, CLM-013).
	var res *gate.ArtifactStatusResolution
	var err error
	res, err = gate.ResolveArtifactStatus(fixtureRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus not usable as an exported entrypoint: %v", err)
	}
	if res == nil {
		t.Fatal("ResolveArtifactStatus returned nil resolution")
	}

	rec := recordByID(t, res, "ISSUE-901")
	if rec.Class != gate.ClassifyArtifactStatus(rec.Kind, rec.Status) {
		t.Error("record Class must equal ClassifyArtifactStatus(Kind, Status) — the classifier is the exported source of truth")
	}
	if rec.Class != gate.ClassSuccessTerminal {
		t.Errorf("ISSUE-901 (closed) class = %v, want success-terminal", rec.Class)
	}
}

// TestResolveArtifactStatus_IssueToPlanLinkage (CLM-014): the resolver indexes plans by
// spec_id and exposes, for ISSUE-NNN, the backing plan whose spec_id == ISSUE-NNN as a
// REAL output. An issue with no backing plan resolves to empty (no panic). This is the
// linkage ISSUE-043's delivered_by validator reuses.
func TestResolveArtifactStatus_IssueToPlanLinkage(t *testing.T) {
	res, err := gate.ResolveArtifactStatus(fixtureRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}

	plan, ok := res.BackingPlan("ISSUE-900")
	if !ok {
		t.Fatal("BackingPlan(ISSUE-900) not found — PLAN-ISSUE-900 (spec_id ISSUE-900) should be the delivering plan")
	}
	if plan.ID != "PLAN-ISSUE-900" {
		t.Errorf("BackingPlan(ISSUE-900).ID = %q, want PLAN-ISSUE-900", plan.ID)
	}
	if plan.Kind != gate.KindPlan {
		t.Errorf("backing plan kind = %q, want %q", plan.Kind, gate.KindPlan)
	}

	// An issue with no backing plan resolves to empty without panicking.
	if _, ok := res.BackingPlan("ISSUE-902"); ok {
		t.Error("BackingPlan(ISSUE-902) should be empty — no PLAN-ISSUE-902 exists in the fixture")
	}
}

// TestStatusClass_String covers StatusClass.String for every class plus the unknown fallback.
func TestStatusClass_String(t *testing.T) {
	cases := []struct {
		c    gate.StatusClass
		want string
	}{
		{gate.ClassNonTerminal, "non-terminal"},
		{gate.ClassSuccessTerminal, "success-terminal"},
		{gate.ClassRetiredTerminal, "retired-terminal"},
		{gate.StatusClass(99), "unknown-status-class"},
	}
	for _, c := range cases {
		if got := c.c.String(); got != c.want {
			t.Errorf("StatusClass(%d).String() = %q, want %q", int(c.c), got, c.want)
		}
	}
}

// TestBackingPlan_EmptyAndNilReceiver covers the BackingPlan guard branches: a nil
// resolution and an unknown id both return false without panicking.
func TestBackingPlan_EmptyAndNilReceiver(t *testing.T) {
	var nilRes *gate.ArtifactStatusResolution
	if _, ok := nilRes.BackingPlan("ISSUE-900"); ok {
		t.Error("nil resolution BackingPlan must return (empty, false)")
	}
	res, err := gate.ResolveArtifactStatus(fixtureRoot())
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	if _, ok := res.BackingPlan("DOES-NOT-EXIST"); ok {
		t.Error("BackingPlan for an unknown id must return (empty, false)")
	}
}

// TestResolveArtifactStatus_EdgeFiles covers the walk skip branches: a partial tree
// (missing plans/directives dirs), a non-matching-suffix file, a fence-less markdown file,
// and malformed YAML in both a fenced artifact and a plan — all skipped, no error, with the
// one well-formed artifact still resolved.
func TestResolveArtifactStatus_EdgeFiles(t *testing.T) {
	root := t.TempDir()
	writeEdgeFile(t, root, "issues/ISSUE-800.issue.md", "---\nissue:\n  id: ISSUE-800\n  status: closed\nclaims:\n  - id: CLM-001\n    tests:\n      - TestEdgeAbsent\n---\nbody\n")
	writeEdgeFile(t, root, "issues/notes.txt", "not an artifact file\n")
	writeEdgeFile(t, root, "issues/nofence.issue.md", "no opening fence here\njust text\n")
	writeEdgeFile(t, root, "issues/bad.issue.md", "---\nissue: [this is: not a valid mapping\n---\nbody\n")
	writeEdgeFile(t, root, "plans/bad.plan.yml", "plan_id: [oops not valid\n")

	res, err := gate.ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("partial/edge tree must resolve without error: %v", err)
	}
	got := recordByID(t, res, "ISSUE-800")
	if got.Class != gate.ClassSuccessTerminal {
		t.Errorf("ISSUE-800 (closed) class = %v, want success-terminal", got.Class)
	}
	// Exactly one record: every malformed/edge file was skipped.
	if len(res.Records) != 1 {
		t.Errorf("expected only the well-formed ISSUE-800 record, got %d: %+v", len(res.Records), res.Records)
	}
}

// TestResolveArtifactStatus_ErrorWhenDirIsFile covers the fail-loud error path: when an
// artifact directory is actually a regular file, ResolveArtifactStatus surfaces a wrapped
// error rather than silently skipping.
func TestResolveArtifactStatus_ErrorWhenDirIsFile(t *testing.T) {
	root := t.TempDir()
	writeEdgeFile(t, root, "specs", "i am a file, not a directory\n")
	if _, err := gate.ResolveArtifactStatus(root); err == nil {
		t.Fatal("expected a wrapped error when an artifact dir is actually a file")
	}
}

// TestResolveArtifactStatus_IssuesOnlyTree covers the missing-directory skip branch across
// specs/plans/directives: a root with ONLY an issues/ dir resolves cleanly (each absent
// dir is a no-op, not an error).
func TestResolveArtifactStatus_IssuesOnlyTree(t *testing.T) {
	root := t.TempDir()
	writeEdgeFile(t, root, "issues/ISSUE-810.issue.md", "---\nissue:\n  id: ISSUE-810\n  status: open\n---\nbody\n")

	res, err := gate.ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("issues-only tree must resolve without error: %v", err)
	}
	if len(res.Records) != 1 {
		t.Fatalf("expected exactly the one issue record, got %d", len(res.Records))
	}
	if _, ok := res.BackingPlan("ISSUE-810"); ok {
		t.Error("no plans dir -> BackingPlan must be empty")
	}
}
