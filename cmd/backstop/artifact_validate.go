package main

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/config"
	"github.com/backstop-ai/backstop-core/pkg/schema"
	"github.com/backstop-ai/backstop-core/pkg/validate"
	"github.com/spf13/cobra"
)

// ValidateConfig holds parsed flags and configuration for artifact validation.
// @waiver:backstop-ai/go-standards/backstop.packs.backstop-ai.go-standards.rules.core.go.core.error-type-suffix:false-positive:2026-10-12 pack rule fix pending — ValidateConfig is not an error type; rule misfires on a non-error struct
type ValidateConfig struct {
	ProjectRoot string            // Project root directory (from backstop.yml location)
	Root        artifact.Root     // RESOLVED artifact root — where the corpus lives
	TypeFilters map[string]string // Type name → optional artifact ID (empty string = all of type)
	All         bool              // --all flag: validate everything
	JSONOutput  bool              // --json flag
	SchemaFS    fs.FS             // Embedded schema filesystem
	// Cohort is the content-derived identity of the schema set this run asserts
	// against. It TRAVELS WITH THE CONFIG rather than being re-derived inside, which
	// is what makes the CLI and the gate incapable of asserting against different
	// values. A zero Cohort disables the assertion — the pre-SPEC-068 semantics, kept
	// reachable so the guard can be shown to be additive.
	Cohort schema.Cohort
	// NonCorpus is the artifact-corpus exclusion set: the tool-agnostic base core
	// carries unioned with the dependency directory names installed packs declare via
	// classification.dependency_dirs (ISSUE-122). Like Cohort above it TRAVELS WITH
	// THE CONFIG rather than being re-derived inside, which is what makes the CLI and
	// the gate incapable of scanning different corpora. Its zero value still excludes
	// the tool-agnostic base, so a caller that does not wire the packs degrades to
	// today-minus-declarations rather than walking `.git`.
	NonCorpus artifact.NonCorpusDirs
}

// ArtifactValidationRecord is what one artifact's validation ASSERTED: which file, of
// which type, against which schema identity.
//
// It exists because REQ-003 binds an identity to EACH validated artifact. A flat
// per-SCHEMA list (which is what the gate result carries) cannot express that binding —
// collapsing the two would silently weaken the claim.
type ArtifactValidationRecord struct {
	Path           string
	Type           string
	SchemaVersion  string
	SchemaIdentity string
	// Schemaless marks an artifact that declares no schema_version at all — a plan,
	// which routes by discovery type. It is recorded AS schema-less rather than left
	// with an empty identity that would read as covered-with-no-content.
	Schemaless bool
}

// ValidateResult holds the aggregated result of validating artifacts.
type ValidateResult struct {
	Pass            bool
	ViolationsCount int
	Violations      []validate.Violation
	ArtifactsFound  int // Number of artifacts discovered and processed
	// ArtifactsAsserted is how many artifacts this run actually asserted against the
	// cohort, and ScannedRoot is the directory it scanned. Together they are what
	// makes an EMPTY pass read as empty rather than as verified.
	ArtifactsAsserted int
	ScannedRoot       string
	Records           []ArtifactValidationRecord
}

// ExitCodeError wraps an error with an exit code for CLI exit status handling.
type ExitCodeError struct {
	Code    int
	Message string
	// Explained suppresses reportError's diagnostic line. It is an explicit opt-OUT of
	// printing, never an opt-in to it (SPEC-055 REQ-011): a command sets it only when it
	// has ALREADY written structured findings the consumer can read, so the human line
	// would merely duplicate them. Only gate (for its ExitViolations verdict, so the
	// exit-2 message keeps printing), pack check, pack test, and artifact validate
	// qualify. Everything else leaves it zero and therefore prints — a command added
	// later that forgets to declare itself explained produces a duplicated diagnostic,
	// which a reviewer notices, rather than a silent failure, which nobody does.
	Explained bool
}

func (e *ExitCodeError) Error() string { return e.Message }

// artifactIDValue returns the identifying value used for --<type> ID
// filtering. Plans use the plan_id metadata key, issues keep their ID
// nested in the issue frontmatter block, and all other types use the
// top-level number metadata key.
func artifactIDValue(art *artifact.ParsedArtifact, artType string) string {
	switch artType {
	case "plan":
		return art.Metadata["plan_id"]
	case "issue":
		if block, ok := art.Frontmatter["issue"].(map[string]interface{}); ok {
			if id, ok := block["id"].(string); ok {
				return id
			}
		}
		return ""
	default:
		return art.Metadata["number"]
	}
}

