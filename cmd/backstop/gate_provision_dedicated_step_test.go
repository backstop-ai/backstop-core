package main

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/check"
)

// ISSUE-112 CLM-002 — TOOL PRESENCE IS A PROPERTY OF THE TOOL, NOT OF THE STEP
// THAT DISPATCHES IT.
//
// The gate provisions over `excludeDedicatedStepRules(packs)`, which strips every
// rule whose engine declares a dedicated-step gate_type (substantiveness,
// contracts, coverage) so those rules are not ALSO dispatched context-free by the
// generic findings step. Using that post-exclusion set as the PROVISIONING set is
// a second, independent bug: a substantiveness pack's ast-grep is then never
// walked at all, so its absence is never noticed and the dedicated step reports a
// clean dimension over an engine that never ran. That is ISSUE-112's own repro
// shape ("any repo + a substantiveness pack on a PATH without ast-grep"), and it
// survives a presence check that only widens WHICH BINDINGS are probed.
//
// These two tests are a pair on purpose. The first pins that provisioning REACHES
// a dedicated-step engine; the second pins that the exclusion still does its own
// job for dispatch. Asserting only the first would be satisfiable by deleting the
// exclusion outright, which would re-introduce the garbage context-free findings
// it exists to prevent.

// dedicatedStepPackWorkspace builds a temp project whose single installed pack
// declares ONE engine with a dedicated-step gate_type (substantiveness) and a
// pinned Provision, bound by its only rule. The pack is written directly into
// .backstop/packs/ rather than installed through `pack add`, following the
// TestBuildGateSteps_PackEngineDispatchFailureYieldsFailStep precedent — the
// packval install pipeline would execute fixtures this test has no use for.
//
// The pinned tool is ast-grep at the version the PRODUCTION trusted-tool allowlist
// carries, so the trust gate passes cleanly and a refusal can only be the PRESENCE
// probe. No convert script and no stdout_artifact are declared: either would
// hard-error inside dispatch for its own unrelated reason and mask which refusal
// actually fired.
func dedicatedStepPackWorkspace(t *testing.T) string {
	t.Helper()
	projectRoot := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectRoot, "backstop.yml"), []byte(
		"project: p\nlanguage: go\npacks:\n  org/subst: \"1.0.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	packRoot := filepath.Join(projectRoot, ".backstop", "packs", "org", "subst")
	if err := os.MkdirAll(filepath.Join(packRoot, "ast-grep"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(packRoot, "ast-grep", "sgconfig.yml"), []byte("ruleDirs: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	manifest := `name: org/subst
version: 1.0.0
language: go
archetype: enforcement
description: Fixture pack whose only engine fills a dedicated gate dimension
engines:
  subst-engine:
    command: ast-grep scan --json
    input_mode: config-file
    input_flag: --config
    scope_kind: file-args
    category: opinion
    gate_type: substantiveness
    provision:
      tool: ast-grep
      version: 0.43.0
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: r1
        standard: standards/go/r1.standard.md
        rule_path: ast-grep/sgconfig.yml
        risk_class: correctness
        engine: subst-engine
        claims:
          - id: c-r1
            text: Rule one.
            fixtures:
              positive:
                - fixtures/positive.go
              negative:
                - fixtures/negative.go
`
	if err := os.WriteFile(filepath.Join(packRoot, "pack.yml"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	return projectRoot
}

// TestGateProvisioning_CoversDedicatedStepEngines proves the gate's provisioning
// reaches an engine whose rules route to a DEDICATED gate step (CLM-002): with the
// pinned ast-grep absent, the pack_engines step must PROBE it and REFUSE.
//
// It drives the REAL wiring — buildGateSteps and the step function it returns —
// rather than calling provisionEngines with a hand-picked pack set, because the
// defect is IN THE CALL SITE. A test that chose its own argument could not see it.
//
// Absence is simulated through the binaryResolver seam, never by mutating PATH,
// and the probe record is asserted alongside the error so the test states WHICH
// tool was looked for rather than only that something failed.
func TestGateProvisioning_CoversDedicatedStepEngines(t *testing.T) {
	projectRoot := dedicatedStepPackWorkspace(t)
	requested := recordingBinaryResolver(t /* nothing present */)

	steps := buildGateSteps(projectRoot, rootAtDir(t, projectRoot))
	var found bool
	for _, step := range steps {
		res := step(context.Background())
		if res.StepName != "pack_engines" {
			continue
		}
		found = true
		if res.Status != "fail" || !res.ConfigErr {
			t.Errorf("a dedicated-step engine whose tool is absent must refuse: got status=%q configErr=%v violations=%#v",
				res.Status, res.ConfigErr, res.Violations)
		}
		if len(res.Violations) == 0 {
			t.Fatal("expected a violation naming the absent tool, got none — a findings dimension that cannot run reported clean")
		}
		if !strings.Contains(res.Violations[0].Message, "ast-grep") {
			t.Errorf("the refusal must NAME the absent tool `ast-grep`, got: %s", res.Violations[0].Message)
		}
	}
	if !found {
		t.Fatal("expected a pack_engines step in the gate step list")
	}
	// THE FINGERPRINT: provisioning actually looked for the dedicated-step engine's
	// tool. Without this the test would also pass if the step failed for an
	// unrelated reason that happened to mention ast-grep.
	if !sliceContains(*requested, "ast-grep") {
		t.Errorf("gate provisioning must PROBE a dedicated-step engine's tool on PATH, probed %v", *requested)
	}
}

// TestGateProvisioning_ExclusionSetIsNotTheProvisioningSet pins BOTH halves of
// CLM-002, which is what stops a later "simplification" from re-collapsing the two
// sets into one:
//
//	(a) excludeDedicatedStepRules STILL strips dedicated-step rules — unchanged, it
//	    exists so those rules are not dispatched context-free by the generic
//	    findings step, a reason unrelated to whether their tool exists.
//	(b) provisioning nonetheless SEES them, so the absent tool is refused.
//
// Asserting only (b) would be satisfiable by deleting the exclusion; asserting only
// (a) is the status quo that hid the defect.
func TestGateProvisioning_ExclusionSetIsNotTheProvisioningSet(t *testing.T) {
	projectRoot := dedicatedStepPackWorkspace(t)
	packs, err := loadInstalledPacks(projectRoot)
	if err != nil {
		t.Fatalf("loading the fixture pack: %v", err)
	}
	if len(packs) != 1 {
		t.Fatalf("fixture must install exactly one pack, got %d", len(packs))
	}
	if len(packs[0].Content.Ruleset.Rules) != 1 {
		t.Fatalf("fixture pack must declare exactly one rule, got %d", len(packs[0].Content.Ruleset.Rules))
	}

	// (a) The exclusion still removes the rule from the DISPATCH set.
	dispatchPacks := excludeDedicatedStepRules(packs)
	if len(dispatchPacks) != 1 {
		t.Fatalf("exclusion must preserve the manifest, got %d", len(dispatchPacks))
	}
	if got := len(dispatchPacks[0].Content.Ruleset.Rules); got != 0 {
		t.Errorf("excludeDedicatedStepRules must still strip the dedicated-step rule from the dispatch set, %d rules survived", got)
	}

	// (b) Provisioning over the FULL installed set still sees it and refuses the
	// absent tool. This is the asymmetry the claim is about: one set for dispatch,
	// the whole set for presence.
	withBinaryResolver(t /* nothing present */)
	provErr := provisionEngines(packs)
	if provErr == nil {
		t.Fatal("provisioning the FULL installed-pack set must refuse the absent dedicated-step tool, got nil")
	}
	var cfgErr *check.ConfigError
	if !errors.As(provErr, &cfgErr) {
		t.Fatalf("the refusal must be a *check.ConfigError (exit 2), got %T: %v", provErr, provErr)
	}
	if !strings.Contains(cfgErr.Error(), "ast-grep") {
		t.Errorf("the refusal must name `ast-grep`, got: %v", cfgErr)
	}

	// CONTROL: provisioning the post-exclusion set — what the gate used to pass —
	// sees no rules at all and therefore cannot refuse anything. This is the defect
	// stated as an executable fact, and it is why (a) and (b) must stay distinct.
	if err := provisionEngines(dispatchPacks); err != nil {
		t.Errorf("control: the post-exclusion set has no rules left to walk, so it cannot refuse; got %v", err)
	}
}
