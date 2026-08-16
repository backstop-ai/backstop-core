---
name: traceability-step-cannot-report-offcorpus
description: A gate requirement_traceability assertion about a file planted OUTSIDE the artifact kind directories is structurally vacuous — refs whose citing path is not in ResolveArtifactStatus's records are silently dropped
metadata:
  type: project
---

`requirement_traceability` CANNOT emit a violation naming a file that lives outside the
resolved artifact root's KIND directories. Any plan that proposes "assert the
requirement_traceability step reports no planted file" as a falsifier for a
`collectTraceRefs` / `DiscoverArtifacts` wiring change is proposing a test that cannot fail.

Evidence chain (verified 2026-08-16):
- `computeRequirementTraceabilitySurfaces` (cmd/backstop/gate.go:1049) →
  `gate.ClassifyRequirementTraceability(res.Records, refs)`.
- `res.Records` ← `gate.ResolveArtifactStatus(root.Path)` (pkg/gate/artifact_status.go:180),
  which reads ONLY `root.Dir(kind)` via `walkArtifactDir` (:412) — `os.ReadDir` +
  `if entry.IsDir() { continue }`, so it is NON-RECURSIVE and never sees `vendor/`,
  `node_modules/`, or even `specs/vendor/`.
- `ClassifyRequirementTraceability` (pkg/gate/requirement_traceability.go:81-86):
  `citer, hasCiter := citers[...ref.CitingPath]; if !hasBundle || !hasCiter { continue }`.
  Every violation whose `File` is a citer path requires `hasCiter`.
=> A planted spec/issue outside the kind dirs contributes refs that are dropped. The step's
violation set is IDENTICAL whether the walk excluded it or discovered it.

**Why:** the traceability classifier joins refs against the STATUS walk, and the status walk
is deliberately non-recursive (SPEC-068 CLM-063 pins the discovery-vs-status asymmetry).

**How to apply:** when a plan threads an exclusion/scoping parameter through
`collectTraceRefs`/`DiscoverArtifacts` and claims a gate-step behavior test falsifies it,
check the DOWNSTREAM consumer first — the walk may pick the file up and the classifier may
still discard it. Contrast with the artifact_validation step, which IS falsifiable because
`realArtifactValidator.ValidateAll` (gate.go:1705) appends
`gate.UngatedFindingsToViolations(FindUngatedArtifacts(...))`, and those violations DO name
off-corpus paths. Also contrast `artifact validate --all --json`, whose `artifacts[]`
envelope (cmd/backstop/output.go:118-143) lists EVERY discovered artifact by path — a
validity-independent, non-vacuous signal for the CLI discovery hop.

Related: [[injected-param-callsite-unfalsified]], [[verified-enumeration-do-not-rederive]].
