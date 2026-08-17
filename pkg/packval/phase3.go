package packval

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/pack/engine"
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
		var binding engine.EngineBinding
		haveBinding := false
		// ruleSource is the rule's declared source file, read through the SINGLE
		// accessor that decides `rule_path` (canonical, what every real pack writes)
		// over the `file` back-compat alias (ISSUE-092).
		ruleSource := rule.RuleSourcePath()
		if ruleSource != "" {
			// The resolve runs FIRST because the rule-ID cross-check below is
			// conditioned on the resolved binding's DECLARED input mode, which is not
			// in hand until it has run. An unknown engine fails loud, naming it — never
			// a silent skip (ISSUE-019).
			b, resErr := resolveEngine(pack, rule.Engine)
			if resErr != nil {
				res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "engine-resolve", Rule: rule.ID, Message: resErr.Error()})
			} else {
				binding = b
				haveBinding = true
			}
			// THE RULE-ID CROSS-CHECK IS CONDITIONED ON THE DECLARED INPUT MODE, NEVER
			// ON THE ENGINE'S NAME (CLM-010). semgrepFileContainsRuleID assumes the
			// declared file IS A LIST OF RULES each carrying an `id` — precisely the
			// semantics of `input_mode: rule-flags` ("repeat input_flag once per RULE
			// FILE"). Under `config-file` the declared file is a PROJECT CONFIG naming
			// rule DIRECTORIES and carrying no ids, so the check is categorically
			// inapplicable rather than merely failing.
			//
			// ⚠ DO NOT rewrite this as an equality test between rule.Engine and a
			// hardcoded engine name. (The literal is deliberately not spelled out
			// here: TestExecutor_ToolConfigResolvesViaBindingNotSwitch greps this file
			// for baked tool names and does not distinguish prose from code.) A baked
			// engine-name literal violates the thin-executor first principle, AND it
			// would silently
			// un-fail cmd/backstop/testdata/hermetic-remote/fixture-fail-pack, whose
			// pack-declared `marker-scan` engine plus a CLAIMLESS rule make the rule-ID
			// mismatch its ONLY failure mechanism. marker-scan declares rule-flags, so
			// under this condition it still fails.
			//
			// The rule-source READ moves inside the same condition: file EXISTENCE is
			// phase 1's job for every engine, so nothing is lost, and a config-file
			// engine's project config is no longer read by a phase that cannot
			// interpret it.
			if haveBinding && binding.InputMode == engine.InputModeRuleFlags {
				ruleFilePath := filepath.Join(packDir, ruleSource)
				ruleData, err := os.ReadFile(ruleFilePath)
				if err != nil {
					res.Errors = append(res.Errors, ValidationError{
						Phase:   res.Phase,
						Check:   "semgrep-rule-id",
						Rule:    rule.ID,
						Message: fmt.Sprintf("failed to read rule file %s: %v", ruleSource, err),
					})
				} else if !semgrepFileContainsRuleID(ruleData, rule.ID) {
					res.Errors = append(res.Errors, ValidationError{
						Phase:   res.Phase,
						Check:   "semgrep-rule-id",
						Rule:    rule.ID,
						Message: fmt.Sprintf("pack rule ID %q not found in rule file %s", rule.ID, ruleSource),
					})
				}
			}
		}
		for _, claim := range rule.Claims {
			for _, f := range claim.Fixtures.Positive {
				if ruleSource != "" && haveBinding {
					// FINDINGS SEAM. RunEngine's Passed means "the engine FIRED"
					// (produced findings) — not "the fixture is acceptable". A positive
					// fixture is the CLEAN example, so a finding on it is a FALSE
					// POSITIVE and therefore a failure (BUNDLE-005 REQ-011).
					r, err := executor.RunEngine(packDir, binding, []string{ruleSource, f.Path})
					switch {
					case err != nil:
						res.Errors = append(res.Errors, engineError(res.Phase, rule.ID, claim.ID, f.Path, err))
					case r.Passed:
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "semgrep-positive", Rule: rule.ID, Claim: claim.ID, Message: "positive fixture triggered the rule (false positive)"})
					}
				}
				// VALIDATOR SEAM — DELIBERATELY THE OPPOSITE SHAPE, DO NOT "HARMONIZE".
				// RunValidator's Passed means "the validator EXITED ZERO", not "it
				// fired". So a positive fixture SHOULD pass here and a negative SHOULD
				// fail — the inverse of the findings seam above. Applying ONE
				// conditional shape to two seams whose Passed means opposite things is
				// exactly what produced ISSUE-092; these branches were always correct
				// and TestPackVal_P3_ValidatorPolarityUnchangedByFindingsFix pins them.
				if rule.Validator != "" {
					r, err := executor.RunValidator(packDir, rule.Validator, []string{f.Path})
					if err != nil || !r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-positive", Rule: rule.ID, Claim: claim.ID, Message: "layer3 positive failed"})
					}
				}
			}
			for _, f := range claim.Fixtures.Negative {
				if ruleSource != "" && haveBinding {
					r, err := executor.RunEngine(packDir, binding, []string{ruleSource, f.Path})
					if err != nil {
						res.Errors = append(res.Errors, engineError(res.Phase, rule.ID, claim.ID, f.Path, err))
					} else if !r.Passed {
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
				if rule.Validator != "" {
					r, err := executor.RunValidator(packDir, rule.Validator, []string{f.Path})
					if err != nil {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-negative", Rule: rule.ID, Claim: claim.ID, Message: "layer3 negative run failed"})
					} else if r.Passed {
						res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "validator-negative", Rule: rule.ID, Claim: claim.ID, Message: "layer3 negative unexpectedly passed"})
					}
				}
			}
			if rule.InputScope == "multi-file" && rule.Validator != "" {
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
		// A pack that needs setup before fixture execution declares it (e.g. on its
		// engine binding); packval bakes NO Go module-tidy pre-flight (ISSUE-019).
		binding, resErr := resolveEngine(pack, tc.Engine)
		if resErr != nil {
			res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "engine-resolve", Rule: tc.ID, Message: resErr.Error()})
			continue
		}
		for _, claim := range tc.Claims {
			// Same FINDINGS SEAM contract as the ruleset loop above: a tool_config
			// positive fixture that FIRES is a false positive.
			for _, f := range claim.Fixtures.Positive {
				r, err := executor.RunEngine(packDir, binding, []string{tc.File, f.Path})
				switch {
				case err != nil:
					res.Errors = append(res.Errors, engineError(res.Phase, tc.ID, claim.ID, f.Path, err))
				case r.Passed:
					res.Errors = append(res.Errors, ValidationError{Phase: res.Phase, Check: "tool-config-positive", Rule: tc.ID, Claim: claim.ID, Message: "tool_config positive triggered the rule (false positive)"})
				}
			}
			for _, f := range claim.Fixtures.Negative {
				r, err := executor.RunEngine(packDir, binding, []string{tc.File, f.Path})
				if err != nil {
					res.Errors = append(res.Errors, engineError(res.Phase, tc.ID, claim.ID, f.Path, err))
				} else if !r.Passed {
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
			} else if scaffold.TestIndicator != "" {
				// Language-neutral: a skeleton carries test structure if ANY of its
				// files contains the pack-DECLARED indicator (ISSUE-019). No file-suffix
				// or function-name convention is baked in.
				hasIndicator := false
				for _, entry := range entries {
					if entry.IsDir() {
						continue
					}
					data, readErr := os.ReadFile(filepath.Join(path, entry.Name()))
					if readErr == nil && strings.Contains(string(data), scaffold.TestIndicator) {
						hasIndicator = true
						break
					}
				}
				if !hasIndicator {
					res.Warnings = append(res.Warnings, ValidationWarning{
						Phase:   res.Phase,
						Check:   "scaffold-skeleton-test-indicator",
						Message: "skeleton scaffold has no file containing the pack-declared test indicator",
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

// engineError builds the phase-3 error for a fixture whose engine run FAILED, as
// opposed to one whose engine ran and produced a verdict (CLM-005).
//
// It is deliberately its OWN Check. A run that errored produced no usable answer, so
// it has no fixture verdict to report: folding it into "positive fixture ..." sends a
// pack author to inspect a fixture that was never the problem, and — far worse —
// absorbing it into a passing negative is a vacuous green, because a missing or broken
// engine would then look exactly like a rule that correctly fired on its violation.
//
// Rule and Claim are ALWAYS populated from the rule and claim being processed. Without
// them a pack author would learn that something broke without learning what broke it,
// which is strictly less useful than the fixture verdict this error replaces.
func engineError(phase, ruleID, claimID, fixturePath string, err error) ValidationError {
	return ValidationError{
		Phase:   phase,
		Check:   "engine-error",
		Rule:    ruleID,
		Claim:   claimID,
		Message: fmt.Sprintf("engine run failed for fixture %s: %v", fixturePath, err),
		FixHint: "The engine did not produce a usable result, so this fixture has no verdict. This is a broken RUN, not a failing fixture — check that the declared engine command exists, is executable, and emits parseable SARIF before looking at the fixture itself.",
	}
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
