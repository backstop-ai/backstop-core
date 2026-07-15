---
title: "Requirement Traceability — Close the Bundle→Spec Hop"
number: BUNDLE-014
created: "2026-07-14"
schema_version: bundle/v2

bundle:
  name: requirement-traceability
  version: "0.9.0"
  created: "2026-07-14"
  updated: "2026-07-14"
  category: feature

status:
  maturity: delivered

problem:
  summary: >
    The requirement chain is mechanically verified from a spec down to its
    implementation: a spec's requirements decompose into claims, claims mandate tests,
    those tests are verified for existence and substantiveness, and the pack-engine
    test run proves they pass — so a green gate on an `implemented` spec is mechanical
    proof of every requirement in that spec. But the TOPMOST hop — bundle REQ → spec
    requirement — is convention only, so the end-to-end pitch "every bundle requirement
    is mechanically verified down to implementation" is NOT yet true. Three convention-
    only gaps in that hop: (1) a `supports: bundle-name:REQ-NNN` ref on a spec/issue
    requirement is regex-FORMAT-checked only (`supportsRe`, pkg/validate/spec.go:16;
    same in pkg/validate/issue.go) and never RESOLVED — it can cite a bundle that does
    not exist, or a REQ the bundle never declared, and validate clean. (2) The COVERAGE
    direction is never checked: nothing requires a bundle REQ to be supported by any
    spec requirement at all, so a bundle can reach `delivered` with uncovered REQs and
    no gate dimension murmurs. (3) `requirements[]` is enforced only AT `defined`/`ready`
    maturity (`validateBundleRequirements`, pkg/validate/bundle.go), and terminal
    bundles are exempted (bundle.go:94) — so a `delivered` bundle with NO requirements
    array validates clean, and the top of the chain can be structurally empty at the
    exact moment it claims success. TWO deeper gaps compound these (added 2026-07-14 as
    the scope expanded): (4) enforcement only ever fires at DELIVERY — there is no
    EARLIER gate, so implementation can begin with bundle REQs that trace into nothing
    planned, and the mismatch is only caught (if ever) at the very end. (5) Requirements
    are UNVERSIONED — when an issue later modifies a requirement, or a spec implements an
    older understanding of one, nothing records WHICH version of a REQ a spec/plan/issue
    actually satisfied, so requirement evolution is silent drift rather than a traceable
    fact. Audited 2026-07-14 evidence the corpus is already drifting under (1)–(3): 10
    dangling supports refs live today (SPEC-002/003/004, all `draft`, cite
    `agent-definitions:REQ-004..REQ-018`, and the delivered `agent-definitions` bundle
    declares NO requirements[] array — those REQs exist only in prose); BUNDLE-011
    (`delivered`) has REQ-004 supported by nothing and REQ-007/REQ-010 supported only by
    SPEC-039, which is `replaced`. `./bin/backstop artifact validate` is currently
    ALL-GREEN over all of it.
  user_story: >
    As the maintainer who sells backstop on "a green gate is mechanical proof that
    every requirement is met down to the implementation," I want the bundle→spec hop
    resolved and covered — not merely regex-shaped — so that a `delivered` bundle is a
    mechanical guarantee that every one of its REQs is supported by a real, `implemented`
    spec requirement, and a supports ref can never cite a bundle or a REQ that does not
    exist. I want chain-of-custody verified at EVERY promotion, not just at delivery:
    there is no sense promoting a spec to planning if it fails validation (the plan will
    be inherently incomplete), and a plan should not start until the whole chain is
    verified — so the gate blocks any state where a downstream artifact has outrun the
    chain that supports it. And because the discipline is "cover everything in one shot at
    planning, then use issues for after-the-fact follow-up (bug fixes, edge cases, changing
    the requirement itself)," I want requirements to carry a semver VERSION that every trace
    pins exactly, backed by a per-REQ version LOG that retains every version and its text,
    so a spec/plan/issue states precisely which version it satisfied and I can answer "what
    exactly did the spec pinned @1.1.0 implement, and how does it differ from 2.0.0" from
    the artifacts themselves — not git archaeology. The enforcement keys off bump SIZE (a
    wording-only patch is free; a meaning-changing major/minor forces downstream rework or a
    new spec), blocks defects and broken promises, and surfaces in-flight gaps as advisory
    warnings — visible but never blocking, never silent.
  success_criteria:
    - >
      Every `supports` ref in the corpus resolves — real bundle, declared REQ, and a pin
      matching a real version-log entry — or `artifact validate` blocks; no dangling,
      unpinned, or fabricated-version ref validates green (REQ-001…REQ-004).
    - >
      No `delivered` bundle can exist with an uncovered REQ or a non-`implemented` citing
      spec: the `requirement_traceability` gate step blocks that corpus state, with
      `replaced` specs not flowing through (REQ-008…REQ-010, REQ-012), and `requirements[]`
      enforced at `delivered` so the top of the chain is never structurally empty (REQ-005).
    - >
      The three-tier posture holds in practice: ref defects and broken-promise states BLOCK,
      while in-flight uncovered REQs surface as ADVISORY warnings — visible, never blocking,
      never silent (REQ-013).
    - >
      A bundle REQ is interrogable by version from the artifacts alone — the version log
      answers "what did @1.1.0 say vs 2.0.0" without git archaeology (REQ-004) — and the
      stale-pin model behaves per bump size × bundle lifecycle (REQ-014).
    - >
      After the one-time reconciliation sweep — the `agent-definitions` cluster deprecated
      (bundle + SPEC-002/003/004 terminal), BUNDLE-011's REQ-007/010 re-stated via retroactive
      SPEC-053, and `1.0.0` stamped on every remaining live REQ + ref — the full corpus is
      green under the new mandatory-pin enforcement (REQ-015).

solution:
  approach: >
    A promoted `defined` shape: ALL TEN open questions are resolved (RDQ-1…RDQ-10) — the
    Open Questions section is empty — and the resolved decisions are lifted faithfully into
    the formal `requirements[]` array (REQ-001…REQ-015). The bundle welds the top link with
    resolution EVERYWHERE, coverage enforced as a corpus-STATE invariant, and a versioned,
    log-backed, after-the-fact-queryable chain. (A) RESOLUTION, in `artifact validate`
    (RDQ-3) — every `supports` ref must resolve to a real bundle AND a REQ declared in its
    requirements[], and its EXACT semver pin (RDQ-6) `bundle-name:REQ-NNN@1.2.0` must match a
    real entry in that REQ's version LOG (RDQ-10); a ref resolving to nothing, missing its
    mandatory pin, or citing a version that never existed is a defect and blocks at ANY
    citing status. Issue supports refs are resolution-checked, queryable lineage links
    (RDQ-4). (B) COVERAGE via a NEW `requirement_traceability` gate step (RDQ-1) on the
    status_drift full-corpus block/advisory pattern (RDQ-3), enforced as a STATE invariant
    (RDQ-8): the step blocks any corpus state where a downstream artifact's upstream chain
    does not verify — a spec advanced past the depth its bundle chain supports, a plan for a
    spec whose bundle chain doesn't verify, `delivered` without full coverage (every citing
    spec `implemented`, DD-1; every REQ supported by ≥1 `implemented` spec requirement, DD-2;
    `replaced` specs don't flow through, DD-3; issue requirements never satisfy coverage,
    DD-9). Core enforces STATES not events — because promotions land as edits and all gates
    run at every step, every transition is judged by construction; dispatch-time refusal
    (blocking `/implement` before it starts) is a hook/runtime SEAM consuming core's verdict,
    not core intercepting events. Posture is three tiers (DD-6): defects block, broken
    promises block, in-flight gaps advisory-warn (visible, never silent, never blocking). (C)
    VERSIONING (DD-10) — each bundle REQ carries a semver version and a per-REQ version LOG
    (every version + its text; RDQ-5/RDQ-10); a lifecycle-keyed, semver-gated stale-pin model
    (RDQ-7/DD-12): a MAJOR/MINOR bump = meaning changed → an UNIMPLEMENTED bundle's downstream
    re-pins and re-works, a DELIVERED bundle keeps old specs immutable at their
    (still-log-resolvable) pin, demands a NEW spec for the new version, and drops out of
    `delivered` until that spec is `implemented`; a PATCH bump = wording only → free, pins
    auto-satisfy within the same major.minor. Plus (D) `requirements[]` required from `ready`
    onward INCLUDING `delivered` (DD-5). Legacy corpus reconciled in one sweep (RDQ-2/RDQ-9,
    amended 2026-07-14): the `agent-definitions` cluster is DEPRECATED — the bundle goes
    terminal (`deprecated`) together with SPEC-002/003/004, no backfill (backfilling would have
    minted a delivered bundle with 18 REQs and zero implemented-spec coverage, which the new
    gate step would itself block); BUNDLE-011's REQ-007/010 supports are re-stated via
    retroactive SPEC-053; and `version: 1.0.0` + `@1.0.0` is stamped on every remaining live REQ
    + ref so mandatory-pin validation turns on with zero grandfathering. Anchors present:
    `ResolveArtifactStatus` (pkg/gate/artifact_status.go:151), `ClassifyStatusDrift`
    (pkg/gate/status_drift.go, the block/advisory pattern), `ValidateAll`
    (cmd/backstop/gate.go:1442, the corpus view). Spec Seeds below carry the full delivery
    shape (Seed 1 versioning+log schema + resolution → Seed 2 reconciliation: deprecate the
    legacy cluster + re-state BUNDLE-011 via SPEC-053 + `1.0.0` stamp → Seed 3 gate step +
    coverage + stale-pin).
  assumptions:
    - >
      The `status_drift` full-corpus infrastructure (`ResolveArtifactStatus`,
      `ClassifyStatusDrift`, `SplitDriftResult`) is reusable as the substrate for the new
      `requirement_traceability` step's block/advisory surfaces (DD-7/DD-8).
    - >
      The bundle `requirements[]` schema tolerates the added per-REQ `version:` and
      version-log fields without a schema-format rewrite — confirmed at the 0.5.0 promotion,
      where each REQ's `version: 1.0.0` validated clean.
    - >
      Dispatch-time refusal is delivered by the hook/runtime SEAM that consumes core's
      verdict (the gate-on-implement hook today, the opencode runtime later); core supplies
      the deterministic state verdict and does not intercept events (DD-11/REQ-012).
    - >
      The whole corpus is reachable at both validate time (`ValidateAll`) and gate time, so
      the resolution check and the coverage step can each see every artifact — the DD-8
      placement split is feasible.

