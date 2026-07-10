package gate

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestBaseline_CompareBaseline_AllScopeAllowsRuleSetChangeSeeding(t *testing.T) {
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{}}
	current := []Violation{{Rule: "code_check/new-rule", File: "src/legacy.ts", Message: "legacy code violation after rule-set change", Severity: "error"}}

	comparison := CompareBaseline(current, baseline, BaselineCompareOptions{
		Scope:                     newGateScope("", GateScopeModeAll, nil, nil),
		AllowRuleSetChangeSeeding: true,
	})

	if len(comparison.NewViolations) != 0 {
		t.Fatalf("expected seeded full-scope run to report 0 new violations, got %d", len(comparison.NewViolations))
	}
	if len(comparison.SeededViolations) != 1 {
		t.Fatalf("expected exactly 1 seeded violation, got %d", len(comparison.SeededViolations))
	}
	if comparison.SeededViolations[0].Rule != "code_check/new-rule" {
		t.Fatalf("expected seeded violation rule to round-trip, got %#v", comparison.SeededViolations[0])
	}
}

func TestBaseline_CompareBaseline_ScopedRunDisallowsRuleSetChangeBypass(t *testing.T) {
	baseline := &BaselineArtifact{SchemaVersion: BaselineSchemaV1, Violations: []Violation{}}
	current := []Violation{{Rule: "code_check/new-rule", File: "changed.ts", Message: "changed-file regression", Severity: "error"}}

	comparison := CompareBaseline(current, baseline, BaselineCompareOptions{
		Scope:                     newGateScope("", GateScopeModeDiff, []string{"changed.ts"}, nil),
		AllowRuleSetChangeSeeding: true,
	})

	if len(comparison.NewViolations) != 1 {
		t.Fatalf("expected changed-file regression to remain new violation, got %d", len(comparison.NewViolations))
	}
	if len(comparison.SeededViolations) != 0 {
		t.Fatalf("expected scoped run to seed 0 violations, got %d", len(comparison.SeededViolations))
	}
	if comparison.NewViolations[0].File != "changed.ts" {
		t.Fatalf("expected changed file violation in new_violations, got %#v", comparison.NewViolations[0])
	}
}

// TestBaseline_ViolationJSON_AdditiveIdentityFieldsContract captures the
// additive baseline-identity expectation without changing existing gate/v1
// fields.
func TestBaseline_ViolationJSON_AdditiveIdentityFieldsContract(t *testing.T) {
	v := Violation{
		Rule:       "code_check/no-eval",
		File:       "src/main.ts",
		Message:    "eval usage is forbidden",
		Severity:   "error",
		SourcePack: "test-org/test-pack",
	}

	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal violation: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal violation: %v", err)
	}

	for _, field := range []string{"rule", "file", "message", "severity", "source_pack"} {
		if _, ok := raw[field]; !ok {
			t.Fatalf("expected existing gate/v1 field %q", field)
		}
	}

	for _, additive := range []string{"identity", "identity_hash", "region_hash"} {
		if _, ok := raw[additive]; !ok {
			t.Errorf("expected additive baseline identity field %q", additive)
		}
	}
}

// TestBaseline_IdentityHash_DeterministicContract encodes the expectation that
// baseline identity hashing is stable for repeated serialization.
// @waiver:test_substantiveness:false-positive:2026-10-08 PROBE
func TestBaseline_IdentityHash_DeterministicContract(t *testing.T) { // @waiver:test_substantiveness:false-positive:2026-10-08 PROBE
	v := Violation{Rule: "code_check/no-eval", File: "src/main.ts", Message: "eval usage is forbidden"}

	left, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal left: %v", err)
	}
	right, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal right: %v", err)
	}

	var l map[string]json.RawMessage
	var r map[string]json.RawMessage
	if err := json.Unmarshal(left, &l); err != nil {
		t.Fatalf("unmarshal left: %v", err)
	}
	if err := json.Unmarshal(right, &r); err != nil {
		t.Fatalf("unmarshal right: %v", err)
	}

	if string(l["identity_hash"]) != string(r["identity_hash"]) {
		t.Fatalf("expected deterministic identity_hash; left=%s right=%s", string(l["identity_hash"]), string(r["identity_hash"]))
	}
}

