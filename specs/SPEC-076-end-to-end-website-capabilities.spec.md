---
title: "End To End Website Capabilities"
number: SPEC-076
created: "2026-08-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.3

implementation:
  summary: >
    BUNDLE-032 Seed 5 only: define eleven strict website capability artifacts and
    their exact @UJ scenarios, then execute those journeys against the Seed 4 built
    site and the identity-matched deployment at backstop.sh. CAP-002 and CAP-003 are
    already reserved historical IDs, so this cohort is exactly CAP-004 through CAP-014.
    A checked-in journey map binds every scenario to exact canonical route/anchor hops,
    Seed 1 evidence and boundary record IDs, typed Seed 3 generated-product provenance,
    and the upstream Seed 1-4 verification commands whose successful verdicts are
    prerequisites. Required predecessor contracts keep content links, structured
    boundary fields, adoption instructions, and their rendered markers with Seeds 1 and 4;
    Seed 5 refuses to run until those exact owner versions and contracts exist. The acceptance
    runner follows rendered links and inspects rendered semantics; it does not duplicate
    product truth, documentation semantics, generated-product-truth logic, visual policy,
    route/link/browser verification, or deployment identity policy. Mutation acceptance
    proves the suite is non-vacuous by independently removing every canonical route,
    every declared journey evidence edge, each required upstream verdict, every journey
    boundary explanation, and every generated provenance edge, with each mutation
    breaking at least one named journey. Adoption acceptance also executes the exact
    provenance-bound install, configuration, and gate commands rendered by the site in a
    disposable repository; browser traversal alone cannot satisfy CAP-009.
  subject: scripts/websitejourney

verification:
  level: integration
  coverage_threshold: 80
  test_command: ./scripts/verify-website-capabilities.sh

