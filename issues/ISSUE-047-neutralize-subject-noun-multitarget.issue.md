---
title: "Neutralize Subject Noun Multitarget"
schema_version: issue/v1

issue:
  id: ISSUE-047
  title: "Neutralize Subject Noun Multitarget"
  type: technical-debt
  status: open
  created: "2026-07-07"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# ISSUE-047: Neutralize Subject Noun Multitarget

## Problem

The substantiveness "noTarget" check — the anti-vacuous-test guard that
verifies a mandated test actually references the code unit it claims to
exercise — derives its target from the spec-level field
`implementation.package`, and applies that ONE target uniformly to every
claim in the spec. This is two compounding thin-executor defects.

### 1. Baked Go + this-repo layout knowledge in the gate binary

`pkg/gate/substantiveness_join.go`, `TargetPackageName(implementationPackage
string)`:

```go
func TargetPackageName(implementationPackage string) string {
	if strings.HasPrefix(implementationPackage, "cmd/") {
		return ""
	}
	if !strings.HasPrefix(implementationPackage, "pkg/") {
		return ""
	}
	return filepath.Base(implementationPackage)
}
```

This bakes two assumptions directly into core: that implementation code
lives under `pkg/`/`cmd/` (a layout convention this repo happens to use,
not a language- or project-neutral fact), and that "the import unit" is
the directory basename (a Go convention — packages other ecosystems don't
necessarily key targets on the last path segment of a directory). That is
baked language/layout knowledge in the gate binary — the exact species of
defect the thin-executor eradication program (DIR-014/DIR-015) exists to
drain. Backstop core is supposed to compare OPAQUE tokens; what counts as
"referencing the subject" is pack territory. The referenced-symbol
EXTRACTION findings this function's output is joined against are already
pack-sourced (`RouteSubstantivenessFindings`) — only this derivation step
is stranded in core, un-migrated.

### 2. `package` is a language-ish noun, and a spec can only declare ONE target for ALL its claims

`implementation.package` is a REQUIRED schema field
(`artifacts/spec/v1/schema.json`, `implementation` block:
`"required_keys": ["summary", "package"]`) and is applied uniformly to
every claim in `pkg/gate/step_testverify.go`:

```go
targetPkg := TargetPackageName(fm.Implementation.Package)

for _, claim := range fm.Claims {
	...
	for _, testName := range claim.Tests {
		tests = append(tests, MandatedTest{
			FuncName:  testName,
			SpecFile:  path,
			TargetPkg: targetPkg,   // same target for every claim in the spec
			...
		})
	}
}
```

A single spec-level scalar cannot express a spec whose claims legitimately
exercise different code units.

### Concrete trigger — SPEC-030

SPEC-030 ("Packs-Only — Native Standards Removal") legitimately spans TWO
packages:

- CLM-002 / CLM-003 / CLM-023 → `TestPkgCheck_*` tests that live in and
  reference `pkg/check` (a LIVE package — the SARIF/coverage/runner
  substrate, NOT the deleted baked engine) → correctly satisfy target
  `check`.
- CLM-015 → `TestGate_SucceedsWithoutStandards`, which now legitimately
  exercises `pkg/gate` (a positive behavioral claim — "gate/code-check
  succeeds on a project with no compiled standards directory" — it is
  *not* `kind: absence`).

Because the single spec-level `implementation.package: pkg/check` can't
express this split, CLM-015's test carries a grandfathered `noTarget`
finding (identity `bd6936adec528aa2e711fc9d6c525cea87ac98c2a42071c4fc3a04e924b20558`
in `.backstop/baseline.json`). It is currently GREEN only because it is
baselined — it is a faithful signal of the schema's one-target-per-spec
limitation, not a false positive to silence and not evidence the guard is
miscalibrated.

Two fixes are explicitly WRONG and are ruled out here so a future
implementer doesn't reach for them:

- **(a) Blindly retargeting the spec to `pkg/gate`.** This backfires: it
  strands the 3 `TestPkgCheck_*` claims as NEW `noTarget` violations,
  trading one baselined finding for three.
- **(b) Mislabeling CLM-015 `kind: absence`.** `NoTargetViolationForTest`
  already skips `kind: absence` tests (that's why absence claims are
  exempt today), so this would silence the finding — but CLM-015 is a
  positive behavioral claim, not an absence claim. Relabeling it to dodge
  the check is dishonest and exactly the vacuous-green failure mode the
  guard exists to prevent.

## Solution (pinned scope — decided, not open questions)

These were decided with the user; the planner should treat them as the
chosen approach, not re-litigate them.

1. **Neutralize the noun.** Rename the schema field
   `implementation.package` to a language-neutral term — recommend
   `subject` (final wording is a small open detail, not a decision point).
   Keep the migration non-breaking for existing specs: either accept the
   old key as a deprecated alias or migrate all existing specs' frontmatter
   in the same change. Do not leave two field names live indefinitely.

