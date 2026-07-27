// @waiver:coverage_threshold:deferred:2026-10-24 upgrade success path unreachable until the scanner/remediation capability seed (BUNDLE-006 REQ-014/018) lands — production wiring assembles unavailable* by design (SPEC-055 REQ-009); remove this waiver with that seed
//
// THE TOKEN ABOVE MUST STAY ON LINE 1. coverage_threshold is a LOCATIONLESS
// dimension — its violations carry a file but no line — so the gate anchors them
// at the file's first line and scans that line alone for an annotation
// (pkg/gate/step_waiver.go:19, coverageAnnotationLine). Moved down onto
// newPackUpgradeCommand, where a reader would naturally expect it, the token
// stops adjudicating: it becomes a dangling waiver AND the coverage red returns.
//
// WHAT IT COVERS, and why no test can: seven statements — the constructor-failure
// branch and the whole success path from "Upgraded %s -> %s" to the final return.
// Both are unreachable under production assembly. newProductionUpgradeCommand
// wires unavailableScanner{}, whose ScanViolations has no success return;
// UpgradeCommand.Run calls it unconditionally and its only success return sits
// after that call, so Run can never come back nil. The assembly branch is
// unreachable for the mirror reason: NewExecGitCloner and NewPackvalValidator are
// one-line constructors that cannot return nil, so assembly cannot fail.
//
// This is the can't-comply case, not deferred effort. Reaching the code would
// take a fake injection seam, which would also dismantle the property SPEC-055
// exists to establish — that production assembles its own dependencies. Retire
// this waiver when BUNDLE-006 REQ-014/REQ-018 make a successful upgrade possible;
// the coverage will then be real rather than waived.

package main

import (
	"github.com/bmanson/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

func newPackUpgradeCommand(jsonFlag *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "upgrade [pack-ref@version]",
		Short: "Upgrade a pack to a new major version",
		Long:  "Accepts an explicit major version target, validates, scans for new violations, generates a remediation bundle, and updates backstop.yml and backstop.lock.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packRef := args[0]

			upgrade, err := newProductionUpgradeCommand()
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			result, err := upgrade.Run(packRef, distribution.UpgradeOptions{
				ProjectDir: ".",
			})
			if err != nil {
				return packLifecycleFailure(cmd.OutOrStdout(), jsonFlag, "pack upgrade", err)
			}

			// STDERR, before the success line. Rendering a warning never changes the
			// exit code: divergence and the coordinate fallback are diagnostics on an
			// otherwise-successful upgrade (SPEC-056 REQ-011).
			renderWarnings(cmd, result.Warnings)

			cmd.Printf("Upgraded %s -> %s\n", result.OldVersion, result.NewVersion)
			if result.RemediationBundle != "" {
				cmd.Printf("  remediation bundle: %s\n", result.RemediationBundle)
			}
			if result.BaselinedViolations > 0 {
				cmd.Printf("  baselined %d violations\n", result.BaselinedViolations)
			}

			return nil
		},
	}
}
