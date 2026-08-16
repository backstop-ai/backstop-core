---
name: gitref-pointer-message-pattern
description: Repo-wide test-assertion gotcha — `%v` on lockfile entry.GitRef (*string) prints a heap address, so a wrong-ref failure message never names the real value
metadata:
  type: project
---

Lock-entry assertions across this repo compare `*entry.GitRef` but format the POINTER:
`t.Errorf("lock entry git_ref = %v, want v1.0.0", entry.GitRef)`. On a non-nil-but-wrong
ref that prints `git_ref = 0xc00047e8b0`. Sites carrying it (as of 2026-08-15):
`cmd/backstop/pack_remote_e2e_test.go`, `cmd/backstop/init_seams_test.go`,
`cmd/backstop/gate_substantiveness_provisioning_test.go`,
`pkg/pack/distribution/add_test.go`. The nil arm is fine (`<nil>`); only the
wrong-value arm is blind.

**Why:** surfaced during the SPEC-037 CLM-031 review by mutating the expected ref in a
scratch copy — the test went red correctly but the message named nothing. Because it is
a copied house pattern rather than a one-off, a new test written "consistently with the
neighbors" reproduces it.

**How to apply:** when red-proving any lock-entry test, mutate the GitRef expectation
specifically and read the MESSAGE, not just the pass/fail. Correct shape is a split
nil-check plus `%q` on the dereference. Relates to [[spec037-clm031-review]].
