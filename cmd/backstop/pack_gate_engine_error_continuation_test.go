package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// ISSUE-209 — ONE ENGINE'S FAILURE MUST NOT ERASE EVERY OTHER ENGINE'S EVIDENCE.
//
// dispatchPackEnginesWithEvidence used to return on the FIRST engine error: the
// findings already collected from earlier engines and packs were dropped, and every
// later engine and pack was never dispatched at all. These tests pin the
// continue-past-error contract at the dispatcher altitude:
//
//   - every engine still runs (CLM-001);
//   - each failure is its own engine-error violation with a fixed shape (CLM-002);
//   - the output order is the dispatch order, with no sorting (CLM-003);
//   - real findings are byte-identical to what the same engine yields alone (CLM-003).
//
// The fixture engines are pack-declared findings engines with neutral commands, no
// Provision (the trust gate returns early), no Convert and a project-wide scope kind
// with no ProjectTarget — so the runner is invoked with the bare command name and its
// canned stdout is parsed directly as SARIF. Nothing here depends on a real tool.

// contTestSarif returns a valid one-result SARIF log for ruleID on file.
func contTestSarif(ruleID, file, message string) []byte {
	return []byte(`{"version":"2.1.0","runs":[{"tool":{"driver":{"name":"cont"}},"results":[` +
		`{"ruleId":"` + ruleID + `","level":"error","message":{"text":"` + message + `"},` +
		`"locations":[{"physicalLocation":{"artifactLocation":{"uri":"` + file + `"},"region":{"startLine":1}}}]}]}]}`)
}

// contTestUnparseable is engine stdout that is not SARIF: ParsePackFindings must
// refuse it, which is the findings-engine failure kind.
func contTestUnparseable() []byte {
	return []byte("error: something went wrong\nnot json at all\n")
}

// contTestBinding is a neutral findings binding invoking command with no args.
func contTestBinding(command string) engine.EngineBinding {
	return engine.EngineBinding{
		Command:   command,
		InputMode: engine.InputModeNone,
		ScopeKind: engine.ScopeKindProjectWide,
		Category:  engine.EngineCategoryOpinion,
	}
}

// contTestEngine describes one engine slot in a fixture pack. A nil binding means
// the rule names an engine absent from BOTH the base registry and manifest.Engines
// (the registry-lookup failure). sandboxMissing makes the slot a sandbox rule whose
// validator does not exist (the sandbox failure).
type contTestEngine struct {
	name           string
	binding        *engine.EngineBinding
	sandboxMissing bool
}

func contTestFindings(name string) contTestEngine {
	b := contTestBinding(name)
	return contTestEngine{name: name, binding: &b}
}

func contTestAbsentEngine(name string) contTestEngine {
	return contTestEngine{name: name}
}

func contTestSandboxMissing() contTestEngine {
	return contTestEngine{name: "sandbox", sandboxMissing: true}
}

// contTestPack builds a manifest whose rules are declared in slot order, one rule
// per engine, so the per-pack dispatch order equals the slot order.
func contTestPack(normalizedName string, slots ...contTestEngine) *pack.Manifest {
	m := &pack.Manifest{NormalizedName: normalizedName, Engines: map[string]pack.EngineSpec{}}
	for _, slot := range slots {
		switch {
		case slot.sandboxMissing:
			m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules,
				pack.Rule{ID: "v", Engine: "sandbox", Validator: "missing.sh"})
		case slot.binding == nil:
			m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules,
				pack.Rule{ID: slot.name + "-rule", Engine: slot.name, Standard: "x"})
		default:
			m.Engines[slot.name] = pack.EngineSpec{Binding: *slot.binding}
			m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules,
				pack.Rule{ID: slot.name + "-rule", Engine: slot.name, Standard: "x"})
		}
	}
	return m
}

