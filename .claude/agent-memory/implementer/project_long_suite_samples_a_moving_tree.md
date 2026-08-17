---
name: long-suite-samples-a-moving-tree
description: A multi-minute full-suite run in the shared tree samples a MOVING tree, so sibling mid-edit states surface as failures attributed to your lane; re-run before believing them
metadata:
  type: project
---

A full-package `go test` run that takes minutes (cmd/backstop is ~300s) does NOT observe a
single tree state. Concurrent lanes commit and edit DURING the run, so a failure can belong
to a sibling's transient mid-edit state and still land in your output looking like yours.

**Why:** measured on PLAN-ISSUE-124 (2026-08-16). A 281s `go test ./cmd/backstop/` reported
8 failures in `pack_gate_producer_findings_test.go` — an UNTRACKED file belonging to an
active sibling lane whose implementation had not landed yet. Building a detached control
worktree at HEAD and copying the sibling's uncommitted work in WITHOUT my change showed
their tests PASSING; re-running in the shared tree minutes later also passed. The sibling
had landed their implementation mid-run. Nothing was wrong with my change, and nothing was
wrong with their code by the time I looked — the run had simply straddled the transition.

**How to apply:** before attributing ANY failure from a long run, re-run just those tests.
If they pass, it was a moving-tree artifact — say so explicitly rather than quietly
dropping it. If they still fail, THEN build the control (see
[[feedback_never_stash_shared_tree]] — detached worktree at HEAD, copy the sibling's
uncommitted files in, never stash). A failing test in a file that is UNTRACKED (`git status
--porcelain` shows `??`) is a sibling's red-phase TDD by default, not your regression:
check tracked-ness first, it is one command and it usually settles the question.

The same effect hits `backstop gate`: its go-build/go-test engines run over the whole
module for tens of seconds, so a sibling mid-write yields an opaque
`non-zero exit with no parseable findings: exit status 1` even though `go build ./...` run
by hand exits 0 (the pack's convert script reads STDOUT while the compiler writes to
STDERR, and pkg/check's `RunStdout` discards stderr — see
[[project_information_existed_surfacing_did_not]]). Do not chase that as a real build
break; re-run it.

★ **THE HEURISTIC CUTS BOTH WAYS, AND I OVER-APPLIED IT THE SAME NIGHT.** "Blame the
sibling" is as unfounded as "blame myself" until a control run says so. Later in the same
lane, a full-chain gate showed 14 failing cmd/backstop tests all tracing to
`pack test ... phase3-fixtures: N validation error(s)`. I attributed it to the sibling's
in-flight packval work because I could see their gate running in `ps` — and I was WRONG. A
worktree at clean HEAD, with NOBODY's uncommitted work present, reproduced all 14. The
breakage was in the COMMITTED baseline. I had to retract the attribution to the lead.

So the rule is symmetric: **the control worktree at clean HEAD is what settles attribution,
in EITHER direction.** `??` in `git status` is a hint about which control to build, never a
verdict on its own — a sibling can commit their breakage, at which point "untracked" stops
being the discriminator. Run HEAD before naming a culprit, especially before naming one to
a teammate; retracting a confident attribution costs more than the ten minutes the control
would have taken.

The same worktree-at-HEAD move also settles COVERAGE attribution, which is otherwise pure
guesswork: measure the file's `covered/total` at HEAD and with your change. On this lane it
showed `gate_substantiveness_e2e.go` at 74.4% BEFORE my change (already under the 80%
floor) and 73.1% after — proving the red was inherited and that shedding every statement I
added could not have made it green. Arithmetic guesses about "what it would have been"
are not evidence; the second measurement is.

★ **IT SAMPLES YOUR OWN MOVING TREE TOO, NOT ONLY THE SIBLINGS'.** PLAN-ISSUE-151
(2026-08-17): I launched a diff-scoped gate, then — while it ran — deliberately backed
the implementation out of `phase2.go` for a few minutes to get an honest TDD red on the
next task's tests. The gate took 22 minutes under contention and sampled that window. Its
report came back naming 7 `unused` findings on my own new file plus 3 failing mandated
tests, all of which described a state that no longer existed by the time I read it. Every
one was a phantom, and they looked exactly like real regressions in my own code.

**How to apply:** a gate or full suite launched in the background is a snapshot of
whenever it happened to run, so do not edit the files it is measuring while it runs — and
if you did, discard the run and relaunch rather than triaging its findings. Timestamp
discipline settles it: compare the run's start time against your own edit times before
believing any finding about a file you touched mid-run.

Related: [[project_init_gate_guard_fires_on_sibling_lanes]],
[[project_shared_tree_assertions_cannot_attribute]],
[[project_red_tdd_state_poisons_package_coverage]].
