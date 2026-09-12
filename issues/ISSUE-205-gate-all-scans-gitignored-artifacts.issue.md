---
title: "Gate All Scans Gitignored Artifacts"
schema_version: issue/v1

issue:
  id: ISSUE-205
  title: "Gate All Scans Gitignored Artifacts"
  type: bug
  status: open
  created: "2026-09-12"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Gate All Scans Gitignored Artifacts

## Problem

`backstop gate --all` builds its file set by walking the raw filesystem, not the
git index, so files inside a **gitignored** directory get fed to pack engines as
if they were real project sources. Diff-scoped `backstop gate` (the default,
no flags) does not have this defect — the two modes disagree, and only one of
them is correct.

### Where this lives

`pkg/gate/scope.go` has two scope resolvers:

- `resolveGateScopeDiff` (`pkg/gate/scope.go:219`) computes the diff-mode file
  set as `git diff --name-only <base>` (tracked changes) plus
  `git ls-files --others --exclude-standard` (untracked files) — at
  `pkg/gate/scope.go:230` and again at `:244`. `--exclude-standard` is git's own
  flag for "honor `.gitignore`/`.git/info/exclude`/core.excludesFile," so
  untracked-but-ignored files are excluded from the diff scope by construction.
- `resolveGateScopeAll` (`pkg/gate/scope.go:253-268`) computes the `--all` file
  set with `filepath.Walk(projectRoot, ...)`. The only directory it skips is one
  whose name starts with `.` (`pkg/gate/scope.go:259`, e.g. `.git`, `.backstop`).
  It never reads `.gitignore`, never shells out to `git`, and never calls
  `resolveGateScopeDiff`'s exclusion logic. Every other file under the tree —
  gitignored or not — is appended to the scan list and later dispatched to pack
  engines (`cmd/backstop/pack_gate.go`'s findings-engine dispatch, downstream of
  this scope).

So the two gate entry points are inconsistent on the exact same input tree:
bare `backstop gate` (diff mode) is `.gitignore`-aware via git plumbing;
`backstop gate --all` is not, and walks everything except dot-directories.

### Evidence (measured today, consumer repo `bclabs-brief-builder`)

- That repo's `.gitignore:58` lists `test-results/`.
- A local Playwright run left trace artifacts at
  `test-results/.playwright-artifacts-2/traces/resources/<sha>.css` — a
  browser-captured stylesheet snapshot, not authored code.
- The next `backstop gate --all` reported **102 violations**, the large majority
  from `bclabs-ai/bclabs-design-system`'s `typography-restraint` and
  `palette-adherence` rules, firing against that one generated CSS file (raw hex
  literals, disallowed font faces — exactly the kind of finding those rules
  exist to catch in real authored CSS, misfiring on a throwaway trace asset).
- `rm -rf test-results` made the gate clean again — the tree's actual tracked
  and untracked-not-ignored content had no violations.
- Full gate output captured at time of observation (2026-09-12): see
  `pack_engines` step, 102 blocking findings, all against paths under
  `test-results/.playwright-artifacts-2/traces/resources/...`.

This is not specific to Playwright or to this one pack — any gitignored
directory holding generated, vendored, or downloaded content (build output,
`node_modules` if ever un-dot-prefixed, coverage HTML, screenshot diffs, cache
directories) will reproduce the same class of false red the moment any pack
with content-shape rules (design-system token/palette checks, secret scanners,
license linters) is adopted. `resolveGateScopeAll`'s dot-directory skip only
accidentally covers the common `.gitignore`d-and-dot-prefixed cases
(`.next`, `.turbo`); it does nothing for the equally common non-dot ones
(`test-results/`, `dist/`, `coverage/`).

### Interaction with the ratchet principle

The baseline/ratchet mechanism (`pkg/gate/baseline.go`) keys grandfathered
findings by an identity hash derived from file + rule + message
(`CompareBaseline`, `pkg/gate/baseline.go:115`). If a baseline snapshot is ever
generated from a `gate --all` run while a gitignored artifact directory exists
on disk, that directory's findings get baked into the baseline file as
grandfathered violations tied to paths that are not part of the tracked
project at all — content that can churn or disappear on the next build with no
corresponding code change. That is baseline pollution from paths that were
never meant to be measured, on top of the immediate false-red problem above.

## Direction

`resolveGateScopeAll` should enumerate files the same way git itself would
answer "what belongs to this working tree, ignoring what's ignored" — via
`git ls-files -co --exclude-standard` (`-c` cached/tracked, `-o` others,
`--exclude-standard` applying `.gitignore`/`.git/info/exclude`/
`core.excludesFile`) — instead of `filepath.Walk`. That is the same
`--exclude-standard` semantics `resolveGateScopeDiff` already relies on for its
untracked half, so this closes the gap by reusing an idiom already trusted
elsewhere in this same file rather than introducing a new one.

Considerations for whoever plans this:

- `resolveGateScopeAll` is also the **fallback** path `resolveGateScopeDiff`
  calls when the project isn't a git repository at all
  (`pkg/gate/scope.go:220-223`) or when `git diff` itself fails
  (`pkg/gate/scope.go:239-242`). In both of those cases, `git ls-files` is
  equally unavailable, so the git-backed enumeration needs its own
  `gitOK(projectRoot, "rev-parse", "--is-inside-work-tree")` guard, falling
  back to the current `filepath.Walk` behavior only when there is no git repo
  to ask.
- Keep an escape hatch for the (presumably rare) case where an operator
  genuinely wants ignored paths swept — e.g. a flag or scope-mode variant that
  opts back into the raw walk — rather than removing the walk-based path
  outright.
- This is scoped to `resolveGateScopeAll` only. `resolveGateScopeDiff` already
  behaves correctly and is not part of this issue.

## Existence-in-world Check

Searched `issues/` and `bundles/` in this repo before filing. Five closed
issues touch gate scope-filtering behavior for `--all`/diff mode (ISSUE-070
project-wide-lint leak into diff scope, ISSUE-091 `--all` under-reporting test
findings via directory-vs-file-list dispatch shape, ISSUE-150 testdata-segment
exclusion as a consequence of ISSUE-091, ISSUE-149 a stale plan-claim
correction for the same, ISSUE-136 an audit of `exempt_from_scope_filter`
flag coverage) and none of them address `.gitignore`/ignored-path filtering —
they are about scope-filter application to already-enumerated violations or
about the `--all` dispatch shape (directory target vs explicit file list), not
about what `resolveGateScopeAll` enumerates in the first place.
`bundles/BUNDLE-008-gate-diff-scope.bundle.md` (status: `delivered`) is the
bundle that established `--all` as a deliberate full-sweep "escape hatch" via
`nil`/`GateScopeModeAll` scope (REQ-002) — it does not discuss `.gitignore`
handling, and its delivery predates this defect being observed. No open issue
or bundle charter currently owns this gap.
