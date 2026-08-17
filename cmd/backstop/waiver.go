package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/waiver"
	"github.com/spf13/cobra"
)

// waiverCmd is the read-only `backstop waiver` parent command (SPEC-049 REQ-011).
var waiverCmd *cobra.Command

// waiverListCmd is `backstop waiver list` — active / expiring-soon / unused.
var waiverListCmd *cobra.Command

// newWaiverCommand builds the read-only `backstop waiver` command tree. Core is
// READ-ONLY with respect to waivers: it NEVER writes or inserts a @waiver token —
// inserting a comment requires language comment-syntax, which is baked-language
// knowledge that belongs to the human or the runtime agent (CLM-049).
func newWaiverCommand() *cobra.Command {
	parent := &cobra.Command{
		Use:   "waiver",
		Short: "Inspect backstop waivers (read-only)",
		Long: `Read-only inspection of backstop waivers. Core backstop never writes or
inserts a @waiver token — authoring and re-certification belong to the human or
the runtime agent. Use ` + "`backstop waiver list`" + ` to see active,
expiring-soon, and unused/dangling waivers over the current scope.`,
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	list := &cobra.Command{
		Use:   "list",
		Short: "List active, expiring-soon, and unused/dangling waivers",
		Long: `List the waivers backstop adjudicates over the current scope, grouped into
active, expiring-soon, and unused/dangling sets. Read-only: this never writes or
edits a waiver token — authoring is the human's or the runtime agent's job.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE:          runWaiverList,
	}
	waiverListCmd = list
	waiverCmd = parent
	waiverCmd.AddCommand(waiverListCmd)
	return waiverCmd
}

// runWaiverList is the read-only RunE handler. It adjudicates waivers over the
// current scope and prints the active, expiring-soon, and unused/dangling sets. It
// NEVER writes a token (REQ-011).
func runWaiverList(cmd *cobra.Command, _ []string) error {
	cfgPath, discoverErr := config.DiscoverConfigPath()
	projectRoot := "."
	// declaredRoot stays empty ONLY on the no-config fallback below, where the project
	// genuinely has no configured root to read. On the configured path it is read from
	// the config — writing a literal "" there would leave `waiver list` adjudicating
	// over a step list that read a `<project>/specs` that does not exist.
	declaredRoot := ""
	if discoverErr == nil {
		projectRoot = filepath.Dir(cfgPath)
		// The PATH-SPECIFIC loader, not config.LoadConfig(): the path is already
		// discovered here, and LoadConfig would only redo that same walk.
		cfg, loadErr := config.LoadConfigFromPath(cfgPath)
		if loadErr != nil {
			return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", loadErr)}
		}
		declaredRoot = cfg.ArtifactRoot
	}
	// ResolveRoot absolutizes its projectRoot, which is what makes the "." fallback safe.
	artifactRoot, rootErr := artifact.ResolveRoot(projectRoot, declaredRoot)
	if rootErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", rootErr)}
	}
	scope, scopeErr := gate.ComputeGateScope(projectRoot, gate.GateScopeModeAll, nil)
	if scopeErr != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", scopeErr)}
	}
	res, err := projectWaiverResult(projectRoot, artifactRoot, scope)
	if err != nil {
		return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", err)}
	}
	cmd.Print(formatWaiverList(res, time.Now()))
	return nil
}

// projectWaiverResult adjudicates the current project's waivable-surface findings
// and returns the full waiver.Result (active / expiring / unused / …). It collects
// the RAW pack_engines + test_substantiveness findings from the assembled gate
// steps (BEFORE any waiver subtraction), then runs the pure pkg/waiver.Adjudicate
// with the production LineReader + Policy. It never mutates source.
func projectWaiverResult(projectRoot string, root artifact.Root, scope *gate.GateScope) (waiver.Result, error) {
	policy, err := buildWaiverPolicy(projectRoot)
	if err != nil {
		return waiver.Result{}, fmt.Errorf("building waiver policy: %w", err)
	}
	reader := buildWaiverLineReader(projectRoot, scope)

	var findings []waiver.Finding
	for _, step := range buildGateSteps(projectRoot, root, scope) {
		result := step(context.Background())
		if result.StepName != gate.StepPackEngines && result.StepName != gate.StepTestSubstantiveness {
			continue
		}
		for _, v := range result.Violations {
			findings = append(findings, waiver.Finding{RuleID: v.Rule, File: v.File, Line: v.Line, Severity: v.Severity})
		}
	}
	res := waiver.Adjudicate(findings, reader, policy, time.Now())
	// The TREE-DRIVEN unbound scan (ISSUE-097), attached onto the adjudication result.
	// Adjudicate cannot report this class: it reads only the association window of a
	// finding it was handed, so a token whose rule no longer fires is never harvested
	// and cannot even reach the Unused bucket.
	res.Unbound = waiver.Unbound(
		harvestProjectWaiverTokens(projectRoot, root),
		lockedPackNamespaces(projectRoot),
	)
	return res, nil
}

// formatWaiverList renders the read-only waiver report: the active set, the
// expiring-soon subset, and the unused/dangling set. Sections are always labeled
// so the report is never silent about a category (REQ-011 / REQ-012 flavor).
func formatWaiverList(res waiver.Result, _ time.Time) string {
	var b strings.Builder
	b.WriteString("Waivers\n")

	fmt.Fprintf(&b, "Active (%d):\n", len(res.Active))
	for _, w := range res.Active {
		fmt.Fprintf(&b, "  - %s (%s, expires %s)\n", w.RuleID, w.Reason, w.Expiry.Format("2006-01-02"))
	}

	fmt.Fprintf(&b, "Expiring soon (%d):\n", len(res.Expiring))
	for _, w := range res.Expiring {
		fmt.Fprintf(&b, "  - %s (expires %s)\n", w.RuleID, w.Expiry.Format("2006-01-02"))
	}

	fmt.Fprintf(&b, "Unused / dangling (%d):\n", len(res.Unused))
	for _, w := range res.Unused {
		fmt.Fprintf(&b, "  - %s\n", w.RuleID)
	}

	// Always labeled even at zero, like the three sections above. A section that
	// disappears when empty is how this class of rot hid in the first place — and each
	// entry carries file:line because the reader's next act is to go edit that token.
	fmt.Fprintf(&b, "Unbound / unknown pack (%d):\n", len(res.Unbound))
	for _, d := range res.Unbound {
		fmt.Fprintf(&b, "  - %s (%s:%d)\n", d.RuleID, d.File, d.Line)
	}

	return b.String()
}
