---
title: "Derived Product Truth Pipeline"
number: SPEC-074
created: "2026-08-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.2

implementation:
  summary: >
    BUNDLE-032 Seed 3 only: add a repository-local, Backstop-specific generator that
    converts four named authoritative product-truth inputs into four checked-in,
    inspectable Markdown fragments consumed by the Seed 1 page owners. The generated
    surfaces are the CLI command catalog, artifact-schema catalog, installed-pack
    catalog, and release history. A checked-in manifest binds every job to its exact
    inputs, output, owner route and anchor, ownership marker, and one regeneration
    command. Git release inputs come from logical refs through Git plumbing. Generation
    canonicalizes exact raw-HTML table records, uses a recoverable journaled four-file
    transaction without claiming cross-file atomic visibility, and produces byte-identical output. Check mode renders to a
    temporary tree and refuses missing, stale, manually edited, multiply owned, or
    unregistered output while naming the job, sources, and output. Jekyll owner pages
    include each generated fragment exactly once and rendered regions reproduce its record digest.
    Every job also exports a complete typed authoritative-source descriptor set inside that
    digest boundary; Seed 4 resolves `site` commit bindings to the full deployment commit and
    renders only immutable GitHub tree, blob, or commit links.
    Stable-tag publication is blocked until latest main carries regenerated history. No parallel
    product registry is introduced. The implementation is deliberately domain-specific and
    must not become a generalized transformation engine, absorb the separately governed
    documentation-semantics pack, or take presentation ownership from Seed 4.
  subject: scripts/producttruth

verification:
  level: integration
  coverage_threshold: 80
  test_command: ./scripts/verify-product-truth.sh

