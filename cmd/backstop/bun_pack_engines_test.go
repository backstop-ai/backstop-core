package main

import (
	"path/filepath"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// bunPackRoot returns the absolute path to the in-repo tracked bun-toolchain pack
// directory (the durable testdata-fixture copy; the external repo carries the same
// DATA — packs are always external, the .backstop/packs copy is gitignored).
func bunPackRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(repoRoot(t), "cmd", "backstop", "testdata", "bun-toolchain",
		".backstop", "packs", "backstop", "bun-toolchain")
}

// bunToolchainManifest loads the hand-authored bun-toolchain pack manifest, failing
// if it does not parse (its five engines must register through the declared
// substrate). RED before the pack.yml exists.
func bunToolchainManifest(t *testing.T) *pack.Manifest {
	t.Helper()
	m, err := pack.ParseManifestFile(filepath.Join(bunPackRoot(t), "pack.yml"))
	if err != nil {
		t.Fatalf("bun-toolchain pack must parse (its engines must register): %v", err)
	}
	return m
}

// bunEngine returns the named engine spec from the bun pack or fails.
func bunEngine(t *testing.T, m *pack.Manifest, name string) pack.EngineSpec {
	t.Helper()
	spec, ok := m.Engines[name]
	if !ok {
		t.Fatalf("bun pack must declare a %q engine in its engines: block, got %v", name, engineKeysOf(m.Engines))
	}
	return spec
}

// TestBunPack_OxlintEngineDeclaredAsLint proves the bun pack declares an oxlint
// engine routed to the LINT dimension via gate_type lint (CLM-001).
func TestBunPack_OxlintEngineDeclaredAsLint(t *testing.T) {
	m := bunToolchainManifest(t)
	spec := bunEngine(t, m, "oxlint")
	if spec.Binding.GateType != engine.GateTypeLint {
		t.Errorf("oxlint must route to the lint dimension (gate_type lint), got %v", spec.Binding.GateType)
	}
	if tool := firstToken(spec.Binding.Command); tool != "oxlint" {
		t.Errorf("oxlint engine command must invoke the oxlint tool, got command %q (tool %q)", spec.Binding.Command, tool)
	}
}

// TestBunPack_PrettierCheckDeclaredAsLintCategoryFormat proves the bun pack
// declares a `prettier --check` engine routed to the LINT dimension as a
// lint-category SARIF findings engine (format ≈ lint per DD-3) — it carries a
// convert that turns unformatted-file output into lint-category findings, and is
// NOT routed to any new "format" dimension (CLM-002).
func TestBunPack_PrettierCheckDeclaredAsLintCategoryFormat(t *testing.T) {
	m := bunToolchainManifest(t)
	spec := bunEngine(t, m, "prettier")
	if spec.Binding.GateType != engine.GateTypeLint {
		t.Errorf("prettier (format-as-lint, DD-3) must route to the LINT dimension (gate_type lint), got %v", spec.Binding.GateType)
	}
	if firstToken(spec.Binding.Command) != "prettier" {
		t.Errorf("prettier engine command must invoke the prettier tool, got %q", spec.Binding.Command)
	}
	if !containsAll(spec.Binding.Command, "--check") {
		t.Errorf("prettier must run in --check mode (a findings pass, not a writer), got %q", spec.Binding.Command)
	}
	// A lint-category SARIF findings engine: its raw `--check` output (a list of
	// unformatted files) is normalized to lint-category SARIF by the pack convert.
	if spec.Binding.Convert == "" {
		t.Errorf("prettier must declare a convert that normalizes unformatted-file output to lint-category SARIF findings (DD-3)")
	}
}

// TestBunPack_TscTypecheckDeclaredAsBuild proves the bun pack declares a tsc
// typecheck engine routed to the BUILD dimension via gate_type build (CLM-003).
func TestBunPack_TscTypecheckDeclaredAsBuild(t *testing.T) {
	m := bunToolchainManifest(t)
	spec := bunEngine(t, m, "typecheck")
	if spec.Binding.GateType != engine.GateTypeBuild {
		t.Errorf("typecheck must route to the build dimension (gate_type build), got %v", spec.Binding.GateType)
	}
	// The typecheck pass is tsc's no-emit mode; identify it by the --noEmit flag
	// and the build routing (the tool name lives in pack DATA, not a Go literal).
	if !containsAll(spec.Binding.Command, "--noEmit") {
		t.Errorf("typecheck must run tsc as a typecheck-only pass (--noEmit), got %q", spec.Binding.Command)
	}
	if _, ok := engine.TrustedToolAllowlist()[firstToken(spec.Binding.Command)]; !ok {
		t.Errorf("the typecheck engine's tool %q must be on the trusted-tool allowlist", firstToken(spec.Binding.Command))
	}
}

// TestBunPack_BunTestDeclaredAsTest proves the bun pack declares a `bun test`
// engine routed to the TEST dimension via gate_type test (CLM-004).
func TestBunPack_BunTestDeclaredAsTest(t *testing.T) {
	m := bunToolchainManifest(t)
	spec := bunEngine(t, m, "bun-test")
	if spec.Binding.GateType != engine.GateTypeTest {
		t.Errorf("bun test must route to the test dimension (gate_type test), got %v", spec.Binding.GateType)
	}
	if firstToken(spec.Binding.Command) != "bun" || !containsAll(spec.Binding.Command, "test") {
		t.Errorf("bun-test engine command must run `bun test`, got %q", spec.Binding.Command)
	}
}

