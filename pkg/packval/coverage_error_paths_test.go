package packval

import (
	"os"
	"path/filepath"
	"testing"
)

// --- Phase 1 error paths ---

func TestPackVal_P1_NilManifest(t *testing.T) {
	r := RunStructural(nil, t.TempDir())
	if r.Status != "fail" {
		t.Error("expected fail for nil manifest")
	}
}

func TestPackVal_P1_InvalidSemver(t *testing.T) {
	m := &PackManifest{Name: "a/b", Version: "not-semver", Language: "go", Archetype: "enforcement", Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}}}}
	r := RunStructural(m, t.TempDir())
	hasVersionErr := false
	for _, e := range r.Errors {
		if e.Check == "version" {
			hasVersionErr = true
		}
	}
	if !hasVersionErr {
		t.Error("expected version error for non-semver")
	}
}

func TestPackVal_P1_UnsupportedLanguage(t *testing.T) {
	m := &PackManifest{Name: "a/b", Version: "1.0.0", Language: "rust", Archetype: "enforcement", Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}}}}
	r := RunStructural(m, t.TempDir())
	hasLangErr := false
	for _, e := range r.Errors {
		if e.Check == "language" {
			hasLangErr = true
		}
	}
	if !hasLangErr {
		t.Error("expected language error for unsupported language")
	}
}

func TestPackVal_P1_InvalidArchetype(t *testing.T) {
	m := &PackManifest{Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "hybrid", Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}}}}
	r := RunStructural(m, t.TempDir())
	hasArchErr := false
	for _, e := range r.Errors {
		if e.Check == "archetype" {
			hasArchErr = true
		}
	}
	if !hasArchErr {
		t.Error("expected archetype error for invalid archetype")
	}
}

func TestPackVal_P1_MissingRuleFile(t *testing.T) {
	m := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness", File: "rules/nonexistent.yml"}}}},
	}
	r := RunStructural(m, t.TempDir())
	hasFileErr := false
	for _, e := range r.Errors {
		if e.Check == "file-exists" && e.Rule == "r1" {
			hasFileErr = true
		}
	}
	if !hasFileErr {
		t.Error("expected file-exists error for missing rule file")
	}
}

func TestPackVal_P1_MissingFixtureFile(t *testing.T) {
	m := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{
			ID: "r1", RiskClass: "correctness",
			Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "fixtures/nonexistent.go"}},
				Negative: []FixtureRef{{Path: "fixtures/also-missing.go"}},
			}}},
		}}}},
	}
	r := RunStructural(m, t.TempDir())
	fileErrs := 0
	for _, e := range r.Errors {
		if e.Check == "file-exists" {
			fileErrs++
		}
	}
	if fileErrs < 2 {
		t.Errorf("expected at least 2 file-exists errors, got %d", fileErrs)
	}
}

func TestPackVal_P1_MissingValidatorFile(t *testing.T) {
	m := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness", Validator: "validators/missing.sh"}}}},
	}
	r := RunStructural(m, t.TempDir())
	hasFileErr := false
	for _, e := range r.Errors {
		if e.Check == "file-exists" && e.Rule == "r1" {
			hasFileErr = true
		}
	}
	if !hasFileErr {
		t.Error("expected file-exists error for missing validator")
	}
}

func TestPackVal_P1_MissingScaffoldPath(t *testing.T) {
	m := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "code",
		Content: Content{
			Ruleset:   Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}},
			Scaffolds: []Scaffold{{ID: "s1", Path: "scaffolds/nonexistent"}},
		},
	}
	r := RunStructural(m, t.TempDir())
	hasFileErr := false
	for _, e := range r.Errors {
		if e.Check == "file-exists" && e.Rule == "s1" {
			hasFileErr = true
		}
	}
	if !hasFileErr {
		t.Error("expected file-exists error for missing scaffold path")
	}
}

func TestPackVal_P1_ToolConfigInvalidRiskClass(t *testing.T) {
	m := &PackManifest{
		Name: "a/b", Version: "1.0.0", Language: "go", Archetype: "enforcement",
		Content:    Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", RiskClass: "correctness"}}}},
		ToolConfig: []ToolConfigEntry{{ID: "tc1", RiskClass: "bogus"}},
	}
	r := RunStructural(m, t.TempDir())
	hasRiskErr := false
	for _, e := range r.Errors {
		if e.Check == "risk-class" && e.Rule == "tc1" {
			hasRiskErr = true
		}
	}
	if !hasRiskErr {
		t.Error("expected risk-class error for invalid tool_config risk_class")
	}
}

// --- Phase 4 error paths ---

func TestPackVal_P4_NilManifest(t *testing.T) {
	r := RunArchetype(nil)
	if r.Status != "fail" {
		t.Error("expected fail for nil manifest")
	}
}

