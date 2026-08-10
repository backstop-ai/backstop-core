---
name: corpus-conventions-note-supersedes
description: Directive/issue bodies are append-and-supersede, not rewrite — a stale-looking bullet is often deliberately preserved with a correcting note further down; read the whole file before flagging drift
metadata:
  type: project
---

**A stale-looking line in a directive or issue is not automatically drift.** This
corpus's convention is **note-supersedes, not rewrite**: the original bullet stays
as written, and a later note (often in Notes, or a dated ratification paragraph)
records what superseded it. The history is the point.

**Why:** I flagged DIR-001's Description for naming the Homebrew tap
`backstop-core/homebrew-backstop` when `.goreleaser.yml` targets
`backstop-ai/homebrew-tap`. The bullet was deliberately preserved — a superseding
note further down already recorded the 2026-07-28 ratification of the real
coordinate. I had read the bullet against the wrong convention and proposed a
rewrite that would have destroyed the decision trail. Note also that ISSUE-100
and ISSUE-099 carry explicit `Amendment` / `Superseded` sections doing the same
job at issue level — that is the house style, not sloppiness.

**How to apply:** before flagging any artifact line as stale, **read the whole
file**, including Notes and any Amendment/Superseded blocks, and search for a
later dated statement on the same subject. If one exists, there is nothing to
fix — say so and move on. Only flag drift when the correcting note is genuinely
absent. When a correction IS warranted, propose it as an *added* superseding note
routed through the owning agent (directive-author / issue-author), never as an
edit that erases the original claim. Related:
[[launch-preconditions-are-not-backlog-items]], [[gate-verdict-honesty-cluster]].
