package check

import (
	"testing"
)

// containsCheckType reports whether checks contains target.
func containsCheckType(checks []CheckType, target CheckType) bool {
	for _, ct := range checks {
		if ct == target {
			return true
		}
	}
	return false
}

// TestCodeCheck_Routing_DefaultsWhenNoManifest verifies built-in defaults
// apply when no manifest files exist. (CLM-016)
func TestCodeCheck_Routing_DefaultsWhenNoManifest(t *testing.T) {
	dir := t.TempDir() // empty dir, no manifests

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// Go files should get all 4 passes
	goChecks := m.RouteFile("main.go")
	if len(goChecks) != 4 {
		t.Errorf("Go file: got %d checks, want 4: %v", len(goChecks), goChecks)
	}

	// Non-Go files route to nothing — REQ-007 removed the non-Go catch-all, so a
	// .py file matches no built-in rule and gets the empty slice.
	pyChecks := m.RouteFile("script.py")
	if len(pyChecks) != 0 {
		t.Errorf("Python file: got %v, want EMPTY (no non-Go catch-all)", pyChecks)
	}
}

// TestCodeCheck_Routing_LoadFromNonExistentDir verifies defaults when dir
// does not exist.
func TestCodeCheck_Routing_LoadFromNonExistentDir(t *testing.T) {
	m, err := LoadManifest("/nonexistent/dir/that/does/not/exist")
	if err != nil {
		t.Fatalf("LoadManifest should return defaults for missing dir: %v", err)
	}
	// Should use defaults
	goChecks := m.RouteFile("main.go")
	if len(goChecks) != 4 {
		t.Errorf("Go file: got %d checks, want 4 (defaults)", len(goChecks))
	}
}

// TestCodeCheck_Routing_CheckTypeString verifies CheckType.String.
func TestCodeCheck_Routing_CheckTypeString(t *testing.T) {
	tests := []struct {
		ct   CheckType
		want string
	}{
		{CheckTypeLint, "lint"},
		{CheckTypeBuild, "build"},
		{CheckTypeTest, "test"},
		{CheckTypeFindings, "findings"},
		{CheckType(99), "unknown(99)"},
	}
	for _, tc := range tests {
		if got := tc.ct.String(); got != tc.want {
			t.Errorf("CheckType(%d).String() = %q, want %q", tc.ct, got, tc.want)
		}
	}
}

// TestCodeCheck_Routing_ParseCheckType verifies parseCheckType.
func TestCodeCheck_Routing_ParseCheckType(t *testing.T) {
	tests := []struct {
		input string
		want  CheckType
		ok    bool
	}{
		{"lint", CheckTypeLint, true},
		{"BUILD", CheckTypeBuild, true},
		{"Test", CheckTypeTest, true},
		{"findings", CheckTypeFindings, true},
		{"unknown", 0, false},
	}
	for _, tc := range tests {
		got, ok := parseCheckType(tc.input)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("parseCheckType(%q) = %v,%v, want %v,%v", tc.input, got, ok, tc.want, tc.ok)
		}
	}
}

// TestCodeCheck_Routing_EmptyDir verifies LoadManifest returns defaults
// when the directory exists but has no manifest files.
func TestCodeCheck_Routing_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest on empty dir: %v", err)
	}
	// Should use defaults — Go files get all 4
	checks := m.RouteFile("main.go")
	if len(checks) != 4 {
		t.Errorf("expected 4 checks for .go with defaults, got %d", len(checks))
	}
}

// TestCodeCheck_Routing_NoExtension verifies a file with no extension (non-Go)
// routes to nothing under the default manifest after the non-Go catch-all was
// removed (REQ-007).
func TestCodeCheck_Routing_NoExtension(t *testing.T) {
	dir := t.TempDir() // empty, uses defaults

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	// No extension, non-Go: matches no built-in rule, routes to the empty slice.
	checks := m.RouteFile("Makefile")
	if len(checks) != 0 {
		t.Errorf("Makefile: got %v, want EMPTY (no non-Go catch-all)", checks)
	}
}

// routeContains reports whether checks contains the given check type.
func routeContains(checks []CheckType, want CheckType) bool {
	for _, c := range checks {
		if c == want {
			return true
		}
	}
	return false
}

