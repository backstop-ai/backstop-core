package gate

import (
	"bufio"
	"context"
	"fmt"
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
	// IsAbsence is the opt-in per-claim signal (ISSUE-035 Category 2): the mandating
	// claim declared `kind: absence`, marking this an absence/structural test that by
	// design does NOT call its target package. When true, the gate SKIPS the noTarget
	// set-join for this test (see NoTargetViolationForTest). DEFAULT false — an
	// unannotated claim keeps FULL noTarget enforcement.
	IsAbsence bool
}

// specFrontmatter is a minimal representation of spec YAML frontmatter
// for extracting claims, mandated test names, verification blocks, and contracts.
type specFrontmatter struct {
	Number string `yaml:"number"`
	// Status is the spec lifecycle status. Terminal statuses (replaced, canceled,
	// deprecated — ISSUE-031 DQ-1) cause the spec to be EXCLUDED from gate
	// enforcement: its mandated tests, verifications, and contracts are not
	// extracted, because a retired spec's promises are deliberately no longer held.
	Status         string `yaml:"status"`
	Implementation struct {
		Package string `yaml:"package"`
	} `yaml:"implementation"`
	Verification struct {
		Level             string `yaml:"level"`
		TestCommand       string `yaml:"test_command"`
		CoverageThreshold int    `yaml:"coverage_threshold"`
		// CoverageMetricThresholds is the OPTIONAL per-metric declared threshold map
		// (SQ-2, SPEC-044 REQ-003): metric label → integer threshold. It is threaded
		// onto SpecVerification.MetricThresholds; a metric absent from it uses the
		// scalar coverage_threshold as its default. A spec declaring only this map
		// (no scalar) is still extracted (the loosened gate below).
		CoverageMetricThresholds map[string]int `yaml:"coverage_metric_thresholds"`
	} `yaml:"verification"`
	Claims []struct {
		ID string `yaml:"id"`
		// Kind is the OPTIONAL per-claim classification. `kind: absence` marks the
		// claim's mandated test(s) as absence/structural (ISSUE-035 Category 2), which
		// sets MandatedTest.IsAbsence and skips the noTarget join for those tests. An
		// absent/other value leaves IsAbsence false (full enforcement).
		Kind  string   `yaml:"kind"`
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
		if isTerminalSpecStatus(fm.Status) {
			continue // terminal specs are excluded from enforcement (ISSUE-031)
		}

		targetPkg := TargetPackageName(fm.Implementation.Package)

		for _, claim := range fm.Claims {
			// Opt-in absence signal (ISSUE-035 Category 2), applied at the same
			// extraction site as the terminal pre-filter above. DEFAULT-OFF: only a
			// claim that EXPLICITLY declares `kind: absence` excuses its tests from the
			// noTarget join; any other value keeps full enforcement.
			isAbsence := claim.Kind == "absence"
			for _, testName := range claim.Tests {
				tests = append(tests, MandatedTest{
					FuncName:  testName,
					SpecFile:  path,
					TargetPkg: targetPkg,
					SpecID:    fm.Number,
					ClaimID:   claim.ID,
					IsAbsence: isAbsence,
				})
			}
		}
	}
	return tests, nil
}

// isTerminalSpecStatus reports whether a spec status is an end-of-life state
// (ISSUE-031 DQ-1). Terminal specs are excluded from gate enforcement — their
// mandated tests and contracts are no longer held as live promises. This is the
// single source of truth the mandated-test step, the contract step, and the
// verification step all key on.
func isTerminalSpecStatus(status string) bool {
	switch status {
	case "replaced", "canceled", "deprecated":
		return true
	default:
		return false
	}
}

// CountTerminalSpecs returns the number of spec files in specDir whose status is
// terminal (replaced/canceled/deprecated) and are therefore excluded from gate
// enforcement. The gate command reports this as an informational line (CLM-017).
// Unparseable specs are not counted (they cannot be classified as terminal).
func CountTerminalSpecs(specDir string) (int, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return 0, fmt.Errorf("reading spec dir %s: %w", specDir, err)
	}
	count := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".spec.md") {
			continue
		}
		fm, err := parseSpecFrontmatter(filepath.Join(specDir, entry.Name()))
		if err != nil {
			continue
		}
		if isTerminalSpecStatus(fm.Status) {
			count++
		}
	}
	return count, nil
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

