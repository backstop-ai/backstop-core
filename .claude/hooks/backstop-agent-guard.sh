#!/usr/bin/env bash

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0

AGENT_NAME="$(echo "$INPUT" | jq -r '.agent_type // empty' 2>/dev/null)"
[[ -z "$AGENT_NAME" ]] && exit 0

FILE_PATH="$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
[[ -z "$FILE_PATH" ]] && exit 0

block() {
  printf '{"decision":"block","reason":"agent %s not permitted to write %s"}\n' "$AGENT_NAME" "$FILE_PATH"
  exit 2
}

case "$AGENT_NAME" in
  bundle-author) [[ "$FILE_PATH" == *.bundle.md ]] && exit 0 || block ;;
  spec-author) [[ "$FILE_PATH" == *.spec.md ]] && exit 0 || block ;;
  adr-author) [[ "$FILE_PATH" == *.adr.md ]] && exit 0 || block ;;
  issue-author) [[ "$FILE_PATH" == *.issue.md ]] && exit 0 || block ;;
  planner) [[ "$FILE_PATH" == *.plan.yml ]] && exit 0 || block ;;
  implementer)
    [[ "$FILE_PATH" == *.go ]] && exit 0
    basename="$(basename "$FILE_PATH")"
    [[ "$basename" == "go.mod" || "$basename" == "go.sum" || "$basename" == "Makefile" ]] && exit 0
    if [[ "$FILE_PATH" == *.json || "$FILE_PATH" == *.yml ]]; then
      [[ "$FILE_PATH" == .backstop/* || "$FILE_PATH" == artifacts/* ]] && exit 0
    fi
    block
    ;;
  spec-reviewer|plan-reviewer|impl-reviewer) block ;;
  general-purpose)
    [[ "$FILE_PATH" == */prototype/* ]] && exit 0
    block
    ;;
  *) block ;;
esac
