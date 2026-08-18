---
name: platform-gated-defect-verification-ceiling
description: Planning a defect whose mechanism is //go:build-gated to a platform the session cannot run — what local verification may claim, and where the real confirmation lives
metadata:
  type: project
---

When the defect's mechanism sits behind a build tag for a platform this session cannot run
(ISSUE-163: `pkg/packval/sandbox_linux.go` + `sandbox_linux_helper.go` are `//go:build linux`,
so on darwin they never compile and `MaybeRunSandboxHelper` resolves to the `!linux`
`return nil` stub), the plan must state a VERIFICATION CEILING as a first-class section and
forbid the obvious-but-impossible verification step.

**Why:** "run the failing test and confirm it passes" is the reflex verification task and it is
unwritable here — the code under test does not exist in this binary. A plan that mandates it
guarantees either a stalled implementer or a report that quietly claims a repro it never had
(false grounding). Hand-simulating the trigger is worse: setting `BACKSTOP_SANDBOX_HELPER_SPEC`
on darwin hits the no-op stub and greens vacuously.

**How to apply:**
- Split verification into what the local platform CAN establish vs. what only CI can. Locally:
  `go build ./...`, `go vet ./<pkg>/...`, and — the real one — the whole affected package suite
  passing unchanged. Name WHY that suite is proof, not ceremony (every `cmd/backstop` test
  depends on the binary `TestMain` builds, so a green suite proves the inserted guard fell
  through to the untouched path).
- Replace the impossible behavioral test with an AST/source-structural pin, which is falsifiable
  on EVERY platform: delete the guard → red anywhere. Precedent in-repo:
  `pkg/packval/sandbox_nonlinux_test.go`'s `TestNonLinuxSandboxHelperTagIsNotNarrowed`.
  Assert the SHAPE (`if err := X(); err != nil`) at `Body.List[0]`, not the call's presence —
  a bare call passes a naive scan and silently disarms the gate.
- Give the real confirmation its own task naming the run: workflow file, job name, `runs-on`,
  the blocking step's exact command, and the `if: always()` report artifact to download.
  Make confirmation a CONJUNCTION — (a) the changed package is actually IN SCOPE for that
  diff-scoped run (read the report's in-scope path list; do not infer it), and (b) the specific
  violation signature is gone. State the expected outcome honestly (ISSUE-163 owned "the
  majority" of 62 violations, not all — so zero is not promised).
- The source issue stays OPEN until that reading exists; honest status is "fix landed, awaiting
  CI confirmation." Related: [[feedback_verify_issue_premises]],
  [[project_closeout_verify_the_fix_landed]].

**Two traps a reviewer caught in exactly this plan — check for both when handing verification
to a later CI run:**

1. **A violation COUNT is never a confirmation criterion across differently-scoped runs.** CI's
   blocking gate is diff-scoped (`gate --base "$BASE"`, never `--all`), so a two-file follow-up
   push has a far smaller in-scope set than the original large-diff run *whether or not the fix
   works* — "the number went down" is trivially true of a run that fixed nothing. Confirmation
   must rest on named, scope-independent facts: the changed package appearing in `.scope.files`
   of `gate-report.json` (`GateResult.Scope` → `GateScope.Files`), plus the specific error
   signature and reproducing test being absent. Report the remaining count as context, labelled
   as context.
2. **A roster/generalization test must assert ONLY the universal property, never the site-specific
   stricter half.** "Apply the same check to every member" is the phrasing that breaks it: the
   single-site test legitimately asserted an extra property (`os.Exit(sandboxHelperExitCode)` —
   valid only because the constant is in the same package), while the already-correct precedent
   `pkg/packval/main_test.go` spells `os.Exit(126)` as a bare literal for a good reason. A roster
   inheriting that assertion reds the one file that was right all along, and the failure reads
   like a real defect. Factor the shared shape into a helper; leave the stricter assertion with
   the single-site test.
