---
title: Go Code Standards
number: STD-GO-001
created: "2026-03-29"
status: active
schema_version: standard/v1
language: go
pack: go
scope: language

sources:
  - title: "Effective Go"
    url: "https://go.dev/doc/effective_go"
    authority: "Go Team"
  - title: "Go Code Review Comments"
    url: "https://go.dev/wiki/CodeReviewComments"
    authority: "Go Team"
  - title: "Go Proverbs"
    url: "https://go-proverbs.github.io"
    authority: "Rob Pike"

rules:
  # ── Structure ──────────────────────────────────────────────────────────

  - id: GO-001
    name: max-file-length
    category: structure
    severity: error
    description: Go source files must not exceed 500 lines excluding tests
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: file_lines
      operator: ">"
      threshold: 500
      exclude: "_test.go"
    fix: Split the file by responsibility boundary
    sources:
      - title: "backstop origin story — the 4,000-line router"

  - id: GO-002
    name: max-function-length
    category: structure
    severity: error
    description: Functions must not exceed 100 lines
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: function_lines
      operator: ">"
      threshold: 100
    fix: Extract helper functions or decompose into smaller units

  - id: GO-003
    name: no-global-mutable-state
    category: structure
    severity: error
    description: Package-level mutable variables are forbidden except sync primitives and immutable lookup tables
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        var $NAME = ...
      exceptions:
        - "sync\\.Once"
        - "sync\\.Mutex"
        - "sync\\.RWMutex"
        - "regexp\\.MustCompile"
        - "map\\[string\\]bool{"
        - "map\\[string\\]\\*?int{"
        - "\\[\\]string{"
    fix: Use dependency injection via constructors
    sources:
      - title: "Go Code Review Comments — Global State"
        url: "https://go.dev/wiki/CodeReviewComments#global-state"

  - id: GO-004
    name: no-init-functions
    category: structure
    severity: error
    description: init() functions are forbidden outside very narrow bootstrap use
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        func init() {
          ...
        }
    fix: Replace init side effects with explicit constructor wiring

  - id: GO-005
    name: constructor-injection-required
    category: structure
    severity: warning
    description: Prefer constructor-based dependency injection over direct struct literal wiring in callers
    compliance_tier: standard
    detection:
      strategy: pattern
      semgrep: |
        $TYPE{
          $FIELD: $VALUE,
          ...
        }
      note: "Aspirational architectural pattern; semgrep can only flag likely dependency-wiring literals and may require review for intent."
    fix: Provide New<Type>(deps...) constructors and inject dependencies explicitly

  - id: GO-006
    name: structured-logging-required
    category: structure
    severity: warning
    description: Use structured logging APIs instead of stdlib log.Print/Fatal style calls
    compliance_tier: standard
    detection:
      strategy: pattern
      semgrep: |
        log.$METHOD(...)
    fix: Use structured logger fields (for example zap/slog) instead of unstructured log calls

  # ── Error Handling ─────────────────────────────────────────────────────

  - id: GO-010
    name: no-ignored-errors
    category: error-handling
    severity: error
    description: Error return values must not be silently discarded
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        $VAL, _ := $FUNC(...)
    fix: Handle the error or use an explicit blank identifier with a comment
    sources:
      - title: "Effective Go — Errors"
        url: "https://go.dev/doc/effective_go#errors"

  - id: GO-011
    name: error-wrapping-required
    category: error-handling
    severity: warning
    description: Errors returned from called functions should be wrapped with context using %w
    compliance_tier: standard
    detection:
      strategy: pattern
      semgrep: |
        if $ERR != nil {
          return ..., $ERR
        }
    fix: "Use fmt.Errorf(\"context: %w\", err) to add context"
    sources:
      - title: "Go Blog — Working with Errors in Go 1.13"
        url: "https://go.dev/blog/go1.13-errors"

  - id: GO-012
    name: no-naked-returns
    category: error-handling
    severity: warning
    description: Naked returns are forbidden in functions longer than 5 lines
    compliance_tier: standard
    detection:
      strategy: pattern
      semgrep: |
        func $NAME(...) (...) {
          ...
          return
        }
      constraint: "function_lines > 5"
    fix: Use explicit return values for clarity

  - id: GO-013
    name: no-panic-in-library-code
    category: error-handling
    severity: error
    description: panic() must not be used in library code — return errors instead
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        panic(...)
      exclude: "main.go"
    fix: Return an error to the caller
    sources:
      - title: "Effective Go — Panic"
        url: "https://go.dev/doc/effective_go#panic"

  # ── Naming ─────────────────────────────────────────────────────────────

  - id: GO-020
    name: no-stuttering-exports
    category: naming
    severity: warning
    description: Exported names should not repeat the package name
    compliance_tier: standard
    detection:
      strategy: delegated
      enforced_by: golangci-lint
      rule: revive/exported
      note: "Delegated to golangci-lint revive/exported; appears in manifest, not semgrep YAML."
    fix: "Use validate.Spec not validate.ValidateSpec"
    sources:
      - title: "Go Code Review Comments — Package Names"
        url: "https://go.dev/wiki/CodeReviewComments#package-names"

  - id: GO-021
    name: error-type-suffix
    category: naming
    severity: warning
    description: Custom error types must have an Error suffix
    compliance_tier: standard
    detection:
      strategy: regex
      pattern: "type\\s+\\w+(?<!Error)\\s+struct.*\\n.*Error\\(\\)"
    fix: "Rename FooFailed to FooError"
    sources:
      - title: "Go Code Review Comments — Error Types"

  # ── Testing ────────────────────────────────────────────────────────────

  - id: GO-030
    name: test-file-required
    category: testing
    severity: error
    description: Every .go source file must have a corresponding _test.go file
    compliance_tier: baseline
    detection:
      strategy: metric
      metric: test_file_exists
      exclude: "main.go"
      note: "Requires file-level correlation between source and test files; enforced by native metric analysis, not semgrep."

  - id: GO-031
    name: table-driven-tests
    category: testing
    severity: warning
    description: Tests with multiple cases should use table-driven patterns with t.Run()
    compliance_tier: standard
    detection:
      strategy: pattern
      note: "Positive pattern — detect tests with >3 assertions that lack t.Run(). Cannot be expressed as a single semgrep pattern; requires custom analysis."
    sources:
      - title: "Go Wiki — Table Driven Tests"
        url: "https://go.dev/wiki/TableDrivenTests"

  - id: GO-032
    name: no-time-sleep-in-tests
    category: testing
    severity: error
    description: time.Sleep must not be used in tests — use channels, tickers, or test helpers
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        time.Sleep(...)
      scope: "_test.go"
    fix: Use synchronization primitives instead of arbitrary sleeps

  # ── Imports ────────────────────────────────────────────────────────────

  - id: GO-040
    name: import-ordering
    category: imports
    severity: warning
    description: Imports must be grouped in order — stdlib, external, internal — separated by blank lines
    compliance_tier: standard
    detection:
      strategy: delegated
      enforced_by: golangci-lint
      rule: goimports
      note: "Delegated enforcement via golangci-lint/goimports; this rule does not produce semgrep YAML."

  # ── Concurrency ────────────────────────────────────────────────────────

  - id: GO-050
    name: no-goroutine-leak
    category: concurrency
    severity: error
    description: Goroutines must have a clear termination path — context cancellation or channel close
    compliance_tier: strict
    detection:
      strategy: pattern
      semgrep: |
        go func() { ... }()
      note: "Flag goroutines without context or done channel for manual review"
    fix: Pass a context.Context or done channel to every goroutine

  - id: GO-051
    name: race-detector-required
    category: concurrency
    severity: error
    description: Tests must run with -race flag enabled
    compliance_tier: baseline
    detection:
      strategy: delegated
      enforced_by: makefile
      rule: test-race-flag

  # ── Security ─────────────────────────────────────────────────────────────

  - id: GO-060
    name: no-hardcoded-credentials
    category: security
    severity: error
    description: Credentials and secrets must not be hardcoded in source
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        $NAME := "$VALUE"
      note: "Detects obvious hardcoded secret assignments (password/token/secret/key variable names)."
    fix: Load secrets from secure configuration or environment at runtime

  - id: GO-061
    name: no-weak-password-hashing
    category: security
    severity: error
    description: Weak hash functions like MD5 and SHA1 must not be used for password hashing
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        md5.$FUNC(...)
    fix: Use adaptive password hashing algorithms such as bcrypt/argon2

  - id: GO-062
    name: no-sql-concatenation
    category: security
    severity: error
    description: SQL query strings must not be built with string concatenation
    compliance_tier: baseline
    detection:
      strategy: pattern
      semgrep: |
        $QUERY := "SELECT " + ...
      note: "Targets obvious SQL string concatenation patterns; false positives may require review."
    fix: Use parameterized queries with placeholders and bound arguments

  - id: GO-063
    name: no-sensitive-data-in-logs
    category: security
    severity: warning
    description: Sensitive values such as passwords, tokens, and secrets must not be logged
    compliance_tier: standard
    detection:
      strategy: pattern
      semgrep: |
        $LOG.$METHOD(..., $SENSITIVE, ...)
    fix: Redact or omit sensitive fields before logging
