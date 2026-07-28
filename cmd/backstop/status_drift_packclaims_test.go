package main

// ISSUE-098 Phase 3 (CLM-004/005/007/008): the drift dimension resolving pack-declared
// claim ids, driven through the REAL wiring (driftStepsFor -> buildStatusDriftSteps).
//
// This file reuses the fixture harness in status_drift_gate_test.go — writeFixture,
// issueFixture, driftStepsFor, runStep — and adds only what that harness lacks: a pack
// builder whose manifest DECLARES CLAIMS. installDriftPack's pack deliberately declares
// none, and forking a second copy of the whole harness for one dimension would be worse
// than the coupling.

import (
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/gate"
)

// installClaimPack writes a backstop.yml plus a pack manifest declaring one rule per
// claim id, so the fixture project's installed fleet carries pack-side evidence. It
// mirrors installDriftPack's classification and test-name patterns so source test
// functions still resolve through the normal path.
//
// engineCommand is a parameter so a test can bind the rules' engine to something that is
// not a real binary; the fixture paths are likewise never required to exist. Neither is
// ever executed here — the drift step is a pure filesystem sweep.
func installClaimPack(t *testing.T, root, engineCommand string, claimIDs ...string) {
	t.Helper()

	var rules strings.Builder
	for _, claimID := range claimIDs {
		rules.WriteString(`      - id: ` + claimID + `-rule
        engine: fixture-engine
        risk_class: correctness
        claims:
          - id: ` + claimID + `
            text: Fixture claim ` + claimID + `.
            fixtures:
              positive:
                - testdata/does-not-exist-positive.txt
              negative:
                - testdata/does-not-exist-negative.txt
`)
	}

	manifest := `name: backstop/claim-drift-toolchain
version: 1.0.0
language: go
archetype: enforcement
description: fixture pack declaring claims, for the ISSUE-098 drift-resolution tests
classification:
  source:
    - "**/*.go"
  test:
    - "**/*_test.go"
    - "**/testdata/**"
test_name_patterns:
  - "^\\s*func\\s+(Test\\w+)\\s*\\("
engines:
  fixture-engine:
    command: "` + engineCommand + `"
    input_mode: pattern-arg
    input_flag: -e
    scope_kind: file-args
    category: opinion
    gate_type: findings
content:
  ruleset:
    version: 1.0.0
    rules:
` + rules.String()

	writeFixture(t, root, "backstop.yml", "project: claim-drift-fixture\npacks:\n    backstop/claim-drift-toolchain: local\n")
	writeFixture(t, root, ".backstop/packs/backstop/claim-drift-toolchain/pack.yml", manifest)
}

// TestDriftGate_PackClaimIDResolvesPresent (the ISSUE-098 restoration at fixture grain):
// a closed issue mandating a name that no test FUNCTION provides but an installed pack
// DECLARES as a claim resolves present, so the dimension raises nothing.
func TestDriftGate_PackClaimIDResolvesPresent(t *testing.T) {
	root := t.TempDir()
	installClaimPack(t, root, "", "fixture-claim-alpha")
	writeFixture(t, root, "issues/ISSUE-700-closed.issue.md", issueFixture("ISSUE-700", "closed", "fixture-claim-alpha"))

	block, advisory := driftStepsFor(t, root)

	res := runStep(block)
	if res.Status != "pass" {
		t.Fatalf("a pack-DECLARED claim id must resolve present; status=%q violations=%+v", res.Status, res.Violations)
	}
	if len(res.Violations) != 0 {
		t.Errorf("expected zero block violations, got %+v", res.Violations)
	}
	if a := runStep(advisory); len(a.Violations) != 0 {
		t.Errorf("advisory surface should be empty for a closed fixture, got %+v", a.Violations)
	}
}

// TestDriftGate_FabricatedPackClaimIDStillBlocks (CLM-008): the no-vacuous-presence
// falsifier at the wiring grain. A mandated name that NO installed pack declares and NO
// test function provides still blocks — the union is additive, never a blanket amnesty.
func TestDriftGate_FabricatedPackClaimIDStillBlocks(t *testing.T) {
	root := t.TempDir()
	installClaimPack(t, root, "", "fixture-claim-alpha")
	writeFixture(t, root, "issues/ISSUE-701-closed.issue.md", issueFixture("ISSUE-701", "closed", "fixture-claim-omega"))

	block, _ := driftStepsFor(t, root)

	res := runStep(block)
	if res.Status != "fail" {
		t.Fatalf("an undeclared, unimplemented mandated name must still block; status=%q violations=%+v", res.Status, res.Violations)
	}
	var errs []gate.Violation
	for _, v := range res.Violations {
		if v.Severity == "error" {
			errs = append(errs, v)
		}
	}
	if len(errs) != 1 {
		t.Fatalf("expected exactly one error-severity violation, got %d: %+v", len(errs), res.Violations)
	}
	if !strings.Contains(errs[0].Message, "fixture-claim-omega") {
		t.Errorf("violation must name the absent id, got %q", errs[0].Message)
	}
}

