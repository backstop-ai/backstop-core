package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// SPEC-035 REQ-006b/CLM-024 — file-mode PACKAGE scoping keys off the binding's
// DECLARED PackageScoped flag, NOT a "go test" command-prefix sniff (Sharp Edge
// 6). filemode_scoping_test.go pins the real go-test engine's scoping behavior;
// this file pins the DECLARED-vs-NAME divergence the retirement requires, using
// the divergent-flags fixture whose flags disagree with their command names.

// TestPackageScoped_KeyedOnDeclaredFlagNotName proves fileModeTestTarget applies
// the file-mode package scoping based on the binding's declared PackageScoped
// flag, NOT a "go test" command-prefix sniff (CLM-024, Sharp Edge 6):
//
//   - a NON-"go test" command (acme-test) with PackageScoped:true IS scoped — a
//     name-sniff would have skipped it, and
//   - a "go test"-NAMED command with PackageScoped:false is NOT scoped — a
//     name-sniff would have wrongly scoped it.
func TestPackageScoped_KeyedOnDeclaredFlagNotName(t *testing.T) {
	m := divergentFlagsManifest(t)
	// A genuine file-mode scope so the override is eligible to apply.
	scope := &gate.GateScope{Mode: gate.GateScopeModeFile, Files: []string{"pkg/widget/widget_test.go"}}

	// Positive half: non-"go test" command, PackageScoped TRUE -> package-scoped.
	// A "go test" prefix sniff would have skipped this binding (ok=false).
	pkgDivergent := m.Engines["package-scoped-divergent"].Binding
	if strings.HasPrefix(strings.TrimSpace(pkgDivergent.Command), "go test") {
		t.Fatalf("fixture invariant: package-scoped-divergent must NOT be a `go test` command, got %q", pkgDivergent.Command)
	}
	if !pkgDivergent.PackageScoped {
		t.Fatal("fixture invariant: package-scoped-divergent must declare PackageScoped:true")
	}
	// ISSUE-093: the fixture declares NO top-level `classification:` block, so BOTH
	// halves resolve through the (C') carve-out rather than state (B). That is
	// deliberate: the property under test is WHICH FLAG DECIDES, evaluated before
	// any classification is consulted, and is therefore state-independent. Adding a
	// classification block here to force state (B) would perturb a fixture
	// TestStrictSarif_GuardKeyedOnDeclaredFlagNotName also reads.
	decision := fileModeTestTargets(m, pkgDivergent, scope)
	if decision.state == fileModeNotApplicable {
		t.Error("a non-`go test` command with PackageScoped:true must be package-scoped (keyed off the flag, not the name)")
	}
	if decision.state == fileModeTargetsDerived && !argsContain(decision.targets, "./pkg/widget") {
		t.Errorf("file-mode package target must be the changed file's package ./pkg/widget, got %v", decision.targets)
	}

	// Negative half: "go test"-NAMED command, PackageScoped FALSE -> NOT scoped.
	// A "go test" prefix sniff would have wrongly scoped it (ok=true).
	goTestNamed := m.Engines["go-test-named-no-pkgscope"].Binding
	if !strings.HasPrefix(strings.TrimSpace(goTestNamed.Command), "go test") {
		t.Fatalf("fixture invariant: go-test-named-no-pkgscope must be a `go test` command, got %q", goTestNamed.Command)
	}
	if goTestNamed.PackageScoped {
		t.Fatal("fixture invariant: go-test-named-no-pkgscope must declare PackageScoped:false")
	}
	if d := fileModeTestTargets(m, goTestNamed, scope); d.state != fileModeNotApplicable {
		t.Errorf("a `go test`-named command with PackageScoped:false must NOT be package-scoped — the scoping keys off the declared flag, not the name; got state %v targets %v", d.state, d.targets)
	}
}

// ── ISSUE-093 ────────────────────────────────────────────────────────────────
// The file-mode package-target derivation is CLAIM-GATED: an engine is dispatched
// under a file-mode scope only when the DISPATCHING pack's own declared
// `classification:` globs claim at least one scoped file. Core never asks "is this
// a directory of <language> source" — it asks the pack what it owns, which is pack
// DATA that is already parsed.
//
// Every manifest below is SYNTHETIC (`acme/*`) with a NEUTRAL command, so no
// tool or ecosystem literal decides any control flow under test.

