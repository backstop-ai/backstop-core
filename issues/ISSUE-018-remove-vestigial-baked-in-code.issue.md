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
| `pkg/check/check.go:315–324` | `CheckTypeSemgrep` degraded-error handling block in `Check` | the `DegradedError` switch arm that calls `delete(opts.Executors, CheckTypeSemgrep)` is the body of the dead path; dies with the executor |
| `pkg/check/check.go:305–326` | `EnsureSemgrep` call-site in `Check` | provisions a tool that is never actually invoked by the engine path |
| `pkg/check/check.go:243–254` | `SemgrepEnsurer` interface + `DefaultSemgrepEnsurer` field on `Options` | wires the unused provisioner into `Options` |
| `pkg/check/check.go:26` | `Options.GolangciLintAvailable bool` | no production reader; confirmed dead by audit |
| `pkg/check/registry.go:249–254` | `execs[CheckTypeSemgrep] = &semgrepExecutor{...}` | registers the dead executor into the pass-order; the whole block including `pinnedVersion: opts.PinnedSemgrepVersion` wiring dies here |
| `pkg/check/semgrep.go` | `EnsureSemgrep`, `ensureSemgrepWith`, `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller` | tool-specific Ensure; the generic `provisionEngines` already exists in `cmd/backstop/pack_gate_provision.go` — this is the exact "EnsureSemgrep should be generic" anomaly called out in earlier audit notes |
| `pkg/config/config.go:34` | `Config.SemgrepVersion` (`semgrep_version` yaml key) | config field that feeds the pinned-version plumbing; no purpose once the in-process executor is gone; REPLACED by SPEC-035's trusted-tool allowlist `{tool → pinned version}` map — removing it is not a capability loss |
| `cmd/backstop/code_check.go:139–140` | `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion` | sets the pinned-version slot that feeds the executor; dead once the executor is removed |
| `cmd/backstop/gate.go:618–619` | `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion` | same pin-forwarding at the gate entry point; dead for the same reason |
| `pkg/check/registry.go:253` | `pinnedVersion: opts.PinnedSemgrepVersion` | consumes the forwarded pin inside the executor registration block; removed as part of the registration block above |
| `pkg/packval/executor.go:25–30` | `RunSemgrep` | pack-validation harness's bespoke semgrep runner; fixture execution should route through generic engine dispatch (see ISSUE-019) |

**Shared sites with SPEC-035 — sequencing note:** SPEC-035 REQ-005 renames
`CheckTypeSemgrep` to a neutral enum value across the codebase. Two of the sites in the
table above — `pkg/check/check.go:322` (`delete(opts.Executors, CheckTypeSemgrep)`) and
`pkg/check/registry.go:249` (`execs[CheckTypeSemgrep] = &semgrepExecutor{...}`) — are
shared between this issue and SPEC-035. **This issue lands first.** ISSUE-018 deletes
those two sites in their entirety as part of removing the in-process executor body; the
`CheckTypeSemgrep` rename in SPEC-035 then applies only to the references that remain
after this deletion. Do not attempt the rename at these two locations — they will already
be gone.

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
   `EnsureSemgrep` call block (including the `CheckTypeSemgrep` degraded-error handling at
   lines 315–324), and the `GolangciLintAvailable` field.
4. In `pkg/check/registry.go`: remove the entire `execs[CheckTypeSemgrep]` registration
   block (lines 249–254), which includes the `pinnedVersion: opts.PinnedSemgrepVersion`
   wiring.
5. In `pkg/config/config.go`: remove the `SemgrepVersion` field (line 34, `semgrep_version`
   yaml key). Per-tool version pinning is REPLACED by SPEC-035's trusted-tool allowlist
   `{tool → pinned version}` map; removing this field is not a capability loss.
6. In `cmd/backstop/code_check.go`: remove the `PinnedSemgrepVersion` forwarding block
   (lines 139–140). In `cmd/backstop/gate.go`: remove the same forwarding block (lines
   618–619). Both sites become dead code once steps 3–5 are complete.
7. In `pkg/packval/executor.go`: remove `RunSemgrep` (lines 25–30); the full engine-
   convergence redesign for packval is ISSUE-019.
8. Add a deletion-assertion test (parallel to SPEC-034 style) that asserts none of the
   deleted symbol names appear in the compiled binary or test binaries, so the check cannot
   be silently re-introduced.

After deletion the gate must run green at `--all` scope; no new test functions need to be
added beyond the deletion-assertion test.

## References

- `pkg/check/check.go` — `semgrepExecutor` (lines 381–428), `semgrepJSON` + parsers (lines 430–503), `SemgrepEnsurer` + `DefaultSemgrepEnsurer` (lines 243–254), `EnsureSemgrep` call block (lines 305–326), `CheckTypeSemgrep` degraded-error handling (lines 315–324), `GolangciLintAvailable` field (line 26)
- `pkg/check/registry.go` — `execs[CheckTypeSemgrep]` registration block (lines 249–254), including `pinnedVersion: opts.PinnedSemgrepVersion` (line 253)
- `pkg/check/semgrep.go` — `EnsureSemgrep`, `ensureSemgrepWith`, `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller` (whole file)
- `pkg/config/config.go:34` — `Config.SemgrepVersion` / `semgrep_version` yaml key; REPLACED by SPEC-035 trusted-tool allowlist
- `cmd/backstop/code_check.go:139–140` — `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion` forwarding
- `cmd/backstop/gate.go:618–619` — `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion` forwarding
- `pkg/packval/executor.go` — `RunSemgrep` (lines 25–30); full packval redesign is ISSUE-019
- `pkg/validate/standard.go` — whole file; the `.standard.md` native-standards validator
- `pkg/config/config.go:24` — `Config.StandardsDirs` (no production reader)
- `pkg/validate/plan.go:698` — dead `.standard.md` entry
- `cmd/backstop/pack_gate_provision.go` — `provisionEngines`; the generic replacement for `EnsureSemgrep`
- SPEC-035 — trusted-tool allowlist replaces `SemgrepVersion` pin; ISSUE-018 must land first (shared sites `check.go:322`, `registry.go:249` are deleted here before SPEC-035's rename applies)
- Packs-only directive (2026-06-16) — strategic decision to remove the native-standards path; BUNDLE-010 context
- SPEC-034 — the pluggable-engine cutover that rendered Section B redundant
