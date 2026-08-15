---
title: "agent-guard's implementer allow-list is Go-only, contradicting the project's own zero-baked-language first principle"
schema_version: issue/v1

issue:
  id: ISSUE-127
  title: "agent-guard's implementer allow-list is Go-only, contradicting the project's own zero-baked-language first principle"
  type: technical-debt
  status: open
  created: "2026-08-15"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# agent-guard's implementer allow-list is Go-only

## Problem

`.claude/hooks/backstop-agent-guard.sh`'s Write/Edit `case "$AGENT_NAME" in ... esac` block has an
`implementer*` arm that hardcodes a single language's file surface:

```sh
implementer*)
  [[ "$FILE_PATH" == *.go ]] && exit 0
  basename="$(basename "$FILE_PATH")"
  [[ "$basename" == "go.mod" || "$basename" == "go.sum" || "$basename" == "Makefile" ]] && exit 0
  if [[ "$FILE_PATH" == *.json || "$FILE_PATH" == *.yml ]]; then
    [[ "$FILE_PATH" == .backstop/* || "$FILE_PATH" == artifacts/* ]] && exit 0
  fi
  wblock
  ;;
```

An implementer agent may write: files ending in `.go`; files named exactly `go.mod`, `go.sum`, or
`Makefile`; and `.json`/`.yml` files under `.backstop/` or `artifacts/`. There is no other path. An
implementer executing a plan that calls for touching a shell script, a Python file, a Rust file, a
TypeScript file, or a plain config file outside those two allowed directories is blocked by
construction, regardless of what the plan actually specifies.

## Why this matters

This project's own `CLAUDE.md` states as its FIRST, superseding principle: "Thin executor: ZERO
baked language/tool knowledge, for ANY language... a baked Go path AND a baked TypeScript path are
BOTH violations... never re-litigate this or ask whether a legacy/baked check 'stays'; only ask how/
when to migrate it." `backstop-agent-guard.sh` is harness tooling, not backstop-core's own product
code — but it still governs how this very project builds itself, and its `implementer*` arm
currently bakes Go as the only language an implementer can touch. That is exactly the shape of
defect the project's first principle names as a defect to eradicate, never to extend.

## Status: latent, not actively firing

This does not block anything today. backstop-core is presently a 100% Go project (module code under
`pkg/`, `cmd/`), so every real implementer write so far has been a `.go` file, `go.mod`/`go.sum`, a
`Makefile` edit, or a `.json`/`.yml` artifact/config write — all of which the current arm already
allows. The gap is dormant: it will fire the moment an implementer is dispatched against a plan that
touches a non-Go file outside the two carved-out directories (a shell script under `.claude/hooks/`,
a Python tool, a TypeScript file in a future pack-adjacent surface, etc.).

## Open question — not decided here

What should the implementer allow-list generalize TO is a genuine open design question, not
something to settle unilaterally in this issue:

- Every text file (broad, but loses the "implementer only touches source/config, not artifacts"
  boundary the current allow-list enforces)?
- A configurable per-project extension/path allow-list, sourced from something in the repo (e.g.
  `backstop.yml`) rather than hardcoded in the hook?
- Some other mechanism that derives "what counts as source" from adopted packs, mirroring how the
  gate itself refuses to bake language knowledge and instead reads it from packs?

Each of these has real tradeoffs (permissiveness vs. drift risk vs. implementation cost) that a
human or a later planning pass should decide — this issue exists to name the defect and its
rationale, not to prescribe the fix.

## Notes / references

- Reported by `team-lead`, surfaced by `backlog-pm` during triage of ISSUE-126 (the agent-guard
  memory write-access gap in this same hook file) as an adjacent, unruled finding; backlog-pm does
  not file issues itself, so this was routed to issue-author.
- Same file, same case-statement mechanism as ISSUE-126 (memory write-access gap) and ISSUE-044
  (roster↔case-statement drift, closed) — three independent gaps in
  `.claude/hooks/backstop-agent-guard.sh`'s Write/Edit routing, none of which overlap in scope.
- Left unhomed to a directive, consistent with ISSUE-126's precedent (also unhomed as of this
  writing).
