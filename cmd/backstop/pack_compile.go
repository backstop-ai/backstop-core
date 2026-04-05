package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmanson/backstop-core/pkg/compile"
	"github.com/bmanson/backstop-core/pkg/config"
	"github.com/bmanson/backstop-core/pkg/schema"
	"github.com/spf13/cobra"
)

// PackCompileResult holds aggregated results from compiling all standards.
type PackCompileResult struct {
	SchemaVersion string                  `json:"schema_version"`
	Standards     []CompileStandardResult `json:"standards"`
	Warnings      []string                `json:"warnings"`
	Errors        []string                `json:"errors"`
	Summary       CompileSummary          `json:"summary"`
}

// CompileStandardResult holds the result of compiling a single standard.
type CompileStandardResult struct {
	Standard    string   `json:"standard"`
	SourceFile  string   `json:"source_file"`
	OutputPaths []string `json:"output_paths"`
	Warnings    []string `json:"warnings,omitempty"`
	Error       string   `json:"error,omitempty"`
}

// CompileSummary holds counts from compilation.
type CompileSummary struct {
	Total    int `json:"total"`
	Compiled int `json:"compiled"`
	Failed   int `json:"failed"`
}

// embeddedSchemaSource implements compile.SchemaSource using an embedded FS.
// It extracts schemas to a temp directory so the existing schema loader can
// read them via filesystem paths. The temp dir is created once per source
// instance and cleaned up when no longer needed.
type embeddedSchemaSource struct {
	fsys    fs.FS
	tempDir string // lazily created
}

func (s *embeddedSchemaSource) LoadSchema(artifactType, version string) (*schema.Schema, error) {
	if s.tempDir == "" {
		dir, err := os.MkdirTemp("", "backstop-schemas-*")
		if err != nil {
			return nil, fmt.Errorf("create temp schema dir: %w", err)
		}
		s.tempDir = dir
	}

	// Extract the requested schema
	srcPath := filepath.Join("artifacts", artifactType, version, "schema.json")
	data, err := fs.ReadFile(s.fsys, srcPath)
	if err != nil {
		return nil, fmt.Errorf("read embedded schema %s: %w", srcPath, err)
	}
	dstDir := filepath.Join(s.tempDir, artifactType, version)
	if err := os.MkdirAll(dstDir, 0o755); err != nil {
		return nil, err
	}
	dstPath := filepath.Join(dstDir, "schema.json")
	if err := os.WriteFile(dstPath, data, 0o644); err != nil {
		return nil, err
	}

	// Also extract the base schema (artifact schemas extend it)
	baseSrcPath := filepath.Join("artifacts", "base", "schema.json")
	baseData, baseErr := fs.ReadFile(s.fsys, baseSrcPath)
	if baseErr == nil {
		baseDir := filepath.Join(s.tempDir, "base")
		if err := os.MkdirAll(baseDir, 0o755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(filepath.Join(baseDir, "schema.json"), baseData, 0o644); err != nil {
			return nil, err
		}
	}

	return schema.LoadArtifactSchema(dstPath, s.tempDir)
}

func (s *embeddedSchemaSource) cleanup() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

// packCompileOpts holds the resolved options for a pack compile run.
// This is separate from Cobra flags to enable testability.
type packCompileOpts struct {
	projectRoot   string
	standardsDirs []string
	outputDir     string
	jsonOutput    bool
	schemaSource  compile.SchemaSource
}

// runPackCompileWithOpts executes the pack compile logic with resolved options.
// It is the testable core of the command — Cobra handler resolves config into
// opts and delegates here.
func runPackCompileWithOpts(opts packCompileOpts) (*PackCompileResult, error) {
	result := &PackCompileResult{
		SchemaVersion: "cli/v1",
		Standards:     make([]CompileStandardResult, 0),
		Warnings:      make([]string, 0),
		Errors:        make([]string, 0),
	}

	// Step 2: Verify configured directories exist.
	// If some exist and some don't, warn for missing and proceed.
	// If none exist, return config error (exit 2).
	validDirs := make([]string, 0, len(opts.standardsDirs))
	for _, dir := range opts.standardsDirs {
		absDir := dir
		if !filepath.IsAbs(dir) {
			absDir = filepath.Join(opts.projectRoot, dir)
		}
		info, err := os.Stat(absDir)
		if err != nil || !info.IsDir() {
			result.Warnings = append(result.Warnings, fmt.Sprintf("standards directory not found: %s", dir))
			continue
		}
		validDirs = append(validDirs, absDir)
	}

	if len(validDirs) == 0 {
		return nil, &ExitCodeError{
			Code:    ExitConfigError,
			Message: fmt.Sprintf("no configured standards directories exist: %v", opts.standardsDirs),
		}
	}

	// Step 3: Discover standard files.
	paths, err := discoverStandards(validDirs)
	if err != nil {
		return nil, fmt.Errorf("discovering standards: %w", err)
	}

	result.Summary.Total = len(paths)

	if len(paths) == 0 {
		return result, nil
	}

	// Step 4: Create output directory and compile each standard.
	outDir := opts.outputDir
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(opts.projectRoot, outDir)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	for _, stdPath := range paths {
		compileResult, compileErr := compile.Compile(stdPath, compile.CompileOptions{
			OutputDir:    outDir,
			SchemaSource: opts.schemaSource,
		})

		stdResult := CompileStandardResult{
			SourceFile: stdPath,
		}

		if compileErr != nil {
			stdResult.Error = compileErr.Error()
			stdResult.Standard = filepath.Base(stdPath)
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %s", filepath.Base(stdPath), compileErr.Error()))
			result.Summary.Failed++
		} else {
			stdResult.Standard = compileResult.Manifest.Standard
			stdResult.OutputPaths = compileResult.OutputPaths
			stdResult.Warnings = compileResult.Warnings
			result.Summary.Compiled++

			// Propagate per-standard warnings to top-level
			for _, w := range compileResult.Warnings {
				result.Warnings = append(result.Warnings, w)
			}
		}

		result.Standards = append(result.Standards, stdResult)
	}

	return result, nil
}

// formatPackCompileResult formats the result as JSON or human-readable text.
func formatPackCompileResult(result *PackCompileResult, jsonOutput bool) string {
	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Sprintf(`{"error": "marshal failed: %s"}`, err)
		}
		return string(data)
	}

	return formatPackCompileHuman(result)
}

