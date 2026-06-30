package gate

import (
	"os"
	"strings"
	"testing"

	"github.com/bmanson/backstop-core/pkg/config"
)

// TestPolarity_StackLabelNoLongerReadsConfigLanguage (CLM-021): the stackLabel
// helper is GONE and PolarityStepResult renders the stack from the new
// CapabilityState.Stack carrier — pkg/gate no longer reads cfg.Language. A source
// guard asserts both the function and the language read are absent; a behavioral
// check proves the rendered stack comes from cap.Stack (the cfg carries no language
// at all, yet the message names the carried stack).
func TestPolarity_StackLabelNoLongerReadsConfigLanguage(t *testing.T) {
	src, err := os.ReadFile("traceability_polarity.go")
	if err != nil {
		t.Fatalf("reading traceability_polarity.go: %v", err)
	}
	if strings.Contains(string(src), "func stackLabel") {
		t.Error("stackLabel must be DELETED from pkg/gate — the cfg.Language-derived label is rehomed onto CapabilityState.Stack (CLM-021)")
	}
	if strings.Contains(string(src), "cfg.Language") {
		t.Error("pkg/gate must no longer read cfg.Language — the stack label is carried on CapabilityState.Stack (CLM-021)")
	}

	// The rendered stack label comes from cap.Stack, not from any language field:
	// the cfg carries no language, yet the fail-loud message names the carried stack.
	cap := CapabilityState{Present: false, PackOrCommand: "a coverage pack", Stack: "go"}
	res := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassCapabilityAbsent, &config.Config{Project: "p"}, cap)
	if len(res.Violations) == 0 {
		t.Fatal("class 2 must carry an advisory violation")
	}
	if !strings.Contains(res.Violations[0].Message, "go") {
		t.Errorf("the rendered stack label must come from CapabilityState.Stack (%q); got message: %s", cap.Stack, res.Violations[0].Message)
	}

	// An empty CapabilityState.Stack renders the "unspecified" hygiene fallback.
	resEmpty := PolarityStepResult(StepCoverageThreshold, DimensionCoverage, ClassCapabilityAbsent, &config.Config{Project: "p"}, CapabilityState{Present: false})
	if len(resEmpty.Violations) == 0 || !strings.Contains(resEmpty.Violations[0].Message, "unspecified") {
		t.Errorf("an empty stack must render \"unspecified\"; got: %v", resEmpty.Violations)
	}
}

// TestPolarity_ClassificationVerdictUnaffectedByLanguageRemoval (CLM-020): the
// capability/polarity classification VERDICTS (ClassNone / ClassCapabilityAbsent /
// ClassBrokenDeclared / ClassDeclaredIntentUnmet) are derived from the declaration
// surface + CapabilityState ALONE — no language field is read — so retiring
// `language:` changes no verdict. Each class is produced from a config carrying NO
// language field at all.
func TestPolarity_ClassificationVerdictUnaffectedByLanguageRemoval(t *testing.T) {
	undeclared := &config.Config{Project: "p"}
	declared := &config.Config{Project: "p", Enforcement: config.Enforcement{
		Toolchain: map[string]config.ToolchainPass{
			"test": {Command: "c", Format: "f", GateType: string(DimensionCoverage)},
		},
	}}

	cases := []struct {
		name string
		cfg  *config.Config
		cap  CapabilityState
		want PolarityClass
	}{
		{"undeclared+present -> none", undeclared, CapabilityState{Present: true, Working: true}, ClassNone},
		{"undeclared+absent -> capability-absent", undeclared, CapabilityState{Present: false}, ClassCapabilityAbsent},
		{"declared+absent -> declared-intent-unmet", declared, CapabilityState{Present: false}, ClassDeclaredIntentUnmet},
		{"declared+present+!working -> broken-declared", declared, CapabilityState{Present: true, Working: false}, ClassBrokenDeclared},
		{"declared+present+working -> none", declared, CapabilityState{Present: true, Working: true}, ClassNone},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ClassifyDimension(tc.cfg, DimensionCoverage, tc.cap); got != tc.want {
				t.Errorf("ClassifyDimension = %v, want %v — verdicts must be language-independent (CLM-020)", got, tc.want)
			}
		})
	}
}
