---
title: "Ci Blocking Job File Scope Policy After Issue 093"
schema_version: issue/v1

issue:
  id: ISSUE-156
  title: "Ci Blocking Job File Scope Policy After Issue 093"
  type: technical-debt
  status: closed
  created: "2026-08-17"
  closed: "2026-08-17"

resolved-by: "73eedb135b491685ea7251fc5fc4365ac9dbd5fa"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Ci Blocking Job File Scope Policy After Issue 093

★ **THIS IS A FOUNDER DECISION REQUEST, NOT A RECORD.** It poses a live policy question
PLAN-ISSUE-093 itself created and explicitly declined to answer — do not read anything below as
a recommendation.

## Problem

PLAN-ISSUE-093 (`plans/PLAN-ISSUE-093-gate-file-nongo-dir-crash.plan.yml`, implementing
`ISSUE-093`) fixed both defects that justified banning `backstop gate --file` from CI's
**blocking** job: (1) a package-scoped engine dispatched into a directory it didn't own used to
crash — it is now skipped with a loud, non-blocking capability-absent advisory instead; and (2)
repeated `--file` flags used to silently keep only the last occurrence (and an empty `--file`
value used to silently fall through to a diff-scoped sweep) — `--file` is now a genuinely
repeatable string array, and an empty value is a hard config error. That fix removes the sole
stated justification for the `--file` half of CI's blocking-job scope ban. Whether that half
should now lift is a genuine founder-level call. This issue asks the question; it does not answer
it.

### The ban, and which half is in question

CI's blocking job never uses `--all` and never uses `--file` — it runs diff-scoped with an
explicit base (`.github/workflows/ci.yml`, step `run: ./bin/backstop gate --base "$BASE"`, which
this filing does not touch). **Only the `--file` half is in question here.** The `--all` half is
a separate, already-filed founder decision — `issues/ISSUE-152-ci-blocking-job-scope-policy-after-issue-091.issue.md`
— triggered by ISSUE-091's fix and explicitly out of scope for that filing's own resolution of
this one; they are siblings asking the same class of question about the two different flags the
same test pins.

### The ban was not an ambient convention — it was a mandated, cited test

`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml` (status: `completed`) is where the ban
was born, mandating `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` and citing both ISSUE-091
(`--all` under-reports) and ISSUE-093 (`--file` crashes on non-Go directories, repeated `--file`
silently dropped) as the reasons. ISSUE-152 already recorded the ISSUE-091 half of this same
retraction pattern. This issue records the ISSUE-093 half — same class, same treatment: recorded
in a new issue, never by rewriting the completed plan. `PLAN-ISSUE-020` is not edited by this
filing.

### Current disposition, stated plainly

**The ban remains in force, and remains enforced by an unmodified test.** PLAN-ISSUE-093's own
task list explicitly surfaced this as a required follow-on (plan line ~608: "the founder-level
question 'may CI's blocking job now use `--file`?' is surfaced as a required follow-on") without
answering it. Verified as they now stand in `workflows_test.go` (repo root):

- The doc comment on `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` now reads, for the
  `--file` half: "ISSUE-093 is ALSO FIXED, and its half of this ban has likewise lost its stated
  reason: a package-scoped engine is now SKIPPED with a loud non-blocking advisory when the
  dispatching pack's declared classification claims nothing in the scope (so an unclaimed file no
  longer crashes the engine), and `--file` accumulates across occurrences instead of keeping only
  the last. Whether the `--file` half of this constraint should now LIFT is a SECOND OPEN FOUNDER
  DECISION, mirroring the `--all` one above. Until both are decided, the ban stays in force and
  this test keeps enforcing it." (CLM-020)
- The flag map's `"--file"` VALUE STRING (the rationale rendered into the test's `t.Errorf` on
  failure) now reads: `"ISSUE-093 is fixed, but \`--file\` is held out of the blocking job pending
  a founder decision"`. The `"--file"` **key**, the map entry itself, the loop, the
  `strings.Contains` check, and the `t.Errorf` call are all **unchanged** — the test still fails
  the same way if `--file` ever appears on the blocking invocation.
