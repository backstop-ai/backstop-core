---
name: spec041-coverage-consumer-pass
description: SPEC-041 (BUNDLE-011 Seed 3) coverage consumer + exempt bridge + catalog PASSED — completes BUNDLE-011; substantive ~30-test rewrite, correct off-plan capability re-key, safe atomic transitional-seam handoff
metadata:
  type: project
---

SPEC-041 (BUNDLE-011 Seed 3, branch bundle/011-codecheck-cutover @ 0c694fe, 27 claims) PASSED — the FINAL piece of BUNDLE-011. Coverage CONSUMER (per-file, metric-blind) + permanent exempt_from_scope_filter bridge + CheckType-consumer catalog. Eradicates the baked Go coverage analyzer (step_coverage.go) + shared_testrun.go.

The THREE honestly-flagged off-plan moves — all scrutinized, all CORRECT:
1. Plan named transitional seam `dispatchBuildViolationProjectWide` which never existed; the real one was `projectWideBuild` (`binding.GateType==GateTypeBuild`, from SPEC-040). Implementer removed the REAL one. Verified: the pack_gate.go diff shows `projectWideBuild := binding.GateType==engine.GateTypeBuild` REPLACED by `exempt := binding.ExemptFromScopeFilter` IN THE SAME HUNK — atomic handoff, no silent-scope-filter window. go-build declares ExemptFromScopeFilter:true.
2. Off-plan re-key of deriveCapabilityState's COVERAGE arm (baked→installed-pack via coverageToolchainPackInstalled, reads cfg.Packs for a `-toolchain` pack). CORRECT + necessary: symmetric with substantiveness(SPEC-037)/contracts(SPEC-038); capability-absent⇒class-2 WARN(exit 0) for undeclared, class-3 block for declared-unmet via SPEC-036 classifier — never silent pass, never wrong block. The old baked arm would have loud-BLOCKED an uninstalled Go project. Gate now shows `coverage_threshold ⚠ coverage_capability_absent` WARN (was the old aggregate red).
3. Test-deletion wider than the 6 named: step_coverage_test.go 33→13 funcs + 5 SPEC-037/038 capability tests aligned. SUBSTANTIVE not gutted — the 20 removed were coupled to the deleted go-test-exec/regex/package-rollup machinery; the 13 new map 1:1 to claims and drive the REAL StepCoverageThresholdScopedFunc.

Biggest-risk verdict (the ~30 rewrite + 5 alignments) — NOT GUTTED:
- New coverage tests assert real per-file behavior: CLM-010 (the load-bearing one) scopes BOTH a 95% + a 2% sibling, asserts the 2% file is flagged and the 95% sibling is NOT — proving no package-aggregate rescue. CLM-007 2%-at-90 reds; CLM-008 unmeasured-non-excluded→loud error, declared-excluded→quiet; CLM-025 changed-file exclusion surfaced as warning(path+reason), unchanged stays quiet; CLM-026 raw-count ratio (7/10 fail, 8/10 boundary pass) + Total==0 N/A; CLM-027 differing-metric files each surface own metric.
- 5 capability tests are genuine assertion INVERSIONS (was `coverage STAYS baked-present` → now `Present==false` no-pack / `Present==true` with -toolchain pack). They'd FAIL if the re-key were wrong. Teeth intact.

Exempt bridge END-TO-END (item 4, critical): CLM-013 TestExemption_BuildBreakUnchangedFileStillRedsDiffScoped runs REAL dispatchPackEngines→violations on NON-scope files→filterThroughGate(real gate.StepCodeCheckScopedFunc→filterViolations)→asserts SURVIVE. CLM-014/015/016 confirm lint/test/findings filtered (exempt=false). CLM-019 true-conflict: both engines same file+rule, exempting copy survives real filter. Per-violation, no aggregation.

Catalog (item 5): 6 rows (C-1 declared-engine-property + C-4..C-8 surviving); C-2(checkViolationsToGate)/C-3(sharedTestRunner) DELETED rows correctly ABSENT. Guard bidirectional + non-tautological: reconciles real surface clean (no red on arrival), fails on injected unlisted keying-site fixture AND stale-entry fixture; discoverGateSemanticSites scans source line-by-line, excludes bare .Pass.String() display.

Eradication complete: all baked coverage symbols gone (coverageRe etc — earlier grep hit was coverageRecords* substring), shared_testrun.go deleted, absence-guards scan source for banned literals. No go-test exec/regex/go.mod in step_coverage.go (only comments).

Item 7: diff-scoped gate pack_engines PASS over 50 changed files (0 net-new standards violations); the 158/--all are pre-existing whole-repo debt (old SPEC-001/019 artifacts etc).

Full `go test ./...` green; `./bin/backstop gate` PASS exit 0 (4 passed/0 failed/3 skipped/4 warned). Coverage 80 threshold met. BUNDLE-011 implementation COMPLETE across all 4 seeds. See [[project_spec040_keystone_cutover_pass]] [[project_spec042_coverage_producer_pass]] [[project_spec039_deadcode_prelude_pass]].
