package engine

import "fmt"

// GateType is the backstop-owned, tool-NEUTRAL gate-type enum a pack-declared
// EngineBinding fills (REQ-005). It is an int type defined in this leaf package
// (mirroring EngineCategory/ScopeKind) so the binding carries NO pkg/check import
// — the gate-type identifier names a STAGE of the kill chain, never a tool. It
// has exactly seven values: lint, build, test, findings, coverage,
// substantiveness, contracts (CLM-019). A pack declares the value via the
// engines: block's gate_type field; an unrecognized value fails loud through
// ParseGateType (CLM-021).
type GateType int

const (
	// GateTypeLint is the lint stage (e.g. golangci-lint, eslint).
	GateTypeLint GateType = iota // nosemgrep: go.core.no-global-mutable-state — immutable const, not mutable global
	// GateTypeBuild is the build stage (e.g. go build).
	GateTypeBuild
	// GateTypeTest is the test stage (e.g. go test).
	GateTypeTest
	// GateTypeFindings is the rule-fed findings stage (semgrep/ast-grep) — the
	// neutral name pkg/check's CheckTypeFindings aligns to (CLM-022/CLM-032).
	GateTypeFindings
	// GateTypeCoverage is the coverage stage.
	GateTypeCoverage
	// GateTypeSubstantiveness is the substantiveness stage.
	GateTypeSubstantiveness
	// GateTypeContracts is the contracts stage.
	GateTypeContracts
)

// gateTypeStrings maps each GateType to its canonical YAML/string spelling. The
// spellings are the surface the pack engines: block declares (gate_type:
// lint|build|test|findings|coverage|substantiveness|contracts), so String() and
// ParseGateType are exact inverses and stay in lockstep with the fixtures.
var gateTypeStrings = map[GateType]string{ // nosemgrep: go.core.no-global-mutable-state — immutable lookup table, never mutated
	GateTypeLint:            "lint",
	GateTypeBuild:           "build",
	GateTypeTest:            "test",
	GateTypeFindings:        "findings",
	GateTypeCoverage:        "coverage",
	GateTypeSubstantiveness: "substantiveness",
	GateTypeContracts:       "contracts",
}

// String returns the canonical spelling of a GateType (the exact gate_type value
// a pack declares), so a parsed binding round-trips back to its YAML surface.
func (g GateType) String() string {
	if s, ok := gateTypeStrings[g]; ok {
		return s
	}
	return fmt.Sprintf("GateType(%d)", int(g))
}

// ParseGateType resolves a raw gate_type string to a GateType, fail-louding on an
// unrecognized value (REQ-005/CLM-021) — parallel to ParseInputMode, never a
// silent default. The seven accepted spellings are the neutral kill-chain stages.
func ParseGateType(s string) (GateType, error) {
	for gt, spelling := range gateTypeStrings {
		if spelling == s {
			return gt, nil
		}
	}
	return 0, fmt.Errorf("unknown gate_type %q: must be one of lint, build, test, findings, coverage, substantiveness, contracts", s)
}
