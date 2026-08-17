package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// issue091Rule is the go-standards rule the fixture violates. It is chosen
// because it carries NO `paths:` restriction, so it fires on a `*_test.go` file
// whenever the engine is actually pointed at one. Do NOT substitute
// go.core.no-ignored-errors: that rule now excludes `*_test.go` outright and
// would prove nothing about dispatch shape.
const issue091Rule = "go.core.no-global-mutable-state"

// issue091GoStandardsManifest returns the REAL installed backstop-ai/go-standards
// manifest together with the repo's real packs dir, failing loudly when the pack
// is not installed.
//
// A MISSING PACK IS A t.Fatal, NOT A SKIP, and the polarity is deliberate. The
// load-bearing assertion below is "the *_test.go finding IS reported"; with the
// pack absent that assertion fails for a reason indistinguishable from the
// defect being unfixed — a false negative reading as "the fix does not work".
// Packs being gitignored is by design (they are reinstallable from
// backstop.lock), so an absent pack here is a broken working tree, not a
// portability condition.
func issue091GoStandardsManifest(t *testing.T, root string) (*pack.Manifest, string) {
	t.Helper()
	packDir := filepath.Join(root, ".backstop", "packs")
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("loadInstalledPacks: %v (install packs with `backstop pack install`)", err)
	}
	var manifest *pack.Manifest
	for _, m := range packs {
		if m.NormalizedName == dogfoodPackName {
			manifest = m
		}
	}
	if manifest == nil {
		t.Fatalf("pack %q is NOT installed under %s; this e2e cannot distinguish an absent pack from an unfixed dispatch. Run `backstop pack install`.", dogfoodPackName, packDir)
	}
	// Stat every declared rule file, resolved from the pack's OWN manifest
	// rule_path rather than a hard-coded rules/ layout a pack release may move.
	packRoot := filepath.Join(packDir, filepath.FromSlash(manifest.NormalizedName))
	for _, rule := range manifest.Content.Ruleset.Rules {
		rulePath := filepath.Join(packRoot, filepath.FromSlash(rule.RulePath))
		if _, statErr := os.Stat(rulePath); statErr != nil {
			t.Fatalf("pack %q declares rule %q at %s but the file is not readable: %v. Run `backstop pack install`.", dogfoodPackName, rule.ID, rulePath, statErr)
		}
	}
	return manifest, packDir
}

// issue091FixtureTree plants two files that violate issue091Rule — one
// `*_test.go` and one plain `.go` sibling — in a FRESH t.TempDir().
//
// The tempdir location is REQUIRED, not stylistic: this plan makes the all-scope
// path prune `testdata` segments, so a fixture living under the repo's
// cmd/backstop/testdata/ would be filtered out of its own test's scope and the
// test would pass vacuously with zero findings. The fixture also carries NO
// `nosemgrep` annotation, so the suppression-filtered ("ACTIVE") layer and the
// raw SARIF layer coincide for it and no count here can be layer-ambiguous.
func issue091FixtureTree(t *testing.T) (projectRoot, testFile, plainFile string) {
	t.Helper()
	projectRoot = t.TempDir()
	testFile = "mutable_state_test.go"
	plainFile = "mutable_state.go"
	writeFileStr(t, filepath.Join(projectRoot, testFile), "package fixture\n\nvar TestScopedCounter = 0\n")
	writeFileStr(t, filepath.Join(projectRoot, plainFile), "package fixture\n\nvar PlainCounter = 0\n")
	return projectRoot, testFile, plainFile
}

// issue091Identity is a (File, Rule, Line) tuple — the identity CLM-008 compares
// two dispatches on. Raw slice order is engine-determined and is never compared.
type issue091Identity struct {
	file string
	rule string
	line int
}

func issue091Identities(violations []gate.Violation) []issue091Identity {
	out := make([]issue091Identity, 0, len(violations))
	for _, v := range violations {
		out = append(out, issue091Identity{file: v.File, rule: v.Rule, line: v.Line})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].file != out[j].file {
			return out[i].file < out[j].file
		}
		if out[i].rule != out[j].rule {
			return out[i].rule < out[j].rule
		}
		return out[i].line < out[j].line
	})
	return out
}

func issue091HasFinding(violations []gate.Violation, file string) bool {
	for _, v := range violations {
		if v.File == file && strings.Contains(v.Rule, issue091Rule) {
			return true
		}
	}
	return false
}

