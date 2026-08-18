---
name: linux-ci-residual-family
description: The v0.2.0 Linux-CI investigation family (158/163/164/165/166/168) all homes DIR-024 on the loud-red test; ISSUE-166's home is CONDITIONAL, and its grep cluster is candidate evidence for ISSUE-164's unmeasured question
metadata:
  type: project
---

Every defect surfaced by the v0.2.0 release investigation on GitHub's Linux
runner homes in `DIR-024 "Gate/Engine Quality"` — ISSUE-158 (item 18),
ISSUE-163 (20), ISSUE-165 (21), ISSUE-166 (22), ISSUE-168 (23) — on the same
test the directive has now drawn eight times: **loud red with a misattributed
or missing legible name is DIR-024; only "computes a result internally but
reports the wrong verdict about it" is DIR-032.** DIR-032 is done and DIR-033
homes only follow-ons *filed by DIR-032 member plans*, so neither competes for
this lineage. This family is item 1's (ISSUE-020, the Linux sandbox) tail.

**Why:** the corpus decides home on OBSERVED state, never on a hypothesized
downstream consequence — otherwise every red test with an untraced cause could
be argued into the verdict-honesty cluster.

**How to apply:**
- ISSUE-166 (`packs/contracts` fails its own phase3 self-validation on Linux;
  grep absence probes report 0 matches for present symbols) was slotted on the
  observed loud red, but its home is **CONDITIONAL** and the condition is
  written into item 22: if the grep cluster traces to the PRODUCTION contracts
  dispatch path — a real Linux consumer's forbidden-symbol check reporting no
  violation for a symbol that IS present — that is a silent false-clean and
  belongs to DIR-032's charter. Awaiting Brandon's ack (INBOX 2026-08-18T12:22Z).
- **The lead, verified in tree 2026-08-18 and worth reusing:** ISSUE-166's
  grep-cluster tests live in `pkg/pack/engine`, `pkg/pack/distribution` and
  `pkg/gate`, and **none of those three declares a `func TestMain`**. The first
  two are exactly the packages `ISSUE-164` names as invisible to ISSUE-163's
  guard and its pin `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`;
  ISSUE-164 is filed as a `question` because nobody measured whether tests there
  actually drive sandboxed dispatch on Linux. ISSUE-166 may be that measurement.
  Two limits: `cmd/backstop` also fails yet DOES carry the guard
  (`integration_test.go:38`), and `pkg/gate`'s contract tests touch neither
  packval nor the sandbox — so `pkg/gate` is a THIRD TestMain-less package
  ISSUE-164's inventory does not name.
- The whole family is darwin-invisible: `sandbox_nonlinux.go`'s
  `MaybeRunSandboxHelper()` is a bare `return nil`, so Linux CI is the only
  falsifier. Never accept "verified locally" on any of these.

Related: [[project_phase3_polarity_and_silent_parse]],
[[project_relative_packdir_masquerades]], [[project_concurrent_pm_triage_races]].
