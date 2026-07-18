#!/bin/sh
# ISSUE-068 fixture: the TEST engine's convert. Reads the shared suite run's
# payload on stdin and emits one located SARIF finding whose message ECHOES the
# payload, so a test can prove BOTH converts received the same shared output.
payload=$(cat)
esc=$(printf '%s' "$payload" | tr -d '\n' | sed 's/\\/\\\\/g; s/"/\\"/g')
printf '{"version":"2.1.0","runs":[{"results":[{"ruleId":"suite-test","level":"error","message":{"text":"test-convert saw: %s"},"locations":[{"physicalLocation":{"artifactLocation":{"uri":"suite.ts"},"region":{"startLine":1}}}]}]}]}\n' "$esc"
