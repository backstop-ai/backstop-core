---
title: "Gate Substantiveness Flags Testmain Absence Tests"
schema_version: issue/v1

issue:
  id: ISSUE-035
  title: "Gate Substantiveness Flags Testmain Absence Tests"
  type: bug
  status: open
  created: "2026-07-05"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# Gate Substantiveness Flags Testmain Absence Tests

## Problem

The gate's `test_substantiveness` step produces false-positive "hollow" / "does not
call package X" violations for two legitimate categories of test that are
structurally correct but don't match the substantiveness heuristic. Reproduced
2026-07-05 on the current tree: `./bin/backstop gate` reports 15
`test_substantiveness` violations, all false positives:

```
test_substantiveness      fail  (419ms)  (15 violations)
  - [test_substantiveness] test function TestMain has no assertions (hollow) (cmd/backstop/integration_test.go)
  - [test_substantiveness] test function TestNoProductionImportOfCompile does not call package check (cmd/backstop/standards_removal_test.go)
  - [test_substantiveness] test function TestCompiledStandardsArtifactsAbsent does not call package check (cmd/backstop/standards_removal_test.go)
  - [test_substantiveness] test function TestPkgCompileDirectoryAbsent does not call package check (cmd/backstop/standards_removal_test.go)
  - [test_substantiveness] test function TestStdGo001SourceAbsent does not call package check (cmd/backstop/standards_removal_test.go)
  - [test_substantiveness] test function TestGate_SucceedsWithoutStandards does not call package check (cmd/backstop/standards_removal_test.go)
  - [test_substantiveness] test function TestDogfood_BackstopYmlDeclaresGoStandardsPack does not call package check (cmd/backstop/dogfood_pack_test.go)
  - [test_substantiveness] test function TestDogfood_GoStandardsLockVerifies does not call package check (cmd/backstop/dogfood_pack_test.go)
  - [test_substantiveness] test function TestDogfood_StaleSlotlyLockEntryRemoved does not call package check (cmd/backstop/dogfood_pack_test.go)
  - [test_substantiveness] test function TestStandardScaffolder_Untouched does not call package check (cmd/backstop/manifest_configerror_test.go)
  - [test_substantiveness] test function TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites does not call package gate (cmd/backstop/checktype_catalog_test.go)
  - [test_substantiveness] test function TestCatalog_SurvivingSitesNotMistaggedDeleted does not call package gate (cmd/backstop/checktype_catalog_test.go)
  - [test_substantiveness] test function TestCatalog_GuardScansGateSemanticSurfaceOnly does not call package gate (cmd/backstop/checktype_catalog_guard_test.go)
  - [test_substantiveness] test function TestCatalog_GuardFailsOnUnlistedConsumer does not call package gate (cmd/backstop/checktype_catalog_guard_test.go)
  - [test_substantiveness] test function TestCatalog_GuardFailsOnStaleEntry does not call package gate (cmd/backstop/checktype_catalog_guard_test.go)
```

Surfaced while implementing ISSUE-018, which pulled these pre-existing test files
into diff scope for the first time.

### Category 1 — `TestMain` flagged as hollow

`[test_substantiveness] test function TestMain has no assertions (hollow)`
(`cmd/backstop/integration_test.go`).

`TestMain(m *testing.M)` is Go's test-harness entry point — setup/teardown plus
`os.Exit(m.Run())`. It is BY DESIGN not an assertion-bearing test; Go itself treats
it specially (it is never run as a regular test case). Flagging it as hollow is
categorically wrong regardless of heuristic tuning — `TestMain` should be
unconditionally exempt from the hollow check.

### Category 2 — structural/absence tests flagged "does not call package X"