// contTestWantEngineError is the D-1 engine-error violation shape: the step's own
// rule name, error severity, no file, scope-filter exempt, and NO SourcePack (a
// non-empty SourcePack would render as a namespaced "[org/pack/pack_engines]" row).
func contTestWantEngineError(msg string) gate.Violation {
	return gate.Violation{
		Rule:        gate.StepPackEngines,
		Severity:    "error",
		File:        "",
		ProjectWide: true,
		Message:     msg,
	}
}

// contTestAssertEngineError compares the D-1 fields exactly.
func contTestAssertEngineError(t *testing.T, got gate.Violation, wantMsg string) {
	t.Helper()
	want := contTestWantEngineError(wantMsg)
	if got.Rule != want.Rule || got.File != want.File || got.Message != want.Message ||
		got.Severity != want.Severity || got.ProjectWide != want.ProjectWide {
		t.Errorf("engine-error violation mismatch:\n got: %#v\nwant: %#v", got, want)
	}
	if got.SourcePack != "" {
		t.Errorf("an engine-error violation must carry an EMPTY SourcePack (it would render as a namespaced pack rule), got %q", got.SourcePack)
	}
}

// contTestIsEngineError reports whether v has the engine-error rule name.
func contTestIsEngineError(v gate.Violation) bool {
	return v.Rule == gate.StepPackEngines
}

// contTestDispatch runs the dispatcher with a sandbox runner that is never expected
// to apply (no convert is declared on any fixture engine).
func contTestDispatch(t *testing.T, packs []*pack.Manifest, packsDir string, runner check.CommandRunner) (packEngineDispatchResult, error) {
	t.Helper()
	sandbox := &recordingSandboxRunner{mode: packval.SandboxModeNative}
	return dispatchPackEnginesWithEvidence(packs, packsDir, t.TempDir(), nil, runner, sandbox)
}

// contTestFindingsErrMsg is the exact wrapped text the dispatcher produces for a
// failing findings engine, computed from the SAME inner call the dispatcher makes,
// so the expected string cannot drift from the real one.
func contTestFindingsErrMsg(t *testing.T, m *pack.Manifest, packsDir, engineName string, runner check.CommandRunner) string {
	t.Helper()
	binding, err := resolveEngineRegistry(m).Lookup(engineName)
	if err != nil {
		t.Fatalf("fixture: lookup %q: %v", engineName, err)
	}
	var rules []pack.Rule
	for _, r := range m.Content.Ruleset.Rules {
		if r.Engine == engineName {
			rules = append(rules, r)
		}
	}
	packRoot := filepath.Join(packsDir, filepath.FromSlash(m.NormalizedName))
	sandbox := &recordingSandboxRunner{mode: packval.SandboxModeNative}
	_, _, innerErr := runFindingsEngineWithEvidence(m, packRoot, t.TempDir(), nil, binding, rules, runner, sandbox)
	if innerErr == nil {
		t.Fatalf("fixture: engine %q in pack %s was expected to fail on its own", engineName, m.NormalizedName)
	}
	return fmt.Sprintf("dispatching findings engine %q for pack %s: %v", engineName, m.NormalizedName, innerErr)
}

// contTestCalled reports whether the fixtureRunner recorded a call for name.
func contTestCalled(r *fixtureRunner, name string) bool {
	for _, c := range r.calls {
		if c.name == name {
			return true
		}
	}
	return false
}

