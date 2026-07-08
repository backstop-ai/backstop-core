---
title: "Obsoleted issue with an absent mandated test"
schema_version: issue/v1

issue:
  id: ISSUE-840
  title: "Obsoleted issue with an absent mandated test"
  type: enhancement
  status: obsoleted
  created: "2026-07-08"
  closed: "2026-07-08"

obsoleted-by: ISSUE-018

claims:
  - id: CLM-001
    tests:
      - TestObsoletedGhost
---

# Obsoleted issue with an absent mandated test

## Problem

Delivered then removed. Its mandated test TestObsoletedGhost no longer exists.
As a retired (obsoleted) terminal it is excluded from the drift broken-promise
check — the absent test is NOT a broken promise.
