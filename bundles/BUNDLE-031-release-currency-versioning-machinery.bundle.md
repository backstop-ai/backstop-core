---
title: "Release Currency Versioning Machinery"
number: BUNDLE-031
created: "2026-07-29"
schema_version: bundle/v2

bundle:
  name: release-currency-versioning-machinery
  version: "0.1.0"
  created: "2026-07-29"
  category: infrastructure

status:
  maturity: exploring

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

solution:
  approach: >
    UNDECIDED — this bundle is `exploring` and records a CANDIDATE direction to be interrogated,
    not a chosen design. The candidate has three parts, each of which the open questions can
    reject or reshape. (1) LEDGER-DERIVED SEMVER: since the last tag, the artifact corpus already
    knows what shipped — closed issues carry a typed `issue.type` (`bug` / `technical-debt` /
    `enhancement` / `question` / `policy-violation`), completed plans carry `delivered_by` chains
    back to issues and specs, and the gate already reads this corpus in its `ledger_integrity` and
    `requirement_traceability` steps. Derive the bump from those typed signals rather than from
    commit messages — "commit messages are vibes, artifacts are data" — and release notes fall out
    as receipts of the same derivation. (2) A RELEASE-CURRENCY CHECK as the forcing surface: a
    warning-severity signal ("`main` is N commits / M closed issues ahead of `vX.Y.Z`") raised
    where the founder already looks every day, which is the gate — loud-but-not-blocking, per the
    standing principle, and possibly a PACK rule (the `go-distribution` pack, ISSUE-101) rather
    than core, per the zero-baked-checks law. (3) A PROPOSER with a HUMAN TRIGGER: scheduled or
    on-demand, it computes the delta and the proposed bump, publishes/updates a standing
    release-proposal surface, and waits — the founder fires it with one word. The proposer must
    read IN-FLIGHT state, not just the git delta: an open plan mid-lane means not-yet. The live
    example is concrete — an auto-tag fired between ISSUE-104 and ISSUE-105 landing would have
    released a HALF-FIXED severity contract. What is genuinely open: the derivation rules
    themselves (OQ-1), what makes `main` not-release-ready (OQ-2), where the machinery lives given
    the zero-baked-checks law (OQ-3), the proposal surface and trigger (OQ-4), the relationship to
    the BUNDLE-020 pack↔core compatibility spine (OQ-5), whether this generalizes to the pack
    fleet (OQ-6), how artifacts JOIN to releases at all (OQ-7), and what the machinery does when
    the corpus does not cover the code that shipped (OQ-8). No design decisions and no
    requirements are recorded yet; the founder drives OQ resolution and promotion.
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

### What is DECIDED vs OPEN

DECIDED (founder framing, not to be re-litigated): the release DECISION stays human — this is not
a request for continuous deployment or auto-tagging; the remembering and the computing are the
machinery's job, and the trigger is a human word. Everything else — including all three parts of
the candidate direction above — is OPEN. No design decisions are recorded in this bundle, and no
requirements: at `exploring`, recording DDs for a direction the founder has not interrogated would
be pre-resolution by another name. OQ-1..OQ-8 below are the genuine forks. The founder drives
resolution and promotion.

## Open Questions

Not pre-resolved. Numbered sequentially; each carries its reasoning and its couplings. No leans are
recorded where the founder's judgment is the whole content of the answer; where an existing
standing principle constrains the space, it is cited so the founder can rule with it in view.

- **OQ-1 — SEMVER DERIVATION RULES.** Which artifact signals map to which bump? The obvious first
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

- **OQ-2 — IN-FLIGHT DETECTION: what makes `main` NOT release-ready?** Candidate signals, each
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

- **OQ-3 — WHERE THE MACHINERY LIVES.** Three homes, each with a different consequence for the
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

- **OQ-4 — THE PROPOSAL SURFACE AND THE TRIGGER.** Two coupled halves. The SURFACE: a standing
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

- **OQ-5 — RELATIONSHIP TO BUNDLE-020 (pack ↔ core version compatibility).** BUNDLE-020 is
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

- **OQ-6 — MULTI-REPO: does this generalize to the pack fleet?** The pack repos version too and
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

- **OQ-7 — THE ARTIFACT ↔ RELEASE JOIN (identified during authoring).** Ledger-derived semver and
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

- **OQ-8 — CORPUS COVERAGE: what happens when code ships with no artifact behind it?
  (identified during authoring).** "Commit messages are vibes, artifacts are data" holds only
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

Maturity stays `exploring`. No OQs pre-resolved, no design decisions recorded, no requirements —
those await founder-driven resolution.

## Spec Seeds

Provisional and explicitly contingent — the decomposition below assumes the candidate direction
survives OQ-1..OQ-8 roughly intact, and OQ-3 in particular can relocate every one of these. Listed
in the order they would most likely be built, not as committed scope.

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