// claimingManifest builds a manifest that DECLARES classification globs (state
// (B)/(C) territory: the pack has said what it owns, so "claims nothing" is
// FALSE rather than UNKNOWABLE).
func claimingManifest(name string, source, test []string) *pack.Manifest {
	return &pack.Manifest{
		NormalizedName: name,
		Classification: pack.Classification{Source: source, Test: test},
	}
}

// packageScopedBinding is the shape that trips ISSUE-093: a project-wide engine
// that DECLARES PackageScoped with a distinctive ProjectTarget sentinel, so a
// fallback to the project target is unmistakable in the recorded args.
func packageScopedBinding(projectTarget string) engine.EngineBinding {
	return engine.EngineBinding{
		Command:       "acme-test run",
		InputMode:     engine.InputModeNone,
		ScopeKind:     engine.ScopeKindProjectWide,
		ProjectTarget: projectTarget,
		PackageScoped: true,
		CrashGuard:    true,
	}
}

// fileScope builds a file-mode gate scope over the given paths.
func fileScope(files ...string) *gate.GateScope {
	return &gate.GateScope{Mode: gate.GateScopeModeFile, Files: files}
}

// TestFileMode_UnclaimedFileSkipsPackageScopedEngine is THE PRIMARY DEFECT
// (CLM-001). A pack declaring source/test globs that claim NONE of the scoped
// files must not have its package-scoped engine dispatched at all — at HEAD the
// derivation hands the unclaimed file's DIRECTORY to the engine, which finds
// nothing to do there, exits non-zero, and the CrashGuard reports a legitimate
// no-op as an engine crash.
//
// The assertion is that the COMMAND WAS NEVER INVOKED, not merely that no error
// came back: "no crash returned" would also pass if the engine ran and happened
// to exit zero.
func TestFileMode_UnclaimedFileSkipsPackageScopedEngine(t *testing.T) {
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, []string{"**/*_spec.acme"})
	runner := emptySarifCapturingRunner()

	_, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope(".github/workflows/ci.yml"), packageScopedBinding("./..."), nil, runner)
	if err != nil {
		t.Fatalf("an unclaimed file must be a clean non-dispatch, not an error: %v", err)
	}
	if runner.calls != 0 {
		t.Errorf("the engine must NOT be invoked when the dispatching pack claims none of the scoped files; got %d invocation(s), args=%v", runner.calls, runner.lastArgs)
	}
}

// TestFileMode_UnclaimedSkipIsReportedNotSilent proves the skip is LOUD
// (CLM-002). A silent skip is a smaller lie than a crash but still a lie: the
// operator reads PASS and cannot tell whether the test engine ran.
//
// The severity string is asserted LITERALLY because blocksVerdict matches
// "warning" and nothing else — "warn" or "info" would silently become BLOCKING
// and turn this fix into a new false RED. File and ProjectWide are asserted
// literally too: they are the fields the NEXT test proves are load-bearing.
func TestFileMode_UnclaimedSkipIsReportedNotSilent(t *testing.T) {
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, nil)
	runner := emptySarifCapturingRunner()

	violations, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope(".github/workflows/ci.yml"), packageScopedBinding("./..."), nil, runner)
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("the skip must emit EXACTLY ONE advisory, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	if v.Severity != "warning" {
		t.Errorf("advisory Severity = %q, want exactly \"warning\" — blocksVerdict matches that string and nothing else, so any other spelling BLOCKS", v.Severity)
	}
	if !v.ProjectWide {
		t.Error("advisory ProjectWide = false; it MUST be true or packValidatorStep's FilterViolations drops it before the StepResult exists and the skip is silent in production")
	}
	if v.File != "" {
		t.Errorf("advisory File = %q, want empty — no scoped file is attributable to this pack, which is precisely why the engine was skipped", v.File)
	}
	if v.SourcePack != m.NormalizedName {
		t.Errorf("advisory SourcePack = %q, want %q", v.SourcePack, m.NormalizedName)
	}
	if !strings.Contains(v.Message, m.NormalizedName) {
		t.Errorf("advisory must NAME the pack; message = %q", v.Message)
	}
	if !strings.Contains(v.Message, "acme-test run") {
		t.Errorf("advisory must NAME the engine command; message = %q", v.Message)
	}
	if v.Rule == "" {
		t.Error("advisory must carry a Rule so it is distinguishable from the capability-absent advisory")
	}
}