contracts:
  - file: docs/_data/website-capability-map.yml
    provides:
      - name: website_capability_journey_map
        kind: variable
        signature: "predecessor_contracts[3] + capabilities[11] + scenarios[22] + dependencies[4] + obligations[evidence|boundary|generated]"
    consumes:
      - source: docs/_data/content-topology.yml
        name: seed1_journey_links_and_adoption_instructions
        kind: variable
      - source: docs/_data/product-model.yml
        name: seed1_structured_boundary_fields
        kind: variable
      - source: docs/_data/evidence-inventory.yml
        name: seed1_claim_evidence
        kind: variable
      - source: docs/_data/derived-product-truth.yml
        name: seed3_generated_source_descriptors_and_url_templates
        kind: variable
      - source: docs/_data/site-presentation.yml
        name: seed4_rendered_journey_boundary_generated_and_adoption_contract
        kind: variable
  - file: scripts/websitejourney/main.go
    provides:
      - name: VerifyBuiltJourneys
        kind: function
        signature: "VerifyBuiltJourneys(root, builtRoot string, journeyMap JourneyMap) []Finding"
      - name: VerifyDeployedJourneys
        kind: function
        signature: "VerifyDeployedJourneys(origin string, commit string, runID int64, journeyMap JourneyMap) []Finding"
      - name: VerifyCapabilityArtifacts
        kind: function
        signature: "VerifyCapabilityArtifacts(root string, journeyMap JourneyMap) []Finding"
      - name: ExecuteAdoptionProof
        kind: function
        signature: "ExecuteAdoptionProof(root string, builtRoot string, journeyMap JourneyMap) []Finding"
  - file: scripts/verify-website-capabilities.sh
    provides:
      - name: verify_website_capabilities
        kind: function
        signature: "verify_website_capabilities [--capability <CAP-ID>] [--built-root <path> | --deployed-origin https://backstop.sh --commit <sha> --run-id <id>]"
    consumes:
      - source: scripts/verify-public-product-model.sh
        name: seed1_product_model_verdict
        kind: function
      - source: scripts/verify-documentation-semantics-integration.sh
        name: seed2_documentation_semantics_verdict
        kind: function
      - source: scripts/verify-product-truth.sh
        name: seed3_product_truth_verdict
        kind: function
      - source: scripts/verify-public-site.sh
        name: seed4_public_site_verdict
        kind: function
  - file: tests/website-capabilities.spec.ts
    provides:
      - name: website_capability_browser_acceptance
        kind: function
        signature: "Playwright traversal for every CAP and @UJ tag against built or deployed output"
  - file: capabilities/CAP-004-understand-backstop/capability.yml
    provides:
      - name: CAP-004
        kind: variable
        signature: "strict capability: Understand Backstop"
  - file: capabilities/CAP-004-understand-backstop/user-journeys.feature
    provides:
      - name: CAP-004_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-005-evaluate-fit/capability.yml
    provides:
      - name: CAP-005
        kind: variable
        signature: "strict capability: Evaluate Fit"
  - file: capabilities/CAP-005-evaluate-fit/user-journeys.feature
    provides:
      - name: CAP-005_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-006-evaluate-guarantees/capability.yml
    provides:
      - name: CAP-006
        kind: variable
        signature: "strict capability: Evaluate Guarantees"
  - file: capabilities/CAP-006-evaluate-guarantees/user-journeys.feature
    provides:
      - name: CAP-006_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-007-evaluate-compatibility/capability.yml
    provides:
      - name: CAP-007
        kind: variable
        signature: "strict capability: Evaluate Compatibility"
  - file: capabilities/CAP-007-evaluate-compatibility/user-journeys.feature
    provides:
      - name: CAP-007_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-008-understand-system/capability.yml
    provides:
      - name: CAP-008
        kind: variable
        signature: "strict capability: Understand the System"
  - file: capabilities/CAP-008-understand-system/user-journeys.feature
    provides:
      - name: CAP-008_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-009-adopt-backstop/capability.yml
    provides:
      - name: CAP-009
        kind: variable
        signature: "strict capability: Adopt Backstop"
  - file: capabilities/CAP-009-adopt-backstop/user-journeys.feature
    provides:
      - name: CAP-009_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-010-apply-backstop/capability.yml
    provides:
      - name: CAP-010
        kind: variable
        signature: "strict capability: Apply Backstop"
  - file: capabilities/CAP-010-apply-backstop/user-journeys.feature
    provides:
      - name: CAP-010_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-011-browse-pack-ecosystem/capability.yml
    provides:
      - name: CAP-011
        kind: variable
        signature: "strict capability: Browse the Pack Ecosystem"
  - file: capabilities/CAP-011-browse-pack-ecosystem/user-journeys.feature
    provides:
      - name: CAP-011_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-012-extend-backstop/capability.yml
    provides:
      - name: CAP-012
        kind: variable
        signature: "strict capability: Extend Backstop"
  - file: capabilities/CAP-012-extend-backstop/user-journeys.feature
    provides:
      - name: CAP-012_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-013-inspect-evidence/capability.yml
    provides:
      - name: CAP-013
        kind: variable
        signature: "strict capability: Inspect the Evidence"
  - file: capabilities/CAP-013-inspect-evidence/user-journeys.feature
    provides:
      - name: CAP-013_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: capabilities/CAP-014-continue-beyond-backstop/capability.yml
    provides:
      - name: CAP-014
        kind: variable
        signature: "strict capability: Continue Beyond Backstop"
  - file: capabilities/CAP-014-continue-beyond-backstop/user-journeys.feature
    provides:
      - name: CAP-014_user_journeys
        kind: variable
        signature: "@UJ-001 + @UJ-002"
  - file: .github/workflows/ci.yml
    provides:
      - name: website_capability_predeploy_gate
        kind: variable
        signature: "Blocking built-site CAP/@UJ acceptance"
  - file: .github/workflows/pages.yml
    provides:
      - name: website_capability_postdeploy_gate
        kind: variable
        signature: "Identity-matched https://backstop.sh CAP/@UJ acceptance after deployment"

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      Author exactly eleven new strict capability directories with the exact IDs, slugs,
      and titles in the Capability matrix below: CAP-004 through CAP-014. CAP-002 and
      CAP-003 are reserved and must not be reused. Every directory must contain schema-valid
      `capability.yml` and `user-journeys.feature`; set `status: draft` until all blocking
      built and deployed gates pass, then `verified`; set `strictness: strict`;
      `infrastructure_specs` must be exactly SPEC-072, SPEC-073, SPEC-074, SPEC-075 in that
      order; `integration_spec` must be SPEC-076; `scenarios.feature_file` must be
      `user-journeys.feature`; `scenarios.id_pattern` must be `^@UJ-[0-9]{3}$`;
      `app_origins` must be exactly `[https://backstop.sh]`, `auth_strategy` must be `none`,
      and setup/teardown commands must be empty. Each artifact must declare exactly two
      blocking acceptance quality gates: built mode invokes
      `./scripts/verify-website-capabilities.sh --capability <CAP-ID>`; deployed mode invokes
      `./scripts/verify-website-capabilities.sh --capability <CAP-ID> --deployed-origin https://backstop.sh --commit "$BACKSTOP_DEPLOY_COMMIT" --run-id "$BACKSTOP_DEPLOY_RUN_ID"`.
      No twelfth website capability, alias ID, reused reservation, relaxed gate,
      nonblocking/single-environment gate, or screenshot-only capability may satisfy this cohort.
  - id: REQ-002
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      The eleven feature files must contain exactly the twenty-two scenarios in the Journey
      matrix below: two per capability, with local IDs @UJ-001 and @UJ-002. A local @UJ ID is
      unique within its capability and the globally unique key is `<CAP-ID>/<UJ-ID>`.
      Each scenario must describe a visitor-observable outcome and traverse the exact ordered
      route/anchor and `JLINK-NNN` sequence in the Journey matrix through an owner-declared
      rendered content or next-action link. SPEC-076 depends on predecessor contracts
      SPEC-072 v1.0.5 and SPEC-075 v1.0.4: SPEC-072 exposes the exact `journey_links[]`
      records in `docs/_data/content-topology.yml` and requires each Seed 1 owner page to carry
      its link at the source anchor; SPEC-075 renders and verifies each as exactly one
      `a[data-journey-link-id=<JLINK-NNN>]` under `main#main` or
      `nav[data-next-action]`, with exact destination route/anchor. The journey map references
      those upstream IDs but does not author page copy or navigation. Acceptance must fail until
      both exact spec versions and all declared source/rendered bindings exist. Direct-loading
      an intermediate page, using global header/footer navigation, or inventing a Seed 5 link is
      prohibited. Every scenario must have one matching executable browser test whose title
      contains the global key. Missing, additional, duplicate, reordered, unexecuted,
      wrong-anchor, global-navigation-substituted, or direct-load-substituted scenarios fail.
  - id: REQ-003
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      `docs/_data/website-capability-map.yml` must bind each global scenario key to its exact
      ordered canonical route/anchor hops, semantic obligations, and owner record edges.
      Obligation `kind` is the closed set `evidence`, `boundary`, or `generated`. Every
      `evidence` obligation must name one existing `claim_id` from
      `docs/_data/evidence-inventory.yml`, its owner route and anchor, and its expected
      `claim_type`; every `boundary` obligation must name one existing `boundary_id` from
      `docs/_data/product-model.yml`, its owner route and anchor, its exact state, and that
      boundary's claim ID. SPEC-072 v1.0.5 structures every boundary's explanation and,
      where required, continuation and guarantee-denial fields without changing the five-state
      vocabulary. SPEC-075 v1.0.4 renders exact stable markers
      `[data-boundary-id][data-boundary-state]`, `[data-boundary-explanation]`,
      `a[data-boundary-continuation]`, and `[data-boundary-guarantee-denial]`; Seed 5 consumes
      those markers and never parses prose to infer a field. Every `generated` obligation must
      name an exact SPEC-074 job ID, owner route/anchor, source output, exact begin/end marker
      forms for both the product-truth and source-descriptor regions, the complete ordered typed
      descriptor set and exact URL templates, and the canonical source-level
      `sha256:<64-lowercase-hex>` provenance-envelope digest. It must separately name the
      SPEC-075-accepted site identity and complete realized immutable rendered-anchor set.
      SPEC-074 v1.0.4 owner-defines the typed source descriptors, exact URL
      templates containing unresolved `<SITE-COMMIT>` where applicable, and canonical
      source-level envelope digest. SPEC-075 v1.0.4 resolves `<SITE-COMMIT>` from the exact
      build/deployment identity, renders each realized immutable URL as
      `a[data-generated-source-link]` inside the one
      `section[data-generated-region][data-product-truth-job]`, reconstructs the canonical
      descriptor/record envelope, and owns rendered-digest acceptance. Seed 5 consumes that
      owner acceptance result and must not resolve or reconstruct it. CAP-011/@UJ-001 binds exactly
      `installed-pack-catalog`; CAP-013/@UJ-002 binds exactly all four jobs
      `cli-command-catalog`, `artifact-schema-catalog`, `installed-pack-catalog`, and
      `release-history`. CAP-014/@UJ-001 has one additional conjunctive identity contract:
      its JLINK-024 hop and BOUNDARY-005 adjacent-guidance continuation obligation must resolve
      to the SAME single rendered anchor inside the BOUNDARY-005 callout. That one anchor must
      carry both `data-journey-link-id="JLINK-024"` and `data-boundary-continuation`, retain the
      exact root-relative `/contributing/#external-ownership` href, and occur after the one
      `[data-boundary-explanation]` and before the one `[data-boundary-guarantee-denial]`.
      The explanation, link label, and guarantee-denial visible bytes must be exactly the bytes
      accepted by SPEC-075's owner rendering contract. A second anchor may not satisfy either
      cardinality. Seed 5 consumes only this accepted SPEC-075 v1.0.4 rendered DOM and the
      owner verdict keyed by JLINK-024 and BOUNDARY-005; it must not parse or canonicalize CLAIM-005,
      the source JLINK marker, boundary source metadata, or other source bytes to manufacture
      either identity. IDs and provenance are imported from accepted owner registries and
      must not be invented or copied into a second truth registry. A missing, duplicate, stale,
      wrong-owner, wrong-kind/type/state/job, malformed marker/source-level digest, incomplete
      descriptor/template/rendered-anchor set, wrong site identity, text-only, prose-inferred,
      Seed-5-resolved, or independently reconstructed edge fails. The required
      role mapping is exactly the Evidence, boundary, and generated matrix below; implementation
      selects concrete accepted Seed 1 IDs for semantic roles once and records them explicitly.
  - id: REQ-004
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      The map must declare exactly four prerequisites: `seed1-product-model` -> SPEC-072 ->
      `./scripts/verify-public-product-model.sh`; `seed2-documentation-semantics` -> SPEC-073 ->
      `./scripts/verify-documentation-semantics-integration.sh`; `seed3-product-truth` ->
      SPEC-074 -> `./scripts/verify-product-truth.sh`; and `seed4-public-site` -> SPEC-075 ->
      `./scripts/verify-public-site.sh`. Every capability depends on all four verdict IDs.
      The runner may execute these public verification entrypoints and consume their exit
      status and attributed result, but must not invoke an owner engine directly, restate a
      documentation-semantic or design-system rule, regenerate product truth, reinterpret
      product evidence/boundaries, or replace Seed 4's route, browser, build, and deployment
      checks. A missing, skipped, stale, synthetic, or nonzero prerequisite fails before a
      capability can be verified.
  - id: REQ-005
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      `./scripts/verify-website-capabilities.sh` must create disposable build, server,
      browser, mutation, and coverage state; run the four prerequisites; build the exact
      Seed 4 output; validate all capability artifacts and scenario/test coverage; and run
      all twenty-two journeys against that built tree. Deployed mode must use only
      `https://backstop.sh`, disable redirect following while resolving canonical ownership,
      require the SPEC-075 deployment marker's commit and run ID on every visited canonical
      page, and run the same journey assertions against the identity-matched deployment.
      Acceptance must follow root-relative rendered links, require HTTP 200 for canonical
      pages, require real case-sensitive anchors and visible semantic obligations, and prove
      terminal outcomes from rendered structure rather than keyword or screenshot matching.
      CAP-009 additionally requires an execution proof: SPEC-072 v1.0.5 owns the three
      exact provenance records and rendered command text in the Adoption command matrix;
      SPEC-075 v1.0.4 renders one `pre[data-adoption-instruction-id][data-command-sha256]`
      per record. After browser traversal, the runner must create a disposable Git repository,
      verify rendered text and SHA-256 against the upstream structured executable/argv/env
      record, then execute ADOPT-INSTALL, ADOPT-CONFIGURE, and ADOPT-ENFORCE in order without
      evaluating rendered text as a shell program. It must prove the installed binary exists,
      configuration exists, and the real final gate exits zero, and retain an in-memory receipt
      binding instruction IDs, command digests, exit codes, and built/deployed identity. A browser
      click-through, mocked command, preinstalled binary, copied command, or unproven gate result
      cannot satisfy CAP-009. Verification ships no application runtime or client JavaScript and
      cleans temporary state on success and failure.
  - id: REQ-006
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      Mutation acceptance must operate only on isolated copies of accepted built output and
      dependency-result fixtures. It must independently (a) remove each of the ten canonical
      routes; (b) remove every evidence element or source link named by a journey-map evidence
      edge; (c) remove or make nonzero each of the four prerequisite verdicts; and (d) remove
      every boundary element, state, explanation, continuation link, or guarantee denial named by a journey-map
      boundary edge; and (e) for every mapped generated obligation independently remove its
      region, job ID, owner anchor, begin marker, end marker, digest, or each source link one at
      a time, including every member of a variable multi-link set. For CAP-014/@UJ-001 it must
      additionally and independently remove the dual-identity anchor, remove its JLINK-024
      identity, remove its boundary-continuation identity, remove or alter its href, split the
      two identities across two anchors, duplicate the dual-identity anchor, move it outside
      BOUNDARY-005, place it before the explanation, place it after the guarantee denial, alter
      the explanation visible bytes, alter the link-label visible bytes, and alter the denial
      visible bytes. Seed 5 must consume the SPEC-075 owner verdict for those byte-preservation
      variants rather than comparing against source metadata itself.
      Each of these variants must fail with a diagnostic naming CAP-014/@UJ-001, JLINK-024, and
      BOUNDARY-005; an upstream generic failure alone is insufficient. Every
      mutation must make at least one specifically named global journey
      fail, and the diagnostic must name mutation class, mutated route/record/verdict, and
      broken journey. It is prohibited to accept only because a generic route/link checker,
      owner pack, screenshot diff, or hard-coded expected failure fired; the Seed 5 runner must
      show the broken journey edge. The accepted unmutated corpus must remain green.
  - id: REQ-007
    supports:
      - website-expansion:REQ-007@2.0.0
    text: >
      `scripts/websitejourney` must remain an integration consumer. It may parse capability
      artifacts, feature tags, the journey map, upstream exit verdicts, built HTML, deployment
      markers, HTTP status, links, anchors, structured adoption commands, and Seed 4 data
      attributes. It must not contain
      product claim prose, canonical concept definitions, boundary explanations, semantic
      classification, generated-product rendering, visual tokens/rules, layout thresholds, or
      a second route/link verifier. The verification wrapper must run measured tests with one
      numeric total and fail below 80.00 coverage or for absent/duplicate/nonnumeric coverage;
      CI must block on built-site acceptance, and Pages must run deployed acceptance after the
      authoritative SPEC-075 deployment proof. Any generic capability primitive missing from
      the existing schema/runtime must receive separate governance rather than being embedded here.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Exactly CAP-004 through CAP-014 exist with the exact IDs, slugs, titles, strict configuration, four infrastructure specs, SPEC-076 integration, and exact blocking built and deployed acceptance gates.
    tests: [TestWebsiteJourney_ExactCapabilityArtifactMatrixPasses]
  - id: CLM-002
    requirement: REQ-001
    text: A missing, extra, aliased, reserved-ID, non-strict, wrong-spec, or nonblocking capability artifact fails with capability and field.
    tests: [TestWebsiteJourney_RejectsInvalidCapabilityArtifactMatrix]
  - id: CLM-003
    requirement: REQ-002
    text: All twenty-two exact global scenario keys, JLINK IDs, and ordered source/destination route-anchor traversals resolve through SPEC-072-owned source links and SPEC-075-owned rendered link bindings and have matching executable tests.
    tests: [TestWebsiteJourney_ExactScenarioAndCoverageMatrixPasses]
  - id: CLM-004
    requirement: REQ-002
    text: A missing or wrong-version predecessor contract, source/rendered JLINK binding, scenario, anchor, or test, or an extra, duplicate, reordered, direct-loaded, global-navigation-substituted, or unexecuted scenario fails with its global key.
    tests: [TestWebsiteJourney_RejectsScenarioAndCoverageDrift]
  - id: CLM-005
    requirement: REQ-003
    text: Every declared evidence and boundary role resolves one-to-one to an explicit accepted Seed 1 ID, owner, type or state, stable field markers, rendered element, and durable link without prose inference.
    tests: [TestWebsiteJourney_EvidenceAndBoundaryBindingsPass]
  - id: CLM-006
    requirement: REQ-003
    text: A missing, duplicate, stale, inferred, text-only, wrong-owner, wrong-type, wrong-state, missing-field-marker, or unlinked evidence/boundary binding fails with global key and record ID.
    tests: [TestWebsiteJourney_RejectsInvalidSemanticBindingMatrix]
  - id: CLM-023
    requirement: REQ-003
    text: CAP-011/@UJ-001 resolves the exact installed-pack-catalog job and CAP-013/@UJ-002 resolves all four exact jobs through SPEC-074-owned descriptors, templates, marker pairs, and source-level digests plus SPEC-075-owned site-commit resolution, immutable rendered anchors, reconstruction, and rendered-digest acceptance.
    tests: [TestWebsiteJourney_GeneratedObligationMatrixPasses]
  - id: CLM-024
    requirement: REQ-003
    text: An unknown or wrong generated job, owner, output, descriptor, template, marker, source-level digest, site identity, realized anchor set, predecessor version, missing SPEC-075 reconstruction verdict, Seed-5-resolved or independently reconstructed region, or evidence/boundary record mislabeled as generated fails with global key and job.
    tests: [TestWebsiteJourney_RejectsInvalidGeneratedObligationMatrix]
  - id: CLM-028
    requirement: REQ-003
    text: CAP-014/@UJ-001 traverses one anchor inside BOUNDARY-005 carrying both data-journey-link-id="JLINK-024" and data-boundary-continuation, with exact href, after explanation and before guarantee denial, while Seed 5 reads no source metadata.
    tests: [TestWebsiteJourney_CAP014DualIdentityAnchorPasses]
  - id: CLM-029
    requirement: REQ-003
    text: A missing identity, anchor, or href, split identities, duplicate anchor, wrong href, containment, explanation-link-denial order, or visible explanation/link/denial bytes, or Seed 5 source parsing/canonicalization fails with CAP-014/@UJ-001, JLINK-024, and BOUNDARY-005.
    tests: [TestWebsiteJourney_RejectsCAP014DualIdentityAnchorMatrix]
  - id: CLM-007
    requirement: REQ-004
    text: The exact four successful upstream commands are consumed as prerequisites by every capability.
    tests: [TestWebsiteJourney_ExactDependencyVerdictMatrixPasses]
  - id: CLM-008
    requirement: REQ-004
    text: A missing, skipped, stale, synthetic, nonzero, wrong-command, or directly reimplemented prerequisite fails before verification.
    tests: [TestWebsiteJourney_RejectsInvalidDependencyVerdictMatrix]
  - id: CLM-009
    requirement: REQ-005
    text: All twenty-two journeys pass against the exact Seed 4 built site through rendered link traversal and semantic outcomes.
    tests: [TestWebsiteJourney_BuiltSiteAllJourneysPass]
  - id: CLM-010
    requirement: REQ-005
    text: The same twenty-two journeys pass against the commit/run-matched backstop.sh deployment without redirect, stale-marker, keyword, screenshot, or direct-load substitution.
    tests: [TestWebsiteJourney_DeployedSiteAllJourneysPass]
  - id: CLM-011
    requirement: REQ-005
    kind: absence
    text: Website capabilities add no application runtime or client JavaScript.
    tests: [TestWebsiteJourney_HasNoPublishedRuntime]
  - id: CLM-012
    requirement: REQ-005
    text: Success and failure remove all temporary build, server, browser, mutation, dependency-fixture, and coverage state.
    tests: [TestWebsiteJourney_VerificationCleanupPasses]
  - id: CLM-025
    requirement: REQ-005
    text: CAP-009 browser traversal plus the three provenance-bound rendered adoption commands installs v0.2.0 into the disposable repository, writes configuration, and produces a real zero-exit gate receipt.
    tests: [TestWebsiteJourney_AdoptionCommandsProduceWorkingGate]
  - id: CLM-026
    requirement: REQ-005
    text: A missing or changed instruction ID, rendered command, digest, executable, argv, environment, installed binary, configuration, or final gate result, or a shell-evaluated, mocked, copied, or preinstalled substitute fails CAP-009.
    tests: [TestWebsiteJourney_RejectsInvalidAdoptionExecutionMatrix]
  - id: CLM-013
    requirement: REQ-006
    text: Removing each of the ten canonical routes independently breaks at least one named global journey.
    tests: [TestWebsiteJourney_EveryCanonicalRouteRemovalBreaksJourney]
  - id: CLM-014
    requirement: REQ-006
    text: Removing every mapped evidence element or its durable source link independently breaks a named global journey.
    tests: [TestWebsiteJourney_EveryEvidenceEdgeRemovalBreaksJourney]
  - id: CLM-015
    requirement: REQ-006
    text: Removing or failing each of the four upstream verdicts independently breaks every dependent capability before traversal.
    tests: [TestWebsiteJourney_EveryDependencyVerdictRemovalBreaksCapabilities]
  - id: CLM-016
    requirement: REQ-006
    text: Removing every mapped boundary element, state, explanation, continuation link, or guarantee denial independently breaks a named global journey.
    tests: [TestWebsiteJourney_EveryBoundaryExplanationRemovalBreaksJourney]
  - id: CLM-017
    requirement: REQ-006
    text: Every mutation diagnostic names its class, target, and specifically broken global journey rather than accepting an upstream or generic failure alone.
    tests: [TestWebsiteJourney_MutationDiagnosticsAreJourneySpecific]
  - id: CLM-027
    requirement: REQ-006
    text: For every mapped generated obligation, independently removing its region, job ID, owner anchor, begin marker, end marker, digest, or each source link breaks CAP-011/@UJ-001 or CAP-013/@UJ-002 as mapped.
    tests: [TestWebsiteJourney_EveryGeneratedProvenanceMutationBreaksJourney]
  - id: CLM-030
    requirement: REQ-006
    text: Independently deleting CAP-014/@UJ-001's shared anchor, either identity, or href; splitting or duplicating the anchor; changing its target; moving it outside BOUNDARY-005; moving it before explanation or after denial; or altering explanation, link-label, or denial visible bytes breaks that journey through the SPEC-075 owner verdict with a diagnostic naming CAP-014/@UJ-001, JLINK-024, and BOUNDARY-005.
    tests: [TestWebsiteJourney_EveryCAP014DualIdentityMutationBreaksNamedJourney]
  - id: CLM-018
    requirement: REQ-007
    kind: absence
    text: The Seed 5 runner contains no copied product prose, concept model, semantic or visual rule, generated-truth renderer, layout threshold, or parallel route/link verifier.
    tests: [TestWebsiteJourney_RemainsIntegrationConsumer]
  - id: CLM-019
    requirement: REQ-007
    text: Any single valid measured websitejourney coverage total at or above 80.00 percent passes.
    tests: [TestWebsiteJourney_VerifierAcceptsCoverageAtThreshold]
  - id: CLM-020
    requirement: REQ-007
    text: Coverage at 79.99, or an absent, duplicate, or nonnumeric total, fails closed.
    tests: [TestWebsiteJourney_VerifierRejectsCoverageFailureMatrix]
  - id: CLM-021
    requirement: REQ-007
    text: CI blocks on built acceptance and Pages runs deployed acceptance only after SPEC-075 authoritative deployment proof.
    tests: [TestWebsiteJourney_WorkflowWiringPasses]
  - id: CLM-022
    requirement: REQ-007
    text: A missing generic capability primitive stops at a named separately governed dependency instead of expanding Seed 5 implementation.
    tests: [TestWebsiteJourney_RefusesUngovernedGenericPrimitive]
