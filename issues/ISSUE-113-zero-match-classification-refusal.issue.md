---
title: "Classification matching zero test files should be a loud config-error refusal, not silent mass join violations"
schema_version: issue/v1

issue:
  id: ISSUE-113
  title: "Classification matching zero test files should be a loud config-error refusal, not silent mass join violations"
  type: enhancement
  status: open
  created: "2026-07-29"
---

# Zero-match classification: refuse loudly

## Problem

When a pack's classification globs match ZERO test files, the substantiveness join silently emits a
"does not call package X" violation for EVERY mandated test — hundreds of misleading findings whose
real cause (empty classification) is named nowhere. Hit twice in one week by bclabs-portal: (1) the
published typescript-substantiveness 1.1.0 shipping harness-baked globs (397 false violations), and
(2) the missing-ast-grep case (same signature, different root). Both cost hours; both would have been
one line: "classification matched 0 test files".

## Direction

Extend the ISSUE-020 config-error refusal philosophy: when mandated tests exist but the classifier
matches zero test files (or the substantiveness evidence set is empty while mandated tests exist),
the step REFUSES with a config-error naming its cause instead of emitting per-test violations.
Founder-ack'd (Brandon, 2026-07-28) for slotting per PM flow (DIR-024 recommended).
