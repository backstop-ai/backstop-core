---
title: "Documentation Semantics Integration"
number: SPEC-073
created: "2026-08-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.0

implementation:
  summary: >
    BUNDLE-032 Seed 2 only: consume the separately governed documentation-semantics pack
    through Core's remote-pack distribution contract and preserve the release/pin boundary
    shared with the design-system pack. Until the owner artifact exists, Core imports the
    documentation pack's manifest identity and independent repository source coordinate from
    that owner's durable release contract; this spec guesses neither value. The owner owns its
    artifact chain, semantic policy, engines, fixtures, release, and release evidence. Core owns
    a checked-in consumer import of that evidence, exact stable declarations, git-source locks,
    clean remote restoration, installed-byte identity, and an actual-corpus integration proof.
    Manifest identity keys consumer state and derives the installed path; source coordinate
    independently resolves the repository and may differ under SPEC-056's loud non-fatal warning.
  subject: scripts/verify-documentation-semantics-integration.sh

verification:
  level: integration
  coverage_threshold: 80
  test_command: >-
    ./scripts/verify-documentation-semantics-integration.sh

contracts:
  - file: .backstop/website-pack-releases.yml
    provides:
      - name: website_pack_owner_release_imports
        kind: variable
        signature: "releases[role, owner refs, manifest identity, source coordinate, version, git ref, commit, hash, checks, fixture dispatch]"
  - file: backstop.yml
    provides:
      - name: website_released_pack_declarations
        kind: variable
        signature: "packs[releases[*].manifest_identity] -> releases[*].version"
  - file: backstop.lock
    provides:
      - name: website_released_pack_locks
        kind: variable
        signature: "git LockEntry keyed by manifest identity with independent source_coordinate"
  - file: scripts/verify-documentation-semantics-integration.sh
    provides:
      - name: verify_documentation_semantics_integration
        kind: function
        signature: verify_documentation_semantics_integration()
    consumes:
      - source: .backstop/website-pack-releases.yml
        name: website_pack_owner_release_imports
        kind: variable
      - source: backstop.yml
        name: declared_pack_versions
        kind: variable
      - source: backstop.lock
        name: locked_pack_identities
        kind: variable
      - source: .backstop/packs/<documentation-semantics-manifest-identity>/pack.yml
        name: installed_documentation_semantics_manifest
        kind: variable
      - source: .backstop/packs/backstop-ai/backstop-design-system/pack.yml
        name: installed_design_system_manifest
        kind: variable
      - source: docs/_data/content-topology.yml
        name: public_content_topology
        kind: variable
      - source: docs/_data/product-model.yml
        name: canonical_product_model
        kind: variable
      - source: docs/_data/evidence-inventory.yml
        name: public_claim_evidence_inventory
        kind: variable
      - source: docs/_data/content-inventory.yml
        name: legacy_content_disposition_inventory
        kind: variable
      - source: ./bin/backstop
        name: pack_install_pack_check_pack_test_and_gate
        kind: function
  - file: .github/workflows/ci.yml
    provides:
      - name: documentation_semantics_consumer_gate
        kind: variable
        signature: "CI step running scripts/verify-documentation-semantics-integration.sh after clean remote pack install"

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-006@2.1.0
    text: >
      Preserve the three-owner boundary exactly. Backstop-specific truth and instances remain in
      Core; visual policy remains in the design-system owner; reusable documentation semantics
      remain in the owner named by imported release evidence. Seed 2 delivery may change only
      `.backstop/website-pack-releases.yml`, `backstop.yml`, `backstop.lock`,
      `scripts/verify-documentation-semantics-integration.sh`, and `.github/workflows/ci.yml`.
      It must not add or modify `pkg/**`, `cmd/**`, `packs/**`, `docs/**`, either owner repository,
      or owner rule, validator, engine, and fixture files. The verifier may validate bindings,
      orchestrate clean install/check/test/gate commands, mutate an isolated actual-site copy, and
      inspect finding attribution; it may not invoke a semantic engine directly or ship policy.
  - id: REQ-002
    supports:
      - website-expansion:REQ-006@2.1.0
      - website-expansion:REQ-012@1.1.0
    text: >
      `.backstop/website-pack-releases.yml` must contain exactly one owner-evidence import for
      roles `documentation-semantics` and `design-system`. `backstop.yml` keys and lock keys/names
      must equal the imported manifest identities; versions, exact `v<VERSION>` refs, hashes, and
      nonempty `source_coordinate` values must equal the imports; and `source_type` must be `git`.
      Installed paths derive from manifest identity while clean remote restoration resolves the
      independent source coordinate. Equal identity/coordinate passes silently; divergence also
      passes and initial add emits SPEC-056's loud non-fatal warning naming coordinate, identity,
      and install path. Only divergence from owner evidence, mutable/local input, missing state,
      stale installation, or hash/manifest mismatch is prohibited.
  - id: REQ-003
    supports:
      - website-expansion:REQ-006@2.1.0
      - website-expansion:REQ-012@1.1.0
    text: >
      Each durable owner evidence record must bind one owner artifact reference and immutable
      evidence reference to manifest identity, independent source coordinate, exact stable version,
      tag, full release commit SHA, pack content hash, and green `pack check` and `pack test` results
      for the same release. Documentation-semantics evidence must additionally bind every exported
      claim to dispatched positive and negative fixtures with production-relative path-fidelity proof.
      Core verifies evidence completeness, declaration/lock equality, clean remote restoration,
      installed manifest/version/hash, and may rerun check/test on those installed hash-matched bytes.
      Core must not live-resolve tags, clone/rerun the complete owner repository, or generically
      introspect fixture filters. Missing or mixed-release evidence, or ISSUE-184 path-fidelity claims
      without durable owner proof or a named separately governed dependency, must fail.
  - id: REQ-004
    supports:
      - website-expansion:REQ-006@2.1.0
    text: >
      After lock verification, the installed released documentation-semantics pack must execute
      through `./bin/backstop gate` against the actual Seed 1 page sources and registries. The
      clean corpus must have no documentation-semantic blocking findings. In an isolated copy,
      injecting a second substantive definition for a canonical concept on a non-owner page
      without the required canonical-owner relationship must produce a blocking finding that
      names the installed pack's responsible rule and the production-relative page path.
      A second isolated proof must delete the installed tree while retaining configuration, lock,
      and evidence; acceptance must then fail as a missing dependency before any local result counts.
      Unattributed Core-local output or owner-fixture-only execution cannot satisfy the proof.
  - id: REQ-005
    supports:
      - website-expansion:REQ-012@1.1.0
    text: >
      Additional generic documentation-generation machinery, harness/runtime capability, or
      Core primitive discovered while integrating the pack must stop at a separately governed
      dependency seam; Seed 2 must not implement it opportunistically. Generalized prose-quality,
      writing-style, grammar, tone, or prose-LSP enforcement is neither owned nor required by this
      spec and must not block BUNDLE-032. The five-file delivery allowlist in REQ-001 cannot expand
      for an unnamed generic enabler. Any new generic dependency requires a named external owner
      artifact, durable release evidence, and an explicit consumer contract.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Seed 2 passes when its Core delivery is confined to the five allowed consumer surfaces.
    tests: [verify_owner_boundary_accepts_bounded_consumer_surfaces]
  - id: CLM-002
    requirement: REQ-001
    text: Any Seed 2 change under pkg, cmd, packs, or docs fails with the forbidden path.
    tests: [verify_owner_boundary_rejects_forbidden_core_surface]
  - id: CLM-003
    requirement: REQ-001
    text: An embedded owner rule, validator, engine, fixture corpus, or direct semantic-engine invocation fails.
    tests: [verify_owner_boundary_rejects_embedded_policy_surface]
  - id: CLM-004
    requirement: REQ-001
    kind: absence
    text: The design-system dependency remains the visual owner and never becomes the documentation-semantic owner.
    tests: [verify_owner_boundary_rejects_design_system_semantic_ownership]
  - id: CLM-005
    requirement: REQ-002
    text: Both evidence-bound pack matrices pass silently when manifest identity and source coordinate are equal.
    tests: [verify_pin_matrix_accepts_equal_identity_and_coordinate]
  - id: CLM-006
    requirement: REQ-002
    text: A differing evidence-bound identity and coordinate passes, installs under identity, resolves the coordinate, and emits SPEC-056's loud non-fatal add warning.
    tests: [verify_pin_matrix_accepts_divergence_with_spec056_warning]
  - id: CLM-007
    requirement: REQ-002
    text: Missing owner evidence, declaration, lock, or installed surface fails with the role and missing surface.
    tests: [verify_pin_matrix_rejects_missing_surface]
  - id: CLM-008
    requirement: REQ-002
    text: A range, prerelease, branch, path, mutable selector, local source, or nonexact tag fails.
    tests: [verify_pin_matrix_rejects_mutable_or_local_source]
  - id: CLM-009
    requirement: REQ-002
    text: Any config, lock, or evidence identity, coordinate, version, ref, or hash mismatch fails naming both surfaces.
    tests: [verify_pin_matrix_rejects_evidence_binding_mismatch]
  - id: CLM-010
    requirement: REQ-002
    text: A missing install, manifest mismatch, empty hash, or installed-byte drift fails before policy findings execute.
    tests: [verify_pin_matrix_rejects_missing_or_drifted_install]
  - id: CLM-011
    requirement: REQ-003
    text: One complete durable evidence record binds every required field and green owner check to one release.
    tests: [verify_owner_evidence_accepts_complete_single_release_binding]
  - id: CLM-012
    requirement: REQ-003
    text: Missing owner/evidence refs, full SHA, hash, green checks, or fields mixed across releases fail.
    tests: [verify_owner_evidence_rejects_incomplete_or_mixed_release]
  - id: CLM-013
    requirement: REQ-003
    text: A clean remote install from the lock reproduces the evidence-bound installed identity, version, and hash.
    tests: [verify_owner_evidence_clean_remote_install_matches_hash]
  - id: CLM-014
    requirement: REQ-003
    text: Hash-matched installed bytes pass the optional Core rerun of pack check and pack test.
    tests: [verify_owner_evidence_installed_pack_checks_pass]
  - id: CLM-015
    requirement: REQ-003
    text: Durable owner evidence covers dispatched positive and negative fixtures and production-relative path fidelity for every exported semantic claim.
    tests: [verify_owner_evidence_accepts_fixture_dispatch_proof]
  - id: CLM-016
    requirement: REQ-003
    text: Missing or filtered fixture proof fails unless a named external owner dependency supplies durable proof.
    tests: [verify_owner_evidence_rejects_unproven_fixture_dispatch]
  - id: CLM-017
    requirement: REQ-003
    kind: absence
    text: The verifier contains no live tag resolution, full owner checkout/rerun, or generic fixture-filter introspection.
    tests: [verify_owner_evidence_excludes_live_owner_and_generic_fixture_introspection]
  - id: CLM-018
    requirement: REQ-004
    text: The installed released documentation-semantics pack produces no blocking semantic findings against the clean actual Seed 1 page and registry corpus.
    tests: [verify_installed_semantics_gate_accepts_clean_actual_site]
  - id: CLM-019
    requirement: REQ-004
    text: An isolated actual-site mutation adding a second substantive canonical definition on a non-owner page blocks with the installed pack rule and production-relative page path.
    tests: [verify_installed_semantics_gate_blocks_duplicate_substantive_owner]
  - id: CLM-020
    requirement: REQ-004
    text: Deleting the installed tree while retaining evidence, declaration, and lock fails as a missing dependency.
    tests: [verify_installed_semantics_gate_rejects_deleted_pack]
  - id: CLM-021
    requirement: REQ-004
    text: Unattributed local output or an owner-fixture-only run cannot satisfy installed-pack actual-site proof.
    tests: [verify_installed_semantics_gate_rejects_unattributed_or_fixture_only_proof]
  - id: CLM-022
    requirement: REQ-005
    kind: absence
    text: Seed 2 adds no generalized prose-quality, writing-style, grammar, tone, or prose-LSP prerequisite.
    tests: [verify_scope_boundary_excludes_generalized_prose_system]
  - id: CLM-023
    requirement: REQ-005
    text: A newly discovered generic enabler is accepted only with a named owner artifact, durable release evidence, and explicit consumer contract.
    tests: [verify_scope_boundary_accepts_separately_governed_enabler]
  - id: CLM-024
    requirement: REQ-005
    text: Expanding the five-file allowlist for an unnamed enabler or making banked prose work a prerequisite fails.
    tests: [verify_scope_boundary_rejects_absorbed_or_prose_prerequisite]
