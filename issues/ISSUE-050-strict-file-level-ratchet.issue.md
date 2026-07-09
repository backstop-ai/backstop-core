---
title: "Strict File Level Ratchet"
schema_version: issue/v1

issue:
  id: ISSUE-050
  title: "Strict File Level Ratchet"
  type: enhancement
  status: open
  created: "2026-07-09"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# ISSUE-050: Strict File Level Ratchet

## Problem

The gate's baseline/ratchet only blocks **net-new** findings on files in the
diff scope (`applies-to: new-code`, `pkg/gate/policy.go`
`applyScopedPolicy`). Pre-existing, baseline-grandfathered findings on a file
the author touches do **not** block. You can edit a debt-heavy file, add zero
new debt, and leave every existing finding in that file untouched — and the
gate still goes green.

That is only half a ratchet. It prevents new debt from accruing but never
drives paydown of existing debt, even in the exact place an author already
has full context to fix it: the file they are editing right now.

Founder verbatim (2026-07-09, decided): "If you touched a file that has
stuff in the baseline scan, you need to fix everything in that file. It's
not sufficient just to say you didn't add any debt to that file."

### Where the grandfathering decision lives

`applyScopedPolicy` (`pkg/gate/policy.go:155-207`) is the per-source
grandfathering resolver. Its counting decision, per violation, is:

```go
// pkg/gate/policy.go:180-187
counts := true
if eff.AppliesTo == AppliesToNewCode && baseline != nil {
    counts = newSet[EnrichViolationIdentity(v).IdentityHash]
}
```

`newSet` is the net-new set computed once for the whole step via
`CompareBaseline` (line 158-162). For an `applies-to: new-code` dimension
with a present baseline, a finding counts **only** if it is net-new versus
baseline — a pre-existing finding on a file the author just rewrote is
excluded from `newSet` and never blocks, no matter how directly the author
touched it.

The unscoped path, `ApplyPolicy` (`pkg/gate/policy.go:101-107`), makes the
identical decision (`p.AppliesTo == AppliesToNewCode && baseline != nil` →
`counted = cmp.NewViolations`) for dimensions with no per-source
`Sources` override.

`GateScope` (`pkg/gate/scope.go:24-30`) already carries what's needed to
close this gap: `Files []string` (the normalized touched-file list),
`Mode GateScopeMode` (`diff`/`file`/`all`), and a `Contains(path) bool`
method (`pkg/gate/scope.go:63-69`) that reports whether a path is in scope
(returning `true` unconditionally in `all` mode).

## Solution

Extend the grandfathering condition in `applyScopedPolicy`
(`pkg/gate/policy.go:186`, and the mirrored unscoped path in `ApplyPolicy`)
so a finding also counts when its file was explicitly touched by the
author, not only when it is net-new:

```go
touched := (scope.Mode == GateScopeModeDiff || scope.Mode == GateScopeModeFile) &&
    scope.Contains(v.Path)
counts = newSet[EnrichViolationIdentity(v).IdentityHash] || touched
```

Semantics: baseline grandfathers a finding **only while nobody touches its
file**. The moment a file enters the diff (or `--file` target list), its
grandfather is revoked and every finding in it must be resolved before the
gate is green — not just findings the author personally introduced.

### Required nuances (acceptance criteria — these keep the rule usable)

1. **Only touched files lose grandfathering, never the whole dimension.**
   Project-wide-scope dimensions (e.g. a go-toolchain lint that scans the
   whole repo even under diff-mode gate) must **keep** grandfathering
   findings on files the author did **not** touch. Revocation is keyed
   strictly on `v.Path ∈ scope.Files` (via `scope.Contains`), so touching one
   file obligates only that file's findings — never the whole repo's
   pre-existing debt in one commit.
2. **`--all` / `GateScopeModeAll` is unchanged.** `scope.Contains` returns
   `true` for every path in `all` mode (`pkg/gate/scope.go:65`), which would
   make `touched` universally true and revoke grandfathering repo-wide if
   applied naively — so the touched-check is gated on
   `scope.Mode == GateScopeModeDiff || scope.Mode == GateScopeModeFile`
   explicitly, excluding `GateScopeModeAll`. A full sweep (`gate --all`,
   `baseline generate`) must keep grandfathering gradually; the strict
   revocation only fires on an explicit diff/file-scoped touch.
