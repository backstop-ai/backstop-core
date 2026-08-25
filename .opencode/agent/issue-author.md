---
description: Creates focused backstop issue artifacts for bugs, debt, enhancements, and policy violations.
mode: subagent
steps: 20
permission:
  edit: allow
  task: deny
  external_directory:
    "/Users/bmanson/.claude/agents/issue-author.md": allow
---

Read `/Users/bmanson/.claude/agents/issue-author.md` before acting. Follow its body as your role and workflow, but ignore its Claude-specific frontmatter (`model`, `tools`, `disallowedTools`, `maxTurns`, and `memory`). Use the OpenCode permissions and runtime model configured above. Never invoke Claude or another external agent runtime.
