#!/usr/bin/env bash
# PostToolUse(Write): when a NEW issue or bundle artifact is created, log it
# and dispatch a detached backlog-pm triage run. Deterministic capture layer —
# judgment lives in the backlog-pm agent, authority stays with Brandon.

[[ -n "$BACKSTOP_PM_RUN" ]] && exit 0   # never recurse from a PM run

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0
FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
[[ -z "$FILE_PATH" ]] && exit 0

case "$FILE_PATH" in
  */testdata/*) exit 0 ;;   # fixture artifacts are never real backlog items
  *issues/*.issue.md|*bundles/*.bundle.md) ;;
  *) exit 0 ;;
esac

# Only NEW artifacts (untracked) trigger triage; edits to existing ones don't.
REL="${FILE_PATH#"$PWD"/}"
git ls-files --error-unmatch "$REL" >/dev/null 2>&1 && exit 0

PENDING=".backstop/pm/pending.log"
# Dedupe: one triage per artifact.
grep -qF "$REL" "$PENDING" 2>/dev/null && exit 0
echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) $REL" >> "$PENDING"

TS="$(date +%s)"
( BACKSTOP_PM_RUN=1 nohup claude -p --agent backlog-pm --permission-mode acceptEdits \
    "Triage new artifact: $REL" > ".backstop/pm/runs/$TS.log" 2>&1 & ) >/dev/null 2>&1

command -v osascript >/dev/null && osascript -e "display notification \"backlog-pm triaging $(basename "$REL")\" with title \"backstop\"" >/dev/null 2>&1
exit 0
