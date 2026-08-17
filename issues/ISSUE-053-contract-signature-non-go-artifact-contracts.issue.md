---
title: "Contract Signature Non Go Artifact Contracts"
schema_version: issue/v1

issue:
  id: ISSUE-053
  title: "Contract Signature Non Go Artifact Contracts"
  type: technical-debt
  status: open
  created: "2026-07-13"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# ISSUE-053: Contract Signature Non Go Artifact Contracts

## Problem

`contract_signature`'s compiler (`packs/contracts/scripts/compile-signature.sh`,
the `ast-grep-contracts` engine) is Go-oriented: it shells `ast-grep --lang
go` (plus a grep fallback) to structurally match a declared signature
against source. A slice of the 143 whole-repo `contract_signature`
violations surfaced under `--all` (see ISSUE-051 for the full split) are
declared on artifacts that are not Go source at all — no signature the
current compiler can emit will ever compile to a match against them,
regardless of whether the code/content behind them is correct.

### Verified instances

Grepping every `kind: constant` / `kind: variable` contract with a
non-Go-shaped (prose) signature turns up, among others:

| Spec | Status | File | Signature |
|---|---|---|---|
| SPEC-003 | `draft` | `.claude/settings.json` | `"hooks.PreToolUse[] entries for Edit and Write tools"` |
| SPEC-003 | `draft` | `.claude/hooks/backstop-agent-guard.sh` | `"backstop-agent-guard.sh (reads JSON from stdin, ...)"` |
| SPEC-004 | `draft` | `artifacts/spec/v1/schema.json` | `"JSON schema definition"` |
| SPEC-004 | `draft` | `.claude/agents/spec-author.md` | `"Agent definition markdown"` |
| SPEC-004 | `draft` | `.claude/agents/impl-reviewer.md` | `"Agent definition markdown"` |
| SPEC-004 | `draft` | `.claude/hooks/backstop-standards-context.sh` | `"Shell script"` |
| SPEC-004 | `draft` | `.claude/settings.json` | `"JSON configuration"` |
| SPEC-033 | `draft` | `specs/SPEC-033-...spec.md` (self) | `"contract-lock: BUNDLE-010 producer → BUNDLE-009 consumer hand-off artifact set"` |

All eight target files/paths above were confirmed to exist on disk. None of
these signatures is a compilable Go fragment — they are prose descriptions
of a JSON config shape, a Markdown agent definition, a shell script's
stdin/stdout contract, or (SPEC-033) a spec declaring itself as its own
"provides." ISSUE-037's audit of `kind: constant` contracts already flagged
the SPEC-004/SPEC-033 entries as this same orthogonal schema-fit gap in
passing, without pursuing it; this issue is that follow-up, scoped and
tracked.

### Important nuance found during this investigation — verify against ISSUE-051's fix

Every currently-known instance above is declared by a spec whose status is
`draft` (SPEC-003, SPEC-004, SPEC-033). ISSUE-051's fix (scope
`ExtractContractEntries` to `implemented` specs only) would stop extracting
**all** of these regardless of this issue — not because the non-Go problem
was solved, but because the declaring specs aren't built yet either. So once
ISSUE-051 lands, **re-audit before assuming this issue still has ~7 live
instances**: it may temporarily drop to zero known instances (latent, not
fixed) rather than genuinely reconcile down to zero. The underlying gap is
real and will resurface the moment any spec — one of these three reaching
`implemented`, or any future spec — declares a `kind: constant`/`variable`
contract on a non-Go artifact while `implemented`. `SPEC-042` (an
`implemented` spec) already shows the shape can occur on live specs too: it
declares `kind: variable` contracts on a pack's `pack.yml` rule entry and a
shell convert script's stdin/stdout contract (`cmd/backstop/testdata/go-toolchain/...`),
both non-Go-signature — worth checking whether the current compiler
correctly handles or silently mis-scores those before assuming this issue's
population is exactly the eight rows above.

### Why it matters

The contracts schema explicitly allows non-`function` kinds
(`type`/`interface`/`method`/`constant`/`variable`) and nothing in the
schema restricts contracts to Go artifacts — declaring a contract on a hook
script, an agent definition, or a JSON schema is a reasonable way to make a
cross-cutting hand-off citable and traceable. But the signature *compiler*
only understands Go, so any such contract is permanently unverifiable
structurally: it will read as either silently vacuous (if never diff-scoped
in) or as a permanent, un-fixable "signature not found" false positive (if
scoped in) — a Go-shaped check force-fit onto an artifact it was never built
to check.

## Solution

Not committed — left open for the plan. Three directions, none clearly
correct without a decision:

1. **Scope contracts out of `contract_signature` when the declared file is
   not Go source.** `ExtractContractEntries` (or a filter ahead of the
   compiler) skips a contract whose `file` doesn't end in `.go`, so
   non-Go contracts stop being run through a compiler that can never verify
   them. Cheapest fix; loses structural verification entirely for these
   entries (same disposition ISSUE-037 used for the retired
   `CheckTypeFindings` — behaviorally covered, not structurally).