contracts:
  - file: docs/_data/derived-product-truth.yml
    provides:
      - name: derived_product_truth_jobs
        kind: variable
        signature: "jobs[] {id, inputs[], output, owner_route, owner_anchor, marker, command, source_link_policy}"
  - file: scripts/generate-product-truth.sh
    provides:
      - name: generate_product_truth
        kind: function
        signature: "generate_product_truth [--write|--check|--recover]"
    consumes:
      - source: docs/_data/derived-product-truth.yml
        name: derived_product_truth_jobs
        kind: variable
  - file: scripts/producttruth/main.go
    provides:
      - name: product_truth_generator
        kind: function
        signature: "main()"
    consumes:
      - source: cmd/backstop
        name: command_tree
        kind: variable
      - source: artifacts
        name: artifact_schema_corpus
        kind: variable
      - source: backstop.yml
        name: declared_pack_set
        kind: variable
      - source: backstop.lock
        name: locked_pack_identities
        kind: variable
      - source: "git for-each-ref + rev-parse + merge-base + show"
        name: logical_stable_release_refs
        kind: variable
  - file: scripts/producttruth/generate.go
    provides:
      - name: SourceLinkDescriptor
        kind: type
        signature: "SourceLinkDescriptor = TreeBlobSourceLink {kind, commit_binding, path} | CommitSourceLink {kind, commit_binding, commit}; no optional members"
      - name: RenderAll
        kind: function
        signature: "RenderAll(root string, manifest Manifest) ([]RenderedJob, error)"
      - name: CheckAll
        kind: function
        signature: "CheckAll(root string, rendered []RenderedJob) ([]Drift, error)"
  - file: scripts/producttruth/transaction.go
    provides:
      - name: WriteAll
        kind: function
        signature: "WriteAll(root string, rendered []RenderedJob) error"
      - name: Recover
        kind: function
        signature: "Recover(root string) error"
  - file: scripts/producttruth/site.go
    provides:
      - name: VerifyRenderedSite
        kind: function
        signature: "VerifyRenderedSite(root string, manifest Manifest, siteCommit string) error"
    consumes:
      - source: docs/_config.yml
        name: jekyll_configuration
        kind: variable
      - source: Gemfile.lock
        name: locked_jekyll_runtime
        kind: variable
  - file: scripts/verify-product-truth.sh
    provides:
      - name: verify_product_truth_pipeline
        kind: function
        signature: verify_product_truth_pipeline()
  - file: docs/_includes/generated/cli-command-catalog.md
    provides:
      - name: generated_cli_command_catalog
        kind: variable
        signature: "Generated Markdown + one immutable-tree source descriptor owned by /reference/#cli-command-catalog"
  - file: docs/_includes/generated/artifact-schema-catalog.md
    provides:
      - name: generated_artifact_schema_catalog
        kind: variable
        signature: "Generated Markdown + one immutable-blob source descriptor per schema row owned by /reference/#artifact-schema-catalog"
  - file: docs/_includes/generated/installed-pack-catalog.md
    provides:
      - name: generated_installed_pack_catalog
        kind: variable
        signature: "Generated Markdown + two immutable-blob source descriptors owned by /packs/#installed-pack-catalog"
  - file: docs/_includes/generated/release-history.md
    provides:
      - name: generated_release_history
        kind: variable
        signature: "Generated Markdown + one immutable-commit source descriptor per release row owned by /status/#release-history"
  - file: docs/reference.md
    consumes:
      - source: docs/_includes/generated/cli-command-catalog.md
        name: generated_cli_command_catalog
        kind: variable
      - source: docs/_includes/generated/artifact-schema-catalog.md
        name: generated_artifact_schema_catalog
        kind: variable
  - file: docs/packs.md
    consumes:
      - source: docs/_includes/generated/installed-pack-catalog.md
        name: generated_installed_pack_catalog
        kind: variable
  - file: docs/status.md
    consumes:
      - source: docs/_includes/generated/release-history.md
        name: generated_release_history
        kind: variable
  - file: .github/workflows/ci.yml
    provides:
      - name: derived_product_truth_drift_gate
        kind: variable
        signature: "Blocking generator, measured coverage, Jekyll build, and rendered-region verification"
  - file: .github/workflows/release.yml
    provides:
      - name: release_history_current
        kind: variable
        signature: "Latest-origin/main history gate required by goreleaser"

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      `docs/_data/derived-product-truth.yml` must declare exactly four generation jobs
      and no implicit jobs. `cli-command-catalog` consumes the deterministic JSON from
      `go run ./cmd/backstop commands` at the current checkout and writes
      `docs/_includes/generated/cli-command-catalog.md`, owned by
      `/reference/#cli-command-catalog`. `artifact-schema-catalog` consumes every
      checked-in `artifacts/*/v*/schema.json` plus `artifacts/base/schema.json` and writes
      `docs/_includes/generated/artifact-schema-catalog.md`, owned by
      `/reference/#artifact-schema-catalog`. `installed-pack-catalog` consumes
      `backstop.yml` and `backstop.lock` and writes
      `docs/_includes/generated/installed-pack-catalog.md`, owned by
      `/packs/#installed-pack-catalog`. `release-history` enumerates Git's logical `refs/tags`
      namespace with `git for-each-ref`, never `.git/refs/tags`, `packed-refs`, or another
      implementation path. Each exact stable SemVer `vMAJOR.MINOR.PATCH` ref is peeled with
      `git rev-parse --verify --end-of-options refs/tags/<TAG>^{commit}`, tested as an ancestor
      of `HEAD` with `git merge-base --is-ancestor`, and read with `git show -s`; loose/packed
      and annotated/lightweight representations must behave identically. Malformed, prerelease,
      and unreachable refs are excluded, while a qualifying unpeelable or unreadable ref fails
      `PT104_GIT_REF`. The job writes `docs/_includes/generated/release-history.md`,
      owned by `/status/#release-history`. Each job must name its complete inputs,
      output, owner route, owner anchor, marker, and the single command
      `./scripts/generate-product-truth.sh`; a missing input, unknown job, duplicate ID,
      duplicate output, duplicate owner route/anchor, path outside the repository, or
      generated file not registered by one job is PROHIBITED. Invoking that command with
      no mode is exactly equivalent to `--write`; `--recover` is the only additional mode,
      and any conflicting or unknown argument is refused before inputs are read.
  - id: REQ-002
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      The generator must render all four jobs deterministically from authoritative
      inputs only. It must fix locale to `C`, timestamps to UTC, line endings to LF,
      indentation and table formatting to one canonical form, and final newlines to
      exactly one. CLI commands are ordered by command path and then flag; schemas by
      artifact type and numeric schema version; packs by declared pack name; and release
      tags by descending semantic version, never lexical version order or filesystem
      iteration order. Pack output includes declared version, locked version, `git_ref`,
      and `content_hash` but excludes nondeterministic `install_date`. Release output
      includes tag, full peeled commit SHA, RFC3339 UTC commit date, and one-line subject.
      Exact record fields and visible columns are: CLI `name`, `path`, `description`, `flags[]`
      -> Command, Path, Description, Flags; schema `artifact_type`, `path_version`,
      `document_version`, `schema_id`, `title`, `source` -> Artifact type, Schema path version,
      Document version, Schema ID, Title, Source; pack `name`, `declared_version`,
      `locked_version`, `git_ref`, `content_hash` -> Pack, Declared version, Locked version,
      Git ref, Content SHA-256; release `tag`, `commit`, `committed_utc`, `subject` -> Version,
      Commit, Committed UTC, Subject. `artifact_type` and `path_version` derive authoritatively
      from the source path. Base schema is `(base,base)` and sorts first. Typed
      `artifacts/<TYPE>/vN/schema.json` records use `<TYPE>` and `vN`; when JSON `artifact_type`
      exists it must agree. Its absence is accepted only for the current `base` and
      `backstop-yml` schemas. `document_version` is the top-level scalar `version`, except
      `artifacts/backstop-yml/v1/schema.json` uses exact sentinel `not-declared` because that real
      schema has no document-version field. `$id` and `title` remain required scalars.
      The `backstop.yml` pack-key set must equal the `backstop.lock` pack-key set; each declared key
      must equal the lock key and lock `name`, declared version must equal locked version, the
      lock-only `git_ref` must equal `v<declared_version>`, and the lock-only `content_hash` must be
      exactly 64 lowercase hexadecimal characters, or the job fails `PT103_PACK_JOIN`.
      `install_date` is ignored. Other missing, nested, or non-scalar required
      fields fail rather than stringify. CLI flags are unique nonempty single-line strings sorted
      bytewise; the visible cell is `—` for none or individually escaped flags joined by literal
      `<br>`, which rendered reconstruction splits on line breaks. Running
      write mode twice with unchanged inputs, or rendering the same fixture inputs in
      different filesystem iteration orders, must produce byte-identical output and a
      clean second run.
  - id: REQ-003
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      Every generated file must start with one canonical HTML comment containing
      `GENERATED PRODUCT TRUTH`, the job ID, every authoritative input locator, the
      owner route and anchor, `./scripts/generate-product-truth.sh`, and `DO NOT EDIT`.
      It is followed by `<!-- PRODUCT-TRUTH:BEGIN job=<ID> digest=sha256:<HEX> -->`, exactly
      one raw-HTML table with the REQ-002 headers and `data-product-truth-job=<ID>`, REQ-009's
      exact source-descriptor block, and the matching
      `<!-- PRODUCT-TRUTH:END job=<ID> -->`. As amended by REQ-009, digest input is the
      explicit provenance-envelope struct containing job, output, owner, record structs, and
      source-link descriptor structs, serialized by Go `encoding/json` with HTML escaping disabled,
      no indentation, and one LF; map-backed records or descriptors are forbidden. Table scalar
      escaping is applied once in this order:
      `&` -> `&amp;`, `<` -> `&lt;`, `>` -> `&gt;`, `"` -> `&quot;`, `'` -> `&#39;`,
      backtick -> `&#96;`, and pipe -> `&#124;`; CRLF/CR normalizes to LF and embedded LF becomes
      literal `<br>`. NUL and C0 controls other than tab/newline fail. The marker is part of generated bytes.
      Write mode may create a missing registered output, and may atomically replace an
      existing output only when its first line is the valid marker for that same job.
      It must refuse to overwrite an existing unmarked file, a file marked for another
      job, a symlink, or any target that resolves outside the repository. All outputs share one
      recoverable journaled transaction directory: stage and fsync all bytes and `journal.json`,
      durably record backup/install rename transitions, and attempt rollback of every completed
      transition on error. Successful rollback restores the prior cohort. Kill or rollback failure
      retains journal/backups and makes check/write fail `PT203_TRANSACTION` until idempotent
      `--recover` restores the prior cohort and fsyncs the directory. No simultaneous cross-file
      atomic-visibility guarantee is made.
  - id: REQ-004
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      `./scripts/generate-product-truth.sh --check` and the blocking CI job must render
      all jobs into an isolated temporary tree, compare exact bytes with the checked-in
      outputs, and return nonzero for a missing output, manual output edit, stale output,
      marker mismatch, orphan generated file, or source/read/render failure. Each drift
      diagnostic must name the job ID, output path, complete authoritative input list,
      first differing line or missing-file state, and the regeneration command; it must
      never rewrite during check mode. Diagnostics use
      `product-truth[CODE] job=<job|pipeline> output=<path|-> inputs=<ordered-list>: <detail>`
      and stable classes `PT001_MANIFEST`, `PT101_COMMAND`, `PT102_SCHEMA`, `PT103_PACK_JOIN`,
      `PT104_GIT_REF`, `PT201_UNSAFE_TARGET`, `PT202_DRIFT`, `PT203_TRANSACTION`,
      `PT204_CONSUMPTION`, and `PT205_COVERAGE`. An outstanding transaction directory is never
      an ignored orphan; it fails PT203 until recovery. Fixture acceptance must independently mutate the
      command descriptor, one schema field, one declared/locked pack identity, and one
      release tag record and prove that each source change produces only the expected
      job's Markdown delta. Fixture acceptance must also tamper with each of the four
      generated outputs and prove CI-equivalent check mode refuses each tamper.
  - id: REQ-005
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      SPEC-072 owns routes `/reference/`, `/packs/`, and `/status/`; this spec introduces only
      subordinate anchors `cli-command-catalog`, `artifact-schema-catalog`,
      `installed-pack-catalog`, and `release-history`. Each owner source must contain exactly one
      `## <TITLE> {#<ANCHOR>}`, then `<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=<ID> -->`, the exact
      Liquid line `{% include generated/<FILE>.md %}`, and the matching include end marker.
      Seed 4 provides root `Gemfile` and committed `Gemfile.lock`; the acceptance build is exactly
      `bundle exec jekyll build --source docs --destination _site --trace`. Rendered verification
      requires exactly one source include region and one source/rendered fragment region, parses the
      rendered region with Go's HTML parser, requires one table with exact job attribute, headers,
      rows, cells, and no unknown row elements, HTML-decodes cells and converts `<br>` to LF,
      reconstructs the explicit REQ-002 structs and REQ-009 source-link descriptors, applies the
      canonical provenance-envelope digest, and compares it with both source markers. Missing,
      duplicated, moved, stale, independently reconstructed, or tampered regions fail PT204. No
      second generated page or site plugin may reread inputs.
  - id: REQ-006
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      `scripts/producttruth` is a closed, Backstop-specific renderer for exactly the four
      jobs in REQ-001. It must not expose arbitrary templates, arbitrary input/output
      mappings, plugin loading, user-defined transforms, expression evaluation, remote
      source fetching, or a reusable transformation API. It must not implement or copy
      documentation-semantic judgments from Seed 2, visual/presentation policy from Seed
      4, product claims or boundaries from Seed 1, or a machine-only/MCP publication.
      If delivery requires a generalized generation engine or generic transformation
      substrate, implementation must stop at a declared dependency seam and route that
      work to separate governance; local expansion of this renderer is not an alternate
      completion path.
  - id: REQ-007
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      `.github/workflows/release.yml` must add `release-history-current`, checking out latest
      `origin/main` with full history/tags separately from the pushed stable-tag checkout, fetching
      that tag, and running check mode there; GoReleaser must depend on this job. A new tag absent
      from latest main therefore cannot publish. Its generated row lands through normal PR/main
      CI and Pages, then the failed tag workflow is rerun; the tag retains the release commit while
      later main owns the public-history update. Seed 4's Pages workflow must run the same freshness
      check before deploy, and tag push itself must not deploy Pages. This two-checkout handshake
      avoids tag-SHA self-reference while preventing stale public release history.
  - id: REQ-008
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      `scripts/verify-product-truth.sh` must create a temporary atomic coverage profile, run
      `go test ./scripts/producttruth/... -race -covermode=atomic`, extract exactly one numeric
      `total:` from `go tool cover -func`, and fail PT205 when absent, duplicated, nonnumeric, or
      below 80.00. It then runs check mode, the exact locked Jekyll build, rendered-site verification,
      and structural workflow tests proving branch CI, Pages, and release publication invoke their
      gates. Generator, transaction, Git-plumbing, site-verifier, and workflow surfaces are included;
      temporary coverage/build outputs are removed on success and failure.
  - id: REQ-009
    supports:
      - website-expansion:REQ-011@1.0.0
    text: >
      Each exact job must declare its closed `source_link_policy` in
      `docs/_data/derived-product-truth.yml`; generation must apply that policy to the
      authoritative records and export the complete realized `source_links` descriptor set
      inside its generated region between exact
      `<!-- PRODUCT-TRUTH:SOURCES-BEGIN job=<ID> owner=<ROUTE>#<ANCHOR> digest=sha256:<HEX> -->`
      and `<!-- PRODUCT-TRUTH:SOURCES-END job=<ID> -->` markers. Between them is exactly one
      `<ul data-generated-source-descriptors data-product-truth-job="<ID>">`; every descriptor
      is one ordered `<li data-generated-source-descriptor data-source-kind="<KIND>"
      data-commit-binding="<BINDING>" data-source-path="<PATH>">` for `tree`/`blob`, or
      `data-source-commit="<COMMIT>"` for `commit`, whose text is the exact URL contract below.
      A path attribute on `commit`, a commit attribute on `tree`/`blob`, an unknown attribute,
      or any element outside that closed shape is invalid. The one digest in the
      existing `PRODUCT-TRUTH:BEGIN` marker and the sources-begin marker must match and must
      be SHA-256 over canonical JSON for the explicit envelope `{job,output,owner_route,
      owner_anchor,records,source_links}`; owner, output, generated records, and source
      provenance therefore cannot drift independently. The envelope is an explicit Go struct
      whose JSON members occur in exactly that written order; each job's `records` uses the
      exact field order declared in REQ-002, and descriptor order is normative.
      `SourceLinkDescriptor` is a closed union of two concrete structs, never one struct with
      optional fields. A tree descriptor serializes exactly three members in this order:
      `{"kind":"tree","commit_binding":"site","path":"<nonempty-path>"}`. A blob
      descriptor serializes exactly three members in this order:
      `{"kind":"blob","commit_binding":"site","path":"<nonempty-path>"}`. A commit
      descriptor serializes exactly three members in this order:
      `{"kind":"commit","commit_binding":"record","commit":"<40-lowercase-hex>"}`.
      For tree/blob, `commit` is absent; for commit, `path` is absent. An inapplicable member
      encoded as empty string or `null` is not equivalent to absence and is invalid; omitting,
      emptying, or nulling any applicable member is likewise invalid. Unknown members are invalid.
      Decoding must select the concrete variant from exact `kind`, reject unknown fields, then
      reserialize the concrete struct before digest comparison, so omitted-versus-empty-versus-null
      representations cannot share an accepted semantic digest. Input object member order does not
      alter this canonical reserialization, but any serializer or generated-envelope mutation that
      emits members outside the required order fails the exact canonical-byte/digest comparison.
      `cli-command-catalog` has exactly one `{kind: tree, commit_binding: site,
      path: cmd/backstop}` descriptor. `artifact-schema-catalog` has exactly one
      `{kind: blob, commit_binding: site, path: <record.source>}` descriptor for every
      rendered schema record in record order and no others. `installed-pack-catalog` has
      exactly two `{kind: blob, commit_binding: site}` descriptors in order for
      `backstop.yml` and `backstop.lock`. `release-history` has exactly one
      `{kind: commit, commit_binding: record, commit: <record.commit>}` descriptor per
      release record in record order and no others. In the descriptor model, a `site`
      descriptor uses literal token `<SITE-COMMIT>`; generated raw-HTML bytes encode that
      token as `&lt;SITE-COMMIT&gt;` so it remains text rather than an HTML element, and HTML
      decoding must recover the exact token in the URL;
      a record-bound release link contains its existing full lowercase 40-hex commit.
      Seed 4 must resolve `<SITE-COMMIT>` to the full lowercase 40-hex build/deployment
      commit and render each descriptor exactly once as `a[data-generated-source-link]`
      inside the one owner section for that job. Final URLs are exactly
      `https://github.com/backstop-ai/backstop-core/tree/<SITE-COMMIT>/cmd/backstop`,
      `https://github.com/backstop-ai/backstop-core/blob/<SITE-COMMIT>/<schema-source>`,
      the corresponding two blob URLs for `backstop.yml` and `backstop.lock`, and
      `https://github.com/backstop-ai/backstop-core/commit/<record.commit>`.
      The manifest declares only these derivation policies; it must not copy schema rows,
      release commits, or another realized link inventory that could drift from generated records.
      Branch names, `HEAD`, `latest`, abbreviated SHAs, relative URLs, missing or extra
      descriptors, wrong kinds/paths/commits/order/owner/output, marker-digest mismatch,
      unresolved placeholders, or removal or mutation of any source link must fail
      generation or rendered verification with `PT204_CONSUMPTION`. This is generated
      provenance owned by Seed 3; it neither replaces Seed 1 evidence records nor gives
      Seed 5 ownership of generation, links, routes, anchors, or journey semantics.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The exact CLI command job input, output, owner, marker, and command pass manifest validation.
    tests: [TestProductTruth_ManifestCLIJobPasses]
  - id: CLM-002
    requirement: REQ-001
    text: The exact artifact-schema job input set, output, owner, marker, and command pass manifest validation.
    tests: [TestProductTruth_ManifestArtifactSchemaJobPasses]
  - id: CLM-003
    requirement: REQ-001
    text: The exact declared-and-locked pack job inputs, output, owner, marker, and command pass manifest validation.
    tests: [TestProductTruth_ManifestInstalledPackJobPasses]
  - id: CLM-004
    requirement: REQ-001
    text: The exact reachable stable-SemVer tag job input, output, owner, marker, and command pass manifest validation.
    tests: [TestProductTruth_ManifestReleaseHistoryJobPasses]
  - id: CLM-005
    requirement: REQ-001
    text: A missing or fifth job, missing input, duplicate job ID, duplicate output, duplicate owner, unknown job kind, escaping path, or unregistered generated file fails with the responsible job and field.
    tests: [TestProductTruth_ManifestRejectsInvalidJobMatrix]
  - id: CLM-006
    requirement: REQ-002
    text: CLI descriptors render in canonical command-path and flag order with LF bytes and one final newline.
    tests: [TestProductTruth_RenderCLICommandCatalogDeterministically]
  - id: CLM-007
    requirement: REQ-002
    text: Artifact schemas render in canonical artifact-type and numeric-version order independent of directory iteration order.
    tests: [TestProductTruth_RenderArtifactSchemaCatalogDeterministically]
  - id: CLM-008
    requirement: REQ-002
    text: Declared and locked packs render by pack name with identity fields while install dates never enter the output.
    tests: [TestProductTruth_RenderInstalledPackCatalogDeterministically]
  - id: CLM-009
    requirement: REQ-002
    text: Reachable stable release tags render by descending SemVer with peeled SHA, UTC date, and subject; prerelease, malformed, and unreachable tags are excluded.
    tests: [TestProductTruth_RenderReleaseHistoryDeterministically]
  - id: CLM-010
    requirement: REQ-002
    text: Two write runs over unchanged inputs produce byte-identical files and a no-diff second run for all four jobs.
    tests: [TestProductTruth_WriteIsByteStableForEveryJob]
  - id: CLM-011
    requirement: REQ-002
    text: Locale, timezone, CRLF input, randomized traversal order, lexical-version ordering, and install-date changes cannot alter canonical output bytes.
    tests: [TestProductTruth_RejectsNondeterministicRenderInputs]
  - id: CLM-012
    requirement: REQ-003
    text: A missing registered output is created with the exact job, sources, owner, command, and do-not-edit marker.
    tests: [TestProductTruth_WriteCreatesMarkedRegisteredOutput]
  - id: CLM-013
    requirement: REQ-003
    text: A successful journaled write replaces the four correctly marked outputs as one coherent recoverable cohort and leaves no transaction directory.
    tests: [TestProductTruth_WriteCommitsRecoverableCohort]
  - id: CLM-014
    requirement: REQ-003
    text: Unsafe targets fail before commit; every injected rename failure rolls back, while kill or rollback failure remains recoverable and blocks check/write until idempotent recovery.
    tests: [TestProductTruth_WriteRollbackAndRecoveryMatrix]
  - id: CLM-015
    requirement: REQ-004
    text: Check mode accepts all four freshly generated checked-in outputs without writing repository bytes.
    tests: [TestProductTruth_CheckAcceptsFreshOutputsReadOnly]
  - id: CLM-016
    requirement: REQ-004
    text: A CLI descriptor source change produces only the expected CLI catalog delta.
    tests: [TestProductTruth_SourceDeltaCLICommandCatalog]
  - id: CLM-017
    requirement: REQ-004
    text: An artifact-schema source change produces only the expected artifact catalog delta.
    tests: [TestProductTruth_SourceDeltaArtifactSchemaCatalog]
  - id: CLM-018
    requirement: REQ-004
    text: A declared or locked pack identity change produces only the expected installed-pack catalog delta.
    tests: [TestProductTruth_SourceDeltaInstalledPackCatalog]
  - id: CLM-019
    requirement: REQ-004
    text: A release-tag fixture change produces only the expected release-history delta.
    tests: [TestProductTruth_SourceDeltaReleaseHistory]
  - id: CLM-020
    requirement: REQ-004
    text: Manual tampering in each of the four generated outputs is independently refused by CI-equivalent check mode.
    tests: [TestProductTruth_CheckRejectsTamperForEveryJob]
  - id: CLM-021
    requirement: REQ-004
    text: Missing output, stale output, marker drift, orphan output, unreadable source, and render failure each return nonzero with job, sources, output, first-difference state, and regeneration command, without rewriting.
    tests: [TestProductTruth_CheckFailureDiagnosticMatrix]
  - id: CLM-022
    requirement: REQ-005
    text: The CLI and artifact-schema exact include regions appear once at their subordinate reference-page anchors and reconstructed rendered records match source digests.
    tests: [TestProductTruth_SiteConsumesReferenceFragments]
  - id: CLM-023
    requirement: REQ-005
    text: The installed-pack exact include region appears once at its subordinate packs-page anchor and reconstructed rendered records match its source digest.
    tests: [TestProductTruth_SiteConsumesInstalledPackFragment]
  - id: CLM-024
    requirement: REQ-005
    text: The release-history exact include region appears once at its subordinate status-page anchor and reconstructed rendered records match its source digest.
    tests: [TestProductTruth_SiteConsumesReleaseHistoryFragment]
  - id: CLM-025
    requirement: REQ-005
    text: A missing, duplicated, moved, stale, separately rendered, or manually reconstructed generated region fails with the fragment and owner.
    tests: [TestProductTruth_SiteRejectsParallelOrInvalidConsumption]
  - id: CLM-026
    requirement: REQ-006
    kind: absence
    text: The renderer exposes no arbitrary template, plugin, remote-fetch, expression, or user-defined transformation surface.
    tests: [TestProductTruth_HasNoGeneralizedTransformationSurface]
  - id: CLM-027
    requirement: REQ-006
    kind: absence
    text: The renderer contains no documentation-semantic, visual-design, product-claim, boundary-classification, MCP, or machine-only publication implementation.
    tests: [TestProductTruth_DoesNotAbsorbOtherSeedOwnership]
  - id: CLM-028
    requirement: REQ-006
    text: An attempted fifth or arbitrary transform reports the separate-governance boundary instead of executing locally.
    tests: [TestProductTruth_RefusesUndeclaredGenericTransform]
  - id: CLM-029
    requirement: REQ-001
    text: Invoking the regeneration command without a mode performs the same writes and produces the same bytes as explicit write mode.
    tests: [TestProductTruth_DefaultModeEqualsWrite]
  - id: CLM-030
    requirement: REQ-001
    text: An unknown argument or conflicting combination of write, check, and recover modes is refused before reading an input or output.
    tests: [TestProductTruth_RejectsInvalidModeArguments]
  - id: CLM-031
    requirement: REQ-001
    text: Loose and packed, annotated and lightweight stable tags produce identical logical release records.
    tests: [TestProductTruth_GitLogicalTagStorageAndFormMatrix]
  - id: CLM-032
    requirement: REQ-001
    text: No physical Git ref layout enters generation; qualifying unpeelable or unreadable refs fail PT104 while malformed, prerelease, and unreachable refs are excluded.
    tests: [TestProductTruth_GitLogicalRefsRejectInvalidMatrix]
  - id: CLM-033
    requirement: REQ-002
    text: Each of the four jobs emits exactly its required fields, visible headers, ordering, and scalar validation.
    tests: [TestProductTruth_ExactRecordAndTableShapeMatrix]
  - id: CLM-034
    requirement: REQ-002
    text: Path-derived base and typed schema records, including backstop-yml's not-declared version sentinel, pass; path/field disagreement, invalid missing/nested scalars, and mismatched pack sets, versions, refs, or hashes fail their stable code.
    tests: [TestProductTruth_RejectsInvalidSchemaAndPackJoinMatrices]
  - id: CLM-035
    requirement: REQ-003
    text: Every special character and newline follows the ordered escaping contract and canonical provenance-envelope JSON reproduces both marker digests.
    tests: [TestProductTruth_TableEscapingAndDigestContract]
  - id: CLM-036
    requirement: REQ-003
    text: NUL, forbidden controls, map-backed records, and nested required scalar values fail before output staging.
    tests: [TestProductTruth_RejectsUnsafeScalarAndRecordShapes]
  - id: CLM-037
    requirement: REQ-003
    text: Failure or process interruption at every journal transition either restores the prior cohort or leaves durable recoverable state.
    tests: [TestProductTruth_TransactionInterruptionRecoveryMatrix]
  - id: CLM-038
    requirement: REQ-003
    text: Recovery is idempotent; recovery failure retains journal and backups and keeps check/write blocked with PT203.
    tests: [TestProductTruth_RecoveryIdempotenceAndFailure]
  - id: CLM-039
    requirement: REQ-005
    text: Exact source include syntax, raw-HTML delimiters, table attributes, headers, rows, and cells reconstruct the canonical records under the four subordinate anchors.
    tests: [TestProductTruth_ExactSourceAndRenderedRegionContract]
  - id: CLM-040
    requirement: REQ-005
    text: Delimiter, header, row, cell, visible-text, digest, owner, or locked-build-command tampering fails PT204.
    tests: [TestProductTruth_RenderedRegionTamperMatrix]
  - id: CLM-041
    requirement: REQ-007
    text: A stable tag absent from latest origin/main blocks release-history-current and therefore GoReleaser.
    tests: [TestProductTruth_ReleaseBlocksStaleLatestMain]
  - id: CLM-042
    requirement: REQ-007
    text: Regenerated latest main plus a rerun permits release while preserving the tag's original release commit.
    tests: [TestProductTruth_ReleaseHandshakePassesAfterMainRegeneration]
  - id: CLM-043
    requirement: REQ-007
    text: The tag checkout cannot substitute for latest-main history, and Pages refuses stale generated history.
    tests: [TestProductTruth_ReleaseAndPagesWorkflowWiring]
  - id: CLM-044
    requirement: REQ-008
    text: Exactly 80.00 percent measured producttruth coverage passes.
    tests: [TestProductTruth_VerifierAcceptsCoverageAtThreshold]
  - id: CLM-045
    requirement: REQ-008
    text: Coverage at 79.99, or an absent, duplicate, or nonnumeric total, fails PT205.
    tests: [TestProductTruth_VerifierRejectsCoverageFailureMatrix]
  - id: CLM-046
    requirement: REQ-008
    text: Verification runs check mode, the exact locked Jekyll build, rendered verification, and structural branch, Pages, and release workflow tests.
    tests: [TestProductTruth_VerifierCoversPipelineAndWorkflowSurfaces]
  - id: CLM-047
    requirement: REQ-009
    text: The CLI job exports exactly one site-commit-bound immutable tree link for cmd/backstop, tied to its owner, output, markers, and envelope digest.
    tests: [TestProductTruth_CLIImmutableSourceLinkPasses]
  - id: CLM-048
    requirement: REQ-009
    text: Removing the CLI source link fails PT204 for cli-command-catalog.
    tests: [TestProductTruth_CLIImmutableSourceLinkRemovalFails]
  - id: CLM-049
    requirement: REQ-009
    text: A mutable commit binding, wrong tree path, wrong owner/output, extra link, or marker/envelope digest drift fails the CLI provenance contract.
    tests: [TestProductTruth_CLIImmutableSourceLinkDriftFails]
  - id: CLM-050
    requirement: REQ-009
    text: The artifact-schema job exports exactly one site-commit-bound immutable blob link for each rendered record source in record order, tied to its owner, output, markers, and envelope digest.
    tests: [TestProductTruth_ArtifactSchemaImmutableSourceLinksPass]
  - id: CLM-051
    requirement: REQ-009
    text: Independently removing any artifact-schema source link fails PT204 for artifact-schema-catalog and names the missing record source.
    tests: [TestProductTruth_ArtifactSchemaImmutableSourceLinkRemovalFails]
  - id: CLM-052
    requirement: REQ-009
    text: A mutable commit binding, wrong/missing/extra/reordered schema path, wrong owner/output, or marker/envelope digest drift fails the artifact-schema provenance contract.
    tests: [TestProductTruth_ArtifactSchemaImmutableSourceLinkDriftFails]
  - id: CLM-053
    requirement: REQ-009
    text: The installed-pack job exports exactly the site-commit-bound immutable blob links for backstop.yml then backstop.lock, tied to its owner, output, markers, and envelope digest.
    tests: [TestProductTruth_InstalledPackImmutableSourceLinksPass]
  - id: CLM-054
    requirement: REQ-009
    text: Independently removing either installed-pack source link fails PT204 for installed-pack-catalog and names the missing path.
    tests: [TestProductTruth_InstalledPackImmutableSourceLinkRemovalFails]
  - id: CLM-055
    requirement: REQ-009
    text: A mutable commit binding, wrong/missing/extra/reordered pack source, wrong owner/output, or marker/envelope digest drift fails the installed-pack provenance contract.
    tests: [TestProductTruth_InstalledPackImmutableSourceLinkDriftFails]
  - id: CLM-056
    requirement: REQ-009
    text: The release-history job exports exactly one immutable commit link for each rendered release record in record order, tied to its owner, output, markers, and envelope digest.
    tests: [TestProductTruth_ReleaseHistoryImmutableSourceLinksPass]
  - id: CLM-057
    requirement: REQ-009
    text: Independently removing any release-record commit link fails PT204 for release-history and names the missing release record.
    tests: [TestProductTruth_ReleaseHistoryImmutableSourceLinkRemovalFails]
  - id: CLM-058
    requirement: REQ-009
    text: An abbreviated or wrong commit, mutable binding, missing/extra/reordered release link, wrong owner/output, or marker/envelope digest drift fails the release-history provenance contract.
    tests: [TestProductTruth_ReleaseHistoryImmutableSourceLinkDriftFails]
  - id: CLM-059
    requirement: REQ-009
    text: Tree/blob descriptors canonically serialize only kind, commit_binding, path in that order and commit descriptors only kind, commit_binding, commit in that order; omitting, emptying, or nulling an applicable member, adding the opposite variant member as empty or null, adding an unknown member, or mutating serializer output order fails validation or exact canonical-byte/digest comparison, while reordered decoder input reserializes to the one canonical order.
    tests: [TestProductTruth_SourceLinkCanonicalJSONAbsentEmptyNullMutationMatrix]
