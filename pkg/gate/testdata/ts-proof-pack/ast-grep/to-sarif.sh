#!/bin/sh
# Real ast-grep stdin->SARIF converter shipped by the pack (DD-7 / REQ-008).
# Reads `ast-grep scan --json` output (a JSON array of findings) on stdin and
# emits SARIF 2.1.0 on stdout. ast-grep reports 0-indexed lines; SARIF startLine
# is 1-indexed, so we add 1. A banner on stderr exercises clean-stdout capture.
#
# The pack STAMPS a `substantiveness_role` property declaring each finding's role —
# `hollow` for its hollow-test rule — so the gate routes by the declared role, not the
# pack's rule NAME (`hollow-test-ts`); ISSUE-064.
echo "ast-grep to-sarif: transforming findings" >&2
jq '{
  version: "2.1.0",
  runs: [
    {
      results: [ .[] | (
        {
          ruleId: .ruleId,
          level: (if .severity == "error" then "error" elif .severity == "warning" then "warning" else "error" end),
          message: { text: .message },
          locations: [ {
            physicalLocation: {
              artifactLocation: { uri: .file },
              region: { startLine: (.range.start.line + 1) }
            }
          } ]
        }
        + (
          (
            (if ((.ruleId // "") | test("hollow")) then { substantiveness_role: "hollow" }
             elif ((.ruleId // "") | test("referenced-symbol")) then { substantiveness_role: "referenced-symbol" }
             else {} end)
          ) as $props
          | if ($props | length) > 0 then { properties: $props } else {} end
        )
      ) ]
    }
  ]
}'
