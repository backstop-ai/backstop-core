---
title: "SPEC-072 Non-Go Provides+Signature Contracts Block Gate on YAML and Mermaid Files"
schema_version: issue/v1

issue:
  id: ISSUE-200
  title: "SPEC-072 Non-Go Provides+Signature Contracts Block Gate on YAML and Mermaid Files"
  type: bug
  status: ready
  created: "2026-09-03"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: build
  test_command: "./bin/backstop gate --file docs/_data/content-inventory.yml && ./scripts/verify-public-product-model.sh"

implementation:
  summary: >
    Edit `specs/SPEC-072-public-product-model.spec.md` to convert five unenforceable
    non-Go `provides`+signature contract entries into `consumes` of
    `scripts/verify-public-product-model.sh`, following the pattern already used for
    `content-topology.yml` and `evidence-inventory.yml` in SPEC-072 v1.0.9 (ISSUE-053).
    Bump `spec_version` to `1.0.10` and add a version history entry. No changes to
    `docs/_data/content-inventory.yml` rows, `scripts/tests/public-product-model/verify-content-inventory.sh`
    pins, any verifier logic, visitor-page copy, or IA.
  package: specs

requirements:
  - id: REQ-001
    text: >
      SPEC-072's contract entry for `docs/_data/content-inventory.yml` must be converted
      from `provides` with `signature: "sources[]"` to a `consumes` of
      `scripts/verify-public-product-model.sh` (name: `legacy_content_disposition_inventory`,
      kind: variable), matching the pattern already used for `content-topology.yml` and
      `evidence-inventory.yml`. After the change, `backstop gate --file
      docs/_data/content-inventory.yml` must not raise a `contract_signature` failure for
      `legacy_content_disposition_inventory` / `sources[]`. The closed world of exactly 6
      sources and 31 unique units must not be modified; `verify-content-inventory.sh` pins
      remain unchanged.
  - id: REQ-002
    text: >
      SPEC-072's contract entry for `docs/_data/product-model.yml` must be converted from
      `provides` with prose signature
      `"concepts[] + architecture_views[] + boundaries[state|explanation_markdown|continuation|guarantee_denial_markdown]"`
      to a `consumes` of `scripts/verify-public-product-model.sh` (name:
      `canonical_product_model`, kind: variable). After the change, `backstop gate --file
      docs/_data/product-model.yml` must not raise a `contract_signature` failure for
      `canonical_product_model` when that file appears in a Gate diff.
  - id: REQ-003
    text: >
      SPEC-072's contract entries for `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd`,
      `docs/_diagrams/ARCH-002-enforcement-loop.mmd`, and
      `docs/_diagrams/ARCH-003-ownership-boundaries.mmd` must each be converted from
      `provides` with prose signatures to `consumes` of
      `scripts/verify-public-product-model.sh` (names: `delivery_lifecycle_architecture`,
      `enforcement_loop_architecture`, and `ownership_boundaries_architecture` respectively;
      kind: variable each). After the change, gating any of the three Mermaid files must not
      raise a `contract_signature` failure for the respective architecture symbol.
  - id: REQ-004
    text: >
      The existing Core verifier logic that enforces the content-inventory, product-model, and
      architecture-diagram worlds must remain untouched and must still pass against the
      unchanged corpus. Specifically, `verify_legacy_content_disposition_inventory`,
      `verify_legacy_useful_unit_inventory`, `verify_canonical_product_model_ownership`, and
      `verify_required_architecture_view_inventory` must all pass after the SPEC-072 contract
      edit. No verifier script, no closed-world pin, and no SPEC-072 claim is modified.
  - id: REQ-005
    text: >
      SPEC-073's consume entries for `legacy_content_disposition_inventory` (source:
      `docs/_data/content-inventory.yml`) and `canonical_product_model` (source:
      `docs/_data/product-model.yml`) must remain resolvable after the SPEC-072 edit.
      The Gate must not report new broken-consume errors against SPEC-073 as a result of
      this change. If a `spec_version` bump on SPEC-072 is required for a contract edit to
      an implemented spec, it must be applied and the version history section updated;
      the prior version history must not be silently rewritten.
  - id: REQ-006
    text: >
      No workaround that retains or approximates the `provides`+signature surface is
      permitted: no parallel `.go` file invented to carry a Go struct field matching
      `sources[]`; no `# sources[]` comment injected into any YAML file; no waiver,
      baseline suppression, or `applies-to` narrowing applied to `contract_signature`; no
      pack vendoring or YAML-file exception baked into the CLI; and no change to
      visitor-page copy or IA. The only permitted file change is the SPEC-072 spec file
      itself (contract conversion + spec_version bump + version history).

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      After the fix, `backstop gate --file docs/_data/content-inventory.yml` produces no
      `contract_signature` failure for symbol `legacy_content_disposition_inventory`.
    tests:
      - TestGate_ContentInventoryContractSignatureProbeClears
  - id: CLM-002
    requirement: REQ-001
    text: >
      The closed-world verifier still passes: exactly 6 sources and 31 unique units,
      with no rows added or removed from `docs/_data/content-inventory.yml`.
    tests:
      - verify_legacy_content_disposition_inventory
      - verify_legacy_useful_unit_inventory
  - id: CLM-003
    requirement: REQ-002
    text: >
      After the fix, gating `docs/_data/product-model.yml` produces no `contract_signature`
      failure for symbol `canonical_product_model`.
    tests:
      - TestGate_ProductModelContractSignatureProbeClears
  - id: CLM-004
    requirement: REQ-002
    text: >
      The product-model ownership verifier still passes after the contract conversion,
      confirming the canonical concept and architecture worlds are unchanged.
    tests:
      - verify_canonical_product_model_ownership
      - verify_canonical_product_model_has_no_parallel_truth
  - id: CLM-005
    requirement: REQ-003
    text: >
      After the fix, gating any of the three Mermaid architecture diagram files produces
      no `contract_signature` failure for its respective architecture symbol.
    tests:
      - TestGate_ArchitectureDiagramContractSignatureProbeClearsOnEdit
  - id: CLM-006
    requirement: REQ-003
    text: >
      The architecture view verifier still passes for ARCH-001, ARCH-002, and ARCH-003
      with their exact Mermaid source, owner route/anchor, and required content unchanged.
    tests:
      - verify_required_architecture_view_inventory
  - id: CLM-007
    requirement: REQ-004
    text: >
      The full public-product-model verifier pipeline passes end-to-end after the SPEC-072
      contract edit, with no verifier script, closed-world pin, or SPEC-072 claim changed.
    tests:
      - verify_public_product_model_complete
  - id: CLM-008
    requirement: REQ-005
    text: >
      After the SPEC-072 contract conversion, the Gate reports no new broken-consume errors
      against SPEC-073 for `legacy_content_disposition_inventory` or `canonical_product_model`.
    tests:
      - TestGate_SPEC073ConsumeGraphResolvesAfterSpec072ContractConversion
  - id: CLM-009
    requirement: REQ-006
    text: >
      The SPEC-072 diff contains no parallel Go file, no injected YAML comment, no waiver or
      baseline suppression, no vendored pack bytes, no CLI change, and no visitor-page edit;
      the only changed file is the spec itself.
    tests:
      - TestArtifact_Spec072ContractConversionContainsNoForbiddenWorkarounds