---

# SPEC-074: Derived Product Truth Pipeline

## Overview

The site should not ask maintainers to copy facts that the repository can state for itself. This
spec establishes a narrow derivation chain for four high-value surfaces: the shipped command tree,
the artifact schema cohort, the installed pack fleet, and release history. Their authoritative
inputs produce checked-in Markdown fragments, and those fragments are what Jekyll publishes.

The design intentionally stops short of a documentation framework. The manifest is an auditable
inventory, not a user-extensible transformation language. The renderer knows four Backstop-specific
jobs and no others. Seed 1 continues to own product meaning, routes, and surrounding prose; Seed 2
owns reusable documentation semantics; Seed 4 owns layouts and rendering; Seed 5 owns journeys.
This seed owns only the accountable arrows between source, generated Markdown, and site.

## Requirements

The frontmatter requirements and claims are normative. The four-row matrix below is the complete
generation inventory; implementations may not infer additional jobs by scanning directories.

| Job | Authoritative input | Checked-in Markdown | Seed 1 owner | Complete source-link descriptors |
|---|---|---|---|---|
| `cli-command-catalog` | JSON from `go run ./cmd/backstop commands` at the current checkout | `docs/_includes/generated/cli-command-catalog.md` | `/reference/#cli-command-catalog` | One `tree`, `site`, `cmd/backstop`. |
| `artifact-schema-catalog` | `artifacts/*/v*/schema.json` and `artifacts/base/schema.json` | `docs/_includes/generated/artifact-schema-catalog.md` | `/reference/#artifact-schema-catalog` | One `blob`, `site`, `<record.source>` per rendered schema row, in row order. |
| `installed-pack-catalog` | `backstop.yml` and `backstop.lock` | `docs/_includes/generated/installed-pack-catalog.md` | `/packs/#installed-pack-catalog` | Two `blob`, `site` descriptors: `backstop.yml`, then `backstop.lock`. |
| `release-history` | Reachable exact `vMAJOR.MINOR.PATCH` repository tags | `docs/_includes/generated/release-history.md` | `/status/#release-history` | One `commit`, `record`, `<record.commit>` per release row, in row order. |