requirements:
  - id: REQ-001
    version: "1.0.0"
    text: >
      `artifact validate` must RESOLVE every `supports` ref in both directions: the named
      bundle must exist AND the cited REQ id must be declared in that bundle's
      `requirements[]` array. A ref that resolves to a missing bundle or an undeclared REQ
      is an error-severity defect and blocks, at ANY citing-artifact status (a `draft` spec
      citing a bundle REQ is normal flow — the ref just has to resolve). Format-shape
      (`supportsRe`) is NOT resolution. (DD-4, DD-8; RDQ-1/RDQ-3.)
  - id: REQ-002
    version: "1.0.0"
    text: >
      Every `supports` ref must carry a MANDATORY exact-semver version pin of the form
      `bundle-name:REQ-NNN@MAJOR.MINOR.PATCH`; `supportsRe` (pkg/validate/spec.go:16 and
      pkg/validate/issue.go) is extended to require it. Pinning is mandatory on every ref —
      no optional pinning, no default-latest — and an unpinned ref is an error-severity
      defect. (DD-4, DD-10; RDQ-6.)
  - id: REQ-003
    version: "1.0.0"
    text: >
      Resolution (REQ-001) must additionally require the ref's pinned version to match a
      REAL entry in the cited REQ's version LOG; a pin to a version that never existed is a
      validate-time defect and blocks, so fabricated versions die at validate rather than
      surviving to the gate. (DD-4, DD-10; RDQ-10.)
  - id: REQ-004
    version: "1.0.0"
    text: >
      Each bundle REQ in `requirements[]` must carry a per-REQ semver `version:`
      (MAJOR.MINOR.PATCH) AND a version LOG that retains EVERY version of that REQ together
      with that version's text — not only the current one — so the differences between
      versions are recoverable from the artifact itself, not git archaeology. (DD-10;
      RDQ-5/RDQ-10.)
  - id: REQ-005
    version: "1.0.0"
    text: >
      `requirements[]` must be required from `ready` maturity onward INCLUDING `delivered`:
      the terminal-state exemption (pkg/validate/bundle.go:94) must no longer skip the
      `requirements[]` check for `delivered` bundles. It remains required at `defined`/`ready`
      as today; this only extends it, never relaxes it. (DD-5.)
  - id: REQ-006
    version: "1.0.0"
    text: >
      The coverage/traceability enforcement must be a NEW gate step named
      `requirement_traceability`. The reserved `ledger_integrity` step name
      (pkg/gate/result.go:23) must NOT be claimed — it stays reserved for SPEC-010's
      provenance ledger hash chain. Step names are part of the gate's JSON output contract.
      (DD-7; RDQ-1.)
  - id: REQ-007
    version: "1.0.0"
    text: >
      Enforcement must be split by placement: the resolution + mandatory-pin + log-match
      checks (REQ-001…REQ-003) live in `artifact validate` (`ValidateAll`,
      cmd/backstop/gate.go:1442), while the coverage checks (REQ-008…REQ-010) live in the
      `requirement_traceability` gate step, built on the `status_drift` full-corpus
      block/advisory pattern (`ClassifyStatusDrift`/`SplitDriftResult`). Validate never
      judges a maturity TRANSITION; the gate judges the RESULTING state, so there is no
      promotion chicken-and-egg. (DD-8; RDQ-3.)
  - id: REQ-008
    version: "1.0.0"
    text: >
      The `requirement_traceability` step must block a `delivered` bundle unless EVERY spec
      citing it is success-terminal (`implemented`). A `draft`/`ready` spec hanging off a
      `delivered` bundle is a broken-promise violation. (DD-1.)
  - id: REQ-009
    version: "1.0.0"
    text: >
      The step must block a `delivered` bundle unless EVERY one of its REQs is supported by
      ≥1 requirement in an `implemented` spec. Coverage is per-REQ, not aggregate — each REQ
      needs a live `implemented` supporter. (DD-2.)
  - id: REQ-010
    version: "1.0.0"
    text: >
      Support must NOT flow through a `replaced` spec: a bundle REQ supported ONLY by a
      `replaced` spec counts as UNCOVERED for REQ-009, and the supports claim must be
      re-stated on the replacing spec or another live spec. (DD-3.)
  - id: REQ-011
    version: "1.0.0"
    text: >
      Issue requirements tracing to a bundle REQ must be first-class lineage links —
      resolution-checked (REQ-001…REQ-003) and queryable in the traceability graph — but
      must NEVER satisfy REQ-009 coverage. Only `implemented` SPEC requirements close a
      bundle REQ; issue support is lineage, not coverage. (DD-9; RDQ-4.)
  - id: REQ-012
    version: "1.0.0"
    text: >
      Chain-of-custody must be enforced as a corpus-STATE invariant, not an event handler:
      the `requirement_traceability` step must BLOCK any corpus state in which a downstream
      artifact exists whose upstream chain does not verify — a spec advanced past the depth
      its bundle chain supports, a plan existing for a spec whose bundle chain does not
      verify, or a `delivered` bundle without full coverage (REQ-008…REQ-010 at full depth).
      Core enforces STATES, not events: because promotions land as edits and all gates run at
      every verification step (the no-check-filtering razor), every transition is judged by
      construction. Dispatch-time REFUSAL — blocking an `/implement` run before it starts — is
      explicitly an enforcement SEAM consumed by hooks/runtime (the gate-on-implement hook
      today, the opencode runtime later): core provides the deterministic verdict on the
      corpus state; it does not intercept events. (DD-11; RDQ-8.)
  - id: REQ-013
    version: "1.0.0"
    text: >
      Enforcement posture must follow loud≠blocking as THREE tiers: (a) DEFECTS — a dangling,
      unpinned, or fabricated-version ref (REQ-001…REQ-003) — BLOCK; (b) BROKEN PROMISES — a
      coverage failure on a `delivered` bundle, or any corpus state where a downstream
      artifact's upstream chain does not verify (REQ-008…REQ-010, REQ-012) — BLOCK; (c)
      IN-FLIGHT GAPS — an uncovered REQ on a non-`delivered` bundle whose downstream has not
      advanced past what its chain supports — an ADVISORY WARNING on the non-policied advisory
      surface (like `status_drift`'s advisory split: visible, never blocking, never
      policy-upgradable). In-flight gaps are visible-but-quiet, NEVER silent and NEVER
      blocking. (DD-6.)
  - id: REQ-014
    version: "1.0.0"
    text: >
      The stale-pin model must be lifecycle-keyed and semver-gated: a PATCH bump is
      wording-only and FREE (pins auto-satisfy within the same major.minor); a MAJOR or MINOR
      bump means the meaning changed, so for an UNIMPLEMENTED bundle every downstream spec/plan
      must re-pin to the new version and re-work before proceeding, while for a DELIVERED
      bundle the old specs stay IMMUTABLE at their (still-log-resolvable) pin, the new version
      demands a NEW spec, and the bundle drops OUT of `delivered` until that spec is
      `implemented`. (DD-12; RDQ-7.)
  - id: REQ-015
    version: "1.1.0"
    versions:
      - version: "1.0.0"
        text: >
          A one-time legacy reconciliation + version-stamp sweep must run so the corpus is
          uniformly explicit and green when mandatory-pin validation turns on: BACKFILL the
          delivered `agent-definitions` bundle by lifting its prose REQs into a formal
          `requirements[]` array; re-state BUNDLE-011's REQ-004/007/010 supports onto live
          `implemented` specs; terminally retire the stale draft specs SPEC-002/003/004; and
          stamp `version: 1.0.0` + `@1.0.0` on every live REQ and supports ref. It must land
          WITH or immediately before the REQ-002 enforcement flip, and be routed through the
          artifact agents (never hand-edited). (RDQ-2/RDQ-9.)
      - version: "1.1.0"
        text: >-
          A one-time legacy reconciliation + version-stamp sweep must run so the corpus is
          uniformly explicit and green when mandatory-pin validation turns on: DEPRECATE the
          `agent-definitions` cluster — take the `agent-definitions` bundle terminal (`deprecated`)
          together with the stale draft specs SPEC-002/003/004, with NO backfill of its
          `requirements[]` (a terminal bundle sits outside the delivered-gate by the same terminal
          exclusion the gate already honors, so this is honest recording, not grandfathering;
          backfilling would instead mint a delivered bundle with 18 REQs and zero implemented-spec
          coverage that the gate step would block); re-state BUNDLE-011's REQ-007/010 supports via a
          retroactive `implemented` spec (SPEC-053 — SPEC-039's work landed via ISSUE-018, and an
          implemented retroactive spec archives it as coverage, since issues are lineage not
          coverage); and stamp `version: 1.0.0` + `@1.0.0` on every REMAINING live REQ and supports
          ref. It must land WITH or immediately before the REQ-002 enforcement flip, and be routed
          through the artifact agents (never hand-edited). (RDQ-2 amended 2026-07-14; RDQ-9.)
    text: >-
      A one-time legacy reconciliation + version-stamp sweep must run so the corpus is
      uniformly explicit and green when mandatory-pin validation turns on: DEPRECATE the
      `agent-definitions` cluster — take the `agent-definitions` bundle terminal (`deprecated`)
      together with the stale draft specs SPEC-002/003/004, with NO backfill of its
      `requirements[]` (a terminal bundle sits outside the delivered-gate by the same terminal
      exclusion the gate already honors, so this is honest recording, not grandfathering;
      backfilling would instead mint a delivered bundle with 18 REQs and zero implemented-spec
      coverage that the gate step would block); re-state BUNDLE-011's REQ-007/010 supports via a
      retroactive `implemented` spec (SPEC-053 — SPEC-039's work landed via ISSUE-018, and an
      implemented retroactive spec archives it as coverage, since issues are lineage not
      coverage); and stamp `version: 1.0.0` + `@1.0.0` on every REMAINING live REQ and supports
      ref. It must land WITH or immediately before the REQ-002 enforcement flip, and be routed
      through the artifact agents (never hand-edited). (RDQ-2 amended 2026-07-14; RDQ-9.)
