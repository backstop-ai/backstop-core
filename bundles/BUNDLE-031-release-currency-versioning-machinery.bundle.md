---
title: "Release Currency Versioning Machinery"
number: BUNDLE-031
created: "2026-07-29"
schema_version: bundle/v2

bundle:
  name: release-currency-versioning-machinery
  version: "0.5.0"
  created: "2026-07-29"
  updated: "2026-08-10"
  category: infrastructure

status:
  maturity: defined

problem:
  summary: >
    ISSUE-087 delivered a real, tag-driven release pipeline (goreleaser, tag-integrity gate,
    Homebrew formula publication, version stamping) and backstop-core has now cut two releases
    through it — `v0.1.0` and `v0.1.1`. But the pipeline's TRIGGER is a human remembering to push
    a tag, and its VERSION is a human deciding a number. The founder's framing (2026-07-29,
    verbatim): manual tag-driven releasing "goes directly against my 'eliminate cognitive
    overhead' principle. me having to remember whether we're tagging the latest main for a
    release and at what semver is going to be very easy to let slip without some kind of
    machinery forcing that decision." Nothing anywhere computes or surfaces release currency: no
    auto-versioning exists in this repo or the pack fleet, nothing reports how far `main` has
    drifted from the last tag, and nothing derives what the next semver SHOULD be. The failure
    mode is not a loud one — it is drift by omission: work ships to `main`, no tag follows, and
    consumers sit on a stale binary while `main` accumulates fixes nobody released. The release
    DECISION is a human judgment and must stay one; the REMEMBERING and the COMPUTING are
    machinery's job and are currently nobody's.

  user_story: >
    As the founder releasing backstop-core (and, eventually, a fleet of pack repos that version
    independently), I want machinery that CONTINUOUSLY tells me how far `main` has moved past the
    last release and what the next version would be if I cut one now — computed from the artifact
    corpus rather than from commit-message vibes — so that letting a release slip requires
    IGNORING a standing signal rather than merely forgetting. I want to fire the release with one
    word against a proposal I did not have to assemble, and I want the machinery to know when
    `main` is mid-lane and NOT release-ready, so a partially-landed change set cannot be shipped
    to consumers as if it were complete.
    CORRECTION (2026-08-10, v0.5.0, founder-ruled): the last two clauses above are SUPERSEDED and
    kept for the lineage. There is no proposal to fire with a word — I have no bandwidth for an
    approval step, so it auto-releases — and the machinery does NOT know or check whether `main` is
    mid-lane; keeping partial work off `main` is my branching discipline, not the pipeline's job.
    What survives unchanged is the first clause: I want the drift computed and the next version
    derived from the corpus rather than remembered. See the DD-2 and DD-4 corrections.

solution:
  approach: >
    DECIDED IN DIRECTION (founder ruling, 2026-08-10; PIVOTED the same day — see the v0.5.0 Version
    History entry, and the dated corrections on DD-2 and DD-4, which this field now reflects).
    LEDGER-DERIVED SEMVER, computed by a script in the `go-distribution` PACK, invoked by a CI job
    in backstop-core's own workflows, which AUTO-TAGS AND PUSHES when the delta is non-empty. This
    is a PURE CI-TIME mechanism with NO `backstop gate` involvement, NO human approval step, and NO
    in-flight detection of any kind. Concretely: (1) the bump derives from the typed artifact corpus
    rather than commit messages — `issue.type: bug` → patch, `issue.type: enhancement` → minor, no
    auto-major before 1.0, with `technical-debt`/`question`/`policy-violation` defaulting to patch as
    an acknowledged narrow first cut (DD-1); corpus-only commits count toward the reported delta but
    never trigger a bump. (2) The derivation ships in the `go-distribution` pack (DD-3, unchanged) —
    but as a plain SCRIPT invoked directly by a workflow step (`bash
    .backstop/packs/backstop-ai/go-distribution/scripts/<script>.sh`), NOT as a gate-dispatched
    engine binding: nothing consumes its output as a gate finding any more, so no SARIF wrapping and
    no engine binding are required. ZERO core binary changes still holds, and a core command is still
    rejected. (3) On EVERY PUSH TO `main`, that CI job computes the delta; if the delta is non-empty
    it computes the proposed semver and TAGS AND PUSHES the tag itself. The existing tag-triggered
    `release.yml` pipeline (ISSUE-087, shipped) takes over from there exactly as it does for a
    hand-pushed tag today — including its `require-green-ci` gate, which is the real safety rail: an
    auto-tag only builds and publishes once CI is green on that commit. (4) IN-FLIGHT WORK IS NOT
    DETECTED AT ALL — nothing reads plan-completion state, there is no caveat and no block, and
    whatever is on `main` is trusted as release-ready, the same trust every other automated mechanism
    in this repo already extends to `main`. Branching discipline, not pipeline machinery, is how
    unfinished work stays out of a release. (5) Corpus coverage gaps are still reported rather than
    blocking — compute anyway, name the uncovered commits (DD-8). (6) The artifact↔release join is
    DERIVED from git at derivation time, not stored in a new `released_in:` field (DD-7). (7)
    BUNDLE-020 stays SEPARATE — a possible future INPUT to the derivation, not a shared spine
    (DD-5) — and the multi-repo pack fleet is an explicit FOLLOW-ON bundle, since packs have no
    artifact corpus to derive from (DD-6). Ten draft requirements remain (REQ-001..007, REQ-009,
    REQ-012..013); REQ-008, REQ-010 and REQ-011 were RETIRED by the pivot and their IDs are not
    reused. Maturity is `defined` (promoted at v0.4.0).

requirements:
  - id: REQ-001
    version: "1.1.0"
    text: >
      The release-delta DERIVATION itself must be READ-ONLY and side-effect-free: given the last
      release tag and `HEAD` it computes and reports on stdout, and the derivation must never
      create a tag, push, or otherwise cut a release. Its output is the delta (commits since the
      tag, closed issues), the proposed semver, and the caveats. ACTING on that output — tagging
      and pushing — belongs to the CI job that invokes it (REQ-012), never to the derivation
      (Spec Seed 1).
    versions:
      - version: "1.0.0"
        text: >
          The release-delta derivation must be READ-ONLY and side-effect-free: given the last release
          tag and `HEAD` it computes and reports, and it must never create a tag, push, or otherwise
          cut a release. Its output is the delta (commits since the tag, closed issues, completed
          plans), the proposed semver, and the caveats (Spec Seed 1; the release DECISION stays human
          per the founder framing carried into DD-4).
      - version: "1.1.0"
        text: >
          The release-delta DERIVATION itself must be READ-ONLY and side-effect-free: given the last
          release tag and `HEAD` it computes and reports on stdout, and the derivation must never
          create a tag, push, or otherwise cut a release. Its output is the delta (commits since the
          tag, closed issues), the proposed semver, and the caveats. ACTING on that output — tagging
          and pushing — belongs to the CI job that invokes it (REQ-012), never to the derivation
          (Spec Seed 1).
  - id: REQ-002
    version: "1.0.0"
    text: >
      The proposed bump must derive from the typed artifact corpus, not from commit messages:
      each closed issue in the delta contributes a tier from its `issue.type` — `bug` contributes
      PATCH and `enhancement` contributes MINOR — and the proposed bump is the HIGHEST tier
      present in the delta (DD-1).
  - id: REQ-003
    version: "1.0.0"
    text: >
      The three remaining `issue.type` enum members — `technical-debt`, `question`, and
      `policy-violation` — must contribute PATCH, the same tier as `bug`, on the basis that none
      represents a user-facing capability change. This mapping is recorded as an acknowledged
      narrow first cut rather than a settled ruling, so it must be expressed as data a later
      refinement can change without reshaping the derivation (DD-1).
  - id: REQ-004
    version: "1.0.0"
    text: >
      While the current version is below `1.0.0` the machinery must REFUSE to propose a major
      bump — no derivation path may produce one, and no bespoke breakage model is introduced to
      suppress it. The refusal defers to standard semver, which already holds the major position
      meaningless pre-1.0; declaring 1.0 stays a founder judgment the machinery does not
      pre-empt (DD-1).
  - id: REQ-005
    version: "1.0.0"
    text: >
      Corpus-only commits — those touching only `issues/`, `plans/`, `bundles/`, `directives/`,
      or docs, shipping no binary change — must COUNT toward the currency delta the machinery
      reports, but must never contribute a bump tier. A delta consisting solely of corpus-only
      commits must therefore report drift while proposing no version (DD-1).
  - id: REQ-006
    version: "1.0.0"
    text: >
      The artifact-to-release join must be DERIVED at proposal time from git — tag ancestry plus
      commit-to-artifact association — and never stored. No `released_in:` (or equivalent) field
      may be added to any artifact schema, and no release may trigger a write pass over the
      corpus (DD-7).
  - id: REQ-007
    version: "1.0.0"
    text: >
      Code commits since the last tag with no traceable artifact behind them must be listed
      INDIVIDUALLY as a named caveat in the derivation's output. The machinery must neither
      refuse to propose because of them nor assign them an implicit tier. The uncovered-commit
      list is a first-class part of the output contract — it doubles as the answer to "is the
      corpus complete enough to release from" — not an error path (DD-8).
  # REQ-008 RETIRED 2026-08-10 (founder pivot). It required an in-flight caveat naming the specific
  # mid-lane plan and its completion fraction. The pivot removed the in-flight MECHANISM entirely —
  # nothing reads plan-completion state, `main` is trusted as release-ready — so the requirement has
  # no referent left to reword. See the DD-2 correction and the *Draft Requirements* section. The ID
  # is retired, not reused.
  - id: REQ-009
    version: "1.0.0"
    text: >
      The same derivation that produces the number must also produce a receipt-shaped
      release-note body, with each line traceable to a closed issue and its `delivered_by` chain
      rather than hand-summarized prose (Spec Seed 1).
  # REQ-010 and REQ-011 RETIRED 2026-08-10 (founder pivot). They required release currency to
  # surface as a standing WARNING-severity signal in `backstop gate`. The founder ruled release
  # currency out of the gate entirely — "that does not belong in backstop gates... that is purely a
  # ci-time consideration" — so there is no gate-side signal to specify at any severity. The only
  # behavior left is the CI job's own: it tags, or it does not. See the DD-4 correction and the
  # *Draft Requirements* section. Both IDs are retired, not reused.
  - id: REQ-012
    version: "2.0.0"
    text: >
      The release must be fired AUTOMATICALLY BY CI, with no human approval step. On every push to
      `main` a CI job in backstop-core's own `.github/workflows/` must invoke the pack's derivation
      script (REQ-013) and, when the resulting delta is NON-EMPTY, compute the proposed semver and
      TAG AND PUSH that tag itself. When the delta is empty it must do nothing. No proposal is
      published and nothing waits on a human word, an orchestrator invocation, a scheduled job, a
      GitHub issue, or a new artifact type. Firing the tag is the whole action: the existing
      tag-triggered `release.yml` pipeline (ISSUE-087) then runs unchanged, and its `require-green-ci`
      gate — not this job — is what withholds a build and publish until CI is green on that commit
      (DD-4 as corrected 2026-08-10, Spec Seed 3).
    versions:
      - version: "1.0.0"
        text: >
          The proposer must run ON DEMAND ONLY — invoked by a founder word to an orchestrating agent
          ("propose a release"), computing the derivation fresh at invocation. No scheduled job, no
          standing GitHub issue, and no new artifact type may be introduced as its surface, and the
          proposer must never fire the release itself (DD-4, Spec Seed 3).
      - version: "2.0.0"
        text: >
          The release must be fired AUTOMATICALLY BY CI, with no human approval step. On every push to
          `main` a CI job in backstop-core's own `.github/workflows/` must invoke the pack's derivation
          script (REQ-013) and, when the resulting delta is NON-EMPTY, compute the proposed semver and
          TAG AND PUSH that tag itself. When the delta is empty it must do nothing. No proposal is
          published and nothing waits on a human word, an orchestrator invocation, a scheduled job, a
          GitHub issue, or a new artifact type. Firing the tag is the whole action: the existing
          tag-triggered `release.yml` pipeline (ISSUE-087) then runs unchanged, and its `require-green-ci`
          gate — not this job — is what withholds a build and publish until CI is green on that commit
          (DD-4 as corrected 2026-08-10, Spec Seed 3).
  - id: REQ-013
    version: "2.0.0"
    text: >
      The derivation must ship in the `go-distribution` PACK as a self-contained script, invoked
      DIRECTLY by a workflow step (e.g. `bash
      .backstop/packs/backstop-ai/go-distribution/scripts/<script>.sh`) and writing its result to
      stdout. It must require ZERO core binary changes and no core command. It must NOT be delivered
      as a gate-dispatched engine binding: no `scope_kind`/`input_mode` declaration, no SARIF
      `convert:` bridge, and no participation in `backstop gate` are required, because nothing
      consumes its output as a gate finding (DD-3, with its invocation mechanism corrected by the
      DD-4 correction of 2026-08-10).
    versions:
      - version: "1.0.0"
        text: >
          The machinery must ship in the `go-distribution` PACK as a script-based engine binding
          declaring `scope_kind: project-wide` and `input_mode: none`, running its git query once per
          gate and normalizing its stdout to SARIF through a pack-owned `convert:` script. It must
          require ZERO core binary changes: no core command and no hand-written CI workflow may be
          added to deliver it (DD-3).
      - version: "2.0.0"
        text: >
          The derivation must ship in the `go-distribution` PACK as a self-contained script, invoked
          DIRECTLY by a workflow step (e.g. `bash
          .backstop/packs/backstop-ai/go-distribution/scripts/<script>.sh`) and writing its result to
          stdout. It must require ZERO core binary changes and no core command. It must NOT be delivered
          as a gate-dispatched engine binding: no `scope_kind`/`input_mode` declaration, no SARIF
          `convert:` bridge, and no participation in `backstop gate` are required, because nothing
          consumes its output as a gate finding (DD-3, with its invocation mechanism corrected by the
          DD-4 correction of 2026-08-10).
