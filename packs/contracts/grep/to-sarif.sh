#!/bin/sh
# Real grep-output -> SARIF converter shipped by the traceability pack (SPEC-038
# REQ-005/CLM-019). Reads `grep -rn -e <pattern> <targets>` output on stdin —
# lines of the form "<file>:<line>:<matched text>" — and emits SARIF 2.1.0 on
# stdout. Each grep match line becomes one SARIF result with a physicalLocation
# (artifactLocation.uri = file, region.startLine = the 1-indexed line grep
# reports). The forbidden token's presence IS the finding; the gate inverts a
# present match to an absence VIOLATION (the polarity lives gate-side, not here).
# A stderr banner exercises clean-stdout capture. Backstop ships no grep knowledge
# — this script is pack DATA the pack author wrote.
echo "grep to-sarif: transforming matches" >&2
# Emit one JSON object per grep match line (newline-delimited), then `jq -s`
# slurps the stream into an array and wraps it as SARIF. Only the first two colons
# are structural ("file:line:"); the match text may itself contain colons.
awk -F: '
  NF >= 3 {
    file=$1; line=$2;
    text=$0;
    sub(/^[^:]*:[^:]*:/, "", text);
    gsub(/\\/, "\\\\", text); gsub(/"/, "\\\"", text); gsub(/\t/, " ", text);
    printf "{\"file\":\"%s\",\"line\":%s,\"text\":\"%s\"}\n", file, line, text;
  }
' | jq -s '{
  version: "2.1.0",
  runs: [
    {
      results: [ .[] | {
        ruleId: "contract-absence",
        level: "error",
        message: { text: ("forbidden symbol present: " + .text) },
        locations: [ {
          physicalLocation: {
            artifactLocation: { uri: .file },
            region: { startLine: (.line | tonumber) }
          }
        } ]
      } ]
    }
  ]
}'