- `.github/workflows/ci.yml`'s blocking "Run the gate" step `run:` line is **untouched** — it
  still passes no `--file`.

### The question, framed without an answer

With ISSUE-093's two defects fixed, may CI's **blocking** job now use `gate --file`, or should it
stay on diff-scoped with an explicit base?

This is **not a free lift either way**, and neither consideration below is this issue's — or
PLAN-ISSUE-093's — to weigh:

- **What changed:** both defects that justified the ban are gone. A file the dispatching pack
  doesn't own no longer crashes the engine — it produces a loud, non-blocking advisory instead.
  Repeated `--file` flags now genuinely accumulate scope rather than silently dropping all but the
  last. An empty `--file` value is now a hard config error rather than a silent fallthrough to a
  full diff-scoped sweep — so a caller can no longer be surprised into a broader sweep than
  requested.
- **What stays true either way:** the ban's *enforcement mechanism* — the mandated test pinning
  the blocking job's actual gate invocation — is sound and unaffected by this question. Nothing
  about `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope`'s test logic needs to change regardless
  of how this is decided; only which flags it pins would change, and only by an explicit,
  reviewed edit at the time the founder decides.
- **What this issue does not evaluate:** whether CI's blocking job even has a use case for
  `--file` (it currently scopes via `--base`, i.e. diff-against-merge-base, not an explicit file
  list) — if the answer is "no realistic CI caller ever wants exact-file scope over diff scope,"
  that itself is a valid resolution and does not require touching the ban.

Raise this for founder review alongside (or after) ISSUE-152. It is a taste/policy call about CI
strategy — exactly the class that escalates rather than getting decided inside an implementation
lane.

## Resolution

**Founder ruling: the ban stays — permanently, on a different and durable basis.** Not because
the original defects (ISSUE-093's crash-on-unowned-directory and silently-dropped repeated
`--file` flags) are unfixed — they are — but as a deliberate, standing policy choice: CI's
blocking job is a latency- and scope-sensitive path, and diff scope with an explicit base is the
deliberately narrow, fast, predictable shape that job wants, independent of whether `--file` is
now correct. This is not a placeholder pending further review; it is a durable decision.

Recorded in code at commit `73eedb135b491685ea7251fc5fc4365ac9dbd5fa`:
`workflows_test.go`'s `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` doc comment and the
`--file` flag-map rationale string were rewritten to state this ruling explicitly — the test's
BEHAVIOR is unchanged (the ban was already enforced; nothing was lifted), only the stated
rationale changed from "pending a founder decision" to "founder ruling: keep it, durably, on
policy grounds." `.github/workflows/ci.yml`'s blocking gate invocation is likewise unchanged — it
still passes no `--file`.

This resolves the `--file` half of the question this issue posed. The `--all` half is the sibling
question owned by ISSUE-152, ruled the same way.

## References

- **ISSUE-093** (`issues/ISSUE-093-gate-file-nongo-directory-crash.issue.md`) — the source defect.
  Its fix is what removes the stated justification this issue asks about.
- **PLAN-ISSUE-093** (`plans/PLAN-ISSUE-093-gate-file-nongo-dir-crash.plan.yml`) — the plan that
  fixed ISSUE-093 (repeatable `--file`, empty-value hard error, capability-absent advisory instead
  of a crash) and explicitly surfaced this founder-decision follow-on without answering it.
- **PLAN-ISSUE-020** (`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml`, status:
  `completed`) — the completed plan that originally mandated
  `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope` and its ISSUE-091/ISSUE-093 citation. Not
  edited by this filing or by any resolution of this question; completed plans are never
  rewritten.
- **ISSUE-152** (`issues/ISSUE-152-ci-blocking-job-scope-policy-after-issue-091.issue.md`) —
  sibling filing, the identical class of question for the `--all` flag (triggered by ISSUE-091's
  fix). Explicitly declined to own the `--file` half, which is what this issue files. Both pin the
  same mandated test and should be decided together or in sequence, but are independent
  founder-level calls.
- **DIR-032** — Gate Verdict Honesty (`directives/DIR-032-gate-verdict-honesty.directive.md`) —
  ISSUE-152's home directive; likely home for this sibling filing too, pending backlog-pm triage.
