---
title: "agent-guard's Write/Edit case statement has no memory carve-out for any author/reviewer family except backlog-pm, forcing a Bash-heredoc fallback for self-improvement notes"
schema_version: issue/v1

issue:
  id: ISSUE-126
  title: "agent-guard's Write/Edit case statement has no memory carve-out for any author/reviewer family except backlog-pm, forcing a Bash-heredoc fallback for self-improvement notes"
  type: bug
  status: closed
  created: "2026-08-15"
  closed: "2026-08-15"

resolved-by: be8d80a4145531e564ef841f4ce41340d5c1ad3d

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# agent-guard blocks memory writes for every agent family except backlog-pm

## Problem

`.claude/hooks/backstop-agent-guard.sh`'s Write/Edit `case "$AGENT_NAME" in ... esac` block grants
each author/reviewer family write access to only its own artifact-type glob (e.g. `spec-author*` →
`*.spec.md` only, `issue-author*` → `*.issue.md` only). Exactly one family, `backlog-pm*`, has an
explicit carve-out for its own memory directory:

```sh
backlog-pm*)
  # PM may write ONLY its own queue and memory — never artifacts, never BACKLOG.yml.
  case "$FILE_PATH" in
    *.backstop/pm/*|*.claude/agent-memory/backlog-pm/*) exit 0 ;;
    *) wblock ;;
  esac ;;
```

Every other family — `bundle-author`, `spec-author`, `adr-author`, `issue-author`, `planner`,
`directive-author`, `implementer`, `spec-reviewer`, `plan-reviewer`, `impl-reviewer`,
`bundle-reviewer`, `general-purpose` — has no such carve-out, even though `.claude/agent-memory/`
already has a directory for every one of those families (confirmed: `ls .claude/agent-memory/`
lists `backlog-pm`, `bundle-author`, `bundle-reviewer`, `directive-author`, `impl-reviewer`,
`implementer`, `issue-author`, `plan-reviewer`, `planner`, `spec-author`, `spec-reviewer`), and this
project actively wants agents writing self-improvement notes there — each family's own `MEMORY.md`
and `feedback_*.md`/`project_*.md` files already exist under those directories and are read back by
future runs of the same family (this agent's own system prompt cites
`.claude/agent-memory/issue-author/MEMORY.md` this way).

Concretely, `spec-reviewer*|plan-reviewer*|impl-reviewer*|bundle-reviewer*` unconditionally
`wblock` on every path with no memory exception, `implementer*` has a completely different
allow-list shape (`.go`, `go.mod`, `go.sum`, `Makefile`, plus `.json`/`.yml` under
`.backstop/`/`artifacts/`) with no memory path in it either, and `general-purpose*`'s only carve-out
is `*/prototype/*`.

## Reproduction / evidence

