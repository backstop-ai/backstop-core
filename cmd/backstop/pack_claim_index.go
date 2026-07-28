package main

// ISSUE-098 (CLM-001/003/006): deriving the pack-claim half of the mandated-test
// vocabulary from the installed pack manifests.
//
// This lives in cmd/backstop beside mergeSourceClassifier and mergeTestNameMatcher,
// following the established convention: manifest reading happens where the manifests are
// visible, and pkg/gate receives an already-derived value it can consume without knowing
// anything about pack structure.

import (
	"github.com/backstop-ai/backstop-core/pkg/gate"
	"github.com/backstop-ai/backstop-core/pkg/pack"
)

// mergePackClaimIndex builds the gate.PackClaimIndex from the ALREADY-LOADED installed
// pack manifests. It indexes exactly the two surfaces ParseManifestFile enforces one
// claim-id namespace over — Content.Ruleset.Rules[].Claims[].ID and
// ToolConfig[].Claims[].ID — and nothing else.
//
// The line is drawn by the manifest, not by taste: both surfaces hold the same Claim
// type, and load-time validation requires every claim to declare a positive AND negative
// fixture pair. A rule id, scaffold id, engine name or tool_config entry id carries no
// fixtures of its own, so admitting one would let a mandated name resolve against
// something with no falsifier behind it — the vacuous-presence failure mode this
// dimension must not have.
//
// The walk is purely structural: no pack-name literal, no id-shape regex, no language
// branch. A pack in any language indexes identically, which is the thin-executor
// requirement (CLM-001). Empty ids are skipped, nil manifests are skipped rather than
// dereferenced, and on a cross-pack claim-id collision the first writer wins — ids are
// unique WITHIN a pack but not across packs, and packs arrive in the deterministic
// declared order, so a collision costs only label precision, never presence, and is
// never an error (CLM-006).
func mergePackClaimIndex(packs []*pack.Manifest) gate.PackClaimIndex {
	index := gate.PackClaimIndex{}
	for _, manifest := range packs {
		if manifest == nil {
			continue
		}
		for _, rule := range manifest.Content.Ruleset.Rules {
			for _, claim := range rule.Claims {
				addPackClaim(index, claim.ID, manifest.NormalizedName+":"+rule.ID)
			}
		}
		for _, entry := range manifest.ToolConfig {
			for _, claim := range entry.Claims {
				addPackClaim(index, claim.ID, manifest.NormalizedName+":tool_config/"+entry.ID)
			}
		}
	}
	return index
}

// addPackClaim records a claim id under its evidence label, skipping empty ids and
// preserving the first label seen for a duplicated id.
func addPackClaim(index gate.PackClaimIndex, claimID, label string) {
	if claimID == "" {
		return
	}
	if _, exists := index[claimID]; exists {
		return
	}
	index[claimID] = label
}