// TestDriftGate_SourceTestFunctionResolutionUnchanged (CLM-004): the source vocabulary
// still decides its own cases. A present Go test function resolves; an absent one still
// blocks. The union disturbs neither direction.
func TestDriftGate_SourceTestFunctionResolutionUnchanged(t *testing.T) {
	t.Run("present source test resolves", func(t *testing.T) {
		root := t.TempDir()
		installClaimPack(t, root, "", "fixture-claim-alpha")
		writeFixture(t, root, "issues/ISSUE-702-closed.issue.md", issueFixture("ISSUE-702", "closed", "TestDriftSourceResolved"))
		writeFixture(t, root, "src/present_test.go", "package src\n\nfunc TestDriftSourceResolved() {}\n")

		block, _ := driftStepsFor(t, root)

		res := runStep(block)
		if res.Status != "pass" || len(res.Violations) != 0 {
			t.Fatalf("an existing test FUNCTION must still resolve present; status=%q violations=%+v", res.Status, res.Violations)
		}
	})

	t.Run("absent source test still blocks", func(t *testing.T) {
		root := t.TempDir()
		installClaimPack(t, root, "", "fixture-claim-alpha")
		writeFixture(t, root, "issues/ISSUE-703-closed.issue.md", issueFixture("ISSUE-703", "closed", "TestDriftNeverWritten"))

		block, _ := driftStepsFor(t, root)

		res := runStep(block)
		if res.Status != "fail" {
			t.Fatalf("a missing test FUNCTION must still block; status=%q violations=%+v", res.Status, res.Violations)
		}
		if !strings.Contains(res.Violations[0].Message, "TestDriftNeverWritten") {
			t.Errorf("violation must name the absent test, got %q", res.Violations[0].Message)
		}
	})
}

// TestDriftGate_PresenceIsDeclarationNotFixtureExecution (CLM-005): the falsifier that
// turns this plan's most contested decision from an argument into a test.
//
// The fixture pack declares fixture-claim-unrunnable under two conditions that make a
// live fixture run impossible: (a) its positive and negative fixture entries name paths
// that exist nowhere on disk, and (b) its rule's engine binds to a command that is not a
// real binary. A pass here is only possible if presence is DECLARATION presence and no
// engine subprocess is spawned. Any future "hardening" that quietly adds a fixture run
// turns this red, which is exactly the alarm that decision deserves.
func TestDriftGate_PresenceIsDeclarationNotFixtureExecution(t *testing.T) {
	root := t.TempDir()
	installClaimPack(t, root, "definitely-not-a-real-engine-binary", "fixture-claim-unrunnable")
	writeFixture(t, root, "issues/ISSUE-704-closed.issue.md", issueFixture("ISSUE-704", "closed", "fixture-claim-unrunnable"))

	// Condition (b) must actually REACH the manifest — if load-time validation refused
	// the bogus engine command, this test would be silently proving less than it claims.
	packs, err := loadInstalledPacks(root)
	if err != nil {
		t.Fatalf("fixture pack must load (condition (b) must reach the manifest): %v", err)
	}
	if len(packs) != 1 {
		t.Fatalf("expected exactly one installed fixture pack, got %d", len(packs))
	}
	if cmd := packs[0].Engines["fixture-engine"].Command; cmd != "definitely-not-a-real-engine-binary" {
		t.Fatalf("engine command = %q; the unrunnable-binary condition did not survive the parse", cmd)
	}

	block, _ := driftStepsFor(t, root)

	res := runStep(block)
	if res.Status != "pass" || len(res.Violations) != 0 {
		t.Fatalf("presence must be DECLARATION presence — no fixture execution, no engine subprocess; status=%q violations=%+v", res.Status, res.Violations)
	}
}

// TestDriftGate_ISSUE036PackClaimsResolveAgainstInstalledFleet (CLM-007): the
// baseline-independent restoration proof. It drives the resolution DIRECTLY over the
// repo and asserts on the raw drift violations rather than shelling out to the gate — a
// green gate proves nothing here while the local baseline grandfathers these five.
//
// If ISSUE-036 is ever retired or rewritten this test's premise dissolves; the correct
// response is to update it openly, never to weaken the assertion.
func TestDriftGate_ISSUE036PackClaimsResolveAgainstInstalledFleet(t *testing.T) {
	root := repoRoot(t)

	packs, err := loadInstalledPacks(root)
	if err != nil {
		// A load ERROR means the declared fleet is broken — fail, never skip.
		t.Fatalf("loadInstalledPacks(%s): %v", root, err)
	}
	var installed bool
	for _, m := range packs {
		if m != nil && m.NormalizedName == installedContractsPackName {
			installed = true
			break
		}
	}
	if !installed {
		t.Skipf("%s is not installed — run `./bin/backstop pack install` (the pack fleet is not installed)", installedContractsPackName)
	}

	res, err := gate.ResolveArtifactStatus(root)
	if err != nil {
		t.Fatalf("ResolveArtifactStatus: %v", err)
	}
	classifier := mergeSourceClassifier(packs)
	matcher, mErr := mergeTestNameMatcher(packs)
	if mErr != nil {
		t.Fatalf("mergeTestNameMatcher: %v", mErr)
	}

	var all []gate.MandatedTest
	for _, rec := range res.Records {
		all = append(all, rec.MandatedTests...)
	}
	all = gate.ResolveMandatedTestPaths(all, root, classifier, matcher)
	packClaims := mergePackClaimIndex(packs)
	present := gate.ResolvePresentTestNames(all, packClaims)

	combined := gate.ClassifyStatusDrift(res.Records, present)
	for _, v := range combined.Violations {
		if v.Severity != "error" {
			continue
		}
		if strings.Contains(v.Message, "ISSUE-036") || strings.Contains(v.File, "ISSUE-036") {
			t.Errorf("ISSUE-036 still reports a broken promise after pack-claim resolution: %s", v.Message)
			continue
		}
		for _, claimID := range issue036MandatedClaimIDs() {
			if strings.Contains(v.Message, claimID) {
				t.Errorf("mandated pack claim id %q still reports ABSENT: %s", claimID, v.Message)
			}
		}
	}

	// The same run must still refuse a fabricated id — presence is never vacuous.
	if present["no-such-claim-go"] {
		t.Errorf("fabricated id resolved present against the real corpus")
	}
	if packClaims.Has("no-such-claim-go") {
		t.Errorf("fabricated id resolved in the index built from the installed fleet")
	}
}
