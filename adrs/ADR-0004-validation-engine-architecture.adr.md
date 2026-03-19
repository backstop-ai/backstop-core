---
number: ADR-0004
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-068, D-069, D-028, D-061"
schema_version: adr/v2
---

# ADR-0004: Validation Engine Architecture

## Context

Backstop's thesis — "if it's green, it ships" — requires a validation engine that is the single source of truth for what "green" means. That engine must serve three consumers with different needs:

1. **Agents** need structured JSON output they can parse mid-session to decide whether to continue or fix violations
2. **CI pipelines** need GitHub-native annotations, check runs, and PR comments that give developers rich feedback without leaving their workflow
3. **Standard library authors** need in-process validation they can call from test suites without shelling out to a CLI

If these three consumers each have their own validation logic, they will inevitably drift. One engine, multiple interfaces.

## Decision

### The validation library is the first-class artifact

All validation logic lives in `pkg/validate/` — a public, importable Go package. It performs:

- **Schema validation** — artifacts checked against their declared schema version
- **Pack checking** — standards pack rules evaluated against code
- **Ledger verification** — hash chain integrity, completeness checks
- **AST analysis** — test body substantiveness, mandated test name presence
- **Baseline comparison** — current violations compared against baseline file
- **Waiver resolution** — active waivers suppress matching violations

The library exposes a clean Go API:

```go
result, err := validate.Run(validate.Config{
    ManifestPath: "backstop.yml",
    Scope:        validate.ChangedFiles,  // or validate.AllFiles
    BaselinePath: ".backstop/baseline.yml",
    WaiverDir:    ".backstop/waivers/",
})
```

The `result` is a structured object containing violations, warnings, waivers applied, baseline comparisons, and a pass/fail verdict. Every consumer reads this same result type.

### Three peer consumers, one engine

| Consumer | Location | Wraps | Output Format |
|----------|----------|-------|---------------|
| **CLI** | `cmd/backstop/` | `pkg/validate/` | Structured JSON to stdout |
| **GitHub Actions** | `actions/validate/` | `pkg/validate/` | Annotations, check runs, PR comments |
| **In-process** | Direct import of `pkg/validate/` | N/A | Go struct |

The CLI and Actions contain zero validation logic. They are thin wrappers that call `validate.Run()` and format the output for their respective consumers.

### Agents interface via CLI exclusively

Regardless of language (Go, TypeScript, Python), agents shell out to `backstop validate --json` for proactive validation. There are no language-specific agent SDKs. The CLI's JSON output is the agent-facing API contract.

```bash
$ backstop validate --json
{
  "pass": false,
  "violations": [
    {
      "rule": "go:security@2.0.0:SEC-0012",
      "file": "internal/auth/handler.go",
      "line": 47,
      "message": "Hardcoded credential detected",
      "severity": "error"
    }
  ],
  "warnings": [],
  "waivers_applied": [],
  "baseline_delta": { "new": 1, "resolved": 0 }
}
```

This JSON format is a stable, versioned API contract. Breaking changes follow the same schema evolution model as artifacts (D-070).

### Schemas are embedded

At build time, all schemas from `artifacts/*/schema.json` and versioned directories are embedded into the Go binary via `go:embed` (D-028). The binary is self-contained — no filesystem dependency for schemas at runtime, no drift between CLI versions.

### CI is the same validation, more infrastructure

There is no `--ci` flag. `backstop validate` runs identically everywhere. The difference is what's available in the environment: CI has databases for integration tests, network access for security scanning, and GitHub API tokens for annotations. Locally, those tests are skipped. The validation engine is the same; the infrastructure differs (D-061).

## Consequences

### What this enables
- **One truth.** CLI, Actions, and in-process callers all use the same validation logic. Drift is structurally impossible.
- **No CLI in CI.** GitHub Actions import the Go library directly. No installation step, no version mismatch, no binary compatibility issues.
- **Testable engine.** The validation library has its own test suite independent of CLI or Actions. Engine correctness is verified in isolation.
- **Extensible.** New validation capabilities (new pack types, new analysis passes) are added to `pkg/validate/` and immediately available to all three consumers.

### What this requires
- **Stable Go API.** The `pkg/validate/` interface is a public contract. Breaking changes require major version bumps.
- **Structured result type.** The result object must be rich enough for Actions to generate line-level annotations and for agents to make decisions. It cannot be a simple pass/fail boolean.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| CLI as the single interface (Actions shell out to CLI) | CI would need to install the CLI binary, manage versions, and parse CLI output. Importing the library directly is simpler, faster, and eliminates a dependency. |
| Separate validation implementations per consumer | Drift is inevitable. Two implementations of "does this schema validate?" will disagree within months. |
| gRPC/protocol buffer for cross-language validation | Over-engineered for the problem. Agents can shell out. Actions can import. A protocol layer adds complexity without value. |
| Validation logic in `internal/` | Makes the engine un-importable by Actions. Forces the "CLI in CI" anti-pattern. |

## References

- D-068: Validation engine as Go library with CLI and Actions as peer consumers
- D-069: CLI is the universal agent API — `backstop validate --json`
- D-028: Schemas embedded in binary via `go:embed`
- D-061: CI is same validation, more infrastructure — no `--ci` flag
- ADR-0003: Project structure (where `pkg/validate/` lives)
- ADR-0009: CI/CD pipeline (forthcoming — how Actions use this engine)
