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
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Remove confirmed-dead / vestigial baked-in code surfaced by the eradication audit

## Problem

A codebase eradication audit identified confirmed-dead or redundant code that survived the
SPEC-034 cutover and the packs-only pivot. None of it is gated on anything downstream.

### Scope decision (2026-07-05) — delete `backstop code check` outright

Section B originally scoped a surgical removal of only the dead in-process `semgrepExecutor`
from within `pkg/check/check.go`, leaving the `backstop code check` command itself in place.
That framing is superseded: the decision is now to **delete the `backstop code check` command
entirely**, not just its dead semgrep executor.

**Why:** `backstop code check` is itself a vestigial pre-packs-only standalone code-checker
carrying baked language/tool knowledge — the exact class of defect the thin-executor invariant
targets. The gate does **not** depend on it. Verified:

- The gate runs lint/build/test/findings through `dispatchPackEngines`
  (`cmd/backstop/pack_gate.go:246`), never through `pkg/check.Run`.
- `cmd/backstop/gate_cutover_step2_test.go:60` and `:119` are existing tests that explicitly
  assert the gate's step list never includes `gate.StepCodeCheck` and never calls
  `pkg/check.Run` — this was already proven by the SPEC-034 cutover, not a new claim.
- `pkg/check/registry.go:214` carries its own comment admitting the toolchain-executor
  construction it performs "serves the SURVIVING `backstop code check` subcommand" — i.e. the
  package itself documents that this machinery has no other caller.

So deleting the command is strictly a superset of the old Section B: removing
`backstop code check` removes the semgrep executor along with it, and removes several more
baked-language findings that Section B did not previously target (see below). This does **not**
touch the gate — the gate's own lint/build/test/findings enforcement is unaffected, because it
never ran through this path.

### Section B (expanded) — delete `backstop code check` and its dead engine machinery

**The command and its entry points:**

| Location | Symbol | Notes |
|---|---|---|
| `cmd/backstop/code_check.go` | whole file — `newCodeCheckCommand`, `codeCheckCmd`, `resolveCheckRun` | the Cobra command itself |
| `cmd/backstop/root.go:70–72` | `codeCmd` namespace registration of `codeCheckCmd` | wiring that exposes the command on the CLI |

**The engine machinery it alone drives** (confirmed by callgraph — every symbol below has its
sole live caller inside the `check.Run` / `Check` / `buildExecutorsForConfigErr` chain that only
`code_check.go` invokes):

| Location | Symbol | Notes |
|---|---|---|
| `pkg/check/check.go` | `Run`, `Check`, `passOrder`, `Options` struct | the whole in-process check engine; `semgrepExecutor` (lines 381–428), `semgrepJSON` + parsers (430–503), `SemgrepEnsurer`/`DefaultSemgrepEnsurer` (243–254), `EnsureSemgrep` call block incl. `CheckTypeSemgrep` degraded-error handling (305–326), `GolangciLintAvailable` field (line 26) — all previously-scoped Section B items are subsumed here |
| `pkg/check/registry.go` | `buildExecutorsForConfig`, `buildExecutorsForConfigErr`, `resolveToolchain`, `declaredEntries`, `validateToolchainKeys`, `DeclaredToolchainExecutorsForTest`, `Toolchain`/`ToolchainEntry` types, `execs[CheckTypeSemgrep]` registration block | the whole toolchain-executor registry; the module's own comment at line 214 states it exists only to serve `backstop code check` |
| `pkg/check/semgrep.go` | whole file — `EnsureSemgrep`, `ensureSemgrepWith`, `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller` | tool-specific Ensure; the generic `provisionEngines` already exists in `cmd/backstop/pack_gate_provision.go` |
| `pkg/check/manifest.go` | `LoadManifest`, `Manifest` type, `RouteFile`, `routeFileDefaults`, `defaultManifest` | **baked file-extension routing** — `routeFileDefaults` has `case ".go", ".ts", ".tsx":` (line 92), a hard-coded stack-extension list. Sole live caller is `check.go:220` `e.Manifest.RouteFile(f)`, itself only reachable from `check.Run`. This is a NEW finding for this issue (see self-pack findings below) — it was not in the original Section B table. Note: this is distinct from the already-removed `.manifest.json` compiled-standards reader (`compiledManifestFile` etc.), which BUNDLE-011 deleted separately; what remains here is the "built-in default routing" branch, a different dead surface. |
| `pkg/config/config.go:34` | `Config.SemgrepVersion` (`semgrep_version` yaml key) | REPLACED by SPEC-035's trusted-tool allowlist `{tool → pinned version}` map — removing it is not a capability loss |
| `cmd/backstop/gate.go:618–619` | `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion` | pin-forwarding at the gate entry point; dead once the executor it fed is gone. (The `code_check.go:139–140` sibling forwarding site is deleted along with the whole file.) |