---

# Requirement Traceability — Close the Bundle→Spec Hop

## Current Thinking

Backstop's central promise is that **a green gate on an `implemented` spec is mechanical
proof of every requirement in that spec** — not a reviewer's assertion, a deterministic
consequence of the chain: spec requirements → claims → mandated tests →
test_verification (existence) → substantiveness → pack-engine test run (pass). DIR-015
(delivered 2026-07-13) scoped the contracts/tests/coverage dimensions to `implemented`
specs so this chain is enforced exactly where it should be and nowhere it shouldn't.

**What is NOT welded is the hop ABOVE the spec: bundle REQ → spec requirement.** That
hop is carried by the `supports:` field on a spec (or issue) requirement — e.g.
`supports: agent-definitions:REQ-004`. Two things are wrong with it today, and a third
sits one level up in the bundle itself:

1. **`supports` is format-checked, never resolved.** `supportsRe` (pkg/validate/spec.go:16,
   mirrored in pkg/validate/issue.go) asserts the string LOOKS like `kebab:REQ-NNN` and
   stops there. Nothing checks that the named bundle exists, and nothing checks that the
   cited REQ is actually declared in that bundle's `requirements[]`. A spec can cite
   `nonexistent-bundle:REQ-999` and validate green.

2. **The coverage direction is entirely unchecked.** No dimension asks "is every bundle
   REQ supported by at least one spec requirement?" A bundle can declare REQ-001…REQ-015,
   have zero specs cite any of them, march to `delivered`, and nothing objects. The
   chain is proven bottom-up (spec down) but *assumed* top-down (bundle to spec).

3. **The top of the chain can be structurally empty at success.** `requirements[]` is
   enforced only at `defined`/`ready` (`validateBundleRequirements`,
   pkg/validate/bundle.go:528), and the terminal-state exemption (bundle.go:94) skips it
   for `delivered`. So `agent-definitions` — a `delivered` bundle — declares NO
   `requirements[]` at all; its "REQs" live only in prose, which is why the 10 specs
   citing `agent-definitions:REQ-NNN` are dangling and yet green.

**Scope expanded 2026-07-14 — the bundle now spans the whole traceable lifecycle, not
just the delivered checkpoint, and it is all resolved.** Two founder-decided additions
widen it from "verify the existing hop" to "make the chain enforceable at every step AND
versioned":

- **Chain-of-custody as a STATE invariant** (DD-11, from RDQ-8). Enforcement is not an event
  handler on transitions but a corpus-STATE invariant: the `requirement_traceability` step
  blocks any state in which a downstream artifact's upstream chain does not verify — a spec
  advanced past the depth its bundle chain supports, a plan existing for a spec whose bundle
  chain doesn't verify, `delivered` without full coverage (DD-1/DD-2/DD-3). The founder's
  reasoning: there is no sense promoting a spec to planning if it fails validation, because
  the plan will inherently be incomplete; and a plan shouldn't start until the entire chain
  of custody is verified. Core enforces STATES not events — because promotions land as edits
  and every gate runs at every verification step (the no-check-filtering razor), every
  transition is judged by construction. Dispatch-time refusal (blocking an `/implement` run
  before it starts) is a downstream enforcement SEAM consumed by hooks/runtime (the
  gate-on-implement hook today, the opencode runtime later): core supplies the deterministic
  verdict on the state; it does not intercept the event. This composes with the validate/gate
  split (RDQ-3): promotions land as edits, the gate judges the RESULTING state, no promotion
  chicken-and-egg.
