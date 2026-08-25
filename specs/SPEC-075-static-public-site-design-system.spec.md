---
title: "Static Public Site Design System"
number: SPEC-075
created: "2026-08-24"
status: draft
schema_version: spec/v1
spec_version: 1.0.4

implementation:
  summary: >
    BUNDLE-032 Seed 4 only: turn SPEC-072's ten canonical Markdown owners and exact
    navigation model, SPEC-073's released-pack identity contract, and SPEC-074's
    generated Markdown regions into one statically rendered public product site. Use a
    locked Jekyll/GitHub Pages build, preserve `backstop.sh`, retain five legacy public
    URLs as serverless redirects, ship no client JavaScript, and verify canonical routes,
    anchors, owner-declared journey links, structured boundary fields, adoption instructions,
    links, navigation, current-page state, responsive behavior, custom-domain continuity,
    generated-region consumption and provenance links, and deployment wiring. The visual direction
    is a journey-oriented technical field guide: one shared shell, a question-led page
    hero, readable long-form model/reference treatment, evidence and boundary callouts,
    responsive tables and diagrams, and the canonical Backstop wordmark. Reusable visual
    and interaction policy remains owned by the released design-system pack. Core runs the
    installed pack through Backstop against the actual built site and proves clean output
    plus attributable failures for token, inline-style, focus, reduced-motion,
    accessibility, wordmark, and reusable-presentation violations. Core must not copy
    those rules into local scripts, CSS comments, tests, or conventions.
  subject: scripts/sitecheck

verification:
  level: integration
  coverage_threshold: 80
  test_command: ./scripts/verify-public-site.sh

contracts:
  - file: Gemfile
    provides:
      - name: public_site_ruby_dependencies
        kind: variable
        signature: "Pinned-source Jekyll and GitHub Pages build dependency declarations"
  - file: Gemfile.lock
    provides:
      - name: locked_public_site_ruby_runtime
        kind: variable
        signature: "Resolved immutable Ruby dependency graph used by every site build"
  - file: package.json
    provides:
      - name: public_site_browser_verification_dependencies
        kind: variable
        signature: "Private verification-only Playwright dependency and scripts; no site runtime bundle"
  - file: package-lock.json
    provides:
      - name: locked_public_site_browser_runtime
        kind: variable
        signature: "Resolved verification-only browser dependency graph"
  - file: docs/_config.yml
    provides:
      - name: public_site_jekyll_configuration
        kind: variable
        signature: "url, domain, permalink, Markdown, excludes, and serverless redirect configuration"
  - file: docs/_data/site-presentation.yml
    provides:
      - name: field_guide_instance_contract
        kind: variable
        signature: "routes[10] {page_kind, hero_question, treatments[], next_action}; dimensions {shell_max_px:1180, prose_max_px:760}"
  - file: .backstop/seed4-delivery-inventory.yml
    provides:
      - name: seed4_delivery_role_inventory
        kind: variable
        signature: "base_commit + changes[] {status, old_path?, path, role}; exact path-role matrix"
  - file: docs/_layouts/default.html
    provides:
      - name: public_site_shell
        kind: variable
        signature: "Semantic document shell consuming shared header, main content, and footer"
  - file: docs/_layouts/redirect.html
    provides:
      - name: serverless_legacy_redirect
        kind: variable
        signature: "No-JavaScript canonical link, immediate meta refresh, and human fallback link"
  - file: docs/_includes/site-header.html
    provides:
      - name: public_site_navigation
        kind: variable
        signature: "Wordmark Home plus exact primary and utility navigation with current-page state"
    consumes:
      - source: docs/_data/content-topology.yml
        name: canonical_navigation_model
        kind: variable
  - file: docs/_includes/site-footer.html
    provides:
      - name: public_site_footer
        kind: variable
        signature: "Project, repository, status, contributing, and license continuation links"
  - file: docs/_includes/page-hero.html
    provides:
      - name: question_led_page_hero
        kind: variable
        signature: "Page responsibility, boundary state, and next-action presentation primitive"
  - file: docs/assets/css/site.css
    provides:
      - name: public_site_composition
        kind: variable
        signature: "Site-specific shell and page composition consuming owner-governed tokens"
    consumes:
      - source: .backstop/packs/backstop-ai/backstop-design-system/pack.yml
        name: released_visual_policy_and_token_contract
        kind: variable
  - file: playwright.config.ts
    provides:
      - name: public_site_browser_matrix
        kind: variable
        signature: "Pinned Chromium; JavaScript-disabled 360x800, 768x1024, 1440x1000; root-font 200% relayout"
  - file: tests/public-site.spec.ts
    provides:
      - name: public_site_rendered_behavior_acceptance
        kind: function
        signature: "Playwright acceptance for responsive layout, navigation, focus, overflow, and no-JavaScript use"
  - file: scripts/sitecheck/main.go
    provides:
      - name: verify_public_site
        kind: function
        signature: "Verify(root string, builtRoot string) []Finding"
      - name: verify_installed_design_system
        kind: function
        signature: "VerifyInstalledDesignSystem(root string, builtRoot string) []Finding"
      - name: LoadOwnerAcceptanceExport
        kind: function
        signature: "LoadOwnerAcceptanceExport(root string) (OwnerExport, error)"
      - name: BuildGateCorpus
        kind: function
        signature: "BuildGateCorpus(root, builtRoot string) ([]string, error)"
      - name: VerifyEightIsolatedCorpora
        kind: function
        signature: "VerifyEightIsolatedCorpora(root, builtRoot string, export OwnerExport) []Finding"
      - name: VerifyRenderedOwnerContracts
        kind: function
        signature: "VerifyRenderedOwnerContracts(root string, builtRoot string, siteCommit string) []Finding; enforces JLINK-024 dual identity, one-anchor cardinality, containment, visible-byte preservation, and order"
    consumes:
      - source: docs/_data/content-topology.yml
        name: canonical_routes_and_navigation
        kind: variable
      - source: .backstop/website-pack-releases.yml
        name: released_design_system_evidence
        kind: variable
      - source: .backstop/packs/backstop-ai/backstop-design-system/contracts/public-site-acceptance.yml
        name: public_site_acceptance_export
        kind: variable
      - source: .backstop/packs/backstop-ai/backstop-design-system/pack.yml
        name: installed_design_system_rules
        kind: variable
  - file: scripts/render-public-site-contracts/main.go
    provides:
      - name: render_public_site_owner_contracts
        kind: function
        signature: "Render(root string, builtRoot string, siteCommit string) []Finding; JLINK-024 binds one source link to one dual-identity boundary-continuation anchor"
    consumes:
      - source: docs/_data/content-topology.yml
        name: journey_links_and_adoption_instructions
        kind: variable
      - source: docs/_data/product-model.yml
        name: structured_boundary_records
        kind: variable
      - source: docs/_data/derived-product-truth.yml
        name: generated_source_link_policies
        kind: variable
      - source: docs/_includes/generated
        name: generated_source_descriptors
        kind: variable
  - file: scripts/verify-public-site.sh
    provides:
      - name: verify_public_site_delivery
        kind: function
        signature: verify_public_site_delivery()
    consumes:
      - source: scripts/verify-documentation-semantics-integration.sh
        name: verify_documentation_semantics_integration
        kind: function
      - source: scripts/generate-product-truth.sh
        name: check_derived_product_truth
        kind: function
  - file: .github/workflows/pages.yml
    provides:
      - name: public_site_pages_delivery
        kind: variable
        signature: "Main/manual locked build, SHA-pinned official actions, deploy, and authoritative post-deploy proof"
  - file: .github/pages-actions.lock.yml
    provides:
      - name: pages_action_pins
        kind: variable
        signature: "allowlisted official action identity -> full 40-hex commit SHA"
  - file: scripts/stamp-pages-artifact.sh
    provides:
      - name: stamp_pages_artifact
        kind: function
        signature: "stamp_pages_artifact --commit <40-hex> --run-id <integer> _site"
  - file: scripts/install-design-assets.sh
    provides:
      - name: install_design_system_assets
        kind: function
        signature: "install_design_system_assets <installed-pack-root> _site"
  - file: scripts/verify-pages-deployment.sh
    provides:
      - name: verify_pages_deployment
        kind: function
        signature: "verify_pages_deployment --repository <owner/name> --run-id <id> --commit <sha> --artifact-id <id> --page-url https://backstop.sh/"
  - file: docs/CNAME
    provides:
      - name: public_site_custom_domain
        kind: constant
        signature: "backstop.sh"