---

# SPEC-076: End-to-End Website Capabilities

## Overview

BUNDLE-032 succeeds only when the product surface enables concrete visitor outcomes. This spec
turns the eleven stable outcome seeds into first-class Backstop capability artifacts and executable
journeys. It is the integration capstone: Seeds 1 through 4 remain authoritative for what pages mean,
how claims are evidenced, how dependencies are governed, how product truth is generated, and how the
site is rendered and deployed. Seed 5 proves that those surfaces compose into something a visitor can
actually use.

The existing capability ID ledger has reservations through CAP-003, while only CAP-001 currently has
a checked-in artifact. The website cohort therefore begins at CAP-004; it never backfills reserved IDs.
@UJ IDs remain local to each feature file under the capability schema, so the acceptance identity is
the unambiguous pair `CAP-NNN/@UJ-NNN`.

## Requirements

### Capability matrix

| ID | Slug / exact directory | Title | Outcome |
|---|---|---|---|
| CAP-004 | `understand-backstop` | Understand Backstop | Determine what Backstop is, is not, and why it exists. |
| CAP-005 | `evaluate-fit` | Evaluate Fit | Decide whether Backstop addresses a concrete problem. |
| CAP-006 | `evaluate-guarantees` | Evaluate Guarantees | Separate shipped behavior, guarantee, limitation, plan, non-goal, and guidance. |
| CAP-007 | `evaluate-compatibility` | Evaluate Compatibility | Determine operability and where it stops short of lifecycle guarantee. |
| CAP-008 | `understand-system` | Understand the System | Build an accurate artifacts, packs, gates, capabilities, and architecture model. |
| CAP-009 | `adopt-backstop` | Adopt Backstop | Reach a working installation and configuration for a repository. |
| CAP-010 | `apply-backstop` | Apply Backstop | Select a concrete use-case path and its next action. |
| CAP-011 | `browse-pack-ecosystem` | Browse the Pack Ecosystem | Find packs and determine which addresses the problem. |
| CAP-012 | `extend-backstop` | Extend Backstop | Decide when a concern belongs in a pack and how to author one. |
| CAP-013 | `inspect-evidence` | Inspect the Evidence | Trace public claims through durable evidence. |
| CAP-014 | `continue-beyond-backstop` | Continue Beyond Backstop | Recognize an intentional boundary and continue with honest guidance. |

