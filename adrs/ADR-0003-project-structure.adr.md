---
number: ADR-0003
created: "2026-03-17"
status: Accepted
deciders: "@bmanson"
decisions: "D-044 (revised), D-068, D-028"
schema_version: adr/v2
---

# ADR-0003: Backstop-Core Project Structure

## Context

Backstop-core is three things in one repository:

1. **A Go validation library** — the importable engine that validates artifacts, runs pack checkers, and verifies ledger integrity
2. **A CLI** — the agent-facing wrapper that shells out JSON for agent consumption
3. **A discipline framework** — schemas, templates, standards packs, and documentation

These three components have different consumers (library importers, CLI callers, framework users), different distribution mechanisms (Go modules, binary releases, GitHub Actions), and different change cadences. The project structure must make these boundaries visible and keep each component testable in isolation.

Additionally, backstop-core defines the framework — it does not consume it. There is no `.backstop/` directory. Artifacts like ADRs live at the root level, not nested under a consumer directory structure.

## Decision

### Top-level directory structure

```
backstop-core/
│
├── adrs/                        ← architectural decision records (this repo's own ADRs)
│
├── artifacts/                   ← schema + template definitions for each primitive
│   ├── adr/
│   │   ├── schema.json          ← current schema (latest)
│   │   ├── v1/                  ← versioned schema directory
│   │   │   └── schema.json
│   │   ├── template.md
│   │   ├── README.md
│   │   └── tests/
│   ├── spec/
│   ├── plan/
│   ├── directive/
│   ├── bundle/
│   ├── issue/
│   └── capability/
│
├── standards/                   ← standards packs (polyglot native checkers)
│   ├── core/                    ← language-agnostic structural standards
│   │   ├── policies/
│   │   ├── testdata/
│   │   └── README.md
│   ├── go/                      ← Go-specific standards (Go checkers)
│   │   ├── policies/
│   │   ├── testdata/
│   │   │   ├── valid/
│   │   │   └── invalid/
│   │   └── README.md
│   └── typescript/              ← TypeScript-specific standards (TS checkers)
│       ├── policies/
│       ├── testdata/
│       │   ├── valid/
│       │   └── invalid/
│       └── README.md
│
├── pkg/                         ← public Go library packages (importable)
│   ├── validate/                ← core validation engine
│   │   ├── validate.go
│   │   ├── validate_test.go
│   │   └── ...
│   ├── schema/                  ← schema loading and version resolution
│   ├── ledger/                  ← ledger parsing, hash chain verification
│   ├── pack/                    ← pack loading, checker invocation
│   └── manifest/                ← backstop.yml parsing and validation
│
├── cmd/                         ← CLI binary
│   └── backstop/
│       └── main.go
│
├── internal/                    ← private Go packages (not importable)
│   ├── cli/                     ← CLI command implementations
│   ├── runtime/                 ← runtime integration (hook generation per target)
│   └── template/                ← template rendering for `backstop new`
│
├── actions/                     ← GitHub Actions (peer consumer of pkg/)
│   ├── validate/
│   │   ├── action.yml
│   │   └── ...
│   └── report/
│       ├── action.yml
│       └── ...
│
├── scripts/                     ← repo-level gate scripts
│   └── tests/
│
├── docs/                        ← user-facing documentation
│   ├── getting-started.md
│   ├── cli-reference.md
│   └── pack-authoring.md
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
├── CONTRIBUTING.md
└── LICENSE
```

### Design principles

**1. `pkg/` is the first-class artifact.**

The validation engine lives in `pkg/validate/`, not in `internal/` or `cmd/`. It is a public, importable Go package with a stable API. The CLI (`cmd/backstop/`) and GitHub Actions (`actions/`) are both thin wrappers that import `pkg/` — neither contains validation logic. This ensures one engine, multiple interfaces (ADR-0004).

**2. Schemas are embedded, not read from disk.**