---

# SPEC-073: Documentation Semantics Integration

## Overview

BUNDLE-032 requires documentation semantics as a known dependency, but first-consumer pressure
does not transfer ownership of that dependency into Core. This spec defines only the consumer
side: which released pack Core consumes, how the two website-policy packs are declared and locked,
what owner-release evidence precedes a pin, and how the installed semantic pack proves itself
against the actual public-product corpus established by SPEC-072.

The documentation pack's manifest identity and independent repository coordinate are imported from
its owner's durable release contract; this spec does not invent either one. The owner decides its
artifact chain, representation, engines, rules, fixtures, and implementation. The visual owner
remains the already released `backstop-ai/backstop-design-system`. Seed 4 owns actual-site
presentation enforcement, while this seed owns the shared released/pinned dependency invariant.

## Requirements

The frontmatter requirements are normative. Their central ownership split is:

| Concern | Owner | Core's Seed 2 authority |
|---|---|---|
| Backstop product truth, topology, evidence, and page instances | `backstop-core` / SPEC-072 | Declare site-specific inputs and supply the actual consumer corpus. |
| Reusable documentation semantics and deterministic enforcement | Owner named by imported release evidence | Pin released bytes and execute their declared interface; never copy the policy. |
| Reusable visual and interaction policy | `backstop-ai/backstop-design-system` | Preserve its released/pinned identity; actual-site visual enforcement is Seed 4. |