**PRESERVE — the shared SARIF surface (do NOT delete):** `pkg/check/parsers.go` mixes dead and
live code in one file. These symbols are LIVE and consumed by the gate's own
`dispatchPackEngines` path — deleting them would break the gate itself:

- `check.ParsePackFindings`, `parseSarif`, `sarifFingerprint`, `sarifSeverity` — used by
  `cmd/backstop/pack_gate.go:647` (`runFindingsEngine`) and `pkg/gate/substantiveness_q1_dispatch.go`.
- `check.ParsePackCoverage` (`pkg/check/coverage.go`) and the `CoverageRecord`/`Violation`/
  `ConfigError`/`CommandRunner`/`CheckType` types — used throughout `pack_gate.go`
  (`dispatchPackCoverage`, `runCoverageEngine`, `dispatchPackEngines`'s `runner check.CommandRunner`
  parameter). `CheckType` in particular stays the gate's pass-identity vocabulary (see the SPEC-041
  catalog sharp edge below) — the enum type itself is not being removed, only the dead-path
  consumers of it.
- `lookupParser` itself is LIVE — `ParsePackFindings` calls `lookupParser("sarif")` (line 47).
  It must **not** be deleted; only trimmed.

**Baked findings this issue must clear from `parsers.go`** (fires 2 self-pack
`no-baked-language-token` findings): the `formatParsers` map (`parsers.go:22–27`) carries
`"eslint-json": parseESLintJSON` and `"tsc": parseTscOutput` entries. Both parsers, and the
`"regex-lines": parseRegexLines` entry plus `parseRegexLines`/`parseRegexLinesWith`/
`defaultRegexLinesPattern`/`eslintFile`, are reachable **only** from `registry.go`'s dead
toolchain-executor construction (`entry.Format` lookups at `registry.go:192` and the
`testEntry.Format = "regex-lines"` assignment at `registry.go:247`) — confirmed no other caller
exists. Trim `formatParsers` down to `{"sarif": parseSarif}` and delete the eslint/tsc/regex-lines
parser bodies; keep `lookupParser`, `Parser` type, and the SARIF entry.

**Confirmed by the backstop/self dogfood pack:** the above `manifest.go` routing and `parsers.go`
format map together fire the following live self-pack findings today, all reachable only through
this vestigial command path:

1. `pkg/check/manifest.go` `routeFileDefaults()` — `case ".go", ".ts", ".tsx":` → 3 findings
   (`no-baked-language-token` ×2, `no-language-literal-on-neutral-spine` ×1).
2. `pkg/check/parsers.go` `formatParsers` map — `"eslint-json"`, `"tsc"` entries → 2 findings
   (`no-baked-language-token` ×2).

### Sharp edge for the planner — SPEC-041 CheckType-consumer catalog goes stale

`cmd/backstop/checktype_catalog.go` (`CheckTypeConsumerCatalog`) is a machine-readable catalog
of "GATE-SEMANTIC" `CheckType`-keying sites, reconciled against live source by a completeness
guard in `cmd/backstop/checktype_catalog_guard.go` (regex `gateSemanticKeyingRe`, matching
`passOrder`, `map[CheckType]`, `parseCheckType`, etc.). Four of its eight catalog rows name sites
this issue deletes:

- **C-4** `cmd/backstop/code_check.go:CheckTypeFindings` — file deleted entirely.
- **C-6** `pkg/check/check.go:passOrder` — deleted along with `check.Run`.
- **C-7** `pkg/check/registry.go:Entries` — deleted along with the toolchain-executor registry.
- **C-8** `pkg/check/manifest.go:parseCheckType` — deleted along with `manifest.go`'s routing
  machinery (note: `parseCheckType` itself is also used by `validateToolchainKeys`, which is
  also being deleted — confirm no other caller before removing the function itself).

**C-5** (`pkg/check/parsers.go:CheckTypeFindings`) is the LIVE call inside `ParsePackFindings`
(`parsers.go:51`, `return parser(out, CheckTypeFindings)`) and must be **kept** in the catalog —
it survives this deletion untouched.