2. **Route non-Go contracts to an artifact-appropriate verification
   instead of skipping them.** E.g. a JSON file's contract could be
   verified by `jq`/schema-validate that the declared shape is actually
   present; a shell script's stdin/stdout contract could be verified by a
   grep/shellcheck-style engine; a Markdown agent definition might not be
   structurally verifiable at all and should fall back to (3). This is the
   most faithful fix but requires new pack-side engine capability per
   artifact type — weigh against actual demand (only ~7-8 instances exist
   today, all currently on unbuilt specs).
3. **Retire the prose-signature contracts as descriptive documentation that
   was never meant to be a compilable Go contract**, and say so explicitly
   in the schema/pack documentation rather than leaving future authors to
   rediscover the mismatch per-contract. This treats "signature" as
   Go-contract-only by convention and non-Go hand-offs as
   documentation-only, living in the `notes` field instead of `signature`.

Whichever direction is chosen, do NOT force-fit a Go-shaped check onto
these artifacts by inventing a Go-looking fictional signature for them (the
same anti-pattern ISSUE-037 named and rejected for iota members) — the fix
should make the schema/compiler boundary honest about what `kind: constant`
+ `signature` actually promises to verify.

## References

- `packs/contracts/scripts/compile-signature.sh` — the Go-oriented compiler;
  the component this issue's contracts can never satisfy
- `pkg/gate/step_testverify.go:509` (`ExtractContractEntries`) — where a
  file-extension or artifact-kind filter would most naturally live under
  Solution direction 1
- `specs/SPEC-003-agent-hooks.spec.md` — declares the `.claude/settings.json`
  and `backstop-agent-guard.sh` non-Go contracts
- `specs/SPEC-004-spec-schema-evolution.spec.md` — declares the
  `artifacts/spec/v1/schema.json`, `spec-author.md`, `impl-reviewer.md`,
  `backstop-standards-context.sh`, and a second `.claude/settings.json`
  non-Go contract
- `specs/SPEC-033-engine-bundle-009-seam.spec.md` — the self-referential
  "contract-lock" entry (a spec declaring itself as its own provides)
- `specs/SPEC-042-coverage-production-engine.spec.md` — an `implemented`
  spec with `kind: variable` contracts on a `pack.yml` rule entry and a
  shell convert script; evidence the gap recurs on live, not just unbuilt,
  specs and should be checked once ISSUE-051 lands
- ISSUE-037 (contracts-compiler-iota-member-const-support) — the audit that
  first flagged the SPEC-004/SPEC-033 prose-signature entries as this
  orthogonal schema-fit gap, without pursuing it
- ISSUE-051 (contract-signature-scopes-to-implemented-specs) — the
  status-scoping fix this issue's residual population depends on; re-audit
  this issue's instance list after ISSUE-051 lands rather than assuming it
  is unchanged
- `artifacts/issue/v1/schema.json` (`contracts.kind_enum`) — the contract
  kind vocabulary that permits non-`function` kinds without restricting them
  to Go artifacts
- `directives/DIR-015-gate-checker-hardening.directive.md` — this issue
  sits in the same gate-correctness cluster as ISSUE-036/037/038
- CLAUDE.md — "don't force-fit a check onto an artifact it wasn't built to
  verify" / no-vacuous-green, no-false-pressure first principles

## Additional evidence

