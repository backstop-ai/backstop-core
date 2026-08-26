---
title: "Documentation Semantics Integration"
number: SPEC-073
created: "2026-08-24"
status: implemented
schema_version: spec/v1
spec_version: 1.1.9

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
  - file: scripts/verify-documentation-semantics-integration.sh
    consumes:
      - source: .backstop/website-pack-releases.yml
        name: website_pack_release_imports
        kind: variable
      - source: .backstop/website-pack-releases.yml
        name: seed_2_predecessor_boundary
        kind: variable
      - source: backstop.yml
        name: website_pack_declarations
        kind: variable
      - source: backstop.lock
        name: website_pack_locks
        kind: variable
      - source: .backstop/packs/<documentation-semantics-manifest-identity>/pack.yml
        name: installed_documentation_semantics_manifest
        kind: variable
      - source: .backstop/packs/<design-system-manifest-identity>/pack.yml
        name: installed_design_system_manifest
        kind: variable
      - source: plans/PLAN-SPEC-072-public-product-model.plan.yml
        name: completed_seed_1_plan
        kind: variable
      - source: docs/index.md
        name: seed_1_home_page
        kind: variable
      - source: docs/evaluate.md
        name: seed_1_evaluation_page
        kind: variable
      - source: docs/model.md
        name: seed_1_model_page
        kind: variable
      - source: docs/adopt.md
        name: seed_1_adoption_page
        kind: variable
      - source: docs/use-cases.md
        name: seed_1_use_cases_page
        kind: variable
      - source: docs/packs.md
        name: seed_1_packs_page
        kind: variable
      - source: docs/extend.md
        name: seed_1_extension_page
        kind: variable
      - source: docs/reference.md
        name: seed_1_reference_page
        kind: variable
      - source: docs/status.md
        name: seed_1_status_page
        kind: variable
      - source: docs/contributing.md
        name: seed_1_contributing_page
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

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-006@2.1.0
    text: >
      Preserve the three-owner boundary exactly. Backstop-specific truth and instances remain in
      Core; visual policy remains in the design-system owner; reusable documentation semantics
      remain in the owner named by imported release evidence. Seed 2 implementation delivery may change only
      `.backstop/website-pack-releases.yml`, `backstop.yml`, `backstop.lock`,
      `scripts/verify-documentation-semantics-integration.sh`, and `.github/workflows/ci.yml`.
      It must not add or modify `pkg/**`, `cmd/**`, `packs/**`, `docs/**`, either owner repository,
      or owner rule, validator, engine, and fixture files. The verifier may validate bindings,
      orchestrate clean install/check/test/gate commands, mutate an isolated actual-site copy, and
      inspect finding attribution; it may not invoke a semantic engine directly or ship policy.
      The linked SPEC-073 and PLAN-SPEC-073 files are governed ledger surfaces, not implementation
      delivery: their validated version/state transitions are excluded from the five-file comparison.
  - id: REQ-002
    supports:
      - website-expansion:REQ-006@2.1.0
      - website-expansion:REQ-012@1.1.0
    text: >
      `.backstop/website-pack-releases.yml` must be one closed `website-pack-releases/v1` YAML
      document with exactly the top-level keys `schema_version`, `seed_2_baseline`, and `releases`.
      `schema_version` is the literal `website-pack-releases/v1`. `seed_2_baseline` has exactly
      `predecessor_plan: PLAN-SPEC-072`, `predecessor_spec: SPEC-072`,
      `predecessor_spec_version: 1.0.7`, and a lowercase 40-hex `terminal_transition_commit`.
      `releases` is a two-item sequence containing exactly one `documentation-semantics` and one
      `design-system` role. Each item has exactly `role`, `owner_artifact`, `release_evidence`,
      `manifest_identity`, `source_coordinate`, `version`, `git_ref`, `release_commit`,
      `content_hash`, `common_checks`, and `documentation_semantics`. Identity and coordinate are
      nonempty case-sensitive `<owner>/<repository>` strings; `version` is stable SemVer without a
      prerelease/build suffix; `git_ref` is exactly `v<version>`; `release_commit` is lowercase
      40-hex; and `content_hash` is lowercase 64-hex. `owner_artifact` and `release_evidence` are
      closed maps with exactly `repository`, `commit`, `path`, and `sha256`: repository is exactly
      `https://github.com/<source_coordinate>.git`, commit is lowercase 40-hex, path is nonempty
      root-relative POSIX without `.` or `..` segments, and sha256 is lowercase 64-hex. Unknown,
      duplicate, null, scalar-substituted, or additional keys and records are prohibited.
  - id: REQ-003
    supports:
      - website-expansion:REQ-006@2.1.0
      - website-expansion:REQ-012@1.1.0
    text: >
      The provider/consumer names are exact: `.backstop/website-pack-releases.yml` provides
      `website_pack_release_imports`; `backstop.yml` provides `website_pack_declarations`;
      `backstop.lock` provides `website_pack_locks`; and the verifier consumes those exact names.
      For both release roles, the declaration key and lock key/name equal `manifest_identity`;
      declaration and lock versions equal `version`; lock `source_type` is `git`; lock
      `source_coordinate`, `git_ref`, and nonempty `content_hash` equal the import; and installed
      paths derive from manifest identity. Clean restoration resolves the independent coordinate.
      Equal identity/coordinate passes silently. Divergence also passes, and initial add emits
      SPEC-056's loud non-fatal warning naming coordinate, identity, and install path. Any imported,
      declaration, lock, installed-manifest, installed-version, installed-hash, or restored-byte
      mismatch; mutable/local source; missing surface; stale tree; or alias substituted for one of
      the exact contract names is prohibited.
  - id: REQ-004
    supports:
      - website-expansion:REQ-006@2.1.0
      - website-expansion:REQ-012@1.1.0
    text: >
      Each role's `common_checks` is an exactly two-item sequence keyed once by `pack-check` and once
      by `pack-test`. Each closed item has exactly `check`, `command`, `exit_code`, `result`,
      `subject_commit`, `subject_content_hash`, and `log_ref`; commands are respectively
      `./bin/backstop pack check .` and `./bin/backstop pack test .`, exit code is integer zero,
      result is `pass`, and subject commit/hash equal the release record. `log_ref` has the same
      immutable-reference shape as REQ-002. Fetched `release_evidence` bytes must parse as one closed
      `website-pack-release-evidence/v1` YAML document with exactly `schema_version`, `subject`,
      `owner_artifact`, `common_checks`, and `documentation_semantics`; `subject` has exactly
      `role`, `manifest_identity`, `source_coordinate`, `version`, `git_ref`, `release_commit`, and
      `content_hash`, and the other three values must structurally equal the corresponding imported
      values. The verifier fetches owner artifact, release evidence, and log bytes by their full commit
      locators without resolving a tag, checks every declared SHA-256, parses the evidence subject,
      and requires its identity, coordinate, version, tag, release commit, content hash, and green
      checks to equal the import. Every reference repository
      must equal the source-coordinate repository; a Core-hosted reference, mutable URL/ref,
      unchecked transcript, copied assertion without matching fetched bytes, digest mismatch, or
      fields mixed across releases is self-authored/unbound evidence and fails. For
      `documentation-semantics`, `documentation_semantics` is a closed non-null map containing a
      nonempty `exported_claims` sequence and one `actual_corpus_probe`; each claim item has exactly
      `claim_id`, distinct nonempty `positive_fixture_path` and `negative_fixture_path`, nonempty
      `production_relative_path`, and immutable `dispatch_evidence_ref`, whose fetched evidence must
      prove both fixtures entered production-equivalent dispatch with positive pass, negative block,
      and the stated path. The probe has exactly nonempty `rule_id`, `marker`, and
      `expected_message_fragment`, all imported from owner evidence. For `design-system`,
      `documentation_semantics` is exactly null; Seed 2 requires its common evidence but not semantic
      fixtures or path-fidelity proof. Unknown documentation-specific keys, absent exported claims,
      filtered/unproven fixtures, or equivalent semantic duties silently imposed on design-system fail.
  - id: REQ-005
    supports:
      - website-expansion:REQ-006@2.1.0
    text: >
      Installed documentation semantics must run through the exact explicit corpus invocation
      `./bin/backstop --json gate` followed by one `--file` argument for each of these fourteen
      paths and no others: `docs/index.md`, `docs/evaluate.md`, `docs/model.md`, `docs/adopt.md`,
      `docs/use-cases.md`, `docs/packs.md`, `docs/extend.md`, `docs/reference.md`, `docs/status.md`,
      `docs/contributing.md`, `docs/_data/content-topology.yml`, `docs/_data/product-model.yml`,
      `docs/_data/evidence-inventory.yml`, and `docs/_data/content-inventory.yml`. Argument order is
      that exact order. These isolated consumer-corpus invocations explicitly set
      `BACKSTOP_PACK_SANDBOX=external`: Core's current native validator profile intentionally grants
      owner-pack reads only, while this trusted released validator must read the copied consumer
      corpus. The ordinary Core blocking gate remains native. The verifier requires the JSON scope
      to equal the fourteen-path set and the
      installed documentation pack's engine step to execute, never skip. It then creates fourteen
      isolated copies; in copy N it inserts only the imported `actual_corpus_probe.marker` into path N
      and runs `./bin/backstop --json gate --all` in external mode, requiring a blocking result from
      the imported probe rule naming path N and the expected message fragment. The mutation run is
      deliberately all-scope because Core's current file-scope filter drops corpus-level sandbox
      findings, which have no structured single-file field; the isolated copy contains the exact
      accepted documentation corpus, so this does not broaden production ownership. Thus every page
      and registry proves entry
      into installed-pack dispatch independently of Git dirt or a clean semantic verdict. The clean
      corpus has no blocking semantic finding. A separate isolated all-scope mutation adds a second
      substantive canonical definition using an existing anchor in the same real page and must block with the responsible installed
      semantic rule and production-relative path. A final isolated copy deletes the installed tree
      while retaining evidence, declaration, and lock and must fail missing dependency before local
      output counts. Diff-default/all-scope substitution, omitted/additional/reordered corpus paths,
      owner-fixture-only execution, probe results from a local script, or unattributed output fails.
  - id: REQ-006
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
  - id: REQ-007
    supports:
      - website-expansion:REQ-006@2.1.0
    text: >
      Seed 2 change scope is computed from the completed SPEC-072 predecessor, not a caller-selected
      branch, merge base, tag, environment variable, or dirty-diff default. Walking current `HEAD`'s
      first-parent history, the verifier must find exactly one earliest commit whose tree changes
      `plans/PLAN-SPEC-072-public-product-model.plan.yml` from absent or non-`completed` in its first
      parent to `status: completed`, still names `spec_id: SPEC-072` and `spec_version: 1.0.7`, and
      passes that tree's `./scripts/verify-public-product-model.sh`. That commit is the deterministic
      terminal transition and must equal `seed_2_baseline.terminal_transition_commit`; zero or
      multiple candidates fail. The normalized Seed 2 change set is the union of every path reported
      by `git -c diff.renames=copies diff --name-status --find-renames=50% --find-copies=50%
      <terminal-transition-commit> --` (including staged, unstaged, deletions, and both paths of each
      `R`/`C` record) and every root-relative untracked path from
      `git ls-files --others --exclude-standard`, less the exact linked artifact paths
      `specs/SPEC-073-documentation-semantics-integration.spec.md` and
      `plans/PLAN-SPEC-073-documentation-semantics-integration.plan.yml` only when both artifacts
      validate and retain their exact IDs/linkage. At final acceptance the remaining set must equal
      exactly REQ-001's five paths. Consequently all SPEC-072 predecessor page, registry, verifier, plan, and
      spec changes are before the boundary, while any later change outside the five paths fails; an
      alternate base, second completed transition, ignored untracked delivery file, or convenient
      diff-base selection is prohibited. PLAN-SPEC-073 execution remains downstream of completed
      SPEC-072 and must never become a SPEC-072 execution prerequisite.

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
    text: The closed v1 document with its exact baseline map and exactly one record for each role passes.
    tests: [verify_release_import_schema_accepts_exact_two_role_document]
  - id: CLM-006
    requirement: REQ-002
    text: A missing, additional, duplicate, null, scalar-substituted, unknown-role, or wrong-cardinality key or record fails with its YAML path.
    tests: [verify_release_import_schema_rejects_shape_and_cardinality_violation]
  - id: CLM-007
    requirement: REQ-002
    text: An invalid stable version, tag, identity, coordinate, commit, hash, immutable repository, or root-relative path fails with its role and field.
    tests: [verify_release_import_schema_rejects_invalid_scalar_or_reference]
  - id: CLM-008
    requirement: REQ-003
    text: Both evidence-bound pack matrices pass silently when manifest identity and source coordinate are equal.
    tests: [verify_pin_matrix_accepts_equal_identity_and_coordinate]
  - id: CLM-009
    requirement: REQ-003
    text: A differing evidence-bound identity and coordinate passes, installs under identity, resolves the coordinate, and emits SPEC-056's loud non-fatal add warning.
    tests: [verify_pin_matrix_accepts_divergence_with_spec056_warning]
  - id: CLM-010
    requirement: REQ-003
    text: Missing release import, declaration, lock, or installed surface fails with the role and exact contract name.
    tests: [verify_pin_matrix_rejects_missing_surface]
  - id: CLM-011
    requirement: REQ-003
    text: A range, prerelease, branch, path, mutable selector, local source, or nonexact tag fails.
    tests: [verify_pin_matrix_rejects_mutable_or_local_source]
  - id: CLM-012
    requirement: REQ-003
    text: Any import, declaration, lock, or installed identity, coordinate, version, ref, or hash mismatch, including a contract-name alias, fails naming both surfaces.
    tests: [verify_pin_matrix_rejects_binding_mismatch_or_contract_alias]
  - id: CLM-013
    requirement: REQ-003
    text: A missing install, manifest mismatch, empty hash, stale tree, restored-byte mismatch, or installed-byte drift fails before policy findings execute.
    tests: [verify_pin_matrix_rejects_missing_or_drifted_install]
  - id: CLM-014
    requirement: REQ-004
    text: Both roles pass when exactly pack-check and pack-test bind zero/pass results and immutable fetched logs to one release commit and content hash.
    tests: [verify_owner_evidence_accepts_common_checks_for_both_roles]
  - id: CLM-015
    requirement: REQ-004
    text: A Core-hosted or mutable reference, missing/duplicate check, nonzero/non-pass result, unfetched assertion, digest mismatch, subject mismatch, or mixed-release field fails.
    tests: [verify_owner_evidence_rejects_self_authored_incomplete_or_mixed_proof]
  - id: CLM-016
    requirement: REQ-004
    text: Documentation-semantics passes only when every exported claim has distinct dispatched positive and negative fixtures with production-relative path fidelity and the owner-declared corpus probe is complete.
    tests: [verify_owner_evidence_accepts_documentation_specific_dispatch_proof]
  - id: CLM-017
    requirement: REQ-004
    text: Missing, duplicate, same-polarity, filtered, path-mismatched, unfetched, or incomplete documentation fixture/probe evidence fails with its claim or probe field.
    tests: [verify_owner_evidence_rejects_unproven_documentation_dispatch]
  - id: CLM-018
    requirement: REQ-004
    text: Design-system passes with the same complete common evidence and exactly null documentation_semantics, without documentation fixture or path-fidelity duties.
    tests: [verify_owner_evidence_accepts_design_system_common_only_matrix]
  - id: CLM-019
    requirement: REQ-004
    text: A non-null design-system documentation_semantics value or a null documentation-semantics value fails instead of silently merging the role-specific evidence matrices.
    tests: [verify_owner_evidence_rejects_role_specific_matrix_contradiction]
  - id: CLM-020
    requirement: REQ-004
    kind: absence
    text: The verifier contains no live tag resolution, full owner checkout/rerun, or generic fixture-filter introspection.
    tests: [verify_owner_evidence_excludes_live_owner_and_generic_fixture_introspection]
  - id: CLM-021
    requirement: REQ-005
    text: The installed released documentation-semantics pack executes without blocking semantic findings over the exact ordered fourteen-file Seed 1 corpus, whose JSON scope equals that set.
    tests: [verify_installed_semantics_gate_accepts_exact_clean_seed1_corpus]
  - id: CLM-022
    requirement: REQ-005
    text: Fourteen isolated owner-declared probe mutations each block through the installed probe rule and name the independently probed production-relative path.
    tests: [verify_installed_semantics_gate_dispatches_every_seed1_path]
  - id: CLM-023
    requirement: REQ-005
    text: A dirty-diff/default/all-scope invocation or any omitted, additional, duplicate, or reordered corpus path fails before a clean verdict can count.
    tests: [verify_installed_semantics_gate_rejects_vacuous_or_inexact_corpus_scope]
  - id: CLM-024
    requirement: REQ-005
    text: An isolated actual-site mutation adding a second substantive canonical definition with an existing same-document anchor blocks with the installed pack rule and production-relative page path.
    tests: [verify_installed_semantics_gate_blocks_duplicate_substantive_owner]
  - id: CLM-025
    requirement: REQ-005
    text: Deleting the installed tree while retaining evidence, declaration, and lock fails as a missing dependency.
    tests: [verify_installed_semantics_gate_rejects_deleted_pack]
  - id: CLM-026
    requirement: REQ-005
    text: Unattributed local output, a local-script probe result, or an owner-fixture-only run cannot satisfy installed-pack actual-site proof.
    tests: [verify_installed_semantics_gate_rejects_unattributed_or_fixture_only_proof]
  - id: CLM-027
    requirement: REQ-006
    kind: absence
    text: Seed 2 adds no generalized prose-quality, writing-style, grammar, tone, or prose-LSP prerequisite.
    tests: [verify_scope_boundary_excludes_generalized_prose_system]
  - id: CLM-028
    requirement: REQ-006
    text: A newly discovered generic enabler is accepted only with a named owner artifact, durable release evidence, and explicit consumer contract.
    tests: [verify_scope_boundary_accepts_separately_governed_enabler]
  - id: CLM-029
    requirement: REQ-006
    text: Expanding the five-file allowlist for an unnamed enabler or making banked prose work a prerequisite fails.
    tests: [verify_scope_boundary_rejects_absorbed_or_prose_prerequisite]
  - id: CLM-030
    requirement: REQ-007
    text: The unique first-parent transition that completes PLAN-SPEC-072 at spec version 1.0.7 and passes the predecessor verifier is accepted as the recorded Seed 2 baseline.
    tests: [verify_seed2_baseline_accepts_unique_seed1_terminal_transition]
  - id: CLM-031
    requirement: REQ-007
    text: Zero or multiple completed transitions, a nonpassing transition tree, a mismatched recorded SHA, or any caller-selected branch, merge base, tag, or environment override fails.
    tests: [verify_seed2_baseline_rejects_ambiguous_or_selected_base]
  - id: CLM-032
    requirement: REQ-007
    text: The normalized terminal-transition-to-worktree change set excludes predecessor delivery and includes committed, staged, unstaged, deleted, renamed, copied, and untracked Seed 2 paths.
    tests: [verify_seed2_change_set_accounts_for_predecessor_and_all_worktree_states]
  - id: CLM-033
    requirement: REQ-007
    text: Any post-baseline path outside the exact five-file set, or any missing allowed delivery path at final acceptance, fails with the normalized path.
    tests: [verify_seed2_change_set_rejects_nonexact_delivery_surface]
  - id: CLM-034
    requirement: REQ-007
    kind: absence
    text: SPEC-072 contains no execution dependency on PLAN-SPEC-073 or any of its tasks.
    tests: [verify_seed2_dependency_direction_excludes_plan073_from_seed1]
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

