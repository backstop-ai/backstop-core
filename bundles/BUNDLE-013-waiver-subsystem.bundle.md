---
title: "Waiver Subsystem — Accountable, Engine-Neutral Suppression"
number: BUNDLE-013
created: "2026-07-09"
schema_version: bundle/v2

bundle:
  name: waiver-subsystem
  version: "0.1.0"
  created: "2026-07-09"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    Backstop has no accountable, gate-visible way to say "I acknowledge this
    finding and am choosing not to fix it right now." Today an author has
    exactly two escape valves, and both are bad. (a) Engine-native
    suppressions — `// nosemgrep` (semgrep), `//nolint` (golangci-lint),
    `//lint:ignore` (staticcheck), `#gitleaks:allow` (gitleaks) — are
    engine-SPECIFIC and suppress the finding INSIDE the engine before
    backstop's SARIF ever sees it, so the suppression is invisible to the
    gate's own accounting. There is no ledger of what was suppressed, no
    justification, no expiry, and each engine has its own dialect. (b) Silent
    baseline grandfathering (baseline.json) hides pre-existing debt with no
    justification and no expiry, and by design the baseline is
    gitignored/CI-regenerated, so it is not a durable record of a human
    decision. Neither valve is backstop-native, engine-neutral, tracked,
    justified, or loud. A waiver is meant to be that fourth thing: a
    first-class, engine-neutral, tracked, justified, gate-visible
    "chosen not to fix," so that every deferral is accounted for in one
    ledger instead of scattered across engine dialects and hidden baselines.

  user_story: >
    As an implementer working under the gate, when I hit a finding I have
    decided not to fix right now — a false positive, an accepted risk, a
    deferral with a ticket — I want to record a backstop-native waiver that
    carries my justification, survives across engines, and shows up loudly
    in gate output as an accounted-for suppression. I never want to reach for
    an engine-specific `//nolint` that vanishes before backstop can see it,
    and I never want to silently bury the finding in a regenerated baseline.
    The gate should still go green, but the deferral should be visible,
    attributable, and reviewable — not a hole in the accounting.
---

# Waiver Subsystem

## Current Thinking

### Why now

Two things make the waiver subsystem load-bearing right now rather than
"someday."

First, the gate already reserves a slot for it and multiple artifacts are
waiting on it:

- **SPEC-010 REQ-006** reserves gate **step 8, "waiver resolution"**: the step
  "must check active waivers and suppress matching violations from preceding
  steps. An active waiver is one [criteria unspecified]." It is currently
  stubbed — `pkg/gate/step_deferred.go` `StepWaiverResolutionScopedFunc`
  returns status `"skipped"`, reason `"waivers not implemented"`. SPEC-010
  also leaves the "active" criteria explicitly open and flags the
  expired-vs-active distinction as an unresolved question. This bundle is
  where that gets designed.
- **SPEC-019 (baseline)** deliberately preserves "a future waiver-aware
  integration point" and records that v1 baseline comparison **ignores
  waivers ONLY because the subsystem is unbuilt** — it must not bake in
  permanent pre-waiver semantics. When waivers exist, waived violations must
  participate in baseline calculation rather than being treated as permanent
  regressions. So the baseline design has an intentional seam waiting for
  this work.

Second, the motivating driver: **ISSUE-050 (strict file-level ratchet)** — the
decision that touching a file revokes baseline grandfathering for ALL of that
file's findings, forcing each one to be either fixed or WAIVED. That ratchet is
only humane if there is an accountable escape valve. Without waivers, the only
valve left when the ratchet fires is the invisible engine-native suppression —
which is exactly the accountability hole this subsystem exists to close. So
waivers should land **alongside or ahead of** ISSUE-050; the ratchet is the
forcing function that makes the escape valve necessary.

### What a waiver is (and is not)

A waiver is the backstop-native, engine-neutral, tracked, justified,
gate-visible alternative to the two bad valves. The design intent, stated as
principles rather than settled mechanism:

- **Engine-neutral.** A waiver is expressed in backstop's own vocabulary and,
  in the leading model, applied to backstop's SARIF-shaped finding stream AFTER
  engines produce findings — not inside any one engine's comment dialect. One
  concept covers semgrep, golangci-lint, staticcheck, gitleaks, and any future
  engine. This whole "waive on the SARIF stream" model is **contingent on
  engines actually emitting the findings they would otherwise suppress**: if a
  pack lets its engine keep suppressing findings internally, backstop never sees
  them and cannot waive them. That contingency is not a detail — it is OQ-9, the
  hinge the ledger aspiration hangs on.
