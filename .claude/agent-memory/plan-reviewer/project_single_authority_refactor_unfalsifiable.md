---
name: single-authority-refactor-unfalsifiable
description: "Route duplicate X through one shared predicate" claims are behaviorally unfalsifiable — the agreement test is green before AND after, so the refactor task can be skipped entirely
metadata:
  type: project
---

A plan claim of the form "consumer C consumes the shared predicate P instead of its own
inline copy" is a STRUCTURAL claim, and a behavioral "assert C's verdict agrees with P"
test cannot see it: the two implementations are identical by construction, so the test is
green before the refactor lands and green after. The refactor task could be deleted and
every mandated test still passes.

**Why:** caught on PLAN-ISSUE-151 (CLM-008 / TASK-008-009, 2026-08-17). The plan
explicitly *rejected* a source-text guard ("breaks the moment someone reformats the line")
and mandated an agreement test instead — which closes zero drift, because drift is exactly
the case where the verdicts differ, and the test only fires after the drift already shipped.

**How to apply:** when a plan carries a single-authority / de-duplication claim, ask what
observation distinguishes "landed" from "not landed". If both sides are pure functions of
the same input, only a structural check (source-text, or deleting the inline branch so the
package stops compiling without the import) can prove it. Accept the reformatting fragility
or drop the claim — do not accept an agreement test as proof.

Second, independent trap on the same pair: the consumer helper often has NO injection seam
(`ciGlobScopingProblems` derives its input by parsing real pack files off disk) AND an
early-return before the branch under test. Verify the mandated test is even WRITABLE against
the existing signature before assessing whether it proves anything — see
[[project_shortcircuit_dependent_guard]].
