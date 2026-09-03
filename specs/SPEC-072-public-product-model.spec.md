---
title: "Public Product Model"
number: SPEC-072
created: "2026-08-24"
status: implemented
schema_version: spec/v1
spec_version: 1.0.10

implementation:
  summary: >
    BUNDLE-032 Seed 1 only: replace the repository-shaped public documentation model with
    a visitor-journey content contract, then author the final human-readable product copy
    against that contract after the information architecture, product-truth registries, and
    accepted SPEC-073 documentation-semantic contract are stable. This is a contract-level design
    prerequisite only: Seed 1 does not wait for any PLAN-SPEC-073 task, released or installed pack,
    or partial Seed 2 implementation. Core owns four checked-in product-truth registries — content topology,
    canonical product model, public-claim evidence, and legacy-content disposition — plus
    the ten named page sources that consume them. The topology assigns all twelve required
    neighborhoods to exactly one authoritative route; the product model assigns every
    canonical concept and architecture view to exactly one substantive owner; the evidence
    inventory makes consequential claims inspectable against durable, claim-appropriate
    sources; and the boundary registry distinguishes supported behavior, limitations,
    planned work, intentional non-goals, and adjacent guidance. A Core-owned static verifier
    checks these Backstop-specific instances and their source existence. This spec does NOT
    implement or locally duplicate the reusable documentation-semantics pack (Seed 2),
    generated-product-truth pipeline (Seed 3), Jekyll layouts/design-system integration
    (Seed 4), or capability/@UJ acceptance (Seed 5). It adopts no external documentation
    pattern, creates no MCP or machine-only publication, and adds no generalized prose
    quality, writing-style, or prose-LSP mechanism. As the Seed 1 owner dependency
    consumed by SPEC-076, Core also owns the exact source-level journey-link records,
    structured boundary explanation/continuation/denial fields, and provenance-bound
    adoption instruction records that later seeds render and execute; this spec does not
    own their browser journeys, HTML markers, or disposable-repository runner.
  subject: docs/public-product-model

verification:
  level: static
  test_command: ./scripts/verify-public-product-model.sh

