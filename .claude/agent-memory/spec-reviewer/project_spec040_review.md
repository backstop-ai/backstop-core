---
name: spec040-review
description: SPEC-040 (BUNDLE-011 Seed 2, keystone toolchain-pack cutover) passed review with one prose nit; key facts verified vs main
metadata:
  type: project
---

SPEC-040 (toolchain-pack-cutover, BUNDLE-011 Seed 2 keystone) — PASS on review 2026-06-24 (spec v1.0.0).

**Why:** Keystone cutover retiring pkg/check.Run as gate Step 2; routes lint/build/test through dispatchPackEngines only. Author RE-SCOPED off a finding: the LIVE baked-language violation is TypeScript, not Go.

**How to apply (verified vs main, reuse for SPEC-041 review):**
- `builtinToolchain("go")` returns EMPTY Entries (registry.go:60-65) — Go already bridged. TS stack (eslint/tsc/regex-lines) STILL baked (registry.go:67-90). Spec correctly targets TS, not a mis-scoped "delete Go path."
- `realCodeChecker` wired Step 2: gate.go:488 (construct) + :520 (StepCodeCheckScopedFunc). `resolveToolchain` registry.go:256, `commandExecutor` :116 — all named in REQ-002/CLM-007.
- Coverage seam (CLM-028): one `newSharedTestRunner` (gate.go:487) feeds BOTH realCodeChecker (:488) AND buildCoverageStep (:501). Spec keeps it transitionally w/ real mandated test — correct.
- `loadBridgedToolchainPacks` has the `language != "go"` short-circuit (gate.go:412-414) — REQ-004/CLM-014 correctly generalize it.
- Warning/StepsWarned mechanism (SPEC-036) exists result.go:80-144 — REQ-005/006 reuse, not invent.
- Live vacuous-green hazard CONFIRMED: `.backstop/packs/backstop/` has go-standards, NOT go-toolchain. go-toolchain exists ONLY under cmd/backstop/testdata/, not the dogfood install.
- Scope fence airtight: REQ-007/010→SPEC-039, REQ-011/012/013→SPEC-041 (incl build-pass exemption cv.Pass==CheckTypeBuild gate.go:1173). CheckType enum correctly survives SPEC-040 transitionally.

**Lone issue (prose nit, non-blocking):** summary lines 33/426 say go-toolchain "already exists on disk at .backstop/packs/backstop/go-toolchain" — false; it's only under testdata. Contradicts spec's own correct grounding flag (lines 430-437). Recommend qualify to "worked example under testdata." Does not affect coverage/scoping.

**Re-review 2026-06-24 (two follow-ups, PASS holds):**
1. Prose fix confirmed: summary now "exists as a fixture under cmd/backstop/testdata/ ... NOT in live dogfood install" — agrees w/ grounding flag, no contradiction.
2. NEW CLM-029 build-exemption seam — substantive & correct vs main: `gate.Violation.ProjectWide` set in EXACTLY ONE place (gate.go:1173 `cv.Pass==check.CheckTypeBuild`, inside deleted checkViolationsToGate); consumed pkg/gate/scope.go:194 (bypass diff-scope filter); engine path NEVER sets it. So post-cutover a build break in an unchanged file would be scope-filtered away → real regression window. CLM-029 closes it transitionally (preserve ProjectWide on engine path OR lockstep) w/ real test TestSeam_BuildBreakInUnchangedFileStillRedsDiffScopedGate + named contract dispatchBuildViolationProjectWide. Correctly TRANSITIONAL, not claiming permanent mechanism.
- Cross-spec handoff CLEAN: SPEC-041 REQ-004 owns permanent `exempt_from_scope_filter` engine-binding property → ProjectWide on engine path (pack_gate.go), PER-VIOLATION, prohibits CheckType-enum identity. SPEC-041 names gate.go:1173 as soon-dead orphan SPEC-040 deletes + prohibits it as locus. No double-ownership.

**Final verification 2026-06-24 (over-deletion fix, PASS holds):**
- Over-deletion fix CORRECT vs main: `backstop code check` subcommand (cmd/backstop/code_check.go) is a real surviving production caller of resolveToolchain/buildExecutorsForConfigErr via resolveCheckRun→check.Run (check.go:293→registry.go:223). Has --file 2s runtime-hook mode (SPEC-008). Bundle NEVER mentions the subcommand (grepped: zero hits for codeCheckCmd/--file/SPEC-008) — targets gate Step 2 / the ENGINE only. So subcommand SURVIVES; deleting resolveToolchain would strand it.
- Delete set consistent: DELETE builtinToolchain (only callers: resolveToolchain being edited + tests) + realCodeChecker/checkViolationsToGate (wired step only). RETAIN-REDUCED resolveToolchain/commandExecutor/buildExecutorsForConfigErr. NOTE: resolveToolchain calls builtinToolchain (registry.go:257) + branches on `builtin` bool — must be EDITED (drop overlay), not left calling deleted fn. Spec CLM-007 says exactly this.
- CLM-031 (TestCutover_NoDeletedSymbolHasSurvivingCaller) is a REAL guard not tautological: asserts deleted-symbols-absent AND retained-caller-chain-compiles; second conjunct fails if resolveToolchain deleted.
- **RESIDUAL SPEC-041 DRIFT (flag, not blocking SPEC-040):** SPEC-041 catalog C-7 prose (:527) + lines :153/:514 still say "SPEC-040 deletes resolveToolchain" — contradicts SPEC-040's corrected RETAIN. It's cross-ref prose, NOT a guard-enforced row verdict (resolveToolchain doesn't key CheckType for scope/dispatch/verdict; map-keying is buildExecutorsForConfig* which C-7 tags SURVIVING), so SPEC-041's REQ-006 guard won't trip — but it's a factual contradiction to fix when SPEC-041 next touched. See [[feedback_catalog_deleted_mislabel]].
- Cross-spec checks coordinator named both ALIGN: C-7 buildExecutorsForConfig* SURVIVING ✓ ; C-2 checkViolationsToGate DELETED ✓.
