---
name: dead-dispatch-fix-reds-its-consumer
description: Planning a fix that makes a dead check RUN — measure the real consumer's verdict by hand first; the consumer is usually wrong and its fix belongs in the same lane
metadata:
  type: project
---

When the defect is "this check never dispatches", the fix is only half the lane. The other
half is whatever the check will now say about the corpus it was blind to. Learned authoring
PLAN-ISSUE-142 (packval pattern-arg dispatch, 2026-08-17).

**Measure the consumer's verdict by hand BEFORE authoring, with the real tool.** Do not
reason about it. Reconstruct the argv the fix will produce and run it: for ISSUE-142 that
was `ast-grep run --json --pattern "<rule pattern>" <fixture>` per rule per fixture, piped
through the pack's own convert. Six of `packs/contracts`' seven rules turned out to have
their fixtures filed under INVERTED polarity slots (positive = must-NOT-fire is the
convention, stated on the findings seam in `RunFixtures` itself) — invisible for as long as
dispatch was dead, and instantly red once it isn't.

**The consumer fix belongs in the SAME lane, as a later phase.** Landing the dispatch fix
alone knowingly reds a shipped test — here
`TestInstallContractsLocalPack_ContractsPackPassesUnconditionalValidation`, plus every suite
that installs the source through `distribution.Add` (validation is unconditional on that
path). Order it: phase 1 core fix + verification that EXPECTS the collateral red, phase 2
the consumer correction, phase 3 full sweep. Say in the phase-1 verification task that a
GREEN consuming suite means dispatch did not actually go live. See
[[project_inherited_coverage_red_at_closeout]] for the separate case of red you did not cause.

**Check whether the argv shapes ALREADY agree before scoping an argv change.** The obvious
task ("add a pattern-arg branch to `buildEngineArgv`") was wrong: packval's builder appends
`input_flag` once then all targets, the gate's `gatherEngineInputs` emits `input_flag,
pattern` then targets — identical once target[0] is the pattern. Tracing both sides removed
a file from scope and kept the lane disjoint from a sibling plan editing that same file.

**A polarity-inverted fixture pair usually cannot be fixed by swapping slots.** `packs/contracts`
authored NAME-AGNOSTIC self-test patterns against fixtures built for NAME-SPECIFIC compiled
ones, so the "mismatch" file still matches `type $NAME $$$`. The fix was new genuinely-clean
fixtures, and the old files had to be KEPT on disk because another package's tests read them
by name. Grep for every reader before deleting a fixture a manifest stops referencing.
