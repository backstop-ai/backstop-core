---
title: "Runbooks"
number: BUNDLE-019
created: "2026-07-20"
schema_version: bundle/v2

bundle:
  name: runbooks
  version: "0.1.0"
  created: "2026-07-20"
  category: feature

status:
  maturity: exploring

problem:
  summary: >
    A RUNBOOK is a declared, receipt-verified operational procedure — "a checklist with
    receipts": an ordered set of steps, each with an execution mode and a paired verify-check,
    whose completion is PROVEN by reading world-state rather than by a human ticking boxes.
    BUNDLE-015 decided the executable-runbook mechanism inside the recipe / standup work
    (DD-23..DD-26) but scoped it to the STANDUP genre — one-time project provisioning. The
    2026-07-20 founder generalization: runbooks are a GENERAL capability of which standup is
    only the FIRST and RAREST genre. The frequent genres are OPS (credential rotation, cert
    renewal, incident diagnostics, backup verification, access reviews) and MAINTENANCE
    (scheduled sweeps). This bundle is CARVED OUT of BUNDLE-015 — exactly as BUNDLE-015 was
    carved from BUNDLE-003 (init) and BUNDLE-017 (provenance ledger) was carved from
    BUNDLE-015 — so that it OWNS the general runbook surface (the step executor, the
    probe/receipt engine, the bootstrap ladder, the diagnose-first UX), while BUNDLE-015
    returns to the file-op recipe capability `backstop init` is blocked on. Strategic tie: an
    ops runbook is TRIBAL KNOWLEDGE — the wiki-in-a-senior's-head — becoming a versioned,
    distributable, receipt-verified PACK ARTIFACT.

  user_story: >
    As an operator (or an agency running standups and ops for many clients), I want to declare
    an operational procedure once — its ordered steps, each step's execution mode, and each
    step's verify-check — so that anyone or any agent running it gets a diagnose-first
    walkthrough that renders ONLY the unmet steps, runs what is automatable, gates
    real-money / real-secret steps behind explicit consent that never dissolves, and PROVES
    completeness via receipts rather than a human ticking boxes. Concretely, the canonical
    half-done procedure — credential ROTATION where the new key is added but the old key is
    never revoked, silently — becomes impossible to leave half-done: a rotation runbook whose
    receipts require BOTH new-key-probes-green AND old-key-probes-correctly-dead makes rotation
    COMPLETENESS machine-verifiable.

