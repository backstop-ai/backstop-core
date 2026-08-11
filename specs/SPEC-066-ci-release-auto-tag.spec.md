---
title: "CI Release Auto Tag"
number: SPEC-066
created: "2026-08-10"
updated: "2026-08-10"
status: draft
schema_version: spec/v1
spec_version: 2.1.0

implementation:
  summary: >
    BUNDLE-031 Spec Seeds 1 and 3, authored against the bundle's v0.5.0 founder-ruled
    pivot (2026-08-10) and NOT against the pre-pivot design: there is no `backstop gate`
    involvement, no SARIF, no engine binding, no approval step, and no in-flight
    detection anywhere in this spec. The capability is a pure CI-time auto-tagger with
    two halves, both shipped as plain shell scripts in the EXTERNAL
    `backstop-ai/go-distribution` pack (DD-3's home, unchanged by the pivot) and invoked
    directly by a thin workflow step. HALF ONE — `derive-release-delta.sh` — is the
    read-only derivation: from the highest reachable `vX.Y.Z` tag to the analyzed commit
    it computes the commit delta, classifies each commit corpus-only or code, derives the
    delta's newly-closed issues from git (never from a stored field), maps each issue's
    typed `issue.type` through a DECLARED DATA TABLE to a semver tier, selects the highest
    tier present, names every uncovered code commit and every artifact it could not read
    as a first-class caveat, renders a receipt-shaped release-note body from
    `delivered_by`/`resolved-by`, and writes one JSON document to stdout. It never tags,
    never pushes, and never writes anything. HALF TWO — `tag-from-release-delta.sh` — is
    the acting half: it parses that JSON STRUCTURALLY, validates it fail-closed against
    seven named conditions including a `head`↔sha-argument identity check, refuses a
    colliding or implausible version, and otherwise creates an annotated tag at the
    analyzed commit and pushes exactly that one ref. The acting logic lives in a SCRIPT
    rather than inline in the workflow for one load-bearing reason: a guard that only
    exists as YAML `run:` text can be asserted structurally but never FALSIFIED, and every
    safety property in this spec — fail-closed on malformed output, refusal on tag
    collision, no force push — is proven by executing the real script against a synthetic
    git repository with a real bare `origin`. The two halves are additionally proven to
    COMPOSE: the real derivation's real stdout is piped into the real acting script and
    the resulting tag is read back off the bare remote. backstop-core's OWN new artifacts
    are the workflow file `.github/workflows/release-auto-tag.yml` and the
    `backstop.yml`/`backstop.lock` fleet entry that makes `backstop pack install`
    materialize the scripts — the latter SUBSUMING ISSUE-111. ZERO core binary changes:
    no new command, no new Go production symbol, no gate step. The workflow chains off the
    CI workflow's successful completion on `main` rather than off the raw push, because
    `release.yml`'s shipped `require-green-ci` job fails closed when no COMPLETED green CI
    run exists for the tagged commit — a tag pushed before CI concludes would be a tag
    that can never release. All mandated tests are Go tests in `cmd/backstop` that execute
    the real scripts and parse the real workflow files.
  subject: cmd/backstop

verification:
  level: integration
  test_command: go test ./cmd/backstop/ -race -coverprofile=cover.out
  coverage_threshold: 80

requirements:
  - id: REQ-001
    text: >
      Both halves of the machinery MUST ship in the EXTERNAL `backstop-ai/go-distribution`
      pack as self-contained shell scripts — `scripts/derive-release-delta.sh` and
      `scripts/tag-from-release-delta.sh` — invoked DIRECTLY by a workflow step as
      `bash .backstop/packs/backstop-ai/go-distribution/scripts/<name>.sh`. backstop-core
      MUST declare `backstop-ai/go-distribution` in `backstop.yml` and `backstop.lock` at
      the SAME published pack version, and that version MUST be one that contains both
      scripts — strictly newer than the pack's current published `v0.1.0`, which predates
      them — so `backstop pack install` materializes both at that path. This declaration is
      the entirety of ISSUE-111's scope, delivered here rather than separately (see
      References). It is PROHIBITED to deliver this capability as a gate-dispatched engine
      binding: no `scope_kind`/`input_mode` declaration for these scripts, no SARIF
      `convert:` bridge, no new gate step, and no participation in `backstop gate` — nothing
      consumes their output as a gate finding. It is PROHIBITED to add a core command, a new
      Go production symbol in `cmd/backstop` or `pkg/`, or any other change to the backstop
      binary; the only backstop-core-native additions are the workflow file, the fleet
      declaration, and tests.
    supports: release-currency-versioning-machinery:REQ-013@2.0.0
  - id: REQ-002
    text: >
      `derive-release-delta.sh` MUST be READ-ONLY and side-effect-free: it MUST NOT create a
      tag, push, commit, check out, reset, merge, rebase, fetch, or update any ref, and MUST
      NOT write to the working tree, the index, or any artifact file. Its ONLY output on
      stdout is a single JSON document; every diagnostic, warning, and progress line MUST go
      to stderr, so `$(bash derive-release-delta.sh)` is parseable without filtering. The
      document MUST carry exactly the keys `schema` (constant `release-delta/v1`),
      `last_tag`, `head`, `delta` (`commit_count`, `code_commit_count`,
      `corpus_only_commit_count`, `closed_issues`), `proposed` (`bump`, `version`),
      `caveats` (`uncovered_commits`, `unreadable_artifacts`), and `release_notes`. `head`
      MUST be the FULL 40-character hexadecimal sha of the analyzed commit — never an
      abbreviation and never a symbolic name — because REQ-013 compares it byte-for-byte
      against the sha the workflow passes the acting half. Every string the document
      interpolates from the corpus or from git — issue titles and commit subjects above all —
      MUST be JSON-escaped so a title containing `"` or `\` cannot produce a malformed
      document. The script MUST depend only on `git` and POSIX shell utilities (`grep`,
      `sed`, `sort`, `awk`); `jq`, `yq`, `python3`, and any other non-POSIX interpreter are
      PROHIBITED IN THIS HALF so the derivation stays portable to any consumer runner.
      Because that POSIX-only constraint makes this the half that reads corpus text with
      line-oriented tools, every TYPED FIELD the derivation extracts from an artifact file —
      `issue.status`, `issue.type`, the issue title, and the `delivered_by` / `resolved-by`
      receipt pointers — MUST be read ONLY from that file's YAML FRONTMATTER BLOCK: the lines
      strictly between the file's FIRST `---` delimiter line and the NEXT `---` delimiter
      line. Extracting any of those fields from the document BODY is PROHIBITED, and an
      unbounded whole-file match for a field name (`grep '^status:' <file>` and its
      equivalents) is PROHIBITED even where it happens to return the right value today. The
      body is prose, and prose already collides with this shape in the live corpus:
      `issues/ISSUE-078-contracts-pack-const-signature-support.issue.md` carries the
      column-zero body line `status: pass` inside a fenced `backstop pack test` transcript, so
      an unbounded scraper is already reading at least one line that is not a field. A file
      that exists at the analyzed commit but whose frontmatter block cannot be delimited (no
      leading `---`, or no closing `---`) is a file whose `issue.status` cannot be read, and
      MUST fail closed exactly as REQ-005 requires — falling back to a whole-file scrape is
      PROHIBITED.
    supports: release-currency-versioning-machinery:REQ-001@1.1.0
  - id: REQ-003
    text: >
      The baseline of the delta MUST be the HIGHEST strict-semver `v<major>.<minor>.<patch>`
      tag REACHABLE from the analyzed commit (`git tag --merged`, version-ordered), never
      the NEAREST tag that `git describe` would report. A tag that is not reachable from the
      analyzed commit MUST be ignored, and a tag that is not a strict
      `v<major>.<minor>.<patch>` — `nightly`, `v1.2`, `0.1.1`, `v1.2.3-rc1` — MUST be ignored
      for baseline selection. The delta is exactly the commits in `<last_tag>..<analyzed
      commit>`, and `delta.commit_count` MUST equal the size of that range. When NO strict-
      semver tag is reachable, the script MUST FAIL CLOSED: exit non-zero, emit no JSON, and
      name the condition on stderr — cutting a repository's first release is a founder act,
      not a derivation.
    supports: release-currency-versioning-machinery:REQ-001@1.1.0
  - id: REQ-004
    text: >
      Every commit in the delta MUST be classified CORPUS-ONLY or CODE. A commit is
      CORPUS-ONLY when EVERY path it touches falls under the declared corpus-path prefix set;
      a commit touching at least one path outside that set is CODE, including a MIXED commit
      that touches both. A commit whose diff name-set is EMPTY — a merge commit or an empty
      commit — MUST be classified CORPUS-ONLY, the direction that can never over-version. The
      prefix set MUST be DATA: one declared list at the top of the script, editable without
      touching the derivation logic, whose current members are `issues/`, `plans/`,
      `bundles/`, `directives/`, `specs/`, `adrs/`, `capabilities/`, `docs/`, and
      repository-root `*.md`. Corpus-only commits MUST COUNT toward `delta.commit_count` and
      `delta.corpus_only_commit_count`. A commit — of either class — MUST NEVER contribute a
      bump tier BY ITSELF: a tier only ever comes from a newly-closed issue via REQ-005 and
      REQ-006. The corpus-only rule therefore binds at DELTA level, not at commit level: when
      `delta.code_commit_count` is `0` — every commit in the delta is corpus-only —
      `proposed.bump` and `proposed.version` MUST BOTH be JSON `null` REGARDLESS of any issue
      the delta closed, because the release artifact would be byte-identical to the one the
      baseline tag already published. Conversely, when the delta carries at least one CODE
      commit it is PROHIBITED for a corpus-only commit in that same delta to suppress a tier
      a newly-closed issue independently earns — a close-out commit touching only
      `issues/*.issue.md` is the ORDINARY way this repository closes an issue, and treating
      it as tier-suppressing would mean the machinery essentially never fires.
    supports: release-currency-versioning-machinery:REQ-005@1.0.0
  - id: REQ-005
    text: >
      The artifact-to-release join MUST be DERIVED from git at derivation time and never
      stored. An issue artifact is IN the delta when ALL THREE hold: at least one commit in
      the delta touched its `issues/*.issue.md` file; its `issue.status` read at the ANALYZED
      commit (`git show <commit>:<path>`) is `closed`; and its `issue.status` read at the
      BASELINE tag (`git show <last_tag>:<path>`) is NOT `closed`, where a path that does not
      exist at `<last_tag>` reads as NOT closed. That third conjunct BOUNDS the join to the
      delta window: an issue that already shipped closed in an earlier release and is merely
      re-touched here (a Resolution note, a citation fix) MUST be excluded, so no issue is
      ever tiered into two releases. Every other `issue.status` value MUST be excluded and
      contribute no tier — the live statuses `open`, `ready`, `in-progress`, `blocked`, and
      the retirement terminals `replaced`, `canceled`, `obsoleted` — as MUST a `closed` issue
      whose file no commit in the delta touched. A touched `issues/*.issue.md` path that does
      NOT EXIST at the analyzed commit (deleted within the delta) MUST contribute no tier,
      MUST NOT fail the derivation, and MUST be listed in `caveats.unreadable_artifacts` as
      `{path, reason: deleted-in-delta}`; artifact deletion is routine in this corpus (the
      2026-08-10 SPEC-063/064/065 deletion), and halting all releasing on it would be a
      brittle coupling. A path that DOES exist at the analyzed commit but whose `issue.status`
      cannot be read MUST fail closed — that is a corrupt artifact, not a routine deletion.
      `caveats.unreadable_artifacts` MUST be present on every successful run, an empty array
      when there are none. It is PROHIBITED to add a `released_in:` field (or any equivalent)
      to any artifact schema, and PROHIBITED for the derivation to write to the corpus at all.
    supports: release-currency-versioning-machinery:REQ-006@1.0.0
  - id: REQ-006
    text: >
      Each delta issue's tier MUST come from its typed `issue.type` read out of the artifact,
      never from a commit message, resolved through a DECLARED DATA TABLE — one associative
      structure at the top of the script — carrying exactly these five entries: `bug` →
      PATCH, `enhancement` → MINOR, `technical-debt` → PATCH, `question` → PATCH,
      `policy-violation` → PATCH. The three PATCH defaults beyond `bug` are an acknowledged
      narrow first cut and MUST be refinable by editing the table alone. A delta issue whose
      `issue.type` key is ABSENT, or whose value is not a key of the table, MUST FAIL CLOSED:
      the script exits non-zero, emits no JSON, and names the offending issue and value on
      stderr. It is PROHIBITED to default, guess, skip, or otherwise absorb an unmapped type —
      an unmapped value means the enum grew past the table, and silently absorbing it
      under-versions a public release.
    supports:
      - release-currency-versioning-machinery:REQ-002@1.0.0
      - release-currency-versioning-machinery:REQ-003@1.0.0
  - id: REQ-007
    text: >
      `proposed.bump` MUST be the HIGHEST tier present among the delta's issues, ordered
      PATCH < MINOR. When no delta issue contributes a tier — an empty delta, a delta whose
      only code commits are uncovered, or a delta whose issues are all excluded by REQ-005 —
      `proposed.bump` and `proposed.version` MUST BOTH be JSON `null`. They MUST likewise
      both be `null` whenever REQ-004's delta-level gate applies (`delta.code_commit_count`
      is `0`), which OVERRIDES any tier the delta's issues would otherwise contribute.
      Otherwise `proposed.version` MUST be `last_tag` with the selected position incremented
      by exactly one and every lower position ZEROED: a MINOR against `v0.1.1` yields
      `v0.2.0`, a PATCH against `v0.1.1` yields `v0.1.2`.
    supports: release-currency-versioning-machinery:REQ-002@1.0.0
  - id: REQ-008
    text: >
      No derivation path may produce a MAJOR bump. While `last_tag`'s major component is `0`,
      MINOR MUST be the highest tier the script can emit and the string `major` MUST NEVER
      appear as a `proposed.bump` value for any combination of delta inputs. When
      `last_tag`'s major component is `1` or greater the script MUST FAIL CLOSED — exit
      non-zero, emit no JSON, name the condition — because the pre-1.0 refusal that makes the
      major position meaningless no longer applies and this spec introduces no post-1.0
      breakage model. It is PROHIBITED to fall back to PATCH or MINOR in that case, and
      PROHIBITED to introduce a bespoke breakage model to suppress a major pre-1.0; the
      refusal defers to standard semver. The operational consequence — the auto-tag job goes
      RED on every push once a `v1.0.0` tag exists, until a successor capability replaces this
      one — is INTENDED and is named as a Sharp Edge rather than softened here.
    supports: release-currency-versioning-machinery:REQ-004@1.0.0
  - id: REQ-009
    text: >
      A CODE commit in the delta is COVERED when its commit message names at least one
      artifact id matching the declared reference pattern
      `(ISSUE|PLAN-ISSUE|SPEC|BUNDLE|DIR)-[0-9]{3}` AND a matching artifact file exists in the
      corpus at the analyzed commit; otherwise it is UNCOVERED, including the case where the
      message names an id no artifact file backs. Every uncovered code commit MUST be listed
      INDIVIDUALLY in `caveats.uncovered_commits` with its short sha and its subject line.
      Uncovered commits MUST NEVER suppress a proposal derived from covered issues, MUST NEVER
      create one, and MUST NEVER receive an implicit tier. `caveats.uncovered_commits` MUST be
      present on every successful run — an empty array when there are none — because it is a
      first-class part of the output contract, not an error path.
    supports: release-currency-versioning-machinery:REQ-007@1.0.0
  - id: REQ-010
    text: >
      `release_notes` MUST be a receipt-shaped markdown body derived from the same query that
      produced the number: EXACTLY one line per delta issue and no line that no delta issue
      backs, each line carrying the issue id, its `issue.type`, its title, and its receipt
      pointer — `delivered_by` when present, else `resolved-by` when present, else the literal
      `no-traceability-pointer`. Hand-summarized prose is PROHIBITED. When the delta has no
      issues, `release_notes` MUST be the empty string.
    supports: release-currency-versioning-machinery:REQ-009@1.0.0
  - id: REQ-011
    text: >
      backstop-core MUST gain `.github/workflows/release-auto-tag.yml` whose trigger key set
      is EXACTLY `{workflow_run}`, with `types: [completed]` and `branches: [main]`. Its
      `workflows:` value MUST NAME THE CI WORKFLOW BY THE `name:` VALUE
      `.github/workflows/ci.yml` ITSELF DECLARES, and that identity is a CROSS-FILE JOIN that
      MUST be verified by reading BOTH files and comparing — never by asserting a hardcoded
      literal on one side, which would let a rename of `ci.yml`'s `name:` silently and
      permanently disable auto-tagging with no failing test (the ISSUE-109 defect class). The
      following trigger families are PROHIBITED: `push`, `pull_request`, `pull_request_target`,
      `schedule`, `workflow_dispatch`, `repository_dispatch`, and any tag trigger. Its single
      job MUST additionally be gated on `github.event.workflow_run.conclusion == 'success'`
      AND `github.event.workflow_run.head_branch == 'main'`. The analyzed and tagged commit
      MUST be `github.event.workflow_run.head_sha`, pinned explicitly on the checkout `ref:`
      with `fetch-depth: 0` and tags fetched; it is PROHIBITED to select the commit via
      `github.sha` or via the checkout default, which under a `workflow_run` event name the
      default branch head rather than the triggering commit. Chaining off CI completion rather
      than off the raw push is REQUIRED, not cosmetic: `release.yml`'s shipped
      `require-green-ci` job fails closed when no COMPLETED successful `ci.yml` run exists for
      the tagged commit, so a tag pushed at push-time would be a tag that can never release.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-012
    text: >
      `tag-from-release-delta.sh`, invoked by that job with the derivation's captured document
      and the analyzed sha, MUST fire the release with NO human approval anywhere. When
      `proposed.version` is `null` it MUST create nothing, push nothing, and exit 0. When
      `proposed.version` is a strict `v<major>.<minor>.<patch>` that passes REQ-013, REQ-014
      and REQ-018, it MUST create an ANNOTATED tag at the analyzed sha whose message is the
      derivation's `release_notes`, and push EXACTLY that one ref via `git push origin
      refs/tags/<version>`. The flags `--force`, `-f`, a leading `+` refspec, `--tags`, and
      `--follow-tags` are PROHIBITED, as is pushing any branch, editing any file, opening a
      pull request or issue, publishing a proposal to any surface, waiting on a human word, or
      introducing a scheduled job or a new artifact type. Everything downstream of the pushed
      tag is `release.yml`'s, unchanged.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-013
    text: >
      The acting half MUST FAIL CLOSED — create and push NOTHING and exit non-zero — on every
      one of these SEVEN conditions: (1) the derivation exited non-zero; (2) its captured
      stdout is not parseable JSON; (3) the parsed document's `schema` value is not
      `release-delta/v1`; (4) the document's `head` value is not byte-identical to the sha
      argument the caller passed — a divergence means the tag would land on a commit the
      derivation never analyzed, silently defeating the whole `workflow_run` design; (5)
      `proposed.version` is non-null but not a strict `v<major>.<minor>.<patch>`; (6)
      `proposed.version` is not strictly greater than the `last_tag` the same document
      reports; or (7) `proposed.version` does not increment EXACTLY ONE semver position of
      `last_tag` with every lower position zeroed — `v0.1.1` may only become `v0.1.2` or
      `v0.2.0`, never `v0.1.3`, `v0.3.0`, or `v1.0.0` — so an arithmetic bug cannot skip a
      public version. Substituting a default or fallback version, retrying with a guessed
      bump, and continuing past any of these conditions (including via `continue-on-error` on
      the job or any step) are PROHIBITED. A wrong tag is worse than no tag: a tag triggers a
      public release pipeline, and no correct recovery exists once one is pushed.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-014
    text: >
      Before creating any tag the acting half MUST probe the REMOTE for the proposed ref
      (`git ls-remote --tags origin refs/tags/<version>`) and, when it already exists, create
      and push NOTHING and exit non-zero with a diagnostic naming the existing tag — never
      overwrite it, never silently succeed. The probe MUST run BEFORE tag creation, not after.
      The workflow MUST declare a `concurrency` group whose value is a FIXED LITERAL — it is
      PROHIBITED to key it on `github.sha`, `github.ref`, `github.run_id`, or any other
      per-run expression — with `cancel-in-progress: false`, so two runs can never derive and
      push concurrently and no run is cancelled between creating a tag and pushing it. Because
      that fixed group serializes the whole channel, the job MUST also declare
      `timeout-minutes` at a finite value, so a hung run cannot hold the group indefinitely and
      block every subsequent auto-tag silently.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-015
    text: >
      The tag push MUST authenticate with a repository secret that is NOT `secrets.GITHUB_TOKEN`
      — a PAT or GitHub App token held in a distinctly named secret — because GitHub does not
      raise workflow events for refs pushed with the default token, so a tag pushed with
      `GITHUB_TOKEN` would never start `release.yml` and would silently produce a tag that
      never releases. This MUST be established by a POSITIVE MECHANISM, not by the absence of a
      string: the checkout step MUST declare `with.token: ${{ secrets.<NAME> }}` where `<NAME>`
      is a literal secret name other than `GITHUB_TOKEN`, so the credential
      `actions/checkout` persists into `.git/config` — and therefore the credential
      `git push origin` inherits — is that named secret. Relying on the checkout DEFAULT is
      PROHIBITED: `actions/checkout`'s `persist-credentials` defaults to `true` and writes the
      AMBIENT `GITHUB_TOKEN` into `.git/config` even when the string `secrets.GITHUB_TOKEN`
      appears nowhere in the file, so a denylist alone would pass green on a broken workflow.
      Setting `persist-credentials: false` on that checkout is likewise PROHIBITED, with no
      exception: it would strip the very credential the push needs. The alternative — allowing
      `persist-credentials: false` where the push step supplies the same named secret by
      another explicit means (a remote URL carrying the token, an `extraheader` config, a
      `GIT_ASKPASS` shim) — was CONSIDERED and REJECTED. This workflow has exactly one push
      and therefore needs exactly one credential mechanism; permitting a second would make the
      requirement an either/or that no implementation exercises, force every claim defending
      it into a two-branch assertion, and leave a token-in-a-URL path available in a workflow
      whose whole point is that the credential is auditable at the checkout step. The design
      accordingly leaves `persist-credentials` at its `actions/checkout` default of `true`
      (Implementation §3, step 1), and the prohibition here is flat so the claim that defends
      it can be flat. Using `secrets.GITHUB_TOKEN` for the push or for the
      checkout credential the push inherits remains PROHIBITED. The workflow's `permissions:`
      block MUST be exactly `contents: read` and nothing broader: the push authenticates with
      the named secret, not with the job's own `GITHUB_TOKEN`, so `contents: write` would grant
      the ambient token a capability nothing in this workflow uses.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-016
    text: >
      The workflow MUST own no derivation and no acting logic — and MUST positively WIRE the
      two halves together. Its `run:` steps may build the CLI, run `backstop pack install`,
      invoke the two pack scripts by their pack-relative paths, and pass values between them.
      The derivation script MUST be invoked EXACTLY ONCE, by its pack-relative path, with its
      stdout captured to a file under the runner's temporary directory (`${{ runner.temp }}` /
      `$RUNNER_TEMP`); capturing it into the workspace is PROHIBITED, because a workspace
      dropping is an untracked file the diff-scoped gate would then block on. The acting script
      MUST likewise be invoked by its pack-relative path, in a step that passes it (a) THAT
      captured file as the document and (b) `${{ github.event.workflow_run.head_sha }}` as the
      sha to tag — the same expression REQ-011 pins on the checkout. The `run:` steps are
      PROHIBITED from computing a version: no arithmetic on version components, no reading of
      `issue.type`, no commit classification, no tier selection, no release-note assembly, and
      no fallback version literal. DD-3's rejection of a CI-workflow home is read narrowly per
      the DD-4 correction: a hand-written workflow OWNING the logic stays rejected; a thin
      workflow step INVOKING the pack's scripts is the intended shape.
    supports: release-currency-versioning-machinery:REQ-013@2.0.0
  - id: REQ-017
    text: >
      The two halves MUST COMPOSE, and that composition MUST be proven END TO END rather than
      inferred from each half passing against hand-built fixtures. The acting half's ONLY input
      contract is the derivation's stdout: it MUST consume that byte stream verbatim, with no
      reshaping, re-keying, or intermediate transform anywhere between them, so that
      `bash derive-release-delta.sh <sha> | bash tag-from-release-delta.sh - <sha>` is a valid
      invocation. Verification MUST include at least one claim that builds a synthetic
      repository with real commits, real tags, real `issues/*.issue.md` artifacts and a real
      bare `origin`, runs the REAL derivation, pipes its ACTUAL stdout into the REAL acting
      script, and reads the resulting tag back off the remote. A field-name or JSON-shape
      divergence between the halves MUST be able to fail a test; a suite in which every acting
      claim is fed a hand-built fixture cannot detect one and is PROHIBITED as the sole proof.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0
  - id: REQ-018
    text: >
      The acting half MUST extract every field it reads through a STRUCTURAL JSON PARSE of the
      whole document — `jq` — and it is PROHIBITED to extract any field by line-oriented text
      matching (`grep`/`sed`/`awk` over the raw document) for `schema`, `head`, `last_tag`,
      `proposed.bump`, `proposed.version`, or `release_notes`. The document interpolates
      human-authored text — issue titles, commit subjects, the rendered `release_notes` — so a
      pattern-matched extractor can be fed a value by an issue whose TITLE contains something
      shaped like `"version": "v9.9.9"`, which is a corpus-authored path to tagging an
      arbitrary version. The acting half MUST verify `jq` is available before doing anything
      else and MUST FAIL CLOSED — create nothing, push nothing, exit non-zero, name the missing
      tool — when it is not; falling back to text extraction is PROHIBITED. This asymmetry with
      REQ-002 (which bans `jq` from the derivation) is deliberate: the derivation is the half a
      consumer may run on any runner, while the acting half runs only in CI, where `jq` is a
      single declared prerequisite and correctness under adversarial corpus text outranks
      portability.
    supports: release-currency-versioning-machinery:REQ-012@2.0.0

claims:
  # ── REQ-001 — home, invocation, and the two absences that keep it law-compliant ──
  - id: CLM-001
    requirement: REQ-001
    text: Both pack scripts are present and executable at the installed pack path `.backstop/packs/backstop-ai/go-distribution/scripts/`, and their absence fails the suite loudly rather than skipping it
    tests:
      - TestReleaseDelta_PackScriptsPresentAtInstalledPath
  - id: CLM-002
    requirement: REQ-001
    text: backstop.yml and backstop.lock both declare `backstop-ai/go-distribution` at the same version, and that version is strictly greater than `0.1.0`, so `backstop pack install` materializes the scripts the workflow invokes
    tests:
      - TestReleaseAutoTag_GoDistributionDeclaredInFleetAboveScriptlessVersion
  - id: CLM-003
    requirement: REQ-001
    kind: absence
    text: DENYLIST — the go-distribution manifest declares no engine binding whose command names either script, so neither is dispatched by `backstop gate`, emits SARIF, or contributes a gate finding
    tests:
      - TestReleaseAutoTag_NoGateEngineBindingForDerivation
  - id: CLM-004
    requirement: REQ-001
    kind: absence
    text: >
      DENYLIST — zero core binary changes: the CLI's registered top-level command set is
      EXACTLY the ten commands registered today — artifact, baseline, commands, completion,
      gate, help, pack, recipe, version, waiver — with no NEW release-, tag- or
      versioning-related command added and the pre-existing `version` command's registration
      unmodified
    tests:
      - TestReleaseAutoTag_RegisteredCommandSetUnchanged
  # ── REQ-002 — read-only posture and the stdout JSON contract ──
  - id: CLM-005
    requirement: REQ-002
    text: A derivation run leaves the repository's ref set byte-identical — no tag created, no ref updated
    tests:
      - TestReleaseDelta_RunLeavesRefsUnchanged
  - id: CLM-006
    requirement: REQ-002
    text: A derivation run leaves the working tree and index byte-identical, writing to no artifact file
    tests:
      - TestReleaseDelta_RunLeavesWorktreeAndIndexUnchanged
  - id: CLM-007
    requirement: REQ-002
    text: Stdout carries the JSON document and nothing else; diagnostics land on stderr, so capturing stdout alone yields a parseable document
    tests:
      - TestReleaseDelta_StdoutCarriesJSONOnlyDiagnosticsOnStderr
  - id: CLM-008
    requirement: REQ-002
    text: The emitted document carries exactly the declared key set at every level — schema, last_tag, head, delta, proposed, caveats (uncovered_commits AND unreadable_artifacts), release_notes — with `schema` equal to `release-delta/v1`
    tests:
      - TestReleaseDelta_OutputCarriesDeclaredKeySet
  - id: CLM-009
    requirement: REQ-002
    text: The `head` value is the FULL 40-character hex sha of the analyzed commit, equal to `git rev-parse` on it — never abbreviated and never symbolic
    tests:
      - TestReleaseDelta_HeadIsFullFortyCharSha
  - id: CLM-010
    requirement: REQ-002
    text: An issue title containing a double quote and a backslash, and a commit subject containing both, still yield a parseable JSON document with the characters preserved
    tests:
      - TestReleaseDelta_EscapesQuotesAndBackslashesInStrings
  - id: CLM-011
    requirement: REQ-002
    kind: absence
    text: DENYLIST — the DERIVATION script invokes no mutating git verb (tag, push, commit, checkout, switch, reset, merge, rebase, fetch, update-ref, gc) and no `jq`/`yq`/`python3`/`node` interpreter
    tests:
      - TestReleaseDelta_ScriptInvokesNoMutatingVerbAndNoNonPosixTool
  # CLM-110 is out of numeric sequence because it was added in spec_version 2.1.0, after the
  # 2.0.0 renumbering; it is grouped with the requirement it defends rather than appended.
  - id: CLM-110
    requirement: REQ-002
    text: >
      A fixture issue whose FRONTMATTER declares `type: bug` and `status: closed` but whose
      BODY carries the column-zero lines `type: enhancement` and `status: open` is read from
      its frontmatter alone — the delta tiers it PATCH (`proposed.bump` is `patch`, not
      `minor`) and still counts it as newly closed, so neither body line reaches a typed field
    tests:
      - TestReleaseDelta_BodyTextShapedLikeFrontmatterIsNotRead
  # ── REQ-003 — baseline tag selection and delta range ──
  - id: CLM-012
    requirement: REQ-003
    text: With v0.1.1 and v0.2.0 both reachable, the baseline is the HIGHEST reachable semver tag (v0.2.0), not the nearest one git describe would report
    tests:
      - TestReleaseDelta_BaselineIsHighestReachableSemverTag
  - id: CLM-013
    requirement: REQ-003
    text: A semver tag on a side branch, unreachable from the analyzed commit, is ignored for baseline selection
    tests:
      - TestReleaseDelta_UnreachableTagIgnoredForBaseline
  - id: CLM-014
    requirement: REQ-003
    text: Non-strict-semver tags — `nightly`, `v1.2`, `0.1.1`, `v1.2.3-rc1` — are ignored for baseline selection even when reachable and lexically higher
    tests:
      - TestReleaseDelta_NonSemverTagsIgnoredForBaseline
  - id: CLM-015
    requirement: REQ-003
    text: With no strict-semver tag reachable at all, the derivation fails closed — non-zero exit, no JSON on stdout, condition named on stderr
    tests:
      - TestReleaseDelta_NoReachableReleaseTagFailsClosed
  - id: CLM-016
    requirement: REQ-003
    text: delta.commit_count equals the exact size of the `<last_tag>..<analyzed commit>` range
    tests:
      - TestReleaseDelta_CommitCountMatchesRangeExactly
  - id: CLM-017
    requirement: REQ-003
    text: An empty delta (the analyzed commit IS the baseline tag) reports commit_count 0 with proposed.bump and proposed.version both null
    tests:
      - TestReleaseDelta_EmptyDeltaProposesNothing
  # ── REQ-004 — commit classification matrix, and the delta-level corpus-only gate ──
  - id: CLM-018
    requirement: REQ-004
    text: A commit touching only corpus paths is classified corpus-only and counts toward commit_count and corpus_only_commit_count
    tests:
      - TestReleaseDelta_CorpusOnlyCommitCountedAsCorpusOnly
  - id: CLM-019
    requirement: REQ-004
    text: A commit touching a path outside the corpus prefix set is classified CODE and counts toward code_commit_count
    tests:
      - TestReleaseDelta_CodeCommitClassifiedAsCode
  - id: CLM-020
    requirement: REQ-004
    text: A MIXED commit touching both corpus and non-corpus paths is classified CODE, not corpus-only
    tests:
      - TestReleaseDelta_MixedPathCommitClassifiedAsCode
  - id: CLM-021
    requirement: REQ-004
    text: A commit whose diff name-set is empty — a merge commit and an empty commit — is classified corpus-only, the direction that cannot over-version
    tests:
      - TestReleaseDelta_EmptyDiffCommitClassifiedAsCorpusOnly
  - id: CLM-022
    requirement: REQ-004
    text: A delta consisting solely of corpus-only commits that closes NO issue reports non-zero drift with proposed.bump and proposed.version both null
    tests:
      - TestReleaseDelta_CorpusOnlyDeltaReportsDriftWithNoVersion
  - id: CLM-023
    requirement: REQ-004
    text: >
      A corpus-only delta whose single commit CLOSES a `bug` issue proposes null for BOTH
      bump and version — the delta-level gate (code_commit_count == 0) overrides the PATCH
      tier REQ-005/REQ-006 would otherwise contribute, because the release artifact would be
      byte-identical to the one the baseline tag published
    tests:
      - TestReleaseDelta_CorpusOnlyDeltaClosingBugIssueProposesNull
  - id: CLM-024
    requirement: REQ-004
    text: >
      A delta of one CODE commit plus a separate corpus-only close-out commit that closes a
      `bug` issue proposes PATCH — the corpus-only commit does not suppress a tier the
      newly-closed issue independently earns, which is this repository's ordinary close-out
      shape
    tests:
      - TestReleaseDelta_CorpusOnlyCloseOutDoesNotSuppressEarnedTier
  - id: CLM-025
    requirement: REQ-004
    text: The corpus-path prefix set is declared once as data carrying the nine declared members, and editing that list alone changes classification with no other script edit
    tests:
      - TestReleaseDelta_CorpusPathPrefixesAreDeclaredData
  # ── REQ-005 — the git-derived join, exhaustive over the issue.status enum plus the window bound ──
  - id: CLM-026
    requirement: REQ-005
    text: An issue touched by a delta commit, `closed` at the analyzed commit and NOT `closed` at the baseline tag, is IN the delta and contributes its tier
    tests:
      - TestReleaseDelta_NewlyClosedIssueTouchedInRangeIsInDelta
  - id: CLM-027
    requirement: REQ-005
    text: An `open` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_OpenIssueExcludedFromDelta
  - id: CLM-028
    requirement: REQ-005
    text: A `ready` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_ReadyIssueExcludedFromDelta
  - id: CLM-029
    requirement: REQ-005
    text: An `in-progress` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_InProgressIssueExcludedFromDelta
  - id: CLM-030
    requirement: REQ-005
    text: A `blocked` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_BlockedIssueExcludedFromDelta
  - id: CLM-031
    requirement: REQ-005
    text: A `replaced` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_ReplacedIssueExcludedFromDelta
  - id: CLM-032
    requirement: REQ-005
    text: A `canceled` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_CanceledIssueExcludedFromDelta
  - id: CLM-033
    requirement: REQ-005
    text: An `obsoleted` issue touched in the delta is excluded and contributes no tier
    tests:
      - TestReleaseDelta_ObsoletedIssueExcludedFromDelta
  - id: CLM-034
    requirement: REQ-005
    text: A `closed` issue whose file no delta commit touched — closed before the baseline tag — is excluded and contributes no tier
    tests:
      - TestReleaseDelta_ClosedIssueOutsideRangeExcluded
  - id: CLM-035
    requirement: REQ-005
    text: >
      An issue ALREADY `closed` at the baseline tag whose file a delta commit merely
      re-touches (a Resolution edit) is EXCLUDED — the closing transition must fall inside the
      delta window, so an already-released issue is never tiered into a second release
    tests:
      - TestReleaseDelta_AlreadyClosedAtBaselineIsNotRetiered
  - id: CLM-036
    requirement: REQ-005
    text: An issue whose file did not EXIST at the baseline tag and is `closed` at the analyzed commit is INCLUDED — a missing baseline file reads as not-closed, not as an error
    tests:
      - TestReleaseDelta_IssueCreatedAndClosedWithinDeltaIsInDelta
  - id: CLM-037
    requirement: REQ-005
    text: >
      A delta commit that DELETED an `issues/*.issue.md` file contributes no tier, does not
      fail the derivation, and lists that path in caveats.unreadable_artifacts with reason
      `deleted-in-delta`
    tests:
      - TestReleaseDelta_DeletedIssueFileSkippedWithCaveat
  - id: CLM-038
    requirement: REQ-005
    text: An issue file that EXISTS at the analyzed commit but whose `issue.status` cannot be read fails the derivation closed — non-zero exit, no JSON — rather than being skipped as a deletion
    tests:
      - TestReleaseDelta_UnreadableStatusOnExistingIssueFailsClosed
  - id: CLM-039
    requirement: REQ-005
    text: With no unreadable artifact the caveats.unreadable_artifacts key is still present as an empty array, never omitted
    tests:
      - TestReleaseDelta_UnreadableArtifactsKeyPresentWhenEmpty
  - id: CLM-040
    requirement: REQ-005
    kind: absence
    text: DENYLIST — no `released_in:` key (or equivalent) is written to any artifact file and no artifact file changes bytes across a derivation run
    tests:
      - TestReleaseDelta_NoReleasedInFieldWritten
  # ── REQ-006 — the issue.type tier table, exhaustive over the enum plus both unmapped shapes ──
  - id: CLM-041
    requirement: REQ-006
    text: A delta issue of type `bug` contributes tier PATCH
    tests:
      - TestReleaseDelta_TypeBugContributesPatch
  - id: CLM-042
    requirement: REQ-006
    text: A delta issue of type `enhancement` contributes tier MINOR
    tests:
      - TestReleaseDelta_TypeEnhancementContributesMinor
  - id: CLM-043
    requirement: REQ-006
    text: A delta issue of type `technical-debt` contributes tier PATCH
    tests:
      - TestReleaseDelta_TypeTechnicalDebtContributesPatch
  - id: CLM-044
    requirement: REQ-006
    text: A delta issue of type `question` contributes tier PATCH
    tests:
      - TestReleaseDelta_TypeQuestionContributesPatch
  - id: CLM-045
    requirement: REQ-006
    text: A delta issue of type `policy-violation` contributes tier PATCH
    tests:
      - TestReleaseDelta_TypePolicyViolationContributesPatch
  - id: CLM-046
    requirement: REQ-006
    text: A delta issue carrying a type value outside the table fails closed — non-zero exit, no JSON, the issue id and the unmapped value named on stderr — rather than defaulting to patch
    tests:
      - TestReleaseDelta_UnmappedIssueTypeFailsClosed
  - id: CLM-047
    requirement: REQ-006
    text: A delta issue with no `issue.type` key at all fails closed the same way, rather than being skipped or defaulted
    tests:
      - TestReleaseDelta_MissingIssueTypeFailsClosed
  - id: CLM-048
    requirement: REQ-006
    text: The type-to-tier map is declared once as data with exactly the five declared entries, so retiering `technical-debt` is a table edit and nothing else
    tests:
      - TestReleaseDelta_TypeTierMapIsDeclaredData
  # ── REQ-007 — tier selection and version arithmetic ──
  - id: CLM-049
    requirement: REQ-007
    text: A delta carrying both a bug and an enhancement proposes MINOR — the highest tier present wins
    tests:
      - TestReleaseDelta_HighestTierWinsAcrossDelta
  - id: CLM-050
    requirement: REQ-007
    text: A delta carrying only patch-tier issues proposes PATCH
    tests:
      - TestReleaseDelta_AllPatchTierProposesPatch
  - id: CLM-051
    requirement: REQ-007
    text: A delta with code commits but no tier-contributing issue proposes null for both bump and version
    tests:
      - TestReleaseDelta_NoTieredIssueProposesNull
  - id: CLM-052
    requirement: REQ-007
    text: A PATCH proposal against v0.1.1 yields v0.1.2
    tests:
      - TestReleaseDelta_PatchArithmeticIncrementsPatchPosition
  - id: CLM-053
    requirement: REQ-007
    text: A MINOR proposal against v0.1.1 yields v0.2.0 — the patch position is zeroed, not carried
    tests:
      - TestReleaseDelta_MinorArithmeticZeroesLowerPositions
  # ── REQ-008 — the major refusal, both directions ──
  - id: CLM-054
    requirement: REQ-008
    text: With a pre-1.0 baseline, no combination of delta inputs — all five issue types present at once — ever yields bump `major`
    tests:
      - TestReleaseDelta_NeverProposesMajorPreOneZero
  - id: CLM-055
    requirement: REQ-008
    text: With a baseline of v1.0.0 or higher the derivation fails closed — non-zero exit, no JSON — rather than falling back to patch or minor
    tests:
      - TestReleaseDelta_PostOneZeroBaselineFailsClosed
  - id: CLM-056
    requirement: REQ-008
    kind: absence
    text: DENYLIST — the derivation script contains no code path that can emit `major` as a bump value and no bespoke breakage model
    tests:
      - TestReleaseDelta_ScriptHasNoMajorBumpPath
  # ── REQ-009 — the uncovered-commit caveat ──
  - id: CLM-057
    requirement: REQ-009
    text: A code commit whose message names an artifact id backed by a real corpus file is COVERED and is not listed as a caveat
    tests:
      - TestReleaseDelta_CodeCommitNamingRealArtifactIsCovered
  - id: CLM-058
    requirement: REQ-009
    text: A code commit whose message names no artifact id is listed individually in caveats.uncovered_commits with its short sha and subject
    tests:
      - TestReleaseDelta_UncoveredCodeCommitListedIndividually
  - id: CLM-059
    requirement: REQ-009
    text: A code commit whose message names an artifact id that no corpus file backs is UNCOVERED and listed — a typo'd id does not read as coverage
    tests:
      - TestReleaseDelta_DanglingArtifactReferenceCountsAsUncovered
  - id: CLM-060
    requirement: REQ-009
    text: Uncovered commits alongside covered newly-closed issues neither suppress the proposal nor change it — the proposal is still made from the covered issues alone
    tests:
      - TestReleaseDelta_UncoveredCommitsDoNotSuppressProposal
  - id: CLM-061
    requirement: REQ-009
    text: A delta whose only code commits are uncovered proposes null — the uncovered commits receive no implicit tier — and still succeeds, listing them
    tests:
      - TestReleaseDelta_UncoveredOnlyDeltaProposesNullAndSucceeds
  - id: CLM-062
    requirement: REQ-009
    text: With no uncovered commits the caveats.uncovered_commits key is still present as an empty array, never omitted
    tests:
      - TestReleaseDelta_UncoveredCommitsKeyPresentWhenEmpty
  # ── REQ-010 — receipts ──
  - id: CLM-063
    requirement: REQ-010
    text: A delta issue carrying delivered_by renders one note line naming its id, type, title, and that PLAN-ISSUE receipt
    tests:
      - TestReleaseDelta_NoteLineUsesDeliveredByReceipt
  - id: CLM-064
    requirement: REQ-010
    text: A delta issue carrying resolved-by but no delivered_by renders its resolved-by pointer as the receipt
    tests:
      - TestReleaseDelta_NoteLineFallsBackToResolvedByReceipt
  - id: CLM-065
    requirement: REQ-010
    text: A delta issue carrying neither pointer renders the literal `no-traceability-pointer` rather than being dropped or given invented prose
    tests:
      - TestReleaseDelta_NoteLineMarksMissingReceiptExplicitly
  - id: CLM-066
    requirement: REQ-010
    text: The note body's line count equals the delta's issue count — no line exists that no delta issue backs
    tests:
      - TestReleaseDelta_NoteLineCountEqualsIssueCount
  - id: CLM-067
    requirement: REQ-010
    text: A delta with no issues renders release_notes as the empty string
    tests:
      - TestReleaseDelta_EmptyIssueSetRendersEmptyNotes
  # ── REQ-011 — trigger confinement and the cross-file CI-name join ──
  - id: CLM-068
    requirement: REQ-011
    text: The auto-tag workflow's trigger key set is EXACTLY {workflow_run} — set equality, which is exhaustive over every trigger GitHub offers
    tests:
      - TestReleaseAutoTagWorkflow_TriggerKeySetIsExactlyWorkflowRun
  - id: CLM-069
    requirement: REQ-011
    kind: absence
    text: DENYLIST — none of push, pull_request, pull_request_target, schedule, workflow_dispatch, repository_dispatch, or a tag trigger appears in the workflow's trigger block
    tests:
      - TestReleaseAutoTagWorkflow_ProhibitedTriggersAbsent
  - id: CLM-070
    requirement: REQ-011
    text: >
      CROSS-FILE JOIN — the auto-tag workflow's `on.workflow_run.workflows` value is read from
      release-auto-tag.yml and compared against the `name:` field READ FROM ci.yml, and the two
      are equal; renaming ci.yml's `name:` fails this test rather than silently disabling
      auto-tagging (the ISSUE-109 falsifier class)
    tests:
      - TestReleaseAutoTagWorkflow_WorkflowRunNameJoinsCIWorkflowName
  - id: CLM-071
    requirement: REQ-011
    text: The workflow_run trigger declares types [completed] and branches [main]
    tests:
      - TestReleaseAutoTagWorkflow_WorkflowRunTypesAndBranches
  - id: CLM-072
    requirement: REQ-011
    text: The job is gated on workflow_run.conclusion == 'success', so a red or cancelled CI run cannot reach the tagging path
    tests:
      - TestReleaseAutoTagWorkflow_JobGatedOnSuccessfulConclusion
  - id: CLM-073
    requirement: REQ-011
    text: The job is additionally gated on workflow_run.head_branch == 'main', so a CI run on any other branch cannot reach the tagging path
    tests:
      - TestReleaseAutoTagWorkflow_JobGatedOnMainHeadBranch
  - id: CLM-074
    requirement: REQ-011
    text: The checkout pins ref to github.event.workflow_run.head_sha with fetch-depth 0 and tags fetched, so the analyzed commit is the triggering one and the baseline tag is resolvable
    tests:
      - TestReleaseAutoTagWorkflow_CheckoutPinsWorkflowRunHeadSha
  - id: CLM-075
    requirement: REQ-011
    kind: absence
    text: DENYLIST — neither github.sha nor an unpinned checkout selects the analyzed commit anywhere in the workflow
    tests:
      - TestReleaseAutoTagWorkflow_NeverSelectsCommitViaGithubSha
  # ── REQ-012 — the acting half fires, or does nothing ──
  - id: CLM-076
    requirement: REQ-012
    text: Given a document whose proposed.version is null, the acting script creates no tag, pushes nothing to the bare origin, and exits 0
    tests:
      - TestReleaseTagFromDelta_NullVersionCreatesNothingAndSucceeds
  - id: CLM-077
    requirement: REQ-012
    text: Given a valid document, the acting script creates an ANNOTATED tag at the analyzed sha whose message is the document's release_notes
    tests:
      - TestReleaseTagFromDelta_CreatesAnnotatedTagAtAnalyzedSha
  - id: CLM-078
    requirement: REQ-012
    text: The push delivers exactly one ref — refs/tags/<version> — to the bare origin, leaving every branch ref and every other tag untouched
    tests:
      - TestReleaseTagFromDelta_PushesExactlyTheOneTagRef
  - id: CLM-079
    requirement: REQ-012
    kind: absence
    text: DENYLIST — the acting script uses none of --force, -f, a leading + refspec, --tags, or --follow-tags, and pushes no branch
    tests:
      - TestReleaseTagFromDelta_ForbiddenPushFlagsAbsent
  - id: CLM-080
    requirement: REQ-012
    kind: absence
    text: DENYLIST — no approval or proposal surface exists anywhere in the workflow or the acting script — no issue or PR creation, no gh api write, no environment approval gate, no artifact write
    tests:
      - TestReleaseAutoTag_NoApprovalOrProposalSurface
  # ── REQ-013 — fail closed, one claim per named condition ──
  - id: CLM-081
    requirement: REQ-013
    text: A non-zero exit from the derivation leaves no tag created and no tag pushed, and the acting step exits non-zero
    tests:
      - TestReleaseTagFromDelta_DerivationFailureCreatesNoTag
  - id: CLM-082
    requirement: REQ-013
    text: Unparseable (non-JSON) captured output leaves no tag created and no tag pushed, and exits non-zero
    tests:
      - TestReleaseTagFromDelta_MalformedJSONCreatesNoTag
  - id: CLM-083
    requirement: REQ-013
    text: A document whose schema value is not `release-delta/v1` leaves no tag created and exits non-zero
    tests:
      - TestReleaseTagFromDelta_UnexpectedSchemaCreatesNoTag
  - id: CLM-084
    requirement: REQ-013
    text: >
      A document whose `head` differs from the sha argument the caller passed leaves no tag
      created and exits non-zero, naming both — the tag must never land on a commit the
      derivation did not analyze
    tests:
      - TestReleaseTagFromDelta_HeadShaMismatchCreatesNoTag
  - id: CLM-085
    requirement: REQ-013
    text: A non-null proposed.version that is not a strict vX.Y.Z — `0.2.0`, `v0.2`, `v0.2.0-rc1`, `latest` — leaves no tag created and exits non-zero
    tests:
      - TestReleaseTagFromDelta_NonSemverProposedVersionCreatesNoTag
  - id: CLM-086
    requirement: REQ-013
    text: A proposed.version not strictly greater than the document's own last_tag — equal, or lower — leaves no tag created and exits non-zero
    tests:
      - TestReleaseTagFromDelta_NonMonotonicVersionCreatesNoTag
  - id: CLM-087
    requirement: REQ-013
    text: >
      A proposed.version that is strictly greater but skips a position against last_tag
      `v0.1.1` — `v0.1.3`, `v0.3.0`, `v1.0.0` — leaves no tag created and exits non-zero, while
      `v0.1.2` and `v0.2.0` are accepted
    tests:
      - TestReleaseTagFromDelta_ImplausibleVersionJumpCreatesNoTag
  - id: CLM-088
    requirement: REQ-013
    kind: absence
    text: DENYLIST — no fallback or default version literal exists in the acting script, and neither the job nor any step declares continue-on-error
    tests:
      - TestReleaseAutoTag_NoFallbackVersionAndNoContinueOnError
  # ── REQ-014 — collision refusal, serialization, and the hang bound ──
  - id: CLM-089
    requirement: REQ-014
    text: When the proposed tag already exists on the remote, the acting script creates nothing, pushes nothing, exits non-zero, and names the existing tag
    tests:
      - TestReleaseTagFromDelta_ExistingRemoteTagRefusesAndNames
  - id: CLM-090
    requirement: REQ-014
    text: The remote collision probe runs BEFORE tag creation — after a refusal the local repository carries no leftover tag object either
    tests:
      - TestReleaseTagFromDelta_CollisionProbePrecedesTagCreation
  - id: CLM-091
    requirement: REQ-014
    text: The workflow's concurrency group is a fixed literal carrying no per-run expression — not github.sha, github.ref, github.run_id, or github.event.workflow_run.head_sha
    tests:
      - TestReleaseAutoTagWorkflow_ConcurrencyGroupIsFixedLiteral
  - id: CLM-092
    requirement: REQ-014
    text: The workflow declares cancel-in-progress false, so a queued run never cancels one that may already have pushed a tag
    tests:
      - TestReleaseAutoTagWorkflow_CancelInProgressIsFalse
  - id: CLM-093
    requirement: REQ-014
    text: The auto-tag job declares a finite timeout-minutes, so a hung run cannot hold the fixed concurrency group and silently block every later auto-tag
    tests:
      - TestReleaseAutoTagWorkflow_JobDeclaresFiniteTimeout
  # ── REQ-015 — the token that actually starts release.yml, proven positively ──
  - id: CLM-094
    requirement: REQ-015
    text: >
      POSITIVE — the checkout step's parsed `with:` block declares `token:` as a
      `${{ secrets.<NAME> }}` expression whose NAME is not GITHUB_TOKEN, so the credential
      persisted into .git/config (and inherited by the push) is that named secret
    tests:
      - TestReleaseAutoTagWorkflow_CheckoutDeclaresNonDefaultPushToken
  - id: CLM-095
    requirement: REQ-015
    text: POSITIVE — that same checkout step does not set persist-credentials false, so the named-secret credential it writes is the one `git push origin` actually uses
    tests:
      - TestReleaseAutoTagWorkflow_CheckoutPersistsTheNamedCredential
  - id: CLM-096
    requirement: REQ-015
    kind: absence
    text: DENYLIST — secrets.GITHUB_TOKEN appears nowhere in the workflow, for neither the push, the checkout, nor any step's env
    tests:
      - TestReleaseAutoTagWorkflow_DefaultTokenNeverAppears
  - id: CLM-097
    requirement: REQ-015
    text: >
      The workflow's permissions block is exactly `contents: read` and nothing broader — the
      push does not use the job's own GITHUB_TOKEN, so write would be an unused grant
    tests:
      - TestReleaseAutoTagWorkflow_PermissionsAreContentsReadOnly
  # ── REQ-016 — the workflow is a thin invoker, wired positively at both ends ──
  - id: CLM-098
    requirement: REQ-016
    text: The workflow invokes the derivation script exactly once, by its pack-relative path, after building the CLI and running pack install
    tests:
      - TestReleaseAutoTagWorkflow_InvokesDerivationOnceByPackPath
  - id: CLM-099
    requirement: REQ-016
    text: POSITIVE — the workflow's job steps include an invocation of tag-from-release-delta.sh by its pack-relative path
    tests:
      - TestReleaseAutoTagWorkflow_InvokesActingScriptByPackPath
  - id: CLM-100
    requirement: REQ-016
    text: POSITIVE — the acting invocation's document argument is the same capture path the derivation step wrote its stdout to, joined by parsing both steps rather than asserting either literal alone
    tests:
      - TestReleaseAutoTagWorkflow_ActingInputJoinsDerivationCapture
  - id: CLM-101
    requirement: REQ-016
    text: POSITIVE — the acting invocation's sha argument is the expression `github.event.workflow_run.head_sha`, the same one the checkout pins
    tests:
      - TestReleaseAutoTagWorkflow_ActingShaArgumentIsWorkflowRunHeadSha
  - id: CLM-102
    requirement: REQ-016
    text: The derivation's captured stdout path is under the runner temp directory (runner.temp / RUNNER_TEMP), never a workspace-relative path that would leave an untracked dropping in the gate's diff scope
    tests:
      - TestReleaseAutoTagWorkflow_CaptureLandsInRunnerTemp
  - id: CLM-103
    requirement: REQ-016
    kind: absence
    text: DENYLIST — the workflow's run steps contain no version arithmetic, no issue.type read, no commit classification, no tier vocabulary, and no release-note assembly
    tests:
      - TestReleaseAutoTagWorkflow_RunStepsCarryNoDerivationLogic
  # ── REQ-017 — the two halves compose, proven on real output ──
  - id: CLM-104
    requirement: REQ-017
    text: >
      END TO END — against a synthetic repo with real commits, tags, issue artifacts and a
      bare origin, the REAL derivation's ACTUAL stdout piped into the REAL acting script
      produces exactly one new tag on the remote, at the analyzed sha, named for the version
      the derivation proposed, with the derivation's release_notes as its annotation
    tests:
      - TestReleaseAutoTag_EndToEndDerivationPipedIntoTaggerPushesProposedTag
  - id: CLM-105
    requirement: REQ-017
    text: END TO END — the same real pipe over a corpus-only delta pushes nothing to the bare origin and exits 0
    tests:
      - TestReleaseAutoTag_EndToEndCorpusOnlyDeltaPushesNothing
  - id: CLM-106
    requirement: REQ-017
    text: END TO END — the same real pipe in a repository with no reachable semver tag pushes nothing and exits non-zero, so a derivation failure cannot be laundered into a tag by the pipe
    tests:
      - TestReleaseAutoTag_EndToEndDerivationFailurePushesNothing
  # ── REQ-018 — structural parse, not text matching ──
  - id: CLM-107
    requirement: REQ-018
    text: >
      A document whose release_notes and issue title contain adversarial prose shaped like
      `"version": "v9.9.9"` and `"schema": "release-delta/v1"` is parsed structurally — the tag
      created is the real proposed.version (v0.1.2), never v9.9.9
    tests:
      - TestReleaseTagFromDelta_AdversarialProseInTextFieldsIsNotMisparsed
  - id: CLM-108
    requirement: REQ-018
    text: With jq unavailable on PATH the acting script fails closed — creates nothing, pushes nothing, exits non-zero, names the missing tool — rather than falling back to text extraction
    tests:
      - TestReleaseTagFromDelta_MissingJqFailsClosed
  - id: CLM-109
    requirement: REQ-018
    kind: absence
    text: DENYLIST — the acting script extracts none of schema, head, last_tag, proposed.bump, proposed.version or release_notes via grep/sed/awk line matching over the raw document
    tests:
      - TestReleaseTagFromDelta_NoTextPatternFieldExtraction

contracts:
  # No entry declares a `provides:` signature, and that is REQ-001's design rather than an
  # omission: this spec adds no Go production symbol anywhere, so there is no signature for
  # the ast-grep presence probe to match. Each entry instead declares what the file CONSUMES.
  # NOTE, stated because the body calls one of these seams "load-bearing": `consumes:` entries
  # carry NO gate enforcement. ExtractContractEntries (pkg/gate/step_testverify.go:577) iterates
  # `c.Provides` and never `c.Consumes`, so every entry below is DOCUMENTARY. What actually
  # falsifies these seams is the claim set — REQ-017's end-to-end pipe above all.
  - file: .github/workflows/release-auto-tag.yml
    consumes:
      - source: .backstop/packs/backstop-ai/go-distribution/scripts
        name: derive-release-delta.sh
        kind: function
      - source: .backstop/packs/backstop-ai/go-distribution/scripts
        name: tag-from-release-delta.sh
        kind: function
      - source: .github/workflows/ci.yml
        name: workflow-name
        kind: constant
      - source: cmd/backstop
        name: NewRootCommand
        kind: function
  - file: backstop.yml
    consumes:
      - source: pkg/config
        name: Config
        kind: type
  - file: backstop.lock
    consumes:
      - source: pkg/pack/distribution
        name: LockEntry
        kind: type
  - file: .backstop/packs/backstop-ai/go-distribution/scripts/derive-release-delta.sh
    consumes:
      - source: artifacts/issue/v1
        name: issue.type
        kind: constant
      - source: artifacts/issue/v1
        name: issue.status
        kind: constant
  - file: .backstop/packs/backstop-ai/go-distribution/scripts/tag-from-release-delta.sh
    consumes:
      - source: .backstop/packs/backstop-ai/go-distribution/scripts/derive-release-delta.sh
        name: release-delta/v1
        kind: constant
---

# SPEC-066: CI Release Auto Tag

## Overview

BUNDLE-031 spent 2026-08-10 pivoting. The design this spec implements is the one that
came out the other side, at bundle v0.5.0: **no `backstop gate` involvement, no approval
step, no in-flight detection.** The founder's constraint was verbatim and load-bearing —
*"i am way too busy and overloaded to switch over to github to manually review and
approve"*, and the rule drawn from it, *"unless there's a seamless way for me to approve
or deny, then it should just be auto released."* A proposal nobody has bandwidth to read
is a queue, not a safety mechanism. So the machinery acts.

What it does: when `main` moves — chained off CI's successful completion, for a reason
below — a job computes how far `main` has moved past the last release tag, derives what
the next semver would be from the typed artifact corpus rather than from commit messages,
and, when there is anything releasable, **creates and pushes the tag itself.** The shipped
tag-triggered `release.yml` pipeline (ISSUE-087) then runs unchanged, and its existing
`require-green-ci` job is the real safety rail: an auto-pushed tag builds and publishes
only once CI has gone green on that commit, exactly as for a hand-pushed one.

Three things this spec deliberately does not contain, because the pivot removed them and
their bundle requirements are retired IDs that are not reused: an in-flight or mid-lane
readiness check of any kind (bundle REQ-008 — `main` is trusted as release-ready, full
stop; keeping partial work off `main` is branching discipline, not pipeline machinery), a
gate-side release-currency signal (bundle REQ-010), and a severity for that signal
(bundle REQ-011). Nothing here emits SARIF, declares an engine binding, or adds a gate
step.

### ISSUE-111 is SUBSUMED, not consumed

`issues/ISSUE-111-backstop-core-adopts-go-distribution-pack.issue.md` (status `open`, no
plan) asks for exactly one thing: `pack add backstop-ai/go-distribution` into
backstop-core's committed `backstop.yml` / `backstop.lock`, plus one `backstop gate --all`
run confirming the verdict does not move. That is REQ-001's fleet clause verbatim, and
this spec **subsumes it entirely** rather than consuming it as a precondition. The reason
is version arithmetic, not preference: ISSUE-111 would adopt the *currently published*
`v0.1.0`, which predates both scripts, and SPEC-066 must then immediately bump the
declaration to a version that contains them. Adopting twice is wasted motion and a
second, redundant verdict-movement check. So REQ-001 requires the declaration at a
script-carrying version, and ISSUE-111's verdict-neutrality run is folded in as a named
implementation step below.

**Intended disposition, for whoever plans this:** once SPEC-066 is implemented, ISSUE-111
should be closed as delivered-by SPEC-066 with a `## Resolution` section pointing here.
This spec does not touch that issue file; it only records the intent.

### Two design consequences, and one named deviation

**The trigger is `workflow_run`, not `push` — a deliberate deviation from bundle REQ-012's
literal text.** Bundle REQ-012 v2.0.0 says *"on every push to `main`."* This spec fires on
CI's successful completion for that push instead. That is a deviation and is named as one
rather than left implicit. It is justified by the bundle's own reasoning: DD-4's correction
locates the safety in `release.yml`'s `require-green-ci` gate, and that gate queries for a
**completed** successful `ci.yml` run for the tagged commit, failing closed when it finds
none — *"NO CI RUN — no completed ci.yml run exists for ${GITHUB_SHA}. That commit has
never been gated."* A tag pushed at push-time races `ci.yml`, loses, and becomes a tag that
can never release, with nothing to re-trigger it. Firing on push would satisfy the bundle's
words while defeating the mechanism the bundle names as where the safety lives. Chaining
off CI's completion makes the precondition structural rather than timing-dependent, and
inherits a second property for free: a red CI run never reaches the tagging path at all.
The observable behavior the bundle asks for — `main` moves, a tag follows, no human in the
loop — is unchanged.

**The acting half is a pack script, not inline workflow YAML.** Every safety property in
this spec — fail closed on malformed output, refuse a colliding tag, never force-push — is
only worth its ink if it can be FALSIFIED. Guards that exist only as `run:` text can be
asserted structurally and never executed. As a script the same guards run against a
synthetic repository with a real bare `origin`, and a test can prove that a malformed
document leaves the remote's tag set unchanged. This also keeps the workflow thin, which
is exactly what the DD-4 correction asks for: *"a hand-written workflow OWNING the
derivation logic is still rejected; a thin workflow step INVOKING the pack's script is the
intended shape."* REQ-017 closes the gap that split creates: the two halves are proven to
compose on the derivation's REAL output, not only against hand-built fixtures.

## Requirements

Requirements, claims, and bundle pins are defined in frontmatter. In summary:

| Requirement | Bundle pin | Boundary |
| --- | --- | --- |
| REQ-001 — both scripts ship in the `go-distribution` pack, invoked directly; the fleet declares it at a script-carrying version (subsuming ISSUE-111); zero core binary changes; never a gate engine binding | `REQ-013@2.0.0` | pack + `backstop.yml`/`backstop.lock` |
| REQ-002 — the derivation is read-only; one JSON document on stdout with `head` as a full 40-char sha; diagnostics on stderr; POSIX + `git` only in THIS half; every typed artifact field read ONLY from the frontmatter block, never scraped from the body | `REQ-001@1.1.0` | `derive-release-delta.sh` |
| REQ-003 — baseline is the highest REACHABLE strict-semver tag; no reachable tag fails closed | `REQ-001@1.1.0` | `derive-release-delta.sh` |
| REQ-004 — no commit ever contributes a tier by itself; the corpus-only rule binds at DELTA level (`code_commit_count == 0` ⇒ null), and a corpus-only close-out never suppresses a tier a newly-closed issue earned | `REQ-005@1.0.0` | `derive-release-delta.sh` |
| REQ-005 — the artifact↔release join is derived from git: touched in delta AND `closed` at head AND NOT `closed` at baseline; a deleted issue file is caveated, not fatal; no stored field | `REQ-006@1.0.0` | `derive-release-delta.sh` |
| REQ-006 — `issue.type` → tier via a declared data table; an unmapped or absent type fails closed | `REQ-002@1.0.0`, `REQ-003@1.0.0` | `derive-release-delta.sh` |
| REQ-007 — highest tier present wins; no tier (or REQ-004's delta gate) means `null`; lower positions zero on a minor | `REQ-002@1.0.0` | `derive-release-delta.sh` |
| REQ-008 — never a major pre-1.0; a ≥ 1.0.0 baseline fails closed | `REQ-004@1.0.0` | `derive-release-delta.sh` |
| REQ-009 — uncovered code commits are named individually, never tiered, never a refusal | `REQ-007@1.0.0` | `derive-release-delta.sh` |
| REQ-010 — notes are receipts: one line per delta issue, `delivered_by`/`resolved-by`/explicit gap | `REQ-009@1.0.0` | `derive-release-delta.sh` |
| REQ-011 — trigger is exactly `workflow_run`, naming CI by ci.yml's OWN `name:` as a cross-file join; the commit is `workflow_run.head_sha` | `REQ-012@2.0.0` | `release-auto-tag.yml` |
| REQ-012 — tag and push, or do nothing; no approval, no proposal surface, no force | `REQ-012@2.0.0` | `tag-from-release-delta.sh` |
| REQ-013 — fail closed on seven conditions: derivation error, malformed output, wrong schema, `head`≠sha argument, bad version, non-monotonic version, implausible version jump | `REQ-012@2.0.0` | `tag-from-release-delta.sh` |
| REQ-014 — refuse a colliding remote tag before creating anything; serialize on a fixed concurrency group; bound the job with `timeout-minutes` | `REQ-012@2.0.0` | script + workflow |
| REQ-015 — the checkout declares an explicit non-default `token:` (positive mechanism, not a denylist) and leaves `persist-credentials` at its default `true`, with `persist-credentials: false` flatly prohibited; `permissions: contents: read` | `REQ-012@2.0.0` | `release-auto-tag.yml` |
| REQ-016 — the workflow owns no logic and positively wires both halves: derivation once, capture to `$RUNNER_TEMP`, acting script fed that capture and `workflow_run.head_sha` | `REQ-013@2.0.0` | `release-auto-tag.yml` |
| REQ-017 — the two halves compose, proven end to end on the derivation's real stdout | `REQ-012@2.0.0` | both scripts |
| REQ-018 — the acting half parses structurally (`jq`), never by text matching; missing `jq` fails closed | `REQ-012@2.0.0` | `tag-from-release-delta.sh` |

Bundle requirements REQ-008, REQ-010 and REQ-011 were RETIRED by the 2026-08-10 pivot and
have no spec requirement, no claim, and no test here. Their IDs are retired, not reused.

### Why REQ-004 and REQ-005 do not contradict each other

An earlier draft of this spec did contradict itself, and the resolution is recorded here
because it is the single most consequential design call in the spec. REQ-004 said
corpus-only commits "must never contribute a bump tier"; REQ-005 said an issue counts when
a delta commit touched its file and it is closed. In this repository, close-out commits
routinely touch ONLY the issue file (`close-out(SPEC-018): implemented`), so a one-commit
delta satisfied both rules with opposite answers.

The resolution keeps the harm DD-1 actually names — *"it cannot justify a version whose
artifact would be byte-identical to the last one"* — and notices that this harm is a
property of the **delta**, not of an individual commit:

1. A **commit** never contributes a tier at all, of either class. Tiers come only from
   newly-closed issues (REQ-005 → REQ-006).
2. When the **whole delta** is corpus-only (`code_commit_count == 0`), the proposal is
   `null` regardless of what closed, because the built artifact would be byte-identical.
3. When the delta contains any code commit, a corpus-only close-out commit inside it does
   **not** suppress the tier its issue earned.

The rejected alternative was to require the closing commit itself to be a CODE commit.
That reading is internally consistent but empirically fatal: this repo's convention is a
separate corpus-only close-out commit, so nearly every real delta would propose `null` and
the machinery would never fire — the exact "a machine that never speaks" failure DD-2 and
DD-8 both reject. The chosen rule also has a useful side effect: work whose code shipped in
release *N* and whose close-out lands after the tag produces a corpus-only delta and
correctly proposes nothing, because the code is already released.

## Implementation

### 1. `scripts/derive-release-delta.sh` — the read-only half (pack repo)

Runs from the consumer repository root; takes the analyzed commit as its single optional
argument, defaulting to `HEAD`. Ordered passes:

1. **Baseline selection.** `git tag --merged <commit>`, filtered to strict
   `^v[0-9]+\.[0-9]+\.[0-9]+$`, version-ordered, highest wins. Not `git describe` —
   describe reports the NEAREST tag, and with two reachable releases the nearest is the
   wrong baseline. No match at all: exit non-zero, no stdout, diagnostic on stderr.
   Baseline major ≥ 1: exit non-zero the same way (REQ-008).
2. **Delta enumeration.** `git rev-list <last_tag>..<commit>` gives the commit set;
   `delta.commit_count` is its size. `head` is `git rev-parse <commit>` — the full sha.
3. **Commit classification.** For each commit, its diff name-set decides: every path under
   the declared corpus prefix list → CORPUS-ONLY; any path outside it → CODE; an EMPTY
   name-set (merge or empty commit) → CORPUS-ONLY. The prefix list is one declared array
   at the top of the file — data, not scattered `case` arms. `code_commit_count` and
   `corpus_only_commit_count` fall out of this pass.
4. **The join.** From the delta's touched paths, take every `issues/*.issue.md`. For each:
   read `issue.status` and `issue.type` from its FRONTMATTER BLOCK ONLY — the lines between
   the file's first `---` and the next `---`, isolated before any field match runs, never a
   whole-file `grep '^status:'` (REQ-002; `ISSUE-078` already carries a body line shaped like
   `status: pass`, and a file with no delimitable frontmatter fails closed) — AT the analyzed
   commit (`git show <commit>:<path>`, so a later edit outside the range cannot rewrite
   history),
   and read `issue.status` AT the baseline tag (`git show <last_tag>:<path>`). Keep it only
   when head-status is `closed` AND baseline-status is not `closed` (a path absent at the
   baseline reads as not-closed). A path absent at the ANALYZED commit — deleted inside the
   delta — is skipped and appended to `caveats.unreadable_artifacts` as
   `{path, reason: deleted-in-delta}`; a path that exists but whose status cannot be read
   exits non-zero. Nothing is written; no `released_in:` field exists anywhere.
5. **Tiering.** Each kept issue's `issue.type` resolves through the declared five-entry
   table. Absent or unmapped: exit non-zero, no stdout, the issue id and value named.
6. **Selection and arithmetic.** Highest tier present, PATCH < MINOR. If
   `code_commit_count == 0`, or no tier is present, both `proposed` fields are `null`
   (REQ-004's delta gate is applied HERE, after tiering, so the counts stay honest).
   Otherwise increment the selected position on `last_tag` and zero every lower one.
7. **Coverage caveats.** Every CODE commit whose message names no
   `(ISSUE|PLAN-ISSUE|SPEC|BUNDLE|DIR)-[0-9]{3}` backed by a real corpus file at the
   analyzed commit is appended to `caveats.uncovered_commits` as `{sha, subject}`. Both
   caveat keys are always present, empty arrays included.
8. **Receipts.** One `release_notes` line per kept issue: id, type, title, and
   `delivered_by` → else `resolved-by` → else the literal `no-traceability-pointer`. Title
   and both receipt pointers come from the SAME isolated frontmatter block pass 4 used — a
   body line shaped like `delivered_by: …` is prose and is never read.
9. **Emission.** One JSON document to stdout with the declared key set, every interpolated
   string JSON-escaped. Diagnostics have gone to stderr throughout.

### 2. `scripts/tag-from-release-delta.sh` — the acting half (pack repo)

Takes the captured document (path or `-` for stdin) and the analyzed sha. It parses the
document ONCE through `jq` (REQ-018) after verifying `jq` exists, then applies ordered
checks — every one of them creates and pushes nothing; all but the fifth exit non-zero,
and the fifth is the quiet success path:

0. `jq` not on PATH → refuse, naming the missing tool.
1. Derivation exit status non-zero (the caller passes it through, or the document is absent).
2. Document not parseable as JSON.
3. `schema` != `release-delta/v1`.
4. `head` != the sha argument. The tag must land on the commit that was ANALYZED; a
   divergence means the document and the world disagree about what is being released.
5. `proposed.version` null → **exit 0**, having created and pushed nothing. This is the
   quiet, common path.
6. `proposed.version` non-null but not a strict `v<major>.<minor>.<patch>`.
7. `proposed.version` not strictly greater than the document's own `last_tag`.
8. `proposed.version` not exactly one semver position above `last_tag` with lower positions
   zeroed (`v0.1.1` → only `v0.1.2` or `v0.2.0`).
9. `git ls-remote --tags origin refs/tags/<version>` returns a match — refuse, naming the
   existing tag. **This probe runs before any tag object is created**, so a refusal leaves
   no local leftover either.

Only past all of these, with a non-null version: `git tag -a <version> <sha> -m
"<release_notes>"` then `git push origin refs/tags/<version>`. One ref, no flags. The
annotated message is where the receipts land durably — a tag object in git, not a line in
a log.

### 3. `.github/workflows/release-auto-tag.yml` — the thin invoker (backstop-core)

```
on:
  workflow_run:
    workflows: [CI]            # MUST equal ci.yml's own `name:` — joined by CLM-070
    types: [completed]
    branches: [main]
permissions:
  contents: read               # the push uses the named secret, not this token
concurrency:
  group: release-auto-tag      # FIXED literal — no per-run expression
  cancel-in-progress: false
jobs:
  auto-tag:
    timeout-minutes: 15        # finite, so a hang cannot hold the group forever
    if: >-
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.head_branch == 'main'
```

Steps, in order:

1. `actions/checkout` with `ref: ${{ github.event.workflow_run.head_sha }}`,
   `fetch-depth: 0`, tags fetched, and `token: ${{ secrets.<RELEASE_TAG_TOKEN_NAME> }}` —
   the explicit non-default credential REQ-015 mandates. `persist-credentials` is left at
   its default `true`, which is what makes that named secret the credential the later
   `git push origin` inherits.
2. `actions/setup-go` from `go.mod`.
3. `go build -o ./bin/backstop ./cmd/backstop`.
4. `./bin/backstop pack install`.
5. Run the derivation script by its pack-relative path, exactly once, redirecting stdout to
   `${{ runner.temp }}/release-delta.json`. Not the workspace: the diff-scoped gate counts
   untracked workspace files, and `TestCIWorkflow_LeavesNoUngitignoredDroppings`
   (`workflows_test.go`) exists because CI already produced that failure once.
6. Run the acting script by its pack-relative path with that temp file and
   `${{ github.event.workflow_run.head_sha }}`.

Nothing else. No version arithmetic, no `issue.type`, no tier vocabulary, no fallback
literal.

### 4. Fleet declaration (backstop-core) — and ISSUE-111's verdict check

`backstop.yml` gains `backstop-ai/go-distribution: <version>` and `backstop.lock` gains
the matching entry, at a pack version that contains both new scripts — which means the
pack repo cuts and TAGS a release (its current published version is `v0.1.0`, which
predates them). `backstop pack install` then materializes the scripts at the path both the
workflow and the tests read.

Because this subsumes ISSUE-111, its acceptance step comes along: **one `backstop gate
--all` run after the declaration lands, confirming the verdict does not move** — that
`go-distribution` reports zero findings against core's shipped release trinity and the
other packs' finding counts are unchanged, consistent with the 2026-07-29 pre-adoption
measurement recorded in ISSUE-111. That run is an implementation step and a Review
Question, not a claim: it is a gate invocation, not a Go test, and mandating it as a test
would mean shelling `backstop gate --all` from inside the test suite.

### 5. Contracts

The `contracts:` block declares five FILES and no `provides:` signature on any of them,
which is REQ-001's design rather than an omission: this spec adds no Go production symbol,
so there is nothing for the ast-grep signature probe to match. What each entry declares
instead is what the file CONSUMES — the workflow consumes the two pack scripts, the CLI it
builds, and `ci.yml`'s `name:`; `backstop.yml` and `backstop.lock` consume the config and
lock-entry shapes that make `pack install` materialize those scripts; the derivation
consumes the `issue.type` and `issue.status` fields of the issue schema; and the acting
half consumes the `release-delta/v1` document the derivation emits.

**That last entry is the load-bearing seam of the capability, and the contracts block does
NOT enforce it.** Stated plainly so the word "contract" does not imply a guarantee the
mechanism does not provide: `ExtractContractEntries` (`pkg/gate/step_testverify.go:577`)
iterates `c.Provides` and never touches `c.Consumes`, so every entry above is DOCUMENTARY —
the gate will not notice if the derivation renames a field the acting half reads. What
actually falsifies that seam is REQ-017's end-to-end claim, which runs the real derivation
into the real acting script. The contracts block records the seam; the claim defends it.

### 6. Out of scope, by name

Fleet-wide currency reporting for the pack repos (DD-6, an explicit follow-on — packs have
no artifact corpus to derive from); pack↔core version compatibility (DD-5, BUNDLE-020's);
how a release is BUILT and PUBLISHED downstream of the tag (DIR-001 / ISSUE-087, shipped
and untouched); a post-1.0 breakage model (REQ-008 refuses instead — see Sharp Edges); and
a better mapping for the three `issue.type` members REQ-006 defaults to PATCH (pinned as
data and explicitly refinable, deliberately not re-guessed here).

## Verification

Verification configuration is in frontmatter: integration level, an 80% coverage threshold,
and `go test ./cmd/backstop/`.

**Every mandated test lives in `cmd/backstop`.** An earlier revision of this spec justified
that with a mechanical claim that is FALSE, and the false claim is corrected here rather
than quietly dropped. It asserted that "no subject value can colocate a root-package test."
A subject value can. Colocation is a directory-leaf comparison on both sides:
`testFileColocatedWithTarget` (`cmd/backstop/gate.go:1209`) takes
`filepath.Base(filepath.Dir(<test file path>))`, and `TargetPackageName`
(`pkg/gate/substantiveness_join.go:46`) takes `filepath.Base(<subject>)`. The test-file paths
come from `filepath.Walk(codeDir, …)` in `collectTestFuncNamesScoped`
(`pkg/gate/step_testverify.go:460`, storing the walked `path` at `:487`), where `codeDir` is
the project root `runGate` derives as `filepath.Dir(cfgPath)` (`cmd/backstop/gate.go:79`)
from `config.DiscoverConfigPath()` — which resolves through `os.Getwd()` and
`filepath.Abs(startDir)` (`pkg/config/config.go:153`). Those paths are absolute, so a
root-package test file's directory leaf is this repository's own directory name,
`backstop-core`, identically no matter which directory the gate was invoked from. The subject
that would colocate a root-package test is therefore not `"."` — whose leaf is `"."`, which
matches nothing the walk can produce — it is the literal string `backstop-core`.

Which is the real reason the root is the wrong home, and a stronger one than the false claim
it replaces:

- **A root subject would have to be the repository's DIRECTORY NAME.** `backstop-core` is not
  a package path, not the module path, and not a value the corpus records anywhere else — it
  is merely whatever directory this repository happens to be cloned into. Clone it as
  `backstop`, or check it out into a worktree directory named anything else, and every
  root-package claim stops colocating and the noTarget decision table
  (`pkg/gate/substantiveness_join.go`) turns each one into a violation, on a checkout where
  nothing about the code changed. No subject anywhere in this corpus depends on a checkout's
  directory name, and a first-of-its-kind release mechanism is not the place to introduce the
  first one.
- **Corpus consistency.** Of the 21 specs at `status: implemented`, the nine that declare a
  top-level `implementation.subject` all name a real package directory — `pkg/check`,
  `pkg/waiver`, `pkg/validate` (×2), `pkg/gate`, `pkg/recipe`, `pkg/pack/distribution` (×3) —
  and every `subject:` value anywhere in `specs/`, claim-level included, is one of
  `cmd/backstop`, `pkg/gate`, `pkg/pack`, `pkg/pack/distribution`, `pkg/validate`,
  `pkg/waiver`, `pkg/recipe`, `pkg/packval`, `pkg/check`. None is the repository root.
  `cmd/backstop` is also SPEC-047's precedent for a spec whose primary deliverable lives in
  an external pack.
- **Coverage.** The root package's only production code is the `embed.go` shim (its `go:embed`
  vars exist there solely because `go:embed` cannot reach above its own package). Declaring a
  coverage threshold against a package that exists for that mechanical reason means little;
  `cmd/backstop` has a real spec-derived floor of 80 (SPEC-047's).

**The cost of that choice, stated rather than hidden:** the repository root already carries
~1,100 lines of workflow-YAML test infrastructure (`workflows_test.go` — the
`workflowFile`/`workflowJob`/`workflowStep` decoders, `loadWorkflow`, `workflowTriggers`,
`allWorkflowSteps`, `readWorkflowSource`, `leadingCommentBlock`, `stepScript`,
`stepMentions`, `anyStep`, `findJob`), and every existing workflow-shape test for `ci.yml`,
`release.yml` and `tag-integrity.yml` lives there. `cmd/backstop` reuses none of it — no
test under `cmd/backstop` reads `.github/workflows` at all today (verified 2026-08-10). So
this spec's workflow-shape claims will need their own YAML decoding in `cmd/backstop`.
Reuse across the boundary is not available: those helpers are UNEXPORTED members of the
external test package `backstopcore_test`, so no other package can import them; "reusing"
them means hosting these tests at the root, which buys the fragility above. The plan should
therefore put the decoding in ONE named helper file under `cmd/backstop` (a deliberate,
localized second decoder) rather than scattering `yaml.Unmarshal` calls across test files,
and a later consolidation into a shared internal test-helper package is the honest way to
retire the duplication.

`level: integration` with an 80 threshold is deliberate on both counts. The work is
integration-shaped — real scripts executed against real git repositories and a real bare
remote — and `cmd/backstop`'s coverage floor is the MAX over the implemented specs naming
it, currently 80 (SPEC-047). Declaring 90 here would silently raise the floor for the whole
package while this spec adds no production Go to it.

The script suites build their fixtures rather than mocking git: `git init` in `t.TempDir()`,
real commits with real messages, real tags, real `issues/*.issue.md` frontmatter, and for
the acting half a `git init --bare` repository wired as `origin`. Every failure claim is
proven by executing the real script and then reading the remote's tag set — a claim that
"no tag was created" is only worth making against a remote that could have received one.
REQ-017's end-to-end claims additionally forbid the fixture-only shape for the acting half:
at least one path through the suite must be the real derivation's real stdout. The pack
scripts are read from the installed pack path; when the pack is absent the helper FAILS with
a message naming `backstop pack install`, never skips. A skip here would make the entire
suite vacuous exactly when the fleet is misconfigured.

## Sharp Edges

- **Once `v1.0.0` exists, this job goes RED on every push — permanently — until something
  replaces it.** REQ-008 fails the derivation closed at a `≥ 1.0.0` baseline, and the
  workflow has no post-1.0 model to fall back to, so the auto-tag channel becomes a standing
  red. That is INTENDED, and it is survivable for one specific reason: REQ-008 also
  guarantees the machinery can never itself produce a `v1.0.0` tag, so the red state is
  always the direct consequence of a deliberate founder act (hand-cutting 1.0), never a
  surprise. It is still a real operational cliff and should not be discovered on the day.
  The disposition when 1.0 approaches is a founder call between three options — a successor
  spec that adds a post-1.0 breakage model, disabling this workflow at the moment 1.0 is
  cut, or accepting a red channel — and this spec deliberately does not pre-empt it. What
  this spec owes is that the failure is LOUD and self-explaining (the diagnostic must name
  the baseline and the missing post-1.0 model), never a silent exit-0.

- **A tag pushed with the default `GITHUB_TOKEN` never starts `release.yml`, and a denylist
  cannot prove you avoided it.** GitHub suppresses workflow events for refs pushed with the
  default token. The trap is that `actions/checkout@v4` defaults to
  `persist-credentials: true` and writes the AMBIENT `GITHUB_TOKEN` into `.git/config` as
  origin's credential — so a workflow in which the string `secrets.GITHUB_TOKEN` appears
  NOWHERE still pushes with it, and a string-absence test passes green on a tag that will
  never release. REQ-015 therefore mandates a POSITIVE mechanism (an explicit `token:` on
  the checkout naming a non-default secret) and keeps the denylist only as a secondary.
  The implementing plan owes the secret's provisioning as a stated step, not a follow-up;
  note that `release.yml` already carries an analogous unprovisioned-secret hazard with
  `HOMEBREW_TAP_TOKEN`, so this is the second instance of the same shape in this pipeline.

- **`github.sha` is the WRONG commit under `workflow_run`.** In a `workflow_run` event
  `github.sha` and the checkout default resolve to the head of the default branch at run
  time, not to the commit whose CI run triggered it. Under any burst of pushes those
  differ, and using the wrong one would tag a commit whose CI has not concluded — which
  `require-green-ci` then refuses, producing exactly the never-releases tag this design
  exists to avoid. Only `github.event.workflow_run.head_sha` is correct. Two independent
  guards exist because a denylist alone is not proof: the wrong value is pinned OUT by an
  absence claim, and the right value is pinned IN positively on both the checkout and the
  acting invocation — and REQ-013's `head`↔sha check catches a divergence at RUN time even
  if both static guards were somehow satisfied.

- **`workflow_run` workflows only run from the default branch's copy of the file.** A
  change to `release-auto-tag.yml` cannot be exercised on a branch; it takes effect when it
  lands on `main`. The first real execution is therefore also the first integration test,
  which is why every guard is proven against a synthetic repository beforehand and why the
  first landing deserves a watched run.

- **A cross-file trigger join is only as good as the join.** `workflow_run.workflows` matches
  on the target workflow's `name:` field, which lives in a different file. A test asserting
  the literal `"CI"` would keep passing after someone renames `ci.yml`'s `name:`, and
  auto-tagging would stop forever with nothing red. This repo has already met that defect
  class (ISSUE-109, the goreleaser `.Env` cross-file falsifier), which is why CLM-070 reads
  BOTH files and compares rather than asserting a literal on one side.

- **Out-of-order CI completion can produce a legitimate collision.** Push A then push B in
  quick succession; if B's CI concludes first, B is tagged `v0.1.2`, and A's run then
  derives from `v0.1.1` (the only tag reachable from A) and proposes `v0.1.2` too.
  REQ-014's remote probe refuses, loudly and with nothing created. That refusal is a JOB
  FAILURE by design rather than a silent exit-0: a proposal colliding with an existing tag
  means the derivation and the world disagree, which is worth a human look even though no
  damage occurred. Expect it to fire occasionally; do not "fix" it by downgrading to a
  silent no-op.

- **The fixed concurrency group is a single-lane channel, so a hang blocks everything.**
  `cancel-in-progress: false` on a literal group is exactly right for tag safety and exactly
  wrong for liveness: one run stuck on a network call holds the lane and every later push
  queues behind it silently. REQ-014's `timeout-minutes` is the bound. Equally,
  `concurrency: group: ${{ github.sha }}` reads as reasonable and is the defect it looks
  like a fix for — it lets two runs for two commits derive and push simultaneously, which is
  the double-tag race — and `cancel-in-progress: true` can cancel a run between `git tag`
  and `git push`. All three are pinned by claims.

- **The delta-level corpus-only gate under-versions by construction in one case.** If a
  release's entire code content shipped in delta *N* but its issue only closes in delta
  *N+1* (a corpus-only delta), *N+1* proposes nothing. That is correct — the code is already
  released — but it means the release NOTES for that issue never appear in any tag's
  annotation, because *N* did not yet see it closed. The receipts have a hole exactly where
  close-out lags the code across a tag boundary. The direction is chosen so the error can
  only ever be a missing note, never a wrong version.

- **Adding `go-distribution` to the fleet makes backstop-core's release trinity
  gate-enforced for the first time.** Its semgrep rules assert release-workflow and
  goreleaser invariants, and backstop-core's own `release.yml` / `.goreleaser.yml` /
  `.gitignore` have never been scanned by them. Measured 2026-08-10 rather than assumed:
  the pack's `valid/` fixtures are byte copies of backstop-core's shipped files, so the
  expectation is green — but if any drift has crept in since, the gate reds on landing.
  Measured too: every rule anchors on the presence of `goreleaser/goreleaser-action`
  (`rules/workflow/release-workflow.yml`), so the new auto-tag workflow — which runs no
  goreleaser — is outside every anchor and cannot trip the `non-tag-trigger` rule despite
  carrying `branches:` under its `workflow_run` block. If a rule does fire on the new
  workflow, the fix is a PACK fix with a version bump, never a waiver and never a code
  scar.

- **The commit↔artifact association is conventional, and REQ-009 is the only thing that
  says so out loud.** DD-7 accepted this cost explicitly: coverage is decided by whether a
  commit message names an artifact id that a real file backs. A code commit with a
  well-formed message naming a real artifact is treated as covered even if the association
  is nonsense, and a genuinely covered commit with a sloppy message reads as uncovered.
  The uncovered list is the honesty channel, not a correctness guarantee — and it must
  never become an error path, or the machinery stops firing in a repository that has fix-up
  commits.

- **The corpus is an INPUT to a pipeline that pushes public tags, so corpus text is
  attacker-adjacent.** Issue titles and commit subjects flow into the JSON document and from
  there into a tag annotation. A text-matching field extractor could be steered by an issue
  titled something shaped like `"version": "v9.9.9"`. REQ-018 closes that on the ACTING side
  with a structural `jq` parse. The DERIVATION side cannot use the same instrument — REQ-002
  bans `jq`/`yq`/`python3` there for portability, so it is the half that genuinely reads
  corpus text with `grep`/`sed`/`awk` — and REQ-002 therefore closes it the only way a POSIX
  reader can: by BOUNDING every typed read to the frontmatter block between the first two
  `---` lines, so a body line shaped like `type: enhancement` is prose and never a field.
  That is not hypothetical shaping: `issues/ISSUE-078-…issue.md:58` already carries a
  column-zero `status: pass` inside a fenced transcript. The general lesson outlives both
  requirements: any future consumer of the document must parse it, never scrape it, and any
  future reader of the corpus must bound its reads to the frontmatter.

- **An unmapped `issue.type` stops all releasing until someone edits the table.** REQ-006
  fails closed on a type outside the five-entry map, which means the day the `issue.type`
  enum grows, auto-tagging stops. That is the intended trade — the alternative silently
  under-versions a public release — but it is a real operational coupling between an
  artifact schema change and the release pipeline, and whoever grows the enum owns the pack
  bump.

- **Deleting an issue artifact is caveated, not fatal — and that is a deliberate asymmetry
  with REQ-006.** An unmapped type halts releasing; a deleted issue file does not. The
  difference is what the two states mean: an unmapped type means the derivation cannot know
  the tier of something it must tier, while a deleted file means there is nothing to tier
  and the delta is merely less informative. Artifact deletion is routine here (three specs
  were deleted on 2026-08-10 alone), so failing closed on it would halt releasing for a
  normal corpus operation. The caveat is what keeps it from being silent.

- **The tests depend on an installed pack, and that dependency is deliberate.**
  `.backstop/packs/` is gitignored, so a fresh clone that has not run `backstop pack
  install` cannot execute the script suites. They FAIL rather than skip. CI already installs
  the fleet before gating, so this is a local-workflow cost, taken because a skipping suite
  would report green on a repository where the deliverable is not present at all.

- **Empty-diff commits are classified corpus-only, which is a deliberate under-count.** A
  merge commit reports no paths under `--name-only`, so it cannot be shown to carry code and
  is treated as non-bumping. In a repository that commits directly to `main` this is nearly
  vacuous; in a consumer repository that merges, a merge commit carrying the only record of
  a change would be missed. The direction is chosen so the error can only ever under-version
  (a missed release is recoverable; a wrong public release is not), and the uncovered-commit
  caveat is what surfaces the shortfall.

## Review Questions

- Does the derivation script contain any path — including error paths and defaults — by
  which `proposed.bump` can be emitted as `major`, or by which an unmapped `issue.type`
  reaches the tier selector without exiting? Grep both, do not reason about them.
- Is REQ-004's delta-level gate applied AFTER tiering, so `delta.code_commit_count` and
  `delta.closed_issues` stay honest while `proposed` goes null? A gate applied earlier would
  make a corpus-only delta report zero closed issues, which is a different (and wrong) claim
  about the world.
- Does the join read `issue.status` at BOTH the analyzed commit and the baseline tag? A
  single read at head satisfies most tests while silently re-tiering already-released issues
  the first time someone edits a closed issue file.
- Is the collision probe genuinely BEFORE `git tag`, or merely before `git push`? Confirm
  by running the acting script against a remote that already holds the tag and asserting
  the LOCAL tag list is also unchanged.
- Does any test assert "no tag was created" against a repository that had no possible way to
  create one? Every fail-closed claim must be exercised against a wired bare `origin`,
  otherwise it passes vacuously.
- Is there at least one test in which the acting script's input is the derivation's REAL
  stdout rather than a hand-built fixture? If every acting test is fixture-fed, a field-name
  divergence between the halves ships green.
- Does the acting script read ANY document field through `grep`/`sed`/`awk` on the raw text,
  including "just" the schema check before jq runs? One text-matched field is enough to
  reopen the injection path REQ-018 closes.
- Does the workflow's tag-push step reference `secrets.GITHUB_TOKEN` anywhere — including
  indirectly, through a checkout whose credential the push inherits? And does the checkout
  step POSITIVELY declare `token:` with a non-default secret, rather than merely omitting the
  default?
- Is the `permissions:` block `contents: read`? If someone "fixed" it to `write`, ask what
  uses it — the push authenticates with the named secret, so `write` is an unused grant.
- Is the concurrency group a literal? Any `${{ ... }}` in it, even one that looks constant,
  reintroduces the double-tag race. And does the job declare a finite `timeout-minutes`?
- Is the derivation's captured stdout written under `$RUNNER_TEMP`? A workspace capture is an
  untracked file in the gate's diff scope — the failure `TestCIWorkflow_LeavesNoUngitignoredDroppings`
  exists to prevent.
- Does the derivation read `issue.status` and `issue.type` from the artifact AS OF the
  analyzed commit, or from the working tree? The latter would let an edit landing after the
  analyzed commit change what a release claims to contain.
- Are the corpus-path prefix list and the type→tier map each declared exactly once, such
  that changing them requires no other edit? Two copies of either is the drift that makes
  REQ-004's and REQ-006's "as data" clauses meaningless.
- Was the ISSUE-111 verdict-neutrality run (`backstop gate --all` after the fleet
  declaration lands) actually EXECUTED and its output read, not assumed from the 2026-07-29
  measurement? And was ISSUE-111 given its `## Resolution` disposition?
- Does any mandated test live outside `cmd/backstop`? A claim whose tests straddle packages
  has no satisfiable `subject:` and detonates only at closure.
- No `follows:` binding appears on any requirement in this spec. The in-repo `standards/`
  tree carries no rules (only `.gitkeep` placeholders), and no shell or GitHub-Actions
  standard exists to bind to; per the spec-authoring contract's escalation-over-guessing rule
  (DD-13 of SPEC-004's authoring design decisions — NOT one of BUNDLE-031's DD-1..DD-8, which
  stop at DD-8) this is escalated rather than filled with an invented mapping. If a standards
  pack covering shell or CI configuration is adopted before implementation, these
  requirements should be re-bound to it.

## References

- **BUNDLE-031 (Release Currency Versioning Machinery)**, v0.5.0 `defined` — the source.
  Read the dated corrections on DD-2 and DD-4 and the v0.5.0 Version History entry before
  this spec; the pre-pivot text elsewhere in the bundle is preserved and stale by design.
  Note the one named deviation from its literal text (bundle REQ-012's "on every push to
  `main`" → `workflow_run` on CI completion), justified in the Overview.
- **ISSUE-111 (Backstop Core Adopts Go Distribution Pack)** — status `open`, no plan.
  **SUBSUMED by REQ-001**, which delivers its entire Scope section (the committed
  `backstop.yml` / `backstop.lock` declaration plus one verdict-neutrality gate run) at a
  script-carrying pack version rather than at the scriptless `v0.1.0` it names. Intended
  disposition: close ISSUE-111 as delivered-by SPEC-066 once this is implemented. This spec
  does not edit that file.
- **ISSUE-087 / PLAN-ISSUE-087 (CI-Driven Release Pipeline)** — the shipped tag-triggered
  pipeline this sits on top of. `release.yml`'s `require-green-ci` job is the safety rail
  REQ-011's trigger choice exists to keep satisfiable; nothing downstream of the tag is
  reopened.
- **ISSUE-101 (Go Distribution Pack)** — the pack this ships into, and the reason DD-3 put
  it there: every check and every script comes from a pack, never from the binary.
- **ISSUE-109 (Goreleaser Derived Env Cross File Falsifier)** — the precedent for the
  cross-file falsifier class REQ-011's CI-name join guards against: an invariant that spans
  two files and is asserted against a literal in one of them is not asserted at all.
- **SPEC-047 (Bun Toolchain Pack And Two-Surface Proof)** — the precedent for a spec whose
  primary deliverable lives in an external pack: `cmd/backstop` as subject, in-repo proof of
  the pack's real scripts, contract entries with empty `provides:`. Also the source of
  `cmd/backstop`'s current 80 coverage floor.
- **DIR-001 (Release Workflow)** — owns how a release is built and published; this spec owns
  only whether and when one is cut, and at what number.

## Version History

- **2.1.0** (2026-08-10) — Second-review revision, four fixes. Replaced the Verification
  section's REPLACEMENT justification, which was itself false: root colocation is NOT
  invocation-directory-dependent (`DiscoverConfigPath` resolves through `os.Getwd()` and
  `filepath.Abs`, so the project root is absolute from any invocation directory) — the true
  and stronger objection is that a root subject would have to be the repository's DIRECTORY
  NAME, `backstop-core`, which breaks on any differently-named clone; corrected two citations
  in that passage (`cmd/backstop/gate.go:79`, `pkg/gate/substantiveness_join.go:46`) and the
  implemented-spec count (21, of which nine declare a top-level subject). Extended REQ-002 to
  BOUND every typed artifact read to the frontmatter block — the derivation is the half that
  actually scrapes corpus text, since REQ-018's `jq` is banned there — and added CLM-110 to
  falsify it. Narrowed REQ-015 by DELETING the `persist-credentials: false` escape hatch the
  design never uses, so CLM-095's flat assertion matches the requirement exactly.
- **2.0.0** (2026-08-10) — Review-driven revision. Resolved the REQ-004/REQ-005
  corpus-only-close-out contradiction at delta level; bounded the join to the delta window
  and gave deleted artifacts a caveat rather than a failure; replaced REQ-015's denylist-only
  token mitigation with a positive checkout-`token:` mechanism and dropped `permissions:` to
  `contents: read`; added the `head`↔sha identity check and a one-position version-jump bound
  to REQ-013; added `timeout-minutes` to REQ-014; added positive acting-script wiring and a
  `$RUNNER_TEMP` capture to REQ-016; added REQ-017 (end-to-end composition on real
  derivation output) and REQ-018 (structural JSON parse, no text matching); made the CI-name
  trigger a cross-file join; corrected CLM-004's premise (`version` is an existing command);
  disclosed and subsumed ISSUE-111; corrected the Verification section's false claim about
  root-package colocation; named the `workflow_run` deviation from bundle REQ-012 explicitly.
  Claims renumbered sequentially from CLM-001; the live set is the frontmatter's.
- **1.0.0** (2026-08-10) — Initial spec, authored against BUNDLE-031 v0.5.0 (post-pivot).
  Covers bundle REQ-001..007, REQ-009, REQ-012, REQ-013; bundle REQ-008/010/011 are retired
  and carry no requirement, claim, or test here.