- **Requirement versioning, log-backed** (DD-10, from RDQ-5/RDQ-6/RDQ-7/RDQ-10). Each bundle
  REQ carries a per-REQ semver `version:` AND a version LOG that retains every version of the
  REQ together with that version's text — not just the current one. Every supports ref pins an
  EXACT version (`bundle:REQ-NNN@1.2.0`, mandatory — no default-latest), and resolution
  requires that pin to match a real historical entry in the log, so a pin to a version that
  never existed dies at validate rather than surviving to the gate. This is what makes the
  interrogability goal real: "what exactly did the spec pinned @1.1.0 implement, and how does
  it differ from 2.0.0" is answerable from the artifacts themselves, not git archaeology. The
  stale-pin behaviour is lifecycle-keyed and semver-gated (DD-12): a **PATCH** bump is
  wording-only and free (pins auto-satisfy within the same major.minor); a **MAJOR/MINOR** bump
  means the requirement's MEANING changed, so an **unimplemented** bundle forces its downstream
  specs/plans to re-pin and re-work before proceeding, while a **delivered** bundle keeps its
  old specs immutable at their pinned version (history is never rewritten, and the pin still
  resolves against the log), demands a NEW spec for the new version, and drops out of
  `delivered` until that spec is `implemented`. This turns after-the-fact requirement change
  from silent drift into a traceable, queryable fact — the mechanism behind the issue-lineage
  model (DD-9): cover everything in one shot at planning time; issues exist for the
  after-the-fact follow-up (bug fixes, edge cases, or modifying the requirement itself, which
  appends a new log entry and bumps its version).

**The corpus is already drifting under the original three holes (audited 2026-07-14).**
`./bin/backstop artifact validate` is currently ALL-GREEN over all of the following:

- **10 dangling supports refs:** SPEC-002/003/004 (all `draft`) cite
  `agent-definitions:REQ-004..REQ-018`; the `delivered` `agent-definitions` bundle has no
  `requirements[]` array — those REQs exist only in prose, so nothing the refs point at is
  declared.
- **BUNDLE-011** (`delivered`): REQ-004 is supported by nothing; REQ-007 and REQ-010 are
  supported only by SPEC-039, which is `replaced` — so under a no-flow-through-`replaced`
  rule they are effectively uncovered.
- **BUNDLE-012, BUNDLE-013** (`delivered`): fully covered by `implemented` specs — the
  proof the chain works *when it is followed*. These must stay green.
- **Non-terminal bundles with uncovered REQs — correct pipeline state, must NOT block
  (advisory-warn only, per the three-tier posture):** baseline (`ready`, 15/15 uncovered —
  SPEC-019 exists but carries no supports refs), pack-manifest-authoring (`ready`,
  REQ-026/028/029/032), stack-aware-traceability (`ready`, REQ-009). A `ready` bundle whose
  specs aren't written yet is the pipeline mid-flight — surfaced as an advisory warning,
  never a block.

The load-bearing frame: **enforcement blocks defects and broken promises, and surfaces
in-flight gaps as advisory warnings — never silent, never nagging-as-a-block**
([[feedback_loud_not_blocking]]). A dangling, unpinned, or fabricated-version ref is a defect
regardless of who cites it → block. An uncovered REQ blocks once a downstream artifact has
advanced past what its chain supports (a broken-promise STATE); before that it surfaces as an
advisory warning — visible but never blocking. This asymmetry — resolution enforced
everywhere, coverage blocking at broken-promise states and advisory-warning in-flight — is
the crux of the whole design.

## Draft Requirements

The formal, traceable requirements are enumerated in the `requirements:` frontmatter array
(REQ-001…REQ-015). They were lifted FAITHFULLY from the resolved decisions (RDQ-1…RDQ-10)
and the Draft Design Decisions (DD-1…DD-12) during the `defined` promotion — no new scope was
invented to clear the gate; each REQ names the DD/RDQ it descends from. Each REQ also dogfoods
the bundle's own convention (RDQ-5/DD-10): it carries `version: 1.0.0`, the initial log entry
the RDQ-9 backfill sweep would stamp. Summary of what the bundle commits to, grouped:

- **Resolution in `artifact validate`** (REQ-001…REQ-003, from DD-4/DD-8/DD-10 via
  RDQ-1/RDQ-3/RDQ-6/RDQ-10) — every `supports` ref resolves both directions, carries a
  mandatory exact-semver pin, and that pin matches a real version-log entry; dangling,
  unpinned, or fabricated-version refs are defects that block at any citing status.
- **Bundle structural** (REQ-004, REQ-005, from DD-10/DD-5) — each REQ carries a semver
  version + a version LOG (every version and its text); `requirements[]` is required at
  `delivered` too (the terminal exemption no longer skips it).
- **Gate step + placement** (REQ-006, REQ-007, from DD-7/DD-8 via RDQ-1/RDQ-3) — coverage is
  a new `requirement_traceability` gate step (not `ledger_integrity`) on the status_drift
  block/advisory pattern; resolution stays in validate; the gate judges resulting state.
- **Delivered coverage** (REQ-008…REQ-011, from DD-1/DD-2/DD-3/DD-9 via RDQ-4) — a `delivered`
  bundle needs every citing spec `implemented` and every REQ covered by ≥1 `implemented` spec
  requirement; `replaced` specs don't flow through; issue requirements are lineage, never
  coverage.
- **Every-state enforcement + three-tier posture** (REQ-012, REQ-013, from DD-11/DD-6 via
  RDQ-8) — the gate blocks any corpus STATE where a downstream artifact's upstream chain
  doesn't verify (dispatch refusal is a hook/runtime seam consuming core's verdict, not core
  intercepting events); posture is three tiers — defects block, broken promises block,
  in-flight gaps advisory-warn (visible, never silent, never blocking).
- **Versioning stale-pin model** (REQ-014, from DD-12 via RDQ-7) — PATCH free within
  major.minor; MAJOR/MINOR forces downstream re-pin/re-work (unimplemented) or a new spec +
  drop-out-of-`delivered` (delivered), old pins staying log-resolvable.
- **Legacy reconciliation + version-stamp** (REQ-015, from RDQ-2 amended/RDQ-9) — the one-time
  sweep that DEPRECATES the `agent-definitions` cluster (bundle + SPEC-002/003/004 terminal, no
  backfill), re-states BUNDLE-011's REQ-007/010 via retroactive SPEC-053, and stamps `1.0.0` on
  every remaining live REQ + ref so mandatory-pin validation turns on green.

## Draft Design Decisions

DD-1…DD-6 are the founder's SETTLED upfront constraints (2026-07-14); DD-7…DD-12 were added
across the same-day OQ-resolution rounds — DD-7/DD-8/DD-9 from OQ1/OQ3/OQ4 (RDQ-1/3/4),
DD-10/DD-11 as the two scope expansions (refined by RDQ-5/6/7/10 and RDQ-8), DD-12 from RDQ-7.
All are decisions, not open questions, and must not be re-litigated; they were lifted into the
formal `requirements[]` array at the `defined` promotion (see Draft Requirements).

**DD-1 — A `delivered` bundle requires every citing spec to be success-terminal
(`implemented`).** A `draft` or `ready` spec hanging off a `delivered` bundle is a
violation: the bundle claims done while a piece of its work is unfinished. *Rationale:*
`delivered` is a success-terminal claim over the whole bundle; an unfinished spec beneath
it makes that claim false. → REQ-008.

**DD-2 — A `delivered` bundle requires every bundle REQ to be supported by ≥1 requirement
in an `implemented` spec.** Coverage is per-REQ, not aggregate: each REQ must have a live
`implemented` supporter. *Rationale:* this is the direction that makes "every bundle
requirement is verified down to implementation" mechanically true rather than assumed.
→ REQ-009.

**DD-3 — No flow-through for `replaced` specs.** A bundle REQ supported ONLY by a
`replaced` spec counts as UNCOVERED; the supports claim must be re-stated in the replacing
spec or another live spec. *Rationale:* a replaced spec's requirements are no longer the
live contract; letting support flow through a tombstone would let a bundle claim coverage
it no longer has. (This is why BUNDLE-011's REQ-007/REQ-010, supported only by the
`replaced` SPEC-039, are effectively uncovered.) → REQ-010.

**DD-4 — Both-direction resolution at ANY citing-spec status, on a mandatory log-resolved
version pin.** Every `supports` ref must resolve to a real bundle AND a REQ declared in that
bundle's `requirements[]`, and must carry its mandatory exact-semver pin (DD-10) that matches
a real entry in that REQ's version LOG (DD-10/RDQ-10). This is INDEPENDENT of the citing
spec's status — a `draft` spec citing a bundle REQ is normal pipeline flow; the ref simply
has to resolve. *Rationale:* a dangling, unpinned, or fabricated-version ref is a defect the
moment it is written, not only at delivery; format-shape (`supportsRe`) is not resolution.
→ REQ-001/REQ-002/REQ-003.

