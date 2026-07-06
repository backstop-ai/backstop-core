package main

import (
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
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

	if b.Command != "go test" {
		t.Errorf("go-test Command = %q, want %q", b.Command, "go test")
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
