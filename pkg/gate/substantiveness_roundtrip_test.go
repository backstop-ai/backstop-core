package gate

import (
	"encoding/json"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// substantiveness_roundtrip_test.go is the ISSUE-062 adversarial proof (REQ-006 /
// CLM-010): an enclosing test name containing spaces AND quotes — a real vitest-style
// it()/test() description, not a single-token Go TestXxx — survives INTACT from the
// ast-grep metavariable through the REAL convert (to-sarif.sh) -> check.ParsePackFindings
// -> gate.Violation.Properties -> the substantiveness join to the correct per-test
// hollow/referenced verdict. A status-quo single-token test cannot prove this; the
// deleted whitespace-delimited parsers would have truncated the name at "surfaces".

// astGrepFinding mirrors the shape the substantiveness convert consumes from
// `ast-grep scan --json`: a ruleId, a human message, a file, a 0-indexed range, and the
// matched metavariables (single.FN / single.PKG .text).
func astGrepFinding(ruleID, message, file, fnText, pkgText string) map[string]any {
	single := map[string]any{"FN": map[string]any{"text": fnText}}
	if pkgText != "" {
		single["PKG"] = map[string]any{"text": pkgText}
	}
	return map[string]any{
		"ruleId":        ruleID,
		"severity":      "error",
		"message":       message,
		"file":          file,
		"range":         map[string]any{"start": map[string]any{"line": 11, "column": 0}},
		"metaVariables": map[string]any{"single": single},
	}
}

// TestSubstantiveness_SpacedQuotedNameRoundTrips (CLM-010): a spaced+quoted test name
// round-trips through convert -> parse -> join to the correct per-test verdict.
func TestSubstantiveness_SpacedQuotedNameRoundTrips(t *testing.T) {
	const packName = "backstop/substantiveness"
	const file = "read_model.test.ts"
	// A real vitest description: spaces AND embedded quotes.
	const spacedQuoted = `surfaces a plan "spec_id" in the response`

	// Build an ast-grep JSON payload the way the engine would for this match: a hollow
	// finding and an extraction finding for the SAME spaced+quoted test name.
	payload := []map[string]any{
		astGrepFinding("hollow-test-go", "test function "+spacedQuoted+" has no assertions (hollow)", file, spacedQuoted, ""),
		astGrepFinding("referenced-symbol-go", "test "+spacedQuoted+" references package readmodel", file, spacedQuoted, "readmodel"),
	}
	astJSON, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal ast-grep payload: %v", err)
	}

	// Run the REAL pack convert (to-sarif.sh) — this is the metavariables -> SARIF
	// properties lift under test.
	script := repoFile("packs", "substantiveness", "ast-grep", "to-sarif.sh")
	sarif := runConvertScript(t, script, astJSON)

	// Parse through the REAL check SARIF parser: Properties must survive intact.
	checkViolations, err := check.ParsePackFindings(sarif)
	if err != nil {
		t.Fatalf("check.ParsePackFindings: %v", err)
	}
	if len(checkViolations) != 2 {
		t.Fatalf("expected 2 parsed findings, got %d: %+v", len(checkViolations), checkViolations)
	}

	// Map to gate.Violation exactly as the dispatch does: namespaced rule id + carried
	// Properties. This is the check.Violation -> gate.Violation bridge (REQ-002).
	flat := make([]Violation, 0, len(checkViolations))
	for _, v := range checkViolations {
		flat = append(flat, Violation{
			Rule:       pack.NamespacedRuleID(packName, v.Rule),
			File:       NormalizePath("", v.File),
			Message:    v.Message,
			Severity:   nonEmptySeverity(v.Severity),
			SourcePack: packName,
			Properties: v.Properties,
		})
	}

	// Route by the pack-declared substantiveness_role property the convert stamped
	// (ISSUE-064) — no rule-name routing key.
	hollow, extraction := RouteSubstantivenessFindings(flat)

	mt := MandatedTest{FuncName: spacedQuoted, FilePath: file}

	// The join must find the spaced+quoted test hollow — the whitespace/quotes survived.
	if !IsTestHollow(hollow, mt) {
		t.Errorf("spaced+quoted test name must map to its hollow finding; the name did not survive convert->parse->join intact")
	}

	// And its referenced symbol must join, keyed on the same spaced+quoted name.
	set := ReferencedSetForTest(extraction, mt)
	if !set["readmodel"] {
		t.Errorf("spaced+quoted test name must join to its referenced symbol readmodel; got %+v", set)
	}

	// Adversarial negative control: the truncated-at-first-space name the DELETED parser
	// would have produced must NOT match — proving there is no residual whitespace
	// truncation anywhere on the path.
	truncated := MandatedTest{FuncName: "surfaces", FilePath: file}
	if IsTestHollow(hollow, truncated) {
		t.Errorf(`the truncated name "surfaces" must NOT match — a whitespace-truncation regression is present`)
	}
	if ReferencedSetForTest(extraction, truncated)["readmodel"] {
		t.Errorf(`the truncated name "surfaces" must NOT join to readmodel — whitespace-truncation regression`)
	}
}
