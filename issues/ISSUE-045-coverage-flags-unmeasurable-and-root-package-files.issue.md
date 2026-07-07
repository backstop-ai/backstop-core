---
title: "Coverage Flags Unmeasurable And Root Package Files"
schema_version: issue/v1

issue:
  id: ISSUE-045
  title: "Coverage Flags Unmeasurable And Root Package Files"
  type: bug
  status: open
  created: "2026-07-06"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-045: Coverage Flags Unmeasurable And Root Package Files

## Problem

**Reframed 2026-07-06 during implementation.** This issue started as two
`coverage_unmeasured` false-positives in `pkg/gate/step_coverage.go`. Both
are real, but implementation proved neither can be fixed where the original
write-up put them, because of a deeper constraint that spans the coverage
engine's sandbox boundary AND the pack-command dispatch AND how this whole
subsystem is tested. This issue is now that larger finding. It is a
**prerequisite for the rest of DIR-015** (Gate Checker Hardening): further
gate-hardening work verified only by unit tests that bypass the sandboxed
dispatch is unproven against the real installed-pack path, and this issue is
the concrete case that exposed it.

### The two original false positives (unchanged as symptoms)

1. **Zero-measurable-content files.** `pkg/pack/engine/fieldcontract.go`
   became types + consts only (0 functions, 0 executable statements) after
   ISSUE-027 moved the field-contract map into pack data. `go test
   -coverprofile` never emits a block for a file with no statements — the
   file is simply absent from the profile, not measured-and-failing.
   `coveragePathsInScope` (`pkg/gate/step_coverage.go:363`) has no
   zero-statement filter, so this file lands in the same coverage-required
   set as a real gap, and the step emits `coverage_unmeasured` for a file
   with nothing to measure.
2. **Root-package (`ROOT`) path collision.** The repo-root `embed.go` (~90%
   covered via `go test .`) is reported unmeasured. Go's raw coverage
   profile always keys records by full module import path
   (`github.com/bmanson/backstop-core/embed.go`), never repo-relative.
   `indexCoverageByPathMetric` (`step_coverage.go:270`) never strips that
   prefix, so exact matching never engages for ANY file, and every match
   falls back to a `strings.HasSuffix` scan requiring `found == 1`
   (`resolveCoverageRecordsForPath`, `step_coverage.go:298`). A root-package
   scope path has zero directory segments, so its suffix collapses to a
   bare basename match (`"/embed.go"`) — and this repo has a second,
   unrelated `cmd/backstop/embed.go`, so the scan finds two candidates,
   `found == 2`, and returns no match. The root `embed.go` record is real
   and present; it is discarded by an ambiguous basename collision.

### Root cause 1 — the sandboxed convert cannot normalize coverage records

The coverage producer's convert step (the go-toolchain pack's
`scripts/coverage-to-records.sh`) runs under `packval.SandboxedRunStdout` —
a macOS `sandbox-exec` deny-all profile (`deny default`, `deny file-write*`,
`deny network*`; `file-read*` scoped only to the pack dir + system dylib
dirs; CWD = pack dir). It is a pure stdin→stdout transformer with **no
toolchain or project access**. This was empirically verified twice: direct
shell reproduction of `go list -m` / `go list ./...` under that profile
fails with `open .../go.mod: operation not permitted`, and a real `backstop
gate` run still fired `coverage_unmeasured` on `fieldcontract.go` after
implementing a `go list`-based convert fix.

This means the go-toolchain convert **cannot** run `go list -m` or read
`go.mod` — so neither fix can happen inside the convert:
- Case 2's module-prefix strip needs the module path.
- Case 1's zero-statement N/A emission needs the package's file list (`go
  list -f '{{.GoFiles}}'`) to know which absent-from-profile files are
  genuinely zero-statement vs. untested-with-statements.

Records therefore reach the gate module-qualified no matter what the
convert does, which is the actual reason file→record matching fails or
collides.

### Root cause 2 — the consumer side cannot fix it without baking Go knowledge

`pkg/gate` is deliberately language-neutral (thin-executor first principle,
`feedback_zero_baked_checks`). Resolving `embed.go` vs. `M/embed.go` vs.
`M/cmd/backstop/embed.go`, or knowing that `fieldcontract.go` is
zero-statement, requires the Go module path / toolchain. Baking that into
`pkg/gate` or the dispatch would violate "zero baked language/tool
knowledge, for any language."