**DD-5 — `requirements[]` is required from `ready` onward, INCLUDING `delivered`.** This
closes the `agent-definitions` hole (a `delivered` bundle with no `requirements[]`). It is
already required at `defined`/`ready` today (`validateBundleRequirements`) — DO NOT relax
that; extend it so the terminal-state exemption no longer skips it for `delivered`.
*Rationale:* the top of the chain cannot be structurally empty at the exact moment it
claims success; a `delivered` bundle with no declared REQs has nothing for coverage to
verify. → REQ-005.

**DD-6 — Enforcement posture per loud≠blocking, in THREE tiers.** (a) A dangling, unpinned,
or fabricated-version ref (DD-4) is a DEFECT → BLOCK. (b) A coverage failure on a `delivered`
bundle at full depth (DD-1/DD-2/DD-3), or any corpus state where a downstream artifact's
upstream chain does not verify (DD-11), is a BROKEN PROMISE → BLOCK. (c) An uncovered REQ on a
non-`delivered` bundle whose downstream has not advanced past what its chain supports is an
IN-FLIGHT GAP → ADVISORY WARN on the non-policied advisory surface (like `status_drift`'s
advisory split — visible, never blocking, never policy-upgradable), NEVER silent. *Rationale:*
[[feedback_loud_not_blocking]] — block defects and broken promises, but make in-flight gaps
visible-but-quiet rather than invisible, so a real gap is never hidden yet the normal in-flight
pipeline is never blocked. The baseline/pack-manifest-authoring/stack-aware bundles above are
the concrete in-flight-warn cases (uncovered but correctly pre-`delivered`). → REQ-013.

**DD-7 — The delivered/coverage check is a NEW gate step named `requirement_traceability`.**
*(from RDQ-1.)* Mint a new step; do NOT claim the reserved `ledger_integrity`
(pkg/gate/result.go:23), which stays reserved for SPEC-010's provenance hash chain.
*Rationale:* structural traceability and a cryptographic ledger hash chain are genuinely
different checks; step names are part of the gate's JSON output contract, and overloading
one name to save a string breeds ambiguity a later provenance feature would have to fight.
→ REQ-006.

**DD-8 — Split placement: resolution in `artifact validate`, coverage checks in the gate
step.** *(from RDQ-3.)* The both-direction resolution + mandatory-pin + log-match check (DD-4)
lives in `artifact validate` (`ValidateAll`, cmd/backstop/gate.go:1442, has the corpus view).
The coverage checks (DD-1/DD-2/DD-3), enforced as a state invariant (DD-11), live in the new
`requirement_traceability` gate step. Validate never judges a maturity TRANSITION; the gate
judges the resulting STATE. *Rationale:* keeps validate fast/local for the defect check while
the corpus-spanning promise checks live in the gate — and judging resulting state rather than
the transition edit dissolves the promotion chicken-and-egg. → REQ-007.

