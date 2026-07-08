---
title: "Close with a free-text resolved-by (vacuous-close hatch)"
schema_version: issue/v1

issue:
  id: ISSUE-832
  title: "Close with a free-text resolved-by (vacuous-close hatch)"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

resolved-by: "just fixed it"
---

# Close with a free-text resolved-by (vacuous-close hatch)

## Problem

Free text is not a structured ref. Accepting it would open a vacuous-close
hatch that bypasses REQ->CLM->test rigor and is invisible to the drift
dimension.

## Resolution

"just fixed it" — deliberately arbitrary prose. Must fail on
resolved-by-malformed.
