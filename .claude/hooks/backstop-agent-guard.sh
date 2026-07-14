#!/usr/bin/env bash

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0

AGENT_NAME="$(echo "$INPUT" | jq -r '.agent_type // empty' 2>/dev/null)"
[[ -z "$AGENT_NAME" ]] && exit 0

TOOL_NAME="$(echo "$INPUT" | jq -r '.tool_name // empty' 2>/dev/null)"

block() {
  printf '{"decision":"block","reason":"%s"}\n' "$1"
  exit 2
}

# ---- Bash: block WRITE-shaped commands targeting artifact files, for every
# subagent. Reads (grep/cat/validate/tests) stay unrestricted. Artifacts change
# only via Write/Edit, where this guard AND validate-artifact both fire.
if [[ "$TOOL_NAME" == "Bash" ]]; then
  CMD="$(echo "$INPUT" | jq -r '.tool_input.command // empty' 2>/dev/null)"
  [[ -z "$CMD" ]] && exit 0
  ART='\.(bundle|spec|issue|adr|directive)\.md|\.plan\.yml|BACKLOG\.yml'
  if echo "$CMD" | grep -qE "$ART"; then
    # redirection whose TARGET is an artifact file (not 2>&1 / >/dev/null)
    echo "$CMD" | grep -qE ">[> ]*\"?'?[^ |;&>\"']*($ART)" \
      && block "agent $AGENT_NAME: bash redirect into artifact file; use the Write/Edit tools"
    echo "$CMD" | grep -qE "\btee\b[^|]*($ART)" \
      && block "agent $AGENT_NAME: bash tee into artifact file; use the Write/Edit tools"
    echo "$CMD" | grep -qE "\b(cp|mv|install|rsync|touch|rm)\b[^|;]*($ART)" \
      && block "agent $AGENT_NAME: bash file-op on artifact file; use the Write/Edit tools"
    echo "$CMD" | grep -qE "\b(sed|perl)\b[^|;]*-i[^|;]*($ART)" \
      && block "agent $AGENT_NAME: bash in-place edit of artifact file; use the Write/Edit tools"
    if echo "$CMD" | grep -qE "\b(python3?|ruby|node)\b"; then
      # open() only with an explicit write/append mode; bare open(...).read() stays free
      echo "$CMD" | grep -qE "open\([^)]*['\"](w|a|r\+)|write_text|writelines|\.write\(|writeFile" \
        && block "agent $AGENT_NAME: scripted write to artifact file; use the Write/Edit tools"
    fi
  fi
  exit 0
fi

# ---- Write/Edit: allow each author FAMILY only its own artifact type.
# Families are glob-matched (spec-author, spec-author-052, ... all count).
FILE_PATH="$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
[[ -z "$FILE_PATH" ]] && exit 0

wblock() { block "agent $AGENT_NAME not permitted to write $FILE_PATH"; }

case "$AGENT_NAME" in
  bundle-author*) [[ "$FILE_PATH" == *.bundle.md ]] && exit 0 || wblock ;;
  spec-author*) [[ "$FILE_PATH" == *.spec.md ]] && exit 0 || wblock ;;
  adr-author*) [[ "$FILE_PATH" == *.adr.md ]] && exit 0 || wblock ;;
  issue-author*) [[ "$FILE_PATH" == *.issue.md ]] && exit 0 || wblock ;;
  planner*) [[ "$FILE_PATH" == *.plan.yml ]] && exit 0 || wblock ;;
  directive-author*)
    [[ "$FILE_PATH" == *.directive.md ]] && exit 0
    [[ "$(basename "$FILE_PATH")" == "BACKLOG.yml" ]] && exit 0
    wblock
    ;;
  implementer*)
    [[ "$FILE_PATH" == *.go ]] && exit 0
    basename="$(basename "$FILE_PATH")"
    [[ "$basename" == "go.mod" || "$basename" == "go.sum" || "$basename" == "Makefile" ]] && exit 0
    if [[ "$FILE_PATH" == *.json || "$FILE_PATH" == *.yml ]]; then
      [[ "$FILE_PATH" == .backstop/* || "$FILE_PATH" == artifacts/* ]] && exit 0
    fi
    wblock
    ;;
  spec-reviewer*|plan-reviewer*|impl-reviewer*|bundle-reviewer*) wblock ;;
  general-purpose*)
    [[ "$FILE_PATH" == */prototype/* ]] && exit 0
    wblock
    ;;
  *) wblock ;;
esac
