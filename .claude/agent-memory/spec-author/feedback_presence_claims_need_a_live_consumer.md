---
name: presence-claims-need-a-live-consumer
description: A claim asserting "X is present in registry/map/allowlist" is only real if some runtime path actually reads that entry — trace the consumer's guard before writing it
metadata:
  type: feedback
---

Before writing a claim of the shape "T is present in <registry/map/allowlist> at <value>",
trace the CONSUMER and its guard. If every call site short-circuits before reading the
entry for that key, the claim tests a guarantee the code never delivers, and it becomes a
blocker for the eventual dead-code cleanup — the assertion is usually a `t.Fatalf`, so
deleting the entry turns the package RED and the cleanup stalls on the spec.

**Why:** SPEC-038 CLM-016 asserted `grep` AND `rg` are on `engine.TrustedToolAllowlist`.
`CheckToolAllowed` is reached only for engine bindings with a non-nil `Provision` block,
and nothing anywhere declares `provision:` for `rg` — so the `rg` half was unreachable
from the day it was written, and it blocked ISSUE-082's cleanup until the claim was
narrowed (2026-08-15).

**How to apply:** when a requirement adds an entry to a shared lookup table, write the
claim over the entry that has a real consumer and say explicitly in the requirement which
entries are PROHIBITED and why. If a sibling issue owns the removal of a dead entry, let
IT own the absence assertion — do not add a duplicate absence claim, or two artifacts
mandate competing tests over the same map. Record the exact test-body change (assertion to
drop, test to rename, header comment to fix) as a Sharp Edge so the implementer landing the
cleanup does it in one edit. Related: [[kind-function-contracts-existence-only]] — the gate
checks existence, not that the thing is wired to anything.