// TestDispatchPackEngines_EarlierFindingsSurviveLaterEngineError is the issue's
// repro (CLM-001, CLM-002): engine A emits a finding, engine B in the same pack
// returns unparseable output. A's finding must not be erased by B's failure.
func TestDispatchPackEngines_EarlierFindingsSurviveLaterEngineError(t *testing.T) {
	packsDir := t.TempDir()
	m := contTestPack("org/p1", contTestFindings("cont-a"), contTestFindings("cont-b"))
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"cont-a": contTestSarif("no-a", "a.go", "finding from A"),
		"cont-b": contTestUnparseable(),
	}}
	wantErrMsg := contTestFindingsErrMsg(t, m, packsDir, "cont-b", runner)

	result, err := contTestDispatch(t, []*pack.Manifest{m}, packsDir, runner)
	if err == nil {
		t.Fatalf("a failed engine must still surface a non-nil error, got nil with %#v", result.Violations)
	}
	if len(result.Violations) != 2 {
		t.Fatalf("want [A's finding, engine-error(B)], got %d: %#v", len(result.Violations), result.Violations)
	}
	a := result.Violations[0]
	if contTestIsEngineError(a) || a.File != "a.go" || a.Message != "finding from A" {
		t.Errorf("violation[0] must be A's real finding, got %#v", a)
	}
	contTestAssertEngineError(t, result.Violations[1], wantErrMsg)
	if err.Error() != wantErrMsg {
		t.Errorf("with one failure the error text must be today's wrapped text:\n got: %s\nwant: %s", err.Error(), wantErrMsg)
	}
}

// TestDispatchPackEngines_LaterEnginesRunAfterEarlierEngineError (CLM-001): a
// failure must not stop dispatch — the later engine in the same pack AND every
// later pack are still run, and their findings are returned.
func TestDispatchPackEngines_LaterEnginesRunAfterEarlierEngineError(t *testing.T) {
	packsDir := t.TempDir()
	p1 := contTestPack("org/p1", contTestFindings("cont-a"), contTestFindings("cont-b"))
	p2 := contTestPack("org/p2", contTestFindings("cont-c"))
	runner := &fixtureRunner{byCmd: map[string][]byte{
		"cont-a": contTestUnparseable(),
		"cont-b": contTestSarif("no-b", "b.go", "finding from B"),
		"cont-c": contTestSarif("no-c", "c.go", "finding from C"),
	}}

	result, err := contTestDispatch(t, []*pack.Manifest{p1, p2}, packsDir, runner)
	if err == nil {
		t.Fatal("the failed engine must still surface a non-nil error")
	}
	if !contTestCalled(runner, "cont-b") {
		t.Errorf("engine B (after the failure, same pack) was never dispatched; calls=%#v", runner.calls)
	}
	if !contTestCalled(runner, "cont-c") {
		t.Errorf("the later pack's engine C was never dispatched; calls=%#v", runner.calls)
	}
	var sawB, sawC bool
	for _, v := range result.Violations {
		if v.File == "b.go" && v.Message == "finding from B" {
			sawB = true
		}
		if v.File == "c.go" && v.Message == "finding from C" {
			sawC = true
		}
	}
	if !sawB || !sawC {
		t.Errorf("findings after the failure must be returned (B=%v C=%v), got %#v", sawB, sawC, result.Violations)
	}
}

