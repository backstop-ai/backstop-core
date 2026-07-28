#!/usr/bin/env bash
# SessionStart: surface PM state so nobody has to remember to look.
INBOX=".backstop/pm/INBOX.md"
if [[ -s "$INBOX" ]]; then
  N="$(grep -c '^## ' "$INBOX" 2>/dev/null || echo 0)"
  [[ "$N" -gt 0 ]] && echo "backlog-pm: $N item(s) awaiting Brandon in $INBOX (escalations + proposals). Surface these if relevant."
fi
exit 0
