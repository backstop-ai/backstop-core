---
title: "Delivered_by close missing the required Resolution section"
schema_version: issue/v1

issue:
  id: ISSUE-907
  title: "Delivered_by close missing the required Resolution section"
  type: enhancement
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"

delivered_by: PLAN-ISSUE-907
---

# Delivered_by close missing the required Resolution section

## Problem

This issue points at a valid completed plan (PLAN-ISSUE-907) but deliberately
omits the `## Resolution` section. It must FAIL the minimum-standalone-content
rule even though the backing plan is otherwise a clean delivered trace.
