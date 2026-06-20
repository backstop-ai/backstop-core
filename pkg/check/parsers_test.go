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

// TestCodeCheck_Parsers_RegexLinesFormat pins the regex-lines configurable
// parser (CLM-005): a named-group pattern (file/line/col/message) against the
// sample yields a violation with captured file/line/message and severity
// defaulting to "error" for matching lines, and nothing for non-matching lines.
func TestCodeCheck_Parsers_RegexLinesFormat(t *testing.T) {
	parser, err := lookupParser("regex-lines")
	if err != nil {
		t.Fatalf("lookupParser(regex-lines): %v", err)
	}

	violations, parseErr := parser([]byte(regexLinesSampleTxt), CheckTypeBuild)
	if parseErr != nil {
		t.Fatalf("regex-lines parse: %v", parseErr)
	}
	// Two matching lines; the middle "no position" line must NOT match.
	if len(violations) != 2 {
		t.Fatalf("got %d violations, want 2 (non-matching line must be skipped)", len(violations))
	}

	first := violations[0]
	if first.File != "src/lib.rs" {
		t.Errorf("v[0].File = %q, want src/lib.rs", first.File)
	}
	if first.Line != 10 {
		t.Errorf("v[0].Line = %d, want 10", first.Line)
	}
	if first.Message != "borrow of moved value" {
		t.Errorf("v[0].Message = %q, want 'borrow of moved value'", first.Message)
	}
	if first.Severity != "error" {
		t.Errorf("v[0].Severity = %q, want error (default)", first.Severity)
	}
	if first.Pass != CheckTypeBuild {
		t.Errorf("v[0].Pass = %v, want build (target check type)", first.Pass)
	}

	second := violations[1]
	if second.File != "src/main.rs" || second.Line != 3 || second.Message != "unused import" {
		t.Errorf("v[1] = %q/%d/%q, want src/main.rs/3/'unused import'", second.File, second.Line, second.Message)
	}
}

// TestCodeCheck_Parsers_FormatRegistryResolution asserts the named-format
// registry resolves each documented format string to a parser, and that an
// unknown format name fails loud with an error (feeds the config-error path in
// Phase 4). CLM-005.
func TestCodeCheck_Parsers_FormatRegistryResolution(t *testing.T) {
	// After the SPEC-034 cutover the bespoke go-build/go-test/golangci-json named
	// formats were removed with their parsers — the Go toolchain output is
	// normalized to SARIF by the go-toolchain pack convert scripts and parsed via
	// "sarif". Only the surviving generic formats must resolve here.
	known := []string{
		"eslint-json",
		"tsc",
		"sarif",
		"regex-lines",
	}
	for _, name := range known {
		parser, err := lookupParser(name)
		if err != nil {
			t.Errorf("lookupParser(%q) returned error %v, want a parser", name, err)
			continue
		}
		if parser == nil {
			t.Errorf("lookupParser(%q) returned nil parser", name)
		}
	}

	if _, err := lookupParser("does-not-exist"); err == nil {
		t.Error("lookupParser(unknown) returned nil error; want a fail-loud error for an unknown format name")
	}

	// The retired bespoke Go formats must now be unknown — resolving them is a
	// fail-loud config error, not a silent wrap of a deleted parser.
	for _, retired := range []string{"golangci-json", "go-build", "go-test"} {
		if _, err := lookupParser(retired); err == nil {
			t.Errorf("lookupParser(%q) must fail loud after the cutover; the bespoke format was removed", retired)
		}
	}
}
