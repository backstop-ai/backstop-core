---
name: linux-ci-residual-family
description: The v0.2.0 Linux-CI investigation family (158/163/164/165/166/168/177) all homes DIR-024 on the loud-red test; ISSUE-166's home is CONDITIONAL, and ISSUE-177's green/red split is measured evidence for ISSUE-164's unmeasured question
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
- **★ ISSUE-177 (2026-08-18, DIR-024 item 27) turned that lead into a
  measured correlation — reuse it before spending a `debug/*` branch.**
  ISSUE-166's `-H -I` fix left exactly ONE test still failing
  `phase3-fixtures: 14 validation error(s)` byte-identically, while ~a dozen
  structural siblings cleared. Verified in tree: every CLEARED sibling lives
  in a package that declares `func TestMain` — `cmd/backstop`
  (`integration_test.go:19`) and `pkg/packval` (`main_test.go:36`); the lone
  RED one lives in `pkg/pack/distribution`, which declares none. So the
  cheap first move on any surviving phase3 red is "does its package declare
  TestMain?", and confirming it also answers ISSUE-164. Still a hypothesis:
  ISSUE-166's grep cluster failed in `cmd/backstop`, which DOES carry the
  guard.
- **Filings in this family cite paths that do not exist — check before
  slotting.** ISSUE-177's References place the failing test at
  `pkg/pack/engine/contracts_local_install_test.go`; the real file is
  `pkg/pack/distribution/contracts_local_install_test.go:51` (CLM-092).
  `ls` every path a filing names; record the correction in the directive
  item AND flag that the issue text itself still needs issue-author.
- **Lifetime caveat on anything touching `packs/contracts`:** DIR-027
  thread-1 tier 2 (undelivered, position 4 — AHEAD of DIR-024) deletes
  `packs/contracts` and de-vendors
  `pkg/pack/distribution/contracts_local_install.go`, which would delete
  ISSUE-177's test outright. Home in DIR-024 anyway (live loud red on the
  gate/engine path), but say "revisit as mooted-by-deletion if tier 2 lands
  first" in the item.
- The whole family is darwin-invisible: `sandbox_nonlinux.go`'s
  `MaybeRunSandboxHelper()` is a bare `return nil`, so Linux CI is the only
  falsifier. Never accept "verified locally" on any of these.

Related: [[project_phase3_polarity_and_silent_parse]],
[[project_relative_packdir_masquerades]], [[project_concurrent_pm_triage_races]].
