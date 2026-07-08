---
title: "Obsoleted issue with a malformed obsoleted-by"
schema_version: issue/v1

issue:
  id: ISSUE-822
  title: "Obsoleted issue with a malformed obsoleted-by"
  type: enhancement
  status: obsoleted
  created: "2026-07-08"
  closed: "2026-07-08"

obsoleted-by: garbage
---

# Obsoleted issue with a malformed obsoleted-by

## Problem

`obsoleted-by` must be a typed artifact id. This fixture uses arbitrary prose,
so it must fail on obsoleted-by-malformed.