contracts:
  - file: docs/_data/content-topology.yml
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: public_content_topology
        kind: variable
  - file: docs/_data/product-model.yml
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: canonical_product_model
        kind: variable
  - file: docs/_data/evidence-inventory.yml
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: public_claim_evidence_inventory
        kind: variable
  - file: docs/_data/content-inventory.yml
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: legacy_content_disposition_inventory
        kind: variable
  - file: docs/_diagrams/ARCH-001-delivery-lifecycle.mmd
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: delivery_lifecycle_architecture
        kind: variable
  - file: docs/_diagrams/ARCH-002-enforcement-loop.mmd
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: enforcement_loop_architecture
        kind: variable
  - file: docs/_diagrams/ARCH-003-ownership-boundaries.mmd
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: ownership_boundaries_architecture
        kind: variable

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-001@2.1.0
    text: >
      Deliver the complete Seed 1 public-product contract for the full-site rethink: all
      ten authoritative page sources, all four checked-in registries, and the static
      verifier named by this spec must be present and mutually consistent. Existing pages,
      headings, navigation, hierarchy, and wording are bootstrap evidence only. Completion
      by retaining the current landing page plus five-document docs shell, by omitting any
      required neighborhood, or by labeling a narrower product-site increment complete is
      PROHIBITED. This requirement governs content responsibility and final copy, not the
      Jekyll layout, visual treatment, deployed routes, or capability journeys owned by
      Seeds 4 and 5.
  - id: REQ-002
    supports:
      - website-expansion:REQ-002@2.0.0
    text: >
      `docs/_data/content-topology.yml` must declare exactly twelve neighborhood records,
      NBR-001 through NBR-012, and exactly ten authoritative page records with these source
      and canonical-path pairs: `docs/index.md` -> `/`; `docs/evaluate.md` -> `/evaluate/`;
      `docs/model.md` -> `/model/`; `docs/adopt.md` -> `/adopt/`; `docs/use-cases.md` ->
      `/use-cases/`; `docs/packs.md` -> `/packs/`; `docs/extend.md` -> `/extend/`;
      `docs/reference.md` -> `/reference/`; `docs/status.md` -> `/status/`; and
      `docs/contributing.md` -> `/contributing/`. Neighborhood ownership must be exactly:
      NBR-001 Discovery -> `/`; NBR-002 What Backstop Is / Is Not and NBR-003 Evaluation ->
      `/evaluate/`; NBR-004 Capabilities, Guarantees, Limits, and Direction -> `/status/`;
      NBR-005 How Backstop Works and NBR-006 Canonical Concepts & Architecture -> `/model/`;
      NBR-007 Use Cases / Adoption Paths -> `/use-cases/`; NBR-008 Pack Ecosystem ->
      `/packs/`; NBR-009 Extend Backstop -> `/extend/`; NBR-010 Reference -> `/reference/`;
      NBR-011 Project Status / Direction -> `/status/`; and NBR-012 Contributing /
      Ecosystem -> `/contributing/`. Each neighborhood appears once and names one owner;
      a page may own several neighborhoods. Primary journey navigation is ordered
      Evaluate, Model, Adopt, Use Cases, Packs, Extend, Reference; Home is reached through
      the wordmark, while Status and Contributing are utility destinations. Each page record must
      also carry the exact `hero_question` literal in the Authoritative page topology table below;
      those ten strings are Seed 1 final copy and downstream presentation may consume but not amend
      them. Route aliases,
      responsive presentation, and generated navigation markup remain Seed 4 concerns.
  - id: REQ-003
    supports:
      - website-expansion:REQ-003@2.0.0
    text: >
      `docs/_data/product-model.yml` must be the one machine-readable index of the human
      canonical model. It must inventory at least these canonical concept territories:
      product category and trust thesis; intent artifacts and their lifecycle; proactive
      and reactive work tracks; plans and bounded execution; standards packs; recipes;
      gates and enforcement policy; baselines and ratchets; waivers; capabilities and user
      journeys; provenance and verification; harness/runtime integration; and ownership /
      trust boundaries. Every concept record must have a stable `concept_id`, name,
      concise definition, durable `source_refs`, related concept IDs, and exactly one
      substantive owner expressed as a canonical route plus anchor. Every architecture
      view must likewise have a stable ID, one owner, durable sources, and a checked-in
      human-readable diagram source. The minimum set is exactly: `ARCH-001`, delivery lifecycle,
      from reactive issue-to-plan and proactive bundle-to-spec-to-plan through implementation,
      validation, and terminal outcomes, sourced at
      `docs/_diagrams/ARCH-001-delivery-lifecycle.mmd` and owned by
      `/model/#delivery-lifecycle`; `ARCH-002`, the enforcement loop across intent artifacts,
      bounded agent execution, pack-declared engines, gate verdict, tests/evidence, and
      provenance feedback, sourced at `docs/_diagrams/ARCH-002-enforcement-loop.mmd` and owned
      by `/model/#enforcement-loop`; and `ARCH-003`, ownership and trust boundaries among Core,
      packs, harness/runtime, and external toolchains, sourced at
      `docs/_diagrams/ARCH-003-ownership-boundaries.mmd` and owned by
      `/model/#ownership-boundaries`. Each diagram is authoritative UTF-8 Mermaid text; Seed 4
      may derive presentation from it but may not substitute an independently editable image.
      Page copy outside the owner may summarize and link to the owner but must not restate a
      competing substantive definition. A missing minimum view, second product-model registry,
      machine-only narrative, MCP publication, or independent agent IA is PROHIBITED.
  - id: REQ-004
    supports:
      - website-expansion:REQ-004@2.0.0
    text: >
      Every consequential capability, guarantee, compatibility, or outcome statement in
      final page copy must reference exactly one `claim_id` from
      `docs/_data/evidence-inventory.yml`. Each claim record must include the exact public
      statement, a `claim_type`, owning route and anchor, mechanism summary, guarantee
      boundary, known limitations, practical adoption implications, relevant direction,
      `statement_markdown`, and durable evidence references. In page source, each consequential
      statement must be exactly one nonempty Markdown block enclosed by the non-nestable,
      non-overlapping paired lines `<!-- backstop-claim: CLAIM-ID -->` and
      `<!-- /backstop-claim -->`. A claim ID occurs in exactly one region, every region resolves
      to exactly one inventory record, and every inventory record resolves back to one region.
      The canonical visible payload of each region, normalized only to LF line endings with one
      terminal newline removed, must equal `statement_markdown`; the record route and anchor must
      match the page and nearest preceding explicit heading ID. Normally the canonical visible
      payload is the complete region interior. The sole exception is the `CLAIM-005` region for
      adjacent-guidance `BOUNDARY-005`: its interior must contain exactly one
      `<!-- backstop-journey-link: JLINK-024 -->` line immediately before the continuation link,
      and canonicalization must delete exactly that marker line plus its terminating LF before
      comparing the remaining bytes to `statement_markdown`. No other comment, marker, whitespace,
      or byte may be discarded during canonicalization. Claim and journey-link markers are
      source-only metadata, not a second rendered narrative. `claim_type` must be one of `mechanism`,
      `runtime-behavior`, `compatibility`, `observed-failure`, or `observed-outcome`.
      Compatibility claims must state separately (a) whether the named tool can operate
      Backstop and (b) which Backstop lifecycle or enforcement guarantees that integration
      does and does not preserve; wording that equates operability with preservation of
      guarantees is PROHIBITED. Non-consequential connective copy does not require a claim
      record, but changing type or wording to evade evidence duties is prohibited.
  - id: REQ-005
    supports:
      - website-expansion:REQ-005@2.0.0
    text: >
      `docs/_data/product-model.yml` must classify every public product-boundary statement
      into exactly one of five states: `supported`, `limitation`, `planned`, `non-goal`, or
      `adjacent-guidance`. Each boundary record must have a stable `boundary_id`, statement,
      owner route and anchor, durable source references, visitor implication, and exactly one
      `claim_id`. Every boundary claim must link back to exactly one boundary, appear in exactly
      one paired Markdown claim region, and satisfy REQ-006's claim-type evidence matrix.
      `supported` permits `mechanism` or `runtime-behavior`; `limitation` permits `mechanism` or
      `observed-failure`; and `planned`, `non-goal`, and `adjacent-guidance` use `mechanism`
      backed respectively by governing work, decision, or boundary sources.
      `supported` must identify the currently shipped mechanism; `limitation` must state
      the current constraint without implying commitment; `planned` must cite a durable
      governing work artifact and must not be described as shipped; `non-goal` must state
      the intentionally unowned responsibility; and `adjacent-guidance` must name the seam,
      explain why Backstop stops there, recommend a continuation path, and explicitly deny
      that the recommendation is a Backstop guarantee. Every boundary record must carry the
      stable structured fields `state`, nonempty `explanation_markdown`, `continuation`, and
      `guarantee_denial_markdown`. For `adjacent-guidance`, `continuation` must be exactly one
      object with a `journey_link_id` resolving to one REQ-008 record, nonempty root-relative
      `route`, explicit `anchor`, and nonempty `label`, and
      `guarantee_denial_markdown` must be nonempty. For `supported`, `limitation`, `planned`,
      and `non-goal`, `continuation` and `guarantee_denial_markdown` must both be null; an
      implementation may not infer any of these fields from prose. The continuation's JLINK
      source route/anchor must equal the boundary owner, its destination and label must equal the
      continuation route/anchor/label, and the boundary's one existing `claim_id` governs every
      non-null structured copy field. Its claim `statement_markdown` must equal the exact
      composition `explanation_markdown`, then for adjacent guidance a blank line, Markdown link
      `[<label>](<route>#<anchor>)`, another blank line, and `guarantee_denial_markdown`.
      `BOUNDARY-005` must use the exact physical claim-region layout defined in the Structured
      boundary fields section: its JLINK-024 source marker is inside CLAIM-005 between the first
      blank line and continuation link, while the canonical visible claim payload excludes only
      that marker line. A boundary
      with zero states,
      multiple states, or prose that contradicts its state must fail verification.
  - id: REQ-006
    supports:
      - website-expansion:REQ-008@2.1.0
    text: >
      `docs/_data/evidence-inventory.yml` must map every consequential claim to at least
      one durable mechanism source of kind `source`, `schema`, `test`, or
      `implementation-commit`. A `runtime-behavior` or `compatibility` claim must
      additionally cite evidence of kind `captured-execution` or
      `reproducible-execution`; an `observed-failure` claim must additionally cite a
      durable `incident` or provenance-bearing `example`; and an `observed-outcome` claim
      must additionally cite a provenance-bearing `example` or `measurement`. Every
      evidence reference must include a stable repository path, immutable commit, or
      published version locator; reproducible executions must include the exact command
      and prerequisites; captured executions and examples must identify their checked-in
      artifact; incidents must identify their durable issue or report; and measurements
      must include provenance, population, period, and method. Conversation memory,
      unnamed observations, mutable branch-only references, and metrics without method are
      inadmissible. `corpus_roles` must point to distinct qualifying entries for at least
      one real failure incident, one failure-to-enforcement before/after example, one
      captured gate result, one source-or-commit trace, and one architecture view. Every
      repository path and commit referenced by the accepted inventory must exist.
  - id: REQ-007
    supports:
      - website-expansion:REQ-010@1.1.0
    text: >
      `docs/_data/content-inventory.yml` must inventory each existing substantive public
      source named by BUNDLE-032 — `docs/index.html`, `docs/getting-started.md`,
      `docs/concepts.md`, `docs/artifact-workflow.md`, `docs/pack-authoring.md`, and
      `docs/cli-reference.md` — exactly once, decomposed into `useful_units`. Each useful unit
      must have a globally unique stable `unit_id`, source path and heading/anchor, concise
      content summary, exactly one disposition from `rewrite`, `merge`, `decompose`, `retain`,
      or `retire`, disposition rationale, and target owner routes. `rewrite` and `merge` require
      exactly one target; `decompose` requires at least two distinct targets; `retain` requires
      exactly one target matching the retained source's owner; and `retire` requires no target.
      The inventory must contain exactly the 31 stable unit IDs and source-topic assignments in
      the Completed inventory and topology rationale table; a missing, additional, duplicate, or
      unmapped unit and invalid disposition-target cardinality are PROHIBITED.
      The Seed 1 implementation plan must sequence final-copy authoring, rewriting, merging,
      decomposition, and retirement after the information architecture, product-truth registries,
      and accepted SPEC-073 documentation-semantic contract are stable. That accepted contract is
      design input; no PLAN-SPEC-073 task, released or installed pack, or partial Seed 2
      implementation may become a Seed 1 execution prerequisite. Final accepted page sources
      must contain their required page responsibilities and no draft placeholder, stale
      Cayman positioning, or substantive duplicate of another page's owned concept. A
      generalized prose-quality, writing-style, or prose-LSP pack is outside this spec and
      cannot become a prerequisite or be added as an implementation shortcut.
  - id: REQ-008
    supports:
      - website-expansion:REQ-002@2.0.0
    text: >
      `docs/_data/content-topology.yml` must declare exactly the 24 stable source-owned
      journey-link records `JLINK-001` through `JLINK-024` in the Journey-link matrix below,
      with no additional `JLINK-*` record. Each record must contain its exact `link_id`, source
      route and explicit source anchor, destination route and explicit destination anchor, and
      nonempty Seed 1-owned link label. The named source page must contain exactly one
      `<!-- backstop-journey-link: JLINK-NNN -->` marker immediately followed by exactly one
      Markdown link whose label equals the record label and whose root-relative destination
      equals `<destination-route>#<destination-anchor>`; the marker and link must occur under
      the record's nearest preceding explicit source anchor. JLINK-009 is one physical source
      link and remains one registry record even though multiple downstream journeys consume it.
      Every source and destination route must be one of REQ-002's ten canonical routes and every
      source and destination anchor must exist exactly once in its Seed 1 page source. Missing,
      additional, duplicate, reordered, wrong-source, wrong-destination, wrong-anchor,
      unmarked, multiply marked, global-navigation-only, or non-root-relative links are
      PROHIBITED. JLINK-024 must occur inside the CLAIM-005 region in the exact BOUNDARY-005
      physical layout defined by REQ-004 and REQ-005; placing that marker outside the claim
      region, between the claim-start marker and explanation, after the continuation link, or
      with any intervening line is PROHIBITED. Seed 1 owns the source link and copy; Seed 4 owns rendered link attributes and
      route/link behavior; Seed 5 may consume these IDs but must not author substitute links.
  - id: REQ-009
    supports:
      - website-expansion:REQ-010@1.1.0
    text: >
      `docs/_data/content-topology.yml` must declare exactly three adoption instruction records,
      `ADOPT-INSTALL`, `ADOPT-CONFIGURE`, and `ADOPT-ENFORCE`, in that order and with the exact
      owners, displayed commands, digests, structured execution, provenance, and postconditions
      in the Adoption instruction matrix below. Every record must contain `instruction_id`,
      `owner_route`, `owner_anchor`, `command_text`, `command_sha256`, `executable`, ordered
      `argv`, `environment`, `working_directory`, `provenance`, `expected_exit_code`, and ordered
      `expected_outputs`. `command_sha256` must be `sha256:` plus the lowercase SHA-256 of the
      exact UTF-8 `command_text` bytes with no trailing newline. `<disposable-root>` is a typed
      runtime placeholder resolved to the newly created disposable Git repository root; it may
      occur only in structured executable, environment, working-directory, provenance-output,
      and expected-output path fields, never in displayed command text. `ADOPT-INSTALL`
      provenance is the exact immutable Go module coordinate
      `github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0`; the later records' provenance
      must point to `ADOPT-INSTALL`'s exact installed output
      `<disposable-root>/.backstop-bin/backstop`. The source page must display each exact
      `command_text` once under its owner anchor. These data must be sufficient for a consumer
      to execute the three records without evaluating displayed text through a shell. Missing,
      additional, duplicate, reordered, digest-mismatched, mutable/unversioned provenance,
      wrong-owner, wrong-executable/argv/environment/working-directory, shell-dependent, or
      postcondition-free records are PROHIBITED. Seed 1 owns instruction structure and copy;
      Seed 4 owns rendered instruction/digest markers; Seed 5 owns disposable execution and
      receipts.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The complete ten-page, four-registry Seed 1 contract passes as one full-site content model.
    tests:
      - verify_public_product_model_complete
  - id: CLM-002
    requirement: REQ-001
    text: Omitting any required page, registry, or neighborhood fails rather than accepting the prior narrow docs-shell shape.
    tests:
      - verify_public_product_model_rejects_narrow_completion
  - id: CLM-003
    requirement: REQ-002
    text: The ten exact source/path pairs and twelve exact neighborhood-to-owner assignments pass.
    tests:
      - verify_content_topology_exact_inventory
  - id: CLM-004
    requirement: REQ-002
    text: A missing, unknown, or multiply owned neighborhood fails and identifies its neighborhood ID.
    tests:
      - verify_content_topology_rejects_invalid_neighborhood_ownership
  - id: CLM-005
    requirement: REQ-002
    text: The primary and utility navigation memberships and primary ordering match the declared visitor-journey model.
    tests:
      - verify_content_topology_navigation_contract
  - id: CLM-006
    requirement: REQ-003
    text: Every required concept territory and architecture view has one stable ID, one owner route/anchor, and durable sources.
    tests:
      - verify_canonical_product_model_ownership
  - id: CLM-007
    requirement: REQ-003
    text: Adding a second substantive owner for a canonical concept or architecture view fails and names both owners.
    tests:
      - verify_canonical_product_model_rejects_duplicate_owner
  - id: CLM-008
    requirement: REQ-003
    kind: absence
    text: No second product-model registry, machine-only narrative, MCP publication, or agent-specific IA is introduced.
    tests:
      - verify_canonical_product_model_has_no_parallel_truth
  - id: CLM-009
    requirement: REQ-004
    text: Every consequential statement resolves to one complete, valid claim record at its owning route and anchor.
    tests:
      - verify_consequential_claim_contract
  - id: CLM-010
    requirement: REQ-004
    text: A consequential statement with no claim ID, an unknown claim type, or missing decision-support field fails with the claim and field.
    tests:
      - verify_consequential_claim_rejects_incomplete_record
  - id: CLM-011
    requirement: REQ-004
    text: A compatibility record passes only when operability and preserved guarantees are stated independently.
    tests:
      - verify_compatibility_claim_separates_operability_and_guarantee
  - id: CLM-012
    requirement: REQ-004
    text: A compatibility record that equates operability with preservation of all Backstop guarantees fails.
    tests:
      - verify_compatibility_claim_rejects_guarantee_equivalence
  - id: CLM-013
    requirement: REQ-005
    text: Records in each of the five allowed product-boundary states pass when all state-specific fields are present and consistent.
    tests:
      - verify_product_boundary_state_matrix_positive
  - id: CLM-014
    requirement: REQ-005
    text: A boundary with no state, several states, an unknown state, or state-contradicting prose fails with its boundary ID.
    tests:
      - verify_product_boundary_state_matrix_negative
  - id: CLM-015
    requirement: REQ-005
    text: Adjacent guidance passes only when it names the seam, rationale, continuation path, and denial of a Backstop guarantee.
    tests:
      - verify_adjacent_guidance_contract
  - id: CLM-016
    requirement: REQ-006
    text: Every accepted claim type carries mechanism evidence and the additional evidence kind required for its type.
    tests:
      - verify_claim_type_evidence_matrix_positive
  - id: CLM-017
    requirement: REQ-006
    text: Each claim type fails when mechanism evidence or its required execution, incident, example, or measurement evidence is missing.
    tests:
      - verify_claim_type_evidence_matrix_negative
  - id: CLM-018
    requirement: REQ-006
    text: Missing repository sources, nonexistent commits, unnamed observations, mutable-only references, and methodless metrics fail with the responsible claim and reference.
    tests:
      - verify_evidence_sources_are_durable_and_resolvable
  - id: CLM-019
    requirement: REQ-006
    text: The evidence corpus contains distinct qualifying entries for all five required corpus roles.
    tests:
      - verify_evidence_corpus_minimum_roles
  - id: CLM-020
    requirement: REQ-006
    text: Removing any required corpus role or pointing it at a nonqualifying entry fails and names the role.
    tests:
      - verify_evidence_corpus_rejects_missing_or_invalid_role
  - id: CLM-021
    requirement: REQ-007
    text: All six legacy public sources are inventoried once with a valid disposition, useful units, rationale, and target owner routes.
    tests:
      - verify_legacy_content_disposition_inventory
  - id: CLM-022
    requirement: REQ-007
    text: A missing, duplicated, or unknown legacy source or a disposition without valid targets fails with the source path.
    tests:
      - verify_legacy_content_disposition_rejects_invalid_entry
  - id: CLM-023
    requirement: REQ-007
    text: The local verifier proves all ten page files, required explicit section IDs, claim-region links, and registry references exist, and rejects literal draft placeholders and known stale Cayman metadata.
    tests:
      - verify_final_copy_structural_completeness
  - id: CLM-024
    requirement: REQ-007
    kind: absence
    text: Seed 1 adds no generalized prose-quality, writing-style, or prose-LSP implementation or prerequisite.
    tests:
      - verify_seed_one_has_no_generalized_prose_system
  - id: CLM-025
    requirement: REQ-004
    text: Every accepted consequential Markdown block is enclosed by one non-nested claim region whose ID, canonical visible statement bytes, route, and anchor exactly match one inventory record; CLAIM-005 passes only with the single embedded JLINK-024 marker line removed by the exact canonicalization rule.
    tests:
      - verify_markdown_claim_region_bijection
      - verify_adjacent_guidance_embedded_jlink_claim_layout
  - id: CLM-026
    requirement: REQ-004
    text: Missing, duplicated, nested, overlapping, unknown, orphaned, byte-mismatched, or non-canonically marked claim regions fail with the page and claim ID; no marker other than the exact CLAIM-005/JLINK-024 line may be stripped from a claim payload.
    tests:
      - verify_markdown_claim_region_rejects_invalid_linkage
      - verify_markdown_claim_region_rejects_invalid_embedded_jlink
  - id: CLM-027
    requirement: REQ-003
    text: ARCH-001, ARCH-002, and ARCH-003 each exist with the exact Mermaid source, owner route/anchor, and required architecture content.
    tests:
      - verify_required_architecture_view_inventory
  - id: CLM-028
    requirement: REQ-003
    text: Removing a required view, duplicating its ID or owner, changing its diagram-source representation, or referencing a missing diagram fails with the architecture ID.
    tests:
      - verify_required_architecture_view_rejects_missing_or_invalid_view
  - id: CLM-029
    requirement: REQ-005
    text: Every boundary has exactly one claim ID, and that claim links back to exactly one boundary, one Markdown region, and complete claim-type evidence.
    tests:
      - verify_product_boundary_claim_bijection
  - id: CLM-030
    requirement: REQ-005
    text: A boundary with no claim, several claims, an unknown claim, a reused claim, an invalid claim type, or incomplete type-specific evidence fails with the boundary and claim IDs.
    tests:
      - verify_product_boundary_rejects_invalid_claim_linkage
  - id: CLM-031
    requirement: REQ-007
    text: All 31 required useful-unit IDs occur exactly once with the required source topic, source locator, summary, valid disposition-specific target set, and rationale.
    tests:
      - verify_legacy_useful_unit_inventory
  - id: CLM-032
    requirement: REQ-007
    text: Duplicate IDs, missing source locators, missing rationale, and every invalid disposition-target cardinality fail with the unit ID.
    tests:
      - verify_legacy_useful_unit_rejects_invalid_record
  - id: CLM-034
    requirement: REQ-002
    text: All ten page records carry exactly their Seed 1-owned hero question literals and no downstream override.
    tests:
      - verify_content_topology_hero_question_matrix
  - id: CLM-035
    requirement: REQ-002
    text: A missing, changed, duplicated, or presentation-owned hero question fails with its canonical route.
    tests:
      - verify_content_topology_rejects_invalid_hero_question
  - id: CLM-036
    requirement: REQ-008
    text: All 24 exact JLINK records resolve one-to-one from their declared source route/anchor markers and Markdown links to their exact destination route/anchor, with JLINK-024 embedded in CLAIM-005 at the one required physical position.
    tests:
      - verify_journey_link_matrix
      - verify_adjacent_guidance_embedded_jlink_claim_layout
  - id: CLM-037
    requirement: REQ-008
    text: A missing, additional, duplicate, reordered, wrong-source, wrong-destination, wrong-anchor, unmarked, multiply marked, global-navigation-only, non-root-relative, or wrongly embedded JLINK fails with its link ID.
    tests:
      - verify_journey_link_matrix_rejects_invalid_edge
      - verify_journey_link_matrix_rejects_invalid_claim_embedding
  - id: CLM-038
    requirement: REQ-005
    text: Every boundary state carries nonempty structured explanation text; adjacent-guidance BOUNDARY-005 carries exactly one JLINK-024-matched continuation object and nonempty guarantee denial whose deterministic visible composition equals CLAIM-005 after removing only the required embedded marker line; and all four other states carry null continuation and denial fields with claim text equal to their explanation.
    tests:
      - verify_product_boundary_structured_field_matrix
      - verify_adjacent_guidance_embedded_jlink_claim_layout
  - id: CLM-039
    requirement: REQ-005
    text: A missing or empty explanation, malformed or JLINK-mismatched continuation, missing adjacent-guidance continuation or denial, claim-composition mismatch, invalid CLAIM-005/JLINK-024 physical layout, or non-null continuation or denial on another state fails with its boundary ID and field.
    tests:
      - verify_product_boundary_rejects_invalid_structured_fields
      - verify_markdown_claim_region_rejects_invalid_embedded_jlink
  - id: CLM-040
    requirement: REQ-009
    text: ADOPT-INSTALL, ADOPT-CONFIGURE, and ADOPT-ENFORCE occur in exact order with the exact owner, command, digest, structured execution, immutable provenance, zero exit expectation, and postconditions in the Adoption instruction matrix.
    tests:
      - verify_adoption_instruction_matrix
  - id: CLM-041
    requirement: REQ-009
    text: A missing, additional, duplicate, reordered, wrong-owner, changed-command, digest-mismatched, mutable-provenance, wrong-executable/argv/environment/working-directory, shell-dependent, or postcondition-free adoption record fails with its instruction ID and field.
    tests:
      - verify_adoption_instruction_matrix_rejects_invalid_record
  - id: CLM-042
    requirement: REQ-009
    text: Each command digest recomputes from the exact no-newline UTF-8 displayed command, and each displayed command occurs exactly once under its declared owner anchor.
    tests:
      - verify_adoption_instruction_source_and_digest_binding
