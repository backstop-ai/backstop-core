---
name: base-registry-binding-overrides-fixture
description: A packval test manifest declaring Engine "semgrep" resolves to the BASE binding (real semgrep, non-nil Provision) — a fixture command only takes effect via a pack-level Engines override
metadata:
  type: project
---

In `pkg/packval`, `resolveEngine` (`manifest.go:51-63`) does `reg := baseengines.Registry()`
then `reg[n] = binding` for each `pack.Engines` entry. A test manifest that declares
`Engine: "semgrep"` with NO pack-level `Engines:` map resolves to the BASE binding —
`Command: "semgrep --sarif --quiet"`, `Provision: {semgrep, 1.96.0}` (pinned by
`pkg/baseengines/base_engines_test.go:35-62`). The fixture's intended command is silently
ignored.

**Why:** PLAN-ISSUE-140's TASK-004 told the implementer to model a `RunFixtures` test on the
shipped `TestPhase3_DispatchErrorBranches` and said "the ONE substantive difference is the real
executor and the path-ful command." Followed literally, that test would run REAL semgrep — which
STARTS fine — so a never-started predicate under test never fires and the test stays red after
the fix: a misattributed, permanently-red falsification.

**How to apply:** when a plan asks for a phase3/RunFixtures test whose "resolved engine binding
Command" is a fixture script, verify the task actually specifies a pack-level `Engines:`
override. Also check `Provision`: RunEngine runs `engine.CheckToolAllowed` FIRST, so a non-nil
Provision that fails the allowlist reds at the trust gate instead of the run — a red that keeps
firing after the fix. Nil Provision (or an allowlisted pin) is required for the red to land on
the behavior under test. Note also that `rule.File != ""` plus a rule file whose contents carry
the declared rule ID is required for an at-HEAD "zero errors" baseline — otherwise
`phase3.go:41-48` appends a `semgrep-rule-id` error.

Related: [[e2e-fixture-already-loud-at-head]], [[nil-seam-default-needs-reachable-data]]
