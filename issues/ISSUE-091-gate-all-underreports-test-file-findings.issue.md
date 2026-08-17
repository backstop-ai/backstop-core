---
title: "Gate All Underreports Test File Findings"
schema_version: issue/v1

issue:
  id: ISSUE-091
  title: "Gate All Underreports Test File Findings"
  type: bug
  status: closed
  created: "2026-07-27"
  closed: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical

delivered_by: PLAN-ISSUE-091
---

# Gate All Underreports Test File Findings

## Resolution

`backstop gate --all` was systematically under-reporting test-file findings relative to a
diff-scoped `backstop gate` run — the reverse of the intuitive assumption that `--all`
(whole-repo) is always a strict superset of a diff-scoped run.

**Root cause:** `cmd/backstop/pack_gate.go`'s `runFindingsEngine` dispatched a
directory-scoped target (bare `projectRoot`) for `--all`, instead of the explicit file-list
dispatch the diff-scoped path already used. Semgrep's directory-walk dispatch applies its own
built-in `.semgrepignore` (which excludes test files) and resolves each rule's `paths.include`
glob against files it discovered itself; explicit-file dispatch bypasses that default ignore
set entirely. The two dispatch shapes are genuinely different code paths inside the engine, and
nothing in backstop's own dispatch verified they agreed.

**Fix:** collapsed the two-branch dispatch in `runFindingsEngine` to always pass an explicit
target list — the project root as a single-element list for a nil scope (preserving existing
nil-scope callers unchanged), or the scope's own (testdata-pruned) file list otherwise. One
dispatch shape now serves every non-project-wide engine, regardless of pack or tool. Landed at
HEAD f97b3c6; verified directly against that commit (not inferred from plan status) — the
mandated tests (`TestIssue091_AllScopeDispatchesExplicitFileListNotProjectRoot`,
`TestIssue091_AllScopeExcludesTestdataPaths`, `TestIssue091_AllScopeEmptyFileListAppendsNoTarget`,
`TestIssue091_NilScopeStillTargetsProjectRoot`, `TestIssue091_AllScopeReportsTestFileFindings_RealEngine`)
all pass at that HEAD.

**Verification:** this fix's own final `gate --all` measurement went from 348 to 392 violations
post-fix — an INCREASE that is explicitly the fix working correctly (previously-hidden
test-file findings are now correctly visible), not a regression.

**Newly-visible debt:** the repo-wide findings this fix exposes are routed into
`issues/ISSUE-094-ci-repo-wide-sweep-debt-inventory.issue.md`'s existing rolling-inventory
mandate rather than a new tracker — ISSUE-094 names closing this issue as its own explicit
retirement trigger for switching off its hand-baked whole-repo sweep.

