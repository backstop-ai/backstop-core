---
name: test-setup-fabricates-impossible-state
description: A green shipped test can sit over a dead mechanism because its SETUP manufactures a state production never produces — read the arrange block, not just the asserts
metadata:
  type: project
---

When planning a fix for a mechanism that already has shipped, green tests asserting it
works, read the test's **arrange block** for state production cannot produce. The
assertions can be perfectly correct and the test still proves nothing.

**Why:** ISSUE-179 (2026-08-19). `coverage-produce.sh`'s reuse check was directionally
backwards — it demanded `cover.out` be no-older-than the stamp, while `test-produce.sh`
writes `cover.out` *first* and touches the stamp *after*, so the condition was never
true at real precision and the whole PLAN-ISSUE-172 speedup was a no-op on Linux CI.
CI was green the whole time because the shipped guard
`TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile` did
`os.Chtimes(stamp, time.Now().Add(-1*time.Second))` — aging the stamp backwards a full
second, i.e. constructing the exact inverse of the production chronology, at a
magnitude coarse enough to satisfy the backwards check under dash *and* macOS sh. Its
own comment named the wrong relation as the thing under test. The script bug and the
test bug were one defect with two halves; fixing only the script leaves the accomplice.

**How to apply:**
- For any mechanism whose correctness depends on an *ordering* (mtimes, sequence
  numbers, versions, write-then-read), derive the ordering from the producing code and
  check the test constructs THAT ordering. `Chtimes`/`touch -d`/hand-set timestamps in
  an arrange block are the tell.
- Make the primary falsifier **precision-independent**: reproduce the true direction at
  a magnitude even a coarse clock resolves (e.g. a 2s gap), so it reds on the developer's
  own machine. Add the realistic sub-second leg *on top*, and say in the plan which legs
  can and cannot go red on which platform — don't let "all legs red" be reported from a
  machine where one structurally can't be.
- A `strings.Contains(script, "-ot")`-style presence pin is satisfied by the backwards
  form and the correct form alike. Pin the **operand order** (require the right shape,
  forbid the wrong one), same rule as [[project_drift_guard_forbid_field_access]].
- **A probe/matrix test must READ its subject from the artifact under test, never
  hardcode it.** Caught in ISSUE-179 plan review: a shell-matrix test evaluating the
  reuse condition had no stated condition *source*, and both obvious implementations
  were broken — hardcoding the FIXED expression makes it permanently green and unable
  to distinguish a fixed producer from a defective one (the same weak-pin class it was
  written to prevent); hardcoding the BROKEN expression makes it permanently red,
  because no later task's `files:` scope touches a Go literal. Only extracting the
  expression from the script at run time gives the red-before / green-after trajectory
  the plan promises. Always name the extraction anchors and require a **hard fail** when
  extraction matches nothing — a silent fallback re-vacuums the test on the next
  reshaping of the subject.
- Preserve the mandated test NAME and rewrite its BODY —
  [[project_second_leg_preserves_mandated_set]]. Renaming it reads as a broken promise
  on the completed plan that mandated it.
