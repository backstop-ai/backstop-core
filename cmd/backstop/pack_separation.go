package main

import (
	"fmt"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// SPEC-034 REQ-007 / Sharp Edge 9 — the mechanism vs opinion pack boundary.
//
// The Go native toolchain decomposes into two SEPARATE packs:
//   - MECHANISM (backstop/go-toolchain): run the Go toolchain (build/test/lint)
//     and normalize its output to SARIF — identical for every Go project. Its
//     rules bind the toolchain engines below.
//   - OPINION (backstop/go-standards): swappable coding-standards rules. Its
//     rules bind the standards engines below and carry a `standard:` declaration.
//
// The boundary is enforced by classifying each rule's declared engine. A rule
// that binds a toolchain engine is mechanism; a rule that binds a standards
// engine is opinion. A pack must not mix the two: build/test/lint mechanism must
// not bleed into the standards pack, and coding-standards opinion must not bleed
// into the toolchain pack.

// engineCategory resolves an engine name to its EngineCategory by looking up the
// engine's binding in the shared registry (ISSUE-015). The registry is the SINGLE
// source of truth for the mechanism/opinion classification: there is no second
// hardcoded engine set to drift. An unknown/unregistered engine resolves to
// EngineCategoryUnset (Lookup fails loud, but for classification that simply means
// "neither mechanism nor opinion" — identical to the pre-ISSUE-015 switches
// returning false for both, so a mis-bound rule trips no boundary check here and
// is caught instead by the gate's fail-loud Lookup at dispatch).
func engineCategory(engineName string) engine.EngineCategory {
	bind, err := resolveEngineRegistry().Lookup(engineName)
	if err != nil {
		return engine.EngineCategoryUnset
	}
	return bind.Category
}

// isToolchainMechanismEngine reports whether engine is a Go native-toolchain
// mechanism engine (build/test/lint run + normalize). The classification is read
// from the engine's EngineBinding.Category in the registry (ISSUE-015), not a
// hardcoded engine set: a rule binding a mechanism engine is toolchain mechanism,
// not coding-standards opinion.
func isToolchainMechanismEngine(engineName string) bool {
	return engineCategory(engineName) == engine.EngineCategoryMechanism
}

// isStandardsOpinionEngine reports whether engine is a coding-standards opinion
// engine (rule-fed semgrep/ast-grep): a rule binding one of these is opinion, not
// toolchain mechanism. The classification is read from the engine's
// EngineBinding.Category in the registry (ISSUE-015).
func isStandardsOpinionEngine(engineName string) bool {
	return engineCategory(engineName) == engine.EngineCategoryOpinion
}

// packClassification is the mechanism/opinion character a pack's rules carry.
type packClassification struct {
	HasMechanism bool // at least one rule binds a toolchain mechanism engine
	HasOpinion   bool // at least one rule binds a standards opinion engine
}

// classifyPack reports whether a manifest carries toolchain mechanism rules,
// coding-standards opinion rules, or (in the violation case) both.
func classifyPack(m *pack.Manifest) packClassification {
	var c packClassification
	if m == nil {
		return c
	}
	for _, r := range m.Content.Ruleset.Rules {
		if isToolchainMechanismEngine(r.Engine) {
			c.HasMechanism = true
		}
		if isStandardsOpinionEngine(r.Engine) {
			c.HasOpinion = true
		}
	}
	return c
}

// packSeparationViolations returns the mechanism/opinion boundary violations for a
// manifest (REQ-007 / CLM-023/CLM-024). A pack must be EITHER mechanism OR opinion,
// never both: a single pack mixing toolchain mechanism and coding-standards opinion
// is rejected. Additionally, a rule in a mechanism-bearing pack must not carry a
// `standard:` declaration (the opinion marker), and a rule in an opinion-bearing
// pack must not bind a toolchain mechanism engine. An empty slice means the pack
// respects the boundary.
func packSeparationViolations(m *pack.Manifest) []string {
	if m == nil {
		return nil
	}
	c := classifyPack(m)
	var violations []string

	// A pack must not carry BOTH mechanism and opinion rules.
	if c.HasMechanism && c.HasOpinion {
		violations = append(violations, fmt.Sprintf(
			"pack %q mixes toolchain mechanism and coding-standards opinion in one pack; they must be separate artifacts (REQ-007)", m.Name))
	}

	// Per-rule bleed checks. The both-mechanism-and-opinion case is already
	// reported above (a rule binding a standards engine in a mechanism pack makes
	// HasOpinion true, tripping that top-level violation). The remaining bleed a
	// single rule can introduce WITHOUT flipping the pack's classification is a
	// `standard:` opinion marker on a rule in a purely-mechanism pack.
	for _, r := range m.Content.Ruleset.Rules {
		mechanism := isToolchainMechanismEngine(r.Engine)

		// In a mechanism-bearing pack, no rule may carry the `standard:` opinion
		// marker (a mechanism rule normalizes tool output; it has no opinion text).
		if c.HasMechanism && !c.HasOpinion && r.Standard != "" {
			violations = append(violations, fmt.Sprintf(
				"rule %q in mechanism pack %q carries a `standard:` declaration (opinion bleed): %q", r.ID, m.Name, r.Standard))
		}

		// In an opinion-bearing pack, no rule may bind a toolchain mechanism engine.
		if c.HasOpinion && !c.HasMechanism && mechanism {
			violations = append(violations, fmt.Sprintf(
				"rule %q in opinion pack %q binds toolchain mechanism engine %q (mechanism bleed)", r.ID, m.Name, r.Engine))
		}
	}
	return violations
}