// TestDispatchPackEngines_EachEngineFailureIsItsOwnErrorViolation (CLM-002): two
// failures in two packs yield two engine-error violations, each naming its own pack
// and engine, and the aggregate error carries both in dispatch order and still
// unwraps to each underlying error.
func TestDispatchPackEngines_EachEngineFailureIsItsOwnErrorViolation(t *testing.T) {
	const bareAbsent = "backstop-absent-engine-209"
	packsDir, p1 := unstartableFindingsPack(t, bareAbsent+" --sarif", false)
	p2 := contTestPack("org/p2", contTestFindings("cont-x"))
	projectRoot := t.TempDir()
	// The real exec runner: p1's engine never starts (a *check.ConfigError), p2's
	// "cont-x" binary is equally absent — two independent failures.
	runner := &check.ExecCommandRunner{Dir: projectRoot}

	result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{p1, p2}, packsDir, projectRoot, nil, runner, &recordingSandboxRunner{mode: packval.SandboxModeNative})
	if err == nil {
		t.Fatal("two failed engines must surface a non-nil error")
	}
	var engineErrs []gate.Violation
	for _, v := range result.Violations {
		if contTestIsEngineError(v) {
			engineErrs = append(engineErrs, v)
		}
	}
	if len(engineErrs) != 2 {
		t.Fatalf("want exactly two engine-error violations, got %d: %#v", len(engineErrs), result.Violations)
	}
	if !strings.Contains(engineErrs[0].Message, "acme/findings") || !strings.Contains(engineErrs[0].Message, `"acme-findings"`) {
		t.Errorf("first engine-error must name its own pack and engine, got %q", engineErrs[0].Message)
	}
	if !strings.Contains(engineErrs[1].Message, "org/p2") || !strings.Contains(engineErrs[1].Message, `"cont-x"`) {
		t.Errorf("second engine-error must name its own pack and engine, got %q", engineErrs[1].Message)
	}
	for _, v := range engineErrs {
		contTestAssertEngineError(t, v, v.Message)
	}
	first := strings.Index(err.Error(), engineErrs[0].Message)
	second := strings.Index(err.Error(), engineErrs[1].Message)
	if first < 0 || second < 0 || first >= second {
		t.Errorf("the aggregate error must carry both messages in dispatch order; got: %s", err.Error())
	}
	var cfgErr *check.ConfigError
	if !errors.As(err, &cfgErr) {
		t.Errorf("errors.As must still reach the never-started *check.ConfigError through the aggregate, got %T", err)
	}
	joined, ok := err.(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("the aggregate must expose each underlying error (Unwrap() []error), got %T", err)
	}
	parts := joined.Unwrap()
	if len(parts) != 2 {
		t.Fatalf("the aggregate must hold exactly the two engine errors, got %d", len(parts))
	}
	for i, part := range parts {
		if !errors.Is(err, part) {
			t.Errorf("errors.Is must reach underlying error %d", i)
		}
		if part.Error() != engineErrs[i].Message {
			t.Errorf("underlying error %d text must equal its violation's message:\n err: %s\nviol: %s", i, part.Error(), engineErrs[i].Message)
		}
	}
}