A sound consumer-side narrowing does exist and is worth keeping regardless:
disable the `HasSuffix` fallback for bare-basename scope paths, so a
root-package file can never silently borrow a nested file's record. But
without repo-relative records, that narrowing only converts "borrows the
wrong record" into "no record → still a false positive" — it does not fix
case 2 on its own, only makes the failure mode honest instead of silently
wrong.

### Root cause 3 — the integration gap that hid all of this (the important one)

Every existing `pkg/gate` coverage test feeds `[]check.CoverageRecord`
**directly**, bypassing the sandboxed dispatch entirely. `go test ./...` is
fully green while the real installed-pack coverage path — sandboxed convert,
module-qualified paths, dispatch limitations and all — is broken. This is
the recurring pack-provisioning integration gap
(`project_pack_provisioning_integration_gap`): implementations nail
unit-level correctness while the cross-boundary wiring goes unproven. It is
also why this issue escalated mid-implementation instead of shipping as
originally scoped — the plan's fix (a `go list`-based convert change) looked
correct under unit tests and was silently inert under the real sandbox.

Because coverage-hardening work under DIR-015 is otherwise verified the same
way (direct record injection), this issue is a **prerequisite**, not just a
sibling: DIR-015 work can't be trusted as "hardened" until at least one real
sandboxed-dispatch e2e path exists to prove the real thing isn't vacuously
green.

### Why it matters

Same enforcement-philosophy violation as ISSUE-034/040: the gate goes loud
on something that is not a defect (`CLAUDE.md`, "Loud ≠ blocking"), forcing
spurious waivers or blocking legitimate changes. Left unfixed:
(1) any future pack-data migration that empties a `.go` file down to
types/consts (the exact shape of ongoing thin-executor eradication work,
e.g. ISSUE-018, ISSUE-027, ISSUE-030) re-triggers case 1;
(2) any change touching a root-package file risks tripping case 2 whenever
another package anywhere in the module happens to reuse the same basename —
a latent, data-dependent flake that can appear or disappear as unrelated
files are added elsewhere in the tree;
(3) beyond this issue's two symptoms, the entire coverage dispatch path is
currently *unproven* against its real sandboxed form — any other
sandbox/dispatch-shaped defect in this path is equally invisible to today's
test suite.

## Solution (fix direction to evaluate — the planner details it)

The architecturally clean fix keeps language-specific enrichment out of both
`pkg/gate` (must stay language-neutral) and the sandboxed convert (has no
toolchain access), and puts it where it belongs: the **un-sandboxed engine
command**, which already runs `go test` via `runner.RunStdout` with full
toolchain and project access.

1. **Enrich in the engine command, not the convert.** The go-toolchain
   coverage engine command additionally captures `go list -m` (module path)
   and the package's `.go` files (`go list -f '{{.GoFiles}}'`), folding them
   into the artifact fed to the convert as plain text the sandboxed convert
   can parse without running any toolchain itself — e.g. appended
   `#backstop-module …` / `#backstop-gofile …` comment lines on `cover.out`.
2. **Parse, don't execute, in the convert.** `coverage-to-records.sh`
   (still sandboxed, still no toolchain) parses those comment lines to
   (a) strip the module prefix → true repo-relative record paths [fixes
   case 2], and (b) emit `total:0` N/A records for files that belong to a
   measured package but are genuinely zero-statement [fixes case 1] —
   critically distinguishing a zero-statement file (N/A) from an
   untested-but-has-statements file (must stay flagged; Go's profile already
   emits `total>0/covered=0` for those, so this distinction is derivable
   without new gate logic). The gate's existing `Total == 0 ⇒ N/A` guard
   then handles case 1 with **no gate-side change**.
