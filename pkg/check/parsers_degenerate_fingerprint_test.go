package check

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// parsers_degenerate_fingerprint_test.go (ISSUE-210) pins the fail-closed
// fingerprint for a SARIF result whose partialFingerprints carry keys but no
// information (gitleaks dir mode emits commitSha/email/author/date/
// commitMessage, all ""). Before the fix every such result got the same
// constant "author=;commitMessage=;commitSha=;date=;email=" fingerprint, so
// every same-rule secret in one file shared one baseline identity. Tests reach
// the fingerprint only through parseSarif, never sarifFingerprint directly.
// All snippet strings are synthetic and non-secret.

// degenerateFPCollapsed is the pre-fix collapsed fingerprint every degenerate
// gitleaks-shaped result used to receive.
const degenerateFPCollapsed = "author=;commitMessage=;commitSha=;date=;email="

// degenerateFPAllEmpty returns the gitleaks dir-mode partialFingerprints
// object: all five keys present, every value empty.
func degenerateFPAllEmpty() map[string]string {
	return map[string]string{
		"commitSha":     "",
		"email":         "",
		"author":        "",
		"date":          "",
		"commitMessage": "",
	}
}

// degenerateFPResult describes one gitleaks-shaped SARIF result. A zero
// column with omitColumn set leaves startColumn out of the region entirely.
type degenerateFPResult struct {
	uri        string
	line       int
	column     int
	omitColumn bool
	snippet    string
	partial    map[string]string
}

// degenerateFPLog builds a gitleaks-shaped SARIF log (rule generic-api-key, no
// result level) holding the given results, marshaled with encoding/json so
// every field is exactly what the test states.
func degenerateFPLog(t *testing.T, results ...degenerateFPResult) []byte {
	t.Helper()
	sarifResults := make([]map[string]any, 0, len(results))
	for _, r := range results {
		region := map[string]any{
			"startLine": r.line,
			"snippet":   map[string]any{"text": r.snippet},
		}
		if !r.omitColumn {
			region["startColumn"] = r.column
		}
		sarifResults = append(sarifResults, map[string]any{
			"message": map[string]any{"text": "generic-api-key has detected secret for file " + r.uri + "."},
			"ruleId":  "generic-api-key",
			"locations": []any{map[string]any{
				"physicalLocation": map[string]any{
					"artifactLocation": map[string]any{"uri": r.uri},
					"region":           region,
				},
			}},
			"partialFingerprints": r.partial,
		})
	}
	log := map[string]any{
		"version": "2.1.0",
		"runs": []any{map[string]any{
			"tool": map[string]any{"driver": map[string]any{
				"name":  "gitleaks",
				"rules": []any{map[string]any{"id": "generic-api-key"}},
			}},
			"results": sarifResults,
		}},
	}
	out, err := json.Marshal(log)
	if err != nil {
		t.Fatalf("marshal degenerate fixture: %v", err)
	}
	return out
}

// degenerateFPParse parses the log and requires exactly want violations.
func degenerateFPParse(t *testing.T, want int, results ...degenerateFPResult) []Violation {
	t.Helper()
	violations, err := parseSarif(degenerateFPLog(t, results...), CheckTypeFindings)
	if err != nil {
		t.Fatalf("parseSarif: %v", err)
	}
	if len(violations) != want {
		t.Fatalf("got %d violations, want %d", len(violations), want)
	}
	return violations
}

