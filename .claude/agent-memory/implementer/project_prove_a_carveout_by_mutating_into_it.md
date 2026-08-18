---
name: prove-a-carveout-by-mutating-into-it
description: When a plan says test A asserts a stricter property than roster test B, prove the carve-out by mutating the code INTO the carved-out shape and showing B stays GREEN while A goes red
metadata:
  type: project
---

When two tests share a shape-helper but one is deliberately stricter (PLAN-ISSUE-163
sharp edge 8b: CLM-001 asserts `os.Exit(sandboxHelperExitCode)` as an identifier;
the CLM-003 roster must NOT, because `pkg/packval/main_test.go` legitimately spells
`os.Exit(126)` as a literal), reading the code to confirm the carve-out is weak
evidence. Mutate INTO the carved-out shape and run BOTH tests.

Measured 2026-08-17 in a detached worktree: flipping the guard to `os.Exit(126)`
made `TestIntegrationTestMain_...FirstStatement` red with the exact identifier
message, while `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`
stayed **PASS**. That single run proves both halves at once — the strict test
really is strict, and the roster really would not red the already-correct
precedent it is modelled on.

**Why:** a roster that over-asserts reds the one file that was right all along,
and the failure reads like a real defect in the precedent. Eyeballing "the helper
doesn't check the exit code" does not falsify that; a mutation does.

**How to apply:** any time a plan carves an assertion out of a broader/roster
check, run the mutation matrix asymmetrically — the carved-out mutation must
produce exactly one red (the strict test) and one green (the roster). Pair it
with the ordinary red-proof ([[project_redproof_by_worktree_flip]]) in the same
worktree; both cost one `go test` each. Related:
[[project_mutation_matrix_beats_sequence_red]],
[[feedback_pin_the_crossing_not_blanket_absence]].
