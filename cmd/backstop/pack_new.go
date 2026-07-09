package main

import (
	"encoding/json"
	"fmt"

	"github.com/bmanson/backstop-core/pkg/pack"
	"github.com/spf13/cobra"
)

// NewPackNewCommand creates the Cobra command for backstop pack new.
// It delegates all scaffolding logic to pkg/pack.
func NewPackNewCommand() *cobra.Command {
	return newPackNewCommandWithRoot(".")
}

// newPackNewCommandWithRoot creates the command with an injectable project root
// for testing.
func newPackNewCommandWithRoot(projectRoot string) *cobra.Command {
	var (
		typeFlag     string
		languageFlag string
		slugFlag     string
		jsonFlag     bool
	)

	cmd := &cobra.Command{
		Use:           "new",
		Short:         "Scaffold a new enforcement pack",
		Long:          "Creates a new engine, mechanism, or toolchain pack: a valid pack.yml with a declared engines: block and a sample rule that passes pack check, pack test, and the gate.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Validate --type
			if typeFlag == "" {
				return &ExitCodeError{Code: ExitConfigError, Message: "missing required flag: --type"}
			}
			if !pack.ValidPackTypes[typeFlag] {
				return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("invalid pack type: %q (must be engine, mechanism, or toolchain)", typeFlag)}
			}

			// Validate --language
			if languageFlag == "" {
				return &ExitCodeError{Code: ExitConfigError, Message: "missing required flag: --language"}
			}
			if err := pack.ValidateLanguage(languageFlag); err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Validate --slug
			if slugFlag == "" {
				return &ExitCodeError{Code: ExitConfigError, Message: "missing required flag: --slug"}
			}
			if err := pack.ValidateSlug(slugFlag); err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Delegate scaffolding to pkg/pack. Engine packs are named by slug, not
			// auto-numbered — the retired ResolvePackNumber scan of standards/ is gone
			// (ISSUE-032 Defect A / ISSUE-030 fold).
			result, err := pack.ScaffoldPack(pack.ScaffoldOptions{
				Type:        typeFlag,
				Language:    languageFlag,
				Slug:        slugFlag,
				ProjectRoot: projectRoot,
			})
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Format output
			if jsonFlag {
				data, jsonErr := json.MarshalIndent(result, "", "  ")
				if jsonErr != nil {
					return fmt.Errorf("formatting JSON output: %w", jsonErr)
				}
				cmd.Println(string(data))
			} else {
				cmd.Print(result.HumanString())
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&typeFlag, "type", "", "Pack type: engine, mechanism, or toolchain")
	cmd.Flags().StringVar(&languageFlag, "language", "", "Target language (e.g., go, typescript)")
	cmd.Flags().StringVar(&slugFlag, "slug", "", "Human-readable pack name (kebab-case, 2-64 chars)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results as structured JSON")

	return cmd
}
