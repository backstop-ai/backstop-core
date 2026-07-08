---
title: "lint and semgrep executors fail on real tool output — invocation robustness"
schema_version: issue/v1

issue:
  id: ISSUE-006
  title: "lint and semgrep executors fail on real tool output — invocation robustness"
  type: bug
  status: obsoleted
  created: "2026-06-11"
  closed: "2026-06-12"

obsoleted-by: ISSUE-018

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/check/..."

implementation:
  summary: >
    Harden the semgrep and lint executors against real tool behavior:
    semgrep gets --quiet plus robust JSON-document extraction from
    combined output (real semgrep emits non-JSON preamble bytes);
    golangci-lint gets version-aware output-flag selection (v1
    --out-format json vs v2 output flags) and failure exits (>=2) surface
    an actionable error including a tool-output excerpt instead of a bare
    exit status.
  package: pkg/check

requirements:
  - id: REQ-001
    text: >
      The semgrep executor must parse findings from real semgrep output
      that includes non-JSON preamble or progress bytes, invoking semgrep
      with --quiet and extracting the JSON document from combined output.
  - id: REQ-002
    text: >
      The lint executor must distinguish golangci-lint findings exits
      (1) from failure exits (>=2): failure exits surface an error that
      includes a tool-output excerpt naming the cause.
  - id: REQ-003
    text: >
      The lint executor must select output flags compatible with the
      installed golangci-lint major version, probed via golangci-lint
      version, so a v2 installation does not fail with unknown-flag
      errors.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: semgrep findings parse from output carrying non-JSON preamble bytes around the JSON document.
    tests:
      - TestCodeCheck_SemgrepExecutor_ToleratesNonJSONPreamble
  - id: CLM-002
    requirement: REQ-001
    text: the semgrep invocation includes --quiet.
    tests:
      - TestCodeCheck_SemgrepExecutor_QuietFlagPassed
  - id: CLM-003
    requirement: REQ-002
    text: a golangci-lint failure exit with unparseable output surfaces an error containing a tool-output excerpt.
    tests:
      - TestCodeCheck_LintExecutor_FailureExitIncludesDiagnostics
  - id: CLM-004
    requirement: REQ-003
    text: the lint executor selects v1 or v2 output flags based on the probed golangci-lint major version.
    tests:
      - TestCodeCheck_LintExecutor_VersionAwareOutputFlag

contracts:
  - file: pkg/check/check.go
    consumes:
      - source: pkg/check/semgrep.go
        name: EnsureSemgrep
        kind: function
---

# lint and semgrep executors fail on real tool output — invocation robustness

## Problem

The first live `backstop code check --all` run (post ISSUE-002/ISSUE-005)
surfaced two executor robustness failures against real tools:

1. **semgrep**: `semgrep error: parsing semgrep output: invalid character
   'â' looking for beginning of value` — the executor feeds semgrep's
   combined output straight to the JSON parser, but real semgrep emits
   non-JSON preamble/progress/banner bytes (UTF-8 punctuation) around the
   JSON document. The parse fails and the entire pass errors.
2. **golangci-lint**: `lint error: golangci-lint: exit status 3` — exit 3
   is golangci-lint's failure exit (config problem, version mismatch with
   the flags we pass, or no config). The executor surfaces the raw exit
   with no diagnostic context (no stderr excerpt, no version probe), so
   the user can't tell tool misconfiguration from code findings.

Both failures are loud (good — they fail the gate rather than skipping
silently), but neither is actionable from the message alone, and the
semgrep one will fire on every healthy semgrep installation.

## Impact

On any repo with routing active, the semgrep pass errors instead of
reporting findings — pack layer-2 rules are effectively still unenforced
at gate time. The lint pass errors with an opaque message wherever
golangci-lint's config or version doesn't match the invocation.

## Fix

1. semgrep: invoke with `--quiet` (suppresses non-JSON output) and/or
   locate the JSON document in the combined output before unmarshaling;
   keep stderr separate from stdout rather than using CombinedOutput for
   a JSON-emitting tool. Add a fixture built from real semgrep output
   including preamble bytes.
2. golangci-lint: distinguish findings exits from failure exits (golangci
   uses 1 = issues found, ≥2 = failure), include a stderr excerpt in the
   violation message on failure exits, and consider probing
   `golangci-lint version` in IsAvailable for flag compatibility (the
   `--out-format` flag this executor passes was renamed in golangci-lint
   v2 — version mismatch is the likely cause of the observed exit 3).

## References

- pkg/check/check.go — lintExecutor / semgrepExecutor (ISSUE-002 implementations)
- ISSUE-002 (real executors), ISSUE-005 (routing fix that made the passes fire)
- Discovered during ISSUE-005 TASK-012 dogfood reckoning, 2026-06-11