requirements:
  - id: REQ-001
    supports:
      - website-expansion:REQ-009@2.0.0
    text: >
      The public site must remain a build-time-only Jekyll site deployed by GitHub Pages.
      Root `Gemfile` and committed `Gemfile.lock` must provide one locked Ruby graph, and
      every local, CI, verification, and Pages build must invoke exactly
      `bundle exec jekyll build --source docs --destination _site --trace` with
      `JEKYLL_ENV=production`. The published site must require no application server,
      database, authentication, persisted user state, transaction processor, SPA router,
      client-side framework, client-side rendering, search service, or client JavaScript.
      `package.json` and `package-lock.json` are private browser-verification dependencies
      only; no JavaScript file or script element may enter `_site`. Introducing any
      prohibited runtime concern requires a new governed requirement and platform decision;
      it is not an implementation option under this spec.
  - id: REQ-002
    supports:
      - website-expansion:REQ-009@2.0.0
    text: >
      The build must publish SPEC-072's exact source/path pairs: `docs/index.md` at `/`,
      `evaluate.md` at `/evaluate/`, `model.md` at `/model/`, `adopt.md` at `/adopt/`,
      `use-cases.md` at `/use-cases/`, `packs.md` at `/packs/`, `extend.md` at `/extend/`,
      `reference.md` at `/reference/`, `status.md` at `/status/`, and `contributing.md`
      at `/contributing/`. Every page must render the wordmark as Home; primary navigation
      exactly once in the order Evaluate, Model, Adopt, Use Cases, Packs, Extend, Reference;
      utility navigation exactly once as Status, Contributing; and exactly one current-page
      marker when its canonical destination is represented. Internal links must use root-relative
      canonical paths, resolve to one built file and, when present, one real case-sensitive ID.
      Rendered `a[href]` classification precedence is: empty fails; fragment-only and query-only
      resolve against the current canonical route and fragments name one case-sensitive ID;
      root-relative is internal and must resolve canonically; path-relative is internal but fails;
      absolute HTTP or HTTPS on normalized host `backstop.sh` is internal but fails the root-relative
      rule; cross-origin HTTPS and valid `mailto` are allowed; cross-origin HTTP, protocol-relative,
      localhost/loopback, filesystem, unknown-scheme, missing, ambiguous, alias-targeted, or
      case-mismatched targets fail. Canonical metadata is separate and must equal
      `https://backstop.sh<CANONICAL_PATH>`; query/fragment never changes ownership. Legacy `/getting-started.html`, `/concepts.html`,
      `/artifact-workflow.html`, `/pack-authoring.html`, and `/cli-reference.html` must remain
      serverless aliases to `/adopt/`, `/model/`, `/model/`, `/extend/`, and `/reference/`
      respectively. After Seed 1 migrates their substantive content, the five existing legacy
      Markdown files must become frontmatter-only users of `docs/_layouts/redirect.html`; that
      layout emits a canonical link, an immediate meta refresh, and a visible ordinary fallback
      link, with no script. The prior `docs/index.html` must be removed after its useful units move
      to `docs/index.md`; retaining both sources or any two sources that emit `/index.html` is a
      build refusal. Canonical content and all internal navigation must never target an alias.
  - id: REQ-003
    supports:
      - website-expansion:REQ-009@2.0.0
      - website-expansion:REQ-013@1.0.0
    text: >
      Browser acceptance uses only the Chromium revision pinned by `package-lock.json`; cross-browser
      compatibility is not a BUNDLE-032 requirement, and this is a deterministic layout/no-JavaScript
      acceptance choice rather than a broader compatibility guarantee. For every canonical route,
      fresh JavaScript-disabled contexts at 360x800, 768x1024, and 1440x1000 await font readiness
      and two animation frames. At every viewport the wordmark, all seven primary
      links, both utility links, current-page state, main landmark, page heading, and footer
      must be visible and keyboard reachable in document order without a pointer-only control,
      hidden menu, focus trap, horizontal page overflow, clipped focus indicator, or overlap. Tab
      traversal from body must equal visible DOM-ordered links, summary controls, and declared
      focusable overflow regions, and return to the
      first within `expected_count+1`; each focused rect stays within viewport or its declared local
      scroller, remains topmost at center, and does not intersect another required control. Focus-
      treatment validity itself is decided only by the installed focus rule and its owner export;
      Core asserts reachability, clipping, occlusion, and attribution, not an outline mechanism.
      Document scroll width may exceed client width
      by at most 1 CSS px.
      The 360px shell stacks a two-column navigation beneath the wordmark; 768px uses a
      wrapped navigation row; 1440px uses the full horizontal shell. Tables and Mermaid-derived
      diagrams may scroll inside labeled focusable regions, but the document viewport may not
      scroll horizontally. Only `[data-overflow-region][role=region][aria-labelledby][tabindex=0]`
      around a table or Mermaid SVG may overflow; ArrowRight must advance it and End expose final
      content. Actual 200% text relayout injects test-only author CSS
      `html { font-size: 200% !important; }`, proves computed root font size is exactly twice baseline
      within 0.01px after fonts/two frames, and reruns every visibility, tab, bounds, overlap, and
      overflow assertion. Screenshot scaling, transform/zoom, and device-pixel scaling do not count.
      Native links and `details`/`summary` are the only interactive primitives in static scope.
  - id: REQ-004
    supports:
      - website-expansion:REQ-009@2.0.0
      - website-expansion:REQ-013@1.0.0
    text: >
      Seed 4 must consume, without redefining, SPEC-072's ten page owners, route and navigation
      model, canonical concept relationships, evidence links, boundary states, final copy, and
      authoritative Mermaid text; SPEC-073's released/pinned dependency evidence and semantic
      gate; and SPEC-074's four exact generated include regions. Each generated region must remain
      on its declared owner route and anchor, appear exactly once in source and rendered output,
      preserve its digest and semantic table structure, and receive presentation only from the
      shared shell. Rendered verification must parse each unique region with Go's HTML parser,
      require one table with the exact job attribute, headers, rows, cells, and no unknown row
      elements, HTML-decode cells and convert `<br>` to LF, reconstruct SPEC-074's explicit record
      structs and source descriptors, recompute its canonical provenance-envelope digest, and compare
      that digest with both source markers. A missing, duplicated, moved, stale, independently
      reconstructed, or tampered region must fail with its job and owner route/anchor. Jekyll/Liquid
      must not reread authoritative product inputs, rebuild generated
      records, duplicate a canonical concept definition, infer evidence or boundary meaning,
      create a machine-only publication, or move a generated fragment to a second page. Seed 4
      may add presentation wrappers and canonical-path aliases only; it may not change ownership.
  - id: REQ-005
    supports:
      - website-expansion:REQ-013@1.0.0
    text: >
      `docs/_data/site-presentation.yml` must enumerate exactly ten SPEC-072 routes with `page_kind`,
      one exact hero question, ordered treatments, and one canonical `next_action`, matching every
      literal cell in the Exact presentation matrix below; every hero value must byte-match the
      SPEC-072 v1.0.5 `hero_question` for that route and is not Seed 4 copy ownership. `page_kind` is one of the ten matrix values;
      treatments are drawn only from `evidence-cards`, `boundary-callouts`, `generated-regions`, and
      `local-overflow`, without duplicates and in the matrix order. Every page renders
      exactly one `body[data-site-shell=field-guide-v1][data-page-kind]`, header/site nav pair,
      `main#main[data-page-route]`, `section[data-page-hero]` with one h1 and
      `[data-page-question]`, `nav[data-next-action]`, and footer. Evidence records render as
      `article[data-evidence-card][data-claim-id]`; boundary records as
      `aside[data-boundary-callout][data-boundary-state]` with one allowed five-state value;
      generated jobs as `section[data-generated-region][data-product-truth-job]`; and every table or
      Mermaid SVG inside the one labeled local-overflow region in REQ-003. At 1440px computed shell
      width is at most 1180px and ordinary prose at most 760px; at 360/768 both fit the viewport.
      Core owns these IDs, cardinalities, registry-to-treatment mapping, and dimensions. The owner
      pack alone owns colors, tokens, typography policy, focus/motion/accessibility/wordmark rules,
      and reusable-presentation validity. The Backstop wordmark is the only mark and links `/`.
      `.backstop/seed4-delivery-inventory.yml` must pin `base_commit` to the full 40-hex commit
      immediately before the first Seed 4 implementation commit and classify the exact output of
      `git diff --name-status --find-renames=100% <base_commit>...HEAD`: `A`, `M`, `D`, or `R` with
      old/new paths for renames. Every changed/deleted/renamed path appears once. Allowed roles are `build-dependency`, `site-data`,
      `site-config`, `delivery-inventory`, `layout`, `include`, `page-wrapper`,
      `stylesheet-composition`, `browser-verification`, `structural-verifier`,
      `rendered-contract-stamper`, `verification-entrypoint`, `owner-asset-installer`, `workflow`, `action-lock`, `deploy-stamp`, `deploy-verifier`, `test`,
      or deletion-only `retired-bootstrap`;
      unlisted paths, unknown/duplicate roles, and roles `visual-rule`, `engine`, `fixture-corpus`,
      `token-declaration`, or `design-policy-validator` fail. Allowed paths must also be declared by
      this spec or be the governed pages/wrappers they name. Exact mappings are: Gem/package files ->
      build-dependency; `.backstop/seed4-delivery-inventory.yml` -> delivery-inventory;
      `docs/_config.yml` and `docs/CNAME` -> site-config; `_data/site-presentation.yml` -> site-data; `_layouts/**` -> layout;
      `_includes/**` -> include; ten canonical/five alias sources -> page-wrapper; `site.css` ->
      stylesheet-composition; Playwright config/tests -> browser-verification; `scripts/sitecheck/**`
      -> structural-verifier or test; `scripts/render-public-site-contracts/**` ->
      rendered-contract-stamper; `scripts/verify-public-site.sh` -> verification-entrypoint;
      `install-design-assets.sh` -> owner-asset-installer; Pages workflow/action lock/stamp/deploy verifier -> their named
      roles. Deletions of `docs/index.html`, `docs/assets/css/backstop.css`, and
      `docs/assets/css/backstop-tokens.css` are the only
      `retired-bootstrap` entries and that role is invalid for A/M/R. `sitecheck` may parse data, hashes, paths, DOM, workflow,
      owner-export fingerprints, and finding attribution; it may not implement a visual matcher or
      invoke an engine directly. Cayman/current landing sources cannot satisfy delivery by retention.
  - id: REQ-006
    supports:
      - website-expansion:REQ-013@1.0.0
    text: >
      SPEC-073 proves only design-system release/declaration/lock/install identity. Seed 4 additionally
      requires the same-release, hash-bound owner export
      `backstop-design-system/public-site-acceptance/v1` with exactly seven cells: token,
      inline-style, focus, reduced-motion, accessibility, wordmark, reusable-presentation. Each cell
      binds exactly one installed `rule_id`, production path filters, clean and negative owner fixtures,
      deterministic mutation `{target_relative_path,unique_before_base64,replacement_base64}`, and
      path-fidelity evidence `{fixture_relative_path,target_relative_path,dispatch_evidence_ref}`.
      Paths must match filters, before bytes occur once, and owner evidence prove both fixtures
      dispatched to that rule at the released commit. Missing/extra/duplicate cells, rule/filter
      mismatch, unfingerprinted or mixed-release export, incomplete mutation, or missing ISSUE-184
      fidelity proof fails before Core execution.

      The export also binds one distributable `token_asset` with installed relative path, media type
      `text/css`, SHA-256, and public output `assets/css/design-system-tokens.css`. After the exact
      Jekyll build, `scripts/install-design-assets.sh` copies those owner bytes unchanged from the
      installed pack into `_site` and verifies the hash; the default layout links that asset before
      `site.css`, which may consume its custom properties but never declare them. The asset is build
      output, never checked-in Core source. Missing, altered, multiply emitted, or unreferenced owner
      token bytes fail before the built-site gate.

      Create exactly eight isolated complete project roots from one clean checkout: one clean and one
      per cell. In each, clean-install the pack, build to `_site`, enumerate every regular
      `_site` HTML/CSS/SVG path in raw-byte-sorted repository-relative order, and run exactly one
      complete `./bin/backstop gate --file <PATH>...` from that root. Every target retains exact
      `_site/<original-relative-path>` identity. Flattened/basename fixtures, source paths, subsets,
      reused roots, multiple calls, `--all`, diff scope, or direct engines are prohibited. Clean has
      zero blocking design findings; each negative applies only exported replacement bytes and must
      block on its intended rule and exact target path. Other-rule-only or unattributed failure fails.

      `protected_file_fingerprints` binds SHA-256 for every owner rule/engine/fixture/export file.
      Core rejects those exact bytes outside the installed tree, except the exact fingerprinted
      distributable token asset at its one declared `_site` output, and mechanically rejects delivered
      paths whose role in `.backstop/seed4-delivery-inventory.yml` is prohibited, absent, unknown, or
      inconsistent with its declared contract/path class. It does not scan visual vocabulary/selectors or claim to detect modified
      conceptual copies; those remain human review violations. Owner fixtures, cached results, local
      scans, screenshots, or reused roots cannot substitute for an actual-site cell.
  - id: REQ-007
    supports:
      - website-expansion:REQ-009@2.0.0
    text: >
      `docs/CNAME` must contain exactly `backstop.sh` plus one LF and the locked Jekyll build
      must copy that exact file to `_site/CNAME`. `.github/workflows/pages.yml` must run only for
      pushes to `main` and explicit manual dispatch, never tag push; use least-privilege read
      permissions before a separate deploy job receives `pages: write` and `id-token: write`;
      serialize the `pages` concurrency group without canceling an in-progress production deploy;
      check out full history, install locked Ruby and Node dependencies, clean-install packs from
      remote sources, run SPEC-073 integration, SPEC-074 check mode, the exact production build,
      the REQ-009 owner-contract annotation pass using the workflow head SHA, this spec's
      structural/browser/installed-pack verification, then upload exactly `_site` and
      deploy it with only `actions/checkout`, `ruby/setup-ruby`, `actions/setup-node`,
      `actions/configure-pages`, `actions/upload-pages-artifact`, and `actions/deploy-pages`, each
      referenced by a full 40-hex SHA matching `.github/pages-actions.lock.yml`.
      `configure-pages` selects workflow build mode. Before upload, stamp
      every canonical HTML head with exact
      `<meta name="backstop-deployment" content="commit=<SHA>;run=<ID>">`, then compute
      `tree_content_sha256` over sorted artifact paths/bytes excluding the standalone marker and write
      `_site/.well-known/backstop-deployment.json` with schema, commit, run ID, and that digest.
      `actions/upload-pages-artifact` must use path `_site` and `include-hidden-files: true`; omitting
      or false hidden-file input is prohibited. Carry its artifact ID,
      then deploy and carry page URL. No deploy may occur when dependency,
      semantics, generated truth, build, route/link, browser, design-system, CNAME, or artifact
      upload verification fails, and no alternate deployment workflow or unverified prebuilt tree
      may publish `backstop.sh`. A required post-deploy job must combine Pages/Actions/deployment API
      state, action outputs, and HTTPS proof: build type is workflow, CNAME is `backstop.sh`, HTTPS is
      enforced, current run/head SHA and artifact ID/name/`archive_digest` agree with the Actions
      artifact and github-pages deployment/environment. The retained or API-downloaded artifact must
      independently recompute `tree_content_sha256` excluding the marker and match
      `https://backstop.sh/.well-known/backstop-deployment.json`; archive and tree digests are distinct
      and never compared directly. With redirects disabled, all ten canonical HTTPS routes must contain
      the exact commit/run meta marker matching the standalone marker; five aliases redirect once;
      downgrade, certificate/host drift, stale identity, 4xx/5xx, wrong content, or partial API-only,
      action-only, or smoke-only proof fails. Post-deploy proof reports bad publication but cannot
      make the preceding deployment transaction atomic.
  - id: REQ-008
    supports:
      - website-expansion:REQ-009@2.0.0
      - website-expansion:REQ-013@1.0.0
    text: >
      `./scripts/verify-public-site.sh` must create disposable build and browser state, run
      `go test ./scripts/sitecheck/... -race -covermode=atomic`, extract exactly one numeric total
      from `go tool cover -func`, and fail below 80.00 or for an absent, duplicate, or nonnumeric
      total. It must then run SPEC-073 integration, SPEC-074 check mode, the exact locked production
      Jekyll build, the REQ-009 owner-contract annotation pass using the tested full commit,
      deterministic source/rendered route-link-nav-generated-region-CNAME checks,
      Playwright's three JavaScript-disabled viewports and 200% reflow checks, the installed
      design-system positive/negative matrix, and structural Pages workflow tests. It must delete
      temporary build, mutation, coverage, browser, and installed-pack state on success and failure,
      never modify checked-in sources, and emit stable diagnostics naming the phase, responsible
      route/path/anchor/rule, and expected versus observed value.
  - id: REQ-009
    supports:
      - website-expansion:REQ-009@2.0.0
      - website-expansion:REQ-013@1.0.0
    text: >
      Seed 4 must consume the accepted SPEC-072 v1.0.5 and SPEC-074 v1.0.4 owner records and,
      immediately after every exact REQ-001 Jekyll build, run one deterministic build-time
      annotation pass over the disposable `_site`; that pass may add only the rendered attributes,
      provenance anchors, and resolved immutable URLs defined here and must never edit checked-in
      source, author a record, infer meaning from prose, or change visible owner copy. For each of
      the exact 24 rows in the Rendered journey-link matrix, the pass must bind the one source
      `<!-- backstop-journey-link: JLINK-NNN -->` marker and its immediately following owner
      Markdown link to exactly one rendered
      `a[data-journey-link-id="JLINK-NNN"]` beneath `main#main` or
      `nav[data-next-action]`, under the exact source route/anchor, with a root-relative `href`
      equal to the exact destination route plus `#` plus destination anchor. No `JLINK-*` attribute
      may exist without its owner record, and a marker, link, attribute, source/destination anchor,
      route, label, cardinality, order, or containment mismatch must fail closed.

      Every accepted boundary record must render as exactly one
      `aside[data-boundary-callout][data-boundary-id][data-boundary-state]` on its owner
      route/anchor, with state equal to the owner record and exactly one nonempty descendant
      `[data-boundary-explanation]` rendered only from `explanation_markdown`. An
      `adjacent-guidance` record must additionally render exactly one
      `a[data-boundary-continuation][data-journey-link-id]` whose ID, label, and root-relative
      route/anchor equal the structured continuation and exactly one nonempty
      `[data-boundary-guarantee-denial]` from `guarantee_denial_markdown`; the other four states
      must render neither field because SPEC-072 requires both owner fields to be null. The pass
      must not derive an explanation, continuation, denial, or state from statement prose.
      For BOUNDARY-005 specifically, the source-only JLINK-024 marker embedded inside CLAIM-005 and
      its immediately following continuation Markdown link must produce exactly one rendered anchor
      inside the BOUNDARY-005 adjacent-guidance callout. That one anchor must carry both
      `data-journey-link-id="JLINK-024"` and `data-boundary-continuation`; it simultaneously satisfies
      the journey-link and boundary-continuation cardinalities. The annotation pass must preserve the
      pre-annotation rendered text-node bytes for the explanation, continuation label, and guarantee
      denial and retain their exact explanation-then-anchor-then-denial DOM order. The source marker
      renders no visible node. Splitting the two identities across anchors, emitting a second anchor,
      omitting either identity, moving the anchor outside the callout, or placing it before the
      explanation or after the denial is prohibited.

      Each SPEC-074 generated job must retain its one
      `section[data-generated-region][data-product-truth-job]`, product-truth marker pair, owner
      route/anchor, source-descriptor marker pair, descriptor order, and shared lowercase SHA-256
      digest. The pass must replace each `site` descriptor's literal `<SITE-COMMIT>` with the exact
      full lowercase 40-hex build/deployment commit and render every descriptor exactly once as
      `a[data-generated-source-link]` inside that job section, with final URL, kind, path or record
      commit, order, and cardinality equal to SPEC-074's closed policy; it must not use a branch,
      tag, `HEAD`, abbreviated SHA, or a second provenance inventory. Each of SPEC-072's three
      ordered adoption records must render exactly once on its owner route/anchor as one
      `pre[data-adoption-instruction-id="ADOPT-*"][data-command-sha256="sha256:<64-lowercase-hex>"]`
      containing one code block whose decoded text byte-matches `command_text`; the attribute must
      equal the owner digest. The pass renders no executable, argv, environment, provenance, or
      postcondition fields because those remain structured consumer data. Missing, extra,
      duplicated, wrong-owner, stale, unresolved, mutable, digest-mismatched, text-inferred, or
      independently reconstructed owner-contract output must fail both rendering and sitecheck
      with the responsible route, anchor, and owner ID or job.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The exact locked production Jekyll command builds the complete site without a runtime service.
    tests: [TestSiteCheck_StaticLockedJekyllBuildPasses]
  - id: CLM-002
    requirement: REQ-001
    text: Any application server, database, auth, state, transaction, SPA, client-render, search-service, or client-JavaScript surface fails with its path and prohibited class.
    tests: [TestSiteCheck_RejectsUnearnedRuntimeMatrix]
  - id: CLM-003
    requirement: REQ-001
    kind: absence
    text: Verification-only Node dependencies produce no script element, JavaScript asset, or browser runtime dependency in the built site.
    tests: [TestSiteCheck_RejectsPublishedVerificationRuntime]
  - id: CLM-004
    requirement: REQ-002
    text: All ten exact source/path pairs build once at their canonical routes.
    tests: [TestSiteCheck_CanonicalRouteMatrixPasses]
  - id: CLM-005
    requirement: REQ-002
    text: A missing, duplicate, wrong, or case-mismatched canonical route fails with source and route.
    tests: [TestSiteCheck_CanonicalRouteMatrixRejectsInvalidCell]
  - id: CLM-006
    requirement: REQ-002
    text: Every canonical page renders the exact Home, ordered primary, and ordered utility navigation with correct current state.
    tests: [TestSiteCheck_NavigationMatrixPasses]
  - id: CLM-007
    requirement: REQ-002
    text: Missing, duplicated, reordered, alias-targeted, or incorrectly current navigation fails with page and item.
    tests: [TestSiteCheck_NavigationMatrixRejectsInvalidCell]
  - id: CLM-008
    requirement: REQ-002
    text: Fragment/query-only and root-relative links resolve canonically, while cross-origin HTTPS and valid mailto pass after ordered classification.
    tests: [TestSiteCheck_LinkPrecedenceAcceptsFragmentQueryRootRelativeCrossOriginHTTPSAndMailto]
  - id: CLM-009
    requirement: REQ-002
    text: Path-relative, same-origin absolute HTTP/S, cross-origin HTTP, protocol-relative, loopback, filesystem, unknown, alias, missing, ambiguous, or case-mismatched targets fail.
    tests: [TestSiteCheck_LinkPrecedenceRejectsRelativeSameOriginAbsoluteAndForbiddenSchemes]
  - id: CLM-010
    requirement: REQ-002
    text: Each of the five legacy URLs redirects serverlessly to its exact canonical replacement.
    tests: [TestSiteCheck_LegacyRedirectMatrixPasses]
  - id: CLM-011
    requirement: REQ-002
    text: Missing, chained, cyclic, client-scripted, or internally linked legacy redirects fail.
    tests: [TestSiteCheck_LegacyRedirectMatrixRejectsInvalidCell]
  - id: CLM-012
    requirement: REQ-003
    text: Pinned Chromium proves the fonts-ready/two-frame JavaScript-off route-by-viewport tab, focus, bounds, topmost, nonintersection, and document-overflow matrix.
    tests: [TestSiteCheck_ChromiumNoJSExactInteractionMatrixPasses]
  - id: CLM-013
    requirement: REQ-003
    text: A hidden or pointer-only target, tab mismatch/trap, invalid focused-element bounds/topmost state, clipping, overlap, or document overflow fails with route, viewport, and selector.
    tests: [TestSiteCheck_ChromiumNoJSInteractionMatrixRejectsInvalidCell]
  - id: CLM-014
    requirement: REQ-003
    text: Only labeled focusable table/Mermaid local scrollers overflow, and ArrowRight plus End expose their content at all shell modes.
    tests: [TestSiteCheck_LocalOverflowAndNavigationModesPass]
  - id: CLM-015
    requirement: REQ-003
    text: Test-only root-font injection proves computed 2x text size and the complete interaction/reflow matrix; screenshot, transform, zoom, or DPR substitutes fail.
    tests: [TestSiteCheck_ActualRootFontRelayoutPasses]
  - id: CLM-016
    requirement: REQ-004
    text: The build consumes all Seed 1 owners and all four Seed 3 regions exactly once without changing ownership; parsed rendered records and descriptors reconstruct each canonical envelope and match both source-marker digests.
    tests: [TestSiteCheck_UpstreamOwnershipAndGeneratedRegionsPass]
  - id: CLM-017
    requirement: REQ-004
    text: A duplicated concept, inferred boundary/evidence relation, missing/moved/duplicated/stale/tampered or independently reconstructed generated region, table-shape or digest drift, or second input reader fails with the job and owner seam.
    tests: [TestSiteCheck_RejectsUpstreamOwnershipViolationMatrix]
  - id: CLM-018
    requirement: REQ-004
    kind: absence
    text: No Jekyll plugin, Liquid template, or layout reconstructs product-truth records or creates a parallel publication.
    tests: [TestSiteCheck_RejectsParallelTruthOrPublication]
  - id: CLM-019
    requirement: REQ-005
    text: All ten presentation records render exact shell markers/cardinalities, registry-backed evidence/boundary/generated/overflow treatments, and 1180/760 width bounds.
    tests: [TestSiteCheck_FieldGuideDOMAndTreatmentMatrixPasses]
  - id: CLM-020
    requirement: REQ-005
    text: Missing/extra route records, wrong page kind/cardinality/treatment/state/job/wrapper/dimension, stock Cayman, or preserved landing hierarchy fails structurally.
    tests: [TestSiteCheck_FieldGuideDOMAndTreatmentMatrixRejectsInvalidCell]
  - id: CLM-021
    requirement: REQ-006
    text: Owner fingerprints and mechanical delivery-role checks reject exact protected bytes and declared local rule/engine/fixture/token/policy-validator surfaces without a local visual vocabulary scan.
    tests: [TestSiteCheck_RejectsMechanicalLocalPolicySurfacesAndProtectedBytes]
  - id: CLM-022
    requirement: REQ-006
    text: The seven-cell owner applicability export fingerprint binds to the same SPEC-073-proven identity, release, manifest, commit, and installed hash.
    tests: [TestSiteCheck_OwnerAcceptanceExportBindingPasses]
  - id: CLM-023
    requirement: REQ-006
    text: Missing/extra/duplicate cells, rule/filter disagreement, incomplete fixture/mutation/fidelity evidence, or stale/mixed export fails before execution.
    tests: [TestSiteCheck_OwnerAcceptanceExportRejectsSchemaAndFidelityMatrix]
  - id: CLM-024
    requirement: REQ-006
    text: Eight fresh roots each install and build independently and execute one full sorted path-preserving gate; the clean corpus has zero blocking design findings.
    tests: [TestSiteCheck_EightIsolatedProjectRootsAndCleanCorpusPass]
  - id: CLM-025
    requirement: REQ-006
    text: The exported raw-value mutation blocks with its intended installed token rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsTokenMutation]
  - id: CLM-026
    requirement: REQ-006
    text: The exported inline-style mutation blocks with its intended installed styling rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsInlineStyleMutation]
  - id: CLM-027
    requirement: REQ-006
    text: The exported focus mutation blocks with its intended installed focus rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsFocusMutation]
  - id: CLM-028
    requirement: REQ-006
    text: The exported motion mutation blocks with its intended installed motion rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsMotionMutation]
  - id: CLM-029
    requirement: REQ-006
    text: The exported accessibility mutation blocks with its intended installed accessibility rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsAccessibilityMutation]
  - id: CLM-030
    requirement: REQ-006
    text: The exported wordmark mutation blocks with its intended installed wordmark rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsWordmarkMutation]
  - id: CLM-031
    requirement: REQ-006
    text: The exported reusable-presentation mutation blocks with its intended installed rule and exact `_site` target path.
    tests: [TestSiteCheck_InstalledDesignSystemRejectsReusablePresentationMutation]
  - id: CLM-032
    requirement: REQ-006
    kind: absence
    text: Flattened/basename/source/subset/reused/multiple-call/--all/diff/direct-engine corpora, cached results, wrong-rule-only, owner fixtures, scans, screenshots, or unattributed failures cannot satisfy a cell.
    tests: [TestSiteCheck_RejectsCorpusExecutionAndProofSubstitutes]
  - id: CLM-033
    requirement: REQ-007
    text: Source and built CNAME bytes are exactly `backstop.sh` plus LF.
    tests: [TestSiteCheck_CustomDomainBytesPass]
  - id: CLM-034
    requirement: REQ-007
    text: Missing, changed, multiply named, or non-LF CNAME state fails before upload.
    tests: [TestSiteCheck_CustomDomainRejectsInvalidMatrix]
  - id: CLM-035
    requirement: REQ-007
    text: Pages main/manual workflow uses lock-matched full-SHA official actions, workflow build mode, exact stamped root, include-hidden-files true, artifact ID, deploy output, least privilege, and serialized concurrency.
    tests: [TestSiteCheck_PagesWorkflowPinnedContractPasses]
  - id: CLM-036
    requirement: REQ-007
    text: Mutable/unallowlisted refs, pin mismatch, tag trigger, widened permission, skipped prerequisite, missing stamp, omitted/false hidden-file upload, wrong root/ID, alternate publish, or deploy-on-failure fails structurally.
    tests: [TestSiteCheck_PagesWorkflowRejectsWorkflowAndActionPinMatrix]
  - id: CLM-037
    requirement: REQ-008
    text: The verification entrypoint runs measured sitecheck tests and every required upstream, build, structural, browser, pack, and workflow phase.
    tests: [TestSiteCheck_VerificationPipelinePasses]
  - id: CLM-038
    requirement: REQ-008
    text: Coverage at 79.99, or an absent, duplicate, or nonnumeric total, fails with the coverage phase.
    tests: [TestSiteCheck_VerificationRejectsCoverageFailureMatrix]
  - id: CLM-039
    requirement: REQ-008
    text: Every injected phase failure is nonzero and names phase plus responsible route, path, anchor, rule, or expected/observed value.
    tests: [TestSiteCheck_VerificationDiagnosticsPass]
  - id: CLM-040
    requirement: REQ-008
    text: Success and failure remove all temporary build, mutation, coverage, browser, and install state without touching checked-in sources.
    tests: [TestSiteCheck_VerificationCleanupPasses]
  - id: CLM-041
    requirement: REQ-002
    text: The migrated docs/index.md is the only source that emits the root index; retaining docs/index.html or another colliding source fails with both paths.
    tests: [TestSiteCheck_RejectsRootOutputCollision]
  - id: CLM-042
    requirement: REQ-007
    text: Authoritative APIs jointly prove workflow build mode, backstop.sh CNAME, HTTPS enforcement, current run/head SHA, artifact ID/name/archive digest, and github-pages deployment identity.
    tests: [TestPagesDeployment_AuthoritativeAPIIdentityPasses]
  - id: CLM-043
    requirement: REQ-007
    text: Downloaded or retained artifact tree digest, HTTPS standalone marker, route commit/run metadata, ten canonical routes, and five one-hop aliases agree without conflating archive and tree digests.
    tests: [TestPagesDeployment_HTTPSMarkerAndRouteMatrixPasses]
  - id: CLM-044
    requirement: REQ-007
    text: API-only, action-only, smoke-only, redirect-following, HTTP downgrade, certificate/host drift, stale marker, or route/alias/content error fails post-deploy proof.
    tests: [TestPagesDeployment_RejectsPartialOrStaleProofMatrix]
  - id: CLM-045
    requirement: REQ-002
    text: Exact same-origin absolute HTTPS is required for canonical metadata but rejected in rendered anchors under link precedence.
    tests: [TestSiteCheck_DistinguishesCanonicalMetadataFromAnchorLinks]
  - id: CLM-046
    requirement: REQ-006
    text: The fingerprinted owner token CSS is copied byte-exactly only into its declared built output, linked before site.css, and consumed without checked-in Core token declarations.
    tests: [TestSiteCheck_OwnerTokenAssetConsumptionPasses]
  - id: CLM-047
    requirement: REQ-006
    text: Missing, altered, duplicated, checked-in, unlinked, or wrong-order owner token assets fail with expected and observed fingerprint/path.
    tests: [TestSiteCheck_OwnerTokenAssetConsumptionRejectsInvalidMatrix]
  - id: CLM-048
    requirement: REQ-005
    text: The pinned base commit and exact A/M/D/R diff classify every Seed 4 path once under the closed path-role matrix, including the three required bootstrap deletions.
    tests: [TestSiteCheck_Seed4DeliveryInventoryPasses]
  - id: CLM-049
    requirement: REQ-005
    text: Wrong base, unlisted/duplicate/renamed paths, invalid change kind, role/path mismatch, or non-deletion retired-bootstrap entry fails with the path and role.
    tests: [TestSiteCheck_Seed4DeliveryInventoryRejectsInvalidMatrix]
  - id: CLM-050
    requirement: REQ-009
    text: All 24 exact owner-declared JLINK records render once at their source route/anchor as owner-copy links with the exact data-journey-link-id and root-relative destination route/anchor.
    tests: [TestSiteCheck_RenderedJourneyLinkMatrixPasses]
  - id: CLM-051
    requirement: REQ-009
    text: A missing, extra, duplicated, reordered, wrong-label, wrong-owner, wrong-anchor, wrong-destination, global-navigation-only, or unregistered rendered JLINK fails with its ID and source route/anchor.
    tests: [TestSiteCheck_RenderedJourneyLinkMatrixRejectsInvalidCell]
  - id: CLM-052
    requirement: REQ-009
    text: Every boundary callout renders the exact owner ID and state plus one structured explanation marker, and adjacent-guidance alone renders its exact continuation JLINK and guarantee-denial markers.
    tests: [TestSiteCheck_StructuredBoundaryRenderingPasses]
  - id: CLM-053
    requirement: REQ-009
    text: A missing, extra, duplicate, prose-inferred, wrong-owner, wrong-state, empty explanation, invalid continuation, or invalid guarantee-denial boundary field fails with boundary ID and owner route/anchor.
    tests: [TestSiteCheck_StructuredBoundaryRenderingRejectsInvalidMatrix]
  - id: CLM-054
    requirement: REQ-009
    text: Every generated job renders its complete ordered source-descriptor set exactly once as immutable data-generated-source-link anchors inside its one region; site-bound tree/blob descriptors use the exact full site commit, while every release-history descriptor uses its generated record's immutable full commit.
    tests: [TestSiteCheck_GeneratedSourceLinkRenderingPasses]
  - id: CLM-055
    requirement: REQ-009
    text: A missing, extra, reordered, wrong-kind/path/commit/owner, out-of-region, unresolved, branch-, tag-, HEAD-, or abbreviated-SHA generated source link fails with job and owner route/anchor.
    tests: [TestSiteCheck_GeneratedSourceLinkRenderingRejectsInvalidMatrix]
  - id: CLM-056
    requirement: REQ-009
    text: ADOPT-INSTALL, ADOPT-CONFIGURE, and ADOPT-ENFORCE render once in owner order with exact instruction IDs, command bytes, and owner SHA-256 digests.
    tests: [TestSiteCheck_AdoptionInstructionRenderingPasses]
  - id: CLM-057
    requirement: REQ-009
    text: A missing, extra, duplicated, reordered, wrong-owner, altered-command, or digest-mismatched adoption instruction fails with instruction ID and owner route/anchor.
    tests: [TestSiteCheck_AdoptionInstructionRenderingRejectsInvalidMatrix]
  - id: CLM-058
    requirement: REQ-009
    kind: absence
    text: The deterministic annotation pass cannot edit checked-in sources, invent owner records or copy, infer fields from prose, emit structured execution internals, or create a second provenance inventory.
    tests: [TestSiteCheck_RenderedOwnerContractsRejectAuthoritySubstitutes]
  - id: CLM-059
    requirement: REQ-009
    text: The embedded JLINK-024 source marker emits no visible node, and its following continuation link renders as one anchor inside BOUNDARY-005 carrying both data-journey-link-id="JLINK-024" and data-boundary-continuation, between the exact preserved explanation and guarantee-denial text-node bytes.
    tests: [TestSiteCheck_EmbeddedJLINK024DualIdentityPasses]
  - id: CLM-060
    requirement: REQ-009
    text: Missing either JLINK-024 identity, splitting identities across anchors, duplicating the anchor, changing visible explanation/link/denial bytes, moving it outside BOUNDARY-005, or placing it before the explanation or after the denial fails with JLINK-024 and BOUNDARY-005.
    tests: [TestSiteCheck_EmbeddedJLINK024RejectsIdentityCardinalityMatrix, TestSiteCheck_EmbeddedJLINK024RejectsContainmentOrderAndVisibleBytesMatrix]
