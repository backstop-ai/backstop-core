package check

import (
	"os"
	"path/filepath"
	"testing"
)

// readSemgrepFixture reads a captured real-semgrep SARIF fixture from the
// cmd/backstop testdata corpus. ONE captured corpus serves both this parser test
// and the cmd/backstop severity contract test, so the bytes are not duplicated
// into two trees; this mirrors readGoToolchainFixture
// (parse_pack_findings_test.go), which reaches across the same way.
//
// Provenance for every file it reads — tool version, exact command, capture
// date, sha256 — is recorded in that directory's PROVENANCE.md. The bytes
// themselves are unmodified tool output.
func readSemgrepFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "cmd", "backstop", "testdata", "semgrep", "fixtures", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read semgrep fixture %s: %v", name, err)
	}
	return b
}

// TestCodeCheck_Parsers_SarifFormat pins the sarif generic format parser
// (CLM-005): File from locations[0].physicalLocation.artifactLocation.uri, Line
// from region.startLine, Message from message.text, Rule from ruleId, and
// Severity mapped from the SARIF level (error/warning, default error when
// absent). Both the error and warning rows must map correctly.
func TestCodeCheck_Parsers_SarifFormat(t *testing.T) {
	parser, err := lookupParser("sarif")
	if err != nil {
		t.Fatalf("lookupParser(sarif): %v", err)
	}

	violations, parseErr := parser([]byte(sarifSampleJSON), CheckTypeLint)
	if parseErr != nil {
		t.Fatalf("sarif parse: %v", parseErr)
	}
	if len(violations) != 3 {
		t.Fatalf("got %d violations, want 3", len(violations))
	}

	// Row 1: error level.
	v := violations[0]
	if v.File != "src/alpha.rs" {
		t.Errorf("v[0].File = %q, want src/alpha.rs", v.File)
	}
	if v.Line != 42 {
		t.Errorf("v[0].Line = %d, want 42", v.Line)
	}
	if v.Message != "undefined symbol referenced" {
		t.Errorf("v[0].Message = %q", v.Message)
	}
	if v.Rule != "EXAMPLE001" {
		t.Errorf("v[0].Rule = %q, want EXAMPLE001", v.Rule)
	}
	if v.Severity != "error" {
		t.Errorf("v[0].Severity = %q, want error", v.Severity)
	}
	if v.Pass != CheckTypeLint {
		t.Errorf("v[0].Pass = %v, want lint (target check type)", v.Pass)
	}

	// Row 2: warning level.
	w := violations[1]
	if w.File != "src/beta.rs" || w.Line != 7 {
		t.Errorf("v[1] file/line = %q/%d, want src/beta.rs/7", w.File, w.Line)
	}
	if w.Rule != "EXAMPLE002" {
		t.Errorf("v[1].Rule = %q, want EXAMPLE002", w.Rule)
	}
	if w.Severity != "warning" {
		t.Errorf("v[1].Severity = %q, want warning", w.Severity)
	}

	// Row 3: level absent → defaults to error.
	d := violations[2]
	if d.Severity != "error" {
		t.Errorf("v[2].Severity = %q, want error (default when level absent)", d.Severity)
	}
	if d.File != "src/gamma.rs" || d.Line != 3 {
		t.Errorf("v[2] file/line = %q/%d, want src/gamma.rs/3", d.File, d.Line)
	}
}

// TestCodeCheck_Parsers_FormatRegistryResolution asserts the named-format
// registry resolves the single surviving "sarif" format to a parser, and that
// every other format name (including the retired eslint-json/tsc/regex-lines and
// the bespoke go-build/go-test/golangci-json) fails loud with an error. After
// ISSUE-018 deleted the in-process check engine, "sarif" is the only format the
// registry carries. CLM-003/CLM-006.
func TestCodeCheck_Parsers_FormatRegistryResolution(t *testing.T) {
	parser, err := lookupParser("sarif")
	if err != nil {
		t.Errorf("lookupParser(\"sarif\") returned error %v, want a parser", err)
	}
	if parser == nil {
		t.Error("lookupParser(\"sarif\") returned nil parser")
	}

	if _, err := lookupParser("does-not-exist"); err == nil {
		t.Error("lookupParser(unknown) returned nil error; want a fail-loud error for an unknown format name")
	}

	// The parsers deleted with the in-process check engine must now be unknown —
	// resolving them is a fail-loud config error, not a silent wrap of a deleted
	// parser.
	for _, retired := range []string{"eslint-json", "tsc", "regex-lines", "golangci-json", "go-build", "go-test"} {
		if _, err := lookupParser(retired); err == nil {
			t.Errorf("lookupParser(%q) must fail loud; the format was removed with the in-process check engine", retired)
		}
	}
}