### Released/pinned dependency matrix

The following matrix applies independently to both known website packs. Every prohibited state is
covered by a negative claim in frontmatter.

| Surface | Accepted | Prohibited |
|---|---|---|
| Owner evidence | One immutable record binding owner artifacts, identity, coordinate, version, tag, commit, hash, checks, and dispatch proof | Self-asserted, incomplete, mutable, or mixed-release record |
| `backstop.yml` | Exact stable semantic version under the exact manifest identity | Missing entry, prerelease, range, branch, path, or mutable selector |
| `backstop.lock` source | `source_type: git`; evidence-matching independent `source_coordinate`; exact `git_ref` | Local/path source or any value differing from evidence |
| Lock identity | Name and version exactly equal declaration and installed manifest | Any name or version divergence |
| Installed tree | Present at `.backstop/packs/<IDENTITY>` with computed hash equal to nonempty lock hash | Missing tree, empty hash, stale tree, or hash drift |

Identity and coordinate equality passes silently. Divergence also passes: install/runtime state is
keyed by manifest identity, remote restoration uses the coordinate, and the initial add emits the
loud non-fatal SPEC-056 warning. Either value differing from owner evidence is a failure.

The design-system row is a pin-boundary assertion here, not a claim that Seed 2 runs its visual
rules. Conversely, a green documentation-semantics owner fixture suite is necessary but not
sufficient: Core must also run the installed released bytes against the actual site.