---

# SPEC-075: Static Public Site Design System

## Overview

This spec supplies the publication and presentation layer for BUNDLE-032. It does not decide
what Backstop means, where product concepts live, which claims are supported, or how generated
facts are derived. SPEC-072, SPEC-073, and SPEC-074 already own those decisions. Seed 4 consumes
them and makes the resulting product surface buildable, navigable, responsive, deployable, and
subject to the released visual-policy owner.

The design may depart materially from the current landing page. The chosen direction is a technical
field guide organized by the visitor's question: a single quiet shell, strong question-led page
openings, legible long-form models and reference material, conspicuous evidence and boundary
treatments, and clear next actions. The old custom page and Cayman-rendered Markdown are useful
source material, not coequal visual baselines. Presentation consistency is proved by installed
policy and actual-site behavior rather than screenshot similarity.

The runtime remains deliberately boring. Jekyll renders committed Markdown to static HTML; GitHub
Pages publishes the result at `backstop.sh`; canonical navigation works with JavaScript unavailable.
Pinned Node tooling exists only to exercise real browser behavior during verification and contributes
no bytes to the published site.

## Requirements

The frontmatter is normative. These exact matrices are restated for implementation clarity.

### Canonical publication matrix

| Source | Canonical route | Current state | Primary navigation |
|---|---|---|---|
| `docs/index.md` | `/` | Home wordmark only | — |
| `docs/evaluate.md` | `/evaluate/` | Evaluate | 1 |
| `docs/model.md` | `/model/` | Model | 2 |
| `docs/adopt.md` | `/adopt/` | Adopt | 3 |
| `docs/use-cases.md` | `/use-cases/` | Use Cases | 4 |
| `docs/packs.md` | `/packs/` | Packs | 5 |
| `docs/extend.md` | `/extend/` | Extend | 6 |
| `docs/reference.md` | `/reference/` | Reference | 7 |
| `docs/status.md` | `/status/` | Status utility | — |
| `docs/contributing.md` | `/contributing/` | Contributing utility | — |

