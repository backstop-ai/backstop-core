package main

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

// TestArtifactValidate_Discover_Spec verifies that *.spec.md files are
// discovered as spec artifacts. (CLM-032)
func TestArtifactValidate_Discover_Spec(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-test.spec.md": validSpecContent("SPEC-001"),
		"specs/SPEC-002-test.spec.md": validSpecContent("SPEC-002"),
		"plans/PLAN-SPEC-001.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	var specs []string
	for _, a := range arts {
		if a.Type == "spec" {
			specs = append(specs, filepath.Base(a.Path))
		}
	}
	sort.Strings(specs)

	if len(specs) != 2 {
		t.Fatalf("expected 2 spec artifacts, got %d: %v", len(specs), specs)
	}
	if specs[0] != "SPEC-001-test.spec.md" || specs[1] != "SPEC-002-test.spec.md" {
		t.Errorf("unexpected spec files: %v", specs)
	}
}

// TestArtifactValidate_Discover_Plan verifies that *.plan.yml files are
// discovered as plan artifacts. (CLM-033)
func TestArtifactValidate_Discover_Plan(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"plans/PLAN-SPEC-001-test.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	var plans []string
	for _, a := range arts {
		if a.Type == "plan" {
			plans = append(plans, filepath.Base(a.Path))
		}
	}

	if len(plans) != 1 {
		t.Fatalf("expected 1 plan artifact, got %d: %v", len(plans), plans)
	}
	if plans[0] != "PLAN-SPEC-001-test.plan.yml" {
		t.Errorf("unexpected plan file: %s", plans[0])
	}
}

// TestArtifactValidate_Discover_ADR verifies that ADR-*.adr.md files are
// discovered as ADR artifacts. (CLM-034)
func TestArtifactValidate_Discover_ADR(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"adrs/ADR-0001-test.adr.md": `---
number: ADR-0001
created: "2026-04-01"
status: Accepted
deciders: "@test"
decisions: "D-001"
schema_version: adr/v1
---

# ADR-0001: Test ADR

## Thesis

Test.

## Alternatives

None.

## Decision

Accepted.

## Consequences

None.
`,
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	var adrs []string
	for _, a := range arts {
		if a.Type == "adr" {
			adrs = append(adrs, filepath.Base(a.Path))
		}
	}

	if len(adrs) != 1 {
		t.Fatalf("expected 1 ADR artifact, got %d: %v", len(adrs), adrs)
	}
	if adrs[0] != "ADR-0001-test.adr.md" {
		t.Errorf("unexpected ADR file: %s", adrs[0])
	}
}

// TestArtifactValidate_Discover_Bundle verifies that *.bundle.md files are
// discovered as bundle artifacts. (CLM-035)
func TestArtifactValidate_Discover_Bundle(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"bundles/cli.bundle.md": `---
title: Test Bundle
schema_version: bundle/v1

bundle:
  name: test
  version: "1.0.0"
  created: "2026-04-01"
  updated: "2026-04-01"
  category: tool
---

# Test Bundle

## Problem Statement

Test.
`,
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	var bundles []string
	for _, a := range arts {
		if a.Type == "bundle" {
			bundles = append(bundles, filepath.Base(a.Path))
		}
	}

	if len(bundles) != 1 {
		t.Fatalf("expected 1 bundle artifact, got %d: %v", len(bundles), bundles)
	}
	if bundles[0] != "cli.bundle.md" {
		t.Errorf("unexpected bundle file: %s", bundles[0])
	}
}

// TestArtifactValidate_Discover_Issue verifies that *.issue.md files are
// discovered as issue artifacts. (CLM-036)
func TestArtifactValidate_Discover_Issue(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"issues/ISSUE-001-test.issue.md": `---
title: Test Issue
number: ISSUE-001
schema_version: issue/v1
---

# ISSUE-001: Test Issue

## Problem

Test.
`,
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	var issues []string
	for _, a := range arts {
		if a.Type == "issue" {
			issues = append(issues, filepath.Base(a.Path))
		}
	}

	if len(issues) != 1 {
		t.Fatalf("expected 1 issue artifact, got %d: %v", len(issues), issues)
	}
	if issues[0] != "ISSUE-001-test.issue.md" {
		t.Errorf("unexpected issue file: %s", issues[0])
	}
}

// Standard artifact type removed — standards live inside packs now (DD-18/DD-45).

// TestArtifactValidate_Discover_IgnoresNonArtifacts verifies that files not
// matching any artifact filename pattern are ignored. (CLM-038)
func TestArtifactValidate_Discover_IgnoresNonArtifacts(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-test.spec.md": validSpecContent("SPEC-001"),
		"README.md":                   "# Readme",
		"notes.txt":                   "some notes",
		"config.json":                 `{"key": "value"}`,
		"src/main.go":                 "package main",
	})

	arts, err := DiscoverArtifacts(dir, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	// Only the spec file should be discovered
	if len(arts) != 1 {
		t.Fatalf("expected 1 artifact, got %d", len(arts))
	}
	if arts[0].Type != "spec" {
		t.Errorf("expected spec type, got %s", arts[0].Type)
	}

	// Verify non-artifacts are not in the list
	for _, a := range arts {
		base := filepath.Base(a.Path)
		if base == "README.md" || base == "notes.txt" || base == "config.json" || base == "main.go" {
			t.Errorf("non-artifact file should not be discovered: %s", base)
		}
	}

	// Also verify with type filter
	filtered, err := DiscoverArtifacts(dir, []string{"plan"})
	if err != nil {
		t.Fatalf("DiscoverArtifacts with filter: %v", err)
	}
	if len(filtered) != 0 {
		t.Errorf("expected 0 plan artifacts, got %d", len(filtered))
	}
}

// setupArtifactTestDirFromFixtures creates a test dir using the embedded
// testdata/artifacts fixtures. Useful for tests that need the full fixture set.
func setupArtifactTestDirFromFixtures(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	fixtureDir := filepath.Join("testdata", "artifacts")
	if _, err := os.Stat(fixtureDir); err != nil {
		t.Skipf("testdata/artifacts not found: %v", err)
	}

	copyFixtureDir(t, fixtureDir, dir)
	return dir
}
