---
title: "`recipe apply` Discards Its Declared `manual:` Fallback — Every Violation-Class Failure Exits 1 With Zero Output"
schema_version: issue/v1

issue:
  id: ISSUE-080
  title: "`recipe apply` Discards Its Declared `manual:` Fallback — Every Violation-Class Failure Exits 1 With Zero Output"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `recipe apply` Discards Its Declared `manual:` Fallback — Every Violation-Class Failure Exits 1 With Zero Output

## Problem

`backstop recipe apply <pack>:<recipe>@<version>` fails silently on every
violation-class error: exit code 1, **no stdout, no stderr**, `--json` included. The
op's declared `manual:` fallback instruction — which the applier carries VERBATIM
into the error message specifically so an unreachable site still leaves the operator
with actionable text (SPEC-054 REQ-011, mandated by CLM-051/CLM-052) — is composed and
then thrown away. The user sees a bare `exit 1` and nothing else.

### Repro (live-reproduced 2026-07-25, scratchpad dogfood consumer)

Both observed independently, same symptom each time:

1. **Missing required param** — apply a recipe with a `REQUIRED` param
   (`RecipeManifest.Params[].Required`, `pkg/recipe/manifest.go:83`) and no default,
   without supplying it. `effectiveParams` (`pkg/recipe/apply.go:896-912`) fails
   substitution. Result: exit 1, zero output.
2. **Merge fragment file not found** — apply a `merge` op whose declared
   `Fragment` path does not resolve. `applyMerge` (`pkg/recipe/apply.go:628-634`)
   returns `fmt.Errorf("read declared fragment %q: %w", op.Fragment, err)`. Result:
   exit 1, zero output.

Both errors carry real diagnostic text (and, for ops that declare one, the op's
`manual:` fallback) — none of it reaches the terminal.

### Root cause

`cmd/backstop/recipe_apply.go:60-70`: `RunE` wraps every non-`ConfigError` failure
from `runRecipeApply` as `&ExitCodeError{Code: ExitViolations, Message: err.Error()}`.
The comment right above the wrap (`recipe_apply.go:61-65`) says the message "already
carries the op's declared manual instruction verbatim" — true of the *composed
string*, but irrelevant, because that string is never printed.

`cmd/backstop/main.go:17-24` implements a convention: for `ExitViolations` (exit 1),
print nothing to stderr, because "violation details are already in the command's
output" (`main.go:19`). That convention holds for `gate` and `pack check`, which
render a structured report to stdout *before* returning the error. It does NOT hold
for `recipe apply`, which emits nothing before failing — `RunE` returns straight out
of the `if err != nil` branch with no prior `cmd.Print*` call. The convention's
precondition (something was already shown) is false here, so applying it discards the
only diagnostic the user would have gotten, including the hand-carried `manual:`
instruction that exists specifically to survive this exact failure path.

### Second victim of this convention

This is the SECOND command found swallowing errors under the same
exit-1-already-explained assumption; **ISSUE-074** (`pack relock` silent failure)
documents the first, with the identical shape: a violation-class error wrapped as
`ExitCodeError{Code: ExitViolations}` by a command that never printed anything first.
Both issues point at the same convention in `cmd/backstop/main.go:17-24` as the
underlying defect — it silently assumes every `ExitViolations` returner pre-renders
its own output, and nothing enforces that assumption. Other `ExitViolations`
returners should be audited for the same trap before it claims a third victim.

### Why tests missed it

The trust-gate/E2E tests for `recipe apply` assert on the in-process error VALUE
returned by the command (`err.Error()` contents), never on the process-level stderr
of an actual failing invocation. A test that shells out to the built binary and
checks stderr would have caught this; none does.

### Expected

On any violation-class failure, `backstop recipe apply` writes a diagnostic —
including the op's declared `manual:` text when the failure carries one — to stderr
before returning the `ExitViolations`-coded error. Either `recipe apply` prints its
own error before returning (mirroring what `gate`/`pack check` do), or
`cmd/backstop/main.go`'s convention gains a way to distinguish "this command already
rendered its own output" from "this command did not" (e.g. a carrier flag on
`ExitCodeError`) so a command that hasn't printed anything doesn't fall through the
same silent branch. A process-level test — invoking the built `recipe apply` binary
against a failing recipe and asserting the `manual:` text appears on stderr — should
guard the fix.

## References

- `cmd/backstop/recipe_apply.go:54-75` — `RunE`; wraps every non-`ConfigError`
  failure as `ExitCodeError{Code: ExitViolations, ...}` with nothing printed
  beforehand; comment at lines 61-65 asserts the message "already carries" the
  manual instruction, which is true of the string but not of anything the user sees
- `cmd/backstop/main.go:17-24` — the `ExitViolations`-suppresses-stderr convention,
  valid for `gate`/`pack check` (pre-render), false for `recipe apply` (no
  pre-render); same convention implicated in ISSUE-074
- `pkg/recipe/apply.go:576-604` (`missingSite`, `injectionLimit`) — where the op's
  declared `manual:` fallback is composed VERBATIM into the error text (REQ-011)
  that then never reaches the user
- `pkg/recipe/apply.go:617-641` (`applyMerge`) — fragment-file-not-found failure
  path reproduced live
- `pkg/recipe/apply.go:896-912` (`effectiveParams`) — missing-required-param failure
  path reproduced live
- `pkg/recipe/manifest.go:83,165-191` — `Params[].Required`; injection-limit op
  families are validated to declare a non-empty `manual:` at manifest-load time
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md` — REQ-011 (manual fallback
  relayed verbatim), CLM-051/CLM-052 (mandate the relay)
- `issues/ISSUE-074-pack-relock-silent-failure.issue.md` — first victim of the same
  `ExitViolations`-suppresses-stderr convention; cite as precedent when auditing
  other `ExitCodeError{Code: ExitViolations}` returners
- `docs/CODEBASE-MAP.md` — "Known gap — `pack relock` silent failure" section
  documents ISSUE-074's instance of this convention; this issue is a second data
  point for the same gap