Every canonical page carries the same ordered navigation. Home has no selected primary or utility
item. A page represented by primary or utility navigation has exactly one `aria-current="page"`.
The wordmark is always the Home route and never masquerades as current-page state on non-Home pages.

### Exact presentation matrix

| Route | `page_kind` | SPEC-072 `hero_question` (consumed verbatim) | Ordered treatments | `next_action` |
|---|---|---|---|---|
| `/` | `home` | What failure does Backstop prevent? | `evidence-cards` | `/evaluate/` |
| `/evaluate/` | `evaluation` | Is Backstop the right control surface for this problem? | `evidence-cards`, `boundary-callouts` | `/model/` |
| `/model/` | `model` | How does Backstop turn intent into a trustworthy verdict? | `evidence-cards`, `local-overflow` | `/adopt/` |
| `/adopt/` | `adoption` | What does a first working adoption require? | `evidence-cards` | `/use-cases/` |
| `/use-cases/` | `use-cases` | Which problem-oriented adoption path applies? | `evidence-cards`, `boundary-callouts` | `/packs/` |
| `/packs/` | `ecosystem` | Which maintained pack already owns this standard? | `evidence-cards`, `generated-regions`, `local-overflow` | `/extend/` |
| `/extend/` | `extension` | When should this concern become a pack? | `evidence-cards`, `boundary-callouts` | `/reference/` |
| `/reference/` | `reference` | What exact interface or behavior do I need? | `generated-regions`, `local-overflow` | `/status/` |
| `/status/` | `status` | What is supported, limited, planned, or intentionally outside Backstop? | `evidence-cards`, `boundary-callouts`, `generated-regions`, `local-overflow` | `/contributing/` |
| `/contributing/` | `contributing` | How can I participate in Backstop and its ecosystem? | `boundary-callouts` | `/` |