contracts:
  - file: specs/SPEC-072-public-product-model.spec.md
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: legacy_content_disposition_inventory
        kind: variable
      - source: scripts/verify-public-product-model.sh
        name: canonical_product_model
        kind: variable
      - source: scripts/verify-public-product-model.sh
        name: delivery_lifecycle_architecture
        kind: variable
      - source: scripts/verify-public-product-model.sh
        name: enforcement_loop_architecture
        kind: variable
      - source: scripts/verify-public-product-model.sh
        name: ownership_boundaries_architecture
        kind: variable
---

# SPEC-072 Non-Go Provides+Signature Contracts Block Gate on YAML and Mermaid Files

## Problem

`docs/_data/content-inventory.yml` carries a top-level YAML key `sources:` and passes
`scripts/tests/public-product-model/verify-content-inventory.sh` (exactly 6 sources, 31 unique
units). Yet `./bin/backstop gate --file docs/_data/content-inventory.yml` fails with:

```
contract_signature fail
symbol legacy_content_disposition_inventory signature not found or mismatched in
docs/_data/content-inventory.yml: expected "sources[]"
```

SPEC-072 (`status: implemented`) declares, for that file:

```yaml
- file: docs/_data/content-inventory.yml
  provides:
    - name: legacy_content_disposition_inventory
      kind: variable
      signature: "sources[]"
```

The installed Go contracts pack compiler
(`.backstop/packs/backstop-ai/go-contracts/scripts/compile-signature.sh`) translates
`sources[]` into an ast-grep Go struct pattern:

```
struct {
$$$
sources[]
$$$
}
```

That pattern matches a Go struct field declaration, not a YAML key. Running
`ast-grep --lang go` against the YAML file exits 1. Injecting `# sources[]` as a comment
also exits 1. The pattern also matches a real Go struct field `sources []string`, confirming
the compiler assumes Go-typed source. The YAML file cannot satisfy this probe by any content
change that respects the closed-world constraint (exactly 6 sources / 31 units); there is no
conforming edit.

The same defect class affects four other non-Go files declared as `provides` in SPEC-072:

- `docs/_data/product-model.yml` — `canonical_product_model` with signature
  `"concepts[] + architecture_views[] + boundaries[state|explanation_markdown|continuation|guarantee_denial_markdown]"`
- `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd` — `delivery_lifecycle_architecture`
- `docs/_diagrams/ARCH-002-enforcement-loop.mmd` — `enforcement_loop_architecture`
- `docs/_diagrams/ARCH-003-ownership-boundaries.mmd` — `ownership_boundaries_architecture`

