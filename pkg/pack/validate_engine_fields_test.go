package pack_test

import (
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/pack"
)

// makeEngineManifest returns a minimal valid enforcement manifest with a single
// rule whose engine is the given value and all other engine-specific fields
// cleared, so a test can populate exactly the fields under test.
func makeEngineManifest(engine string) *pack.Manifest {
	m := makeMinimalManifest()
	r := &m.Content.Ruleset.Rules[0]
	r.Engine = engine
	r.RulePath = ""
	r.Standard = ""
	r.Category = ""
	r.InputScope = ""
	r.Validator = ""
	r.Justification = ""
	return m
}

// hasFieldEngineError reports whether any validation error names both the given
// field substring and the given engine in its Field/Message, so the matrix
// tests assert the error is attributed to the offending field and engine
// (REQ-003) rather than merely "some error occurred".
func hasFieldEngineError(errs []pack.ValidationError, field, engine string) bool {
	for _, e := range errs {
		joined := e.Field + " " + e.Message
		if strings.Contains(joined, field) && strings.Contains(joined, engine) {
			return true
		}
	}
	return false
}

func anyError(errs []pack.ValidationError) bool { return len(errs) > 0 }

// --- semgrep field-contract: requires rule_path+standard, forbids category/input_scope/validator ---

func TestEngineFit_SemgrepValidFields(t *testing.T) {
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	m.Content.Ruleset.Rules[0].Standard = "No eval"
	errs := pack.ValidateManifest(m)
	if hasFieldEngineError(errs, "rule_path", "semgrep") || hasFieldEngineError(errs, "standard", "semgrep") {
		t.Fatalf("valid semgrep rule must pass field-contract, got %#v", errs)
	}
}

func TestEngineFit_SemgrepMissingRulePathFails(t *testing.T) {
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].Standard = "No eval"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "rule_path", "semgrep") {
		t.Fatalf("semgrep missing rule_path must fail naming field+engine, got %#v", errs)
	}
}

func TestEngineFit_SemgrepForbidsInputScope(t *testing.T) {
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	m.Content.Ruleset.Rules[0].Standard = "No eval"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "input_scope", "semgrep") {
		t.Fatalf("semgrep must forbid input_scope, got %#v", errs)
	}
}

func TestEngineFit_SemgrepForbidsCategory(t *testing.T) {
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	m.Content.Ruleset.Rules[0].Standard = "No eval"
	m.Content.Ruleset.Rules[0].Category = "structural"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "category", "semgrep") {
		t.Fatalf("semgrep must forbid category, got %#v", errs)
	}
}

func TestEngineFit_SemgrepForbidsValidator(t *testing.T) {
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	m.Content.Ruleset.Rules[0].Standard = "No eval"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "validator", "semgrep") {
		t.Fatalf("semgrep must forbid validator, got %#v", errs)
	}
}

// --- ast-grep field-contract: requires rule_path, forbids category/input_scope/validator, no standard required ---

func TestEngineFit_AstGrepValidFields(t *testing.T) {
	m := makeEngineManifest("ast-grep")
	m.Content.Ruleset.Rules[0].RulePath = "ast-grep/demo.yml"
	errs := pack.ValidateManifest(m)
	if hasFieldEngineError(errs, "rule_path", "ast-grep") || hasFieldEngineError(errs, "standard", "ast-grep") {
		t.Fatalf("valid ast-grep rule (no standard required) must pass, got %#v", errs)
	}
}

func TestEngineFit_AstGrepMissingRulePathFails(t *testing.T) {
	m := makeEngineManifest("ast-grep")
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "rule_path", "ast-grep") {
		t.Fatalf("ast-grep missing rule_path must fail, got %#v", errs)
	}
}

func TestEngineFit_AstGrepForbidsInputScope(t *testing.T) {
	m := makeEngineManifest("ast-grep")
	m.Content.Ruleset.Rules[0].RulePath = "ast-grep/demo.yml"
	m.Content.Ruleset.Rules[0].InputScope = "multi-file"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "input_scope", "ast-grep") {
		t.Fatalf("ast-grep must forbid input_scope, got %#v", errs)
	}
}

func TestEngineFit_AstGrepForbidsCategory(t *testing.T) {
	m := makeEngineManifest("ast-grep")
	m.Content.Ruleset.Rules[0].RulePath = "ast-grep/demo.yml"
	m.Content.Ruleset.Rules[0].Category = "structural"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "category", "ast-grep") {
		t.Fatalf("ast-grep must forbid category, got %#v", errs)
	}
}

func TestEngineFit_AstGrepForbidsValidator(t *testing.T) {
	m := makeEngineManifest("ast-grep")
	m.Content.Ruleset.Rules[0].RulePath = "ast-grep/demo.yml"
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "validator", "ast-grep") {
		t.Fatalf("ast-grep must forbid validator, got %#v", errs)
	}
}

// --- sandbox field-contract: requires validator/input_scope/category, forbids rule_path; value enums ---

func makeSandboxManifest() *pack.Manifest {
	m := makeEngineManifest("sandbox")
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	m.Content.Ruleset.Rules[0].Category = "structural"
	return m
}

