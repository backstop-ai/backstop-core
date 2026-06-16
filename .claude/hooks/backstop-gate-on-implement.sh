#!/usr/bin/env bash
# SubagentStop: when the IMPLEMENTER finishes a task, run the gate over the
# working-tree diff and refuse to let it stop while the gate is red — i.e.
# refactor-until-green happens without the human noticing/prompting it.
#
# Fires ONLY for the implementer. Other agents are capability-bounded (the
# agent-guard blocks them from writing code), so gating them would deadlock:
# blocked-from-stopping AND unable-to-fix. Routing enforcement to the one agent
# that can satisfy it is the point, not a loophole.

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0

AGENT="$(printf '%s' "$INPUT" | jq -r '.agent_type // empty' 2>/dev/null)"
[[ "$AGENT" != "implementer" ]] && exit 0

result="$(./bin/backstop gate --json 2>/dev/null)"
[[ -z "$result" ]] && exit 0

pass="$(printf '%s' "$result" | jq -r '.pass // empty' 2>/dev/null)"
[[ "$pass" == "true" ]] && exit 0

count="$(printf '%s' "$result" | jq -r '.total_violations // "?"' 2>/dev/null)"
failed="$(printf '%s' "$result" | jq -r \
  '.steps[]? | select(.status=="fail") | "  - \(.step_name): \(.violations // [] | map(.message) | join("; "))"' \
  2>/dev/null | head -40)"

{
  echo "backstop gate is RED on your changes ($count violation(s)). Do not finish — refactor until it passes:"
  [[ -n "$failed" ]] && echo "$failed"
  echo "Run \`./bin/backstop gate\` for full detail."
} >&2
exit 2
