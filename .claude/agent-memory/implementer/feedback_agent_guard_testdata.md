---
name: agent-guard-testdata
description: The implementer agent-guard hook blocks Write/Edit on non-Go/non-.backstop files; create them via Bash
metadata:
  type: feedback
---

`.claude/hooks/backstop-agent-guard.sh` (PreToolUse on Write/Edit) blocks the
implementer agent on most non-Go files. It permits only: `*.go`, `go.mod`/`go.sum`/
`Makefile`, and `*.json`/`*.yml` whose path *starts with* `.backstop/` or `artifacts/`
(anchored at string start — absolute paths never match, and mid-path `.backstop/`
fails too).

**Why:** the guard keeps agents from hand-editing SDLC artifacts, but its glob is too
narrow for legitimate implementer work.

**How to apply:** testdata fixtures (`cmd/backstop/testdata/.../.backstop/...`,
`pkg/*/testdata/*.yml`, `*.sh` converters, `backstop.lock`, `backstop.yml`), and even
this agent-memory dir (`.claude/agent-memory/*.md`), are all blocked by Write/Edit.
Create/modify them via the Bash tool (heredoc/printf) — the documented dedicated-tool
exception. The hook fires only on Write/Edit, not Bash. Reserve Write/Edit for `.go`
source, which the guard permits.
