---
title: "Close carrying both delivered_by and resolved-by"
schema_version: issue/v1

issue:
  id: ISSUE-836
  title: "Close carrying both delivered_by and resolved-by"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-830
resolved-by: a1b2c3d4e5
---

# Close carrying both delivered_by and resolved-by

## Problem

At most one close pointer is allowed. Carrying both is ambiguous — no silent
precedence, no double-counting.

## Resolution

Deliberately carries BOTH delivered_by and resolved-by, so it must fail
fail-loud on close-pointer-conflict.
