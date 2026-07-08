---
title: "Directly-fixed issue closed via a typed artifact ref"
schema_version: issue/v1

issue:
  id: ISSUE-831
  title: "Directly-fixed issue closed via a typed artifact ref"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

resolved-by: ISSUE-838
---

# Directly-fixed issue closed via a typed artifact ref

## Problem

Fixed by work tracked in another artifact (ISSUE-838). The close traces to that
existing artifact via a typed resolved-by ref.

## Resolution

Resolved by ISSUE-838, which carries the resolving work. The typed ref resolves
to a real sibling artifact file.
