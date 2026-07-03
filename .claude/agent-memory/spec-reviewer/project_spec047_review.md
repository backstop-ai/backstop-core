---
name: spec047-review
description: SPEC-047 (BUNDLE-012 Seed 5, bun pack + 2-surface proof + ratchet→block) single-spec review — PASS w/ prose should-fixes; REQ-007 per-pack policy verified vs live code
metadata:
  type: project
---

SPEC-047 single-spec review (post-reconciliation) = **PASS** (validator PASS; 2 prose should-fixes, no blockers).

Owns bundle REQ-006 (5-engine bun-toolchain pack: oxlint/prettier-as-lint/tsc/bun test/bun coverage→lcov), REQ-009 (in-repo stubbed fixture + external guarded fork acceptance), REQ-008 (ratchet→block flip). All in-scope bundle REQs covered; Out-of-Scope respected.

**REQ-007 (the unreviewed reconciliation addition — per-pack/per-rule-source enforcement key) is SOUND, verified vs live code 2026-06-29:**
- `gate.Violation.SourcePack` EXISTS (pkg/gate/result.go:41) — REQ-007's foundational claim is true.
- `ApplyPolicy` live signature matches the spec contract EXACTLY (steps, baseline, policy map[string]DimensionPolicy, scope) — "signature unchanged" holds.
- Both `DimensionPolicy` types exist (pkg/config/config.go:49 + pkg/gate/policy.go:18), base shape {Level,Baseline} matches contract.
- `pack_engines` is a real StepName (cmd/backstop/gate.go); dogfood backstop.yml currently `pack_engines:{baseline:true,level:block}`; .backstop/baseline.json grandfathers ~70 self entries — EXACTLY the shared-dimension situation REQ-007 fixes.
- `no-language-literal-on-neutral-spine` (self pack.yml) is a plain semgrep findings rule, NOT routed out by RouteSubstantivenessFindings → stays in flat pack_engines stream → lands in pack_engines step. Premise holds end-to-end.
- Prior art for per-SourcePack partitioning within pack_engines: RouteSubstantivenessFindings (substantiveness_join.go). Feasibility confirmed.
- Claims complete: CLM-036 parse+backward-compat, 037 self-fresh BLOCKS, 038/039 go-standards/go-toolchain stay grandfathered (isolation), 040 denylist (no whole-dim zero-baseline). Tests substantive/fail-on-revert.
- REQ-007 is a JUSTIFIED necessary mechanism for bundle REQ-008 (closes cross-pass blocker #3 from [[bundle012-crosspass]] — "scope a finer policy key as a real requirement"), traces via supports→REQ-008. NOT unrequested scope.

**Should-fix (prose drift from the unreviewed reconciliation, authoritative frontmatter is correct):**
1. Prose Verification "Command:" (line ~676) omits `./pkg/config/` but frontmatter test_command (line 61) includes it — REQ-007 added pkg/config + CLM-036 config-parse test. Implementer following prose would skip the config test.
2. Overview line ~509 says "REQ-001 … REQ-006" but spec has 7 REQs (REQ-007 exists; the summary table below it was updated, the sentence wasn't).

**Note/risk:** external fork acceptance (CLM-027/029) always SKIPS in backstop-core Go CI; in-repo CI proof is the stubbed-runner fixture only. Bundle-sanctioned (OQ-4) but "REQUIRED" acceptance needs a real run-evidence mechanism on the fork or it's vacuously skipped forever.
