---
name: baseline-as-attribution-oracle
description: The CI-pulled .backstop/baseline.json is a cheap decisive oracle for "is this gate red pre-existing or mine" — grep it before spending 10+ minutes on a control worktree
metadata:
  type: project
---

When a diff-scoped `backstop gate` surfaces a red on a file you edited, the standard
proof-of-inheritance is a clean-HEAD control worktree. For findings that live in
`.backstop/baseline.json` that is unnecessarily expensive: **grep the baseline first.**

A finding recorded there was observed by CI at baseline-generation time, which is
proof it predates your change. Verified 2026-08-19 on PLAN-ISSUE-172: editing the
go-toolchain fixture `pack.yml` surfaced a `contract_signature` red for
`go-coverage-rule`. One `python3 -c` over the baseline found it already recorded
(308 total violations, 2 of them `contract_signature`) — seconds, versus the ~6 min a
control worktree costs in this repo.

**Why:** this is the [[project_editing_file_pulls_it_into_gate_scope]] /
[[project_gate_scope_entry_surfaces_pack_false_positives]] class — your edit drags a
file's whole latent finding set into diff scope. The baseline is the record of what
was already latent.

**Two traps that make the baseline look like it isn't working:**

- **Baseline entries carry ABSOLUTE CI paths** (`/home/runner/work/backstop-core/...`).
  A local run's identity uses local absolute paths, so the entry does NOT suppress the
  finding locally — it still shows as a violation. That is fine for ATTRIBUTION (which
  is all you want) but means "it's in the baseline" never implies "the gate will be
  green". Especially true for `contract_signature`, whose message embeds the absolute
  path and which is also unwaivable ([[project_contract_signature_unwaivable]]).
- `baseline_comparison` frequently reports **"skipped (superseded by per-dimension
  enforcement policy)"**, so the baseline may not be consulted by the run at all. Read
  the FILE directly; do not infer from the step's verdict.

**How to apply:** on any unexpected gate red, `grep`/parse `.backstop/baseline.json`
for the rule and symbol BEFORE building a control worktree. Use the worktree only for
findings absent from the baseline (e.g. a failing test like ISSUE-162's
`TestPackAuthoringLoop_EndToEnd`, which is a pack_engines test failure, not a baselined
violation). Never waive either kind — record, name the owning issue, report.
