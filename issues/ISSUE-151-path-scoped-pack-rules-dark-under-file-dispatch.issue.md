---
title: "Path Scoped Pack Rules Dark Under File Dispatch"
schema_version: issue/v1

issue:
  id: ISSUE-151
  title: "Path Scoped Pack Rules Dark Under File Dispatch"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Path Scoped Pack Rules Dark Under File Dispatch

## Problem

A pack rule whose `paths.include:` names repo-relative, **directory-prefixed** globs (e.g.
`cmd/backstop/pack_gate*.go`, `pkg/gate/*.go`) fires when semgrep is handed a **directory**
target, and does **not** fire when the same file is handed over **explicitly**. Stated as the
observable contract, without speculating about semgrep's internals: a `paths.include` glob
naming a directory prefix is satisfied under directory dispatch and unsatisfied under
explicit-file dispatch.

This is filed per `PLAN-ISSUE-091` TASK-006 item 3 — the "THIRD DIVERGENCE" found during that
plan's review. It is the follow-on with the **widest blast radius** of the four TASK-006 items,
because it is not scoped to backstop-core's own tree — it applies to any pack, in any consuming
repo, that scopes a rule with a directory-prefixed include.

### Say this plainly: the hole is pre-existing, not newly introduced

The diff-scoped gate has **always** dispatched explicit files — that is its normal, everyday
shape, unchanged by `PLAN-ISSUE-091`. So a rule like this has already been effectively **dead**
on the everyday `backstop gate` today, before that plan ever touched anything.
`PLAN-ISSUE-091`'s fix did **not** create this hole. It collapsed `--all`'s dispatch onto the
same explicit-file-list shape the diff-scoped gate already used, and that collapse only
**extends** the existing blindness to `--all`, which is what makes it **visible** for the first
time (previously `--all`'s directory-target dispatch masked it by accident). Do not soften this
framing — the defect predates that plan and was not introduced by it.

### Measured, at the current working tree, real semgrep 1.156.0

Site: backstop-self's `no-structural-name-split-on-spine` rule
(`.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml`), whose `paths.include:` list is:

```yaml
paths:
  include:
    - "pkg/gate/*.go"
    - "pkg/check/manifest.go"
    - "pkg/check/parsers.go"
    - "pkg/validate/plan.go"
    - "cmd/backstop/gate.go"
    - "cmd/backstop/pack_gate*.go"
    - "pkg/pack/engine/binding.go"
    - "*structural-name-split*.go"
    - "*structured-property-read*.go"
  exclude:
    - "*_test.go"
    - "*testdata*"
```

Two rows, cited **by function name, never by line number** — both files are under concurrent
edit by other lanes tonight, and `cmd/backstop/pack_gate.go` alone shifted ~112 lines during
`PLAN-ISSUE-091`'s own review:

- the `strings.Fields` call inside `splitCommand` in `cmd/backstop/pack_gate.go`
- the `strings.Fields` call inside `engineToolName` in `cmd/backstop/pack_gate_provision.go`

Both rows appear under directory dispatch and are **absent** under explicit-file dispatch.
Reproduced with targets spanning two top-level directories as well as with a file passed alone,
so it is not a common-root artifact.

### Exact reproduction

```bash
RULE_FILE="$(pwd)/.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml"
CONFIGS=(--config "$RULE_FILE")   # array — see the zsh note below

# directory dispatch: fires
semgrep --sarif --quiet "${CONFIGS[@]}" cmd/backstop

# explicit-file dispatch: does not fire, same rule file, same files
semgrep --sarif --quiet "${CONFIGS[@]}" cmd/backstop/pack_gate.go cmd/backstop/pack_gate_provision.go
```

Filter the SARIF `results[]` to rows carrying no `suppressions` entry (the **ACTIVE** layer —
what the gate actually prints, i.e. post-`parseSarif`) and to `ruleId` ending in
`no-structural-name-split-on-spine`. The directory-dispatch run yields exactly the two rows
above (at `cmd/backstop/pack_gate.go:1016` and `cmd/backstop/pack_gate_provision.go:188` as of
this writing — expect drift); the explicit-file run yields zero.

General recipe for re-deriving the config flags against any pack: assemble `--config <abs path>`
for each `rule_path` the pack's `pack.yml` declares (`backstop-ai/backstop-self` currently
declares exactly one, `rules/no-baked.yml`, so the array above has one entry — a pack with
multiple rule files needs one `--config` per file), then run `semgrep --sarif --quiet <configs>
<target>` once per dispatch shape and diff the active rows.

