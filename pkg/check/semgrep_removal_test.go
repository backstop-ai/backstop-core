package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ISSUE-018 Section B deletion-assertion + live-path-survival tests. The absence
// assertions pin that the vestigial in-process semgrep path is GONE from
// production source (pkg/check, pkg/config, cmd/backstop) so it cannot be
// silently reintroduced. The survival assertions pin that the live pack-findings
// SARIF path and the shared error contract OUTLIVE the deletion — guarding
// against over-deletion.

// readProductionSources returns the concatenated contents of every non-test .go
// file under the given directory (relative to this package dir), so absence
// assertions scan production source only. Missing dirs are skipped.
func readProductionSources(t *testing.T, dir string) string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var b strings.Builder
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		raw, rerr := os.ReadFile(filepath.Join(dir, name))
		if rerr != nil {
			t.Fatalf("reading %s/%s: %v", dir, name, rerr)
		}
		b.WriteString("// FILE: " + name + "\n")
		b.Write(raw)
		b.WriteString("\n")
	}
	return b.String()
}

// sectionBProductionSources concatenates the production source of the three
// packages the Section B symbols must be absent from.
func sectionBProductionSources(t *testing.T) string {
	t.Helper()
	return readProductionSources(t, ".") +
		readProductionSources(t, filepath.Join("..", "config")) +
		readProductionSources(t, filepath.Join("..", "..", "cmd", "backstop"))
}

// TestInProcessSemgrepExecutor_Removed proves the in-process semgrep executor
// and its JSON-parsing helpers are gone from production source. CLM-004/005/006.
func TestInProcessSemgrepExecutor_Removed(t *testing.T) {
	src := sectionBProductionSources(t)
	for _, sym := range []string{
		"semgrepExecutor",
		"semgrepJSON",
		"parseSemgrepJSON",
		"ParseSemgrepJSONForTest",
		"extractJSONDocument",
		"semgrepSeverity",
	} {
		if containsCheckIdent(src, sym) {
			t.Errorf("production source still references in-process semgrep symbol %q; it must be deleted", sym)
		}
	}
	if _, err := os.Stat("semgrep.go"); err == nil {
		t.Error("pkg/check/semgrep.go still exists; the in-process semgrep executor/resolver file must be deleted")
	}
}

// TestPkgCheck_NoManifestDirFieldOnSemgrepFeed is the SPEC-030 v1.1.0 CLM-003
// absence self-check: no `manifestDir`-style field (nor any compiled-standards
// `--config` feed) survives on any in-process semgrep executor in pkg/check,
// because the in-process semgrep executor TYPE itself is gone under the
// thin-executor strategy (ISSUE-018 removes semgrepExecutor). It scans
// production source only and must NOT assert that any in-process semgrep
// invocation occurs.
func TestPkgCheck_NoManifestDirFieldOnSemgrepFeed(t *testing.T) {
	src := readProductionSources(t, ".")
	// The executor type itself must be gone — a fortiori no field hangs off it.
	if containsCheckIdent(src, "semgrepExecutor") {
		t.Error("pkg/check still defines an in-process semgrepExecutor; no compiled-standards manifestDir feed may survive on it — the executor type must be removed entirely")
	}
	// No surviving manifestDir field/assignment may feed a compiled-standards
	// directory into a semgrep --config arm.
	if containsCheckIdent(src, "manifestDir") {
		t.Error("pkg/check still references a manifestDir field; the compiled-standards --config feed on the in-process semgrep executor must be gone")
	}
	if containsCheckIdent(src, "ManifestDir") {
		t.Error("pkg/check still references an Options.ManifestDir field; the compiled-standards rule-config wiring must be removed")
	}
}

// TestPkgCheck_NoResidualStandardsConfigWhenNoPacks is the SPEC-030 CLM-002
// absence self-check: with the in-process semgrep pass deleted, there is no
// residual compiled-standards `--config` source — no leftover standards-dir
// manifest is assembled into a semgrep `--config`. It asserts the deleted
// compiled-standards arm (the `ExtraSemgrepConfigs` assembly and the
// `manifestDir` `--config` feed) is gone from pkg/check production source, and
// must NOT assert that any in-process semgrep pass is invoked.
func TestPkgCheck_NoResidualStandardsConfigWhenNoPacks(t *testing.T) {
	src := readProductionSources(t, ".")
	for _, sym := range []string{
		"ExtraSemgrepConfigs",
		"manifestDir",
		"ManifestDir",
	} {
		if containsCheckIdent(src, sym) {
			t.Errorf("pkg/check production source still references the deleted compiled-standards arm symbol %q; with zero packs there must be no residual standards-dir --config source", sym)
		}
	}
}