**DD-9 — Issue requirements are first-class lineage links but CANNOT satisfy coverage.**
*(from RDQ-4.)* An issue requirement tracing to a bundle REQ is resolution-checked (DD-4) and
queryable in the traceability graph — so a REQ can be interrogated after the fact ("which
issues patched what the spec missed"). But ONLY `implemented` spec requirements count toward
DD-2 coverage; issue support never closes a bundle REQ. *Rationale (founder):* the discipline
means everything should be covered in one shot at planning time; issues exist for
after-the-fact follow-up — bug fixes, edge cases, or modifying the requirement itself — so
they are lineage, not primary coverage. Letting an issue satisfy coverage would also blur the
[[feedback_artifact_tracks]] boundary (specs come from bundles; issues never do). → REQ-011.

**DD-10 — Requirement versioning: per-REQ semver, mandatory exact pin, backed by a version
LOG.** *(scope expansion; RDQ-5/RDQ-6/RDQ-10.)* Each bundle REQ carries a per-REQ `version:`
in MAJOR.MINOR.PATCH semver (RDQ-5) AND a version LOG in `requirements[]` retaining every
version of the REQ together with that version's text — not just the current one (RDQ-10).
Every supports ref pins an EXACT version — `bundle-name:REQ-NNN@1.2.0` — with pinning MANDATORY
on every ref (no optional pinning, no default-latest; RDQ-6), and resolution requires the pin
to match a real LOG entry (DD-4). *Rationale:* semver is chosen for expressiveness — bump SIZE
is a first-class signal the stale-pin model (DD-12) keys off. Mandatory full pinning means
every trace states exactly what it implemented; the ref churn a bump causes is the POINT (an
explicit re-pin is the traceable record of re-work), not a cost. The log makes the chain
INTERROGABLE from the artifacts alone — differences between versions are recoverable without
git archaeology — and lets a fabricated version die at validate. This is the mechanism
after-the-fact evolution (DD-9) rests on. → REQ-002/REQ-003/REQ-004.

**DD-11 — Chain-of-custody is a STATE invariant the gate enforces; dispatch refusal is a
seam.** *(scope expansion; RDQ-8.)* Core enforces STATES, not events: the
`requirement_traceability` step blocks any corpus state in which a downstream artifact's
upstream chain does not verify — a spec advanced past the depth its bundle chain supports, a
plan existing for a spec whose bundle chain doesn't verify, `delivered` without full coverage
(DD-1/DD-2/DD-3). Because promotions land as edits and every gate runs at every verification
step (the no-check-filtering razor), every transition is judged by construction. Dispatch-time
refusal — blocking an `/implement` run before it starts — is an enforcement SEAM consumed by
hooks/runtime (the gate-on-implement hook today, the opencode runtime later); core supplies the
deterministic verdict on the state, it does not intercept the event. *Rationale (founder):*
there is no sense promoting a spec to planning if it fails validation, because the plan will
inherently be incomplete; and a plan shouldn't start until the entire chain of custody is
verified — expressed as a STATE check so core stays event-agnostic and the runtime/hook layer
consumes the verdict. → REQ-012.

**DD-12 — Stale-pin behaviour is lifecycle-keyed and semver-gated.** *(from RDQ-7.)* When a
REQ's version bumps, what happens downstream depends on BOTH the bump size and the bundle's
lifecycle state:
- **PATCH bump = wording only → FREE.** Pins auto-satisfy within the same major.minor; no
  rework, no un-delivering.
- **MAJOR or MINOR bump = meaning changed → the rev/new-spec machinery fires**, keyed on the
  bundle's state: if the bundle is UNIMPLEMENTED (in-flight), everything downstream must rev
  and re-work — downstream specs/plans re-pin to the new version before proceeding; if the
  bundle is DELIVERED, old specs stay IMMUTABLE at their pinned version (history is never
  rewritten, and the pin still resolves against the log — DD-10), the new requirement version
  demands a NEW spec, and the bundle drops OUT of `delivered` until that new spec is
  `implemented`.
*Rationale (founder):* meaning-changes must propagate but wording-fixes must not nag —
semver's major/minor-vs-patch line is exactly that distinction, so the enforcement keys off
it. Immutability of delivered specs preserves history; re-entering the `delivered` gate on a
meaning-change keeps "delivered" honest against the CURRENT requirement. → REQ-014.

## Resolved Design Questions

OQ1–OQ4 were resolved 2026-07-14 (RDQ-1…RDQ-4); OQ5–OQ9 the same day (RDQ-5…RDQ-9); OQ10 last
(RDQ-10). All ten are resolved. Original framing + resolution + rationale below; the decisions
they produced live in Draft Design Decisions above and the formal requirements[] array.

**RDQ-1 — Gate step name.** *(was OQ1.)* **Resolution (b):** MINT a new gate step
`requirement_traceability`; `ledger_integrity` stays reserved for SPEC-010's provenance hash
chain. → DD-7/REQ-006. *Rationale:* two genuinely different checks; don't overload a
JSON-output-contract name.

**RDQ-2 — Legacy corpus reconciliation.** *(was OQ2.)* **Original resolution (2026-07-14, c/mix):**
backfill the delivered `agent-definitions` (lift its prose REQs into a `requirements[]` array),
re-state BUNDLE-011's REQ-004/007/010 on live `implemented` specs, AND terminally retire the
stale draft specs SPEC-002/003/004. **AMENDED 2026-07-14 (supersedes the backfill half):** the
whole legacy cluster is DEPRECATED instead — `agent-definitions.bundle.md` goes terminal
(`deprecated`) together with SPEC-002/003/004; NO backfill. The BUNDLE-011 half (re-state on live
specs) STANDS, now realized via a retroactive `implemented` SPEC-053 for REQ-007/010. *Why the
amendment:* the cross-spec consistency pass exposed that backfilling `agent-definitions` would
mint a `delivered` bundle with 18 REQs and ZERO implemented-spec coverage once its draft citers
retire — this bundle's own gate step (DD-1/DD-2) would then block it, and keeping it delivered
would demand 18 retroactive-coverage claims with thin anchors (fake rigor, rejected). Deprecation
is honest: the scaffolding-era history is superseded by the living `.claude` roster + agent-guard
+ ISSUE-044's roster-consistency check, and a terminal bundle sits OUTSIDE the delivered-gate by
the same terminal-exclusion the gate already honors — so the zero-grandfathering posture stays
intact (this is honest recording, not a carve-out). SPEC-039's work landed via ISSUE-018; issues
are lineage not coverage, so an `implemented` retroactive spec (SPEC-053) is what archives
REQ-007/010 as real coverage. *Rationale (unchanged half):* the stale drafts are abandoned and
their dangling refs should cease to exist ([[feedback_align_predating_artifacts]]). → Seed 2/REQ-015.

**RDQ-3 — Where each check lives.** *(was OQ3.)* **Resolution (a, split):** resolution check →
`artifact validate`; coverage checks → the `requirement_traceability` gate step. Validate
never judges maturity transitions; the gate judges resulting state, so there is no promotion
chicken-and-egg. → DD-8/REQ-007. *Rationale:* fast/local defect check in validate,
corpus-spanning promise checks in the gate; judging resulting state sidesteps the edit-time
deadlock.

**RDQ-4 — Do issue requirements count as coverage?** *(was OQ4.)* **Resolution (spec-mandatory,
issues-as-lineage):** issue requirements are first-class, resolution-checked, queryable lineage
links — but cannot satisfy coverage; only `implemented` spec requirements count. → DD-9/REQ-011.
*Rationale (founder):* cover everything in one shot at planning time; issues are for
after-the-fact follow-up.

**RDQ-5 — Requirement version format.** *(was OQ5.)* **Resolution: semver per REQ.** Each REQ
carries a per-REQ `version:` in `requirements[]`, MAJOR.MINOR.PATCH. → DD-10/REQ-004.
*Rationale:* expressiveness — bump SIZE is a first-class signal the stale-pin enforcement
(DD-12) keys off.

**RDQ-6 — Pin syntax / mandatory vs optional.** *(was OQ6.)* **Resolution: mandatory full
pin.** Every supports ref pins an exact version, `bundle-name:REQ-NNN@1.2.0` — no optional
pinning, no default-latest. → DD-10/REQ-002. *Rationale:* every trace states exactly what it
implemented; ref churn on bumps is the point (the traceable record of re-work), not a cost.

**RDQ-7 — Stale-pin semantics.** *(was OQ7.)* **Resolution: lifecycle-keyed, semver-gated.**
PATCH bump = wording only → free, pins auto-satisfy within the same major.minor; MAJOR/MINOR
bump = meaning changed → an UNIMPLEMENTED bundle forces downstream to re-pin and re-work, a
DELIVERED bundle keeps old specs immutable at their pin, demands a NEW spec for the new
version, and drops out of `delivered` until that spec is `implemented`. → DD-12/REQ-014.
*Rationale (founder):* meaning-changes must propagate, wording-fixes must not nag; semver's
major/minor-vs-patch line is exactly that distinction, and immutability preserves history while
re-entering the delivered gate keeps "delivered" honest against the current requirement.

**RDQ-8 — Pre-impl gate trigger + trace target.** *(was OQ8.)* **Resolution: enforce as a STATE
invariant at every verification, with dispatch refusal as a seam.** Chain-of-custody is a
corpus-STATE invariant the `requirement_traceability` step enforces — it blocks any state where
a downstream artifact's upstream chain doesn't verify (a spec past its supported depth, a plan
for a spec whose bundle chain doesn't verify, `delivered` without full coverage), `delivered` at
full depth (DD-1/2/3). Because promotions land as edits and all gates run at every step, every
transition is judged by construction. Dispatch-time refusal (blocking `/implement` before it
starts) is a SEAM for hooks/runtime (the gate-on-implement hook today, the opencode runtime
later); core gives the verdict on the state, not the event interception. → DD-11/REQ-012.
*Rationale (founder):* no sense promoting a spec to planning if it fails validation (the plan
will be incomplete), and a plan shouldn't start until the whole chain is verified — expressed
as a state check so core stays event-agnostic (composing with the RDQ-3 split: promotions are
edits, the gate judges resulting state).

**RDQ-9 — Migration of the unversioned corpus.** *(was OQ9.)* **Resolution: explicit backfill
pass.** A one-time sweep stamps each REQ's initial log entry `version: 1.0.0` and `@1.0.0` on
every existing supports ref, folded into the RDQ-2 reconciliation. → Seed 2/REQ-015.
*Rationale:* pinning is mandatory (RDQ-6), so nothing may stay implicit; an explicit stamp makes
the corpus uniformly versioned from day one and lets mandatory-pin validation turn on with zero
grandfathering.

**RDQ-10 — What a version pin resolves against.** *(was OQ10.)* **Resolution (b): recorded
version history — a version LOG per REQ.** The bundle's `requirements[]` retains EVERY version
of each REQ together with that version's text, not just the current one. Resolution (in
`artifact validate`) requires a pin to match a real historical LOG entry — a pin to a version
that never existed is a validate-time defect. → DD-10/DD-4/REQ-003/REQ-004. *Rationale
(founder):* this is what makes the interrogability goal real — "what exactly did the spec
pinned @1.1.0 implement, and how does it differ from 2.0.0" must be answerable from the
artifacts themselves, not git archaeology — and fabricated versions die at validate rather than
surviving to the gate.

## Open Questions

None — all ten open questions (OQ1–OQ10) are resolved and recorded as RDQ-1…RDQ-10 above, and
their decisions are lifted into the formal `requirements[]` array (REQ-001…REQ-015). The bundle
was promoted `exploring` → `defined` on 2026-07-14.

## Spec Seeds

Suggested decomposition into specs, in implementation order; no requirement belongs to two
seeds.

**Seed 1 — Versioning + version-log schema + both-direction resolution in `artifact validate`.**
The foundational mechanism. Add to the bundle `requirements[]` schema a per-REQ semver
`version:` field and a per-REQ version LOG retaining every version + its text (REQ-004), and
validate their shape. Extend `supportsRe` (pkg/validate/spec.go:16 + pkg/validate/issue.go) to
require the mandatory exact pin `bundle-name:REQ-NNN@X.Y.Z` (REQ-002). Implement both-direction
resolution in `ValidateAll` — the named bundle exists, the REQ id is declared, and the pin
matches a real entry in that REQ's version LOG (REQ-001/REQ-003). Narrow the terminal-state
exemption so `requirements[]` is required at `delivered` too (REQ-005). Covers REQ-001, REQ-002,
REQ-003, REQ-004, REQ-005. Depends on nothing. Note: the mandatory-pin ENFORCEMENT flip must not
precede Seed 2's reconciliation sweep, or the unversioned corpus goes red.

**Seed 2 — Legacy reconciliation + `1.0.0` version-stamp sweep.** One-time corpus reconciliation
(REQ-015, RDQ-2 amended 2026-07-14): DEPRECATE the `agent-definitions` cluster — take the
`agent-definitions` bundle terminal (`deprecated`) together with the stale draft specs
SPEC-002/003/004, with NO backfill of its `requirements[]` (a terminal bundle sits outside the
delivered-gate, so this is honest recording; backfilling would mint a delivered bundle with 18
uncovered REQs the gate step would block); re-state BUNDLE-011's REQ-007/010 supports via a
retroactive `implemented` SPEC-053 (SPEC-039's work landed via ISSUE-018; an implemented
retroactive spec archives it as coverage); and stamp `version: 1.0.0` + `@1.0.0` on every
REMAINING live REQ and supports ref. Lands WITH or immediately before Seed 1's mandatory-pin
enforcement flips on, so the corpus is uniformly explicit and green at the moment validation
tightens. Covers REQ-015. Routed through the artifact agents (never hand-edited).

**Seed 3 — `requirement_traceability` gate step: state-invariant coverage + stale-pin model.**
The enforcement keystone. Mint the new `requirement_traceability` gate step (REQ-006, NOT
`ledger_integrity`) on the `status_drift` full-corpus / block-advisory pattern (REQ-007).
Implement delivered-gate coverage — every citing spec `implemented` (REQ-008), every REQ
supported by ≥1 `implemented` spec requirement (REQ-009), no flow-through for `replaced`
(REQ-010), issue requirements resolution-checked but non-coverage (REQ-011). Enforce the chain
as a STATE invariant — block any corpus state where a downstream artifact's upstream chain
doesn't verify, `delivered` at full depth (REQ-012), judging resulting state — with the
three-tier posture: defects and broken promises block, in-flight gaps advisory-warn on the
non-policied surface (REQ-013). Expose dispatch-time refusal as the seam hooks/runtime consume
(gate-on-implement hook today, opencode runtime later); core supplies the verdict, not the
interception (REQ-012). Implement the lifecycle-keyed, semver-gated stale-pin model (REQ-014)
reading the version LOG: PATCH auto-satisfies within major.minor; MAJOR/MINOR forces downstream
re-pin/re-work for an unimplemented bundle, or a new spec + drop-out-of-`delivered` for a
delivered one. Covers REQ-006, REQ-007, REQ-008, REQ-009, REQ-010, REQ-011, REQ-012, REQ-013,
REQ-014. Depends on Seed 1 (version + log schema + resolution) and Seed 2 (green corpus).

## Notes / References

- **The corpus sweep this needs already exists.** `ResolveArtifactStatus`
  (pkg/gate/artifact_status.go:151) resolves every artifact record project-wide;
  `ClassifyStatusDrift` (pkg/gate/status_drift.go) is the worked example of a full-corpus
  check that splits a policied BLOCK surface from a non-policied ADVISORY surface
  (`SplitDriftResult`) — the block/advisory shape DD-6's three tiers want (the advisory tier
  IS this surface), and the pattern the new `requirement_traceability` step (DD-7/DD-8) and its
  state-invariant coverage (DD-11) reuse. `ValidateAll` (cmd/backstop/gate.go:1442) holds the
  whole corpus for the resolution check.