---

# Go Code Standards

## Overview

These standards define the mechanical enforcement rules for Go code within backstop-managed projects. Every rule compiles to a semgrep pattern, metric check, or regex match that runs automatically — no human judgment required.

Rules are organized by category and assigned a compliance tier (baseline, standard, strict) that maps to the project's configured enforcement level. Baseline rules are always enforced. Standard rules are enforced for most projects. Strict rules are opt-in for high-assurance contexts.

## Rules

### Structure (GO-001 – GO-006)

Structural rules prevent the accumulation of complexity that makes codebases unmaintainable. The 500-line file limit and 100-line function limit are hard gates — the exact kind of constraint that would have caught the 4,000-line router before it became a problem.

Global mutable state is banned because it creates invisible coupling between packages. The only exceptions are synchronization primitives (sync.Once, sync.Mutex) and compiled regexes (regexp.MustCompile), which are effectively immutable after initialization.

The pack also forbids init() functions for regular application wiring and encourages constructor-driven dependency injection. Structured logging is required over free-form log.Print/Fatal calls so observability pipelines can parse fields consistently.

### Error Handling (GO-010 – GO-013)

Go's explicit error handling is a strength, but only if errors are actually handled. Ignored errors are the most common source of silent failures in Go programs. Error wrapping with `%w` ensures that when something fails three layers deep, the caller knows what happened and where.

