package pack

import (
	"strconv"
	"strings"

	"github.com/bmanson/backstop-core/pkg/pack/engine"
)

// ValidationError describes a manifest validation violation.
type ValidationError struct {
	Field   string
	Message string
	Rule    string
}

// ValidateManifest validates manifest constraints and returns all violations.
func ValidateManifest(m *Manifest) []ValidationError {
	if m == nil {
		return []ValidationError{{
			Field:   "manifest",
			Message: "manifest is required",
			Rule:    "CLM-001",
		}}
	}

	var errs []ValidationError
	errs = append(errs, validateContentTypes(m)...)
	errs = append(errs, validateEngineFields(m)...)
	errs = append(errs, validateSecurityFixtures(m)...)
	errs = append(errs, validateToolConfigTrace(m)...)
	errs = append(errs, validateCoOccurrence(m)...)
	errs = append(errs, validateFixtureProof(m)...)
	errs = append(errs, validateFixtureDirNaming(m)...)
	return errs
}

// ExpectedLayout returns the expected pack layout.
func ExpectedLayout(m *Manifest) []string {
	seen := map[string]struct{}{}
	layout := make([]string, 0, 6)
	add := func(path string) {
		if _, ok := seen[path]; ok {
			return
		}
		seen[path] = struct{}{}
		layout = append(layout, path)
	}

	add("pack.yml")
	add("go.mod")
	add("fixtures/rules/")

	hasRuleFiles := false
	hasValidators := false
	if m != nil {
		registry := resolveEngineRegistryForValidation(m)
		for _, rule := range m.Content.Ruleset.Rules {
			// Derive the rules/ vs validators/ layout from the rule's RESOLVED
			// binding InputMode — NOT a "semgrep"/"ast-grep" engine-name switch
			// (SPEC-035 REQ-006c/CLM-025). A rule-fed input mode (rule-flags /
			// rule-dir / pattern-arg) ships rule files under rules/; an
			// input_mode none engine (the sandbox shape) ships validators under
			// validators/. An unknown engine is caught by validateEngineFields.
			binding, ok := registry[rule.Engine]
			if !ok {
				continue
			}
			switch binding.InputMode {
			case engine.InputModeRuleFlags, engine.InputModeRuleDir, engine.InputModePatternArg:
				hasRuleFiles = true
			case engine.InputModeNone:
				// input_mode none is BOTH the sandbox shape (ships validators) and
				// the project-wide native toolchain (go build/test, ships nothing).
				// ScopeKind separates them: the sandbox runs per-scoped-file
				// (file-args) and ships validators/; a project-wide toolchain engine
				// shapes its own target and ships no per-rule asset directory.
				if binding.ScopeKind == engine.ScopeKindFileArgs {
					hasValidators = true
				}
			}
		}
		if m.Archetype == "code" {
			add("scaffolds/")
		}
	}
	if hasRuleFiles {
		add("rules/")
	}
	if hasValidators {
		add("validators/")
	}
	return layout
}

func validateContentTypes(m *Manifest) []ValidationError {
	var errs []ValidationError

	if m.Archetype == "enforcement" && len(m.Content.Scaffolds) > 0 {
		errs = append(errs, ValidationError{
			Field:   "content.scaffolds",
			Message: "enforcement packs must not define scaffolds",
			Rule:    "CLM-002",
		})
	}
	if m.Archetype == "enforcement" && m.Content.SDK != nil {
		errs = append(errs, ValidationError{
			Field:   "content.sdk",
			Message: "enforcement packs must not define sdk",
			Rule:    "CLM-003",
		})
	}

	if len(m.Content.Ruleset.Rules) == 0 && len(m.Content.Scaffolds) == 0 && m.Content.SDK == nil {
		errs = append(errs, ValidationError{
			Field:   "content",
			Message: "unknown or empty content type",
			Rule:    "CLM-005",
		})
	}

	return errs
}

