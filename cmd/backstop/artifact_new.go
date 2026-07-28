package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/backstop-ai/backstop-core/pkg/scaffold"
	"github.com/spf13/cobra"
)

// ArtifactNewResult holds the output data for a successful artifact creation.
type ArtifactNewResult struct {
	ArtifactType  string `json:"artifact_type"`
	ID            string `json:"id"`
	FilePath      string `json:"file_path"`
	SchemaVersion string `json:"schema_version"`
}

// NewArtifactNewCommand creates the Cobra command for backstop artifact new
// using package-level defaults. It is a thin adapter: flag parsing, wiring,
// file I/O, and output formatting. Template rendering and ID resolution are
// delegated to pkg/scaffold.
func NewArtifactNewCommand() *cobra.Command {
	return newArtifactNewCommandWithDeps(scaffold.ArtifactNewDeps{
		Executor:    &scaffold.RealGitExecutor{},
		ProjectRoot: ".",
	})
}

// newArtifactNewCommandWithDeps creates the command with injectable dependencies
// for testing.
func newArtifactNewCommandWithDeps(deps scaffold.ArtifactNewDeps) *cobra.Command {
	var slugFlag string
	var sourceFlag string
	var jsonFlag bool

	cmd := &cobra.Command{
		Use:           "new [type]",
		Short:         "Scaffold a new artifact",
		Long:          "Creates a new backstop artifact from a template with an auto-assigned ID.",
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Validate exactly one positional arg
			if len(args) < 1 {
				return &ExitCodeError{Code: ExitConfigError, Message: "missing artifact type argument"}
			}

			artifactType := args[0]

			// Validate type
			if _, ok := scaffold.ValidArtifactTypes[artifactType]; !ok {
				return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("invalid artifact type: %q", artifactType)}
			}

			// Validate slug
			if err := scaffold.ValidateSlug(slugFlag); err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Validate --source for plan type
			if artifactType == "plan" {
				if sourceFlag == "" {
					return &ExitCodeError{Code: ExitConfigError, Message: "missing --source: required for plan type"}
				}
				if err := scaffold.ValidateSource(sourceFlag); err != nil {
					return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
				}
			}

			// Resolve ID via pkg/scaffold
			id, err := scaffold.ResolveID(artifactType, scaffold.IDOptions{
				ProjectRoot: deps.ProjectRoot,
				Executor:    deps.Executor,
				MaxRetries:  3,
			})
			if err != nil {
				if _, ok := err.(*scaffold.RetriesExhaustedError); ok {
					return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
				}
				return err
			}

			// Get today's date
			date := time.Now().Format("2006-01-02")
			if deps.DateFunc != nil {
				date = deps.DateFunc()
			}

			// Render template via pkg/scaffold
			content, err := scaffold.Scaffold(artifactType, id, slugFlag, date, sourceFlag)
			if err != nil {
				return err
			}

			// Compute target directory and filename
			targetDir := scaffold.TargetDir(artifactType, deps.ProjectRoot)
			filename := scaffold.Filename(artifactType, id, slugFlag, sourceFlag)
			filePath := filepath.Join(targetDir, filename)

			// Create target directory if needed
			if err := os.MkdirAll(targetDir, 0o755); err != nil {
				return fmt.Errorf("creating directory %s: %w", targetDir, err)
			}

			// Check if file already exists (exit 1 = conflict)
			if _, err := os.Stat(filePath); err == nil {
				return &ExitCodeError{Code: ExitViolations, Message: fmt.Sprintf("file already exists: %s", filePath)}
			}

			// Write the file
			if err := os.WriteFile(filePath, content, 0o644); err != nil {
				return fmt.Errorf("writing file %s: %w", filePath, err)
			}

			// Format output using Formatter interface from output.go
			result := ArtifactNewResult{
				ArtifactType:  artifactType,
				ID:            id,
				FilePath:      filePath,
				SchemaVersion: "cli/v1",
			}

			var formatter ArtifactNewFormatter
			if jsonFlag {
				formatter = &JSONArtifactNewFormatter{}
			} else {
				formatter = &HumanArtifactNewFormatter{}
			}
			out, err := formatter.FormatNewResult(result)
			if err != nil {
				return err
			}
			cmd.Print(out)

			return nil
		},
	}

	cmd.Flags().StringVar(&slugFlag, "slug", "", "Human-readable filename suffix")
	cmd.Flags().StringVar(&sourceFlag, "source", "", "Backing spec or issue ID (SPEC-NNN or ISSUE-NNN, required for plan type)")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output results as structured JSON")

	return cmd
}
