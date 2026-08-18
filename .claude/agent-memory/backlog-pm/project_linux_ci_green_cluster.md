---
name: linux-ci-green-cluster
description: DIR-024 now carries a 5+ item Linux-CI-green cluster (ISSUE-158/163/165/166/168) all traced to one CI run; they are item 1's (ISSUE-020) delivery residuals and default to clear fits, not new themes
metadata:
  type: project
---

Since ISSUE-163's `TestMain` fix (`970512b`) first let the suite run far enough
on GitHub's Linux runner, one CI run — **`32108003542`** — has produced a
family of issues that all home in `DIR-024 "Gate/Engine Quality"` as **item 1's
(ISSUE-020, the Linux sandbox) delivery residuals**: items 18 (ISSUE-158), 20
(ISSUE-163), 21 (ISSUE-165), 22 (ISSUE-166), 23 (ISSUE-168, `/dev/null` writes
denied by both sandbox profiles). Expect more from the same run.

**Why:** the Linux sandbox shipped via `PLAN-ISSUE-020` (completed) but was
never exercised by real CI until 2026-08-18. Everything it got wrong is
surfacing at once, from one report. `PLAN-ISSUE-020` is the lane that authored
most of the now-failing guards, which is exactly what makes these residuals
rather than new themes.

**How to apply:**
1. A new `pkg/packval` sandbox / Linux-CI failure from this run is a **clear
   fit for DIR-024 by default** — slot it, don't escalate. Reject DIR-032/033
   with the standing test: these go **loudly RED**, so no unearned verdict
   (fails DIR-032's charter), and DIR-033 homes follow-ons *filed by DIR-032
   member plans*, not this investigation's lineage.
2. Also reject `BUNDLE-021 "Pack Command Execution Governance"` even though its
   OQ-2 is literally *sandbox profile shape* — it owns the pack-execution trust
   **posture** as a design question; a concrete profile defect fix does not wait
   on it.
3. **Sandbox fixes are darwin-invisible** — `//go:build linux` guards and
   Landlock-only enforcement mean the only falsifier is Linux CI. Never promise
   Brandon a local red-to-green.
4. The cluster, not the directive, is the launch-relevant unit: DIR-024 is a
   23-item catch-all at BACKLOG.yml position 5, and promoting it drags eighteen
   unrelated items. If green Linux CI ever gates a release, the honest move is a
   **carve-out proposal** (the way DIR-032 was carved from DIR-024), not a
   reorder. Offered to Brandon 2026-08-18; not yet taken up.

Known loose end as of 2026-08-18: `TestLinuxSandbox_NetworkAllowedControlLeg
Succeeds` fails because TCP **and** UDP are blocked under a capability that
*permits* the network — a defect distinct from the `/dev/null` denials in its
own output, and **apparently still unfiled**. Check before assuming ISSUE-168
covers it.

See [[project_concurrent_pm_triage_races]], [[project_dir033_and_the_three_way_gate_boundary]],
[[project_relative_packdir_masquerades]].
