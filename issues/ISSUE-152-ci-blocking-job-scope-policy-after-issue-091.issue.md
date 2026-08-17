---
title: "Ci Blocking Job Scope Policy After Issue 091"
schema_version: issue/v1

issue:
  id: ISSUE-152
  title: "Ci Blocking Job Scope Policy After Issue 091"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Ci Blocking Job Scope Policy After Issue 091

★ **THIS IS A FOUNDER DECISION REQUEST, NOT A RECORD.** Every other follow-on filed by
PLAN-ISSUE-091 TASK-006 documents a fact. This one poses a live policy question the fix itself
creates, and it is filed unresolved on purpose — do not read anything below as a recommendation.

## Problem

PLAN-ISSUE-091 (`plans/PLAN-ISSUE-091-gate-all-underreports-test-findings.plan.yml`) fixed
`ISSUE-091` — `gate --all` under-reporting test-file findings relative to a diff-scoped run. That
fix removes the SOLE stated justification for one half of an existing CI policy: the ban on CI's
**blocking** job ever invoking `backstop gate --all`. Whether that ban should now lift is a
genuine founder-level call. This issue asks the question; it does not answer it, and TASK-006
item 4 (the task that filed this issue) explicitly forbids the implementer from answering it too.

### The ban, and which half is in question

CI's blocking job never uses `--all` and never uses `--file` — it runs diff-scoped with an
explicit base. **Only the `--all` half is in question here.** The `--file` half is owned by
`issues/ISSUE-093-gate-file-nongo-directory-crash.issue.md` (`--file` crashes on non-Go
directories and silently drops repeated flags) and is **completely unaffected** by PLAN-ISSUE-091.
Do not touch or reopen that half; it keeps its own reason independent of anything below.

### The ban was not an ambient convention — it was a mandated, cited test

`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml` (status: `completed`) is where the ban
was born, and it did not leave the rationale implicit. Its own words, quoted verbatim (located at
the mandated-test description, ~line 3042):

> `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` — no `--all` and no `--file` on the gate
> invocation. ISSUE-091 (`--all` under-reports vs diff) and ISSUE-093 (`--file` crashes on
> non-Go directories, repeated `--file` silently dropped) are the reasons; cite both in the test
> comment so the constraint survives its authors.

And its own "RESIDUALS AT CLOSE" entry:

> ISSUE-091 and ISSUE-093 UNCHANGED, which is precisely why CI runs diff+base scope and never
> `--all` or `--file`.

This is the **same class** as sibling filing `ISSUE-149` (`plans/PLAN-ISSUE-010`'s superseded
CLM-004 prose) — a completed plan's delivered, mandated rationale retracted by a later fix — and
it gets the same treatment: **recorded in a new issue, never by rewriting the completed plan.**
`PLAN-ISSUE-020` is not edited by this filing and must not be edited to resolve this question
either.

### A second artifact is already waiting on this exact event

`issues/ISSUE-094-ci-repo-wide-sweep-debt-inventory.issue.md` already names **closing ISSUE-091**
as its own explicit trigger to switch its (non-blocking, best-effort) repo-wide sweep job from
hand-baked `golangci-lint` to `backstop gate --all`:

> Once ISSUE-091 is fixed: `backstop gate --all`, replacing the hand-baked golangci-lint
> invocation. ISSUE-091 (`gate --all` under-reports vs. diff-scoped `backstop gate` — confirmed
> 111 diff-only findings on files `--all` reported zero for) is the explicit retirement trigger.

So a second artifact is already waiting on ISSUE-091's closure to change its own behavior. That
is a **separate job, a separate question, and not this issue's territory** — ISSUE-094 owns the
repo-wide **debt inventory** job's scope; this issue owns the **blocking** job's scope policy.
Cite ISSUE-094 as a consumer of the same triggering event, and do **not** fold this question into
it — the founder ruled ISSUE-094's own shape already (one rolling inventory issue, not
per-file filings); that ruling says nothing about whether the blocking job's ban should lift.

### Current disposition, stated plainly

**The ban remains in force, and remains enforced by an unmodified test.** PLAN-ISSUE-091 changed
only the now-false rationale TEXT, at three source sites, all TEXT-ONLY — it changed nothing
about CI's actual behavior. Verified as they now stand:

- `.github/workflows/ci.yml` — the comment on the blocking "Run the gate" step now reads: "NEVER
  the all-scope flag (ISSUE-091's under-report is FIXED — the all scope now dispatches its own
  explicit file list and agrees with a diff-scoped run over the same files — so that defect no
  longer justifies the ban; the flag stays out of the blocking job pending a founder decision)".
  The step's `run:` line — the actual gate invocation — is **untouched**; it still passes no
  `--all`.
