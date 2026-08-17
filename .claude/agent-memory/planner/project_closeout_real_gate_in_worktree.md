---
name: closeout-real-gate-in-worktree
description: A detached worktree lets a close-out run the real full suite and `gate --all` without racing live lanes — but needs .backstop/packs copied in and go-arch-lint on PATH, or the reading is silently degraded
metadata:
  type: project
---

A close-out CAN take its own full-suite and `gate --all` readings instead of carrying the
implementer's, by working in a detached worktree at the closing revision
(`git worktree add <scratch> <sha> --detach`). Two setup steps are mandatory or the reading
is degraded in ways that look like clean results:

- **Copy `.backstop/packs` in from the main tree.** It is gitignored, so a fresh checkout has
  none, and everything pack-driven fails with "declared pack ... is missing" — noise that
  masks real failures. Watch the copy target: if the worktree already has a `.backstop/`,
  `cp -R .../.backstop <wt>/.backstop` NESTS it as `.backstop/.backstop`.
- **Put `go-arch-lint` on PATH** (`PATH="$HOME/go/bin:$PATH"`). It is an assume-present
  Layer-0 tool backstop never auto-provisions; without it `pack_engines` dies in ~2ms with a
  single tool-not-found violation, which reads like a near-clean gate.

**Confirm the reading was real before quoting it:** `pack_engines` should show a multi-minute
duration and a large violation count. A fast `pack_engines` means the engines never ran.

**Redirect `gate --all` to a file** — backgrounded output gets truncated to the tail, and the
per-step summary lives at the top. Then attribute by grepping the output for the lane's OWN
file paths; a repo-wide RED is expected and is not the lane's.

**Why:** [[project_plan_closeout_convention]] says not to race live implementers with a full
suite or `gate --all`, and prescribes carried readings as the honest substitute. A worktree
dissolves the tradeoff — no race AND a first-hand reading. On PLAN-ISSUE-067 (2026-08-16) this
converted three carried claims into measured ones and answered a question the hand-off could
not: 40 `error-wrapping-required` findings existed repo-wide and ZERO named the lane's files.

**How to apply:** when a hand-off asks you to verify a residual "against real gate output," or
whenever a carried number is load-bearing. Pair it with a CONTROL run at the commit before the
lane's first code landed — an identical failure set there is what proves pre-existence rather
than asserting it ([[project_inherited_coverage_red_at_closeout]] is the same move for
coverage).
