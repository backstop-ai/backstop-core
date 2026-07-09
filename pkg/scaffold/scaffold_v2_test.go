package scaffold

import (
	"strings"
	"testing"
)

// TestArtifactNew_Bundle_StampsV2 (ISSUE-032 Defect E / CLM-009): a scaffolded numbered
// BUNDLE-NNN bundle stamps schema_version bundle/v2 — the only version whose
// filename_pattern accepts the numbered BUNDLE-NNN- prefix. bundle/v1 rejects it, so a
// freshly scaffolded bundle would be invalid against the schema its filename requires.
func TestArtifactNew_Bundle_StampsV2(t *testing.T) {
	content, err := Scaffold("bundle", "001", "my-bundle", "2026-07-08", "")
	if err != nil {
		t.Fatalf("Scaffold(bundle): %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "schema_version: bundle/v2") {
		t.Errorf("bundle scaffold must stamp schema_version bundle/v2, got:\n%s", s)
	}
	if strings.Contains(s, "schema_version: bundle/v1") {
		t.Error("bundle scaffold must NOT stamp the retired bundle/v1")
	}
	// The numbered id must be present so the filename_pattern coupling is real.
	if !strings.Contains(s, "number: BUNDLE-001") {
		t.Errorf("bundle scaffold should carry number BUNDLE-001, got:\n%s", s)
	}
}
