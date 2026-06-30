#!/bin/sh
# bun-toolchain coverage convert script (SPEC-047 REQ-003; consumes SPEC-044 REQ-006).
# Re-expresses lcov coverage knowledge OUTSIDE the core binary (the bun analogue of
# go-toolchain's coverage-to-records.sh): reads an lcov `.info` report on stdin and
# emits per-FILE coverage-records JSON on stdout — NOT SARIF.
#
# lcov per-file block shape (the fields this convert reads):
#   SF:<path>           start of a file record
#   LF:<n> / LH:<n>     lines found (total) / lines hit (covered)
#   BRF:<n> / BRH:<n>   branches found (total) / branches hit (covered)
#   end_of_record
#
# For EACH measured file it emits TWO canonical records sharing one path:
#   {metric:"line",   covered:LH,  total:LF}
#   {metric:"branch", covered:BRH, total:BRF}
# Raw counts only — NEVER a pre-computed percent, NEVER an aggregate of the two,
# NEVER a single collapsed record. A file with no branchable code (BRF:0) emits a
# branch record carrying total:0 (the N/A cell, faithfully preserved, never coerced
# to 0%) while its line record is still measured. Both records are measured:true,
# excluded:false, and use EXACTLY the canonical
# {path,covered,total,measured,excluded,metric} keys so they parse through the
# EXISTING check.ParsePackCoverage (DisallowUnknownFields) with NO new field.
#
# Implemented in POSIX awk (no gawk-only features) so it runs under the macOS system
# awk inside the sandbox. A converter banner on stderr exercises clean-stdout capture.
echo "bun-toolchain coverage-to-records: emitting per-file line+branch records from lcov" >&2

awk '
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); return s }
  /^SF:/  { file = substr($0, 4); lf = 0; lh = 0; brf = 0; brh = 0; have = 1; next }
  /^LF:/  { lf  = substr($0, 4) + 0; next }
  /^LH:/  { lh  = substr($0, 4) + 0; next }
  /^BRF:/ { brf = substr($0, 5) + 0; next }
  /^BRH:/ { brh = substr($0, 5) + 0; next }
  /^end_of_record/ {
    if (have) {
      order[++n] = file
      LFv[file] = lf; LHv[file] = lh; BRFv[file] = brf; BRHv[file] = brh
    }
    have = 0
    next
  }
  END {
    printf "["
    sep = ""
    for (i = 1; i <= n; i++) {
      f = order[i]
      printf "%s{\"path\":\"%s\",\"covered\":%d,\"total\":%d,\"measured\":true,\"excluded\":false,\"metric\":\"line\"}", \
        sep, esc(f), LHv[f] + 0, LFv[f] + 0
      sep = ","
      printf "%s{\"path\":\"%s\",\"covered\":%d,\"total\":%d,\"measured\":true,\"excluded\":false,\"metric\":\"branch\"}", \
        sep, esc(f), BRHv[f] + 0, BRFv[f] + 0
    }
    printf "]\n"
  }
'
