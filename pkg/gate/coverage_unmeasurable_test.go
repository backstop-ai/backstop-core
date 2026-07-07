package gate

import (
	"context"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// TestCoverage_ZeroStatementFileInMeasuredDir_NoUnmeasuredViolation (CLM-001):
// the fieldcontract.go-shaped case. pkg/pack/engine/fieldcontract.go is types +
// consts only (zero funcs), so a Go -coverprofile emits ZERO lines for it — it is
// un-measurable-by-construction (N/A), not unmeasured. The FIX is PRODUCER-SIDE and
// LANGUAGE-SPECIFIC (ISSUE-045 Resolution 1): the go-toolchain producer emits a
// total:0 N/A record for a zero-statement file in a MEASURED package (a Go measured
// package instruments every statement, so an absent file has genuinely zero
// statements). The language-neutral gate is UNCHANGED — a gate-side "no record in a
// measured directory => N/A" proxy is unsound cross-language (it would vacuous-green
// an lcov-omitted untested-but-has-statements file, defeating the bun
// anti-vacuous-green guard). This test therefore pins the CONSUMER half of the
// producer contract: given the producer's total:0 N/A record for the zero-statement
// file (alongside its measured sibling), the gate's EXISTING `Total==0 => N/A` guard
// treats it as N/A — NO coverage_unmeasured and NO coverage_threshold for it.
func TestCoverage_ZeroStatementFileInMeasuredDir_NoUnmeasuredViolation(t *testing.T) {
	root := t.TempDir()
	zeroStmt := "pkg/pack/engine/fieldcontract.go"
	sibling := "pkg/pack/engine/registry.go"
	// Both files present on disk so they survive ISSUE-034's os.Stat guard.
	writeSourceFile(t, root, zeroStmt, "package engine\n\ntype FieldContract struct{ Name string }\n\nconst DefaultField = \"x\"\n")
	writeSourceFile(t, root, sibling, "package engine\n\nfunc Register() int { return 1 }\n")

	// Repo-relative records exactly as the producer now emits them: the measured
	// SIBLING carries a real record, and the zero-statement file carries a total:0
	// N/A record (covered 0 / total 0) — the producer's zero-statement emission, NOT
	// a missing record.
	records := []check.CoverageRecord{
		{Path: sibling, Covered: 9, Total: 10, Measured: true, Metric: "statement"},
		{Path: zeroStmt, Covered: 0, Total: 0, Measured: true, Metric: "statement"},
	}

	scope := newGateScope(root, GateScopeModeDiff, []string{zeroStmt}, nil)
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(80), scope, goSourceClassifier(),
	)(context.Background())

	for _, v := range result.Violations {
		if v.File == zeroStmt {
			t.Fatalf("a zero-statement file carrying the producer's total:0 N/A record must be treated as N/A — no coverage_unmeasured and no coverage_threshold; got [%s] %s", v.Rule, v.Message)
		}
	}
}

// TestCoverage_FunctionsFileInUnmeasuredDir_StillFlagged (CLM-002): the case-1
// over-correction guard. A file WITH a real func, present on disk, in a directory
// that has NO coverage record at all (its whole package was never measured) is a
// GENUINE coverage gap and must STILL fire coverage_unmeasured (severity error).
// This proves the case-1 fix narrows to zero-content-in-a-MEASURED-dir only and
// does not blind the genuine unmeasured-package check — it must genuinely fail if
// the exclusion is widened to swallow any zero-record file.
func TestCoverage_FunctionsFileInUnmeasuredDir_StillFlagged(t *testing.T) {
	root := t.TempDir()
	lonely := "pkg/orphan/lonely.go"
	writeSourceFile(t, root, lonely, "package orphan\n\nfunc Lonely() int { return 1 }\n")

	// Records only for an UNRELATED directory; pkg/orphan has none.
	records := []check.CoverageRecord{
		{Path: "pkg/other/thing.go", Covered: 5, Total: 5, Measured: true, Metric: "statement"},
	}

	scope := newGateScope(root, GateScopeModeDiff, []string{lonely}, nil)
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(80), scope, goSourceClassifier(),
	)(context.Background())

	var found *Violation
	for i := range result.Violations {
		if result.Violations[i].Rule == "coverage_unmeasured" && result.Violations[i].File == lonely {
			found = &result.Violations[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("a functions file in an UNMEASURED directory (no record anywhere in its dir) must STILL fire coverage_unmeasured; got %#v", result.Violations)
	}
	if found.Severity != "error" {
		t.Errorf("the genuine unmeasured-package violation must be a blocking error, got severity %q", found.Severity)
	}
	if result.Status != "fail" {
		t.Errorf("a genuinely unmeasured source file must red the step, got status %q: %#v", result.Status, result.Violations)
	}
}