// degenerateFPSnippetOracle is the independent oracle for the CLM-002 encoding.
func degenerateFPSnippetOracle(snippet string) string {
	sum := sha256.Sum256([]byte("snippet:" + snippet))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// degenerateFPPositionOracle is the independent oracle for the CLM-003 encoding.
func degenerateFPPositionOracle(line, column int) string {
	sum := sha256.Sum256([]byte("position:" + strconv.Itoa(line) + ":" + strconv.Itoa(column)))
	return "sha256:" + hex.EncodeToString(sum[:])
}

// TestParseSarif_DegenerateFingerprintDistinctSnippetsHashed pins ISSUE-210
// CLM-002 and CLM-004 at the parser: degenerate partialFingerprints with an
// informative snippet hash that snippet ("snippet:" domain), so two same-rule
// secrets in one file get distinct, raw-text-free fingerprints, and the same
// snippet on another line keeps its fingerprint.
func TestParseSarif_DegenerateFingerprintDistinctSnippetsHashed(t *testing.T) {
	const alpha, bravo = "synthetic-snippet-alpha", "synthetic-snippet-bravo"
	violations := degenerateFPParse(t, 3,
		degenerateFPResult{uri: "src/config.go", line: 3, column: 8, snippet: alpha, partial: degenerateFPAllEmpty()},
		degenerateFPResult{uri: "src/config.go", line: 9, column: 8, snippet: bravo, partial: degenerateFPAllEmpty()},
		degenerateFPResult{uri: "src/config.go", line: 40, column: 2, snippet: alpha, partial: degenerateFPAllEmpty()},
	)
	wantSnippets := []string{alpha, bravo, alpha}
	for i, v := range violations {
		want := degenerateFPSnippetOracle(wantSnippets[i])
		if v.Fingerprint != want {
			t.Errorf("violation %d Fingerprint = %q, want %q (sha256 of \"snippet:\"+%q)", i, v.Fingerprint, want, wantSnippets[i])
		}
		if !strings.HasPrefix(v.Fingerprint, "sha256:") {
			t.Errorf("violation %d Fingerprint = %q, want a sha256: prefix", i, v.Fingerprint)
		}
		if v.Fingerprint == degenerateFPCollapsed {
			t.Errorf("violation %d Fingerprint is the collapsed constant %q", i, degenerateFPCollapsed)
		}
		for _, raw := range []string{alpha, bravo} {
			if strings.Contains(v.Fingerprint, raw) {
				t.Errorf("violation %d Fingerprint %q contains raw snippet %q", i, v.Fingerprint, raw)
			}
		}
	}
	if violations[0].Fingerprint == violations[1].Fingerprint {
		t.Errorf("distinct snippets share fingerprint %q: same-rule secrets in one file collapse", violations[0].Fingerprint)
	}
	if violations[0].Fingerprint != violations[2].Fingerprint {
		t.Errorf("same snippet on line 3 and line 40 got %q vs %q, want a line-independent fingerprint", violations[0].Fingerprint, violations[2].Fingerprint)
	}
}

// TestParseSarif_DegenerateFingerprintRedactedUsesPosition pins ISSUE-210
// CLM-003: a degenerate result whose snippet is the redaction marker, empty or
// whitespace-only hashes its position ("position:<line>:<column>"), so
// different lines or columns stay distinct.
func TestParseSarif_DegenerateFingerprintRedactedUsesPosition(t *testing.T) {
	type pos struct{ line, column int }
	rows := []struct {
		name    string
		snippet string
		a, b    pos
	}{
		{name: "redacted different line", snippet: "REDACTED", a: pos{3, 8}, b: pos{9, 8}},
		{name: "redacted same line different column", snippet: "REDACTED", a: pos{3, 8}, b: pos{3, 20}},
		{name: "empty snippet", snippet: "", a: pos{3, 8}, b: pos{9, 8}},
		{name: "whitespace-only snippet", snippet: "   ", a: pos{3, 8}, b: pos{9, 8}},
		// Accepted noisy consequence of the 2026-10-13 fail-closed decision: a
		// redacted secret that moves to another line reads as a NEW finding
		// (never silently grandfathered), because only its position identifies it.
		{name: "redacted moved to next line reads as new", snippet: "REDACTED", a: pos{3, 8}, b: pos{4, 8}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations := degenerateFPParse(t, 2,
				degenerateFPResult{uri: "src/config.go", line: row.a.line, column: row.a.column, snippet: row.snippet, partial: degenerateFPAllEmpty()},
				degenerateFPResult{uri: "src/config.go", line: row.b.line, column: row.b.column, snippet: row.snippet, partial: degenerateFPAllEmpty()},
			)
			for i, p := range []pos{row.a, row.b} {
				v := violations[i]
				want := degenerateFPPositionOracle(p.line, p.column)
				if v.Fingerprint != want {
					t.Errorf("violation %d Fingerprint = %q, want %q (sha256 of \"position:%d:%d\")", i, v.Fingerprint, want, p.line, p.column)
				}
				if strings.Contains(v.Fingerprint, "REDACTED") {
					t.Errorf("violation %d Fingerprint %q contains the redaction marker", i, v.Fingerprint)
				}
			}
			if violations[0].Fingerprint == violations[1].Fingerprint {
				t.Errorf("positions %v and %v share fingerprint %q", row.a, row.b, violations[0].Fingerprint)
			}
		})
	}
	t.Run("absent startColumn decodes as zero", func(t *testing.T) {
		violations := degenerateFPParse(t, 1,
			degenerateFPResult{uri: "src/config.go", line: 3, omitColumn: true, snippet: "REDACTED", partial: degenerateFPAllEmpty()},
		)
		if want := degenerateFPPositionOracle(3, 0); violations[0].Fingerprint != want {
			t.Errorf("Fingerprint = %q, want %q (sha256 of \"position:3:0\")", violations[0].Fingerprint, want)
		}
	})
}