The guard test is designed to fail loud exactly when a cataloged site disappears without
reconciliation (it already has a precedent for this: rows C-2/C-3 were removed from the live
catalog when their sites were genuinely deleted in an earlier phase). The implementer must
update `CheckTypeConsumerCatalog()` to remove the C-4/C-6/C-7/C-8 rows (or otherwise reconcile
them per the guard's documented convention) as part of this deletion, or the guard test will
correctly go red. This is a real dependency this issue introduces on a SPEC-041 dogfood
artifact — flagging it explicitly so the planner sequences it as a required step, not an
afterthought discovered mid-implementation.

### Section F — dead legacy native-standards validator + a baked file-category literal

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

**New addition — `pkg/validate/plan.go` `fileCategory()` baked `.go` literal:** `fileCategory`
(around line 703) maps a task's touched file paths to a work category so
`checkFinalPhaseCategoryCoverage` can require the plan's final verification phase to cover every
category of file the plan touches. Its implementation is:

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

The `.go` suffix hard-codes "code" category membership to Go source. A TypeScript (or any
non-Go) plan's impl files (`.ts`, `.tsx`, `.py`, …) fall through to `""` (uncategorized) and are
silently exempted from the final-phase-verification-coverage requirement this function enforces
— the opposite of a baked-check false-negative: it doesn't wrongly flag non-Go plans, it wrongly
lets them skip a real requirement. This fires 1 self-pack `no-language-literal-on-neutral-spine`
finding. This file is already touched by this issue (the dead `.standard.md` reference at
~line 698), so folding the `.go` de-baking in here avoids a second pass over the same file.

**Open design point — flagged for the planner, not resolved here:** how should `fileCategory`
learn "this is an impl/code file" language-neutrally?

- *Option A:* source the classification from pack-declared classification globs — SPEC-043
  added pack-declared classification globs for exactly this kind of test-vs-impl / file-role
  question. This is the "real" fix but may be heavier than this issue's blast radius (artifact
  validation running with no project/pack context loaded at validate time is a real question —
  confirm whether `pkg/validate/plan.go` has access to pack config at the point `fileCategory`
  runs).
- *Option B:* if pack-glob wiring is too heavy for this issue, at minimum move the literal off
  the neutral-spine strict zone per the self-pack's `no-language-literal-on-neutral-spine` rule
  (e.g. an explicit narrow exemption/allowlist mechanism already used elsewhere in the codebase
  for similar spine literals), deferring the language-neutral glob-sourcing to a follow-on.

Plan TDD-ordering / final-phase-coverage validation inherently needs *some* way to distinguish
"this is code that needs verification" from "this is prose/docs" — how it should learn that
without baking a language is a genuinely open question, not a known recipe. The planner should
resolve this explicitly rather than defaulting to whichever option is easiest to code.

### Out of scope — moved to ISSUE-019

`pkg/packval/executor.go:25–30` (`RunSemgrep`) was previously listed in this issue's Section B
table. It is **removed from this issue's scope.** ISSUE-019 ("De-Go the packval pack-validation
harness") owns `pkg/packval/` in full, including a proper engine-convergence redesign rather than
a narrow deletion of `RunSemgrep` in isolation — see ISSUE-019's own "Why this is the same root
cause" / Solution sections. Do not touch `pkg/packval/` as part of this issue.

### Why this is standalone

All of the above is pure deletion (plus one narrow, flagged-open de-baking) with no gate
behavior change. None of it rolls into the larger pluggable-engine spec (SPEC-035) nor
BUNDLE-009 — the gate's own lint/build/test/findings enforcement runs entirely through
`dispatchPackEngines` and is untouched by any of this. Capturing as a focused issue avoids
contaminating larger specs with janitor work, while the SPEC-041 catalog reconciliation and the
`fileCategory` open question are called out explicitly so they aren't discovered as surprises
mid-implementation.

**Shared sites with SPEC-035 — sequencing note (unchanged from original scope):** SPEC-035
REQ-005 renames `CheckTypeSemgrep` to a neutral enum value across the codebase. Since this issue
deletes the entire `semgrepExecutor` body (including its `CheckTypeSemgrep` sites) as part of
deleting `check.Run` wholesale, **this issue lands first**; SPEC-035's rename then applies only
to whatever `CheckTypeSemgrep` references remain after this deletion (likely none in `pkg/check`,
possibly some in shared enum definitions — confirm at implementation time).

## Solution

Delete the whole `backstop code check` command and its dead engine machinery, the dead
native-standards validator, and de-bake the one `plan.go` literal. Suggested pass order:

1. Delete `pkg/validate/standard.go`. Remove `StandardsDirs` from `pkg/config/config.go`.
   Remove the `.standard.md` reference from `pkg/validate/plan.go:698`. Delete
   `standard_test.go` (its only caller is the deleted function).
2. In `pkg/validate/plan.go`: resolve the `fileCategory()` `.go`-literal open design point (see
   above) — either wire pack-declared classification globs (SPEC-043) or apply the narrower
   neutral-spine exemption, per the planner's decision. Update/add tests accordingly.
3. Delete `cmd/backstop/code_check.go` in full. Remove its registration from
   `cmd/backstop/root.go:70–72`.
4. **Before deleting anything in `pkg/check`, run a callgraph pass** confirming which symbols are
   referenced only from the `code_check.go` → `check.Run` chain versus symbols reused by the live
   gate/coverage paths (`ParsePackFindings`, `parseSarif`, `sarifFingerprint`, `sarifSeverity`,
   `lookupParser`, `ParsePackCoverage`, `CoverageRecord`, `Violation`, `ConfigError`,
   `CommandRunner`, `CheckType`). This issue's own verification pass (see Problem section) found
   the split; the implementer must re-confirm against the tree at implementation time rather than
   trusting this issue's line numbers, since they will have shifted.
5. Delete `pkg/check/check.go`'s `Run`, `Check`, `passOrder`, `Options` and everything reachable
   only from them: `semgrepExecutor` + `Execute`/`IsAvailable`, `semgrepJSON` + parser helpers,
   `SemgrepEnsurer`/`DefaultSemgrepEnsurer`, the `EnsureSemgrep` call block, the `CheckTypeSemgrep`
   degraded-error handling, `GolangciLintAvailable`.
6. Delete `pkg/check/semgrep.go` in full.
7. Delete `pkg/check/registry.go`'s `buildExecutorsForConfig`, `buildExecutorsForConfigErr`,
   `resolveToolchain`, `declaredEntries`, `validateToolchainKeys`,
   `DeclaredToolchainExecutorsForTest`, `Toolchain`/`ToolchainEntry`, and the
   `execs[CheckTypeSemgrep]` registration block.
8. Delete `pkg/check/manifest.go`'s `LoadManifest`, `Manifest`, `RouteFile`, `routeFileDefaults`,
   `defaultManifest`. Keep `CheckType`, `String()`, `parseCheckType` ONLY if still referenced by a
   surviving caller after step 7 — otherwise delete it too (feeds the SPEC-041 catalog
   reconciliation in step 10).
9. In `pkg/check/parsers.go`: trim `formatParsers` to `{"sarif": parseSarif}`; delete
   `parseESLintJSON`, `parseTscOutput`, `eslintFile`, `parseRegexLines`, `parseRegexLinesWith`,
   `defaultRegexLinesPattern`. Keep `Parser`, `lookupParser`, `ParsePackFindings`, `parseSarif`,
   `sarifFingerprint`, `sarifSeverity` untouched — these are live.
10. Update `cmd/backstop/checktype_catalog.go`'s `CheckTypeConsumerCatalog()` to remove the C-4,
    C-6, C-7, C-8 rows (their sites no longer exist), consistent with how C-2/C-3 were previously
    reconciled. Keep C-5 (`parsers.go:CheckTypeFindings`) — it survives. Confirm
    `checktype_catalog_guard.go`'s completeness guard passes clean afterward.
11. In `pkg/config/config.go`: remove the `SemgrepVersion` field (`semgrep_version` yaml key).
12. In `cmd/backstop/gate.go`: remove the `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion`
    forwarding block (its `code_check.go` sibling is gone with the whole file in step 3).
13. Add deletion-assertion tests (parallel to the existing `pkg/validate/deletion_assertion_test.go`
    ISSUE-018 Section F style) asserting: the `code check` subcommand is absent from the CLI
    (`backstop code check` / `backstop code` produce a "no such command" style result, or the
    `code` namespace itself is gone if nothing else lives under it — confirm at implementation
    time whether `codeCmd` has other children), and that none of the deleted `pkg/check` symbol
    names appear in non-test source.

After deletion: `go build ./...` and `go test ./...` must be clean, and the backstop/self dogfood
gate must no longer RED on any of the 6 findings enumerated in the Problem section (manifest.go
×3, parsers.go ×2, plan.go ×1). The gate (`pack_engines` dispatch) must be unaffected, since it
never used the `code check` path — no new test functions are needed to prove this beyond the
existing cutover tests (`gate_cutover_step2_test.go`) which already assert it.

## References