// TestPackEngines_UnclaimedSkipSurvivesScopeFilter is THE POST-FILTER PROOF, and
// it is NOT redundant with the test above. Every other test in this file drives
// runFindingsEngine / dispatchPackEngines, which sit BEFORE packValidatorStep's
// `activeScope.FilterViolations(violations)` call. An advisory with File:"" and
// ProjectWide:false passes every one of them and is STILL DROPPED in production,
// leaving pack_engines reporting `pass` with zero violations — CLM-002 false end
// to end with a green test suite.
//
// SHAPE (a) was chosen: the advisory comes from a REAL dispatch and is then fed
// through the real filter. A hand-written gate.Violation{ProjectWide: true}
// literal would only re-prove that filterViolations keeps project-wide
// violations (already covered in pkg/gate/scope_test.go) and would stay GREEN if
// the PRODUCTION construction site set ProjectWide:false — the exact failure mode
// this test exists to catch.
func TestPackEngines_UnclaimedSkipSurvivesScopeFilter(t *testing.T) {
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, nil)
	runner := emptySarifCapturingRunner()

	produced, err := dispatchPackEngines(
		[]*pack.Manifest{withOneEngine(m, "acme-test", packageScopedBinding("./..."))},
		t.TempDir(), t.TempDir(), fileScope(".github/workflows/ci.yml"), runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines: %v", err)
	}
	if len(produced) != 1 {
		t.Fatalf("expected the real dispatch to produce exactly one advisory, got %d: %#v", len(produced), produced)
	}

	// The production filter, with a scope whose Files do NOT include the
	// advisory's (empty) file — exactly what packValidatorStep applies.
	filterScope := fileScope(".github/workflows/ci.yml")
	survivors := filterScope.FilterViolations(produced)
	if len(survivors) != 1 {
		t.Fatalf("the advisory was DROPPED by the production scope filter (%d survived of %d) — pack_engines would report a clean pass for an engine that never ran", len(survivors), len(produced))
	}
	if verdict := gate.StepVerdict(survivors); verdict != "warning" {
		t.Errorf("StepVerdict over the surviving advisory = %q, want \"warning\" — a skipped engine must neither read as a clean pass nor block the run", verdict)
	}
}

// TestFileMode_UnclaimedSkipNeverFallsBackToProjectTarget is the ANTI-FALLBACK
// contract (CLM-003). Falling back to ProjectTarget would run the pack's entire
// project-wide pass because someone scoped the gate to a README — both a scope
// lie and a large regression on the fast per-file loop. This extends the ratified
// ISSUE-010 CLM-003 rule from the rule-fed branch to the project-wide
// package-scoped branch.
func TestFileMode_UnclaimedSkipNeverFallsBackToProjectTarget(t *testing.T) {
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, nil)
	projectRoot := t.TempDir()
	runner := emptySarifCapturingRunner()

	if _, err := runFindingsEngine(m, t.TempDir(), projectRoot,
		fileScope("README.md"), packageScopedBinding("ALL-TARGETS"), nil, runner); err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if runner.calls != 0 {
		t.Fatalf("the engine must not run at all for an unclaimed scope; got args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, "ALL-TARGETS") {
		t.Errorf("ANTI-FALLBACK: the skip must NEVER fall through to binding.ProjectTarget; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, projectRoot) {
		t.Errorf("ANTI-FALLBACK: the skip must not hand the engine the project root either; args=%v", runner.lastArgs)
	}
}

// TestFileMode_TestdataPathsAreNotPackageTargets proves the derivation drops
// testdata path segments FIRST (CLM-004, SE1). A pack that folds its fixture
// convention into its TEST globs genuinely CLAIMS a file under testdata, so only
// the drop prevents a testdata directory becoming a package target — the same
// false RED arriving through a different door.
//
// It also pins the NEGATIVE half of the ISSUE-040 convention: the drop is an
// exact SEGMENT match, so a look-alike name is NOT dropped. A substring match
// would silently narrow the gate.
func TestFileMode_TestdataPathsAreNotPackageTargets(t *testing.T) {
	m := claimingManifest("acme/claims-testdata", []string{"**/*.acme"}, []string{"**/testdata/**"})
	runner := emptySarifCapturingRunner()

	// Only-testdata scope: the file IS claimed, but the drop empties the list, so
	// the run lands in the CLM-001 skip rather than deriving ./pkg/gate/testdata.
	violations, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope("pkg/gate/testdata/contract-target.acme"), packageScopedBinding("./..."), nil, runner)
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if runner.calls != 0 {
		t.Errorf("an all-testdata scope must land in the skip, not derive a testdata package target; args=%v", runner.lastArgs)
	}
	for _, a := range runner.lastArgs {
		if strings.Contains(a, "testdata") {
			t.Errorf("no derived target may contain a testdata segment; args=%v", runner.lastArgs)
		}
	}
	if len(violations) != 1 || violations[0].Severity != "warning" {
		t.Errorf("the all-testdata scope must produce the CLM-001 skip advisory, got %#v", violations)
	}

	// The negative half: look-alikes survive the exact-segment match and DO
	// become targets.
	lookalikeRunner := emptySarifCapturingRunner()
	if _, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope("pkg/foo/testdata_util.acme", "pkg/mytestdata/x.acme"),
		packageScopedBinding("./..."), nil, lookalikeRunner); err != nil {
		t.Fatalf("runFindingsEngine (look-alikes): %v", err)
	}
	if lookalikeRunner.calls != 1 {
		t.Fatalf("look-alike paths are NOT testdata and must be dispatched; got %d call(s)", lookalikeRunner.calls)
	}
	if !argsContain(lookalikeRunner.lastArgs, "./pkg/foo") {
		t.Errorf("testdata_util.acme is not under a testdata SEGMENT and must yield ./pkg/foo; args=%v", lookalikeRunner.lastArgs)
	}
	if !argsContain(lookalikeRunner.lastArgs, "./pkg/mytestdata") {
		t.Errorf("mytestdata is not the segment `testdata` and must yield ./pkg/mytestdata; args=%v", lookalikeRunner.lastArgs)
	}
}

