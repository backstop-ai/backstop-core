package gate

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// MandatedTest represents a test function mandated by a spec claim.
type MandatedTest struct {
	FuncName  string
	FilePath  string // path to the file containing the test (set during verification)
	SpecFile  string // path to the spec that mandates the test
	TargetPkg string // last component of the spec's implementation.package
	SpecID    string
	ClaimID   string
}

// specFrontmatter is a minimal representation of spec YAML frontmatter
// for extracting claims, mandated test names, verification blocks, and contracts.
type specFrontmatter struct {
	Number         string `yaml:"number"`
	Implementation struct {
		Package string `yaml:"package"`
	} `yaml:"implementation"`
	Verification struct {
		Level             string `yaml:"level"`
		TestCommand       string `yaml:"test_command"`
		CoverageThreshold int    `yaml:"coverage_threshold"`
	} `yaml:"verification"`
	Claims []struct {
		ID    string   `yaml:"id"`
		Tests []string `yaml:"tests"`
	} `yaml:"claims"`
	Contracts []struct {
		File     string `yaml:"file"`
		Provides []struct {
			Name      string `yaml:"name"`
			Kind      string `yaml:"kind"`
			Signature string `yaml:"signature"`
			// Absent asserts the named symbol MUST NOT exist in File. Optional,
			// defaults false; mutually exclusive with Signature.
			Absent bool `yaml:"absent"`
			// Scope is the absence file-OR-path the grep probe scans (REQ-012/
			// CLM-040). Optional; when empty the absence verdict falls back to File.
			Scope string `yaml:"scope"`
		} `yaml:"provides"`
	} `yaml:"contracts"`
}

// ExtractMandatedTests parses all spec files in specDir and extracts
// mandated test names from claims.
func ExtractMandatedTests(specDir string) ([]MandatedTest, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, err
	}

	var tests []MandatedTest
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".spec.md") {
			continue
		}

		path := filepath.Join(specDir, entry.Name())
		fm, err := parseSpecFrontmatter(path)
		if err != nil {
			continue // skip unparseable specs
		}

		targetPkg := TargetPackageName(fm.Implementation.Package)

		for _, claim := range fm.Claims {
			for _, testName := range claim.Tests {
				tests = append(tests, MandatedTest{
					FuncName:  testName,
					SpecFile:  path,
					TargetPkg: targetPkg,
					SpecID:    fm.Number,
					ClaimID:   claim.ID,
				})
			}
		}
	}
	return tests, nil
}

// parseSpecFrontmatter reads YAML frontmatter from a spec markdown file.
func parseSpecFrontmatter(path string) (*specFrontmatter, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)

	// Find opening ---
	if !scanner.Scan() || strings.TrimSpace(scanner.Text()) != "---" {
		return nil, os.ErrNotExist
	}

	var yamlLines []string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "---" {
			break
		}
		yamlLines = append(yamlLines, line)
	}

	var fm specFrontmatter
	if err := yaml.Unmarshal([]byte(strings.Join(yamlLines, "\n")), &fm); err != nil {
		return nil, err
	}
	return &fm, nil
}

// funcPattern matches Go test function declarations.
var funcPattern = regexp.MustCompile(`^func\s+(Test\w+)\s*\(`)

// StepTestVerificationFunc returns a StepFunc that verifies mandated test names
// exist as actual test functions in the codebase. This is a mechanical check —
// grep for exact function name in *_test.go files.
func StepTestVerificationFunc(specDir, codeDir string) StepFunc {
	return StepTestVerificationScopedFunc(specDir, codeDir, nil)
}

// StepTestVerificationScopedFunc verifies tests only in files allowed by scope.
func StepTestVerificationScopedFunc(specDir, codeDir string, scope *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		mandated, err := ExtractMandatedTests(specDir)
		if err != nil {
			return StepResult{
				StepName:   StepTestVerification,
				Status:     "fail",
				Violations: []Violation{{Rule: "test_verification", Message: "failed to extract mandated tests: " + err.Error(), Severity: "error"}},
			}
		}

		if len(mandated) == 0 {
			return StepResult{
				StepName:   StepTestVerification,
				Status:     "pass",
				Violations: []Violation{},
			}
		}

		found := collectTestFuncNames(codeDir)
		if scope != nil && scope.Mode != GateScopeModeAll {
			mandated = ResolveMandatedTestPaths(mandated, codeDir)
		}

		var violations []Violation
		for _, mt := range mandated {
			if scope != nil && scope.Mode != GateScopeModeAll {
				if mt.FilePath != "" && !scope.Contains(mt.FilePath) && !scope.Contains(mt.SpecFile) {
					continue
				}
				if mt.FilePath == "" && !scope.Contains(mt.SpecFile) {
					continue
				}
			}
			if _, ok := found[mt.FuncName]; !ok {
				violations = append(violations, Violation{
					Rule:     "test_verification",
					Message:  "mandated test function " + mt.FuncName + " not found (spec " + mt.SpecID + ", claim " + mt.ClaimID + ")",
					Severity: "error",
				})
			}
		}

		status := "pass"
		if len(violations) > 0 {
			status = "fail"
		}
		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName:   StepTestVerification,
			Status:     status,
			Violations: violations,
		}
	}
}

