package pack

import (
	"os"
	"strings"
	"testing"
)

// ISSUE-032 Defect A / ISSUE-030 fold deletion-assertion tests. They pin the ABSENCE
// of the dead native-standards scaffolder cluster so it cannot be silently
// reintroduced (the SPEC-034-style deletion guard). Parallel to
// pkg/validate/deletion_assertion_test.go.

// nonTestGoSources returns the contents of every non-test .go file in the current
// package directory (tests run with CWD == the package dir), keyed by file name, so
// the assertions scan production source only.
func nonTestGoSources(t *testing.T) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading package dir: %v", err)
	}
	out := map[string]string{}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		b, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		out[name] = string(b)
	}
	return out
}

// TestScaffolder_NoStandardsWriter proves no production source in pkg/pack writes a
// `standards/` directory, a `.standard.md` file, or stamps schema_version standard/v1
// — the retired native-standards scaffolder output (CLM-003). Comments referencing the
// deletion (the lineage tombstone) are allowed; live emitters are not.
func TestScaffolder_NoStandardsWriter(t *testing.T) {
	for name, src := range nonTestGoSources(t) {
		// The tombstone comment in scaffold.go legitimately mentions these tokens in
		// prose; scan for the CODE forms that would re-create the emitter.
		if strings.Contains(src, "scaffoldRulePack") {
			t.Errorf("%s still references scaffoldRulePack; the .standard.md scaffolder must be deleted", name)
		}
		if strings.Contains(src, "scaffoldCodePack") {
			t.Errorf("%s still references scaffoldCodePack; the .recipe.md scaffolder must be deleted", name)
		}
		if strings.Contains(src, "schema_version: standard/v1") {
			t.Errorf("%s still emits schema_version standard/v1; the standards emitter must be deleted", name)
		}
		if strings.Contains(src, `"standards"`) {
			t.Errorf("%s still writes to a standards/ directory; the standards writer must be deleted", name)
		}
	}
}

// TestScaffolder_ResolvePackNumberDeleted proves ResolvePackNumber (the standards/
// scan) is gone from every production source (CLM-003).
func TestScaffolder_ResolvePackNumberDeleted(t *testing.T) {
	for name, src := range nonTestGoSources(t) {
		if strings.Contains(src, "func ResolvePackNumber(") {
			t.Errorf("%s still defines ResolvePackNumber; the standards-numbering scan must be deleted", name)
		}
	}
}

// TestScaffolder_ValidTypesAreEngineModel proves ValidPackTypes reflects the live
// engine-model shapes and no longer carries the retired rule/code types (CLM-002).
func TestScaffolder_ValidTypesAreEngineModel(t *testing.T) {
	want := map[string]bool{"engine": true, "mechanism": true, "toolchain": true}
	if len(ValidPackTypes) != len(want) {
		t.Fatalf("ValidPackTypes = %v, want exactly %v", ValidPackTypes, want)
	}
	for k := range want {
		if !ValidPackTypes[k] {
			t.Errorf("ValidPackTypes missing %q", k)
		}
	}
	for _, retired := range []string{"rule", "code"} {
		if ValidPackTypes[retired] {
			t.Errorf("ValidPackTypes still carries retired type %q", retired)
		}
	}
}

// TestScaffolder_StandardV1SchemaDeleted proves the standard/v1 artifact schema is
// removed from the repo (CLM-003). It scans up from the package dir to the repo root.
func TestScaffolder_StandardV1SchemaDeleted(t *testing.T) {
	// pkg/pack -> repo root is two levels up.
	if _, err := os.Stat("../../artifacts/standard/v1/schema.json"); err == nil {
		t.Error("artifacts/standard/v1/schema.json still exists; the standard/v1 schema must be deleted")
	}
}