Every capability has the exact same dependency spine and gate posture required by REQ-001. Its two
scenarios are not a minimal click tour: each terminates only after the declared semantic obligation is
visible and connected to its owner record.

### Journey matrix

Each arrow is one predecessor-owned rendered link. Link IDs are physical owner contracts; the final
row deliberately reuses JLINK-009 because both scenarios consume the same compatibility-boundary
edge rather than requiring duplicate page copy.

| Global key | Scenario | Exact ordered route/anchor traversal | Link IDs |
|---|---|---|---|
| CAP-004/@UJ-001 | Recognize the failure class and why Backstop exists | `/#why-backstop` -> `/evaluate/#failure-fit` | JLINK-001 |
| CAP-004/@UJ-002 | Distinguish what Backstop is from what it is not | `/evaluate/#what-backstop-is` -> `/model/#operating-model` | JLINK-002 |
| CAP-005/@UJ-001 | Confirm fit and continue to adoption | `/use-cases/#choose-use-case` -> `/evaluate/#fit-decision` -> `/adopt/#install` | JLINK-003, JLINK-004 |
| CAP-005/@UJ-002 | Confirm no-fit and continue to boundary guidance | `/evaluate/#not-a-fit` -> `/status/#adjacent-guidance` | JLINK-005 |
| CAP-006/@UJ-001 | Distinguish a shipped mechanism from a guarantee | `/evaluate/#guarantees` -> `/status/#supported-and-limited` | JLINK-006 |
| CAP-006/@UJ-002 | Compare every public boundary state and its implication | `/status/#boundary-states` -> `/model/#ownership-boundaries` | JLINK-007 |
| CAP-007/@UJ-001 | Determine whether a named harness, model, or toolchain can operate Backstop | `/evaluate/#compatibility` -> `/reference/#compatibility` | JLINK-008 |
| CAP-007/@UJ-002 | Determine which lifecycle guarantees that compatibility does not preserve | `/evaluate/#compatibility-limits` -> `/status/#adjacent-guidance` | JLINK-009 |
| CAP-008/@UJ-001 | Follow the artifact-to-plan-to-gate operating model | `/model/#operating-model` -> `/reference/#artifact-schema-catalog` | JLINK-010 |
| CAP-008/@UJ-002 | Inspect architecture and ownership boundaries | `/model/#ownership-boundaries` -> `/status/#project-boundaries` | JLINK-011 |
| CAP-009/@UJ-001 | Move from installation instructions to working configuration guidance | `/adopt/#install` -> `/reference/#configuration` | JLINK-012 |
| CAP-009/@UJ-002 | Verify the configured repository's enforcement path | `/adopt/#verify-enforcement` -> `/model/#enforcement-loop` -> `/reference/#gate` | JLINK-013, JLINK-014 |
| CAP-010/@UJ-001 | Select a concrete use case and its adoption action | `/use-cases/#choose-use-case` -> `/adopt/#adoption-paths` | JLINK-015 |
| CAP-010/@UJ-002 | Connect a use case to an applicable pack | `/use-cases/#pack-backed-use-cases` -> `/packs/#choose-a-pack` | JLINK-016 |
| CAP-011/@UJ-001 | Browse the generated installed-pack catalog | `/packs/#installed-pack-catalog` -> `/reference/#pack-commands` | JLINK-017 |
| CAP-011/@UJ-002 | Determine which pack addresses a problem and inspect its status | `/packs/#choose-a-pack` -> `/status/#pack-direction` | JLINK-018 |
| CAP-012/@UJ-001 | Decide whether a concern belongs in a pack and start authoring | `/extend/#pack-or-not` -> `/reference/#pack-artifact` | JLINK-019 |
| CAP-012/@UJ-002 | Continue from pack authoring to the contribution path | `/extend/#author-a-pack` -> `/contributing/#contribution-paths` | JLINK-020 |
| CAP-013/@UJ-001 | Trace an evaluation claim to its durable source | `/evaluate/#evidence` -> `/reference/#source-traceability` | JLINK-021 |
| CAP-013/@UJ-002 | Trace all generated product truth to authoritative sources | `/packs/#installed-pack-catalog` -> `/reference/#cli-command-catalog` -> `/status/#release-history` | JLINK-022, JLINK-023 |
| CAP-014/@UJ-001 | Follow adjacent guidance beyond an intentional boundary | `/status/#adjacent-guidance` -> `/contributing/#external-ownership` | JLINK-024 |
| CAP-014/@UJ-002 | Confirm that adjacent guidance is not a Backstop guarantee | `/evaluate/#compatibility-limits` -> `/status/#adjacent-guidance` | JLINK-009 |

