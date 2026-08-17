#!/bin/sh
# go-toolchain test convert script (SPEC-034 REQ-003/REQ-004, DD-2).
# Re-expresses the retired parseGoTestFailures normalization OUTSIDE the core
# binary: reads raw `go test` output on stdin and emits located SARIF on stdout.
# A converter banner on stderr exercises clean-stdout (SandboxedRunStdout).
#
# Mirrors the bespoke logic: for each `--- FAIL: TestName` line, scan forward to
# the first `file.go:NN: detail` position before the next FAIL. The violation
# message is "TestName: detail" when a detail is found, else "TestName failed";
# File/Line come from that position (empty/0 when absent).
#
# ISSUE-067 WIDENING. Once the pack's test producer folds `go test`'s stderr into
# the payload, TWO further shapes arrive that this converter previously never saw,
# and both are handled below WITHOUT touching the `--- FAIL:` machinery above:
#
#   (a) COMPILER / VET DIAGNOSTICS, which Go prints under a `# <import-path>`
#       header and OUTSIDE any `--- FAIL:` block.
#   (b) The `FAIL\t<import-path> [build failed]` SUMMARY, which is all core used
#       to see for such a run.
#
# The `#` header is the DISCRIMINATOR, and it is deliberately strict: merging
# stderr means anything a test writes there now reaches this script, so a passing
# test printing a `notes.go:12: check this`-shaped line must manufacture NO
# finding. Only a diagnostic reported UNDER a header counts.
#
# The header is MODE STATE, not a one-line lookback. Go emits MANY diagnostics
# under ONE header; clearing after the first would convert only the first and
# silently drop the rest — an under-report, the same defect class this widening
# exists to close, just quieter.
#
# POSIX awk only (no gawk-only constructs) — it must run under the macOS system
# awk inside the sandbox.
echo "go-toolchain test-to-sarif: normalizing test failures" >&2

awk '
  BEGIN { printf "{\"version\":\"2.1.0\",\"runs\":[{\"results\":["; sep=""; hdr=0; nres=0; nbf=0 }
  function esc(s) { gsub(/\\/, "\\\\", s); gsub(/"/, "\\\"", s); gsub(/\t/, "\\t", s); return s }
  function result(msg, file, lno) {
    printf "%s{\"ruleId\":\"go-test\",\"level\":\"error\",\"message\":{\"text\":\"%s\"},\"locations\":[{\"physicalLocation\":{\"artifactLocation\":{\"uri\":\"%s\"},\"region\":{\"startLine\":%s}}}]}", sep, esc(msg), esc(file), lno
    sep=","
    nres++
  }
  function flush() {
    if (cur == "") return
    msg = (detail == "") ? (cur " failed") : (cur ": " detail)
    result(msg, file, (lno=="" ? "0" : lno))
    cur=""; file=""; lno=""; detail=""; have_pos=0
  }
  # ISSUE-067: normalize ONE compiler/vet diagnostic line into a located finding.
  # The optional column is stripped exactly as build-to-sarif.sh does; the two
  # scripts read as siblings on this shape.
  function diagnostic(l,   idx, dfile, rest, dlno) {
    idx = index(l, ".go:")
    dfile = substr(l, 1, idx+2)          # includes ".go"
    rest = substr(l, idx+4)              # after ".go:"
    if (match(rest, /^[0-9]+/) == 0) return
    dlno = substr(rest, 1, RLENGTH)
    rest = substr(rest, RLENGTH+1)
    sub(/^:/, "", rest)                  # drop the colon after the line number
    if (match(rest, /^[0-9]+:/) > 0) { rest = substr(rest, RLENGTH+1) }  # optional column
    gsub(/^[ \t]+/, "", rest)
    result(rest, dfile, dlno)
  }
  {
    raw = $0
    line = raw
    gsub(/^[ \t]+/, "", line); gsub(/[ \t]+$/, "", line)

    # New FAIL block boundary (UNCHANGED machinery). A test-failure block also
    # ends any header mode: what follows belongs to the test, not the compiler.
    if (match(line, /^--- FAIL: [^ ]+/) > 0) {
      flush()
      hdr = 0
      # extract test name: token after "--- FAIL: "
      t = substr(line, 11)            # after "--- FAIL: "
      n = index(t, " ")
      if (n > 0) t = substr(t, 1, n-1)
      cur = t
      have_pos = 0
      next
    }

    # ISSUE-067 (a): a `# <import-path>` header SETS the sticky mode flag. Go emits
    # consecutive header lines for a vet failure (`# pkg` then `# [pkg]`); both set
    # it, neither clears it.
    if (substr(line, 1, 1) == "#") { hdr = 1; next }

    # ISSUE-067 (b): the build-failure SUMMARY. NOTE THE EXACT BYTES, measured with
    # od -c against real `go test` output: ONE TAB after FAIL, then the import path,
    # then a SPACE — not a tab — before `[build failed]`. A pattern written with a
    # tab there matches nothing, the floor below never fires, and a partially
    # captured failure falls straight back onto the opaque crash.
    if (raw ~ /^FAIL\t[^ \t]+ \[build failed\]/) {
      hdr = 0
      nf = split(raw, ff, "\t")
      if (nf >= 2) {
        pkg = ff[2]
        sub(/ \[build failed\].*$/, "", pkg)
        nbf++
        bf[nbf] = pkg
      }
      next
    }

    # ISSUE-067 (a): a diagnostic OUTSIDE a `--- FAIL:` block becomes a finding ONLY
    # while a header is in effect. STICKY: the flag survives across consecutive
    # diagnostic lines, so ALL N diagnostics under one header become N findings.
    if (cur == "" && hdr == 1 && line ~ /^[^ \t]+\.go:[0-9]+:/) {
      diagnostic(line)
      next
    }

    # Within a block, capture first file.go:NN: detail position (UNCHANGED).
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
      next
    }

    # Any other line — a blank line, an `ok` line, the trailing bare `FAIL` —
    # clears the header mode.
    hdr = 0
  }
  END {
    flush()
    # ISSUE-067 FLOOR: if the whole run produced NO results at all but named at
    # least one [build failed] package, emit one unlocated finding per package. A
    # non-zero run whose output still names a failure can then never reach the
    # crash-guard path, which is what made a partially captured failure opaque.
    if (nres == 0 && nbf > 0) {
      for (i = 1; i <= nbf; i++) {
        result("build failed: " bf[i], "", "0")
      }
    }
    printf "]}]}\n"
  }
'
