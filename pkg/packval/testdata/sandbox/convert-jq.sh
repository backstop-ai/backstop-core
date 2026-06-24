#!/bin/sh
# convert-jq.sh — real-interpreter convert fixture for the macOS sandbox tests
# (ISSUE-029, CLM-003). It pipes its stdin through real jq — a dynamically
# linked interpreter (libjq + oniguruma + libSystem) that SIGABRTs at dyld load
# under the UNFIXED sandbox profile (packDir-only file-read*).
#
# IMPORTANT: this script MUST be driven through the REAL SandboxedRunStdout /
# sandbox-exec path — NEVER the /bin/sh stub (stubSandboxedRunStdout). Running it
# under the stub bypasses sandbox-exec and would vacuously pass, which is the
# exact vacuous green ISSUE-029 exists to kill.
#
# Transform: read {"a":N} on stdin, emit N+1 on stdout. Locate jq the same way
# the test does (PATH, then Intel/Apple-Silicon Homebrew prefixes).
if command -v jq >/dev/null 2>&1; then
  JQ=jq
elif [ -x /usr/local/bin/jq ]; then
  JQ=/usr/local/bin/jq
elif [ -x /opt/homebrew/bin/jq ]; then
  JQ=/opt/homebrew/bin/jq
else
  echo "convert-jq.sh: jq not found" 1>&2
  exit 127
fi
exec "$JQ" '.a + 1'
