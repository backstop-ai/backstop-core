---
name: draft-spec-drift-is-silent
description: A refactor that falsifies a `status: draft` spec's prose reds NOTHING — reconcile at the extraction, not at the eventual close-out flip; and never anchor a cross-reference on a caller ORDINAL
metadata:
  type: feedback
---

When a refactor moves the function that houses a behavior, reconcile EVERY spec that
describes it in the same dispatch — including specs still at `status: draft`.

**Why:** the `contracts` and `requirement_traceability` gate dimensions read only
`implemented` specs, so drift in a draft spec is real, total, and completely silent.
Worse, the mandated test usually keeps passing BY DELEGATION (the old function still
exists with the same signature; it just delegates now), so nothing in the tree signals
it either. The drift surfaces for the first time at that spec's own close-out flip —
months later, with the context to resolve it gone. Observed 2026-08-16 on SPEC-035 when
PLAN-ISSUE-134 extracted `collectRequiredEngineTools` out of `provisionEngines`: four
sites in SPEC-035 asserted the walk and the trust gate lived in `provisionEngines`, and
`TestProvisionEngines_UnallowlistedToolFailsLoudBeforeProvision` stayed green throughout.

**How to apply:** on any extraction/move, grep every spec for the OLD function name, not
just the specs the brief names. Then hold the line on scope: correct the DESCRIPTION
(claim parentheticals, contract notes, caller tables, implementation bullets, review
questions) and add a `provides` entry for the newly-extracted function; do NOT delete
claims, rename mandated tests, or add/remove requirements. If the fix needs more than
that, STOP and report — that is a follow-on issue, not a silent rewrite. Record anything
seen-but-deliberately-untouched in the version history so a later reader knows it was
noticed, not missed.

**The sub-lesson worth generalizing:** never anchor a cross-reference on an ORDINAL
("the second `resolveEngineRegistry` caller", "the third caller"). An ordinal is
guaranteed to go stale the moment a caller is added, and no grep finds it because the
number is not the thing that changed. Name the function and its ROLE instead. Same class
of invisible gap as [[feedback_kind_function_contracts_existence_only]] and
[[feedback_omitted_subject_inherits_wrong_package]]: correct-looking text that no
mechanical check can falsify.
