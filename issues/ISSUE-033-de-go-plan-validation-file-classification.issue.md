---
title: "De-Go plan validation file classification"
schema_version: issue/v1

issue:
  id: ISSUE-033
  title: "De-Go plan validation file classification"
  type: technical-debt
  status: open
  created: "2026-07-05"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# De-Go plan validation file classification

## Problem

`pkg/validate/plan.go`'s `fileCategory()` (~line 703) classifies a plan task's touched
files into work categories — `"artifact"`, `"code"`, or `""` (uncategorized) — and that
classification drives TDD-ordering checks and `checkFinalPhaseCategoryCoverage`'s
final-phase verification-coverage requirement. The `"code"` bucket is decided by a single
baked literal:

```go
func fileCategory(path string) string {
	artifactExts := []string{".spec.md", ".plan.yml", ".adr.md", ".bundle.md", ".issue.md"}
	for _, ext := range artifactExts {
		if strings.HasSuffix(path, ext) {
			return "artifact"
		}
	}
	if strings.HasSuffix(path, ".go") {
		return "code"
	}
	// Other extensions (e.g. .md for docs) don't map to a required category
	return ""
}
```

`strings.HasSuffix(path, ".go")` bakes a Go-specific file-naming assumption into
backstop's own artifact/plan validation — a violation of the zero-baked-language
invariant (backstop must know no language; see CLAUDE.md's "What backstop IS"). For a
plan describing work in a TypeScript, Python, or any non-Go project, whose task files are
`.ts`, `.tsx`, `.py`, etc., the classification silently mis-fires: those files fall
through to `""` (uncategorized) rather than `"code"`, and are silently exempted from the
final-phase-verification-coverage requirement this function exists to enforce. This is
the dangerous direction of failure — it doesn't wrongly flag a non-Go plan, it wrongly
lets one skip a real requirement. The `backstop/self` dogfood pack flags this line under
`no-language-literal-on-neutral-spine`.

### Why this is deferred rather than fixed inline in ISSUE-018

ISSUE-018 (deleting the vestigial `backstop code check` command and its dead
native-standards validator) touches this same file already — for an unrelated, dead
`.standard.md` reference — and originally flagged this `.go` literal as an "open design
point for the planner" with two options: (A) source the classification from pack-declared
classification globs (SPEC-043), or (B) apply a narrower neutral-spine exemption now and
defer glob-sourcing to a follow-on.

**Decided 2026-07-05:** ISSUE-018 goes with Option B. Its implementation keeps the
current `.go` behavior and suppresses the self-pack finding with a scoped `// nosemgrep`
comment that references **this issue (ISSUE-033)** by ID — so the baked literal stays
loud and tracked in the self-pack's own findings output rather than silently buried by
the suppression. This issue is that tracked follow-on: it owns Option A, the real
language-neutral fix.

The reason Option A doesn't collapse into ISSUE-018 is a genuine, unresolved design
question, not scope padding — see below.

### Why this isn't a drop-in glob swap

SPEC-043 already built the exact mechanism this needs, for a sibling problem: the gate's
coverage step used to have the same kind of baked-extension classification, and now
reads a pack-declared, language-neutral classifier instead —
`gate.SourceClassifier` (`pkg/gate/classification.go`), constructed by
`mergeSourceClassifier(packs []*pack.Manifest) gate.SourceClassifier`
(`cmd/backstop/gate.go:1017`) from the merged `classification.source` /
`classification.test` globs across every installed toolchain pack's manifest
(`pkg/pack/manifest.go`'s `Manifest.Classification`). `fileCategory`'s `"code"` bucket is
conceptually the same question `SourceClassifier.IsMeasurableSource` already answers.

The reuse is not mechanical, because `validate.Plan` runs in a genuinely different
context than the gate:

- **`validate.Plan`'s entry point discards its second argument.**
  `pkg/validate/plan.go:31` — `func Plan(art *artifact.ParsedArtifact, _ *schema.Schema)
  ValidationResult` — takes a `*schema.Schema`, not a classifier or pack/project handle,
  and doesn't use even that. There is currently no channel for pack-declared config to
  reach `fileCategory` at all.
- **That signature is shared across every artifact type, not just plans.**
  `cmd/backstop/artifact_route.go:14` defines `ValidatorFunc` as
  `func(*artifact.ParsedArtifact, *schema.Schema) validate.ValidationResult` and
  `validatorRouter` (line 17-24) maps ALL SIX artifact types (`spec`, `plan`, `adr`,
  `bundle`, `issue`, `directive`) onto that one function type. Widening `validate.Plan`'s
  signature to accept a classifier means either breaking the shared `ValidatorFunc`
  contract (forcing every other validator to accept a parameter it doesn't use) or
  finding a different injection seam (e.g. a plan-specific override path, a
  package-level/contextual default, or a broader `ValidatorFunc` redesign). Which of
  these is right is not decided.
- **The only production caller is `ValidateArtifacts` in
  `cmd/backstop/artifact_validate.go`** (the `validatorFn(art, sch)` call at line 169),
  invoked from the standalone `backstop artifact validate` CLI path — a path that
  currently has no pack-discovery or `mergeSourceClassifier` wiring at all (that wiring
  today lives entirely in `cmd/backstop/gate.go`'s gate-run path). Whether/how
  `artifact validate` should discover and load installed packs just to classify plan
  task files is itself an open question, not a known recipe.
- **The test blast radius is large but mechanical.** `validate.Plan(art, nil)` (or
  `(art, sch)`) is called directly from ~90 call sites across 8 test files
  (`pkg/validate/plan_test.go`, `plan_gate_test.go`, `plan_type_test.go`,
  `plan_final_test.go`, `plan_testtask_test.go`, `plan_compat_test.go`,
  `terminal_rules_test.go`, `terminal_acceptance_test.go`) plus one production call site.
  A signature change touches all of them; this is busywork, not design risk, but it's
  real scope the planner needs to size.

None of this makes the fix unclear in direction — SPEC-043's classifier is the right
target — but it does mean "wire the classifier in" is cross-cutting work with a real
unresolved seam (where does `validate.Plan` get pack/project context from), not a
same-file literal swap.

## Solution

Make `pkg/validate/plan.go`'s file classification (the `fileCategory` "code" bucket, and
any sibling baked `.go`/`_test.go` literal introduced by ISSUE-018's interim fix) source
its test-vs-impl / code-vs-other decision from pack-declared classification globs
(SPEC-043's `Classification.Source` / `Classification.Test`, merged the same way
`mergeSourceClassifier` does for the gate) instead of a baked extension literal.

Suggested shape (for the planner to confirm, not prescribed):

1. Resolve the injection-seam question first: how does `validate.Plan` (or its caller)
   obtain a `gate.SourceClassifier`-equivalent at the point `fileCategory` runs, given
   `ValidatorFunc` is shared across all six artifact types and `artifact validate` has no
   existing pack-discovery wiring. Candidate directions include a plan-specific validator
   signature carve-out, a classifier passed through `ValidateConfig`/`ValidateArtifacts`
   and threaded only to the plan path, or relocating `fileCategory`'s call sites behind an
   interface that defaults to a no-op/absent classifier when none is available (mirroring
   the gate coverage step's "classification capability absent" non-blocking state rather
   than silently falling through to `""`).
2. Replace the `strings.HasSuffix(path, ".go")` check with a classifier lookup once the
   seam is decided.
3. Remove the interim `// nosemgrep` suppression that ISSUE-018 adds (it must reference
   this issue's ID; removing it here closes the loop).
4. Update/add tests for `fileCategory` and `checkFinalPhaseCategoryCoverage` covering a
   non-Go classifier configuration (e.g. `.ts`/`.py` source globs) to prove the
   language-neutral path actually classifies correctly, not just that it compiles.

**Acceptance:** `fileCategory` (and any sibling baked `.go`/`_test.go` literal in plan
validation) carries zero baked language tokens; a plan describing a non-Go project's task
files classifies them correctly via pack-declared globs; the `backstop/self`
`no-language-literal-on-neutral-spine` finding on `pkg/validate/plan.go` is genuinely
eradicated (not suppressed); the interim `// nosemgrep` referencing ISSUE-033 is removed.

## References

- `pkg/validate/plan.go:703-715` — `fileCategory()`, the baked `.go`-suffix literal
- `pkg/validate/plan.go:31` — `func Plan(art *artifact.ParsedArtifact, _ *schema.Schema)
  ValidationResult`, the discarded schema argument / entry point with no pack context
- `pkg/validate/plan.go:671,681` — `fileCategory` call sites feeding TDD-ordering and
  `checkFinalPhaseCategoryCoverage`
- `cmd/backstop/artifact_route.go:12-24` — `ValidatorFunc` shared signature and
  `validatorRouter`, mapping all six artifact types onto one function type
- `cmd/backstop/artifact_validate.go:101-189,169` — `ValidateArtifacts`, the sole
  production caller (`validatorFn(art, sch)`) and the `backstop artifact validate` CLI
  path with no existing pack-discovery wiring
- `pkg/validate/plan_test.go`, `plan_gate_test.go`, `plan_type_test.go`,
  `plan_final_test.go`, `plan_testtask_test.go`, `plan_compat_test.go`,
  `terminal_rules_test.go`, `terminal_acceptance_test.go` — ~90 `validate.Plan(...)` test
  call sites; mechanical blast radius for a signature change
- SPEC-043 (`specs/SPEC-043-pack-declared-globs-coverage-consumer.spec.md`) — the
  pack-declared classification-globs mechanism this issue builds on
- `pkg/gate/classification.go` — `SourceClassifier`, `NewSourceClassifier`,
  `IsMeasurableSource` / `IsTestFile`; the reference implementation of the language-neutral
  classifier for the sibling (gate coverage) problem
- `pkg/pack/manifest.go` — `Manifest.Classification` (`Source`/`Test` glob lists parsed
  from a pack's `classification:` block)
- `cmd/backstop/gate.go:1007-1026` — `mergeSourceClassifier(packs)`, the merge-across-packs
  logic to reuse/adapt for the plan-validation path
- ISSUE-018 — the deletion issue whose interim `// nosemgrep` suppression (Option B) defers
  the real fix to this issue; its "Open design point" section (Option A vs B) is the origin
  of this issue's scope
- `backstop/self` pack rule `no-language-literal-on-neutral-spine` — the dogfood finding
  this issue eradicates for real (rather than suppresses)
