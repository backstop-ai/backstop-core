---
title: "Reconcile Contract Drift Exposed By Kind Aware Compiler"
schema_version: issue/v1

issue:
  id: ISSUE-038
  title: "Reconcile Contract Drift Exposed By Kind Aware Compiler"
  type: technical-debt
  status: open
  created: "2026-07-06"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-038: Reconcile Contract Drift Exposed By Kind Aware Compiler

## Problem

ISSUE-036 made the `backstop/contracts` signature compiler
(`packs/contracts/scripts/compile-signature.sh`) kind-aware: it previously
only handled `kind: function` contracts, silently compiling every `type` /
`const` / `method` / `interface` / `variable` contract into an ast-grep
pattern that could never match real Go source. Because `contract_signature`
is diff-scoped (`StepContractSignatureScopedFunc`,
`pkg/gate/step_contract.go:50-54`), those non-`func` contracts were never
evaluated for any diff that didn't happen to touch their declared file — they
read as green for a reason that had nothing to do with correctness (see
ISSUE-036's "Interaction with diff-scope" section, which predicted exactly
this).

Un-vacuuming the compiler exposed that this is not hypothetical: a
non-trivial number of specs carry contract declarations that have silently
drifted from the current code, or reference symbols the BUNDLE-011/012
native-toolchain cutover deleted outright. These contracts were always
broken — the func-only compiler simply never asked the question. This issue
is the tracked backlog to reconcile that drift, spec by spec, now that the
question is actually being asked.

**Scope statement:** this is an audit of ALL specs' `contracts:` declarations
against current code, not a fixed list. `contract_signature` is diff-scoped,
so drift surfaces incrementally as more files enter a gate's diff — the
instances below are a verified sample as of this writing, not an exhaustive
enumeration. Treat the true count as exploratory, the same posture ISSUE-036
and ISSUE-037 both took toward their own blast radius.

### Why this isn't a regression or a gate bug

- These are pre-existing facts about the code and specs, not something this
  session's work introduced. The gate's own `new_violations: 0` on every
  surfaced item confirms none of them were caused by the kind-aware compiler
  landing — the compiler only made a pre-existing, always-broken state
  observable.
- As part of closing out that work, `contract_signature` was given
  `baseline: true` in `backstop.yml` (consistent with the other quantitative
  dimensions — `coverage_threshold`, `substantiveness_threshold`,
  `pack_engines` — that already carry a baseline), and the baseline was
  refreshed. Confirmed in `backstop.yml`:

  ```yaml
  contract_signature:
      baseline: true
      level: block
  ```

  This means the drift below is **grandfathered**, not blocking: the gate
  will not go red on these pre-existing violations, only on *new* ones added
  after the baseline was cut. This issue is the deliberate, tracked ratchet-down
  to drain that grandfathered debt over time — not an emergency, but not
  something to leave grandfathered indefinitely either.

### Verified instances (2026-07-06 — sample, not exhaustive; re-verify at plan/fix time)

Each entry below was independently re-verified against current code during
authoring of this issue (not just carried over from the originating session),
because the two-way split matters for how each gets fixed: **stale**
contracts reference a symbol that no longer exists at all (retire the
contract) vs. **drifted** contracts reference a symbol that still exists but
whose file, package, or signature has moved (repoint/fix the contract).

**Stale — symbol confirmed deleted from the codebase (retire these contracts):**

| Symbol | Declaring spec | Declared file | Verification |
|---|---|---|---|
| `(*realCodeChecker).runCheck` | SPEC-030 | `cmd/backstop/gate.go` | `grep -rn realCodeChecker --include=*.go .` → zero non-comment hits; `realCodeChecker` was deleted in the BUNDLE-011/012 cutover |
| `loadBridgedToolchainPacks` | SPEC-040, SPEC-046 (declared as `provides` in **both**) | `cmd/backstop/gate.go` | `grep -rn "func loadBridgedToolchainPacks"` → zero hits; deleted |
| `funcPattern` | SPEC-045 | `pkg/gate/step_testverify.go` | The file's own comments describe it as "the DELETED baked Go-shaped `funcPattern`" (step_testverify.go:196, step_testverify_test.go:655) — SPEC-045 explicitly deleted this as a baked-language literal; the spec's own `provides` entry for it was left behind |
| `checkSubstantiveness` | SPEC-037 | `pkg/gate/step_testverify.go` | No `func checkSubstantiveness` definition anywhere in the codebase; only appears in strangler-harness test comments describing it as the "pre-deletion ... analyzer" oracle — reads as deleted, though this needs final confirmation against the strangler migration's actual end-state before retiring (see caveat below) |

