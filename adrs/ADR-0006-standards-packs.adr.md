---
number: ADR-0006
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-016–D-023, D-030–D-034, D-031v2, D-032v2, D-046, D-075, D-076"
schema_version: adr/v2
---

# ADR-0006: Standards Packs — Polyglot Native Checkers as Library Code

## Context

Backstop enforces code standards through "packs" — collections of rules, checkers, and test fixtures that define what good code looks like for a given language and concern. The pack model must answer three questions:

1. **What categories of standards exist?** (structure, security, performance, etc.)
2. **How are checkers implemented?** (same language as the code they check, not a universal DSL)
3. **How do teams adopt packs on existing codebases?** (without blocking all work on day one)

## Decision

### Ten sub-component types per language pack

Each language pack (backstop-go, backstop-typescript, etc.) contains up to ten sub-components. Each is independently versioned and toggleable:

| Sub-component | What it enforces |
|---------------|-----------------|
| **core** | Language idioms, structure, naming conventions |
| **test** | Test quality — no vacuous assertions, naming, table tests, coverage patterns |
| **security** | ASVS-mapped code-level checks — injection, auth, crypto, data protection |
| **performance** | N+1 queries, unbounded fetches, pagination, connection pooling, O(n²) patterns |
| **observability** | Structured logging, trace propagation, metrics, health endpoints |
| **integration** | Standards for connecting TO external systems — SQL, HTTP clients, messaging |
| **contracts** | Standards for what you expose OUTWARD — REST shape, auth middleware, API versioning |
| **concurrency** | Goroutine/async/thread patterns (language-dependent, omitted where N/A) |
| **accessibility** | WCAG compliance — semantic HTML, ARIA, keyboard nav (frontend packs only) |
| **resilience** | Retry patterns, circuit breakers, graceful degradation, timeout handling |

Sub-components are referenced with explicit qualification: `go:security@2.0.0:SEC-0012`. No implicit prefix resolution — unambiguous regardless of ecosystem growth.

### Polyglot native checkers

Checkers are written in the language they police. Go checkers for Go code. TypeScript checkers for TypeScript code.

- **Same-language invocation:** checker runs in-process via library import (no subprocess overhead)
- **Cross-language invocation:** CLI shells out to a thin runner in the target language with standard JSON I/O protocol
- **New language support:** publish a new library + register with the CLI. Never rewrite the CLI.

This replaced the original D-032 decision (Go-only checkers compiled to binaries for all languages). Checkers must speak the language they police — a Go binary cannot understand TypeScript AST idioms.

### Pack structure

Each pack is self-contained:

```
standards/go/
  policies/
    core/            ← rule definitions (YAML)
    test/
    security/
    performance/
    ...
  testdata/
    valid/           ← code that passes all rules
    invalid/         ← code that violates specific rules
  src/               ← checker source code (Go for Go packs)
  README.md
```

The `testdata/` fixtures are the behavioral contract. `valid/` contains code that must pass. `invalid/` contains code that must fail with specific violations. No fixture, no merge for pack PRs. Fixtures are the pack's test suite and its documentation — look at `invalid/sec-0012-hardcoded-credential.go` to understand exactly what SEC-0012 catches.

### Checker tools per concern

Purpose-built checker tools, one per concern:

| Tool | Scope |
|------|-------|
| `backstop-go-ast` | Go production code AST — structure, idioms, error handling, DI |
| `backstop-go-test-ast` | Go test AST — vacuous assertions, table tests, naming, reachability |
| `backstop-ts-ast` | TypeScript production AST — any usage, return types, module exports |
| `backstop-ts-test-ast` | TypeScript test AST — testing-library patterns, async assertions |
| `backstop-bash-ast` | Bash script analysis — set -euo pipefail, quoting, injection |
| `backstop-opa` | OPA policy evaluation — cross-cutting security rules |

A single standard can have multiple checkers (`enforcedBy` is always an array). Belt and suspenders.

### Waivers

When a team needs an exception to a pack rule, they create a waiver file (D-075):

```yaml
# .backstop/waivers/WAIVER-0001-legacy-auth.waiver.yml
waiver:
  id: WAIVER-0001
  rule: go:security@2.0.0:SEC-0012
  scope: internal/auth/legacy.go
  justification: "Legacy OAuth1 integration, replacement scheduled for Q3"
  expires: 2026-09-01
  approved_by: "@bmanson"
  created: 2026-03-18
```

Waivers fully suppress the rule while active. When the expiry passes, the rule re-enforces and fails hard. Expiry is required by default (configurable via `enforcement.allow_open_waivers` in backstop.yml).

### Baseline scan

For existing codebases adopting backstop, `backstop baseline` records all pre-existing violations (D-076). Subsequent `backstop validate` runs only fail on new violations above the baseline. The baseline ratchets — it can only get better, never worse. `backstop baseline --update` lowers the count when violations are fixed.

## Consequences

### What this enables
- **Language-native enforcement.** Checkers understand the idioms and AST of the language they check. No false positives from cross-language impedance mismatch.
- **Granular adoption.** Enable `go:security` without `go:performance`. Pin versions per sub-component.
- **Day-one adoption on legacy codebases.** Baseline scan + waivers mean enforcement starts immediately without blocking existing work.
- **Provable pack behavior.** testdata fixtures define exactly what passes and fails. No ambiguity.

### What this requires
- **Checker implementation per language.** Each supported language needs its own checker tools. This is the cost of polyglot native enforcement.
- **Fixture maintenance.** Every rule needs valid/ and invalid/ fixtures. This is the pack's test suite — skipping it means the rule is untested.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Go-only checkers compiled to binaries | Cannot understand target language AST. A Go binary checking TypeScript is guessing, not verifying. |
| Universal DSL for rules (like Rego for everything) | Loses language-native understanding. OPA is good for policy but not for "is this Go idiomatic?" |
| No sub-components — one monolithic pack per language | Versioning becomes impossible. A test quality fix forces re-validation of all production code standards. |
| Floating rule versions | Non-deterministic. The same code could pass Monday and fail Tuesday. Explicit version pinning is mandatory. |

## References

- D-016: Ten canonical sub-component types (revised from original six)
- D-017–D-023: Pack versioning, enforcement, qualification, checker ecosystem
- D-030–D-034: Pack structure, testdata contract, third-party authoring (post-MVP)
- D-031v2, D-032v2: Polyglot native checkers (revised from Go-only)
- D-046: Pack checkers as library code
- D-075: Waiver files with required justification and expiry
- D-076: Baseline scan for bulk adoption amnesty
- ADR-0005: backstop.yml manifest (pack configuration lives here)
- ADR-0007: Security standards pack (forthcoming — ASVS-specific pack details)