Panic is reserved for truly unrecoverable situations in application entry points. Library code must never panic — it returns errors and lets the caller decide.

### Naming (GO-020 – GO-021)

Go's package-qualified naming means that `validate.Spec` is the natural name, not `validate.ValidateSpec`. Stuttering exports are a signal that the package boundary is wrong or the name needs rethinking.

### Testing (GO-030 – GO-032)

Every source file gets a test file. Table-driven tests with `t.Run()` are the idiomatic pattern for multiple cases. `time.Sleep` in tests is a reliability anti-pattern — use synchronization primitives.

### Imports (GO-040)

Import ordering (stdlib → external → internal) is enforced by goimports and documented here for completeness.

### Concurrency (GO-050 – GO-051)

Goroutines without termination paths are memory leaks. The race detector catches data races but only if it's enabled — which is why `-race` is a baseline requirement, not optional.

### Security (GO-060 – GO-063)

Security baseline rules catch high-signal mistakes early: hardcoded credentials, weak password hashing primitives, SQL string concatenation, and logging of sensitive values. These rules are intentionally conservative and flag obvious violations for remediation and review.

## Examples

### ❌ Invalid — Ignored Error

```go
result, _ := doSomething()
```

### ✅ Valid — Error Handled

```go
result, err := doSomething()
if err != nil {
    return fmt.Errorf("doing something: %w", err)
}
```

### ❌ Invalid — Global Mutable State

```go
var cache = map[string]string{}
```

### ✅ Valid — Injected Dependency

```go
type Service struct {
    cache map[string]string
}

func NewService() *Service {
    return &Service{cache: make(map[string]string)}
}
```

### ❌ Invalid — Panic in Library Code

```go
func ParseConfig(path string) Config {
    data, err := os.ReadFile(path)
    if err != nil {
        panic(err) // never in library code
    }
    // ...
}
```

### ✅ Valid — Return Error

```go
func ParseConfig(path string) (Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return Config{}, fmt.Errorf("reading config %s: %w", path, err)
    }
    // ...
}
```
