---
name: full-cli-gate-fixture-reqs
description: A temp-dir project driven through the full `gate` CLI needs backstop.lock and specs/ or it exits 2 before any assertion — unlike step-level tests that call buildGateSteps directly
metadata:
  type: project
---

A test fixture that runs the WHOLE gate command (`runGateCommand`, `cmd/backstop/gate_base_test.go`)
needs two things the step-level fixtures do not, and both surface as exit 2 (config
error) that reads like a violation:

- **`backstop.lock` is mandatory once `backstop.yml` declares a pack.**
  `pack_lock_verification` is the FIRST step and a missing lock is
  `missing_lockfile` -> exit 2. Use a `source_type: local` entry — `VerifyLock`
  skips local packs instead of hashing them (`pkg/pack/distribution/verify.go`),
  so you get a valid lock without a real content hash.
- **An absent `specs/` directory hard-fails `test_verification`**
  ("failed to extract mandated tests: reading spec dir ..."). An empty dir is fine.

The contrast that makes this easy to miss: `cmd/backstop/gate_scope_test.go`'s
`newDiffScopedPackGateProject` declares a pack with NO lock and works fine, because
those tests call `buildGateSteps` and run ONLY the `pack_engines` step — they never
reach lock verification.

**Why:** debugging time went into "the ratchet is broken" that was actually a missing
lock file, because the exit code (2) and the gate's own summary both look like the
assertion under test failed.

**How to apply:** when a full-CLI gate fixture exits 2, read the printed step list
BEFORE suspecting the behaviour you are testing. Related: [[project_editing_file_pulls_it_into_gate_scope]].

**On baseline ratchet tests:** do NOT reach for `ProjectWide: true` to make a finding
survive the diff-scope filter so it reaches the baseline comparison.
ProjectWide/exempt findings on UNTOUCHED files are retained on purpose (CLM-005 — an
unchanged-file build break must still red the gate), so such a test asserts the
opposite of an intended guarantee. Non-exempt findings on untouched files are dropped
by the scope filter, and that IS the grandfather at the exit-code level.
