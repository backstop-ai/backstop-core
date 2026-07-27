---
title: "ID resolver allocates from git tags only — fallback (no-remote) allocations are never reconciled against the max, so tags can re-issue a number already used on disk"
schema_version: issue/v1

issue:
  id: ISSUE-090
  title: "ID resolver allocates from git tags only — fallback (no-remote) allocations are never reconciled against the max, so tags can re-issue a number already used on disk"
  type: bug
  status: open
  created: "2026-07-27"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# ID resolver allocates from git tags only — fallback (no-remote) allocations are never reconciled against the max, so tags can re-issue a number already used on disk

## Problem

`scaffold.ResolveID` (`pkg/scaffold/idresolver.go:200`) tries `GitTagResolver.Resolve`
(`idresolver.go:73`) first and falls back to `LocalScanResolver.Resolve` (`idresolver.go:158`,
a plain `os.ReadDir` scan of the artifact directory) whenever the git path returns a
`*FallbackError` — which includes `!IsGitAvailable() || !IsGitRepo()` (`idresolver.go:74-76`)
and, critically, `FetchTags()` failing (`idresolver.go:78-80`). `RealGitExecutor.FetchTags`
(`git_executor_real.go:43-48`) just runs `git fetch --tags`, which fails whenever the repo has
no configured remote — this repo's actual state until launch (see MEMORY:
[[project_launch_plan]]).

The two resolvers are not reconciled with each other. `GitTagResolver.Resolve` computes
`nextNum` as `max(numeric suffix of backstop/<type>/NNN tags)+1` and, on success, creates and
pushes an annotated tag reserving that number (`idresolver.go:90-135`). `LocalScanResolver`
independently computes `max(numeric ID parsed from filenames in the target dir)+1`
(`idresolver.go:175-194`) and returns it with **no tag created** — filesystem-scan allocations
leave no reservation record. Each call to `ResolveID` picks one path or the other in isolation;
neither consults the other's view of the max.

Consequence: every artifact created during a no-remote (or otherwise git-failing) window gets a
number from the filesystem-scan max with no backing tag. When the remote is later configured and
`FetchTags` starts succeeding again, `GitTagResolver` resumes computing its next number purely
from `max(tag)+1` — a number that is unaware of any higher numbers already consumed by untagged,
fallback-created files. The tag path can then re-issue an ID that already names a file on disk,
producing two different artifacts under the same ID.

### Evidence (PM-verified 2026-07-27, orchestrator-remediated)

- `ISSUE-089` was committed with no reservation tag — the next issue created after a remote
  lands and `FetchTags` starts succeeding would have collided with it (tag-side max would not
  have accounted for 089).
- The `adrs/` directory had 18 files and **zero** reservation tags — `ADR-001` would have been
  re-issued immediately on the next tag-path allocation.
- `plans/` has 102 tags with 8 gaps below the current max of 110 and non-numeric filenames,
  which made a file-to-number audit impossible — this fix removes the need for that audit going
  forward, since the resolver itself would keep tag-max and disk-max reconciled.

### Interim remediation already applied (2026-07-27, does not fix the resolver)

All missing reservation tags were backfilled: `issue/089` and `adr/001`–`adr/018`, annotated,
each pointing at the file's introducing commit. Burnt tags whose number exceeds the current
max-on-disk (e.g. `spec/057`–`spec/061`, pointing at unrelated commits) were left in place
deliberately — a burnt tag still blocks re-issuance of that number, so it is harmless; only a
file *without* a tag is at risk of collision. This backfill closes the immediate collision risk
for the numbers that existed at backfill time, but does nothing to prevent the same lag from
reappearing the next time git ops fail (no remote, offline, auth failure, etc.) — that
recurrence risk is what this issue tracks.

### New evidence — display/reservation split within one call (2026-07-27 ~23:10)

