#!/bin/bash
# validator.sh <fixture-path>
# Checks that a Go package directory containing Slack HTTP route registrations
# also contains a call to slack.NewSecretsVerifier.
#
# input_scope: multi-file
# category: presence
#
# Exit 0 = pass (no violation — verification is present)
# Exit 1 = fail (violation — verification is absent)

set -euo pipefail

FIXTURE_DIR="$1"

if [ ! -d "$FIXTURE_DIR" ]; then
  echo "error: fixture path is not a directory: $FIXTURE_DIR"
  exit 2
fi

# Check if this package registers Slack routes (heuristic: contains "/slack/" in route strings)
has_slack_routes=false
if grep -rql '"/slack/' "$FIXTURE_DIR" --include="*.go" 2>/dev/null; then
  has_slack_routes=true
fi

if [ "$has_slack_routes" = false ]; then
  # Not a Slack handler package — rule does not apply, pass
  exit 0
fi

# Package registers Slack routes — check for signature verification presence
if grep -rql 'NewSecretsVerifier' "$FIXTURE_DIR" --include="*.go" 2>/dev/null; then
  # Verification present — pass
  exit 0
fi

echo "VIOLATION: Package at $FIXTURE_DIR registers Slack routes but contains no call to slack.NewSecretsVerifier"
exit 1
