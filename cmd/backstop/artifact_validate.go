package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmanson/backstop-core/pkg/artifact"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/bmanson/backstop-core/pkg/validate"
	"github.com/spf13/cobra"
)

// ValidateConfig holds parsed flags and configuration for artifact validation.
type ValidateConfig struct {
	ProjectRoot string            // Project root directory (from backstop.yml location)
	TypeFilters map[string]string // Type name → optional artifact ID (empty string = all of type)
	All         bool              // --all flag: validate everything
	JSONOutput  bool              // --json flag
	SchemaFS    fs.FS             // Embedded schema filesystem
}

// ValidateResult holds the aggregated result of validating artifacts.
type ValidateResult struct {
	Pass            bool
	ViolationsCount int
	Violations      []validate.Violation
	ArtifactsFound  int // Number of artifacts discovered and processed
}

// ExitCodeError wraps an error with an exit code for CLI exit status handling.
type ExitCodeError struct {
	Code    int
	Message string
}

func (e *ExitCodeError) Error() string { return e.Message }

// idFieldForType returns the metadata field name used for ID matching
// per artifact type. Plans use plan_id; all others use number.
func idFieldForType(artType string) string {
	if artType == "plan" {
		return "plan_id"
	}
	return "number"
}

// loadSchemaFromFS reads schema files from an fs.FS and calls the existing
// schema loading functions. It extracts the needed files to a temp directory
// to bridge between the embedded FS and the file-path-based schema API.
func loadSchemaFromFS(fsys fs.FS, schemaRelPath string) (*schema.Schema, error) {
	tmpDir, err := os.MkdirTemp("", "backstop-schema-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir for schema loading: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	// Copy the extension schema from the embedded FS
	if err := copyFSFile(fsys, schemaRelPath, tmpDir); err != nil {
		return nil, fmt.Errorf("reading schema from embed: %w", err)
	}

	// Copy the base schema if it exists (needed for extends resolution)
	basePath := filepath.Join("artifacts", "base", "schema.json")
	_ = copyFSFile(fsys, basePath, tmpDir) // Ignore error — base may not exist

	fullPath := filepath.Join(tmpDir, schemaRelPath)
	artifactsRoot := filepath.Join(tmpDir, "artifacts")
	return schema.LoadArtifactSchema(fullPath, artifactsRoot)
}

// copyFSFile copies a single file from an fs.FS to a destination directory,
// preserving the relative path structure.
func copyFSFile(fsys fs.FS, relPath, destRoot string) error {
	data, err := fs.ReadFile(fsys, relPath)
	if err != nil {
		return err
	}
	destPath := filepath.Join(destRoot, relPath)
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destPath, data, 0o644)
}

// ValidateArtifacts orchestrates discovery, routing, validation, and aggregation.
// It returns a ValidateResult with aggregated violations, or an error for
// config-level failures (exit code 2).
func ValidateArtifacts(cfg ValidateConfig) (ValidateResult, error) {
	// Determine type filters
	var typeFilters []string
	if !cfg.All && len(cfg.TypeFilters) > 0 {
		for t := range cfg.TypeFilters {
			typeFilters = append(typeFilters, t)
		}
	}
	// If All is set or no filters, discover all types (typeFilters = nil)

	// Discover artifacts
	discovered, err := DiscoverArtifacts(cfg.ProjectRoot, typeFilters)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("discovering artifacts: %w", err)
	}

	// Zero artifacts: return empty result (caller handles warning)
	if len(discovered) == 0 {
		return ValidateResult{Pass: true}, nil
	}

	var allViolations []validate.Violation
	artifactsProcessed := 0

	for _, da := range discovered {
		// Parse the artifact
		art, err := artifact.ParseFile(da.Path)
		if err != nil {
			return ValidateResult{}, fmt.Errorf("failed to parse artifact %s: %w", da.Path, err)
		}

		// If ID filter is set for this type, match by per-type metadata field
		if cfg.TypeFilters != nil {
			if idFilter, hasType := cfg.TypeFilters[da.Type]; hasType && idFilter != "" {
				field := idFieldForType(da.Type)
				if art.Metadata[field] != idFilter {
					continue // Skip non-matching artifacts
				}
			}
		}

		// Route to the correct validator.
		// Plans are pure YAML without schema_version — route by discovery type.
		// All other artifacts route by schema_version metadata (REQ-001).
		var validatorFn ValidatorFunc
		if da.Type == "plan" {
			validatorFn = validatorRouter["plan"]
		} else {
			var routeErr error
			validatorFn, routeErr = RouteValidator(art)
			if routeErr != nil {
				return ValidateResult{}, fmt.Errorf("routing artifact %s: %w", da.Path, routeErr)
			}
		}

		// Load schema from embedded FS (plans don't use schemas)
		var sch *schema.Schema
		if da.Type != "plan" {
			schemaPath, err := schema.ResolveSchemaPath(art)
			if err != nil {
				return ValidateResult{}, fmt.Errorf("resolving schema for %s: %w", da.Path, err)
			}
			sch, err = loadSchemaFromFS(cfg.SchemaFS, schemaPath)
			if err != nil {
				return ValidateResult{}, fmt.Errorf("loading schema for %s: %w", da.Path, err)
			}
		}

		// Run validation
		result := validatorFn(art, sch)
		allViolations = append(allViolations, result.Violations...)
		artifactsProcessed++
	}

	// Check if ID scoping found zero matches (after filtering)
	if artifactsProcessed == 0 && cfg.TypeFilters != nil {
		for artType, id := range cfg.TypeFilters {
			if id != "" {
				return ValidateResult{}, fmt.Errorf("no %s artifact found with ID %s", artType, id)
			}
		}
	}

	return ValidateResult{
		Pass:            len(allViolations) == 0,
		ViolationsCount: len(allViolations),
		Violations:      allViolations,
		ArtifactsFound:  artifactsProcessed,
	}, nil
}