Hero strings are copied verbatim from SPEC-072 v1.0.5 and cannot be authored or overridden here. Treatments are present only
when their upstream registry records exist on that route; the matrix names the allowed ordered set,
while sitecheck cross-checks every rendered ID/state/job against SPEC-072/074 rather than accepting
self-consistent invented values.

### Legacy redirect matrix

| Legacy route | Canonical destination |
|---|---|
| `/getting-started.html` | `/adopt/` |
| `/concepts.html` | `/model/` |
| `/artifact-workflow.html` | `/model/` |
| `/pack-authoring.html` | `/extend/` |
| `/cli-reference.html` | `/reference/` |

The aliases preserve already-published entry points without giving them content ownership. They are
one-hop serverless redirects, do not appear in navigation or page copy, and do not create a second
canonical URL.

### Installed design-system acceptance matrix

| Cell | Clean positive | Isolated negative mutation | Required attribution |
|---|---|---|---|
| Tokens | Built corpus uses the installed token contract | Replace one applicable token use with a raw value | Installed token rule + built path |
| Styling | Built corpus contains allowed authored styling | Add one `style` attribute | Installed styling rule + built path |
| Focus | Keyboard focus treatment remains applicable and present | Remove the applicable treatment | Installed focus rule + built path |
| Motion | Motion and reduced-motion treatment agree | Retain motion but remove its reduced-motion override | Installed motion rule + built path |
| Accessibility | Names, landmarks, and relations satisfy applicable rules | Remove one applicable name or landmark relation | Installed accessibility rule + built path |
| Wordmark | Every shell uses the canonical mark | Corrupt one mark | Installed wordmark rule + built path |
| Reusable presentation | Applicable shared patterns are used once and consistently | Duplicate a pattern the owner rule forbids | Installed presentation rule + built path |