// TestNameMatcher holds the compiled UNION of pack-declared test-name regexes
// (SPEC-045 REQ-002). It replaces the DELETED baked Go-shaped `funcPattern`: the
// test-name/indicator pattern now comes from pack DATA (Manifest.TestNamePatterns),
// merged across declared toolchain packs and compiled here. It carries NO language
// knowledge — data (the declared patterns) plus match logic only (DD-1). Each
// pattern's capture group 1 is the test name.
type TestNameMatcher struct {
	patterns []*regexp.Regexp
}

// NewTestNameMatcher compiles the merged list of pack-declared test-name regexes
// (SPEC-045 REQ-002/CLM-016). An INVALID regex returns a LOUD construction error —
// never a silently-dropped pattern that would make discovery find nothing and then
// mass-fail every mandated test. Each pattern must expose capture group 1 as the
// test name; a nil/empty list yields a matcher whose HasPatterns reports false.
func NewTestNameMatcher(patterns []string) (TestNameMatcher, error) {
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, p := range patterns {
		re, err := regexp.Compile(p)
		if err != nil {
			return TestNameMatcher{}, fmt.Errorf("invalid test_name_pattern %q: %w", p, err)
		}
		compiled = append(compiled, re)
	}
	return TestNameMatcher{patterns: compiled}, nil
}

// FindName returns capture group 1 of the FIRST declared pattern that matches the
// line, or ok=false (SPEC-045 REQ-002/CLM-010..CLM-015). With only bun patterns
// declared, a Go `func TestFoo(` line returns ok=false — no baked Go literal.
func (m TestNameMatcher) FindName(line string) (string, bool) {
	for _, re := range m.patterns {
		if sub := re.FindStringSubmatch(line); len(sub) > 1 {
			return sub[1], true
		}
	}
	return "", false
}

// HasPatterns reports whether any test-name patterns are declared (SPEC-045
// REQ-005), so the step can surface the DISTINCT discovery-capability-absent state
// instead of a misleading mass not-found fail.
func (m TestNameMatcher) HasPatterns() bool { return len(m.patterns) > 0 }

// StepTestVerificationFunc returns a StepFunc that verifies mandated test names
// exist as actual test functions in the codebase. This is a mechanical check:
// discover test FILES via the pack-declared TEST globs (classifier.IsTestFile) and
// extract test NAMES via the pack-declared TestNameMatcher — no baked `_test.go`
// walk, no baked `func Test` regex.
func StepTestVerificationFunc(specDir, codeDir string, classifier SourceClassifier, matcher TestNameMatcher) StepFunc {
	return StepTestVerificationScopedFunc(specDir, codeDir, nil, classifier, matcher)
}

