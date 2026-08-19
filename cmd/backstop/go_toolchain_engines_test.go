package main

import (
	"path/filepath"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
)

// TestGoToolchain_DeclaresBuildTestLintEngines proves the go-toolchain pack declares
// the three Go MECHANISM engines (go-build, go-test, golangci) in its own engines:
// block — alongside the existing go-coverage — so they resolve from PACK DATA, not
// the baked DefaultRegistry fallback. This declared-substrate property is exactly
// what makes the ISSUE-027 deletion of DefaultRegistry safe (CLM-009).
func TestGoToolchain_DeclaresBuildTestLintEngines(t *testing.T) {
	m := goToolchainManifest(t)

	for _, name := range []string{"go-build", "go-test", "golangci", "go-coverage"} {
		if _, ok := m.Engines[name]; !ok {
			t.Errorf("go-toolchain must declare engine %q in its engines: block, got %v", name, engineKeysOf(m.Engines))
		}
	}
}

// TestGoToolchain_GoBuildBindingFromPackData proves the go-build binding carries its
// full record from the pack manifest (command, project-wide scope + target, convert,
// crash guard, build gate type, scope-filter exemption, nil provision).
func TestGoToolchain_GoBuildBindingFromPackData(t *testing.T) {
	m := goToolchainManifest(t)
	spec, ok := m.Engines["go-build"]
	if !ok {
		t.Fatalf("go-toolchain must declare go-build engine, got %v", engineKeysOf(m.Engines))
	}
	b := spec.Binding

	if b.Command != "go build" {
		t.Errorf("go-build Command = %q, want %q", b.Command, "go build")
	}
	if b.InputMode != engine.InputModeNone {
		t.Errorf("go-build InputMode = %q, want none", b.InputMode)
	}
	if b.ScopeKind != engine.ScopeKindProjectWide {
		t.Errorf("go-build ScopeKind = %v, want project-wide", b.ScopeKind)
	}
	if b.ProjectTarget != "./..." {
		t.Errorf("go-build ProjectTarget = %q, want %q", b.ProjectTarget, "./...")
	}
	if b.Convert != "scripts/build-to-sarif.sh" {
		t.Errorf("go-build Convert = %q, want %q", b.Convert, "scripts/build-to-sarif.sh")
	}
	if !b.CrashGuard {
		t.Errorf("go-build CrashGuard = false, want true")
	}
	if b.GateType != engine.GateTypeBuild {
		t.Errorf("go-build GateType = %v, want build", b.GateType)
	}
	if !b.ExemptFromScopeFilter {
		t.Errorf("go-build ExemptFromScopeFilter = false, want true")
	}
	if b.Provision != nil {
		t.Errorf("go-build Provision = %+v, want nil (assume-present toolchain)", b.Provision)
	}
}

// TestGoToolchain_GoTestBindingFromPackData proves the go-test binding carries its
// record from pack data (command, convert, crash guard, test gate type, package
// scoping, nil provision).
func TestGoToolchain_GoTestBindingFromPackData(t *testing.T) {
	m := goToolchainManifest(t)
	spec, ok := m.Engines["go-test"]
	if !ok {
		t.Fatalf("go-toolchain must declare go-test engine, got %v", engineKeysOf(m.Engines))
	}
	b := spec.Binding

	// ISSUE-172 — THE SINGLE-RUN CONVENTION. The command carries -coverprofile so ONE
	// whole-module run emits both the test output this engine converts and the profile
	// go-coverage consumes, instead of the two independent `go test ./...` runs that
	// made the gate's two dominant steps one workload paid for twice. The claim this
	// test makes is unchanged: the binding's record comes from PACK DATA.
	const wantTestCommand = "go test -coverprofile=cover.out"
	if b.Command != wantTestCommand {
		t.Errorf("go-test Command = %q, want %q", b.Command, wantTestCommand)
	}
	if b.Convert != "scripts/test-to-sarif.sh" {
		t.Errorf("go-test Convert = %q, want %q", b.Convert, "scripts/test-to-sarif.sh")
	}
	if !b.CrashGuard {
		t.Errorf("go-test CrashGuard = false, want true")
	}
	if b.GateType != engine.GateTypeTest {
		t.Errorf("go-test GateType = %v, want test", b.GateType)
	}
	if !b.PackageScoped {
		t.Errorf("go-test PackageScoped = false, want true")
	}
	if b.Provision != nil {
		t.Errorf("go-test Provision = %+v, want nil (assume-present toolchain)", b.Provision)
	}
}

