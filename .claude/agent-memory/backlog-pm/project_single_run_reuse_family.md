---
name: single-run-reuse-family
description: The go-toolchain coverage-reuse (single-run) mechanism — its guard test manufactures an inverted mtime relation, so darwin-green proves nothing; check chronology before believing any "verified locally" perf claim
metadata:
  type: project
---

The go-toolchain single-run/coverage-reuse family (ISSUE-068 → ISSUE-172 →
PLAN-ISSUE-172 → ISSUE-179) shipped a speedup that was a **complete no-op on
Linux CI**, and its own "executable falsifier" is what let it through.

**The mechanism.** `scripts/test-produce.sh` runs `go test -coverprofile=cover.out
./...` (`:37`) then stamps `.backstop/go-coverage-fresh` (`:58`) — profile FIRST,
stamp AFTER, always. `scripts/coverage-produce.sh:38` reuses the profile only
`if [ ! cover.out -ot "$stamp" ]`, i.e. reuse iff the profile is not older than
the stamp — which the real chronology can never satisfy.

**Why darwin hid it (the reusable trap):** macOS `/bin/sh` `test -ot`/`-nt`
compares at whole-SECOND resolution, so a few-ms gap TIES and the backwards
condition passes by accident. Ubuntu `dash` compares at nanoseconds and gets it
right every time. **Any mtime-ordering logic verified only on darwin is
unverified.** Same shape as the darwin/Linux split in
[[project_linux_ci_residual_family]] and [[project_relative_packdir_masquerades]].

**Why: the guard test is a vacuous green.**
`cmd/backstop/gotoolchain_single_run_test.go`'s
`TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile` writes `cover.out`,
writes the stamp, then **ages the stamp 1s via `os.Chtimes`** — manufacturing the
one filesystem state production cannot produce, and documenting the inversion in
its own comment as "a deliberate mtime nudge". A second guard,
`pkg/pack/engine/gotoolchain_installed_pack_singlerun_test.go`, asserts the
producer text merely *contains* `-ot` (a spelling assertion, cf.
[[project_ast_wiring_guards_assert_spelling]]) and bars the version above 1.6.0.

**How to apply:**
- A filing that says "fix lives entirely in the external pack repo" for anything
  in this family is **wrong** — grep `cmd/backstop/gotoolchain_single_run_test.go`
  and `pkg/pack/engine/gotoolchain_installed_pack_singlerun_test.go` first; they
  pin the expression and the version bar, so backstop-core always changes too.
- The stamp is only consumed (`rm -f`) **if `coverage-produce.sh` actually runs**.
  An interrupted `gate --all` leaves a stamp behind; a later narrowed run
  overwrites `cover.out` with a PARTIAL profile. So "presence of stamp is
  sufficient" is false, and any pure mtime ordering still allows a stale
  whole-module profile. Binding the stamp to the profile (mtime/hash written into
  the stamp) is the only complete shape.
- Homing: correct verdict, unremoved COST ⇒ DIR-024 (precedent: its own item 7,
  ISSUE-099 "measured 2x CI cost"), never DIR-032 — a Go unit test that cannot
  falsify is not a gate dimension computing a false verdict. See
  [[project_gate_verdict_honesty_cluster]] for the boundary.
- ISSUE-179 is DIR-024 item 28 (slotted 2026-08-19); ISSUE-172/ISSUE-068 are
  closed and cited by no directive.