Note: `checkSubstantiveness` was reported by the originating session as
"drifted/moved" rather than stale. Re-verification during authoring did not
find a live `checkSubstantiveness` function anywhere and found comment
language ("pre-deletion... analyzer") consistent with it having been retired
as part of the substantiveness strangler migration. Treat its disposition
(stale vs. drifted) as unconfirmed pending a closer read of
`pkg/gate/substantiveness_strangler_migration_test.go` and siblings at fix
time — do not assume either bucket without that check.

**Drifted/moved — symbol exists but file/package/signature has changed (repoint or fix):**

| Symbol | Declaring spec(s) | Declared file | What's actually drifted |
|---|---|---|---|
| `GateResult` | SPEC-010 | `cmd/backstop/gate.go` | `cmd/backstop/gate.go:21` now reads `type GateResult = gate.GateResult` — the real struct moved to package `pkg/gate`; the contract's declared file now holds only a type alias, not the definition |
| `StepResult` | SPEC-010 | `cmd/backstop/gate.go` | Same pattern: `cmd/backstop/gate.go:24` is `type StepResult = gate.StepResult`, real type moved to `pkg/gate` |
| `buildGateSteps` | SPEC-019, SPEC-034, SPEC-036, SPEC-040, SPEC-043 (five specs each declare a `provides` for this one symbol) | `cmd/backstop/gate.go` | Current signature is `func buildGateSteps(projectRoot string, scope ...*gate.GateScope) []gate.StepFunc` (gate.go:586) — five specs asserting on one evolving symbol means at least some of these declared signatures are stale relative to the current one and need reconciling as a set, not independently |
| `SourceClassifier.IsTestFile` | SPEC-045 | declared at `pkg/gate/step_testverify.go` | Method now lives in `pkg/gate/classification.go` (confirmed: `func (c SourceClassifier) IsTestFile(path string) bool`, classification.go:50) — same package, wrong file |
| `SourceClassifier.HasTestGlobs` | SPEC-045 | declared at `pkg/gate/step_testverify.go` | Same as above — now in `pkg/gate/classification.go` (classification.go:62), not `step_testverify.go` |
| `spec-schema` | SPEC-004 | `artifacts/spec/v1/schema.json` | Declared `kind: constant` with `signature: "JSON schema definition"` — this is prose describing an artifact, not a Go declaration at all. This likely isn't independent drift so much as the **same non-Go-signature schema-fit gap ISSUE-037's audit already flagged for SPEC-004 and SPEC-033** ("their `kind: constant` entries are not Go declarations at all ... a pre-existing schema fit issue orthogonal to this one"). Needs a decision at fix time on whether it's reconciled here or folded into that separate schema-fit thread rather than force-fit into a Go signature. |

### Why this matters

This is the same "silent vacuous-green hole in the gate's own enforcement"
category ISSUE-036 exists to close, one layer down: the compiler now asks the
right question of every contract kind, but a chunk of the *answers* it gets
back are stale because nobody has gone back through the specs that predate
the fix to reconcile them. Leaving grandfathered debt grandfathered
indefinitely is its own quiet failure mode — the baseline exists to unblock
the gate, not to make the debt invisible again.

## Solution

Not committed — left open for the plan, per the honest uncertainty about
total scope. Direction to evaluate:

1. **Enumerate exhaustively**, not just re-confirm the sample above. Scan
   every `specs/*.spec.md` `contracts:` block's `provides` entries against
   current code (a script pass mirroring what this issue's authoring did by
   hand — `grep`/`ast-grep` each declared `{file, name, kind}` triple against
   the live symbol) to produce the full worklist. Expect the true count to
   exceed the ~10 verified here.
