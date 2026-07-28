package main

// ISSUE-098 Phase 2 (CLM-001/003/006/007/008): deriving the pack-claim index from
// installed pack manifests.
//
// Every fixture here is a REAL pack.yml written to a temp dir and read back through
// pack.ParseManifestFile rather than a hand-constructed struct. That is deliberate: the
// presence semantics this index encodes rest on load-time validation (a claim cannot
// parse without a positive AND negative fixture pair; claim ids are unique across the
// rule and tool_config surfaces), so the tests must ride the same validation the
// production path does.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// parsePackClaimManifest writes body as a pack.yml in a fresh temp dir and parses it
// through the production reader, failing loudly if the fixture does not satisfy
// load-time validation.
func parsePackClaimManifest(t *testing.T, body string) *pack.Manifest {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.yml")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write fixture manifest: %v", err)
	}
	manifest, err := pack.ParseManifestFile(path)
	if err != nil {
		t.Fatalf("fixture manifest must parse (the index's semantics rest on load-time validation): %v", err)
	}
	return manifest
}

// packClaimRuleManifest is a two-rule enforcement manifest, each rule declaring one
// claim. The engine command is empty so nothing here could ever execute.
const packClaimRuleManifest = `name: acme/two-rule-pack
version: 1.0.0
language: neutral
archetype: enforcement
description: Two rules, one claim each — the rule-claim surface of the claim-id namespace.
engines:
  inert:
    command: ""
    input_mode: rule-flags
    input_flag: "--config"
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: alpha-rule
        engine: inert
        risk_class: correctness
        claims:
          - id: alpha-claim
            text: The alpha property holds.
            fixtures:
              positive:
                - testdata/alpha-positive.txt
              negative:
                - testdata/alpha-negative.txt
      - id: beta-rule
        engine: inert
        risk_class: correctness
        claims:
          - id: beta-claim
            text: The beta property holds.
            fixtures:
              positive:
                - testdata/beta-positive.txt
              negative:
                - testdata/beta-negative.txt
`

// TestMergePackClaimIndex_IndexesRuleClaims (CLM-003): every rule claim id is indexed,
// and its label names both the declaring pack and the declaring rule.
func TestMergePackClaimIndex_IndexesRuleClaims(t *testing.T) {
	manifest := parsePackClaimManifest(t, packClaimRuleManifest)

	idx := mergePackClaimIndex([]*pack.Manifest{manifest})

	for claimID, ruleID := range map[string]string{"alpha-claim": "alpha-rule", "beta-claim": "beta-rule"} {
		if !idx.Has(claimID) {
			t.Errorf("claim id %q not indexed; declared rule claims must resolve", claimID)
			continue
		}
		label := idx[claimID]
		if !strings.Contains(label, "acme/two-rule-pack") {
			t.Errorf("label %q for %q does not name the declaring pack", label, claimID)
		}
		if !strings.Contains(label, ruleID) {
			t.Errorf("label %q for %q does not name the declaring rule %q", label, claimID, ruleID)
		}
	}
}

// packClaimToolConfigManifest declares a standalone tool_config entry carrying a claim —
// the second surface ParseManifestFile enforces the one claim-id namespace over.
const packClaimToolConfigManifest = `name: acme/tool-config-pack
version: 1.0.0
language: neutral
archetype: enforcement
description: A standalone tool_config entry carrying a claim — the second claim-id surface.
engines:
  inert:
    command: ""
    input_mode: rule-flags
    input_flag: "--config"
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: carrier-rule
        engine: inert
        risk_class: correctness
tool_config:
  - id: strictness-config
    tool: acme-linter
    file: acme.json
    risk_class: correctness
    claims:
      - id: strictness-claim
        text: The strict setting is enforced by config.
        fixtures:
          positive:
            - testdata/strict-positive.txt
          negative:
            - testdata/strict-negative.txt
`