---

# SPEC-072: Public Product Model

## Overview

BUNDLE-032 makes the public site a product surface rather than a styled projection of the
repository. This spec turns that direction into a bounded content contract. It decides where
each visitor question is answered, where the canonical explanation of Backstop lives, how a
public claim is proven, how product boundaries are named, and what happens to the existing
content. It also includes the final-copy work that realizes those responsibilities; it does
not pretend that a topology with placeholder prose is a delivered website model.

The contract is deliberately inspectable. Human-facing Markdown remains the publication
substrate for people and agents, while YAML registries make ownership and evidence relations
deterministically checkable. Those registries are Backstop-specific product truth, not a local
replacement for the documentation-semantics pack. Seed 2 governs reusable semantic rules and
their installed-pack acceptance. Seed 3 governs derived Markdown. Seed 4 turns these sources
into the built, designed site. Seed 5 proves the complete journeys.

No external documentation pattern is adopted by this spec. The topology is derived from
BUNDLE-032's resolved questions, required neighborhoods, current public sources, implementation
history, and durable dogfood incidents. SPEC-071 is historical evidence of the rejected narrow
docs-shell decomposition and carries no live requirement.

## Requirements

The machine-readable frontmatter is the normative acceptance contract. The body tables below
make the topology and record shapes legible; they must remain exactly consistent with the
frontmatter.