- `workflows_test.go` (repo root) — the doc comment on
  `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` now reads: "ISSUE-091 is FIXED: `--all` now
  dispatches its own explicit file list rather than a directory target, and AGREES with a
  diff-scoped run over the same files (its CLM-008), so the historical under-report no longer
  independently justifies keeping `--all` out of the blocking job. Whether the `--all` half of
  this constraint should now LIFT is an OPEN FOUNDER DECISION — see the follow-on issue filed by
  PLAN-ISSUE-091 TASK-006 item 4." Its closing sentence, which had also gone false, now reads:
  "Diff scope with an explicit base is the shape CI's blocking job actually uses today, and it
  stays that way pending that founder decision. (CLM-020)"
- `workflows_test.go` — the flag map's `"--all"` VALUE STRING (the `%s` rationale rendered into
  the test's `t.Errorf` on failure) now reads: `"ISSUE-091 is fixed, but --all is held out of the
  blocking job pending a founder decision"`. The `"--all"` **key**, the map entry itself, the
  loop, the `strings.Contains` check, and the `t.Errorf` call are all **unchanged** — the test
  still fails the same way if `--all` ever appears on the blocking invocation.

### The question, framed without an answer

With ISSUE-091's under-report fixed, may CI's **blocking** job now use `gate --all`, or should it
stay on diff-scoped with an explicit base?

This is **not a free lift either way**, and neither consideration below is this issue's — or
PLAN-ISSUE-091's — to weigh:

- **Moving the blocking job to `--all` would surface the repo-wide debt volume** that
  `gate --all` now reports honestly (test-file findings across the whole tree) — that volume is
  ISSUE-094's territory (a rolling inventory, not per-file filings), not something the blocking
  job's own scope policy is equipped to absorb without a plan for it.
- **`--all` is not strictly louder than diff scope in every respect.** Sibling filing
  `ISSUE-151` (path-scoped pack rules dark under explicit-file dispatch — the "THIRD DIVERGENCE")
  documents that a pack rule whose `paths.include` names directory-prefixed globs now goes dark
  under `--all`'s explicit-file dispatch the same way it has always been dark on the diff-scoped
  path — so `--all` post-fix *loses* rows it used to report under the old directory-target
  dispatch, even as it gains the previously under-reported test-file rows.
- **A third consideration, measured during the fix and not named by the plan:**
  `resolveGateScopeAll` (`pkg/gate/scope.go`) skips dot-directories during its walk, so post-fix
  `--all` can never hand any file under `.github/` to an engine, and the `ci-workflows` pack's
  rules can therefore never fire on an `--all` sweep. Measured at the time of the fix: this loses
  nothing today — the four `ci-workflows` rule files yield zero findings under both dispatch
  shapes against the current workflow — so it is a **latent capability gap**, not a regression.
  But it means `--all` is not a strict superset of a diff-scoped run for dot-directory files,
  which bears directly on whether "switch the blocking job to `--all`" is as complete a move as
  it sounds. Recorded here as measured, not projected.

Raise this for founder review after PLAN-ISSUE-091 lands. It is a taste/policy call about CI
strategy — exactly the class that escalates rather than getting decided inside an implementation
lane.

## References

- **ISSUE-091** (`issues/ISSUE-091-gate-all-underreports-test-file-findings.issue.md`) — the
  source defect. Its fix is what removes the stated justification this issue asks about.
- **PLAN-ISSUE-091** (`plans/PLAN-ISSUE-091-gate-all-underreports-test-findings.plan.yml`) — the
  plan that fixed ISSUE-091. TASK-006 item 4 files this issue; TASK-004 corrections 7-9 make the
  three text-only corrections described above, deliberately without deciding this question.
- **PLAN-ISSUE-020** (`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml`, status:
  `completed`) — the completed plan that originally mandated
  `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` and its ISSUE-091/ISSUE-093 citation. Not
  edited by this filing or by any resolution of this question; completed plans are never
  rewritten.
- **ISSUE-093** (`issues/ISSUE-093-gate-file-nongo-directory-crash.issue.md`) — owns the `--file`
  half of the same ban, independently and unaffected by ISSUE-091's fix. Not in question here.
- **ISSUE-094** (`issues/ISSUE-094-ci-repo-wide-sweep-debt-inventory.issue.md`) — a consumer of
  the same triggering event (ISSUE-091 closing), for a **different** job (the non-blocking
  repo-wide sweep, not the blocking job). Does not own this question and is not folded into it.
- **ISSUE-149** (`issues/ISSUE-149-plan-issue-010-allscope-claim-superseded.issue.md`) — sibling
  filing, same class of completed-plan-claim-retraction, filed by the same TASK-006.
- **ISSUE-150** (`issues/ISSUE-150-gate-all-no-longer-reports-testdata-findings.issue.md`) —
  sibling filing (TASK-006 item 2), the testdata-exclusion behavior change.
- **ISSUE-151** (`issues/ISSUE-151-path-scoped-pack-rules-dark-under-file-dispatch.issue.md`) —
  sibling filing (TASK-006 item 3), the THIRD DIVERGENCE this issue cites as one of the
  considerations bearing on (but not deciding) the question above.
- **DIR-032** — Gate Verdict Honesty (`directives/DIR-032-gate-verdict-honesty.directive.md`) —
  this issue's home directive; slot alongside the rest of the ISSUE-091 follow-on cluster.
