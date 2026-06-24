#!/bin/sh
# Real ast-grep stdin->SARIF converter shipped by the pack (REQ-008).
# Reads `ast-grep scan --config sgconfig.yml --json` output (a JSON array of
# findings spanning EVERY rule in one invocation) on stdin and emits SARIF 2.1.0
# on stdout. Each finding's .ruleId (rule-one / rule-two) becomes the SARIF
# ruleId, so findings from BOTH rules survive the pipe distinctly. ast-grep
# reports 0-indexed lines; SARIF startLine is 1-indexed, so we add 1.
jq '{
  version: "2.1.0",
  runs: [
    {
      results: [ .[] | {
        ruleId: .ruleId,
        level: (if .severity == "error" then "error" elif .severity == "warning" then "warning" else "error" end),
        message: { text: .message },
        locations: [ {
          physicalLocation: {
            artifactLocation: { uri: .file },
            region: { startLine: (.range.start.line + 1) }
          }
        } ]
      } ]
    }
  ]
}'
