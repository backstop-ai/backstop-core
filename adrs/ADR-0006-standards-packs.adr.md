---
number: ADR-0006
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-016–D-023, D-030–D-034, D-031v2, D-032v2, D-046, D-075, D-076, D-085, D-086"
schema_version: adr/v2
---

# ADR-0006: Standards Packs — Semgrep-Powered Declarative Enforcement

## Context

Backstop enforces code standards through "packs" — collections of rules and test fixtures that define what good code looks like for a given language and concern. The pack model must answer three questions:

1. **What categories of standards exist?** (structure, security, performance, etc.)
2. **How are checks implemented?** (what engine runs the rules?)
3. **How do teams adopt packs on existing codebases?** (without blocking all work on day one)

An earlier version of this ADR designed language-native AST checker tools (`backstop-go-ast`, `backstop-ts-ast`, etc.) — one compiled binary per language per concern. In practice, these checkers devolved into regex and string matchers. Effective enough to catch violations, but brittle, hard to maintain, and impossible for third parties to author.

Semgrep OSS resolves all three problems. It provides AST-aware pattern matching across 30+ languages with a single declarative YAML rule syntax. Running semgrep against 71k lines of production Go returned ~500 built-in checks with near-zero false positives on production code. Custom rules for style opinions (DI patterns, `init()` bans, naming conventions) required ~200 lines of YAML total. The decision was obvious: semgrep is the checker engine, backstop curates and distributes the rules.

## Decision

### Semgrep OSS as the universal checker engine (D-085)

All code-level enforcement runs through Semgrep OSS. There are no language-native checker binaries. There is no custom AST parsing. Backstop does not compete with semgrep — it layers opinionated rule packs on top of semgrep's engine.

What semgrep provides:
- **AST-aware pattern matching** — structural code analysis, not regex
- **30+ supported languages** — Go, TypeScript, Python, Java, C#, Rust, and more
- **~300-600 built-in rules per major language** — security, correctness, best practices
- **Intraprocedural taint analysis** — data flow tracking within a function
- **One rule syntax (YAML)** — the same pattern language for every target language
- **Constant propagation** — detects hardcoded secrets and magic numbers

What semgrep does NOT provide (and backstop handles separately):
- **Cross-file analysis** — semgrep OSS is single-file. Cross-file traceability lives in backstop's artifact validators and the implementation verifier
- **Interprocedural dataflow** — taint tracking stops at function boundaries in OSS
- **Test substantiveness** — D-057 verification (does the test call production code, make meaningful assertions, cover sharp edges) requires backstop's own verifier
- **Runtime verification** — tests passing, coverage thresholds, contract signatures
- **Architectural enforcement** — dependency direction, package boundaries, module structure

### Ten rule categories per language pack (D-016, revised)

The ten sub-component types survive as **rule categories** within a pack, not as separate checker tools. Each category is a directory of semgrep YAML rules:

| Category | What it enforces | Semgrep coverage |
|----------|-----------------|------------------|
| **core** | Language idioms, structure, naming, DI patterns, `init()` bans | Strong — custom rules |
| **test** | No vacuous assertions, naming, table tests, coverage patterns | Partial — substantiveness needs verifier |
| **security** | ASVS-mapped checks — injection, auth, crypto, data protection | Strong — semgrep's home turf |
| **performance** | N+1 queries, unbounded fetches, pagination, connection pooling | Partial — algorithmic complexity needs verifier |
| **observability** | Structured logging, trace propagation, metrics, health endpoints | Strong — pattern matching |
| **integration** | SQL parameterization, HTTP client patterns, messaging standards | Strong — pattern matching |
| **contracts** | REST shape, auth middleware, API versioning patterns | Partial — full contract verification needs verifier |
| **concurrency** | Goroutine/async patterns, mutex misuse, channel patterns | Strong — pattern matching |
| **accessibility** | WCAG compliance, semantic HTML, ARIA, keyboard nav (frontend only) | Strong — pattern matching |
| **resilience** | Retry patterns, circuit breakers, graceful degradation, timeouts | Partial — correctness needs verifier |

Categories are referenced with explicit qualification: `go:security@2.0.0:SEC-0012`. No implicit prefix resolution — unambiguous regardless of ecosystem growth.

### Pack structure (D-030, revised)

Each pack is self-contained. The `src/` directory of compiled checkers is replaced by `rules/` containing semgrep YAML:

