---
name: baseline-committed-premise
description: SPEC-047's CLM-030/031/032 tests read the gitignored .backstop/baseline.json as if it were committed — any "CI has no baseline" issue is a TEST defect, not a gate defect, and the local baseline has zero backstop-self findings so "just generate one" is a vacuous green
metadata:
  type: project
---

`.backstop/baseline.json` is **gitignored by design** (`.gitignore:43`; `git ls-files
.backstop/` returns only `coverage-exclusions`) — `DIR-018` (done) states it in its own
charter: "gitignored, CI-regenerated, and by design not a durable record." Yet
`SPEC-047` (`implemented`) claims CLM-030/031/032 say "the **committed** baseline," and
their three mandated tests in `cmd/backstop/bun_ratchet_flip_test.go:20-22` `os.ReadFile`
the repo-root baseline and `t.Fatalf` when absent. They fail in **any clean checkout**,
not just CI; they pass on a dev box only because a generated artifact happens to sit
untracked in the tree.

**Why it matters:** ISSUE-176 filed this as "CI's `gate` job has no baseline" and proposed
wiring one into `.github/workflows/ci.yml`. That is the wrong defect site — the gate
degrades exactly as `SPEC-019` specifies: `resolveBaselineCache`
(`cmd/backstop/gate.go:331-338`) tries a remote refresh, then WARNS ("baseline
unavailable… run `backstop baseline pull`") and skips comparison. Nothing in the product
is broken.

**The trap, and it is the real reason to escalate rather than slot:** the local baseline
(2026-08-16, 75 `pack_engines` findings) carries **zero** `backstop-ai/backstop-self`
findings — all 75 are `go-toolchain` — and the `no-language-literal-on-neutral-spine`
rule appears 0 times. So these tests pass by finding nothing in a dimension that may never
have been evaluated, and the artifact cannot distinguish clean from unevaluated. Making CI
generate a baseline would flip three reds green while proving nothing: `ISSUE-086`'s
second verification criterion in miniature, and precisely the ratchet-clean risk of the
**founder-reaffirmed hold** on `DIR-003` (2026-08-10: "precondition-met is not the same as
risk-proven-safe").

**How to apply:**
- Any future artifact of the shape "CI/the gate lacks a baseline" → check `gate.go:331`
  FIRST. The gate handles absence; the failing thing is almost always a test reading real
  repo state.
- Home such an issue on the **defect site** (repo test-harness false red → `DIR-024`, the
  ISSUE-137 discriminator), not the subject matter (`DIR-003` owns the baseline
  SUBSYSTEM) — but escalate, because `DIR-003` genuinely also has claim: it owns
  `ISSUE-086`, and `PLAN-SPEC-019` (stale `draft`; `SPEC-019` is already `implemented`)
  still lists `.github/workflows/ci.yml` in scope.
- Never accept `t.Skip`-on-absent as the fix (silent green) and never accept an in-job
  generated baseline consumed blind. The hermetic precedent is in the same package:
  `writeRatchetBaseline` (`gate_base_ratchet_test.go:124-139`) plants one in a temp root.
- Amending the CLM text is **orphaned work**: `SPEC-047` ← `BUNDLE-012` ← `DIR-014`
  (`done` 2026-07-06). See [[project_homed_but_orphaned_bundles]].

Related: [[feedback_verify_the_loss_claim]], [[project_zero_baked_violations_have_no_home]],
[[project_linux_ci_residual_family]].
