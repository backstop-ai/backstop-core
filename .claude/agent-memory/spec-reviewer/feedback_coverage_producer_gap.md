---
name: coverage-producer-gap
description: SPEC-041's per-file CoverageRecord consumer contract has no producer — the go-toolchain pack's go-test emits only failure findings, no coverage
metadata:
  type: project
---

SPEC-041 (BUNDLE-011 Seed 3) re-implements coverage language-agnostic over a per-file `CoverageRecord{Path, Pct, Measured, Excluded}` that the SPEC-040 toolchain test pass is supposed to PRODUCE. Verified on main: that producer does NOT exist yet.

**Why:** The `go-toolchain` pack (cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml) `go-test` engine runs `go test ./...` and its convert script `scripts/test-to-sarif.sh` normalizes ONLY `--- FAIL:` blocks into SARIF findings — NO `-coverprofile`, NO per-file coverage emission, no coverage at all. SPEC-040 keeps the OLD baked `go test ./... -coverprofile=/dev/null` shared-runner as the explicitly-TRANSITIONAL coverage feed (SPEC-040 CLM-028, whole-module granularity), which SPEC-041 then eradicates. So the per-file consumer contract is unsatisfiable until a producer emits per-file coverage.

**How to apply:** SPEC-041 itself FLAGS this correctly (REQ-001: "the producer MUST emit coverage at this per-file/path granularity"; body: "flagged for producer↔consumer coherence review, not authored into SPEC-040 here") — so it is NOT a SPEC-041 authoring defect, it's a real SPEC-040-side corrective: SPEC-040's go-toolchain test pass must gain a coverage emission (e.g. `go test -coverprofile` → a coverage-to-records convert) at per-file granularity, declared as a coverage engine/GateTypeCoverage output. Until that lands, SPEC-041 is implementable but its coverage step has nothing to consume. Flag to the SPEC-040 author / planner as a sequencing dependency, not a SPEC-041 blocker. Related: [[projectwide-locus-seam]], [[catalog-deleted-mislabel]] (same SPEC-041).
