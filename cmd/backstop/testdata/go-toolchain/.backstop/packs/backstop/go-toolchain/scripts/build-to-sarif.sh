#!/bin/sh
# go-toolchain build convert script (SPEC-034 REQ-003/REQ-004, DD-2).
# Re-expresses the retired parseGoBuildErrors normalization OUTSIDE the core
# binary: reads raw `go build ./...` stdout on stdin and emits located SARIF
# (file:line:message) on stdout. A converter banner on stderr exercises the
# clean-stdout (SandboxedRunStdout) capture; it never reaches the SARIF bytes.
#
# Mirrors the bespoke regex ^(.+?\.go):(\d+):(?:(\d+):)?\s*(.+)$ — the optional
# column is stripped; "# package" header lines and non-positional notes are
# ignored. Implemented in POSIX awk (no gawk match() captures) so it runs under
# the macOS system awk inside the sandbox.
echo "go-toolchain build-to-sarif: normalizing compiler errors" >&2

awk '
  BEGIN { printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":["; sep="" }
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); gsub(/\t/, "\\t", s); return s }
  {
    line = $0
    gsub(/^[ \t]+/, "", line); gsub(/[ \t]+$/, "", line)
    if (line == "" || substr(line,1,1) == "#") next
    if (line !~ /\.go:[0-9]+:/) next
    idx = index(line, ".go:")
    file = substr(line, 1, idx+2)           # includes ".go"
    rest = substr(line, idx+4)              # after ".go:"
    if (match(rest, /^[0-9]+/) == 0) next
    lno = substr(rest, 1, RLENGTH)
    rest = substr(rest, RLENGTH+1)          # after line number
    sub(/^:/, "", rest)                     # drop the colon after line number
    if (match(rest, /^[0-9]+:/) > 0) { rest = substr(rest, RLENGTH+1) }  # drop optional column
    gsub(/^[ \t]+/, "", rest)
    msg = rest
    printf "%s{\"ruleId\":\"go-build\",\"level\":\"error\",\"message\":{\"text\":\"%s\"},\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},\"region\":{\"startLine\":%s}}}]}", sep, esc(msg), esc(file), lno
    sep=","
  }
  END { printf "]}]}\n" }
'
