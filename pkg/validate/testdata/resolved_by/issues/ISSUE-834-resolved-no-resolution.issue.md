---
title: "Resolved-by close with no Resolution section"
schema_version: issue/v1

issue:
  id: ISSUE-834
  title: "Resolved-by close with no Resolution section"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

resolved-by: 0f1e2d3c4b5a
---

# Resolved-by close with no Resolution section

## Problem

A resolved-by close must still carry a Resolution section for standalone
readability. This fixture deliberately omits it, so it must fail on
resolved-by-resolution-required.
