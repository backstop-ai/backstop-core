package gate

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// dimension_pathnorm_test.go (ISSUE-046 Phase 2) — every non-pack_engines
// dimension (coverage / contract / test_verification / substantiveness) must emit
// a canonical repo-relative Violation.File and therefore a scope-stable identity,
// so one finding has one identity gate-wide. These drive the REAL scoped step
// funcs / construction paths, not hand-shaped records.

// TestCoverageViolation_FileIsCanonicalRepoRelative (CLM-008): a coverage record
// whose path arrives ABSOLUTE (ProjectRoot-joined — the non-canonical form a
// producer can emit under a full-scope sweep) produces a below-threshold
// coverage_threshold Violation whose File is the repo-relative "pkg/x.go".
func TestCoverageViolation_FileIsCanonicalRepoRelative(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "pkg", "x.go")
	records := []check.CoverageRecord{
		{Path: absPath, Covered: 1, Total: 10, Measured: true, Metric: "statement"},
	}
	scope := newGateScope(root, GateScopeModeAll, nil, nil)
	result := StepCoverageThresholdScopedFunc(records, coverageSpecs(90), scope, goSourceClassifier())(context.Background())

	var found *Violation
	for i := range result.Violations {
		if result.Violations[i].Rule == "coverage_threshold" {
			found = &result.Violations[i]
			break
		}
	}
	if found == nil {
		t.Fatalf("expected a coverage_threshold violation for the below-threshold file, got %#v", result.Violations)
	}
	if found.File != "pkg/x.go" {
		t.Fatalf("coverage Violation.File not canonical repo-relative: got %q want %q", found.File, "pkg/x.go")
	}
}

// TestContractViolation_FileIsCanonicalRepoRelative (CLM-008): a present-contract
// entry whose File is the non-canonical "./pkg/x.go" and which the engine did NOT
// match produces a signature Violation whose File is the canonical "pkg/x.go".
func TestContractViolation_FileIsCanonicalRepoRelative(t *testing.T) {
	v, raised := VerifyContractVerdict(ContractEngineResult{
		Entry: ContractEntry{
			File:      "./pkg/x.go",
			Name:      "Foo",
			Kind:      "function",
			Signature: "func Foo() error",
			Absent:    false,
		},
		Matched: false,
		Scanned: true,
	})
	if !raised {
		t.Fatalf("expected an unmatched present-contract to raise a violation")
	}
	if v.File != "pkg/x.go" {
		t.Fatalf("contract Violation.File not canonical repo-relative: got %q want %q", v.File, "pkg/x.go")
	}
}

// TestTestVerifyViolation_FileIsCanonicalRepoRelative (CLM-008): a mandated test
// that is not found is attributed to its declaring spec path. The spec path
// arrives ABSOLUTE (joined under the temp project root) — the non-canonical form —
// and the resulting test_verification Violation must carry the canonical
// repo-relative "specs/test.spec.md".
func TestTestVerifyViolation_FileIsCanonicalRepoRelative(t *testing.T) {
	root := t.TempDir()
	specDir := filepath.Join(root, "specs")
	codeDir := filepath.Join(root, "pkg")
	mustMkdir(t, specDir)
	mustMkdir(t, codeDir)
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_MissingMandated"},
	})

	scope := newGateScope(root, GateScopeModeDiff, []string{"specs/test.spec.md"}, nil)
	result := StepTestVerificationScopedFunc(specDir, codeDir, scope, goTestClassifier(), goTestMatcher(t))(context.Background())

	if len(result.Violations) != 1 {
		t.Fatalf("expected exactly 1 missing-mandated-test violation, got %#v", result.Violations)
	}
	if result.Violations[0].File != "specs/test.spec.md" {
		t.Fatalf("test_verification Violation.File not canonical repo-relative: got %q want %q", result.Violations[0].File, "specs/test.spec.md")
	}
}

// TestSubstantivenessViolation_FileIsCanonicalRepoRelative (CLM-008): a carried
// hollow-test engine finding whose File is the non-canonical "./pkg/x_test.go"
// produces a test_substantiveness Violation whose File is the canonical
// "pkg/x_test.go".
func TestSubstantivenessViolation_FileIsCanonicalRepoRelative(t *testing.T) {
	out := HollowFindingsToViolations([]Violation{{
		Rule:    "some-pack/hollow-test",
		File:    "./pkg/x_test.go",
		Message: "test TestFoo has no assertions (hollow) func=TestFoo",
	}})
	if len(out) != 1 {
		t.Fatalf("expected exactly 1 substantiveness violation, got %#v", out)
	}
	if out[0].File != "pkg/x_test.go" {
		t.Fatalf("substantiveness Violation.File not canonical repo-relative: got %q want %q", out[0].File, "pkg/x_test.go")
	}
}

// TestAllDimensions_SameFileDifferentForm_ByteIdenticalIdentity (CLM-008):
// belt-and-suspenders on the Phase-1 chokepoint through the real construction
// paths — for each dimension, two inputs differing ONLY in File textual form yield
// an IDENTICAL EnrichViolationIdentity IdentityHash.
func TestAllDimensions_SameFileDifferentForm_ByteIdenticalIdentity(t *testing.T) {
	// Contract dimension: same entry, two File forms.
	contractA, raisedA := VerifyContractVerdict(ContractEngineResult{
		Entry:   ContractEntry{File: "pkg/x.go", Name: "Foo", Signature: "func Foo() error"},
		Matched: false, Scanned: true,
	})
	contractB, raisedB := VerifyContractVerdict(ContractEngineResult{
		Entry:   ContractEntry{File: "./pkg/x.go", Name: "Foo", Signature: "func Foo() error"},
		Matched: false, Scanned: true,
	})
	if !raisedA || !raisedB {
		t.Fatalf("expected both unmatched present-contracts to raise a violation, got raisedA=%v raisedB=%v", raisedA, raisedB)
	}
	assertSameIdentity(t, "contract", contractA, contractB)

	// Substantiveness dimension: same finding, two File forms.
	subA := HollowFindingsToViolations([]Violation{{File: "pkg/x_test.go", Message: "test TestFoo has no assertions (hollow) func=TestFoo"}})[0]
	subB := HollowFindingsToViolations([]Violation{{File: "pkg/./x_test.go", Message: "test TestFoo has no assertions (hollow) func=TestFoo"}})[0]
	assertSameIdentity(t, "substantiveness", subA, subB)
}

func assertSameIdentity(t *testing.T, dim string, a, b Violation) {
	t.Helper()
	ea := EnrichViolationIdentity(a)
	eb := EnrichViolationIdentity(b)
	if ea.IdentityHash != eb.IdentityHash {
		t.Fatalf("%s: identity differs across File forms: %s (%q) != %s (%q)", dim, ea.IdentityHash, a.File, eb.IdentityHash, b.File)
	}
}

func mustMkdir(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
}