### Authoritative page topology

| Source | Canonical path | Seed 1-owned `hero_question` | Authoritative responsibility |
|---|---|---|---|
| `docs/index.md` | `/` | What failure does Backstop prevent? | NBR-001: recognize the failure class, explain why Backstop exists, and route the visitor by question. |
| `docs/evaluate.md` | `/evaluate/` | Your agent already writes the code. | NBR-002 and NBR-003: additive positioning against tools the visitor already uses; the artifact chain as working state; the blocking gate; and the CI-is-too-late fit decision. |
| `docs/model.md` | `/model/` | How does Backstop turn intent into a trustworthy verdict? | NBR-005 and NBR-006: progressively explain how Backstop works, then provide the dense canonical concept and architecture model. |
| `docs/adopt.md` | `/adopt/` | What does a first working adoption require? | Move from evaluation to a working first adoption, state prerequisites and adoption cost, and link rather than duplicate canonical concepts. |
| `docs/use-cases.md` | `/use-cases/` | Which problem-oriented adoption path applies? | NBR-007: provide problem-oriented adoption paths and direct each path into the canonical model, packs, adoption, and reference material it needs. |
| `docs/packs.md` | `/packs/` | Which maintained pack already owns this standard? | NBR-008: expose the pack ecosystem, problem coverage, composition, maintenance state, and selection guidance. |
| `docs/extend.md` | `/extend/` | When should this concern become a pack? | NBR-009: decide whether a concern belongs in a pack and guide extension without duplicating exact reference material. |
| `docs/reference.md` | `/reference/` | What exact interface or behavior do I need? | NBR-010: own exact CLI, artifact, schema, configuration, and behavioral lookup. Seed 3 may generate derived sections into this owner. |
| `docs/status.md` | `/status/` | What is supported, limited, planned, or intentionally outside Backstop? | NBR-004 and NBR-011: own shipped capabilities, guarantees, limitations, plans, non-goals, adjacent guidance, and project direction. |
| `docs/contributing.md` | `/contributing/` | How can I participate in Backstop and its ecosystem? | NBR-012: own repository, governance, contribution, and ecosystem participation paths. |

