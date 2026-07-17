#!/usr/bin/env bash
#
# Block scratch/handoff cruft from being written into the tracked repo.
# HANDOFF-*, SCRATCH*, *.scratch, *.tmp belong in ./tmp/ (gitignored), never
# in the working tree. Writes OUTSIDE the repo (auto-memory under ~/.claude,
# the session scratchpad in /private/tmp) and writes under ./tmp/ are allowed.

INPUT="$(cat)"
[[ -z "$INPUT" ]] && exit 0

FILE_PATH="$(echo "$INPUT" | jq -r '.tool_input.file_path // empty' 2>/dev/null)"
[[ -z "$FILE_PATH" ]] && exit 0

REPO_ROOT="$(git rev-parse --show-toplevel 2>/dev/null || echo "$PWD")"

# Resolve to an absolute path so relative and absolute targets compare alike.
case "$FILE_PATH" in
  /*) ABS="$FILE_PATH" ;;
  *)  ABS="$REPO_ROOT/$FILE_PATH" ;;
esac

# Only guard the tracked repo. Anything outside it is not our concern.
[[ "$ABS" == "$REPO_ROOT"/* ]] || exit 0
# ./tmp/ is the sanctioned scratch drop — allow.
[[ "$ABS" == "$REPO_ROOT"/tmp/* ]] && exit 0

base="$(basename "$ABS")"
case "$base" in
  HANDOFF*|SCRATCH*|*.scratch|*.tmp)
    printf '{"decision":"block","reason":"%s"}\n' \
      "scratch/handoff cruft ($base) must not be written into the tracked repo — write it to ./tmp/ (gitignored) instead."
    exit 2
    ;;
esac

exit 0
