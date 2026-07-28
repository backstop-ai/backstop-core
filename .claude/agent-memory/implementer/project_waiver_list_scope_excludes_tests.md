---
name: waiver-list-scope-excludes-tests
description: `backstop waiver list` scopes to non-test sources, so a waiver in a *_test.go file never appears there — the gate's waiver_resolution step is the authoritative adjudication signal
metadata:
  type: project
---

`backstop waiver list` and the gate's `waiver_resolution` step use DIFFERENT
scopes. Measured 2026-07-26 (ISSUE-075): three freshly added `@waiver` tokens in
`tests/smoke/smoke_test.go` adjudicated correctly under the gate — `pack_engines`
dropped 4→0 and `waiver_resolution` reported `PASS · 3 waivers (<rule ids>)` — while
`backstop waiver list` showed a completely different set of 3 (the pre-existing
tokens in `cmd/backstop/artifact_validate.go`, `pack_gate.go`,
`pack_gate_provision.go`, all non-test files) and `Unused / dangling (0)`.

**Why:** `waiver list` appears to enumerate over the non-test source
classification only. A correctly-associating waiver on a test file is therefore
absent from its output entirely — it is NOT reported as unused/dangling, it is
simply out of scope. Reading `waiver list` as the verification signal makes a
working waiver look like it never registered.

**How to apply:** when a plan requires proving waivers are ACTIVE, read the gate's
`waiver_resolution` step (its `reason` names each adjudicated rule id) plus the
before/after violation count on the step the waiver covers. Do not conclude a
waiver failed to associate because `waiver list` omits it. `waiver list` has no
scope flag (`--all` is rejected). See [[project_no_code_check_command]] for the
sibling "the command you expect does not exist" trap.