`backstop/spec/*` and other artifact-reservation tags are not releases. Prerelease tags are omitted
because this first public status surface is explicitly the stable release history; adding prerelease
publication later is a product decision, not a permissive parser accident. The release job peels
annotated and lightweight tags to commits and excludes tags not reachable from `HEAD` so a tag on a
side branch cannot become published truth.

Every fragment's first line has this semantic shape, with values serialized canonically from its job:

```markdown
<!-- GENERATED PRODUCT TRUTH | job=<id> | inputs=<ordered locators> | owner=<route#anchor> | regenerate=./scripts/generate-product-truth.sh | DO NOT EDIT -->
```

The marker is provenance and an overwrite guard. It is not a waiver and does not make generated
bytes independently authoritative.

Each region also contains the exact sources-begin/end marker pair from REQ-009 and its closed
descriptor set. The shared digest covers one canonical envelope containing job, output, owner,
records, and descriptors. Generated source HTML-encodes the typed `<SITE-COMMIT>` deployment-binding
token as `&lt;SITE-COMMIT&gt;`; it is not a mutable URL. Seed 4 decodes and resolves that token to the full build/deployment SHA and emits
the exact immutable GitHub anchors. Release rows already carry their immutable commit and require no
site binding.

Canonical descriptor JSON has no optional-member ambiguity. Tree and blob values are concrete
`{kind,commit_binding,path}` structs in that member order; commit values are concrete
`{kind,commit_binding,commit}` structs in that member order. The opposite variant member is absent,
never `null` or an empty string. Applicable members cannot be omitted, empty, or null, and unknown
members are refused before the provenance envelope is hashed.

