package main

import (
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/validate"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// version is the raw link-time build stamp, set by goreleaser via
// `-ldflags "-X main.version=..."`. It is NOT what callers should read: an
// uninjected build leaves it "dev" while build info may still name a real
// released version. Read effectiveVersion() (version.go) instead, which applies
// the injection-wins precedence and rejects VCS-synthesized pseudo-versions.
var version = "dev" // nosemgrep: go.core.no-global-mutable-state — link-time build stamp; -ldflags -X can only write a package-level var, and it is never mutated at runtime

// NewRootCommand builds the Cobra command tree with all namespaces and
// top-level commands.
func NewRootCommand() *cobra.Command {
	var jsonFlag bool

	rootCmd := &cobra.Command{
		Use:   "backstop",
		Short: "Backstop CLI — governance enforcement for software projects",
		Long: `Backstop is a governance enforcement CLI that validates artifacts,
checks code against security standards, compiles enforcement packs,
and gates releases. Every agent, runtime, and workflow interacts with
backstop by shelling out to CLI commands.`,
		SilenceErrors: true,
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
	}

	// Persistent --json flag on root command
	rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", false, "Output results as structured JSON")

	// Config loading as PersistentPreRunE on enforcement commands only
	enforcementPreRun := func(cmd *cobra.Command, _ []string) error {
		_, err := config.LoadConfig()
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		return nil
	}

	// --- Namespace: artifact ---
	artifactCmd := &cobra.Command{
		Use:   "artifact",
		Short: "Artifact lifecycle commands",
		Long:  "Commands for validating, creating, and managing backstop artifacts such as specs, plans, ADRs, and standards.",
	}

	artifactValidateCmd := NewArtifactValidateCommand()
	artifactValidateCmd.PersistentPreRunE = enforcementPreRun

	artifactNewCmd := NewArtifactNewCommand()

	artifactCmd.AddCommand(artifactValidateCmd, artifactNewCmd)

	// --- Namespace: pack ---
	packCmd := &cobra.Command{
		Use:   "pack",
		Short: "Enforcement content commands",
		Long:  "Commands for compiling, managing, and distributing enforcement packs containing rules and code standards.",
	}

	packNewCmd := NewPackNewCommand()
	packCheckCmd := newPackCheckCommand(&jsonFlag)
	packTestCmd := newPackTestCommand(&jsonFlag)

	// Distribution lifecycle commands
	packAddCmd := newPackAddCommand(&jsonFlag)
	packRemoveCmd := newPackRemoveCommand(&jsonFlag)
	packInstallCmd := newPackInstallCommand(&jsonFlag)
	packUpdateCmd := newPackUpdateCommand(&jsonFlag)
	packUpgradeCmd := newPackUpgradeCommand(&jsonFlag)
	packListCmd := newPackListCommand(&jsonFlag)
	packRelockCmd := newPackRelockCommand(&jsonFlag)

	packCmd.AddCommand(packNewCmd, packCheckCmd, packTestCmd,
		packAddCmd, packRemoveCmd, packInstallCmd, packUpdateCmd, packUpgradeCmd, packListCmd, packRelockCmd)

	// --- Top-level: gate ---
	gateCmd := newGateCommand(&jsonFlag)

	// --- Top-level: baseline ---
	baselineCmd := newBaselineCommand()

	// --- Top-level: version ---
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print version and schema cohort information",
		Long:  "Displays the CLI binary version, the embedded schema cohort identifier (derived from the set of embedded schema versions), and the Go version used to build the binary.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			schemas, err := ListSchemas()
			if err != nil {
				return fmt.Errorf("listing schemas: %w", err)
			}
			schemaCohort := computeCohortID(schemas)
			// Resolved ONCE and shared by both renderings so the JSON payload and
			// the text output cannot drift apart.
			reportedVersion := effectiveVersion()

			if jsonFlag {
				data := map[string]interface{}{
					"version":        reportedVersion,
					"schema_cohort":  schemaCohort,
					"go_version":     runtime.Version(),
					"schema_version": "cli/v1",
				}
				out, err := json.MarshalIndent(data, "", "  ")
				if err != nil {
					return err
				}
				cmd.Println(string(out))
			} else {
				cmd.Printf("backstop version %s\n", reportedVersion)
				cmd.Printf("schema cohort: %s\n", schemaCohort)
				cmd.Printf("go version: %s\n", runtime.Version())
			}
			return nil
		},
	}

	// --- Top-level: commands ---
	commandsCmd := &cobra.Command{
		Use:   "commands",
		Short: "List all available commands for agent discovery",
		Long:  "Returns a JSON array describing the full command tree. Each entry includes the command name, full path, description, and available flags. This is the agent discovery endpoint.",
		RunE: func(cmd *cobra.Command, _ []string) error {
			descriptors := buildCommandTree(rootCmd, "")
			out, err := json.MarshalIndent(descriptors, "", "  ")
			if err != nil {
				return err
			}
			cmd.Println(string(out))
			return nil
		},
	}

	// --- Top-level: waiver (read-only) ---
	waiverCmd := newWaiverCommand()

	// --- Namespace: recipe ---
	recipeCmd := newRecipeCommand()

	rootCmd.AddCommand(artifactCmd, packCmd, gateCmd, baselineCmd, versionCmd, commandsCmd, waiverCmd, recipeCmd)

	return rootCmd
}

// CommandDescriptor describes a single command in the tree.
type CommandDescriptor struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Flags       []string `json:"flags"`
}

// buildCommandTree walks the Cobra command tree and builds descriptors.
func buildCommandTree(cmd *cobra.Command, parentPath string) []CommandDescriptor {
	var descriptors []CommandDescriptor

	for _, sub := range cmd.Commands() {
		if sub.Name() == "help" || sub.Name() == "completion" {
			continue
		}
		path := sub.Name()
		if parentPath != "" {
			path = parentPath + " " + sub.Name()
		}

		var flags []string
		sub.Flags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, "--"+f.Name)
		})
		// Include inherited persistent flags
		sub.InheritedFlags().VisitAll(func(f *pflag.Flag) {
			flags = append(flags, "--"+f.Name)
		})

		descriptors = append(descriptors, CommandDescriptor{
			Name:        sub.Name(),
			Path:        path,
			Description: sub.Short,
			Flags:       flags,
		})

		// Recurse into subcommands
		if sub.HasSubCommands() {
			descriptors = append(descriptors, buildCommandTree(sub, path)...)
		}
	}

	return descriptors
}

// computeCohortID builds a cohort identifier from schema paths.
func computeCohortID(schemas []string) string {
	if len(schemas) == 0 {
		return "empty"
	}
	// Use count and first/last schema as a lightweight cohort ID
	parts := make([]string, 0, len(schemas))
	for _, s := range schemas {
		// Extract artifact-type/version from path
		s = strings.TrimPrefix(s, "artifacts/")
		s = strings.TrimSuffix(s, "/schema.json")
		parts = append(parts, s)
	}
	return fmt.Sprintf("%d-schemas[%s]", len(schemas), strings.Join(parts, ","))
}

// outputResult formats and prints a result using the appropriate formatter.
func outputResult(cmd *cobra.Command, jsonFlag *bool, result validate.ValidationResult) error {
	if jsonFlag != nil && *jsonFlag {
		f := &JSONFormatter{}
		out, err := f.FormatResult(result)
		if err != nil {
			return err
		}
		cmd.Println(out)
	} else {
		f := &HumanFormatter{}
		out, err := f.FormatResult(result)
		if err != nil {
			return err
		}
		cmd.Print(out)
	}
	return nil
}