// loadSchemaFromFS reads schema files from an fs.FS and calls the existing
// schema loading functions. It extracts the needed files to a temp directory
// to bridge between the embedded FS and the file-path-based schema API.
func loadSchemaFromFS(fsys fs.FS, schemaRelPath string) (*schema.Schema, error) {
	tmpDir, err := os.MkdirTemp("", "backstop-schema-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir for schema loading: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Copy the extension schema from the embedded FS
	if err := copyFSFile(fsys, schemaRelPath, tmpDir); err != nil {
		return nil, fmt.Errorf("reading schema from embed: %w", err)
	}

	// Copy the base schema if it exists (needed for extends resolution). A missing
	// base is expected and non-fatal; surface any other copy failure.
	basePath := filepath.Join("artifacts", "base", "schema.json")
	if err := copyFSFile(fsys, basePath, tmpDir); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, fmt.Errorf("copying base schema: %w", err)
	}

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
	discovered, err := DiscoverArtifacts(cfg.Root, typeFilters, cfg.NonCorpus)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("discovering artifacts: %w", err)
	}

	// Zero artifacts under a root that EXISTS is a legitimate PASS: the refusal below
	// is scoped strictly to "I found something I cannot prove I can validate", never to
	// absence. Tightening this into a failure would break `backstop init`'s acceptance
	// bar from a layer away.
	//
	// It is no longer a bare `ValidateResult{Pass: true}` though — it names what it
	// scanned and how much it asserted, so an empty pass reads as EMPTY rather than as
	// VERIFIED.
	if len(discovered) == 0 {
		return ValidateResult{Pass: true, ScannedRoot: cfg.Root.Path}, nil
	}

	var allViolations []validate.Violation
	var records []ArtifactValidationRecord
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
				if artifactIDValue(art, da.Type) != idFilter {
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

		// THE COHORT ASSERTION, BEFORE ANY SCHEMA IS LOADED. DD-15 inverts this
		// codebase's usual loud-but-non-blocking default here on purpose: an artifact
		// pinned to a schema_version this binary does not carry cannot be shown valid,
		// and reporting a pass over it is the false green REQ-002 exists to remove.
		//
		// A plan declares no schema_version — it routes by discovery type — so it is
		// recorded AS schema-less rather than measured against the cohort.
		record := ArtifactValidationRecord{Path: da.Path, Type: da.Type}
		if da.Type == "plan" {
			record.Schemaless = true
		} else {
			record.SchemaVersion = declaredSchemaVersion(art)
			if cfg.Cohort.ID != "" {
				identity, covered := cfg.Cohort.SchemaIdentity(record.SchemaVersion)
				if !covered {
					// The diagnostic names all three of the artifact, the version it
					// declared, and the cohort it was measured against — an operator
					// who cannot see which cohort refused cannot act on the refusal.
					return ValidateResult{}, fmt.Errorf(
						"refusing to report a verdict for %s: it declares schema_version %q, which is not covered by this binary's schema cohort %s",
						da.Path, record.SchemaVersion, cfg.Cohort.ID)
				}
				record.SchemaIdentity = identity
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
		records = append(records, record)
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

	// Corpus resolution pass (SPEC-050 REQ-001/REQ-003). Built from a FULL-corpus
	// discovery INDEPENDENT of cfg's type filter, so a --spec-scoped run resolves
	// identically to an unscoped run and to the gate (which delegates here). Placed
	// in this shared walk — not a per-artifact validator — so a ref resolves against
	// a bundle in a different file and the CLI and gate share one verdict.
	resolutionViolations, err := buildResolutionViolations(cfg.Root, cfg.NonCorpus)
	if err != nil {
		return ValidateResult{}, fmt.Errorf("resolving supports refs: %w", err)
	}
	allViolations = append(allViolations, resolutionViolations...)

	return ValidateResult{
		Pass:              len(allViolations) == 0,
		ViolationsCount:   len(allViolations),
		Violations:        allViolations,
		ArtifactsFound:    artifactsProcessed,
		ArtifactsAsserted: artifactsProcessed,
		ScannedRoot:       cfg.Root.Path,
		Records:           records,
	}, nil
}

// declaredSchemaVersion reads the raw schema_version an artifact declares, matching
// ResolveSchemaPath's case-insensitive key lookup so the value the cohort is asked
// about is the same value the schema loader routes on. An artifact declaring none
// yields "", which the cohort reports as uncovered.
func declaredSchemaVersion(art *artifact.ParsedArtifact) string {
	for k, v := range art.Metadata {
		if strings.EqualFold(k, "schema_version") || strings.EqualFold(k, "schema-version") {
			return v
		}
	}
	return ""
}

// buildResolutionViolations runs the SPEC-050 corpus resolution pass over the FULL
// artifact corpus under the RESOLVED ARTIFACT ROOT, independent of any type-scoping
// filter.
// It discovers every bundle to build the version-log catalog, harvests supports
// refs from every spec/issue (terminal citers skipped inside the harvest), and
// resolves each ref both directions plus version-log match. Files that fail to
// parse are skipped here — the per-artifact loop (and the gate's unscoped run)
// surface parse errors authoritatively — so a scoped CLI run does not error on an
// unrelated malformed file.
func buildResolutionViolations(root artifact.Root, nonCorpus artifact.NonCorpusDirs) ([]validate.Violation, error) {
	bundleDiscovered, err := DiscoverArtifacts(root, []string{"bundle"}, nonCorpus)
	if err != nil {
		return nil, fmt.Errorf("discovering bundles for resolution catalog: %w", err)
	}
	var bundles []*artifact.ParsedArtifact
	for _, da := range bundleDiscovered {
		art, parseErr := artifact.ParseFile(da.Path)
		if parseErr != nil {
			continue
		}
		bundles = append(bundles, art)
	}

	citerDiscovered, err := DiscoverArtifacts(root, []string{"spec", "issue"}, nonCorpus)
	if err != nil {
		return nil, fmt.Errorf("discovering citers for resolution: %w", err)
	}
	var citers []*artifact.ParsedArtifact
	for _, da := range citerDiscovered {
		art, parseErr := artifact.ParseFile(da.Path)
		if parseErr != nil {
			continue
		}
		citers = append(citers, art)
	}

	catalog := validate.BuildBundleReqCatalog(bundles)
	refs := validate.CollectSupportRefs(citers)
	return validate.ResolveSupports(catalog, refs), nil
}

// NewArtifactValidateCommand creates the Cobra command for backstop artifact validate.
func NewArtifactValidateCommand() *cobra.Command {
	var (
		specFlag      string
		planFlag      string
		adrFlag       string
		bundleFlag    string
		issueFlag     string
		directiveFlag string
		allFlag       bool
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
			jsonFlag, err := cmd.Flags().GetBool("json")
			if err != nil {
				return fmt.Errorf("reading --json flag: %w", err)
			}

			// THE TWO ROOTS ARE RECONCILED HERE (Sharp Edge 10). This command used to
			// root the corpus at os.Getwd() while `gate` rooted it at the backstop.yml
			// directory, so a validate run from a SUBDIRECTORY resolved a different
			// corpus than the gate did — both green, about different things. Both now
			// derive the project root from config discovery and the artifact root from
			// the same resolver.
			cfgFile, err := config.LoadConfig()
			if err != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", err)}
			}
			cfgPath, discoverErr := config.DiscoverConfigPath()
			projectRoot := "."
			if discoverErr == nil {
				projectRoot = filepath.Dir(cfgPath)
			}
			artifactRoot, rootErr := artifact.ResolveRoot(projectRoot, cfgFile.ArtifactRoot)
			if rootErr != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: %s", rootErr)}
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
			if cmd.Flags().Changed("directive") {
				typeFilters["directive"] = directiveFlag
				flagSet = true
			}

			// The cohort is computed ONCE here and travels with the config, so the
			// identity this run asserts against is the identity it reports.
			cohort, cohortErr := schema.ComputeCohort(SchemaFS)
			if cohortErr != nil {
				return &ExitCodeError{Code: ExitConfigError, Message: fmt.Sprintf("config: computing schema cohort: %s", cohortErr)}
			}

			// The artifact-corpus exclusion set comes from the INSTALLED PACKS, never
			// from a literal in this binary (ISSUE-122). A pack-load failure must NOT
			// fail this command — `artifact validate` has to keep working in a project
			// with no packs installed at all — so it degrades to the zero value, which
			// still excludes the tool-agnostic base.
			installedPacks, packsErr := loadInstalledPacks(projectRoot)
			var nonCorpus artifact.NonCorpusDirs
			if packsErr == nil {
				nonCorpus = artifact.NewNonCorpusDirs(mergeDependencyDirs(installedPacks))
			}

			cfg := ValidateConfig{
				ProjectRoot: projectRoot,
				Root:        artifactRoot,
				All:         allFlag,
				JSONOutput:  jsonFlag,
				SchemaFS:    SchemaFS,
				Cohort:      cohort,
				NonCorpus:   nonCorpus,
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
					if outErr := outputResult(cmd, &jsonFlag, configResult); outErr != nil {
						cmd.PrintErrln("Error:", outErr)
					}
				} else {
					cmd.PrintErrln("Error:", err)
				}
				return &ExitCodeError{Code: ExitConfigError, Message: err.Error()}
			}

			// Zero artifacts warning
			if result.ArtifactsFound == 0 && result.Pass {
				cmd.PrintErrln("warning: no artifacts found to validate")
			}

			// The WIDENED result is rendered directly. It is deliberately not narrowed
			// to a validate.ValidationResult first: that type carries only Violations,
			// so converting here would discard the asserted count, the scanned root,
			// the per-artifact records and the binary identity three lines before they
			// are printed — leaving the fields implemented in the struct and invisible
			// on the real path.
			if err := outputValidateResult(cmd, &jsonFlag, result); err != nil {
				return err
			}

			// Signal violations via exit code 1. Explained: the renderer already wrote
			// the violation report above, so the human line would duplicate it
			// (SPEC-055 REQ-011 / CLM-081). This applies to the VIOLATIONS return only —
			// the ExitConfigError further up prints nothing the operator can act on
			// beyond its own message and must stay loud.
			if !result.Pass {
				return &ExitCodeError{Code: ExitViolations, Message: "violations found", Explained: true}
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
	cmd.Flags().StringVar(&directiveFlag, "directive", "", "Validate directive artifacts (optional ID)")
	cmd.Flags().BoolVar(&allFlag, "all", false, "Validate all artifacts (takes precedence over type flags)")

	return cmd
}