`backstop artifact new plan --slug pack-distribution-content-identity --source SPEC-057`
printed `Created … (ID: 001)`, yet the durable outcomes were correct: the frontmatter is keyed
`plan_id: PLAN-SPEC-057` (plans carry no bare numeric ID — verified in
`plans/PLAN-SPEC-057-pack-distribution-content-identity.plan.yml`), and reservation tag
`backstop/plan/111` exists at HEAD (verified via `git tag --points-at`; `git tag -l
'backstop/plan/*'` shows the prior max was 110, so the tag path allocated correctly). Tracing
the code (not just the symptom) shows this is not two independently-diverging resolvers as
first suspected, but a single `ResolveID` call undermining its own successful reservation:
`GitTagResolver.Resolve` runs `git tag -a` to create the local annotated tag *before* attempting
`git push origin <tag>` (`idresolver.go:114-123`); this repo currently has no `origin` remote
configured (`git remote` is empty), so the push fails non-conflict and is wrapped as a
`FallbackError` (`idresolver.go:147-148`), which makes `ResolveID` discard the just-created
"111" tag's ID and fall through to `LocalScanResolver` — but that tag was already written to
`.git/refs` and persists as `backstop/plan/111` regardless. `LocalScanResolver` then computes
its max from filenames matching `^PLAN-(\d+)` (`idresolver.go:172-173`), but every real plan
filename is `PLAN-SPEC-NNN-...` or `PLAN-ISSUE-NNN-...` (`scaffold.go:107-115`) — the digit
group never follows `PLAN-` directly — so this regex can never match an existing plan file,
`maxNum` is always 0, and the fallback always prints `001` for the plan type no matter how many
plans already exist on disk. Compounding this, `Filename()` and `Scaffold()`'s `"plan"` case
(`scaffold.go:100-120,154-167`) never use the resolved `id` for the plan's actual filename or
frontmatter at all — for plans, the resolved numeric ID governs only the git tag and the printed
CLI message, so its correctness is externally unverifiable from the artifact itself. Two
consequences for scoping the fix: (1) the printed ID, the created tag, and any filename number
must all trace to one resolution outcome — a resolver must not create a durable local
reservation and then report a different, unrelated number as "the" ID; and (2) for artifact
types whose filenames don't carry a bare numeric ID (plan is the only one today —
`Filename()`/`Scaffold()` key plans off `sourceID`, not `id`), `LocalScanResolver`'s
`^PREFIX-(\d+)` filename scan is not just lagging but structurally incapable of ever finding the
true max, so a real fallback for plans would collide at "001" on every single invocation, not
merely on the first one after a gap — this is a strictly worse case than the general
tag-vs-disk drift already described above and should be covered by the same fix's acceptance
criteria, not treated as a separate follow-on.

## Expected

`ResolveID` (or `GitTagResolver.Resolve` directly) computes the next number from
`max(git tag scan, local filesystem scan)`, not from either resolver's view alone, so a
fallback (untagged) allocation can never later be re-issued once the tag path reactivates.
Whether that means `GitTagResolver` consults `LocalScanResolver`'s max before allocating (both
on the tag-success path and to decide `nextNum` before the retry loop), or `ResolveID` itself
merges both maxes before delegating, is an implementation choice for the plan — the invariant is
that the two allocation paths must never disagree about the current max.

Secondary, lower-priority hardening worth scoping in the same plan: when the tag path succeeds
after a prior fallback-created gap, backfill the missing tag(s) opportunistically so the git
history converges back to "every file has a reservation tag" without requiring another manual
audit-and-backfill pass.

## References

- `pkg/scaffold/idresolver.go:73` — `GitTagResolver.Resolve`, tag-only max computation
- `pkg/scaffold/idresolver.go:78-80` — `FetchTags` failure triggers `FallbackError`
- `pkg/scaffold/idresolver.go:158-194` — `LocalScanResolver.Resolve`, filesystem-only max
  computation, no tag created on allocation
- `pkg/scaffold/idresolver.go:200-230` — `ResolveID`, the two-path dispatcher (tries git, falls
  back to local scan on `*FallbackError`; does not merge the two maxes)
- `pkg/scaffold/git_executor_real.go:43-48` — `RealGitExecutor.FetchTags`, `git fetch --tags`,
  fails whenever no remote is configured
- `cmd/backstop/artifact_new.go:74` — call site, `scaffold.ResolveID(artifactType, ...)`
- Home: `DIR-001` (`directives/DIR-001-release-workflow.directive.md`) — the triggering event is
  literally "add the remote"; PM recommendation, founder-ratified 2026-07-26/27 in the
  "backfill now, fix resolver before it recurs" ruling
- Track: issue → plan (not a bundle-track change; contained fix to one resolution function).
  Not launch-blocking now that the tag backfill has landed — should land before any future
  extended no-remote or git-failure development window recurs.
