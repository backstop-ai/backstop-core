---
name: mirror-sync-guard-family
description: "Sync-guard issues (\"nothing keeps X aligned with the released pack\") split by LIFETIME, not subject matter — ISSUE-137's fixture half is permanent (DIR-024), ISSUE-174's source half is scheduled for deletion by DIR-027"
metadata:
  type: project
---

There is a recurring issue shape in this repo: *"nothing automatically keeps
`<in-repo thing>` in sync with its released external pack."* Two are filed and they
read almost identically, but they home differently — and the discriminator is the
**lifetime of the in-repo half**, not the subject matter.

- `ISSUE-137 "No automated guard keeps the go-toolchain pack fixture in sync with the
  released pack"` → homed **DIR-024 "Gate/Engine Quality"** (2026-08-16). Its pair is
  `cmd/backstop/testdata/go-toolchain/**` (a TEST FIXTURE) vs the released
  `backstop-ai/go-toolchain`. The fixture is **permanent** — the test corpus needs it,
  nothing plans to delete it — so the gap is durable and wants a real guard. Homing
  test used at the time: nothing here reports a wrong gate verdict, and the drift risk
  lives in backstop-core's own `go test` corpus, so it's DIR-024 and not DIR-032 (same
  test that kept ISSUE-115/ISSUE-125 in DIR-024).
- `ISSUE-174 "Pack Source Mirror Sync No Guard"` (filed 2026-08-18 by `PLAN-ISSUE-166`
  TASK-010) → **ambiguous, escalated, recommendation DIR-027**. Its pair is the in-repo
  pack SOURCE vs the released mirror. **Measured: `ls packs/` = `base-engines`,
  `contracts`, `substantiveness`.** `base-engines` is compiled in (`pkg/baseengines`)
  and has no mirror, so the COMPLETE exposure set is `packs/contracts` +
  `packs/substantiveness` — which is verbatim DIR-027's acceptance criterion #1
  ("no longer exist inside `backstop-core`", thread 1 tier 2, the de-vendoring).
  Transitional gap: de-vendoring deletes one side of the pair and closes it with zero
  new mechanism.

**Correction that must ride with ISSUE-174 wherever it lands:** it names "the
traceability packs" as a third instance. Measurably false —
`pkg/gate/testdata/traceability-pack` and `pkg/gate/testdata/ts-proof-pack` are
fixtures with **no external mirror** (no such repo under `~/src/projects/`, absent
from `backstop.lock`/`backstop.yml`). That third case is ISSUE-137's class. Two
confirmed instances, not three.

**Why:** following the ISSUE-137 precedent on wording alone would have slotted 174
into DIR-024 under the standing clear-fit grant and quietly committed the repo to
building cross-repo diff tooling for a duplication it has already agreed to delete.

**How to apply:** when a "keeps X in sync with the released pack" issue arrives, ask
whether the in-repo half is scheduled to be deleted before you match it to a sibling
by shape. Check `ls packs/` and DIR-027's acceptance criteria. Also verify the
claimed instance list — `diff` the actual files against the mirror repo under
`~/src/projects/backstop-<name>-pack/` and confirm each named "mirror" is really
published. Related: [[project_check_the_siblings_plan]],
[[feedback_verify_the_loss_claim]], [[project_mechanism_vs_ecosystem_gap]].
