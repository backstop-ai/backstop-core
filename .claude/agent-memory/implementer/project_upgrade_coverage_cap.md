---
name: upgrade-coverage-cap
description: pack_upgrade.go cannot reach the 80% per-file coverage floor while REQ-009's scan capability is unavailable — its whole success path is unreachable by design
metadata:
  type: project
---

After SPEC-055 REQ-009, `pack upgrade` wires an explicit
`unavailableScanner` that always returns `*CapabilityUnavailableError`, so
`distribution.Upgrade` can never return successfully. Everything in
`cmd/backstop/pack_upgrade.go` after the `Run` error check — the "Upgraded
X -> Y" line, the remediation-bundle line, the baselined-violations line —
is therefore DEAD until BUNDLE-006 REQ-014/REQ-018 land. Measured ceiling:
8/14 statements (~57%) against a per-file floor of 80%.

**Why:** the spec calls this out as a Sharp Edge — pack upgrade gets WORSE
before it gets better, because a zero-violation major upgrade that never
ran a scan is a vacuous green. The coverage red is the honest consequence,
not a testing gap. It is NOT fixable by writing a better test: no test can
reach code the production wiring makes unreachable.

**How to apply:** do not chase this with a test double or a coverage waiver
dressed as a fix. Report it as a structural consequence and let the founder
decide (waiver-with-reference vs. accepting the red until REQ-014 ships).
Same reasoning applies to every `newProduction*Command()` assembly-error
branch: the constructors only fail on nil arguments, which production never
passes, so those branches are permanently uncovered by construction.