The same-release owner export, not SPEC-073's identity proof alone, binds each exact rule, production
filter, mutation bytes, fixtures, and dispatch evidence. Core does not guess selectors or transcribe
a rule. If the released pack lacks the exact seven-cell export, Seed 4 stops at that separately
governed dependency.

Each cell has `rule_id`, `path_filters`, clean/negative fixture refs, deterministic base64 mutation,
and path-fidelity evidence. Acceptance uses eight fresh project roots—clean plus seven cells—with a
fresh install/build and exactly one full, byte-sorted `_site` gate per root. Another rule's incidental
failure never satisfies the intended cell.

### Field-guide instance contract

Core's structural surface is exact: one marked shell/header/nav/main/hero/next-action/footer per page;
claim IDs map to evidence cards, boundary IDs, five-state values, and structured boundary fields to
boundary callouts, generated job IDs and immutable source links to generated regions, adoption IDs
and command digests to command blocks, and each table/Mermaid diagram to one labeled focusable local scroller.
The 1440px shell/prose maxima are 1180px/760px. These are instance structure and composition bounds,
not a local copy of reusable visual policy.

### Rendered journey-link matrix

SPEC-072 owns every record, label, source marker, and Markdown link. Seed 4 owns only the exact
rendered binding below. Each source and destination anchor is case-sensitive and exists once.

