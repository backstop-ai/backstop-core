---
title: "Obsoleted issue missing its obsoleted-by pointer"
schema_version: issue/v1

issue:
  id: ISSUE-821
  title: "Obsoleted issue missing its obsoleted-by pointer"
  type: enhancement
  status: obsoleted
  created: "2026-07-08"
  closed: "2026-07-08"
---

# Obsoleted issue missing its obsoleted-by pointer

## Problem

An obsoleted artifact must name the work that removed it. This fixture omits
`obsoleted-by`, so it must fail fail-loud on obsoleted-by-required.
