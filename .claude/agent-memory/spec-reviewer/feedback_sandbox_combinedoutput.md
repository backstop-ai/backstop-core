---
name: sandbox-combinedoutput
description: SPEC-031 convert pipe routes through SandboxedRun which uses CombinedOutput — contaminates SARIF, contradicts the spec's own clean-stdout REQ
metadata:
  type: feedback
---

When a spec mandates a clean stdout→parse contract for one runner but routes a parallel
output path through a DIFFERENT capture function, check that the second function also
captures cleanly — fixing one runner doesn't fix the other.

**Why:** SPEC-031 REQ-009 adds `RunStdout` (clean stdout) to pkg/check/runner.go to stop
CombinedOutput corrupting SARIF. But REQ-007 routes the `convert` step (whose stdout IS the
final SARIF handed to parseSarif) through pkg/packval `SandboxedRun`, which at sandbox.go:18
uses `c.CombinedOutput()`. So a converter writing any stderr banner interleaves into the
bytes parsed as SARIF — exactly the corruption REQ-009 exists to kill, left unaddressed for
the more critical convert path. No REQ/CLM covers giving SandboxedRun a clean-stdout variant.

**How to apply:** For SPEC-031-family specs (engine dispatch / convert / sandbox), whenever a
requirement says "X's stdout becomes the SARIF," grep the actual capture call (RunStdout vs
SandboxedRun's CombinedOutput) and confirm BOTH paths are stderr-clean. Flag as major if the
convert/sandbox path still rides CombinedOutput.