// collectTestFuncNames walks codeDir recursively and finds all test function
// names in *_test.go files using grep-level line matching.
func collectTestFuncNames(codeDir string) map[string]string {
	return collectTestFuncNamesScoped(codeDir, nil)
}

func collectTestFuncNamesScoped(codeDir string, scope *GateScope) map[string]string {
	found := make(map[string]string) // funcName → filePath

	_ = filepath.Walk(codeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(info.Name(), "_test.go") {
			return nil
		}
		if scope != nil && scope.Mode != GateScopeModeAll && !scope.Contains(path) {
			return nil
		}

		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			if matches := funcPattern.FindStringSubmatch(scanner.Text()); len(matches) > 1 {
				found[matches[1]] = path
			}
		}
		return nil
	})

	return found
}

// SPEC-037 (BUNDLE-009 Seed 3): the baked go/parser substantiveness ANALYZER —
// StepTestSubstantivenessFunc / StepTestSubstantivenessScopedFunc, checkSubstantiveness,
// hasAssertions, the assertionSelectors vocabulary, callsTargetPackage, and the
// lowercase targetPackageName helper — was DELETED here. Substantiveness is now an
// INSTALLED ast-grep pack (Q1 hollow-test findings + Q2 referenced-symbol extraction)
// consumed gate-side by the language-agnostic set-join in substantiveness_join.go and
// wired through the real dispatch seam in cmd/backstop/gate.go. The relocation of the
// target-package derivation lives there as the exported TargetPackageName (behavior-
// preserving). The deletion was licensed by the strangler-equivalence pass
// (substantiveness_strangler_test.go) proving the pack path reproduced this analyzer's
// verdicts on real Go fixtures BEFORE removal. ExtractMandatedTests / MandatedTest /
// ResolveMandatedTestPaths and the test-existence step are RETAINED.

// ExtractSpecVerifications parses all spec files in specDir and extracts
// verification metadata for the coverage threshold step. test_command is kept
// as documentation/compatibility metadata; the gate owns test scheduling.
func ExtractSpecVerifications(specDir string) ([]SpecVerification, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, err
	}

	var specs []SpecVerification
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".spec.md") {
			continue
		}

		path := filepath.Join(specDir, entry.Name())
		fm, err := parseSpecFrontmatter(path)
		if err != nil {
			continue // skip unparseable specs
		}

		if fm.Verification.TestCommand != "" && fm.Verification.CoverageThreshold > 0 {
			specs = append(specs, SpecVerification{
				SpecID:                fm.Number,
				TestCommand:           fm.Verification.TestCommand,
				CoverageThreshold:     fm.Verification.CoverageThreshold,
				File:                  path,
				ImplementationPackage: fm.Implementation.Package,
			})
		}
	}
	return specs, nil
}

// ExtractContractEntries parses all spec files in specDir and extracts
// contract declarations for use by the contract signature verification step.
// projectRoot is prepended to relative file paths in contracts.
func ExtractContractEntries(specDir, projectRoot string) ([]ContractEntry, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, err
	}

	var contracts []ContractEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".spec.md") {
			continue
		}

		path := filepath.Join(specDir, entry.Name())
		fm, err := parseSpecFrontmatter(path)
		if err != nil {
			continue // skip unparseable specs
		}

		for _, c := range fm.Contracts {
			filePath := c.File
			if !filepath.IsAbs(filePath) {
				filePath = filepath.Join(projectRoot, filePath)
			}
			for _, p := range c.Provides {
				// Scope is the declared absence file-OR-path (REQ-012/CLM-040/041).
				// Like File, a relative scope is joined onto projectRoot so the grep
				// probe receives an absolute path through pattern-arg. Extraction stays
				// a pure data-record builder — it reads the declared frontmatter field
				// only; it does NOT parse, AST-walk, or compile (CLM-042).
				scope := p.Scope
				if scope != "" && !filepath.IsAbs(scope) {
					scope = filepath.Join(projectRoot, scope)
				}
				contracts = append(contracts, ContractEntry{
					File:      filePath,
					Name:      p.Name,
					Kind:      p.Kind,
					Signature: p.Signature,
					Scope:     scope,
					Absent:    p.Absent,
				})
			}
		}
	}
	return contracts, nil
}

// ResolveMandatedTestPaths takes mandated tests and a map of found test functions
// (from collectTestFuncNames) and fills in the FilePath field for each found test.
// Returns the updated list.
func ResolveMandatedTestPaths(mandated []MandatedTest, codeDir string) []MandatedTest {
	found := collectTestFuncNames(codeDir)
	for i := range mandated {
		if path, ok := found[mandated[i].FuncName]; ok {
			mandated[i].FilePath = path
		}
	}
	return mandated
}