// TestMergePackClaimIndex_IndexesToolConfigClaims (CLM-003): a tool_config claim id
// resolves exactly as a rule claim id does. Its label distinguishes the surface so a
// diagnostic can point a reader at the right block.
func TestMergePackClaimIndex_IndexesToolConfigClaims(t *testing.T) {
	manifest := parsePackClaimManifest(t, packClaimToolConfigManifest)

	idx := mergePackClaimIndex([]*pack.Manifest{manifest})

	if !idx.Has("strictness-claim") {
		t.Fatalf("tool_config claim id not indexed; it is half of the claim-id namespace")
	}
	label := idx["strictness-claim"]
	if !strings.Contains(label, "acme/tool-config-pack") {
		t.Errorf("label %q does not name the declaring pack", label)
	}
	if !strings.Contains(label, "strictness-config") {
		t.Errorf("label %q does not name the declaring tool_config entry", label)
	}
}

// packClaimNonClaimIdentifierManifest gives its rule id, scaffold id, tool_config entry
// id and engine name strings that are all DISTINCT from the one claim id, so a test can
// tell which identifier the index actually read.
const packClaimNonClaimIdentifierManifest = `name: acme/mixed-identifiers
version: 1.0.0
language: neutral
archetype: enforcement
description: Distinct rule, scaffold, tool_config and engine identifiers around one claim id.
engines:
  engine-identifier:
    command: ""
    input_mode: rule-flags
    input_flag: "--config"
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 1.0.0
    rules:
      - id: rule-identifier
        engine: engine-identifier
        risk_class: correctness
        claims:
          - id: claim-identifier
            text: The only identifier that may resolve.
            fixtures:
              positive:
                - testdata/positive.txt
              negative:
                - testdata/negative.txt
  scaffolds:
    - id: scaffold-identifier
      version: 1.0.0
      tier: skeleton
      path: scaffolds/example
      test_command: "true"
      description: A scaffold whose id must never resolve as a mandated test.
      use_when:
        - never
      assumes:
        - nothing
      pairs_with:
        rules:
          - rule-identifier
tool_config:
  - id: toolconfig-identifier
    tool: acme-linter
    file: acme.json
    risk_class: correctness
    claims:
      - id: toolconfig-claim-identifier
        text: The tool_config claim, which may resolve.
        fixtures:
          positive:
            - testdata/tc-positive.txt
          negative:
            - testdata/tc-negative.txt
`

// TestMergePackClaimIndex_IgnoresNonClaimIdentifiers (CLM-003, the exclusion falsifier):
// ONLY claim ids resolve. A rule id, scaffold id, engine name, pack name or tool_config
// entry id must never become a mandated-test answer — none of them carries a fixture
// pair, so admitting one would let a mandated name resolve against something with no
// falsifier behind it.
func TestMergePackClaimIndex_IgnoresNonClaimIdentifiers(t *testing.T) {
	manifest := parsePackClaimManifest(t, packClaimNonClaimIdentifierManifest)

	idx := mergePackClaimIndex([]*pack.Manifest{manifest})

	for _, claimID := range []string{"claim-identifier", "toolconfig-claim-identifier"} {
		if !idx.Has(claimID) {
			t.Errorf("claim id %q must resolve", claimID)
		}
	}
	for _, nonClaim := range []string{
		"rule-identifier",
		"scaffold-identifier",
		"engine-identifier",
		"toolconfig-identifier",
		"acme/mixed-identifiers",
	} {
		if idx.Has(nonClaim) {
			t.Errorf("non-claim identifier %q resolved as present; only fixture-bearing claim ids may", nonClaim)
		}
	}
	if len(idx) != 2 {
		t.Errorf("index has %d entries, want exactly the 2 claim ids: %v", len(idx), idx)
	}
}

// packClaimAgnosticManifest shares NOTHING with the dogfood packs: not the pack name,
// not the id vocabulary, not a language suffix.
const packClaimAgnosticManifest = `name: acme/widget-lint
version: 2.3.1
language: neutral
archetype: enforcement
description: A pack whose name and claim ids share nothing with the dogfood packs.
engines:
  inert:
    command: ""
    input_mode: rule-flags
    input_flag: "--config"
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 2.3.1
    rules:
      - id: widget-shape
        engine: inert
        risk_class: correctness
        claims:
          - id: widget-shape-check
            text: A widget declares its shape.
            fixtures:
              positive:
                - testdata/widget-positive.txt
              negative:
                - testdata/widget-negative.txt
`

