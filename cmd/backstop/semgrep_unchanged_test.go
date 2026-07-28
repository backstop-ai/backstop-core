package main

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// SPEC-034 REQ-006 — semgrep is the UNCHANGED shared engine SPEC-031
// established. This spec does NOT re-implement or re-wire semgrep's findings
// path; its only semgrep-adjacent change is retiring EnsureSemgrep's bespoke
// install into the declared provisioning model (REQ-008, the Phase-5 deletion).
// These tests pin that posture so a future edit cannot silently alter the semgrep
// findings path under cover of this spec.

// TestSemgrep_UnchangedSharedEngine proves the semgrep pass retains its SPEC-031
// shared-engine shape and SARIF contract (CLM-020): it is a rule-flags findings
// engine invoked `semgrep --sarif --quiet` (SARIF natively on stdout), with NO
// pack convert script and NO CrashGuard — its non-zero-with-findings contract is
// the SARIF on stdout, exactly as SPEC-031 built it.
func TestSemgrep_UnchangedSharedEngine(t *testing.T) {
	bind, err := baseengines.Registry().Lookup("semgrep")
	if err != nil {
		t.Fatalf("semgrep must remain a registered shared engine: %v", err)
	}
	if !strings.HasPrefix(bind.Command, "semgrep") {
		t.Errorf("semgrep command must remain `semgrep ...`, got %q", bind.Command)
	}
	if !strings.Contains(bind.Command, "--sarif") {
		t.Errorf("semgrep must emit SARIF natively (--sarif) — the SPEC-031 output contract, got %q", bind.Command)
	}
	if bind.InputMode != engine.InputModeRuleFlags {
		t.Errorf("semgrep must remain a rule-flags engine (--config per rule), got %q", bind.InputMode)
	}
	if bind.InputFlag != "--config" {
		t.Errorf("semgrep input flag must remain --config, got %q", bind.InputFlag)
	}
	if bind.ScopeKind != engine.ScopeKindFileArgs {
		t.Errorf("semgrep must remain file-args scoped (scans the files it is pointed at), got %v", bind.ScopeKind)
	}
	// SARIF-native: no converter, and its non-zero-with-findings contract means it
	// must NOT carry the build/test CrashGuard.
	if bind.Convert != "" {
		t.Errorf("semgrep is SARIF-native and must declare NO convert script, got %q", bind.Convert)
	}
	if bind.CrashGuard {
		t.Error("semgrep must NOT set CrashGuard — it legitimately exits non-zero WHEN it reports findings (SARIF is its contract)")
	}
}

// TestSemgrep_OnlyProvisioningChanges proves this spec's only semgrep-adjacent
// change is retiring EnsureSemgrep's bespoke install into declared provisioning,
// NOT altering the findings path (CLM-021). Two halves so the Sharp-Edge-10
// silent-gap cannot open: (a) semgrep stays PINNED + auto-provisioned (a non-nil
// Provision record distinct from the assume-present native toolchain), and (b)
// the split-provisioning fail-loud does NOT touch semgrep — a semgrep-only pack
// passes provisionEngines even with nothing on PATH (it is provisioned, not
// assume-present).
func TestSemgrep_OnlyProvisioningChanges(t *testing.T) {
	bind, err := baseengines.Registry().Lookup("semgrep")
	if err != nil {
		t.Fatalf("semgrep lookup: %v", err)
	}
	// (a) semgrep remains pinned + auto-provisioned — distinct from the nil-Provision
	// assume-present native toolchain (go/golangci-lint).
	if bind.Provision == nil {
		t.Fatal("semgrep must remain pinned + auto-provisioned (non-nil Provision), not assume-present")
	}
	if bind.Provision.Tool != "semgrep" || bind.Provision.Version == "" {
		t.Errorf("semgrep Provision must pin the semgrep tool to a version, got %+v", bind.Provision)
	}

	// (b) The new split-provisioning fail-loud (REQ-008) does NOT apply to semgrep:
	// a semgrep-only pack passes provisioning with NOTHING on PATH, because semgrep
	// is provisioned through the declared model, not assume-present (REQ-006 keeps
	// the findings path untouched).
	withBinaryResolver(t /* nothing present */)
	if perr := provisionEngines([]*pack.Manifest{semgrepOnlyManifest(t)}); perr != nil {
		t.Fatalf("semgrep provisioning must NOT fail loud on absent PATH (it is auto-provisioned, not assume-present): %v", perr)
	}
}
