package check

import (
	"testing"
)

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
