#!/usr/bin/env bash

RULES_DIR=".backstop/rules"
[[ ! -d "$RULES_DIR" ]] && exit 0

manifests=("$RULES_DIR"/*.manifest.json)
[[ ! -e "${manifests[0]}" ]] && exit 0

echo "Available standards:"
for manifest in "${manifests[@]}"; do
  std_id=$(/usr/local/bin/jq -r '.standard_id // empty' "$manifest" 2>/dev/null)
  language=$(/usr/local/bin/jq -r '.language // "unknown"' "$manifest" 2>/dev/null)
  rule_count=$(/usr/local/bin/jq -r '.rules | length' "$manifest" 2>/dev/null)
  [[ -n "$std_id" ]] && echo "  $std_id ($language, $rule_count rules)"
done

exit 0