**zsh note, recorded so the next reader does not lose a cycle to it:** zsh does **not**
word-split unquoted parameter expansions the way bash does. Building the `--config` flags as a
single space-joined string and interpolating it unquoted fails with `semgrep: error: File name
too long` when the joined string ends up being fed as one argument (or splits wrong). Build the
config flags as a shell **array** (`CONFIGS=(--config "$RULE_FILE" --config "$OTHER_FILE" ...)`
and expand as `"${CONFIGS[@]}"`) — that cost the implementer a debugging cycle during
`PLAN-ISSUE-091`.

### Independent in-tree corroboration

The doc comment on `ciGlobScopingProblems` in `cmd/backstop/ci_recipes_harness_test.go` already
documented this exact mechanism **before** `PLAN-ISSUE-091` existed, written by someone else,
from a different starting point — CI-recipe glob scoping, not this rule. That makes it
corroborating evidence, not a restatement of this issue's own measurement. Quoting its mechanism
half (its trailing `--all`-scoped clauses were just updated by `PLAN-ISSUE-091` TASK-004
correction 6 to reflect that `--all` no longer has a directory-dispatch branch to be missing;
this mechanism half is unchanged and correct):

> a multi-segment include matches ZERO files ... under the gate's DEFAULT diff-scoped dispatch,
> which hands semgrep EXPLICIT FILE targets

Same asymmetry, same root cause, discovered independently, in a completely different corner of
the codebase.

### This is a pack-contract question, not a core arg-shaping bug

The open question is: **how does a pack express path scoping such that it survives BOTH
dispatch shapes?** This issue does not propose or design an answer. `PLAN-ISSUE-091` explicitly
refused to add path rewriting, an `--include`-style flag, or a directory-target fallback in
backstop core, because doing so would re-introduce the exact two-code-paths disease that plan
existed to cure (one dispatch shape for diff-scoped gates, a different one for `--all`). Any fix
belongs on the pack-authoring side of the contract — how a rule's `paths.include` is written, or
how packval validates it — not on backstop's dispatch code.

### Severity

Severity is the founder's call. Stated honestly without pre-judging it: **any pack rule using a
directory-prefixed `paths.include` is currently a silent no-op on the diff-scoped gate** — the
vacuous-green class. That is not limited to backstop-self or to this one rule; it is a property
of how semgrep resolves `paths.include` against a directory target versus an explicit file list,
and it applies to every pack, in every consuming repo, that writes a rule this way.

### One more consequence, and it moves the opposite direction from what a first reading suggests

These two findings are currently the **only** thing that pulls two stale `@waiver:` tokens
sitting one line above them (`cmd/backstop/pack_gate.go`, `cmd/backstop/pack_gate_provision.go`,
both keying the pre-rename `backstop/self/...` rule-ID prefix) into `Adjudicate`'s harvest
window, where they surface in the `waiver_resolution` dimension's "N unused/dangling" clause.
Once `PLAN-ISSUE-091`'s fix silences this rule under explicit-file dispatch, nothing harvests
those tokens and they cannot even reach the `Unused` bucket — their contribution to that count
goes from 2 to 0. **That is a loss of honesty, not a cleanup.** This consequence is owned by
`ISSUE-097` ("Unbound Selfpack Waivers Fail Open"), which has already been appended to with this
exact evidence — it is referenced here rather than duplicated.

## References

- `ISSUE-091` — "gate --all under-reports test findings"
  (`issues/ISSUE-091-gate-all-underreports-test-file-findings.issue.md`), the source issue whose
  fix surfaces this pre-existing hole.
- `PLAN-ISSUE-091` — `plans/PLAN-ISSUE-091-gate-all-underreports-test-findings.plan.yml`, notes
  section "THIRD DIVERGENCE, FOUND DURING REVISION — A LOSS, NOT A GAIN" (the measurement and
  framing this issue is filed from) and TASK-006 item 3 (files this issue).
- `ISSUE-097` — "Unbound Selfpack Waivers Fail Open" (`issues/ISSUE-097-unbound-selfpack-waivers-
  fail-open.issue.md`) — owns the two stale `@waiver:` tokens whose visibility this defect
  currently (and only currently) provides; do not duplicate that ownership here.
- `ISSUE-149` — sibling filing (`PLAN-ISSUE-091` TASK-006 item 1), the `PLAN-ISSUE-010` CLM-004
  supersession.
- `ISSUE-150` — sibling filing (`PLAN-ISSUE-091` TASK-006 item 2), `gate --all` no longer
  reporting `testdata/` findings.
- `DIR-032` — Gate Verdict Honesty, this issue's home directive; slot alongside the rest of the
  `ISSUE-091` follow-on cluster.
- `cmd/backstop/ci_recipes_harness_test.go` — `ciGlobScopingProblems`, the independent in-tree
  corroboration of the same mechanism from CI-recipe glob scoping.
- `.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml` — the `no-structural-name-split-
  on-spine` rule definition carrying the directory-prefixed `paths.include` list measured above.
