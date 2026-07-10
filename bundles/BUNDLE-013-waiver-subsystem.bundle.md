---
title: "Waiver Subsystem — Accountable, Engine-Neutral Suppression"
number: BUNDLE-013
created: "2026-07-09"
schema_version: bundle/v2

bundle:
  name: waiver-subsystem
  version: "0.3.0"
  created: "2026-07-09"
  updated: "2026-07-09"
  category: feature

status:
  maturity: delivered

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

solution:
  approach: >
    An inline, per-finding waiver expressed as a backstop-native DSL token —
    `@waiver:<rule-id>:<reason-code>:<expiry>` (plus optional trailing
    note/issue-ref) — that a human (or the runtime agent) writes inside
    whatever comment their language already uses, at the source location of the
    finding. Backstop adjudicates by LINE-SCANNING the finding's own SARIF
    location for the token: because `@waiver` is a foreign token to every
    engine, engines emit the finding normally and backstop does all
    suppressing, so the model works uniformly across engines with no
    report-everything pack contract and stays zero-baked (core reads bytes at a
    location the engine already reported; it never parses source or knows any
    language's comment syntax). Identity is the finding's location, not a stored
    fingerprint, and waivers are per-finding — no file- or rule-blanket
    shortcut. EVERY waiver carries a mandatory expiry (default duration set by
    reason-code); on expiry the shield lifts and the finding re-fires under
    normal enforcement, with a loud heads-up before the flip. A declared
    non-waivable tier (rules/severities that self-declare un-waivable in their
    pack manifest — shipping with backstop/self zero-baked-language rules and
    critical-severity secrets) makes the escape valve unable to neutralize
    cardinal invariants. This replaces silent baseline grandfathering and
    invisible engine-native suppression with one loud, expiring, accountable
    ledger, and is the accountable "or waived" branch of the ISSUE-050
    file-level ratchet.

requirements:
  - id: REQ-001
    text: >
      Backstop must define a waiver DSL token of the form
      `@waiver:<rule-id>:<reason-code>:<expiry>` with an optional trailing
      free-text note and/or issue reference. `reason-code` is a closed enum:
      `false-positive`, `accepted-risk`, `deferred`, `third-party`. The grammar
      must be specified precisely enough to parse and validate deterministically.
  - id: REQ-002
    text: >
      Waiver adjudication must be inline and engine-neutral: engines emit
      findings normally (no report-everything mode required), and backstop
      suppresses a finding by LINE-SCANNING the finding's own reported SARIF
      location for a matching `@waiver` token. Adjudication must remain
      zero-baked — backstop reads bytes at a location the engine already
      reported and must NOT parse source or encode any language's comment
      syntax.
  - id: REQ-003
    text: >
      Waiver identity must be per-finding and location-based, not a stored
      content fingerprint. A waiver applies only to the specific finding at the
      token's associated location. There is no file-blanket or rule-blanket
      scope: a debt-heavy file that is touched is discharged finding-by-finding.
  - id: REQ-004
    text: >
      Every waiver must carry a mandatory expiry; no waiver is permanent. Each
      reason-code sets a default duration (e.g. `false-positive` long-lived,
      `deferred`/`accepted-risk`/`third-party` short-lived; concrete durations
      are tunable). On expiry the waiver becomes inactive and the finding
      re-fires under normal enforcement. The gate must emit a loud heads-up
      BEFORE a waiver expires (the pre-expiry warning is the grace period).
  - id: REQ-005
    text: >
      The gate must detect and warn on unused/dangling waivers — a `@waiver`
      token whose associated location no longer has a matching live finding.
      An unused waiver is surfaced as a warning; it must not silently persist.
  - id: REQ-006
    text: >
      Backstop must support a declared non-waivable tier: a rule or severity
      self-declares itself un-waivable in its pack manifest / enforcement
      policy (declared, not a core-hardcoded list). The shipped configuration
      places backstop/self rules and critical-severity secrets in the
      non-waivable set. A `@waiver` token targeting a non-waivable rule is a
      gate ERROR, not a suppression.
  - id: REQ-007
    text: >
      Malformed `@waiver` tokens (bad grammar, unknown reason-code, missing or
      invalid expiry) must themselves be gate findings. The waiver grammar is
      enforced, not best-effort.
  - id: REQ-008
    text: >
      Token-to-finding location matching semantics must be defined: the line
      association rule (same-line trailing token vs token on the line above),
      handling of multi-line findings, and avoidance of false matches when the
      literal string `@waiver:...` appears inside a string literal or other
      non-suppressing context rather than as an author's waiver.
  - id: REQ-009
    text: >
      `rule-id` in the token must be a stable, ergonomic identifier for the
      rule being waived, and the subsystem must define behavior when a pack
      renames a rule (e.g. the waiver no longer matches and surfaces as
      unused/dangling per REQ-005, rather than silently waiving a different
      rule).
  - id: REQ-010
    text: >
      Waivers apply to CODE-LOCATED findings only. Artifact/structural gate
      dimensions (artifact_status_drift, contract_signature, test_verification)
      are NOT waivable and keep their existing accountable resolution paths
      (retire/replace/resolved-by/obsoleted). File-level coverage findings get
      a defined annotation convention rather than a source-location token.
  - id: REQ-011
    text: >
      The core CLI is read-only with respect to waivers: `backstop waiver list`
      must report active, expiring-soon, and unused/dangling waivers. Core
      backstop must NOT write `@waiver` tokens (inserting a comment requires
      language comment-syntax = baked-language knowledge); authoring and
      re-certification belong to the human or the runtime agent.
  - id: REQ-012
    text: >
      Gate reporting must never be silent about waivers. A run that passes
      because of active waivers is a DISTINCT terminal state (e.g.
      `PASS · N waivers`), visually distinguishable from a clean pass. An
      active-waiver summary is always shown; the actionable subset
      (expiring-soon, unused) is surfaced inline on every run; full per-waiver
      detail is available on demand. Engine-native VISIBLE suppressions
      (e.g. semgrep's SARIF-reported suppressions) are reported on a separate
      line.
  - id: REQ-013
    text: >
      Waivers must interact correctly with the ISSUE-050 file-level ratchet:
      a valid ACTIVE waiver satisfies the ratchet for its finding (the
      accountable "or waived" branch). An EXPIRED waiver does not satisfy the
      ratchet (the live finding demands action). An UNUSED waiver satisfies
      nothing. Baseline generation (machine snapshot) and waiver authoring
      (human decision) remain distinct operations.
  - id: REQ-014
    text: >
      When the gate blocks on a waivable finding, its output must hand the
      author a pre-filled, neutral `@waiver:<rule>:<reason>:<expiry>` token for
      that specific finding, so that acknowledging is at most as much friction
      as writing an engine-native `//nolint`.
  - id: REQ-015
    text: >
      The active-waiver set must be exposed to the gate's step-9 audit/ledger
      surface so that "what are we deliberately ignoring, why, and until when"
      is a first-class, auditable question. (Step 9 itself is unbuilt; this
      requirement is the data-feed contract, not step-9 implementation.)
---

# Waiver Subsystem

## Current Thinking

### Why now

Two things make the waiver subsystem load-bearing now rather than "someday."

First, the gate already reserves a slot for it and multiple artifacts wait on
it:

- **SPEC-010 REQ-006** reserves gate **step 8, "waiver resolution"**: the step
  "must check active waivers and suppress matching violations from preceding
  steps." It is currently stubbed — `pkg/gate/step_deferred.go`
  `StepWaiverResolutionScopedFunc` returns status `"skipped"`, reason
  `"waivers not implemented"`. SPEC-010 also left the "active waiver" criteria
  and the expired-vs-active distinction explicitly open; this bundle settles
  them.
- **SPEC-019 (baseline)** deliberately preserves "a future waiver-aware
  integration point" and records that v1 baseline comparison ignores waivers
  ONLY because the subsystem was unbuilt — it must not bake in permanent
  pre-waiver semantics. This bundle fills that reserved seam.

Second, the motivating driver: **ISSUE-050 (strict file-level ratchet)** — the
decision that touching a file revokes baseline grandfathering for ALL of that
file's findings, forcing each one to be either fixed or WAIVED. That ratchet is
only humane if there is an accountable escape valve; without waivers, the only
valve left when the ratchet fires is the invisible engine-native suppression —
the exact accountability hole this subsystem closes. **The key insight: the
ratchet is the bridge that converts silent, baseline-grandfathered debt into
loud, expiring, accountable waivers at the moment someone touches the file.**

### The settled model

All ten open questions and two scope boundaries are resolved (see Draft Design
Decisions). The shape of the answer:

- **Inline, per-finding, location-based.** A waiver is a backstop-native DSL
  token — `@waiver:<rule-id>:<reason-code>:<expiry>` — that a human writes
  inside whatever comment their language already uses, at the finding's source
  location. Identity is the location, not a stored fingerprint; the waiver
  moves and dies with the code.
- **Foreign-token adjudication dissolves the hardest problem.** Because
  `@waiver` is a foreign token to every engine, engines emit findings normally
  and backstop does 100% of the suppressing by line-scanning the finding's own
  reported location. This works uniformly — golangci-lint included — with **no
  forced report-everything pack contract**, and it keeps core zero-baked:
  backstop reads bytes at a location the engine already reported and never
  learns a single language's comment syntax.
- **Everything expires.** No permanent waivers. Reason-code sets the default
  duration; on expiry the shield lifts and the finding re-fires under normal
  enforcement, with a loud pre-expiry warning as the grace period. This is the
  structural defense against waivers becoming the new silent debt.
- **A declared non-waivable tier protects the invariants.** Rules and
  severities self-declare un-waivable in their pack manifest (declared, not
  core-hardcoded); backstop/self zero-baked-language rules and critical-severity
  secrets ship in that set. The escape valve cannot neutralize a cardinal
  invariant.
- **Loud by construction.** A pass that depends on waivers is a distinct
  terminal state; the active-waiver summary is always shown and the actionable
  subset (expiring-soon, unused) is surfaced inline every run.

### The suppression accounting model

Today, suppression happens in three disconnected places — inside each engine
(invisible), in the baseline (silent), and nowhere else. The waiver channel
becomes a single gate-visible ledger of every "chosen not to fix." Notably, the
subsystem does NOT wage a migration crusade against engine-native suppressions:
`//nolint`/`nosemgrep`/`#gitleaks:allow` are accepted as debt/risk rather than
fought, because the foreign-token design already gives an accountable channel
without needing to force engines out of their own suppression modes. An
empirical asymmetry is worth recording as design context: semgrep REPORTS
`nosemgrep`-suppressed findings in SARIF (`suppressions: kind inSource`) so
backstop can surface them, whereas golangci-lint DROPS `//nolint`'d findings
entirely (invisible to backstop). The design accepts that asymmetry rather than
trying to erase it.

### Scope and enforceability boundaries

Waivers apply to code-located findings only; artifact/structural dimensions keep
their existing accountable resolution paths. The non-waivable secrets guarantee
is scoped to backstop's OWN `@waiver` channel, where it is enforceable — it is
airtight for semgrep-based backstop/self findings. It does not extend to
engine-native allows: a `gitleaks:allow` on a secret is an EXPLICIT,
source-visible human choice to expose the secret — categorically different from
silent debt — so it is out of scope to "plug," and no report-everything
requirement is imposed to chase it.

## Draft Requirements

Requirements are formally captured in the frontmatter `requirements` block
(REQ-001 through REQ-015). Summary:

- **REQ-001** — `@waiver:<rule-id>:<reason-code>:<expiry>` DSL grammar +
  closed reason-code enum (false-positive / accepted-risk / deferred /
  third-party).
- **REQ-002** — Inline, engine-neutral, zero-baked line-scan adjudication:
  engines emit, backstop suppresses at the reported location.
- **REQ-003** — Per-finding, location-based identity; no file/rule blanket
  scope; touched files discharged finding-by-finding.
- **REQ-004** — Mandatory expiry with reason-code default durations; re-fire on
  expiry; loud pre-expiry warning.
- **REQ-005** — Unused/dangling waiver detection surfaced as a warning.
- **REQ-006** — Declared non-waivable tier (pack-manifest-declared); ships with
  backstop/self + critical secrets; waiving one is a gate error.
- **REQ-007** — Grammar enforcement: a malformed `@waiver` is itself a gate
  finding.
- **REQ-008** — Token-location matching semantics: line association, multi-line
  findings, false-match avoidance (token inside a string literal).
- **REQ-009** — Stable ergonomic `rule-id` + defined behavior on pack rule
  rename.
- **REQ-010** — Scope boundary: code-located findings only; structural
  dimensions non-waivable; coverage annotation convention.
- **REQ-011** — Read-only core CLI `backstop waiver list`; core never writes
  tokens (authoring belongs to human/agent).
- **REQ-012** — Reporting: distinct pass-with-waivers terminal state, always-on
  summary, inline actionable subset, separate engine-native-visible line.
- **REQ-013** — Ratchet interaction: active satisfies, expired/unused do not;
  baseline-generation and waiver-authoring stay distinct.
- **REQ-014** — Gate emits a pre-filled neutral token on a blocked waivable
  finding.
- **REQ-015** — Active-waiver set feeds the step-9 audit/ledger surface.

## Draft Design Decisions

- **DD-1 (OQ-1 identity):** Waiver identity is per-finding and LOCATION-based,
  expressed as an inline annotation — NO stored fingerprint. The waiver lives at
  the finding's source location and applies to exactly that finding. Rationale:
  a content fingerprint (as the baseline uses) drifts on any nearby edit and
  goes silently stale; a co-located token instead moves, and dies, with the code
  it annotates. Per-finding rather than per-file/per-rule keeps accountability
  finding-by-finding and forecloses a blanket "waive this whole file/rule"
  shortcut.

- **DD-2 (OQ-2 justification):** A waiver is a compact DSL token
  `@waiver:<rule-id>:<reason-code>:<expiry>` with an optional trailing
  note/issue-ref. `reason-code` is a closed enum: `false-positive`,
  `accepted-risk`, `deferred`, `third-party`. Rationale: structured enough to
  report/filter and to drive default expiry durations (DD-3), compact enough
  that writing one is no more friction than an engine-native ignore.

- **DD-3 (OQ-3 lifecycle):** EVERY waiver expires — none is permanent. The
  reason-code sets the default duration (`false-positive` long-lived, e.g.
  ~1yr; `deferred`/`accepted-risk`/`third-party` short-lived, e.g. ~90d;
  durations are tuning, not contract). On expiry the waiver becomes inactive and
  the finding re-fires under normal enforcement; the gate emits a loud heads-up
  BEFORE the flip. An unused/dangling waiver (no live finding at its location)
  surfaces as a warning. Rationale: permanent waivers would recreate exactly the
  silent-baseline debt the subsystem exists to kill; forced expiry guarantees
  periodic reckoning, and the pre-expiry warning is the grace so re-fire is
  never a surprise.

- **DD-4 (OQ-4 storage):** Waivers are stored inline in source, located by
  LINE-SCANNING the finding's own SARIF location for the token — NOT by parsing
  comments, and NOT in a central file. Rationale: this sidesteps the
  first-principles collision that an earlier option raised — per-language
  comment PARSING would be baked-language knowledge, the cardinal zero-baked
  violation. Here backstop reads bytes at a location the engine already
  reported; the human writes the token inside whatever comment their language
  uses, so core never learns comment syntax. Co-located and tracked in source =
  durable, with no central merge-conflict hotspot.

- **DD-5 (OQ-5 baseline/ratchet):** A valid ACTIVE waiver satisfies ISSUE-050's
  file-level ratchet for its finding (the accountable "or waived" branch). An
  EXPIRED waiver does NOT satisfy it (the live finding stands and the ratchet
  demands action). An UNUSED waiver satisfies nothing. Because identity is
  per-finding (DD-1), there is no file-blanket shortcut — a touched, debt-heavy
  file is discharged finding-by-finding. Baseline generation (a machine
  snapshot) and waiver authoring (a human decision) stay DISTINCT operations.
  Rationale — the key framing: the ratchet is the bridge that converts silent,
  baseline-grandfathered debt into loud, expiring, accountable waivers at the
  moment someone touches the file.

- **DD-6 (OQ-6 migration):** No migration crusade. Native engine suppressions
  (`//nolint`, `nosemgrep`, `#gitleaks:allow`, etc.) are accepted as debt/risk,
  not fought. Rationale: the foreign-token design (DD-9) already yields an
  accountable channel without forcing engines out of their suppression modes,
  so a forced migration would add adoption cost for little gain. Optional
  loud-surfacing of engine-VISIBLE suppressions is a follow-on (see
  Notes / Ideas), not a v1 requirement.

- **DD-7 (OQ-7 CLI/ergonomics):** Primary authoring is writing the inline
  comment, not running a command. Core backstop READS `@waiver` (line-scan) but
  does NOT WRITE it — inserting a comment requires language comment-syntax,
  which is baked-language knowledge, so authoring and re-certification belong to
  the human or the runtime AGENT (which may legitimately hold language
  knowledge). The core CLI is read-only: `backstop waiver list`
  (active/expiring/unused). Grammar is enforced — a malformed `@waiver` is
  itself a gate finding (REQ-007). When the gate blocks on a waivable finding it
  hands the author a pre-filled neutral `@waiver:<rule>:<reason>:<expiry>`
  token. North star: authoring a waiver must be ≤ the friction of `//nolint`.
  Rationale: keeps core zero-baked (writing comments is the one place language
  knowledge would leak in) and pushes authoring to the layer that legitimately
  owns it.

- **DD-8 (OQ-8 reporting):** Waiver suppression is never silent. A run that
  passes because of active waivers is a DISTINCT terminal state (e.g.
  `PASS · N waivers`). The active-waiver summary is always shown; the actionable
  subset (expiring-soon, unused) is surfaced inline every run; full per-waiver
  detail is available on demand. Engine-native VISIBLE suppressions (e.g.
  semgrep's SARIF-reported suppressions) get a separate line. On expiry the
  shield lifts and normal enforcement resumes immediately — the pre-expiry
  warning was the grace — and the runtime agent notices and prompts
  re-certification. The active-waiver set feeds the step-9 audit/ledger surface.
  Rationale: loud-not-blocking — a green built on waivers must show its work.

- **DD-9 (OQ-9 report-don't-suppress):** No forced report-everything pack
  contract. Because `@waiver` is a FOREIGN token to engines, they emit the
  finding normally and backstop adjudicates — this works uniformly, golangci
  included. Empirical design context: semgrep REPORTS `nosemgrep`-suppressed
  findings in SARIF (`suppressions: kind inSource`), so backstop can see them;
  golangci-lint DROPS `//nolint`'d findings entirely (invisible). The design
  accepts that asymmetry. Rationale: the foreign-token insight dissolves what
  looked like the subsystem's hardest hinge — the ledger works WITHOUT changing
  any engine's mode, so no pack-contract churn is needed.

- **DD-10 (OQ-10 non-waivable tier + authority):** A rule or severity
  self-declares itself un-waivable in its pack manifest / enforcement policy —
  declared, not a core-hardcoded list. The shipped set includes backstop/self
  rules and critical-severity secrets. A waiver targeting a non-waivable rule is
  a gate ERROR. Enforceability caveat (recorded, not a gap to plug): the
  guarantee is only as strong as the engine's suppression-reporting — airtight
  for semgrep-based backstop/self, but a `gitleaks:allow` on a secret is an
  EXPLICIT, source-visible human choice to expose (categorically different from
  silent debt), so it is out of scope and imposes no report-everything
  requirement. Author/approval mechanics are deferred to diff-visibility.
  Rationale: an unrestricted waiver is itself a vacuous-green vector; a declared
  protected tier keeps the escape valve from neutralizing cardinal invariants
  while staying pack-driven and zero-baked.

- **DD-11 (scope boundary — waivable surface):** Waivers apply to CODE-LOCATED
  findings only. Artifact/structural dimensions (artifact_status_drift,
  contract_signature, test_verification) are NOT waivable — they retain their
  existing accountable resolution paths (retire / replace / resolved-by /
  obsoleted). File-level coverage gets a defined annotation convention rather
  than a source-location token. Rationale: the inline-location model only makes
  sense where a finding has a source location, and structural dimensions already
  have first-class accountable lifecycles.

- **DD-12 (scope boundary — secrets guarantee):** The non-waivable secrets
  guarantee is scoped to backstop's OWN `@waiver` channel, where it is
  enforceable. Engine-native allows are the human's explicit, source-visible
  choice, not a backstop hole. Rationale: bound the guarantee to what backstop
  can actually enforce rather than overclaiming coverage it cannot deliver.

## Spec Seeds

Suggested decomposition; each requirement belongs to exactly one seed.
Implementation order A → B → C → D (D integrates the rest).

- **Seed A — Waiver DSL + zero-baked line-scan adjudication (foundation).**
  The `@waiver` token grammar and reason-code enum, per-finding location
  identity, the engine-neutral line-scan that suppresses at the finding's
  reported location without parsing source, token-to-finding location matching
  semantics (line association, multi-line findings, string-literal false-match
  avoidance), stable `rule-id` + rename behavior, and malformed-token-as-gate-
  finding enforcement. Covers REQ-001, REQ-002, REQ-003, REQ-007, REQ-008,
  REQ-009. Implement first — everything else consumes it.

- **Seed B — Lifecycle & expiry.** Mandatory expiry, reason-code default
  durations, re-fire on expiry, the loud pre-expiry warning, and
  unused/dangling detection. Covers REQ-004, REQ-005.

- **Seed C — Declared non-waivable tier.** Pack-manifest/enforcement-policy
  declaration of un-waivable rules/severities, the shipped backstop/self +
  critical-secrets set, and gate-error-on-waiving-a-protected-rule. Covers
  REQ-006.

- **Seed D — Gate step-8 integration, reporting, ratchet & CLI.** Replace the
  `pkg/gate/step_deferred.go` waiver-resolution stub with real adjudication;
  the distinct pass-with-waivers terminal state, always-on summary and inline
  actionable subset (and the separate engine-native-visible suppression line);
  the ISSUE-050 ratchet interaction; the pre-filled neutral token emitted on a
  blocked waivable finding; the read-only `backstop waiver list` CLI; the
  code-located-only scope boundary + coverage annotation convention; and the
  active-waiver feed into the step-9 audit surface. Covers REQ-010, REQ-011,
  REQ-012, REQ-013, REQ-014, REQ-015.

## Notes / Ideas

Out-of-v1 follow-ons, captured so they are not lost:

- **Migration nudge (not a crusade):** a `backstop/self` grep rule that flags
  native suppression tokens (`//nolint`, `nosemgrep`, `#gitleaks:allow`) and
  suggests converting them to `@waiver`. Steers without forcing.
- **Surface engine-VISIBLE native suppressions:** render semgrep's
  SARIF-reported `suppressions` as a non-blocking gate line, so the visible
  subset of engine-native ignores is at least loud.
- **Step-9 audit-ledger integration:** wire the active-waiver feed into the
  step-9 audit/ledger surface once step 9 itself is built (it is currently
  unbuilt; REQ-015 defines only the data-feed contract).
- **Future secrets hardening:** optionally run the secrets engine in a
  report-everything mode so that `gitleaks:allow`'d secrets still surface —
  weighed against the DD-12 position that a visible `gitleaks:allow` is an
  explicit human choice, not a hole.

## Version History

- 0.1.0 (2026-07-09): Initial bundle at **exploring**. Framed the problem: no
  accountable, engine-neutral, gate-visible way to say "chosen not to fix,"
  with today's only valves being invisible engine-native suppressions and
  silent baseline grandfathering. Grounded why-now in SPEC-010 step 8
  (reserved/stubbed), SPEC-019's reserved waiver seam, and ISSUE-050's ratchet
  as the forcing function. Surfaced 8 open questions.

- 0.1.0 (2026-07-09, review pass): Completed the OQ set — added OQ-9
  (report-don't-suppress pack/engine contract) and OQ-10 (non-waivable tier +
  authority) — and sharpened framing (OQ-4b as a first-principles zero-baked
  collision; OQ-5 reframed to the live ratchet edges; OQ-1 precedence elevated).
  Still exploring, still fully unresolved.

- 0.2.0 (2026-07-09): Advanced to **defined** on founder approval. Resolved all
  10 open questions plus two scope boundaries into DD-1 through DD-12: inline
  per-finding location-based `@waiver` DSL (DD-1/DD-2), zero-baked line-scan
  adjudication (DD-4), mandatory reason-code-defaulted expiry with pre-expiry
  warning and unused-waiver detection (DD-3), ratchet interaction as the
  accountable "or waived" branch (DD-5), no migration crusade (DD-6),
  read-only core CLI with human/agent authoring (DD-7), loud distinct
  pass-with-waivers reporting (DD-8), foreign-token adjudication dissolving the
  report-everything hinge (DD-9), declared non-waivable tier (DD-10), and the
  code-located-only + secrets-channel scope boundaries (DD-11/DD-12). Added
  `solution.approach`, formal requirements REQ-001 through REQ-015, Draft
  Requirements and Spec Seeds (4 seeds), and Notes / Ideas follow-ons. Removed
  the now-resolved Open Questions section.

- 0.3.0 (2026-07-09): Advanced to **delivered** (success-terminal). The bundle's
  sole spec, SPEC-049, covers all 15 requirements (REQ-001 through REQ-015) and
  is implemented and committed (eee4700), reviewed at every stage — Seeds A–D
  (DSL + zero-baked line-scan adjudication, lifecycle & expiry, declared
  non-waivable tier, and gate step-8 integration / reporting / ratchet / CLI)
  all shipped. The four Notes / Ideas follow-ons (migration nudge, surfacing
  engine-VISIBLE native suppressions, step-9 audit-ledger wiring, secrets
  report-everything hardening) remain explicitly out of v1 and do not block
  delivery. No requirement or design-decision content changed in this bump.

## References

- DIR-003 / BUNDLE-007 (baseline): the CI-generated baseline whose silent
  grandfathering this subsystem replaces with accountable waivers.
- ISSUE-050 (strict file-level ratchet): the driver — touching a file forces
  every finding to be fixed or waived; waivers are the accountable valve.
- SPEC-010 REQ-006 (gate step 8, waiver resolution): the reserved, stubbed step
  this subsystem implements.
- SPEC-019 (baseline): reserved the waiver-aware baseline integration seam.
- ISSUE-046: exported `NormalizePath` / path normalization — the reported-
  location substrate the line-scan adjudication relies on.
- `pkg/gate/step_deferred.go`: `StepWaiverResolutionScopedFunc` — the current
  "skipped / waivers not implemented" stub to be replaced (Seed D).