The wordmark provides Home. Primary navigation is exactly Evaluate, Model, Adopt, Use Cases,
Packs, Extend, Reference. Status and Contributing are utility destinations. Seed 4 owns how
that model renders responsively and whether canonical-path aliases are needed; it may not
silently change the ownership model.

### Completed inventory and topology rationale

The ten-page topology results from reading all six current sources at heading/topic-unit level
against the twelve required neighborhoods and the bundle's eleven future capability journeys. No
route is retained merely because a current file exists. `/evaluate/` combines category definition
with fit and guarantee evaluation because category mistakes distort those decisions. `/model/`
combines progressive operation with the dense canonical architecture so introductory explanations
lead into one owner. `/status/` combines capability, guarantee, boundary state, and direction because
they share one five-state vocabulary and evidence contract. `/packs/` remains separate from
`/extend/` because selecting a maintained pack and deciding to author one are different decisions;
`/reference/` remains separate from `/model/` because exact lookup and conceptual explanation serve
different jobs.

`/adopt/` deliberately owns no numbered neighborhood. It is the procedural bridge required by the
bundle's Adopt Backstop outcome: prerequisites, installation, configuration, first run, and first
diagnosis. `/use-cases/` alone owns NBR-007 and answers which problem-oriented path applies, then
routes into `/adopt/`. Combining them was rejected because it would mix reusable problem selection
with one linear setup procedure and cause the setup sequence to be duplicated as use cases grow.

This is the completed source-unit analysis that the checked-in inventory must preserve and elaborate
with exact source anchors and rationales. Prefixes bind units to sources: `HOME` is
`docs/index.html`, `GET` is `docs/getting-started.md`, `CON` is `docs/concepts.md`, `ART` is
`docs/artifact-workflow.md`, `PACK` is `docs/pack-authoring.md`, and `CLI` is
`docs/cli-reference.md`.

| Unit | Current source topic | Disposition | Target owner route(s) |
|---|---|---|---|
| HOME-001 | Landing failure/category framing | rewrite | `/` |
| HOME-002 | Define / enforce / drift model | decompose | `/`, `/model/` |
| HOME-003 | Composable adoption modes | decompose | `/evaluate/`, `/use-cases/` |
| HOME-004 | Adoption call to action | rewrite | `/adopt/` |
| GET-001 | Before you start | merge | `/adopt/` |
| GET-002 | Project, install, configure, first run | merge | `/adopt/` |
| GET-003 | Failure-to-green walkthrough | decompose | `/adopt/`, `/use-cases/` |
| GET-004 | Troubleshooting | merge | `/reference/` |
| GET-005 | What Backstop did not do | decompose | `/evaluate/`, `/status/` |
| CON-001 | Premise and trust thesis | decompose | `/evaluate/`, `/model/` |
| CON-002 | Packs and zero bundled checks | decompose | `/model/`, `/packs/` |
| CON-003 | Thin-executor distinction | merge | `/model/` |
| CON-004 | Gate, dimensions, severity, policy | decompose | `/model/`, `/reference/` |
| CON-005 | Baselines and waivers | merge | `/model/` |
| CON-006 | Artifacts and system composition | merge | `/model/` |
| ART-001 | Two work tracks | merge | `/model/` |
| ART-002 | CLI creation and ID reservation | decompose | `/adopt/`, `/reference/` |
| ART-003 | Lifecycle states | decompose | `/model/`, `/reference/` |
| ART-004 | Closure and traceability | decompose | `/model/`, `/reference/` |
| ART-005 | Artifact validation and gate integration | merge | `/model/` |
| PACK-001 | Pack definition and selection boundary | decompose | `/packs/`, `/extend/` |
| PACK-002 | Scaffold, manifest, and engine authoring | decompose | `/extend/`, `/reference/` |
| PACK-003 | Rules, claims, fixtures, and tools | decompose | `/extend/`, `/reference/` |
| PACK-004 | Path-filter sharp edge | decompose | `/extend/`, `/status/` |
| PACK-005 | Check, test, findings, iteration | merge | `/extend/` |
| PACK-006 | Publishing and ecosystem continuation | decompose | `/extend/`, `/contributing/` |
| CLI-001 | Conventions, exit codes, JSON, discovery | merge | `/reference/` |
| CLI-002 | Init, doctor, and gate | decompose | `/adopt/`, `/reference/` |
| CLI-003 | Pack commands | merge | `/reference/` |
| CLI-004 | Artifact commands | merge | `/reference/` |
| CLI-005 | Recipe, baseline, waiver, version, commands | merge | `/reference/` |

### Journey-link matrix

These are Seed 1 source-content edges, not Seed 5 scenarios. Each row is one exact
`journey_links[]` record and one marked Markdown link under the source anchor. JLINK-009 is
intentionally one shared physical edge.

