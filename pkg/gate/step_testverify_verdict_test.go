package gate

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// step_testverify_verdict_test.go drives the ISSUE-118 widening of the
// test_verification dimension: it now answers BOTH "does the mandated test exist"
// and "did it pass". The two halves are independent — a test that exists, is
// substantive, and is RED is the case that reads as a clean pass at HEAD.

// verdictSupplier builds the step's verdict channel: the routed test-verdict
// violations an earlier step collected, and whether any installed pack DECLARED a
// test-verdict engine at all.
func verdictSupplier(verdicts []Violation, engineDeclared bool) TestVerdictSupplier {
	return func() ([]Violation, bool) { return verdicts, engineDeclared }
}

// failingVerdict is a routed test-verdict finding naming funcName, shaped like the
// real converter's output ("<TestName>: <detail>").
func failingVerdict(funcName, detail string) Violation {
	return Violation{
		Rule:       "backstop/go-toolchain/go-test",
		File:       "widget_test.go", // a bare basename, exactly as a runner reports it
		Message:    funcName + ": " + detail,
		Severity:   "error",
		SourcePack: "backstop/go-toolchain",
		GateType:   engine.GateTypeTest.String(),
	}
}

// violationWithRule returns the first violation carrying rule, and whether one was
// found.
func violationWithRule(violations []Violation, rule string) (Violation, bool) {
	for _, v := range violations {
		if v.Rule == rule {
			return v, true
		}
	}
	return Violation{}, false
}

// TestTestVerification_BlocksWhenMandatedTestFailedVerdict (CLM-003, CLM-005): a
// due mandated test that EXISTS in the code — so today's name-presence check
// passes cleanly — plus a routed verdict finding naming it makes the step FAIL.
// This is the "exists, is substantive, and is RED" case that reads as a pass at
// HEAD.
func TestTestVerification_BlocksWhenMandatedTestFailedVerdict(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FixtureTestExists"},
	})
	writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

	verdicts := []Violation{failingVerdict("TestGate_FixtureTestExists", "expected 5, got 7")}
	step := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
		verdictSupplier(verdicts, true))
	result := step(context.Background())

	if result.Status != "fail" {
		t.Fatalf("Status = %q, want \"fail\"; violations: %#v", result.Status, result.Violations)
	}
	// The two halves are INDEPENDENT: the name-presence check must still report
	// nothing, because the test genuinely exists.
	if _, found := violationWithRule(result.Violations, "test_verification"); found {
		t.Fatalf("the name-presence check reported a violation for a test that exists: %#v", result.Violations)
	}
	v, found := violationWithRule(result.Violations, "mandated_test_failed")
	if !found {
		t.Fatalf("no mandated_test_failed violation: %#v", result.Violations)
	}
	if v.Severity != "critical" {
		t.Fatalf("Severity = %q, want \"critical\"", v.Severity)
	}
	if !strings.Contains(v.Message, "TestGate_FixtureTestExists") {
		t.Fatalf("message %q does not name the failing mandated test", v.Message)
	}
	if result.ConfigErr {
		t.Fatalf("ConfigErr = true; a failing mandated test is a finding, not a config error")
	}
}

// TestTestVerification_VerdictCapabilityAbsentWhenNoTestEngineDeclared (CLM-006):
// due mandated tests exist and are all found in code, but NO installed pack
// declares a test-verdict engine. The step must surface a DISTINCT non-blocking
// advisory naming what is missing rather than an unqualified pass that verified
// nothing but spelling.
func TestTestVerification_VerdictCapabilityAbsentWhenNoTestEngineDeclared(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FixtureTestExists"},
	})
	writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

	step := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
		verdictSupplier(nil, false))
	result := step(context.Background())

	if result.Status != "warning" {
		t.Fatalf("Status = %q, want \"warning\"; violations: %#v", result.Status, result.Violations)
	}
	if result.ConfigErr {
		t.Fatalf("ConfigErr = true; an un-adopted capability is a warning, not a config error")
	}
	if len(result.Violations) != 1 {
		t.Fatalf("got %d violations, want exactly 1 advisory: %#v", len(result.Violations), result.Violations)
	}
	v := result.Violations[0]
	if v.Rule != "test_verification_verdict_capability_absent" {
		t.Fatalf("Rule = %q, want \"test_verification_verdict_capability_absent\"", v.Rule)
	}
	if v.Severity != "warning" {
		t.Fatalf("Severity = %q, want \"warning\"", v.Severity)
	}
	// It must NAME what is missing and say it is non-blocking, so a reader can act
	// on it rather than guessing.
	for _, want := range []string{"gate_type: test", "non-blocking"} {
		if !strings.Contains(v.Message, want) {
			t.Fatalf("advisory message %q does not carry %q", v.Message, want)
		}
	}
	// It must be DISTINCT from the discovery advisory: a project can have discovery
	// and lack a verdict engine, and the two name different missing pieces.
	if v.Rule == "test_verification_capability_absent" {
		t.Fatalf("the verdict advisory collapsed into the discovery advisory")
	}
	// NON-BLOCKING: a warning-only result must not make the gate fail.
	if blocksVerdict(v) {
		t.Fatalf("the capability advisory blocks the verdict; it must be non-blocking (exit 0)")
	}
}

