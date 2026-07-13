package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/gate"
)

// gate_substantiveness_e2e_test.go is the REAL over-installed-pack END-TO-END proof
// (SPEC-037 REQ-010 / CLM-032..034). It INSTALLS the substantiveness pack as a LOCAL
// pack via the real distribution.Add path and runs the WHOLE production gate
// substantiveness path over it (real pack resolution → real dispatchPackEngines → real
// ast-grep → real convert-under-sandbox → SARIF → route + set-join), asserting a
// genuinely hollow backstop *_test.go yields a REAL test_substantiveness violation. It is
// UNSTUBBABLE: the negative twin runs the SAME hollow fixture uninstalled and asserts NO
// violation, so the verdict can only come from the real installed pack actually running.
// ast-grep absence is a t.Fatal (NOT t.Skip): a skipped real-engine E2E is a silent gap.

// requireAstGrepE2E fails loud (NOT skip) if ast-grep is absent.
func requireAstGrepE2E(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("ast-grep"); err != nil {
		t.Fatalf("ast-grep binary not found on PATH: %v — install ast-grep 0.43.0 "+
			"(e.g. `brew install ast-grep`); this over-installed-pack E2E hard-requires the "+
			"real binary and MUST NOT be skipped (a skip is silent vacuous green)", err)
	}
}

// hasSubstantivenessHollowViolation reports whether the step result carries a real
// test_substantiveness hollow violation for the mandated hollow test.
func hasSubstantivenessHollowViolation(res gate.StepResult) bool {
	for _, v := range res.Violations {
		if v.Rule == gate.StepTestSubstantiveness && strings.Contains(v.Message, "has no assertions (hollow)") {
			return true
		}
	}
	return false
}

// TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed (CLM-032) — with the
// substantiveness pack INSTALLED as a local pack, a genuinely hollow backstop *_test.go
// run through the WHOLE production pipeline yields a REAL test_substantiveness violation,
// proven WITHOUT a stub, WITHOUT pointing production at testdata, and NOT merely via the
// dispatch-seam spy.
func TestE2E_SubstantivenessInstalledLocalPack_RealGate_HollowRed(t *testing.T) {
	requireAstGrepE2E(t)
	repoRoot := repoRoot(t)

	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := ws.installSubstantivenessLocalPack(repoRoot); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	if res.Status != "fail" {
		t.Fatalf("a hollow test over the INSTALLED pack must FAIL the substantiveness step; got status %q, violations %#v", res.Status, res.Violations)
	}
	if !hasSubstantivenessHollowViolation(res) {
		t.Fatalf("the whole production pipeline must yield a real test_substantiveness hollow violation; got %#v", res.Violations)
	}
}

// TestE2E_SubstantivenessUninstalled_NoVacuousGreen (CLM-033) — the negative twin: with
// the local pack declaration/lock ABSENT, the SAME hollow fixture produces NO
// substantiveness violation through the production path, so the test cannot pass
// vacuously — pinning that the violation came from the real installed pack, not a
// residual baked path.
func TestE2E_SubstantivenessUninstalled_NoVacuousGreen(t *testing.T) {
	requireAstGrepE2E(t)

	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	// NOTE: do NOT install the pack — the local declaration/lock is ABSENT.

	res := ws.runProductionSubstantivenessStep()

	if hasSubstantivenessHollowViolation(res) {
		t.Fatalf("with the substantiveness pack UNINSTALLED, the production path must produce NO "+
			"substantiveness violation (no residual baked path); got %#v", res.Violations)
	}
	// Cross-check against the installed run: the SAME fixture DOES produce a violation
	// once installed, proving the negative is meaningful (not a broken harness).
	repoRoot := repoRoot(t)
	if err := ws.installSubstantivenessLocalPack(repoRoot); err != nil {
		t.Fatalf("installing local pack for cross-check: %v", err)
	}
	installedRes := ws.runProductionSubstantivenessStep()
	if !hasSubstantivenessHollowViolation(installedRes) {
		t.Fatalf("cross-check: the SAME hollow fixture must produce a violation once the pack is installed; got %#v", installedRes.Violations)
	}
}

