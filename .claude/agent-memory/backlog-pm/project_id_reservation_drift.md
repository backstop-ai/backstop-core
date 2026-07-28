---
name: id-reservation-drift
description: Artifact IDs allocate from git tags with a silent filesystem fallback; this repo has no remote, so tags (001-088) lag files (089+) and adding the launch remote will re-issue colliding IDs
metadata:
  type: project
---

Artifact ID allocation reads **git tags only** (`GitTagResolver.Resolve`,
`pkg/scaffold/idresolver.go:73` — `max(backstop/<type>/NNN) + 1`, never looks
at disk). `ChainedResolver` (`:198`) silently falls back to
`LocalScanResolver` (filesystem `ReadDir`, `:161`) whenever a git op fails,
including `FetchTags` (`:78`) and tag push (`:148`).

As of 2026-07-27 this repo has **no git remote**, so every recent scaffold has
taken the fallback path. State: 88 issue tags (001–088) vs 89+ issue files —
ISSUE-089 is committed with no reservation tag, ISSUE-090 likewise. Escalated
to Brandon 2026-07-27; no issue filed yet (recommended DIR-001 as home).

**Why:** the launch plan calls for adding a real private
`backstop-ai/backstop-core` remote. The moment that lands, `FetchTags`
succeeds, the tag path reactivates, finds max tag 088, and hands out **089** —
colliding with the committed ISSUE-089. The hazard is created by a named
launch task, so it wants fixing *before* that step. See
[[project_launch_tiering]].

**How to apply:** when triaging duplicate-ID, missing-tag, or "scaffolded in
error" artifacts, do NOT assume a missing `backstop/issue/NNN` tag means the
artifact was hand-created — the fallback path is the norm here, and tagless is
currently expected. Conversely, don't reason about ID burn/reuse from tags
alone. Verify with `git tag -l "backstop/issue/*" | wc -l` vs the file count
before making an ID-safety claim. If this memory still says "no remote" but
`git remote -v` returns something, the collision may have already fired —
check for duplicate IDs first.
