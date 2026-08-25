package main

import (
	"context"
	"os/exec"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
	engine "github.com/backstop-ai/backstop-core/pkg/pack/engine"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// gate_contract_e2e_test.go (SPEC-038 TASK-038, REQ-014): the REAL over-installed-pack
// END-TO-END proof. With the Go contracts pack INSTALLED as a local pack (via the
// distribution helper over packs/contracts/), the WHOLE production pipeline (real pack
// resolution → buildContractStep → produceContractEngineResults → real ast-grep
// [signature presence] + real grep [symbol absence] → real convert-under-sandbox → SARIF
// → gate verdict) yields REAL blocking violations on BOTH polarities. UNSTUBBABLE: the
// negative twin runs the SAME fixtures uninstalled and asserts NO violation. t.Fatal
// (NOT t.Skip) if ast-grep/grep is absent.

func requireContractEngines(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"ast-grep", "grep"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("%s binary not found on PATH: %v — this over-installed-pack E2E hard-requires "+
				"the real binary and MUST NOT be skipped (a skip is silent vacuous green)", tool, err)
		}
	}
}

// hasContractViolation reports whether the step carries a contract_signature violation
// whose message contains the given needle.
func hasContractViolation(res gate.StepResult, needle string) bool {
	for _, v := range res.Violations {
		if v.Rule == gate.StepContractSignature && strings.Contains(v.Message, needle) {
			return true
		}
	}
	return false
}

// TestE2E_ContractsInstalledLocalPack_RealGate_MissingSignatureRed (CLM-046): with the
// contracts pack INSTALLED, a contract whose declared signature is MISSING/mismatched
// (no ast-grep match) run through the WHOLE production pipeline yields a REAL blocking
// contract violation (exit 2 / step fail), without a stub and not merely via the spy.
func TestE2E_ContractsInstalledLocalPack_RealGate_MissingSignatureRed(t *testing.T) {
	requireContractEngines(t)
	root := repoRoot(t)

	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := ws.installContractsLocalPack(root); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	res := ws.runProductionContractStep()

	if res.Status != "fail" {
		t.Fatalf("a missing/mismatched signature over the INSTALLED pack must FAIL the contract step; got status %q, violations %#v", res.Status, res.Violations)
	}
	if !hasContractViolation(res, "RouteFile") {
		t.Fatalf("the whole production pipeline must yield a real contract violation for the missing RouteFile signature; got %#v", res.Violations)
	}
}

// TestE2E_ContractsInstalledLocalPack_RealGate_PresentForbiddenSymbolRed (CLM-047): with
// the pack INSTALLED, an absence contract whose forbidden symbol is PRESENT (real grep
// match) yields a REAL blocking absence violation through the production pipeline.
func TestE2E_ContractsInstalledLocalPack_RealGate_PresentForbiddenSymbolRed(t *testing.T) {
	requireContractEngines(t)
	root := repoRoot(t)

	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := ws.installContractsLocalPack(root); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	res := ws.runProductionContractStep()

	if res.Status != "fail" {
		t.Fatalf("a present forbidden symbol over the INSTALLED pack must FAIL the contract step; got status %q", res.Status)
	}
	if !hasContractViolation(res, "legacyProbeSymbol") {
		t.Fatalf("the production pipeline must yield a real absence violation for the present legacyProbeSymbol; got %#v", res.Violations)
	}
}

// TestE2E_ContractsUninstalled_NoVacuousGreen (CLM-048): the negative twin. With the
// local pack declaration ABSENT, the SAME missing-signature + present-forbidden-symbol
// fixtures produce NO contract/absence violation through the production path — so the
// verdict can ONLY come from the real installed pack, not a residual baked path, a stub,
// or testdata-in-production.
func TestE2E_ContractsUninstalled_NoVacuousGreen(t *testing.T) {
	requireContractEngines(t)
	root := repoRoot(t)

	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	// NOTE: do NOT install the pack — the local declaration/lock is ABSENT, and the temp
	// workspace has no pkg/gate/testdata fallback, so the pack is unresolvable.

	res := ws.runProductionContractStep()

	if hasContractViolation(res, "RouteFile") || hasContractViolation(res, "legacyProbeSymbol") {
		t.Fatalf("with the contracts pack UNINSTALLED, the production path must produce NO contract "+
			"violation (no residual baked path / no testdata-in-production); got %#v", res.Violations)
	}

	// Cross-check: the SAME fixtures DO produce violations once installed, proving the
	// negative is meaningful (not a broken harness).
	if err := ws.installContractsLocalPack(root); err != nil {
		t.Fatalf("installing local pack for cross-check: %v", err)
	}
	installedRes := ws.runProductionContractStep()
	if !hasContractViolation(installedRes, "RouteFile") || !hasContractViolation(installedRes, "legacyProbeSymbol") {
		t.Fatalf("cross-check: the SAME fixtures must produce both violations once installed; got %#v", installedRes.Violations)
	}
}

// TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert (CLM-049): the pipeline
// exercises BOTH the real ast-grep signature dispatch AND the real grep absence dispatch
// through the convert scripts under the real sandbox — not a single-engine or
// sandbox-bypassed shortcut. We prove BOTH engines ran by asserting BOTH polarities of
// violation are present in the SAME run (the signature violation can only come from
// ast-grep, the absence violation only from grep).
func TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert(t *testing.T) {
	requireContractEngines(t)
	root := repoRoot(t)

	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := ws.installContractsLocalPack(root); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	// SPY on the REAL sandbox seam: record every convert script run under the sandbox.
	// This is the load-bearing guard — a raw-exec bypass (the prior defect) would NOT
	// route the convert through this seam, so this test would catch it. We wrap (not
	// replace) the real sandbox so the pipeline still produces genuine SARIF.
	var sandboxedConverts []string
	native, err := packval.NewSandboxRunner(packval.SandboxModeNative)
	if err != nil {
		t.Fatal(err)
	}
	spy := &recordingSandboxRunner{mode: packval.SandboxModeNative,
		runFn: native.Run,
		stdoutFn: func(cmd string, args []string, packDir string, stdin []byte) (packval.SandboxRunResult, error) {
			sandboxedConverts = append(sandboxedConverts, cmd)
			return native.RunStdout(cmd, args, packDir, stdin)
		},
	}

	step := buildContractStepWithSandbox(ws.specDir, ws.root, nil, spy)
	res := step(context.Background())

	// ast-grep signature dispatch ran: the missing-signature violation is present.
	if !hasContractViolation(res, "RouteFile") {
		t.Fatalf("real ast-grep signature dispatch must have run (missing-signature violation expected); got %#v", res.Violations)
	}
	// grep absence dispatch ran: the present-forbidden-symbol violation is present.
	if !hasContractViolation(res, "legacyProbeSymbol") {
		t.Fatalf("real grep absence dispatch must have run (present-symbol violation expected); got %#v", res.Violations)
	}

	// The convert MUST have run under the REAL sandbox seam (CLM-049) — a raw-exec
	// bypass would record ZERO sandboxed converts. BOTH engines' convert scripts ran
	// (ast-grep/to-sarif.sh + grep/to-sarif.sh), proving neither engine bypassed it.
	if len(sandboxedConverts) == 0 {
		t.Fatal("the contract convert MUST run under the real sandbox (packval.SandboxedRunStdout) — zero sandboxed converts means a raw-exec bypass (CLM-049)")
	}
	var sawAst, sawGrep bool
	for _, c := range sandboxedConverts {
		if strings.Contains(c, "ast-grep/to-sarif.sh") {
			sawAst = true
		}
		if strings.Contains(c, "grep/to-sarif.sh") {
			sawGrep = true
		}
	}
	if !sawAst || !sawGrep {
		t.Fatalf("both the ast-grep AND grep convert scripts must run under the sandbox (CLM-049); sandboxed converts: %v", sandboxedConverts)
	}
}

// TestContractE2EHarness_InstallErrorPath covers the install error branch (a repoRoot
// with no packs/contracts/ source makes distribution.Add fail).
func TestContractE2EHarness_InstallErrorPath(t *testing.T) {
	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding workspace: %v", err)
	}
	// A bogus repo root with no packs/contracts/ source.
	if err := ws.installContractsLocalPack(t.TempDir()); err == nil {
		t.Error("installing from a repo root with no contracts pack source must error")
	}
}

// TestE2E_ContractsGrepGatedByAllowlist (REQ-005): the contract path's grep engine is
// gated by the trusted-tool allowlist. With grep REMOVED from the allowlist, the absence
// dispatch fails loud (the engine cannot run un-trusted) — proving grep passes
// CheckToolAllowed on the contract path when allowlisted, and is blocked when not.
func TestE2E_ContractsGrepGatedByAllowlist(t *testing.T) {
	requireContractEngines(t)
	root := repoRoot(t)

	ws, err := newContractE2EWorkspace(t.TempDir())
	if err != nil {
		t.Fatalf("scaffolding e2e workspace: %v", err)
	}
	if err := ws.installContractsLocalPack(root); err != nil {
		t.Fatalf("installing local pack: %v", err)
	}

	// Remove grep/rg from the allowlist via the seam (NOT stubbed open).
	origAllow := trustedToolAllowlist
	t.Cleanup(func() { trustedToolAllowlist = origAllow })
	trustedToolAllowlist = func() map[string]string {
		stripped := map[string]string{}
		for k, v := range engine.TrustedToolAllowlist() {
			if k == "grep" || k == "rg" {
				continue
			}
			stripped[k] = v
		}
		return stripped
	}

	res := ws.runProductionContractStep()

	// The grep absence engine must be REJECTED loud (a config error), not silently run.
	if res.Status != "fail" || !res.ConfigErr {
		t.Fatalf("with grep removed from the allowlist, the contract absence dispatch must fail loud (config error); got %#v", res)
	}
	sawAllowlist := false
	for _, v := range res.Violations {
		if strings.Contains(v.Message, "allowlist") || strings.Contains(v.Message, "grep") {
			sawAllowlist = true
		}
	}
	if !sawAllowlist {
		t.Errorf("the loud error must name the un-allowlisted grep tool; got %#v", res.Violations)
	}
}
