---
name: synthetic-zerogap-arrangement-ties
description: A hand-written write-then-touch mtime arrangement ties ~95% of the time and hides an mtime-ordering defect; arrange through the REAL producer so the process-teardown gap is present
metadata:
  type: project
---

Reproducing an mtime-ordering defect by writing the two files back-to-back in a shell or
in Go (`: > a; : > b`, or two `os.WriteFile` calls) is **not** a faithful model of a
production chronology whose two writes are separated by a subprocess.

**Measured (ISSUE-179, 2026-08-19, `ubuntu:24.04` dash):** a zero-gap arrangement tied on
exact nanosecond mtime **191 times out of 200**, so a directionally-backwards `-ot` check
reported the "wrong" answer only 4.5% of the time and the defect looked non-reproducible.
Arranging the same state by running the REAL `test-produce.sh` (whose `go` subprocess
writes the profile and then exits) produced a ~5ms gap and a **deterministic 0/50**
reproduction. Post-fix: 50/50.

**Why:** the kernel stamps mtimes from a coarse clock (~1ms granularity on ext4/overlayfs);
a zero-gap pair lands inside one tick, a process teardown straddles one.

**Why:** it matters beyond one issue — a plan can specify the unsound arrangement in good
faith and pair it with a "if it reports X, the diagnosis is wrong, STOP" instruction. That
combination turns a sound diagnosis into a false abort. PLAN-ISSUE-179 TASK-001 did exactly
this.

**How to apply:** when falsifying anything that compares two files' mtimes, arrange the
state by running the real producer, never by hand. If you must hand-arrange, set the mtimes
EXPLICITLY with `os.Chtimes`/`touch -d` at a magnitude wider than the filesystem's
granularity (2s is safe) — that is what makes a leg platform-independent. And distrust any
single-trial mtime observation: run it 50x and report the ratio.

Related: [[project_predict_the_number_before_the_run]], [[feedback_fixtures_from_real_output]].