// StepTestVerificationScopedFunc verifies tests only in files allowed by scope.
//
// Discovery needs BOTH inputs — test globs to FIND the test file AND name patterns
// to EXTRACT the test name — so capability is PRESENT only when BOTH are declared.
// When `!classifier.HasTestGlobs() || !matcher.HasPatterns()` (EITHER missing) and
// mandated tests exist, the step returns a DISTINCT non-blocking `warning` whose
// Reason NAMES the missing input (SPEC-045 REQ-005/CLM-031/CLM-032/CLM-037),
// intercepting the partial globs-but-no-patterns case BEFORE FindName returning
// false for every line becomes a mass false "not found" fail — never an unqualified
// pass nor a mass not-found fail. The guard MUST stay either-absent (`||`): an AND
// guard would let a globs-but-no-patterns pack slip through as "capability present"
// and mass-fail every mandated test. When BOTH are declared (capability fully
// present), a genuinely-missing mandated test stays a LOUD blocking failure.
func StepTestVerificationScopedFunc(specDir, codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher) StepFunc {
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

		// EITHER-absent capability guard (REQ-005). Checked BEFORE the discovery walk
		// so a partial config (globs but no patterns) cannot become a mass not-found
		// fail misattributing the config gap to the codebase.
		if !classifier.HasTestGlobs() || !matcher.HasPatterns() {
			return testDiscoveryCapabilityAbsent(classifier.HasTestGlobs(), matcher.HasPatterns())
		}

		found := collectTestFuncNames(codeDir, classifier, matcher)
		if scope != nil && scope.Mode != GateScopeModeAll {
			mandated = ResolveMandatedTestPaths(mandated, codeDir, classifier, matcher)
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

// testDiscoveryCapabilityAbsent builds the DISTINCT, VISIBLE, NON-blocking warning
// for REQ-005: when EITHER the test globs OR the test-name patterns are not declared
// (capability is present only when BOTH are), the step cannot discover/verify
// mandated tests, so it surfaces a warning that NAMES the missing input rather than
// (a) silently passing or (b) mass-failing every mandated test as falsely "not
// found". It reuses the EXISTING capability-absent convention (warning status,
// ConfigErr false, exit 0) the traceability/coverage dimensions emit.
func testDiscoveryCapabilityAbsent(hasGlobs, hasPatterns bool) StepResult {
	var missing string
	switch {
	case !hasGlobs && !hasPatterns:
		missing = "no toolchain pack declares classification.test globs (to find test files) NOR test_name_patterns (to extract test names)"
	case !hasGlobs:
		missing = "no toolchain pack declares classification.test globs to find test files"
	default: // !hasPatterns
		missing = "no toolchain pack declares test_name_patterns to extract test names (test files may be found, but no test name can be read from them)"
	}
	msg := "test-discovery capability absent: " + missing +
		" — install/declare a toolchain pack carrying both classification.test globs and test_name_patterns. This advisory is non-blocking (exit 0); it is NOT a report that the mandated tests are missing from the codebase."
	return StepResult{
		StepName:   StepTestVerification,
		Status:     "warning",
		ConfigErr:  false,
		Reason:     "test-discovery capability absent (" + missing + ")",
		Violations: []Violation{{Rule: "test_verification_capability_absent", Message: msg, Severity: "warning"}},
	}
}

// collectTestFuncNames walks codeDir recursively and finds all test names in
// pack-declared TEST files (classifier.IsTestFile) by applying the pack-declared
// TestNameMatcher per line — no baked `_test.go` walk, no baked `func Test` regex.
func collectTestFuncNames(codeDir string, classifier SourceClassifier, matcher TestNameMatcher) map[string]string {
	return collectTestFuncNamesScoped(codeDir, nil, classifier, matcher)
}

func collectTestFuncNamesScoped(codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher) map[string]string {
	found := make(map[string]string) // testName → filePath

	_ = filepath.Walk(codeDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		// Discovery keys on the pack-declared TEST globs (REQ-001), never a baked
		// extension literal. The classifier matches on the project-relative path,
		// so derive it from codeDir.
		rel, relErr := filepath.Rel(codeDir, path)
		if relErr != nil {
			rel = path
		}
		if !classifier.IsTestFile(rel) {
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
			if name, ok := matcher.FindName(scanner.Text()); ok {
				found[name] = path
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
		if isTerminalSpecStatus(fm.Status) {
			continue // terminal specs are excluded from enforcement (ISSUE-031)
		}

		// The extraction gate is LOOSENED (SPEC-044 REQ-003): a spec is extracted when
		// it declares a test command AND either a scalar coverage_threshold OR a
		// per-metric coverage_metric_thresholds map. A spec declaring only per-metric
		// thresholds (no scalar) is therefore still extracted; a scalar-only spec
		// extracts with a nil MetricThresholds, preserving today's behavior (REQ-004).
		if fm.Verification.TestCommand != "" &&
			(fm.Verification.CoverageThreshold > 0 || len(fm.Verification.CoverageMetricThresholds) > 0) {
			specs = append(specs, SpecVerification{
				SpecID:                fm.Number,
				TestCommand:           fm.Verification.TestCommand,
				CoverageThreshold:     fm.Verification.CoverageThreshold,
				MetricThresholds:      fm.Verification.CoverageMetricThresholds,
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
		if isTerminalSpecStatus(fm.Status) {
			continue // terminal specs are excluded from enforcement (ISSUE-031)
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
func ResolveMandatedTestPaths(mandated []MandatedTest, codeDir string, classifier SourceClassifier, matcher TestNameMatcher) []MandatedTest {
	found := collectTestFuncNames(codeDir, classifier, matcher)
	for i := range mandated {
		if path, ok := found[mandated[i].FuncName]; ok {
			mandated[i].FilePath = path
		}
	}
	return mandated
}
