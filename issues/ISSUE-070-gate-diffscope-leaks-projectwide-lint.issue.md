---
title: "Gate Diffscope Leaks Projectwide Lint"
schema_version: issue/v1

issue:
  id: ISSUE-070
  title: "Gate Diffscope Leaks Projectwide Lint"
  type: bug
  status: open
  created: "2026-07-18"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Gate Diffscope Leaks Projectwide Lint

## Problem

In default diff mode (`backstop gate`, no `--all`), the `golangci` lint engine's violations
(errcheck / unused / staticcheck / error-wrapping / no-global-mutable-state) are reported on files
that are NOT in the change unit — redding the gate on pre-existing debt in files the change never
touched. This violates the file-level diff-scope contract.

### Why it's a bug, not intended behavior

The `golangci` engine in the go-toolchain pack is declared `scope_kind: project-wide` (it MUST run
over `./...` — Go whole-package analysis can't be file-scoped at the tool level) but it is NOT
`exempt_from_scope_filter` (only `go-build` sets that). Per the design, only exempt engines'
violations skip the diff-scope filter (a build break anywhere must red); a non-exempt engine's
project-wide violations are supposed to be FILTERED DOWN to the changed files after the
project-wide run. So golangci's violations should be filtered to the change unit — but they are
not.

### Evidence

Measured on ISSUE-068's change: the change unit was 19 files (`git diff` + untracked). `backstop
gate` (default diff, header: "26 changed files … use --all for full sweep") reported 113
`pack_engines` violations spread across 30 files. Of those 30, exactly 2
(`cmd/backstop/pack_gate.go`, `pkg/pack/manifest.go` — the files the change touched) are in scope;
the other 28 are entirely outside the diff (e.g. `cmd/backstop/coverage_cli_test.go`,
`cmd/backstop/baseline.go`, all of `pkg/pack/distribution/*`, `pkg/scaffold/*`, `pkg/packval/*`).
So 28/30 violation-bearing files are unchanged — the leak.

### Contrast (localizes the bug)

The `artifact_status_drift` dimension IS correctly diff-scoped — it carries 7 pre-existing
broken-promise violations (closed ISSUE-018 / ISSUE-036 with absent mandated tests) yet PASSES,
because those files are out of scope. So the diff-scope filter works for some dimensions; the
failure is specific to the `pack_engines` / golangci violation path.

## Leading hypothesis (confirm at plan/impl time — not asserted)

File-path normalization mismatch. golangci-lint v2 emits SARIF whose result file paths are in a
form (absolute, `file://`, or differently-rooted) that `pkg/gate/scope.go`'s `scope.Contains(file)`
does not match against the changed-file set (which is repo-relative). Because the path isn't
recognized, the violation isn't identified as out-of-scope and passes through instead of being
filtered. Same `NormalizePath` concern the baseline fingerprint work hit.

Alternative to rule out: whether these violations are wrongly stamped `ProjectWide=true` (they
shouldn't be — golangci is not exempt).

## Where to look

- `cmd/backstop/pack_gate.go` — violation stamping, `ProjectWide` set from
  `binding.ExemptFromScopeFilter` (~lines 792-830)
- `pkg/gate/scope.go` — `scope.Contains` / the diff-scope filter (~line 196)
- Path normalization between golangci SARIF output and the scope's changed-file list

## Impact

Foundational. The gate reds on ANY Go change regardless of whether the change itself is clean
(ISSUE-068 adds ZERO new violations yet the gate reds on 28 unchanged files). It makes diff-scope
untrustworthy — the whole gate-hardening / ratchet story depends on the gate accurately scoping to
changed files. It also forces false choices (waive/grandfather) on debt the author never touched.

Distinct from DIR-003 baseline: baseline grandfathers UNCHANGED files for dims that DO scope; this
is the diff-scope FILTER itself failing for the golangci path.

## Acceptance

- Under default diff, golangci lint violations on files NOT in the change unit are FILTERED OUT
  (not reported, do not red).
- Violations in CHANGED files are still reported in full (whole changed file in scope —
  you-touch-it-you-fix-it stays intact).
- `go-build` (exempt) still reds project-wide on an unchanged-file build break (no regression to
  the exempt path).
- A regression test proves: an unchanged file carrying a golangci violation does NOT red a
  diff-scoped gate when a DIFFERENT file changed; and a changed file's violations DO red.
- Other dimensions (status_drift, coverage, contracts) unaffected.

## Notes / references

- Surfaced during ISSUE-068's investigation (2026-07-18): `backstop gate` on that change's 19-file
  diff.
- This issue will receive a PLAN — mandated test names are deferred to that plan, not declared
  here.
