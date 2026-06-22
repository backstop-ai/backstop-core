package engine

import (
	"slices"
	"testing"
)

// SPEC-035 Phase 6 / TASK-032 — DefaultRegistry / DefaultFieldContracts OQ-1
// disposition invariant tests (option-AGNOSTIC).
//
// OQ-1 resolved to OPTION (i): the incremental, overridable fallback. These tests
// assert the INVARIANTS that hold under either option — the built-in bindings
// remain available to dispatch (whether sourced from DefaultRegistry fallback or
// a default pack), the trusted-tool allowlist gates a built-in binding's tool
// exactly as a pack-declared one, and the name-keyed DefaultFieldContracts /
// engineFieldClaim travel under the SAME disposition as DefaultRegistry (a pack-
// declared field-contract OVERRIDES the baked default for the same engine name).
//
// This file exercises the leaf-package half of the invariant — the registry
// merge precedence and the field-contract override at the binding level — without
// importing the cmd dispatch path. The dispatch-time allowlist half of CLM-028
// (the gate firing on a built-in binding's tool through dispatchPackEngines) lives
// in cmd/backstop/migration_test.go, which has access to the dispatch seam.

// mergeRegistries is the stage-1 merge SEMANTICS under test: a base/fallback
// registry (the built-ins, whether sourced from DefaultRegistry fallback or a
// default pack) merged with a set of pack-declared bindings, where a pack-declared
// binding of the SAME NAME OVERRIDES the built-in. It mirrors resolveEngineRegistry
// (cmd/backstop) and resolveEngineRegistryForValidation (pkg/pack): copy the base,
// then let pack-declared names win. The invariant is the precedence, not where
// this helper lives.
func mergeRegistries(base Registry, declared map[string]EngineBinding) Registry {
	merged := make(Registry, len(base))
	for name, binding := range base {
		merged[name] = binding
	}
	for name, binding := range declared {
		merged[name] = binding
	}
	return merged
}

// TestMigration_BuiltinBindingsRemainAvailable asserts that under the resolved
// merge semantics the built-in engine bindings — their commands AND their pinned
// provision versions — remain available to dispatch (CLM-027). go-toolchain rules
// (go-build/go-test/golangci) and the standards engines (semgrep/ast-grep) must
// still resolve through the merge with their commands and pins intact, whether the
// built-ins come from the DefaultRegistry fallback (option i) or a default pack
// (option ii). The test pins the commands and the pinned provision versions so a
// silent drop or version drift fails loud here.
func TestMigration_BuiltinBindingsRemainAvailable(t *testing.T) {
	base := DefaultRegistry()

	// A pack-declared engine that does NOT collide with any built-in name. After
	// the merge the built-ins must STILL be present and unchanged — extending the
	// registry with a new engine never drops the fallback built-ins.
	declared := map[string]EngineBinding{
		"acme-scan": {Command: "acme-scan --sarif", InputMode: InputModeRuleFlags, InputFlag: "--config"},
	}
	merged := mergeRegistries(base, declared)

	// Every built-in must remain dispatchable via Lookup (no fail-loud unknown).
	builtins := []string{"semgrep", "ast-grep", "sandbox", "config-file", "golangci", "go-build", "go-test"}
	for _, name := range builtins {
		if _, err := merged.Lookup(name); err != nil {
			t.Errorf("built-in engine %q must remain available to dispatch after the merge, got: %v", name, err)
		}
	}

	// The new pack-declared engine is reachable too (the merge EXTENDS, not replaces).
	if _, err := merged.Lookup("acme-scan"); err != nil {
		t.Errorf("a non-colliding pack-declared engine must be reachable after the merge, got: %v", err)
	}

	// The built-in COMMANDS survive the merge (a dropped/blanked command would make
	// the toolchain pass un-runnable).
	wantCommand := map[string]string{
		"semgrep":  "semgrep --sarif --quiet",
		"go-build": "go build",
		"go-test":  "go test",
		"golangci": "golangci-lint run --output.sarif.path stdout --show-stats=false",
	}
	for name, want := range wantCommand {
		got, err := merged.Lookup(name)
		if err != nil {
			t.Fatalf("lookup %q: %v", name, err)
		}
		if got.Command != want {
			t.Errorf("built-in %q command must survive the merge: want %q, got %q", name, want, got.Command)
		}
	}

	// The PINNED provision versions survive the merge (CLM-027 names the pins
	// explicitly — these mirror the allowlist trust floor).
	wantPin := map[string]Provision{
		"semgrep":  {Tool: "semgrep", Version: "1.96.0"},
		"ast-grep": {Tool: "ast-grep", Version: "0.43.0"},
	}
	for name, want := range wantPin {
		got, err := merged.Lookup(name)
		if err != nil {
			t.Fatalf("lookup %q: %v", name, err)
		}
		if got.Provision == nil {
			t.Errorf("built-in %q must keep its pinned Provision after the merge, got nil", name)
			continue
		}
		if *got.Provision != want {
			t.Errorf("built-in %q pinned provision must survive the merge: want %+v, got %+v", name, want, *got.Provision)
		}
	}

	// A pack-declared binding of the SAME NAME as a built-in OVERRIDES it (pack-wins
	// precedence) while the rest of the built-ins stay available. This is the
	// override half of CLM-027/CLM-004: the disposition is "fallback the merge
	// overrides", not "a frozen baked table".
	override := map[string]EngineBinding{
		"semgrep": {Command: "vendored-semgrep --sarif", InputMode: InputModeRuleFlags, InputFlag: "--config"},
	}
	overridden := mergeRegistries(base, override)
	got, err := overridden.Lookup("semgrep")
	if err != nil {
		t.Fatalf("lookup overridden semgrep: %v", err)
	}
	if got.Command != "vendored-semgrep --sarif" {
		t.Errorf("a pack-declared engine of the same name must OVERRIDE the built-in: want %q, got %q", "vendored-semgrep --sarif", got.Command)
	}
	// The base DefaultRegistry must be UNMUTATED by the override merge (the merge
	// makes a fresh map) — the built-in fallback is still the original.
	if base["semgrep"].Command != "semgrep --sarif --quiet" {
		t.Errorf("the override merge must not mutate the DefaultRegistry fallback, got %q", base["semgrep"].Command)
	}
}

