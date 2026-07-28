package main

import (
	"github.com/backstop-ai/backstop-core/pkg/pack/distribution"
	"github.com/spf13/cobra"
)

// newPackRelockCommand creates `backstop pack relock <path>` (ISSUE-032 Defect F /
// CLM-010): it refreshes a locally-edited pack's backstop.lock entry by recomputing its
// content hash over the installed pack dir — the clean alternative to remove+add.
func newPackRelockCommand(_ *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relock [path]",
		Short: "Refresh a local pack's lock entry after editing it in place",
		Long:  "Re-reads a locally-installed pack, recomputes its content hash, and overwrites its backstop.lock entry — no remove+add. Only local-source packs are relockable; git packs update through pack update/upgrade.",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := distribution.Relock(".", args[0])
			if err != nil {
				return &ExitCodeError{Code: ExitViolations, Message: err.Error()}
			}
			cmd.Printf("Relocked %s (hash: %s)\n", result.PackName, result.ContentHash)
			return nil
		},
	}
	return cmd
}
