package packval_test

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/packval"
)

func TestPackVal_ResultPassWhenNoErrors(t *testing.T) {
	r := &packval.Result{}
	r.FinalizeStatus()
	if r.Status != "pass" {
		t.Fatalf("want pass, got %s", r.Status)
	}
}
func TestPackVal_ResultFailWhenErrors(t *testing.T) {
	r := &packval.Result{Errors: []packval.ValidationError{{Message: "x"}}}
	r.FinalizeStatus()
	if r.Status != "fail" {
		t.Fatalf("want fail, got %s", r.Status)
	}
}
func TestPackVal_ValidationErrorFields(t *testing.T) {
	e := packval.ValidationError{Phase: "p", Check: "c", Rule: "r", Claim: "cl", Message: "m", FixHint: "h", ManifestPath: "x"}
	if e.Message == "" || e.Check == "" {
		t.Fatal("expected populated fields")
	}
}
func TestPackVal_PhaseResultSkippedReason(t *testing.T) {
	p := packval.PhaseResult{Status: "skipped", Reason: "x"}
	if p.Reason == "" {
		t.Fatal("want reason")
	}
}
func TestPackVal_ValidationWarningFields(t *testing.T) {
	w := packval.ValidationWarning{Message: "m", Check: "c"}
	if w.Message == "" {
		t.Fatal("want message")
	}
}
func TestPackVal_ParseManifestValid(t *testing.T) {
	dir := makePackDir(t)
	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil || m.Name == "" {
		t.Fatalf("parse failed: %v", err)
	}
}
func TestPackVal_ParseManifestInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", ":\n:")
	if _, err := packval.ParseManifest(filepath.Join(dir, "pack.yml")); err == nil {
		t.Fatal("expected parse error")
	}
}
func TestPackVal_ParseManifestMissingFile(t *testing.T) {
	if _, err := packval.ParseManifest(filepath.Join(t.TempDir(), "missing.yml")); err == nil {
		t.Fatal("expected missing file error")
	}
}
func TestPackVal_ParseManifestRulesExtracted(t *testing.T) {
	dir := makePackDir(t)
	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil {
		t.Fatalf("parse manifest: %v", err)
	}
	if len(m.Content.Ruleset.Rules) == 0 {
		t.Fatal("expected rules")
	}
}
func TestPackVal_ParseManifestToolConfigExtracted(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "pack.yml", strings.TrimSpace(`
name: acme/example
version: 1.0.0
language: go
archetype: enforcement
tool_config:
  - id: T1
    tool: golangci-lint
    file: .golangci.yml
    risk_class: style
    claims:
      - id: C1
        fixtures:
          positive: [fixtures/p.go]
          negative: [fixtures/n.go]
content: {ruleset: {rules: []}}
`))
	m, err := packval.ParseManifest(filepath.Join(dir, "pack.yml"))
	if err != nil || len(m.ToolConfig) != 1 {
		t.Fatalf("want tool config, err=%v", err)
	}
}
func TestPackVal_JSONOutputStructure(t *testing.T) {
	r := &packval.Result{Status: "pass"}
	out, err := packval.FormatResult(r, "json")
	if err != nil || !strings.Contains(out, `"status": "pass"`) {
		t.Fatalf("bad json: %v", err)
	}
}
func TestPackVal_ErrorFieldsComplete(t *testing.T) {
	b, err := json.Marshal(packval.ValidationError{Phase: "p", Check: "c", Message: "m"})
	if err != nil || !strings.Contains(string(b), "phase") {
		t.Fatalf("marshal failed: %v", err)
	}
}
func TestPackVal_PhaseStatusFields(t *testing.T) {
	p := packval.RunStructural(baseManifest(), makePackDir(t))
	if p.Status != "pass" {
		t.Fatal("status")
	}
	if p.Checks == 0 {
		t.Fatal("checks should be > 0")
	}
}
func TestPackVal_SkippedPhaseReason(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", "invalid: [yaml")
	r := packval.NewPipeline(dir, packval.PipelineOptions{Mode: "check"}).Run()
	for _, ph := range r.Phases {
		if ph.Status == "skipped" {
			if !strings.Contains(ph.Reason, "phase1") {
				t.Fatalf("expected reason to name the failed phase, got %q", ph.Reason)
			}
			return
		}
	}
	t.Fatal("expected at least one skipped phase")
}
func TestPackVal_TextFormat(t *testing.T) {
	out, err := packval.FormatResult(&packval.Result{Status: "pass"}, "text")
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !strings.Contains(out, "status") {
		t.Fatal("text")
	}
}
func TestPackVal_DefaultFormatJSON(t *testing.T) {
	out, err := packval.FormatResult(&packval.Result{Status: "pass"}, "")
	if err != nil {
		t.Fatalf("format: %v", err)
	}
	if !strings.Contains(out, `"status"`) {
		t.Fatal("json default")
	}
}

func TestPackVal_P1_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "pack.yml", "bad: [")
	if _, err := packval.ParseManifest(filepath.Join(dir, "pack.yml")); err == nil {
		t.Fatal("want invalid yaml error")
	}
}
func TestPackVal_P1_MissingName(t *testing.T) {
	m := baseManifest()
	m.Name = ""
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_MissingVersion(t *testing.T) {
	m := baseManifest()
	m.Version = ""
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_MissingLanguage(t *testing.T) {
	m := baseManifest()
	m.Language = ""
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_MissingArchetype(t *testing.T) {
	m := baseManifest()
	m.Archetype = ""
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_MissingContent(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules = nil
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_InvalidArchetype(t *testing.T) {
	m := baseManifest()
	m.Archetype = "x"
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_InvalidVersion(t *testing.T) {
	m := baseManifest()
	m.Version = "x"
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_NonGoLanguageAccepted(t *testing.T) {
	// The harness is language-neutral (ISSUE-019): a non-Go language is accepted.
	m := baseManifest()
	m.Language = "java"
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass for a non-Go pack (no language gate)")
	}
}
func TestPackVal_P1_LanguageGoAccepted(t *testing.T) {
	if packval.RunStructural(baseManifest(), makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P1_RuleFileMissing(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].File = "missing.yml"
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_ReferencedFileNotFound(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive = []packval.FixtureRef{{Path: "nope.go"}}
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_ValidManifest(t *testing.T) {
	if packval.RunStructural(baseManifest(), makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P1_ToolConfigFileExcluded(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: "missing.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("tool_config.file should be excluded")
	}
}
func TestPackVal_P1_MissingRiskClass(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = ""
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_InvalidRiskClass(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "bad"
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_ToolConfigMissingRiskClass(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: "x.yml", Claims: m.Content.Ruleset.Rules[0].Claims}}
	if packval.RunStructural(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P1_ValidRiskClassSecurity(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "security"
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P1_ValidRiskClassCorrectness(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "correctness"
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P1_ValidRiskClassStyle(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "style"
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P1_ValidRiskClassPerf(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].RiskClass = "perf"
	if packval.RunStructural(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}

func TestPackVal_P2_RuleNoClaims(t *testing.T) {
	m := baseManifest()
	// Clear the engine so the has-claims requirement is ENFORCED: a claimless rule with
	// a resolvable engine is now exempt (ISSUE-032 CLM-005), so this test exercises the
	// non-exempt (no-engine) path where missing claims still fails.
	m.Content.Ruleset.Rules[0].Engine = ""
	m.Content.Ruleset.Rules[0].Claims = nil
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_ClaimNoPositiveFixtures(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive = nil
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_ClaimNoNegativeFixtures(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Negative = nil
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_FixtureFileNotFound(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims[0].Fixtures.Positive = []packval.FixtureRef{{Path: "missing.go"}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_FixtureFileEmpty(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "fixtures/p.go", "   ")
	if packval.RunCoherence(baseManifest(), dir).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_DuplicateClaimIDs(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].Claims = append(m.Content.Ruleset.Rules[0].Claims, m.Content.Ruleset.Rules[0].Claims[0])
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_DuplicateRuleIDRulesetAndToolConfig(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "R1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: m.Content.Ruleset.Rules[0].Claims}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_DuplicateRuleIDWithinRuleset(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules, m.Content.Ruleset.Rules[0])
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_UniqueRuleIDsAcrossSources(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{
		ID:        "T2",
		Tool:      "golangci-lint",
		File:      ".golangci.yml",
		RiskClass: "style",
		Claims: []packval.Claim{{
			ID: "TC2",
			Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.go"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
			},
		}},
	}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P2_ToolConfigOwnIDNoClaims(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style"}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_ToolConfigOwnIDNoFixtures(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{ID: "T1", Tool: "golangci-lint", File: ".golangci.yml", RiskClass: "style", Claims: []packval.Claim{{ID: "TC"}}}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "fail" {
		t.Fatal("expected fail")
	}
}
func TestPackVal_P2_ToolConfigOwnIDWithClaimsAndFixtures(t *testing.T) {
	m := baseManifest()
	m.ToolConfig = []packval.ToolConfigEntry{{
		ID:        "T1",
		Tool:      "golangci-lint",
		File:      ".golangci.yml",
		RiskClass: "style",
		Claims: []packval.Claim{{
			ID: "TC1",
			Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.go"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
			},
		}},
	}}
	if packval.RunCoherence(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P2_DanglingPairsWithWarning(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].PairsWith.Rules = []string{"MISSING"}
	if len(packval.RunCoherence(m, makePackDir(t)).Warnings) == 0 {
		t.Fatal("expected warning")
	}
}
func TestPackVal_P2_DanglingPairsWithNotError(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules[0].PairsWith.Rules = []string{"MISSING"}
	if packval.RunCoherence(m, makePackDir(t)).Status != "pass" {
		t.Fatal("warning should not fail")
	}
}
func TestPackVal_P2_ValidPairsWithReference(t *testing.T) {
	m := baseManifest()
	m.Content.Ruleset.Rules = append(m.Content.Ruleset.Rules, packval.Rule{
		ID:        "R2",
		RiskClass: "style",
		Claims: []packval.Claim{{
			ID: "C2",
			Fixtures: packval.Fixtures{
				Positive: []packval.FixtureRef{{Path: "fixtures/p.go"}},
				Negative: []packval.FixtureRef{{Path: "fixtures/n.go"}},
			},
		}},
	})
	m.Content.Ruleset.Rules[0].PairsWith.Rules = []string{"R2"}
	if packval.RunCoherence(m, makePackDir(t)).Status != "pass" {
		t.Fatal("expected pass")
	}
}
func TestPackVal_P2_OrphanFixtureWarning(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "fixtures/orphan.go", "package p")
	if len(packval.RunCoherence(baseManifest(), dir).Warnings) == 0 {
		t.Fatal("expected warning")
	}
}
func TestPackVal_P2_OrphanFixtureNotError(t *testing.T) {
	dir := makePackDir(t)
	writeFile(t, dir, "fixtures/orphan.go", "package p")
	if packval.RunCoherence(baseManifest(), dir).Status != "pass" {
		t.Fatal("warning should not fail")
	}
}

func TestPackVal_FormatResultUnknownFormat(t *testing.T) {
	_, err := packval.FormatResult(&packval.Result{Status: "pass"}, "xml")
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPackVal_AllRulesNilManifest(t *testing.T) {
	rules := packval.AllRules(nil)
	if rules != nil {
		t.Fatalf("expected nil, got %v", rules)
	}
}
