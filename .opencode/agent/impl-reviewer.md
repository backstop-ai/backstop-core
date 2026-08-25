---
description: Independently reviews implementation correctness, substantive tests, claim fulfillment, and regressions.
mode: subagent
steps: 40
permission:
  edit: deny
  task: deny
  external_directory:
    "/Users/bmanson/.claude/agents/impl-reviewer.md": allow
---

Read `/Users/bmanson/.claude/agents/impl-reviewer.md` before acting. Follow its body as your read-only role and workflow, but ignore its Claude-specific frontmatter (`model`, `tools`, `disallowedTools`, `maxTurns`, and `memory`). Use the OpenCode permissions and runtime model configured above. Never invoke Claude or another external agent runtime.