- **Tracked and durable.** Unlike the gitignored/CI-regenerated baseline, a
  waiver is meant to be a durable record of a human decision. Where exactly it
  lives — a tracked file, in-code annotations backstop parses, or the lock
  file — is an open question (OQ-4), but the durability intent is not.
- **Justified.** A waiver without a reason is just a hidden baseline with extra
  steps. The whole point is that a deferral carries its "why." What counts as
  sufficient justification without becoming ceremony is OQ-2.
- **Loud, not silent.** Per backstop's loud-not-blocking principle, a waiver
  suppresses a finding from BLOCKING while keeping the suppression VISIBLE and
  auditable in gate output. The enemy is silent/vacuous green; a waiver is a
  green that still shows its work. How that surfaces is OQ-8.

A waiver is NOT a way to make findings disappear, and it is NOT the baseline.
The baseline answers "did you make it worse than main?"; a waiver answers "this
specific finding is a deliberate, justified, on-the-record deferral." Keeping
those two operations distinct (OQ-5) matters — mass-waiving and
baseline-generation should not collapse into each other.

### The suppression accounting model

The core reframing: today, suppression happens in three disconnected places —
inside each engine (invisible), in the baseline (silent), and nowhere else.
The aspiration is a SINGLE gate-visible ledger of every "chosen not to fix,"
so that a reviewer can answer "what are we deliberately ignoring, why, and for
how long?" from one place. Whether the subsystem should go further and actively
push authors OFF engine-native suppressions and INTO waivers — by having the
`backstop/self` dogfood pack flag `//nolint`/`nosemgrep`/etc. — is a real fork
(OQ-6): it is the difference between offering a better valve and mandating that
all deferrals flow through the one accountable channel.

### Deliberately unresolved

This bundle is at **exploring** on purpose. The problem and the seams are well
understood — SPEC-010 and SPEC-019 have been holding space for this — but the
shape of the answer is genuinely open. The Open Questions below are real forks
with live tradeoffs, not a checklist to rubber-stamp. Granularity (OQ-1),
lifecycle/expiry (OQ-3), and storage (OQ-4) in particular have no obvious
right answer and pull against each other. These are for the founder to drive.

## Open Questions

### OQ-1: Waiver identity and scope granularity

At what granularity does a waiver match findings? This is the single most
consequential fork because it sets the precision/churn/staleness tradeoff for
everything else.

- **(a) Per-finding fingerprint** — bind the waiver to the exact violation
  identity backstop already computes (the `NormalizePath`/fingerprint scheme
  from ISSUE-046 that the baseline uses). Most precise: waives exactly one
  finding and nothing else. But fingerprints are content/region-derived, so a
  waiver can go stale the moment the surrounding code is edited — the finding
  re-fires under a new identity and the waiver silently stops applying. Precise
  but brittle; high churn on active code.
