package main

import (
	"go/ast"
	"strings"
	"testing"
)

// REQ-005: build identity is SURFACED, never compared.

// TestDoctorBuildIdentity_ReportsStampedVersionCommitDateAndCohort (CLM-021).
//
// TWO HALVES, because either alone locks only one hop. The first drives the RENDERER
// with a fully stamped identity, so all four values are assertable without building a
// binary per case. The second drives the REAL COMMAND and compares what doctor reports
// against what `backstop version` reports — which is the assertion that would catch a
// second resolver creeping in, and which a renderer test handed a value it built itself
// can never make.
func TestDoctorBuildIdentity_ReportsStampedVersionCommitDateAndCohort(t *testing.T) {
	stamped := BuildIdentity{
		Version:   "v9.9.9",
		Commit:    "1ccb2a60b2f7",
		BuildDate: "2026-08-15T00:00:00Z",
	}
	result := describeBuildIdentity(stamped, "cohort-abc123", nil)

	if result.Status != "pass" {
		t.Errorf("status = %q on a fully stamped binary, want pass: %s", result.Status, result.Message)
	}
	for _, value := range []string{stamped.Version, stamped.Commit, stamped.BuildDate, "cohort-abc123"} {
		if !strings.Contains(result.Message, value) {
			t.Errorf("the reported identity omits %q: %q", value, result.Message)
		}
	}

	// THE WIRING HOP. `backstop version` and `backstop doctor --check build-identity`
	// must report the SAME version string for the same binary; a divergence means one of
	// them acquired a resolver of its own, which is the exact class of failure the one
	// resolution exists to close.
	project := stageDoctorProject(t, "clean")
	payload, _ := runDoctorJSON(t, project, "--check", doctorCheckBuildIdentity)
	reported := payload.field(t, doctorCheckBuildIdentity, "message")

	identity := effectiveBuildIdentity()
	if !strings.Contains(reported, identity.Version) {
		t.Errorf("doctor reports a version the one resolution does not (%q): %q", identity.Version, reported)
	}
	if !strings.Contains(reported, identity.Commit) {
		t.Errorf("doctor reports a commit the one resolution does not (%q): %q", identity.Commit, reported)
	}
	if !strings.Contains(reported, identity.BuildDate) {
		t.Errorf("doctor reports a build date the one resolution does not (%q): %q", identity.BuildDate, reported)
	}
}

// TestDoctorBuildIdentity_WarnsWhenBuildIdentityAbsent (CLM-022).
//
// WARN, NEVER FAIL. This is the highest-ranked sharp edge — a weeks-old binary reporting
// bare `dev`, whose skew was misdiagnosed as a pack producer error — made visible. The
// binary still runs, so blocking would be wrong.
func TestDoctorBuildIdentity_WarnsWhenBuildIdentityAbsent(t *testing.T) {
	cases := map[string]BuildIdentity{
		"bare dev version": {Version: "dev", Commit: "1ccb2a60b2f7", BuildDate: "2026-08-15T00:00:00Z"},
		"no commit":        {Version: "v9.9.9", Commit: unknownBuildField, BuildDate: "2026-08-15T00:00:00Z"},
		"no build date":    {Version: "v9.9.9", Commit: "1ccb2a60b2f7", BuildDate: unknownBuildField},
	}
	for name, identity := range cases {
		t.Run(name, func(t *testing.T) {
			result := describeBuildIdentity(identity, "cohort-abc123", nil)
			if result.Status == "fail" {
				t.Errorf("an absent build identity FAILED; the binary still runs, so this must never block: %s", result.Message)
			}
			if result.Status != "warn" {
				t.Errorf("status = %q, want warn: %s", result.Status, result.Message)
			}
			// The identity is still SURFACED alongside the warning: a warn that reported
			// nothing would hide the very values the check exists to make visible.
			if !strings.Contains(result.Message, identity.Version) {
				t.Errorf("the warning does not surface the version it read: %q", result.Message)
			}
		})
	}
}

// TestDoctorBuildIdentity_PerformsNoPackCapabilityComparison (CLM-023).
//
// A SOURCE-LEVEL assertion, not a reading of today's output: capability-set comparison is
// BUNDLE-020's mechanism via SPEC-068, and doctor surfaces what those stamp and
// re-decides nothing. Asserting only that the current message carries no comparison would
// pass over an implementation that read a manifest and merely said nothing about it.
func TestDoctorBuildIdentity_PerformsNoPackCapabilityComparison(t *testing.T) {
	found := false
	for _, file := range parseNonTestPackageFiles(t) {
		ast.Inspect(file.file, func(node ast.Node) bool {
			decl, ok := node.(*ast.FuncDecl)
			if !ok || decl.Body == nil || decl.Name.Name != "checkBuildIdentity" {
				return true
			}
			found = true
			ast.Inspect(decl.Body, func(inner ast.Node) bool {
				selector, isSelector := inner.(*ast.SelectorExpr)
				if !isSelector {
					return true
				}
				// No pack read of any kind: not the gathered manifests, not their error.
				if selector.Sel.Name == "Packs" || selector.Sel.Name == "PacksErr" {
					t.Errorf("checkBuildIdentity reads ctx.%s; it must perform no pack read at all, which is what keeps this claim falsifiable", selector.Sel.Name)
				}
				return true
			})
			return true
		})
	}
	if !found {
		t.Fatalf("checkBuildIdentity was not found in the non-test sources — the scan proved nothing")
	}
}