## Implementation

The implementation must execute these passes in order:

1. Parse the checked-in manifest, require the exact four-job closed set, resolve every declared path
   beneath the repository root, and reject duplicate job, output, or owner identities.
2. Load all inputs before writing. Invoke the checked-out command tree for its JSON descriptor;
   enumerate and parse the checked-in schema cohort; join declared packs to their lock identities;
   and peel, filter, and sort release tags. A command failure, malformed JSON/YAML/schema, missing lock
   entry, malformed hash, or unpeelable qualifying tag fails the responsible job.
3. Normalize records, order them by the rules in REQ-002, and render UTF-8 Markdown with LF endings,
   canonical tables, escaped Markdown cells, and one final newline. No wall-clock timestamp, absolute
   checkout path, install timestamp, environment-specific Go path, or map iteration order enters output.
4. Construct each job's exact ordered source-link descriptors, serialize the complete provenance
   envelope, compute its digest, and emit the ownership, region, and source markers plus raw-HTML
   table and descriptors. All four jobs must render successfully before either mode proceeds.
5. In `--check`, refuse any outstanding transaction, render into a temporary directory, compare bytes to the repository, scan
   `docs/_includes/generated/` for unregistered files, print attributable drift diagnostics, and exit
   without writing. Multiple drifts are all reported in stable job order.