| Link ID | Source route/anchor | Destination route/anchor |
|---|---|---|
| JLINK-001 | `/#define-work` | `/evaluate/#failure-fit` |
| JLINK-002 | `/evaluate/#working-state` | `/model/#operating-model` |
| JLINK-003 | `/use-cases/#choose-use-case` | `/evaluate/#fit-decision` |
| JLINK-004 | `/evaluate/#fit-decision` | `/adopt/#install` |
| JLINK-005 | `/model/#product-category` | `/status/#adjacent-guidance` |
| JLINK-006 | `/model/#gates-and-policy` | `/status/#supported-and-limited` |
| JLINK-007 | `/status/#boundary-states` | `/model/#ownership-boundaries` |
| JLINK-008 | `/model/#harness-integration` | `/reference/#compatibility` |
| JLINK-009 | `/reference/#compatibility` | `/status/#adjacent-guidance` |
| JLINK-010 | `/model/#operating-model` | `/reference/#artifact-schema-catalog` |
| JLINK-011 | `/model/#ownership-boundaries` | `/status/#project-boundaries` |
| JLINK-012 | `/adopt/#install` | `/reference/#configuration` |
| JLINK-013 | `/adopt/#verify-enforcement` | `/model/#enforcement-loop` |
| JLINK-014 | `/model/#enforcement-loop` | `/reference/#gate` |
| JLINK-015 | `/use-cases/#choose-use-case` | `/adopt/#adoption-paths` |
| JLINK-016 | `/use-cases/#pack-backed-use-cases` | `/packs/#choose-a-pack` |
| JLINK-017 | `/packs/#installed-pack-catalog` | `/reference/#pack-commands` |
| JLINK-018 | `/packs/#choose-a-pack` | `/status/#pack-direction` |
| JLINK-019 | `/extend/#pack-or-not` | `/reference/#pack-artifact` |
| JLINK-020 | `/extend/#author-a-pack` | `/contributing/#contribution-paths` |
| JLINK-021 | `/model/#provenance-and-verification` | `/reference/#source-traceability` |
| JLINK-022 | `/packs/#installed-pack-catalog` | `/reference/#cli-command-catalog` |
| JLINK-023 | `/reference/#cli-command-catalog` | `/status/#release-history` |
| JLINK-024 | `/status/#adjacent-guidance` | `/contributing/#external-ownership` |

The record's nonempty `label` is final Seed 1 copy and must equal its marked Markdown link text.
Seed 4 may translate the record and source marker into a rendered `data-journey-link-id`, but
neither Seed 4 nor Seed 5 may change the edge or mint a replacement.

### Structured boundary fields

Every boundary record carries the same four stable fields: `state`, nonempty
`explanation_markdown`, `continuation`, and `guarantee_denial_markdown`. The cardinality is exact:

| Boundary state | `explanation_markdown` | `continuation` | `guarantee_denial_markdown` |
|---|---|---|---|
| `supported` | Nonempty | null | null |
| `limitation` | Nonempty | null | null |
| `planned` | Nonempty | null | null |
| `non-goal` | Nonempty | null | null |
| `adjacent-guidance` | Nonempty | Exactly one `{journey_link_id, route, anchor, label}` object | Nonempty |

The adjacent-guidance continuation route is root-relative, its anchor is explicit, and both
resolve within the ten-page Seed 1 topology. Its JLINK source equals the boundary owner and its
JLINK destination and label equal the continuation. The boundary claim's exact
`statement_markdown` is the explanation, blank line, continuation Markdown link, blank line, and
guarantee denial; non-adjacent claim text is the explanation alone. These fields are product truth.
Seed 4 renders stable markers from them; Seed 5 consumes the IDs and cardinality without inferring
semantic structure from prose.

BOUNDARY-005, CLAIM-005, and JLINK-024 have exactly this physical source layout under
`/status/#adjacent-guidance`:

```text
<!-- backstop-claim: CLAIM-005 -->
Backstop stops at an inspectable verdict because external orchestration and organizational enforcement have different owners.

<!-- backstop-journey-link: JLINK-024 -->
[Continue outside Backstop](/contributing/#external-ownership)

That continuation is guidance, not a guarantee provided by Backstop.
<!-- /backstop-claim -->
```

The claim verifier takes the bytes between the paired claim markers, normalizes line endings and
the terminal newline as REQ-004 specifies, then deletes exactly the JLINK-024 marker line and its
terminating LF. The remaining visible Markdown must byte-equal CLAIM-005 `statement_markdown`.
Deleting any other comment or whitespace, placing JLINK-024 outside CLAIM-005, or separating its
marker from the continuation link is invalid. This single source link simultaneously realizes the
structured boundary continuation and the journey edge; it is not duplicated to satisfy either
consumer.

### Adoption instruction matrix

These are the exact three source-owned records in `content-topology.yml`. `command_sha256` hashes
the exact displayed UTF-8 command with no trailing newline. `working_directory` is
`<disposable-root>` for all three records and `expected_exit_code` is `0` for all three.

| Instruction | Owner | Exact `command_text` | Exact `command_sha256` | Structured execution, provenance, and ordered postconditions |
|---|---|---|---|---|
| ADOPT-INSTALL | `/adopt/#install` | `GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0` | `sha256:54523aa4d4d52abcfa3b58816f5cae70ddb58773f9d90a782ebfe3afd4420ced` | `executable: go`; `argv: [install, github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0]`; `environment: {GOBIN: <disposable-root>/.backstop-bin}`; provenance `{kind: go-module-version, coordinate: github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0}`; outputs `[executable-file:<disposable-root>/.backstop-bin/backstop]` |
| ADOPT-CONFIGURE | `/adopt/#configure` | `backstop init` | `sha256:0ebe03e241fa9eddc23c902c17397044d2bc5d38f980405b8c66c024e9d74198` | `executable: <disposable-root>/.backstop-bin/backstop`; `argv: [init]`; `environment: {}`; provenance `{kind: instruction-output, instruction_id: ADOPT-INSTALL, path: <disposable-root>/.backstop-bin/backstop}`; outputs `[file:<disposable-root>/backstop.yml]` |
| ADOPT-ENFORCE | `/adopt/#verify-enforcement` | `backstop gate` | `sha256:85ffedc97983c354f036e2c3a5b71e4e0df3c850c7e2f0511ca3cc05ff2976a3` | `executable: <disposable-root>/.backstop-bin/backstop`; `argv: [gate]`; `environment: {}`; provenance `{kind: instruction-output, instruction_id: ADOPT-INSTALL, path: <disposable-root>/.backstop-bin/backstop}`; outputs `[verdict:exit-0]` |

The relative `GOBIN=./.backstop-bin` is copy a visitor can paste from the disposable repository
root. The structured environment resolves it to the absolute typed placeholder so a downstream
runner can invoke `go` directly without a shell. The configure and enforce records similarly use
the installed absolute executable while preserving concise displayed copy.

### Registry record shapes

`content-topology.yml` contains `pages`, `neighborhoods`, `navigation`, `journey_links`, and
`adoption_instructions`. A page record names
its source, canonical path, role, exact Seed 1-owned hero question, neighborhood IDs, required content blocks, and source material.
A neighborhood record names its stable ID, visitor question, authoritative owner, and required
answer. Navigation stores the exact primary and utility memberships above. Journey links bind
source-owned content edges by ID and exact route/anchor endpoints. Adoption instructions bind
displayed copy to structured, provenance-bearing, digest-checked direct execution.

`product-model.yml` contains `concepts`, `architecture_views`, and `boundaries`. Concept and
architecture records have exactly one route/anchor owner; the three required architecture records
point to the exact authoritative Mermaid sources in REQ-003, and other pages use summaries and links.
Boundary records use exactly one state from the five-state vocabulary, carry the four stable
structured fields and state-specific cardinality in REQ-005, and link one-to-one to evidence-bearing
claims.

