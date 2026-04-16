package packval

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func RunFixtures(pack *PackManifest, packDir string, executor FixtureExecutor) *PhaseResult {
	res := &PhaseResult{
		Phase:  "phase3-fixtures",
		Status: "pass",
		Checks: 5, // semgrep, tool-config, validators, scaffolds, sdk
	}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	if executor == nil {
		executor = &DefaultExecutor{}
	}

	for _, rule := range pack.Content.Ruleset.Rules {
		if rule.File != "" {
			ruleFilePath := filepath.Join(packDir, rule.File)
			ruleData, err := os.ReadFile(ruleFilePath)
			if err != nil {
				res.Errors = append(res.Errors, ValidationError{
					Phase:   res.Phase,
					Check:   "semgrep-rule-id",
					Rule:    rule.ID,
					Message: fmt.Sprintf("failed to read rule file %s: %v", rule.File, err),
				})
			} else if !semgrepFileContainsRuleID(ruleData, rule.ID) {
				res.Errors = append(res.Errors, ValidationError{
					Phase:   res.Phase,
					Check:   "semgrep-rule-id",
					Rule:    rule.ID,
					Message: fmt.Sprintf("pack rule ID %q not found in rule file %s", rule.ID, rule.File),
				})
			}
		}
		for _, claim := range rule.Claims {
			for _, f := range claim.Fixtures.Positive {
				if rule.File != "" {
					r, err := executor.RunSemgrep(packDir, rule.File, f.Path)
					if err != nil || !r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-positive", Rule: rule.ID, Claim: claim.ID, Message: "positive fixture failed"})
					}
				}
				if rule.Layer == 3 && rule.Validator != "" {
					r, err := executor.RunValidator(packDir, rule.Validator, []string{f.Path})
					if err != nil || !r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-positive", Rule: rule.ID, Claim: claim.ID, Message: "layer3 positive failed"})
					}
				}
			}
			for _, f := range claim.Fixtures.Negative {
				if rule.File != "" {
					r, err := executor.RunSemgrep(packDir, rule.File, f.Path)
					if err != nil {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-negative", Rule: rule.ID, Claim: claim.ID, Message: "negative fixture run failed"})
					} else if r.Passed {
						res.Errors = append(res.Errors, ValidationError{
							Phase:   res.Phase,
							Check:   "semgrep-negative",
							Rule:    rule.ID,
							Claim:   claim.ID,
							Message: "negative fixture not triggered",
							FixHint: "This negative fixture did not trigger the rule and may indicate an engine limitation. The fixture may represent a pattern the rule engine cannot detect. Consider removing this fixture and documenting the limitation rather than shipping an untestable claim.",
						})
					}
				}
				if rule.Layer == 3 && rule.Validator != "" {
					r, err := executor.RunValidator(packDir, rule.Validator, []string{f.Path})
					if err != nil {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-negative", Rule: rule.ID, Claim: claim.ID, Message: "layer3 negative run failed"})
					} else if r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-negative", Rule: rule.ID, Claim: claim.ID, Message: "layer3 negative unexpectedly passed"})
					}
				}
			}
			if rule.Layer == 3 && rule.InputScope == "multi-file" && rule.Validator != "" {
				var all []string
				for _, f := range claim.Fixtures.Positive {
					all = append(all, f.Path)
				}
				for _, f := range claim.Fixtures.Negative {
					all = append(all, f.Path)
				}
				r, err := executor.RunValidator(packDir, rule.Validator, all)
				if err != nil || !r.Passed {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-multi-file", Rule: rule.ID, Claim: claim.ID, Message: "multi-file validator failed"})
				}
			}
		}
	}

	for _, tc := range pack.ToolConfig {
		if tc.ID == "" {
			continue
		}
		if err := goModTidyTempCopy(packDir); err != nil {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "go-mod-tidy", Rule: tc.ID, Message: err.Error()})
			continue
		}
		for _, claim := range tc.Claims {
			for _, f := range claim.Fixtures.Positive {
				r, err := executor.RunToolConfig(packDir, tc.Tool, tc.File, f.Path)
				if err != nil || !r.Passed {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-positive", Rule: tc.ID, Claim: claim.ID, Message: "tool_config positive failed"})
				}
			}
			for _, f := range claim.Fixtures.Negative {
				r, err := executor.RunToolConfig(packDir, tc.Tool, tc.File, f.Path)
				if err != nil {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-negative", Rule: tc.ID, Claim: claim.ID, Message: "tool_config negative failed"})
				} else if r.Passed {
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-negative", Rule: tc.ID, Claim: claim.ID, Message: "tool_config negative not triggered"})
				}
			}
		}
	}

	for _, scaffold := range pack.Content.Scaffolds {
		if scaffold.Tier == "complete" {
			renderErr := false
			for relPath, content := range scaffold.SampleConfig {
				target := filepath.Join(packDir, scaffold.Path, relPath)
				if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
					renderErr = true
					res.Errors = append(res.Errors, ValidationError{
						Phase:   res.Phase,
						Check:   "scaffold-complete-config",
						Rule:    scaffold.ID,
						Message: fmt.Sprintf("failed to render sample_config %s: %v", relPath, err),
					})
					continue
				}
				if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
					renderErr = true
					res.Errors = append(res.Errors, ValidationError{
						Phase:   res.Phase,
						Check:   "scaffold-complete-config",
						Rule:    scaffold.ID,
						Message: fmt.Sprintf("failed to render sample_config %s: %v", relPath, err),
					})
				}
			}
			if renderErr {
				continue
			}
			r, err := executor.RunScaffoldTest(packDir, scaffold.Path, scaffold.TestCommand)
			if err != nil || !r.Passed {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "scaffold-complete", Rule: scaffold.ID, Message: "complete scaffold test failed"})
			}
		}
		if scaffold.Tier == "skeleton" {
			path := filepath.Join(packDir, scaffold.Path)
			entries, err := os.ReadDir(path)
			if err != nil || len(entries) == 0 {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "scaffold-skeleton", Rule: scaffold.ID, Message: "skeleton scaffold missing structure"})
			} else {
				hasTestFunc := false
				for _, entry := range entries {
					if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
						continue
					}
					data, readErr := os.ReadFile(filepath.Join(path, entry.Name()))
					if readErr == nil && strings.Contains(string(data), "func Test") {
						hasTestFunc = true
						break
					}
				}
				if !hasTestFunc {
					res.Warnings = append(res.Warnings, ValidationWarning{
						Phase:   res.Phase,
						Check:   "scaffold-skeleton-test-names",
						Message: "skeleton scaffold has no test function names in _test.go files",
					})
				}
			}
		}
	}

	if pack.Content.SDK != nil {
		for _, provide := range pack.Content.SDK.Provides {
			if strings.TrimSpace(provide) == "" {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "sdk-provides", Message: "sdk provides missing entry"})
			}
		}
	}

	if len(res.Errors) > 0 {
		res.Status = "fail"
	}
	return res
}

func goModTidyTempCopy(packDir string) error {
	tmp, err := os.MkdirTemp("", "packval-tidy-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmp)
	if err := copyDir(packDir, tmp); err != nil {
		return err
	}
	cmd := exec.Command("go", "mod", "tidy")
	cmd.Dir = tmp
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("go mod tidy failed: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode())
	})
}

// semgrepFileContainsRuleID parses a semgrep YAML rule file and checks
// whether any rule entry has an id field matching the given ruleID.
func semgrepFileContainsRuleID(data []byte, ruleID string) bool {
	var doc struct {
		Rules []struct {
			ID string `yaml:"id"`
		} `yaml:"rules"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return false
	}
	for _, r := range doc.Rules {
		if r.ID == ruleID {
			return true
		}
	}
	return false
}