- **(b) Per-rule** — waive a rule everywhere (e.g. "we don't enforce
  rule X anywhere"). Extremely durable and low-churn, but blunt: it is barely
  distinguishable from disabling the rule, and it defeats the per-finding
  accountability the subsystem is trying to create.
- **(c) Per-file** — waive all findings (or a rule's findings) in a given
  file. Survives edits within the file, coarser blast radius. Interacts
  directly with ISSUE-050's file-level ratchet (a file-scoped waiver is a
  natural unit for "I touched this file and am deferring its debt"), but risks
  waiving future NEW findings in that file that nobody reviewed.
- **(d) Per-rule-per-path** — waive rule X under path/glob Y. A middle ground:
  more durable than a fingerprint, more targeted than whole-rule. But glob
  semantics introduce their own ambiguity and can quietly over-suppress.

Tradeoff axis: precision (a) ↔ durability/low-churn (b/c). The tighter the
identity, the more accountable each waiver is — and the faster it goes stale.

**Sub-decision — multi-granularity precedence.** The subsystem may not pick one
granularity at all; it could support several (a fingerprint waiver AND a
per-file waiver AND a per-rule waiver coexisting). If it does, precedence
becomes a first-class design question, not a tail detail: when a finding is
covered by more than one waiver, which one applies — the narrowest (most
specific wins), the first-declared, or the most-permissive? Precedence drives
real behavior: it decides which waiver's justification and expiry govern the
suppression, which waiver is "used" vs "unused" for staleness reporting
(OQ-3), and whether a broad per-rule waiver can silently shadow — and thereby
hide the staleness of — a narrow fingerprint one. Choosing to support multiple
granularities is choosing to owe a precedence rule.

### OQ-2: Justification — what makes a waiver accountable without being ceremony

A waiver's reason is what separates it from a silent baseline. How structured
should it be?

- **(a) Freeform text** — a required non-empty reason string. Lowest friction,
  keeps authors in flow, but unenforceable quality: "temp" and "false positive"
  pass. Accountability depends entirely on culture.
- **(b) Structured (reason-code + text)** — a small enum of reason categories
  (e.g. false-positive / accepted-risk / deferred / third-party) plus free
  text. Enables reporting and filtering ("show me all accepted-risk waivers"),
  at the cost of forcing a taxonomy that may not fit every case.
- **(c) Mandatory link to an issue/ticket** — every waiver must reference a
  tracked work item (an issue in this repo, or an external ticket). Strongest
  accountability and a built-in remediation path, but the heaviest ceremony —
  it forces issue creation at the exact moment an author is trying to get
  unblocked, which is precisely the friction that drives people back to
  `//nolint`.

Tension with the whole thesis: the more ceremony a waiver demands, the more
attractive the invisible engine-native suppression becomes — which would
defeat the subsystem. The bar has to be high enough to be accountable and low
enough that the accountable path is the path of least resistance.

### OQ-3: Lifecycle — how is a waiver "active," and do waivers expire?

SPEC-010 left "an active waiver is one [criteria unspecified]" deliberately
open. This is that decision.

- **(a) Permanent until removed** — a waiver applies until someone deletes it.
  Simplest; matches how most lint-ignore comments behave. But waivers
  accumulate silently and become permanent invisible debt — the same failure
  mode as the baseline, just relocated.
- **(b) Time-boxed with auto-reactivation** — a waiver carries an expiry
  (explicit date, or a TTL from creation). Past expiry it becomes inactive and
  the finding re-fires (and can re-block). Forces periodic reckoning and
  prevents zombie waivers, but can re-block work at an inconvenient time and
  needs a well-defined clock (commit time? wall clock at gate run?).
- **(c) Periodic-review required** — waivers don't hard-expire but are
  surfaced for review on a cadence; unreviewed waivers get loudly flagged.
  Softer than auto-reactivation, but "flagged" only works if something actually
  makes the team look.

Cross-cutting sub-questions regardless of the above:

- **How is "active" actually determined at gate time** — presence + not-expired
  + fingerprint-still-matches? A waiver whose target finding no longer exists is
  "unused" — is that an error, a warning, or silently ignored?
- **How are expired / unused / stale waivers surfaced?** Per the
  loud-not-blocking principle, these should be LOUD (a waiver that no longer
  matches anything, or has expired, is drift the gate should call out) — but
  loud-blocking vs loud-warning is itself a choice tied to OQ-8.

### OQ-4: Storage and format — where does a waiver durably live?

- **(a) A tracked file** — e.g. `waivers.yml` at repo root, or a
  `.backstop/waivers/` directory (one file per waiver, or per area). Tracked in
  git, so waivers are durable, reviewable in PRs, and diffable. Central,
  greppable ledger. Cost: waivers live away from the code they waive, so they
  can rot out of sync, and the file can become a merge-conflict hotspot (the
  exact pain the baseline design fled from).
- **(b) In-code annotations backstop itself parses** — a backstop-native
  comment (NOT an engine's dialect) that backstop reads and turns into a waiver,
  e.g. a `backstop:waive` marker with structured fields. Keeps the waiver next
  to the code (co-located, moves/deletes with it), and is engine-neutral in the
  sense that backstop — not the engine — parses it. **But this option collides
  head-on with a cardinal first principle.** To find a `backstop:waive` comment,
  backstop must know each language's comment syntax (`//` vs `#` vs `--` vs
  `<!-- -->` vs `;`) and scan source per-language. That IS baked
  language knowledge — the exact zero-baked-knowledge invariant backstop exists
  to protect, where "new language = a new pack, never new core code." This is
  not merely "a comment-scanning surface we otherwise avoid"; it is the cardinal
  invariant violation, and it likely **disqualifies 4(b) on invariant grounds**,
  independent of any ergonomic upside. Flagged at full weight for the founder;
  co-located waivers are also harder to audit as a single ledger. (A pack could
  in principle own the comment-parsing, pushing the language knowledge back out
  of core — but that is a materially different, more complex design than "in-code
  annotation" implies, and would need its own exploration.)
- **(c) Extend `backstop.lock`** — fold waivers into the existing tracked lock
  file that is already the durability boundary. Reuses an established
  tracked-artifact story. Cost: overloads the lock file's purpose (pins vs
  human decisions) and may not suit per-finding volume.

Explicit relationship to `baseline.json`: the baseline is gitignored and
CI-regenerated (ephemeral, machine-owned). The leading intuition is that
waivers are the OPPOSITE — tracked, durable, human-owned — precisely because
they encode decisions the baseline is not allowed to. Stating that contrast as
the design's north star, but the concrete format is open.

### OQ-5: Relationship to the baseline and the ISSUE-050 ratchet

- **Does a waiver satisfy the ratchet — and at which edges?** ISSUE-050 says
  touching a file forces every finding on it to be fixed OR waived. That a
  *valid, active* waiver is the accountable "or waived" branch is close to
  foreclosed — it is the reason the two subsystems are being built together, and
  it is NOT the live decision here. The live decisions are the edges, and they
  are where the founder's judgment is actually needed:
  - **Expired waiver.** A file is touched; a finding on it is covered by a waiver
    that has expired (OQ-3b). Does the ratchet treat that finding as unresolved
    (block until re-waived or fixed), or does the mere existence of a
    once-valid waiver satisfy it? An expired waiver satisfying the ratchet would
    reopen the silent-debt hole.
  - **Unused / non-matching waiver.** A waiver exists but no longer matches any
    live finding (its fingerprint drifted, OQ-1a). Does it still "count" toward
    the ratchet for that file, or is it dead weight that satisfies nothing?
  - **Scope mismatch — the sharp one.** ISSUE-050's ratchet is FILE-level, but a
    waiver may be fingerprint-scoped (OQ-1a). Does a single fingerprint waiver
    satisfy a file-level ratchet only for that one finding (leaving the file's
    other findings still ratcheted), or is a FILE-scoped waiver (OQ-1c) the only
    unit that cleanly discharges a file-level ratchet? This is where OQ-1's
    granularity choice and OQ-5's ratchet semantics directly collide.
- **Are baseline-generation and mass-waiving distinct operations?** They should
  be. Baseline generation is a machine operation (CI snapshots existing debt);
  mass-waiving is a human decision (someone accepts specific findings with
  reasons). Collapsing them would let a bulk "waive everything" masquerade as
  accountability. Leading position: keep them distinct — but how they compose at
  gate step 7 (baseline) → step 8 (waiver) needs design, since SPEC-019
  reserved that seam.

### OQ-6: Migration off engine-native suppressions

Should this subsystem actively push authors off engine-native suppressions and
into waivers — centralizing every "chosen not to fix" into the one gate-visible
ledger?

- **(a) In scope** — the `backstop/self` dogfood pack (or an equivalent
  pack rule) eventually FLAGS `//nolint`, `nosemgrep`, `//lint:ignore`,
  `#gitleaks:allow`, etc. as findings in their own right, steering authors
  toward waivers. This is the only way to actually close the invisibility hole
  rather than just offering an alternative beside it. But it is a real
  behavior-change campaign and expands scope considerably.
- **(b) Follow-on** — build the waiver subsystem first as a strictly better
  valve; defer any flagging/migration of engine-native suppressions to a later
  bundle once waivers have proven themselves. Smaller, faster, lower-risk, but
  leaves the invisible-suppression hole open in the meantime.

This is a scope-boundary fork, not just a mechanism choice: (a) makes the
subsystem's mission "all deferrals flow through one ledger"; (b) makes it
"there exists a better ledger to flow through."

### OQ-7: CLI and authoring ergonomics

How does a developer create a waiver at the exact moment they hit a blocking
finding, without breaking flow? Ergonomics here directly determine whether the
accountable path beats reaching for `//nolint`.

- **CLI surface** — some subset of `backstop waiver add / list / rm / prune`.
  `add` at the point of blockage; `list` for the ledger; `rm` to revoke;
  `prune` to clear expired/unused. Open: does `add` take a fingerprint the gate
  just printed, a file+rule, or interactively pick from the current finding
  set?
- **In-flow creation** — can the gate output itself hand the author a
  copy-pasteable `backstop waiver add ...` for the specific finding it just
  blocked on, so acknowledging is one command? This is the ergonomic crux: the
  moment of friction is the moment of blockage, and that is where the
  accountable path has to be easiest.
- **Interaction with OQ-4** — if waivers are in-code annotations (OQ-4b), the
  "CLI" may partly be "write a comment," which changes this surface entirely.

### OQ-8: Gate reporting — keeping suppression loud

How are active waivers surfaced in gate output so suppression stays loud and
auditable rather than silent?

- What does the gate print when step 8 suppresses findings? A count? A
  per-waiver line with reason and (if any) expiry? Grouped by reason-code
  (depends on OQ-2)?
- Loud-not-blocking says the gate can go GREEN while still prominently
  reporting "N findings suppressed by active waivers." Is a run with active
  waivers visually distinct from a clean run, so a reviewer can't miss that
  suppression happened?
- How do expired / unused / stale waivers (OQ-3) appear — as warnings in the
  same output, and do any of them block? A stale/expired waiver is drift; the
  loud-not-blocking principle suggests warn-loudly, but whether an EXPIRED
  waiver that is now re-exposing a real finding should block is genuinely open.
- Does the waiver ledger feed the broader auditability/ledger surface
  (gate step 9) so "what are we deliberately ignoring" is a first-class,
  forensically-replayable question?

### OQ-9: The "report-don't-suppress" pack/engine contract

This is the hinge the entire ledger aspiration hangs on. Every other OQ quietly
assumes that backstop's SARIF stream actually RECEIVES the finding, so step 8
can waive it. But engine-native suppression kills the finding INSIDE the engine,
upstream of backstop: a `//nolint` makes golangci-lint never emit the finding, a
`nosemgrep` makes semgrep never emit it. A finding already suppressed at the
engine is **unwaivable** — backstop cannot waive what it never sees — and,
worse, it is exactly the invisible-suppression hole the subsystem set out to
close. So the "single gate-visible ledger" goal and OQ-6's migrate-off-engine-
suppressions goal are both **impossible unless packs run their engines in a
report-everything / suppress-nothing mode** and hand ALL findings to backstop,
letting backstop do 100% of the suppressing via waivers.

The fork:

- **(a) Require it as a pack contract.** Packs MUST configure their engines to
  disable native suppression and emit every finding (e.g. run golangci-lint with
  its nolint handling off, semgrep without honoring `nosemgrep`, etc.), so
  backstop is the sole suppression authority. This is what makes the one-ledger
  and OQ-6 migration actually achievable — but it is a real pack-contract change
  with genuine adoption cost: every existing pack must be updated, engines vary
  in whether they even *can* be told to report-everything, and authors lose the
  familiar inline-ignore ergonomics they may still want.
- **(b) Accept that engines keep suppressing.** Backstop waives only what
  reaches its stream and lives alongside engine-native suppression rather than
  replacing it. Far lower adoption cost and no pack-contract churn — but then
  **OQ-6's migration collapses**: you cannot centralize "chosen not to fix" into
  one ledger if engines are still silently eating findings before backstop can
  count them. The subsystem becomes "a better valve beside the invisible ones,"
  not "the one accountable channel."

Whichever way this resolves cascades into OQ-6 (migration is only coherent under
9a) and OQ-8 (a finding suppressed at the engine can never appear in gate
reporting, no matter how loud the waiver surface is). If the honest answer is
"we can't force engines to stop suppressing," the founder should know that
foreclosing the one-ledger vision before committing to it downstream.

### OQ-10: Is every finding waivable? (non-waivable tier + waiver authority)

An unrestricted waiver is itself a vacuous-green vector — the precise failure
mode this subsystem exists to kill. If any finding can be waived by anyone with
a one-line reason, the waiver becomes a universal "make it green" button and the
gate's teeth are optional. Two distinct sub-forks:

- **(a) Is there a NON-WAIVABLE class?** A waiver that could suppress a
  `backstop/self` zero-baked-language finding, or a critical secret leak, would
  let the escape valve undercut a cardinal invariant — the exact defects the
  gate is supposed to be non-negotiable about. Fork: "everything is waivable"
  (maximally flexible, but the escape valve can neutralize any rule including the
  ones that define backstop's thesis) vs "some rules/severities are
  un-waivable" (a protected tier — e.g. secrets above a severity, and
  zero-baked-language self-pack findings — that no waiver can touch). A
  non-waivable tier preserves the invariant but requires deciding WHICH rules are
  sacred and giving packs/severities a way to declare themselves un-waivable.
- **(b) Who may create a waiver, and is it reviewed?** "Accountable" implies a
  waiver is not silently self-approved by the same author dodging the finding.
  Fork: does the gate treat an unreviewed / self-added waiver differently
  (e.g. warn louder, require a second party, or require the waiver to land via
  PR review rather than a local edit) vs treat all waivers equally regardless of
  provenance? This is **lower priority for a solo founder today** — there is no
  second party to review — but it is a real fork the moment the subsystem is used
  by a team, and the storage choice (OQ-4: a tracked file reviewed in a PR vs a
  local annotation) quietly pre-decides part of it.

## Version History

- 0.1.0 (2026-07-09): Initial bundle at **exploring**. Framed the problem: no
  accountable, engine-neutral, gate-visible way to say "chosen not to fix,"
  with today's only valves being invisible engine-native suppressions
  (`//nolint`, `nosemgrep`, `//lint:ignore`, `#gitleaks:allow`) and silent
  baseline grandfathering. Captured why-now (SPEC-010 step 8 reserved and
  stubbed in `pkg/gate/step_deferred.go`; SPEC-019's reserved waiver-aware
  seam; ISSUE-050's strict file-level ratchet as the forcing function that
  demands an accountable escape valve). Surfaced 8 genuine, unresolved open
  questions: granularity (OQ-1), justification (OQ-2), lifecycle/expiry
  (OQ-3), storage/format (OQ-4), baseline+ratchet relationship (OQ-5),
  migration off engine-native suppressions (OQ-6), CLI/ergonomics (OQ-7), and
  gate reporting (OQ-8). All OQs left open for founder-driven resolution; no
  design decisions, requirements, or spec seeds committed yet.

- 0.1.0 (2026-07-09, review pass): Completed and sharpened the OQ set from a
  bundle review; still **exploring**, still fully unresolved, no promotion.
  Added two missing open questions: OQ-9 (the "report-don't-suppress"
  pack/engine contract — the hinge on which the single-ledger and OQ-6 migration
  goals depend, since engine-native suppression kills findings upstream of
  backstop) and OQ-10 (is every finding waivable — non-waivable tier for
  cardinal invariants like zero-baked-language and critical secrets, plus waiver
  authority/review). Sharpened framing without resolving: named OQ-4(b)'s in-code
  annotation parsing as a first-principles zero-baked-knowledge collision that
  likely disqualifies it on invariant grounds; softened the "waive on the SARIF
  stream" line from settled principle to leading model contingent on OQ-9;
  reframed OQ-5 so the founder resolves the LIVE ratchet edges (expired/unused
  waiver, file-scoped vs fingerprint-scoped satisfying a file-level ratchet)
  rather than re-ratifying the near-foreclosed "yes"; elevated OQ-1's
  multi-granularity precedence from a tail mention to an explicit sub-decision.

## References

- SPEC-010 (gate): REQ-006 reserves step 8 "waiver resolution"; leaves
  "active waiver" criteria and the expired-vs-active distinction open.
- SPEC-019 (baseline): preserves a future waiver-aware integration point;
  v1 baseline comparison ignores waivers only because this subsystem is unbuilt.
- ISSUE-050 (strict file-level ratchet): the forcing function — touching a file
  forces every finding to be fixed or waived; waivers are the accountable valve.
- BUNDLE-007 (baseline): REQ-013 / DD-18 record the deferred waiver seam.
- ISSUE-046: exported `NormalizePath` / fingerprint violation identity — the
  candidate substrate for per-finding waiver matching (OQ-1a).
- `pkg/gate/step_deferred.go`: `StepWaiverResolutionScopedFunc` — the current
  stub returning "skipped / waivers not implemented."
