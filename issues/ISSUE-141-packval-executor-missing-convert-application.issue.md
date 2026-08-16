---
title: "pkg/packval/executor.go's RunEngine never applies a binding's declared Convert script — non-SARIF engine output fails to parse on backstop pack test/check"
schema_version: issue/v1

issue:
  id: ISSUE-141
  title: "pkg/packval/executor.go's RunEngine never applies a binding's declared Convert script — non-SARIF engine output fails to parse on backstop pack test/check"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# packval executor never applies a binding's declared Convert script

## Problem

`DefaultExecutor.RunEngine` (`pkg/packval/executor.go:62-97`, the command dispatch behind
`backstop pack test` / `backstop pack check` phase3 fixtures) feeds an engine's raw stdout
straight into `check.ParsePackFindings`:

```go
runErr := cmd.Run()
var execErr *exec.Error
if errors.As(runErr, &execErr) {
    return ExecutionResult{Passed: false, ...}, fmt.Errorf("engine %q failed to run: %w", ...)
}
findings, parseErr := check.ParsePackFindings(stdout.Bytes())
```

There is no reference to `binding.Convert` anywhere in `pkg/packval/executor.go`. By contrast, the
REAL gate dispatch path — `runFindingsEngine` in `cmd/backstop/pack_gate.go` (~lines 760-774) —
applies the binding's declared `Convert` script to the payload BEFORE parsing:

```go
sarifBytes := payload
if binding.Convert != "" {
    convertPath := filepath.Join(packRoot, filepath.FromSlash(binding.Convert))
    ...
    converted, convErr := resolveSandboxedRunStdout()(convertPath, nil, packRoot, payload)
    ...
    sarifBytes = converted
}
checkViolations, parseErr := check.ParsePackFindings(sarifBytes)
```

`RunEngine` has no equivalent step. A pack engine whose native output is not SARIF — and that
therefore declares a `convert:` script in its binding specifically to reshape that output into
SARIF — has its RAW, non-SARIF output handed directly to `check.ParsePackFindings`, which fails to
unmarshal it.

## Reproduction

`packs/substantiveness/pack.yml:25` declares:

```yaml
ast-grep-substantiveness:
  command: ast-grep scan --json
  ...
  convert: ast-grep/to-sarif.sh
```

`ast-grep scan --json` (verified installed at v0.43.0 on this machine) does not emit a SARIF
document — its native `--json` output is a plain JSON array of match objects, not an object with a
`runs` key, which is why the pack ships `ast-grep/to-sarif.sh` as its `convert:` script in the
first place (per that script's own header comment: "Real ast-grep stdin->SARIF converter shipped
by the pack (DD-7 / REQ-008, ISSUE-062)").

Tracing `pkg/packval/executor.go`'s `RunEngine` (`executor.go:62-97`) end to end confirms it has
no step equivalent to `pack_gate.go`'s Convert-application block — `stdout.Bytes()` goes straight
into `check.ParsePackFindings` at `executor.go:91`, unconditionally, regardless of whether
`binding.Convert` is set.

## Impact

Any pack whose findings engine emits non-SARIF native output and therefore declares a `convert:`
script (e.g. `packs/substantiveness`'s ast-grep engine) can never pass `backstop pack test` /
`backstop pack check`'s phase3 fixture validation via the `RunEngine` dispatch path — the raw,
unconverted output reaches `check.ParsePackFindings`, which returns a parse error, and the fixture
step reports a hard engine error instead of a genuine pass/fail signal on the pack's rules.

This is a hard PREREQUISITE for `PLAN-ISSUE-092` (fixing ISSUE-092, the `rule_path`/`file`
manifest-model drift that currently makes phase3 dispatch dead code for every real pack) reaching
its own final verification phase: once ISSUE-092's fix restores real dispatch to `RunEngine`, that
plan's evidence depends on a real in-repo pack genuinely passing `pack test`, and
`packs/substantiveness` — the pack this issue's reproduction targets — specifically cannot, due to
this independent Convert-application gap, regardless of ISSUE-092's own fix landing correctly.

## Direction

`RunEngine` (or its caller in the phase3 dispatch chain, `pkg/packval/phase3.go`) needs to apply
the declared `Convert` script the same way `pack_gate.go`'s `runFindingsEngine` does, before
feeding output to `check.ParsePackFindings`. Whoever plans the fix should:

1. Check `.backstop/packs/backstop-ai/backstop-core-architecture/architecture/backstop-core.yml`
   for what `pkg/packval` may depend on before assuming the conversion logic can be shared verbatim
   between `cmd/backstop/pack_gate.go` and `pkg/packval/executor.go` — `cmd/backstop` and
   `pkg/packval` may sit on different sides of an import-direction constraint (PLAN-ISSUE-118 hit a
   comparable constraint between `pkg/gate` and `pkg/pack/engine`).
2. If a shared helper isn't architecturally permitted, decide whether packval needs its own copy of
   the Convert-application step — and if so, ensure it does not drift from the gate-side behavior
   the way ISSUE-140's never-started check drifted (see Notes).
3. Confirm the sandboxed-run mechanism `pack_gate.go` uses for the convert step
   (`resolveSandboxedRunStdout()`) is available/appropriate to reuse from `pkg/packval`, or that an
   equivalent sandboxing guarantee is preserved if reimplemented.

## Notes

- Dependent lane: `ISSUE-092` (`issues/ISSUE-092-pack-test-phase3-fixtures-cannot-fail.issue.md`)
  and its plan `PLAN-ISSUE-092` — this issue is a separate, independent blocker on top of
  ISSUE-092's own fix. ISSUE-092 is about `RunEngine` never being INVOKED (dead dispatch due to the
  `rule_path`/`file` manifest-struct drift); this issue is about `RunEngine`, once invoked, not
  applying `Convert` before parsing. Fixing ISSUE-092 alone does not unblock a Convert-declaring
  pack like `packs/substantiveness` from passing `pack test` — this issue must also be fixed.
- Sibling, not duplicate: `ISSUE-140` (`issues/ISSUE-140-packval-executor-narrow-neverstarted-check.issue.md`)
  is a different defect in the same function (`RunEngine`) — a narrow `*exec.Error`-only
  never-started check that misses path-ful engine commands. That issue documents the same general
  pattern (packval's `RunEngine` drifting from the gate-side `pack_gate.go` dispatch it was meant to
  mirror) on a different mechanism than this issue's missing Convert step.
- Existence-in-world check performed 2026-08-16 before filing: searched `issues/` and `bundles/`
  for `packval`/`executor.go`/`convert` references. No open issue or bundle charter already owns
  this specific Convert-application gap; ISSUE-092 and ISSUE-140 are related but mechanistically
  distinct, as detailed above.
- Verified directly against current code 2026-08-16: `pkg/packval/executor.go:62-97` (no `Convert`
  reference anywhere in the file), `cmd/backstop/pack_gate.go` lines ~760-774 (the real
  Convert-application block `RunEngine` lacks an equivalent of), `packs/substantiveness/pack.yml:25`
  (`convert: ast-grep/to-sarif.sh` declaration) and `packs/substantiveness/ast-grep/to-sarif.sh`
  (its header comment confirming it exists because ast-grep's native `--json` output is not SARIF).
  `ast-grep --version` confirmed installed at 0.43.0 on the verifying machine.
