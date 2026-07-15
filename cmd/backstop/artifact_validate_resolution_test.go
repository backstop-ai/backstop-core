package main

import (
	"testing"
)

// resolutionBundleFixture returns a bundle declaring trace-demo:REQ-001@1.0.0 so
// a spec ref can resolve against it through the wired resolution pass.
func resolutionBundleFixture() string {
	return `---
title: "Trace Demo"
schema_version: bundle/v1
bundle:
  name: trace-demo
  version: 1.0.0
  created: "2026-07-14"
  category: feature
status:
  maturity: exploring
requirements:
  - id: REQ-001
    text: A real requirement
    version: 1.0.0
---

# Trace Demo

## Current Thinking

Demo bundle.
`
}

// resolutionSpecFixture returns a draft spec whose single requirement cites the
// given supports ref.
func resolutionSpecFixture(number, supports string) string {
	return `---
title: "` + number + `: Trace Citer"
number: ` + number + `
created: "2026-07-14"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: A citing spec.
  subject: pkg/test

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90

requirements:
  - id: REQ-001
    text: The citer requirement
    supports: ` + supports + `

claims:
  - id: CLM-001
    requirement: REQ-001
    text: A claim
    tests:
      - TestSomething
---

# ` + number + `: Trace Citer

## Overview

Overview.

## Requirements

In frontmatter.

## Implementation

Impl.

## Verification

Verify.
`
}

func resolutionResultHasRule(result ValidateResult, rule string) bool {
	for _, v := range result.Violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// TestValidateArtifacts_ResolutionPassSharedByCLIAndGate proves the resolution
// pass runs inside ValidateArtifacts — the shared walk both `backstop artifact
// validate` and the gate's ValidateAll flow through — surfacing a dangling ref as
// a violation, and staying clean on a resolvable corpus (so it cannot go vacuous).
func TestValidateArtifacts_ResolutionPassSharedByCLIAndGate(t *testing.T) {
	// Dangling: spec cites a bundle absent from the corpus → resolution violation.
	danglingDir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"trace-demo.bundle.md":         resolutionBundleFixture(),
		"SPEC-901-trace-citer.spec.md": resolutionSpecFixture("SPEC-901", "no-such-bundle:REQ-001@1.0.0"),
	})
	danglingResult, err := ValidateArtifacts(ValidateConfig{ProjectRoot: danglingDir, All: true, SchemaFS: SchemaFS})
	if err != nil {
		t.Fatalf("ValidateArtifacts (dangling) errored: %v", err)
	}
	if !resolutionResultHasRule(danglingResult, "supports/missing-bundle") {
		t.Errorf("expected supports/missing-bundle from the wired pass, got: %v", danglingResult.Violations)
	}

	// Clean: spec cites the real, declared, logged REQ → no resolution violation.
	cleanDir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"trace-demo.bundle.md":         resolutionBundleFixture(),
		"SPEC-902-trace-citer.spec.md": resolutionSpecFixture("SPEC-902", "trace-demo:REQ-001@1.0.0"),
	})
	cleanResult, err := ValidateArtifacts(ValidateConfig{ProjectRoot: cleanDir, All: true, SchemaFS: SchemaFS})
	if err != nil {
		t.Fatalf("ValidateArtifacts (clean) errored: %v", err)
	}
	if resolutionResultHasRule(cleanResult, "supports/missing-bundle") {
		t.Errorf("expected a resolvable ref to produce no missing-bundle, got: %v", cleanResult.Violations)
	}
}

// TestResolveSupports_TypeScopedRunUsesFullCorpusCatalog proves a --spec-scoped
// run builds the resolution catalog from the FULL corpus (independent of scope),
// so a real ref resolves clean instead of false-redding as a missing bundle — and
// the scoped verdict matches the unscoped one.
func TestResolveSupports_TypeScopedRunUsesFullCorpusCatalog(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"trace-demo.bundle.md":         resolutionBundleFixture(),
		"SPEC-903-trace-citer.spec.md": resolutionSpecFixture("SPEC-903", "trace-demo:REQ-001@1.0.0"),
	})

	// Scoped to specs only: the per-artifact loop sees no bundles, but the catalog
	// is built from a full-corpus bundle discovery, so the ref must still resolve.
	scoped, err := ValidateArtifacts(ValidateConfig{
		ProjectRoot: dir,
		TypeFilters: map[string]string{"spec": ""},
		SchemaFS:    SchemaFS,
	})
	if err != nil {
		t.Fatalf("scoped ValidateArtifacts errored: %v", err)
	}
	if resolutionResultHasRule(scoped, "supports/missing-bundle") {
		t.Errorf("scoped --spec run false-redded the ref as missing-bundle: %v", scoped.Violations)
	}

	// Unscoped run over the same corpus must produce the identical resolution verdict.
	unscoped, err := ValidateArtifacts(ValidateConfig{ProjectRoot: dir, All: true, SchemaFS: SchemaFS})
	if err != nil {
		t.Fatalf("unscoped ValidateArtifacts errored: %v", err)
	}
	if resolutionResultHasRule(unscoped, "supports/missing-bundle") {
		t.Errorf("unscoped run unexpectedly produced missing-bundle: %v", unscoped.Violations)
	}
}