// TestFileMode_ClaimIsPerPackNotMergedUnion proves the claim check reads the
// DISPATCHING pack's own classification, never the cross-pack merged classifier
// buildGateSteps assembles for coverage (CLM-005, SE2). A merged-union
// implementation passes every other test in this file and fails only this one: in
// a multi-language repo it would let pack A's engine fire because pack B claimed
// the file.
func TestFileMode_ClaimIsPerPackNotMergedUnion(t *testing.T) {
	packA := withOneEngine(claimingManifest("acme/alpha", []string{"**/*.acme"}, nil), "alpha-test", packageScopedBinding("./..."))
	packB := withOneEngine(claimingManifest("acme/beta", []string{"**/*.beta"}, nil), "beta-test", packageScopedBinding("./..."))

	runnerA := emptySarifCapturingRunner()
	if _, err := runFindingsEngine(packA, t.TempDir(), t.TempDir(),
		fileScope("pkg/thing/x.beta"), packageScopedBinding("./..."), nil, runnerA); err != nil {
		t.Fatalf("runFindingsEngine (pack A): %v", err)
	}
	if runnerA.calls != 0 {
		t.Errorf("pack A claims only **/*.acme and must NOT run for a .beta file — the claim is PER-PACK, not the merged union; args=%v", runnerA.lastArgs)
	}

	runnerB := emptySarifCapturingRunner()
	if _, err := runFindingsEngine(packB, t.TempDir(), t.TempDir(),
		fileScope("pkg/thing/x.beta"), packageScopedBinding("./..."), nil, runnerB); err != nil {
		t.Fatalf("runFindingsEngine (pack B): %v", err)
	}
	if runnerB.calls != 1 {
		t.Fatalf("pack B DOES claim the scoped file and must run; got %d call(s)", runnerB.calls)
	}
	if !argsContain(runnerB.lastArgs, "./pkg/thing") {
		t.Errorf("pack B must receive the claimed file's package ./pkg/thing; args=%v", runnerB.lastArgs)
	}
}

