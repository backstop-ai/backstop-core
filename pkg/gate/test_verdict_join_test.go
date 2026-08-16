package gate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// test_verdict_join_test.go drives the ISSUE-118 mandated-test verdict join: the
// gate-type routing of a flat dispatch stream and the by-NAME join of the routed
// test-verdict findings to the mandated tests an `implemented` spec promised.
//
// WHY THE INPUTS ARE REAL. Every finding below is parsed from
// testdata/test-verdict/go-test-failure.sarif.json, which is the DERIVED output of
// the committed go-toolchain converter run over the committed real `go test`
// capture (cmd/backstop/testdata/go-toolchain/fixtures/go-test-failures.txt). Its
// two load-bearing properties are the whole reason this join is keyed on the test
// NAME rather than the reported path:
//
//	widget_test.go / gadget_test.go are BARE BASENAMES for a package whose import
//	path is github.com/example/project/pkg/widget, so no path-keyed join can ever
//	match a gate scope's canonicalized repo-relative file set; and TestNoPos has no
//	location at all (uri "", startLine 0), so there is no path to key on whatsoever.
//
// Hand-built violations appear ONLY where no real payload can exist — the routing
// negatives (there is no captured lint/build/coverage go-test SARIF) and the pure
// string helper's contract cases.

// verdictFindings parses the derived real converter output and stamps the gate
// type a `gate_type: test` binding's dispatch would stamp, exactly as production
// delivers it to the collector.
func verdictFindings(t *testing.T) []Violation {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("testdata", "test-verdict", "go-test-failure.sarif.json"))
	if err != nil {
		t.Fatalf("reading derived converter output: %v", err)
	}
	parsed, err := check.ParsePackFindings(raw)
	if err != nil {
		t.Fatalf("parsing derived converter output as SARIF: %v", err)
	}
	if len(parsed) != 3 {
		t.Fatalf("derived fixture must carry exactly 3 findings, got %d", len(parsed))
	}
	out := make([]Violation, 0, len(parsed))
	for _, v := range parsed {
		out = append(out, Violation{
			Rule:       "backstop/go-toolchain/" + v.Rule,
			File:       v.File,
			Line:       v.Line,
			Message:    v.Message,
			Severity:   v.Severity,
			SourcePack: "backstop/go-toolchain",
			GateType:   engine.GateTypeTest.String(),
		})
	}
	return out
}

// dueMandatedTest builds an `implemented`-spec mandated test, which is the only
// kind that is DUE (ISSUE-054).
func dueMandatedTest(funcName, filePath, specFile string) MandatedTest {
	return MandatedTest{
		FuncName: funcName,
		FilePath: filePath,
		SpecFile: specFile,
		SpecID:   "SPEC-118",
		ClaimID:  "CLM-003",
		Status:   "implemented",
	}
}

// TestRouteTestVerdictFindings_RoutesByDeclaredGateTypeOnly (CLM-002): the
// test-verdict subset is selected by the DECLARED gate type alone. The stream
// deliberately contains a lint-typed violation whose Rule string contains "test"
// and a test-typed violation whose Rule string contains "lint", so any routing
// that sniffs the rule/pack/message name gets BOTH wrong.
func TestRouteTestVerdictFindings_RoutesByDeclaredGateTypeOnly(t *testing.T) {
	stream := []Violation{
		{Rule: "acme/pack.test-naming", Message: "lint finding that mentions a test", GateType: engine.GateTypeLint.String()},
		{Rule: "acme/pack.lint-shaped", Message: "first routed verdict", GateType: engine.GateTypeTest.String()},
		{Rule: "acme/pack.go-build", Message: "build finding", GateType: engine.GateTypeBuild.String()},
		{Rule: "acme/pack.go-test", Message: "second routed verdict", GateType: engine.GateTypeTest.String()},
		{Rule: "acme/pack.cover", Message: "coverage finding", GateType: engine.GateTypeCoverage.String()},
		{Rule: "test_verification", Message: "a core-emitted violation naming tests", GateType: ""},
	}

	routed := RouteTestVerdictFindings(stream)

	if len(routed) != 2 {
		t.Fatalf("routed %d findings, want 2: %#v", len(routed), routed)
	}
	if routed[0].Message != "first routed verdict" || routed[1].Message != "second routed verdict" {
		t.Fatalf("routing must preserve input order, got %q then %q", routed[0].Message, routed[1].Message)
	}
	for _, v := range routed {
		if v.GateType != engine.GateTypeTest.String() {
			t.Fatalf("routed a %q-typed finding: %#v", v.GateType, v)
		}
		if strings.Contains(v.Rule, "test-naming") {
			t.Fatalf("routed the LINT-declared finding whose rule name merely contains \"test\": %#v", v)
		}
	}

	// An empty GateType is a violation no pack engine binding produced. It must
	// never be routed — otherwise every core-emitted violation joins the verdict
	// stream and the dimension starts blocking on its own findings.
	for _, v := range routed {
		if v.GateType == "" {
			t.Fatalf("routed a non-pack-produced (empty GateType) violation: %#v", v)
		}
	}

	if got := RouteTestVerdictFindings(nil); len(got) != 0 {
		t.Fatalf("routing a nil stream yielded %d findings, want 0", len(got))
	}
}