// TestTestVerification_VerdictPresentAndGreenIsCleanPass (CLM-008): a declared
// verdict engine that produced NO failing findings is a plain pass with zero
// violations and NO advisory. The advisory fires on ABSENCE of the capability,
// never on a green run.
func TestTestVerification_VerdictPresentAndGreenIsCleanPass(t *testing.T) {
	specDir := t.TempDir()
	codeDir := t.TempDir()
	writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
		{"CLM-001", "TestGate_FixtureTestExists"},
	})
	writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

	step := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
		verdictSupplier(nil, true))
	result := step(context.Background())

	if result.Status != "pass" {
		t.Fatalf("Status = %q, want \"pass\"; violations: %#v", result.Status, result.Violations)
	}
	if len(result.Violations) != 0 {
		t.Fatalf("a green verdict run reported %d violations: %#v", len(result.Violations), result.Violations)
	}

	// And a green run over findings naming only NON-mandated tests is still clean.
	unrelated := []Violation{failingVerdict("TestSomethingNobodyMandated", "boom")}
	green := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
		verdictSupplier(unrelated, true))(context.Background())
	if green.Status != "pass" || len(green.Violations) != 0 {
		t.Fatalf("a failure naming a NON-mandated test perturbed the dimension: %q %#v", green.Status, green.Violations)
	}
}

// TestTestVerification_NamePresenceBehaviorUnchanged (CLM-008): the existing
// missing-mandated-test, all-draft-set, and either-absent-capability behaviors are
// what they were. The verdict channel is additive; it must not perturb the
// name-presence half.
func TestTestVerification_NamePresenceBehaviorUnchanged(t *testing.T) {
	t.Run("missing mandated test still fails through the verdict constructor", func(t *testing.T) {
		specDir := t.TempDir()
		codeDir := t.TempDir()
		writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
			{"CLM-001", "TestGate_DoesNotExist"},
		})

		result := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
			verdictSupplier(nil, true))(context.Background())

		if result.Status != "fail" {
			t.Fatalf("Status = %q, want \"fail\"", result.Status)
		}
		v, found := violationWithRule(result.Violations, "test_verification")
		if !found {
			t.Fatalf("the missing-test violation is gone: %#v", result.Violations)
		}
		if !strings.Contains(v.Message, "not found") {
			t.Fatalf("the missing-test message changed: %q", v.Message)
		}
	})

	t.Run("an all-draft mandated set is still a clean pass", func(t *testing.T) {
		specDir := t.TempDir()
		codeDir := t.TempDir()
		writeScopeSpec(t, specDir, "draft.spec.md", "draft", "TestGate_NotYetWritten")

		// No verdict engine declared either — the draft set must short-circuit BEFORE
		// the verdict advisory, exactly as it short-circuits before the discovery one.
		result := StepTestVerificationVerdictFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t),
			verdictSupplier(nil, false))(context.Background())

		if result.Status != "pass" || len(result.Violations) != 0 {
			t.Fatalf("an all-draft set produced %q with %#v", result.Status, result.Violations)
		}
	})

	t.Run("the scoped constructor is unchanged and wires no verdict channel", func(t *testing.T) {
		specDir := t.TempDir()
		codeDir := t.TempDir()
		writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
			{"CLM-001", "TestGate_FixtureTestExists"},
		})
		writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

		// The pre-existing constructors (used by baseline.go / waiver.go) must keep
		// behaving exactly as before: no verdict join, no verdict advisory.
		result := StepTestVerificationScopedFunc(specDir, codeDir, nil, goTestClassifier(), goTestMatcher(t))(context.Background())

		if result.Status != "pass" || len(result.Violations) != 0 {
			t.Fatalf("the un-wired constructor changed behavior: %q %#v", result.Status, result.Violations)
		}
	})
}