### Release-import schema

`.backstop/website-pack-releases.yml` is a closed `website-pack-releases/v1` document. Its exact
shape is:

| Path | Type and cardinality | Exact contract |
|---|---|---|
| `schema_version` | string, once | `website-pack-releases/v1` |
| `seed_2_baseline` | map, once | Exactly `predecessor_plan: PLAN-SPEC-072`, `predecessor_spec: SPEC-072`, `predecessor_spec_version: 1.0.7`, and lowercase 40-hex `terminal_transition_commit`. |
| `releases` | sequence, exactly two | Exactly one `documentation-semantics` role and one `design-system` role. |
| `releases[*]` | closed map | Exactly `role`, `owner_artifact`, `release_evidence`, `manifest_identity`, `source_coordinate`, `version`, `git_ref`, `release_commit`, `content_hash`, `common_checks`, and `documentation_semantics`. |
| `owner_artifact`, `release_evidence`, every `log_ref`, every `dispatch_evidence_ref` | closed map | Exactly `repository`, `commit`, `path`, `sha256`; repository is `https://github.com/<source_coordinate>.git`, commit is lowercase 40-hex, path is root-relative POSIX without `.`/`..`, and sha256 is lowercase 64-hex. |
| identity / coordinate | string | Nonempty case-sensitive `<owner>/<repository>`. |
| `version`, `git_ref` | strings | Stable SemVer without prerelease/build suffix; exact `v<version>` ref. |
| `release_commit`, `content_hash` | strings | Lowercase 40-hex and lowercase 64-hex respectively. |
| `common_checks` | sequence, exactly two | Once each for `pack-check` and `pack-test`; closed records are defined below. |
| documentation role's `documentation_semantics` | closed map | Nonempty `exported_claims` plus exactly one `actual_corpus_probe`. |
| design-system role's `documentation_semantics` | null | No semantic fixture or path-fidelity obligation in Seed 2. |

