---
title: "Bare close with no delivered_by and no traceability chain"
schema_version: issue/v1

issue:
  id: ISSUE-908
  title: "Bare close with no delivered_by and no traceability chain"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"
---

# Bare close with no delivered_by and no traceability chain

## Problem

A closed issue with neither a delivered_by pointer nor its own requirements/
claims/verification. This must FAIL — full traceability is still required for
a non-plan-backed close.