- **`supports` is regex-only today, and has no version segment.** `supportsRe =
  ^[a-z0-9-]+:REQ-\d{3}$` (pkg/validate/spec.go:16; same in pkg/validate/issue.go). It asserts
  shape, resolves nothing, and Seed 1 extends it to require the mandatory `@X.Y.Z` pin resolved
  against the REQ's version log.
- **The terminal exemption is the empty-top hole.** pkg/validate/bundle.go:94 skips
  `validateBundleRequirements` (and the maturity/placeholder gates) for terminal statuses,
  which is why `delivered` bundles escape the `requirements[]` requirement (DD-5 narrows this
  so `requirements[]` survives the exemption).
- **`ledger_integrity` stays reserved (DD-7/RDQ-1).** pkg/gate/result.go:23
  (`StepLedgerIntegrity = "ledger_integrity"`), in the step order at :52, currently skipped;
  SPEC-010 documents it as the provenance ledger hash chain — the new step is
  `requirement_traceability`, not this.
- **Dispatch-time refusal is a downstream seam, not core (DD-11/REQ-012).** Core supplies the
  deterministic verdict on corpus state; the gate-on-implement hook consumes it today and the
  opencode runtime will later. Backstop is the layer above tool-use hooks
  ([[project_higher_order_control_plane]]); it does not intercept events.
- [[project_gate_scoped_to_implemented]] — DIR-015 scoped contracts/tests/coverage to
  `implemented` specs; this bundle is the layer ABOVE it (bundle→spec), extending the same
  "mechanical proof" property one hop up, enforced as a state invariant.
- [[feedback_loud_not_blocking]] — governs DD-6's three tiers and DD-12: block defects + broken
  promises; surface in-flight gaps as advisory warnings (never silent); keep a PATCH bump free.
- [[feedback_no_check_filtering]] — the razor DD-11 leans on: all gates run at every
  verification step, so every corpus state (and thus every transition) is judged by construction.
- [[feedback_align_predating_artifacts]] — governs RDQ-2's legacy reconciliation (amended
  2026-07-14): the scaffolding-era `agent-definitions` cluster is DEPRECATED (bundle +
  SPEC-002/003/004 terminal — superseded by the living `.claude` roster + agent-guard +
  ISSUE-044), BUNDLE-011's REQ-007/010 re-stated on a live spec (retroactive SPEC-053); deprecate
  vs backfill because a terminal bundle is honest recording outside the gate, not grandfathering.
- [[project_artifact_terminal_states]] — the terminal-exclusion RDQ-2's deprecation leans on: a
  `deprecated` bundle sits outside the delivered-gate, so deprecating `agent-definitions` is
  honest recording (zero-grandfathering posture intact), not a carve-out.
- [[feedback_artifact_tracks]] — the issue-vs-spec track boundary underpinning DD-9/RDQ-4.
- [[project_artifact_terminal_states]] — ISSUE-031's `replaced`/`delivered` semantics that
  DD-1/DD-3/DD-12 build on (no flow-through for `replaced`; `delivered` is the success
  boundary; delivered specs immutable).
- Corpus evidence (audited 2026-07-14, `artifact validate` all-green): 10 dangling refs
  (SPEC-002/003/004 → `agent-definitions:REQ-004..018`, bundle has no requirements[] — Seed 2
  DEPRECATES the whole cluster: bundle + those specs go terminal, no backfill); BUNDLE-011 REQ-004
  uncovered, REQ-007/010 only on `replaced` SPEC-039 (Seed 2 re-states REQ-007/010 via retroactive
  SPEC-053); BUNDLE-012/013 fully covered by
  `implemented` specs (must stay green); baseline / pack-manifest-authoring /
  stack-aware-traceability non-terminal with uncovered REQs (must not BLOCK — advisory-warn
  only, correctly pre-`delivered`).

## Version History

- **0.1.0 (2026-07-14, exploring)** — Initial bundle. Problem framed on the verified corpus
  state: the bundle→spec hop is convention-only (three holes — `supports` resolved by
  regex-shape only per pkg/validate/spec.go:16; coverage direction unchecked; `requirements[]`
  unenforced at `delivered` per the bundle.go:94 terminal exemption). Recorded the founder's SIX
  settled constraints (DD-1…DD-6) and left four open questions OPEN (OQ1 step-name, OQ2
  legacy-corpus reconciliation, OQ3 validate-vs-gate placement, OQ4 whether issue requirements
  count as coverage). No formal `requirements[]`, Spec Seeds, or `defined`-promotion. Maturity
  `exploring`.
- **0.2.0 (2026-07-14, exploring)** — RESOLVED OQ1–OQ4 (RDQ-1…RDQ-4) and EXPANDED scope. RDQ-1
  mint a new `requirement_traceability` gate step; RDQ-2 reconcile the legacy corpus via a mix;
  RDQ-3 split placement (resolution in `artifact validate`, coverage in the gate step, no
  promotion chicken-and-egg); RDQ-4 issue requirements are lineage links but not coverage.
  Captured DD-7 (gate step name), DD-8 (validate/gate split), DD-9 (issue-lineage-not-coverage),
  and added the two founder scope expansions as DD-10 (requirement VERSIONING) and DD-11
  (PRE-IMPLEMENTATION TRACE GATE). Replaced OQ1–OQ4 with five new OQs (OQ5 version
  format/home/bump-trigger; OQ6 pin syntax + mandatory-vs-optional; OQ7 stale-pin semantics;
  OQ8 pre-impl gate trigger + trace target; OQ9 migration). `version` 0.1.0 → 0.2.0. Maturity
  `exploring`.
