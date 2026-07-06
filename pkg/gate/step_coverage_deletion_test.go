package gate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// goSourceClassifier is the Go-toolchain classification the deletion regression
// exercises: source **/*.go, test **/*_test.go — supplied explicitly because
// SPEC-043 de-Go'd the baked measurability literal (measurability now comes from
// pack-declared globs).
func goSourceClassifier() SourceClassifier {
	return NewSourceClassifier([]string{"**/*.go"}, []string{"**/*_test.go"})
}

// TestCoverage_DeletedInScopeFile_NoUnmeasuredViolation (CLM-001): a git-DELETED
// measurable-source .go path is in the diff scope but has NO file on disk. Because
// a deleted file cannot be measured, it must NOT carry a coverage_unmeasured
// obligation — the coverage step's on-disk existence guard excludes it from the
// coverage-required set entirely. This is the ISSUE-034 false positive that
// ISSUE-018's deletions (cmd/backstop/code_check.go, pkg/check/registry.go)
// surfaced. Fails before the fix: coveragePathsInScope keeps the glob-matching
// deleted path and the no-record scan reds it.
func TestCoverage_DeletedInScopeFile_NoUnmeasuredViolation(t *testing.T) {
	root := t.TempDir()
	deleted := "pkg/widget/deleted.go"
	// Deliberately DO NOT create the deleted file on disk — its absence IS the
	// deletion signal the fix keys on.

	scope := newGateScope(root, GateScopeModeDiff, []string{deleted}, nil)
	result := StepCoverageThresholdScopedFunc(
		[]check.CoverageRecord{}, coverageSpecs(80), scope, goSourceClassifier(),
	)(context.Background())

	for _, v := range result.Violations {
		if v.Rule == "coverage_unmeasured" && v.File == deleted {
			t.Fatalf("a git-deleted (not-on-disk) in-scope .go file must NOT produce a coverage_unmeasured violation; got %#v", v)
		}
	}
}

// TestCoverage_AddedUnmeasuredFile_StillFlagged (CLM-002): the over-correction
// guard. An ADDED measurable-source .go path that IS present on disk but has NO
// coverage record must STILL fire coverage_unmeasured (severity error). This proves
// the fix narrows to deletions (absent-on-disk) only and does not blind the genuine
// unmeasured-new-file check. It must genuinely fail if the existence guard is
// widened to swallow present-but-unmeasured files.
func TestCoverage_AddedUnmeasuredFile_StillFlagged(t *testing.T) {
	root := t.TempDir()
	added := "pkg/widget/added.go"
	writeSourceFile(t, root, added, "package widget\n\nfunc Added() int { return 1 }\n")

	scope := newGateScope(root, GateScopeModeDiff, []string{added}, nil)
	result := StepCoverageThresholdScopedFunc(
		[]check.CoverageRecord{}, coverageSpecs(80), scope, goSourceClassifier(),
	)(context.Background())

	var found *Violation
	for i := range result.Violations {
		if result.Violations[i].Rule == "coverage_unmeasured" && result.Violations[i].File == added {
			found = &result.Violations[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("an added, on-disk, measurable-source .go file with no coverage record must STILL fire coverage_unmeasured; got %#v", result.Violations)
	}
	if found.Severity != "error" {
		t.Errorf("the unmeasured-new-file violation must be a blocking error, got severity %q", found.Severity)
	}
	if result.Status != "fail" {
		t.Errorf("an unmeasured added source file must red the step, got status %q: %#v", result.Status, result.Violations)
	}
}

// TestCoverage_DeletedAndAddedInScope_OnlyAddedFlagged pins the two-sided behavior
// in one assertion, mirroring the real ISSUE-018 shape (a delete alongside a
// surviving genuine gap): with BOTH the deleted (absent) and the added (present)
// path in one diff scope and empty coverage, EXACTLY ONE coverage_unmeasured
// violation fires and it names the ADDED path, never the deleted one.
func TestCoverage_DeletedAndAddedInScope_OnlyAddedFlagged(t *testing.T) {
	root := t.TempDir()
	deleted := "pkg/widget/deleted.go"
	added := "pkg/widget/added.go"
	writeSourceFile(t, root, added, "package widget\n\nfunc Added() int { return 1 }\n")

	scope := newGateScope(root, GateScopeModeDiff, []string{deleted, added}, nil)
	result := StepCoverageThresholdScopedFunc(
		[]check.CoverageRecord{}, coverageSpecs(80), scope, goSourceClassifier(),
	)(context.Background())

	var unmeasured []Violation
	for _, v := range result.Violations {
		if v.Rule == "coverage_unmeasured" {
			unmeasured = append(unmeasured, v)
		}
	}
	if len(unmeasured) != 1 {
		t.Fatalf("expected EXACTLY ONE coverage_unmeasured violation (the added path only), got %d: %#v", len(unmeasured), unmeasured)
	}
	if unmeasured[0].File != added {
		t.Errorf("the sole coverage_unmeasured violation must name the added path %q, not %q", added, unmeasured[0].File)
	}
}

// writeSourceFile creates a non-empty Go source file at root-relative rel, making
// the file genuinely present on disk so the coverage step's existence guard treats
// it as a not-deleted (added/modified) path.
func writeSourceFile(t *testing.T, root, rel, content string) {
	t.Helper()
	abs := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", rel, err)
	}
	if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", rel, err)
	}
}
