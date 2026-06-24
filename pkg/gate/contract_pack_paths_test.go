package gate

import (
	"os"
	"path/filepath"
	"testing"
)

// contract_pack_paths_test.go (SPEC-038): unit coverage for the contracts pack-path
// resolution and verdict-scope helpers, including the unresolvable-pack branch that
// keeps an uninstalled workspace from passing vacuously.

// TestContractsPackResolvable_InstalledTestdataAndAbsent exercises all three branches:
// installed local pack, testdata fallback, and unresolvable.
func TestContractsPackResolvable_InstalledTestdataAndAbsent(t *testing.T) {
	// (a) The module root resolves via the testdata traceability-pack fallback.
	root := eqRepoRoot(t)
	if !ContractsPackResolvable(root) {
		t.Error("module root must resolve the contracts pack via the testdata fallback")
	}

	// (b) An installed local pack at <root>/.backstop/packs/backstop/contracts resolves.
	ws := t.TempDir()
	installed := filepath.Join(ws, ".backstop", "packs", "backstop", "contracts", "scripts")
	if err := os.MkdirAll(installed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installed, "compile-signature.sh"), []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !ContractsPackResolvable(ws) {
		t.Error("an installed local contracts pack must resolve")
	}
	paths, ok := resolveContractPackPaths(ws)
	if !ok || paths.compileSig == "" {
		t.Error("resolveContractPackPaths must prefer the installed pack scripts")
	}

	// (c) An empty workspace with neither installed nor testdata is unresolvable.
	empty := t.TempDir()
	if ContractsPackResolvable(empty) {
		t.Error("an empty workspace must be unresolvable (no vacuous green)")
	}
}

// TestPackContractResult_UnresolvablePackYieldsUnscanned: when the pack is not
// resolvable, PackContractResult returns an unscanned result (no probe run) so an
// absence entry becomes a loud config error upstream and a present entry no-matches.
func TestPackContractResult_UnresolvablePackYieldsUnscanned(t *testing.T) {
	empty := t.TempDir()
	r, err := PackContractResult(empty, ContractEntry{File: "x.go", Name: "F", Signature: "func F()"})
	if err != nil {
		t.Fatalf("PackContractResult over an unresolvable pack must not error: %v", err)
	}
	if r.Scanned || r.Matched {
		t.Errorf("unresolvable pack must yield Scanned=false Matched=false, got %+v", r)
	}
}

// TestContractScope_FallsBackToFile: contractScope returns Scope when set, else File.
func TestContractScope_FallsBackToFile(t *testing.T) {
	if got := contractScope(ContractEntry{Scope: "s", File: "f"}); got != "s" {
		t.Errorf("contractScope with Scope set = %q, want s", got)
	}
	if got := contractScope(ContractEntry{File: "f"}); got != "f" {
		t.Errorf("contractScope falls back to File = %q, want f", got)
	}
}

// TestFirstNonEmpty covers both branches.
func TestFirstNonEmpty(t *testing.T) {
	if firstNonEmpty("a", "b") != "a" {
		t.Error("firstNonEmpty must return the first non-empty")
	}
	if firstNonEmpty("", "b") != "b" {
		t.Error("firstNonEmpty must fall back to the second")
	}
}

// TestAnalyzerVerdict_AllFixtureBranches covers every captured-verdict branch including
// the paramname-variant and absence-clean cases.
func TestAnalyzerVerdict_AllFixtureBranches(t *testing.T) {
	root := eqRepoRoot(t)
	td := func(n string) string { return filepath.Join(root, "pkg", "gate", "testdata", n) }
	cases := []struct {
		entry ContractEntry
		want  bool
	}{
		{ContractEntry{File: td("contract-sig-present.go"), Name: "RouteFile"}, false},
		{ContractEntry{File: td("contract-sig-paramname-variant.go"), Name: "RouteFile"}, false},
		{ContractEntry{File: td("contract-sig-mismatch.go"), Name: "RouteFile"}, true},
		{ContractEntry{File: td("contract-absence-present.go"), Name: "legacyProbeSymbol", Absent: true, Scope: td("contract-absence-present.go")}, true},
		{ContractEntry{File: td("contract-absence-clean.go"), Name: "legacyProbeSymbol", Absent: true, Scope: td("contract-absence-clean.go")}, false},
		{ContractEntry{File: td("missing.go"), Name: "x", Absent: true, Scope: td("missing.go")}, true},
		{ContractEntry{File: td("unknown.go"), Name: "x"}, false},
	}
	for _, c := range cases {
		if got := AnalyzerVerdict(c.entry); got != c.want {
			t.Errorf("AnalyzerVerdict(%s) = %v, want %v", filepath.Base(c.entry.File), got, c.want)
		}
	}
}

