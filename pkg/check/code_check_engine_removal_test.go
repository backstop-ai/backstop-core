package check

import (
	"os"
	"strings"
	"testing"
)

// ISSUE-018 deletion-assertion + live-SARIF-survival tests for the in-process
// check engine removal. The absence assertions pin that the DEAD in-process
// engine machinery (the `backstop code check` engine: Run/RunWith, the Engine
// orchestrator, the toolchain-executor registry, and the baked file-extension
// routing manifest) is GONE from pkg/check production source. The survival
// assertions pin that the LIVE SARIF/coverage surface consumed by the gate's
// dispatchPackEngines path OUTLIVES the deletion — guarding against
// over-deletion. Red while the engine still exists; green after TASK-004/005.

// TestInProcessCheckEngine_Removed proves the DEAD in-process check-engine
// identifiers are absent from every non-test pkg/check source. CLM-002.
func TestInProcessCheckEngine_Removed(t *testing.T) {
	src := readProductionSources(t, ".")
	for _, sym := range []string{
		"func Run(",
		"func RunWith(",
		"type Engine",
		"func (e *Engine) RunPasses",
		"passOrder",
		"buildExecutorsForConfig",
		"resolveToolchain",
		"validateToolchainKeys",
		"DeclaredToolchainExecutorsForTest",
		"type Toolchain",
		"LoadManifest",
		"func (m *Manifest) RouteFile",
		"routeFileDefaults",
		"defaultManifest",
	} {
		if containsCheckSubstr(src, sym) {
			t.Errorf("pkg/check production source still contains dead engine construct %q; it must be deleted", sym)
		}
	}
	// The registry file that housed the toolchain-executor construction is deleted
	// in full.
	if _, err := os.Stat("registry.go"); err == nil {
		t.Error("pkg/check/registry.go still exists; the dead toolchain-executor registry file must be deleted")
	}
}

// TestSARIFSurface_Preserved proves the LIVE pack-findings SARIF surface
// consumed by dispatchPackEngines/substantiveness SURVIVES the engine deletion:
// the parser entry points are still present in source AND a minimal SARIF
// document parses through the live ParsePackFindings stamping CheckTypeFindings.
// CLM-002.
func TestSARIFSurface_Preserved(t *testing.T) {
	src := readProductionSources(t, ".")
	for _, sym := range []string{
		"func ParsePackFindings(",
		"func parseSarif(",
		"func lookupParser(",
		"sarifFingerprint",
		"sarifSeverity",
	} {
		if !containsCheckSubstr(src, sym) {
			t.Errorf("pkg/check production source is missing LIVE SARIF surface %q; it must be preserved (consumed by dispatchPackEngines)", sym)
		}
	}

	sarif := []byte(`{
	  "runs": [
	    {
	      "results": [
	        {
	          "ruleId": "org/pack/no-panic",
	          "level": "error",
	          "message": {"text": "panic is forbidden"},
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
		t.Fatalf("ParsePackFindings on live SARIF: %v", err)
	}
	if len(vs) != 1 {
		t.Fatalf("expected exactly 1 violation from the live SARIF path, got %d", len(vs))
	}
	if vs[0].Pass != CheckTypeFindings {
		t.Errorf("violation Pass = %v, want CheckTypeFindings (the live pack-findings tag)", vs[0].Pass)
	}
	if vs[0].Rule != "org/pack/no-panic" || vs[0].File != "pkg/widget/widget.go" || vs[0].Line != 42 {
		t.Errorf("live SARIF finding not parsed correctly: %+v", vs[0])
	}
}

// TestSharedTypes_Preserved proves the shared types the gate/coverage path
// depends on remain constructible after the engine deletion. CLM-002.
func TestSharedTypes_Preserved(t *testing.T) {
	// Compile-time references: if any of these were deleted this file would not
	// build. The runtime assertions pin their observable contract.
	if (Violation{Pass: CheckTypeFindings, File: "a.go"}).File != "a.go" {
		t.Error("Violation must remain a constructible shared type")
	}
	if CheckTypeFindings.String() != "findings" {
		t.Errorf("CheckType.String() = %q, want \"findings\"", CheckTypeFindings.String())
	}
	var _ CommandRunner
	if (&ConfigError{Message: "x"}).Error() != "x" {
		t.Error("ConfigError must remain a constructible shared error type")
	}
	if (&DegradedError{Message: "y"}).Error() != "y" {
		t.Error("DegradedError must remain a constructible shared error type")
	}
	if (CoverageRecord{Path: "a.go"}).Path != "a.go" {
		t.Error("CoverageRecord must remain a constructible shared type")
	}
}

// containsCheckSubstr reports whether text contains sub as a plain substring. The
// engine-removal scan matches on declaration fragments (e.g. "func RunWith(")
// rather than bare identifiers, so substring matching is the intended semantics.
func containsCheckSubstr(text, sub string) bool {
	return sub != "" && strings.Contains(text, sub)
}