6. In `--write`, validate every target, stage and fsync all files plus a journal in one transaction
   directory, back up/install each file while durably recording transitions, and roll back completed
   transitions on error. A kill or rollback failure leaves recoverable state; check/write refuse until
   idempotent `--recover` restores the prior cohort. Do not claim simultaneous cross-file visibility.
7. Verify the exact include markers and four subordinate anchors in the three Seed 1 route owners. The generator does
   not edit surrounding page prose and does not synthesize Liquid expressions.
8. After Seed 4's exact locked Jekyll build, parse each unique rendered HTML region, reconstruct its
   explicit record structs and source-link descriptors, compare the canonical envelope SHA-256 with
   both source markers, and require the exact immutable URL set for the supplied site commit.
9. On stable tags, block GoReleaser until a separate latest-`origin/main` checkout contains the new
   release row; land that row through normal main/Pages flow, then rerun the tag workflow.
10. Run the verifier's measured coverage, drift, build, rendered-region, and workflow-wiring checks.

The Go code under `scripts/producttruth` is repository tooling, not a `pkg/` API or `backstop` CLI
command. The wrapper gives maintainers and CI one stable command. If four explicit render functions
begin converging on a generic template/input engine, planning must stop and govern that dependency
separately rather than generalizing this package in place.

## Verification