3. **Fix the dispatch limitation that blocks step 1.** The current dispatch
   cannot run an enriched command like this: `splitCommand`
   (`cmd/backstop/pack_gate.go:805`) is a naive `strings.Fields` split with
   no quote handling, and the resulting first token is exec'd as-is with no
   packRoot resolution — so a compound `sh -c '...'` command, or a
   pack-provided producer script, is not runnable today. This is a change
   to `cmd/backstop/pack_gate.go` — the thin executor itself — so it needs
   careful, reviewed design: support a packRoot-resolved pack-provided
   producer/command, without baking any language knowledge into the
   dispatch (the dispatch stays language-neutral; only the go-toolchain
   pack's own command/script gains Go knowledge).
4. **Close the integration gap.** Add a real sandboxed-dispatch end-to-end
   test that runs the coverage path through the actual installed pack +
   the real `sandbox-exec` profile (not direct `[]check.CoverageRecord`
   injection), so a unit-green run can never again hide a broken real path.
   This is acceptance-critical for this issue, independent of cases 1/2.

### Constraints / non-negotiables

- Do **not** relax the sandbox for coverage converts — that guts the
  ISSUE-029 convert trust boundary. Explicitly rejected as a fix direction.
- Do **not** bake Go/language knowledge into `pkg/gate` or into the
  dispatch itself (`cmd/backstop/pack_gate.go`). Language knowledge belongs
  only in the go-toolchain pack's own command/scripts.
- The anti-vacuous-green guards must survive unchanged: an
  untested-with-statements file, in any language — e.g. the bun seeded-defect
  fixture `src/app.ts` — must still fire `coverage_unmeasured`. Only
  genuinely zero-statement files may become N/A.
  `TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen` is the
  concrete regression guard for this: an earlier gate-side "directory
  proxy" approach for case 1 (treat any no-record file whose directory has
  ≥1 record as N/A) was implemented, found to defeat this exact guard
  (Go and bun are structurally identical in record-only data — a
  zero-statement Go file and an untested-but-real bun file both present as
  "no own record, dir is measured" — so no language-neutral, record-only
  proxy can tell them apart), and was reverted. Do not re-attempt a
  gate-side record-only proxy for case 1; the fix must be producer-side per
  the direction above.

**Tests to add (both cases, plus the integration-gap closer, exercising the
real `StepCoverageThresholdScopedFunc` AND, for the e2e closer, the real
sandboxed dispatch):**
- A changed measurable-source file with zero functions/statements
  (`fieldcontract.go`-shaped fixture) produces no `coverage_unmeasured`
  violation.
