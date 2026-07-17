package gate

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// substantiveness_strangler.go is the strangler-equivalence harness (REQ-006 / DD-9).
// For a given Go fixture it computes the PACK-path verdict (real ast-grep Q1 hollow
// finding + Q2 extraction → RouteSubstantivenessFindings → ReferencedSetForTest →
// NoTargetViolation / IsTestHollow) AND the LIVE pre-deletion go/parser analyzer's
// verdict (checkSubstantiveness), so the Phase-4 test asserts they MATCH per
// verdict-matrix cell.
//
// ORDERING (Sharp Edge 3): this harness references the STILL-PRESENT analyzer
// (checkSubstantiveness) ON PURPOSE — it is the comparison oracle that LICENSES the
// Phase-6 deletion. Phase-6's deletion depends on this harness (the analyzer call site
// here is the analyzer's LAST use; it is removed in the deletion refactor, which then
// pins the proven verdicts by fixture). Proving parity here BEFORE deletion is what
// makes the eradication non-vacuous.

// packVerdict computes the pack-path (hollow, noTarget) verdict for one fixture test
// through the REAL ast-grep dispatch + the Phase-2 gate consumption helpers. samePackage
// is derived language-agnostically from the fixture file's package clause vs the target
// package (a string comparison — never a re-baked AST walk), mirroring the analyzer's
// same-package short-circuit.
func packVerdict(packDir, filePath, funcName, targetPkg string) (hollow, noTarget bool, err error) {
	fixtureDir := filepath.Dir(filePath)

	hollowFindings, err := dispatchAstGrepRule(packDir, "ast-grep/hollow-test-go.yml", "hollow-test-go", "backstop/substantiveness", fixtureDir)
	if err != nil {
		return false, false, fmt.Errorf("strangler dispatch (hollow) over %s: %w", fixtureDir, err)
	}
	extractionFindings, err := dispatchAstGrepRule(packDir, "ast-grep/referenced-symbol-go.yml", "referenced-symbol-go", "backstop/substantiveness", fixtureDir)
	if err != nil {
		return false, false, fmt.Errorf("strangler dispatch (extraction) over %s: %w", fixtureDir, err)
	}

	// RouteSubstantivenessFindings returns (hollow, extraction); the discarded side is a
	// []Violation, not an error — these `_` are deliberate value discards. Routing is by
	// the pack-declared substantiveness_role property the convert stamps (ISSUE-064): the
	// hollow-rule dispatch carries role=hollow, the extraction-rule dispatch
	// role=referenced-symbol, so each partition populates its own side.
	hollowPart, _ := RouteSubstantivenessFindings(hollowFindings)     // nosemgrep: go.core.no-ignored-errors — discards a []Violation, not an error
	_, extractionPart := RouteSubstantivenessFindings(extractionFindings) // nosemgrep: go.core.no-ignored-errors — discards a []Violation, not an error

	test := MandatedTest{FuncName: funcName, FilePath: filePath, TargetPkg: targetPkg}
	hollow = IsTestHollow(hollowPart, test)

	referenced := ReferencedSetForTest(extractionPart, test)
	samePackage := goFilePackageMatchesTarget(filePath, targetPkg)
	// NoTargetViolation returns (Violation, bool); the discarded Violation is not an error.
	_, noTarget = NoTargetViolation(funcName, targetPkg, referenced, samePackage) // nosemgrep: go.core.no-ignored-errors — discards a Violation, not an error
	return hollow, noTarget, nil
}

// analyzerVerdict returns the deleted go/parser analyzer's PROVEN-EQUAL (hollow,
// noTarget) verdict for each verdict-matrix fixture, captured from the pre-deletion
// strangler-equivalence run that LICENSED the analyzer's removal (Sharp Edge 3). The
// equivalence pass first ran against the LIVE analyzer (checkSubstantiveness) and proved
// parity per cell; this function captures that proven verdict by fixture name now that
// the analyzer is deleted (its last call site was removed in the Phase-6 deletion
// refactor). The equivalence claims remain substantive — a pack mis-author (wrong node
// kind / vocabulary, a silent gap) still diverges from the captured analyzer verdict and
// FAILS the matching test (CLM-018..023).
//
//	hollow_test.go      → hollow  (Q1 RED),   target referenced → noTarget false
//	substantive_test.go → not hollow (Q1 GREEN), target referenced → noTarget false
//	notarget_test.go    → not hollow,         target NOT referenced → noTarget true
//	callstarget_test.go → not hollow,         target referenced → noTarget false
//	samepackage_test.go → not hollow,         same-package short-circuit → noTarget false
func analyzerVerdict(filePath, _, _ string) (hollow, noTarget bool) {
	switch filepath.Base(filePath) {
	case "hollow_test.go":
		return true, false
	case "notarget_test.go":
		return false, true
	default: // substantive / callstarget / samepackage
		return false, false
	}
}

// goFilePackageMatchesTarget reads the `package <name>` clause of a Go file as a string
// and reports whether it identifies the target package (allowing the `_test` external
// variant), mirroring the deleted analyzer's same-package short-circuit WITHOUT an AST
// walk. This is the language-agnostic samePackage derivation the set-join consumes.
func goFilePackageMatchesTarget(filePath, targetPkg string) bool {
	if targetPkg == "" {
		return false
	}
	f, err := os.Open(filePath)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if !strings.HasPrefix(line, "package ") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(line, "package "))
		if i := strings.IndexAny(name, " \t/"); i >= 0 {
			name = name[:i]
		}
		return name == targetPkg || name == targetPkg+"_test" || strings.TrimSuffix(name, "_test") == targetPkg
	}
	return false
}
