package main

import (
	"os"
	"strings"
	"testing"
)

// gate_substantiveness_coloc_test.go covers SPEC-045 REQ-003: the language-neutral
// same-unit predicate testFileColocatedWithTarget that REPLACES the deleted Go
// `package`-clause reader goFilePackageMatchesTarget. Same-unit is decided by the
// test file's DIRECTORY LEAF (filepath.Base(filepath.Dir(filePath))) == targetPkg —
// no file read, no `package` clause, carrying no Go assumption.

// TestColocated_SameDirectoryLeafIsSameUnit (CLM-019): a test file co-located with
// its target (directory leaf == targetPkg) is reported same-unit.
func TestColocated_SameDirectoryLeafIsSameUnit(t *testing.T) {
	if !testFileColocatedWithTarget("pkg/gate/foo_test.go", "gate") {
		t.Fatal("a test file whose directory leaf matches targetPkg must be same-unit")
	}
}

// TestColocated_DifferentDirectoryNotSameUnit (CLM-020): a test file in a different
// directory than its target is NOT same-unit.
func TestColocated_DifferentDirectoryNotSameUnit(t *testing.T) {
	if testFileColocatedWithTarget("pkg/other/foo_test.go", "gate") {
		t.Fatal("a test file in a non-matching directory must NOT be same-unit")
	}
}

// TestColocated_EmptyTargetIsFalse (CLM-021): an empty targetPkg yields false
// (preserved guard) regardless of the file path.
func TestColocated_EmptyTargetIsFalse(t *testing.T) {
	if testFileColocatedWithTarget("pkg/gate/foo_test.go", "") {
		t.Fatal("an empty targetPkg must yield false")
	}
}

// TestColocated_TSFileSameUnitWithoutPackageClause (CLM-022, de-Go proof): a TS
// `.test.ts` file with a matching directory leaf is reported same-unit WITHOUT any
// Go `package` clause existing in the file — the path need not even exist on disk,
// proving no clause is read.
func TestColocated_TSFileSameUnitWithoutPackageClause(t *testing.T) {
	// This path does not exist on disk; a verdict of true proves the predicate did
	// NOT open the file or read a `package` clause.
	if !testFileColocatedWithTarget("app/widget/foo.test.ts", "widget") {
		t.Fatal("a co-located TS test file must be same-unit by directory leaf, with no package clause read")
	}
}

// TestSubstantiveness_NoBakedGoPackageClauseReader (CLM-023, source guard): the Go
// `package`-clause reader goFilePackageMatchesTarget is DELETED from
// cmd/backstop/gate.go — reintroducing a clause-reading same-package matcher is
// caught.
func TestSubstantiveness_NoBakedGoPackageClauseReader(t *testing.T) {
	data, err := os.ReadFile("gate.go")
	if err != nil {
		t.Fatalf("reading cmd/backstop/gate.go: %v", err)
	}
	if strings.Contains(string(data), "goFilePackageMatchesTarget") {
		t.Error("cmd/backstop/gate.go still references goFilePackageMatchesTarget — the Go package-clause reader must be DELETED, replaced by testFileColocatedWithTarget (CLM-023)")
	}
}