// TestMergePackClaimIndex_IsPackAgnostic (CLM-001/006): the zero-baked falsifier. The
// index is a structural walk — no pack-name literal, no id-shape regex, no language
// suffix knowledge — so a pack named nothing like the dogfood fleet indexes identically.
// Also pins the empty cases CLM-006 names.
func TestMergePackClaimIndex_IsPackAgnostic(t *testing.T) {
	manifest := parsePackClaimManifest(t, packClaimAgnosticManifest)

	idx := mergePackClaimIndex([]*pack.Manifest{manifest})

	if !idx.Has("widget-shape-check") {
		t.Errorf("claim id with no language suffix, in a pack with an unfamiliar name, did not resolve — the index is not structural")
	}
	if label := idx["widget-shape-check"]; !strings.Contains(label, "acme/widget-lint") {
		t.Errorf("label %q does not name the declaring pack", label)
	}

	// CLM-006: zero declared packs yields an empty index, degenerating the presence
	// union to today's source-only behavior. Neither shape may panic.
	for _, tc := range []struct {
		name  string
		packs []*pack.Manifest
	}{
		{name: "nil slice", packs: nil},
		{name: "empty slice", packs: []*pack.Manifest{}},
		{name: "nil manifest in slice", packs: []*pack.Manifest{nil}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			empty := mergePackClaimIndex(tc.packs)
			if len(empty) != 0 {
				t.Errorf("index has %d entries, want 0: %v", len(empty), empty)
			}
			if empty.Has("widget-shape-check") {
				t.Errorf("empty index resolved a name")
			}
		})
	}
}

// issue036MandatedClaimIDs are the five pack claim ids ISSUE-036's CLM-008 mandates.
// They are the restoration trigger for ISSUE-098: today the drift resolver cannot see
// them and reports five false broken promises.
var issue036MandatedClaimIDs = []string{
	"type-signature-go",
	"const-signature-go",
	"var-signature-go",
	"method-signature-go",
	"interface-signature-go",
}

// installedContractsPackName is the pack that actually declares them — the INSTALLED,
// LOCKED pack the gate indexes, NOT the tracked <repoRoot>/packs/contracts fixture,
// which is under no lock entry and has already diverged on both name and version.
const installedContractsPackName = "backstop-ai/go-contracts"

// TestMergePackClaimIndex_InstalledContractsPackDeclaresISSUE036ClaimIDs (CLM-007): the
// restoration-trigger proof at the manifest grain. It resolves through the PRODUCTION
// path — loadInstalledPacks over the repo root — so it reads exactly the manifests the
// gate reads.
func TestMergePackClaimIndex_InstalledContractsPackDeclaresISSUE036ClaimIDs(t *testing.T) {
	root := repoRoot(t)

	manifests, err := loadInstalledPacks(root)
	if err != nil {
		// A load ERROR means the declared fleet is broken — that must fail, never skip.
		t.Fatalf("loadInstalledPacks(%s): %v", root, err)
	}

	var found bool
	for _, m := range manifests {
		if m != nil && m.NormalizedName == installedContractsPackName {
			found = true
			break
		}
	}
	if !found {
		t.Skipf("%s is not installed — run `./bin/backstop pack install` (the pack fleet is not installed)", installedContractsPackName)
	}

	idx := mergePackClaimIndex(manifests)

	for _, claimID := range issue036MandatedClaimIDs {
		if !idx.Has(claimID) {
			// Not an expectation to relax: this means the installed, locked pack no
			// longer declares an id ISSUE-036 promises — a real broken promise.
			t.Errorf("claim id %q mandated by ISSUE-036 CLM-008 is not declared by any installed pack", claimID)
		}
	}
	if idx.Has("no-such-claim-go") {
		t.Errorf("fabricated claim id resolved against the installed fleet; presence must never be vacuous")
	}
}