// TestGoToolchain_GolangciBindingFromPackData proves the golangci binding carries its
// record from pack data (golangci-lint run command, config-file input, project-wide
// scope + target, strict SARIF, mechanism category, nil provision).
func TestGoToolchain_GolangciBindingFromPackData(t *testing.T) {
	m := goToolchainManifest(t)
	spec, ok := m.Engines["golangci"]
	if !ok {
		t.Fatalf("go-toolchain must declare golangci engine, got %v", engineKeysOf(m.Engines))
	}
	b := spec.Binding

	if !containsAll(b.Command, "golangci-lint", "run") {
		t.Errorf("golangci Command must run `golangci-lint run`, got %q", b.Command)
	}
	if b.InputMode != engine.InputModeConfigFile {
		t.Errorf("golangci InputMode = %q, want config-file", b.InputMode)
	}
	if b.ScopeKind != engine.ScopeKindProjectWide {
		t.Errorf("golangci ScopeKind = %v, want project-wide", b.ScopeKind)
	}
	if b.ProjectTarget != "./..." {
		t.Errorf("golangci ProjectTarget = %q, want %q", b.ProjectTarget, "./...")
	}
	if !b.StrictSarif {
		t.Errorf("golangci StrictSarif = false, want true")
	}
	if b.Category != engine.EngineCategoryMechanism {
		t.Errorf("golangci Category = %v, want mechanism", b.Category)
	}
	if b.Provision != nil {
		t.Errorf("golangci Provision = %+v, want nil (assume-present toolchain)", b.Provision)
	}
}

// goToolchainBinding resolves one engine's DECLARED binding record from the real
// go-toolchain pack manifest, fail-louding if the pack does not declare it.
func goToolchainBinding(t *testing.T, m *pack.Manifest, name string) engine.EngineBinding {
	t.Helper()
	spec, ok := m.Engines[name]
	if !ok {
		t.Fatalf("go-toolchain must declare engine %q, got %v", name, engineKeysOf(m.Engines))
	}
	return spec.Binding
}

// TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind proves the
// EngineBinding's exempt_from_scope_filter property is DECOUPLED from ScopeKind,
// asserted against the DECLARED binding records of the real go-toolchain pack (not a
// struct literal): go-build AND go-test declare it true while golangci(lint) declares
// it false/unset — even though all three share ScopeKindProjectWide. golangci is the
// non-exempt engine that carries the decoupling proof: same ScopeKind, divergent
// exempt values ⇒ ScopeKind does not drive the exempt decision (SPEC-041 CLM-011).
func TestExemption_BindingDeclaresExemptFromScopeFilterDecoupledFromScopeKind(t *testing.T) {
	// Assert against the go-toolchain pack's DECLARED bindings, parsed from its
	// manifest — the exempt/scope-kind values are pack DATA (pkg/pack), read into
	// engine.EngineBinding records.
	m, err := pack.ParseManifestFile(filepath.Join(goToolchainPackRoot(t), "pack.yml"))
	if err != nil {
		t.Fatalf("go-toolchain pack must parse: %v", err)
	}
	build := goToolchainBinding(t, m, "go-build")
	lint := goToolchainBinding(t, m, "golangci")
	test := goToolchainBinding(t, m, "go-test")

	// The declared binding records carry the exempt property per SPEC-041 REQ-004.
	if !build.ExemptFromScopeFilter {
		t.Errorf("go-build declared binding must set exempt_from_scope_filter:true, got false")
	}
	if lint.ExemptFromScopeFilter {
		t.Errorf("golangci declared binding must leave exempt_from_scope_filter false/unset, got true")
	}
	if !test.ExemptFromScopeFilter {
		t.Errorf("go-test declared binding must set exempt_from_scope_filter:true (ISSUE-129), got false")
	}

	// Decoupling is observable ONLY when the exempt values diverge while ScopeKind
	// is held constant across all three bindings.
	for _, tc := range []struct {
		name string
		b    engine.EngineBinding
	}{{"go-build", build}, {"golangci", lint}, {"go-test", test}} {
		if tc.b.ScopeKind != engine.ScopeKindProjectWide {
			t.Fatalf("%s ScopeKind = %v, want ScopeKindProjectWide (constant across all three so the decoupling is observable)", tc.name, tc.b.ScopeKind)
		}
	}
	if build.ExemptFromScopeFilter == lint.ExemptFromScopeFilter {
		t.Errorf("exempt decision must be DECOUPLED from ScopeKind: go-build(%v) and golangci(%v) share ScopeKindProjectWide yet must diverge on exempt", build.ExemptFromScopeFilter, lint.ExemptFromScopeFilter)
	}
	if test.ExemptFromScopeFilter == lint.ExemptFromScopeFilter {
		t.Errorf("exempt decision must be DECOUPLED from ScopeKind: go-test(%v) and golangci(%v) share ScopeKindProjectWide yet must diverge on exempt", test.ExemptFromScopeFilter, lint.ExemptFromScopeFilter)
	}
}