// TestRouting_DefaultManifestWhenNoStandards verifies that with an empty
// .backstop/rules/ (no .manifest.json), LoadManifest returns the built-in
// default manifest and RouteFile routes a .go file to the default check types.
// (CLM-008)
func TestRouting_DefaultManifestWhenNoStandards(t *testing.T) {
	dir := t.TempDir()

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	checks := m.RouteFile("pkg/server/handler.go")
	want := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	if len(checks) != len(want) {
		t.Fatalf(".go routed to %v, want %v", checks, want)
	}
	for _, ct := range want {
		if !routeContains(checks, ct) {
			t.Errorf(".go route missing %v: got %v", ct, checks)
		}
	}
}

// TestRouting_GoFileUnchangedAfterStandardsRemoval verifies that a .go file
// routes to the same four passes (lint/build/test/findings) via the built-in
// default manifest — the .go route is unchanged and no pass is dropped.
// (CLM-009)
func TestRouting_GoFileUnchangedAfterStandardsRemoval(t *testing.T) {
	dir := t.TempDir()

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	checks := m.RouteFile("cmd/backstop/main.go")
	for _, ct := range []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings} {
		if !routeContains(checks, ct) {
			t.Errorf(".go route dropped %v after standards removal: got %v", ct, checks)
		}
	}
}

// TestRouting_NonGoFileRoutesToSemgrepAfterRemoval verifies that a non-Go file
// routes to nothing via the default manifest after the non-Go catch-all was
// removed (REQ-007): semgrep/findings on arbitrary files is now an opt-in
// declared pack rule, never a baked default. (CLM-022)
func TestRouting_NonGoFileRoutesToSemgrepAfterRemoval(t *testing.T) {
	dir := t.TempDir()

	m, err := LoadManifest(dir)
	if err != nil {
		t.Fatalf("LoadManifest: %v", err)
	}

	for _, f := range []string{"README.md", "config.yml"} {
		checks := m.RouteFile(f)
		if len(checks) != 0 {
			t.Errorf("%s routed to %v, want EMPTY (non-Go catch-all removed)", f, checks)
		}
	}
}

// TestCheckType_SemgrepRenamedToNeutralFindings asserts the pack-findings pass
// tag is the tool-NEUTRAL CheckTypeFindings (SPEC-035 CLM-022). It pins the
// surviving identifier sites — the const decl, the String() case, the
// parseCheckType case, and the default routing slices — onto the neutral name,
// and proves the routing semantics are unchanged by the rename (a .go file still
// routes to the findings pass; an unknown file routes to nothing after REQ-007
// removed the non-Go catch-all).
func TestCheckType_SemgrepRenamedToNeutralFindings(t *testing.T) {
	// Const decl + String() case survive under the neutral name.
	if got := CheckTypeFindings.String(); got != "findings" {
		t.Fatalf("CheckTypeFindings.String() = %q, want \"findings\" (neutral pack-findings tag)", got)
	}
	// parseCheckType case survives under the neutral name.
	ct, ok := parseCheckType("findings")
	if !ok || ct != CheckTypeFindings {
		t.Fatalf("parseCheckType(\"findings\") = %v,%v, want CheckTypeFindings,true", ct, ok)
	}
	// Default routing slice (.go → all four passes) carries the neutral tag — the
	// neutral-findings rename stays pinned on the surviving site.
	m := defaultManifest()
	goRoutes := m.RouteFile("main.go")
	if !containsCheckType(goRoutes, CheckTypeFindings) {
		t.Errorf(".go routing = %v, want to include CheckTypeFindings", goRoutes)
	}
	// Unknown-extension default route is now EMPTY (non-Go catch-all removed).
	other := m.RouteFile("notes.txt")
	if len(other) != 0 {
		t.Errorf("unknown-ext routing = %v, want EMPTY (non-Go catch-all removed)", other)
	}
}

// TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep asserts the gate-type
// STRING surface is tool-neutral (SPEC-035 CLM-032): CheckType.String() returns
// "findings" (never "semgrep"), parseCheckType accepts "findings", and the old
// "semgrep" string no longer maps to this type.
func TestCheckType_StringAndParseUseNeutralFindingsNotSemgrep(t *testing.T) {
	if got := CheckTypeFindings.String(); got == "semgrep" {
		t.Errorf("CheckType.String() = %q, want the neutral \"findings\", never the tool name \"semgrep\"", got)
	}
	if got := CheckTypeFindings.String(); got != "findings" {
		t.Errorf("CheckType.String() = %q, want \"findings\"", got)
	}
	if _, ok := parseCheckType("findings"); !ok {
		t.Error("parseCheckType(\"findings\") did not parse; the neutral gate-type string must be accepted")
	}
	if ct, ok := parseCheckType("semgrep"); ok {
		t.Errorf("parseCheckType(\"semgrep\") = %v,true, want false; the tool name must no longer map to a gate type", ct)
	}
}
