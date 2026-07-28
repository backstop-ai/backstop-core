---
name: rederive-retirement-class
description: A plan's "exactly N sanctioned retirements" is a stale enumeration — scan for the CLASS mechanically before editing, and never absorb an extra
metadata:
  type: feedback
---

When a plan sanctions a fixed number of test retirements (or any "closed edit
set"), re-derive the set by scanning for the defining CLASS before editing —
never trust the enumerated list. In PLAN-SPEC-055 Phase 10 the plan named THREE
absent-dependency-premise retirements; a one-line grep for nil dependency
assignments across the six suites found FIVE MORE of the identical class
(`TestPackAdd_NilValidatorSkipsValidation`, `TestPackUpdate_NilValidator...`,
`TestPackUpgrade_SkipsValidationWhenValidatorNil` / `_NoRemediationWhenGeneratorNil`
/ `_SkipsScanningWhenScannerNil`).

**Why:** the enumeration was made against an older snapshot of a moving corpus,
and the plan itself says a fourth means "the enumeration missed a class, which is
worth knowing before it is absorbed silently." Silently absorbing extras is how
coverage disappears under a sanction.

**How to apply:** derive the class predicate from the plan's own reasoning ("a
test whose premise is an absent dependency"), grep for it, and diff against the
enumerated list. If you find extras: report to the lead, leave them UNTOUCHED and
still compiling (during a transitional phase the old free functions usually still
exist, so an unmigrated test keeps passing), and keep migrating everything else —
stopping cold wastes the phase, absorbing wastes the guard. Distinguish a test
whose SUBJECT is the absence (unmigratable) from one that merely passes nil
incidentally (migrates fine with a permissive mock) — see
[[project_no_grandfather_changed_files]] and [[feedback_netnegative_gate_baseline]].
