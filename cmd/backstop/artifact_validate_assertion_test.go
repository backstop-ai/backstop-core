package main

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/schema"
)

// realCohort computes the cohort over the binary's own embedded schemas — the value
// production asserts against. Tests build their ValidateConfig from this rather than
// from a hand-written identity so a drift between the two is impossible.
func realCohort(t *testing.T) schema.Cohort {
	t.Helper()
	c, err := schema.ComputeCohort(SchemaFS)
	if err != nil {
		t.Fatalf("computing the embedded schema cohort: %v", err)
	}
	if c.ID == "" {
		t.Fatal("the embedded schema cohort has an empty ID; every assertion below would be vacuous")
	}
	return c
}

// assertionConfig builds the ValidateConfig an --all run produces for dir.
func assertionConfig(t *testing.T, dir string) ValidateConfig {
	t.Helper()
	return ValidateConfig{
		ProjectRoot: dir,
		Root:        rootAtDir(t, dir),
		All:         true,
		SchemaFS:    SchemaFS,
		Cohort:      realCohort(t),
	}
}

// uncoveredSpecContent is a spec declaring a schema_version no binary carries. The
// number and body are otherwise well-formed, so the ONLY reason it can fail is the
// cohort assertion.
const uncoveredSpecContent = `---
title: "SPEC-777: Uncovered Schema Version"
number: SPEC-777
created: "2026-08-14"
status: draft
schema_version: spec/v99
spec_version: 1.0.0

implementation:
  summary: An artifact pinned to a schema version no binary carries.
  subject: pkg/nowhere

verification:
  level: unit
  test_command: go test ./...
  coverage_threshold: 90
---

# SPEC-777: Uncovered Schema Version

## Overview

Overview.

## Requirements

Requirements are declared in frontmatter.

## Implementation

Implementation details.

## Verification

Verification details.
`

// TestValidateArtifacts_UncoveredSchemaVersionRefusesGreen pins CLM-007. The refusal
// diagnostic must name ALL THREE of the artifact PATH, the declared SCHEMA_VERSION and
// the COHORT IDENTIFIER.
//
// Asserting only "it errored" would pass against the status quo: today an unknown
// schema_version produces a generic wrapped `loading schema for %s` that names none of
// the three, so the three substring assertions are the claim.
func TestValidateArtifacts_UncoveredSchemaVersionRefusesGreen(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-777-uncovered.spec.md": uncoveredSpecContent,
	})
	cohort := realCohort(t)

	result, err := ValidateArtifacts(assertionConfig(t, dir))
	if err == nil && result.Pass {
		t.Fatal("an artifact declaring an uncovered schema_version validated GREEN; the cohort assertion did not run")
	}

	diagnostic := refusalDiagnostic(result, err)
	for _, want := range []string{"SPEC-777-uncovered.spec.md", "spec/v99", cohort.ID} {
		if !strings.Contains(diagnostic, want) {
			t.Errorf("the refusal diagnostic does not name %q — it must name the artifact path, the declared schema_version AND the cohort identifier.\ngot: %s", want, diagnostic)
		}
	}
}

// TestValidateArtifacts_UncoveredSchemaVersionExitsNonZero pins CLM-008. DD-15 inverts
// this codebase's usual loud-but-non-blocking default here deliberately: on "I cannot
// tell whether this is valid", REFUSE. A pass with zero violations is the outcome this
// claim forbids.
func TestValidateArtifacts_UncoveredSchemaVersionExitsNonZero(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-777-uncovered.spec.md": uncoveredSpecContent,
	})

	result, err := ValidateArtifacts(assertionConfig(t, dir))
	if err != nil {
		return // a config-level refusal is exit 2, which is non-zero
	}
	if result.Pass {
		t.Fatal("an uncovered schema_version produced Pass=true; the refusal must be non-zero, never a clean pass")
	}
	if result.ViolationsCount == 0 || len(result.Violations) == 0 {
		t.Fatal("an uncovered schema_version produced a non-pass with ZERO violations; nothing would be reported to the operator")
	}
}

