---
title: "routing manifest schema mismatch — compiled standards disable all check passes"
schema_version: issue/v1

issue:
  id: ISSUE-005
  title: "routing manifest schema mismatch — compiled standards disable all check passes"
  type: bug
  status: obsoleted
  created: "2026-06-11"
  closed: "2026-06-11"

obsoleted-by: ISSUE-018

complexity:
  scope: contained
  uncertainty: known
  risk: critical

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/check/..."

implementation:
  summary: >
    Teach check.LoadManifest to recognize compiled standards manifests
    (the SPEC-001 output schema with standard/language/rules[].enforcement)
    and derive routing from them: files of the standard's language route to
    the native toolchain passes (lint/build/test) plus semgrep when the
    manifest carries semgrep-enforced rules. Keep the existing routing
    format (extensions/path_patterns/check_types) working. Manifests
    present but yielding zero routable rules become a config error rather
    than an empty route table. Add a cross-package contract test that
    round-trips real compiler output through LoadManifest.
  package: pkg/check

requirements:
  - id: REQ-001
    text: >
      LoadManifest must derive file routing from compiled standards
      manifests: a manifest declaring language go routes .go files to
      lint, build, test, and (when any rule has enforcement semgrep)
      semgrep — equivalent to the built-in defaults for that language.
  - id: REQ-002
    text: >
      Manifest files that parse but yield zero routable rules must produce
      a config error (exit 2), never a silently empty route table that
      skips every pass as a green result.
  - id: REQ-003
    text: >
      A contract test must pin the producer-consumer boundary: the
      compiler-emitted manifest schema in .backstop/rules must round-trip
      through LoadManifest and produce non-empty routing for the
      standard's language.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: LoadManifest recognizes a compiled standards manifest and derives language-based routing from it.
    tests:
      - TestCodeCheck_LoadManifest_DerivesRoutingFromCompiledManifest
  - id: CLM-002
    requirement: REQ-001
    text: RouteFile routes .go files to lint/build/test/semgrep under a compiled go-language manifest with semgrep rules.
    tests:
      - TestCodeCheck_LoadManifest_CompiledManifestRoutesGoFiles
  - id: CLM-003
    requirement: REQ-002
    text: manifest files yielding zero routable rules surface a config error instead of routing nothing.
    tests:
      - TestCodeCheck_LoadManifest_NoRoutableRulesIsConfigError
  - id: CLM-004
    requirement: REQ-003
    text: the real compiler-emitted manifest fixture round-trips through LoadManifest into non-empty routing.
    tests:
      - TestCodeCheck_LoadManifest_CompilerOutputContractRoundTrip

contracts:
  - file: pkg/check/manifest.go
    provides:
      - name: LoadManifest
        kind: function
        signature: "func LoadManifest(dir string) (*Manifest, error)"
---

# routing manifest schema mismatch — compiled standards disable all check passes

## Problem

Two incompatible schemas occupy `.backstop/rules/*.manifest.json`:

**What the standards compiler (SPEC-001) emits** — e.g. `STD-GO-001.manifest.json`:
```json
{
  "standard": "STD-GO-001",
  "language": "go",
  "semgrep_config": "STD-GO-001.semgrep.yml",
  "rules": [
    {"id": "GO-001", "name": "max-file-length", "enforcement": "native", ...}
  ]
}
```

**What the check routing layer expects** — `check.LoadManifest`
(`pkg/check/manifest.go:57-114`):
```json
{
  "rules": [
    {"extensions": [".go"], "check_types": ["lint", "build", "test", "semgrep"]}
  ]
}
```

`LoadManifest` parses the compiler's file without error (both have a
`rules` array), finds the rules carry no `extensions`/`path_patterns`/
`check_types`, and — because `allRules` is non-empty — does NOT fall back
to the built-in defaults. `RouteFile` then matches nothing, so
`applicableChecks` routes every file to zero check types.

## Impact

On any repo with compiled standards in `.backstop/rules` (including
backstop-core itself), `backstop code check --all` reports:

```
✓ All checks passed
  [skip] lint: not applicable to files in scope
  [skip] build: not applicable to files in scope
  [skip] test: not applicable to files in scope
  [skip] semgrep: not applicable to files in scope
```

Every pass is skipped — vacuous enforcement, rendered as a green check.
This persists even now that the pass executors are real (ISSUE-002):
the executors are never dispatched because routing matches nothing.
Discovered during ISSUE-002's dogfood reckoning (the first honest
`code check --all` run against this repo).

This is the same disease as ISSUE-001: two packages writing/reading the
same file location with incompatible schemas, undetected because no
contract spans the boundary and the failure mode is silent.

## Root Cause

`pkg/check/manifest.go` and the standards compiler (SPEC-001 outputs)
were specified independently. The routing manifest format
(`extensions`/`path_patterns`/`check_types`) appears to have no producer:
nothing in the repo emits it, so the routing layer has only ever run
against its built-in defaults (no manifests) or been silently disabled
(compiler manifests present).

Compounding factor: `LoadManifest` treats "manifest files exist but
contain zero routing rules" as "route nothing" rather than "fall back to
defaults" or "fail loudly" — a config error rendered as a green skip.

## Fix

Decide the contract, then enforce it:

1. **Single schema**: either teach the compiler to emit a routing section
   `LoadManifest` understands, or teach `LoadManifest` to read the
   compiled-manifest schema (deriving routing from `language` +
   `enforcement` fields). The compiled schema already carries enough
   information (`language: go`, per-rule `enforcement: semgrep|native`)
   to derive correct routing.
2. **Fail loud on schema mismatch**: a manifest whose rules all lack
   routing fields should be a config error (exit 2), not an empty route
   table. Silent skip-everything must be unrepresentable.
3. Add a cross-package contract test: compiler output must round-trip
   through `LoadManifest` and produce non-empty routing for the
   standard's language.

## References

- `pkg/check/manifest.go:57-141` — routing manifest parser and defaults fallback
- `.backstop/rules/STD-GO-001.manifest.json` — compiler-emitted schema in the wild
- SPEC-001 (standards compiler — producer side)
- ISSUE-001 (same cross-package schema-drift disease, packs field)
- ISSUE-002 (real executors — exposed this during dogfood reckoning)