| Link ID | Source route/anchor | Destination route/anchor |
|---|---|---|
| `JLINK-001` | `/#why-backstop` | `/evaluate/#failure-fit` |
| `JLINK-002` | `/evaluate/#what-backstop-is` | `/model/#operating-model` |
| `JLINK-003` | `/use-cases/#choose-use-case` | `/evaluate/#fit-decision` |
| `JLINK-004` | `/evaluate/#fit-decision` | `/adopt/#install` |
| `JLINK-005` | `/evaluate/#not-a-fit` | `/status/#adjacent-guidance` |
| `JLINK-006` | `/evaluate/#guarantees` | `/status/#supported-and-limited` |
| `JLINK-007` | `/status/#boundary-states` | `/model/#ownership-boundaries` |
| `JLINK-008` | `/evaluate/#compatibility` | `/reference/#compatibility` |
| `JLINK-009` | `/evaluate/#compatibility-limits` | `/status/#adjacent-guidance` |
| `JLINK-010` | `/model/#operating-model` | `/reference/#artifact-schema-catalog` |
| `JLINK-011` | `/model/#ownership-boundaries` | `/status/#project-boundaries` |
| `JLINK-012` | `/adopt/#install` | `/reference/#configuration` |
| `JLINK-013` | `/adopt/#verify-enforcement` | `/model/#enforcement-loop` |
| `JLINK-014` | `/model/#enforcement-loop` | `/reference/#gate` |
| `JLINK-015` | `/use-cases/#choose-use-case` | `/adopt/#adoption-paths` |
| `JLINK-016` | `/use-cases/#pack-backed-use-cases` | `/packs/#choose-a-pack` |
| `JLINK-017` | `/packs/#installed-pack-catalog` | `/reference/#pack-commands` |
| `JLINK-018` | `/packs/#choose-a-pack` | `/status/#pack-direction` |
| `JLINK-019` | `/extend/#pack-or-not` | `/reference/#pack-artifact` |
| `JLINK-020` | `/extend/#author-a-pack` | `/contributing/#contribution-paths` |
| `JLINK-021` | `/evaluate/#evidence` | `/reference/#source-traceability` |
| `JLINK-022` | `/packs/#installed-pack-catalog` | `/reference/#cli-command-catalog` |
| `JLINK-023` | `/reference/#cli-command-catalog` | `/status/#release-history` |
| `JLINK-024` | `/status/#adjacent-guidance` | `/contributing/#external-ownership` |

JLINK-024 is the sole embedded claim-region link. Its source-only marker sits inside CLAIM-005 and
immediately precedes the continuation link. Seed 4 must bind that one physical link to one rendered
anchor carrying both the JLINK-024 identity and the boundary-continuation identity; two rendered
anchors are a cardinality failure, not a valid representation of two consumer roles.

### Structured owner-rendering matrix

| Owner record | Required rendered contract |
|---|---|
| Any boundary | One owner-route/anchor callout with exact `data-boundary-id`, `data-boundary-state`, and one nonempty `[data-boundary-explanation]`. |
| `supported`, `limitation`, `planned`, or `non-goal` boundary | No continuation or guarantee-denial element because both owner fields are null. |
| `adjacent-guidance` boundary | Exactly one `a[data-boundary-continuation][data-journey-link-id]` and one nonempty `[data-boundary-guarantee-denial]`, matching owner fields. For BOUNDARY-005, the one anchor carries `data-journey-link-id="JLINK-024"` and both identities on that same element, appears between explanation and denial inside the callout, and preserves all three visible payloads byte-for-byte through annotation. |
| `cli-command-catalog` | One tree source link for `cmd/backstop`, resolved at the exact site commit. |
| `artifact-schema-catalog` | One blob source link per generated schema record's `source`, in record order, resolved at the exact site commit. |
| `installed-pack-catalog` | Exactly two blob source links, `backstop.yml` then `backstop.lock`, resolved at the exact site commit. |
| `release-history` | One commit source link per generated release record's full commit, in record order. |
| `ADOPT-INSTALL` at `/adopt/#install` | One `pre[data-adoption-instruction-id="ADOPT-INSTALL"][data-command-sha256]` with exact owner command bytes and digest. |
| `ADOPT-CONFIGURE` at `/adopt/#configure` | One `pre[data-adoption-instruction-id="ADOPT-CONFIGURE"][data-command-sha256]` with exact owner command bytes and digest. |
| `ADOPT-ENFORCE` at `/adopt/#verify-enforcement` | One `pre[data-adoption-instruction-id="ADOPT-ENFORCE"][data-command-sha256]` with exact owner command bytes and digest. |

The generated links remain inside their one owner job section and source-marker pair. `site` commit
bindings use the exact full lowercase build or deployment commit; record-bound release commits remain
their generated full commits. Visible link labels and command text remain upstream-owned copy.

## Implementation

Implementation proceeds in this order:

1. Verify the SPEC-073 design-system import and require a newer same-release seven-cell owner export
   with fingerprints and path-fidelity evidence. Do not infer applicability from identity alone.
2. Add the locked Ruby build. Replace Cayman configuration, make each SPEC-072 source emit its exact
   permalink, and convert the five retired legacy sources into frontmatter-only users of the shared
   no-JavaScript redirect layout. Remove the migrated `docs/index.html` so `docs/index.md` alone owns
   the root output. Confirm the exact build command before presentation work.
3. Implement one semantic default layout and shared header, footer, and page-hero includes. Render
   navigation from the topology data without allowing Liquid to infer or change content ownership.
4. Implement the field-guide composition in `site.css` using the released token interface. Keep the
   header and every destination visible without JavaScript. Use the three declared responsive shell
   modes; contain table/diagram overflow locally. After each build, copy and hash-verify the one
   owner-exported token CSS asset into `_site`, linked before `site.css`; never check it into Core.
5. Integrate Seed 3's four source include regions exactly where declared. Jekyll may wrap and style
   them but must not parse their authoritative inputs or regenerate their records.
6. After the exact Jekyll build, run the deterministic owner-contract annotation pass with the full
   site commit. Match source markers and owner records structurally; bind all rendered JLINKs and
   structured boundary fields, resolve generated source descriptors, and mark the three adoption
   command blocks without changing visible copy or checked-in source. Bind embedded JLINK-024's one
   continuation link to one dual-identity anchor inside BOUNDARY-005, preserve explanation/link/denial
   text-node bytes and order, and refuse split or duplicate anchors. Refuse an unresolved or ambiguous
   owner edge instead of guessing from prose.
7. Parse every disposable `_site` document and validate the exact route, redirect, canonical, link,
   anchor, navigation, current-state, landmark, JLINK, structured-boundary, generated-region,
   generated-source-link, adoption-instruction, digest, asset, JavaScript-absence, and CNAME
   contracts. Diagnostics are stable and path-specific.
8. Run Playwright against a disposable local static-file server with JavaScript disabled. Exercise all
   navigation destinations by keyboard at all three viewport sizes, assert document overflow and
   element overlap, capture focus clipping, then inject the exact 200% root-font rule, verify computed
   2x size, await fonts/two frames, and rerun the complete matrix. This server is test
   infrastructure only and never part of deployment.
9. Create eight independent complete project roots. In each, clean-install, build, and annotate with
   the same full commit, preserve every
   candidate as `_site/<original-relative-path>`, and invoke one full sorted gate. Apply only the
   exported mutation in each negative and require its intended installed rule and exact path.
10. Validate full-SHA official action locks, annotate with the workflow head SHA, stamp/upload exactly `_site`, deploy, then run the required
   post-deploy API/action/marker/HTTPS consensus proof over all canonical routes and aliases.

The local checker owns facts about this Core instance: route membership, exact link resolution,
rendered owner-record binding, upstream include placement, workflow wiring, and installed-result attribution. It does not own token,
style, focus, motion, accessibility, wordmark, or reusable-presentation judgments. Those verdicts
must originate from the installed pack.

## Verification

`./scripts/verify-public-site.sh` is the single acceptance entrypoint. It runs measured Go tests,
the two upstream integration checks, the exact locked production build, the deterministic
owner-contract annotation pass, rendered-site structural checks, the real no-JavaScript browser
matrix, the installed design-system positive/negative matrix, and Pages workflow tests. It fails
closed on missing dependencies and leaves the working tree unchanged.

Local/PR verification proves post-deploy workflow wiring and the verifier's failure matrices; the
deployed workflow itself executes authoritative Pages/Actions API and HTTPS checks after deploy.
Neither local structural checks nor one external evidence channel may stand in for that consensus.

The positive path is not sufficient by itself. Verification must independently delete or corrupt
each canonical route, navigation cell, link class, redirect, generated region, CNAME surface,
viewport behavior, JLINK binding, boundary subfield, generated source link, adoption instruction,
design-system matrix cell, and Pages prerequisite described in frontmatter. JLINK-024 mutations must
independently remove each identity, split the identities, duplicate the anchor, alter each visible
payload, move the anchor outside BOUNDARY-005, and place it on either wrong side of the
explanation/denial sequence. A
generic "site failed" message is not acceptable evidence; the responsible phase and public or owner
identity must be visible.