`evidence-inventory.yml` contains `claims` and `corpus_roles`. All claim types require mechanism
evidence. Runtime and compatibility claims additionally require execution evidence; observed
failures require an incident or provenance-bearing example; observed outcomes require a
provenance-bearing example or measurement. The five corpus roles are a minimum site-wide mix,
not permission to attach irrelevant evidence to a claim. Paired source markers link one Markdown
block to one claim record by ID, exact normalized bytes, route, and explicit heading anchor.

`content-inventory.yml` decomposes the existing six public sources into stable useful units, then
records each unit's source locator, summary, disposition, target owners, and rationale. `retain` is
allowed only when the complete source already has the right owner and responsibility; it does not
privilege the old file by default.

## Implementation

The planner must preserve this ordering and these ownership seams:

1. Materialize the completed six-source useful-unit analysis above, including exact locators and
   rationales, before retiring or rewriting any source.
2. Materialize the exact content topology and navigation registries, including required page
   blocks, the twelve single-owner neighborhood assignments, all 24 JLINK source edges, and the
   three provenance-bound adoption instructions.
3. Materialize the canonical concept, architecture, boundary, and claim-evidence registries from
   durable repository sources. Do not use conversation recollection as evidence.
4. Implement the Core-owned static verifier for exact inventory completeness, claim-region canonical
   visible-byte and reference integrity (including the sole CLAIM-005/JLINK-024 embedded-marker
   normalization), single structural ownership, the exact JLINK source/destination and marker matrix,
   structured boundary-field cardinality, adoption instruction command/digest/execution/
   provenance binding, state classification, claim-type evidence matrices, durable source existence,
   and minimum corpus roles. The verifier checks deterministic Backstop-specific structure only; it
   does not define a reusable documentation-semantic engine.
5. Author the ten final Markdown sources after the topology, product-truth registries, and accepted
   SPEC-073 documentation-semantic contract are stable, rewriting, merging, decomposing, or retiring
   the prior sources exactly as the content inventory directs. Consume that contract as design input
   only; do not wait for a PLAN-SPEC-073 task, pack release or installation, or partial Seed 2
   implementation. Summaries outside canonical owners link inward and do not become competing definitions.
6. Run the local verifier against the complete Seed 1 contract. Hand the accepted content contract
   to Seed 2 for reusable documentation-semantic enforcement, Seed 3 for derived sections, Seed 4
   for Jekyll/design-system rendering and deployment, and Seed 5 for capability/@UJ traversal
   without absorbing any of their work here.

No implementation pass may introduce a local documentation-semantics rule set, generated-docs
engine, Jekyll layout/design implementation, capability artifact, MCP publication, or generalized
prose system. If a generic missing mechanism is discovered, implementation stops at its owner seam
and records separately governed dependency work.

## Verification

The frontmatter test command runs `./scripts/verify-public-product-model.sh`. The script performs
static, fixture-backed syntax, cardinality, exact-byte, path, and reference verification against the
checked-in registries and page sources. It does not perform generalized documentation-semantic
judgments or require a released pack, declaration, lock, or installation from Seed 2. Claims are
defined in frontmatter. Negative tests mutate isolated fixture copies, never the accepted site corpus, and print
the relevant page, neighborhood, concept, architecture view, boundary, claim, useful unit, JLINK,
adoption instruction, corpus role, source path, or commit in the failure.

Numeric code coverage is not applicable to this static, fixture-backed integration contract: the exact test command
is a Bash dispatcher and fixture verifier, not an instrumented coverage producer, and the spec defines
no executable code-coverage domain. Verification completeness is instead demonstrated by exhaustive
requirement-to-claim and claim-to-mandated-test mapping, isolated positive and negative fixtures, the
exact acceptance command above, and the repository's spec and artifact validation gates.

Claim-region verification compares the canonical visible payload, not unfiltered source metadata.
For CLAIM-005 only, the positive fixture must prove the exact physical layout above and delete only
the embedded JLINK-024 marker line plus its LF before byte comparison. Independent negative fixtures
must move the marker before the explanation, outside the claim, after the link, and add an intervening
line; each must fail with CLAIM-005, JLINK-024, or BOUNDARY-005. A fixture that inserts another HTML
comment into any claim region must remain byte-significant and fail rather than being silently stripped.

The claim-type evidence matrix is exhaustive:

| Claim type | Mechanism evidence | Additional required evidence |
|---|---|---|
| `mechanism` | `source`, `schema`, `test`, or `implementation-commit` | None |
| `runtime-behavior` | `source`, `schema`, `test`, or `implementation-commit` | `captured-execution` or `reproducible-execution` |
| `compatibility` | `source`, `schema`, `test`, or `implementation-commit` | `captured-execution` or `reproducible-execution` |
| `observed-failure` | `source`, `schema`, `test`, or `implementation-commit` | `incident` or provenance-bearing `example` |
| `observed-outcome` | `source`, `schema`, `test`, or `implementation-commit` | provenance-bearing `example` or `measurement` |

The product-boundary matrix is likewise exhaustive: `supported`, `limitation`, `planned`,
`non-goal`, and `adjacent-guidance` pass only with their REQ-005 fields; missing, multiple, unknown,
or contradictory states fail.

## Sharp Edges

- **Ownership can be duplicated through paraphrase.** A second page can avoid copying words while
  still becoming a competing substantive definition. The inventory must distinguish a short linked
  summary from a second owner, and review must inspect meaning as well as identical strings.
- **Evidence can exist without proving the claim.** A source path or commit hash is not automatically
  relevant. Claim records must explain the mechanism edge and include claim-appropriate execution or
  observation evidence; the five corpus roles cannot be satisfied by unrelated artifacts.
- **Compatibility language invites guarantee inflation.** A harness may invoke Backstop while skipping
  required lifecycle steps or ignoring a blocking verdict. Operability and preservation of guarantees
  must remain two explicit answers.
- **Status language decays.** `supported`, `planned`, and `limitation` can become stale as implementation
  changes. Every record needs durable sources, and later derived-doc work must not create a parallel
  status truth.
- **Final copy has contract-level ordering, not cross-plan orchestration.** Writing polished pages
  before the topology, product-truth registries, and accepted SPEC-073 semantic contract stabilize
  recreates the premature SPEC-071 failure. Waiting for PLAN-SPEC-073 tasks, released or installed
  pack bytes, or partial Seed 2 implementation would instead recreate the invalid execution cycle.
- **Jekyll filenames are not the deployment proof.** This spec declares canonical paths as the content
  contract, but Seed 4 owns build, permalink, alias, link, and Pages verification. Passing Seed 1 does
  not claim that the routes are deployed.
- **Current docs contain valuable truth mixed with obsolete topology.** Whole-file retention can preserve
  the wrong owner; whole-file retirement can discard grounded claims. Disposition occurs at useful-unit
  level before file-level action.
- **Markdown linkage can drift invisibly.** A marker may survive while prose moves, splits, or changes.
  Paired regions, exact canonical visible `statement_markdown`, unique IDs, and route/anchor checks
  must fail that drift. CLAIM-005's one embedded JLINK-024 marker is an explicit exception to raw
  source-byte equality, not permission to strip arbitrary comments or normalize other whitespace.
- **The verifier seam can absorb Seed 2.** The local script may check IDs, exact bytes, cardinality,
  paths, and references only. Generalized semantic judgments remain downstream Seed 2 work; adding a
  temporary local copy of a pack rule would create a second policy owner and an invalid prerequisite.