// TestValidateArtifacts_PlanRecordedAsSchemaless pins CLM-009. A plan routes by
// discovery type and declares no schema_version, so it is recorded AS schema-less
// rather than silently counted as cohort-covered — an empty SchemaIdentity that reads
// as covered-with-no-content is the shape this claim forbids.
func TestValidateArtifacts_PlanRecordedAsSchemaless(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, map[string]string{
		"specs/SPEC-001-test.spec.md":       validSpecContent("SPEC-001"),
		"plans/PLAN-SPEC-001-test.plan.yml": validPlanContent("PLAN-SPEC-001", "SPEC-001"),
	})

	result, err := ValidateArtifacts(assertionConfig(t, dir))
	if err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	planRecords := recordsOfType(result, "plan")
	if len(planRecords) != 1 {
		t.Fatalf("expected exactly 1 plan record, got %d (records: %+v)", len(planRecords), result.Records)
	}
	plan := planRecords[0]
	if !plan.Schemaless {
		t.Errorf("the plan record is not marked Schemaless; a plan declares no schema_version and must not be counted as cohort-covered")
	}
	if plan.SchemaIdentity != "" {
		t.Errorf("the plan record carries SchemaIdentity %q; a schema-less artifact has no identity to record", plan.SchemaIdentity)
	}

	// The control: a spec in the SAME run IS covered and DOES carry an identity, so
	// this test cannot pass against an implementation that marks everything schemaless.
	specRecords := recordsOfType(result, "spec")
	if len(specRecords) != 1 {
		t.Fatalf("expected exactly 1 spec record, got %d", len(specRecords))
	}
	if specRecords[0].Schemaless {
		t.Error("the spec record is marked Schemaless; only plans are schema-less")
	}
	if specRecords[0].SchemaIdentity == "" {
		t.Error("the spec record carries no SchemaIdentity, so the plan assertion above holds for the wrong reason")
	}
}

// TestValidateArtifacts_ZeroArtifactsUnderExistingRootStillPasses pins CLM-010, THE
// PROTECTED CASE. Zero artifacts under a root that EXISTS is a legitimate pass — the
// refusal is scoped strictly to "I found something I cannot prove I can validate", and
// tightening "empty is suspicious" into "empty is a failure" would break the init
// seed's acceptance bar before it is written.
//
// The legibility pair is asserted alongside the pass, because the early return this
// covers must stop being a bare `ValidateResult{Pass: true}`: an empty result that
// reports nothing reads as VERIFIED when it means EMPTY.
func TestValidateArtifacts_ZeroArtifactsUnderExistingRootStillPasses(t *testing.T) {
	dir := setupArtifactTestDir(t, artifactTestBackstopYML, nil)
	root := rootAtDir(t, dir)

	result, err := ValidateArtifacts(assertionConfig(t, dir))
	if err != nil {
		t.Fatalf("an existing but empty artifact root must not error: %v", err)
	}
	if !result.Pass {
		t.Fatal("an existing but empty artifact root did not pass; the refusal is scoped to artifacts that cannot be proven valid, not to their absence")
	}
	if result.ArtifactsAsserted != 0 {
		t.Errorf("ArtifactsAsserted = %d, want 0", result.ArtifactsAsserted)
	}
	if result.ScannedRoot != root.Path {
		t.Errorf("ScannedRoot = %q, want the scanned root %q; an empty pass that does not name what it scanned reads as verified when it means empty", result.ScannedRoot, root.Path)
	}
}