- `cmd/backstop/code_check.go` — whole file, the `backstop code check` command
- `cmd/backstop/root.go:70–72` — `codeCmd` registration of `codeCheckCmd`
- `cmd/backstop/gate_cutover_step2_test.go:60,119` — existing tests proving the gate never
  wires `gate.StepCodeCheck` / never calls `pkg/check.Run`
- `cmd/backstop/pack_gate.go:246` — `dispatchPackEngines`, the gate's real dispatch path
- `pkg/check/check.go` — `Run`, `Check`, `passOrder`, `Options`; `semgrepExecutor` (381–428),
  `semgrepJSON` + parsers (430–503), `SemgrepEnsurer`/`DefaultSemgrepEnsurer` (243–254),
  `EnsureSemgrep` call block incl. `CheckTypeSemgrep` degraded-error handling (305–326),
  `GolangciLintAvailable` (line 26)
- `pkg/check/registry.go` — `buildExecutorsForConfig`, `buildExecutorsForConfigErr`,
  `resolveToolchain`, `declaredEntries`, `validateToolchainKeys`,
  `DeclaredToolchainExecutorsForTest`, `execs[CheckTypeSemgrep]` block; line 214's own comment
  documents the whole registry as serving only `backstop code check`
- `pkg/check/manifest.go:90–92` — `routeFileDefaults` baked `case ".go", ".ts", ".tsx":` routing;
  `LoadManifest`/`RouteFile`/`defaultManifest`
- `pkg/check/parsers.go:22–27` — `formatParsers` map (`eslint-json`, `tsc`, `regex-lines` entries
  to trim; `sarif` entry to keep); line 47 `lookupParser("sarif")` inside the LIVE
  `ParsePackFindings` — `lookupParser` itself must be preserved
- `pkg/check/semgrep.go` — whole file: `EnsureSemgrep`, `ensureSemgrepWith`,
  `DefaultSemgrepEnsurer`, `SemgrepResolver`, `DefaultSemgrepInstaller`
- `pkg/config/config.go:34` — `Config.SemgrepVersion` / `semgrep_version` yaml key; REPLACED by
  SPEC-035 trusted-tool allowlist
- `cmd/backstop/gate.go:618–619` — `opts.PinnedSemgrepVersion = cfg.Enforcement.SemgrepVersion`
  forwarding
- `cmd/backstop/checktype_catalog.go` — `CheckTypeConsumerCatalog()`, rows C-4/C-6/C-7/C-8 go
  stale, C-5 survives
- `cmd/backstop/checktype_catalog_guard.go` — the completeness guard that reconciles the catalog
  against live source; must pass after the catalog update in Solution step 10
- `pkg/validate/standard.go` — whole file; the `.standard.md` native-standards validator
- `pkg/config/config.go:24` — `Config.StandardsDirs` (no production reader)
- `pkg/validate/plan.go:698` — dead `.standard.md` entry
- `pkg/validate/plan.go:702–715` — `fileCategory()` baked `.go` suffix; open design point, see
  Problem section
- `pkg/validate/deletion_assertion_test.go:48` — existing deletion-assertion style precedent
  (ISSUE-018 Section F) to extend for this issue's new deletions
- `cmd/backstop/pack_gate_provision.go` — `provisionEngines`; the generic replacement for
  `EnsureSemgrep`
- SPEC-035 — trusted-tool allowlist replaces `SemgrepVersion` pin; ISSUE-018 must land first
- SPEC-043 — pack-declared classification globs; candidate mechanism for de-baking
  `fileCategory`'s `.go` literal (Option A above)
- ISSUE-019 — owns `pkg/packval/` in full (including `RunSemgrep`), moved out of this issue's scope
- ISSUE-030 — separately eradicated the `.standard.md` scaffolder (`pkg/pack/scaffold.go`) and
  the dead `.manifest.json` compiled-standards reader in `manifest.go`; this issue's `manifest.go`
  deletions target the *different*, still-live "built-in default routing" branch, not that
  already-removed reader
- Packs-only directive (2026-06-16) — strategic decision to remove the native-standards path;
  BUNDLE-010 context
- SPEC-034 — the pluggable-engine cutover that rendered the original Section B redundant, and
  whose cutover tests (`gate_cutover_step2_test.go`) substantiate this issue's "gate is
  unaffected" claim
- Thin-executor eradication backlog (2026-06-20 audit) — origin of both original clusters
- CLAUDE.md — zero-baked-checks first principle; "a baked language/tool branch is a defect to
  eradicate, never to extend"
