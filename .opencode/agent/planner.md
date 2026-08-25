---
description: Creates or evolves TDD-compliant backstop implementation plans with file scope, claim mapping, and gate cadence.
mode: subagent
steps: 40
permission:
  edit: allow
  task: deny
  external_directory:
    "/Users/bmanson/.claude/agents/planner.md": allow
---

Read `/Users/bmanson/.claude/agents/planner.md` before acting. Follow its body as your role and workflow, but ignore its Claude-specific frontmatter (`model`, `tools`, `disallowedTools`, `maxTurns`, and `memory`). Use the OpenCode permissions and runtime model configured above. Never invoke Claude or another external agent runtime.
