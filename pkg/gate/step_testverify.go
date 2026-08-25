package gate

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
)

// MandatedTest represents a test function mandated by a spec claim.
type MandatedTest struct {
	FuncName  string
	FilePath  string // path to the file containing the test (set during verification)
	SpecFile  string // path to the spec that mandates the test
	TargetPkg string // reduced opaque subject token (per-claim subject override, else spec default)
	SpecID    string
	ClaimID   string
	// Status carries the mandating spec's lifecycle status (populated by
	// ExtractMandatedTests from the spec frontmatter). It lets the test_verification
	// and test_substantiveness CONSUMERS apply the implemented-only scope
	// (contractsAreDue) while ExtractMandatedTests itself stays UNFILTERED, so the
	// artifact_status_drift consumer keeps full draft-spec visibility (ISSUE-054 CLM-002).
	Status string
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
	Status string `yaml:"status"`
	// Implementation carries the spec-level target unit. `subject` is the canonical
	// language-neutral key (ISSUE-047); `package` is a DEPRECATED ALIAS retained so the
	// ~40 unmigrated specs keep resolving. yaml.v3 struct tags cannot alias two keys
	// onto one field, so BOTH are read into distinct fields and coalesced at read time
	// (subject wins, else package) via implementationSubject — a naive single `subject`
	// tag would SILENTLY DROP `package` and zero the target for every unmigrated spec.
	Implementation struct {
		Subject string `yaml:"subject"`
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
		Kind string `yaml:"kind"`
		// Subject is the OPTIONAL per-claim target override (ISSUE-047 multi-target
		// specs): when non-empty it overrides the spec-level implementation subject for
		// THIS claim's mandated tests, reduced to an opaque token by TargetPackageName.
		// Absent leaves the claim inheriting the spec default.
		Subject string   `yaml:"subject"`
		Tests   []string `yaml:"tests"`
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

// implementationSubject resolves the spec-level target unit, coalescing the
// canonical `subject` with the deprecated `package` alias (subject wins, else
// package). It returns the FULL declared value (path or bare token) — coverage's
// path-form directory matching legitimately needs the path form, so only the
// noTarget join reduces it to a leaf via TargetPackageName. yaml cannot alias two
// keys onto one field, so this coalesce is where the alias is honored (ISSUE-047).
func (fm *specFrontmatter) implementationSubject() string {
	if s := strings.TrimSpace(fm.Implementation.Subject); s != "" {
		return s
	}
	return strings.TrimSpace(fm.Implementation.Package)
}

// ExtractMandatedTests parses all spec files in specDir and extracts
// mandated test names from claims.
func ExtractMandatedTests(specDir string) ([]MandatedTest, error) {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return nil, fmt.Errorf("reading spec dir %s: %w", specDir, err)
	}

	var tests []MandatedTest
	for _, entry := range entries {
		// SPEC DISCOVERY GOES THROUGH THE SHARED LAYOUT TABLE (ISSUE-124). The same
		// three-line decision appears in CountTerminalSpecs, ExtractSpecVerifications
		// and ExtractContractEntries; all four used to carry a private ".spec.md".
		//
		// ★ ClassifyFilename RATHER THAN LayoutFor(KindSpec).Extension, and the choice is
		// a safety property rather than a style one. LayoutFor returns a ZERO-VALUE
		// KindLayout when its ok is false, and that zero value's Extension is the EMPTY
		// STRING — so `!strings.HasSuffix(name, layout.Extension)` with a discarded ok
		// silently inverts this filter from "specs only" into "accept every file in the
		// directory", and the extractors below start parsing README.md as spec
		// frontmatter. ClassifyFilename has no such hazard: its bool IS the answer, and a
		// false simply means the name is not artifact-shaped, which is precisely the
		// skip this loop wants. There is no ok to discard here and no impossible state to
		// report. The IsDir guard is PRESERVED — a directory named x.spec.md classifies
		// happily and must still be skipped.
		kind, isArtifact := artifact.ClassifyFilename(entry.Name())
		if entry.IsDir() || !isArtifact || kind != artifact.KindSpec {
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

		// Spec-level default subject (subject wins, else the deprecated package alias),
		// reduced once to its opaque token. A per-claim subject overrides it below.
		specDefaultSubject := fm.implementationSubject()

		for _, claim := range fm.Claims {
			// Opt-in absence signal (ISSUE-035 Category 2), applied at the same
			// extraction site as the terminal pre-filter above. DEFAULT-OFF: only a
			// claim that EXPLICITLY declares `kind: absence` excuses its tests from the
			// noTarget join; any other value keeps full enforcement.
			isAbsence := claim.Kind == "absence"
			// Per-claim subject override (ISSUE-047 multi-target specs): when the claim
			// declares its own subject use it, else inherit the spec default. The chosen
			// subject is reduced to its opaque token by TargetPackageName — the absence
			// signal is orthogonal and does not affect target derivation.
			claimSubject := specDefaultSubject
			if s := strings.TrimSpace(claim.Subject); s != "" {
				claimSubject = s
			}
			targetPkg := TargetPackageName(claimSubject)
			for _, testName := range claim.Tests {
				tests = append(tests, MandatedTest{
					FuncName:  testName,
					SpecFile:  path,
					TargetPkg: targetPkg,
					SpecID:    fm.Number,
					ClaimID:   claim.ID,
					Status:    fm.Status,
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

// contractsAreDue reports whether a spec's contracts are held as a live promise.
// Contracts are due ONLY once a spec reaches `implemented` (ISSUE-051):
// pre-implementation specs (draft, ready-for-implementation) describe intended
// future code, and all terminal specs (replaced, canceled, deprecated,
// obsoleted) are retired — neither is enforced. This is the single source of
// truth for contract-due exclusion on the contract path; unlike
// isTerminalSpecStatus it also correctly excludes obsoleted and the
// pre-implementation statuses without enumerating them.
func contractsAreDue(status string) bool {
	return status == "implemented"
}

// ContractsAreDue is the exported wrapper over the unexported contractsAreDue so the
// cmd/backstop test_substantiveness consumer can apply the same implemented-only scope
// (ISSUE-054). It is additive: the unexported predicate stays the single source of
// truth (and keeps its existing pkg/gate callers/tests intact).
func ContractsAreDue(status string) bool { return contractsAreDue(status) }

// filterDueMandatedTests keeps only the mandated tests whose mandating spec is due
// (implemented) per contractsAreDue, applying the ISSUE-054 implemented-only scope at
// the test_verification consumer. It does NOT touch the shared ExtractMandatedTests,
// so artifact_status_drift keeps consuming the unfiltered list.
func filterDueMandatedTests(mandated []MandatedTest) []MandatedTest {
	due := mandated[:0:0]
	for _, mt := range mandated {
		if contractsAreDue(mt.Status) {
			due = append(due, mt)
		}
	}
	return due
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
		kind, isArtifact := artifact.ClassifyFilename(entry.Name())
		if entry.IsDir() || !isArtifact || kind != artifact.KindSpec {
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
		return nil, fmt.Errorf("opening spec %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

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
		return nil, fmt.Errorf("parsing spec frontmatter %s: %w", path, err)
	}
	return &fm, nil
}

// TestNameMatcher holds the compiled UNION of pack-declared test-name regexes
// (SPEC-045 REQ-002). It replaces the DELETED baked Go-shaped func-name pattern matcher: the
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

// FindNames returns every nonempty capture-group-1 match across the declared
// patterns. Source position is authoritative; declared-pattern order breaks ties.
func (m TestNameMatcher) FindNames(line string) []string {
	type match struct {
		name         string
		start        int
		patternIndex int
	}
	matches := make([]match, 0)
	for patternIndex, re := range m.patterns {
		for _, indices := range re.FindAllStringSubmatchIndex(line, -1) {
			if len(indices) < 4 || indices[2] < 0 || indices[3] <= indices[2] {
				continue
			}
			matches = append(matches, match{
				name:         line[indices[2]:indices[3]],
				start:        indices[0],
				patternIndex: patternIndex,
			})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool {
		if matches[i].start != matches[j].start {
			return matches[i].start < matches[j].start
		}
		return matches[i].patternIndex < matches[j].patternIndex
	})
	names := make([]string, 0, len(matches))
	seen := make(map[struct {
		name  string
		start int
	}]struct{}, len(matches))
	for _, candidate := range matches {
		key := struct {
			name  string
			start int
		}{name: candidate.name, start: candidate.start}
		if _, duplicate := seen[key]; duplicate {
			continue
		}
		seen[key] = struct{}{}
		names = append(names, candidate.name)
	}
	return names
}

// FindName preserves the original single-name API by returning the first
// deterministic result from FindNames, or empty/false when no name was found.
func (m TestNameMatcher) FindName(line string) (string, bool) {
	names := m.FindNames(line)
	if len(names) == 0 {
		return "", false
	}
	return names[0], true
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
//
// NAME PRESENCE IS ONLY HALF THE DIMENSION. Since ISSUE-118 test_verification also
// answers whether the mandated tests PASSED, via the verdict channel
// StepTestVerificationVerdictFunc takes. This constructor wires no such channel and
// therefore checks spelling only — which is exactly the unqualified pass over a
// present-but-failing mandated test that ISSUE-118 reported.
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
	return StepTestVerificationVerdictFunc(specDir, codeDir, scope, classifier, matcher, nil)
}

// TestVerdictSupplier yields the routed test-verdict findings an EARLIER gate step
// collected, and whether any installed pack DECLARED a test-verdict engine at all.
//
// It is a SUPPLIER rather than a slice because the collector is populated at RUN
// time by the pack-dispatch step, not at step-BUILD time — the assembled order is
// what guarantees the dispatch runs first, and a slice captured at build time would
// always be empty.
//
// A nil supplier means NO verdict channel is wired: the dimension then behaves
// exactly as it did before ISSUE-118 (name presence only, no join, no advisory).
// That is what keeps the pre-existing constructors — used by the baseline and
// waiver step builders — unchanged.
type TestVerdictSupplier func() (verdicts []Violation, engineDeclared bool)

// StepTestVerificationVerdictFunc is the full-arity constructor: it verifies
// mandated test names exist AND, when a verdict channel is wired, that the mandated
// tests actually PASSED.
//
// The dimension answers TWO questions, and ISSUE-118 was filed because it only ever
// answered the first. Name presence is a spelling check: it proves a mandated test
// EXISTS. The verdict join proves it did not FAIL. A test that exists, is
// substantive, and is RED satisfied the old check completely — an unqualified pass
// over a broken promise, which is the false assurance the issue reported.
//
// The verdict half consumes findings a PACK engine produced, routed by the
// DECLARED gate_type; core still runs no test suite itself.
func StepTestVerificationVerdictFunc(specDir, codeDir string, scope *GateScope, classifier SourceClassifier, matcher TestNameMatcher, verdictChannel TestVerdictSupplier) StepFunc {
	return func(_ context.Context) StepResult {
		mandated, err := ExtractMandatedTests(specDir)
		if err != nil {
			return StepResult{
				StepName:   StepTestVerification,
				Status:     "fail",
				Violations: []Violation{{Rule: "test_verification", Message: "failed to extract mandated tests: " + err.Error(), Severity: "error"}},
			}
		}

		// Implemented-only scope (ISSUE-054): a mandated test is a live promise only
		// once its spec is `implemented`; draft / ready-for-implementation specs
		// describe planned-but-unbuilt code. Applied at the CONSUMER (not the shared
		// ExtractMandatedTests, which artifact_status_drift needs unfiltered) and BEFORE
		// the len==0 early-return + capability guard, so an all-draft set is a clean
		// pass rather than a capability warning.
		mandated = filterDueMandatedTests(mandated)

		if len(mandated) == 0 {
			return StepResult{
				StepName:   StepTestVerification,
				Status:     "pass",
				Violations: []Violation{},
			}
		}

		// Resolve the verdict channel ONCE, ABOVE the discovery guard. A nil channel
		// is the pre-ISSUE-118 behavior: no join, no advisory. engineDeclared
		// defaults true for that case so an un-wired constructor never emits the
		// verdict advisory.
		var verdicts []Violation
		verdictEngineDeclared := true
		verdictWired := verdictChannel != nil
		if verdictWired {
			verdicts, verdictEngineDeclared = verdictChannel()
		}

		// EITHER-absent capability guard (REQ-005). Checked BEFORE the discovery walk
		// so a partial config (globs but no patterns) cannot become a mass not-found
		// fail misattributing the config gap to the codebase.
		if !classifier.HasTestGlobs() || !matcher.HasPatterns() {
			absent := testDiscoveryCapabilityAbsent(classifier.HasTestGlobs(), matcher.HasPatterns())
			if verdictWired {
				// CLM-010. A project with a working test-verdict engine but no declared
				// test globs would otherwise still swallow a failing mandated test —
				// a narrower instance of the exact defect this dimension exists to
				// close. The join takes NEITHER the classifier NOR the matcher, so it
				// is callable here; the advisory stays warning-severity and still says
				// what it says, but a blocking verdict alongside it makes the step fail.
				//
				// THE nil SCOPE IS DELIBERATE — DO NOT PASS `scope` HERE, AND DO NOT
				// HAND-ROLL A FilePath/SpecFile CHECK AS A CONSISTENCY GESTURE.
				// ResolveMandatedTestPaths has not run above this guard, so mt.FilePath
				// is structurally "" and GateScope.Contains("") is false in diff mode.
				// A scoped call would therefore keep a mandated test only when its SPEC
				// FILE happened to land in the diff — and in an all-test-file diff, the
				// canonical ISSUE-118 shape, it does not. The guard would silently eat
				// the very verdict this code exists to surface.
				//
				// Path resolution IS the absent capability on this path, so there is
				// nothing honest to scope against. The accepted cost is NOISE (a
				// possibly out-of-diff verdict) chosen over SILENCE. Attribution falls
				// back to the mandated test's SPEC FILE for the same reason: a degraded
				// LOCATION on a verdict that is still reported and still blocks.
				//
				// TestTestVerification_VerdictSurvivesDiscoveryCapabilityAbsent pins
				// this under a diff-mode scope. Narrowing it reds that test, and that
				// alarm is the point.
				absent.Violations = append(absent.Violations, MandatedTestFailures(mandated, verdicts, nil)...)
				absent.Status = StepVerdict(absent.Violations)
			}
			return absent
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
				// Attribute the finding to the mandated test's file (falling back to the
				// spec that declared it) in the ONE canonical repo-relative form
				// (ISSUE-046), so its identity is scope-stable. scope may be nil
				// (all-mode entry), so ProjectRoot is read defensively.
				projectRoot := ""
				if scope != nil {
					projectRoot = scope.ProjectRoot
				}
				file := mt.FilePath
				if file == "" {
					file = mt.SpecFile
				}
				violations = append(violations, Violation{
					Rule:     "test_verification",
					File:     NormalizePath(projectRoot, file),
					Message:  "mandated test function " + mt.FuncName + " not found (spec " + mt.SpecID + ", claim " + mt.ClaimID + ")",
					Severity: "error",
				})
			}
		}

		// THE VERDICT HALF. Name-presence findings come first, then verdict findings,
		// so a reader sees "missing" before "failed".
		if verdictWired {
			if !verdictEngineDeclared {
				// Un-adopted capability, not a broken promise: a NON-blocking advisory
				// naming what is missing, so the dimension never reports an unqualified
				// pass having verified nothing but spelling (CLM-006). It is APPENDED
				// rather than returned in place of the name-presence findings — a
				// genuinely missing mandated test must still block even when no verdict
				// engine exists to say whether it would have passed.
				violations = append(violations, verdictCapabilityAbsentViolation())
			} else {
				violations = append(violations, MandatedTestFailures(mandated, verdicts, scope)...)
			}
		}

		if violations == nil {
			violations = []Violation{}
		}
		return StepResult{
			StepName: StepTestVerification,
			// Severity-aware, so the warning-only capability advisory yields "warning"
			// while any error/critical finding yields "fail". A raw count here would
			// report a non-blocking advisory as a failure.
			Status:     StepVerdict(violations),
			Violations: violations,
		}
	}
}

// verdictCapabilityAbsentViolation builds the DISTINCT, VISIBLE, NON-blocking
// advisory for CLM-006: due mandated tests exist, but no installed pack declares an
// engine with `gate_type: test`, so nothing can report whether those tests passed.
//
// It is deliberately SEPARATE from testDiscoveryCapabilityAbsent's advisory and
// carries its own rule id. A project can have full test DISCOVERY and still lack a
// test-verdict ENGINE; the two name different missing pieces and a consumer needs to
// know which one to install. It reuses the same capability-absent convention
// (warning severity, ConfigErr false, exit 0) the discovery/traceability/coverage
// dimensions emit.
func verdictCapabilityAbsentViolation() Violation {
	return Violation{
		Rule: "test_verification_verdict_capability_absent",
		Message: "test-verdict capability absent: no installed pack declares an engine with `gate_type: test`, " +
			"so the gate can confirm the mandated tests EXIST but cannot report whether they PASSED — " +
			"install/declare a toolchain pack carrying a `gate_type: test` engine. This advisory is " +
			"non-blocking (exit 0); it is NOT a report that any mandated test failed.",
		Severity: "warning",
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
		defer func() { _ = f.Close() }()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			for _, name := range matcher.FindNames(scanner.Text()) {
				found[name] = path
			}
		}
		return nil
	})

	return found
}

// SPEC-037 (BUNDLE-009 Seed 3): the baked go/parser substantiveness ANALYZER —
// StepTestSubstantivenessFunc / StepTestSubstantivenessScopedFunc, its go/ast
// assertion-checking predicate, hasAssertions, the assertionSelectors vocabulary,
// callsTargetPackage, and the lowercase targetPackageName helper — was DELETED here. Substantiveness is now an
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
		return nil, fmt.Errorf("reading spec dir %s: %w", specDir, err)
	}

	var specs []SpecVerification
	for _, entry := range entries {
		kind, isArtifact := artifact.ClassifyFilename(entry.Name())
		if entry.IsDir() || !isArtifact || kind != artifact.KindSpec {
			continue
		}

		path := filepath.Join(specDir, entry.Name())
		fm, err := parseSpecFrontmatter(path)
		if err != nil {
			continue // skip unparseable specs
		}
		if !contractsAreDue(fm.Status) {
			continue // coverage due only at `implemented` (ISSUE-054)
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
				ImplementationPackage: fm.implementationSubject(),
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
		return nil, fmt.Errorf("reading spec dir %s: %w", specDir, err)
	}

	var contracts []ContractEntry
	for _, entry := range entries {
		kind, isArtifact := artifact.ClassifyFilename(entry.Name())
		if entry.IsDir() || !isArtifact || kind != artifact.KindSpec {
			continue
		}

		path := filepath.Join(specDir, entry.Name())
		fm, err := parseSpecFrontmatter(path)
		if err != nil {
			continue // skip unparseable specs
		}
		if !contractsAreDue(fm.Status) {
			continue // contracts are due only at `implemented` (ISSUE-051)
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
