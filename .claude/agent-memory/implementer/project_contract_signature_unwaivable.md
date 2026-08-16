---
name: contract-signature-unwaivable
description: contract_signature findings cannot be waived at all — Line 0 + absolute path means the inline @waiver: token is never harvested, and waiver_resolution reports "no active waivers" rather than rejecting it
metadata:
  type: project
---

An inline `@waiver:contract_signature:...` token is a NO-OP everywhere in the repo. Verified
empirically 2026-08-16 on `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`
(ISSUE-053 finding): the violation stayed red and `waiver_resolution` reported
`pass 0, reason: "clean — no active waivers"` — the token was not suppressed, not rejected,
and not even listed as unused/dangling.

**Why:** two independent structural blockers.
1. `pkg/gate/contract_verdict.go:97-101` constructs the Violation with NO `Line` (zero value).
   `pkg/waiver/adjudicate.go:224 windowLines` therefore yields `[0]`, and the production reader
   `buildWaiverLineReader` (`cmd/backstop/gate.go:1666`) returns `("", false)` for `line <= 0`.
   Empty association window means no line is ever byte-scanned.
2. The same Violation carries an ABSOLUTE file path while the reader does
   `filepath.Join(projectRoot, file)` — unresolvable even with a nonzero line.

There is no declarative fallback: `backstop.yml` exposes only `waiver_warning_days`, and
`backstop waiver` is read-only ("Inspect backstop waivers").

**How to apply:** when told to "apply an interim waiver" on a `contract_signature` red, do NOT
write the token and move on — it looks applied and does nothing, which is worse than the red.
Probe first (add token, run diff-scoped gate, read `waiver_resolution`'s `reason`), then revert
and escalate. The same Line-0 shape likely affects any other file-level gate step that omits
Line; check the constructor before promising a waiver. Note also that a SECOND identical-shape
contract_signature finding sits on an UNMODIFIED file in the same fixture tree
(`scripts/coverage-to-records.sh`) — useful as instant proof the class is a checker limitation
rather than drift your diff introduced. Related: [[project_struct_contract_compiler_gap]],
[[feedback_waivers_are_last_resort]].