Hit live, 2026-08-15: a `spec-author-054-contract-fix` agent, mid-task on SPEC-054 contract work,
discovered a real, generalizable lesson — kind:function contracts are existence-only ast-grep
queries, so signature drift on a `kind: function` contract is invisible to the gate. It attempted to
write that lesson via **Write/Edit** to
`.claude/agent-memory/spec-author/feedback_kind_function_contracts_existence_only.md` and was
blocked by this hook. `spec-author*`'s case arm is `[[ "$FILE_PATH" == *.spec.md ]] && exit 0 ||
wblock` — a memory-directory path under `.claude/agent-memory/spec-author/` does not match
`*.spec.md`, so it falls straight to `wblock`.

**Correction (backlog-pm review, 2026-08-15):** the lesson was NOT lost. The agent fell back to a
Bash heredoc into the same path, which the guard's Write/Edit case statement never governs — the
guard's separate Bash-command arm restricts only artifact-file globs (`.bundle.md`, `.spec.md`,
etc.), not `.claude/agent-memory/` paths. `.claude/agent-memory/spec-author/feedback_kind_function_contracts_existence_only.md`
exists on disk, written successfully via that fallback. A repo-wide check at review time found 62
other memory files written across the supposedly-blocked agent families since the guard's last
change, confirming this fallback is routinely available and routinely used, not a lucky one-off.

## Why this matters

The real defect is narrower than first framed: a **Write/Edit-vs-Bash tool-policy inconsistency**,
not silent, unrecoverable loss. The guard blocks the direct Write/Edit call for every family except
`backlog-pm`, but a Bash heredoc into the identical `.claude/agent-memory/<family>/*` path was never
restricted — so an agent that tries Write/Edit first burns a blocked call and a retry, then either
falls back to the Bash-heredoc path (which works, if the agent thinks to try it) or hands the lesson
back to the orchestrating session as a chat message (which only durably persists if a
human/orchestrator is present to relay it). The blast radius is one blocked tool call with a working
fallback most agents won't reach for by default — not a lost lesson. This is the same failure shape
ISSUE-044 named for the roster↔case-statement coupling (an inconsistent policy surface that reads
like deliberate design, not a gap) — different mechanism, same category — and the same project
convention (`backlog-pm`'s own memory carve-out, and this `issue-author` agent's own instructions)
already establishes that per-family memory write access via Write/Edit is the intended design, just
not implemented for every family until this fix.

## Resolution

Fixed directly as harness config (this repo treats `.claude/` hook scripts as harness config, not
product code needing the full artifact pipeline — no plan/spec lineage). Commit
`be8d80a4145531e564ef841f4ce41340d5c1ad3d`, "fix(hooks): grant every agent family write access to
its own memory dir."

**Mechanism:** adds a single pre-check to `.claude/hooks/backstop-agent-guard.sh`, ahead of the
existing per-family `case "$AGENT_NAME" in ... esac` block: any Write/Edit whose `$FILE_PATH`
matches `.claude/agent-memory/<family>/*` is allowed when the acting agent's name is prefixed with
that same `<family>` — the family is derived from the path itself, not a hardcoded per-family list,
so it composes with every existing artifact-type restriction unchanged and needs no update when a
new family is added.

**Verification:** all 46 mandated tests in `specs/tests/spec-003-agent-hooks-test.sh` still pass,
plus five manual probes: own-memory Write/Edit allowed, another family's memory Write/Edit still
blocked, a previously-always-blocked reviewer family (e.g. `impl-reviewer*`) now allowed its own
memory, and two regression checks confirming unrelated artifact-type restrictions are untouched.

This closes the narrower Write/Edit-vs-Bash inconsistency described above. Whether every family
listed in "Direction" below (in particular `general-purpose*`, which had no pre-existing memory
directory) ended up with a carve-out is a matter of the shipped diff, not re-litigated here.

## Direction (original triage, superseded by Resolution above)

The fix is mechanical and low-risk: extend each family's case arm to additionally allow
`.claude/agent-memory/<that-family>/*`, mirroring the existing `backlog-pm*` pattern:

- `bundle-author*` → also allow `.claude/agent-memory/bundle-author/*`
- `spec-author*` → also allow `.claude/agent-memory/spec-author/*`
- `adr-author*` → also allow `.claude/agent-memory/adr-author/*`
- `issue-author*` → also allow `.claude/agent-memory/issue-author/*`
- `planner*` → also allow `.claude/agent-memory/planner/*`
- `directive-author*` → also allow `.claude/agent-memory/directive-author/*`
- `implementer*` → also allow `.claude/agent-memory/implementer/*` (added alongside its existing,
  differently-shaped allow-list)
- `spec-reviewer*|plan-reviewer*|impl-reviewer*|bundle-reviewer*` → split into per-family arms (or
  keep combined but branch on `$AGENT_NAME` inside) so each reviewer family gets its own
  `.claude/agent-memory/<family>/*`, since a combined arm can't route to four different
  memory-directory names cleanly
- `general-purpose*` → also allow `.claude/agent-memory/general-purpose/*` (note: this directory
  does not currently exist under `.claude/agent-memory/`; the fix should confirm whether
  general-purpose is meant to have durable memory at all, or whether its scope is intentionally
  narrower than the SDLC author/reviewer families)

A regression test proving the fix should assert, for each family, that a Write/Edit into
`.claude/agent-memory/<family>/<file>.md` is allowed and a Write/Edit into another family's memory
directory (or into an artifact type outside the family's own) is still blocked — mirroring how
ISSUE-044's fix added a Go test asserting roster↔guard-case agreement rather than relying on manual
inspection.

## Notes / references

- Reported by `team-lead` mid-session (2026-08-15); reproduced directly by reading
  `.claude/hooks/backstop-agent-guard.sh` and `.claude/agent-memory/` in full rather than taken as a
  hypothesis.
- Related to but distinct from ISSUE-044 (closed): ISSUE-044 fixed missing-agent-in-guard drift
  (roster vs. case-statement completeness) with a Go test; this issue is about an access-scope gap
  that exists even for agents the guard already has correct, complete cases for.
