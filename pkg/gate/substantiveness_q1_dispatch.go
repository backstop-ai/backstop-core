package gate

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"

	"github.com/backstop-ai/backstop-core/pkg/check"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// substantiveness_q1_dispatch.go is the pkg/gate-local glue that runs a
// substantiveness pack's ast-grep rule through the REAL ast-grep engine path over a
// fixture directory and hands the resulting flat findings to the Phase-2 consumption
// helpers (RouteSubstantivenessFindings / IsTestHollow). It mirrors the production
// dispatch (cmd/backstop dispatchPackEngines → runFindingsEngine): run the engine,
// pipe its native JSON through the pack-relative stdin→SARIF convert script, parse the
// SARIF via check.ParsePackFindings, and namespace each finding's rule ID
// (pack.NamespacedRuleID). It adds NO language/tool knowledge to the gate — the engine
// command, the rule, and the convert script all come from the pack; the gate only runs
// what the pack declares and consumes SARIF. Used by the Phase-3 integration tests to
// assert Go and TS RED/GREEN polarity through the genuine engine path (not a stub), and
// by the Phase-4 strangler-equivalence harness.

// dispatchAstGrepRule runs the pack's ast-grep rule over scanTarget through the real
// ast-grep binary + the pack's convert script, returning the flat, namespaced
// []Violation the gate's routing helpers consume. packDir is the pack root (holding the
// ast-grep/ rule + convert script); rulePath/convert resolve relative to it; ruleID is
// the bare pack rule ID and packName the normalized pack name, combined into the
// namespaced rule ID exactly as production does.
func dispatchAstGrepRule(packDir, rulePath, ruleID, packName, scanTarget string) ([]Violation, error) {
	rule := filepath.Join(packDir, filepath.FromSlash(rulePath))
	convert := filepath.Join(packDir, "ast-grep", "to-sarif.sh")

	// Run the REAL ast-grep engine: `ast-grep scan --rule <rule> --json <target>`
	// (the same command + input_mode the ast-grep EngineBinding declares).
	rawJSON, err := runStdoutCmd(packDir, "ast-grep", "scan", "--rule", rule, "--json", scanTarget)
	if err != nil {
		return nil, fmt.Errorf("running ast-grep over %s: %w", scanTarget, err)
	}

	// Pipe the engine's native JSON through the pack's stdin→SARIF convert script
	// (ast-grep emits its own JSON, so the binding declares a Convert — REQ-008).
	sarif, err := runConvert(packDir, convert, rawJSON)
	if err != nil {
		return nil, fmt.Errorf("converting ast-grep output to SARIF: %w", err)
	}

	checkViolations, err := check.ParsePackFindings(sarif)
	if err != nil {
		return nil, fmt.Errorf("parsing pack findings SARIF: %w", err)
	}

	out := make([]Violation, 0, len(checkViolations))
	for _, v := range checkViolations {
		rid := v.Rule
		if rid == "" {
			rid = ruleID
		}
		out = append(out, Violation{
			Rule: pack.NamespacedRuleID(packName, rid),
			// Canonical repo-relative File (ISSUE-046). No ProjectRoot is threaded into
			// this ast-grep dispatch; ast-grep reports repo-relative paths, so
			// NormalizePath("", …) — the idempotent Clean+ToSlash+strip-"./" subset of
			// the SINGLE helper — canonicalizes them. Any absolute path is handled by
			// the Phase-1 identity chokepoint.
			File:       NormalizePath("", v.File),
			Message:    v.Message,
			Severity:   nonEmptySeverity(v.Severity),
			SourcePack: packName,
			// Carry the pack's structured properties across (ISSUE-062): the join now
			// reads func/symbol from here rather than parsing the message.
			Properties: v.Properties,
		})
	}
	return out, nil
}

// runStdoutCmd runs name+args in dir and returns stdout only (stderr is captured
// separately so an engine banner never corrupts the JSON the converter reads). A
// findings engine exits non-zero when it reports findings, so a non-zero exit is not
// treated as fatal as long as stdout parses downstream.
func runStdoutCmd(dir, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), name, args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	_ = cmd.Run() // non-zero exit is expected when findings are reported
	return stdout.Bytes(), nil
}

// runConvert shells the pack's convert script with the engine output on stdin and
// returns its stdout (the SARIF), mirroring the production clean-stdout convert step.
func runConvert(dir, convert string, stdin []byte) ([]byte, error) {
	cmd := exec.CommandContext(context.Background(), "/bin/sh", convert)
	cmd.Dir = dir
	cmd.Stdin = bytes.NewReader(stdin)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("convert script %s failed: %w (stderr: %s)", convert, err, stderr.String())
	}
	return stdout.Bytes(), nil
}

// nonEmptySeverity defaults an empty severity to "error" (fail-loud), matching the
// production namespacing step.
func nonEmptySeverity(s string) string {
	if s == "" {
		return "error"
	}
	return s
}