Verification uses hermetic fixture repositories for source mutation, tag graphs, symlinks, unsafe
paths, partial-write failures, locale/timezone variation, and Jekyll consumption. Tests exercise each
job independently and the complete pipeline. They must compare bytes, not only parsed Markdown or
successful exit codes. Every job independently proves its complete source-link set, immutable target,
and removal/drift refusal. The real-repository check then proves the checked-in outputs match the current
authoritative inputs. `scripts/verify-product-truth.sh` enforces one numeric coverage total at or above
80.00, runs the exact locked Jekyll command and rendered verifier, and structurally proves CI, Pages,
and release workflow wiring. Claims and mandated test names are defined in frontmatter.

The rendered-site assertions depend on Seed 4's build contract. Until that implementation is present,
the Seed 3 plan may land generator and source-level include verification first, but SPEC-074 cannot be
promoted to `implemented` until the built-region assertions run against the actual `_site` output.

## Integration Contract

SPEC-072 owns the three page routes and their human-authored context. SPEC-074 introduces only the
four subordinate generated-region anchors and may generate only into those owners. It may not add a new page. The static
include expressions are consumer pointers; the four fragment files remain the only Markdown generated
from these inputs.

Seed 2's installed documentation-semantics pack evaluates the resulting page meaning through its
released interface. `scripts/producttruth` does not decide whether two passages are semantically
duplicative or whether a product claim has adequate evidence. Seed 4 must render these exact fragments
and resolve their typed `site` bindings to the full build/deployment commit; it may wrap them for
presentation, but it must not fetch the inputs again, rebuild their tables, or invent source links.
Seed 5 may traverse the built regions and consume the owner-exported job/output/owner/marker/digest/link
tuple without becoming another generator, evidence registry, route owner, or journey source of truth.

