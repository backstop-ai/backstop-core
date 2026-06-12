package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/check"
)

// TestCLI_GateBridge_PreservesRuleAndSourcePack drives realCodeChecker's
// check.Violation -> gate.Violation conversion and asserts the pack-namespaced
// rule ID survives into gate.Violation.Rule (NOT collapsed to "semgrep") and
// that SourcePack is derived as everything before the LAST "/" in the
// namespaced ID. Because pack.NamespacedRuleID prefixes a rule ID with the
// two-segment NormalizedName ("org/pack"), a full ID is "org/pack/rule-id" and
// SourcePack must be "org/pack" — not "org" — matching the layer-3 convention
// at pack_gate.go (SourcePack = manifest.NormalizedName). A non-pack violation
// with an empty Rule falls back to Pass.String(). (CLM-006)
func TestCLI_GateBridge_PreservesRuleAndSourcePack(t *testing.T) {
	in := []check.Violation{
		{
			Pass:     check.CheckTypeSemgrep,
			File:     "pkg/server/handler.go",
			Line:     31,
			Message:  "panic() is disallowed",
			Severity: "error",
			Rule:     "org/pack/rule-id",
		},
		{
			Pass:     check.CheckTypeLint,
			File:     "a.go",
			Line:     10,
			Message:  "unused variable",
			Severity: "warning",
			// no Rule — a built-in (non-pack) violation
		},
	}

	out := checkViolationsToGate(in)
	if len(out) != 2 {
		t.Fatalf("got %d gate violations, want 2", len(out))
	}

	// Pack-namespaced semgrep violation.
	packV := out[0]
	if packV.Rule != "org/pack/rule-id" {
		t.Errorf("Rule = %q, want the verbatim namespaced ID org/pack/rule-id (not %q)", packV.Rule, check.CheckTypeSemgrep.String())
	}
	if packV.SourcePack != "org/pack" {
		t.Errorf("SourcePack = %q, want org/pack (everything before the LAST slash, two-segment pack name)", packV.SourcePack)
	}
	if packV.File != "pkg/server/handler.go" {
		t.Errorf("File not carried: %q", packV.File)
	}
	if packV.Message != "panic() is disallowed" || packV.Severity != "error" {
		t.Errorf("Message/Severity not carried: %q/%q", packV.Message, packV.Severity)
	}

	// Non-pack violation: empty Rule falls back to Pass.String(), empty SourcePack.
	plainV := out[1]
	if plainV.Rule != "lint" {
		t.Errorf("Rule = %q, want fallback to Pass.String() (lint)", plainV.Rule)
	}
	if plainV.SourcePack != "" {
		t.Errorf("SourcePack = %q, want empty for a non-pack rule", plainV.SourcePack)
	}
}