Unknown, duplicate, null where a map/list/scalar is required, scalar-substituted, and additional
keys or records fail. Each common-check record has exactly `check`, `command`, `exit_code`,
`result`, `subject_commit`, `subject_content_hash`, and `log_ref`. The two commands are exactly
`./bin/backstop pack check .` and `./bin/backstop pack test .`; both require integer zero, `pass`,
and the release record's commit and hash.

Fetched release-evidence bytes are themselves a closed `website-pack-release-evidence/v1` YAML
document with exactly `schema_version`, `subject`, `owner_artifact`, `common_checks`, and
`documentation_semantics`. `subject` contains exactly `role`, `manifest_identity`,
`source_coordinate`, `version`, `git_ref`, `release_commit`, and `content_hash`; the remaining three
values structurally equal their imported counterparts. This makes green results and their
commit/hash join parseable rather than relying on prose in an arbitrary transcript.

Each documentation exported-claim record has exactly `claim_id`, distinct nonempty
`positive_fixture_path` and `negative_fixture_path`, `production_relative_path`, and
`dispatch_evidence_ref`. Fetched evidence must prove both polarities used production-equivalent
dispatch, the positive passed, the negative blocked, and the stated path was preserved. The
`actual_corpus_probe` has exactly the owner-imported nonempty `rule_id`, `marker`, and
`expected_message_fragment` used only to establish per-path dispatch.