// TestDispatchPackEngines_MixedPackAndEngineOrderIsDeterministic (CLM-003): pack P1
// [ok, fail], pack P2 [fail, ok] yields exactly f(P1), err(P1), err(P2), f(P2) —
// dispatch order, no sorting — identically on every run, and each real finding is
// identical to what its engine yields when dispatched alone.
func TestDispatchPackEngines_MixedPackAndEngineOrderIsDeterministic(t *testing.T) {
	packsDir := t.TempDir()
	p1 := contTestPack("org/p1", contTestFindings("cont-ok1"), contTestFindings("cont-bad1"))
	p2 := contTestPack("org/p2", contTestFindings("cont-bad2"), contTestFindings("cont-ok2"))
	newRunner := func() *fixtureRunner {
		return &fixtureRunner{byCmd: map[string][]byte{
			"cont-ok1":  contTestSarif("no-one", "one.go", "finding one"),
			"cont-bad1": contTestUnparseable(),
			"cont-bad2": contTestUnparseable(),
			"cont-ok2":  contTestSarif("no-two", "two.go", "finding two"),
		}}
	}
	err1Msg := contTestFindingsErrMsg(t, p1, packsDir, "cont-bad1", newRunner())
	err2Msg := contTestFindingsErrMsg(t, p2, packsDir, "cont-bad2", newRunner())

	first, err := contTestDispatch(t, []*pack.Manifest{p1, p2}, packsDir, newRunner())
	if err == nil {
		t.Fatal("two failed engines must surface a non-nil error")
	}
	if len(first.Violations) != 4 {
		t.Fatalf("want exactly f(P1), err(P1), err(P2), f(P2); got %d: %#v", len(first.Violations), first.Violations)
	}
	if v := first.Violations[0]; contTestIsEngineError(v) || v.Message != "finding one" {
		t.Errorf("position 0 must be P1's finding, got %#v", v)
	}
	contTestAssertEngineError(t, first.Violations[1], err1Msg)
	contTestAssertEngineError(t, first.Violations[2], err2Msg)
	if v := first.Violations[3]; contTestIsEngineError(v) || v.Message != "finding two" {
		t.Errorf("position 3 must be P2's finding, got %#v", v)
	}

	second, secondErr := contTestDispatch(t, []*pack.Manifest{p1, p2}, packsDir, newRunner())
	if secondErr == nil || secondErr.Error() != err.Error() {
		t.Errorf("the aggregate error must be deterministic across runs:\n first: %v\nsecond: %v", err, secondErr)
	}
	if !reflect.DeepEqual(first.Violations, second.Violations) {
		t.Errorf("dispatch output must be deterministic across runs:\n first: %#v\nsecond: %#v", first.Violations, second.Violations)
	}

	// An unrelated engine's failure must not perturb a real finding's identity.
	alone := func(m *pack.Manifest, engineName string) gate.Violation {
		t.Helper()
		single := contTestPack(m.NormalizedName, contTestFindings(engineName))
		res, aloneErr := contTestDispatch(t, []*pack.Manifest{single}, packsDir, newRunner())
		if aloneErr != nil || len(res.Violations) != 1 {
			t.Fatalf("fixture: engine %q alone must yield one finding, got %#v err=%v", engineName, res.Violations, aloneErr)
		}
		return res.Violations[0]
	}
	for i, pair := range []struct {
		got  gate.Violation
		want gate.Violation
	}{
		{first.Violations[0], alone(p1, "cont-ok1")},
		{first.Violations[3], alone(p2, "cont-ok2")},
	} {
		g, w := pair.got, pair.want
		if g.Rule != w.Rule || g.File != w.File || g.Message != w.Message || g.Severity != w.Severity || g.SourcePack != w.SourcePack {
			t.Errorf("finding %d identity perturbed by an unrelated failure:\n got: %#v\nalone: %#v", i, g, w)
		}
	}
}

// TestDispatchPackEngines_SandboxAndRegistryErrorsContinueLikeFindingsErrors
// (CLM-001, CLM-002): the sandbox failure and the registry-lookup failure continue
// exactly like a findings failure — the later finding survives, and the error
// violation carries today's wrapped text for that kind.
func TestDispatchPackEngines_SandboxAndRegistryErrorsContinueLikeFindingsErrors(t *testing.T) {
	cases := []struct {
		name    string
		failing contTestEngine
		wantMsg func(packsDir string, m *pack.Manifest) string
	}{
		{
			name:    "sandbox engine missing validator",
			failing: contTestSandboxMissing(),
			wantMsg: func(packsDir string, m *pack.Manifest) string {
				validator := filepath.Join(packsDir, filepath.FromSlash(m.NormalizedName), "missing.sh")
				return fmt.Sprintf("dispatching sandbox engine %q for pack %s: broken pack %s: missing validator %s",
					"sandbox", m.NormalizedName, m.NormalizedName, validator)
			},
		},
		{
			name:    "registry lookup failure",
			failing: contTestAbsentEngine("cont-unregistered-209"),
			wantMsg: func(_ string, m *pack.Manifest) string {
				_, lookupErr := resolveEngineRegistry(m).Lookup("cont-unregistered-209")
				if lookupErr == nil {
					t.Fatal("fixture: the absent engine must not resolve")
				}
				return fmt.Sprintf("pack %s: %v", m.NormalizedName, lookupErr)
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			packsDir := t.TempDir()
			m := contTestPack("org/p1", tc.failing, contTestFindings("cont-after"))
			runner := &fixtureRunner{byCmd: map[string][]byte{
				"cont-after": contTestSarif("no-after", "after.go", "finding after"),
			}}

			result, err := contTestDispatch(t, []*pack.Manifest{m}, packsDir, runner)
			if err == nil {
				t.Fatal("the failed engine must surface a non-nil error")
			}
			if len(result.Violations) != 2 {
				t.Fatalf("want [engine-error, later finding], got %d: %#v", len(result.Violations), result.Violations)
			}
			want := tc.wantMsg(packsDir, m)
			contTestAssertEngineError(t, result.Violations[0], want)
			if v := result.Violations[1]; contTestIsEngineError(v) || v.File != "after.go" || v.Message != "finding after" {
				t.Errorf("the finding after the failure must survive, got %#v", v)
			}
			if err.Error() != want {
				t.Errorf("single-failure error text must be today's wrapped text:\n got: %s\nwant: %s", err.Error(), want)
			}
		})
	}
}

