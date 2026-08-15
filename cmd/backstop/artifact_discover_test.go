package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/artifact"
	"github.com/backstop-ai/backstop-core/pkg/config"
)

// rootAtDir resolves an UNCONFIGURED artifact root at dir — the shape every project
// that declares no artifact_root gets, and therefore the mechanical repair for the
// pre-existing DiscoverArtifacts call sites in this file: their project root IS their
// artifact root, so what they assert is unchanged.
//
// Handing in a bare artifact.Root{} would compile and resolve a RELATIVE root read
// against the test process's CWD rather than the fixture, which is why the resolution
// runs for real here and its error is fatal.
func rootAtDir(t *testing.T, dir string) artifact.Root {
	t.Helper()
	root, err := artifact.ResolveRoot(dir, "")
	if err != nil {
		t.Fatalf("resolving an unconfigured artifact root at %s: %v", dir, err)
	}
	return root
}

// TestArtifactValidate_Discover_Spec verifies that *.spec.md files are
// discovered as spec artifacts. (CLM-032)
func TestArtifactValidate_Discover_Spec(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-test.spec.md":  validSpecContent("SPEC-001"),
		"specs/SPEC-002-test.spec.md":  validSpecContent("SPEC-002"),
		"plans/PLAN-SPEC-001.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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

	arts, err := DiscoverArtifacts(rootAtDir(t, dir), nil)
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
	filtered, err := DiscoverArtifacts(rootAtDir(t, dir), []string{"plan"})
	if err != nil {
		t.Fatalf("DiscoverArtifacts with filter: %v", err)
	}
	if len(filtered) != 0 {
		t.Errorf("expected 0 plan artifacts, got %d", len(filtered))
	}
}

// layoutProfileDir copies the named layout-profile fixture project into a t.TempDir()
// and returns the copy's path. Fixtures are copied rather than used in place so no test
// can mutate the committed fixture.
func layoutProfileDir(t *testing.T, profile string) string {
	t.Helper()
	src := filepath.Join("testdata", "layout-profiles", profile)
	if _, err := os.Stat(src); err != nil {
		t.Fatalf("layout-profile fixture %q not found: %v", profile, err)
	}
	dst := t.TempDir()
	copyFixtureDir(t, src, dst)
	return dst
}

// layoutProfileRoot resolves the artifact root a layout-profile fixture DECLARES, by
// reading its own backstop.yml. Resolving from the fixture's config rather than from a
// literal is what makes these tests exercise the real consumer path — a hand-written
// ".backstop" would pass even if the config key were never read.
func layoutProfileRoot(t *testing.T, dir string) artifact.Root {
	t.Helper()
	cfg, err := config.LoadConfigFromPath(filepath.Join(dir, "backstop.yml"))
	if err != nil {
		t.Fatalf("loading the fixture's backstop.yml at %s: %v", dir, err)
	}
	root, err := artifact.ResolveRoot(dir, cfg.ArtifactRoot)
	if err != nil {
		t.Fatalf("resolving the fixture's declared artifact root %q at %s: %v", cfg.ArtifactRoot, dir, err)
	}
	return root
}

// discoveredRelSet maps each discovered artifact to its path RELATIVE to root, keyed
// with forward slashes so the expectation reads the same on every platform.
func discoveredRelSet(t *testing.T, root string, arts []DiscoveredArtifact) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, a := range arts {
		rel, err := filepath.Rel(root, a.Path)
		if err != nil {
			t.Fatalf("relativizing %s against %s: %v", a.Path, root, err)
		}
		out[filepath.ToSlash(rel)] = a.Type
	}
	return out
}

// TestDiscoverArtifacts_WalksResolvedRootIncludingDotBackstop pins CLM-045 and is
// Sharp Edge 1. Before this change artifact_discover.go skipped `.backstop`
// UNCONDITIONALLY, so a .backstop/-rooted project discovered ZERO artifacts and
// ValidateArtifacts returned Pass:true over the empty set — the false green this whole
// spec is named after.
//
// The assertion is on the discovered SET, not on a count: a count of three is satisfied
// by three wrong files.
func TestDiscoverArtifacts_WalksResolvedRootIncludingDotBackstop(t *testing.T) {
	dir := layoutProfileDir(t, "dotbackstop-root")
	root := layoutProfileRoot(t, dir)

	if !root.Configured {
		t.Fatal("the dotbackstop-root fixture resolved an UNCONFIGURED root; its backstop.yml declares artifact_root and the rest of this test would be about the wrong directory")
	}

	arts, err := DiscoverArtifacts(root, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	got := discoveredRelSet(t, root.Path, arts)
	want := map[string]string{
		"specs/SPEC-001-sample.spec.md":       "spec",
		"bundles/BUNDLE-001-sample.bundle.md": "bundle",
		"plans/PLAN-SPEC-001-sample.plan.yml": "plan",
	}
	for rel, kind := range want {
		gotKind, ok := got[rel]
		if !ok {
			t.Errorf("discovery did not reach %q under the resolved .backstop root; discovered: %v", rel, got)
			continue
		}
		if gotKind != kind {
			t.Errorf("%q classified as %q, want %q", rel, gotKind, kind)
		}
	}
}

// TestDiscoverArtifacts_DotBackstopRootStillExcludesInstalledPacks pins CLM-046, which
// is spec Review Question 1: several installed packs are themselves backstop repos
// carrying their own artifacts, so the exclusion must become ROOT-RELATIVE (always skip
// .backstop/packs) rather than simply deleted along with the wholesale .backstop skip.
func TestDiscoverArtifacts_DotBackstopRootStillExcludesInstalledPacks(t *testing.T) {
	dir := layoutProfileDir(t, "dotbackstop-root")
	root := layoutProfileRoot(t, dir)

	// The fixture really does carry an artifact-shaped file inside an installed pack —
	// otherwise the assertion below would hold vacuously.
	planted := filepath.Join(dir, ".backstop", "packs", "test-org", "sample-pack", "specs", "SPEC-999-installed-pack.spec.md")
	if _, err := os.Stat(planted); err != nil {
		t.Fatalf("the fixture's installed-pack artifact is missing, so this test would pass for the wrong reason: %v", err)
	}

	arts, err := DiscoverArtifacts(root, nil)
	if err != nil {
		t.Fatalf("DiscoverArtifacts: %v", err)
	}

	for _, a := range arts {
		if strings.Contains(a.Path, filepath.Join(".backstop", "packs")) {
			t.Errorf("discovery reached inside an installed pack: %s", a.Path)
		}
		if filepath.Base(a.Path) == "SPEC-999-installed-pack.spec.md" {
			t.Errorf("the installed pack's own spec was discovered as part of the consumer corpus: %s", a.Path)
		}
	}
}
