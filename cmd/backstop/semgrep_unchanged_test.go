package main

import (
	"errors"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/baseengines"
	"github.com/backstop-ai/backstop-core/pkg/check"
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

// TestSemgrep_OnlyProvisioningChanges proves semgrep's FINDINGS path is unchanged
// while only its PROVISIONING behavior moves — still two halves, so the
// Sharp-Edge-10 silent gap cannot open.
//
// (a) semgrep stays PINNED (a non-nil Provision record naming a tool + version,
//     distinct from the assume-present native toolchain) — UNCHANGED, and it is
//     the guard on the promise that this lane never touches the findings path.
//
// (b) ISSUE-112 SUPERSEDES this half's former assertion ("a semgrep-only pack
//     passes provisionEngines even with nothing on PATH"). The pin never installed
//     semgrep, so an absent semgrep scanned nothing; provisioning now REFUSES.
//     Only WHEN a tool is refused changed — never how a tool that runs is parsed.
//     The name is kept for continuity with the SPEC-034 history a reader following
//     this test will want.
//
// The CrashGuard assertion is a DIFFERENT test (TestSemgrep_UnchangedSharedEngine,
// above) and is untouched by ISSUE-112.
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

	// (b) PROVISIONING is the only thing that changed: with NOTHING on PATH, a
	// semgrep-only pack now fails loud, because the pin never installed semgrep and
	// an absent binary scans nothing (ISSUE-112). The findings path is untouched —
	// this governs WHEN a tool is refused, never how a tool that runs is parsed.
	withBinaryResolver(t /* nothing present */)
	perr := provisionEngines([]*pack.Manifest{semgrepOnlyManifest(t)})
	if perr == nil {
		t.Fatal("semgrep absent from PATH must fail loud at provisioning, got nil — a pinned tool that never ran is not a clean scan")
	}
	var cfgErr *check.ConfigError
	if !errors.As(perr, &cfgErr) {
		t.Fatalf("absent semgrep must surface as *check.ConfigError (exit 2), got %T: %v", perr, perr)
	}
	if !strings.Contains(cfgErr.Error(), "semgrep") {
		t.Errorf("the refusal must name `semgrep`, got: %v", cfgErr)
	}
}
