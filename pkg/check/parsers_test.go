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