solution:
  approach: >
    A runbook is a declared, ordered set of STEPS. Each step carries an execution MODE
    (auto / assisted / consent / decision) whose EFFECTIVE value is COMPUTED against current
    standing state via precondition RECEIPTS (the bootstrap ladder), and each step is paired
    with verify-checks that DERIVE its state from the world rather than storing it. The
    capability is a new USE of existing substrate, not new substrate: receipts reuse the
    BUNDLE-015 DD-16 read-and-compare selectors + the existing check-command engines (NO new
    verification primitive); steps ride IN recipes as (per BUNDLE-015's residual lean) a fifth
    op family declared in the 015 manifest; the thin adoption record (015 DD-20) tracks
    adoption. The division of labor with BUNDLE-015 is explicit: BUNDLE-019 supplies the step
    EXECUTOR + the probe/receipt engine + the runbook SURFACE (diagnose-first rendering, the
    automation escalation ladder API > CLI > deep-link > click-instruction, the `--explain`
    action log); BUNDLE-015 supplies the op schema + apply sequencing. Core NEVER performs a
    platform operation itself — a human or agent executes with their own tools; core RENDERS,
    sequences, and VERIFIES. Consent gates (real money / real secrets) never dissolve. The
    capability spans three GENRES: STANDUP / provisioning (the first, rarest — once per project;
    the bclabs-portal go-live capture is the standing fixture), OPS (the frequent genre —
    rotation, cert renewal, incident diagnostics, backup verification, access reviews), and
    MAINTENANCE (scheduled sweeps). Four inherited DDs (015 DD-23..DD-26) are recorded here as
    DECIDED with lineage; the generalization then surfaces genuinely OPEN mechanism questions
    the standup-only frame never had to answer — time-based / aging receipts and recurrence,
    the triggering model (user-invoked vs scheduled vs event-driven), where recurring-run
    history lives (the BUNDLE-017 rich ledger vs a minimal last-run record vs derived-only),
    whether genre is first-class metadata, and the exact step-declaration seam with 015 — all
    left to founder-driven resolution. Maturity stays `exploring`; the founder drives OQ
    resolution and promotion.
---

# Runbooks

## Current Thinking

### Provenance: carved out of BUNDLE-015, generalized beyond standup (2026-07-20)

This bundle is a **carve-out** of **BUNDLE-015 (pack-scaffolding-recipes)**, exactly as
BUNDLE-015 was itself carved from **BUNDLE-003 (init)** and **BUNDLE-017 (provenance ledger)**
was carved from BUNDLE-015. The trigger was the 2026-07-20 founder generalization: the
executable-runbook mechanism BUNDLE-015 recorded (DD-23..DD-26) was scoped to the STANDUP
genre, but standup is only ONE use for runbooks — "and often less frequent than ops runbooks."
Runbooks are a GENERAL capability. BUNDLE-015 v0.8.0 slims: DD-23..DD-26 are MIGRATED here with
lineage stubs left behind, and 015 returns to the file-op recipe capability init is blocked on.

### The genres — standup is the first and rarest, ops is the frequent one

Runbooks are declared, receipt-verified operational procedures. Three genres, ordered by the
founder from rarest to most frequent:

- **(a) STANDUP / provisioning** — run ONCE per project (spin up Supabase + Vercel + GitHub +
  the composition layer). The rarest genre. The standing fixture is the bclabs-portal go-live
  capture — **"Portal Go-Live: Physical Scaffolding & Admin/Setup Reference"** at
  `~/.claude/uploads/3826af56-28a4-484f-975c-492a561591c4/eaeae70d-golivescaffolding.md`,
  cross-checked against the portal repo (deployment-substrate gap; aspirational `POSTHOG_*`;
  dead `PORTAL_OIDC_SIGNING_KEY`; live `PORTAL_REPO_PATH`). This capture is the origin forcing
  function for the inherited DDs; it lives in BUNDLE-015's Current Thinking and References.

- **(b) OPS — the frequent genre.** Credential rotation, cert renewal, incident diagnostics,
  backup verification, access reviews. The KILLER example is rotation: rotation is the canonical
  HALF-DONE procedure — a new key is added and the old key is never revoked, silently. A
  rotation runbook with receipts that require BOTH **new-key-probes-green** AND
  **old-key-probes-correctly-dead** makes rotation COMPLETENESS machine-verifiable. This is the
  genre that recurs, and the one whose recurrence surfaces the open questions below.

- **(c) MAINTENANCE — scheduled sweeps.** BUNDLE-015 DD-25's own click-instruction promotion
  sweep ("did this provider grow an API/CLI yet?") is ITSELF a runbook of this genre — the
  capability is self-describing. Ties to the future maintenance-agents thread (scheduled
  autonomous agents over core).

### Strategic tie — tribal knowledge becomes a verified artifact

An ops runbook is the wiki-in-a-senior's-head made durable: the procedure only that one person
knows how to do correctly becomes a VERSIONED, DISTRIBUTABLE, RECEIPT-VERIFIED pack artifact.
This is the "integrate-don't-build / the bundle is the product" thesis applied to operational
knowledge — and it is the tribal-knowledge-substrate direction landing on a concrete surface.

### The thin-executor invariant still holds (inherited)

All platform / provider knowledge lives in runbook DATA, never in core. Core renders declared
steps, sequences them, and runs declared receipts; it NEVER performs a platform operation. A
human or agent executes with their own tools. This is BUNDLE-015 DD-3 / DD-23 carried forward —
the invariant every decision here is measured against, and what `backstop/self` guards.

### What is DECIDED vs OPEN here

DECIDED (migrated from BUNDLE-015's founder sessions, recorded below as DD-1..DD-5 with lineage
to 015 DD-23/24/25/26; do NOT reopen): the executable runbook of step-ops with modes and
receipts; the bootstrap ladder of preconditions and computed effective mode; the diagnose-first
UX + automation escalation ladder; the first-class composition step; and the substrate
dependency on BUNDLE-015. OPEN (surfaced by the generalization from standup to the recurring
genres — genuine forks, NOT pre-resolved; the founder drives): time-based / aging receipts and
recurrence (OQ-1), the triggering model (OQ-2), recurring-run history's home (OQ-3), the genre
taxonomy (OQ-4), and the exact step-declaration seam with 015 (OQ-5). Maturity stays
`exploring`; promotion is the founder's next step.

## Draft Design Decisions

DD-1..DD-4 are INHERITED from BUNDLE-015's founder sessions and recorded here as DECIDED with
lineage — this bundle now OWNS the runbook capability, so they live here; the record that they
were decided in the 015 sessions is preserved via the citations. DD-5 is the substrate-dependency
contract with BUNDLE-015. None are reopened; the open mechanism details the generalization
surfaces are OQ-1..OQ-5, not reopenings of these.

- **DD-1: The EXECUTABLE RUNBOOK — step ops with modes and receipts.** (Lineage: BUNDLE-015
  DD-23, founder-decided 2026-07-20.) A runbook is an ordered set of declared platform-state
  STEPS with paired verify-checks — "a checklist with receipts." Each step carries an EXECUTION
  MODE: `auto` (command declared, runnable) → `assisted` (machine does everything but the click;
  DD-3) → `consent` (runnable only on explicit-go: real money / real secrets — the consent gate
  NEVER dissolves regardless of automation; answering ≠ go, as runbook data) → `decision` (human
  choice collapsed to one prompt + a sane default). Receipts span declared-read-and-compare
  (DD-16-shaped selectors) to run-a-check-command (the existing engine model) — NO new
  verification primitive. Core RENDERS / sequences declared steps and runs receipts; it NEVER
  performs a platform operation itself (thin executor). Step STATE is DERIVED from receipts at
  evaluation time, never stored (consistent with the thin adoption record; rich history is the
  OQ-3 question). BUNDLE-019 owns the step EXECUTOR and probe/receipt engine.

- **DD-2: The BOOTSTRAP LADDER — preconditions and computed effective mode.** (Lineage:
  BUNDLE-015 DD-24, founder-decided 2026-07-20.) Scriptability is a FUNCTION OF STANDING STATE,
  not a static step property. Three tiers: **Tier 0 — ACCOUNT EXISTENCE** (create account / org /
  billing / ToS — NEVER scriptable, provider-gated via CAPTCHA / payment / legal; once per
  provider); **Tier 1 — AUTH BOOTSTRAP** (`vercel login` / `gh auth login` / supabase token /
  provider key — human DOES, machine VERIFIES via receipts; once per machine or credential
  expiry); **Tier 2 — PROVISIONING** (the scriptable majority, reachable only when T0/T1 receipts
  are green). Steps declare PRECONDITIONS (receipts); the EFFECTIVE mode is COMPUTED against
  current state — the same step is `auto` for an established operator and blocked-on-human for a
  greenfield user. Honest value framing: the runbook does not make the procedure scriptable — it
  makes the human core EXPLICIT, MINIMAL, VERIFIABLE, and ONE-TIME; the one-time-ness IS the
  agency economics (standing T0/T1 state amortizes across every subsequent client). Cross-bundle
  tie: Tier-1 outputs (tokens) land in **Stash (BUNDLE-018)**; future precondition receipts read
  presence-in-stash, later `use`-brokered so tokens never surface.

- **DD-3: DIAGNOSE-FIRST UX + the AUTOMATION ESCALATION LADDER.** (Lineage: BUNDLE-015 DD-25,
  founder-decided 2026-07-20.) The runbook's entry point is a SILENT PROBE PASS (all receipts);
  ONLY unmet steps render, each with its fix attached ("↳ run: X" / "↳ needs you: <exact URL +
  click-level instructions>"). Green is invisible; a fully-set-up machine sees NO runbook at all.
  **PROBE LAW:** receipts must be read-only / idempotent (never provision-to-test), cheap (local
  before network; runs every invocation), and fail-soft (provider outage = "unknown, can't
  verify," never a false red). **ASSISTED-rung mechanics** (the `gh auth` device-flow as the
  canonical model): deep-link to the exact page (pre-filled params where supported), stage
  paste-values (from params / Stash; clipboard auto-clear for secrets), click-level instructions
  for the remainder, and POLL THE RECEIPT so the step completes ITSELF — never "press enter when
  done." **AUTOMATION ESCALATION LADDER** (mirrors the engine escalation ladder): API > CLI >
  deep-link URL > click-instructions — always take the highest available rung; every
  click-instruction is a STANDING PROMOTION CANDIDATE (the maintenance-genre sweep, DD-25's own
  example). **CLICK-INSTRUCTIONS ARE DECLARED BEST-EFFORT** (founder-explicit): the most rot-prone
  data in the system; staleness is a WARN-AND-REV (variants version the instructions), NEVER a
  red — "instructions are vibes, receipts are law": a rotted click-path degrades UX but cannot
  produce false-green because the receipt polls the OUTCOME, not the path. Guardrail: invisible ≠
  uninspectable — an `--explain` / action-log face records everything probed and executed.

- **DD-4: The COMPOSITION / FRAMEWORK step is first-class (in the standup genre).** (Lineage:
  BUNDLE-015 DD-26, founder-decided 2026-07-20.) Provisioned infra + implemented components ≠ a
  running app — the portal proves it (a fully-provisioned Supabase still 500s while the deployment
  substrate and composition root are absent). A standup runbook MUST include the code-wiring step
  as first-class: **Mode B** (SDLC-mediated, BUNDLE-015 DD-11 — the recipe supplies the adapter
  template WHAT + an acceptance-check GUARANTEE, e.g. "app boots; ingest route returns non-500";
  the plan supplies the WHERE), with the acceptance check as the step's RECEIPT. Infra
  provisioning without this step is a VACUOUS standup. Scoped to the standup genre specifically;
  the ops and maintenance genres do not generally carry a composition step.

- **DD-5: This capability DEPENDS ON the BUNDLE-015 substrate; the seam is explicit.** (New here;
  formalizes the 015 ↔ 019 division of labor.) Runbook steps ride IN recipes — per BUNDLE-015's
  residual lean, likely a FIFTH OP FAMILY (`step`, alongside create / merge / transform / insert)
  declared in the 015 recipe manifest and interleaved with file ops in one ordered apply (real
  standup writes `vercel.json` → provisions → sets env → deploys). Receipts reuse the 015 DD-16
  read-and-compare SELECTORS + the existing check-command ENGINES (no new verification primitive).
  Adoption is tracked by the 015 DD-20 thin ADOPTION RECORD. Division of labor: **BUNDLE-019
  supplies the step EXECUTOR / probe engine + the runbook surface; BUNDLE-015 supplies the op
  SCHEMA + apply sequencing.** The exact finalization of the step shape
  `{description, command?, mode, preconditions, verify}` and WHERE it is declared vs executed is
  OQ-5.

## Open Questions

The generalization from the standup-only frame to the recurring OPS and MAINTENANCE genres
surfaces these — they are GENUINE forks the standup frame never had to answer. NOT pre-resolved;
the founder drives resolution and triggers promotion. Numbered sequentially; kept with their
reasoning.

- **OQ-1 — TIME-BASED RECEIPTS / RECURRENCE.** Standup receipts are done-forever (the bucket
  exists or it doesn't). OPS receipts AGE: "rotated 91 days ago" is stale; a cert's expiry is
  literally a time-receipt; a backup-verification receipt is only meaningful within a freshness
  window. Do receipts carry VALIDITY WINDOWS? Options: (a) receipts stay boolean and time is
  modeled OUTSIDE the receipt (the trigger/schedule owns freshness — couples to OQ-2); (b)
  receipts gain an optional validity window / max-age and evaluate STALE as a distinct state; (c)
  a dedicated time-receipt selector kind (read a timestamp / expiry, compare to now + a declared
  window — DD-16-shaped, generic compare, no baked knowledge). And how does staleness RENDER —
  WARN ("rotate soon") vs RED ("overdue")? The DD-3 probe-law (read-only / cheap / fail-soft) and
  the loud-≠-blocking principle both bear on the warn-vs-red line. No lean recorded — founder's.

- **OQ-2 — TRIGGERING MODEL.** What INVOKES a runbook, and what does the declaration carry about
  it? The genres pull differently: STANDUP is USER-INVOKED (run it when you stand up a project);
  MAINTENANCE is SCHEDULED (ties to the future maintenance-agents thread); an incident-diagnostics
  OPS runbook is EVENT-DRIVEN (fired by an alert). Does the runbook DECLARATION carry a trigger
  descriptor (manual / cron-like schedule / event hook), and what actually SCHEDULES it — core?
  an external cron? the runtime (the maintenance-agents thread)? Thin-executor tension: core
  running a scheduler may itself be a step too far — the founder may want the schedule DECLARED as
  data and fired by something outside core (consistent with "core renders/verifies, never
  executes"). Couples to OQ-1 (a schedule is one way freshness is enforced) and OQ-3 (a scheduled
  run needs a last-run record). No lean — founder's.

- **OQ-3 — RECURRING-RUN HISTORY.** A recurring runbook needs SOME durable record of "last
  rotated at / last verified at" — but the thin adoption record (015 DD-20) is per-ADOPTION, not
  per-RUN. Where does recurrence history live? Options: (a) the **BUNDLE-017 rich provenance
  ledger** — it already owns per-application forensic history and fleet dashboards; recurring-run
  history is arguably the same reader (and a fleet view of "which clients are overdue for
  rotation" is exactly its hosted-capture surface); (b) a **minimal last-run record** owned here
  (the runbook analog of the thin adoption record — just `{runbook ref, last-run, last-outcome}`);
  (c) **derived-only** wherever possible — the receipt reads the WORLD ("the key's created-at is
  91 days ago") so no stored run-history is needed, consistent with DD-1's derive-don't-store.
  The lean-shaped tension: DD-1 says state is derived not stored, which argues (c) where the world
  carries the timestamp — but not every procedure leaves a world-legible timestamp, which is where
  (a) or (b) earn their keep. Cite the BUNDLE-017 tie explicitly. No lean — founder's.

- **OQ-4 — GENRE TAXONOMY.** Is GENRE (standup / ops / maintenance) FIRST-CLASS metadata on a
  runbook — driving rendering defaults, trigger defaults (OQ-2), and receipt-freshness defaults
  (OQ-1) — or is it just CONVENTION / free-form tags with no mechanism attached? First-class
  buys defaults and validation (a maintenance runbook without a schedule is suspect; a standup
  runbook carries the DD-4 composition step) at the cost of baking a fixed taxonomy that a fourth
  genre later strains. Convention/tags stay flexible but give the mechanism nothing to key on.
  Restraint precedent (015 DD-22's one-fixture rule): don't formalize a taxonomy three genres
  wide off one real fixture. No lean — founder's.

- **OQ-5 — STEP DECLARATION SEAM (confirm the split with BUNDLE-015).** Confirm the DD-5 division
  concretely: the step OP is DECLARED in the recipe manifest (BUNDLE-015's schema — 015 owns the
  op schema + apply sequencing) and EXECUTED / rendered by the runbook engine (here). Where does
  the step SHAPE — `{description, command?, mode, preconditions, verify}` — get FINALIZED: in the
  015 manifest-schema spec (as the fifth op family) or in a 019 runbook-engine spec that the 015
  schema references? BUNDLE-015 recorded this as a spec-time residual (its "fifth op family"
  lean); it is a genuine seam decision now that the capability is a separate bundle. Couples to
  DD-5. No lean — founder's, and a coordination point with the 015 spec author.

Maturity stays `exploring`. No OQs pre-resolved; no requirements recorded yet — those await
founder-driven OQ resolution.

## Spec Seeds

Provisional — this bundle is `exploring` and the decomposition firms up once OQ-1..OQ-5 resolve.
The first seed MIGRATES the runbook/step-op seed out of BUNDLE-015; the others follow the genres.

- **Runbook / step-op executor + probe/receipt engine (core)** — MIGRATED from BUNDLE-015's
  "runbook / step-op + probe/receipt engine" seed. The executable-runbook mechanism (DD-1):
  render / sequence declared STEPS (`{description, command?, mode, preconditions, verify}`), each
  carrying an execution MODE (auto / assisted / consent / decision) whose EFFECTIVE value is
  COMPUTED against current standing state via precondition RECEIPTS (the bootstrap ladder, DD-2).
  The PROBE / RECEIPT engine runs read-only / idempotent / cheap / fail-soft checks (probe law,
  DD-3), DERIVES step state from receipts (never stored, DD-1), and drives the DIAGNOSE-FIRST
  surface (only unmet steps render with their fix; automation escalation API > CLI > deep-link >
  click-instruction; `--explain` action-log; DD-3). Core NEVER performs a platform op itself —
  receipts reuse the BUNDLE-015 DD-16 read-and-compare selectors + the existing check-command
  engine, so NO new verification primitive and ZERO baked provider knowledge. The CONSENT gate
  (real-money / real-secrets) is runbook DATA, never dissolved (DD-1). Click-instruction staleness
  is WARN-AND-REV, never red (DD-3). Depends on the BUNDLE-015 apply mechanism for step-op
  sequencing (DD-5 / OQ-5). This seed absorbs whatever OQ-1 (time-based receipts) resolves into.

- **Standup-genre runbook fragments (backstop-packs)** — the RUNBOOK halves of the provider
  standup packs (`supabase` / `vercel` / `nextjs` / the github-actions CI pack). BUNDLE-015 owns
  each pack's FILE recipes (`vercel.json`, shells, workflows); this seed owns their runbook
  fragments — the provision / env / cron / deploy STEPS, the bootstrap-ladder preconditions, the
  §5-gotcha receipts (Supavisor transaction-pooler + `prepare=false`, `storage.objects` ownership,
  OIDC audience parity, prod/preview split — near-verbatim verify-checks from the capture), and
  the DD-4 composition step. Sourced from the bclabs-portal capture (the standing standup fixture).
  Coordinates with BUNDLE-015's "provider standup packs" seed across the file/runbook seam.

- **Ops-genre exemplar — a rotation runbook (backstop-packs)** — a SECOND fixture in the OPS
  genre: credential rotation whose receipts require BOTH new-key-probes-green AND
  old-key-probes-correctly-dead, making rotation completeness machine-verifiable (the killer
  example). **NOTE: this is a FUTURE CAPTURE-FIRST candidate, not scoped here** — following the
  capture-first precedent (a real rotation is captured before the exemplar is authored) and the
  015 DD-22 one-fixture restraint (no ops-genre packs until a real ops capture demands them). It
  is the concrete forcing function that will exercise OQ-1 (aging receipts), OQ-2 (triggering),
  and OQ-3 (run history) — recorded so the genre isn't lost, not committed as scope.

## Notes / Ideas

- **Rotation is the headline — completeness, not just execution.** The strength of the ops genre
  is that a receipt-verified runbook makes an OPERATION'S COMPLETENESS provable, not just its
  execution. Rotation is the canonical case: today "I rotated the key" means the new key works;
  the silent failure is the old key still living. A runbook whose receipts assert BOTH halves
  turns a half-done-by-default operation into one that CANNOT report green until it is actually
  complete. This is the same "loud ≠ blocking, block broken promises" enforcement thesis applied
  to operational procedures.
- **Tribal knowledge → verified artifact.** The ops/maintenance genres are the concrete landing
  of the tribal-knowledge-substrate direction: the procedure only one senior knows how to do
  correctly becomes a versioned, distributable, receipt-verified pack. Distribute the runbook and
  every operator (or agent) runs the senior's procedure with the senior's completeness checks.
- **The capability is self-describing.** DD-25's own click-instruction promotion sweep is itself a
  MAINTENANCE-genre runbook. The mechanism can express its own upkeep — a small proof the
  abstraction is the right shape.
- **Recurrence is where the fleet moat compounds (OQ-3 tie).** If recurring-run history lands in
  the BUNDLE-017 rich ledger, "which clients are overdue for rotation / cert renewal across the
  fleet" becomes a dashboard — the sharpest hosted-capture surface the ops genre offers, the same
  agency-model logic that makes the migration dashboard valuable. This is why OQ-3 is not merely a
  storage question.

## Version History

- 0.1.0 (2026-07-20): Initial bundle at `exploring`, CARVED OUT of BUNDLE-015
  (pack-scaffolding-recipes) — exactly as BUNDLE-015 was carved from BUNDLE-003 (init) and
  BUNDLE-017 (provenance ledger) was carved from BUNDLE-015. Founder generalization (2026-07-20):
  runbooks are a GENERAL declared, receipt-verified operational-procedure capability of which
  STANDUP is only the first and rarest genre; OPS (rotation / cert renewal / incident diagnostics /
  backup verification / access reviews) is the frequent genre and MAINTENANCE (scheduled sweeps)
  the third. Recorded the three genres and the tribal-knowledge → verified-artifact strategic tie
  in Current Thinking; migrated the four inherited runbook DDs from BUNDLE-015 as DECIDED with
  lineage — DD-1 (executable runbook: step-ops with modes auto/assisted/consent/decision +
  receipts, state derived not stored ← 015 DD-23), DD-2 (bootstrap ladder: Tier-0/1/2 +
  computed effective mode + Stash tie ← 015 DD-24), DD-3 (diagnose-first UX + automation
  escalation ladder + probe law + receipts-are-law ← 015 DD-25), DD-4 (first-class
  composition/framework step in the standup genre ← 015 DD-26) — plus DD-5 (the BUNDLE-015
  substrate-dependency seam: steps as a fifth op family, receipts reuse DD-16 selectors +
  check-command engines, adoption via DD-20; 019 supplies the executor/surface, 015 the op schema +
  sequencing). Recorded FIVE genuinely open questions the generalization surfaces — OQ-1
  (time-based / aging receipts + recurrence), OQ-2 (triggering model: user-invoked / scheduled /
  event-driven), OQ-3 (recurring-run history's home: BUNDLE-017 ledger / minimal last-run record /
  derived-only), OQ-4 (genre taxonomy: first-class vs convention), OQ-5 (the step-declaration seam
  with BUNDLE-015) — none pre-resolved. Three provisional spec seeds (the migrated runbook/step-op
  executor + probe/receipt engine; the standup-genre runbook fragments of the provider packs; a
  future capture-first ops-genre rotation exemplar, noted not scoped). References BUNDLE-015
  (parent / substrate), BUNDLE-017 (ledger tie for OQ-3), BUNDLE-018 (Stash custody), the portal
  capture (first fixture), and the maintenance-agents future thread. No OQs pre-resolved; no
  requirements or maturity self-promotion — the founder drives resolution and promotion.

## References

- **BUNDLE-015 (Pack Scaffolding Recipes)** — the PARENT and the substrate dependency. Owns the
  recipe apply mechanism + op schema + apply sequencing this capability rides on, the DD-16
  read-and-compare selectors and check-command engines the receipts reuse, and the DD-20 thin
  adoption record that tracks adoption. DD-23..DD-26 (the runbook, bootstrap ladder, diagnose-first
  UX, and composition step) originated in its 2026-07-20 standup session and are MIGRATED here as
  DD-1..DD-4; 015 v0.8.0 leaves lineage stubs pointing to this bundle. The step-op is (per 015's
  residual lean) a fifth op family in the 015 manifest — the OQ-5 seam.
- **BUNDLE-017 (Recipe Provenance Ledger)** — the ledger tie for OQ-3 (recurring-run history):
  the rich provenance ledger already owns per-application forensic history and fleet dashboards, so
  "last-rotated-at" history and a fleet "who's overdue" view are candidate readers of it rather
  than a new store. Itself a prior carve-out of BUNDLE-015 (the carve precedent this bundle
  follows).
- **BUNDLE-018 (Stash — local resource / credential store)** — the credential-custody tie for the
  bootstrap ladder (DD-2): Tier-1 auth outputs (provider tokens) land in Stash; future precondition
  receipts read presence-in-stash, later `use`-brokered so tokens never surface.
- **The standup capture — bclabs-portal go-live (the first fixture, 2026-07-20)** —
  `~/.claude/uploads/3826af56-28a4-484f-975c-492a561591c4/eaeae70d-golivescaffolding.md`. The
  standing STANDUP-genre fixture: a full go-live capture of bclabs-portal on Supabase + Vercel +
  GitHub + PostHog, cross-checked against the portal repo. The forcing function for the inherited
  DDs; full detail (empirics, cross-check corrections) lives in BUNDLE-015's Current Thinking and
  References. Slices land in each born pack's `fixtures/captured/`.
- **Maintenance-agents (future thread)** — scheduled autonomous agents over core. The MAINTENANCE
  genre (DD-25's click-instruction promotion sweep is itself one) and the OQ-2 scheduling question
  both point at this thread as the plausible SCHEDULER outside core (consistent with the
  thin-executor invariant: core renders / verifies, something else fires the schedule).
- **backstop/self pack** — enforces the zero-baked-provider boundary the step executor and receipt
  engine must respect (the thin-executor invariant, DD-1).