// TestParseSarif_CarriesResultProperties pins CLM-001: parseSarif copies a SARIF
// result's string-valued `properties` object verbatim onto
// check.Violation.Properties, preserving values that contain spaces and quotes
// (the structured channel that replaces machine-data parsed out of the message).
func TestParseSarif_CarriesResultProperties(t *testing.T) {
	const in = `{
  "version": "2.1.0",
  "runs": [
    {
      "results": [
        {
          "ruleId": "referenced-symbol-go",
          "level": "error",
          "message": { "text": "test X has no assertions (hollow)" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "a_test.go" },
                "region": { "startLine": 5 }
              }
            }
          ],
          "properties": {
            "func": "surfaces a plan spec_id in the response",
            "symbol": "readmodel"
          }
        }
      ]
    }
  ]
}`
	violations, err := parseSarif([]byte(in), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	v := violations[0]
	if v.Properties == nil {
		t.Fatalf("v.Properties = nil, want the result's properties map")
	}
	if got := v.Properties["func"]; got != "surfaces a plan spec_id in the response" {
		t.Errorf("v.Properties[func] = %q, want the verbatim spaced value", got)
	}
	if got := v.Properties["symbol"]; got != "readmodel" {
		t.Errorf("v.Properties[symbol] = %q, want readmodel", got)
	}
	// Additive: existing fields are unchanged by the new channel.
	if v.Rule != "referenced-symbol-go" || v.File != "a_test.go" || v.Line != 5 {
		t.Errorf("existing fields changed: Rule=%q File=%q Line=%d", v.Rule, v.File, v.Line)
	}
	if v.Message != "test X has no assertions (hollow)" {
		t.Errorf("v.Message = %q, want the human message unchanged", v.Message)
	}
}

// TestParseSarif_NoPropertiesIsEmptyNotError pins CLM-002: a SARIF result with no
// `properties` object yields a nil/empty Properties map and NO error — the change
// is additive and a property-less finding behaves exactly as before.
func TestParseSarif_NoPropertiesIsEmptyNotError(t *testing.T) {
	const in = `{
  "version": "2.1.0",
  "runs": [
    {
      "results": [
        {
          "ruleId": "EXAMPLE001",
          "level": "error",
          "message": { "text": "a finding with no properties" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "b.go" },
                "region": { "startLine": 9 }
              }
            }
          ]
        }
      ]
    }
  ]
}`
	violations, err := parseSarif([]byte(in), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif returned an error for a property-less result: %v", err)
	}
	if len(violations) != 1 {
		t.Fatalf("got %d violations, want 1", len(violations))
	}
	if len(violations[0].Properties) != 0 {
		t.Errorf("v.Properties = %v, want nil/empty for a result with no properties", violations[0].Properties)
	}
}

// TestParseSarif_DescriptorLevelSuppliesSeverityWhenResultLevelAbsent is THE
// FALSIFIER for ISSUE-104, and it runs on CAPTURED REAL SEMGREP BYTES.
//
// Real semgrep does not state severity on the result. It states it on the RULE
// DESCRIPTOR — runs[].tool.driver.rules[].defaultConfiguration.level — joined to
// the result by ruleId, and omits `level` from the result object entirely (a
// missing key, not an empty value). A parser that reads only results[].level
// therefore sees "" for EVERY semgrep finding and, by the correct fail-closed
// default, blocks all of them — whatever severity the pack author declared.
//
// Both directions are asserted so the fix cannot be satisfied by a parser that
// simply stopped blocking: the warning capture must resolve to warning AND the
// error capture must still resolve to error.
func TestParseSarif_DescriptorLevelSuppliesSeverityWhenResultLevelAbsent(t *testing.T) {
	warnings, err := parseSarif(readSemgrepFixture(t, "descriptor-warning.sarif"), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(descriptor-warning.sarif): %v", err)
	}
	if len(warnings) != 1 {
		t.Fatalf("got %d violations from the captured warning fixture, want 1: %+v", len(warnings), warnings)
	}
	if got := warnings[0].Severity; got != "warning" {
		t.Errorf("severity = %q, want \"warning\": real semgrep declares the rule's severity ONLY on "+
			"tool.driver.rules[].defaultConfiguration.level (the result carries no `level` key at all), so a "+
			"parser that reads results[].level alone blocks every declared-WARNING pack rule", got)
	}
	if got := warnings[0].Rule; got != "capture.capture-sample-panic" {
		t.Errorf("rule = %q, want the captured ruleId — the descriptor join keys on it", got)
	}

	errors, err := parseSarif(readSemgrepFixture(t, "descriptor-error.sarif"), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(descriptor-error.sarif): %v", err)
	}
	if len(errors) != 1 {
		t.Fatalf("got %d violations from the captured error fixture, want 1: %+v", len(errors), errors)
	}
	if got := errors[0].Severity; got != "error" {
		t.Errorf("severity = %q, want \"error\": the SAME descriptor path must carry a declared ERROR "+
			"through as blocking, or the fallback has merely disarmed pack enforcement", got)
	}
}

// TestParseSarif_ResultLevelWinsOverDescriptorLevel pins precedence: a level
// stated ON THE RESULT wins, and the descriptor is not consulted for it.
//
// Two halves, because the two producer classes fail differently.
func TestParseSarif_ResultLevelWinsOverDescriptorLevel(t *testing.T) {
	// NON-REGRESSION, ON REAL BYTES ALREADY IN THE TREE. golangci-lint v2 emits
	// native SARIF that states severity on each RESULT under a driver with only a
	// `name` and no `rules` array — the exact producer class the descriptor
	// fallback must leave undisturbed. Adding a fallback must not perturb a log
	// that has nothing to fall back to.
	vs, err := ParsePackFindings(readGoToolchainFixture(t, "golangci-v2.sarif"))
	if err != nil {
		t.Fatalf("ParsePackFindings(golangci-v2.sarif): %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("got %d findings from the golangci capture, want 2: %+v", len(vs), vs)
	}
	var gotError, gotWarning bool
	for _, v := range vs {
		switch v.Severity {
		case "error":
			gotError = true
		case "warning":
			gotWarning = true
		default:
			t.Errorf("unexpected severity %q on %+v", v.Severity, v)
		}
	}
	if !gotError || !gotWarning {
		t.Errorf("the real golangci capture's result-level severities (one error, one warning) must survive "+
			"the descriptor fallback unchanged; got %+v", vs)
	}

	// PRECEDENCE, ON A HAND-BUILT SHAPE — LABELLED AS SUCH.
	//
	// This input is SPEC-DERIVED, not captured, and it is the only fixture in this
	// lane that is. It stands in for SARIF 2.1.0 §3.27.10 (result.level), which
	// makes the result's own level the authoritative statement for that result and
	// the rule descriptor's defaultConfiguration merely the default it overrides.
	// NO PRODUCER ON HAND EMITS THIS SHAPE: semgrep omits the result level
	// entirely, golangci emits no descriptor, and neither contradicts itself. A
	// self-contradicting log therefore cannot be captured from reality — which is
	// exactly why precedence has to be pinned on a constructed one.
	const conflicting = `{"version":"2.1.0","runs":[{
	  "tool":{"driver":{"name":"synthetic","rules":[
	    {"id":"contradicted-rule","defaultConfiguration":{"level":"error"}}
	  ]}},
	  "results":[
	    {"ruleId":"contradicted-rule","level":"warning",
	     "message":{"text":"the result overrides its own rule's default"},
	     "locations":[{"physicalLocation":{"artifactLocation":{"uri":"a.go"},"region":{"startLine":1}}}]}
	  ]}]}`
	conflict, err := parseSarif([]byte(conflicting), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(conflicting): %v", err)
	}
	if len(conflict) != 1 {
		t.Fatalf("got %d violations, want 1", len(conflict))
	}
	if got := conflict[0].Severity; got != "warning" {
		t.Errorf("severity = %q, want \"warning\": the result stated its own level, so the descriptor's "+
			"contradicting default must not be consulted (SARIF §3.27.10)", got)
	}

	// The mirror direction, so the assertion above cannot be satisfied by a parser
	// that simply prefers "warning".
	const conflictingInverse = `{"version":"2.1.0","runs":[{
	  "tool":{"driver":{"name":"synthetic","rules":[
	    {"id":"contradicted-rule","defaultConfiguration":{"level":"warning"}}
	  ]}},
	  "results":[
	    {"ruleId":"contradicted-rule","level":"error",
	     "message":{"text":"the result overrides its own rule's default"},
	     "locations":[{"physicalLocation":{"artifactLocation":{"uri":"a.go"},"region":{"startLine":1}}}]}
	  ]}]}`
	inverse, err := parseSarif([]byte(conflictingInverse), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(conflictingInverse): %v", err)
	}
	if got := inverse[0].Severity; got != "error" {
		t.Errorf("severity = %q, want \"error\": the result's own level wins in BOTH directions, not just "+
			"the permissive one", got)
	}
}

// TestParseSarif_NoLevelAnywhereDefaultsToError is the fail-closed FLOOR, and it
// must remain true unchanged, forever.
//
// When neither the result nor a rule descriptor supplies a level, severity is
// "error". The alternative — reading an undeclared level as non-blocking — would
// let any pack disable enforcement by omitting a field, which is the vacuous
// green this project exists to prevent. sarifSeverity is not modified by
// ISSUE-104; only the set of PLACES a level is looked for changed.
//
// It reuses sarifSampleJSON's EXAMPLE003 — a result with no `level`, in a log
// whose driver has no `rules` array — rather than restating that shape in a
// fourth JSON blob.
func TestParseSarif_NoLevelAnywhereDefaultsToError(t *testing.T) {
	violations, err := parseSarif([]byte(sarifSampleJSON), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(sarifSampleJSON): %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("got %d violations, want 3", len(violations))
	}

	var absent *Violation
	for i := range violations {
		if violations[i].Rule == "EXAMPLE003" {
			absent = &violations[i]
		}
	}
	if absent == nil {
		t.Fatalf("EXAMPLE003 (the level-absent result) missing from %+v", violations)
	}
	if absent.Severity != "error" {
		t.Errorf("severity = %q, want \"error\": no level on the result and no rule descriptor to fall back "+
			"to means nothing was declared, and silence must read as the STRICT answer", absent.Severity)
	}

	// sarifSeverity itself is untouched by the descriptor fallback: everything that
	// is not the literal "warning" still maps to error.
	if got := sarifSeverity(""); got != "error" {
		t.Errorf("sarifSeverity(\"\") = %q, want error", got)
	}
	if got := sarifSeverity("note"); got != "error" {
		t.Errorf("sarifSeverity(\"note\") = %q, want error", got)
	}
	if got := sarifSeverity("warning"); got != "warning" {
		t.Errorf("sarifSeverity(\"warning\") = %q, want warning", got)
	}
}

// TestParseSarif_DescriptorJoinIsPerRunAndByRuleID guards the scoping the fix
// must have: the descriptor map is built ONCE PER RUN, not once per log.
//
// Two runs may legitimately carry different rule sets, and the same rule id may
// be declared at different severities in each. A log-wide map would let run A's
// severity leak onto run B's finding — a silent cross-run contamination that
// produces a plausible-looking wrong answer rather than an error. The same test
// pins the join key: a result whose ruleId appears in NO descriptor falls
// through to the fail-closed default rather than borrowing a neighbour's level.
func TestParseSarif_DescriptorJoinIsPerRunAndByRuleID(t *testing.T) {
	const twoRuns = `{"version":"2.1.0","runs":[
	  {
	    "tool":{"driver":{"name":"run-a","rules":[
	      {"id":"shared-rule","defaultConfiguration":{"level":"warning"}}
	    ]}},
	    "results":[
	      {"ruleId":"shared-rule","message":{"text":"run A finding"},
	       "locations":[{"physicalLocation":{"artifactLocation":{"uri":"a.go"},"region":{"startLine":1}}}]}
	    ]
	  },
	  {
	    "tool":{"driver":{"name":"run-b","rules":[
	      {"id":"shared-rule","defaultConfiguration":{"level":"error"}},
	      {"id":"other-rule","defaultConfiguration":{"level":"warning"}}
	    ]}},
	    "results":[
	      {"ruleId":"shared-rule","message":{"text":"run B finding"},
	       "locations":[{"physicalLocation":{"artifactLocation":{"uri":"b.go"},"region":{"startLine":2}}}]},
	      {"ruleId":"undeclared-rule","message":{"text":"no descriptor anywhere"},
	       "locations":[{"physicalLocation":{"artifactLocation":{"uri":"c.go"},"region":{"startLine":3}}}]}
	    ]
	  }
	]}`

	violations, err := parseSarif([]byte(twoRuns), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif(twoRuns): %v", err)
	}
	if len(violations) != 3 {
		t.Fatalf("got %d violations, want 3: %+v", len(violations), violations)
	}

	byFile := map[string]Violation{}
	for _, v := range violations {
		byFile[v.File] = v
	}

	if got := byFile["a.go"].Severity; got != "warning" {
		t.Errorf("run A finding severity = %q, want \"warning\" from run A's OWN descriptor", got)
	}
	if got := byFile["b.go"].Severity; got != "error" {
		t.Errorf("run B finding severity = %q, want \"error\": the same rule id is declared at a DIFFERENT "+
			"severity in run B, and each result must resolve against its own run's driver. A log-wide "+
			"descriptor map would leak run A's warning onto this finding", got)
	}
	if got := byFile["c.go"].Severity; got != "error" {
		t.Errorf("undeclared-rule severity = %q, want \"error\": its ruleId matches no descriptor in its "+
			"run, so it must fall through to the fail-closed default rather than borrowing a neighbouring "+
			"rule's level", got)
	}
}