2. **De-bake the derivation.** Remove the `pkg/`/`cmd/` path-prefix and
   basename layout logic from core (`TargetPackageName`). Core should hold
   and compare opaque subject tokens only; the language/layout-specific
   notion of "does this test reference the subject" moves pack-side,
   alongside the existing Q2 referenced-symbol extraction rule it's joined
   against. No baked path-prefix or basename convention may remain
   anywhere in the gate path after this fix.

3. **Multi-target = PER-CLAIM subject with a spec-level default.** Each
   claim MAY declare its own `subject`; a claim without one inherits the
   spec-level default. This was chosen OVER a spec-level any-of array
   (`subjects: [pkg/check, pkg/gate]`, matched against whichever is
   referenced) — the array was considered and rejected because it weakens
   the anti-vacuous-green guard: a test could reference the "wrong" listed
   subject (any member of the array) and still pass, which is the same
   failure class as the guard exists to catch. Per-claim override with a
   default keeps every claim's target unambiguous and sharp. Under this
   design, SPEC-030 migrates with ONE override line on CLM-015
   (`subject: pkg/gate`); the 3 `TestPkgCheck_*` claims keep inheriting the
   spec default (`pkg/check`).

4. **Close the self-pack coverage gap.** Add a rule to the `backstop/self`
   dogfood pack (lives in its own repo per
   `feedback_packs_always_external` — this issue only scopes adding the
   rule, not editing that repo directly) that catches the baked
   subject-noun / layout-prefix pattern going forward. This is currently a
   COVERAGE GAP, not a rule that exists and passes: the self-pack today
   hunts tool/extension literals (e.g. `go build`, `.go`), not the
   `package` field noun or `pkg/`/`cmd/` prefix conventions. Closing the
   gap is part of the fix, not a follow-on, so this exact defect class
   can't regress silently.

## Acceptance

- SPEC-030's CLM-015 `noTarget` finding clears LEGITIMATELY — via a
  per-claim `subject: pkg/gate` override, NOT by mislabeling it
  `kind: absence` and NOT by a blind spec-level retarget — and the 3
  `TestPkgCheck_*` claims (CLM-002/CLM-003/CLM-023) stay satisfied (no new
  `noTarget` violations introduced by the migration).
- The guard stays SHARP (non-vacuous) after the change. Both directions
  are covered by tests:
  - Positive control: a claim whose test references its correct (default
    or overridden) subject is satisfied.
  - Negative control: a claim whose test references the WRONG subject
    still fires `noTarget` loudly.
- No baked `pkg/`/`cmd/` path-prefix or Go-basename layout literal remains
  anywhere in the gate path (`pkg/gate/...`) — grep-verifiable and
  self-pack-verifiable.
- The new self-pack rule (closing the coverage gap in item 4 above) is in
  place and the `backstop/self` pack run is green.
- `backstop artifact validate` passes for the migrated schema and any
  touched specs; `backstop gate` stays exit 0 on this repo's own tree.

## Non-goals / boundaries

- This is the ISSUE only. Do NOT implement here, do NOT edit SPEC-030 or
  `artifacts/spec/v1/schema.json` as part of authoring this issue — the
  SPEC-030 migration and schema change happen later, via the plan/implement
  loop and the spec-author agent respectively.
- Do NOT hand-edit any other artifact as a side effect of filing this one.
- Keep the schema change non-breaking for existing specs (see Solution
  item 1) — this is a boundary condition on the eventual fix, not
  optional polish.

## References

- **DIR-015** (Gate Checker Hardening) — parent theme; this issue is part
  of the checker-hardening / thin-executor-eradication program.
- **`feedback_zero_baked_checks`** (agent memory, STANDING RULE) — zero
  baked checks/policies/tool-knowledge; any baked logic is an eradication
  target by default. `TargetPackageName`'s `pkg/`/`cmd/`/basename logic is
  exactly this category.
- **`feedback_packs_always_external`** (agent memory) — the self-pack rule
  added under item 4 lives in the `backstop/self` pack's own repo, not
  vendored into this repo's tracked tree.
- **SPEC-037** (substantiveness pack split, BUNDLE-009 Seed 3) — prior art
  for the gate-side/pack-side split this issue extends; `TargetPackageName`
  was relocated (not de-baked) during that split, per the comment in
  `pkg/gate/substantiveness_join.go` describing it as a "behavior-preserving
  relocation of the deleted analyzer's targetPackageName" — this issue is
  the follow-up that actually removes the baked behavior rather than just
  relocating it.
- **ISSUE-035** (gate substantiveness flags TestMain/absence tests) — prior
  work in the same `noTarget`/absence decision-table area
  (`NoTargetViolationForTest`, `kind: absence` skip logic) that this issue's
  fix must not regress.
