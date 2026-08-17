---
title: "pkg/packval/executor.go's RunEngine never honors a binding's declared StdoutArtifact — Convert/parse runs on the wrong bytes"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-144

issue:
  id: ISSUE-144
  title: "pkg/packval/executor.go's RunEngine never honors a binding's declared StdoutArtifact — Convert/parse runs on the wrong bytes"
  type: bug
  status: closed
  created: "2026-08-16"
  closed: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# packval executor never honors a binding's declared StdoutArtifact

## Problem

`DefaultExecutor.RunEngine` (`pkg/packval/executor.go:61-98`, the command dispatch behind
`backstop pack test` / `backstop pack check` phase3 fixtures) captures a run's raw stdout into a
buffer and feeds that buffer straight to `check.ParsePackFindings`:

```go
cmd := exec.Command(name, args...)
cmd.Dir = packDir
var stdout bytes.Buffer
cmd.Stdout = &stdout
...
runErr := cmd.Run()
...
findings, parseErr := check.ParsePackFindings(stdout.Bytes())
```

There is no reference to `binding.StdoutArtifact` anywhere in `pkg/packval/executor.go`. By
contrast, the REAL gate dispatch path — `runFindingsEngine` in `cmd/backstop/pack_gate.go`
(lines 759-766) — resolves the payload BEFORE any Convert/parse step runs:

```go
payload := stdout
if binding.StdoutArtifact != "" {
    artifactPath := filepath.Join(projectRoot, filepath.FromSlash(binding.StdoutArtifact))
    body, readErr := os.ReadFile(artifactPath)
    if readErr != nil {
        return nil, fmt.Errorf("pack %s findings engine %q: declared stdout_artifact %q not produced (%s): %w", ...)
    }
    payload = body
}
```

Per SPEC-048 (REQ-002/CLM-005..008, DEFECT-2), a `stdout_artifact`-declaring binding writes its
real machine-readable output to that FILE, printing only a human-summary to stdout — e.g. `bun
test --reporter=junit --reporter-outfile=<file>` writes JUnit XML to the file while stdout is a
one-line pass/fail summary with no `<testcase>` elements. `runFindingsEngine` reads the declared
file and uses THAT as the payload; a declared-but-missing artifact is a fail-loud broken run, not
a silent fallback to stdout noise. `RunEngine` has no equivalent step: it always parses whatever
landed in `stdout.Bytes()`, regardless of whether `binding.StdoutArtifact` is set.

## Distinction from ISSUE-141

`ISSUE-141` (`issues/ISSUE-141-packval-executor-missing-convert-application.issue.md`, its plan
`PLAN-ISSUE-141` in flight the night this issue was filed) covers `RunEngine` never applying a
binding's declared `Convert` script — a missing PIPELINE STAGE. This issue is mechanically
distinct: even once Convert is applied, `RunEngine` has no step that chooses the CORRECT INPUT
BYTES to apply it to. In the real gate path the sequence is `payload := stdout-or-artifact-file`,
then `sarifBytes := Convert(payload)`. `RunEngine` today skips both stages; after ISSUE-141's fix
lands it will still skip the payload-selection stage — Convert would run on raw `stdout.Bytes()`
unconditionally, which is the wrong input for any binding that declares `StdoutArtifact`. This
issue is filed separately, as directed, so the fix for "Convert never runs" does not absorb scope
for "Convert may run on the wrong bytes" — a different failure mode requiring its own reproduction
and its own fix, on a lane (`PLAN-ISSUE-141`) that is a hard blocking prerequisite for the actively
implementing `PLAN-ISSUE-092`.

## Impact

Any pack whose findings engine declares `stdout_artifact` (writing real output to a file while
stdout carries only a human summary) cannot pass `backstop pack test` / `backstop pack check`'s
phase3 fixture validation via the `RunEngine` dispatch path: the fixture run parses stdout noise
instead of the declared artifact file, so a positive fixture that should produce findings reads as
clean (Passed=false) — a vacuous pass — or, if the noise happens to fail SARIF parsing outright, a
misleading parse-error instead of a genuine pass/fail signal on the pack's rules. This is the same
shape of silent-vacuous-green gap SPEC-048 fixed on the real gate path; `pkg/packval`'s dispatch
never received the equivalent fix.

## Direction

`RunEngine` needs a payload-selection step equivalent to `pack_gate.go`'s
`payload := stdout; if binding.StdoutArtifact != "" { ...read file... }` block, applied BEFORE
whatever Convert-application fix ISSUE-141 lands (Convert consumes the selected payload, not raw
stdout). Whoever plans the fix should:

1. Verify ISSUE-141's fix has actually landed first (check `PLAN-ISSUE-141`'s status and read
   `pkg/packval/executor.go` directly) — this issue's fix inserts a stage that precedes Convert in
   the real path's ordering, so sequencing relative to ISSUE-141 matters for where the new code
   goes, not just whether it exists.