// TestParseSarif_DegenerateFingerprintWhitespaceOnlyValuesDegenerate pins
// ISSUE-210 CLM-002's TrimSpace rule and the one-key case: whitespace-only
// values and a single empty key are degenerate and take the snippet hash.
func TestParseSarif_DegenerateFingerprintWhitespaceOnlyValuesDegenerate(t *testing.T) {
	const alpha = "synthetic-snippet-alpha"
	rows := []struct {
		name    string
		partial map[string]string
	}{
		{name: "whitespace-only values", partial: map[string]string{
			"commitSha": " ", "email": "\t", "author": " ", "date": "\t", "commitMessage": " \t ",
		}},
		{name: "single empty key", partial: map[string]string{"commitSha": ""}},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			violations := degenerateFPParse(t, 1,
				degenerateFPResult{uri: "src/config.go", line: 3, column: 8, snippet: alpha, partial: row.partial},
			)
			got := violations[0].Fingerprint
			if want := degenerateFPSnippetOracle(alpha); got != want {
				t.Errorf("Fingerprint = %q, want %q (degenerate map must take the snippet hash)", got, want)
			}
			if strings.Contains(got, "=") || strings.Contains(got, alpha) {
				t.Errorf("Fingerprint = %q leaks the k=v form or the raw snippet", got)
			}
		})
	}
}

// TestParseSarif_DegenerateFingerprintMixedValuesByteIdentical pins ISSUE-210
// CLM-001 (control): when at least one partialFingerprints value is non-empty
// the fingerprint stays byte-identical to the pre-fix form — sorted keys,
// "k=v" joined by ";", values verbatim (untrimmed), empty entries kept.
func TestParseSarif_DegenerateFingerprintMixedValuesByteIdentical(t *testing.T) {
	violations := degenerateFPParse(t, 1, degenerateFPResult{
		uri: "src/config.go", line: 3, column: 8, snippet: "synthetic-snippet-alpha",
		partial: map[string]string{
			"commitSha": "abc123", "email": "", "author": " spaced ", "date": "", "commitMessage": "",
		},
	})
	const want = "author= spaced ;commitMessage=;commitSha=abc123;date=;email="
	if got := violations[0].Fingerprint; got != want {
		t.Errorf("Fingerprint = %q, want byte-identical %q", got, want)
	}
}
