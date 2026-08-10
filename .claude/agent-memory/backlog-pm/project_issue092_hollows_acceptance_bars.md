---
name: issue092-hollows-acceptance-bars
description: ISSUE-092 (pack test phase-3 fixtures cannot fail) makes any "passes pack test" / "fixtures falsify" acceptance criterion vacuous — check it before accepting one
metadata:
  type: project
---

Any artifact whose acceptance bar rests on `pack test` fixture phases — "passes
`pack test` across all phases", "every negative fixture actually fails the rule
it targets" — is **satisfiable by a vacuous signal** while **ISSUE-092
"pack test phase-3 fixtures cannot fail"** (`risk: critical`, open, cited only
by DIR-024) stands.

**Why:** phase 3 executes no fixture checks for `rule_path:`-style rules —
verify by grepping `rule_path` in `pkg/packval/` (as of 2026-07-29 it appears
nowhere). A pack with fabricated or non-falsifying fixtures passes anyway.

**How to apply:** when triaging anything that proposes pack-authoring or
fleet-CI acceptance criteria, re-run that grep (don't cite this memory as
fact — 092 may have landed) and, if still live, flag the bar as vacuous in the
escalation and note that 092 gates it. Three artifacts have hit this so far:
ISSUE-096's step-3 falsification, DIR-027 thread 5's fleet-CI criterion, and
ISSUE-101's entire verification section. That recurrence is the standing
argument for promoting ISSUE-092 out of DIR-024 — see
[[project_gate_verdict_honesty_cluster]].

Related surface for pack-authoring triage: the recipe applier's own open
defects, ISSUE-080 (silent clobber of a diverged edit at exit 0) and ISSUE-081
Gap 3 (insert placement unpinned) — both DIR-019's, both routinely omitted from
issues that cite SPEC-054 as "delivered and E2E-proven."