// TestRouteTestVerdictFindings_LockstepWithDeclaredGateTypeEnum (CLM-002) is the
// DRIFT GUARD for the one spelling this join routes on.
//
// pkg/gate declares its own `gateTypeTest` string rather than importing
// pkg/pack/engine, because the architecture contract for this package is
// `gate: mayDependOn: [artifact, check, config, pack, waiver]` — pack/engine is
// deliberately excluded. traceability_polarity.go declares its dimension
// spellings across the same boundary for the same reason. What an import would
// have bought (one authoritative spelling, drift caught loudly) is bought here
// BEHAVIORALLY instead, and more sharply: this test drives REAL engine.GateType
// values through the REAL router and asserts the routing decision itself.
//
// If either side is renamed or re-spelled — engine.GateTypeTest stops rendering
// "test", or gate's constant changes — this goes red. If a second gate type were
// ever wrongly routed, the exhaustive negative half catches that too. Test files
// are exempt from the architecture rule (`excludeFiles: ^.*_test\.go$`), which is
// what lets the guard import the enum where production code may not.
func TestRouteTestVerdictFindings_LockstepWithDeclaredGateTypeEnum(t *testing.T) {
	// Every gate type the enum defines, so a NEW one added later arrives here as a
	// compile-visible addition rather than silently defaulting to unrouted.
	all := []engine.GateType{
		engine.GateTypeLint,
		engine.GateTypeBuild,
		engine.GateTypeTest,
		engine.GateTypeFindings,
		engine.GateTypeCoverage,
		engine.GateTypeSubstantiveness,
		engine.GateTypeContracts,
	}

	var routedTypes []string
	for _, gt := range all {
		spelling := gt.String()
		v := Violation{Rule: "acme/pack.probe", Message: "probe finding", GateType: spelling}
		routed := RouteTestVerdictFindings([]Violation{v})

		if gt == engine.GateTypeTest {
			if len(routed) != 1 {
				t.Fatalf("a violation stamped engine.GateTypeTest.String() (%q) was NOT routed by "+
					"RouteTestVerdictFindings — pkg/gate's gateTypeTest constant has drifted from "+
					"the pack-declared enum spelling", spelling)
			}
			routedTypes = append(routedTypes, spelling)
			continue
		}
		if len(routed) != 0 {
			t.Fatalf("a violation stamped %q (engine gate type %v) was routed as a test verdict; "+
				"only the test gate type may route", spelling, gt)
		}
	}

	if len(routedTypes) != 1 || routedTypes[0] != gateTypeTest {
		t.Fatalf("exactly one gate type must route and it must equal gate's own gateTypeTest (%q), got %v",
			gateTypeTest, routedTypes)
	}
}

