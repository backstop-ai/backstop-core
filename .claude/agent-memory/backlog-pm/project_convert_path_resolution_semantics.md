---
name: convert-path-resolution-semantics
description: A binding's `convert:` resolves against the CONSUMING pack's root, not the declaring pack's — so base-engines' dangling convert is by design, and ~15 fixture converts dangle harmlessly; any "convert must exist" check is consumer-side
metadata:
  type: project
---

An engine binding's `convert:` path is **resolved against the pack being
dispatched, not the pack that declared the binding**. `cmd/backstop/pack_gate.go:271`
computes `packRoot := filepath.Join(packDir, manifest.NormalizedName)` for the
manifest under dispatch; `:845` joins `binding.Convert` onto THAT root
(`pkg/packval/executor.go:217` does the same on the packval path).

**Why:** base-engine bindings (`packs/base-engines/pack.yml`, embedded whole via
root `embed.go`'s `//go:embed all:packs/base-engines`, served by
`pkg/baseengines.Registry()`) are registry DEFAULTS merged into a consuming
pack's dispatch. `packs/base-engines/` contains ONLY `pack.yml` — yet its
`ast-grep` binding declares `convert: ast-grep/to-sarif.sh`. That is a
CONVENTION CONTRACT on consumers, not a path inside base-engines:
`packs/contracts/ast-grep/to-sarif.sh` and
`packs/substantiveness/ast-grep/to-sarif.sh` both exist at exactly that
relative path. Verified in tree 2026-08-18.

**How to apply:**
- Any proposed structural check of the form "a declared `convert:` must exist
  relative to the pack root" is **WRONG AS STATED** — it false-reds the base
  engine pack compiled into every backstop binary. The correct framing is
  consumer-side: every pack that inherits a convert-bearing base binding ships
  the script at the declared relative path. Say this before a planner
  implements the naive version (ISSUE-175 sketches the naive version).
- **A dangling `convert:` in a declaration-only manifest fixture is normal, not
  a defect.** ~15 such references across seven files
  (`pkg/pack/testdata/pack-pattern-arg.yml`, `pack-divergent-flags.yml`,
  `pkg/pack/engine/testdata/contracts-grep-engine.yml`,
  `contracts-astgrep-engine.yml`, `engines-block-valid.yml`,
  `cmd/backstop/testdata/exempt-matrix-bindings.yml`,
  `coverage-routing-bindings.yml`). Every convert inside a REAL pack directory
  resolves. Don't let an issue frame one of these as a singleton find.
- **Dispatch is already LOUD on a missing convert** — both call sites `os.Stat`
  and refuse with `broken pack %s: missing convert script %s`. So this class is
  authoring-time ergonomics (→ DIR-024), never a silent false-clean (→ would
  have been DIR-032's charter). Check the stat before accepting a
  "silent hole" framing. See [[feedback_verify_the_loss_claim]].
- `pkg/packval/phase1.go` already stats rule sources (`:52`), fixtures (`:63`),
  `rule.Validator` (`:69`) and scaffold paths (`:76`) — `convert:` is the one
  path-bearing field it skips. That's the fix site, with the precedent shape
  already in the function.
- **Sibling, still unfiled:** `PLAN-ISSUE-142` (completed) residual R3 —
  a phase-1 check for "this rule declares no engine input at all", explicitly
  recorded as new ecosystem-wide strictness and NOT filed. Same family as
  ISSUE-175; recommend one lane for both. Its R2 (unify `pkg/pack.Rule` and
  `pkg/packval.Rule`) is also unfiled.

Related: [[project_packval_phase3_family]], [[project_pack_rule_path_scoping_dispatch]],
[[project_linux_ci_residual_family]].
