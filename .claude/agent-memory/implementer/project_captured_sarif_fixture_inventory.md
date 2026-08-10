---
name: captured-sarif-fixture-inventory
description: Before hand-building or freshly capturing a SARIF shape, inventory committed captures — cmd/backstop/testdata/go-toolchain/fixtures/golangci-v2.sarif is REAL golangci output with result-level severities (error+warning, no driver.rules); semgrep emits descriptor-only severity
metadata:
  type: project
---

Found during PLAN-ISSUE-104 review (2026-07-29): the plan was about to
hand-build (or re-capture) a result-level-severity SARIF fixture while a
REAL one sat committed at cmd/backstop/testdata/go-toolchain/fixtures/
golangci-v2.sarif — two results carrying `level` (error and warning), no
tool.driver.rules — already read cross-package by readGoToolchainFixture
(pkg/check/parse_pack_findings_test.go:15-19).

The emission-shape map, measured:
- semgrep (1.156.0): NO results[].level; severity lives ONLY on
  tool.driver.rules[].defaultConfiguration.level, joined on ruleId.
- golangci-lint (v2 sarif): results[].level present per finding.
- The CONFLICT shape (result-level + contradicting descriptor) is
  producible by no tool on hand — the one legitimate hand-built fixture,
  labeled as such.

SEMGREP CAPTURES NOW EXIST - do not re-capture (added 2026-07-28 by
PLAN-ISSUE-104): cmd/backstop/testdata/semgrep/fixtures/ holds
descriptor-warning.sarif and descriptor-error.sarif (1.156.0) plus
descriptor-warning-1.96.0.sarif (the PINNED version backstop provisions,
shape-identical: no result-level key, severity on the descriptor). Ships
capture inputs, a re-runnable capture.sh that reproduces the 1.156.0 bytes
EXACTLY (verified by diff), and PROVENANCE.md with sha256 per file.
pkg/check reads them via readSemgrepFixture.

Pinned-version capture gotcha: `uv pip install semgrep==1.96.0 setuptools`
resolves setuptools 83.0.0, which no longer ships pkg_resources, and semgrep
dies on ModuleNotFoundError. Pin setuptools==70.3.0. The PATH scrub is
separately load-bearing - with the ambient PATH the 1.96.0 CLI delegates to a
newer semgrep-core and emits semanticVersion 1.156.0 under a 1.96.0 filename,
so ALWAYS assert the version inside the captured file.

Capture discipline for semgrep fixtures: run from the fixtures dir with a
RELATIVE --config (absolute config paths get mangled into the rule id,
baking /Users/... into committed bytes and breaking provenance sha256
portability), and assert the capture holds EXACTLY the result count the
consuming harness requires. Re-verify against the tool versions in play.
