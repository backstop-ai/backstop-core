---
title: "// nosemgrep suppressions are silently ignored in the engine/SARIF path — ParsePackFindings counts suppressed findings as violations"
schema_version: issue/v1

issue:
  id: ISSUE-017
  title: "// nosemgrep suppressions are silently ignored in the engine/SARIF path — ParsePackFindings counts suppressed findings as violations"
  type: bug
  status: closed
  created: "2026-06-20"
  closed: "2026-07-08"

resolved-by: 276652b

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# // nosemgrep suppressions are silently ignored in the engine/SARIF path — ParsePackFindings counts suppressed findings as violations

## Problem

`semgrep --sarif` (the invocation used by the gate's `pack_engines` step) honors `// nosemgrep` inline suppression comments. When a comment is present, semgrep does NOT drop the finding — it emits the result in the SARIF output with a non-empty `suppressions` array, per the SARIF 2.1.0 spec:

```json
{
  "ruleId": "go.core.no-global-mutable-state",
  "suppressions": [{ "kind": "inSource" }],
  "locations": [...]
}
```

A SARIF result carrying a non-empty `suppressions` array is a suppressed (inactive) finding. Active findings carry no `suppressions` key or an empty array.

**The gate's SARIF ingestion path does not inspect the `suppressions` field.**

`parseSarif` in `pkg/check/parsers.go` (lines 172–190) iterates `run.Results` unconditionally and appends every result as a violation regardless of suppression state. The `sarifLog` struct (lines 135–156) does not decode `suppressions` at all — the field is absent from the Go type, so its content is silently discarded during `json.Unmarshal`.

`ParsePackFindings` (lines 45–51) is the sole entry point for engine-path SARIF; it routes directly to `parseSarif`. Every suppressed result produced by semgrep becomes an active gate violation.

**Net effect:** `// nosemgrep` is a silent no-op in the engine/SARIF path. A legitimately-suppressed false positive still fails `pack_engines`. The standard inline suppression escape-hatch does not work for any pack running through the engine model.

### How it surfaced

A package-level `const` block defining `EngineCategory` iota constants (in `pkg/pack/engine/binding.go`) was false-flagged by the `go-standards` pack's `no-global-mutable-state` rule. The constants are immutable by definition; a `// nosemgrep: go.core.no-global-mutable-state` annotation was added to suppress the false positive. The gate remained red. Inspecting the raw semgrep SARIF output confirmed the result was emitted with `"suppressions": [{"kind": "inSource"}]` but counted as a violation anyway.

This bug has been adding friction across the bundle-010 implementation work; it is being fixed now as part of unblocking ISSUE-015.

## Impact

Any `// nosemgrep` suppression placed on a legitimately false-positive finding produces a permanently-red gate step with no actionable guidance — the developer has already applied the suppression the tool advertises, yet the gate refuses to pass. The only workarounds are:

1. Restructure real code to avoid triggering the rule (code contortion to satisfy a false positive), or
2. Disable the rule globally in the pack (blunt, loses enforcement for true positives).

Neither is acceptable. The suppression escape-hatch is a correctness contract. Breaking it silently makes the gate untrustworthy for any codebase that uses inline suppression as a workflow.

## Solution

**Minimum fix — drop suppressed results in `parseSarif`.**

1. Add a `Suppressions` field to the result struct in `sarifLog`:

```go
Suppressions []struct {
    Kind string `json:"kind"`
} `json:"suppressions"`
```

2. In the `parseSarif` loop, skip any result where `len(r.Suppressions) > 0` before appending to violations:

```go
if len(r.Suppressions) > 0 {
    continue // suppressed by inline annotation; not an active finding
}
```

This implements the SARIF spec rule: a result with a non-empty `suppressions` array is inactive and must not be treated as a violation by consumers.

**Test coverage.**

Add a test in `pkg/check/` (alongside `parse_pack_findings_test.go`) with a SARIF fixture containing one suppressed result (non-empty `suppressions`) and one active result (no `suppressions` key). Assert that `ParsePackFindings` returns exactly one violation — the active finding — and that the suppressed finding is absent from the output.

Suggested test name: `TestParsePackFindings_SuppressedResultsDropped`.

## References

- `pkg/check/parsers.go` lines 135–156 — `sarifLog` struct (missing `Suppressions` field)
- `pkg/check/parsers.go` lines 162–191 — `parseSarif` (unconditional append, no suppression check)
- `pkg/check/parsers.go` lines 45–51 — `ParsePackFindings` (sole SARIF entry point for engine path)
- `pkg/check/parse_pack_findings_test.go` — existing `ParsePackFindings` test file; new test belongs alongside
- SARIF 2.1.0 spec §3.27.23 — `result.suppressions` property; non-empty array = inactive/suppressed finding
- ISSUE-015 — the `EngineCategory` iota const block that surfaced this bug; fix here unblocks that work
- ISSUE-010 — sibling gate correctness bug (diff-scope leak); same `pack_engines` step

## Resolution

parseSarif now decodes the SARIF `suppressions` field and skips suppressed results, so `// nosemgrep` works in the engine/SARIF path.
