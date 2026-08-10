---
name: buildtag-file-never-measurable
description: A file tagged for a platform CI never compiles (e.g. !linux && !darwin) is PERMANENTLY coverage_unmeasured while in diff scope — unlike a linux-tagged file, which resolves on the runner; fold the arm into a tag the CI matrix DOES compile
metadata:
  type: project
---

`coverage_unmeasured` fires per FILE, not per percentage: "no coverage
measurement for in-scope changed measurable-source file X (any metric)". A file
excluded by build tags is not compiled, so the producer emits no record and the
file reds for as long as it stays in diff scope.

Measured 2026-07-28 (ISSUE-020 phase 3b). `//go:build linux` and
`//go:build !linux && !darwin` both red on a darwin dev box, and they are NOT the
same case: the linux-tagged file resolves the moment the gate runs on a Linux
runner, while `!linux && !darwin` resolves NOWHERE in a darwin-dev + linux-CI
matrix. Its red is permanent.

**Why:** the tidy-looking three-way split (`darwin` / `linux` /
`!linux && !darwin`) for platform dispatch cannot ship under a coverage gate.
Founder-ratified resolution: fold the unsupported-platform refusal into the
`!linux` file alongside the darwin implementation, guarded by
`sandboxPlatformSupported(runtime.GOOS)`. Cost is two dead guard statements on
darwin (90.0% — thin) and nothing on Linux, which compiles none of that file.
"Claims beat prose": CLM-028's unconditional green outranked the plan's file
layout.

**RESOLVED 2026-07-28 (ISSUE-020).** The open question this left is settled: the exec-erased functions moved to a `//go:build linux` HELPER file, the consumer declares that path excluded in a TRACKED `.backstop/coverage-exclusions` (`path<TAB>justification`; the go-toolchain pack folds it into the profile as `#backstop-coverage-exclude`, and a declaration with NO justification is DROPPED not honoured — fail-closed), and the measured remainder cleared 90% via a thin ABI-prober seam. Proven live: run 30395875188 rendered the justification and did not block.

**How to apply:** before adding a build-tagged file, ask which CI platform
compiles it — if none, do not create it. Two follow-on traps measured the same
day: a `!linux` file holding a darwin implementation still needs its tag to stay
`!linux` (narrowing costs `MaybeRunSandboxHelper` its cross-platform symbol), and
bare `return nil, err` guards trip go-standards `error-wrapping-required` — wrap
them. Related: [[project_new_file_coverage_floor]],
[[project_code_motion_shifts_gate_scope]].