func TestPackVal_P4_CodePackDanglingScaffoldRef(t *testing.T) {
	m := &PackManifest{
		Archetype: "code",
		Content: Content{
			Ruleset: Ruleset{Rules: []Rule{{ID: "r1", PairsWith: PairsWith{Scaffolds: []string{"s1"}}}}},
			Scaffolds: []Scaffold{{ID: "s1", PairsWith: PairsWith{Rules: []string{"nonexistent-rule"}}}},
		},
	}
	r := RunArchetype(m)
	hasDangling := false
	for _, e := range r.Errors {
		if e.Message == "scaffold pairs_with unresolved rule" {
			hasDangling = true
		}
	}
	if !hasDangling {
		t.Error("expected dangling reference error")
	}
}

// --- Phase 5 error paths ---

func TestPackVal_P5_NilManifest(t *testing.T) {
	r := RunLayer(nil, t.TempDir())
	if r.Status != "fail" {
		t.Error("expected fail for nil manifest")
	}
}

func TestPackVal_P5_InvalidLayer(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{ID: "r1", Layer: 5}}}}}
	r := RunLayer(m, t.TempDir())
	hasLayerErr := false
	for _, e := range r.Errors {
		if e.Check == "layer" {
			hasLayerErr = true
		}
	}
	if !hasLayerErr {
		t.Error("expected layer error for invalid layer 5")
	}
}

func TestPackVal_P5_Layer3MissingValidator(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Category: "presence", InputScope: "single-file", Validator: "",
	}}}}}
	r := RunLayer(m, t.TempDir())
	hasValErr := false
	for _, e := range r.Errors {
		if e.Check == "validator" && e.Message == "missing validator" {
			hasValErr = true
		}
	}
	if !hasValErr {
		t.Error("expected missing validator error")
	}
}

func TestPackVal_P5_Layer3ValidatorFileNotFound(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Category: "presence", InputScope: "single-file", Validator: "validators/gone.sh",
	}}}}}
	r := RunLayer(m, t.TempDir())
	hasFileErr := false
	for _, e := range r.Errors {
		if e.Check == "validator" && e.Message == "validator file not found" {
			hasFileErr = true
		}
	}
	if !hasFileErr {
		t.Error("expected validator file not found error")
	}
}

func TestPackVal_P5_Layer3OtherMissingJustification(t *testing.T) {
	dir := t.TempDir()
	validatorPath := filepath.Join(dir, "v.sh")
	os.WriteFile(validatorPath, []byte("#!/bin/sh\nexit 0\n"), 0o755)

	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Category: "other", InputScope: "single-file", Validator: "v.sh", Justification: "",
	}}}}}
	r := RunLayer(m, dir)
	hasJustErr := false
	for _, e := range r.Errors {
		if e.Check == "justification" {
			hasJustErr = true
		}
	}
	if !hasJustErr {
		t.Error("expected justification error for category other without justification")
	}
}

func TestPackVal_P5_Layer12WithCategory(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{
		{ID: "r1", Layer: 1, Category: "presence"},
		{ID: "r2", Layer: 2, Category: "structural"},
	}}}}
	r := RunLayer(m, t.TempDir())
	catErrors := 0
	for _, e := range r.Errors {
		if e.Check == "category" {
			catErrors++
		}
	}
	if catErrors < 2 {
		t.Errorf("expected 2 category forbidden errors, got %d", catErrors)
	}
}

// --- Phase 6 error paths ---

func TestPackVal_P6_NilManifest(t *testing.T) {
	r := RunRiskClass(nil)
	if r.Status != "fail" {
		t.Error("expected fail for nil manifest")
	}
}

func TestPackVal_P6_SharedFixturesAcrossSecurityClaims(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", RiskClass: "security",
		Claims: []Claim{
			{ID: "c1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "shared.go"}},
				Negative: []FixtureRef{{Path: "bad.go", BypassAttempt: true}},
			}},
			{ID: "c2", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "shared.go"}}, // shared with c1
				Negative: []FixtureRef{{Path: "bad2.go", BypassAttempt: true}},
			}},
		},
	}}}}}
	r := RunRiskClass(m)
	hasSharedErr := false
	for _, e := range r.Errors {
		if e.Check == "independent-fixtures" {
			hasSharedErr = true
		}
	}
	if !hasSharedErr {
		t.Error("expected independent-fixtures error for shared fixture across security claims")
	}
}

// --- Phase 3 error paths (semgrepFileContainsRuleID, copyDir) ---

func TestPackVal_P3_SemgrepEmptyData(t *testing.T) {
	ok := semgrepFileContainsRuleID([]byte{}, "rule-1")
	if ok {
		t.Error("expected false for empty data")
	}
}