Nine test functions across five files are flagged as not calling their target
package: `TestNoProductionImportOfCompile`, `TestCompiledStandardsArtifactsAbsent`,
`TestPkgCompileDirectoryAbsent`, `TestStdGo001SourceAbsent`,
`TestGate_SucceedsWithoutStandards` (`cmd/backstop/standards_removal_test.go`);
`TestDogfood_BackstopYmlDeclaresGoStandardsPack`,
`TestDogfood_GoStandardsLockVerifies`, `TestDogfood_StaleSlotlyLockEntryRemoved`
(`cmd/backstop/dogfood_pack_test.go`); `TestStandardScaffolder_Untouched`
(`cmd/backstop/manifest_configerror_test.go`);
`TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites`,
`TestCatalog_SurvivingSitesNotMistaggedDeleted`
(`cmd/backstop/checktype_catalog_test.go`);
`TestCatalog_GuardScansGateSemanticSurfaceOnly`,
`TestCatalog_GuardFailsOnUnlistedConsumer`, `TestCatalog_GuardFailsOnStaleEntry`
(`cmd/backstop/checktype_catalog_guard_test.go`).

These are legitimate structural-invariant / absence tests: they assert that a
directory/symbol/import is ABSENT, that a lock verifies, that a filesystem scan
finds no violations, and so on. Such tests correctly do NOT reference the target
package by a package-qualified call — there may be nothing to call (the thing they
assert absent was deleted), or the assertion is purely about repository/filesystem
structure (walking directories, reading `backstop.yml`/`backstop.lock`, `os.Stat`).
The `does not call package X` heuristic mis-fires on this whole class.

### Why it matters

The eradication backlog (ISSUE-018 and its siblings) is full of deletion and
absence-assertion tests — that is how you *prove* baked code is gone
(SPEC-034-style deletion-assertion tests, e.g. `pkg/validate/deletion_assertion_test.go`).
If `test_substantiveness` false-flags every absence test, it penalizes exactly the
test pattern the thin-executor program depends on, and pushes authors toward
manufacturing vacuous "call the package" boilerplate just to satisfy the check.
That inverts the gate's own enforcement philosophy: the goal is to block real
hollowness, not to force cosmetic calls that add no assertion value (see
CLAUDE.md, "Loud ≠ blocking" / "block defects, don't manufacture false positives").

## Root cause (grounded in current code, 2026-07-05)