---

# Release Currency Versioning Machinery

## Current Thinking

### The problem is omission, not error

Verified 2026-07-29: `v0.1.0` (commit `a0fb366`) and `v0.1.1` (commit `87b12cf`) both exist, and
`87b12cf` is currently `HEAD` — so `main` is exactly current with the last release AT THIS MOMENT.
That is the good case, and it is the reason to build the machinery now rather than after the first
slip: there is no drift backlog to reconcile, so whatever lands starts from a clean zero.

The failure mode this bundle exists to close is not a wrong release — it is a release that never
happens. A tag-driven pipeline is only as current as the human who remembers to push a tag. Every
other forcing function in this project works by making the un-done state VISIBLE and LOUD (the gate
going red, `status_drift` catching a stale artifact, `requirement_traceability` catching an
unclaimed requirement). Release currency has no such surface. It is, today, purely a memory
obligation on one person — which is precisely the shape the "eliminate cognitive overhead"
principle exists to attack.

### Why ledger-derived rather than commit-message-derived

The industry-standard answer here is Conventional Commits + semantic-release: parse `feat:` /
`fix:` / `BREAKING CHANGE:` out of commit messages and compute a bump. That answer is a poor fit
for this repo specifically, for two reasons:

1. **Commit volume is agent-generated and high.** Commits land directly on `main` in dense lanes
   (`feat(ISSUE-087): phase 3`, `phase 4`, `phase 5`, then a close commit). Commit-message
   discipline across that volume is exactly the kind of per-commit human obligation the founder is
   trying to eliminate — it swaps "remember to tag" for "remember to prefix," which is not a net
   reduction in cognitive overhead.
2. **This repo already has a better source of truth.** The artifact corpus is structured, typed,
   and gate-enforced. `issue.type` is a closed enum. `delivered_by` chains link plans to issues.
   Plan status is tracked. The gate ALREADY reads this corpus in two steps (`ledger_integrity`,
   `requirement_traceability` — see `pkg/gate/result.go:23` for the step identifier). Deriving a
   release from data the gate already validates is strictly stronger than deriving it from prose
   nobody validates. Commit messages are vibes; artifacts are data.

The corollary that makes this attractive beyond versioning: if the bump is derived from the
artifact corpus, RELEASE NOTES are a byproduct of the same query, and they are receipts — each
line traceable to a closed issue with a `delivered_by` chain, not a hand-summarized changelog.

### Founder prior art: slotly's PR-label auto-versioning, and why it does not transplant

The founder's prior repo (`slotly`) ran an `auto-version.yml` workflow that computed the semver
bump from PR LABELS. It is real, working prior art for "machinery decides the number, human
decides nothing at release time" — and it is worth mining for shape. But it does not transplant to
backstop-core as-is, for a structural reason: **slotly's mechanism keys on pull requests, and
backstop-core has none.** Work lands as direct commits to `main`, at high agent-driven volume, with
no PR to carry a label. Any transplant would have to re-home the signal from a PR label onto
something backstop-core actually has — which is the artifact corpus, which is what the candidate
direction proposes. The precedent's real lesson is the ERGONOMICS (the human never types a version
number), not the mechanism.

### The forcing surface should be the gate — but that raises a law question

