---
title: "Artifact New Issue Id Collision On Fallback Path"
schema_version: issue/v1

issue:
  id: ISSUE-167
  title: "Artifact New Issue Id Collision On Fallback Path"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Artifact New Issue Id Collision On Fallback Path

## Problem

`backstop artifact new issue --slug <x>` reserves new issue IDs primarily via pushed git tags
(`backstop/issue/<N>`, resolved by `GitTagResolver.Resolve` in `pkg/scaffold/idresolver.go`),
with a local-filesystem-scan fallback (`LocalScanResolver.Resolve`) used when
`GitTagResolver.Resolve` returns a `*FallbackError`. Tonight, running this command three times
in quick sequence — one call at a time, in one terminal, filing what became ISSUE-165/166/168
plus this issue itself — produced two separate ID collisions in ordinary sequential invocations,
not concurrent ones.

Ledger-numbered artifacts are meant to never collide by design: per this repo's own working
invariants, "Ledger-numbered artifacts only. New artifacts via `backstop artifact new <type>
--slug <kebab>` — the CLI assigns the next number. Never hand-number, never suffix." This filing
records the observed sequence precisely as evidence; per the instructions this issue was filed
under, it deliberately does not speculate about the exact code-level root cause beyond what
follows — that requires real investigation in `pkg/scaffold/idresolver.go`.

### The observed sequence, in order

1. **First call** (in a batch of three, run for what became ISSUE-165/166/168): resolved to ID
   **164** — already claimed by an existing, git-committed issue file
   (`issues/ISSUE-164-packval-importing-packages-missing-testmain-guard.issue.md`, committed at
   `8b3b3d8`, confirmed via `git log`) — even though a `backstop/issue/164` git tag already
   existed on the local repo (from when that issue was originally created), which should have
   caused a tag-conflict retry rather than a reuse.
2. **Second call** (same batch): correctly resolved to **165**, with a real tag push confirmed
   (`backstop/issue/165` present in `git tag -l`).
3. **Third call** (same batch): correctly resolved to **166**, with a real tag push confirmed
   (`backstop/issue/166` present in `git tag -l`).
4. **Later, separate call** (for filing this very meta-issue): resolved to **167** — but 167 had
   JUST been manually assigned moments earlier, by hand (not via the tool), to what became the
   `/dev/null`-write issue. A `backstop/issue/167` git tag WAS pushed for this (second, colliding)
   call, confirmed present in `git tag -l` — suggesting the tag-based reservation path itself ran
   and completed for that call, while something about the numbering it computed, or its timing
   relative to the hand-assignment, raced against it.

Net effect: at least one confirmed collision (step 1, reusing an already-committed ID), and a
second collision-shaped event (step 4, reusing a moments-old hand-assignment) in a short sequence
of ordinary, one-at-a-time, sequential invocations — no parallel or concurrent calls were made.

### What is, and is not, known

**Known:** the tag `backstop/issue/164` existed on the local git repo before the colliding first
call ran (step 1). `git tag -l "backstop/issue/*"` currently lists a contiguous run from 158
through 167 with no gaps, meaning every number in that range now has SOME tag on it — but that
does not establish which calls created which tags in which order relative to the collisions
described above.

**Explicitly not known, and not traced tonight:** whether this is a tag-vs-local-scan race
(e.g. `GitTagResolver.Resolve`'s `FetchTags`/`ListTags` step not seeing a tag that exists only
locally, undercounting `maxNum`, and falling through to a state where the subsequent
`CreateAnnotatedTag`/`PushTag` either silently succeeds against an already-existing local tag or
is not correctly classified as a `*TagConflictError` and retried); a bug specific to the
local-scan fallback path (`LocalScanResolver.Resolve`, which counts committed files matching
`^ISSUE-(\d+)` in the target directory and would undercount if invoked before a colliding file
existed on disk); a timing/ordering issue between when a file is committed vs. when its tag is
visible to a subsequent `ListTags` call; or something else entirely. None of these should be
read as the diagnosis — this is the open surface for a plan to investigate, not a conclusion.

## Impact

The core guarantee ledger-numbered artifacts are supposed to provide — IDs assigned serially by
the CLI, never colliding, never hand-numbered — was violated twice in one ordinary working
session. A collision that silently reuses an already-committed ID risks two artifacts occupying
the same identifier (were it not caught before commit), which would break traceability
(`REQ`/`CLM` references, `resolved_by`/`delivered_by` pointers, cross-artifact links) for
whichever artifact loses the naming conflict. This is a tooling-integrity issue for backstop's
own artifact workflow, not a downstream consumer defect.

## References

- `pkg/scaffold/idresolver.go` — the ID resolution mechanism this issue is about.
  - `GitTagResolver.Resolve` (`idresolver.go:83-165`) — primary path: `FetchTags` →
    `ListTags(pattern + "*")` → compute `maxNum` from existing tag numbers → loop
    create-annotated-tag + push, retrying on `*TagConflictError` up to `maxRetries` (default 3,
    `idresolver.go:217-219`).
  - `LocalScanResolver.Resolve` (`idresolver.go:171-211`) — fallback path: scans
    `targetDir` for filenames matching `^<PREFIX>-(\d+)`, takes the max, returns max+1.
  - `ResolveID` (`idresolver.go:216-245`) — orchestrates the two: tries `GitTagResolver` first,
    falls back to `LocalScanResolver` only on `*FallbackError`; a `*RetriesExhaustedError` is
    returned directly, NOT eligible for fallback.
- `issues/ISSUE-164-packval-importing-packages-missing-testmain-guard.issue.md` — the
  already-committed issue (commit `8b3b3d8`) whose ID was reused in step 1 above.
- `git tag -l "backstop/issue/*"` — confirms tags 158 through 167 all currently present with no
  gap, consistent with (but not proof of) the sequence described.
- CLAUDE.md (global, backstop working invariants) — "Ledger-numbered artifacts only... Never
  hand-number, never suffix" — the invariant this collision violates.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for
"idresolver", "id collision", and "artifact new... collision" matched `ISSUE-090` ("Id Resolver
Max Tags And Disk" — a different, already-filed concern about `idresolver.go`'s tag/disk
interaction at scale, not this specific sequential-collision observation), `ISSUE-016` (a
resolve-ID contract issue in a different file, unrelated), `ISSUE-125` (an unrelated
constructor-injection false-positive issue), and `BUNDLE-015`/`BUNDLE-003` (pack-scaffolding and
onboarding bundle charters, neither of which owns this specific defect). None duplicate this
issue's surface, though `ISSUE-090` is adjacent enough that its owner may want to review both
together.