- **A page without a neighborhood can look accidental.** `/adopt/` is deliberate journey infrastructure,
  while `/use-cases/` remains NBR-007's sole owner. Future edits must not assign both pages the same
  neighborhood merely to make the topology symmetrical.
- **Diagram rendering can create parallel truth.** Mermaid text is authoritative; rendered SVG or HTML is
  derived Seed 4 presentation and cannot become independently editable product truth.
- **A journey link can exist without being the declared content edge.** Global navigation or a second
  link to the same route cannot substitute for the one JLINK marker under the exact source anchor.
  Source-marker cardinality and exact destination anchors keep downstream traversal from passing by
  taking an accidental route.
- **Rendered link identifiers belong downstream.** Seed 1 owns JLINK records, source markers, labels,
  and destinations; Seed 4 owns HTML attributes and browser-level link correctness. Teaching the Seed 1
  verifier to inspect built HTML would duplicate the Seed 4 seam.
- **Displayed commands are not execution plans.** Shell-looking text can conceal environment and path
  assumptions. The command digest binds visitor copy, while executable/argv/environment/provenance
  fields give Seed 5 a shell-free execution contract. Neither representation may be reconstructed from
  the other.
- **A mutable install coordinate invalidates the adoption proof.** `latest`, a branch, a local checkout,
  or an already installed binary can make the demonstration pass against bytes the page did not name.
  The exact `@v0.2.0` coordinate and ADOPT-INSTALL output provenance are load-bearing.
- **Boundary prose must not become the schema.** Seed 5 cannot infer explanation, continuation, or denial
  from nearby words. Null-versus-present structured fields are deliberate, and Seed 4 must render them
  without changing their cardinality.

## Integration Contract

Seed 2 consumes the four registries and page responsibilities as real fixtures for the separately
governed documentation-semantics pack; it may not relocate product truth into that pack. Seed 3 may
generate Markdown only into owners named here and must retain one authoritative source per derived
surface. Seed 4 consumes the ten page sources, canonical paths, navigation model, JLINK records,
structured boundary fields, and adoption instructions without changing content ownership; it owns
their rendered markers and behavior. For BOUNDARY-005, Seed 4 must consume the embedded JLINK-024
source marker as the single continuation link and render one anchor carrying both the boundary-
continuation and journey-link identities; it must not render duplicate links from the two registries.
Seed 5 traverses that one rendered JLINK-024 edge, cites the claim/evidence/boundary records, follows
the remaining JLINK edges, and directly executes the structured adoption records without redefining
any of them or treating this spec as the owner of capability scenarios.

## Review Questions

1. Does every one of the twelve bundle neighborhoods map to exactly one of the ten authoritative
   routes, with no territory silently dropped because it shares a page?
2. Can any non-owner page be read as a second substantive definition of a canonical concept rather
   than a brief summary that links to `/model/`?
3. Does each compatibility claim answer both operability and guarantee preservation, including the
   negative boundary, without inference by the reader?
4. For every consequential statement, would removing its mechanism source or required type-specific
   evidence make the static verifier fail with the responsible claim ID?
5. Are the five corpus roles backed by distinct, relevant, durable evidence rather than five labels
   pointing at one generic source?
6. Does every planned item cite governing work while every supported item cites shipped mechanism,
   with no wording that swaps those states?
7. Does adjacent guidance explicitly deny product guarantee and still give a useful continuation path?
8. Did the implementation preserve useful units from all six legacy sources before deleting or
   replacing any file?
9. Was final copy authored against the stable accepted SPEC-073 contract without waiting for any
   PLAN-SPEC-073 task, released or installed pack, or partial Seed 2 implementation?
10. Did Seed 1 avoid absorbing semantic-pack enforcement, generated-doc machinery, presentation,
    capability journeys, MCP publication, or generalized prose tooling?
11. Does every JLINK source marker occur exactly once under its declared source anchor and point to
    the exact destination anchor rather than merely the right page?
12. Does CLAIM-005 use the exact embedded JLINK-024 physical layout, with canonicalization deleting
    only that marker line while preserving one physical continuation link for all consumers?
13. Do boundary consumers read the structured explanation/continuation/denial fields rather than
    parsing page prose, with non-adjacent states retaining explicit nulls?
14. Can a consumer execute all three adoption records in order from structured executable, argv,
    environment, working directory, and provenance without invoking a shell or using a preinstalled
    Backstop binary?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` v0.6.0 — source bundle, Seed 1 partition,
  OQ-1/OQ-4/OQ-5/OQ-7/OQ-9/OQ-10, and DD-1 through DD-5 plus DD-8.
- `specs/SPEC-073-documentation-semantics-integration.spec.md` v1.1.0 — accepted downstream
  semantic contract used as final-copy design input, never as a cross-plan execution dependency.
- `specs/SPEC-071-website-expansion.spec.md` — canceled historical narrow decomposition; evidence only.
- `docs/index.html`, `docs/getting-started.md`, `docs/concepts.md`,
  `docs/artifact-workflow.md`, `docs/pack-authoring.md`, and `docs/cli-reference.md` — current
  substantive public source inventory.
- Commit `33aff3b4810205c85d3893f8c2d2f30c24daed90` — technical-documentation bootstrap.
- Commit `63f70f7e668486202cc1897cfcce94f82769b477` — landing-page and released design-system consumption.
- `issues/ISSUE-182-recipe-literal-placeholder-escaping.issue.md`,
  `issues/ISSUE-183-local-pack-relock-refreshes-stale-install.issue.md`, and
  `issues/ISSUE-184-fixture-path-filter-diagnostics.issue.md` — durable first-party failure evidence.
- `artifacts/capability/v1/schema.json`, `pkg/pack/engine/binding.go`, and
  `pkg/recipe/manifest.go` — product-model and mechanism source material.
- `specs/SPEC-075-static-public-site-design-system.spec.md` v1.0.3 — downstream renderer currently
  pinned to SPEC-072 v1.0.4 and requiring amendment to consume the v1.0.5 embedded-marker layout.
- `specs/SPEC-076-end-to-end-website-capabilities.spec.md` v1.0.2 — downstream consumer currently
  pinned to SPEC-072 v1.0.4 and requiring amendment to consume the single rendered JLINK-024 edge
  produced from the v1.0.5 embedded-marker layout.

## Version History

- **1.0.10** (2026-09-03): The five remaining non-Go `provides`+signature contract
  entries — `content-inventory.yml`, `product-model.yml`, and
  `ARCH-001`/`ARCH-002`/`ARCH-003` — are consumed by
  `./scripts/verify-public-product-model.sh` rather than left as Go-compiler
  false-REDs (ISSUE-200), extending the two-file conversion recorded in the 1.0.9
  entry to the whole non-Go set. Enforcement is unchanged: the content-inventory,
  product-model, and architecture-diagram worlds are still verified by the bash
  verifier pipeline, whose scripts and closed-world pins are untouched. No
  requirement, claim, or mandated test is edited. PLAN-SPEC-072 stays `completed`;
  its `spec_version` pin stays at `1.0.8`.
- **1.0.9** (2026-08-30): JLINK-001 and CLAIM-017 on `/` use the canonical homepage
  section `define-work`. `why-backstop` is not a public homepage anchor.
  YAML `provides` on `content-topology.yml` and `evidence-inventory.yml` are
  consumed by `./scripts/verify-public-product-model.sh` rather than left as
  Go-compiler false-REDs (ISSUE-053). PLAN-SPEC-072 stays `completed`; its
  `spec_version` pin stays at `1.0.8`.