## Implementation

The planner must preserve this order:

1. The independently governed owner completes and releases its artifact chain, then publishes durable
   evidence binding one release's identity, coordinate, tag, full commit SHA, content hash, green pack
   checks, and dispatched positive/negative production-path fixtures.
2. Import that evidence into `.backstop/website-pack-releases.yml`; reject mutable, incomplete,
   self-asserted, or mixed-release records.
3. Add the pack by evidence coordinate and version through the normal remote command. Record the
   resulting manifest identity, SPEC-056 warning when applicable, declaration, lock, and installed tree.
4. Reconstruct a clean consumer with `pack install` from the lock and compare the manifest and bytes
   with owner evidence. The verifier may rerun check/test over installed hash-matched pack bytes, but
   never clones or reruns the complete owner repository and never introspects generic fixture filters.
5. In an isolated temporary project copy, inject one second substantive canonical definition onto a
   non-owner page while leaving the canonical-owner relationship absent. Run the installed gate and
   require a blocking finding that names the responsible pack rule and the production-relative page
   path. The accepted working tree is never mutated by this negative proof.
6. Add the verifier to CI after clean remote pack installation and before any job reports website policy
   compliance. Seed 4 may compose its design-system actual-site proof after this step; it must not
   replace this semantic proof.

The verifier may parse declarations, lock entries, installed manifests, hashes, command results,
paths, and findings. It may not decide semantic validity itself. If it contains vocabulary or logic
that detects a competing definition instead of asking the installed pack, it has crossed the owner
boundary.

## Verification

The single integration command runs all claims defined in frontmatter. It uses isolated directories
for clean consumer reconstruction, injected actual-site failure, and delete-installed attribution.
Diagnostics name the evidence role and disagreeing surface for pin failures, the evidence reference
for release failures, and the installed pack rule plus production-relative path for semantic failures.
Core consumes durable owner dispatch evidence; it does not implement ISSUE-184-style generic filter
introspection.

The positive path is deliberately conjunctive:

1. complete single-release owner evidence;
2. exact stable declarations and evidence-bound git locks;
3. clean remote restoration with matching installed manifests and hashes;
4. optional installed-byte pack checks plus durable owner dispatch proof;
5. clean actual-site gate and blocking negative mutation; and
6. delete-installed failure attributed to the missing dependency.

No self-authored transcript, fixture-only run, local relock, unattributed output, or clean gate
without both negative proofs can satisfy that conjunction.

## Sharp Edges

- **Identity and coordinate are different contracts.** Manifest identity owns install/runtime state;
  coordinate owns remote resolution. Divergence is accepted with a warning, while either value
  differing from durable owner evidence is a refusal.
- **Local relock is specifically untrustworthy evidence today.** ISSUE-183 records a successful
  relock that continued to hash and execute stale installed bytes. Seed 2 therefore rejects local
  source type outright, even if the reported hash appears stable.