func TestPackVal_P3_SemgrepBadYAML(t *testing.T) {
	ok := semgrepFileContainsRuleID([]byte("not: [valid: yaml: {{"), "rule-1")
	if ok {
		t.Error("expected false for malformed YAML")
	}
}

func TestPackVal_P3_SemgrepNoMatchingRule(t *testing.T) {
	data := []byte("rules:\n  - id: other-rule\n    pattern: foo\n")
	ok := semgrepFileContainsRuleID(data, "my-rule")
	if ok {
		t.Error("expected false when rule ID doesn't match")
	}
}

func TestPackVal_P3_SemgrepMatchingRule(t *testing.T) {
	data := []byte("rules:\n  - id: my-rule\n    pattern: foo\n")
	ok := semgrepFileContainsRuleID(data, "my-rule")
	if !ok {
		t.Error("expected true when rule ID matches")
	}
}

func TestPackVal_P3_CopyDir_NonexistentSource(t *testing.T) {
	err := copyDir("/nonexistent/source", t.TempDir())
	if err == nil {
		t.Error("expected error for nonexistent source dir")
	}
}

func TestPackVal_P3_CopyDir_ValidCopy(t *testing.T) {
	src := t.TempDir()
	os.WriteFile(filepath.Join(src, "file.txt"), []byte("hello"), 0o644)
	os.MkdirAll(filepath.Join(src, "sub"), 0o755)
	os.WriteFile(filepath.Join(src, "sub", "nested.txt"), []byte("world"), 0o644)

	dst := t.TempDir()
	err := copyDir(src, dst)
	if err != nil {
		t.Fatalf("copyDir: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(dst, "file.txt"))
	if err != nil || string(data) != "hello" {
		t.Error("expected copied file content to match")
	}
	data, err = os.ReadFile(filepath.Join(dst, "sub", "nested.txt"))
	if err != nil || string(data) != "world" {
		t.Error("expected nested file content to match")
	}
}

// --- Phase 3 RunFixtures error branches ---

func TestPackVal_P3_NilManifest(t *testing.T) {
	r := RunFixtures(nil, t.TempDir(), &MockExecutor{})
	if r.Status != "fail" {
		t.Error("expected fail for nil manifest")
	}
}

func TestPackVal_P3_NilExecutorUsesDefault(t *testing.T) {
	// nil executor should default to DefaultExecutor without panic
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{}}}}
	r := RunFixtures(m, t.TempDir(), nil)
	if r.Status != "pass" {
		t.Error("expected pass for empty rules with default executor")
	}
}

func TestPackVal_P3_SemgrepRuleFileReadError(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", File: "nonexistent.yml",
	}}}}}
	r := RunFixtures(m, t.TempDir(), &MockExecutor{})
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "semgrep-rule-id" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected semgrep-rule-id error for missing rule file")
	}
}

func TestPackVal_P3_SemgrepPositiveFixtureFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "rule.yml"), []byte("rules:\n  - id: r1\n"), 0o644)
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", File: "rule.yml",
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "good.go"}},
			Negative: []FixtureRef{{Path: "bad.go"}},
		}}},
	}}}}}
	mock := &MockExecutor{
		SemgrepFn: func(_, _, fp string) (ExecutionResult, error) {
			if fp == "good.go" {
				return ExecutionResult{Passed: false, ExitCode: 1}, nil // positive fails
			}
			return ExecutionResult{Passed: false, ExitCode: 1}, nil // negative triggers (correct)
		},
	}
	r := RunFixtures(m, dir, mock)
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "semgrep-positive" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected semgrep-positive error when positive fixture fails")
	}
}

func TestPackVal_P3_SemgrepNegativeNotTriggered(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "rule.yml"), []byte("rules:\n  - id: r1\n"), 0o644)
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", File: "rule.yml",
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "good.go"}},
			Negative: []FixtureRef{{Path: "bad.go"}},
		}}},
	}}}}}
	mock := &MockExecutor{
		SemgrepFn: func(_, _, fp string) (ExecutionResult, error) {
			return ExecutionResult{Passed: true, ExitCode: 0}, nil // both pass — negative not triggered
		},
	}
	r := RunFixtures(m, dir, mock)
	hasHint := false
	for _, e := range r.Errors {
		if e.Check == "semgrep-negative" && e.FixHint != "" {
			hasHint = true
		}
	}
	if !hasHint {
		t.Error("expected semgrep-negative error with engine limitation fix hint")
	}
}

