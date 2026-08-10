---
title: "Substantiveness Notarget Findings Have No Waivable Line"
schema_version: issue/v1

issue:
  id: ISSUE-117
  title: "Substantiveness Notarget Findings Have No Waivable Line"
  type: bug
  status: open
  created: "2026-08-09"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Substantiveness Notarget Findings Have No Waivable Line

## Problem

ISSUE-116 fixes inline waiver suppression for hollow-test `test_substantiveness` findings by carrying
`Line` through `HollowFindingsToViolations`. That fix does not — and cannot, without further design —
extend to the substantiveness join's OTHER kind of finding: Q2 "noTarget" violations ("test X does not
call package Y").

`NoTargetViolation` constructs its `Violation` with no `Line` field at all:

```go
// pkg/gate/substantiveness_join.go:65-69
return Violation{
	Rule:     StepTestSubstantiveness,
	Message:  "test function " + funcName + " does not call package " + targetPkg,
	Severity: "error",
}, true
```

Unlike the hollow-finding case, this is not a dropped field — there is no source line to carry. A
noTarget violation is not a 1:1 conversion of one SARIF-reported finding the way a hollow finding is;
it's SYNTHESIZED by the gate's own set-membership decision table (`NoTargetViolation`,
`substantiveness_join.go:55-70`), firing when a target package name is absent from a test's
`ReferencedSymbolSet` (`substantiveness_join.go:26`) — itself a presence-only `map[string]bool`
assembled from zero or more extraction findings, carrying no location data of its own.

The underlying `MandatedTest` type (`pkg/gate/step_testverify.go:16-33`) — the thing a noTarget
violation is really "about" — has no declaration-line field either: `FuncName`, `FilePath`, `SpecFile`,
`TargetPkg`, `SpecID`, `ClaimID`, `Status`, `IsAbsence`. `FilePath` names the file the test lives in,
but nothing records what line the test function itself starts at, so even reaching for "the test's own
declaration line" as a stand-in requires new data that doesn't exist yet.

**Why this is architecturally harder than ISSUE-116, not just unimplemented:** ISSUE-116's fix is a
values-in-input, forward-a-field substitution — the line already exists upstream and was dropped in
transit. Here there is no line anywhere in the input to forward. Giving a noTarget violation a waivable
line means either (a) threading a real source location into `MandatedTest`/`ReferencedSymbolSet` from
wherever the test function is actually parsed, or (b) picking some other convention (e.g. a synthetic
anchor, mirroring how `coverage_threshold` findings use a fixed first-line convention per
`step_waiver.go`'s `coverageAnnotationLine` handling) — and it's not obvious from the current code which
of those is right, or whether a synthetic anchor would satisfy what users actually expect when they try
to waive a noTarget finding. No fix direction is proposed here; that decision belongs to whoever picks
this issue up next.

## Impact

Until this is resolved, noTarget `test_substantiveness` findings remain un-waivable inline even after
ISSUE-116 lands — a user hitting a noTarget false positive has no `@waiver` escape hatch at all, only
the same two unacceptable workarounds (restructure code to dodge the rule, or disable the rule
globally) that ISSUE-017 and ISSUE-116 both document for their respective cases.

## References

- `pkg/gate/substantiveness_join.go:55-70` — `NoTargetViolation`, the set-join decision table that
  synthesizes the violation with no `Line`
- `pkg/gate/substantiveness_join.go:26` — `ReferencedSymbolSet`, presence-only (`map[string]bool`),
  carries no location data
- `pkg/gate/step_testverify.go:16-33` — `MandatedTest`, no declaration-line field
- ISSUE-116 — the sibling hollow-finding case this issue is scoped out of; fixed there by forwarding an
  already-existing `Line`, which is not available here