// engineFieldClaim maps an (engine, field, kind) check to the pack validation
// claim code it must report. kind is "requires" (field missing) or "forbids"
// (field present). The codes are re-keyed faithfully from the retired
// retired per-layer field checks so existing claim-traceability is preserved
// across the layer->engine cutover (REQ-003 / CLM-016).
var engineFieldClaim = map[string]string{
	// semgrep (ex-layer-2)
	"semgrep|rule_path|requires":  "CLM-007",
	"semgrep|standard|requires":   "CLM-008",
	"semgrep|category|forbids":    "CLM-018",
	"semgrep|input_scope|forbids": "CLM-025",
	"semgrep|validator|forbids":   "CLM-027",
	// ast-grep (rule-fed like semgrep, no standard requirement)
	"ast-grep|rule_path|requires":  "CLM-007",
	"ast-grep|category|forbids":    "CLM-018",
	"ast-grep|input_scope|forbids": "CLM-025",
	"ast-grep|validator|forbids":   "CLM-027",
	// sandbox (ex-layer-3)
	"sandbox|validator|requires":   "CLM-022",
	"sandbox|input_scope|requires": "CLM-021",
	"sandbox|category|requires":    "CLM-015",
	"sandbox|rule_path|forbids":    "CLM-010",
	// config-file (ex-layer-1 native linter)
	"config-file|rule_path|forbids":   "CLM-009",
	"config-file|category|forbids":    "CLM-017",
	"config-file|input_scope|forbids": "CLM-024",
	"config-file|validator|forbids":   "CLM-026",
}

func claimFor(engineName, field, kind string) string {
	if code, ok := engineFieldClaim[engineName+"|"+field+"|"+kind]; ok {
		return code
	}
	return "CLM-003-engine-fit"
}

// validateEngineFields verifies each rule's populated fields satisfy its
// declared engine's requires/forbids field-contract (REQ-003), emitting a
// ValidationError naming the offending field AND engine. It is verify-only: it
// never inspects rule content, recommends an engine, or reclassifies a rule
// (REQ-004). It replaces the retired per-layer field validation, re-keyed
// engine-for-layer with every per-layer forbid preserved (REQ-003 / CLM-016).
func validateEngineFields(m *Manifest) []ValidationError {
	var errs []ValidationError
	registry := resolveEngineRegistryForValidation(m)
	for i, rule := range m.Content.Ruleset.Rules {
		fieldPrefix := "content.ruleset.rules[" + strconv.Itoa(i) + "]"
		// An empty engine is caught at parse time; in direct ValidateManifest
		// calls on hand-built structs, surface it here too so validation is loud.
		if rule.Engine == "" {
			errs = append(errs, ValidationError{
				Field:   fieldPrefix + ".engine",
				Message: "engine is required (layer is retired)",
				Rule:    "CLM-005-engine-required",
			})
			continue
		}
		binding, ok := registry[rule.Engine]
		if !ok {
			errs = append(errs, ValidationError{
				Field:   fieldPrefix + ".engine",
				Message: "unknown engine " + rule.Engine,
				Rule:    "CLM-020-unknown-engine",
			})
			continue
		}

		// Verify the rule's populated fields against the engine's DECLARED
		// FieldContract on the binding (SPEC-035 REQ-003/CLM-036), NOT a name-keyed
		// map. A pack-declared engine supplies its contract inline; a built-in
		// whose binding carries no declared contract falls back to the name-keyed
		// DefaultFieldContracts (CLM-037, OQ-1 resolved to option i: the baked
		// default is the OVERRIDABLE fallback a declared contract wins over).
		contract := contractForEngine(rule.Engine, binding)
		for _, field := range contract.Requires {
			if ruleFieldValue(rule, field) == "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + "." + field,
					Message: "engine " + rule.Engine + " requires " + field,
					Rule:    claimFor(rule.Engine, field, "requires"),
				})
			}
		}
		for _, field := range contract.Forbids {
			if ruleFieldValue(rule, field) != "" {
				errs = append(errs, ValidationError{
					Field:   fieldPrefix + "." + field,
					Message: "engine " + rule.Engine + " must not define " + field,
					Rule:    claimFor(rule.Engine, field, "forbids"),
				})
			}
		}

		// The sandbox engine additionally enforces the category value-enum, the
		// other-requires-justification rule, and the input_scope value-enum —
		// re-keyed unchanged from the layer-3 checks.
		if rule.Engine == "sandbox" {
			errs = append(errs, validateSandboxValueRules(rule, fieldPrefix)...)
		}
	}

	return errs
}