The graph deliberately reaches all ten canonical routes. No route exists solely to satisfy a route
counter: removing any route severs at least one concrete journey.

### Evidence, boundary, and generated matrix

The semantic role is a selection constraint over accepted Seed 1 records, not a new product-truth ID.
Implementation records the selected concrete `claim_id`/`boundary_id` in the journey map. Each role
must resolve to exactly one record; ambiguity is a failure requiring Seed 1 truth to be clarified.

| Global key | Required evidence role | Required boundary role | Required generated jobs |
|---|---|---|---|
| CAP-004/@UJ-001 | mechanism claim explaining the enforced failure class, owned by `/` or `/evaluate/` | non-goal distinguishing Backstop from an agent/harness runtime | none |
| CAP-004/@UJ-002 | mechanism claim for the product category, owned by `/evaluate/` | limitation or non-goal naming what Backstop does not do | none |
| CAP-005/@UJ-001 | runtime-behavior or mechanism claim proving the selected use case is supported | supported boundary identifying the adoption path | none |
| CAP-005/@UJ-002 | mechanism or observed-failure claim establishing why the case is not a fit | adjacent-guidance record providing the no-fit continuation | none |
| CAP-006/@UJ-001 | runtime-behavior claim with execution evidence | supported boundary naming the shipped mechanism | none |
| CAP-006/@UJ-002 | mechanism evidence for status classification | one rendered record of each state: supported, limitation, planned, non-goal, adjacent-guidance | none |
| CAP-007/@UJ-001 | compatibility claim with reproducible or captured execution | limitation separating operability from preserved guarantees | none |
| CAP-007/@UJ-002 | compatibility claim explicitly naming unpreserved lifecycle guarantees | adjacent-guidance continuation for the unowned integration concern | none |
| CAP-008/@UJ-001 | mechanism claim sourced to artifact, plan, pack, and gate implementation | non-goal preventing the model from claiming harness execution ownership | none |
| CAP-008/@UJ-002 | mechanism claim linked to an authoritative Mermaid architecture view | non-goal or adjacent-guidance record for an external toolchain boundary | none |
| CAP-009/@UJ-001 | runtime-behavior claim with reproducible installation/configuration execution | limitation naming prerequisites or unsupported adoption state | none |
| CAP-009/@UJ-002 | runtime-behavior claim with captured gate result | supported boundary naming current enforcement behavior | none |
| CAP-010/@UJ-001 | mechanism or runtime-behavior claim attached to the selected use case | limitation or adjacent-guidance for the use case's stopping point | none |
| CAP-010/@UJ-002 | mechanism claim connecting use case to pack | planned, limitation, or adjacent-guidance record when no released pack owns the concern | none |
| CAP-011/@UJ-001 | mechanism claim identifying catalog meaning | limitation that catalog presence is not a universal guarantee | `installed-pack-catalog` |
| CAP-011/@UJ-002 | mechanism claim identifying pack purpose and release state | supported, planned, or limitation boundary matching that state | none |
| CAP-012/@UJ-001 | mechanism claim for the pack-extension decision and authoring path | non-goal for concerns that do not belong in a Backstop pack | none |
| CAP-012/@UJ-002 | mechanism claim for contribution ownership | adjacent-guidance record where another repository owns the work | none |
| CAP-013/@UJ-001 | consequential claim with one durable mechanism source and claim-type-appropriate second source | none | none |
| CAP-013/@UJ-002 | mechanism claim explaining the derivation chain | none | `cli-command-catalog`, `artifact-schema-catalog`, `installed-pack-catalog`, `release-history` |
| CAP-014/@UJ-001 | mechanism claim supporting why the boundary exists | exact `BOUNDARY-005` adjacent-guidance with reason, JLINK-024 continuation, and guarantee denial | none |
| CAP-014/@UJ-002 | compatibility or mechanism claim whose guarantee boundary is explicit | adjacent-guidance explicitly denying a Backstop guarantee | none |