// TestMandatedTestFailures_BlocksOnNamedFailingMandatedTest (CLM-003): a due
// mandated test NAMED by a routed verdict finding produces exactly one blocking,
// non-waivable `mandated_test_failed` violation carrying the failure detail.
func TestMandatedTestFailures_BlocksOnNamedFailingMandatedTest(t *testing.T) {
	verdicts := verdictFindings(t)

	t.Run("finding with a reported position", func(t *testing.T) {
		mandated := []MandatedTest{dueMandatedTest("TestWidgetFrobnicate", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}

		got := MandatedTestFailures(mandated, verdicts, nil)

		if len(got) != 1 {
			t.Fatalf("got %d violations, want exactly 1: %#v", len(got), got)
		}
		v := got[0]
		if v.Rule != "mandated_test_failed" {
			t.Fatalf("Rule = %q, want \"mandated_test_failed\"", v.Rule)
		}
		// Sharp edge 3: `critical` is the ONLY route by which a core-emitted rule
		// reaches the production waiver policy's non-waivable set, which is keyed to
		// the severity {"critical"} and otherwise harvested exclusively from pack
		// manifests. `error` blocks identically but is silently WAIVABLE.
		if v.Severity != "critical" {
			t.Fatalf("Severity = %q, want \"critical\" — \"error\" silently makes a failing mandated test waivable", v.Severity)
		}
		for _, want := range []string{"TestWidgetFrobnicate", "SPEC-118", "CLM-003", "TestWidgetFrobnicate: expected 5, got 7"} {
			if !strings.Contains(v.Message, want) {
				t.Fatalf("message %q does not carry %q", v.Message, want)
			}
		}
	})

	t.Run("finding with NO location at all", func(t *testing.T) {
		mandated := []MandatedTest{dueMandatedTest("TestNoPos", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}

		got := MandatedTestFailures(mandated, verdicts, nil)

		if len(got) != 1 {
			t.Fatalf("got %d violations, want exactly 1: %#v", len(got), got)
		}
		if got[0].Rule != "mandated_test_failed" || got[0].Severity != "critical" {
			t.Fatalf("locationless finding produced %q at %q, want mandated_test_failed/critical", got[0].Rule, got[0].Severity)
		}
		if !strings.Contains(got[0].Message, "TestNoPos failed") {
			t.Fatalf("message %q lost the finding's own detail", got[0].Message)
		}
	})

	t.Run("several findings naming one mandated test dedupe to one violation", func(t *testing.T) {
		mandated := []MandatedTest{dueMandatedTest("TestNoPos", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}
		doubled := append(append([]Violation{}, verdicts...), Violation{
			Rule:     "backstop/go-toolchain/go-test",
			Message:  "TestNoPos/subcase failed too",
			Severity: "error",
			GateType: engine.GateTypeTest.String(),
		})

		got := MandatedTestFailures(mandated, doubled, nil)

		if len(got) != 1 {
			t.Fatalf("got %d violations for one mandated test, want exactly 1: %#v", len(got), got)
		}
		if !strings.Contains(got[0].Message, "subcase failed too") {
			t.Fatalf("the additional detail was dropped rather than folded in: %q", got[0].Message)
		}
	})
}

// TestMandatedTestFailures_AttributesToMandatedTestFileNotFindingPath (CLM-003,
// CLM-004): attribution is the MANDATED TEST'S own resolved file, never the
// finding's unresolvable reported path — an unresolvable path leaking into
// identity/scope is exactly what re-opens the defect.
func TestMandatedTestFailures_AttributesToMandatedTestFileNotFindingPath(t *testing.T) {
	verdicts := verdictFindings(t)

	t.Run("resolved FilePath wins", func(t *testing.T) {
		mandated := []MandatedTest{dueMandatedTest("TestWidgetFrobnicate", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}

		got := MandatedTestFailures(mandated, verdicts, nil)

		if len(got) != 1 {
			t.Fatalf("got %d violations, want 1", len(got))
		}
		if got[0].File != "pkg/gate/some_test.go" {
			t.Fatalf("File = %q, want the mandated test's own file", got[0].File)
		}
		if got[0].File == "widget_test.go" {
			t.Fatalf("the finding's bare-basename path leaked into attribution")
		}
		if got[0].File == "" {
			t.Fatalf("attribution is empty; a locationless violation cannot be scoped or baselined")
		}
	})

	t.Run("empty FilePath falls back to the spec file", func(t *testing.T) {
		mandated := []MandatedTest{dueMandatedTest("TestWidgetFrobnicate", "", "specs/SPEC-118.spec.md")}

		got := MandatedTestFailures(mandated, verdicts, nil)

		if len(got) != 1 {
			t.Fatalf("got %d violations, want 1", len(got))
		}
		if got[0].File != "specs/SPEC-118.spec.md" {
			t.Fatalf("File = %q, want the spec file fallback", got[0].File)
		}
		if got[0].File == "widget_test.go" || got[0].File == "" {
			t.Fatalf("File = %q — never the finding's path, never empty", got[0].File)
		}
	})
}

// TestMandatedTestFailures_BareBasenamePathStillBlocks (CLM-004) — THE FALSIFIER
// FOR THIS PHASE.
//
// COMMITTED EVIDENCE. The shipped converter emits `"uri":"widget_test.go"` for a
// package whose import path is `github.com/example/project/pkg/widget`, and emits
// NO uri at all for TestNoPos. A path-keyed join is therefore unfixable by
// construction: the scope's canonicalized file set can never contain either form.
// This test drives a diff scope whose files are entirely test files and which does
// NOT contain `widget_test.go` by path, and requires the block anyway.
func TestMandatedTestFailures_BareBasenamePathStillBlocks(t *testing.T) {
	verdicts := verdictFindings(t)
	scope := newGateScope("", GateScopeModeDiff, []string{"pkg/gate/some_test.go", "pkg/gate/other_test.go"}, nil)

	if scope.Contains("widget_test.go") {
		t.Fatalf("scope must NOT contain the finding's reported path; the case is void otherwise")
	}

	mandated := []MandatedTest{dueMandatedTest("TestWidgetFrobnicate", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}
	got := MandatedTestFailures(mandated, verdicts, scope)

	if len(got) != 1 {
		t.Fatalf("got %d violations, want 1 — the join must not need the finding's path to resolve", len(got))
	}
	if got[0].Severity != "critical" {
		t.Fatalf("Severity = %q, want \"critical\"", got[0].Severity)
	}
	if got[0].File != "pkg/gate/some_test.go" {
		t.Fatalf("File = %q, want the mandated test's own in-scope file", got[0].File)
	}
}

// TestMandatedTestFailures_BoundaryAnchoredNameMatch (CLM-009): the name match is
// boundary-anchored in BOTH directions, using the prefix collisions the committed
// fixture already supplies. The positive control keeps the anchoring from being
// over-tight — an anchor that also rejects ordinary punctuation would silently
// stop the join from ever firing.
func TestMandatedTestFailures_BoundaryAnchoredNameMatch(t *testing.T) {
	verdicts := verdictFindings(t)

	t.Run("finding longer than mandate", func(t *testing.T) {
		// The fixture's finding names TestWidgetFrobnicate. A mandate for the strict
		// prefix TestWidget must NOT be implicated by it.
		mandated := []MandatedTest{dueMandatedTest("TestWidget", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}
		if got := MandatedTestFailures(mandated, verdicts, nil); len(got) != 0 {
			t.Fatalf("TestWidget was implicated by a finding naming TestWidgetFrobnicate: %#v", got)
		}
	})

	t.Run("mandate longer than finding", func(t *testing.T) {
		// The fixture's finding names TestGadgetSpin. A mandate for the longer
		// TestGadgetSpinner must NOT be satisfied by it.
		mandated := []MandatedTest{dueMandatedTest("TestGadgetSpinner", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}
		if got := MandatedTestFailures(mandated, verdicts, nil); len(got) != 0 {
			t.Fatalf("TestGadgetSpinner was implicated by a finding naming TestGadgetSpin: %#v", got)
		}
	})

	t.Run("positive control: ordinary punctuation still matches", func(t *testing.T) {
		// A pure string contract, so constructed inputs are the right instrument —
		// these assert what the helper promises, not what any tool emits.
		for _, message := range []string{
			"TestFoo: got 3 want 4",
			"failure in TestFoo (0.00s)",
			"TestFoo",
			"--- FAIL: TestFoo",
		} {
			if !verdictNamesTest(message, "TestFoo") {
				t.Fatalf("over-tight anchoring: %q does not name TestFoo", message)
			}
		}
		for _, message := range []string{
			"TestFooBar: got 3 want 4",
			"xTestFoo failed",
			"TestFoo2 failed",
			"TestFoo_Extra failed",
		} {
			if verdictNamesTest(message, "TestFoo") {
				t.Fatalf("under-tight anchoring: %q must not name TestFoo", message)
			}
		}
	})
}

// TestMandatedTestFailures_PassingSuiteYieldsNoViolations (CLM-008): the join
// cannot be satisfied by a constant. A green suite, a suite failing only
// NON-mandated tests, and an empty due set all yield zero violations.
func TestMandatedTestFailures_PassingSuiteYieldsNoViolations(t *testing.T) {
	mandated := []MandatedTest{dueMandatedTest("TestWidgetFrobnicate", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")}

	if got := MandatedTestFailures(mandated, nil, nil); len(got) != 0 {
		t.Fatalf("an empty verdict stream produced %d violations: %#v", len(got), got)
	}

	unrelated := []Violation{{
		Rule:     "backstop/go-toolchain/go-test",
		Message:  "TestSomethingElse: boom",
		Severity: "error",
		GateType: engine.GateTypeTest.String(),
	}}
	if got := MandatedTestFailures(mandated, unrelated, nil); len(got) != 0 {
		t.Fatalf("a failure naming only a NON-mandated test produced %d violations: %#v", len(got), got)
	}

	if got := MandatedTestFailures(nil, verdictFindings(t), nil); len(got) != 0 {
		t.Fatalf("an empty mandated set produced %d violations: %#v", len(got), got)
	}
}

// TestMandatedTestFailures_OnlyDueMandatedTests (CLM-008, sharp edge 6): a
// mandated test from a NON-implemented spec is never implicated, even when a
// finding names it. A draft spec's not-yet-written test is not a broken promise.
func TestMandatedTestFailures_OnlyDueMandatedTests(t *testing.T) {
	verdicts := verdictFindings(t)

	draft := dueMandatedTest("TestWidgetFrobnicate", "pkg/gate/some_test.go", "specs/SPEC-118.spec.md")
	draft.Status = "draft"

	if got := MandatedTestFailures([]MandatedTest{draft}, verdicts, nil); len(got) != 0 {
		t.Fatalf("a draft-spec mandated test was implicated: %#v", got)
	}

	// Non-vacuity: the SAME input at `implemented` does block, so the filter is
	// what excluded it rather than the name join failing.
	implemented := draft
	implemented.Status = "implemented"
	if got := MandatedTestFailures([]MandatedTest{implemented}, verdicts, nil); len(got) != 1 {
		t.Fatalf("the implemented control must block; got %d violations", len(got))
	}
}
