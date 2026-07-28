---
name: pm-interview-tooling-constraints
description: In headless PM triage runs, `claude -p --resume --fork-session` is blocked by the sandbox (needs approval, unavailable non-interactively) — but reading session transcripts with grep DOES work; fingerprint instead of interviewing
metadata:
  type: project
---

Fork-interviews (`claude -p --resume <id> --fork-session "<brief>"`) are
**not available** to the hook-invoked, headless backlog-pm run: the Bash
sandbox classifies the command as requiring approval, and a non-interactive
session cannot grant it. Observed 2026-07-27 (ISSUE-086 triage).

**Reading transcripts DOES work**, contrary to the note left on the
2026-07-27T00:06Z ISSUE-083 entry. `~/.claude/projects/<slug>/*.jsonl` is
readable with `ls -lt` (mtime ordering) and plain `grep -c <ARTIFACT-ID>`
(fingerprinting). What fails:
- `python3 -c "..."` inline scripts → approval-gated.
- `grep -o` with a complex bracket-expression pattern → sometimes
  approval-gated, and on a 6 MB transcript can blow the 120 s timeout from
  backtracking. Keep patterns literal-anchored and short.
- `for` loops in Bash → rejected outright ("Contains for_statement").

**How to apply:** when in-flight coverage matters, substitute
*fingerprint-by-grep-count* for the interview — `grep -c` each recent
session for the artifact IDs in question, rank by mtime, and reason from
counts plus the corpus (artifact status, plan files, `git log`). Say so
explicitly in the INBOX entry: "corpus-based, not interview-confirmed." Do
not silently present a fingerprint as an interview, and do not burn turns
retrying the fork command. If an interview is genuinely load-bearing,
escalate to Brandon to run it from an interactive session.

See [[pm-write-path]].