// TestFileMode_UndeclaredClassificationPreservesDerivationAndWarns is the (C')
// CARVE-OUT (CLM-006). A pack that declares package_scoped but NO classification
// globs has not told us what it owns, so "claims nothing" is UNKNOWABLE, not
// FALSE. Collapsing (C') into the skip would turn a missing declaration into a
// silent pass — the exact vacuous-green shape ISSUE-112 filed. So today's
// derivation SHAPE is preserved and a DISTINCT capability-absent advisory fires.
//
// Two states that report identically are one state, so the Rule must differ from
// the skip notice's while the mandated scope-field shape stays the same.
func TestFileMode_UndeclaredClassificationPreservesDerivationAndWarns(t *testing.T) {
	m := &pack.Manifest{NormalizedName: "acme/undeclared"} // no classification block
	runner := emptySarifCapturingRunner()

	violations, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope("pkg/widget/widget_spec.acme"), packageScopedBinding("./..."), nil, runner)
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if runner.calls != 1 {
		t.Fatalf("(C') must PRESERVE the derivation and dispatch; got %d call(s)", runner.calls)
	}
	if !argsContain(runner.lastArgs, "./pkg/widget") {
		t.Errorf("(C') must derive the scoped file's package exactly as today; args=%v", runner.lastArgs)
	}
	if argsContain(runner.lastArgs, "./...") {
		t.Errorf("(C') must not fall back to the project target; args=%v", runner.lastArgs)
	}
	if len(violations) != 1 {
		t.Fatalf("(C') must emit exactly one capability-absent advisory, got %d: %#v", len(violations), violations)
	}
	v := violations[0]
	if v.Severity != "warning" || v.File != "" || !v.ProjectWide {
		t.Errorf("the capability-absent advisory must carry the SAME mandated shape as the skip notice (warning / File:\"\" / ProjectWide:true), got severity=%q file=%q projectWide=%v", v.Severity, v.File, v.ProjectWide)
	}
	if v.SourcePack != m.NormalizedName {
		t.Errorf("advisory SourcePack = %q, want %q", v.SourcePack, m.NormalizedName)
	}

	// DISTINGUISHABLE from the skip notice, or the two states report identically
	// and are therefore one state.
	skipRunner := emptySarifCapturingRunner()
	skipViolations, err := runFindingsEngine(
		claimingManifest("acme/undeclared", []string{"**/*.other"}, nil),
		t.TempDir(), t.TempDir(), fileScope("pkg/widget/widget_spec.acme"),
		packageScopedBinding("./..."), nil, skipRunner)
	if err != nil {
		t.Fatalf("runFindingsEngine (skip comparison): %v", err)
	}
	if len(skipViolations) != 1 {
		t.Fatalf("comparison run must produce the skip advisory, got %#v", skipViolations)
	}
	if v.Rule == skipViolations[0].Rule {
		t.Errorf("the capability-absent advisory and the skip notice share Rule %q — two states that report identically are one state", v.Rule)
	}
	if v.Message == skipViolations[0].Message {
		t.Errorf("the two advisories share a message: %q", v.Message)
	}
}

// TestFileMode_UndeclaredClassificationDerivesEveryScopedFile is THE CARVE-OUT'S
// OWN DEFECT-4 GUARD. "Preserve today's derivation" means preserve its SHAPE (no
// claim check, no testdata drop), NOT preserve its ARITY: a Files[0]-only (C')
// would test exactly ONE package while reading as thorough for a multi-file
// scope, re-manufacturing DEFECT-4 through the carve-out door instead of the main
// one. A Files[0]-preserving implementation passes the single-file test above and
// fails ONLY here.
func TestFileMode_UndeclaredClassificationDerivesEveryScopedFile(t *testing.T) {
	m := &pack.Manifest{NormalizedName: "acme/undeclared"}
	runner := emptySarifCapturingRunner()

	violations, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope("pkg/alpha/a.acme", "pkg/beta/b.acme", "pkg/alpha/c.acme"),
		packageScopedBinding("./..."), nil, runner)
	if err != nil {
		t.Fatalf("runFindingsEngine: %v", err)
	}
	if !argsContain(runner.lastArgs, "./pkg/alpha") || !argsContain(runner.lastArgs, "./pkg/beta") {
		t.Errorf("(C') must derive a target for EVERY scoped file, not just Files[0]; args=%v", runner.lastArgs)
	}
	if n := countArg(runner.lastArgs, "./pkg/alpha"); n != 1 {
		t.Errorf("./pkg/alpha must appear exactly once (deduped), got %d; args=%v", n, runner.lastArgs)
	}
	if len(violations) != 1 {
		t.Errorf("the capability-absent advisory is emitted once per DISPATCH, not once per file; got %d: %#v", len(violations), violations)
	}

	// THE STATED NON-BEHAVIOR, pinned so a later reader cannot mistake it for an
	// oversight: (C') does NOT drop testdata. The drop is sequenced with the claim
	// check and belongs to it; applying it here would leave an EMPTY target list
	// with no honest answer (skipping is the ISSUE-112 silent-skip shape (C')
	// exists to prevent; ProjectTarget is the fallback CLM-003 forbids). This is
	// HEAD's behavior, preserved deliberately — CLM-004's drop closes the door
	// only for packs that declare what they own.
	testdataRunner := emptySarifCapturingRunner()
	if _, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
		fileScope("pkg/x/testdata/y.acme"), packageScopedBinding("./..."), nil, testdataRunner); err != nil {
		t.Fatalf("runFindingsEngine (C' testdata): %v", err)
	}
	if !argsContain(testdataRunner.lastArgs, "./pkg/x/testdata") {
		t.Errorf("(C') deliberately does NOT drop testdata — the derived target must still be ./pkg/x/testdata; args=%v", testdataRunner.lastArgs)
	}
}

