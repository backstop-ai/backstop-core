#!/bin/sh
# bun-toolchain prettier format convert (SPEC-047 REQ-001 / DD-3).
# Re-expresses prettier's `--check` result OUTSIDE the core binary as a
# LINT-CATEGORY SARIF findings engine: reads raw `prettier --check` output on
# stdin and emits one located SARIF finding per UNFORMATTED file on stdout. Format
# is modeled as lint (DD-3) — these findings ride the EXISTING lint dimension; no
# new "format" gate dimension exists. A converter banner on stderr exercises the
# clean-stdout (SandboxedRunStdout) capture; it never reaches the SARIF bytes.
#
# prettier --check prints one `[warn] <path>` line per unformatted file (plus a
# trailing summary line). The convert keys on the `[warn] ` prefix and ignores the
# summary. Region is file-level (startLine 1) — a formatting issue spans the file.
# Implemented in POSIX awk (no gawk-only features) so it runs under the macOS
# system awk inside the sandbox.
echo "bun-toolchain format-to-sarif: normalizing unformatted files to lint findings" >&2

awk '
  BEGIN { printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":["; sep="" }
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); gsub(/\t/, "\\t", s); return s }
  {
    line = $0
    gsub(/^[ \t]+/, "", line); gsub(/[ \t]+$/, "", line)
    # Only `[warn] <path>` lines name an unformatted file; the trailing
    # "[warn] Code style issues found ..." summary names no path and is skipped.
    if (substr(line, 1, 7) != "[warn] ") next
    file = substr(line, 8)
    if (file == "" || file ~ /Code style issues/ || file ~ /Forgot to run/) next
    printf "%s{\"ruleId\":\"prettier\",\"level\":\"warning\",\"message\":{\"text\":\"File is not formatted (prettier --check)\"},\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},\"region\":{\"startLine\":1}}}]}", sep, esc(file)
    sep=","
  }
  END { printf "]}]}\n" }
'
