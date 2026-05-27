package gate

import (
	"bufio"
	"context"
	"go/ast"
	"go/parser"
	"go/token"
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

		targetPkg := targetPackageName(fm.Implementation.Package)

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

// StepTestSubstantivenessFunc returns a StepFunc that checks whether mandated
// test functions are substantive (not hollow). Uses Go AST parsing.
func StepTestSubstantivenessFunc(tests []MandatedTest) StepFunc {
	return StepTestSubstantivenessScopedFunc(tests, nil)
}

// StepTestSubstantivenessScopedFunc checks only mandated tests in scoped files.
func StepTestSubstantivenessScopedFunc(tests []MandatedTest, scope *GateScope) StepFunc {
	return func(_ context.Context) StepResult {
		var violations []Violation

		for _, mt := range tests {
			if mt.FilePath == "" {
				continue // skip tests not found (already reported by verification)
			}
			if scope != nil && scope.Mode != GateScopeModeAll && !scope.Contains(mt.FilePath) {
				continue
			}

			hollow, noTarget := checkSubstantiveness(mt.FilePath, mt.FuncName, mt.TargetPkg)
			if hollow {
				violations = append(violations, Violation{
					Rule:     "test_substantiveness",
					File:     mt.FilePath,
					Message:  "test function " + mt.FuncName + " has no assertions (hollow)",
					Severity: "error",
				})
			}
			if noTarget {
				violations = append(violations, Violation{
					Rule:     "test_substantiveness",
					File:     mt.FilePath,
					Message:  "test function " + mt.FuncName + " does not call package " + mt.TargetPkg,
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
			StepName:   StepTestSubstantiveness,
			Status:     status,
			Violations: violations,
		}
	}
}

// checkSubstantiveness parses a Go file and checks if the named test function
// has assertions and calls the target package. Returns (hollow, noTargetCall).
func checkSubstantiveness(filePath, funcName, targetPkg string) (bool, bool) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, nil, parser.AllErrors)
	if err != nil {
		return true, true // can't parse → treat as hollow
	}

	// If the test file's package matches the target package (or is the
	// _test variant), the test IS in the target package — skip the
	// target-call check. Same-package tests call functions directly
	// without a package qualifier (Function() not pkg.Function()).
	filePkg := ""
	if file.Name != nil {
		filePkg = file.Name.Name
	}
	samePackage := filePkg == targetPkg ||
		filePkg == targetPkg+"_test" ||
		strings.TrimSuffix(filePkg, "_test") == targetPkg

	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Name.Name != funcName || fn.Body == nil {
			continue
		}

		hasAssertion := hasAssertions(fn.Body)

		// Same-package tests don't need a qualified call to the target.
		if samePackage {
			return !hasAssertion, false
		}
		if targetPkg == "" {
			return !hasAssertion, false
		}

		callsTarget := callsTargetPackage(fn.Body, targetPkg)
		return !hasAssertion, !callsTarget
	}

	// Function not found in file
	return true, true
}

func targetPackageName(implementationPackage string) string {
	if strings.HasPrefix(implementationPackage, "cmd/") {
		return ""
	}
	if !strings.HasPrefix(implementationPackage, "pkg/") {
		return ""
	}
	return filepath.Base(implementationPackage)
}

// assertionSelectors are method names on *testing.T that count as assertions.
var assertionSelectors = map[string]bool{
	"Fatal":   true,
	"Fatalf":  true,
	"Error":   true,
	"Errorf":  true,
	"Fail":    true,
	"FailNow": true,
	"Skip":    true,
	"Skipf":   true,
	"Log":     true,
	"Logf":    true,
	"Run":     true, // subtests
	"Helper":  true, // test helpers that delegate
}

// hasAssertions checks if a function body contains at least one assertion call.
// Recognizes:
//   - t.Fatal, t.Error, t.Run, etc. (selector expressions on *testing.T)
//   - Helper functions whose names suggest assertions: require*, assert*,
//     check*, verify*, expect*, must* (plain function calls)
//   - Any function call that receives a *testing.T as first argument
func hasAssertions(body *ast.BlockStmt) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}

		// Check selector expressions: t.Fatal, t.Error, etc.
		if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
			if assertionSelectors[sel.Sel.Name] {
				found = true
				return false
			}
		}

		// Check plain function calls that look like assertion helpers.
		if ident, ok := call.Fun.(*ast.Ident); ok {
			name := strings.ToLower(ident.Name)
			if strings.HasPrefix(name, "require") ||
				strings.HasPrefix(name, "assert") ||
				strings.HasPrefix(name, "check") ||
				strings.HasPrefix(name, "verify") ||
				strings.HasPrefix(name, "expect") ||
				strings.HasPrefix(name, "must") {
				found = true
				return false
			}
			// Any function that takes t as first arg is likely an assertion helper.
			if len(call.Args) > 0 {
				if argIdent, ok := call.Args[0].(*ast.Ident); ok && argIdent.Name == "t" {
					found = true
					return false
				}
			}
		}

		return true
	})
	return found
}

// callsTargetPackage checks if a function body contains at least one call to
// a function/method from the target package (identified by package name).
func callsTargetPackage(body *ast.BlockStmt, pkg string) bool {
	found := false
	ast.Inspect(body, func(n ast.Node) bool {
		if found {
			return false
		}
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		ident, ok := sel.X.(*ast.Ident)
		if !ok {
			return true
		}
		if ident.Name == pkg {
			found = true
			return false
		}
		return true
	})
	return found
}

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
				contracts = append(contracts, ContractEntry{
					File:      filePath,
					Name:      p.Name,
					Kind:      p.Kind,
					Signature: p.Signature,
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