The gate is where the founder already looks. A release-currency signal there ("`main` is N commits /
M closed issues ahead of `vX.Y.Z`") converts a memory obligation into a standing, dated, visible
one — and per the loud-≠-blocking principle it should WARN, not block: an unreleased `main` is
un-adopted capability, not a broken promise, and blocking every gate run on "you haven't released"
would be exactly the ceremony this project rejects.

The complication is the zero-baked-checks law. A release-currency check baked into the CLI binary
is a baked check, and this repo's first principle says every check comes from a pack. That routes
the natural home to the `go-distribution` pack (ISSUE-101, open) — which is already being authored
to productize the release trinity ISSUE-087 delivered, and already plans to ship RULES that assert
release-pipeline invariants survive consumer adaptation. A release-currency rule is a natural
member of that rule set. But a pack rule is a semgrep-shaped static analysis over files, and
"how many commits since the last tag" is a git query, not a file pattern — so whether the check is
EXPRESSIBLE as a pack rule at all is a real, unanswered mechanism question (OQ-3), not a
preference.

### In-flight awareness is the part that makes this non-trivial

A naive "N commits ahead → propose a bump" is not merely imprecise, it is dangerous. The concrete
live example, from this week: ISSUE-104 (SARIF severity descriptor fallback) and ISSUE-105 (step
verdict ignores severity without policy entry) were two hops of ONE severity contract. A tag fired
between them landing would have shipped consumers a half-fixed contract — worse than no fix,
because the half-state is the kind of partial behavior that looks correct until it silently isn't.

So the proposer must read state the git graph does not carry: is there an open plan whose phases
are partially committed? Is a directive mid-drain? This is artifact-native information, and it is
the reason this capability belongs near the corpus rather than near the CI pipeline. Whether
in-flight state HARD-BLOCKS the proposal or merely annotates it as a caveat is genuinely open
(OQ-2) — hard-blocking risks a proposer that never fires in a repo that is essentially always
mid-lane; caveat-only risks the founder firing past a warning they stopped reading.

### CORRECTION (2026-08-10, v0.5.0, founder-ruled pivot) — two subsections above are superseded

The two subsections immediately above are preserved as written and are both SUPERSEDED. *"The
forcing surface should be the gate — but that raises a law question"*: there is no gate surface at
all any more; the founder ruled release currency out of `backstop gate` entirely as a purely CI-time
concern. The law question it raised is moot in the same stroke — the derivation still lives in the
`go-distribution` pack, so zero-baked-checks is satisfied, but not via a gate check. *"In-flight
awareness is the part that makes this non-trivial"*: in-flight awareness does not exist. Nothing
reads plan state; `main` is trusted as release-ready; the ISSUE-104/105 hazard is handled by
branching discipline, not by detection. See the DD-2 and DD-4 corrections for the full rulings and
the founder's reasoning.

### What is DECIDED vs OPEN

**Superseded 2026-08-10 — kept for the lineage. Read the correction below it.**

> DECIDED (founder framing, not to be re-litigated): the release DECISION stays human — this is not
> a request for continuous deployment or auto-tagging; the remembering and the computing are the
> machinery's job, and the trigger is a human word. Everything else — including all three parts of
> the candidate direction above — is OPEN. No design decisions are recorded in this bundle, and no
> requirements: at `exploring`, recording DDs for a direction the founder has not interrogated would
> be pre-resolution by another name. OQ-1..OQ-8 below are the genuine forks. The founder drives
> resolution and promotion.

**CORRECTION (2026-08-10, founder-ruled): ALL EIGHT open questions are now RESOLVED**, and the
candidate direction survived interrogation substantially intact — the three parts are now decided
design, recorded as DD-1..DD-8 under *Draft Design Decisions*. The founder's original framing above
still holds and is still not re-litigable: the release DECISION stays human, and the trigger is a
human word (now specifically an on-demand founder word to an orchestrating agent, DD-4).

What CHANGED beyond mere confirmation is OQ-3, the load-bearing architecture question. It resolved
to the `go-distribution` pack on the strength of NEW mechanism evidence that did not exist when the
OQ was written — the pack engine model already supports project-wide script engines that run a git
query and emit SARIF, so the "a git query has no file to match" expressibility objection that made
OQ-3 genuinely uncertain is answered rather than merely traded off. See DD-3 for the citations.

What is still OPEN is NOT the design but the PROMOTION: this bundle stays `exploring`. Resolving
eight OQs settles the shape; `defined` additionally required a drafted `requirements[]` array,
which v0.3.0 (2026-08-10) supplied as REQ-001..013 — so the only thing still standing between this
bundle and `defined` is the founder's own promotion call, which no agent takes. The narrowest known
soft spot in the decided design is
DD-1's treatment of `technical-debt`, `question`, and `policy-violation` — deliberately defaulted to
patch as a first cut, explicitly refinable, and flagged as such rather than presented as settled.

## Draft Requirements

**CORRECTION (2026-08-10, v0.5.0, founder-ruled pivot) — read this before the v0.3.0 text below,
which is preserved unedited and is stale in three places.** The pivot recorded in the DD-2 and DD-4
corrections **RETIRED three requirements and rewrote two**:

- **REQ-008 REMOVED (in-flight caveat).** It required the proposal to carry a caveat naming the
  specific mid-lane plan and its completion fraction. The pivot removed the in-flight MECHANISM
  itself — nothing reads plan-completion state at all, and `main` is trusted as release-ready — so
  there is no longer anything for the requirement to constrain. This is a removal, not a rewording:
  a reworded REQ-008 would imply a detection step that no longer exists.
- **REQ-010 and REQ-011 REMOVED (gate surfacing, and its WARNING severity).** The founder ruled
  release currency out of `backstop gate` entirely: *"that does not belong in backstop gates... that
  is purely a ci-time consideration."* With no gate-side signal there is no signal CONTENT to
  specify (REQ-010) and no SEVERITY to specify for it (REQ-011). The whole *Release-currency
  surfacing* seed loses its requirements; the CI job's own behavior — it tags, or it does not — is
  now the entire observable surface, and that is REQ-012's business.
- **REQ-012 REWRITTEN, and INVERTED (v1.0.0 → v2.0.0).** Was: on-demand only, fired by a founder
  word, *"must never fire the release itself."* Now: automatic on every push to `main`, no human
  approval anywhere, and it explicitly DOES fire the release — computing the delta and, when the
  delta is non-empty, tagging and pushing. The safety rail is not a human and not this job: it is
  the `require-green-ci` gate already inside the tag-triggered `release.yml` pipeline (ISSUE-087),
  which withholds the build and publish until CI is green on that commit exactly as it does for a
  hand-pushed tag today.
- **REQ-013 REWRITTEN, home unchanged (v1.0.0 → v2.0.0).** The `go-distribution` pack is still the
  home and zero core binary changes still holds (DD-3 stands). What changed is the INVOCATION
  MECHANISM: it is no longer a gate-dispatched engine binding, so the `scope_kind: project-wide` +
  `input_mode: none` + SARIF `convert:` construction is no longer required — nothing consumes the
  output as a gate finding. It is now a plain script a workflow step runs. Note the v1.0.0 clause
  banning *"a hand-written CI workflow"* is likewise void: a CI workflow is now the intended invoker.
- **REQ-001 CLARIFIED (v1.0.0 → v1.1.0), substance unchanged.** The derivation is still read-only
  and still never tags — but for a different reason than it was written for. It is now a SEPARATION
  OF CONCERNS between a computing script and an acting workflow, not evidence that a human is in the
  loop; the clause appealing to *"the release DECISION stays human per DD-4"* is void. *"Completed
  plans"* was also dropped from its output contract, since nothing reads plan state any more.

**Ten requirements survive**: REQ-001..007, REQ-009, REQ-012, REQ-013. The three retired IDs are
NOT reused. The seed partition is correspondingly reduced to two live groups — **REQ-001..007 and
REQ-009 to release-delta derivation**, **REQ-012 to the trigger** (no longer "the proposer"), with
**REQ-013 cross-cutting**; the release-currency-surfacing group is empty.

---

*Original v0.3.0 text, preserved unedited:*

REQ-001 through REQ-013 are carried in the frontmatter `requirements` block (v0.3.0, 2026-08-10).
Every one is derived from a resolved OQ's design decision (DD-1..DD-8) or from one of the first
three spec seeds — nothing below introduces scope those did not already settle. They partition
across the three in-scope seeds with no overlap: **REQ-001..009 belong to release-delta
derivation**, **REQ-010..011 to release-currency surfacing**, **REQ-012 to the proposer and its
trigger**. **REQ-013 is cross-cutting** — it fixes the HOME all three share, and is the one
requirement that constrains every other.

### Release-delta derivation (REQ-001 – REQ-009)

- **Posture** (REQ-001): read-only and side-effect-free. It computes and reports the delta, the
  proposed semver, and the caveats; it never tags. This is the seed's own framing and the
  mechanical form of the founder's standing rule that the release DECISION stays human.
- **The bump rules** (REQ-002, REQ-003, REQ-004, REQ-005) — DD-1, split along its own seams so
  the settled part and the soft part are separately testable. REQ-002 is the settled core:
  `issue.type` over commit messages, `bug`→PATCH, `enhancement`→MINOR, highest tier in the delta
  wins. REQ-003 carries DD-1's acknowledged narrow cut — the other three enum members default to
  PATCH — and requires the mapping be data, so refining it later is a table edit rather than a
  redesign. REQ-004 is the pre-1.0 major REFUSAL (deferring to semver, not inventing a cap).
  REQ-005 is the corpus-only case: counts toward drift REPORTING, never toward a bump.
- **The join** (REQ-006): derived from git at proposal time, with an explicit ban on adding a
  stored `released_in:` field or a per-release write pass over the corpus (DD-7).
- **Honesty about the corpus** (REQ-007): uncovered code commits are named individually as a
  first-class caveat — neither a refusal (DD-8's rejected fail-closed option) nor a silently
  invented patch tier (its rejected guess option). This is also DD-7's accepted-cost mitigation.
- **In-flight caveat** (REQ-008): names the specific plan and its completion fraction and never
  blocks. The specificity requirement is DD-2's stated mitigation for the caveat-you-click-past
  risk, so it is written as a testable constraint rather than left as advice.
- **Notes as receipts** (REQ-009): the same query that yields the number yields the release-note
  body, each line traceable to a closed issue and its `delivered_by` chain (Spec Seed 1).

### Release-currency surfacing (REQ-010 – REQ-011)

- **The signal** (REQ-010): the standing "`main` is N commits / M closed issues ahead of
  `vX.Y.Z`, proposed `vX.Y.Z+1`" line in `backstop gate` — zero new surface, already in the
  founder's daily path (DD-4).
- **Its severity** (REQ-011): WARNING, never blocking, stated in the pack severity contract's own
  terms (SARIF `level: warning` is non-blocking by contract) so the requirement is checkable
  against the mechanism rather than against an adjective.

### The proposer and its trigger (REQ-012)

- On-demand only, fired by a founder word to an orchestrating agent, computing fresh; the
  rejected alternatives (scheduled job, standing GitHub issue, a new artifact type as surface)
  are written into the requirement as prohibitions, since DD-4 rejected them explicitly and a
  spec-author would otherwise be free to reintroduce them.

### Cross-cutting home (REQ-013)

- The `go-distribution` pack, as a project-wide script engine (`scope_kind: project-wide` +
  `input_mode: none`) whose `convert:` script bridges stdout to SARIF, with ZERO core binary
  changes and an explicit ban on a core command or a hand-written CI workflow (DD-3). This is the
  load-bearing decision of the bundle, so it is a requirement rather than context: it is what
  keeps the capability law-compliant under zero-baked-checks.

### Deliberately NOT requirements here

- **Fleet generalization** — the fourth spec seed and DD-6. Packs have no artifact corpus, so the
  derivation does not transplant; fleet-wide currency reporting is recorded as a follow-on bundle
  and contributes NO requirement to this one. A spec-author should not read REQ-010/011 as
  implying a fleet surface.
- **Pack↔core version compatibility** — DD-5. BUNDLE-020 stays separate; its capability contracts
  are a possible future INPUT to REQ-002's tiering, not scope here, and nothing above depends on
  BUNDLE-020 landing.
- **How a release is BUILT and PUBLISHED** — owned by DIR-001 and already shipped by ISSUE-087.
  This bundle stops at proposing the number and the notes; everything downstream of the tag is
  untouched.
- **Refining DD-1's three defaulted `issue.type` members** — REQ-003 pins the current default and
  requires it be refinable, but the better mapping is deliberately left unspecified rather than
  guessed at requirement level.

## Draft Design Decisions

All eight are FOUNDER DECISIONS taken 2026-08-10, resolving OQ-1..OQ-8 one-for-one (OQ-N → DD-N).
They are DRAFT in the bundle sense — they fix the direction and are the material a spec-author
should build from, but they are not yet requirements and this bundle is not yet `defined`. Each
carries enough of its originating reasoning to be read standalone; the fuller discussion, including
the options rejected, is preserved under *Open Questions → Resolved*.

- **DD-1: The semver bump derives from `issue.type` — `bug` → PATCH, `enhancement` → MINOR, and the
  machinery REFUSES to propose a major before 1.0.** (Resolves OQ-1. Conservative first cut,
  explicitly refinable.) The derivation reads the typed artifact corpus, not commit messages:
  closed issues since the last tag carry a typed `issue.type`, and that type — not a `feat:` prefix
  anyone remembered to write — determines the tier. The proposed bump for a release is the HIGHEST
  tier present in the delta.

  *No auto-major before 1.0, by convention rather than by policy invention:* standard semver already
  holds that the major position carries no meaning pre-1.0 (0.x makes MINOR the breaking lane), so
  the machinery does not need a bespoke rule to suppress majors — it simply declines to propose a
  position semver says is not yet meaningful. This is a refusal, not a cap: a 1.0 decision is a
  founder judgment the machinery has no business pre-empting.

  *The three unmapped types are an ACKNOWLEDGED NARROW CUT, not a ruling.* `technical-debt`,
  `question`, and `policy-violation` all close and all default to PATCH — the same tier as `bug` —
  on the reasoning that none of the three represents a user-facing capability change, which is what
  a minor is for. This is recorded as narrower than ideal and open to later refinement; it is NOT a
  permanent judgment that those three are patch-shaped forever. A spec-author should treat the
  `bug`/`enhancement` mapping as settled and the other three as a default with a known soft edge.

  *Corpus-only commits count toward REPORTING but never toward a BUMP.* A commit touching only
  `issues/`, `plans/`, `bundles/`, `directives/`, or docs ships no binary change, so it cannot
  justify a version whose artifact would be byte-identical to the last one. It still counts in the
  "how far ahead is `main`" currency signal, because the founder wants to see drift accumulating
  even when that drift is not yet releasable. This is the inverse case to DD-8 and the two should be
  read together.

- **DD-2: In-flight state produces a CAVEAT on the proposal, never a HARD BLOCK.** (Resolves OQ-2.)
  The proposal is still computed and still fires when an open plan has partially-committed phases;
  it is annotated with a caveat naming the specific in-flight plan and its completion fraction —
  e.g. *"PLAN-ISSUE-XXX is mid-lane, 3/7 tasks committed — releasing now would ship partial work"* —
  and the founder reads the caveat and decides.

  *Why caveat over hard block:* this matches the standing loud-≠-blocking law, but the decisive
  argument is empirical rather than principled — **this repo is nearly always mid-lane on
  something.** A hard block keyed on in-flight state risks a proposer that effectively never fires,
  which is a worse failure than a proposal the founder has to read carefully: a machine that never
  speaks restores exactly the pure-memory obligation this bundle exists to eliminate. Hard-blocking
  the proposal outright is EXPLICITLY REJECTED.

  *The known cost, carried openly:* OQ-2 raised the real counter-argument — a caveat the founder
  learns to click past is a vacuous-green surface of the kind this project exists to defeat. That
  risk is accepted, not dissolved. It is why the caveat must NAME the specific plan and its
  fraction rather than emit a generic "work may be in flight": a specific, changing, checkable
  caveat is materially harder to tune out than a constant one, and a spec-author should treat the
  specificity of the caveat text as load-bearing rather than cosmetic.

  **CORRECTION (2026-08-10, founder-ruled) — DD-2 is SUPERSEDED IN FULL. There is NO in-flight
  detection mechanism at all.** Not a caveat, not a block, not a degraded signal: nothing in this
  capability reads plan-completion state, directive drain state, or any other in-flight indicator.
  The founder's words: *"in-flight work should not be included in a release... in flight work should
  be irrelevant here... easier to get more disciplined on branching vs building in all kinds of weird
  shit into the release pipeline."*

  *What the premise becomes:* **whatever is on `main` is trusted as release-ready, full stop** — the
  same trust every other automated mechanism in this repo already extends to `main`. The question
  DD-2 was answering ("what makes `main` NOT release-ready") is not re-answered with a different
  option; it is DISSOLVED. `main` is release-ready by definition, and keeping unfinished work off
  `main` is a BRANCHING-DISCIPLINE problem solved in the workflow, not a detection problem solved in
  the pipeline.

  *Why this is a better trade than the caveat:* DD-2 accepted a known cost (a caveat the founder
  learns to click past) and paid for it with a mechanism that had to read state the git graph does
  not carry — plan phases, completion fractions, mid-lane directives. The founder judged that to be
  pipeline weight bought to compensate for a workflow habit, and chose to fix the habit instead. The
  ISSUE-104/105 example that motivated DD-2 remains a real hazard; under this ruling it is guarded by
  landing both hops on one branch before merging to `main`, not by teaching the release machinery to
  recognize half-landed contracts.

  *Downstream:* REQ-008 is RETIRED rather than reworded (see *Draft Requirements*), and the
  derivation's output contract no longer includes completed-plan state (REQ-001 v1.1.0). DD-8's
  uncovered-COMMIT caveat is untouched and still stands — it is a corpus-coverage honesty signal
  about what the derivation could not see, not an in-flight signal, and it needs only git and the
  corpus.

- **DD-3: The machinery lives in the `go-distribution` PACK as a project-wide SCRIPT ENGINE.**
  (Resolves OQ-3 — the load-bearing architecture question. Decided on BOTH expressibility AND cost,
  which is the change from v0.1.1, where only cost evidence existed.)

  *The expressibility objection is ANSWERED, not traded off.* OQ-3's genuine uncertainty was
  mechanical: pack rules were understood as semgrep-shaped static analysis over FILES, and "how many
  commits since the last tag" is a git query with no file to match — so whether the check was
  EXPRESSIBLE as a pack rule at all was an open mechanism question, not a preference. Confirmed live
  2026-08-10 (while scoping an unrelated defect, SPEC-036's blocked contract-compiler gap in the
  `backstop-go-contracts-pack` repo): the pack engine model ALREADY supports exactly this shape.
  `engine.ScopeKindProjectWide` and `engine.InputModeNone` both exist and are both live in dispatch
  — `pkg/pack/engine/binding.go:24-26` documents `InputModeNone` as *"the executable is the logic"*
  and `:55-57` documents `ScopeKindProjectWide` as running the command once project-wide, ignoring
  the scanned file set. In the dispatch path: `cmd/backstop/pack_gate.go:600` is `runFindingsEngine`
  branching on `ScopeKindProjectWide`, `:543` is `buildEngineArgs` handling `InputModeNone` (it
  injects no rules and no config), and `:440` is the parallel project-wide branch in the COVERAGE
  channel, `dispatchPackCoverage`. So a pack can declare a script-based engine that runs ONCE,
  PROJECT-WIDE, executes arbitrary logic (a `git log` / `git tag` query here), and emits SARIF —
  with **ZERO core binary changes.**

  *This is a shipped pattern, not a theoretical capability.* Verified 2026-08-10: the installed
  `backstop-ai/go-toolchain` pack's `go-build` and `go-test` bindings ALREADY combine
  `input_mode: none` + `scope_kind: project-wide` + `project_target: "./..."` + a pack-owned
  `convert:` script that normalizes arbitrary command stdout into SARIF
  (`.backstop/packs/backstop-ai/go-toolchain/pack.yml:52-90`). A release-currency engine is the same
  construction pointed at `git` instead of `go` — the SARIF bridge is the `convert:` script, which
  is the existing seam, not a new one.

  *Combined with the cost evidence already recorded at v0.1.1:* implementer-101's 2026-07-29
  assessment found the capability fits the pack as a SECOND RECIPE with no new engine work, since
  the pack already declares the provisioned engine binding `recipe apply` demands. That evidence
  addressed cost but explicitly NOT expressibility; the two halves now meet, and OQ-3 is decided on
  both grounds rather than on cost alone.

  *Both alternatives are EXPLICITLY REJECTED.* A CORE COMMAND violates the zero-baked-checks law
  directly — a release-currency check inside the CLI binary is precisely the baked-check class this
  repo's first principle names as a defect to eradicate, and `backstop/self` guards that boundary. A
  CI WORKFLOW is new hand-written pipeline debt of exactly the kind ISSUE-101 exists to retire, so
  it would add debt to a class already being drained.

- **DD-4: The surface is a GATE WARNING; the trigger is a FOUNDER WORD, on demand.** (Resolves OQ-4,
  both coupled halves.)

  *Surface:* a WARNING-severity signal in `backstop gate` output — *"`main` is N commits / M closed
  issues ahead of `vX.Y.Z` (proposed: `vX.Y.Z+1`, patch)"*. Chosen because it is ZERO NEW SURFACE
  and already in the founder's daily path; warning severity follows loud-≠-blocking, since an
  unreleased `main` is un-adopted capability, not a broken promise, and blocking every gate run on
  "you haven't released" is exactly the ceremony this project rejects.

  *Trigger:* ON-DEMAND ONLY. The founder says something like *"propose a release"* to an
  orchestrating agent, which runs the derivation fresh and reports back. This matches how work is
  actually driven in this repo today and requires no new infrastructure.

  *Rejected:* a SCHEDULED JOB (new infrastructure this decision deliberately avoids) and a STANDING
  GITHUB ISSUE (this repo does not otherwise use GitHub issues as a work surface, so it would be an
  external surface adopted for one purpose). Note this splits the difference OQ-4 identified — the
  proposer COULD have been autonomous precisely because it only proposes; the founder ruled for the
  smaller-infrastructure option anyway.

  **CORRECTION (2026-08-10, founder-ruled) — DD-4 is SUPERSEDED IN FULL, in BOTH halves. This is a
  PURE CI-TIME MECHANISM that AUTO-TAGS, with no gate involvement and no approval step.**

  *No gate surface, at any severity.* The founder ruled release currency out of `backstop gate`
  entirely: *"that does not belong in backstop gates... that is purely a ci-time consideration."*
  There is no WARNING, no finding, no gate step, and nothing for the pack severity contract to
  classify. The DD-4 surface question is dissolved rather than re-answered — the mechanism has no
  surface in the founder's daily path because it does not need one.

  *No approval step, because there is no seamless way to give one.* The founder's constraint,
  verbatim: *"i am way too busy and overloaded to switch over to github to manually review and
  approve"*, and the rule he drew from it: **"unless there's a seamless way for me to approve or
  deny, then it should just be auto released."** Every approval surface DD-4 and OQ-4 considered — a
  standing GitHub issue, a `workflow_dispatch`, even the on-demand founder word DD-4 chose — costs a
  context switch he does not have. A proposal nobody has bandwidth to read is not a safety
  mechanism; it is a queue. So the machinery ACTS.

  *The mechanism, concretely.* On EVERY PUSH TO `main`, a CI job in backstop-core's own
  `.github/workflows/` runs the `go-distribution` pack's delta-derivation script directly (`bash
  .backstop/packs/backstop-ai/go-distribution/scripts/<script>.sh` or similar). If the delta is
  EMPTY it does nothing. If the delta is NON-EMPTY it computes the proposed semver per DD-1 and
  **tags and pushes the tag itself.** The existing tag-triggered `release.yml` pipeline (ISSUE-087,
  already shipped) then takes over exactly as it does for a hand-pushed tag today.

  *Where the safety actually lives.* Not in a human and not in this job: in `release.yml`'s existing
  `require-green-ci` gate. An auto-pushed tag only builds and publishes once CI has gone green on
  that commit — the same rail that already protects a manual tag. That rail is the reason removing
  the human is not removing the check.

  *Consequence for DD-3.* The HOME is unchanged — the derivation still lives in the `go-distribution`
  pack, still with zero core binary changes — but its INVOCATION simplifies: it is no longer
  dispatched through `backstop gate`'s pack-engine mechanism, so no `scope_kind: project-wide` /
  `input_mode: none` binding, no SARIF `convert:` bridge, and no engine registration are needed.
  Nothing consumes the output as a gate finding, so nothing needs it in SARIF. It is a script a
  workflow step runs and reads. Note also that DD-3's rejection of a "CI workflow" home is now read
  narrowly: what was rejected was a hand-written workflow OWNING THE DERIVATION LOGIC; a thin
  workflow step that INVOKES the pack's script is the intended shape, and the logic still lives in
  the pack.

  *Downstream:* REQ-010 and REQ-011 are RETIRED (see *Draft Requirements*), REQ-012 is INVERTED
  (v2.0.0 — it now fires the release), and REQ-013 is rewritten to drop the engine-binding
  construction (v2.0.0).

- **DD-5: This bundle and BUNDLE-020 stay SEPARATE; BUNDLE-020 may later be an INPUT.** (Resolves
  OQ-5 — a scope-boundary ruling.) Versioning MACHINERY (this bundle: what version to cut) and
  version COMPATIBILITY (BUNDLE-020: which versions can work together) are NOT merged into one
  spine and do NOT share a bundle or a spec.

  *The future relationship is CONSUMPTION, not sharing:* if BUNDLE-020 lands capability contracts,
  that becomes an INPUT this bundle's derivation could consume — a core release that ADDS a named
  capability would have a machine-visible reason to be a MINOR rather than a patch, which is a real
  derivation signal DD-1 would want. That is a one-directional dependency to be picked up later,
  not a reason to co-own the problem now. The standing hazard this avoids is one bundle owning two
  problem spaces, which this project has been burned by before.

- **DD-6: The multi-repo pack fleet is OUT OF SCOPE for this bundle; fleet-wide currency reporting
  is an explicit FOLLOW-ON.** (Resolves OQ-6.) Packs have NO artifact corpus — no issues, no plans,
  no `delivered_by` chains — so this bundle's ledger-derivation (DD-1) has nothing to read in a pack
  repo and does not transplant as-is.

  *What IS recorded as a likely follow-on bundle:* fleet-wide release-CURRENCY reporting, which
  needs only git (tags and commit counts) and not the corpus, and is therefore separable from the
  derivation. This matters because the fleet arguably suffers the target failure mode MORE acutely
  than core does — a pack fix that never gets tagged is a fix no consumer can `pack update` to, and
  the fix→bump→relock flywheel stalls silently. Recorded so it is not lost; explicitly NOT built
  here.

- **DD-7: The artifact↔release join is DERIVED from git, not STORED on artifacts.** (Resolves OQ-7.)
  No new `released_in:` field is added to any artifact type. "Which artifacts shipped in which
  release" is COMPUTED at proposal time from git — tag ancestry plus commit-to-artifact association.

  *Why derived:* it matches this project's standing preference for deriving state from the world
  rather than storing new fields everywhere, and it avoids the costs the stored option carries — a
  schema change across artifact types, a write pass over the corpus at every release, and a NEW
  CLASS OF DRIFT (a `released_in` value that disagrees with the actual tags), which would be a fresh
  instance of exactly the staleness problem `status_drift` exists to catch.

  *The accepted cost:* OQ-7 named it — the derivation depends on a commit↔artifact association that
  is conventional rather than guaranteed, and it is recomputed (possibly differently) on every run.
  DD-8 is the mitigation: where the association cannot be made, the machinery says so out loud
  rather than silently dropping the commit.

- **DD-8: Uncovered code commits do NOT block the proposal — the machinery computes anyway and NAMES
  them as an explicit caveat.** (Resolves OQ-8.) When code commits exist since the last tag with no
  traceable artifact behind them, the output lists those specific commits as a named caveat, and the
  founder adjudicates whether the corpus is complete enough to trust the derivation.

  *Both alternatives rejected, for opposite reasons.* REFUSING to propose (fail-closed) is the
  strongest honesty posture but would fire constantly given fix-up commits, doc-adjacent code edits,
  and mechanical renames — another route to a proposer that never usefully fires (the same failure
  DD-2 guards against). Silently treating uncovered commits as an implicit PATCH-tier signal is
  worse: it INVENTS DATA the machinery does not have, which is the commit-messages-are-vibes problem
  re-entering through the back door, and it would under-version silently — a vacuous-green failure
  pointed at consumers instead of at the gate.

  *The caveat is arguably the more valuable output.* As OQ-8 observed, "is the corpus complete enough
  to release from" may be a more useful signal than the proposed number itself. A spec-author should
  treat the uncovered-commit list as a first-class part of the derivation's output contract, not as
  an error path.

## Open Questions

**ALL EIGHT RESOLVED 2026-08-10 by founder ruling** — each maps one-for-one to a design decision
(OQ-N → DD-N). The original text and reasoning of every OQ is preserved below verbatim, per this
repo's dated-correction convention; resolutions are APPENDED, nothing is deleted. Maturity stays
`exploring` regardless: promotion is a separate founder call. (The `requirements[]` array this
originally noted as missing was drafted at v0.3.0 — see *Draft Requirements*.)

- OQ-1 Semver derivation rules — **RESOLVED** → DD-1 (`bug`→patch, `enhancement`→minor, no
  auto-major pre-1.0; three types default to patch as an acknowledged narrow cut; corpus-only
  commits report but never bump)
- OQ-2 In-flight detection — **RESOLVED** → DD-2 (caveat only, never a hard block) — **DD-2
  SUPERSEDED 2026-08-10 (v0.5.0): no in-flight detection at all; `main` is trusted as release-ready**
- OQ-3 Where the machinery lives — **RESOLVED** → DD-3 (`go-distribution` pack, project-wide script
  engine; decided on expressibility AND cost, with new mechanism evidence)
- OQ-4 Proposal surface and trigger — **RESOLVED** → DD-4 (gate warning; on-demand founder word) —
  **DD-4 SUPERSEDED 2026-08-10 (v0.5.0): no gate surface and no approval step; a CI job on every push
  to `main` auto-tags when the delta is non-empty**
- OQ-5 Relationship to BUNDLE-020 — **RESOLVED** → DD-5 (separate; possible future input)
- OQ-6 Multi-repo pack fleet — **RESOLVED** → DD-6 (out of scope; explicit follow-on)
- OQ-7 Artifact↔release join — **RESOLVED** → DD-7 (derived from git, not stored)
- OQ-8 Corpus coverage gaps — **RESOLVED** → DD-8 (warn and compute anyway, naming uncovered
  commits)

### Resolved (kept for the reasoning)

The preamble below is the original v0.1.0 framing, preserved as written. Every OQ that follows
carries its original text plus an appended, dated resolution.

Not pre-resolved. Numbered sequentially; each carries its reasoning and its couplings. No leans are
recorded where the founder's judgment is the whole content of the answer; where an existing
standing principle constrains the space, it is cited so the founder can rule with it in view.

- **OQ-1 — SEMVER DERIVATION RULES. (RESOLVED 2026-08-10 → DD-1.)** Which artifact signals map to which bump? The obvious first
  cut is `issue.type` → bump: `bug` → patch, `enhancement` → minor, and something → major. But
  three sub-questions are live inside that. (a) The enum has five members, not two —
  `technical-debt`, `question`, and `policy-violation` also close, and none of them map obviously
  to a bump tier. (b) What does "breaking" even MEAN pre-1.0? Semver convention says 0.x makes the
  MINOR position the breaking lane (0.1.x → 0.2.0 signals breakage), which would mean this repo
  has no major-bump path at all until 1.0 and the enhancement/breaking distinction collapses into
  one position. Is that the intended reading, or should the machinery model breakage separately
  and simply refuse to auto-major before 1.0? (c) CORPUS-ONLY CHANGES: a large share of this
  repo's commits touch only `issues/`, `plans/`, `bundles/`, `directives/`, and docs — no code
  ships. Are those releasable at all (a release whose binary is byte-identical to the last one is
  arguably noise), skippable (the deriver ignores them and the delta reads zero), or something in
  between (they count toward "how far ahead" reporting but never toward a bump)? Note this cuts
  BOTH ways with OQ-8, which asks the inverse. No lean — founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-1. A conservative first cut, labelled as one.**
  `bug` → PATCH, `enhancement` → MINOR. Sub-question (b) — pre-1.0 breaking semantics — resolved by
  DEFERRING to standard semver rather than modelling breakage separately: convention already holds
  that the major position carries no meaning pre-1.0, so the machinery simply REFUSES to propose an
  auto-major before 1.0 rather than inventing a rule to suppress one. Sub-question (a) — the three
  remaining enum members — resolved NARROWLY and explicitly provisionally: `technical-debt`,
  `question`, and `policy-violation` all default to PATCH (same as `bug`), on the reasoning that
  none of the three represents a user-facing capability change. This is called out in DD-1 as
  narrower than ideal and refinable later, NOT as a permanent ruling on those three types.
  Sub-question (c) — corpus-only changes — resolved to the "something in between" option the OQ
  itself named: they COUNT toward the "how far ahead" currency REPORTING but NEVER trigger a bump on
  their own, since a release whose binary is byte-identical to the last one is noise.

- **OQ-2 — IN-FLIGHT DETECTION: what makes `main` NOT release-ready? (RESOLVED 2026-08-10 → DD-2.)**
  Candidate signals, each
  with a different cost: an open plan with committed partial phases (artifact-native, and exactly
  the ISSUE-104/105 case — but this repo is nearly always mid-lane on something, so a hard block
  on it may mean the proposer never fires); red CI on the head commit (cheap, external, already
  computed by the existing tag-integrity gate — but it catches breakage, not incompleteness); a
  directive mid-drain (coarser, probably too coarse); nothing at all (the founder eyeballs it).
  And the posture question that rides on top: is in-flight state a HARD BLOCK on the proposer, or a
  STATED CAVEAT on a proposal that still gets made? The loud-≠-blocking principle argues for the
  caveat — but a caveat the founder learns to click past is a vacuous-green surface of exactly the
  kind this project exists to defeat, so the principle does not settle it. Couples to OQ-4 (a
  caveat needs somewhere to be stated). No lean — founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-2. CAVEAT ONLY, never a hard block.** The proposal
  still computes and still fires even when an open plan has partially-committed phases; it is
  annotated with a caveat naming the SPECIFIC in-flight plan and its completion fraction (e.g.
  *"PLAN-ISSUE-XXX is mid-lane, 3/7 tasks committed — releasing now would ship partial work"*), and
  the founder reads it and decides. This matches the standing loud-≠-blocking law, but the decisive
  argument was the empirical one the OQ itself raised: **this repo is nearly always mid-lane on
  something**, so hard-blocking risks a proposer that effectively NEVER FIRES — which restores the
  pure-memory obligation this bundle exists to eliminate. Hard-blocking the proposal outright is
  EXPLICITLY REJECTED. The OQ's counter-risk (a caveat the founder learns to click past) is ACCEPTED
  rather than dissolved; DD-2 records the mitigation, which is that the caveat must name the plan
  and its fraction rather than emit a generic warning — a specific, changing caveat is materially
  harder to tune out than a constant one.

  **SUPERSEDED 2026-08-10 (v0.5.0, founder-ruled) — see the DD-2 correction.** The caveat option is
  gone along with the whole in-flight mechanism: nothing reads plan state, `main` is trusted as
  release-ready, and unfinished work is kept out of a release by branching discipline rather than by
  detection. OQ-2's question is dissolved, not re-answered.

- **OQ-3 — WHERE THE MACHINERY LIVES. (RESOLVED 2026-08-10 → DD-3.)** Three homes, each with a different consequence for the
  zero-baked-checks law and for non-backstop consumers. (a) A CORE COMMAND (`backstop release
  propose`?) — has native access to the artifact corpus, the schema loaders, and git; but a
  release-currency CHECK baked into the binary is precisely the baked-check class this repo's
  first principle names as a defect to eradicate, and the corpus it reads is backstop's own
  artifact vocabulary, which no external consumer has. (b) A CI WORKFLOW — no baked-check problem,
  but it is hand-written pipeline infrastructure of exactly the kind ISSUE-101 exists to
  productize away, so it would be new debt in a class already being retired. (c) The
  `go-distribution` PACK (ISSUE-101) as a rule and/or recipe — law-compliant and fleet-reusable,
  and the pack is already being authored; but pack rules are semgrep-shaped static analysis over
  FILES, and "commits since the last tag" is a git query with no file to match, so it may simply
  not be expressible in the pack rule model as it stands. A fourth possibility worth naming: the
  DERIVATION (corpus → proposed bump) and the SURFACING (the standing "you are N ahead" warning)
  are separable and may not want the same home. What does each choice do to a non-backstop
  consumer of `go-distribution`, who has a git history and tags but NO artifact corpus at all? No
  lean — founder's, and it is the load-bearing architecture question of this bundle.

  **Implementation-cost evidence for option (c), recorded 2026-07-29 — INPUT to this OQ, not a
  resolution of it.** The implementer who had just built the `go-distribution` pack's first recipe
  assessed the shape an auto-version capability would take there: it fits as a SECOND RECIPE — a
  compute-next-semver payload plus a propose-or-tag-on-trigger workflow payload — with paired
  enforcement rules. Two specifics reduce the estimated cost. First, the pack ALREADY declares the
  provisioned engine binding that `recipe apply` demands, so a second recipe adds NO engine work.
  Second, the trigger half is a WORKFLOW payload — exactly the artifact shape the pack's existing
  rules already guard — so the enforcement half rides an established pattern rather than inventing
  one. Two caveats travel with this evidence and are why it does not settle the question. It is
  ASYMMETRIC: the core-command (a) and CI-workflow (b) options have NOT been assessed for cost by
  anyone, so this is one measured option against two unmeasured ones, not a comparison. And it
  speaks to COST, not to the EXPRESSIBILITY concern raised above — a recipe is a payload the
  consumer applies, which is a different mechanism from a rule answering "how many commits since
  the last tag," so the git-query-has-no-file-to-match problem is untouched by it. What it does
  establish is that the pack home is cheap IF the mechanism fits.

  **NEW MECHANISM EVIDENCE, confirmed live 2026-08-10 — this is what closed the question.** The
  expressibility concern above ("a git query has no file to match, so whether the check is
  EXPRESSIBLE as a pack rule at all is a real, unanswered mechanism question") rested on an
  incomplete picture of the pack engine model. While scoping an unrelated defect — SPEC-036's
  blocked contract-compiler gap in the `backstop-go-contracts-pack` repo — it was confirmed that
  `pkg/pack/engine` ALREADY has both primitives this needs, and that both are LIVE in dispatch, not
  merely declared:
  - `engine.InputModeNone` — documented at `pkg/pack/engine/binding.go:24-26` as *"the executable is
    the logic"*: it injects no rules and no config file. Handled in dispatch at
    `cmd/backstop/pack_gate.go:543` (`buildEngineArgs`, `case engine.InputModeNone: return nil, nil`).
  - `engine.ScopeKindProjectWide` — documented at `pkg/pack/engine/binding.go:55-57` as running the
    command ONCE project-wide, ignoring the scanned file set. Branched on in the FINDINGS path at
    `cmd/backstop/pack_gate.go:600` (inside `runFindingsEngine`, which begins at `:573`), and again
    in the parallel COVERAGE path at `cmd/backstop/pack_gate.go:440` (`dispatchPackCoverage`).

  So a pack CAN declare a script-based engine that runs once, project-wide, executes arbitrary
  logic — a `git log` / `git tag` query in this case — and emits SARIF, with **ZERO core binary
  changes**. The mechanism question is ANSWERED in the affirmative: it IS expressible.

  Stronger still, this is a SHIPPED pattern rather than an unused capability. Verified 2026-08-10:
  the installed `backstop-ai/go-toolchain` pack's `go-build` and `go-test` bindings already combine
  `input_mode: none` + `scope_kind: project-wide` + `project_target: "./..."` + a pack-owned
  `convert:` script that normalizes arbitrary command stdout into SARIF
  (`.backstop/packs/backstop-ai/go-toolchain/pack.yml:52-90`). A release-currency engine is that
  same construction pointed at `git` instead of `go`; the `convert:` script is the SARIF bridge and
  it is an existing seam, not a new one.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-3: option (c), the `go-distribution` PACK, as a
  project-wide script engine.** Decided on BOTH cost AND expressibility — the asymmetry caveat
  recorded at v0.1.1 (one measured option against two unmeasured) no longer carries the decision
  alone, because the mechanism evidence above removes the objection that made (c) uncertain in
  principle rather than merely unmeasured in cost. The two rejections are explicit: (a) a CORE
  COMMAND violates the zero-baked-checks law directly, which `backstop/self` guards; (b) a CI
  WORKFLOW is new hand-written pipeline debt of exactly the kind ISSUE-101 exists to retire. The
  OQ's fourth possibility — that DERIVATION and SURFACING might not want the same home — is resolved
  by unification: a project-wide script engine does both, computing the delta and emitting the
  warning as SARIF in one pass.

- **OQ-4 — THE PROPOSAL SURFACE AND THE TRIGGER. (RESOLVED 2026-08-10 → DD-4.)** Two coupled halves. The SURFACE: a standing
  GitHub issue that the proposer opens and updates in place (durable, commentable, visible from a
  phone — but external, and this repo does not otherwise use GitHub issues as a work surface); a
  gate warning only (zero new surface, already in the founder's daily path — but ephemeral, and it
  cannot carry proposed release notes); or something artifact-native (a `release` artifact type, or
  a proposal file in the repo — consistent with how everything else in this project is
  represented, at the cost of a new artifact type and its schema/validation weight). The TRIGGER:
  a founder word to the orchestrator (matches how work is actually driven today, requires no new
  infrastructure, but is a human obligation again — though a much smaller one than remembering the
  number); a `workflow_dispatch` (explicit, auditable, but a context switch to the GitHub UI); a
  label or a scheduled sweep that proposes without being asked. Note the trigger for the PROPOSER
  and the trigger for the RELEASE are different questions — the proposer can be scheduled and
  autonomous precisely BECAUSE it only proposes. Couples to OQ-2 (a caveat needs a place to be
  stated) and OQ-6 (a fleet-wide proposer probably cannot be a per-repo gate warning). No lean —
  founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-4: GATE WARNING + FOUNDER-WORD TRIGGER.** SURFACE: a
  WARNING-severity signal in `backstop gate` output — *"`main` is N commits / M closed issues ahead
  of `vX.Y.Z` (proposed: `vX.Y.Z+1`, patch)"* — chosen because it is ZERO NEW SURFACE and already in
  the founder's daily path. TRIGGER: ON-DEMAND ONLY, fired by the founder saying something like
  *"propose a release"* to an orchestrating agent, which runs the derivation fresh and reports back.
  EXPLICITLY REJECTED: a standing GitHub issue (this repo does not otherwise use GitHub issues as a
  work surface) and a scheduled job (new infrastructure this decision deliberately avoids). Note the
  founder ruled AGAINST the OQ's own observation that the proposer could safely be autonomous
  because it only proposes — the smaller-infrastructure option won anyway. The OQ-2 coupling is
  satisfied: the gate warning and the on-demand report are both places a caveat can be stated.

  **SUPERSEDED 2026-08-10 (v0.5.0, founder-ruled) — see the DD-4 correction.** Both halves are
  reversed. There is no gate surface (*"that does not belong in backstop gates... that is purely a
  ci-time consideration"*) and no trigger word: a CI job runs on every push to `main` and AUTO-TAGS
  when the delta is non-empty, because no seamless approval path exists (*"unless there's a seamless
  way for me to approve or deny, then it should just be auto released"*). The OQ-2 coupling this
  paragraph satisfied is moot — there is no caveat left to state.

- **OQ-5 — RELATIONSHIP TO BUNDLE-020 (pack ↔ core version compatibility). (RESOLVED 2026-08-10 →
  DD-5.)** BUNDLE-020 is
  `exploring` and owns the question of what a pack declares about the core versions it works with;
  its OQ-1 resolved (conditionally) toward named CAPABILITY CONTRACTS — a SET comparison rather
  than a version comparison. The first LIVE instance of a pack needing core-version awareness has
  now appeared: `go-distribution`'s warning tier depends on behavior that requires core ≥ 0.1.1.
  So: do versioning MACHINERY (this bundle: what version do we cut next) and version COMPATIBILITY
  (BUNDLE-020: which versions can work together) share a spine, or stay separate? The case for
  shared: both are about what a version NUMBER means, and if BUNDLE-020 lands capability contracts
  instead of version comparison, then a core release that adds a capability has a machine-visible
  reason to be a minor rather than a patch — which is a derivation signal this bundle would want.
  The case for separate: BUNDLE-020 governs a WIRE SEAM between two artifacts at runtime; this
  bundle governs a HUMAN WORKFLOW around cutting a tag; conflating them risks one bundle owning two
  problem spaces, which this project has been burned by before. A third reading: they are
  independent but one CONSUMES the other's output (BUNDLE-020's capability set is an input to this
  bundle's bump derivation). No lean — founder's, and it is a scope-boundary ruling as much as a
  design one.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-5: STAYS SEPARATE.** Versioning MACHINERY (this
  bundle: what version to cut) and version COMPATIBILITY (BUNDLE-020: which versions can work
  together) are NOT merged into one spine. The founder took the OQ's THIRD reading rather than
  either of the first two: they are independent, but if BUNDLE-020 lands capability contracts, that
  becomes an INPUT this bundle's derivation could consume LATER — a core release adding a named
  capability would have a machine-visible reason to be a minor rather than a patch. That is a future
  CONSUMPTION relationship, explicitly not a shared bundle or a shared spec. The rejected "shared
  spine" case would have had one bundle owning two problem spaces, which this project has been
  burned by before.

- **OQ-6 — MULTI-REPO: does this generalize to the pack fleet? (RESOLVED 2026-08-10 → DD-6.)** The pack repos version too and
  ship real releases (`go-toolchain` shipped `1.3.0` this week). Their release cadence is arguably
  MORE prone to the exact slip this bundle targets, because a pack fix that never gets tagged is a
  fix no consumer can `pack update` to — the fix→bump→relock flywheel stalls silently. But packs
  have NO artifact corpus: there are no issues, plans, or `delivered_by` chains in a pack repo, so
  the entire ledger-derived derivation (the candidate's part 1) has nothing to read there. Options:
  (a) backstop-core is SPECIAL — it has a corpus, so it gets the derived version, and the fleet
  gets only the currency WARNING (part 2), which needs nothing but git; (b) the machinery is
  corpus-OPTIONAL by design — full derivation where a corpus exists, degraded-but-useful currency
  reporting where it doesn't, which is also the answer to "what does a non-backstop consumer of
  `go-distribution` get" (OQ-3); (c) packs grow a lighter-weight signal of their own. Whether the
  fleet is in scope for THIS bundle or is a follow-on is itself part of the question. No lean —
  founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-6: OUT OF SCOPE for this bundle; an explicit
  FOLLOW-ON.** The reason is the one the OQ names: packs have NO artifact corpus — no issues, no
  plans, no `delivered_by` chains — so this bundle's ledger-derivation does not transplant to them
  as-is. Fleet-wide release-CURRENCY reporting, which needs only git and not the corpus, is recorded
  as a LIKELY FOLLOW-ON BUNDLE rather than built here. The embedded sub-question ("is the fleet in
  scope for THIS bundle") is therefore answered NO, which is the scope half of the ruling; the
  fleet's real exposure to the target failure mode — a pack fix that never gets tagged is a fix no
  consumer can `pack update` to, stalling the fix→bump→relock flywheel silently — is preserved as
  the motivation for the follow-on.

- **OQ-7 — THE ARTIFACT ↔ RELEASE JOIN (identified during authoring). (RESOLVED 2026-08-10 →
  DD-7.)** Ledger-derived semver and
  receipt-shaped release notes both require answering "which artifacts shipped in which release,"
  and NOTHING records that today. Two shapes: (a) DERIVED — compute it at proposal time from git
  (tag ancestry plus commit-to-artifact association), storing nothing, consistent with this
  project's standing preference for deriving state from the world rather than storing it; the risk
  is that the derivation depends on a commit↔artifact association that may be conventional rather
  than reliable, and it is recomputed (possibly differently) every time. (b) STORED — artifacts
  gain a `released_in: v0.1.2` field written at release time, making "what shipped in v0.1.2" a
  direct read and making the record durable and auditable; the cost is a schema change across
  artifact types, a write pass over the corpus at every release, and a new class of drift (a
  `released_in` that disagrees with the tags). This is the same derive-vs-store fork BUNDLE-019's
  OQ-3 records for run history, and it should probably be answered consistently with it. It is
  load-bearing: OQ-1's derivation and OQ-4's release-notes-as-receipts both sit on top of whatever
  this resolves to. No lean — founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-7: option (a), DERIVED — not stored.** No new
  `released_in:` field on artifacts. "Which artifacts shipped in which release" is COMPUTED at
  proposal time from git (tag ancestry plus commit-to-artifact association), consistent with this
  project's standing preference for deriving state from the world rather than storing new fields
  everywhere. The stored option's costs are what sank it: a schema change across artifact types, a
  write pass over the corpus at every release, and a NEW CLASS OF DRIFT (a `released_in` disagreeing
  with the tags) — a fresh instance of exactly the staleness problem `status_drift` exists to catch.
  The accepted cost is the one the OQ named: the derivation leans on a commit↔artifact association
  that is conventional rather than guaranteed. DD-8 is the mitigation — where the association cannot
  be made, the machinery says so out loud instead of silently dropping the commit. Note this also
  answers the OQ's cross-reference to BUNDLE-019's OQ-3 in the derive direction.

- **OQ-8 — CORPUS COVERAGE: what happens when code ships with no artifact behind it?
  (identified during authoring). (RESOLVED 2026-08-10 → DD-8.)** "Commit messages are vibes, artifacts are data" holds only
  insofar as the artifact corpus actually COVERS the code that changed. Ledger-derived semver
  inherits the corpus's coverage gaps exactly: a code change that lands with no closed issue is
  INVISIBLE to the deriver, which under-versions the release — and an under-versioned release is a
  silent-green failure of precisely the class this project exists to defeat, only now pointed at
  consumers instead of at the gate. OQ-1(c) asks about corpus changes with no code; this asks the
  inverse, and it is the more dangerous direction. What should the machinery do when it finds code
  commits since the last tag with no traceable artifact? Options: (a) REFUSE to compute a bump
  (fail-closed — strongest honesty, but likely fires constantly given fix-up commits, doc-adjacent
  code edits, and mechanical renames); (b) WARN and compute anyway, naming the uncovered commits in
  the proposal so the founder adjudicates (matches loud-≠-blocking, and turns the gap into a
  visible line item rather than a silent omission); (c) treat uncovered code commits as an implicit
  patch-tier signal (pragmatic, but it invents data — the machinery would be guessing, which is the
  vibes problem re-entering through the back door). Note that whatever is chosen here doubles as
  the ANSWER to "is the corpus complete enough to release from," which may be a more useful signal
  than the bump itself. No lean — founder's.

  **RESOLUTION (2026-08-10, founder-ruled) → DD-8: option (b), WARN AND COMPUTE ANYWAY, naming the
  uncovered commits.** When code commits exist since the last tag with no traceable artifact behind
  them, the machinery does NOT refuse to propose a bump and does NOT silently guess a patch-tier
  signal. It computes the proposal and explicitly LISTS the uncovered commits as a named caveat in
  the output, so the founder can adjudicate whether the corpus is complete enough to trust. Both
  alternatives were rejected for opposite reasons: (a) REFUSING is too likely to fire constantly on
  fix-up commits, doc-adjacent edits, and mechanical renames — the same never-fires failure DD-2
  guards against; (c) treating uncovered commits as an implicit patch signal REINTRODUCES THE VIBES
  PROBLEM through the back door, inventing data the machinery does not have and under-versioning
  silently. The OQ's closing observation is upheld and promoted into DD-8: the uncovered-commit list
  doubles as the answer to "is the corpus complete enough to release from," and is a first-class
  part of the derivation's output contract rather than an error path.

**Maturity stays `exploring` (2026-08-10).** All eight OQs are resolved and DD-1..DD-8 are recorded,
but resolution is not promotion, and promotion is the founder's call. *Updated at v0.3.0:* the
`requirements[]` array this note recorded as absent has since been drafted (REQ-001..013); maturity
is unchanged, and the promotion call remains entirely the founder's.

## Spec Seeds

Provisional and explicitly contingent — the decomposition below assumes the candidate direction
survives OQ-1..OQ-8 roughly intact, and OQ-3 in particular can relocate every one of these. Listed
in the order they would most likely be built, not as committed scope.

**UPDATE (2026-08-10): the contingency has largely DISCHARGED, and the seeds survive intact.** The
candidate direction did survive OQ-1..OQ-8 (see DD-1..DD-8), so these four seeds stand as written
and are no longer contingent on the load-bearing unknown. Specifically: DD-3 places the first two
seeds in the `go-distribution` pack as a project-wide script engine, which RESOLVES the "if a pack
rule cannot express a git query" branch the second seed hedges against — it can, so that seed does
not change shape. DD-4 settles the third seed's open shape: the trigger is an orchestrator-invoked
agent (a founder word), not a workflow and not an artifact writer. DD-6 confirms the fourth seed's
own guess about itself — the fleet case is a FOLLOW-ON BUNDLE, not scope here. The seeds have NOT
been rewritten to fold these in, because they remain provisional in the bundle sense: this bundle is
still `exploring`, and a spec-author should read the seeds together with DD-1..DD-8 rather than
treating either alone as the scope.

**UPDATE (2026-08-10, v0.3.0): the first three seeds now carry requirements; the fourth
deliberately does not.** REQ-001..009 attach to release-delta derivation, REQ-010..011 to
release-currency surfacing, REQ-012 to the proposer and its trigger, and REQ-013 is the
cross-cutting home all three share. Fleet generalization contributes NO requirement, per DD-6 —
it stays recorded here as a follow-on and nothing else. See *Draft Requirements* for the mapping.

**UPDATE (2026-08-10, v0.5.0, founder-ruled pivot): the second seed is DISSOLVED and the third is
REDEFINED.** Seed 2 (*release-currency surfacing*) no longer exists as scope — with release currency
out of `backstop gate` there is no standing signal to build, and its two requirements (REQ-010,
REQ-011) are retired. Seed 3 is no longer "the proposer and its trigger" but **the CI auto-tagger**:
a workflow job on every push to `main` that invokes seed 1's script and tags when the delta is
non-empty (REQ-012 v2.0.0) — no readiness check, no proposal surface, no waiting. Seed 1 is
unchanged in substance but is now a plain script rather than a gate engine (REQ-013 v2.0.0), and it
no longer reports completed-plan state. Seed 4 (fleet) is still a follow-on and still out of scope.
The seed text below is preserved unedited.

- **Release-delta derivation (home TBD per OQ-3)** — the read-only computation: given the last
  release tag and `HEAD`, produce the delta (commits, closed issues, completed plans since the
  tag), the proposed semver bump from the OQ-1 rules, and the receipt-shaped release-note body.
  Depends on OQ-7's join answer for how artifacts are associated with releases at all, and carries
  OQ-8's uncovered-commit handling as part of its output contract rather than as a separate
  feature. Deliberately read-only and side-effect-free — it computes and reports; it never tags.

- **Release-currency surfacing (gate warning and/or `go-distribution` pack rule)** — the standing
  forcing signal: "`main` is N commits / M closed issues ahead of `vX.Y.Z`." Warning severity per
  loud-≠-blocking. This is the seed most exposed to OQ-3: if a pack rule cannot express a git
  query, this either becomes a core surface (with the baked-check tension that entails) or changes
  shape entirely. It is also the seed that DEGRADES most gracefully for consumers with no artifact
  corpus (OQ-6b), since currency reporting needs only git.

- **The proposer and its trigger** — the scheduled/on-demand agent or workflow that runs the
  derivation, applies the OQ-2 in-flight readiness check, and publishes/updates the OQ-4 proposal
  surface, then waits for a human word. Strictly downstream of the first two seeds and of OQ-2 and
  OQ-4 both; the trigger and surface choices determine whether this is a workflow, an artifact
  writer, or an orchestrator-invoked agent, so it cannot be scoped before those resolve.

- **Fleet generalization (contingent on OQ-6, likely a follow-on)** — extending currency reporting
  to the pack repos, which have git and tags but no artifact corpus. Recorded so the fleet case is
  not lost; explicitly NOT committed as scope here, since OQ-6 may rule backstop-core special or
  route this to its own bundle.

## Notes / Ideas

- **Release notes as receipts, not prose.** If the bump derives from closed issues with
  `delivered_by` chains, then the changelog is a QUERY RESULT, and every line is traceable back to
  an issue, a plan, and the commits that delivered it. That is a materially stronger artifact than
  a hand-written changelog, and it is nearly free once the derivation exists — the same query
  produces both the number and the notes. It may also be the most compelling demo surface this
  capability has: a release whose notes are provably complete relative to the corpus.

- **The clean-zero window is now, and it will not recur.** `main` is currently AT `v0.1.1` with
  zero commits of drift. Building the machinery against a zero delta means there is no backlog to
  reconcile and no judgment call about how to classify a year of un-released history. Every day of
  delay makes the first run of this machinery a harder problem than it needs to be.

- **This is the same enforcement thesis pointed at a new target.** Everywhere else in this project,
  the pattern is: make the un-done state machine-visible so that skipping it requires ignoring a
  signal rather than merely forgetting. `status_drift` does it for stale artifacts,
  `requirement_traceability` for unclaimed requirements, the ratchet for un-fixed violations.
  Release currency is the same shape applied to distribution — and the fact that it is currently a
  pure memory obligation is the anomaly, not the norm.

- **Watch for the ceremony trap.** A machinery that nags on every gate run about an unreleased
  `main` in a repo that is mid-lane by default will be tuned out within a week, and a tuned-out
  warning is worse than no warning — it teaches the founder to ignore a surface that also carries
  real signals. Whatever OQ-2 and OQ-4 resolve to should be measured against "would this still be
  read on day 30," which is the practical form of the discipline-≠-ceremony principle.

## Version History

- 0.5.0 (2026-08-10): **FOUNDER-RULED PIVOT — DD-2 and DD-4 SUPERSEDED IN FULL. The capability
  becomes a PURE CI-TIME AUTO-TAGGER: no `backstop gate` involvement, no approval step, no in-flight
  detection.** Maturity unchanged at `defined`; DD-1, DD-3, DD-5, DD-6, DD-7 and DD-8 all still hold.
  Original DD-2/DD-4 text is preserved unedited with dated corrections appended, per the
  note-supersedes convention.

  **DD-4 superseded — surface AND trigger both reversed.** Release currency is out of the gate
  entirely: *"that does not belong in backstop gates... that is purely a ci-time consideration."* And
  there is no approval step, because the founder has no bandwidth for one — *"i am way too busy and
  overloaded to switch over to github to manually review and approve"*, with the rule drawn from it:
  **"unless there's a seamless way for me to approve or deny, then it should just be auto
  released."** A proposal nobody has bandwidth to read is a queue, not a safety mechanism. The
  mechanism is now: a CI job in backstop-core's own `.github/workflows/` runs on EVERY PUSH TO
  `main`, invokes the `go-distribution` pack's delta-derivation script directly, and — when the delta
  is NON-EMPTY — computes the semver per DD-1 and TAGS AND PUSHES the tag itself. The shipped
  tag-triggered `release.yml` pipeline (ISSUE-087) takes over unchanged, and ITS existing
  `require-green-ci` gate is the real safety rail: an auto-tag builds and publishes only once CI is
  green on that commit, exactly as for a hand-pushed tag.

  **DD-2 superseded — no in-flight detection exists at all.** Not a caveat, not a block: nothing
  reads plan-completion state or any other in-flight indicator. Founder's reasoning: *"in-flight work
  should not be included in a release... in flight work should be irrelevant here... easier to get
  more disciplined on branching vs building in all kinds of weird shit into the release pipeline."*
  The premise becomes **whatever is on `main` is trusted as release-ready, full stop** — the same
  trust every other automated mechanism here already extends to `main` — and unfinished work is kept
  out of a release by BRANCHING DISCIPLINE, not by pipeline detection. The DD-2 question is dissolved
  rather than re-answered.

  **DD-3's home holds; only its invocation simplifies.** The derivation still ships in the
  `go-distribution` pack with ZERO core binary changes and no core command. But it is no longer
  dispatched through `backstop gate`'s pack-engine mechanism, so the `scope_kind: project-wide` +
  `input_mode: none` + SARIF `convert:` construction is no longer required — nothing consumes the
  output as a gate finding. It is a plain script a workflow step invokes (`bash
  .backstop/packs/backstop-ai/go-distribution/scripts/<script>.sh` or similar). DD-3's rejection of a
  "CI workflow" home is correspondingly read narrowly: a hand-written workflow OWNING the derivation
  logic is still rejected; a thin workflow step INVOKING the pack's script is the intended shape.

  **Requirements: three retired, two rewritten, one clarified — ten survive.** RETIRED (IDs not
  reused): REQ-008 (in-flight caveat — removed rather than reworded, since the mechanism it described
  no longer exists), REQ-010 and REQ-011 (the `backstop gate` signal and its WARNING severity — with
  no gate surface there is no signal content and no severity to specify; the *release-currency
  surfacing* seed now carries no requirements). INVERTED: REQ-012 → v2.0.0 — was on-demand,
  founder-word-triggered, and *"must never fire the release itself"*; is now automatic on every push
  to `main`, approval-free, and explicitly DOES tag and push. REWRITTEN: REQ-013 → v2.0.0 — same
  pack home and same zero-core-changes constraint, but the engine-binding construction is dropped and
  its ban on a hand-written CI workflow is void, since a CI workflow is now the intended invoker.
  CLARIFIED: REQ-001 → v1.1.0 — still read-only and still never tags, but now as a separation of
  concerns between a computing script and an acting workflow rather than as evidence a human is in
  the loop; "completed plans" dropped from its output contract. `solution.approach` rewritten to the
  post-pivot direction. Superseded prose everywhere else (DD-2, DD-4, the OQ-2/OQ-4 resolutions, the
  OQ summary list, the v0.3.0 *Draft Requirements* preamble) is preserved with dated corrections
  appended, never deleted.

- 0.4.0 (2026-08-10): **PROMOTED `exploring` → `defined` by founder ruling.** No content change:
  every structural precondition for `defined` was already in place from today's two prior passes —
  all eight OQs resolved into DD-1..DD-8 (v0.2.0), 13 drafted requirements REQ-001..013 in
  frontmatter with a matching `## Draft Requirements` section, `solution.approach` decided, and
  Spec Seeds / Design Decisions / Version History sections present (v0.3.0). Both earlier entries
  correctly withheld the promotion as the founder's own call; this version records that call being
  made. Consequently the statements at v0.2.0 and v0.3.0 — and the sentence in `solution.approach`
  reading "Maturity deliberately stays `exploring`" — are SUPERSEDED as of this version: they
  described the state accurately when written and are preserved unedited per the dated-correction
  convention, but maturity is now `defined`. Nothing else was touched: no DD, requirement, OQ
  resolution, or spec seed was altered, added, or reopened.

- 0.3.0 (2026-08-10): **Draft Requirements authored** — 13 formal requirements (REQ-001..013)
  added to frontmatter plus a matching `## Draft Requirements` section. Each traces to a design
  decision recorded at v0.2.0 or to one of the first three spec seeds; no new scope is introduced,
  and no OQ is reopened. Partitioned non-overlappingly: REQ-001..009 to release-delta derivation
  (read-only posture; the DD-1 bump rules split into their settled core, the acknowledged narrow
  three-type default, the pre-1.0 major refusal, and the corpus-only reporting case; the derived
  git join with its explicit ban on a stored `released_in:` field; the uncovered-commit caveat;
  the in-flight caveat with caveat SPECIFICITY written as a testable constraint; receipt-shaped
  release notes), REQ-010..011 to release-currency surfacing (the gate signal's content, and
  warning severity stated in the pack severity contract's own SARIF terms so it is checkable
  against the mechanism), REQ-012 to the on-demand founder-word proposer with DD-4's rejected
  alternatives written in as prohibitions, and REQ-013 as the cross-cutting HOME — the
  `go-distribution` pack project-wide script engine, zero core binary changes, no core command and
  no hand-written CI workflow. Recorded what is deliberately NOT a requirement here: fleet
  generalization (the fourth seed and DD-6 — it contributes none), pack↔core version compatibility
  (DD-5, BUNDLE-020's), how a release is BUILT and PUBLISHED (DIR-001 / ISSUE-087, downstream of
  the tag and untouched), and any better mapping for DD-1's three defaulted `issue.type` members
  (REQ-003 pins the current default and requires it be refinable rather than guessing a
  replacement). Stale "no `requirements[]` yet" statements in `solution.approach`, Current
  Thinking, the Open Questions preamble and closing note, and the Spec Seeds preamble were
  corrected in place with dated updates rather than deleted. **Maturity unchanged: `exploring`** —
  the structural precondition for `defined` is now met, but promotion remains the founder's own
  call and no agent takes it.

- 0.2.0 (2026-08-10): **ALL EIGHT open questions RESOLVED by founder ruling; eight design decisions
  recorded (DD-1..DD-8, one per OQ); `solution.approach` moved from UNDECIDED to a decided
  direction. Maturity deliberately UNCHANGED at `exploring`** — resolving the OQs settles the shape,
  but promotion to `defined` is a separate founder call and additionally requires a drafted
  `requirements[]` array, which this version does NOT add. No requirements recorded, no promotion
  attempted. Original OQ text and reasoning preserved verbatim per the dated-correction convention;
  resolutions are appended, and the superseded "What is DECIDED vs OPEN" paragraph is block-quoted
  with its correction beneath rather than deleted.

  **OQ-1 (semver derivation rules) → DD-1, a conservative first cut labelled as one.**
  `issue.type: bug` → PATCH, `issue.type: enhancement` → MINOR, highest tier in the delta wins. No
  auto-major before 1.0 — resolved by DEFERRING to standard semver (which already holds the major
  position meaningless pre-1.0) rather than modelling breakage separately, so the machinery simply
  REFUSES to propose one. `technical-debt`, `question`, and `policy-violation` are left unmapped and
  default to PATCH alongside `bug`, since none represents a user-facing capability change — recorded
  explicitly as NARROWER THAN IDEAL and refinable later, NOT as a permanent ruling on those three.
  Corpus-only commits (touching only `issues/`, `plans/`, `bundles/`, `directives/`, docs — no code)
  count toward the "how far ahead" currency REPORTING but never trigger a bump on their own.

  **OQ-2 (in-flight detection) → DD-2: CAVEAT ONLY, never a hard block.** The proposal still
  computes and still fires when an open plan has partially-committed phases, annotated with a caveat
  naming the SPECIFIC plan and its completion fraction ("PLAN-ISSUE-XXX is mid-lane, 3/7 tasks
  committed — releasing now would ship partial work"). Matches the standing loud-≠-blocking law, but
  the decisive argument was empirical: this repo is nearly always mid-lane, so hard-blocking risks a
  proposer that effectively NEVER FIRES — restoring the very memory obligation the bundle exists to
  eliminate. Hard-blocking explicitly rejected. The counter-risk (a caveat learned-past) is accepted
  rather than dissolved, with caveat SPECIFICITY recorded as the mitigation.

  **OQ-3 (where the machinery lives) → DD-3: the `go-distribution` PACK as a project-wide SCRIPT
  ENGINE — the load-bearing decision, and now decided on BOTH expressibility AND cost.** This
  version adds NEW mechanism evidence the bundle did not previously have, confirmed live 2026-08-10
  while scoping an unrelated defect (SPEC-036's blocked contract-compiler gap in the
  `backstop-go-contracts-pack` repo): `pkg/pack/engine` already has BOTH primitives this needs and
  both are LIVE in dispatch — `engine.InputModeNone` ("the executable is the logic",
  `pkg/pack/engine/binding.go:24-26`) handled at `cmd/backstop/pack_gate.go:543` in
  `buildEngineArgs`, and `engine.ScopeKindProjectWide` (`binding.go:55-57`) branched on at
  `cmd/backstop/pack_gate.go:600` inside `runFindingsEngine` (which begins at `:573`) and again in
  the parallel COVERAGE path at `cmd/backstop/pack_gate.go:440` (`dispatchPackCoverage`). This
  directly ANSWERS OQ-3's stated uncertainty ("a git query has no file to match, so whether the
  check is EXPRESSIBLE as a pack rule at all is a real, unanswered mechanism question"): it IS
  expressible — a pack can declare a script-based engine that runs once, project-wide, executes
  arbitrary logic (a git log/tag query) and emits SARIF, with ZERO core binary changes. Recorded
  additionally, and stronger than a bare capability claim: this is a SHIPPED pattern — the installed
  `backstop-ai/go-toolchain` pack's `go-build` and `go-test` bindings already combine
  `input_mode: none` + `scope_kind: project-wide` + `project_target: "./..."` + a pack-owned
  `convert:` script normalizing arbitrary stdout to SARIF
  (`.backstop/packs/backstop-ai/go-toolchain/pack.yml:52-90`), so a release-currency engine is that
  construction pointed at `git` instead of `go` and the `convert:` script is an existing SARIF seam,
  not a new one. Combined with the 2026-07-29 cost evidence already carried at v0.1.1
  (implementer-101: fits as a second recipe, no new engine work, since the pack already declares the
  provisioned engine binding `recipe apply` demands), OQ-3 is decided on both grounds rather than on
  cost alone — the v0.1.1 asymmetry caveat no longer carries the decision by itself. Core-command
  and CI-workflow homes explicitly REJECTED: the former violates zero-baked-checks directly, the
  latter is new hand-written pipeline debt of exactly the kind ISSUE-101 exists to retire. The OQ's
  fourth possibility (derivation and surfacing may not want the same home) resolves by unification —
  one project-wide script engine does both in one pass.

  **OQ-4 (proposal surface + trigger) → DD-4: GATE WARNING + FOUNDER-WORD TRIGGER.** Surface: a
  WARNING-severity signal in `backstop gate` ("`main` is N commits / M closed issues ahead of
  `vX.Y.Z` (proposed: `vX.Y.Z+1`, patch)") — zero new surface, already in the founder's daily path.
  Trigger: ON-DEMAND ONLY, the founder saying something like "propose a release" to an orchestrating
  agent that runs the derivation fresh and reports back. REJECTED: a standing GitHub issue (this
  repo does not otherwise use GitHub issues as a work surface) and a scheduled job (new
  infrastructure this decision deliberately avoids) — the founder ruled for smaller infrastructure
  even though the OQ itself noted the proposer could safely be autonomous because it only proposes.

  **OQ-5 (relationship to BUNDLE-020) → DD-5: STAYS SEPARATE.** Versioning MACHINERY (what version
  to cut) and version COMPATIBILITY (which versions can work together) are not merged into one
  spine. The founder took the OQ's THIRD reading: independent, but if BUNDLE-020 lands capability
  contracts that becomes an INPUT this bundle's derivation could consume later (a core release
  adding a capability would have machine-visible reason to be minor, not patch) — a future
  consumption relationship, not a shared bundle or spec. Avoids one bundle owning two problem
  spaces, a shape this project has been burned by before.

  **OQ-6 (multi-repo pack fleet) → DD-6: OUT OF SCOPE here; explicit FOLLOW-ON.** Packs have no
  artifact corpus (no issues, plans, or `delivered_by` chains), so the ledger-derivation does not
  transplant to them as-is. Fleet-wide release-currency REPORTING — which needs only git, not the
  corpus — is recorded as a likely follow-on bundle rather than built here, with the fleet's real
  exposure preserved as its motivation (an untagged pack fix is a fix no consumer can `pack update`
  to, stalling the fix→bump→relock flywheel silently).

  **OQ-7 (artifact↔release join) → DD-7: DERIVED, not stored.** No new `released_in:` field on
  artifacts; "which artifacts shipped in which release" is computed at proposal time from git (tag
  ancestry + commit-to-artifact association), consistent with the standing preference for deriving
  state from the world over storing new fields everywhere. The stored option was sunk by its costs:
  a schema change across artifact types, a write pass over the corpus every release, and a new class
  of drift (`released_in` disagreeing with the tags) — a fresh instance of the staleness problem
  `status_drift` exists to catch. Accepted cost: the commit↔artifact association is conventional
  rather than guaranteed, mitigated by DD-8.

  **OQ-8 (corpus coverage gaps) → DD-8: WARN AND COMPUTE ANYWAY, naming uncovered commits.** When
  code commits exist since the last tag with no traceable artifact, the machinery neither refuses to
  propose (too likely to fire constantly on fixup/doc-adjacent commits — the same never-fires
  failure DD-2 guards against) nor silently guesses a patch-tier signal (that reintroduces the vibes
  problem through the back door and under-versions silently). It computes the proposal and lists the
  uncovered commits as an explicit named caveat for founder adjudication. The OQ's closing
  observation is promoted into the DD: that list doubles as the answer to "is the corpus complete
  enough to release from," and is a first-class part of the derivation's output contract rather than
  an error path.

  Also in this version: `solution.approach` rewritten from UNDECIDED to the decided direction
  (ledger-derived semver via a `go-distribution` pack script engine, gate-warning surfacing,
  founder-word trigger, caveat-only in-flight handling); the *What is DECIDED vs OPEN* subsection
  superseded with a dated correction that preserves the original as a block quote and names DD-1's
  three-unmapped-types default as the narrowest known soft spot in the decided design; and the Spec
  Seeds preamble annotated to record that its OQ-3 contingency has discharged and all four seeds
  survive intact (seeds deliberately NOT rewritten — they remain provisional and should be read
  together with DD-1..DD-8).

- 0.1.1 (2026-07-29): Evidence append under OQ-3 — no OQ resolved, no design decision recorded,
  maturity unchanged at `exploring`. Recorded implementer-101's measured assessment of the
  pack-recipe option (c), made immediately after building the `go-distribution` pack's first
  recipe: an auto-version capability fits as a SECOND recipe (compute-next-semver payload +
  propose-or-tag-on-trigger workflow payload) with paired enforcement rules; the pack already
  declares the provisioned engine binding `recipe apply` demands, so a second recipe adds no
  engine work; and the trigger half is a workflow payload, the artifact shape the pack's existing
  rules already guard. Recorded with two explicit caveats so it is not mistaken for a ruling: the
  evidence is ASYMMETRIC (options (a) core command and (b) CI workflow remain unassessed by
  anyone, so this is one measured option against two unmeasured), and it addresses COST rather
  than the EXPRESSIBILITY concern OQ-3 raises — a consumer-applied recipe payload is a different
  mechanism from a rule answering a git query, so "a git query has no file to match" is untouched
  by it.

- 0.1.0 (2026-07-29): Initial bundle at `exploring`. Founder framing (2026-07-29, preserved
  verbatim in Current Thinking): manual tag-driven releasing "goes directly against my 'eliminate
  cognitive overhead' principle" — remembering whether to tag `main` and at what semver "is going
  to be very easy to let slip without some kind of machinery forcing that decision." The release
  DECISION stays human; the REMEMBERING and the COMPUTING become machinery. Grounded against
  verified repo state: `v0.1.0` (`a0fb366`) and `v0.1.1` (`87b12cf`) both exist via the ISSUE-087
  pipeline, `87b12cf` is currently `HEAD` (zero drift today), and no auto-versioning exists
  anywhere in this repo or the pack fleet. Recorded the CANDIDATE direction as candidate, not
  decision — ledger-derived semver over the typed artifact corpus (`issue.type`, `delivered_by`
  chains, plan completion; the gate already reads this corpus per `pkg/gate/result.go:23`
  `ledger_integrity`), a warning-severity release-currency signal in the gate as the forcing
  surface, and a proposer with a human trigger that must read in-flight state (the live
  ISSUE-104/105 severity-contract example: an auto-tag between them landing would have released a
  half-fixed contract). Recorded the founder's `slotly` `auto-version.yml` PR-label prior art and
  why it does not transplant (backstop-core commits directly to `main`, has no PRs, and runs high
  agent commit volume — the transplantable part is the ergonomics, not the mechanism). Recorded
  EIGHT open questions, none pre-resolved: OQ-1 (semver derivation rules, incl. the five-member
  `issue.type` enum, pre-1.0 breaking semantics, and corpus-only changes), OQ-2 (in-flight
  detection and whether it hard-blocks or caveats), OQ-3 (where the machinery lives — core command
  vs CI workflow vs `go-distribution` pack — and what each does to the zero-baked-checks law and to
  corpus-less consumers), OQ-4 (proposal surface and trigger), OQ-5 (shared spine with BUNDLE-020
  or separate, given `go-distribution`'s core ≥ 0.1.1 warning tier as the first live pack↔core
  version dependency), OQ-6 (multi-repo generalization to a fleet with no artifact corpus), plus
  two identified during authoring — OQ-7 (the artifact↔release join: derived vs a stored
  `released_in` field, the same fork BUNDLE-019 OQ-3 records) and OQ-8 (corpus coverage: what the
  machinery does when code ships with no artifact behind it, the inverse of OQ-1(c) and the more
  dangerous direction, since it under-versions silently). Four provisional spec seeds (delta
  derivation; currency surfacing; the proposer and its trigger; contingent fleet generalization),
  all explicitly contingent on OQ-3. No design decisions and no requirements recorded — at
  `exploring`, recording DDs for an uninterrogated direction would be pre-resolution by another
  name. The founder drives OQ resolution and promotion.

## References

- **DIR-001 (Release Workflow)** — `directives/DIR-001-release-workflow.directive.md`, status
  `queued`. The framing directive for release infrastructure; its "Release-pipeline follow-on
  routings" section (2026-07-28) carries the `go-distribution` pack framing, the ratified
  `backstop-ai/homebrew-tap` coordinates, linux/arm64 as a shipped fourth platform, and the
  formula-over-cask ruling. Still open after ISSUE-087 for the self-gating fast-follow. This bundle
  is adjacent to it, not inside it: DIR-001 owns how a release is BUILT and PUBLISHED; this bundle
  owns whether and when one is CUT, and at what number.
- **ISSUE-087 / PLAN-ISSUE-087 (CI-Driven Release Pipeline)** — the shipped tag-triggered pipeline
  this bundle sits on top of: goreleaser four-platform builds, the tag-integrity gate, version
  stamping that survives both goreleaser ldflags and `go install @vX.Y.Z`, and Homebrew formula
  publication, all executed for real on 2026-07-28. The pipeline's TRIGGER (a human pushing a tag)
  and its VERSION (a human choosing a number) are exactly what this bundle questions — everything
  downstream of the tag is already solved and is not reopened here.
- **ISSUE-101 (Go Distribution Pack — Productize the Release Trinity)** — `open`, filed 2026-07-29.
  Productizes ISSUE-087's hand-written trinity into `backstop-ai/go-distribution` (recipes plus
  semgrep rules that assert release invariants survive consumer adaptation). The leading candidate
  HOME for parts of this bundle's machinery (OQ-3), and the source of the concrete pack↔core
  version dependency that motivates OQ-5. Its rules are calibrated by the same loud-≠-blocking law
  this bundle's currency signal would be.
- **BUNDLE-020 (Pack Core Version Compatibility)** — `exploring`, v0.2.0. The adjacency question of
  OQ-5: it owns what a pack DECLARES about compatible core versions (resolving conditionally toward
  named capability contracts and SET comparison rather than version comparison), while this bundle
  owns what version core CUTS next. Its resolved DD-4..DD-7 and its still-open OQ-2/3/4/6 are the
  material for deciding whether the two share a spine.
- **BUNDLE-019 (Runbooks)** — cited for OQ-7 specifically: its OQ-3 (recurring-run history —
  rich ledger vs minimal record vs derived-only) is the SAME derive-vs-store fork this bundle's
  artifact↔release join question raises. The two should probably be answered consistently.
- **ISSUE-104 / ISSUE-105 (severity-contract hops, closed 2026-07-29)** — the live in-flight
  example grounding OQ-2: two hops of one severity contract, where a tag fired between them landing
  would have shipped consumers a half-fixed contract. Both are ancestors of `v0.1.1`.
- **`slotly` `auto-version.yml` (founder prior art)** — PR-label-driven auto-versioning from the
  founder's prior repo. Real working precedent for machinery-decides-the-number ergonomics; not
  transplantable here because it keys on pull requests and backstop-core has none. See Current
  Thinking for the full read.
- **`backstop/self` pack** — enforces the zero-baked-checks law that OQ-3 has to answer to: a
  release-currency check living in the CLI binary is a baked check, and every check is supposed to
  come from a pack.