// TestIssue091_AllScopeReportsTestFileFindings_RealEngine (CLM-002, CLM-008) is
// the LOAD-BEARING proof of this plan. The dispatch-shape tests in
// pack_gate_issue091_regression_test.go prove backstop changed what it HANDS the
// engine; only this proves the engine's REPORTED FINDINGS changed. Without it,
// "fixed" would be an arg-shape assertion dressed up as a verdict-honesty fix —
// precisely the vacuous-green class ISSUE-091 belongs to.
//
// It drives the REAL production path: dispatchPackEngines with a real
// check.ExecCommandRunner, the real installed backstop-ai/go-standards pack, and
// an all-scope from the production gate.ComputeGateScope resolver. A direct
// exec.Command semgrep invocation would bypass the very branch under change.
//
// Every count this test reports is at the ACTIVE (suppression-filtered) layer by
// construction: dispatchPackEngines returns violations that have already passed
// through parseSarif, so nosemgrep-suppressed rows are gone before the test sees
// them.
func TestIssue091_AllScopeReportsTestFileFindings_RealEngine(t *testing.T) {
	if _, lookErr := exec.LookPath("semgrep"); lookErr != nil {
		t.Skip("semgrep not installed on PATH; the ISSUE-091 verdict-honesty proof needs a real engine")
	}

	root := repoRoot(t)
	manifest, packDir := issue091GoStandardsManifest(t, root)
	projectRoot, testFile, plainFile := issue091FixtureTree(t)

	runner := &check.ExecCommandRunner{Dir: projectRoot}
	manifests := []*pack.Manifest{manifest}

	allScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
	if err != nil {
		t.Fatalf("ComputeGateScope(all): %v", err)
	}
	allViolations, err := dispatchPackEngines(manifests, packDir, projectRoot, allScope, runner)
	if err != nil {
		t.Fatalf("dispatchPackEngines(all scope): %v", err)
	}

	// CONTROL: the plain .go sibling must be reported both pre- and post-fix. If
	// the control ever stops firing, the fixture or the rule is broken and the
	// positive assertion below would be meaningless — say so rather than
	// degrading silently.
	if !issue091HasFinding(allViolations, plainFile) {
		t.Fatalf("CONTROL FAILED: the plain .go sibling %q was not flagged by %s under an all-scope run. The fixture or the rule is broken, so the positive assertion below proves nothing. ACTIVE violations: %+v", plainFile, issue091Rule, allViolations)
	}

	// POSITIVE (CLM-002): the *_test.go finding — the one an all-scope run drops
	// today — must be reported. THIS is the assertion that fails pre-fix.
	if !issue091HasFinding(allViolations, testFile) {
		t.Fatalf("all-scope run did NOT report the %s finding in %q; the *_test.go finding is still being dropped. ACTIVE violations (%d): %+v", issue091Rule, testFile, len(allViolations), allViolations)
	}

	// SUPERSET (CLM-008): dispatched over the identical file set — once via an
	// all-scope, once via a file-scope naming those files — the two finding sets
	// must be EQUAL. This is the property ISSUE-091 actually demands, and it is
	// stronger than either count alone.
	fileScope, err := gate.ComputeGateScope(projectRoot, gate.GateScopeModeFile, []string{testFile, plainFile})
	if err != nil {
		t.Fatalf("ComputeGateScope(file): %v", err)
	}
	fileRunner := &check.ExecCommandRunner{Dir: projectRoot}
	fileViolations, err := dispatchPackEngines(manifests, packDir, projectRoot, fileScope, fileRunner)
	if err != nil {
		t.Fatalf("dispatchPackEngines(file scope): %v", err)
	}

	gotAll := issue091Identities(allViolations)
	gotFile := issue091Identities(fileViolations)
	if len(gotAll) != len(gotFile) {
		t.Fatalf("all-scope and file-scope disagree over the same files: all=%d %+v, file=%d %+v", len(gotAll), gotAll, len(gotFile), gotFile)
	}
	for i := range gotAll {
		if gotAll[i] != gotFile[i] {
			t.Fatalf("all-scope and file-scope disagree at index %d: all=%+v, file=%+v (all=%+v file=%+v)", i, gotAll[i], gotFile[i], gotAll, gotFile)
		}
	}
}