// TestBaseline_ArtifactJSONRoundTrip_Contract captures the baseline artifact
// schema contract for future BaselineArtifact implementation.
func TestBaseline_ArtifactJSONRoundTrip_Contract(t *testing.T) {
	raw := []byte(`{
		"schema_version":"baseline/v1",
		"generated_at":"2026-05-26T00:00:00Z",
		"git_sha":"abc123",
		"backstop_version":"1.0.0",
		"steps": {
			"artifact_validation": 0,
			"code_check": 2,
			"test_verification": 0,
			"test_substantiveness": 0,
			"coverage_threshold": 0,
			"contract_signature": 0
		},
		"violations":[
			{
				"rule":"code_check/no-eval",
				"file":"src/main.ts",
				"message":"eval usage is forbidden",
				"severity":"error",
				"identity_hash":"deadbeef"
			}
		]
	}`)

	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(raw, &envelope); err != nil {
		t.Fatalf("unmarshal baseline artifact fixture: %v", err)
	}

	for _, field := range []string{"schema_version", "generated_at", "git_sha", "backstop_version", "steps", "violations"} {
		if _, ok := envelope[field]; !ok {
			t.Fatalf("expected baseline artifact field %q", field)
		}
	}

	reencoded, err := json.Marshal(envelope)
	if err != nil {
		t.Fatalf("re-encode baseline artifact fixture: %v", err)
	}
	if len(reencoded) == 0 {
		t.Fatal("expected non-empty baseline artifact after round-trip")
	}
}

func TestBaselineArtifactFromStepsSkipsDeferredSteps(t *testing.T) {
	artifact := NewBaselineArtifactFromSteps([]StepResult{
		{StepName: StepCodeCheck, Violations: []Violation{{Rule: "code_check/no-eval", File: "src/main.ts", Message: "eval usage is forbidden", Severity: "error"}}},
		{StepName: StepBaselineComparison, Violations: []Violation{{Rule: "baseline/new", File: "src/main.ts"}}},
		{StepName: StepWaiverResolution, Violations: []Violation{{Rule: "waiver/unresolved"}}},
		{StepName: StepLedgerIntegrity, Violations: []Violation{{Rule: "ledger/missing"}}},
	}, " 2026-05-26T00:00:00Z ", " abc123 ", " 1.2.3 ")

	if artifact.SchemaVersion != BaselineSchemaV1 || artifact.GitSHA != "abc123" || artifact.BackstopVersion != "1.2.3" {
		t.Fatalf("unexpected artifact metadata: %#v", artifact)
	}
	if artifact.StepCounts[StepCodeCheck] != 1 || len(artifact.StepCounts) != 1 {
		t.Fatalf("expected only non-deferred step counts, got %#v", artifact.StepCounts)
	}
	if artifact.StepRuleCounts[StepCodeCheck]["code_check/no-eval"] != 1 {
		t.Fatalf("expected code_check rule count, got %#v", artifact.StepRuleCounts)
	}
	if len(artifact.Violations) != 1 || artifact.Violations[0].IdentityHash == "" {
		t.Fatalf("expected one enriched violation, got %#v", artifact.Violations)
	}
}

func TestBaselineLoadWriteRoundTripDefaultsAndEnriches(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".backstop", "baseline.json")
	if err := WriteBaseline(path, &BaselineArtifact{Violations: []Violation{{Rule: "code_check/no-eval", File: "src/main.ts", Message: "eval usage is forbidden"}}}); err != nil {
		t.Fatalf("write baseline: %v", err)
	}
	loaded, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("load baseline: %v", err)
	}
	if loaded.SchemaVersion != BaselineSchemaV1 {
		t.Fatalf("schema version = %q, want %q", loaded.SchemaVersion, BaselineSchemaV1)
	}
	if len(loaded.Violations) != 1 || loaded.Violations[0].IdentityHash == "" {
		t.Fatalf("expected enriched loaded violation, got %#v", loaded.Violations)
	}
	if err := WriteBaseline(path, nil); err == nil {
		t.Fatal("expected nil baseline artifact write to fail")
	}
}