`none` means the journey has no obligation of that kind. Roles permitting multiple boundary states
require the map to choose one exact accepted record and state. CAP-006's second scenario is the sole
five-record boundary row because its outcome is to distinguish the whole state vocabulary.

The generated obligation shape is exact for each named job: job ID; SPEC-074 owner route/anchor;
registered source output; literal product-truth begin/end markers; literal
`<!-- PRODUCT-TRUTH:SOURCES-BEGIN job=<ID> owner=<ROUTE>#<ANCHOR> digest=sha256:<HEX> -->` and
`<!-- PRODUCT-TRUTH:SOURCES-END job=<ID> -->` markers; complete ordered typed descriptors and URL
templates; and the canonical lowercase 64-hex source-level provenance-envelope digest. It separately
binds the complete immutable anchor set and reconstruction/digest verdict realized under SPEC-075
v1.0.4. The product-truth begin marker and sources-begin marker carry the same owner digest. Seed 5 compares those owner outputs and
verdicts; it neither substitutes `<SITE-COMMIT>` nor reconstructs the envelope. The four exact
job/owner/output tuples remain:

| Job | Owner | Output | SPEC-074 source descriptor / URL-template contract |
|---|---|---|---|
| `cli-command-catalog` | `/reference/#cli-command-catalog` | `docs/_includes/generated/cli-command-catalog.md` | One `{kind: tree, commit_binding: site, path: cmd/backstop}` descriptor; template `https://github.com/backstop-ai/backstop-core/tree/<SITE-COMMIT>/cmd/backstop`. |
| `artifact-schema-catalog` | `/reference/#artifact-schema-catalog` | `docs/_includes/generated/artifact-schema-catalog.md` | One ordered `{kind: blob, commit_binding: site, path: <record.source>}` descriptor per schema record and no others; template `https://github.com/backstop-ai/backstop-core/blob/<SITE-COMMIT>/<record.source>`. |
| `installed-pack-catalog` | `/packs/#installed-pack-catalog` | `docs/_includes/generated/installed-pack-catalog.md` | Exactly two ordered `{kind: blob, commit_binding: site}` descriptors for `backstop.yml`, then `backstop.lock`; corresponding blob URL templates with `<SITE-COMMIT>`. |
| `release-history` | `/status/#release-history` | `docs/_includes/generated/release-history.md` | One ordered `{kind: commit, commit_binding: record, commit: <record.commit>}` descriptor per release record and no others; template `https://github.com/backstop-ai/backstop-core/commit/<record.commit>`. |

`<SITE-COMMIT>` remains an exact literal, HTML-encoded token in SPEC-074 source output; Seed 3 does
not resolve it and does not own a rendered anchor. SPEC-075 v1.0.4 alone substitutes the full
lowercase 40-hex HEAD for built acceptance or authoritative page-deployment commit for deployed
acceptance, renders the immutable anchors, reconstructs the canonical descriptor/record envelope,
and compares the source-level and rendered digests. The Seed 5 runner consumes that accepted
reconstruction and compares the realized link set structurally against the owner descriptors and
site identity; it never resolves the token itself or follows a mutable branch URL.

### Required predecessor contracts

SPEC-076 owns none of the following page semantics. Its map declares these exact predecessor versions
and acceptance fails before traversal when an owner version, contract, or field is missing.