- A changed measurable-source file that DOES have functions but genuinely
  has no coverage record still produces the violation (regression guard,
  mirroring ISSUE-034's CLM-002 pattern).
- A root-package file (module-prefixed record path with zero directory
  segments after the module prefix) that IS measured produces no
  `coverage_unmeasured` violation, including a fixture where a same-basename
  file exists elsewhere in the tree (reproducing the `embed.go` /
  `cmd/backstop/embed.go` collision directly).
- A root-package file that is genuinely unmeasured (no record at all) still
  produces the violation (regression guard).
- The bun seeded-defect fixture (`src/app.ts`) continues to red
  (`TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen` stays
  green as a guard) — proves the Go-specific producer-side fix does not
  leak into or weaken the non-Go path.
- **New:** a real end-to-end test that installs the go-toolchain pack and
  drives the coverage step through the actual sandboxed dispatch (real
  `sandbox-exec`, real `go test`, real convert script) against a repo
  fixture containing both a zero-statement file and a root-package
  same-basename collision — proving the fix works through the real path,
  not just against direct record injection.

**Acceptance:** a change touching only `fieldcontract.go`-shaped
(zero-statement) files produces zero `coverage_unmeasured` violations for
them; a change touching a measured root-package file (with or without a
same-basename collision elsewhere in the module) produces zero
`coverage_unmeasured` violations for it; both companion "still genuinely
unmeasured" regression cases keep firing; the bun anti-vacuous-green guard
is unaffected; and the new real-sandboxed-dispatch e2e test passes,
demonstrating the fix holds through the actual installed-pack path rather
than only through direct-record-injection unit tests.

## Meta — working-tree state at time of this rewrite (2026-07-06)

Nothing from this work is committed. Current tree state, for whoever picks
this back up:
- `pkg/gate/step_coverage.go` has a sound consumer-side case-2 narrowing
  (disabling the ambiguous bare-basename `HasSuffix` fallback) — worth
  keeping, but insufficient alone per Root cause 2 above.
- `.../go-toolchain/scripts/coverage-to-records.sh` (in the local pack
  testdata fixture used by `cmd/backstop` tests) has `go list`-based changes
  for both cases — these are **inert under the real sandbox** (Root cause
  1) and need reworking into the plain-text-comment-parsing approach
  described in the Solution section.
- An unsound gate-side case-1 proxy (directory-level "no record + measured
  sibling ⇒ N/A") was implemented, found to defeat
  `TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen`, and was
  reverted. Do not re-attempt it (see constraints above).
- New test files `pkg/gate/coverage_rootpath_test.go` and
  `pkg/gate/coverage_unmeasurable_test.go` exist locally exercising the
  direct-record-injection path only; they do not yet cover the real
  sandboxed dispatch (Root cause 3's gap).
- A draft plan exists at
  `plans/PLAN-ISSUE-045-coverage-unmeasured-and-root-files.plan.yml`
  reflecting the pre-reframe (original two-symptom) scope and needs
  re-authoring against this rewritten issue before implementation resumes.

## References

- `pkg/gate/step_coverage.go:363` (`coveragePathsInScope`) — builds the
  coverage-required set from `scope.Files` filtered only by
  `classifier.IsMeasurableSource`; no zero-statement filter
- `pkg/gate/step_coverage.go:270` (`indexCoverageByPathMetric`) — keys
  records by `normalizeScopePath("", r.Path)`, which never strips a Go
  module import-path prefix, so the map key is never a true repo-relative
  path
- `pkg/gate/step_coverage.go:298` (`resolveCoverageRecordsForPath`) — exact
  match never hits (per above); the `strings.HasSuffix` fallback requires
  `found == 1`, and a root-package scope path's bare-basename suffix
  collides with `cmd/backstop/embed.go` for the `embed.go` case
  (`found == 2`, no match returned)
- `pkg/gate/scope.go:91` (`normalizeScopePath`) — `filepath.Clean` +
  `ToSlash` only; no module-path knowledge, by design (language-neutral
  gate)
- `pkg/pack/engine/fieldcontract.go` — the concrete zero-function fixture:
  post-ISSUE-027, types + consts only, 0 lines matched in a real
  `-coverprofile` run
- `embed.go` (repo root) / `cmd/backstop/embed.go` — the concrete
  same-basename collision that breaks case 2's suffix fallback
- `cmd/backstop/pack_gate.go:805` (`splitCommand`) — naive
  `strings.Fields` split, no quote handling, no packRoot resolution; the
  dispatch limitation blocking the producer-side enrichment fix
- `packval.SandboxedRunStdout` — the macOS `sandbox-exec` deny-all profile
  the coverage convert runs under; confirmed empirically to deny `go list`
  / `go.mod` reads (see Root cause 1)
- go-toolchain pack (`backstop/go-toolchain`,
  `scripts/coverage-to-records.sh`) — the coverage convert; currently emits
  Go's raw module-prefixed profile paths unmodified; the `go list`-based fix
  attempted here is inert under the sandbox (see Meta)
- `TestBunFixture_SeededUncoveredTsSourceRedsGateNotVacuousGreen`
  (`cmd/backstop`) — the anti-vacuous-green guard that the reverted
  gate-side case-1 proxy defeated; the concrete regression case-1's fix must
  not break
- DIR-015 (Gate Checker Hardening) — parent directive; this issue is now a
  **prerequisite** for the rest of DIR-015, not just scoped work under it,
  because it is the case that surfaced the sandboxed-dispatch integration
  gap DIR-015's other hardening work is equally exposed to
- ISSUE-034 (gate-coverage-flags-deleted-files, closed) — sibling in the
  same family: coverage asserting a positive per-path obligation off
  `scope.Files` without enough filtering, there for deletions, here for
  zero-content and root-package files
- ISSUE-029 (macos-sandbox-blocks-convert-interpreters) — establishes the
  convert sandbox trust boundary this issue's fix must not relax
- ISSUE-040 (gate-substantiveness-scans-testdata-fixtures, open) — sibling
  family: the gate scanning/measuring the wrong file set in a different
  dimension (`test_substantiveness` vs `coverage_threshold`)
- ISSUE-027 (eradicate-default-registry-into-packs, open) — the change that
  emptied `fieldcontract.go` down to types/consts and surfaced case 1; also
  the general shape of thin-executor migrations (pack-data extraction) that
  will keep re-triggering case 1 until fixed
- Project memory `project_pack_provisioning_integration_gap` — the
  recurring pattern (impls nail unit tests, miss cross-package/real-dispatch
  wiring) this issue is now the concrete DIR-015 instance of
- Project memory `feedback_zero_baked_checks` — the standing rule that
  ruled out both a consumer-side Go-aware fix in `pkg/gate` and a
  dispatch-level language-aware fix in `pack_gate.go`
- Project memory `gate_scope_and_coverage` — prior coverage/scope hardening
  precedent this issue continues
