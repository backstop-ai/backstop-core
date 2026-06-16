#!/usr/bin/env bash
# PostToolUse(Write|Edit|MultiEdit): after an artifact file is written, validate it
# and surface ITS violations back to the agent so it self-corrects immediately.
# Scoped to the just-edited file so pre-existing unrelated failures stay silent.

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0

FILE_PATH="$(printf '%s' "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
[[ -z "$FILE_PATH" ]] && exit 0

# Only backstop artifact files have a schema to validate against.
case "$FILE_PATH" in
  *.bundle.md|*.spec.md|*.plan.yml|*.adr.md|*.issue.md|*.directive.md|*.standard.md) ;;
  *) exit 0 ;;
esac

base="$(basename "$FILE_PATH")"
out="$(./bin/backstop artifact validate --json 2>/dev/null)"
[[ -z "$out" ]] && exit 0

viol="$(printf '%s' "$out" | jq -r --arg f "$base" \
  '.violations[]? | select(.file==$f) | "  [\(.rule)] \(.message)"' 2>/dev/null)"

if [[ -n "$viol" ]]; then
  {
    echo "backstop: $base is not schema-valid. Fix these before continuing:"
    echo "$viol"
  } >&2
  exit 2
fi
exit 0