### Released/pinned dependency matrix

The following matrix applies independently to both known website packs:

| Surface | Accepted | Prohibited |
|---|---|---|
| Owner evidence | Owner-repository bytes fetched by full commit, matching declared SHA-256 and one release subject | Core-hosted/mutable locator, unfetched copied assertion, digest mismatch, incomplete or mixed-release record |
| Common checks | Exactly green `pack-check` and `pack-test` for the subject commit/hash | Missing/duplicate check, nonzero/non-pass result, wrong subject, or unchecked transcript |
| `backstop.yml` / `website_pack_declarations` | Exact stable version under the exact manifest identity | Missing entry, alias, prerelease, range, branch, path, or mutable selector |
| `backstop.lock` / `website_pack_locks` | Git source, imported coordinate/ref/hash, name/version equal declaration | Local/path source, alias, empty hash, or evidence divergence |
| Installed tree | `.backstop/packs/<manifest_identity>` manifest/version/hash and clean restored bytes match | Missing/stale tree, manifest mismatch, or byte drift |

Identity and coordinate equality passes silently. Divergence also passes: install/runtime state is
keyed by manifest identity, remote restoration uses the coordinate, and initial add emits the loud
non-fatal SPEC-056 warning. Either value differing from fetched owner evidence fails. The exact
release/declaration/lock contract names above are intentional; aliases are not implicit subcontracts.