Each of these files will trigger the same `contract_signature` failure the moment it appears
in a Gate diff. `enforcement.policy.contract_signature` is `applies-to: new-code`, so on
`main` the files are grandfathered. However, any plan that edits `content-inventory.yml`
(e.g., PLAN-ISSUE-198 retargeting PACK-001..006 and CON-002, or PLAN-ISSUE-193) or
`product-model.yml` (PLAN-ISSUE-195) relights the probe as a blocking Gate red. This is
why preview-branch CI is currently red on PLAN-ISSUE-198.

SPEC-072 already resolves an identical defect class for `content-topology.yml` and
`evidence-inventory.yml` in v1.0.9 (ISSUE-053): those files use `consumes` of the Core
verifier rather than `provides`+signature, so the Go probe never fires. The same resolution
applies here for all five remaining non-Go `provides` entries.

## Solution

Convert all five SPEC-072 non-Go `provides`+signature entries to `consumes` of
`scripts/verify-public-product-model.sh`, following the established v1.0.9 pattern:

```yaml
# before (content-inventory.yml)
- file: docs/_data/content-inventory.yml
  provides:
    - name: legacy_content_disposition_inventory
      kind: variable
      signature: "sources[]"

# after
- file: docs/_data/content-inventory.yml
  consumes:
    - source: scripts/verify-public-product-model.sh
      name: legacy_content_disposition_inventory
      kind: variable
```

Apply the same transformation to `product-model.yml` and the three Mermaid diagram entries.
Bump `spec_version` to `1.0.10` and add a version history entry citing this issue.

No change is permitted to `docs/_data/content-inventory.yml` rows,
`scripts/tests/public-product-model/verify-content-inventory.sh` pins, any verifier
script logic, SPEC-073's consume graph, visitor-page copy, or IA.

The real enforcement of the content-inventory, product-model, and architecture-diagram worlds
stays in `./scripts/verify-public-product-model.sh` and its child scripts; those checks are
unchanged and must still pass.

## Verification

Run `./bin/backstop gate --file docs/_data/content-inventory.yml`. It must exit 0 and must
not mention `legacy_content_disposition_inventory`, `sources[]`, `canonical_product_model`,
or any architecture symbol in a `contract_signature` failure line.

Run `./scripts/verify-public-product-model.sh`. The complete verifier pipeline must pass:
`verify_legacy_content_disposition_inventory`, `verify_legacy_useful_unit_inventory`,
`verify_canonical_product_model_ownership`, `verify_canonical_product_model_has_no_parallel_truth`,
`verify_required_architecture_view_inventory`, and `verify_public_product_model_complete` must
all exit 0 against the unchanged corpus.

The combined test command `./bin/backstop gate --file docs/_data/content-inventory.yml &&
./scripts/verify-public-product-model.sh` is the authoritative acceptance gate.

Gate-level integration tests (`TestGate_ContentInventoryContractSignatureProbeClears`,
`TestGate_ProductModelContractSignatureProbeClears`,
`TestGate_ArchitectureDiagramContractSignatureProbeClearsOnEdit`) run the Gate against the
relevant files in the fixed state and assert that no `contract_signature` result is emitted for
the respective symbol. These tests must not grep YAML content for the string `sources[]`; the
signal is Gate exit code and output, not raw file content.

`TestGate_SPEC073ConsumeGraphResolvesAfterSpec072ContractConversion` runs the Gate against
SPEC-073's scope and confirms no new broken-consume errors are introduced by the SPEC-072 edit.

`TestArtifact_Spec072ContractConversionContainsNoForbiddenWorkarounds` validates the diff
contains only the spec file (no parallel Go, no YAML comment injection, no waiver, no baseline,
no CLI change, no visitor-page edit).

## References

- `specs/SPEC-072-public-product-model.spec.md` v1.0.9 — implemented spec containing the five
  unenforceable non-Go `provides`+signature contract entries.
- `specs/SPEC-073-documentation-semantics-integration.spec.md` v1.1.11 — downstream consumer
  whose `consumes` entries for `legacy_content_disposition_inventory` and
  `canonical_product_model` must remain intact.
- `.backstop/packs/backstop-ai/go-contracts/scripts/compile-signature.sh` — Go contracts pack
  compiler that translates signature strings into ast-grep Go struct patterns; cannot match YAML.
- `scripts/tests/public-product-model/verify-content-inventory.sh` — closed-world pin (6 sources,
  31 units); must remain unchanged.
- `scripts/verify-public-product-model.sh` — Core-owned static verifier that actually enforces
  the content-inventory, product-model, and diagram worlds.
- ISSUE-053 / SPEC-072 v1.0.9 version history entry — precedent for converting non-Go YAML
  `provides` to `consumes` of the verifier (`content-topology.yml`, `evidence-inventory.yml`).
- `docs/_data/content-inventory.yml` — 6-source / 31-unit legacy-content disposition registry;
  closed world (ISSUE-197, ISSUE-199).
- PLAN-ISSUE-198, PLAN-ISSUE-193, PLAN-ISSUE-195 — active plans whose diffs over
  `content-inventory.yml` and `product-model.yml` re-light the `contract_signature` red and
  are blocked by this defect.
