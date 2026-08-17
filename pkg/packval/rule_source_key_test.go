package packval_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
	"github.com/backstop-ai/backstop-core/pkg/packval"
)

// yamlKeys returns the yaml key each field of the struct type decodes from, with the
// ",omitempty"/",inline" modifiers stripped. Reading the TAGS rather than a
// hand-copied constant is the whole point: a hand-copied string would drift in
// exactly the way this guard exists to catch.
func yamlKeys(t reflect.Type) map[string]bool {
	out := map[string]bool{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("yaml")
		if tag == "" || tag == "-" {
			continue
		}
		out[strings.Split(tag, ",")[0]] = true
	}
	return out
}

// TestPackVal_ManifestRuleSourceKeyMatchesGateRuntimeModel (CLM-001, CLM-009) is the
// DRIFT GUARD. packval's authoring-time Rule and pkg/pack's gate-runtime Rule are two
// hand-maintained models of the same pack.yml. ISSUE-092's root cause was that they
// silently disagreed about which yaml key names a rule's source file: the runtime
// model read `rule_path` (which every real pack declares) while packval read `file`
// (which no real pack declares), so every File-guarded dispatch site in phase 3 was
// dead for the entire fleet while reporting pass.
//
// This test fails if EITHER struct ever drops or renames that key again.
func TestPackVal_ManifestRuleSourceKeyMatchesGateRuntimeModel(t *testing.T) {
	const ruleSourceKey = "rule_path"

	runtimeKeys := yamlKeys(reflect.TypeOf(pack.Rule{}))
	if !runtimeKeys[ruleSourceKey] {
		t.Fatalf("gate-runtime pack.Rule no longer declares a %q yaml key; keys present: %v", ruleSourceKey, runtimeKeys)
	}

	authoringKeys := yamlKeys(reflect.TypeOf(packval.Rule{}))
	if !authoringKeys[ruleSourceKey] {
		t.Fatalf("authoring-time packval.Rule does not declare the %q yaml key the gate-runtime model reads "+
			"— this is the ISSUE-092 drift: packval would read a key no real pack.yml writes. keys present: %v",
			ruleSourceKey, authoringKeys)
	}

	// The reconciliation is only meaningful if the value actually flows: a struct tag
	// nothing reads would satisfy the assertions above and change nothing.
	if got := (packval.Rule{RulePath: "rules/r.yml"}).RuleSourcePath(); got != "rules/r.yml" {
		t.Fatalf("packval.Rule.RuleSourcePath() must surface the rule_path value, got %q", got)
	}
}

// TestPackVal_RuleSourcePathPrefersRulePathAndFallsBackToFile (CLM-001) pins the
// SINGLE accessor that decides precedence, so no future caller can re-derive it
// differently. `rule_path` is canonical because it is what the gate-runtime model
// reads and what real packs write; `file` survives only as a back-compat alias.
func TestPackVal_RuleSourcePathPrefersRulePathAndFallsBackToFile(t *testing.T) {
	cases := []struct {
		name string
		rule packval.Rule
		want string
	}{
		{"rule_path alone resolves to rule_path", packval.Rule{RulePath: "rules/canonical.yml"}, "rules/canonical.yml"},
		{"file alone resolves to file", packval.Rule{File: "rules/legacy.yml"}, "rules/legacy.yml"},
		{"both present: the runtime key wins", packval.Rule{RulePath: "rules/canonical.yml", File: "rules/legacy.yml"}, "rules/canonical.yml"},
		{"neither present resolves to empty", packval.Rule{}, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := tc.rule.RuleSourcePath(); got != tc.want {
				t.Fatalf("RuleSourcePath() = %q, want %q", got, tc.want)
			}
		})
	}
}