// TestMigration_AllowlistAppliesToBuiltinBindingsToo asserts the trusted-tool
// allowlist (REQ-002) is the trust floor for a built-in binding's tool EXACTLY as
// for a pack-declared binding's tool (CLM-028) — the same CheckToolAllowed check
// gates both, keyed on the tool, regardless of where the binding came from. A
// built-in semgrep binding (Provision tool "semgrep") and a pack-declared binding
// that ALSO names tool "semgrep" both pass under an allowlist that pins semgrep and
// both fail loud under an allowlist that omits it. The allowlist does not special-
// case the binding source.
func TestMigration_AllowlistAppliesToBuiltinBindingsToo(t *testing.T) {
	builtin := DefaultRegistry()["semgrep"]
	if builtin.Provision == nil {
		t.Fatal("fixture invariant: the built-in semgrep binding must carry a Provision so its tool is allowlist-gated")
	}
	packDeclared := EngineBinding{
		Command:   "semgrep --sarif",
		InputMode: InputModeRuleFlags,
		InputFlag: "--config",
		Provision: &Provision{Tool: "semgrep", Version: "1.96.0"},
	}

	// An allowlist that PINS semgrep at the locked version: both the built-in and
	// the pack-declared binding (same tool) pass the SAME check.
	pinned := map[string]string{"semgrep": "1.96.0"}
	if err := CheckToolAllowed(pinned, builtin.Provision.Tool, builtin.Provision.Version); err != nil {
		t.Errorf("the allowlist must permit a built-in binding's allowlisted+pinned tool: %v", err)
	}
	if err := CheckToolAllowed(pinned, packDeclared.Provision.Tool, packDeclared.Provision.Version); err != nil {
		t.Errorf("the allowlist must permit a pack-declared binding's allowlisted+pinned tool: %v", err)
	}

	// An allowlist that OMITS semgrep: the built-in binding's tool is rejected
	// exactly as the pack-declared one — the allowlist is the trust floor
	// regardless of source. A built-in source does NOT bypass the gate.
	without := map[string]string{"some-other-tool": "1.0.0"}
	if err := CheckToolAllowed(without, builtin.Provision.Tool, builtin.Provision.Version); err == nil {
		t.Error("CLM-028: an un-allowlisted tool must be rejected even when carried by a BUILT-IN binding — the built-in source must not bypass the trust floor")
	}
	if err := CheckToolAllowed(without, packDeclared.Provision.Tool, packDeclared.Provision.Version); err == nil {
		t.Error("an un-allowlisted tool carried by a pack-declared binding must be rejected (control case)")
	}

	// Version-divergence is gated identically for a built-in binding's tool: the
	// allowlist pins a DIFFERENT version than the binding's locked version.
	diverged := map[string]string{"semgrep": "9.9.9"}
	if err := CheckToolAllowed(diverged, builtin.Provision.Tool, builtin.Provision.Version); err == nil {
		t.Error("CLM-028: a version-divergent built-in tool must fail the allowlist exactly as a pack-declared one")
	}
}