// TestE2EWorkspaceScaffoldErrorPaths exercises the newE2EWorkspace filesystem
// error-return branches by pre-seeding directory/file collisions at each write site,
// so a scaffold failure surfaces the wrapped error instead of a silent nil. Each
// sub-case makes exactly ONE of MkdirAll(specs) / WriteFile(backstop.yml) /
// WriteFile(spec) / WriteFile(hollow) fail. No ast-grep is required — this drives the
// harness's own construction, not the real gate.
func TestE2EWorkspaceScaffoldErrorPaths(t *testing.T) {
	t.Run("mkdir_specs_fails_when_specs_is_a_file", func(t *testing.T) {
		tmp := t.TempDir()
		// A regular file at the specs/ position makes MkdirAll(tmp/specs) fail.
		if err := os.WriteFile(filepath.Join(tmp, "specs"), []byte("x"), 0o644); err != nil {
			t.Fatalf("seeding specs-as-file: %v", err)
		}
		if _, err := newE2EWorkspace(tmp); err == nil {
			t.Fatal("newE2EWorkspace must fail when specs/ cannot be created")
		}
	})

	t.Run("write_backstop_yml_fails_when_it_is_a_dir", func(t *testing.T) {
		tmp := t.TempDir()
		// backstop.yml pre-created as a directory makes WriteFile fail (is a directory).
		if err := os.Mkdir(filepath.Join(tmp, "backstop.yml"), 0o755); err != nil {
			t.Fatalf("seeding backstop.yml-as-dir: %v", err)
		}
		if _, err := newE2EWorkspace(tmp); err == nil {
			t.Fatal("newE2EWorkspace must fail when backstop.yml cannot be written")
		}
	})

	t.Run("write_spec_fails_when_spec_is_a_dir", func(t *testing.T) {
		tmp := t.TempDir()
		// e2e.spec.md pre-created as a directory (which also creates specs/) makes the
		// spec WriteFile fail while MkdirAll(specs) and the backstop.yml write succeed.
		if err := os.MkdirAll(filepath.Join(tmp, "specs", "e2e.spec.md"), 0o755); err != nil {
			t.Fatalf("seeding spec-as-dir: %v", err)
		}
		if _, err := newE2EWorkspace(tmp); err == nil {
			t.Fatal("newE2EWorkspace must fail when the spec file cannot be written")
		}
	})

	t.Run("write_hollow_fails_when_hollow_is_a_dir", func(t *testing.T) {
		tmp := t.TempDir()
		// subject_test.go pre-created as a directory makes the hollow-fixture WriteFile
		// fail after every prior write has succeeded.
		if err := os.Mkdir(filepath.Join(tmp, "subject_test.go"), 0o755); err != nil {
			t.Fatalf("seeding hollow-as-dir: %v", err)
		}
		if _, err := newE2EWorkspace(tmp); err == nil {
			t.Fatal("newE2EWorkspace must fail when the hollow fixture cannot be written")
		}
	})
}

// TestE2EInstallLocalPackErrorPath exercises installSubstantivenessLocalPack's
// distribution.Add error branch by pointing it at a repoRoot whose
// packs/substantiveness/ source is absent, so Add fails and the wrapped install error
// is returned rather than a false "installed" flag. No ast-grep is required.
func TestE2EInstallLocalPackErrorPath(t *testing.T) {
	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	// A repoRoot with no packs/substantiveness/pack.yml makes distribution.Add fail.
	bogusRepoRoot := t.TempDir()
	if err := ws.installSubstantivenessLocalPack(bogusRepoRoot); err == nil {
		t.Fatal("installSubstantivenessLocalPack must fail when the pack source is absent")
	}
	if ws.installed {
		t.Fatal("a failed install must NOT mark the workspace installed")
	}
}

// TestE2E_SubstantivenessMultiRuleDispatch_AndSandboxedConvert (CLM-034) — the pipeline
// exercises BOTH the multi-rule ast-grep dispatch (the pack's Q1 hollow + Q2 extraction
// rules both dispatch in one invocation via the pack-shipped sgconfig, ISSUE-028) AND the
// convert script under the REAL macOS sandbox (ast-grep→SARIF, ISSUE-029) — not a
// single-rule or sandbox-bypassed shortcut.
func TestE2E_SubstantivenessMultiRuleDispatch_AndSandboxedConvert(t *testing.T) {
	requireAstGrepE2E(t)
	repoRoot := repoRoot(t)

	ws, err := newE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	// Add a SUBSTANTIVE-but-noTarget mandated test alongside the hollow one so BOTH
	// rules must have dispatched: the Q1 hollow rule fires on TestE2EHollowSubject and
	// the Q2 extraction rule must have run for the noTarget verdict to be computable.
	noTargetSrc := "package other_test\n\nimport \"testing\"\n\n" +
		"func TestE2ENoTarget(t *testing.T) {\n\tgot := helperOnly()\n\tif got != 1 {\n\t\tt.Fatalf(\"x\")\n\t}\n}\n"
	if err := writeFile(t, ws.root, "other_test.go", noTargetSrc); err != nil {
		t.Fatalf("writing noTarget fixture: %v", err)
	}
	// Mandate the noTarget test too.
	appendMandatedTest(t, ws.specDir, "TestE2ENoTarget")

	if err := ws.installSubstantivenessLocalPack(repoRoot); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	res := ws.runProductionSubstantivenessStep()

	// The Q1 hollow rule dispatched: the hollow violation is present.
	if !hasSubstantivenessHollowViolation(res) {
		t.Fatalf("multi-rule dispatch: the Q1 hollow rule must have dispatched (hollow violation present); got %#v", res.Violations)
	}
	// The Q2 extraction rule dispatched: the noTarget test (which references no target
	// package `gate`) must raise a noTarget violation — that verdict is ONLY computable
	// if the extraction rule ran in the same multi-rule invocation, and the convert
	// script normalized its findings under the sandbox.
	foundNoTarget := false
	for _, v := range res.Violations {
		if v.Rule == gate.StepTestSubstantiveness && strings.Contains(v.Message, "does not call package") {
			foundNoTarget = true
		}
	}
	if !foundNoTarget {
		t.Fatalf("multi-rule dispatch: the Q2 extraction rule must have dispatched so the noTarget "+
			"set-join is computable (noTarget violation expected for TestE2ENoTarget); got %#v", res.Violations)
	}
}
