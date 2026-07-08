---
title: "Close with a typed resolved-by that resolves to no file"
schema_version: issue/v1

issue:
  id: ISSUE-833
  title: "Close with a typed resolved-by that resolves to no file"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

resolved-by: SPEC-999
---

# Close with a typed resolved-by that resolves to no file

## Problem

A typed ref must resolve to a real artifact file. SPEC-999 is well-shaped but
names no artifact under a sibling specs/ dir.

## Resolution

Resolved by SPEC-999, which does not exist. Must fail on
resolved-by-artifact-not-found.