func TestPackVal_P3_ValidatorPositiveFails(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Validator: "v.sh", InputScope: "single-file",
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "good/"}},
			Negative: []FixtureRef{{Path: "bad/"}},
		}}},
	}}}}}
	mock := &MockExecutor{
		ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
			return ExecutionResult{Passed: false, ExitCode: 1}, nil
		},
	}
	r := RunFixtures(m, t.TempDir(), mock)
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "validator-positive" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected validator-positive error")
	}
}

func TestPackVal_P3_ValidatorNegativePassesUnexpectedly(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Validator: "v.sh", InputScope: "single-file",
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "good/"}},
			Negative: []FixtureRef{{Path: "bad/"}},
		}}},
	}}}}}
	mock := &MockExecutor{
		ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
			return ExecutionResult{Passed: true, ExitCode: 0}, nil // negative passes = bad
		},
	}
	r := RunFixtures(m, t.TempDir(), mock)
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "validator-negative" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected validator-negative error when negative passes")
	}
}

func TestPackVal_P3_ToolConfigPositiveFails(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.21\n"), 0o644)
	m := &PackManifest{
		ToolConfig: []ToolConfigEntry{{
			ID: "tc1", Tool: "golangci-lint", File: ".golangci.yml",
			Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "good.go"}},
				Negative: []FixtureRef{{Path: "bad.go"}},
			}}},
		}},
	}
	mock := &MockExecutor{
		ToolConfigFn: func(_, _, _, fp string) (ExecutionResult, error) {
			if fp == "good.go" {
				return ExecutionResult{Passed: false, ExitCode: 1}, nil
			}
			return ExecutionResult{Passed: false, ExitCode: 1}, nil
		},
	}
	r := RunFixtures(m, dir, mock)
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "tool-config-positive" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected tool-config-positive error")
	}
}

func TestPackVal_P3_ToolConfigNegativeNotTriggered(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\ngo 1.21\n"), 0o644)
	m := &PackManifest{
		ToolConfig: []ToolConfigEntry{{
			ID: "tc1", Tool: "golangci-lint", File: ".golangci.yml",
			Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
				Positive: []FixtureRef{{Path: "good.go"}},
				Negative: []FixtureRef{{Path: "bad.go"}},
			}}},
		}},
	}
	mock := &MockExecutor{
		ToolConfigFn: func(_, _, _, _ string) (ExecutionResult, error) {
			return ExecutionResult{Passed: true, ExitCode: 0}, nil
		},
	}
	r := RunFixtures(m, dir, mock)
	hasErr := false
	for _, e := range r.Errors {
		if e.Check == "tool-config-negative" {
			hasErr = true
		}
	}
	if !hasErr {
		t.Error("expected tool-config-negative error when negative passes")
	}
}

func TestPackVal_P3_MultiFileValidator(t *testing.T) {
	m := &PackManifest{Content: Content{Ruleset: Ruleset{Rules: []Rule{{
		ID: "r1", Layer: 3, Validator: "v.sh", InputScope: "multi-file",
		Claims: []Claim{{ID: "c1", Fixtures: Fixtures{
			Positive: []FixtureRef{{Path: "a.go"}, {Path: "b.go"}},
			Negative: []FixtureRef{{Path: "c.go"}},
		}}},
	}}}}}
	calls := 0
	mock := &MockExecutor{
		ValidatorFn: func(_, _ string, paths []string) (ExecutionResult, error) {
			calls++
			if len(paths) == 1 && paths[0] == "c.go" {
				// Single negative fixture should fail (validator catches the bad code)
				return ExecutionResult{Passed: false, ExitCode: 1}, nil
			}
			// Positive single-file calls and multi-file batch call pass
			return ExecutionResult{Passed: true, ExitCode: 0}, nil
		},
	}
	r := RunFixtures(m, t.TempDir(), mock)
	if r.Status != "pass" {
		t.Errorf("expected pass, got %s with errors: %v", r.Status, r.Errors)
	}
	if calls < 3 {
		t.Errorf("expected at least 3 validator calls for multi-file, got %d", calls)
	}
}

// --- goModTidyTempCopy error path ---

func TestPackVal_P3_GoModTidyCopy_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	err := goModTidyTempCopy(dir)
	// go mod tidy in a dir without go.mod may error — we just exercise the path
	_ = err
}

func TestPackVal_P3_GoModTidyCopy_WithGoMod(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.21\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644)
	err := goModTidyTempCopy(dir)
	if err != nil {
		t.Logf("goModTidyTempCopy: %v (may fail without network, acceptable)", err)
	}
	// Verify original go.mod unchanged
	data, _ := os.ReadFile(filepath.Join(dir, "go.mod"))
	if string(data) != "module test\n\ngo 1.21\n" {
		t.Error("go.mod was modified — should use temp copy")
	}
}
