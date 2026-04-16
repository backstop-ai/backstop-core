package packval

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func RunFixtures(pack *PackManifest, packDir string, executor FixtureExecutor) *PhaseResult {
	res := &PhaseResult{Phase: "phase3-fixtures", Status: "pass"}
	if pack == nil {
		res.Status = "fail"
		res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "manifest", Message: "manifest is nil"})
		return res
	}
	if executor == nil {
		executor = &DefaultExecutor{}
	}

	for _, rule := range pack.Content.Ruleset.Rules {
		for _, claim := range rule.Claims {
			for _, f := range claim.Fixtures.Positive {
				if rule.File != "" {
					r, err := executor.RunSemgrep(packDir, rule.File, f.Path)
					if err != nil || !r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-positive", Rule: rule.ID, Claim: claim.ID, Message: "positive fixture failed"})
					}
					if !strings.Contains(r.Output, rule.ID) && r.Output != "" {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-rule-id", Rule: rule.ID, Claim: claim.ID, Message: "output missing rule id"})
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
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-negative", Rule: rule.ID, Claim: claim.ID, Message: "negative fixture not triggered", FixHint: "engine limitation possible"})
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
