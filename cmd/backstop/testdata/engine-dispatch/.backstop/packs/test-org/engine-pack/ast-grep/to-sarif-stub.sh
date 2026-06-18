#!/bin/sh
# Stub ast-grep stdin->SARIF converter for the dispatch-PLUMBING proof (CLM-030).
# It transforms nothing: it drains stdin and emits canned SARIF on stdout, plus a
# banner on stderr to prove clean-stdout capture does not corrupt the SARIF bytes.
cat >/dev/null
echo "to-sarif-stub: converting (this banner must not corrupt SARIF)" >&2
cat <<'SARIF'
{
  "version": "2.1.0",
  "runs": [
    {
      "results": [
        {
          "ruleId": "ast-grep-proof",
          "level": "error",
          "message": { "text": "forbiddenCall is not allowed" },
          "locations": [
            {
              "physicalLocation": {
                "artifactLocation": { "uri": "main.go" },
                "region": { "startLine": 7 }
              }
            }
          ]
        }
      ]
    }
  ]
}
SARIF
