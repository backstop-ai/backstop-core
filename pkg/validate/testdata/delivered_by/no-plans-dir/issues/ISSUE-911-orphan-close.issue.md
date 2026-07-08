---
title: "Delivered_by failure fixture ISSUE-911"
schema_version: issue/v1

issue:
  id: ISSUE-911
  title: "Delivered_by failure fixture ISSUE-911"
  type: enhancement
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-911
---

# Delivered_by failure fixture ISSUE-911

## Problem

delivered_by names PLAN-ISSUE-911, but this subtree has no sibling plans/ dir, so plan resolution must fail fail-loud.

## Resolution

Present so the only validation defect is the delivered_by value under test.