// TestDispatchPackEngines_SingleEngineErrorKeepsTodaysMessage (CLM-002): with ONE
// failing engine the returned error is byte-identical to the pre-change wrapped
// string, the never-started refusal still unwraps to *check.ConfigError, and the
// single engine-error violation's message equals err.Error().
func TestDispatchPackEngines_SingleEngineErrorKeepsTodaysMessage(t *testing.T) {
	t.Run("unparseable output", func(t *testing.T) {
		packsDir := t.TempDir()
		m := contTestPack("org/p1", contTestFindings("cont-bad"))
		runner := &fixtureRunner{byCmd: map[string][]byte{"cont-bad": contTestUnparseable()}}
		want := contTestFindingsErrMsg(t, m, packsDir, "cont-bad", runner)

		result, err := contTestDispatch(t, []*pack.Manifest{m}, packsDir, runner)
		if err == nil {
			t.Fatal("an unparseable engine must fail")
		}
		if err.Error() != want {
			t.Errorf("error text changed:\n got: %s\nwant: %s", err.Error(), want)
		}
		if len(result.Violations) != 1 {
			t.Fatalf("want exactly the one engine-error violation, got %#v", result.Violations)
		}
		contTestAssertEngineError(t, result.Violations[0], err.Error())
	})

	t.Run("never-started refusal", func(t *testing.T) {
		const bareAbsent = "backstop-absent-engine-209"
		packsDir, m := unstartableFindingsPack(t, bareAbsent+" --sarif", false)
		projectRoot := t.TempDir()
		runner := &check.ExecCommandRunner{Dir: projectRoot}
		binding, lookupErr := resolveEngineRegistry(m).Lookup("acme-findings")
		if lookupErr != nil {
			t.Fatalf("fixture lookup: %v", lookupErr)
		}
		_, _, innerErr := runFindingsEngineWithEvidence(m, filepath.Join(packsDir, "acme", "findings"), projectRoot, nil, binding, m.Content.Ruleset.Rules, runner, &recordingSandboxRunner{mode: packval.SandboxModeNative})
		if innerErr == nil {
			t.Fatal("fixture: the never-started engine must fail on its own")
		}
		want := fmt.Sprintf("dispatching findings engine %q for pack %s: %v", "acme-findings", "acme/findings", innerErr)

		result, err := dispatchPackEnginesWithEvidence([]*pack.Manifest{m}, packsDir, projectRoot, nil, runner, &recordingSandboxRunner{mode: packval.SandboxModeNative})
		if err == nil {
			t.Fatal("a never-started engine must fail loud")
		}
		if err.Error() != want {
			t.Errorf("error text changed:\n got: %s\nwant: %s", err.Error(), want)
		}
		var cfgErr *check.ConfigError
		if !errors.As(err, &cfgErr) {
			t.Errorf("the refusal must still unwrap to *check.ConfigError, got %T", err)
		}
		if len(result.Violations) != 1 {
			t.Fatalf("want exactly the one engine-error violation, got %#v", result.Violations)
		}
		contTestAssertEngineError(t, result.Violations[0], err.Error())
	})
}
