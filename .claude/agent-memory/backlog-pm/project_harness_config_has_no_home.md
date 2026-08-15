---
name: harness-config-has-no-home
description: The .claude/ Claude Code harness (agent-guard hook, agent definitions) has NO live owner in the corpus — DIR-016 done, SPEC-003 deprecated; harness issues are NO FIT by default
metadata:
  type: project
---

Issues about the local Claude Code harness under `.claude/` — the agent-guard hook, agent
definitions, hook wiring — have no directive home, and that is a standing state, not an
oversight to route around.

- `DIR-016 "Directive / Issue / Plan Lifecycle Hardening"` homed `ISSUE-044` (the agent-guard
  roster self-check) and its third acceptance criterion was exactly "a config self-check
  detects a missing agent-guard case." It is `status: done`, completed 2026-07-08.
- `SPEC-003 "Agent File-Type Enforcement Hooks"` is the hook's actual design owner. It is
  `status: deprecated`, and its parent `agent-definitions` bundle is `maturity: deprecated`.
- All 12 open directives charter backstop *product* surfaces (init, recipes, pack fleet,
  gate/engine quality, verdict honesty, baseline, telemetry, traceability, self-update,
  contracts, pack distribution, trace). None covers "the harness we develop in."

**Why:** established during ISSUE-126 triage (2026-08-15, agent-guard memory carve-outs).
Checking all twelve rather than the two likely ones is what surfaced that the surface itself
is orphaned, not that I'd picked wrong.

**How to apply:** classify harness-config issues NO FIT and escalate. Default recommendation
is *fix it directly as harness config, don't manufacture a directive* — Brandon's standing
call is that this Claude Code setup is a stopgap, optimize for watch/steer and cheap
guardrails, don't gold-plate; a ten-line shell edit does not earn a backlog epic. If he wants
tracking, the least-bad option is a new directive reopening DIR-016's config-self-consistency
lane. Do not slot into a product directive on adjacency. Related: the same done-directive trap
as [[project_homed_but_orphaned_bundles]] and [[project_zero_baked_violations_have_no_home]] —
always check a candidate home's *status* before calling anything homed.

Standing sub-fact worth a ruling if it ever comes up again: the guard's `implementer*` arm
allow-lists Go only (`.go`, `go.mod`, `go.sum`, `Makefile`), which sits awkwardly against the
project's zero-baked-language first principle. Escalated 2026-08-15, unruled.