3. **Resolution = fix it or record a waiver.** The accountable waiver
   mechanism is **BUNDLE-013** (waiver subsystem), which does not exist
   yet. Until it lands, the only interim valve for a revoked finding an
   author cannot immediately fix is engine-native suppression
   (`// nosemgrep`, `//nolint`, etc.) — which is why this issue is
   **sequenced after BUNDLE-013**: shipping strict revocation without any
   accountable "acknowledge and defer" path turns every touched debt-heavy
   file into an unplannable, all-or-nothing gate block.

### Sequencing / motivation

- **Depends on BUNDLE-013** (waiver subsystem) landing first, so a revoked
  finding an author can't fix on the spot has an accountable, tracked
  escape hatch instead of only engine-native suppression comments.
- **Motivation:** this is the other half of the ratchet — it pairs with the
  existing net-new-zero half (`applies-to: new-code` blocking new debt) to
  also drive down existing debt, specifically at the point of maximum
  context (the author is already in the file). At least one
  imminently-launching project is TypeScript-only with real accumulated
  debt, so paydown pressure matters now, not hypothetically.
- A directive tying this issue to BUNDLE-013's completion should be written
  once BUNDLE-013 is further along.

## Verification

Unit-level, in `pkg/gate`: exercise `applyScopedPolicy` (and the unscoped
`ApplyPolicy` path) directly against constructed `StepResult`/`Violation`/
`BaselineArtifact`/`GateScope` fixtures — no real engine invocation needed.

```
go test ./pkg/gate/... -run Policy
```

### Mandated tests

- `TestPolicy_TouchedFileRevokesGrandfathering` — a baseline-present
  (pre-existing, non-net-new) finding on a file listed in
  `scope.Files` under `GateScopeModeDiff` now **blocks** (counts == true),
  where before this change it was silently grandfathered.
- `TestPolicy_UntouchedFileUnderProjectWideDimensionStaysGrandfathered` — a
  baseline-present finding on a file **not** in `scope.Files` (simulating a
  project-wide-scanning dimension that reports findings outside the diff)
  stays grandfathered (counts == false) — proves revocation is scoped to
  the touched file only, not the whole dimension/repo.
- `TestPolicy_AllModeUnaffectedByStrictRevocation` — the identical
  baseline-present finding, under `GateScopeModeAll`, stays grandfathered
  (counts == false) — proves `--all`/`baseline generate` behavior is
  unchanged by this issue.
- `TestPolicy_RevokedFindingClearsWhenFixedOrSuppressed` — a revoked
  finding (touched-file, baseline-present) that is fixed (absent from
  `s.Violations` on the next run) or suppressed (engine-native
  `nosemgrep`/`nolint`, absent from the SARIF findings entirely) no longer
  counts and the step returns to pass.

## References

- **BUNDLE-013** (Waiver Subsystem) — the accountable "acknowledge and
  defer" mechanism this issue is sequenced after; strict revocation without
  it leaves only engine-native suppression as the interim valve for a
  finding an author can't fix immediately.
- **ISSUE-041** (Rename Baseline Policy Key To Applies To) — introduced the
  `applies-to: new-code | all-code` vocabulary and the
  `AppliesToNewCode`/`AppliesToAllCode` constants this issue extends the
  semantics of.
- **ISSUE-046** (Baseline Fingerprint Scope Dependent Unstable) — fixed
  violation identity stability via `NormalizePath`
  (`pkg/gate/scope.go:96-107`); this issue's `scope.Contains(v.Path)` check
  depends on `v.Path` being normalized the same way `scope.Files` is, which
  ISSUE-046 already guarantees.
- `pkg/gate/policy.go:155-207` (`applyScopedPolicy`), `pkg/gate/policy.go:65-124`
  (`ApplyPolicy`) — the two grandfathering decision sites this issue
  modifies.
- `pkg/gate/scope.go:24-30, 63-69` (`GateScope`, `GateScope.Contains`) — the
  existing touched-file primitive this issue reuses rather than building
  new.
- CLAUDE.md "Loud ≠ blocking" — this change moves a class of finding from
  silently tolerated to blocking; it is a deliberate strictness increase,
  not a new silent-green risk, and is gated behind BUNDLE-013 for exactly
  that reason.