// TestNoTestRequiresManifestDirOrStandardsConfig is the SPEC-030 CLM-023 source
// self-check: no remaining pkg/check TEST file constructs an in-process semgrep
// executor with a `manifestDir` field or asserts a standards-dir
// (`.backstop/rules`) `--config` path. Scanning the test sources (not just
// production) enforces the green `go test ./...` guarantee rather than assuming
// it — a re-introduced standards-dir assertion against the now-absent in-process
// semgrep pass is caught here.
func TestNoTestRequiresManifestDirOrStandardsConfig(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("reading pkg/check dir: %v", err)
	}
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, "_test.go") || name == "semgrep_removal_test.go" {
			// Skip this file itself: it names the forbidden tokens in string
			// literals as the very thing it scans for, which is not a revived
			// manifestDir/standards-dir assertion.
			continue
		}
		raw, rerr := os.ReadFile(name)
		if rerr != nil {
			t.Fatalf("reading %s: %v", name, rerr)
		}
		text := string(raw)
		if strings.Contains(text, "manifestDir:") {
			t.Errorf("%s still constructs a semgrep feed with a manifestDir: field; the in-process compiled-standards feed is gone and no test may require it", name)
		}
		if strings.Contains(text, "containsConfigFor") {
			t.Errorf("%s still asserts a standards-dir --config path (containsConfigFor); no test may assert a compiled-standards --config against the now-absent in-process semgrep pass", name)
		}
	}
}

// TestNoFallback_PopulatedRulesDirNotASource is the SPEC-030 CLM-018 behavioral
// check (consolidated with the former CLM-019 at spec_version 1.3.0): a
// populated `.backstop/rules/` directory is NOT an implicit rule/route source. A
// leftover STD-*.semgrep.yml planted there (with no *.manifest.json) is never
// collected by the sole production reader of that directory — LoadManifest
// returns the built-in default manifest, so the populated dir contributes zero
// rules and zero routes. Behavioral over LoadManifest; it does NOT assert any
// in-process semgrep invocation (none exists), and is distinct from the
// CLM-002/003 production-source token scans.
func TestNoFallback_PopulatedRulesDirNotASource(t *testing.T) {
	dir := t.TempDir()
	rulesDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(rulesDir, 0o755); err != nil {
		t.Fatalf("mkdir rules: %v", err)
	}
	// A leftover compiled-standards file — NOT a *.manifest.json routing file.
	if err := os.WriteFile(filepath.Join(rulesDir, "STD-GO-001.semgrep.yml"), []byte("rules: []\n"), 0o644); err != nil {
		t.Fatalf("write leftover: %v", err)
	}

	m, err := LoadManifest(rulesDir)
	if err != nil {
		t.Fatalf("LoadManifest over a rules dir with only STD-*.semgrep.yml: %v", err)
	}
	// A dir with no *.manifest.json yields the built-in default manifest; the
	// leftover compiled-standards file must not resurrect a second rule source.
	// Post-SPEC-039 (the .manifest.json reader is deleted) this is asserted via
	// LoadManifest's surviving contract: it returns the built-in default routing,
	// so a .go file gets the full default pass set (lint/build/test/findings) and
	// a non-Go file matched by no built-in rule routes to the catch-all default —
	// the populated rules dir altered nothing. The leftover STD-*.semgrep.yml is
	// never collected as a rule/route source.
	goChecks := m.RouteFile("main.go")
	wantGo := []CheckType{CheckTypeLint, CheckTypeBuild, CheckTypeTest, CheckTypeFindings}
	if len(goChecks) != len(wantGo) {
		t.Errorf("a .backstop/rules dir containing only STD-*.semgrep.yml (no *.manifest.json) must yield the built-in default manifest; .go routed to %v, want the full default pass set %v", goChecks, wantGo)
	}
	for _, ct := range wantGo {
		if !containsCheckType(goChecks, ct) {
			t.Errorf(".go default routing missing %v: got %v (populated rules dir must NOT become an implicit rule/route source)", ct, goChecks)
		}
	}
}

// TestSemgrepEnsurer_Removed proves the bespoke ensurer/resolver/installer
// wiring is gone from production source. CLM-007/008/009.
func TestSemgrepEnsurer_Removed(t *testing.T) {
	src := sectionBProductionSources(t)
	for _, sym := range []string{
		"SemgrepEnsurer",
		"DefaultSemgrepEnsurer",
		"EnsureSemgrep",
		"ensureSemgrepWith",
		"SemgrepResolver",
		"DefaultSemgrepInstaller",
		"GolangciLintAvailable",
		"PinnedSemgrepVersion",
		"ensurerForTest",
	} {
		if containsCheckIdent(src, sym) {
			t.Errorf("production source still references retired ensurer symbol %q; it must be deleted", sym)
		}
	}
}

