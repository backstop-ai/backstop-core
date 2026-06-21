---
title: "Remove confirmed-dead / vestigial baked-in code surfaced by the eradication audit"
schema_version: issue/v1

issue:
  id: ISSUE-018
  title: "Remove confirmed-dead / vestigial baked-in code surfaced by the eradication audit"
  type: technical-debt
  status: open
  created: "2026-06-20"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Remove confirmed-dead / vestigial baked-in code surfaced by the eradication audit

## Problem

A codebase eradication audit identified two clusters of confirmed-dead or redundant code that survived the SPEC-034 cutover and the packs-only pivot. Neither cluster is gated on anything downstream — they can be deleted in a standalone pass with no behavior change.

### Section B — vestigial in-process semgrep path

`pkg/check/check.go` contains a bespoke `semgrepExecutor` type (lines 381–428) that shells
out to `semgrep --json --quiet <files>` with no `--config` argument. There is no root
`semgrep.yml` / `.semgrep/` config in the repo, so the invocation scans zero rules and
contributes zero findings. All real semgrep enforcement runs through the data-driven engine
dispatch (`pkg/pack/engine` + `cmd/backstop/dispatchPackEngines` → SARIF). The
`semgrepExecutor` is not a distinct capability — it is a dead code path that was never
reached by the engine model and can never produce useful output in its current state.

**Dead symbols (confirmed by audit):**

| Location | Symbol | Notes |
|---|---|---|
| `pkg/check/check.go:381–428` | `semgrepExecutor` type + `Execute` + `IsAvailable` | bespoke in-process executor; configless invocation is a no-op |
| `pkg/check/check.go:430–503` | `semgrepJSON`, `parseSemgrepJSON`, `semgrepSeverity`, `extractJSONDocument`, `ParseSemgrepJSONForTest` | tool-specific non-SARIF parser; sole caller is the executor above |
| `pkg/check/check.go:305–326` | `EnsureSemgrep` call-site in `Check` | provisions a tool that is never actually invoked by the engine path |
| `pkg/check/check.go:243–254` | `SemgrepEnsurer` interface + `DefaultSemgrepEnsurer` field on `Options` | wires the unused provisioner into `Options` |
| `pkg/check/check.go:26` | `Options.GolangciLintAvailable bool` | no production reader; confirmed dead by audit |
| `pkg/check/registry.go:248–254` | `execs[CheckTypeSemgrep] = &semgrepExecutor{...}` | registers the dead executor into the pass-order |
| `pkg/check/semgrep.go` | `EnsureSemgrep`, `ensureSemgrepWith`, `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller` | tool-specific Ensure; the generic `provisionEngines` already exists in `cmd/backstop/pack_gate_provision.go` — this is the exact "EnsureSemgrep should be generic" anomaly called out in earlier audit notes |
| `pkg/packval/executor.go:25–30` | `RunSemgrep` | pack-validation harness's bespoke semgrep runner; fixture execution should route through generic engine dispatch |

**Verification note:** before deleting the `CheckTypeSemgrep` entry from the pass-order in
`registry.go`, confirm it is not referenced by any test fixture or gate step that would
silently lose coverage. A SPEC-034-style deletion-assertion test asserting that the symbols
no longer exist is the safe approach.

### Section F — dead legacy native-standards validator

`pkg/validate/standard.go` implements a `validate.Standard` function that validates
`.standard.md` artifact files. This is the native-standards path slated for full removal
per the packs-only directive (2026-06-16). `validate.Standard` has no production caller —
only `standard_test.go` calls it. The function bakes a language allowlist
(`{go, typescript, python, bash}`), category/severity/detection enums, and a `semgrep`
field special-case (~line 330). None of this is reachable from any gate or CLI surface.

**Dead symbols (confirmed by audit):**

| Location | Symbol | Notes |
|---|---|---|
| `pkg/validate/standard.go` | entire file | native-standards validator; zero production callers |
| `pkg/config/config.go:24` | `Config.StandardsDirs` | no production reader |
| `pkg/validate/plan.go:698` | `.standard.md` entry/reference | dead reference to the removed artifact type |

### Why this is standalone

Both clusters are pure deletion with no behavior change. Neither rolls into the larger
pluggable-engine spec (SPEC-035) nor BUNDLE-009. The semgrep cleanup is not the
"EnsureSemgrep → generic provisionEngines" generalization task (that belongs to SPEC-034's
successor work); it is the removal of the dead executor that precedes or is independent of
that generalization. Capturing as a focused issue avoids contaminating larger specs with
trivial janitor work.

## Solution

Delete both clusters in a single branch. Suggested pass order:

1. Delete `pkg/validate/standard.go`. Remove `StandardsDirs` from `pkg/config/config.go`.
   Remove the `.standard.md` reference from `pkg/validate/plan.go:698`. Delete
   `standard_test.go` (its only caller is the deleted function).
2. Delete `pkg/check/semgrep.go` in full.
3. In `pkg/check/check.go`: remove `semgrepExecutor`, `semgrepJSON`, the semgrep parser
   helpers, the `SemgrepEnsurer` interface and `DefaultSemgrepEnsurer` field, the
   `EnsureSemgrep` call block, and the `GolangciLintAvailable` field.
4. In `pkg/check/registry.go`: remove the `execs[CheckTypeSemgrep]` wiring line.
5. In `pkg/packval/executor.go`: remove `RunSemgrep`; any callsite that used it routes
   through the generic engine dispatch.
6. Add a deletion-assertion test (parallel to SPEC-034 style) that asserts none of the
   deleted symbol names appear in the compiled binary or test binaries, so the check cannot
   be silently re-introduced.

After deletion the gate must run green at `--all` scope; no new test functions need to be
added beyond the deletion-assertion test.

## References

- `pkg/check/check.go` — `semgrepExecutor` (lines 381–428), `semgrepJSON` + parsers (lines 430–503), `SemgrepEnsurer` + `DefaultSemgrepEnsurer` (lines 243–254), `EnsureSemgrep` call block (lines 305–326), `GolangciLintAvailable` field (line 26)
- `pkg/check/registry.go` — `execs[CheckTypeSemgrep]` wiring (lines 248–254)
- `pkg/check/semgrep.go` — `EnsureSemgrep`, `ensureSemgrepWith`, `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller` (whole file)
- `pkg/packval/executor.go` — `RunSemgrep` (lines 25–30)
- `pkg/validate/standard.go` — whole file; the `.standard.md` native-standards validator
- `pkg/config/config.go:24` — `Config.StandardsDirs` (no production reader)
- `pkg/validate/plan.go:698` — dead `.standard.md` entry
- `cmd/backstop/pack_gate_provision.go` — `provisionEngines`; the generic replacement for `EnsureSemgrep`
- Packs-only directive (2026-06-16) — strategic decision to remove the native-standards path; BUNDLE-010 context
- SPEC-034 — the pluggable-engine cutover that rendered Section B redundant
