package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ValidPackTypes is the registry of valid pack types. ISSUE-032 (Defect A +
// ISSUE-030 fold): the retired native-standards world's "rule"/"code" types — which
// emitted a `.standard.md` / `.recipe.md` compiled by the DELETED standards compiler
// — are gone. The live pack shapes are engine-model packs (SPEC-031/SPEC-035): an
// `engine` pack (a self-contained sandbox-validator engine), a `mechanism` pack
// (wraps a native tool as an engine), and a `toolchain` pack (a language's native
// build/test/lint passes). All three scaffold to a VALID enforcement pack.yml with a
// declared engines: block — the packs-only substrate, no compiler in between.
var ValidPackTypes = map[string]bool{ // nosemgrep: go.core.no-global-mutable-state — immutable lookup table, never mutated after init
	"engine":    true,
	"mechanism": true,
	"toolchain": true,
}

// ISSUE-030 LINEAGE TOMBSTONE (folded into ISSUE-032 Defect A). The native-standards
// SCAFFOLDER once emitted `STD-<LANG>-NNN-<slug>.standard.md` (schema_version
// standard/v1) into `standards/<language>/` and `<slug>.recipe.md` (schema_version
// recipe/v1) into `recipes/<language>/<slug>/`, numbered by ResolvePackNumber's scan
// of `standards/`. Those artifacts were the INPUT to a standards→pack COMPILER that the
// 2026-06-16 packs-only decision retired (MEMORY: packs_only_no_native_standards): a
// pack is now authored DIRECTLY as pack.yml, never compiled from a `.standard.md`. The
// scaffolder, the `standards/` writer, ResolvePackNumber, the standard/v1 schema, and
// the sole standard fixture are all DELETED here; scaffoldEnginePack replaces them,
// emitting the live engine-pack shape. This comment is the in-code lineage record so
// the dead path cannot be silently reintroduced (see deletion_assertion_test.go).

var slugPattern = regexp.MustCompile(`^[a-z][a-z0-9]*(-[a-z0-9]+)*$`) // nosemgrep: go.core.no-global-mutable-state — compile-once immutable regexp, never reassigned

// ValidateSlug checks that the slug conforms to ^[a-z][a-z0-9]*(-[a-z0-9]+)*$
// with a minimum length of 2 and maximum length of 64 characters.
func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is required")
	}
	if len(slug) < 2 {
		return fmt.Errorf("slug must be at least 2 characters, got %d", len(slug))
	}
	if len(slug) > 64 {
		return fmt.Errorf("slug must be at most 64 characters, got %d", len(slug))
	}
	if !slugPattern.MatchString(slug) {
		return fmt.Errorf("slug must match ^[a-z][a-z0-9]*(-[a-z0-9]+)*$, got %q", slug)
	}
	return nil
}

// ScaffoldOptions holds options for pack scaffolding.
type ScaffoldOptions struct {
	Type        string
	Language    string
	Slug        string
	ProjectRoot string
}

// ScaffoldResult holds the result of scaffolding.
type ScaffoldResult struct {
	Paths         []string `json:"paths"`
	PackID        string   `json:"pack_id"`
	Type          string   `json:"type"`
	Language      string   `json:"language"`
	Slug          string   `json:"slug"`
	SchemaVersion string   `json:"schema_version"`
}

// HumanString formats the result for human display.
func (r *ScaffoldResult) HumanString() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Created %s pack: %s\n", r.Type, r.PackID))
	for _, p := range r.Paths {
		sb.WriteString(fmt.Sprintf("  %s\n", p))
	}
	// Next-step hint: pack check/pack test read pack.yml from the CURRENT dir, so the
	// author must cd into the freshly-scaffolded pack before validating it (ISSUE-049).
	sb.WriteString(fmt.Sprintf("\nNext: cd %s && backstop pack check   # then: backstop pack test\n", r.Slug))
	return sb.String()
}

// ScaffoldPack creates the pack directory and template files for the given pack
// type. Every valid type produces a self-contained enforcement pack whose declared
// engines: block + sample rule pass `pack check`, `pack test`, and the gate (Defect
// A / CLM-001).
func ScaffoldPack(opts ScaffoldOptions) (*ScaffoldResult, error) {
	if !ValidPackTypes[opts.Type] {
		return nil, fmt.Errorf("unsupported pack type: %q (must be engine, mechanism, or toolchain)", opts.Type)
	}
	return scaffoldEnginePack(opts)
}

