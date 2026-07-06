---
title: "Gate Substantiveness Flags Testmain Absence Tests"
schema_version: issue/v1

issue:
  id: ISSUE-035
  title: "Gate Substantiveness Flags Testmain Absence Tests"
  type: bug
  status: closed
  created: "2026-07-05"
  closed: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: safe

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/gate/..."

implementation:
  summary: >
    Fixed in two independently-landable tracks. Category 1 (TestMain flagged
    hollow): added a name-based exclusion to the pack's hollow-test-go.yml
    ast-grep rule (both tracked copies), then reinstalled the
    backstop/substantiveness pack so the gate runs the fixed rule. Category 2
    (absence/structural tests flagged "does not call package X"): added an
    opt-in, default-off `kind: absence` field to the claims schema and to the
    claims frontmatter parsed by ExtractMandatedTests, which sets
    MandatedTest.IsAbsence per-claim; a new NoTargetViolationForTest wrapper in
    substantiveness_join.go skips the noTarget set-join for IsAbsence tests
    while leaving the underlying NoTargetViolation decision table (and its
    caller) unchanged for every unannotated test. The 11 genuine
    absence/structural claims in SPEC-030 (6) and SPEC-041 (5) were then
    annotated `kind: absence` via the spec-author agent, clearing the original
    Category-2 false positives. Delivered on main in the squash commit
    d5efd5b ("feat: eradicate `backstop code check` + un-vacuum gate
    dimensions").
  package: pkg/gate

requirements:
  - id: REQ-001
    text: >
      TestMain (Go's test-harness entry point, `func TestMain(m *testing.M)`)
      must be unconditionally exempt from the substantiveness pack's
      hollow-test rule, while an ordinary genuinely-hollow `Test*` stub in the
      same file must still be flagged. The exemption must be name-scoped, not
      a blanket suppression.
  - id: REQ-002
    text: >
      A spec claim may opt in to `kind: absence`, marking its mandated
      test(s) as absence/structural. ExtractMandatedTests must set
      MandatedTest.IsAbsence to true for a test mandated by such a claim, and
      false for a test mandated by an ordinary claim in the same spec (the
      flag is per-claim, not per-spec).
  - id: REQ-003
    text: >
      When a mandated test's IsAbsence is true, the gate must skip the
      noTarget substantiveness set-join for that test — it must raise no
      violation even under inputs (non-empty target package, not
      same-package, target absent from the referenced set) that would
      otherwise be a noTarget violation.
  - id: REQ-004
    text: >
      The absence annotation must be default-off: an UNannotated mandated
      test (IsAbsence false) must still raise a noTarget violation under the
      same not-in-set conditions, so the capability cannot silently blanket-
      blind the check.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Running the substantiveness pack's hollow-test-go rule over a fixture
      containing TestMain produces no hollow finding keyed to TestMain.
    tests:
      - TestQ1_Go_TestMain_ProducesNoHollowFinding
  - id: CLM-002
    requirement: REQ-001
    text: >
      In the same fixture pass that exempts TestMain, a genuinely hollow
      `Test*` stub in the same file still produces a hollow finding keyed to
      it (the exemption is name-scoped, not blanket).
    tests:
      - TestQ1_Go_TestMainExemption_StillFlagsGenuineHollow
  - id: CLM-003
    requirement: REQ-002
    text: >
      ExtractMandatedTests sets IsAbsence=true for a test mandated by a
      `kind: absence` claim and IsAbsence=false for a test mandated by an
      ordinary claim in the same spec.
    tests:
      - TestExtractMandatedTests_SetsIsAbsenceFromClaimKind
  - id: CLM-004
    requirement: REQ-003
    text: >
      NoTargetViolationForTest returns no violation for an IsAbsence=true
      mandated test even when the target package is non-empty, the test is
      not same-package, and the target is absent from the referenced set.
    tests:
      - TestNoTargetViolationForTest_SkipsAbsenceTest
  - id: CLM-005
    requirement: REQ-004
    text: >
      NoTargetViolationForTest still raises a noTarget violation for an
      IsAbsence=false mandated test under the identical not-in-set inputs
      that CLM-004 skips — the anti-blinding guard proving the annotation is
      opt-in, not a blanket exemption.
    tests:
      - TestNoTargetViolationForTest_UnannotatedStillRaises

contracts:
  - file: pkg/gate/substantiveness_join.go
    provides:
      - name: NoTargetViolationForTest
        kind: function
        signature: "func NoTargetViolationForTest(mt MandatedTest, referenced ReferencedSymbolSet, samePackage bool) (Violation, bool)"
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

## Fix directions evaluated

1. **`TestMain` exemption (Category 1)** — straightforward: add a name-based
   exclusion to `hollow-test-go.yml` in `packs/substantiveness/`. Low uncertainty.
   **CHOSEN** — shipped as-is.
2. **Broaden the hollow heuristic** — treat a test as substantive if it makes ANY
   assertion OR references os/filesystem/exec/lock-style APIs (e.g. `os.Stat`,
   `filepath.Walk`, `exec.Command`), not only the fixed verb regex. Still a pack-YAML
   change, but changes the semantics of "hollow" more broadly than a single
   exemption — needs care not to swallow genuinely empty stub tests. **REJECTED** —
   a genuinely hollow stub that happens to touch `os.Stat` would be silently
   excused, trading a false-positive problem for a worse false-negative one.
3. **Absence-test annotation for the noTarget check (Category 2)** — allow a spec
   claim to mark a mandated test as an absence/structural claim (e.g. a `kind:
   absence` field alongside `tests` in the claims schema, or a claim-text
   convention the gate recognizes) so `NoTargetViolation` in
   `pkg/gate/substantiveness_join.go` skips the target-package join for tests so
   marked. This is a core + schema change, not a pack change. **CHOSEN** — shipped
   as an opt-in, default-off `kind: absence` claim field (see Resolution).
4. **Reframe the noTarget join around what's actually being proven** — e.g. treat a
   test whose target package no longer exists on disk (the deletion case) as
   automatically satisfying "does not call package X" without a claim annotation.
   Doesn't cover the `backstop.yml`/`backstop.lock`-reading structural tests
   (dogfood_pack_test.go), which reference no deleted package at all. **REJECTED as
   primary** — covers only the deletion subset; the annotation subsumes it with
   zero false negatives.

**Core uncertainty, stated honestly:** distinguishing a genuine hollow/no-target
stub from a legitimate absence or structural-invariant test is not mechanical from
the test body alone — both "forgot to write real assertions" and "correctly proves
absence" can look identical to an AST-only heuristic (no call to the target
package, or no recognized assertion verb). The shipped fix resolves this by taking
the signal from OUTSIDE the test body (the claim annotation), exactly as this
section anticipated, rather than a purely syntactic tweak.

## Resolution

Delivered on `main` in the squash commit `d5efd5b` ("feat: eradicate `backstop
code check` + un-vacuum gate dimensions").

**Category 1 (`TestMain` hollow)** — added a name-based exclusion to
`hollow-test-go.yml` so the match is "`Test`-named AND not `TestMain` AND has no
assertion call." Applied to both tracked copies (`packs/substantiveness/ast-grep/
rules/hollow-test-go.yml`, the durable source, and `pkg/gate/testdata/
substantiveness-pack/ast-grep/hollow-test-go.yml`, the unit-harness copy the Go
tests execute) so the source pack and the test harness agree. The
`backstop/substantiveness` pack was reinstalled (`backstop pack remove
backstop/substantiveness && backstop pack add ./packs/substantiveness`) so the
gitignored installed copy the real gate executes carries the fix too, and
`backstop.lock`'s content hash was recomputed.

**Category 2 (noTarget false positives)** — added an opt-in, default-off `kind`
field to the claims schema (`artifacts/spec/v1/schema.json`) and to the claims
frontmatter parsed by `ExtractMandatedTests` (`pkg/gate/step_testverify.go`),
which sets `MandatedTest.IsAbsence` per-claim (`claim.Kind == "absence"`). A new
`NoTargetViolationForTest` wrapper in `pkg/gate/substantiveness_join.go` skips the
noTarget set-join when `IsAbsence` is true; the original `NoTargetViolation`
decision table and its caller are unchanged, so an unannotated claim keeps full
enforcement (the anti-blinding guard, CLM-005). This keeps the mislabel risk
explicit and review-visible rather than silent: an author can only excuse a test
by visibly writing `kind: absence` into a reviewed spec claim.

The 11 genuine absence/structural claims that motivated this issue were then
annotated via the spec-author agent (not hand-edited):
- **SPEC-030** (`specs/SPEC-030-packs-only-native-standards-removal.spec.md`) — 6
  claims mandating `TestNoProductionImportOfCompile`,
  `TestCompiledStandardsArtifactsAbsent`, `TestPkgCompileDirectoryAbsent`,
  `TestStdGo001SourceAbsent`, `TestGate_SucceedsWithoutStandards`,
  `TestDogfood_BackstopYmlDeclaresGoStandardsPack`,
  `TestDogfood_GoStandardsLockVerifies`,
  `TestDogfood_StaleSlotlyLockEntryRemoved`.
- **SPEC-041** (`specs/SPEC-041-coverage-reimpl-checktype-catalog.spec.md`) — 5
  claims mandating `TestCatalog_EnumeratesGateSemanticConsumersExcludesDisplaySites`,
  `TestCatalog_SurvivingSitesNotMistaggedDeleted`,
  `TestCatalog_GuardScansGateSemanticSurfaceOnly`,
  `TestCatalog_GuardFailsOnUnlistedConsumer`, `TestCatalog_GuardFailsOnStaleEntry`.

`TestStandardScaffolder_Untouched` (SPEC-039) needed no annotation — SPEC-039 is
`status: replaced` (terminal), already excluded from enforcement.

**Accepted residual (not forced green):**
- Per-claim granularity is coarse: a claim mixing an absence test and a genuine
  call-the-package test would blanket-skip both. Mitigation is convention (keep
  absence claims single-purpose), not enforced.
- Mislabel risk: an author could tag a real stub's claim `kind: absence` to dodge
  the gate. Accepted because it is explicit and review-visible, not silent green.
- Tracked follow-ups surfaced by this same commit but out of scope here:
  ISSUE-036 (kind-aware contracts signature compiler), ISSUE-037 (iota-member
  const contracts), ISSUE-038 (contract-drift ratchet), ISSUE-039
  (`TestGate_SucceedsWithoutStandards` lost its behavioral assertion), ISSUE-040
  (gate scans testdata fixtures).

## Verification

- `go test ./pkg/gate/...` — green, including CLM-001–CLM-005's mandated tests.
- `./bin/backstop gate` — the `test_substantiveness` step no longer reports the
  original 15 false positives (TestMain cleared via the rule exemption +
  reinstall; the 11 SPEC-030/SPEC-041 tests cleared via the `kind: absence`
  annotation; `TestStandardScaffolder_Untouched` cleared via SPEC-039's terminal
  exclusion, not this fix).
- `./bin/backstop artifact validate --all` — the evolved claims schema, the
  annotated SPEC-030/SPEC-041, and this issue all validate.

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