### Evidence duties by role

| Evidence | Documentation semantics | Design system |
|---|---|---|
| Immutable owner artifact and release-evidence bytes | Required | Required |
| Green `pack-check` and `pack-test` logs bound to release commit/hash | Required | Required |
| Exported-claim positive/negative fixture dispatch and path fidelity | Required | Not required; `documentation_semantics` is null |
| Actual-corpus dispatch-probe contract | Required | Not required; Seed 4 owns actual-site visual proof |

This is not permission to trust self-authored Core assertions. The verifier fetches each immutable
reference from the source-coordinate repository, checks its digest, parses the release subject, and
joins every imported field to that subject. It does not resolve tags, clone/rerun the complete owner,
or generically infer fixture filters.

### Exact actual-site corpus

The installed documentation pack receives exactly these paths, in this order, through explicit
`--file` arguments: the ten pages `docs/index.md`, `docs/evaluate.md`, `docs/model.md`,
`docs/adopt.md`, `docs/use-cases.md`, `docs/packs.md`, `docs/extend.md`, `docs/reference.md`,
`docs/status.md`, and `docs/contributing.md`; then the four registries
`docs/_data/content-topology.yml`, `docs/_data/product-model.yml`,
`docs/_data/evidence-inventory.yml`, and `docs/_data/content-inventory.yml`.

