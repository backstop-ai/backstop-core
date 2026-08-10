---
name: fix-menus-overstate-core-gaps
description: Issues/plans that defer work often assert a core capability gap that does not exist — verify the premise in pkg/ before homing the ask as "core capability"
metadata:
  type: project
---

When an issue's Solution section offers a menu like "(1) do it in the pack /
(2) build the capability in core," **verify the core-gap premise before
letting it decide the home.** The menu is written by an implementer who
worked around the problem under time pressure, not by someone who surveyed
the engine surface.

Measured instance (ISSUE-109, 2026-07-29): the issue and `PLAN-ISSUE-101`
both asserted that core cannot hand an engine a correlated group of files,
so the escalation-ladder's script rung "would mean building engine machinery
for one rule." In fact `engine.ScopeKindProjectWide` and `InputModeNone`
already exist AND already dispatch in the findings path
(`cmd/backstop/pack_gate.go` branches on both) — a pack can run one
project-wide script over any files it likes and emit SARIF with zero core
change. The real blocker was governance (BUNDLE-021 OQ-3: are pack-declared
engine commands sandboxed?), not mechanism.

**Why:** the home ruling swings on it. "Core capability" routes to DIR-024 /
a new engine directive and reads as a project; "pack-side, already
dispatchable" routes to the pack's own home and reads as hours. Homing off
an unverified premise files a small job under a big directive.

**How to apply:** on any issue whose fix directions include "add a
capability to core," grep the named package for the capability first and
check whether the dispatch path already branches on it. Say in the INBOX
entry that the finding is yours and that the planner should falsify it —
don't assert it as the issue's own claim. Related:
[[project_mechanism_vs_ecosystem_gap]],
[[project_workaround_and_file_pattern]].