- **Confirmed live recurrence (2026-08-16), during PLAN-ISSUE-129's diff-scoped gate run.**
  This is the exact re-audit the "Important nuance" section above called for — ISSUE-051 is
  now `status: closed` (plan `completed`) and SPEC-042 is `status: implemented`, so the
  re-audit trigger this issue already named is live, and this annotation *is* that re-audit.
  This is no longer latent/exploratory: it is a **confirmed, blocking recurrence**.

  The real false-RED: `contract_signature` reported `symbol go-coverage-rule signature not
  found or mismatched … expected "rule id: go-coverage, engine: go-coverage, gate_type:
  coverage"` against
  `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`. All
  three facts the signature asserts are genuinely present in that file — the `go-coverage`
  engine block (around line 65) and the `go-coverage` rule entry (around lines 136-137) — in a
  structurally valid pack.yml, not a malformed or incomplete fixture.

  Root cause confirmed directly (not inferred): `packs/contracts/scripts/compile-signature.sh`
  is a Go-source-syntax compiler only. SPEC-042's signature string doesn't start with a Go
  keyword, so the compiler falls through to its struct-field wrap path and emits a Go
  `struct{...}` ast-grep pattern — a pattern that can never match a `.yml` file, independent of
  whether the asserted facts are co-located or split across blocks. (Note: the "split across
  two blocks" framing used earlier in this issue's References section was incidental, not the
  actual cause — worth correcting if that framing resurfaces elsewhere.)

  Practical consequence: this dormant, pre-existing false-RED blocked PLAN-ISSUE-129 — an
  unrelated, pack-data-only fix — from reaching a clean gate. It only surfaced because
  PLAN-ISSUE-129's unrelated edit put this fixture file into diff scope.

  **Correction (2026-08-16):** the sentence above this one, in the original 2026-08-16
  annotation, claimed "an interim waiver citing ISSUE-053 is being applied to that specific
  violation elsewhere in that plan's implementation, pending this issue's real fix." That never
  actually happened. The PLAN-ISSUE-129 implementer tried exactly this, ran the real gate, and
  found the `@waiver` token was never harvested: the violation still fired while
  `waiver_resolution` reported clean/no-active-waivers. The cause is not the Line=0 or
  absolute-path bugs noted below (those are real but moot here) — it's that `contract_signature`
  is a STRUCTURAL dimension, deliberately excluded from `waivableDimension()`
  (`pkg/gate/step_waiver.go:66-68`, "The waivable surface is EXACTLY pack_engines +
  test_substantiveness") per SPEC-049 REQ-010's exhaustive waivable-surface matrix (spec lines
  923-938). The same non-waivable-by-design treatment applies to `artifact_status_drift`,
  `test_verification`, and `artifact_validation` (CLM-042/044/065), with a stated rationale
  (BUNDLE-013 DD-11): structural dimensions already have first-class accountable lifecycles
  (retire/replace/resolved-by/obsoleted), not inline waivers. This was never a coverage gap —
  `CLM-043` → `TestGateWaiver_Scope_ContractSignatureNotWaivable`
  (`pkg/gate/step_waiver_scope_test.go:48-54`) is an existing, passing, mandated test that
  already asserts contract_signature findings survive waiver reconciliation. The implementer's
  attempt rested on a wrong assumption about how to reach a "last resort," not a gap this issue
  needs to close. The correct accountable path for a genuine contract_signature red, per
  SPEC-049's matrix, is retire/replace/resolved-by/obsoleted on the artifact/spec side — not
  `@waiver`. PLAN-ISSUE-129 (closed) correctly did NOT leave an inert waiver token in the repo
  once this was discovered: the red is accepted and documented as a known, pre-existing residual
  outside that plan's scope, not suppressed.

  **Real, still-latent bugs found during this investigation (moot for contract_signature, since
  it never reaches the waiver adjudicator, but worth recording for whichever waivable dimension
  next needs Line/File handling):** `contract_verdict.go` never sets `Violation.Line` on a
  contract_signature mismatch, and `Violation.File` stays absolute rather than
  project-root-relative because `NormalizePath` is a no-op when called with an empty
  `projectRoot`. Neither caused this false-RED and neither should be read as the reason
  contract_signature is unwaivable — that reason is the structural-dimension exclusion above.

- **Third confirmed recurrence (2026-08-16, PLAN-ISSUE-140 implementation, bare full-diff
  `backstop gate`).** Identical finding, same file, same symbol: `symbol go-coverage-rule
  signature not found or mismatched in
  cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`. The
  implementer confirmed the fixture file is committed and unmodified in their working tree —
  last touched by commit `2a18148` (ISSUE-129's relock) — and that their own diff-scoped
  `gate --file` runs on their actual lane (pkg/check, pkg/packval/executor.go) pass clean; the
  finding only surfaces because a bare (non-diff-scoped) `gate` run spans the full diff against
  merge-base across every in-flight lane that night (190+ changed files), which happens to
  include this already-red fixture. Nothing new here beyond corroboration: same root cause
  (Go-syntax-only compiler, `.yml` target, `baseline: true` grandfathers it non-blocking), same
  non-waivable-by-design disposition. Recorded so a future session doesn't re-diagnose this from
  scratch a third time, and to note the trigger condition generalizes — any bare `gate --all`/
  full-diff run touching this testdata path will keep re-surfacing it until this issue's fix
  lands.

- **Fourth confirmed recurrence (2026-08-16, implementer-issue067, editing the same pack.yml's
  go-build/go-test bindings).** Identical finding, same file, same symbol: `symbol
  go-coverage-rule signature not found or mismatched in
  cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml: expected
  "rule id: go-coverage, engine: go-coverage, gate_type: coverage"`. Verified latent and
  edit-independent before this annotation was written: the go-coverage rule block is
  byte-identical to HEAD (diff clean around it), the actual edit only touched the go-build/go-test
  bindings, and the expected literal string appears zero times in the file at HEAD or after the
  edit — confirming (again) that touching this file at all pulls its dormant finding into diff
  scope regardless of what changed. A "split across two locations" theory was independently
  proposed again during this occurrence (`- id: go-coverage` in the rule block vs. `gate_type:
  coverage` in the engine binding) — this is the SAME theory the 2026-08-16 correction above
  already retired as incidental, not causal; the real cause remains the Go-syntax-only compiler
  (`compile-signature.sh`) being force-fit against a `.yml` target, independent of whether the
  asserted facts are co-located. No new information beyond a fourth corroboration; recorded so a
  fifth session doesn't re-propose the retired theory or re-diagnose from scratch. Also
  re-confirms the non-waivable disposition: `contract_signature` findings carry `Line 0` and an
  absolute path, but per the correction above the real reason `@waiver` cannot interim-suppress
  this class is structural (excluded from `waivableDimension()`), not the Line/path bugs — do not
  attempt a waiver workaround for this issue's instances; the fix is the compiler/schema change
  described in Solution, not a per-occurrence suppression.