// formatPackCompileHuman formats the result for human consumption.
func formatPackCompileHuman(result *PackCompileResult) string {
	var sb strings.Builder

	header := fmt.Sprintf("Compiled %d standards", result.Summary.Compiled)
	if result.Summary.Failed > 0 {
		header += fmt.Sprintf(" (%d failed)", result.Summary.Failed)
	}
	sb.WriteString(header + ":\n")

	for _, std := range result.Standards {
		if std.Error != "" {
			sb.WriteString(fmt.Sprintf("  FAIL %s: %s\n", std.SourceFile, std.Error))
		} else {
			sb.WriteString(fmt.Sprintf("  OK   %s\n", std.SourceFile))
			for _, p := range std.OutputPaths {
				sb.WriteString(fmt.Sprintf("       -> %s\n", p))
			}
		}
	}

	if len(result.Warnings) > 0 {
		sb.WriteString("\nWarnings:\n")
		for _, w := range result.Warnings {
			sb.WriteString(fmt.Sprintf("  - %s\n", w))
		}
	}

	sb.WriteString(fmt.Sprintf("\nSummary: %d total, %d compiled, %d failed\n",
		result.Summary.Total, result.Summary.Compiled, result.Summary.Failed))

	return sb.String()
}

// newPackCompileCommand creates the Cobra command for backstop pack compile.
// It is called from NewRootCommand to register the command.
func newPackCompileCommand(jsonFlag *bool) *cobra.Command {
	return &cobra.Command{
		Use:   "compile",
		Short: "Compile enforcement packs from standards",
		Long: `Discovers all .standard.md files in the project's standards directories,
compiles each using the standards compiler, and writes enforcement output
(manifest JSON, semgrep YAML, native checks JSON) to .backstop/rules/.`,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// Step 1: Load config.
			cfg, err := config.LoadConfig()
			if err != nil {
				return &ExitCodeError{
					Code:    ExitConfigError,
					Message: fmt.Sprintf("config: %s", err),
				}
			}

			// Resolve project root from config file location.
			cfgPath, cfgErr := config.DiscoverConfigPath()
			projectRoot := "."
			if cfgErr == nil {
				projectRoot = filepath.Dir(cfgPath)
			}

			schSrc := &embeddedSchemaSource{fsys: SchemaFS}
			defer schSrc.cleanup()

			opts := packCompileOpts{
				projectRoot:   projectRoot,
				standardsDirs: defaultStandardsDirs(cfg.StandardsDirs),
				outputDir:     ".backstop/rules",
				jsonOutput:    jsonFlag != nil && *jsonFlag,
				schemaSource:  schSrc,
			}

			result, runErr := runPackCompileWithOpts(opts)
			if runErr != nil {
				return runErr
			}

			cmd.Print(formatPackCompileResult(result, opts.jsonOutput))

			if result.Summary.Failed > 0 {
				return &ExitCodeError{
					Code:    ExitViolations,
					Message: fmt.Sprintf("%d standards failed to compile", result.Summary.Failed),
				}
			}

			return nil
		},
	}
}