2. **Per-spec reconciliation**, spec by spec, via the spec-author agent (per
   this project's align-predating-artifacts principle — these are stale spec
   artifacts conflicting with current, aligned code; update them openly
   rather than working around them):
   - **Stale** (symbol confirmed deleted): retire the contract entry from its
     declaring spec, with a note pointing at the cutover/change that deleted
     it (BUNDLE-011/012, SPEC-045's baked-literal deletion, etc.).
   - **Drifted/moved** (symbol exists, file/signature changed): repoint the
     contract's `file` and/or fix its `signature` to match current code —
     never rewrite the underlying code to match a stale contract; the
     contract describes reality, not the other way around.
   - Where one symbol is declared by multiple specs (e.g. `buildGateSteps`
     across five specs), reconcile as a set: confirm which declarations are
     still live obligations of that spec's requirements vs. which have been
     superseded by a later spec's own contract on the same symbol.
3. **Resolve the `spec-schema` / non-Go-signature question** (SPEC-004,
   SPEC-033, and any sibling `kind: constant` entries whose `signature` is
   prose rather than a Go declaration) either as part of this backlog or by
   explicitly deferring it to a schema-fit issue — don't silently force a
   Go-shaped fix onto a contract that was never describing Go.
4. **Re-run `contract_signature` (unbaselined, on the full sweep) after each
   batch** to confirm the reconciled contracts actually pass and no new drift
   was introduced — then, once the backlog is meaningfully drained, revisit
   whether `contract_signature`'s `baseline: true` should be lifted back to
   a hard block, per this project's baseline-is-a-ratchet-not-a-waiver
   philosophy.

**Core uncertainty, stated honestly:** the total repo-wide count of drifted
contracts is not known — `contract_signature` is diff-scoped, so it has only
ever surfaced what recent diffs happened to touch. The plan should budget for
an exhaustive scan turning up meaningfully more than the ~10 instances
verified here, and should not assume the fix is "ten small edits."

## References

- `packs/contracts/scripts/compile-signature.sh` — the kind-aware compiler
  (ISSUE-036) whose fix exposed this drift
- `pkg/gate/step_contract.go:50-54` — `StepContractSignatureScopedFunc`, the
  diff-scope filter that kept this drift dark until now
- `backstop.yml` — `contract_signature: { baseline: true, level: block }`,
  the grandfathering mechanism this issue is draining
- ISSUE-036 (contracts-pack-compiler-func-only-signatures) — the un-vacuuming
  fix that exposed this backlog
- ISSUE-037 (contracts-compiler-iota-member-const-support) — the sibling gap
  (bare iota-block consts) in the same compiler; its audit already flagged
  SPEC-004/SPEC-033's non-Go-signature `kind: constant` entries as a related,
  orthogonal schema-fit issue that overlaps this issue's `spec-schema` item
- ISSUE-018 (remove-vestigial-baked-in-code) / BUNDLE-011 / BUNDLE-012 — the
  native-toolchain cutover that deleted `realCodeChecker` and
  `loadBridgedToolchainPacks`, the source of two of the "stale" instances above
- SPEC-038-traceability-contracts-pack.spec.md — the contracts pack's origin
  spec (compiler, diff-scoped step, `kind` enum)
- SPEC-010-gate.spec.md — declares the `GateResult`/`StepResult` contracts now
  drifted to type aliases pointing at `pkg/gate`
- SPEC-019, SPEC-034, SPEC-036, SPEC-040, SPEC-043 — the five specs that each
  independently declare a `provides` contract on `buildGateSteps`
- SPEC-030-packs-only-native-standards-removal.spec.md — declares the stale
  `(*realCodeChecker).runCheck` contract
- SPEC-040-toolchain-pack-cutover.spec.md,
  SPEC-046-retire-language-toolchain-bridge.spec.md — both declare the stale
  `loadBridgedToolchainPacks` contract
- SPEC-037-traceability-substantiveness-pack.spec.md — declares
  `checkSubstantiveness`, disposition (stale vs. drifted) unconfirmed, see
  caveat above
- SPEC-045-de-go-test-verification-discovery.spec.md — declares `funcPattern`
  (confirmed stale/deleted) and `SourceClassifier.IsTestFile` /
  `SourceClassifier.HasTestGlobs` (confirmed drifted to
  `pkg/gate/classification.go`)
- SPEC-004-spec-schema-evolution.spec.md — declares the `spec-schema`
  contract with a prose (non-Go) signature
- Project memory `feedback_align_predating_artifacts` — governs how this
  reconciliation should proceed: update the stale spec artifacts openly via
  the spec-author agent, don't work around them
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy: the baseline exists
  to make debt visible-but-non-blocking, not to make it permanently invisible
