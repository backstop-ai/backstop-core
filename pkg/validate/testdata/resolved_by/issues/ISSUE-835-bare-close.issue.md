---
title: "Bare close with no close pointer at all"
schema_version: issue/v1

issue:
  id: ISSUE-835
  title: "Bare close with no close pointer at all"
  type: bug
  status: closed
  created: "2026-07-08"
  closed: "2026-07-08"
---

# Bare close with no close pointer at all

## Problem

A closed issue carrying neither delivered_by nor resolved-by must still run the
full REQ->CLM->tests rigor — the relaxation is conditional, not a general
loosening.

## Resolution

Intentionally none of the traceability chain is present, so the bare close must
still report requirements-required / claims-required.