## Sharp Edges

- **SPEC-073 identity is not visual applicability.** Seed 4 needs the same-release seven-cell owner
  export; treating a correct lock/hash as mutation evidence is a false handoff.
- **Temporary paths can evade production filters.** Every corpus is a full project root and every
  candidate retains exact `_site/<relative-path>` identity. A flattened fixture is not actual-site proof.
- **Another rule can fail accidentally.** Only the exported intended rule and target path satisfies a
  negative cell; generic nonzero exit is insufficient.
- **Pack applicability can be guessed incorrectly.** The seven mutation cells are selected from the
  installed release's manifest and evidence. A local selector chosen because it sounds plausible can
  miss the production filter and reproduce ISSUE-184's false green.
- **A local checker can become a second policy owner.** It may confirm that an installed finding names
  an installed rule and built path; it may not scan for raw colors, style attributes, focus selectors,
  motion queries, accessibility patterns, wordmark bytes, or reusable components itself.
- **Generated Markdown can be rendered but not consumed.** A page that rebuilds an equivalent table
  from source has the right screenshot and the wrong ownership. Seed 3's marked fragment, digest, and
  one-owner region remain load-bearing.
- **Annotation can become a shadow content owner.** The pass may bind owner records to rendered
  elements and resolve commit placeholders, but it may not synthesize a missing link, boundary field,
  command, or provenance record. Ambiguity is a build failure, not permission to choose plausible copy.
- **A correct URL can still have mutable provenance.** Generated `site` links must contain the full
  tested/deployed commit, while release links retain their record commits. `main`, `HEAD`, a tag, or
  an abbreviated SHA produces an apparently useful but non-reproducible evidence edge.
- **Structured boundary fields can collapse back into prose.** A visually complete callout without
  separate explanation, continuation, and denial markers is not machine-verifiable and leaves Seed 5
  guessing. Each owner field and its forbidden-null case remains load-bearing.
- **One source link can serve two rendered identities.** JLINK-024 is both the journey link and the
  BOUNDARY-005 continuation. Rendering two anchors duplicates visible copy and violates both owner
  cardinalities; putting one identity on each anchor is not equivalent to one dual-identity element.
- **A displayed command can drift from executable owner data.** Exact visible bytes and the owner
  digest are both required. Seed 4 does not expose argv/environment as page copy or execute commands;
  Seed 5 separately proves that structured execution reaches a real gate result.
- **Liquid delimiter collision remains real.** Any owner capability that emits Liquid-bearing files
  must use the released ISSUE-182 fix or avoid nested templating. Fragile escaping is not an accepted
  implementation detail.
- **Legacy redirects can become parallel IA.** The five aliases exist only for inbound compatibility.
  Linking to them, chaining them, or giving them independently editable content creates a second route
  model.
- **No-JavaScript can pass structurally and fail behaviorally.** Visible links in HTML do not prove
  keyboard order, focus visibility, overlap, or narrow-screen reachability. The real browser matrix is
  required in addition to parsing output.
- **Test infrastructure can leak into runtime.** Playwright and its static server are allowed only in
  verification. A script tag, bundled asset, service worker, or runtime call in `_site` fails even if
  the feature is useful.
- **Pages can publish stale truth.** A successful Jekyll build does not imply Seed 3 freshness or
  released-pack integrity. Pages must rerun those checks and cannot upload before they pass.
- **Tag and main publication have different clocks.** SPEC-074's tag release may wait for regenerated
  history on main. Pages deploys main only; adding tags as a Pages trigger breaks that handshake.
- **CNAME presence is weaker than continuity.** Source and built bytes, workflow artifact root, and
  deployed Pages configuration all matter. API, action output, deployment marker, and HTTPS routes
  must agree on current identity.
- **Pages upload excludes hidden paths by default.** The deployment marker lives under `.well-known`;
  `include-hidden-files: true` is mandatory and structurally negative-tested.
- **Policy equivalence is not mechanically decidable.** Exact protected fingerprints and forbidden
  delivery roles are bounded checks; modified conceptual copies remain a human review violation.
- **DPR and screenshots do not prove text relayout.** Computed 2x root font, fonts-ready/two-frame
  stabilization, and the rerun interaction matrix are load-bearing.
- **Same-origin absolute HTTPS causes link drift.** It is required in canonical metadata but rejected
  in anchors, before the generic cross-origin HTTPS allowance.

## Integration Contract

SPEC-072 v1.0.5 remains the owner of content, routes, navigation meaning, JLINK records and labels,
structured boundaries, adoption instructions, concepts, claims, evidence, and final copy. SPEC-073 remains the owner of Core's separately released pack-consumption
contract and documentation-semantic execution. SPEC-074 remains the owner of generated records,
markers, digests, source descriptors, freshness, and release-history handshake. This spec owns
layouts, responsive rendering, rendered owner-record attributes and immutable source URLs, built
route/link behavior, actual-site design-system execution, and Pages deployment. Seed
5 may traverse and falsify the built result but must not replace any of these checks with journey-only
assertions.

## Review Questions

1. Does the built route matrix exactly match SPEC-072, including the page with no numbered
   neighborhood, without inventing another canonical owner?
2. Are all seven primary and both utility destinations visible, ordered, current, and keyboard
   reachable at 360px with JavaScript disabled?
3. Can any internal link resolve only because the verifier normalizes case, supplies `.html`, ignores
   a missing fragment, or follows a legacy alias?
4. Does each of the seven design-system negatives prove installed-rule applicability before mutation
   and return that exact installed rule plus the production built path?
5. Could deleting the installed pack leave a local scanner or cached result that still reports green?
6. Did site-specific composition consume owner tokens without copying the owner's policy, selectors,
   token declarations, fixtures, or validator logic?
7. Do the four generated regions retain exact owner route/anchor, marker, digest, and one-time
   consumption, including every ordered immutable source link, after layout wrapping?
8. Does 200% reflow or a 360px viewport make any primary destination, focus ring, table, diagram, or
   next action unreachable?
9. Can a tag, alternate workflow, widened permission, skipped check, or prebuilt directory reach the
   Pages deployment action?
10. Is every new dependency build-time or verification-only, with no runtime complexity introduced
    without a separately governed functional requirement?
11. Does each JLINK render exactly once under its source anchor with the owner label and exact
    destination, without falling back to global navigation or an annotation-authored link?
12. Can removing any structured boundary field, generated source link, or adoption instruction
    marker produce an attributed failure before Seed 5 journey acceptance runs?
13. Does embedded JLINK-024 remain one dual-identity anchor inside BOUNDARY-005, in exact visible
    explanation/link/denial order, with either missing identity and every split/duplicate mutation rejected?

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md` v0.6.0 — source bundle, Seed 4 partition,
  REQ-009@2.0.0, REQ-013@1.0.0, OQ-6, DD-9, DD-12, and cross-repository sharp edges.
- `specs/SPEC-072-public-product-model.spec.md` v1.0.5 — ten source/path pairs, exact navigation,
  JLINK and adoption-instruction records, structured boundary fields, content ownership,
  claim/evidence/boundary registries, final copy, and Mermaid authority.
- `specs/SPEC-073-documentation-semantics-integration.spec.md` — released pack identity/pin boundary,
  clean install, and semantics integration; it does not establish design-system applicability.
- `specs/SPEC-074-derived-product-truth-pipeline.spec.md` v1.0.4 — four checked-in generated regions,
  source include markers, typed source descriptors and URL templates, source-level freshness,
  and the tag/latest-main release handshake. This spec owns their Jekyll build, rendered digest and
  immutable-anchor verification, Pages freshness/no-tag behavior, and deployment acceptance.
- `specs/SPEC-071-website-expansion.spec.md` — canceled narrow docs-shell decomposition; historical
  source material only.
- `docs/index.html`, `docs/_config.yml`, `docs/CNAME`, `docs/assets/css/backstop.css`, and
  `docs/assets/css/backstop-tokens.css` — current site and custom-domain source material.
- `.github/workflows/ci.yml` and `.github/workflows/release.yml` — current branch and tag delivery
  behavior that Pages must compose with rather than bypass.
- `backstop.yml`, `backstop.lock`, future `.backstop/website-pack-releases.yml`, and future installed
  `contracts/public-site-acceptance.yml` —
  design-system declaration, lock, and durable owner-release evidence.
- `.github/pages-actions.lock.yml` — future immutable official-action pin contract.
- Commit `63f70f7e668486202cc1897cfcce94f82769b477` — current landing page and initial released
  design-system consumption.
- `issues/ISSUE-182-recipe-literal-placeholder-escaping.issue.md`,
  `issues/ISSUE-183-local-pack-relock-refreshes-stale-install.issue.md`, and
  `issues/ISSUE-184-fixture-path-filter-diagnostics.issue.md` — literal-template, stale-install, and
  production-path-fidelity incidents.