// TestExemption_ScopeKindDecoupledFromExemptDecision proves ScopeKind stays
// arg-shaping-only and is NOT consulted for the exempt/ProjectWide decision. All three
// declared go-toolchain bindings (go-build, golangci, go-test) remain
// ScopeKindProjectWide AND each still declares its `./...` ProjectTarget (the
// arg-shaping role ScopeKind actually plays) — yet go-build and go-test are
// exempt_from_scope_filter while golangci is NOT. A single ScopeKind value maps to
// BOTH exempt and non-exempt bindings, so ScopeKind cannot be the input to the exempt
// decision; golangci is the project-wide-scope engine that keeps that proof
// observable after go-test flipped for ISSUE-129 (SPEC-041 CLM-017).
func TestExemption_ScopeKindDecoupledFromExemptDecision(t *testing.T) {
	// Assert against the go-toolchain pack's DECLARED bindings, parsed from its
	// manifest — ScopeKind/ProjectTarget/exempt are pack DATA (pkg/pack).
	m, err := pack.ParseManifestFile(filepath.Join(goToolchainPackRoot(t), "pack.yml"))
	if err != nil {
		t.Fatalf("go-toolchain pack must parse: %v", err)
	}
	build := goToolchainBinding(t, m, "go-build")
	lint := goToolchainBinding(t, m, "golangci")
	test := goToolchainBinding(t, m, "go-test")

	// ScopeKind's real job — arg-shaping — is intact for all three: project-wide with
	// a `./...` target each appends for itself.
	for _, tc := range []struct {
		name string
		b    engine.EngineBinding
	}{{"go-build", build}, {"golangci", lint}, {"go-test", test}} {
		if tc.b.ScopeKind != engine.ScopeKindProjectWide {
			t.Errorf("%s ScopeKind = %v, want ScopeKindProjectWide", tc.name, tc.b.ScopeKind)
		}
		if tc.b.ProjectTarget != "./..." {
			t.Errorf("%s ProjectTarget = %q, want %q (ScopeKind's arg-shaping role)", tc.name, tc.b.ProjectTarget, "./...")
		}
	}

	// go-build and go-test are exempt; golangci is not. Since all three share
	// ScopeKindProjectWide, the exempt decision demonstrably does NOT read ScopeKind:
	// identical ScopeKind, divergent exempt. If ScopeKind drove exemption, golangci
	// (also ProjectWide) would be exempt too.
	if !build.ExemptFromScopeFilter {
		t.Errorf("go-build must be exempt_from_scope_filter, got false")
	}
	if !test.ExemptFromScopeFilter {
		t.Errorf("go-test must be exempt_from_scope_filter (ISSUE-129), got false")
	}
	if lint.ExemptFromScopeFilter {
		t.Errorf("ScopeKind must NOT be consulted for the exempt decision: golangci(exempt=%v) shares go-build's and go-test's ScopeKindProjectWide yet must stay non-exempt — it is what keeps the decoupling observable", lint.ExemptFromScopeFilter)
	}
}