At build time, schemas from `artifacts/*/schema.json` and versioned directories are embedded into the Go binary via `go:embed` (D-028). The binary carries its own schema definitions. No filesystem dependency at runtime, no schema drift between CLI versions.

**3. Everything tests itself where it lives.**

- Artifact schema tests live in `artifacts/<type>/tests/`
- Pack checker tests live in `standards/<pack>/testdata/` with `valid/` and `invalid/` fixtures
- Go package tests are colocated with Go code (`*_test.go`)
- Gate script tests live in `scripts/tests/`

There is no centralized test directory. If you want to understand what a pack enforces, look at its testdata. If you want to understand what a schema requires, look at its tests.

**4. Standards packs are native to their language.**

Go packs contain Go checkers. TypeScript packs contain TypeScript checkers. The `standards/` directory does not contain Go binaries for all languages — it contains source code in the target language (ADR-0006). Same-language checkers run in-process via library import. Cross-language invocation uses a subprocess with JSON I/O protocol.

**5. backstop-core does not consume itself.**

There is no `.backstop/` directory. ADRs live in `adrs/`. This repo defines the framework; consuming repos use it via `backstop init` which creates the `.backstop/` directory structure defined in ADR-0002.

### Key boundaries

| Component | Location | Consumer | Distribution |
|-----------|----------|----------|-------------|
| Validation library | `pkg/` | Go importers, CLI, Actions | `go get` |
| CLI | `cmd/backstop/` | Agents (via shell) | Binary releases |
| GitHub Actions | `actions/` | CI pipelines | GitHub Marketplace / repo reference |
| Schemas | `artifacts/` | Embedded in binary at build | `go:embed` |
| Standards packs | `standards/` | Validation engine | Embedded in binary |
| Documentation | `docs/` | Humans | Published to docs site |

## Consequences

### What this enables
- **The validation library is independently importable.** GitHub Actions import `pkg/validate/` directly — no CLI installation in CI.
- **Schema versioning is filesystem-visible.** `artifacts/spec/v1/schema.json`, `artifacts/spec/v2/schema.json` — versions are directories, not branches.
- **Pack development is self-contained.** Everything a pack author needs (policies, fixtures, README) lives under `standards/<pack>/`.
- **The repo structure is the architecture diagram.** `pkg/` = library, `cmd/` = CLI, `actions/` = CI, `artifacts/` = schemas, `standards/` = rules.

### What this requires
- **Discipline about `pkg/` vs `internal/`.** Public API surfaces go in `pkg/`. Implementation details go in `internal/`. The boundary is enforced by Go's module system.
- **Embedded schema build step.** The Makefile must embed schemas before building the binary.
- **Cross-language test infrastructure.** TypeScript pack tests need Node.js in CI. Go pack tests need Go. The test matrix grows with language support.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Monorepo with separate Go modules per component | Adds module management overhead. The components are tightly coupled enough that a single module with `pkg/`/`internal/` separation is cleaner. |
| Validation logic in `internal/` only | Makes the validation engine un-importable. GitHub Actions would need to install and shell out to the CLI, adding a CI dependency. |
| Separate repo for GitHub Actions | Splits the validation engine across repos. Actions and CLI would drift. One repo with one engine is simpler. |
| Schemas in a separate repo | Creates a distribution/versioning problem. Embedding schemas in the binary eliminates it. |
| `.backstop/` directory for backstop-core's own artifacts | Circular — the framework consuming itself creates bootstrap confusion. Root-level directories are clearer. |

## References

- D-044: Original repo structure decision (revised by this ADR)
- D-068: Validation engine as Go library with CLI and Actions as peer consumers
- D-028: Schemas embedded in binary via `go:embed`
- ADR-0001: Agent-first discipline framework
- ADR-0002: Canonical artifact primitives (defines what lives in `artifacts/`)
- ADR-0004: Validation engine architecture (forthcoming — defines `pkg/validate/` API)