// TestValidateArtifacts_CoveredSchemaVersionsValidateUnchanged pins CLM-012: the
// assertion ADDS a guard, it does not change which corpora are accepted. The
// repo-root layout fixture is a fully covered corpus and must validate exactly as it
// does without the cohort — proven by running it BOTH ways and comparing the violation
// sets, not by asserting a bare pass.
func TestValidateArtifacts_CoveredSchemaVersionsValidateUnchanged(t *testing.T) {
	dir := layoutProfileDir(t, "repo-root")

	withCohort, err := ValidateArtifacts(assertionConfig(t, dir))
	if err != nil {
		t.Fatalf("validating a fully covered corpus WITH the cohort: %v", err)
	}

	// The same run with no cohort supplied is the pre-assertion semantics.
	noCohortCfg := assertionConfig(t, dir)
	noCohortCfg.Cohort = schema.Cohort{}
	withoutCohort, err := ValidateArtifacts(noCohortCfg)
	if err != nil {
		t.Fatalf("validating a fully covered corpus WITHOUT the cohort: %v", err)
	}

	if withCohort.ArtifactsFound == 0 {
		t.Fatal("the covered corpus validated zero artifacts, so the comparison below is vacuous")
	}
	if withCohort.Pass != withoutCohort.Pass {
		t.Errorf("the cohort assertion changed the verdict on a fully covered corpus: with=%v without=%v", withCohort.Pass, withoutCohort.Pass)
	}
	if got, want := violationKeys(withCohort), violationKeys(withoutCohort); !equalStrings(got, want) {
		t.Errorf("the cohort assertion changed the violation set on a fully covered corpus.\nwith:    %v\nwithout: %v", got, want)
	}
}

// TestValidateArtifacts_LeavesArtifactFilesUnmodified pins CLM-018 and Sharp Edge 6:
// the RECORD is the identity carrier, not the artifact. Validation writes nothing.
//
// Both content hashes AND modification times are snapshotted — a rewrite with identical
// bytes would leave the hash alone and move the mtime.
func TestValidateArtifacts_LeavesArtifactFilesUnmodified(t *testing.T) {
	dir := layoutProfileDir(t, "repo-root")

	before := fileFingerprints(t, dir)
	if len(before) == 0 {
		t.Fatal("the fixture produced no fingerprints, so the comparison below is vacuous")
	}

	if _, err := ValidateArtifacts(assertionConfig(t, dir)); err != nil {
		t.Fatalf("ValidateArtifacts: %v", err)
	}

	after := fileFingerprints(t, dir)
	if len(after) != len(before) {
		t.Fatalf("validation changed the file set: %d files before, %d after", len(before), len(after))
	}
	for path, fp := range before {
		got, ok := after[path]
		if !ok {
			t.Errorf("validation removed %s", path)
			continue
		}
		if got != fp {
			t.Errorf("validation modified %s: %s -> %s", path, fp, got)
		}
	}
}

// refusalDiagnostic renders whichever channel the refusal came back on — a returned
// error or violation messages — so the substring assertions do not depend on which
// one the implementation chose.
func refusalDiagnostic(result ValidateResult, err error) string {
	parts := []string{}
	if err != nil {
		parts = append(parts, err.Error())
	}
	for _, v := range result.Violations {
		parts = append(parts, v.Rule+" "+v.File+" "+v.Message)
	}
	return strings.Join(parts, "\n")
}

// recordsOfType selects the per-artifact records of one artifact type.
func recordsOfType(result ValidateResult, artifactType string) []ArtifactValidationRecord {
	var out []ArtifactValidationRecord
	for _, r := range result.Records {
		if r.Type == artifactType {
			out = append(out, r)
		}
	}
	return out
}

// violationKeys renders a violation set as a sorted, comparable list.
func violationKeys(result ValidateResult) []string {
	out := make([]string, 0, len(result.Violations))
	for _, v := range result.Violations {
		out = append(out, v.Rule+"|"+filepath.Base(v.File)+"|"+v.Message)
	}
	sort.Strings(out)
	return out
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// fileFingerprints maps every file under dir to a content-hash + modification-time
// fingerprint.
func fileFingerprints(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		sum := sha256.Sum256(data)
		rel, relErr := filepath.Rel(dir, path)
		if relErr != nil {
			return relErr
		}
		out[rel] = hex.EncodeToString(sum[:]) + "@" + info.ModTime().UTC().Format("2006-01-02T15:04:05.000000000Z")
		return nil
	})
	if err != nil {
		t.Fatalf("fingerprinting %s: %v", dir, err)
	}
	return out
}