// TestLocationSuffix_WithAndWithoutLine covers both locationSuffix branches and the
// no-locations case.
func TestLocationSuffix_WithAndWithoutLine(t *testing.T) {
	withLine := ContractEngineResult{Locations: []SarifLocation{{File: "a.go", Line: 7}}}
	if got := locationSuffix(withLine); got != " in a.go:7" {
		t.Errorf("locationSuffix with line = %q", got)
	}
	noLine := ContractEngineResult{Locations: []SarifLocation{{File: "a.go"}}}
	if got := locationSuffix(noLine); got != " in a.go" {
		t.Errorf("locationSuffix without line = %q", got)
	}
	none := ContractEngineResult{}
	if got := locationSuffix(none); got != "" {
		t.Errorf("locationSuffix with no locations = %q, want empty", got)
	}
}

// TestPackVerdict_PresentAndAbsencePolarities exercises PackVerdict end-to-end over real
// fixtures (both polarities) so its happy paths are covered through the public verdict.
func TestPackVerdict_PresentAndAbsencePolarities(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	td := func(n string) string { return filepath.Join(root, "pkg", "gate", "testdata", n) }

	// Present signature -> no violation.
	raised, err := PackVerdict(root, ContractEntry{File: td("contract-sig-present.go"), Name: "RouteFile", Signature: "func RouteFile(path string, mode int) (string, error)"})
	if err != nil || raised {
		t.Errorf("present signature: raised=%v err=%v, want false/nil", raised, err)
	}
	// Absence present -> violation.
	raised, err = PackVerdict(root, ContractEntry{File: td("contract-absence-present.go"), Name: "legacyProbeSymbol", Absent: true, Scope: td("contract-absence-present.go")})
	if err != nil || !raised {
		t.Errorf("absence present: raised=%v err=%v, want true/nil", raised, err)
	}
}

// TestPackContractResult_AllPolaritiesOverRealEngines drives PackContractResult through
// each real-engine branch (present match, present no-match, absence match, absence empty).
func TestPackContractResult_AllPolaritiesOverRealEngines(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	td := func(n string) string { return filepath.Join(root, "pkg", "gate", "testdata", n) }

	// Present match.
	r, err := PackContractResult(root, ContractEntry{File: td("contract-sig-present.go"), Name: "RouteFile", Signature: "func RouteFile(path string, mode int) (string, error)"})
	if err != nil || !r.Matched || !r.Scanned {
		t.Errorf("present match: %+v err=%v", r, err)
	}
	// Present no-match.
	r, err = PackContractResult(root, ContractEntry{File: td("contract-sig-mismatch.go"), Name: "RouteFile", Signature: "func RouteFile(path string, mode int) (string, error)"})
	if err != nil || r.Matched {
		t.Errorf("present no-match: %+v err=%v", r, err)
	}
	// Absence match (forbidden present).
	r, err = PackContractResult(root, ContractEntry{File: td("contract-absence-present.go"), Name: "legacyProbeSymbol", Absent: true, Scope: td("contract-absence-present.go")})
	if err != nil || !r.Matched || !r.Scanned {
		t.Errorf("absence match: %+v err=%v", r, err)
	}
	// Absence empty (forbidden absent).
	r, err = PackContractResult(root, ContractEntry{File: td("contract-absence-clean.go"), Name: "legacyProbeSymbol", Absent: true, Scope: td("contract-absence-clean.go")})
	if err != nil || r.Matched || !r.Scanned {
		t.Errorf("absence empty: %+v err=%v", r, err)
	}
}

