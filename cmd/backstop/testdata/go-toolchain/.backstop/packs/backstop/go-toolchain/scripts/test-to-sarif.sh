#!/bin/sh
# go-toolchain test convert script (SPEC-034 REQ-003/REQ-004, DD-2).
# Re-expresses the retired parseGoTestFailures normalization OUTSIDE the core
# binary: reads raw `go test` stdout on stdin and emits located SARIF on stdout.
# A converter banner on stderr exercises clean-stdout (SandboxedRunStdout).
#
# Mirrors the bespoke logic: for each `--- FAIL: TestName` line, scan forward to
# the first `file.go:NN: detail` position before the next FAIL. The violation
# message is "TestName: detail" when a detail is found, else "TestName failed";
# File/Line come from that position (empty/0 when absent).
echo "go-toolchain test-to-sarif: normalizing test failures" >&2

awk '
  BEGIN { printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":["; sep="" }
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); gsub(/\t/, "\\t", s); return s }
  function flush() {
    if (cur == "") return
    msg = (detail == "") ? (cur " failed") : (cur ": " detail)
    printf "%s{\"ruleId\":\"go-test\",\"level\":\"error\",\"message\":{\"text\":\"%s\"},\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},\"region\":{\"startLine\":%s}}}]}", sep, esc(msg), esc(file), (lno==""?"0":lno)
    sep=","
    cur=""; file=""; lno=""; detail=""; have_pos=0
  }
  {
    raw = $0
    line = raw
    gsub(/^[ \t]+/, "", line); gsub(/[ \t]+$/, "", line)
    # New FAIL block boundary.
    if (match(line, /^--- FAIL: [^ ]+/) > 0) {
      flush()
      # extract test name: token after "--- FAIL: "
      t = substr(line, 11)            # after "--- FAIL: "
      n = index(t, " ")
      if (n > 0) t = substr(t, 1, n-1)
      cur = t
      have_pos = 0
      next
    }
    # Within a block, capture first file.go:NN: detail position.
    if (cur != "" && have_pos == 0 && line ~ /^[^ \t]+\.go:[0-9]+:/) {
      idx = index(line, ".go:")
      file = substr(line, 1, idx+2)
      rest = substr(line, idx+4)
      if (match(rest, /^[0-9]+/) > 0) {
        lno = substr(rest, 1, RLENGTH)
        rest = substr(rest, RLENGTH+1)
        sub(/^:/, "", rest)
        gsub(/^[ \t]+/, "", rest)
        detail = rest
        have_pos = 1
      }
    }
  }
  END { flush(); printf "]}]}\n" }
'