- **Green fixtures can be outside production scope.** ISSUE-184 shows how fixture names can miss
  include filters. The owner must publish durable dispatch/path-fidelity proof; Core must not close
  that generic gap by introspecting filters locally. The actual-site injection still reports its path.
- **Pack tests and consumer integration prove different things.** Owner tests prove the reusable
  rule contract; Core's installed gate proves distribution, configuration, path scope, and the
  first real consumer. Neither substitutes for the other.
- **A Core orchestration script can quietly become a second semantic engine.** It may inject a known
  bad example and inspect the pack's finding, but it may not itself classify prose or canonical
  ownership.
- **An orchestration script can hide a local lookalike.** The five-file allowlist makes policy-owning
  Core surfaces bounded, while deletion of the installed pack proves unattributed local output cannot
  substitute for the dependency.
- **The two website packs have different acceptance owners.** This seed shares their pin invariant,
  but Seed 4 alone proves token, styling, focus, motion, accessibility, wordmark, and presentation
  rules against the built site.
- **Release evidence can be self-asserted or mixed.** One immutable owner evidence reference must bind
  every field and green result to one release; a loose transcript is not a release contract.
- **Evidence and restoration prove different edges.** Durable owner evidence proves owner release
  governance; a clean coordinate-based install and content hash prove the consumer can reproduce it.
- **The external identity is imported, not guessed.** Renaming identity or coordinate requires a new
  owner release record and coordinated migration; Core cannot ratify an alias on its own.

## Integration Contract

SPEC-072 supplies the actual page sources and four Backstop-specific registries. The external
documentation-semantics pack consumes those instances through its released interface but may not
become their truth owner. SPEC-074 may add generated Markdown to owners declared by SPEC-072; this
integration must evaluate that generated content through the same installed pack once it enters the
actual site corpus. Seed 4 consumes the shared design-system pin invariant and adds built-site visual
acceptance. Seed 5 consumes the resulting dependency verdicts as journey evidence without rerunning
or redefining pack policy.

## Review Questions

1. Is every Core change confined to the five allowed consumer surfaces, with no policy implementation?
2. Does identity/coordinate divergence succeed with SPEC-056's warning while any evidence mismatch fails?
3. Does one durable owner record bind identity, coordinate, tag, full commit, hash, green checks, and fixtures?
4. Does Core consume owner dispatch proof without live tag resolution, full-owner reruns, or generic filter introspection?
5. Does the negative consumer proof mutate an isolated copy of a real non-owner page and require the
   installed pack's rule and production-relative path in the blocking finding?
6. Would deleting the installed documentation-semantics pack make acceptance fail even if a local
   script emitted the same diagnostic?
7. Is design-system work here limited to release/pin identity, leaving actual visual enforcement to
   Seed 4?
8. Has any generalized prose, style, grammar, tone, or LSP mechanism become an unnamed prerequisite?
9. If a generic missing enabler was discovered, is there a named external owner artifact, durable
   release evidence, and explicit consumer contract before Seed 2 depends on it?
10. Does a clean remote restoration from the recorded coordinate reproduce the evidence-bound bytes?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` — ready scope, REQ-006@2.1.0,
  REQ-012@1.1.0, OQ-3, OQ-8, DD-6, DD-11, Seed 2, and cross-repository sharp edges.
- `specs/SPEC-072-public-product-model.spec.md` — stable Seed 1 product-truth and actual-site
  consumer contract.
- `backstop.yml` — current exact-version pack declaration surface.
- `backstop.lock` — current git source-coordinate, tag, version, and content-hash lock surface.
- `pkg/pack/distribution/lockfile.go` — lock-entry representation.
- `pkg/pack/distribution/identity.go` and `pkg/pack/distribution/install.go` — manifest identity,
  independent coordinate resolution, installed path, and clean restoration mechanics.
- `pkg/pack/distribution/verify.go` — installed remote-pack content verification.
- `cmd/backstop/pack_gate.go` — lock-first installed-pack gate assembly.
- `specs/SPEC-015-pack-distribution.spec.md` — pack release/install lifecycle and validation gate.
- `specs/SPEC-056-remote-identity-version-validation.spec.md` — manifest identity, independent
  source coordinate, coordinate-based restoration, and loud non-fatal divergence contract.
- `specs/SPEC-032-pack-fixture-engine-execution.spec.md` — positive/negative findings-fixture
  execution semantics.
- `issues/ISSUE-183-local-pack-relock-refreshes-stale-install.issue.md` — observed stale local
  relock failure.
- `issues/ISSUE-184-fixture-path-filter-diagnostics.issue.md` — observed fixture path-scope failure.