// NewArtifactValidateCommand creates the Cobra command for backstop artifact validate.
func NewArtifactValidateCommand() *cobra.Command {
	var (
		specFlag     string
		planFlag     string
		adrFlag      string
		bundleFlag   string
		issueFlag    string
		standardFlag string
		allFlag      bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate artifacts against schemas",
		Long: `Validates backstop artifact files against their schema definitions.
Discovers artifacts in the project directory and validates each one against
the appropriate type-specific schema. Supports scoping by type and artifact ID.

Exit codes: 0 (all pass), 1 (violations found), 2 (config error).`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Get the json flag from the root command
			jsonFlag, _ := cmd.Flags().GetBool("json")

			// Determine project root from current directory (backstop.yml location)
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}

			// Build type filters from flags
			typeFilters := make(map[string]string)
			flagSet := false

			if cmd.Flags().Changed("spec") {
				typeFilters["spec"] = specFlag
				flagSet = true
			}
			if cmd.Flags().Changed("plan") {
				typeFilters["plan"] = planFlag
				flagSet = true
			}
			if cmd.Flags().Changed("adr") {
				typeFilters["adr"] = adrFlag
				flagSet = true
			}
			if cmd.Flags().Changed("bundle") {
				typeFilters["bundle"] = bundleFlag
				flagSet = true
			}
			if cmd.Flags().Changed("issue") {
				typeFilters["issue"] = issueFlag
				flagSet = true
			}
			if cmd.Flags().Changed("standard") {
				typeFilters["standard"] = standardFlag
				flagSet = true
			}

			cfg := ValidateConfig{
				ProjectRoot: cwd,
				All:         allFlag,
				JSONOutput:  jsonFlag,
				SchemaFS:    SchemaFS,
			}

			if flagSet && !allFlag {
				cfg.TypeFilters = typeFilters
			}

			result, err := ValidateArtifacts(cfg)
			if err != nil {
				// Config error — exit 2
				if jsonFlag {
					configResult := validate.ValidationResult{
						Violations: []validate.Violation{{
							Rule:     "config-error",
							Message:  err.Error(),
							Severity: "error",
						}},
					}
					_ = outputResult(cmd, &jsonFlag, configResult)
				} else {
					cmd.PrintErrln("Error:", err)
				}
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Zero artifacts warning
			if result.ArtifactsFound == 0 && result.Pass {
				cmd.PrintErrln("warning: no artifacts found to validate")
			}

			// Convert to validate.ValidationResult for output formatting
			valResult := validate.ValidationResult{
				Violations: result.Violations,
			}

			if err := outputResult(cmd, &jsonFlag, valResult); err != nil {
				return err
			}

			// Signal violations via exit code 1
			if !result.Pass {
				return &ExitCodeError{Code: ExitViolations, Message: "violations found"}
			}

			return nil
		},
	}

	// Type-scoping flags with optional ID arguments
	cmd.Flags().StringVar(&specFlag, "spec", "", "Validate spec artifacts (optional ID)")
	cmd.Flags().StringVar(&planFlag, "plan", "", "Validate plan artifacts (optional ID)")
	cmd.Flags().StringVar(&adrFlag, "adr", "", "Validate ADR artifacts (optional ID)")
	cmd.Flags().StringVar(&bundleFlag, "bundle", "", "Validate bundle artifacts (optional ID)")
	cmd.Flags().StringVar(&issueFlag, "issue", "", "Validate issue artifacts (optional ID)")
	cmd.Flags().StringVar(&standardFlag, "standard", "", "Validate standard artifacts (optional ID)")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Validate all artifacts (takes precedence over type flags)")

	return cmd
}