**Follow-on issues filed during this fix** (per CLM-009, "the consequences are filed, not
absorbed"):
- `ISSUE-149` — PLAN-ISSUE-010's superseded `--all`-gains-only CLM-004 prose.
- `ISSUE-150` — `--all` no longer reports `testdata/`-scoped findings (deliberate, subtractive
  behavior change: `excludeTestdataPaths` now applies to both scopes, not just diff).
- `ISSUE-151` — the "THIRD DIVERGENCE": a pack rule whose `paths.include` names a
  directory-prefixed glob fires under directory dispatch and goes dark under explicit-file
  dispatch — a pre-existing blind spot on the diff-scoped path that this fix now extends to
  `--all` as well.
- `ISSUE-152` — filed, deliberately unresolved: whether CI's blocking job may now safely use
  `--all`, since this fix removes the sole stated justification for banning it there. A
  founder-level policy call, not an implementer decision.
- Evidence appended to the pre-existing `ISSUE-097` (unbound backstop-self waiver tokens):
  the two stale `no-structural-name-split-on-spine` waiver tokens were failing open under
  `--all`'s prior directory dispatch (visible only via the "unused/dangling" waiver clause);
  post-fix they go dark entirely rather than being cleaned up, per ISSUE-097's own scope.
  Measured 2→0 confirmation plus a third, previously-invisible stale token, appended in a
  follow-up commit (6f16e00).

## Problem

`gate --all` is not a superset of the diff-scoped gate — it silently under-reports findings on
test files, in the direction opposite ISSUE-070 (diff scope leaking project-wide lint; closed).
Here `--all`, which reads as "everything," silently isn't.

Measured by implementer-087 during PLAN-ISSUE-087 TASK-016 (2026-07-28), comparing two gate runs
at the SAME HEAD: the diff-scoped gate reported 124 findings `--all` did not; `--all` reported 61
findings diff-scope did not; only 22 findings were shared between the two runs. The all-only side
is explained (files outside diff scope legitimately don't appear under the diff-scoped run). The
diff-only side is NOT explained: 111 of the diff-only findings sit on files `--all` reported ZERO
findings for. Concrete falsifier: `cmd/backstop/artifact_new_test.go` has five confirmed
`code, _ :=` sites; the diff-scoped gate reports 4, `--all` reports none. Repo-wide, `--all` cited
only 37 distinct files despite `go-standards`/semgrep rules like `*_test.go`-scoped rules applying
far more broadly. The gap concentrates in `go-standards`/semgrep findings over `*_test.go` files.

### Consequence (already realized)

PLAN-ISSUE-087's TASK-004 method ("intersect `gate --all` with the swept files") sized a founder
scope ruling at ~31 violations when the enforcing (diff-scoped) truth was 153 rows. A founder made
a scope decision on an undercount produced by the gate itself, in the gate's own full-scope mode —
this is the vacuous-green class of defect, just surfacing in `--all` rather than in a check that
reports zero findings outright.

## Root cause

Confirmed by reading `cmd/backstop/pack_gate.go` (the rule-fed/config-file engine dispatch inside
`dispatchPackEngines`, around lines 622-646) — this, not `pkg/gate/scope.go`, is where the file-set
divergence actually happens. `pkg/gate/scope.go`'s `resolveGateScopeAll`/`resolveGateScopeDiff`
both correctly enumerate their respective file lists; the divergence is in how those lists get
handed to each findings engine:

- Diff scope (`scope != nil && scope.Mode != GateScopeModeAll`, `pack_gate.go:634-645`): the
  engine is invoked with an EXPLICIT list of changed files —
  `cmdArgs = append(cmdArgs, excludeTestdataPaths(scope.Files)...)`.
- All scope (`scope == nil || scope.Mode == gate.GateScopeModeAll`, `pack_gate.go:632-633`): the
  engine is invoked with the BARE `projectRoot` DIRECTORY as its single scan target —
  `cmdArgs = append(cmdArgs, projectRoot)` — leaving the engine (semgrep, for the `go-standards`
  pack) to do its own recursive directory walk and its own file discovery/`.gitignore`/default-
  ignore resolution, then apply each rule's `paths.include` glob (e.g.
  `.backstop/packs/backstop/go-standards/rules/test/go-test.yml`'s `paths: include: "*_test.go"`)
  against files it discovered ITSELF, rather than against files it was handed directly. These are
  two different code paths inside semgrep, and nothing in backstop's own dispatch verifies they
  agree. The falsifier (`artifact_new_test.go` matches under explicit-file dispatch, not under
  directory-walk dispatch) is consistent with this: semgrep's own directory-based file discovery
  and `paths.include` glob resolution is the suspected proximate mechanism, though the semgrep-
  internal reason those two matching paths disagree has not been isolated to a specific semgrep
  version behavior — that isolation is scoped to whichever plan fixes this.

This is the same `ScopeKind != ProjectWide` branch documented at `pack_gate.go:593-599` (SPEC-034
REQ-010/CLM-034, Ratified Design Constraint 3) as the deliberate "rule-fed engines get a scan
target" path — the constraint that a target must be appended is intact; what's unverified is
whether "the whole project root as one directory" and "every file the diff would have listed"
produce the same finding set from the underlying engine.

## Direction (to be specified)

Do not accept "scan `projectRoot` as one directory" as equivalent to "scan every file in scope"
for a rule-fed engine without proof. Options for the eventual plan to weigh: (a) under `--all`,
enumerate the full project file list the same way diff scope does (mirroring
`resolveGateScopeAll`'s own `filepath.Walk`) and pass that explicit list to the engine instead of
the bare directory, so both scopes exercise the identical semgrep code path; or (b) if a directory
target is kept for performance/engine-native-discovery reasons, add a reconciliation check that a
directory-target run and an explicit-file-list run produce the same finding set on a known fixture,
so a future engine-behavior drift fails loud instead of silently under-reporting again. Whichever
direction is chosen, the fix must be provable against the `artifact_new_test.go` falsifier: `--all`
must report the same 5 `code, _ :=` findings the diff-scoped run does when that file is in scope.

## Notes / references

- Reported by team-lead via implementer-087's evidence from PLAN-ISSUE-087 TASK-016 (2026-07-28).
- Inverse direction of ISSUE-070 (closed): ISSUE-070 was diff scope leaking project-wide findings
  it should have filtered OUT; this is `--all` failing to include findings it should report.
- Sibling to the gate-verdict-honesty cluster: ISSUE-066 (narrow `-run` filter silently bounds what
  must pass) and ISSUE-067 (test failures surface as an opaque crash, not a finding) are different
  defects but the same class — a gate signal that reads as complete/authoritative and silently
  isn't. This issue strengthens that cluster's case for a shared directive/PM home rather than
  three independently-triaged issues.
- Severity is `critical` (not `moderate`): the gate's `--all` mode is the one non-diff-scoped
  ground truth available for scope/prioritization decisions, and it already produced a wrong
  founder ruling once (see Consequence above) before this defect was named.
