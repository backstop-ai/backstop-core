---
name: coverage-producer-stream-bug
description: SPEC-042 coverage producer feeds the convert go-test's SUMMARY stdout, not the cover.out profile, so changed .go files get no coverage record and the gate falsely reds
metadata:
  type: project
---

The SPEC-042 coverage producer path (`cmd/backstop/pack_gate.go` →
`runCoverageEngine`) pipes the WRONG stream into the pack's coverage convert.

**The bug:** the go-toolchain `go-coverage` engine command is
`go test -coverprofile=cover.out ./...` (pack.yml). That writes the coverage
PROFILE to the FILE `cover.out` (in the runner's Dir = projectRoot) and writes
only the test SUMMARY (`ok pkg (cached) coverage: 96.2%`) to STDOUT.
`runCoverageEngine` line ~412 pipes that command's **stdout** (the summary) into
`scripts/coverage-to-records.sh` as stdin. The convert parses Go profile lines
(`path:startLine.col,endLine.col numStmt count`), so summary lines yield `[]` —
near-zero records. Proven: convert fed stdout -> `[]`; convert fed the `cover.out`
file -> full per-file records.

**Why it stays hidden:** in diff scope, if no changed `.go` file is in coverage
scope, `coveragePathsInScope` is empty -> "no in-scope files to measure" PASS.
The instant ANY changed `.go` source file lands in scope, it has no record ->
`step_coverage.go:72` `!hasRecord` fires a LOUD blocking
`coverage_threshold: no coverage measurement for in-scope changed file <f> ...
refusing to pass with nothing measured`. So merely editing one .go file reds the
gate. Violation counts also fluctuate run-to-run (go-test `(cached)`).

**Not a real coverage shortfall** -- it's a MISSING RECORD. The files are
genuinely well-covered (e.g. pkg/validate 96.2%); the producer just never
surfaces their records.

**Minimal correct fix (binary, out of a leaf pack's reach):** add an optional
declared `stdout_artifact` (e.g. `cover.out`) to `engine.EngineBinding`; when set,
the producer reads that file's contents (relative to runner Dir/projectRoot) and
feeds IT to the convert instead of command stdout. Keeps the binary
language/tool-blind (filename is pack DATA), convert contract unchanged. A pure
pack-data fix does NOT exist: `-coverprofile=/dev/stdout` over `./...` is
unreliable (per-package serialization clobbers/interleaves), and the convert
(cwd=packDir) cannot locate projectRoot/cover.out on its own.

**Provenance:** introduced by `2751832 wip(BUNDLE-011 Seed 4): implement SPEC-042
coverage producer`. Independent of ISSUE-031 (which touched none of the producer
path). Surfaced 2026-06-27 by an in-scope edit to pkg/validate/terminal.go.

**RESOLVED 2026-06-27.** Two bugs, both fixed:
1. Wrong stream -> added `EngineBinding.StdoutArtifact` declared field
   (`pkg/pack/engine/binding.go` + `manifest.go` parse); `runCoverageEngine`
   reads that file and feeds it to the convert (declared-but-missing = fail-loud).
   go-toolchain pack.yml declares `stdout_artifact: cover.out` (INSTALLED-CACHE
   only — `.backstop/` is gitignored; must be carried into the go-toolchain pack's
   OWN repo to be durable).
2. ALSO found a second masked bug: the Go profile yields MODULE-QUALIFIED paths
   (`github.com/org/repo/pkg/x/f.go`) while the gate scope is repo-relative, so
   produced records never matched -> `resolveCoverageRecord` (pkg/gate/
   step_coverage.go) reconciles by exact-then-unique-suffix match (language-neutral;
   ambiguous suffix => no-match; below-threshold still fails).
Proven by RED->GREEN producer unit test + consumer test + live gate (terminal.go
went from "no measurement" to measured 26/29). step_coverage.go's 77.9%->79.9%
is INHERITED debt (pre-existing on HEAD), explicitly NOT chased (commit-to-green
per diff-scope design, no origin/main).