| Owner version | Required owner contract |
|---|---|
| SPEC-072 v1.0.5 | The exact JLINK records above in `content-topology.yml`; stable structured boundary `explanation_markdown`, continuation, and guarantee-denial fields in `product-model.yml`; the BOUNDARY-005/CLAIM-005/JLINK-024 single-source-link composition; and the exact adoption instruction records below. Seed 1 source pages own the link and instruction copy. |
| SPEC-074 v1.0.4 | Every generated job exports its complete ordered typed source-descriptor set, exact URL templates, marker pair, and canonical source-level provenance-envelope digest. Site-bound descriptors retain literal `<SITE-COMMIT>`; Seed 3 does not resolve it or create rendered anchors. |
| SPEC-075 v1.0.4 | Owner-declared rendered JLINK and boundary-field markers; JLINK-024/BOUNDARY-005 single-anchor dual identity and order; `<SITE-COMMIT>` resolution; immutable generated anchors; generated descriptor/record reconstruction and rendered-digest acceptance; and adoption instruction ID/digest markers. Sitecheck verifies these upstream rendering contracts. |

For boundary obligations, `data-boundary-id`, `data-boundary-state`, and
`data-boundary-explanation` are mandatory. `adjacent-guidance` additionally requires exactly one
`a[data-boundary-continuation]` and one `[data-boundary-guarantee-denial]`; any other row that names a
continuation or denial in the matrix requires that corresponding marker. Seed 5 compares structured
IDs and presence/cardinality only; it never judges the prose.

CAP-014/@UJ-001 is the deliberate overlap between the journey-link and boundary-continuation
contracts. JLINK-024 and BOUNDARY-005 do not produce two links. The accepted rendered shape is one
anchor inside the BOUNDARY-005 callout with both identities, exact destination, and the DOM order
`[data-boundary-explanation]`, shared anchor, `[data-boundary-guarantee-denial]`. Seed 5 inspects that
one DOM relationship and consumes SPEC-075's JLINK-024/BOUNDARY-005-keyed verdict for exact visible
explanation, link-label, and denial byte preservation. It does not read, strip, normalize,
canonicalize, or otherwise interpret CLAIM-005, the source marker, or boundary source metadata.

### Adoption command matrix

SPEC-072 v1.0.5 owns these exact three structured records and displayed commands. The executable,
argv, and environment are owner data; SPEC-076 executes that structure directly and merely proves
the rendered code block has the same command digest.

| Instruction | Owner | Exact rendered command | Structured execution |
|---|---|---|---|
| ADOPT-INSTALL | `/adopt/#install` | `GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0` | executable `go`; argv `install`, `github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0`; env `GOBIN=<disposable-root>/.backstop-bin` |
| ADOPT-CONFIGURE | `/adopt/#configure` | `backstop init` | executable `<disposable-root>/.backstop-bin/backstop`; argv `init`; no added environment |
| ADOPT-ENFORCE | `/adopt/#verify-enforcement` | `backstop gate` | executable `<disposable-root>/.backstop-bin/backstop`; argv `gate`; no added environment |

## Implementation

The planner must preserve this order:

1. Reserve CAP-004 through CAP-014 through the real CLI in numerical order and fill the exact
   artifact matrix without changing an existing capability reservation or CAP-001.
2. Require the independently reviewed SPEC-072 v1.0.5, SPEC-074 v1.0.4, and SPEC-075 v1.0.4
   owner versions. Verify their exact link, structured-boundary, generated-source-link,
   adoption-instruction, and rendered-marker contracts before Seed 5 implementation continues.
3. After Seed 1 implementation has accepted claim and boundary IDs, select records satisfying each
   semantic role and write `docs/_data/website-capability-map.yml`. Import the four generated job
   records, declare all three predecessor versions, and refuse ambiguous or absent roles; do not mint
   aliases, copy public statements, or reconstruct generated provenance in the map.
4. Author the eleven feature files with their exact two scenarios. Each Given/When/Then sequence must
   express the route traversal and terminal semantic obligation from the matrices, not implementation
   details or keyword assertions.
5. Implement artifact/map validation and exact Gherkin-to-test coverage. Test identities use the
   global key so repeated local @UJ IDs cannot collide in reports.
6. Implement built-tree traversal. Begin at the scenario's first route, follow the exact owner-rendered
   JLINK for every subsequent route, resolve exact anchors, and verify typed evidence, boundary,
   generated-provenance, or continuation obligations through owner markers. Treat JLINK-024 and
   BOUNDARY-005 as one rendered anchor with two identities and verify containment/order from DOM only;
   do not parse or canonicalize their source representation.
7. Implement CAP-009's execution proof. Create a disposable Git repository, execute the three
   structured owner commands directly in order, compare their canonical display digests with the
   rendered blocks, and require installed binary, generated configuration, and real zero-exit gate
   receipt. Never pass rendered text to a shell.
8. Compose public upstream verification entrypoints once, then run the complete built-site cohort.
   Seed 5 records only exit verdict and attribution; owner logic remains opaque.
9. Run all five mutation classes exhaustively over isolated copies, including every field of every
   generated obligation and every same-anchor identity, link, cardinality, containment, and order
   variant plus each visible explanation/link-label/denial byte variant for CAP-014/@UJ-001. Consume
   SPEC-075's keyed owner verdict for byte preservation rather than implementing a source comparison.
   Report failures in stable capability/UJ order and prove each mutant is killed by a journey-specific assertion.
10. Wire built acceptance into branch CI. After Seed 4 deploys and proves authoritative identity, run
   deployed mode against the same commit/run-marked pages. A post-deploy failure reports a broken
   capability but does not make GitHub Pages deployment transactional.
11. Promote a capability to `verified` only when artifact coverage, built acceptance, mutation
   acceptance, and the matching deployed acceptance are green for the same release identity.

The map is an integration graph, not an alternate evidence inventory. It may contain record IDs,
routes, anchors, JLINK IDs, obligation kinds, states/types/job IDs, dependency IDs, command provenance
references, and scenario edges; it may not contain claim prose, boundary explanations, generated
records, evidence-source definitions, adoption copy, or copied upstream rules.

## Verification

`./scripts/verify-website-capabilities.sh` is the single Seed 5 entrypoint. With no deployment flags,
it runs measured `scripts/websitejourney` tests, invokes the four upstream verification contracts,
builds the exact Seed 4 site into a disposable tree, validates all capability/feature/test mappings,
executes every browser journey, runs CAP-009's disposable-repository proof, and kills the complete
five-class mutation matrix. `--capability` filters reporting and journey execution only after the
complete shared artifact/dependency/map integrity checks pass.

Deployed mode accepts only the canonical HTTPS origin plus explicit commit and run identity. It does
not follow aliases or redirects, and it refuses any canonical page whose deployment marker differs.
It reruns semantic journey traversal, not the destructive mutation matrix, against production. Pages
workflow ordering ensures SPEC-075 authoritative API/HTTPS proof precedes this check.

Coverage applies to the Go integration parser/runner and mutation machinery; Playwright scenario
execution remains mandatory but is not misrepresented as Go statement coverage. A green route/link
verifier with zero executed Gherkin scenarios is a failure, as is a complete click tour that never
resolves its declared evidence, boundary, or generated records. CAP-009 remains red if its browser
journeys pass but any provenance-bound adoption command does not execute to a real gate result.
CAP-014/@UJ-001 remains red unless the one traversed link is simultaneously the one BOUNDARY-005
continuation in the required DOM position; source parsing or two lookalike anchors cannot substitute.

## Integration Contract