// TestTestVerification_VerdictSurvivesDiscoveryCapabilityAbsent (CLM-010, sharp
// edge 15).
//
// At HEAD the step returns testDiscoveryCapabilityAbsent BEFORE anything else, so a
// project with a real, working test-verdict engine but no declared test globs (or
// no test-name patterns) still swallows a failing mandated test — a narrower
// instance of the exact defect this dimension exists to close. The verdict join
// must run ABOVE that guard and its violations be carried out WITH the advisory.
//
// THE SCOPE MODE IS THE POINT. This case is driven with a DIFF-mode scope whose
// files are entirely test files containing NEITHER the mandated test's spec file
// NOR anything it would resolve to — exactly ISSUE-118's own shape. Above the
// guard ResolveMandatedTestPaths has not run, so mt.FilePath is structurally empty
// and Contains("") is false in diff mode: a scope guard on this path would keep the
// mandated test only when its SPEC FILE landed in the diff, which here it does not,
// and the verdict would be dropped. The path is deliberately UNSCOPED. If a scope
// guard is ever reintroduced there, THIS is the test that must go red.
func TestTestVerification_VerdictSurvivesDiscoveryCapabilityAbsent(t *testing.T) {
	noGlobs := NewSourceClassifier([]string{"**/*.go"}, nil)
	noPatterns, err := NewTestNameMatcher(nil)
	if err != nil {
		t.Fatalf("NewTestNameMatcher(nil): %v", err)
	}

	cases := []struct {
		name       string
		classifier SourceClassifier
		matcher    TestNameMatcher
	}{
		{"no test globs", noGlobs, goTestMatcher(t)},
		{"no test-name patterns", goTestClassifier(), noPatterns},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			specDir := t.TempDir()
			codeDir := t.TempDir()
			writeSpecFixture(t, specDir, "test.spec.md", []struct{ id, testName string }{
				{"CLM-001", "TestGate_FixtureTestExists"},
			})
			specPath := filepath.Join(specDir, "test.spec.md")
			writeTestFile(t, codeDir, "gate_test.go", []string{"TestGate_FixtureTestExists"})

			verdicts := []Violation{failingVerdict("TestGate_FixtureTestExists", "expected 5, got 7")}

			// ISSUE-118's canonical shape: a diff-mode scope of entirely test files
			// holding neither the spec nor the mandated test's own file.
			diffScope := newGateScope("", GateScopeModeDiff,
				[]string{"pkg/other/unrelated_test.go", "pkg/other/second_test.go"}, nil)
			if diffScope.Contains(specPath) {
				t.Fatalf("the scope must NOT contain the spec file; the case is void otherwise")
			}
			for _, f := range diffScope.Files {
				if !strings.HasSuffix(f, "_test.go") {
					t.Fatalf("the diff must be ENTIRELY test files; %q is not one", f)
				}
			}

			result := StepTestVerificationVerdictFunc(specDir, codeDir, diffScope, tc.classifier, tc.matcher,
				verdictSupplier(verdicts, true))(context.Background())

			if result.Status != "fail" {
				t.Fatalf("Status = %q, want \"fail\" — the verdict blocks even though discovery is absent; violations: %#v", result.Status, result.Violations)
			}
			if result.ConfigErr {
				t.Fatalf("ConfigErr = true; the discovery gap is a warning and the verdict is a finding")
			}
			// The discovery advisory is a real, still-true statement about the project
			// and must be carried ALONGSIDE the verdict, not replaced by it.
			advisory, found := violationWithRule(result.Violations, "test_verification_capability_absent")
			if !found {
				t.Fatalf("the discovery advisory was dropped: %#v", result.Violations)
			}
			if advisory.Severity != "warning" {
				t.Fatalf("advisory Severity = %q, want \"warning\"", advisory.Severity)
			}
			verdict, found := violationWithRule(result.Violations, "mandated_test_failed")
			if !found {
				t.Fatalf("the verdict was swallowed by the discovery guard: %#v", result.Violations)
			}
			if verdict.Severity != "critical" {
				t.Fatalf("verdict Severity = %q, want \"critical\"", verdict.Severity)
			}
			// Path resolution has NOT run on this path, so the mandated test's spec
			// file is the expected (degraded) LOCATION on a verdict that is
			// nonetheless reported and blocking.
			if verdict.File != NormalizePath("", specPath) {
				t.Fatalf("verdict File = %q, want the mandated test's spec file %q", verdict.File, NormalizePath("", specPath))
			}

			// SCOPE-MODE INSENSITIVITY. The same input under all-mode must produce the
			// same outcome — this path is project-wide equivalent by design.
			allScope := newGateScope("", GateScopeModeAll, nil, nil)
			allResult := StepTestVerificationVerdictFunc(specDir, codeDir, allScope, tc.classifier, tc.matcher,
				verdictSupplier(verdicts, true))(context.Background())
			if allResult.Status != "fail" {
				t.Fatalf("all-mode Status = %q, want \"fail\"; the discovery-absent path must be scope-mode-insensitive", allResult.Status)
			}
			if _, found := violationWithRule(allResult.Violations, "mandated_test_failed"); !found {
				t.Fatalf("all-mode dropped the verdict: %#v", allResult.Violations)
			}
		})
	}
}
