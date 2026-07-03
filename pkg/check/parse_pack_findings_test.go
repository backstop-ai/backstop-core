package check

import (
	"os"
	"path/filepath"
	"testing"
)

// ParsePackFindings is the SARIF entry the go-toolchain engine path (convert
// scripts + golangci v2 native SARIF) parses its normalized output through. It
// STAYS after the SPEC-034 cutover (the bespoke Go parsers it once sat beside
// were deleted), so it must remain genuinely tested in-package with hardcoded
// expected findings — not a bespoke comparison.

// readGoToolchainFixture reads a shared SPEC-034 captured-output fixture from the
// go-toolchain testdata.
func readGoToolchainFixture(t *testing.T, name string) []byte {
	t.Helper()
	p := filepath.Join("..", "..", "cmd", "backstop", "testdata", "go-toolchain", "fixtures", name)
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("read fixture %s: %v", name, err)
	}
	return b
}

// TestParsePackFindings_GolangciV2Sarif asserts ParsePackFindings normalizes the
// golangci v2 native SARIF fixture into the located findings and SARIF
// level->severity mapping (error/warning), plus the empty (clean) and malformed
// (fail-loud) branches.
func TestParsePackFindings_GolangciV2Sarif(t *testing.T) {
	vs, err := ParsePackFindings(readGoToolchainFixture(t, "golangci-v2.sarif"))
	if err != nil {
		t.Fatalf("ParsePackFindings: %v", err)
	}
	if len(vs) != 2 {
		t.Fatalf("expected 2 SARIF findings, got %d: %+v", len(vs), vs)
	}
	var gotError, gotWarning bool
	for _, v := range vs {
		switch v.Severity {
		case "error":
			gotError = true
			if v.File != "pkg/widget/widget.go" || v.Line != 14 {
				t.Errorf("error finding not located: %+v", v)
			}
		case "warning":
			gotWarning = true
		}
	}
	if !gotError || !gotWarning {
		t.Errorf("SARIF level must map to error+warning severities, got %+v", vs)
	}

	// Empty input is a clean (no-findings) parse, not an error.
	empty, eerr := ParsePackFindings([]byte("  \n"))
	if eerr != nil {
		t.Fatalf("empty SARIF must parse cleanly, got: %v", eerr)
	}
	if len(empty) != 0 {
		t.Errorf("empty SARIF must yield zero findings, got %+v", empty)
	}

	// Malformed (non-JSON) input must fail loud, not silently read as zero.
	if _, berr := ParsePackFindings([]byte("not json at all")); berr == nil {
		t.Error("malformed SARIF must return a parse error, not a silent zero-findings green")
	}
}

// TestParsePackFindings_SuppressedResultsDropped pins ISSUE-017: a SARIF result
// carrying a non-empty `suppressions` array is INACTIVE (semgrep in --sarif mode
// emits `// nosemgrep`-suppressed findings as suppressed results, not by dropping
// them). ParsePackFindings must NOT count a suppressed result as a violation, or
// an inline-justified finding reads as a false failure. The fixture carries one
// suppressed result and one active result; only the active one may surface.
func TestParsePackFindings_SuppressedResultsDropped(t *testing.T) {
	sarif := []byte(`{
	  "runs": [
	    {
	      "results": [
	        {
	          "ruleId": "go.core.no-global-mutable-state",
	          "level": "error",
	          "message": {"text": "suppressed false positive on a const"},
	          "locations": [
	            {"physicalLocation": {"artifactLocation": {"uri": "pkg/engine/binding.go"}, "region": {"startLine": 73}}}
	          ],
	          "suppressions": [{"kind": "inSource"}]
	        },
	        {
	          "ruleId": "go.core.no-panic",
	          "level": "error",
	          "message": {"text": "real active finding"},
	          "locations": [
	            {"physicalLocation": {"artifactLocation": {"uri": "pkg/widget/widget.go"}, "region": {"startLine": 42}}}
	          ]
	        }
	      ]
	    }
	  ]
	}`)

	vs, err := ParsePackFindings(sarif)
	if err != nil {
		t.Fatalf("ParsePackFindings: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected exactly 1 active violation (the suppressed result must be dropped), got %d: %+v", len(vs), vs)
	}
	v := vs[0]
	if v.Rule != "go.core.no-panic" || v.File != "pkg/widget/widget.go" || v.Line != 42 {
		t.Errorf("the surviving violation must be the ACTIVE finding, got %+v", v)
	}
	for _, got := range vs {
		if got.Rule == "go.core.no-global-mutable-state" {
			t.Errorf("the suppressed result (suppressions:[{kind:inSource}]) must not be emitted as a violation, got %+v", got)
		}
	}
}

// TestParsePackFindings_FingerprintFromSarif asserts the content-based,
// line-INDEPENDENT identity is carried off the SARIF result: partialFingerprints
// (deterministically ordered) when present, else the region snippet text, else
// empty (the coarse message-level fallback). This is what stops multiple same-rule
// findings in one file from collapsing in the baseline.
func TestParsePackFindings_FingerprintFromSarif(t *testing.T) {
	const withFP = `{"version":"2.1.0","runs":[{"results":[
		{"ruleId":"R","level":"error","message":{"text":"M"},
		 "partialFingerprints":{"b":"222","a":"111"},
		 "locations":[{"physicalLocation":{"artifactLocation":{"uri":"f.go"},"region":{"startLine":10,"snippet":{"text":"ignored := os.Remove(p)"}}}}]}
	]}]}`
	vs, err := ParsePackFindings([]byte(withFP))
	if err != nil {
		t.Fatalf("ParsePackFindings: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(vs))
	}
	// partialFingerprints win over snippet, ordered by key for determinism.
	if got := vs[0].Fingerprint; got != "a=111;b=222" {
		t.Errorf("fingerprint from partialFingerprints = %q, want %q", got, "a=111;b=222")
	}

	const snippetOnly = `{"version":"2.1.0","runs":[{"results":[
		{"ruleId":"R","level":"error","message":{"text":"M"},
		 "locations":[{"physicalLocation":{"artifactLocation":{"uri":"f.go"},"region":{"startLine":10,"snippet":{"text":"  ignored := os.Remove(p)  "}}}}]}
	]}]}`
	vs2, err := ParsePackFindings([]byte(snippetOnly))
	if err != nil {
		t.Fatalf("ParsePackFindings(snippet): %v", err)
	}
	if got := vs2[0].Fingerprint; got != "ignored := os.Remove(p)" {
		t.Errorf("fingerprint from snippet = %q, want trimmed snippet", got)
	}

	const neither = `{"version":"2.1.0","runs":[{"results":[
		{"ruleId":"R","level":"error","message":{"text":"M"},
		 "locations":[{"physicalLocation":{"artifactLocation":{"uri":"f.go"},"region":{"startLine":10}}}]}
	]}]}`
	vs3, err := ParsePackFindings([]byte(neither))
	if err != nil {
		t.Fatalf("ParsePackFindings(neither): %v", err)
	}
	if got := vs3[0].Fingerprint; got != "" {
		t.Errorf("fingerprint with no partialFingerprints/snippet = %q, want empty", got)
	}
}