// TestFileMode_AllScopedFilesBecomePackageTargets closes DEFECT-4 on the main
// path (CLM-007). Today's derivation reads scope.Files[0] ONLY, so files 2..n are
// already never tested through the positional-args form. Making --file repeatable
// without fixing this would make a multi-file invocation LOOK thorough while
// testing exactly one package — a false green manufactured by the very fix that
// removes the crash.
func TestFileMode_AllScopedFilesBecomePackageTargets(t *testing.T) {
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, nil)

	var first []string
	for run := 0; run < 2; run++ {
		runner := emptySarifCapturingRunner()
		if _, err := runFindingsEngine(m, t.TempDir(), t.TempDir(),
			fileScope("pkg/alpha/a.acme", "pkg/beta/b.acme", "pkg/alpha/c.acme"),
			packageScopedBinding("./..."), nil, runner); err != nil {
			t.Fatalf("runFindingsEngine: %v", err)
		}
		if !argsContain(runner.lastArgs, "./pkg/alpha") {
			t.Errorf("run %d: ./pkg/alpha missing; args=%v", run, runner.lastArgs)
		}
		if !argsContain(runner.lastArgs, "./pkg/beta") {
			t.Errorf("run %d: a Files[0]-only derivation drops ./pkg/beta — every claimed file must become a target; args=%v", run, runner.lastArgs)
		}
		if n := countArg(runner.lastArgs, "./pkg/alpha"); n != 1 {
			t.Errorf("run %d: ./pkg/alpha must be deduped to exactly one occurrence, got %d; args=%v", run, n, runner.lastArgs)
		}
		if run == 0 {
			first = runner.lastArgs
			continue
		}
		if strings.Join(first, " ") != strings.Join(runner.lastArgs, " ") {
			t.Errorf("derivation order must be STABLE across runs: %v vs %v", first, runner.lastArgs)
		}
	}
}

// TestFileMode_DiffScopeStillWholeModule is the LEAK GUARD (CLM-013). None of the
// new claim-gating may reach a non-file-mode scope: diff scope, --all and a nil
// scope must all keep handing a project-wide package-scoped engine its
// ProjectTarget, so unchanged-file breakage still REDs a full run.
func TestFileMode_DiffScopeStillWholeModule(t *testing.T) {
	// The pack claims NOTHING in the file list, so a leaked claim check would be
	// unmistakable: the engine would be skipped.
	m := claimingManifest("acme/claims-source", []string{"**/*.acme"}, nil)

	for _, tc := range []struct {
		name  string
		scope *gate.GateScope
	}{
		{"diff scope with a file list", &gate.GateScope{Mode: gate.GateScopeModeDiff, Files: []string{"README.md", "docs/x.txt"}}},
		{"all scope", &gate.GateScope{Mode: gate.GateScopeModeAll, Files: []string{"README.md"}}},
		{"nil scope", nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runner := emptySarifCapturingRunner()
			if _, err := runFindingsEngine(m, t.TempDir(), t.TempDir(), tc.scope,
				packageScopedBinding("./..."), nil, runner); err != nil {
				t.Fatalf("runFindingsEngine: %v", err)
			}
			if runner.calls != 1 {
				t.Fatalf("the engine must still be dispatched outside file mode; got %d call(s)", runner.calls)
			}
			if !argsContain(runner.lastArgs, "./...") {
				t.Errorf("a non-file-mode scope must keep the engine's ProjectTarget ./...; args=%v", runner.lastArgs)
			}
		})
	}
}

// withOneEngine returns a shallow copy of m carrying exactly one engine binding
// plus one rule pointing at it, so dispatchPackEngines has something to group.
func withOneEngine(m *pack.Manifest, engineName string, binding engine.EngineBinding) *pack.Manifest {
	cp := *m
	cp.Engines = map[string]pack.EngineSpec{engineName: {Binding: binding}}
	cp.Content = pack.Content{Ruleset: pack.Ruleset{Rules: []pack.Rule{
		{ID: engineName + "-rule", Engine: engineName, Standard: "x"},
	}}}
	return &cp
}

// countArg returns how many recorded args equal target.
func countArg(args []string, target string) int {
	n := 0
	for _, a := range args {
		if a == target {
			n++
		}
	}
	return n
}
