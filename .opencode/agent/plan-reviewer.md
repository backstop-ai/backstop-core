---
description: Independently reviews backstop plans for claim coverage, TDD ordering, file scope, overlap, and gate cadence.
mode: subagent
steps: 30
permission:
  edit: deny
  task: deny
  external_directory:
    "/Users/bmanson/.claude/agents/plan-reviewer.md": allow
---

Read `/Users/bmanson/.claude/agents/plan-reviewer.md` before acting. Follow its body as your read-only role and workflow, but ignore its Claude-specific frontmatter (`model`, `tools`, `disallowedTools`, `maxTurns`, and `memory`). Use the OpenCode permissions and runtime model configured above. Never invoke Claude or another external agent runtime.