// TestBunPack_BunCoverageLcovDeclaredAsCoverage proves the bun pack declares a
// `bun test --coverage --coverage-reporter=lcov` engine routed to the COVERAGE
// dimension (gate_type coverage), with a pack-relative convert
// (scripts/coverage-to-records.sh) and an lcov stdout_artifact (CLM-005).
func TestBunPack_BunCoverageLcovDeclaredAsCoverage(t *testing.T) {
	m := bunToolchainManifest(t)
	spec := bunEngine(t, m, "bun-coverage")
	if spec.Binding.GateType != engine.GateTypeCoverage {
		t.Errorf("bun coverage must route to the coverage dimension (gate_type coverage), got %v", spec.Binding.GateType)
	}
	if firstToken(spec.Binding.Command) != "bun" || !containsAll(spec.Binding.Command, "--coverage", "lcov") {
		t.Errorf("bun-coverage command must run `bun test --coverage --coverage-reporter=lcov`, got %q", spec.Binding.Command)
	}
	if spec.Binding.Convert != "scripts/coverage-to-records.sh" {
		t.Errorf("bun-coverage must declare convert scripts/coverage-to-records.sh, got %q", spec.Binding.Convert)
	}
	if spec.Binding.StdoutArtifact != "coverage/lcov.info" {
		t.Errorf("bun-coverage must declare stdout_artifact coverage/lcov.info (lcov lands in a FILE, not stdout), got %q", spec.Binding.StdoutArtifact)
	}
}

// TestBunPack_PrettierIntroducesNoFormatGateDimension is the DD-3 DENYLIST: after
// loading the bun pack the gate's dimension set is UNCHANGED — every declared
// engine routes to one of the existing dimensions {lint, build, test, coverage},
// NO engine declares a "format" gate_type, and prettier rides lint specifically
// (CLM-006).
func TestBunPack_PrettierIntroducesNoFormatGateDimension(t *testing.T) {
	m := bunToolchainManifest(t)
	allowed := map[engine.GateType]bool{
		engine.GateTypeLint:     true,
		engine.GateTypeBuild:    true,
		engine.GateTypeTest:     true,
		engine.GateTypeCoverage: true,
	}
	dims := map[engine.GateType]bool{}
	for name, spec := range m.Engines {
		gt := spec.Binding.GateType
		if !allowed[gt] {
			t.Errorf("engine %q routes to %v — the bun pack must NOT introduce a new gate dimension beyond lint/build/test/coverage (DD-3 denylist)", name, gt)
		}
		if gt.String() == "format" {
			t.Errorf("engine %q declares a \"format\" gate_type; format ≈ lint (DD-3) and must ride the existing lint dimension", name)
		}
		dims[gt] = true
	}
	// The distinct dimension set the bun pack introduces is exactly the four
	// existing ones — prettier folds into lint, adding no fifth dimension.
	if !dims[engine.GateTypeLint] || !dims[engine.GateTypeBuild] || !dims[engine.GateTypeTest] || !dims[engine.GateTypeCoverage] {
		t.Errorf("the bun pack must cover lint/build/test/coverage, got %v", dims)
	}
	if len(dims) != 4 {
		t.Errorf("the bun pack must declare exactly four dimensions (lint/build/test/coverage) with prettier riding lint, got %d distinct: %v", len(dims), dims)
	}
}

// TestBunPack_EngineCommandsRunThroughDeclaredTrustSubstrate proves every bun
// engine's tool (oxlint/bun/tsc/prettier) is on engine.TrustedToolAllowlist() and
// every command is read from the pack's declared engines: block, NOT a
// baked-into-the-binary command string (CLM-007, the thin-executor first
// principle).
func TestBunPack_EngineCommandsRunThroughDeclaredTrustSubstrate(t *testing.T) {
	m := bunToolchainManifest(t)
	allowlist := engine.TrustedToolAllowlist()
	if len(m.Engines) == 0 {
		t.Fatal("the bun pack must declare its engines as DATA (engines: block), not rely on baked binary commands")
	}
	// Derive every dispatched tool from the pack-DECLARED command (never a baked Go
	// literal) and prove it clears the trust floor. This both proves CLM-007 (every
	// command rides the allowlist) AND that TASK-006 added the bun tools — the tool
	// set is read from the pack, not asserted against a hardcoded list.
	tools := map[string]bool{}
	for name, spec := range m.Engines {
		cmd := spec.Binding.Command
		if cmd == "" {
			t.Errorf("engine %q has an empty command — every command must be pack-DECLARED DATA", name)
			continue
		}
		tool := firstToken(cmd)
		tools[tool] = true
		if _, ok := allowlist[tool]; !ok {
			t.Errorf("engine %q dispatches tool %q which is NOT allowlisted — the command must clear the declared-engine trust floor", name, tool)
		}
	}
	// The five engines collapse to exactly four distinct tools (bun runs both test
	// and coverage), each allowlisted — the bun toolchain is fully trust-gated.
	if len(tools) != 4 {
		t.Errorf("expected the bun pack's five engines to dispatch four distinct allowlisted tools (oxlint/prettier/bun/tsc), got %d: %v", len(tools), tools)
	}
}

// firstToken returns the first whitespace-delimited token of a command string (the
// tool name), using the shared splitCommand splitter so the test reads the tool the
// dispatch would.
func firstToken(command string) string {
	name, _ := splitCommand(command)
	return name
}
