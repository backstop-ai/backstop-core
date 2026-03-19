---
number: ADR-0013
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-045, D-046, D-047, D-048, D-049, D-050, D-053, D-058"
schema_version: adr/v2
---

# ADR-0013: Standard Library Model — Recipes as Importable Code

## Context

Backstop enforces how code is written (ADR-0006, ADR-0007) and how it is verified (ADR-0010). But enforcement alone does not solve the problem of agents producing inconsistent, unreliable integrations. An agent told to "build an HTTP server" will produce a different structure, error handling strategy, and observability surface every time — even with standards packs in place. Packs can reject bad output, but they cannot guide an agent toward good output.

The missing piece is a standard library: a set of pre-built, backstop-enforced implementations of common patterns that agents can import instead of hand-rolling. If the HTTP server recipe exists as a library, the agent imports it. The result is enterprise-grade by construction, not by enforcement after the fact.

Recipes (ADR-0002) already define patterns as machine-readable artifacts. The leap is generating importable library code from those recipes — one recipe producing N language implementations, all following identical patterns, error handling, and observability conventions. This transforms backstop from a linter into an infrastructure layer.

## Decision

### Recipes define patterns, libraries implement them

A recipe is a machine-readable description of how to do something: stand up an HTTP server, connect to a database, authenticate a user, call an external API. Recipes declare the pattern — its inputs, outputs, error modes, observability hooks, and configuration surface. They do not contain implementation code.

Libraries are generated from recipes. One recipe produces implementations across supported languages: `backstop-go/http`, `backstop-ts/http`, `backstop-py/http`. Each implementation follows the same patterns, handles errors the same way, emits the same telemetry, and exposes the same configuration. The recipe is the single source of truth; the libraries are its compiled artifacts.

### useBackstopLibraries: true

The `useBackstopLibraries` flag in backstop.yml instructs agents to prefer backstop standard libraries over hand-rolled integrations. When enabled, an agent building an HTTP server reaches for `backstop-go/http` instead of writing its own `net/http` wrapper. An agent integrating with Postgres reaches for `backstop-go/postgres` instead of writing raw `database/sql` calls.

This is not a hard constraint — agents can still write custom code when no library exists for the pattern. But when a library exists, the agent uses it. The result is that 80% of integration code in a backstop-managed project is library code: tested, enforced, and consistent across every project that uses backstop.

### Universal SDK layer (D-053)

The natural extension of language-level libraries is service-level SDKs: `backstop-go/twilio`, `backstop-go/stripe`, `backstop-go/gdrive`, `backstop-ts/postgres`. Each wraps a third-party service with backstop-standard error handling, retry logic, circuit breaking, observability, and configuration.

The SDK layer follows the same recipe-to-library pipeline. A recipe for "Twilio SMS" produces SDK implementations that handle rate limiting, authentication, error classification, and telemetry uniformly. An agent using `backstop-go/twilio` produces the same quality integration as a senior engineer who has read all of Twilio's API documentation and handled every edge case.

### Auto-generated recipe catalog (D-058)

The CLI introspects project dependencies, parses test ASTs to derive capabilities, and parses exports to derive API surfaces. The result is cached to `.backstop/cache/recipes.json` — a machine-readable catalog of every recipe available to the project.

Libraries remain normal packages with no backstop-specific metadata. The CLI extracts everything it needs from code structure and tests. This keeps backstop libraries usable outside of backstop while still enabling the catalog to index them automatically.

### Recipes as first-class artifacts

Recipes are declared in specs, enforced by packs, and recorded in ledgers — the same artifact lifecycle as every other backstop primitive (ADR-0002). A recipe has a schema version, is validated by the validation engine (ADR-0004), and its usage is recorded in the provenance ledger (ADR-0011). This makes recipes auditable: you can trace which library was used, which recipe it came from, and which version was active at build time.

## Consequences

### What this enables
- **Enterprise-grade output from any agent.** An agent using backstop libraries produces integrations with proper error handling, observability, retry logic, and configuration — not because the agent is smart, but because the library is.
- **Consistency across projects and languages.** Every project using `backstop-go/http` gets the same HTTP server pattern. Every project using `backstop-ts/postgres` gets the same database access pattern. Standards are baked into the library, not enforced after the fact.
- **Backstop transitions from tool to infrastructure.** The standard library is the product moat. Developers adopt backstop not just for enforcement but for the library ecosystem that makes their agents dramatically more productive.
- **Recipe catalog enables agent discovery.** Agents can query the catalog to find available libraries before writing custom code. This closes the loop: the agent knows what's available, uses what exists, and only writes custom code for genuinely novel patterns.

### What this requires
- **Library maintenance is a core responsibility.** Each library must be tested, versioned, and updated. This is infrastructure work that scales with the number of supported languages and services.
- **Recipe quality determines library quality.** A poorly specified recipe produces a poorly implemented library. Recipe authoring becomes a high-leverage, high-stakes activity.
- **Agent prompt engineering for library preference.** The `useBackstopLibraries: true` flag must translate into effective agent instructions. If agents ignore the flag, the library goes unused.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Templates instead of libraries | Templates are copied and diverge. Libraries are imported and stay consistent. Templates solve the initial generation problem but not the maintenance problem. |
| Agent instructions only ("always use proper error handling") | Instructions are suggestions. Libraries are guarantees. An agent can ignore an instruction; it cannot ignore a library's implementation. |
| Third-party SDK wrappers without the recipe layer | Loses the single-source-of-truth property. Without recipes, each language wrapper is authored independently and inevitably diverges. |
| Backstop-specific metadata in library packages | Couples libraries to backstop, making them unusable outside the ecosystem. The CLI extracts metadata from code and tests instead, keeping libraries standalone. |

## References

- D-045, D-046, D-047, D-048, D-049, D-050: Recipe and library model decisions
- D-053: Universal SDK layer
- D-058: Auto-generated recipe catalog
- ADR-0002: Canonical artifact primitives — recipes as first-class artifacts
- ADR-0006: Standards packs — enforcement layer that validates library usage
- ADR-0010: Verification kill chain — how library-based code is verified
- ADR-0011: Provenance ledger — where library usage is recorded
