---
name: gate-all-underreports-vs-diff
description: gate --all is NOT a superset of diff-scoped gate — it under-reports pack_engines on files it does scan, so "intersect gate --all with changed files" systematically under-counts the blocking set
metadata:
  type: project
---

Measured 2026-07-27 on backstop-core (ISSUE-087 rename sweep, 324 files):
`./bin/backstop gate --json` (diff) reported **153** pack_engines rows;
`./bin/backstop gate --all --json` reported **90**. They are not nested sets —
124 diff-only, 61 all-only, only 22 shared.

The all-only 61 are fully explained (all on files outside diff scope — unswept
files that never referenced the module path). The diff-only side is NOT:
**111 diff-only findings sit on files `--all` reported ZERO findings for.**
Concrete: `cmd/backstop/artifact_new_test.go` has five real `code, _ :=` sites;
diff scope reports 4 no-ignored-errors rows, `--all` reports none. `--all` cited
only **37 distinct files** across the whole repo.

The gap is go-standards (semgrep) on `*_test.go`: `--all` returns almost no
test-file semgrep rows, while diff scope (explicit per-file targets) returns
many. go-toolchain/golangci-lint rows do appear for test files in both modes.

**Why:** a plan that says "run `gate --all`, intersect its violation list with
the files you changed, that intersection is your activated set" — as
PLAN-ISSUE-087 TASK-004 does — will UNDER-COUNT badly. It sized TASK-016 at ~31
rows; the real blocking set was 153.

**How to apply:** derive the activated/blocking set from the DIFF-SCOPED gate
(`--json`, parse `steps[].violations`, fields `rule/file/message/identity/
region_hash`), never from `--all`. Use `--all` only for the dormant inventory on
files you did NOT touch. When comparing before/after, run the SAME mode both
sides — that same-mode `--all` diff is what proved the rename created zero new
findings. Related: [[project_editing_file_pulls_it_into_gate_scope]],
[[feedback_netnegative_gate_baseline]], [[project_selfpack_b2_token_rule_scope]].