// resolveEngineRegistryForValidation projects the manifest's pack-declared
// engines: block over the fallback DefaultRegistry, with a pack-declared binding
// OVERRIDING a same-named built-in (the deterministic merge, SPEC-035 CLM-004).
// The validator's layout-derivation and field-contract checks both resolve a
// rule's binding through this union so a pack-declared engine is a first-class
// citizen, not a name the validator fails to recognize.
func resolveEngineRegistryForValidation(m *Manifest) map[string]engine.EngineBinding {
	registry := make(map[string]engine.EngineBinding)
	for name, binding := range engine.DefaultRegistry() {
		registry[name] = binding
	}
	for name, spec := range m.Engines {
		registry[name] = spec.Binding
	}
	return registry
}

// contractForEngine returns the engine's field-contract to validate against. A
// pack-declared engine's DECLARED FieldContract on the binding wins (CLM-036); a
// built-in whose binding carries no declared contract falls back to the
// name-keyed DefaultFieldContracts default. This is the field-contract half of the
// OQ-1 disposition (SPEC-035 CLM-037, resolved to option i): the baked default is
// the OVERRIDABLE fallback a same-named pack-declared contract wins over, mirroring
// how resolveEngineRegistry lets a pack-declared binding override the built-in.
func contractForEngine(name string, binding engine.EngineBinding) engine.FieldContract {
	if len(binding.FieldContract.Requires) > 0 || len(binding.FieldContract.Forbids) > 0 {
		return binding.FieldContract
	}
	return engine.DefaultFieldContracts()[name]
}

// validateSandboxValueRules applies the sandbox engine's value-enum and
// justification checks (category in {presence,structural,other}; other requires
// justification; input_scope in {single-file,multi-file}) re-keyed from layer-3.
func validateSandboxValueRules(rule Rule, fieldPrefix string) []ValidationError {
	var errs []ValidationError
	if rule.Category != "" && !isValidSandboxCategory(rule.Category) {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".category",
			Message: "engine sandbox category must be presence, structural, or other",
			Rule:    "CLM-016",
		})
	}
	if rule.Category == "other" && strings.TrimSpace(rule.Justification) == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".justification",
			Message: "engine sandbox category other requires justification",
			Rule:    "CLM-014",
		})
	}
	if rule.InputScope != "" && rule.InputScope != "single-file" && rule.InputScope != "multi-file" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix + ".input_scope",
			Message: "engine sandbox input_scope must be single-file or multi-file",
			Rule:    "CLM-023",
		})
	}
	if rule.InputScope == "" || rule.Validator == "" {
		errs = append(errs, ValidationError{
			Field:   fieldPrefix,
			Message: "engine sandbox requires isolation fields",
			Rule:    "CLM-040",
		})
	}
	return errs
}

// ruleFieldValue returns the populated value of the named pack.yml field on a
// rule, so the field-contract loop can check presence generically.
func ruleFieldValue(rule Rule, field string) string {
	switch field {
	case engine.FieldRulePath:
		return rule.RulePath
	case engine.FieldStandard:
		return rule.Standard
	case engine.FieldCategory:
		return rule.Category
	case engine.FieldInputScope:
		return rule.InputScope
	case engine.FieldValidator:
		return rule.Validator
	default:
		return ""
	}
}

func isValidSandboxCategory(category string) bool {
	switch category {
	case "presence", "structural", "other":
		return true
	default:
		return false
	}
}

func validateSecurityFixtures(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		if rule.RiskClass != "security" {
			continue
		}
		if hasBypassFixture(rule.Claims) {
			continue
		}
		errs = append(errs, ValidationError{
			Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims.fixtures",
			Message: "security rules require at least one bypass fixture",
			Rule:    "CLM-029",
		})
	}
	return errs
}

func hasBypassFixture(claims []Claim) bool {
	for _, claim := range claims {
		for _, fixture := range claim.Fixtures.Positive {
			if fixture.BypassAttempt {
				return true
			}
		}
		for _, fixture := range claim.Fixtures.Negative {
			if fixture.BypassAttempt {
				return true
			}
		}
	}
	return false
}

func validateToolConfigTrace(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, tc := range m.ToolConfig {
		if tc.ID == "" && tc.RequiredBy == "" {
			errs = append(errs, ValidationError{
				Field:   "tool_config[" + strconv.Itoa(i) + "]",
				Message: "tool_config requires id or required_by",
				Rule:    "CLM-033",
			})
		}
	}
	return errs
}