```
standards/go/
  rules/
    core/              ← semgrep YAML rules for Go idioms, structure, naming
    test/              ← test quality rules
    security/          ← ASVS-mapped security rules
    performance/       ← performance anti-pattern rules
    observability/
    integration/
    contracts/
    concurrency/
    resilience/
  testdata/
    valid/             ← code that passes all rules
    invalid/           ← code that violates specific rules (one file per rule)
  semgrep.yml          ← pack-level config (includes, severity overrides)
  README.md
```

The `testdata/` fixtures remain the behavioral contract. `valid/` contains code that must pass. `invalid/` contains code that must fail with specific violations. Fixtures are the pack's test suite and its documentation — look at `invalid/sec-0012-hardcoded-credential.go` to understand exactly what SEC-0012 catches.

### Custom rule anatomy

A backstop custom rule is a standard semgrep rule with backstop metadata:

```yaml
rules:
  - id: go.core.no-init-functions
    patterns:
      - pattern: |
          func init() { ... }
    message: >
      Do not use init() — use explicit initialization via constructors or
      setup functions. init() creates hidden execution order dependencies.
    severity: ERROR
    languages: [go]
    metadata:
      backstop:
        category: core
        rule_id: CORE-0001
        rationale: "Explicit initialization over implicit. DI-friendly."
```

Custom rules average ~10 lines of YAML each. The entire custom rule set for a language pack is typically under 500 lines — orders of magnitude less code than language-native checker implementations.

### Two enforcement engines (D-086)

Backstop uses exactly two enforcement mechanisms:

| Engine | What it checks | How it runs |
|--------|---------------|-------------|
| **Semgrep OSS** | Code patterns, style, security, correctness — everything expressible as AST pattern matching | `semgrep --config .backstop/rules/ --json` |
| **Backstop verifier** | Test substantiveness (D-057), contract signature verification, coverage thresholds, cross-file traceability | `backstop verify` (built into CLI) |

A single standard can be enforced by both engines (`enforcedBy` is always an array). Belt and suspenders. Example: a security rule might have a semgrep check for SQL concatenation AND a verifier check that parameterized query wrappers are imported from the standard library.

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
- **Instant multi-language support.** Adding a new language is authoring rules, not building a compiler. Semgrep already parses the AST.
- **Community-authored rule packs.** Publishing a checker plugin is publishing a directory of YAML files, not shipping a compiled binary.
- **~500 checks for free per language.** Semgrep's built-in rules cover security, correctness, and best practices before a single custom rule is written.
- **Granular adoption.** Enable `go:security` without `go:performance`. Pin versions per category.
- **Day-one adoption on legacy codebases.** Baseline scan + waivers mean enforcement starts immediately without blocking existing work.
- **Provable pack behavior.** testdata fixtures define exactly what passes and fails. No ambiguity.

### What this requires
- **Semgrep OSS as a runtime dependency.** Must be installed alongside backstop CLI. Version pinning for reproducibility.
- **Fixture maintenance.** Every rule needs valid/ and invalid/ fixtures. This is the pack's test suite — skipping it means the rule is untested.
- **Verifier for what semgrep can't do.** Test substantiveness, contract verification, coverage gates, and cross-file analysis remain backstop's responsibility.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Language-native AST checkers (original D-032v2) | In practice these devolved into regex and string matchers. Semgrep gives real AST analysis with less code, across all languages, with a single rule syntax. |
| Universal DSL for rules (like Rego for everything) | OPA/Rego is good for policy but not for code pattern matching. Semgrep's pattern syntax is closer to source code — rule authors write patterns that look like the code they're matching. |
| Semgrep Pro instead of OSS | Pro adds cross-file analysis and interprocedural taint, but introduces a commercial dependency. OSS covers the code-pattern layer; backstop's verifier handles the cross-file layer. |
| No sub-components — one monolithic pack per language | Versioning becomes impossible. A test quality fix forces re-validation of all production code standards. |
| Floating rule versions | Non-deterministic. The same code could pass Monday and fail Tuesday. Explicit version pinning is mandatory. |

## References

- D-016: Ten canonical rule categories (revised from sub-component types)
- D-017–D-023: Pack versioning, enforcement, qualification, checker ecosystem
- D-030–D-034: Pack structure, testdata contract, third-party authoring
- D-085: Semgrep OSS as universal checker engine
- D-086: Two enforcement engines (semgrep + backstop verifier)
- D-075: Waiver files with required justification and expiry
- D-076: Baseline scan for bulk adoption amnesty
- ADR-0005: backstop.yml manifest (pack configuration lives here)
- ADR-0007: Security standards pack (ASVS-specific pack details)
- ADR-0010: Verification kill chain (verifier architecture)
