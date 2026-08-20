---
name: linux-ci-residual-family
description: The v0.2.0 Linux-CI investigation family (158/163/164/165/166/168/177/180) all homes DIR-024 on the loud-red test; ISSUE-180 CONFIRMED the missing-TestMain re-exec mechanism and ruled pkg/gate OUT; ISSUE-166's home stays CONDITIONAL
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
- **★ RESOLVED 2026-08-19 by ISSUE-180 (DIR-024 item 29) — the TestMain lead
  was RIGHT, and `pkg/gate` was WRONG. Stop repeating the `pkg/gate` half.**
  Verified in tree: `grep -rl "backstop-core/pkg/packval" --include="*.go"`
  over the whole module hits EXACTLY FOUR directories — `cmd/backstop`,
  `pkg/pack/distribution`, `pkg/pack/engine`, `pkg/packval`. **`pkg/gate` does
  not import packval by any file**, so it can never be a re-exec target; the
  earlier note here calling it "a THIRD TestMain-less package ISSUE-164's
  inventory does not name" was a false lead I wrote and have now retired.
  Module-wide `grep -rn "^func TestMain"` returns only
  `pkg/packval/main_test.go:36` and `cmd/backstop/integration_test.go:19`
  (plus one `pkg/gate/testdata` fixture) — so of the four packval importers,
  `pkg/pack/distribution` and `pkg/pack/engine` are the only exposed ones,
  exactly ISSUE-164's original pair.
- **The mechanism, now confirmed and reusable:** the Linux sandbox is a
  re-exec trampoline spawning `os.Executable()` — under `go test`, the CALLING
  PACKAGE'S OWN test binary — with `BACKSTOP_SANDBOX_HELPER_SPEC` set. No
  `TestMain` to intercept it ⇒ Go's default main ⇒ the child **re-runs that
  package's entire suite** in the scratch-copy cwd, dies off any `go.mod`
  ancestry, exits 1. Go writes it to **stdout**, so
  `foldHelperStderrIntoError` sees empty stderr and says "wrote no
  diagnostic." That signature — exit 1 + empty diagnostic + N errors where N =
  every fixture in the pack — IS this defect. First diagnostic move on any
  such red: does the package declare `TestMain`?
- **ISSUE-164 CLOSURE HAZARD — flag it every time.** ISSUE-180 promotes only
  the `pkg/pack/distribution` half of ISSUE-164's question. `pkg/pack/engine`
  stays unconfirmed, AND the guard's structural blind spot is untouched:
  `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain` builds its
  roster via `scanGoPackages` then `if pkg.testMain == nil { continue }` — so
  a packval-importing package with NO `TestMain` is invisible to it BY
  CONSTRUCTION, which is exactly how this shipped. ISSUE-164 must not be
  closed when PLAN-ISSUE-180 lands; 180 will look like it answers 164 and
  does not.
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