func validateCoOccurrence(m *Manifest) []ValidationError {
	var errs []ValidationError

	if m.Archetype == "enforcement" && len(m.Content.Scaffolds) > 0 {
		errs = append(errs, ValidationError{
			Field:   "content.scaffolds",
			Message: "enforcement packs must not include scaffolds",
			Rule:    "CLM-047",
		})
	}
	if m.Archetype == "enforcement" && m.Content.SDK != nil {
		errs = append(errs, ValidationError{
			Field:   "content.sdk",
			Message: "enforcement packs must not include sdk",
			Rule:    "CLM-048",
		})
	}

	if m.Archetype != "code" {
		return errs
	}

	ruleSet := map[string]struct{}{}
	for _, rule := range m.Content.Ruleset.Rules {
		ruleSet[rule.ID] = struct{}{}
	}
	scaffoldSet := map[string]struct{}{}
	for _, scaffold := range m.Content.Scaffolds {
		scaffoldSet[scaffold.ID] = struct{}{}
	}

	for i, scaffold := range m.Content.Scaffolds {
		if len(scaffold.PairsWith.Rules) == 0 {
			errs = append(errs, ValidationError{
				Field:   "content.scaffolds[" + strconv.Itoa(i) + "].pairs_with.rules",
				Message: "scaffold must pair with at least one rule",
				Rule:    "CLM-045",
			})
			continue
		}
		for _, ruleID := range scaffold.PairsWith.Rules {
			if _, ok := ruleSet[ruleID]; !ok {
				errs = append(errs, ValidationError{
					Field:   "content.scaffolds[" + strconv.Itoa(i) + "].pairs_with.rules",
					Message: "scaffold references missing rule",
					Rule:    "CLM-045",
				})
				break
			}
		}
	}

	for i, rule := range m.Content.Ruleset.Rules {
		if len(rule.PairsWith.Scaffolds) == 0 && rule.PairsWith.SDK == "" {
			errs = append(errs, ValidationError{
				Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].pairs_with",
				Message: "rule must pair with at least one scaffold or sdk",
				Rule:    "CLM-046",
			})
			continue
		}
		for _, scaffoldID := range rule.PairsWith.Scaffolds {
			if _, ok := scaffoldSet[scaffoldID]; !ok {
				errs = append(errs, ValidationError{
					Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].pairs_with.scaffolds",
					Message: "rule references missing scaffold",
					Rule:    "CLM-046",
				})
				break
			}
		}
	}

	return errs
}

func validateFixtureProof(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		if len(rule.Claims) == 0 {
			errs = append(errs, ValidationError{
				Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims",
				Message: "rule requires at least one claim",
				Rule:    "CLM-051",
			})
			continue
		}
		for j, claim := range rule.Claims {
			if len(claim.Fixtures.Positive) == 0 || len(claim.Fixtures.Negative) == 0 {
				errs = append(errs, ValidationError{
					Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures",
					Message: "claim requires positive and negative fixtures",
					Rule:    "CLM-049",
				})
			}
		}
	}
	return errs
}

func validateFixtureDirNaming(m *Manifest) []ValidationError {
	var errs []ValidationError
	for i, rule := range m.Content.Ruleset.Rules {
		want := "fixtures/rules/" + strings.ToLower(rule.ID) + "/"
		for j, claim := range rule.Claims {
			for k, fixture := range claim.Fixtures.Positive {
				if !strings.HasPrefix(fixture.Path, want) {
					errs = append(errs, ValidationError{
						Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures.positive[" + strconv.Itoa(k) + "]",
						Message: "fixture path must use lowercase rule id directory",
						Rule:    "CLM-035",
					})
				}
			}
			for k, fixture := range claim.Fixtures.Negative {
				if !strings.HasPrefix(fixture.Path, want) {
					errs = append(errs, ValidationError{
						Field:   "content.ruleset.rules[" + strconv.Itoa(i) + "].claims[" + strconv.Itoa(j) + "].fixtures.negative[" + strconv.Itoa(k) + "]",
						Message: "fixture path must use lowercase rule id directory",
						Rule:    "CLM-035",
					})
				}
			}
		}
	}
	return errs
}