- SPEC-072 owns every route, claim, boundary, canonical concept, and final-copy statement. Seed 5
  imports IDs and inspects Seed 4's rendered data attributes. Its v1.0.5 contract owns JLINK records,
  structured boundary fields, and adoption instruction structure/copy.
- SPEC-073 owns released documentation-semantics consumption and finding attribution. Seed 5 consumes
  its verification command's verdict and never calls its engine.
- SPEC-074 owns four generated regions, typed source descriptors, URL templates, and source-level
  provenance-envelope digests. Its v1.0.4 contract leaves `<SITE-COMMIT>` unresolved and owns no
  rendered anchor. Seed 5 consumes typed generated obligations but never generates, resolves, or
  reconstructs them.
- SPEC-075 owns Jekyll, links, navigation, browser interaction, visual policy, deployment, and the
  commit/run marker. Its v1.0.4 contract owns the JLINK-024/BOUNDARY-005 single dual-identity anchor,
  resolves `<SITE-COMMIT>`, owns immutable generated anchors
  and rendered descriptor/record reconstruction/digest acceptance, and owns rendered JLINK,
  structured boundary, generated-link, and adoption-command markers. Seed 5 consumes those surfaces
  and does not restate their checks.

## Sharp Edges

- **Capability IDs have tombstones.** CAP-002 and CAP-003 are reserved even though their artifacts are
  absent. Backfilling them would violate the ledger and make later history ambiguous.
- **@UJ identity is scoped.** Reusing @UJ-001 in eleven feature files is schema-valid; reporting only
  `@UJ-001` is not. Every test and diagnostic must carry the CAP/UJ pair.
- **A click tour can be vacuously green.** Reaching a URL proves navigation, not understanding. Each
  terminal edge must resolve a registry-backed semantic obligation and a real continuation/source.
- **Journey links belong to page owners.** Seed 5 naming an anchor does not authorize it to add copy or
  navigation. Missing or wrong-version SPEC-072/SPEC-075 contracts keep acceptance red rather than inviting a local
  test-only link.
- **Two registries can describe one physical link.** JLINK-024 is BOUNDARY-005's continuation, not a
  second CTA. Checking the identities independently would let split or duplicate anchors pass and
  would tempt Seed 5 to reimplement Seed 1's source canonicalization. Same-element identity,
  containment, order, and the SPEC-075-owned visible-byte verdict are therefore mandatory rendered
  checks.
- **Generated provenance is not ordinary evidence.** A product-truth job has output, owner, marker
  pair, digest, and source links that must stay typed together. Treating it as a claim ID or checking
  only visible table text would let a hand-reconstructed region pass.
- **Instructions are not adoption until executed.** Browser-visible commands can be stale, unsafe, or
  disconnected from the shipped binary. CAP-009 requires structured owner provenance and a real
  disposable install/configure/gate run, without turning displayed shell text into executable input.
- **Seed 5 can become a policy landfill.** Parsing rendered owner IDs is integration; deciding whether
  prose is a guarantee, whether focus is visible, or whether generated records are current belongs to
  upstream owners.
- **A route deletion may fail upstream first.** Mutation acceptance still must attribute a specifically
  broken journey. Merely observing SPEC-075's generic route error does not prove the capability suite.
- **Production can be newer than the workflow under review.** Commit/run markers on every visited page
  prevent a green test against an unrelated deployment.
- **Cross-cutting journeys are not a forced funnel.** Evidence inspection and boundary continuation
  start where their relevant record appears; they are not hidden behind an artificial linear path.
- **External guidance is not a guarantee.** A working outbound HTTPS link does not make Backstop own or
  warrant the adjacent tool. The boundary record and explicit denial remain load-bearing.
- **Post-deploy failure cannot roll back Pages atomically.** It marks the deployed capability broken and
  blocks promotion; it must not claim the preceding publication never occurred.
- **Mutation fixtures can accidentally edit the source tree.** All destructive variants operate on
  disposable built/dependency copies and cleanup is asserted on both success and failure.

## Review Questions

1. Are CAP-004 through CAP-014 the exact eleven artifacts, with CAP-002 and CAP-003 left reserved?
2. Does every feature contain exactly its two local @UJ tags and every report use the global CAP/UJ key?
3. Does each intermediate route arise from its exact owner-declared JLINK and source/destination
   anchors rather than a direct load, global navigation, or Seed 5-authored link?
4. Does every semantic role resolve once to a concrete accepted Seed 1 record through stable field
   markers without copied or parsed prose?
5. Do CAP-011/@UJ-001 and CAP-013/@UJ-002 bind the exact generated jobs, marker pairs, digests,
   owners, outputs, and complete source links as a third obligation kind?
6. Does CAP-014/@UJ-001 traverse one anchor that simultaneously carries JLINK-024 and BOUNDARY-005
   continuation identities in exact containment/order and preserve exact explanation/link/denial
   visible bytes through the SPEC-075 owner verdict, without Seed 5 source parsing?
7. Does CAP-009 execute the provenance-bound rendered install/configure/enforce sequence against a
   disposable repository and produce a real gate receipt after its browser journey?
8. Can deleting any canonical route, mapped evidence edge, prerequisite verdict, boundary
   explanation, or generated-provenance field be tied to a specifically broken journey?
9. Does the runner consume public Seed 1-4 contracts without duplicating semantic, generation, visual,
   route, browser, or deployment enforcement?
10. Does deployed acceptance prove every visited page belongs to the expected commit and run?
11. Can a capability reach `verified` only after both built and identity-matched deployed gates pass?
12. If implementation needs a generic capability/runtime primitive, does work stop for separate
    governance rather than expanding this integration runner?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` v0.6.0 — REQ-007@2.0.0, OQ-2, DD-7,
  Seed 5, User Capability Seeds, success criteria, and mutation acceptance.
- `specs/SPEC-072-public-product-model.spec.md` v1.0.5 — ten route owners, claim/evidence inventory,
  five-state boundary registry, canonical model, final-copy boundary, exact journey links, structured
  boundary fields, the BOUNDARY-005/CLAIM-005/JLINK-024 single-source-link composition, and adoption
  instruction contracts.
- `specs/SPEC-073-documentation-semantics-integration.spec.md` v1.0.0 — released semantic-pack
  consumer verdict and owner boundary.
- `specs/SPEC-074-derived-product-truth-pipeline.spec.md` v1.0.4 — four generated regions, typed
  source descriptors and URL templates, source-level provenance-envelope digests,
  source-to-Markdown chain, and Seed 5 consumption seam.
- `specs/SPEC-075-static-public-site-design-system.spec.md` v1.0.4 — built/deployed site contract,
  exact routes, `<SITE-COMMIT>` resolution, immutable rendered generated anchors and reconstruction,
  owner-rendered journey/boundary/generated/adoption markers, the JLINK-024/BOUNDARY-005 single
  dual-identity anchor with visible-byte preservation, installed design-system acceptance, and
  deployment marker.
- `artifacts/capability/v1/schema.json` — capability directory, scenario, quality-gate, traceability,
  and lifecycle contract.
- `capabilities/CAP-001-pack-gate-enforcement/capability.yml` and
  `capabilities/CAP-001-pack-gate-enforcement/user-journeys.feature` — current capability artifact and
  local @UJ precedent.