// TestRunScript_ErrorOnMissingScript covers the runScript error branch.
func TestRunScript_ErrorOnMissingScript(t *testing.T) {
	if _, err := runScript("/nonexistent/script.sh", "x"); err == nil {
		t.Error("runScript over a missing script must error")
	}
	if _, err := runScriptStdin("/nonexistent/script.sh", []byte("x")); err == nil {
		t.Error("runScriptStdin over a missing script must error")
	}
}

// TestConvertToLocations_BadSarifErrors covers the convert-parse error branch by feeding
// a convert script that emits non-JSON.
func TestConvertToLocations_BadSarifErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad.sh")
	if err := os.WriteFile(bad, []byte("#!/bin/sh\necho 'not json'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := convertToLocations(bad, []byte("[]")); err == nil {
		t.Error("convertToLocations over non-JSON convert output must error")
	}
}

// TestPackContractResult_ScopeFallbackAndMissingFile covers the absence empty-Scope
// fallback (uses File) and the present missing-file unscanned branch.
func TestPackContractResult_ScopeFallbackAndMissingFile(t *testing.T) {
	requireEngines(t)
	root := eqRepoRoot(t)
	td := func(n string) string { return filepath.Join(root, "pkg", "gate", "testdata", n) }

	// Absence with EMPTY Scope -> falls back to File (which is the present fixture).
	r, err := PackContractResult(root, ContractEntry{File: td("contract-absence-present.go"), Name: "legacyProbeSymbol", Absent: true})
	if err != nil || !r.Matched {
		t.Errorf("absence empty-scope fallback to File: %+v err=%v", r, err)
	}

	// Present signature over a MISSING file -> unscanned (no probe).
	r, err = PackContractResult(root, ContractEntry{File: td("does-not-exist.go"), Name: "X", Signature: "func X()"})
	if err != nil || r.Scanned || r.Matched {
		t.Errorf("present over missing file must be unscanned: %+v err=%v", r, err)
	}

	// Absence over a MISSING scope -> unscanned.
	r, err = PackContractResult(root, ContractEntry{File: td("does-not-exist.go"), Name: "X", Absent: true, Scope: td("does-not-exist.go")})
	if err != nil || r.Scanned {
		t.Errorf("absence over missing scope must be unscanned: %+v err=%v", r, err)
	}
}

// TestPackVerdict_ErrorWhenConvertCrashes covers PackVerdict's error branch by installing
// a workspace pack whose ast-grep convert script crashes, so the probe errors.
func TestPackVerdict_ErrorWhenConvertCrashes(t *testing.T) {
	requireEngines(t)
	ws := t.TempDir()
	pack := filepath.Join(ws, ".backstop", "packs", "backstop", "contracts")
	for _, d := range []string{"scripts", "ast-grep", "grep"} {
		if err := os.MkdirAll(filepath.Join(pack, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	// A real compiler, but convert scripts that crash (exit 1).
	mustWrite(t, filepath.Join(pack, "scripts", "compile-signature.sh"), "#!/bin/sh\nprintf 'func X()'\n")
	mustWrite(t, filepath.Join(pack, "ast-grep", "to-sarif.sh"), "#!/bin/sh\nexit 1\n")
	mustWrite(t, filepath.Join(pack, "grep", "to-sarif.sh"), "#!/bin/sh\nexit 1\n")
	// A real source file to scan.
	src := filepath.Join(ws, "x.go")
	mustWrite(t, src, "package x\nfunc X() {}\n")

	_, err := PackVerdict(ws, ContractEntry{File: src, Name: "X", Signature: "func X()"})
	if err == nil {
		t.Error("PackVerdict must propagate an error when the convert script crashes")
	}
}

func mustWrite(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