// packTypeBlurb returns the one-line description stamped into the scaffolded pack's
// `description:` field. It is the only per-type difference — the three types share
// the same VALID engine-pack skeleton so each passes check/test/gate identically; the
// blurb hints at the shape the author is expected to grow into.
func packTypeBlurb(packType string) string {
	switch packType {
	case "mechanism":
		return "Mechanism pack scaffolded by backstop pack new — wrap a native tool/command as an enforcement engine. Replace the sample sandbox validator with your own."
	case "toolchain":
		return "Toolchain pack scaffolded by backstop pack new — bundle a language's native build/test/lint passes as engines. Replace the sample sandbox validator with your own."
	default: // engine
		return "Engine pack scaffolded by backstop pack new — your validator executable IS the enforcement logic. Replace the sample sandbox validator with your own."
	}
}

// scaffoldEnginePack writes a VALID enforcement engine pack (ISSUE-032 CLM-001): a
// pack.yml declaring a self-contained sandbox engine (input_mode: none, no external
// tool) in its engines: block with STRING enum values, a single sample rule carrying
// an engine + risk_class + claims with positive/negative fixtures + a referenced
// validator, plus the validator script and fixture files. The sandbox shape makes the
// scaffolded pack gate-green with NO tool the author must install (runSandboxEngine
// runs the always-pass validator); the author swaps in real detection logic.
func scaffoldEnginePack(opts ScaffoldOptions) (*ScaffoldResult, error) {
	packDir := filepath.Join(opts.ProjectRoot, opts.Slug)
	if _, err := os.Stat(packDir); err == nil {
		return nil, fmt.Errorf("directory already exists: %s", packDir)
	}

	packName := fmt.Sprintf("local/%s", opts.Slug)
	engineName := fmt.Sprintf("%s-engine", opts.Slug)
	ruleID := fmt.Sprintf("%s-sample", opts.Slug)
	claimID := fmt.Sprintf("%s-clm-001", opts.Slug)
	validatorRel := filepath.Join("validators", opts.Slug+".sh")
	positiveRel := filepath.Join("fixtures", "valid", "example.txt")
	negativeRel := filepath.Join("fixtures", "invalid", "example.txt")

	manifest := fmt.Sprintf(`name: %s
version: 0.1.0
language: %s
archetype: enforcement
description: %q
# Declared engine block (SPEC-035): the engine is DATA. This sample declares a
# self-contained SANDBOX engine — input_mode none + no command means "the pack
# validator executable IS the logic", so the pack needs NO external tool to run
# green. The string enum values (scope_kind, gate_type) exercise the same parse the
# consumer uses. Swap this for a semgrep/ast-grep/native-tool engine as needed.
engines:
  %s:
    command: ""
    input_mode: none
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 0.1.0
    rules:
      # Sample enforcement rule. Its validator (validators/%s.sh) currently always
      # passes — replace it with your real detection and make the negative fixture
      # trigger it while the positive fixture stays clean.
      - id: %s
        engine: %s
        validator: %s
        risk_class: correctness
        claims:
          - id: %s
            text: "Sample enforced property — replace with your own claim."
            fixtures:
              positive:
                - %s
              negative:
                - %s
`,
		packName,
		opts.Language,
		packTypeBlurb(opts.Type),
		engineName,
		opts.Slug,
		ruleID,
		engineName,
		filepath.ToSlash(validatorRel),
		claimID,
		filepath.ToSlash(positiveRel),
		filepath.ToSlash(negativeRel),
	)

	validator := "#!/bin/sh\n# Sample sandbox validator: always passes. Replace with real enforcement —\n# exit non-zero and print a message to flag a violation.\nexit 0\n"
	positiveFixture := "// Positive fixture: the sample rule must NOT fire on this compliant example.\npackage sample\n"
	negativeFixture := "// Negative fixture: the sample rule SHOULD fire here once you add real logic.\npackage sample\n"

	files := []struct {
		rel  string
		data string
		mode os.FileMode
	}{
		{"pack.yml", manifest, 0o644},
		{validatorRel, validator, 0o755},
		{positiveRel, positiveFixture, 0o644},
		{negativeRel, negativeFixture, 0o644},
	}

	var paths []string
	for _, f := range files {
		full := filepath.Join(packDir, f.rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			return nil, fmt.Errorf("creating pack directory %s: %w", filepath.Dir(full), err)
		}
		if err := os.WriteFile(full, []byte(f.data), f.mode); err != nil {
			return nil, fmt.Errorf("writing %s: %w", f.rel, err)
		}
		paths = append(paths, full)
	}

	return &ScaffoldResult{
		Paths:         paths,
		PackID:        packName,
		Type:          opts.Type,
		Language:      opts.Language,
		Slug:          opts.Slug,
		SchemaVersion: "pack-new/v1",
	}, nil
}