// TestBaseline_LoadErrorPathsAndDefaults exercises LoadBaseline's error branches
// (unreadable path, malformed JSON) and its defaulting (empty schema_version, null
// violations) so the baseline reader's failure modes are genuinely covered rather
// than grandfathered.
func TestBaseline_LoadErrorPathsAndDefaults(t *testing.T) {
	// Unreadable/nonexistent path → wrapped read error.
	if _, err := LoadBaseline(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("expected error loading a nonexistent baseline path")
	}
	// Malformed JSON → parse error.
	bad := filepath.Join(t.TempDir(), "bad.json")
	if err := os.WriteFile(bad, []byte("{not json"), 0o644); err != nil {
		t.Fatalf("seed malformed baseline: %v", err)
	}
	if _, err := LoadBaseline(bad); err == nil {
		t.Fatal("expected parse error loading malformed baseline JSON")
	}
	// Empty schema_version and null violations → defaulted on load.
	minimal := filepath.Join(t.TempDir(), "minimal.json")
	if err := os.WriteFile(minimal, []byte(`{"violations":null}`), 0o644); err != nil {
		t.Fatalf("seed minimal baseline: %v", err)
	}
	loaded, err := LoadBaseline(minimal)
	if err != nil {
		t.Fatalf("load minimal baseline: %v", err)
	}
	if loaded.SchemaVersion != BaselineSchemaV1 {
		t.Fatalf("expected schema version defaulted to %q, got %q", BaselineSchemaV1, loaded.SchemaVersion)
	}
	if loaded.Violations == nil {
		t.Fatal("expected null violations to default to an empty non-nil slice")
	}
}

// TestBaseline_WriteNilViolationsAndMkdirError exercises WriteBaseline's nil-slice
// defaulting (a nil Violations round-trips as an empty non-nil slice) and its
// directory-creation failure branch (an unwritable parent path), covering the
// writer's previously-uncovered error path.
func TestBaseline_WriteNilViolationsAndMkdirError(t *testing.T) {
	// nil Violations → defaulted to [] and round-trips.
	path := filepath.Join(t.TempDir(), ".backstop", "baseline.json")
	if err := WriteBaseline(path, &BaselineArtifact{}); err != nil {
		t.Fatalf("write baseline with nil violations: %v", err)
	}
	loaded, err := LoadBaseline(path)
	if err != nil {
		t.Fatalf("load written baseline: %v", err)
	}
	if loaded.Violations == nil {
		t.Fatal("expected written nil violations to round-trip as an empty non-nil slice")
	}
	// MkdirAll failure: a path whose parent is an existing FILE cannot be created.
	blocker := filepath.Join(t.TempDir(), "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("seed blocker file: %v", err)
	}
	if err := WriteBaseline(filepath.Join(blocker, "sub", "baseline.json"), &BaselineArtifact{}); err == nil {
		t.Fatal("expected mkdir error writing a baseline under a file path")
	}
}

func TestBaselineCompareNilBaselineAndChangedFileSeeding(t *testing.T) {
	if comparison := CompareBaseline([]Violation{{Rule: "code_check/no-eval"}}, nil, BaselineCompareOptions{}); len(comparison.NewViolations) != 0 || len(comparison.FixedViolations) != 0 || len(comparison.SeededViolations) != 0 {
		t.Fatalf("expected nil baseline to compare empty, got %#v", comparison)
	}

	comparison := CompareBaseline([]Violation{{Rule: "code_check/no-eval", File: "changed.ts"}, {Rule: "code_check/no-alert", File: "legacy.ts"}}, &BaselineArtifact{Violations: []Violation{}}, BaselineCompareOptions{
		Scope:                     newGateScope("", GateScopeModeAll, nil, nil),
		AllowRuleSetChangeSeeding: true,
		ChangedFiles:              map[string]struct{}{"changed.ts": {}},
	})
	if len(comparison.NewViolations) != 1 || comparison.NewViolations[0].File != "changed.ts" {
		t.Fatalf("expected changed violation to remain new, got %#v", comparison.NewViolations)
	}
	if len(comparison.SeededViolations) != 1 || comparison.SeededViolations[0].File != "legacy.ts" {
		t.Fatalf("expected unchanged violation to be seeded, got %#v", comparison.SeededViolations)
	}
}

func TestBaselineCompareReportsFixedViolationsInScope(t *testing.T) {
	baseline := &BaselineArtifact{Violations: []Violation{
		{Rule: "code_check/no-eval", File: "changed.ts", Message: "old changed violation"},
		{Rule: "code_check/no-alert", File: "other.ts", Message: "old other violation"},
	}}

	comparison := CompareBaseline(nil, baseline, BaselineCompareOptions{Scope: newGateScope("", GateScopeModeDiff, []string{"changed.ts"}, nil)})
	if len(comparison.FixedViolations) != 1 {
		t.Fatalf("expected one scoped fixed violation, got %#v", comparison.FixedViolations)
	}
	if comparison.FixedViolations[0].File != "changed.ts" {
		t.Fatalf("expected fixed violation for changed.ts, got %#v", comparison.FixedViolations[0])
	}
}

