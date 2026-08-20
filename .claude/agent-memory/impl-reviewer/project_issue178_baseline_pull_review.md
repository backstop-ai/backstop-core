---
name: issue178-baseline-pull-review
description: ISSUE-178 baseline-pull workflow-name filter review — PASS; live end-to-end proof recipe (gh tracing shim + pre-fix binary) and the go-arch-lint PATH gotcha
metadata:
  type: project
---

ISSUE-178 (`resolveLatestSuccessfulMainRun` unfiltered on workflow name) reviewed PASS.
Two reusable techniques came out of it.

**Live end-to-end proof for a CLI that talks to an external tool.** When the command is
silent on success and you need to know *what it actually did*, put a tracing shim first on
PATH that logs its args and `exec`s the real tool, then run BOTH binaries — the fixed one
built from the working tree and a mutated pre-fix one built from an rsync scratchpad copy —
against the real remote. Here that produced the whole verdict in two runs: pre-fix selected
the Pages run and failed `missing artifact`, post-fix selected the CI run and wrote a valid
`.backstop/baseline.json`. Build review binaries into the scratchpad, never over `bin/backstop`,
when other agents share the tree.

**Why:** unit tests over a fake `gh` cannot distinguish "selects the right run" from "selects
the run my fixture happened to put first"; only the real API answers that.

**How to apply:** any lane touching `baseline pull`, `pack add`, or another `exec`-shelling
path. `.backstop/baseline.json` is gitignored, so re-pulling it is a safe, reversible side effect.

**`go-arch-lint` lives in `$(go env GOPATH)/bin`, not on the default sandbox PATH.** Without it
`backstop gate` exits 2 at `pack_engines` with a Layer-0 tool-missing error and every step after
it is unmeasured. Export `PATH="$(go env GOPATH)/bin:$PATH"` before any gate run. This corrects
the pessimistic note in [[issue163-sandbox-guard-review]] that missing go-arch-lint makes local
gate readings non-authoritative — it is a PATH gap, not an absent tool.

Attribution baseline for this tree (measured 2026-08-20): diff-scoped gate = 1 blocking violation,
`TestPackAuthoringLoop_EndToEnd` (ISSUE-162, darwin-only, phase3-fixtures/validator-positive);
`--all` = 120 blocking, all inherited. Reproduce a suspect red against an rsync copy with the
three changed files restored via `git show HEAD:<path>` — cheaper and safer than a worktree or
a stash in a shared tree.

Related: [[issue116-line-carry-pass]], [[issue082-allowlist-review]].
