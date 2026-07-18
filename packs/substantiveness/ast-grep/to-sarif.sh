#!/bin/sh
# Real ast-grep stdin->SARIF converter shipped by the pack (DD-7 / REQ-008, ISSUE-062).
# Reads `ast-grep scan --config sgconfig.yml --json` output (a JSON array of findings
# spanning EVERY rule in one invocation) on stdin and emits SARIF 2.1.0 on stdout.
# Each finding's .ruleId (hollow-test-go / referenced-symbol-go) becomes the SARIF
# ruleId, so findings from BOTH rules survive the pipe distinctly.
#
# The matched metavariables are lifted into the SARIF result-level `properties`:
# `func` from $FN (the enclosing test name) and `symbol` from $PKG (the referenced
# package). This is the STRUCTURED channel the gate substantiveness join reads, so a
# test name containing spaces or quotes survives intact instead of being parsed out of
# the prose message and truncated at the first space (ISSUE-062). The human-readable
# `message` is retained for the report surface but carries NO machine-parsed contract.
# A finding whose rule captured neither metavariable emits no `func`/`symbol` property.
#
# The pack also STAMPS a `substantiveness_role` property declaring what each finding IS —
# `hollow` (hollow-test rule) or `referenced-symbol` (referenced-symbol rule) — mapping
# its OWN rule names to the role vocabulary the gate routes on (ISSUE-064). The gate
# partitions purely on this declared role, so the pack's rule NAMES are no longer a
# routing key; only the pack knows (and declares) which of its rules plays which role.
#
# ast-grep reports 0-indexed lines; SARIF startLine is 1-indexed, so we add 1. A stderr
# banner exercises clean-stdout capture.
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
            (if .metaVariables.single.FN then { func: .metaVariables.single.FN.text } else {} end)
            + (if .metaVariables.single.PKG then { symbol: .metaVariables.single.PKG.text } else {} end)
            + (if ((.ruleId // "") | test("hollow")) then { substantiveness_role: "hollow" }
               elif ((.ruleId // "") | test("referenced-symbol")) then { substantiveness_role: "referenced-symbol" }
               else {} end)
          ) as $props
          | if ($props | length) > 0 then { properties: $props } else {} end
        )
      ) ]
    }
  ]
}'
