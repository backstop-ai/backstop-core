---
title: "Directly-fixed issue closed via a commit ref"
schema_version: issue/v1

issue:
  id: ISSUE-830
  title: "Directly-fixed issue closed via a commit ref"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

resolved-by: a1b2c3d4e5
---

# Directly-fixed issue closed via a commit ref

## Problem

Fixed directly by a single commit — no issue->plan track and no test. A
completed backing plan (delivered_by) cannot apply, so the close traces to the
resolving commit via resolved-by.

## Resolution

Resolved directly by commit a1b2c3d4e5. There is no backing plan and no
mandated test; the fix was a self-contained direct commit.