func TestEngineFit_SandboxValidFields(t *testing.T) {
	errs := pack.ValidateManifest(makeSandboxManifest())
	if hasFieldEngineError(errs, "validator", "sandbox") || hasFieldEngineError(errs, "category", "sandbox") || hasFieldEngineError(errs, "input_scope", "sandbox") {
		t.Fatalf("valid sandbox rule must pass field-contract, got %#v", errs)
	}
}

func TestEngineFit_SandboxMissingInputScopeFails(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].InputScope = ""
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "input_scope", "sandbox") {
		t.Fatalf("sandbox missing input_scope must fail, got %#v", errs)
	}
}

func TestEngineFit_SandboxForbidsRulePath(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "rule_path", "sandbox") {
		t.Fatalf("sandbox must forbid rule_path, got %#v", errs)
	}
}

func TestEngineFit_SandboxMissingValidatorFails(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].Validator = ""
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "validator", "sandbox") {
		t.Fatalf("sandbox missing validator must fail, got %#v", errs)
	}
}

func TestEngineFit_SandboxMissingCategoryFails(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].Category = ""
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "category", "sandbox") {
		t.Fatalf("sandbox missing category must fail, got %#v", errs)
	}
}

func TestEngineFit_SandboxCategoryEnumEnforced(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].Category = "business"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "category", "sandbox") {
		t.Fatalf("sandbox out-of-enum category must fail, got %#v", errs)
	}
}

func TestEngineFit_SandboxOtherCategoryRequiresJustification(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].Category = "other"
	m.Content.Ruleset.Rules[0].Justification = "  "
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "justification", "sandbox") {
		t.Fatalf("sandbox category other with empty justification must fail, got %#v", errs)
	}
}

func TestEngineFit_SandboxInputScopeEnumEnforced(t *testing.T) {
	m := makeSandboxManifest()
	m.Content.Ruleset.Rules[0].InputScope = "repo"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "input_scope", "sandbox") {
		t.Fatalf("sandbox out-of-enum input_scope must fail, got %#v", errs)
	}
}

// --- config-file field-contract: forbids rule_path/category/input_scope/validator, requires nothing ---

func TestEngineFit_ConfigFileForbidsRulePath(t *testing.T) {
	m := makeEngineManifest("config-file")
	m.Content.Ruleset.Rules[0].RulePath = "rules/demo.yml"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "rule_path", "config-file") {
		t.Fatalf("config-file must forbid rule_path, got %#v", errs)
	}
}

func TestEngineFit_ConfigFileForbidsCategory(t *testing.T) {
	m := makeEngineManifest("config-file")
	m.Content.Ruleset.Rules[0].Category = "structural"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "category", "config-file") {
		t.Fatalf("config-file must forbid category, got %#v", errs)
	}
}

func TestEngineFit_ConfigFileForbidsInputScope(t *testing.T) {
	m := makeEngineManifest("config-file")
	m.Content.Ruleset.Rules[0].InputScope = "single-file"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "input_scope", "config-file") {
		t.Fatalf("config-file must forbid input_scope, got %#v", errs)
	}
}

func TestEngineFit_ConfigFileForbidsValidator(t *testing.T) {
	m := makeEngineManifest("config-file")
	m.Content.Ruleset.Rules[0].Validator = "validators/demo.sh"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "validator", "config-file") {
		t.Fatalf("config-file must forbid validator, got %#v", errs)
	}
}

// --- verify-not-guide (REQ-004) ---

func TestEngineFit_VerifiesDoesNotGuide(t *testing.T) {
	// A rule whose populated fields satisfy its DECLARED engine passes, even
	// though its rule_path content might "look like" another engine's input.
	// Validation must accept the author's declared engine without inspecting
	// content.
	m := makeEngineManifest("ast-grep")
	m.Content.Ruleset.Rules[0].RulePath = "ast-grep/looks-like-semgrep.yml"
	errs := pack.ValidateManifest(m)
	if anyError(filterEngineErrors(errs)) {
		t.Fatalf("engine-fit must accept the declared engine without inspecting content, got %#v", errs)
	}
}

func TestEngineFit_NeverReclassifies(t *testing.T) {
	// A semgrep rule missing rule_path fails AS A SEMGREP rule (field-contract
	// violation) — it is never silently reclassified onto an engine whose
	// contract it would satisfy (e.g. config-file, which requires nothing).
	m := makeEngineManifest("semgrep")
	m.Content.Ruleset.Rules[0].Standard = "x"
	errs := pack.ValidateManifest(m)
	if !hasFieldEngineError(errs, "rule_path", "semgrep") {
		t.Fatalf("a semgrep rule missing rule_path must fail as semgrep, not be reclassified, got %#v", errs)
	}
	// And it must still report the rule's engine as semgrep in the error.
	if m.Content.Ruleset.Rules[0].Engine != "semgrep" {
		t.Fatalf("validation must not mutate the declared engine")
	}
}

// filterEngineErrors keeps only field-contract errors (those whose Rule code is
// in the engine field-contract family), dropping unrelated structural errors
// (fixtures, pairs_with) so verify-not-guide assertions are precise.
func filterEngineErrors(errs []pack.ValidationError) []pack.ValidationError {
	var out []pack.ValidationError
	for _, e := range errs {
		if strings.Contains(e.Message, "ast-grep") || strings.Contains(e.Message, "semgrep") ||
			strings.Contains(e.Message, "sandbox") || strings.Contains(e.Message, "config-file") {
			out = append(out, e)
		}
	}
	return out
}