2. Confirm the working directory this issue's read should be relative to. `pack_gate.go` resolves
   `StdoutArtifact` relative to `projectRoot` (the gate's real project root); `RunEngine` runs with
   `cmd.Dir = packDir` (the pack directory), a different root in packval's context — the fix must
   pick the semantically correct base for the phase3/fixture-testing context, not copy
   `projectRoot` verbatim.
3. Check `ISSUE-143` (`issues/ISSUE-143-packval-gate-convert-dual-implementation.issue.md`, filed
   the same night) before landing a second hand-maintained copy of gate-path logic in packval —
   its Direction section proposes extracting a shared step; if that extraction is already underway
   by the time this issue is picked up, the StdoutArtifact payload-selection logic belongs in the
   same shared location as the Convert step, not a third independent copy.
4. Add a reproduction fixture: a binding with `stdout_artifact` set, a process that writes real
   SARIF (or convert-target) content to that file while printing unrelated/empty content to
   stdout, and an assertion that `RunEngine` parses the FILE's content, not stdout's.

## Notes

- Sibling, not duplicate: `ISSUE-141` is "Convert never runs at all" (missing stage); this issue is
  "the stage that selects Convert's input never runs" (correct stage fed incorrect input, once
  Convert exists). Filed separately per explicit instruction to avoid scope creep on
  `PLAN-ISSUE-141`, which is a hard blocking prerequisite for `PLAN-ISSUE-092`.
- Related, not duplicate: `ISSUE-143` covers the STRUCTURAL duplication risk of two independently
  maintained Convert-application implementations (`pack_gate.go` vs `packval/executor.go`) once
  ISSUE-141 lands. This issue is about a missing BEHAVIOR (StdoutArtifact selection), not about
  implementation architecture — but the two are adjacent: if ISSUE-143's extraction lands before
  this issue is picked up, this issue's fix should target the same shared location.
- Same drift family as `ISSUE-092` (manifest-model drift), `ISSUE-140` (narrow never-started
  check), and `ISSUE-141` (missing Convert stage) — `pkg/packval`'s dispatch repeatedly drifting
  from the real `cmd/backstop/pack_gate.go` dispatch it is meant to mirror. May fit `DIR-032`
  ("Gate Verdict Honesty" — vacuous-green defects in gate-step verdict reporting) alongside those
  siblings; left unslotted here for backlog-pm/directive-author triage rather than hand-edited,
  per this repo's artifact-authoring convention.
- Existence-in-world check performed 2026-08-16 before filing: searched `issues/` and `bundles/`
  for `StdoutArtifact`/`RunEngine`/`packval` references. No open issue or bundle charter already
  owns this specific StdoutArtifact-selection gap. `ISSUE-141` and the newly-filed `ISSUE-143`
  (concurrent with this filing) are related but mechanistically distinct, as detailed above.
- Verified directly against current code 2026-08-16: `pkg/packval/executor.go` (grep for
  `StdoutArtifact` returns zero matches in the entire file) and `cmd/backstop/pack_gate.go:759-766`
  (the real payload-selection block `RunEngine` lacks an equivalent of). `PLAN-ISSUE-141` confirmed
  `status: draft` (not yet landed) at filing time.

## Resolution

`pkg/packval/executor.go`'s `RunEngine` now honors a binding's declared `StdoutArtifact`. Before
this fix, `RunEngine` always fed `stdout.Bytes()` straight into Convert/SARIF parsing regardless of
what the binding declared — a binding that writes its real machine-readable output to a file (e.g.
go-toolchain's `go-coverage` engine, `stdout_artifact: cover.out`) had that declaration silently
ignored, so the fixture run parsed stdout noise instead of the artifact file. The fix mirrors the
real gate path's payload-selection block (`cmd/backstop/pack_gate.go:759-766`): `RunEngine` now
resolves `binding.StdoutArtifact` relative to `packDir` (`pkg/packval/executor.go:127-132`) and
reads that file as the payload when the field is set, failing loudly with the declared path if the
artifact was not produced, rather than silently falling back to stdout.

Falsified pre-fix: the lying-verdict claim reproduced literally — `Passed:false`, nil error, exit
0, empty `Output`, the success condition for a negative phase3 fixture on a run whose real output
was never read.

Delivered by `PLAN-ISSUE-144` (`status: completed`, committed at `4d29e36`).

**Residual, not fixed here:** `RunEngine` also ignores three further binding fields —
`CrashGuard`, `StrictSarif`, `Producer` — all honored by `cmd/backstop`'s dispatch path. This
belongs beside `ISSUE-143`'s single-authority consolidation family but must not be folded into it
silently, since each field needs its own falsification mechanics. Left as an open gap for a
follow-on issue rather than addressed here.
