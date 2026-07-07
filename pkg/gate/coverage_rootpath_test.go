package gate

import (
	"context"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// The PRODUCTION false positive is that Go's -coverprofile emits module-qualified
// record paths (github.com/bmanson/backstop-core/embed.go), which the go-toolchain
// producer now strips to repo-relative (TASK-004), verified by the final gate run.
// These unit tests pin the CONSUMER's half of the repo-relative contract: given
// repo-relative records, a measured root-package file resolves to its OWN record
// under a same-basename collision, and an unmeasured root file must not borrow a
// same-basename nested sibling's record.

// TestCoverage_MeasuredRootFile_UnderBasenameCollision_NoUnmeasured (CLM-003):
// scope is a ROOT-package file whose gate-scope path is a bare basename with zero
// directory segments (embed.go). A real same-basename file exists elsewhere in the
// module (cmd/backstop/embed.go). Both are measured. Before the fix the exact match
// misses (nothing keyed "embed.go" when producers were module-qualified) and the
// bare-basename suffix "/embed.go" matches BOTH records (found==2 => nil,false),
// discarding the root file's own record and firing coverage_unmeasured. After the
// fix — repo-relative records + exact-match-only for bare basenames — embed.go
// resolves to its OWN 9/10 record: no unmeasured, and 9/10 >= 90 so no
// coverage_threshold either.
func TestCoverage_MeasuredRootFile_UnderBasenameCollision_NoUnmeasured(t *testing.T) {
	root := t.TempDir()
	rootFile := "embed.go"
	nested := "cmd/backstop/embed.go"
	writeSourceFile(t, root, rootFile, "package main\n\nfunc ListSchemas() int { return 1 }\n")
	writeSourceFile(t, root, nested, "package backstop\n\nfunc Embed() int { return 1 }\n")

	// Repo-relative records reproducing the collision directly: the root file and a
	// same-basename nested file both measured.
	records := []check.CoverageRecord{
		{Path: rootFile, Covered: 9, Total: 10, Measured: true, Metric: "statement"},
		{Path: nested, Covered: 8, Total: 10, Measured: true, Metric: "statement"},
	}

	scope := newGateScope(root, GateScopeModeDiff, []string{rootFile}, nil)
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(90), scope, goSourceClassifier(),
	)(context.Background())

	for _, v := range result.Violations {
		if v.File == rootFile {
			t.Fatalf("a MEASURED root file must resolve to its own record under a same-basename collision (no unmeasured, no below-threshold since 9/10 >= 90); got [%s] %s", v.Rule, v.Message)
		}
	}
}

// TestCoverage_UnmeasuredRootFile_UnderBasenameCollision_StillFlagged (CLM-004):
// the case-2 over-correction guard AND the red-then-green driver for the
// suffix-fallback narrowing. Scope is the ROOT file embed.go (present on disk) but
// records exist ONLY for the same-basename NESTED file cmd/backstop/embed.go, NONE
// for the root. Before the fix the bare-basename suffix "/embed.go" mis-resolves to
// the nested record (found==1) and the root file silently passes as measured. After
// the narrowing the bare-basename path uses exact match only, finds no root record,
// and correctly fires coverage_unmeasured. Guards against a root file borrowing a
// nested sibling's record.
func TestCoverage_UnmeasuredRootFile_UnderBasenameCollision_StillFlagged(t *testing.T) {
	root := t.TempDir()
	rootFile := "embed.go"
	nested := "cmd/backstop/embed.go"
	writeSourceFile(t, root, rootFile, "package main\n\nfunc ListSchemas() int { return 1 }\n")
	writeSourceFile(t, root, nested, "package backstop\n\nfunc Embed() int { return 1 }\n")

	// ONLY the nested same-basename file is measured; the root file has NO record.
	records := []check.CoverageRecord{
		{Path: nested, Covered: 8, Total: 10, Measured: true, Metric: "statement"},
	}

	scope := newGateScope(root, GateScopeModeDiff, []string{rootFile}, nil)
	result := StepCoverageThresholdScopedFunc(
		records, coverageSpecs(90), scope, goSourceClassifier(),
	)(context.Background())

	var found *Violation
	for i := range result.Violations {
		if result.Violations[i].Rule == "coverage_unmeasured" && result.Violations[i].File == rootFile {
			found = &result.Violations[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("an unmeasured root file must STILL fire coverage_unmeasured and must NOT borrow the same-basename nested file's record; got %#v", result.Violations)
	}
	if found.Severity != "error" {
		t.Errorf("the unmeasured root-file violation must be a blocking error, got severity %q", found.Severity)
	}
}