## Sharp Edges

- **A tag and its SHA-bearing row cannot inhabit one commit.** The tag identifies the immutable
  release commit; a later normal main commit records it publicly. A two-checkout release gate keeps
  GoReleaser blocked until latest main is regenerated, avoiding self-reference and stale publication.
- **A marker is not permission to clobber anything.** The job ID must match, the path must remain under
  the repository, and symlinks are refused. Otherwise an attacker or accidental rename can turn the
  writer into an arbitrary-file overwrite.
- **Four renames are not one atomic rename.** The journal, backups, rollback, loud residual state,
  and idempotent recovery make the cohort recoverable; this spec does not promise simultaneous visibility.
- **Git's physical ref layout is not an API.** Loose and packed, annotated and lightweight tags must
  behave identically through Git plumbing; reading `.git/refs` silently omits packed releases.
- **Map and filesystem ordering are nondeterministic.** YAML maps, JSON maps, glob traversal, Git ref
  iteration, locale, and timezone must be normalized explicitly. A no-diff run on one laptop is not
  proof of cross-run byte stability.
- **Pack locks contain nondeterministic installation time.** Publishing `install_date` would guarantee
  meaningless churn. The output is limited to declared version and locked identity fields.
- **A built page can look correct while bypassing the fragment.** Digest-backed region verification
  is required because a hand-copied table can visually match once and then drift independently.
- **A source path is not immutable provenance.** `cmd/backstop`, schema paths, and pack files become
  durable public targets only when Seed 4 binds `site` to the full deployment SHA. Branch, HEAD,
  latest, abbreviated-SHA, and unresolved-placeholder links are failures even if they currently open.
- **Liquid delimiters are a known collision surface.** ISSUE-182 concerns recipe substitution of
  downstream `{{ ... }}` bytes. This generator does not use recipes or generate include expressions;
  the include sites are static Seed 1 page bytes. If a plan changes that fact, it must resolve the
  literal-byte risk rather than inventing fragile escaping.
- **A useful fifth transform is not automatically local scope.** The closed job set is intentional.
  A generalized engine may become a hard dependency, but first-consumer pressure does not transfer
  ownership into this website seed.

## Review Questions

1. Can each output byte be traced to the exact input set named in its marker, with no wall clock,
   checkout path, or hidden network input?
2. Does every job have a source-mutation fixture proving the expected localized Markdown delta, and
   does every output have an independent tamper-refusal fixture?
3. Does every transaction interruption point roll back or leave an idempotently recoverable journal?
4. Do all four physical-storage/tag-form fixtures behave identically with no `.git/refs` dependency?
5. Does the installed-pack catalog join declaration and lock identity without publishing
   `install_date` or silently accepting a missing/mismatched entry?
6. Do exact source/rendered delimiters reconstruct the same records and digest through the locked build?
7. Has any general template, plugin, expression, remote-fetch, prose-semantic, or presentation system
   entered `scripts/producttruth` under the name of convenience?
8. If implementation exposed a genuinely generic missing mechanism, was it separately governed and
   consumed through an explicit dependency rather than absorbed here?
9. Does stale latest main block GoReleaser and Pages until normal regeneration lands?
10. Does measured generator coverage reach 80.00 with malformed coverage output failing closed?
11. Does each job export exactly its complete descriptor set, with owner/output/records/links covered
    by the same digest and every built link resolved to a full immutable commit?
12. Does independently removing or mutating every source link fail its owning job without Seed 5
    reconstructing provenance or Seed 1 evidence?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` v0.6.0 — source bundle, REQ-011@1.0.0,
  resolved OQ-5, DD-10, DD-11, Seed 3 acceptance, and source/generated ownership sharp edge.
- `specs/SPEC-072-public-product-model.spec.md` v1.0.2 — authoritative page owners, route/anchor
  boundary, human-readable product truth, and the Seed 2/3/4 seams.
- `cmd/backstop/root.go` — current command-tree construction and deterministic `commands` JSON surface.
- `artifacts/spec/v1/schema.json`, `artifacts/capability/v1/schema.json`, and
  `artifacts/base/schema.json` — representative members of the checked-in schema cohort.
- `backstop.yml` and `backstop.lock` — declared and content-identity-locked pack truth.
- `.github/workflows/release.yml` and `.goreleaser.yml` — current stable-tag release mechanism and
  release artifact configuration.
- `.github/workflows/ci.yml` — current blocking gate and build order into which drift refusal must fit.
- `issues/ISSUE-182-recipe-literal-placeholder-escaping.issue.md` — durable Liquid/recipe delimiter
  collision evidence; this spec avoids recipe-emitted Liquid bytes.
- `specs/SPEC-076-end-to-end-website-capabilities.spec.md` v1.0.1 — downstream generated-obligation
  consumer and the v1.0.2 predecessor-amendment matrix; it does not own these provenance links.
