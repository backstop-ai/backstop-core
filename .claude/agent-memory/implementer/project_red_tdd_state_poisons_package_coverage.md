---
name: red-tdd-state-poisons-package-coverage
description: A deliberately-RED test makes `go test` exit 1, which kills the coverage producer's profile for the WHOLE package — every changed file in it then reds coverage_unmeasured; it is a phantom that self-clears when the phase goes green
metadata:
  type: project
---

Mid-plan, between a TDD red phase and its green phase, `backstop gate` reports

    coverage_unmeasured | <your new file>.go
    "no coverage measurement for in-scope changed measurable-source file ...
     and it is not pack-declared excluded — refusing to pass with nothing measured"

on files that are perfectly well covered. Mechanism: the red test makes
`go test ./<pkg>` exit non-zero, the coverage producer's profile for that
package is never produced, and so EVERY changed file in the package reads as
unmeasured — not just the failing test's subject.

Measured on ISSUE-098: two deliberately-red `TestDriftGate_*` tests produced a
`coverage_unmeasured` red on `cmd/backstop/pack_claim_index.go`, a file whose
own tests were green. It vanished the moment the wiring task turned the two
tests green. Nothing was written to fix it.

**Why:** it is easy to misread this as the new-file coverage floor
([[project_new_file_coverage_floor]]) and "fix" it by writing coverage that was
never missing — burning a cycle and adding tests for the wrong reason. The two
look identical in the gate output; only the package's test exit code
distinguishes them.

**How to apply:** when `coverage_unmeasured` fires on a file during a red phase,
check `go test ./<pkg>` FIRST. Non-zero exit with unrelated failures means the
finding is a phantom — note it, finish the green phase, and re-check at the
phase boundary. It is a real finding only if it survives a green suite. Same
family of trap as [[project_subprocess_e2e_earns_no_coverage]] and
[[project_buildtag_file_never_measurable]], where the coverage signal is absent
for a reason that has nothing to do with how well the file is tested.

Sibling-lane variant, measured ISSUE-146 (2026-08-17): the poisoning package
failure does not have to be YOURS. `gate --all` reported 17 `coverage_threshold`
reds spread across nearly every non-test file in `cmd/backstop`
(`pack_new.go 8/30`, `waiver.go 6/55`, `pack_separation.go 0/28` …) while the
same files' coverage was fine — the cause was five FAILING tests in that package
owned by two other lanes (a sibling's in-flight `packs/substantiveness/testdata`
edits plus ISSUE-147). The tell is the SHAPE: a coverage red that lands on a
whole package at once, including files no lane touched, is a degraded profile,
not 17 independent regressions. Confirm by finding the package's failing tests
and proving them inherited, not by writing coverage. A `gate --file` run scoped
to your own files passed `coverage_threshold` in the same tree, same minute.

Corollary for shared trees: a lone `go test` CrashGuard red
("engine \"go test\" crashed: non-zero exit with no parseable findings") can also
be a TRANSIENT from a sibling agent editing another package while your gate
compiles the module. Re-run before believing it — ISSUE-098 saw exactly one such
red clear on an immediate re-run with no tree change.