The clean run uses `./bin/backstop --json gate` plus those fourteen explicit arguments. Its JSON
scope must equal the set and its installed-pack engine step must execute. Fourteen disposable runs
then insert only the owner-imported dispatch marker into one distinct path apiece. Each must block
through the owner-imported probe rule and name that path. This per-path matrix, not Git dirt or the
absence of findings, proves that every Seed-1 input entered installed-pack dispatch.

### Deterministic Seed-2 change boundary

The baseline is not chosen by a caller. Walking current `HEAD`'s first-parent history, the verifier
finds the unique earliest commit that transitions PLAN-SPEC-072 from absent/non-completed to
`completed`, still binds SPEC-072 v1.0.7, and passes that tree's exact Seed-1 verifier. That commit
must equal the recorded `terminal_transition_commit`. The Seed-2 set is the normalized union of
`git -c diff.renames=copies diff --name-status --find-renames=50% --find-copies=50% <commit> --`
(committed, index, worktree, deletions, and both paths of each `R`/`C` record) and
`git ls-files --others --exclude-standard`. Final acceptance requires exactly the five
REQ-001 delivery paths after subtracting only the validated linked SPEC-073 and PLAN-SPEC-073
ledger surfaces. Therefore predecessor docs/registry work is before the boundary and any
post-boundary scope addition remains visible; no branch, merge-base, tag, environment, or dirty-diff
selector may replace this computation.

## Implementation

The planner must preserve this order:

1. Locate the deterministic first-parent PLAN-SPEC-072 completed-transition commit, prove that tree
   binds SPEC-072 v1.0.7 and passes the Seed-1 verifier, and record that exact SHA in the closed v1
   import. Validate the linked SPEC-073/PLAN-SPEC-073 ledger transition before excluding those two
   artifact paths. Do not accept a caller-supplied branch, merge base, tag, or environment override.
2. The independently governed documentation owner completes its artifact chain and publishes
   immutable owner-repository evidence binding one release's identity, coordinate, version/tag,
   full commit, content hash, two green common checks, every exported claim's dispatched
   positive/negative production-path fixtures, and the actual-corpus dispatch probe. The design-system
   owner supplies the same common release/check evidence without documentation-specific evidence.
3. Import exactly the two closed role records into `.backstop/website-pack-releases.yml`. Fetch each
   owner artifact, release evidence, log, and dispatch proof by full commit; verify its declared
   SHA-256 and parsed subject joins. Reject Core-hosted, mutable, unfetched, incomplete,
   self-asserted, role-confused, or mixed-release evidence.
4. Add the documentation pack by imported coordinate and version through the normal remote command.
   Reconcile `website_pack_declarations` and `website_pack_locks` exactly to both imports, preserving
   the design-system pin and the independent identity/coordinate semantics and warning from SPEC-056.
5. Reconstruct a clean consumer with `pack install` from the locks and compare installed manifests,
   versions, hashes, and bytes with evidence. The verifier may rerun check/test over installed
   hash-matched bytes, but never resolves live tags, clones/reruns complete owner repositories,
   introspects generic fixture filters, or invokes semantic engines directly.
