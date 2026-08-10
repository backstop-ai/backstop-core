---
name: recipe-payload-sharp-edges
description: Recipe payload gotchas — the substituter reads {{ }} inside COMMENTS too, and a rule anchored on a mere string MENTION fires on planning artifacts
metadata:
  type: project
---

Two payload/rule traps measured 2026-07-29 building the go-distribution pack.

**1. The substituter does not skip comments.** `pkg/recipe/substitute.go` scans raw
text, so a `{{ .Env.X }}` written in PROSE inside a payload comment is read as a
param name and hard-fails the apply (`unresolvable placeholder`). A payload that
wants to TALK about a template must write it without delimiters (`.Env.<NAME>`).
The self-emitting pass-through param trick (name = inner text, default = the
wrapped text; values are never rescanned) covers real templates, not prose.

**2. An anchor that only checks for a MENTION fires on documents about the thing.**
A rule anchored `(?s)\A.*goreleaser/goreleaser-action.*\z` fired on backstop-core's
own PLAN ARTIFACTS — YAML that discusses release pipelines at length. Fixtures
(ci.yml, backstop.yml) did not catch it; the GATE did, on the real corpus. Fix:
require STRUCTURE alongside the mention — for a workflow, `^on:` and `^jobs:` at
column 0 — via a PCRE lookahead conjunction, which preserves the whole-file span so
a paired `pattern-not-regex` can still cancel it.
`\A..\z` anchors and lookaheads both work in semgrep pattern-regex (measured
1.156.0); prefer `\A` over `^name:` — a workflow may legally omit `name:`, and an
anchor that cannot match is a rule that silently never fires.

**Why:** both failures are silent-green shaped: one blocks the apply loudly, the
other reds innocent files while looking like a working rule.

**How to apply:** before shipping a self-scoping rule, run it against the whole
consuming repo (not just fixtures) and check WHICH files match. See
[[project_packtest_phase3_vacuous]] and [[project_pack_recipes_archetype_gate_order]].