The two categories have two different root causes, in two different layers of the
substantiveness pipeline (SPEC-037's pack/core split):

**Category 1 (`TestMain` hollow)** — the assertion vocabulary lives entirely inside
the pack's ast-grep rule, `packs/substantiveness/ast-grep/rules/hollow-test-go.yml`.
The rule matches any `func Test*` declaration whose body has no descendant
`call_expression` matching the regex
`require|assert|check|verify|expect|must|Fatal|Fatalf|Error|Errorf|Fail|FailNow|Skip|Skipf`.
There is no exemption for the function name `TestMain`, so it is caught by the same
"any Test*-named function must contain an assertion call" rule as ordinary test
functions. **This is a pack-rule fix** (the dogfood `backstop/substantiveness` pack,
source at `packs/substantiveness/`, installed copy at
`.backstop/packs/backstop/substantiveness/`) — add a `not: {has: {field: name,
regex: "^TestMain$"}}` guard (or equivalent) to `hollow-test-go.yml`, matching the
pattern the file's own header comment describes ("the assertion vocabulary lives
ENTIRELY in this rule YAML — the binary carries no hardcoded assertion-selector
list", CLM-006).

**Category 2 ("does not call package X")** — this is NOT produced by a pack rule
directly; it is core gate join logic. The pack's `referenced-symbol-go.yml`
(Q2 extraction rule) only *extracts* which packages a test references — it is
"spec-UNAWARE" (per its own header comment) and performs no join. The actual
noTarget verdict is computed in `pkg/gate/substantiveness_join.go`,
`NoTargetViolation` (lines 55-70): given a `MandatedTest` (a test function named in
a spec claim's `tests` list, extracted by `ExtractMandatedTests` in
`pkg/gate/step_testverify.go`), the target package derived from the claim's spec
`implementation.package`, and the test's referenced-symbol set from the Q2
extraction, it raises a violation whenever the target package is non-empty, the
test isn't in the same package, and the target package is absent from the
referenced set. **This decision table has no way to know a mandated test is an
absence/structural assertion** — it treats every non-same-package mandated test as
one that must call its target package, which is true for ordinary "does the new
code do X" claims but false for "prove X is now absent" claims. This is core logic,
not a pack rule, though the pack's Q2 extraction feeds it.

## Fix directions to evaluate (not committed — surface at plan time)

1. **`TestMain` exemption (Category 1)** — straightforward: add a name-based
   exclusion to `hollow-test-go.yml` in `packs/substantiveness/`. Low uncertainty.
2. **Broaden the hollow heuristic** — treat a test as substantive if it makes ANY
   assertion OR references os/filesystem/exec/lock-style APIs (e.g. `os.Stat`,
   `filepath.Walk`, `exec.Command`), not only the fixed verb regex. Still a pack-YAML
   change, but changes the semantics of "hollow" more broadly than a single
   exemption — needs care not to swallow genuinely empty stub tests.
3. **Absence-test annotation for the noTarget check (Category 2)** — allow a spec
   claim to mark a mandated test as an absence/structural claim (e.g. a `kind:
   absence` field alongside `tests` in the claims schema, or a claim-text
   convention the gate recognizes) so `NoTargetViolation` in
   `pkg/gate/substantiveness_join.go` skips the target-package join for tests so
   marked. This is a core + schema change, not a pack change.
4. **Reframe the noTarget join around what's actually being proven** — e.g. treat a
   test whose target package no longer exists on disk (the deletion case) as
   automatically satisfying "does not call package X" without a claim annotation.
   Doesn't cover the `backstop.yml`/`backstop.lock`-reading structural tests
   (dogfood_pack_test.go), which reference no deleted package at all.

**Core uncertainty, stated honestly:** distinguishing a genuine hollow/no-target
stub from a legitimate absence or structural-invariant test is not mechanical from
the test body alone — both "forgot to write real assertions" and "correctly proves
absence" can look identical to an AST-only heuristic (no call to the target
package, or no recognized assertion verb). Any fix for Category 2 in particular
needs a signal from OUTSIDE the test body (a claim annotation, a "package no longer
exists" fact, or similar) rather than a purely syntactic tweak. This should be
treated as exploratory at plan time, not assumed solvable by a small heuristic
patch.

## References

- `pkg/gate/step_testverify.go` — `MandatedTest` type + `ExtractMandatedTests`;
  mandated tests originate from spec claims' `tests` lists
- `pkg/gate/substantiveness_join.go` — `NoTargetViolation` (lines 55-70, the
  "does not call package X" decision table), `IsTestHollow`,
  `RouteSubstantivenessFindings`, `HollowFindingsToViolations` — the core,
  language-agnostic join/decision half of the SPEC-037 pack/core split
- `packs/substantiveness/ast-grep/rules/hollow-test-go.yml` — source of the Q1
  hollow-test pack rule (the fixed assertion-verb regex with no `TestMain`
  exemption); installed copy at
  `.backstop/packs/backstop/substantiveness/ast-grep/rules/hollow-test-go.yml`
- `packs/substantiveness/ast-grep/rules/referenced-symbol-go.yml` — source of the
  Q2 referenced-symbol extraction pack rule (spec-unaware; feeds but does not
  perform the noTarget join)
- `SPEC-037-traceability-substantiveness-pack.spec.md` — the spec that split
  substantiveness into pack (Q1/Q2 language-specific rules) + core (routing/join)
- `cmd/backstop/integration_test.go` — `TestMain`, the Category 1 false positive
- `cmd/backstop/standards_removal_test.go`, `dogfood_pack_test.go`,
  `manifest_configerror_test.go`, `checktype_catalog_test.go`,
  `checktype_catalog_guard_test.go` — the nine Category 2 false positives
- ISSUE-018 — the discovering change; pulled these files into diff scope, first
  surfacing the false positives
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy; block defects, don't
  manufacture false positives that erode trust in the gate