6. Run the exact ordered fourteen-file JSON gate and verify its scope and installed-pack engine step.
   In fourteen independent temporary copies, insert the imported dispatch marker into one path at a
   time and require the imported probe rule, expected message, and production-relative path. Then run
   the separate duplicate-definition and deleted-install negative proofs. No accepted page is mutated.
7. Compute the final delivery set from the recorded predecessor boundary through the complete Git
   index/worktree/untracked state and require exactly the five allowed paths. This pass occurs after
   all other delivery mutations so a later forbidden file cannot escape an earlier scope check.
8. Add the verifier to CI after clean remote pack installation and before any job reports website
   policy compliance. Seed 4 may compose design-system actual-site proof after this step; it cannot
   replace semantic proof. PLAN-SPEC-073 is downstream execution only and never becomes a SPEC-072
   prerequisite.

The verifier may parse the closed import, fetched owner evidence, declarations, lock entries,
installed manifests, hashes, Git history/change sets, JSON scope/step results, and attributed
findings. It may not decide semantic validity itself. If it contains vocabulary or logic that
detects a competing definition instead of asking the installed pack, it has crossed the owner
boundary.

## Verification

The single integration command runs all claims defined in frontmatter. It uses isolated directories
for immutable-evidence fetches, clean consumer reconstruction, fourteen per-path dispatch probes,
the injected semantic failure, and delete-installed attribution. Diagnostics name the YAML path for
schema failures; evidence role/reference/digest for provenance failures; disagreeing exact contract
surfaces for pin failures; predecessor candidate or normalized change path for scope failures; and
the installed pack rule plus production-relative path for dispatch and semantic failures. Core
consumes durable owner dispatch evidence; it does not implement ISSUE-184-style generic filter
introspection.

The positive path is deliberately conjunctive:

1. one exact closed v1 import and a unique deterministic Seed-1 terminal-transition baseline;
2. immutable owner-hosted bytes and common green evidence for both roles, plus documentation-only
   fixture/path/probe evidence;
3. exact named declarations and evidence-bound git locks;
4. clean remote restoration with matching installed manifests, hashes, and bytes;
5. exact ordered fourteen-file JSON scope with an executed installed-pack step;
6. fourteen attributed per-path dispatch-probe blocks, clean corpus, and the attributed substantive
   negative mutation;
7. delete-installed failure attributed to the missing dependency; and
8. a final normalized change set equal to exactly the five delivery paths.

No self-authored transcript, fixture-only run, local relock, dirty-diff/default-scope result,
unattributed output, convenient baseline, or clean gate without the per-path and negative proofs can
satisfy that conjunction. Claims and mandated test names are defined in frontmatter; completeness is
the exhaustive role/evidence, pin, corpus-path, mutation, missing-install, and change-set matrices,
not an inferred pass from one green command.

## Sharp Edges

- **Identity and coordinate are different contracts.** Manifest identity owns install/runtime state;
  coordinate owns remote resolution. Divergence is accepted with a warning, while either value
  differing from durable owner evidence is a refusal.
- **Local relock is specifically untrustworthy evidence today.** ISSUE-183 records a successful
  relock that continued to hash and execute stale installed bytes. Seed 2 therefore rejects local
  source type outright, even if the reported hash appears stable.
- **Green fixtures can be outside production scope.** ISSUE-184 shows how fixture names can miss
  include filters. The owner must publish durable dispatch/path-fidelity proof; Core must not close
  that generic gap by introspecting filters locally. Fourteen owner-declared actual-corpus probes
  establish that every real Seed-1 path enters installed dispatch.
- **A clean actual-site gate can be vacuous.** Explicit `--file` arguments prove gate scope but a
  pack filter can still skip individual inputs. JSON scope plus an executed engine step is necessary,
  while the independent per-path probe matrix is the evidence that closes the remaining skip seam.
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
  every field and green result to one release; merely copying a URL and green booleans into Core is
  not evidence. Full-commit fetch, content digest, owner repository, and parsed subject must all join.
- **The two roles do not have identical evidence matrices.** Both owe common release/check proof.
  Documentation semantics additionally owes exported-claim fixture/path proof and a corpus probe;
  imposing those duties on design-system would silently move Seed-4 acceptance into Seed 2.
- **Evidence and restoration prove different edges.** Durable owner evidence proves owner release
  governance; a clean coordinate-based install and content hash prove the consumer can reproduce it.
- **The external identity is imported, not guessed.** Renaming identity or coordinate requires a new
  owner release record and coordinated migration; Core cannot ratify an alias on its own.
- **A convenient diff base can erase forbidden delivery.** Seed 1 legitimately changes `docs/**`, so
  repository dirt relative to an arbitrary branch is neither the Seed-2 set nor a safe exclusion.
  The unique PLAN-SPEC-072 terminal transition fixes predecessor acceptance; later deletions,
  renames, copies, and untracked files remain visible except for the two validated linked artifact
  ledger surfaces.

## Integration Contract

