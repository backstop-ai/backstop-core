---
name: deletion-strands-downstream-deadcode
description: A whole-branch deletion can strand the code DOWNSTREAM of it (the only consumer) as dead+untested; check what the deleted producer was feeding, not just the branch itself
metadata:
  type: feedback
---

When a spec deletes a whole branch/producer, trace what that branch was the SOLE FEEDER
of. The deletion can strand downstream code — types, struct fields, methods, alternate
control-flow arms — as unreachable, because their only live caller was inside the deleted
branch. Go does NOT error on unused unexported types/fields/funcs/methods (only unused
imports + local vars), so the build stays green and the gap is silent — but the stranded
code is vestigial AND, if its tests rode the deleted feeder, becomes UNTESTED, pressuring
the coverage gate.

**Why:** SPEC-039 review #3. After the spec (correctly) deleted the whole `.manifest.json`
walk in `LoadManifest`, `&Manifest{rules: allRules}` (manifest.go:239) — the SOLE producer
of a NON-default Manifest — went with it, leaving `defaultManifest(){isDefaults:true}` as
the only Manifest producer. That stranded `RouteFile`'s non-default `for _,rule:=range m.rules`
branch, `matchesRule` (sole caller = that branch), `ManifestRule`, and the `Manifest.rules`
field as dead. Worse, the spec's REQ-010 surviving-symbol list + prohibition + CLM-016
EXPLICITLY mandated retaining `matchesRule`/`matchGlobPattern` as "surviving API intact" —
mislabeling stranded code as live. And `matchesRule`'s only test feeders were the
`writeManifest`/`writeRawManifest` pattern tests the SAME spec swept as orphans → dead AND
untested → coverage hit. (Subtlety: `matchGlobPattern`/`matchDoubleStarPattern` survived
TESTED because separate tests call them directly, not via `.manifest.json` — so trace each
downstream symbol's test feeders individually.)

**How to apply:** For any whole-branch/producer deletion: (1) identify what the deleted code
was the sole PRODUCER/FEEDER of (grep the constructed type, the populated field, the
returned value); (2) for each downstream consumer, check whether ANY surviving producer
still feeds it — if not, it is stranded dead code; (3) check whether the spec's
"surviving API / do-not-delete" list mislabels stranded code as live (internal
contradiction); (4) check whether the stranded code's TESTS rode the deleted feeder (then
they orphan too → coverage gap). Correct disposition: delete the stranded path too (further
shrinks the surface) OR document it as knowingly-dormant with a NAMED near-term re-feeder —
never assert it is "surviving API intact." This is the INVERSE of
[[feedback_vestigial_retain_via_test_liveness]]: that one = don't keep a dead branch alive
for tests; this one = don't keep the dead code DOWNSTREAM of a deleted branch either.