// TestConfig_SemgrepVersion_Removed proves the semgrep_version config knob is
// gone from production source (its declared-provisioning replacement supersedes
// it). CLM-010.
func TestConfig_SemgrepVersion_Removed(t *testing.T) {
	src := sectionBProductionSources(t)
	if containsCheckIdent(src, "SemgrepVersion") {
		t.Error("production source still references the SemgrepVersion config field; it must be removed")
	}
	if strings.Contains(src, "semgrep_version") {
		t.Error("production source still references the semgrep_version yaml tag; it must be removed")
	}
}

// TestCheckTypeFindings_StillPresent guards against over-deletion: the
// pack-findings pass-tag enum is the live findings tag and MUST survive Section
// B. SPEC-035 renamed it from the tool-named CheckTypeSemgrep to the neutral
// CheckTypeFindings (String() "findings"), so this guard tracks the neutral
// name. CLM-011 (survival) / CLM-022+CLM-032 (neutral name+string).
func TestCheckTypeFindings_StillPresent(t *testing.T) {
	// Compile-time reference: if the enum were deleted this file would not build.
	if CheckTypeFindings.String() != "findings" {
		t.Errorf("CheckTypeFindings.String() = %q, want \"findings\"; the live pack-findings pass tag must survive under its neutral name", CheckTypeFindings.String())
	}
}

// TestParsePackFindings_SurvivesExecutorRemoval proves the LIVE SARIF
// pack-findings parser still works after the in-process JSON parser was removed:
// only the dead path was deleted, not parseSarif. CLM-011.
func TestParsePackFindings_SurvivesExecutorRemoval(t *testing.T) {
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
		t.Fatalf("expected exactly 1 violation from the live SARIF path, got %d: %+v", len(vs), vs)
	}
	v := vs[0]
	if v.Pass != CheckTypeFindings {
		t.Errorf("violation Pass = %v, want CheckTypeFindings (the pack-findings tag)", v.Pass)
	}
	if v.Rule != "org/pack/no-panic" || v.File != "pkg/widget/widget.go" || v.Line != 42 {
		t.Errorf("live SARIF finding not parsed correctly, got %+v", v)
	}
}

// TestSharedErrorTypes_SurviveSemgrepFileDeletion proves the cross-cutting error
// contract (ConfigError = fail-loud exit-2, DegradedError = degraded skip)
// outlives the semgrep.go deletion — the c1 relocation to errors.go preserved
// it. CLM-013.
func TestSharedErrorTypes_SurviveSemgrepFileDeletion(t *testing.T) {
	cfgErr := &ConfigError{Message: "hard stop"}
	if cfgErr.Error() != "hard stop" {
		t.Errorf("ConfigError.Error() = %q, want \"hard stop\"; the exit-2 error must round-trip", cfgErr.Error())
	}
	degErr := &DegradedError{Message: "skip me"}
	if degErr.Error() != "skip me" {
		t.Errorf("DegradedError.Error() = %q, want \"skip me\"; the degraded-skip error must round-trip", degErr.Error())
	}

	// The two must be distinguishable by concrete type so cmd code can switch to
	// ExitConfigError on *ConfigError while degraded-skipping on *DegradedError.
	var err error = cfgErr
	if _, ok := err.(*ConfigError); !ok {
		t.Error("*ConfigError must be type-assertable from error for the exit-2 contract")
	}
	if _, ok := err.(*DegradedError); ok {
		t.Error("*ConfigError must NOT also satisfy *DegradedError; the two contracts are distinct")
	}

	// And errors.go must be the surviving home of the relocated types.
	if _, statErr := os.Stat("errors.go"); statErr != nil {
		t.Errorf("pkg/check/errors.go must exist as the relocated home of ConfigError/DegradedError: %v", statErr)
	}
}

// containsCheckIdent reports whether text contains sym as a whole Go identifier
// (word-bounded), so a substring inside a longer name does not false-positive
// (e.g. "EnsureSemgrep" inside a comment about "EnsureSemgrepRetired" tests is
// only matched as a standalone token). Production source is scanned including
// comments, matching the SPEC-034 deletion-guard convention.
func containsCheckIdent(text, sym string) bool {
	idx := 0
	for {
		j := strings.Index(text[idx:], sym)
		if j < 0 {
			return false
		}
		pos := idx + j
		left := pos == 0 || !isCheckIdentRune(rune(text[pos-1]))
		rightPos := pos + len(sym)
		right := rightPos >= len(text) || !isCheckIdentRune(rune(text[rightPos]))
		if left && right {
			return true
		}
		idx = pos + len(sym)
	}
}

func isCheckIdentRune(r rune) bool {
	return r == '_' ||
		(r >= 'a' && r <= 'z') ||
		(r >= 'A' && r <= 'Z') ||
		(r >= '0' && r <= '9')
}
