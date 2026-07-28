package main

import (
	"fmt"
	"io"

	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

// formatAddedLine renders the pack-add success line. For a versionless (local) pack the
// "@version" slot is omitted so there is no bare trailing `@`; a versioned pack renders
// `<name>@<version>` as before.
func formatAddedLine(name, version, hash string) string {
	if version == "" {
		return fmt.Sprintf("Added %s (hash: %s)", name, hash)
	}
	return fmt.Sprintf("Added %s@%s (hash: %s)", name, version, hash)
}

// writeWarning writes one "warning: <message>" line to w and reports whether it landed.
//
// It mirrors writeDiagnostic (main.go:60) and exists for the same reason: the four pack
// lifecycle commands each render result warnings, and keeping the write CHECKED in one
// documented place is what stops the error being silently discarded at four call sites.
// No caller can act on the answer — a warning is advisory and the command is mid-success
// — but discarding it unexamined at each site is what the standard forbids.
//
// The "warning: " prefix is load-bearing: existing tests and consumer greps match on that
// lowercase word.
func writeWarning(w io.Writer, message string) bool {
	_, err := fmt.Fprintf(w, "warning: %s\n", message)
	return err == nil
}

// renderWarnings writes every warning a lifecycle result carried, to STDERR. Rendering a
// warning NEVER changes an exit code (SPEC-056 REQ-011).
func renderWarnings(cmd *cobra.Command, warnings []string) {
	for _, w := range warnings {
		writeWarning(cmd.ErrOrStderr(), w)
	}
}

func newPackAddCommand(jsonFlag *bool) *cobra.Command {
	var versionFlag string

	cmd := &cobra.Command{
		Use:   "add [pack-ref]",
		Short: "Add an enforcement pack to the project",
		Long:  "Resolves, clones, validates, installs, merges config, and locks an enforcement pack. Accepts org/pack-name@version for git packs or a local filesystem path.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			packRef := args[0]

			add, err := newProductionAddCommand()
			if err != nil {
				// A command that cannot be assembled is a mis-built binary, not a
				// finding about the operator's project, so it exits as a
				// configuration error rather than as a violation.
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			result, err := add.Run(packRef, distribution.AddOptions{
				ProjectDir: ".",
				Version:    versionFlag,
			})
			if err != nil {
				return packLifecycleFailure(cmd.OutOrStdout(), jsonFlag, "pack add", err)
			}

			// Diagnostics first, on STDERR, before any success line reaches stdout.
			// Rendering one never changes the exit code — divergence is loud, not fatal.
			renderWarnings(cmd, result.Warnings)

			if result.AlreadyCurrent {
				cmd.Printf("Pack %s is already installed and up to date\n", result.PackName)
				return nil
			}

			cmd.Println(formatAddedLine(result.PackName, result.Version, result.ContentHash))
			return nil
		},
	}

	cmd.Flags().StringVar(&versionFlag, "version", "", "Version to install (overrides version in pack reference)")

	return cmd
}