SPEC-072 supplies the exact ten page sources and four Backstop-specific registries named by REQ-005.
Its PLAN-SPEC-072 completed transition is the deterministic predecessor boundary, but no
PLAN-SPEC-073 execution, task, pack byte, or partial Seed-2 state may become a SPEC-072 prerequisite.
The external documentation-semantics pack consumes the fourteen instances through its released
interface but may not become their truth owner. SPEC-074 may add generated Markdown to owners
declared by SPEC-072; this integration must evaluate that generated content through the same
installed pack once it enters the actual corpus. Seed 4 consumes the shared design-system pin
invariant and adds built-site visual acceptance. Seed 5 consumes the resulting dependency verdicts
as journey evidence without rerunning or redefining pack policy.

## Review Questions

1. Is every Core change confined to the five allowed consumer surfaces, with no policy implementation?
2. Does the closed release import reject every unknown key, wrong cardinality, role duplication,
   malformed immutable reference, unstable version, and mismatched tag?
3. Are `website_pack_release_imports`, `website_pack_declarations`, and `website_pack_locks` the exact
   provider/consumer names everywhere, without an undeclared alias?
4. Does identity/coordinate divergence succeed with SPEC-056's warning while any evidence mismatch fails?
5. Are owner artifacts, release evidence, common-check logs, and dispatch evidence fetched by full
   commit from the source-coordinate repository and digest-checked before imported assertions count?
6. Do both roles supply common green check evidence while only documentation-semantics supplies
   exported-claim fixture/path proof and a corpus probe, with design-system's field exactly null?
7. Does Core consume owner dispatch proof without live tag resolution, full-owner reruns, direct
   engine calls, or generic filter introspection?
8. Does the JSON gate carry the exact ordered fourteen explicit `--file` arguments, independent of
   Git dirt, and does its reported scope equal that set with the installed engine step executed?
9. Does each of the fourteen inputs independently block through the owner-declared probe rule when it
   alone receives the marker, naming that production-relative path?
10. Does the substantive negative proof duplicate an existing anchor within an isolated copy of a real page and require the
   installed pack's rule and production-relative path in the blocking finding?
11. Would deleting the installed documentation-semantics pack make acceptance fail even if a local
   script emitted the same diagnostic?
12. Is design-system work here limited to common release/pin evidence, leaving actual visual enforcement to
   Seed 4?
13. Does the unique first-parent PLAN-SPEC-072 completed transition fix the baseline, and does the
   normalized set include committed, staged, unstaged, deleted, renamed/copied, and untracked paths?
14. Would an alternate base, second completed transition, missing delivery path, or forbidden later
   path fail rather than conveniently changing the Seed-2 set?
15. Has any PLAN-SPEC-073 task or partial execution become a SPEC-072 prerequisite?
16. Has any generalized prose, style, grammar, tone, or LSP mechanism become an unnamed prerequisite?
17. If a generic missing enabler was discovered, is there a named external owner artifact, durable
   release evidence, and explicit consumer contract before Seed 2 depends on it?
18. Does a clean remote restoration from the recorded coordinate reproduce the evidence-bound bytes?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` — ready scope, REQ-006@2.1.0,
  REQ-012@1.1.0, OQ-3, OQ-8, DD-6, DD-11, Seed 2, and cross-repository sharp edges.
- `specs/SPEC-072-public-product-model.spec.md` — stable Seed 1 product-truth and actual-site
  consumer contract.
- `plans/PLAN-SPEC-072-public-product-model.plan.yml` — predecessor completion transition used to
  derive the Seed-2 boundary without depending on predecessor task IDs.
- `backstop.yml` — current exact-version pack declaration surface.
- `backstop.lock` — current git source-coordinate, tag, version, and content-hash lock surface.
- `pkg/pack/distribution/lockfile.go` — lock-entry representation.
- `pkg/pack/distribution/identity.go` and `pkg/pack/distribution/install.go` — manifest identity,
  independent coordinate resolution, installed path, and clean restoration mechanics.
- `pkg/pack/distribution/verify.go` — installed remote-pack content verification.
- `cmd/backstop/pack_gate.go` — lock-first installed-pack gate assembly.
- `pkg/gate/scope.go` and `pkg/gate/result.go` — explicit gate scope and structured result surfaces
  consumed by the exact-corpus proof.
- `specs/SPEC-015-pack-distribution.spec.md` — pack release/install lifecycle and validation gate.
- `specs/SPEC-056-remote-identity-version-validation.spec.md` — manifest identity, independent
  source coordinate, coordinate-based restoration, and loud non-fatal divergence contract.
- `specs/SPEC-032-pack-fixture-engine-execution.spec.md` — positive/negative findings-fixture
  execution semantics.
- `issues/ISSUE-183-local-pack-relock-refreshes-stale-install.issue.md` — observed stale local
  relock failure.
- `issues/ISSUE-184-fixture-path-filter-diagnostics.issue.md` — observed fixture path-scope failure.
