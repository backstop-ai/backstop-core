---
title: "Sandbox Helper Doc Comment Stale Probe Claim"
schema_version: issue/v1

issue:
  id: ISSUE-169
  title: "Sandbox Helper Doc Comment Stale Probe Claim"
  type: bug
  status: open
  created: "2026-08-18"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Sandbox Helper Doc Comment Stale Probe Claim

## Problem

`newSandboxHelperCommand`'s doc comment (`pkg/packval/sandbox_linux.go:163-175`) contains a
sentence that ISSUE-165's fix inverted the truth of, and left byte-unchanged on purpose. The
comment reads, in full:

> "`newSandboxHelperCommand` builds the parent-side command that re-execs this binary in helper
> mode. It negotiates the Landlock ABI first and REFUSES loudly when the mechanism is
> unavailable, executing nothing. probeABI is a PARAMETER so the refusal path is reachable from
> a test: on a healthy host the probe always succeeds, so the "mechanism unavailable" branch —
> and the two callers' error wraps that depend on it — could never execute. The seam is
> deliberately THIN, matching the shape resolveLandlockMechanism already takes one level down:
> it substitutes the ABI ANSWER, never what a Landlock rule is or how it is applied.
> **TestSandboxLinux_ProductionPathUsesTheRealABIProbe asserts both production call sites hand
> it the real probeLandlockABI, so the seam cannot become a place where test and production
> diverge.**"

The bolded sentence (source lines 171-173) is now false about what the rewritten guard checks —
not stale in the sense of describing removed code, but stale in the more dangerous sense of
describing behavior the guard's own rewrite deliberately inverted.

### What the guard actually asserts now, and why the old sentence is backwards

Before ISSUE-165, the guard (`TestSandboxLinux_ProductionPathUsesTheRealABIProbe`,
`pkg/packval/sandbox_linux_errors_test.go`) did assert, incorrectly, that the literal identifier
`probeLandlockABI` appears at every tracked call site — which is what the doc comment describes.
ISSUE-165's fix (commit `8d35706`, then two adversarially-found evasions closed at `fc2b8ce`)
rewrote the guard to classify each call site by its ENCLOSING FUNCTION
(`pkg/packval/sandbox_wiring_guard_test.go`, now an untagged file, `proberWiringViolations`).
At the two platform-neutral dispatch seams (`platformSandboxedRun`, `platformSandboxedRunStdout`)
the guard still requires the literal `probeLandlockABI`. But at the two INNER delegation seams
this doc comment sits inside — `linuxSandboxedRunWith` and `linuxSandboxedRunStdoutWith` calling
`newSandboxHelperCommand` — the requirement is now the opposite: the last argument must be the
enclosing function's OWN injected `LandlockABIProbe`-typed parameter, and a literal
`probeLandlockABI` there is flagged as its own violation. DIR-024 item 21 (lines 1076-1092 of
`directives/DIR-024-gate-engine-quality.directive.md`) names this explicitly as the refused
"option (a)" shape: renaming the parameter to literally `probeLandlockABI` would make the naive
check pass, but it would shadow the package-level function inside the body and make the guard
pass for ANY caller-supplied prober including a fake one — "a vacuous green," in that directive's
words, that "retires the exact divergence the guard's own header says it exists to catch."

So the doc comment, read today, states as settled fact the exact claim ISSUE-165 disproved and
DIR-024 explicitly rejected as a fix. It was left untouched deliberately: ISSUE-165's fix states
among its own claims that `sandbox_linux.go` is untouched by the change (the guard rewrite lives
entirely in the new `sandbox_wiring_guard_test.go` / the moved `sandbox_linux_errors_test.go`) —
which is correct and load-bearing for that issue's "no production code was touched" claim, but it
means this stale sentence was never in scope for that lane to fix, and nothing else has fixed it
since.

### Why this is a hazard, not just untidiness

This is precisely the sentence a future reader would cite to justify reintroducing the rejected
"option (a)" shape: renaming `linuxSandboxedRunWith`'s or `linuxSandboxedRunStdoutWith`'s
`probeABI` parameter to literally `probeLandlockABI`, on the reasoning that "the doc comment
already says both production call sites hand it the real `probeLandlockABI`, so this makes the
code match its own documentation." That reasoning is backwards — it would make the parameter
shadow the package-level `probeLandlockABI` function inside the body, silently defeating the
seam's entire purpose (letting a test drive the refusal path with a fake prober) exactly as
DIR-024 item 21 describes. A doc comment that argues for the wrong fix is worse than no comment.

## Impact

Low direct impact today — the comment is documentation, not executable code, so it changes no
runtime behavior on its own. The risk is entirely in what a future editor (human or agent) might
do while trusting it: a plausible, comment-justified edit would silently reopen the exact
test/production divergence ISSUE-165's guard rewrite exists to catch, and the guard itself would
no longer catch it, because the shadowed identifier would satisfy the (correct, post-fix)
enclosing-function check by construction.

## References

- `pkg/packval/sandbox_linux.go:163-175` — `newSandboxHelperCommand`'s doc comment; the stale
  claim is at lines 171-173: "TestSandboxLinux_ProductionPathUsesTheRealABIProbe asserts both
  production call sites hand it the real probeLandlockABI."
- `pkg/packval/sandbox_linux.go:217-221` — `linuxSandboxedRunWith`, one of the two inner
  delegation seams the stale sentence describes backwards.
- `pkg/packval/sandbox_linux.go:243-246` — `linuxSandboxedRunStdoutWith`, the sibling seam.
- `pkg/packval/sandbox_wiring_guard_test.go` — the rewritten, enclosing-function-aware guard
  (`proberWiringViolations`) that now asserts the inverse of the stale sentence at these two
  seams.
- `directives/DIR-024-gate-engine-quality.directive.md:1076-1092` (item 21) — the directive's own
  independent analysis naming the exact "option (a)" shadowing hazard this stale comment would
  argue a future editor into re-introducing.
- `ISSUE-165` (`issues/ISSUE-165-sandbox-abi-probe-wiring-test-naming-mismatch.issue.md`) — origin
  issue; its fix (commits `8d35706`, `fc2b8ce`) rewrote the guard and left `sandbox_linux.go`
  byte-unchanged by design, which is why this comment was never in that lane's scope to fix.
- `PLAN-ISSUE-165-sandbox-abi-probe-test-parameter-naming.plan.yml` TASK-005 — the closeout task
  that could not author this follow-on itself (no issue-authoring capability in that lane) and
  named it as owed.

A fix is small and contained: rewrite the sentence at `sandbox_linux.go:171-173` to describe the
current, correct requirement (the dispatch seams pass the literal `probeLandlockABI`; the inner
seams forward their own injected parameter, and the guard now flags a literal `probeLandlockABI`
there as the violation it would be) — or point the comment at the guard file for the authoritative
statement instead of restating it. The exact wording is left to this issue's plan.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for
"newSandboxHelperCommand", "stale.*comment" (case-insensitive), and "probeLandlockABI" matched
only `ISSUE-165` (the origin issue, which explicitly does not fix this comment) and `ISSUE-020`
(the historical issue that introduced the seam and its original, then-accurate, doc comment). No
open issue or bundle charter already owns this specific stale-comment surface.
