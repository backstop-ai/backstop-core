---
title: "Delivered_by failure fixture ISSUE-906"
schema_version: issue/v1

issue:
  id: ISSUE-906
  title: "Delivered_by failure fixture ISSUE-906"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-906
---

# Delivered_by failure fixture ISSUE-906

## Problem

delivered_by names PLAN-ISSUE-906, a completed plan whose spec_id is ISSUE-999 (backs a different issue).

## Resolution

Present so the only validation defect is the delivered_by value under test.