// TestFieldContract_DefaultsFollowRegistryDisposition asserts that the name-keyed
// DefaultFieldContracts() travel under the SAME OQ-1 disposition as DefaultRegistry
// (CLM-037): a pack-declared field-contract OVERRIDES the baked default for the
// same engine name — they are NOT an independent unscoped baked map immune to pack
// override. Under option (i) the baked default is the fallback a declared contract
// wins over; the resolution mirrors the binding precedence (pack-declared wins,
// built-in is the fallback).
func TestFieldContract_DefaultsFollowRegistryDisposition(t *testing.T) {
	defaults := DefaultFieldContracts()

	// resolveContract is the field-contract resolution under test (option i): a
	// binding's DECLARED FieldContract wins; an empty declared contract falls back
	// to the name-keyed baked default. This mirrors contractForEngine (pkg/pack):
	// the baked default is overridable, not an independent unscoped map.
	resolveContract := func(name string, binding EngineBinding) FieldContract {
		if len(binding.FieldContract.Requires) > 0 || len(binding.FieldContract.Forbids) > 0 {
			return binding.FieldContract
		}
		return defaults[name]
	}

	// A built-in semgrep binding with NO declared contract resolves to the baked
	// default — the fallback is still in force (built-ins stay available, CLM-037).
	bareSemgrep := DefaultRegistry()["semgrep"]
	if len(bareSemgrep.FieldContract.Requires) != 0 || len(bareSemgrep.FieldContract.Forbids) != 0 {
		t.Fatal("fixture invariant: the built-in semgrep binding must carry NO inline contract so the default fallback is exercised")
	}
	fallback := resolveContract("semgrep", bareSemgrep)
	if !slices.Contains(fallback.Requires, FieldRulePath) || !slices.Contains(fallback.Requires, FieldStandard) {
		t.Errorf("a built-in with no declared contract must fall back to the baked default for its name, got requires=%v", fallback.Requires)
	}

	// A pack-declared field-contract for the SAME engine name OVERRIDES the baked
	// default — the default is a fallback the declaration wins over, traveling under
	// the same disposition as DefaultRegistry (CLM-037). Here a pack ships a
	// "semgrep"-named binding whose declared contract differs from the baked one.
	declared := EngineBinding{
		Command:   "semgrep --sarif",
		InputMode: InputModeRuleFlags,
		InputFlag: "--config",
		FieldContract: FieldContract{
			Requires: []string{FieldRulePath},
			Forbids:  []string{FieldStandard},
		},
	}
	resolved := resolveContract("semgrep", declared)
	if slices.Contains(resolved.Requires, FieldStandard) {
		t.Errorf("CLM-037: a pack-declared contract must OVERRIDE the baked default — the default required %q but the declaration drops it; got requires=%v", FieldStandard, resolved.Requires)
	}
	if !slices.Contains(resolved.Forbids, FieldStandard) {
		t.Errorf("CLM-037: the pack-declared contract must win — it forbids %q, got forbids=%v", FieldStandard, resolved.Forbids)
	}
	if !slices.Contains(resolved.Requires, FieldRulePath) {
		t.Errorf("the pack-declared contract's own requires must be honored, got requires=%v", resolved.Requires)
	}

	// The baked DefaultFieldContracts map must be UNMUTATED by an override — it is a
	// fallback, not state the override edits. Re-reading the default still yields the
	// original semgrep contract (requires standard), proving the override is scoped
	// to the resolution, not a mutation of the baked map.
	if !slices.Contains(DefaultFieldContracts()["semgrep"].Requires, FieldStandard) {
		t.Error("the baked DefaultFieldContracts default must remain intact as the fallback after a pack-declared override")
	}
}
