#!/bin/sh
# go-toolchain coverage convert script (SPEC-042 REQ-007/CLM-021).
# Re-expresses Go coverage-profile knowledge OUTSIDE the core binary (the coverage
# analogue of test-to-sarif.sh): reads a Go `-coverprofile` profile on stdin and
# emits per-FILE coverage-records JSON on stdout — NOT SARIF.
#
# Go profile line shape:  path:startLine.col,endLine.col numStmt count
# Per FILE: total   = sum(numStmt) over the file's blocks
#           covered = sum(numStmt) over blocks whose count > 0
# A file whose blocks are all 0-statement aggregates to total==0 (the N/A cell) —
# never coerced to a 0% value. Every file present in the profile is `measured: true`,
# and a record is emitted for measured-and-PASSING files too (not only shortfalls),
# so the consumer can distinguish measured-and-passed from not-measured.
# Metric is stamped "statement" (Go's -coverprofile granularity).
#
# Implemented in POSIX awk (no gawk-only features) so it runs under the macOS system
# awk inside the sandbox. A converter banner on stderr exercises clean-stdout capture.
echo "go-toolchain coverage-to-records: aggregating per-file statement coverage" >&2

awk '
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
  {
    line = $0
    # Skip the mode header (e.g. "mode: atomic") and blank lines.
    if (line ~ /^mode:/) next
    if (line == "") next
    # Field 1 is "path:startLine.col,endLine.col"; field 2 is numStmt; field 3 count.
    nf = split(line, f, " ")
    if (nf < 3) next
    loc = f[1]
    numstmt = f[2] + 0
    count = f[3] + 0
    # File path is everything before the final ":" that introduces the position.
    idx = index(loc, ".go:")
    if (idx == 0) next
    file = substr(loc, 1, idx + 2)   # includes ".go"
    if (!(file in seen)) { order[++n] = file; seen[file] = 1 }
    total[file] += numstmt
    if (count > 0) covered[file] += numstmt
  }
  END {
    printf "["
    sep = ""
    for (i = 1; i <= n; i++) {
      file = order[i]
      printf "%s{\"path\":\"%s\",\"covered\":%d,\"total\":%d,\"measured\":true,\"excluded\":false,\"metric\":\"statement\"}", \
        sep, esc(file), covered[file] + 0, total[file] + 0
      sep = ","
    }
    printf "]\n"
  }
'
