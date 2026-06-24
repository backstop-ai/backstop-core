#!/bin/sh
# Real ast-grep stdin->SARIF converter shipped by the traceability pack (SPEC-038).
# Reads `ast-grep run --pattern <compiled> --json` output (a JSON array of matches
# for the compiled signature pattern) on stdin and emits SARIF 2.1.0 on stdout. A
# MATCH means the declared signature is PRESENT — the gate verdicts that as
# SATISFIED; an EMPTY array means no match -> the gate raises a signature VIOLATION
# (the match-verdict lives gate-side, not here). ast-grep reports 0-indexed lines;
# SARIF startLine is 1-indexed, so we add 1. A stderr banner exercises clean-stdout
# capture.
echo "ast-grep to-sarif: transforming matches" >&2
jq '{
  version: "2.1.0",
  runs: [
    {
      results: [ .[] | {
        ruleId: "contract-signature",
        level: "error",
        message: { text: ("signature present: " + (.text // "")) },
        locations: [ {
          physicalLocation: {
            artifactLocation: { uri: (.file // "") },
            region: { startLine: ((.range.start.line // 0) + 1) }
          }
        } ]
      } ]
    }
  ]
}'