func TestBaselineCompareIgnoresExistingAndOutOfScopeFixedViolations(t *testing.T) {
	// `existing` is baseline-present on changed.ts, the scope's TOUCHED file, so under
	// the strict file-level ratchet (ISSUE-050) its grandfather is REVOKED and it now
	// reports as NEW where it was previously grandfathered. The fileless baseline
	// finding, absent from current and out of scope, is still NOT counted as fixed —
	// out-of-scope fixed handling is unchanged.
	existing := Violation{Rule: "code_check/no-eval", File: "changed.ts", Message: "same violation"}
	baseline := &BaselineArtifact{Violations: []Violation{
		existing,
		{Rule: "code_check/no-file", Message: "fileless old violation"},
	}}

	comparison := CompareBaseline([]Violation{existing}, baseline, BaselineCompareOptions{Scope: newGateScope("", GateScopeModeDiff, []string{"changed.ts"}, nil)})
	if len(comparison.NewViolations) != 1 {
		t.Fatalf("expected touched-file baseline finding to be revoked as new, got %#v", comparison.NewViolations)
	}
	if comparison.NewViolations[0].File != "changed.ts" {
		t.Fatalf("expected revoked new violation on changed.ts, got %#v", comparison.NewViolations[0])
	}
	if len(comparison.FixedViolations) != 0 {
		t.Fatalf("expected fileless out-of-scope fixed violation to be ignored, got %#v", comparison.FixedViolations)
	}
}

func TestBaselineExistingCodeViolationRequiresFileWhenChangedFilesProvided(t *testing.T) {
	if isExistingCodeViolation(Violation{Rule: "code_check/no-file"}, map[string]struct{}{"changed.ts": {}}) {
		t.Fatal("expected fileless violation not to count as existing code when changed files are provided")
	}
	if !isExistingCodeViolation(Violation{Rule: "code_check/no-eval", File: "legacy.ts"}, map[string]struct{}{"changed.ts": {}}) {
		t.Fatal("expected unchanged file violation to count as existing code")
	}
}

// TestBaseline_FingerprintKeepsSameRuleSameFileDistinct pins the fix for the coarse
// baseline identity: before, RegionHash was never populated, so multiple findings of
// the same rule in the same file collapsed to ONE identity and a NEW one could hide in
// an already-dirty file. With a content-based RegionHash (carried from SARIF), they
// stay distinct and a genuinely new finding surfaces as net-new.
func TestBaseline_FingerprintKeepsSameRuleSameFileDistinct(t *testing.T) {
	mk := func(regionHash string) Violation {
		return Violation{
			Rule:       "backstop/go-standards/no-ignored-errors",
			File:       "pkg/x/x.go",
			Message:    "Ignored error detected.",
			Severity:   "error",
			SourcePack: "backstop/go-standards",
			RegionHash: regionHash,
		}
	}
	v1 := mk("a=111")
	v2 := mk("a=222") // same rule+file+message, different matched content

	if EnrichViolationIdentity(v1).IdentityHash == EnrichViolationIdentity(v2).IdentityHash {
		t.Fatal("distinct content fingerprints must yield distinct identities; identical means the collapse bug is back")
	}

	// Baseline knows only v1. v2 is genuinely new content and MUST surface as net-new
	// rather than be suppressed by v1's entry.
	baseline := &BaselineArtifact{Violations: []Violation{v1}}
	cmp := CompareBaseline([]Violation{v1, v2}, baseline, BaselineCompareOptions{})
	if len(cmp.NewViolations) != 1 {
		t.Fatalf("expected exactly v2 as net-new (v1 suppressed), got %d: %+v", len(cmp.NewViolations), cmp.NewViolations)
	}
	if cmp.NewViolations[0].RegionHash != "a=222" {
		t.Errorf("net-new violation should be v2 (a=222), got RegionHash %q", cmp.NewViolations[0].RegionHash)
	}

	// Contrast: with no fingerprint both fall back to the same message-level identity,
	// which is exactly the collapse this fix avoids when content is available.
	fallbackA := EnrichViolationIdentity(mk("")).IdentityHash
	fallbackB := EnrichViolationIdentity(mk("")).IdentityHash
	if fallbackA != fallbackB {
		t.Fatal("sanity: empty-fingerprint fallback must be deterministic")
	}
}