- **0.3.0 (2026-07-14, exploring)** — RESOLVED OQ5–OQ9 (RDQ-5…RDQ-9) and added the full Spec
  Seeds decomposition. RDQ-5 semver per REQ; RDQ-6 mandatory full pin `bundle:REQ-NNN@1.2.0`;
  RDQ-7 lifecycle-keyed, semver-gated stale-pin model → DD-12; RDQ-8 enforce chain-of-custody at
  EVERY promotion gate → broadened DD-11 from a single pre-impl chokepoint to every transition;
  RDQ-9 explicit `1.0.0` backfill sweep folded into the RDQ-2 reconciliation. Refined DD-4/DD-6/
  DD-10. Added a Spec Seeds section (Seed 1 versioning schema + resolution → Seed 2 reconciliation
  + backfill → Seed 3 gate step + coverage + stale-pin). One genuinely new downstream fork
  surfaced and was recorded OPEN as OQ10 (does a version pin resolve against recorded history or
  REQ-identity-only). `version` 0.2.0 → 0.3.0. Maturity `exploring`.
- **0.4.0 (2026-07-14, exploring)** — RESOLVED the last open question, OQ10 → RDQ-10 (option b:
  recorded version history — a per-REQ version LOG). The bundle's `requirements[]` retains EVERY
  version of each REQ with that version's text; resolution requires a pin to match a real
  historical log entry, so a fabricated version dies at validate. Threaded the log through DD-10,
  DD-4, DD-12, Current Thinking, solution.approach, and all three Spec Seeds. The Open Questions
  section became explicitly EMPTY — all ten (OQ1–OQ10) resolved. `version` 0.3.0 → 0.4.0.
  Maturity `exploring`.
- **0.5.0 (2026-07-14, defined)** — PROMOTED `exploring` → `defined` (founder-initiated). No new
  scope invented: the resolved decisions (RDQ-1…RDQ-10 / DD-1…DD-12) were lifted FAITHFULLY into
  a 15-entry formal `requirements[]` array (REQ-001…REQ-015), each REQ atomic, testable, and
  naming the DD/RDQ it descends from. Each REQ dogfoods the bundle's own versioning convention
  (RDQ-5/DD-10) by carrying `version: 1.0.0`, the initial log entry the RDQ-9 sweep would stamp.
  Added the required **Draft Requirements** section (grouped traceability summary). Updated
  `solution.approach` and the Open Questions note; annotated each DD with the REQ it produced.
  `version` 0.4.0 → 0.5.0, `bundle.updated` → 2026-07-14.
- **0.6.0 (2026-07-14, defined)** — TWO founder corrections to the requirements, held at
  `defined`. (1) **REQ-013 posture — silent was never the intent.** Rewrote the posture as THREE
  tiers: defects (dangling/unpinned/fabricated-version refs) BLOCK; broken promises (delivered
  coverage failure, or any state where a downstream's upstream chain doesn't verify) BLOCK;
  in-flight uncovered REQs are an ADVISORY WARNING on the non-policied advisory surface (like
  `status_drift`'s advisory split — visible, never blocking, never policy-upgradable). Updated
  DD-6, Current Thinking's load-bearing frame + the "must NOT flag" corpus examples, the Draft
  Requirements summary, and Notes so nothing still says "not a violation"/"no warn"/"invisible" —
  in-flight gaps are visible-but-quiet, never silent. (2) **REQ-012 restated STATE-WISE with the
  dispatch seam named.** Replaced the four-transitions/events framing with a corpus-STATE
  invariant: the `requirement_traceability` step BLOCKS any state where a downstream artifact's
  upstream chain doesn't verify (spec past its supported depth, plan for an unverified spec,
  `delivered` without full coverage). Core enforces STATES not events — because promotions land
  as edits and all gates run at every step, every transition is judged by construction — and
  dispatch-time refusal (blocking `/implement` before it starts) is explicitly a hook/runtime
  SEAM consuming core's verdict (gate-on-implement hook today, opencode runtime later), not core
  intercepting events. Updated DD-11, RDQ-8, Current Thinking's DD-11 bullet, solution.approach,
  Seed 3, and Notes to match. `version` 0.5.0 → 0.6.0; `bundle.updated` → 2026-07-14. The founder
  reviews the requirements[] list line by line.
- **0.7.0 (2026-07-14, ready)** — PROMOTED `defined` → `ready` (founder-initiated). No new
  scope: the `ready` maturity gate's two additional frontmatter requirements were satisfied
  from content already in the bundle. Added `problem.success_criteria` (five testable success
  conditions, each restating a committed REQ at a success level — resolution green-or-block,
  no uncovered `delivered` bundle, the three-tier posture holding, per-version
  interrogability + stale-pin behaviour, and a green corpus after the reconciliation/backfill
  sweep). Filled `solution.assumptions` (previously empty) with four assumptions faithfully
  derived from anchors already stated — status_drift infra reuse, the 0.5.0-confirmed schema
  tolerance of the version/log fields, dispatch-refusal-as-a-seam, and full-corpus reachability
  at both validate and gate time. `version` 0.6.0 → 0.7.0; `bundle.updated` → 2026-07-14. All
  ten OQs remain resolved; the requirements[] array (REQ-001…REQ-015) is unchanged.
- **0.8.0 (2026-07-14, ready)** — AMENDED RDQ-2's legacy-reconciliation decision (held at
  `ready`). The cross-spec consistency pass exposed that the original "backfill
  `agent-definitions`' requirements[]" choice would mint a `delivered` bundle with 18 REQs and
  ZERO implemented-spec coverage once its draft citers retire — this bundle's own gate step
  (DD-1/DD-2) would block it, and keeping it delivered would demand 18 retroactive-coverage
  claims with thin anchors (fake rigor). **Founder decision (supersedes the backfill half):** the
  whole legacy cluster is DEPRECATED instead — `agent-definitions.bundle.md` goes terminal
  (`deprecated`) together with SPEC-002/003/004, no backfill; a terminal bundle sits outside the
  delivered-gate by the terminal-exclusion the gate already honors, so this is honest recording,
  not grandfathering (zero-grandfathering posture intact). The BUNDLE-011 half (re-state on live
  specs) STANDS, now realized via a retroactive `implemented` SPEC-053 for REQ-007/010 (SPEC-039's
  work landed via ISSUE-018; issues are lineage not coverage, so an implemented retroactive spec
  archives it). Amended RDQ-2 (marked + dated + why), and every section repeating the backfill
  plan: `solution.approach`, REQ-015 (text rewritten to deprecate-not-backfill; its `version`
  bumped 1.0.0 → 1.1.0 per the bundle's OWN convention — a meaning change is a minor bump under
  DD-12; a full version-LOG representation awaits Seed 1's schema, so the bump is recorded at the
  `version:` field the current frontmatter supports), `problem.success_criteria`, the Draft
  Requirements summary, Seed 2 (+ Seed 1's cross-reference), and two Notes bullets (added a
  terminal-exclusion pointer). The factual "agent-definitions is delivered with no requirements[]"
  mentions (problem.summary, Current Thinking's holes) are left intact — they describe the drift,
  not the fix. `version` 0.7.0 → 0.8.0; `bundle.updated` → 2026-07-14. Maturity STAYS `ready`.
- **0.9.0 (2026-07-14, ready)** — Gave REQ-015 an explicit `versions:` log now that Seed 1's
  schema (SPEC-050) exists: entry `1.0.0` carries the ORIGINAL pre-amendment backfill wording,
  entry `1.1.0` carries the current deprecate-not-backfill text, so its 0.8.0 meaning-change is
  now recoverable from the artifact itself rather than only the Version History prose. Top-level
  `version:` stays `1.1.0` and its `text` equals the newest entry (the two compared scalars use
  strip-chomp `>-` so they normalize identically regardless of document position). No other REQ
  touched. `version` 0.8.0 → 0.9.0; `bundle.updated` → 2026-07-14. Maturity STAYS `ready`.
