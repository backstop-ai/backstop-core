---
number: ADR-0017
created: "2026-03-28"
status: Accepted
deciders: "@bmanson"
decisions: "D-089, D-090, D-091, D-092, D-093, D-094, D-095, D-096, D-097, D-098, D-099"
schema_version: adr/v2
---

# ADR-0017: Pack Registry and Plugin Architecture

## Context

Backstop's enforcement engine runs on packs — rule packs (semgrep YAML) and code packs (recipe-generated code). Without a distribution model, packs are embedded in the CLI binary or copied around as files. This limits adoption to what the core team authors and makes private/proprietary packs impossible.

The ecosystem needs three things: a way to discover packs, a way to install them, and a way to publish them. This is a solved problem — npm, Go modules, Cargo, and PyPI all demonstrate the model. Backstop applies it to enforcement rules and standard libraries.

## Decision

### Two pack types (D-089)

| Type | What it contains | Authored as | Compiled to | Distributed via |
|------|-----------------|-------------|-------------|-----------------|
| **Rule pack** | Semgrep YAML rules + testdata fixtures | Markdown + YAML frontmatter | Semgrep rule files | Backstop registry (hosted) |
| **Code pack** | Importable code/SDK implementing a recipe | Markdown + YAML frontmatter (recipe artifact) | Language-native packages | Native package manager (npm, Go modules, PyPI), cataloged in backstop registry |

Both types follow the same authoring primitive: a backstop artifact (markdown + YAML frontmatter) that defines requirements, claims, contracts, and tests. The artifact is the spec; the pack contents are the implementation.

### Manifest structure (D-090)

Packs are declared in `backstop.yml` with explicit type separation:

```yaml
packs:
  rules:
    - "@backstop/go-security@2.1.0"
    - "@backstop/go-core@1.0.0"
    - "@acme/go-fintech@1.3.0"
  code:
    - "@backstop/go-http@2.0.0"
    - "@backstop/ts-auth@1.1.0"
```

The type is determined by where it's declared, not by the name. An agent reading this manifest immediately understands: rules enforce, code enables.

### Namespaced identifiers (D-091)

Every pack is namespaced with `@scope/pack-name`, following the npm convention:

```
@backstop/go-security        # official backstop rule pack
@acme/go-fintech             # Acme's proprietary rules
@trailofbits/go-crypto       # community contributor
@bmanson/go-clean-code       # individual author
```

The `@backstop` namespace is reserved for official packs. Unscoped references like `go:security` in rule qualifications (`go:security@2.0.0:SEC-0012`) are shorthand for `@backstop/go-security`.

### Registry tiers (D-092)

| Tier | What | Model |
|------|------|-------|
| **Public** | Community and official packs, free to publish and consume | Free — ecosystem growth |
| **Team** | Private registry tenant for proprietary packs | SaaS subscription |
| **Enterprise** | Self-hosted registry, air-gapped, SSO, audit logs | Enterprise license |

Scope-based registry resolution in backstop.yml:

```yaml
registries:
  "@acme": https://registry.acme.com/backstop
  default: https://registry.backstop.dev
```

The CLI resolves `@acme/*` against Acme's private tenant and everything else against the public registry.

### Registry as verification service (D-093)

The registry is not just a file server. When a pack is published, the registry runs backstop's own gates before listing it:

1. **Validate the artifact** — schema, requirements, claims, contracts all pass
2. **Verify the implementation** — test functions exist per claims, contract signatures match
3. **Run the tests** — coverage threshold met
4. **Run rule packs against the code** — code packs must pass the same rules they help others comply with
5. **Sign and publish** — the registry stamps it as verified

This is backstop eating its own dog food. The verification gates are compute — free for public packs, paid for private tenants.

### Code pack distribution model

Code packs are cataloged in the backstop registry but distributed through native package managers:

```
@backstop/go-http     → listed in backstop registry, code lives on Go modules
@backstop/ts-auth     → listed in backstop registry, code lives on npm
```

The registry holds metadata: description, compatibility, which rule packs it pairs with, which compliance tier it supports, recipe artifact. The actual code installs through the native package manager.

```bash
backstop pack add @backstop/go-http    # detects Go project, adds to go.mod
backstop pack add @backstop/go-security  # pulls YAML rules into .backstop/rules/
```

One command, two distribution channels. The CLI does the right thing based on pack type.

### Polyglot code packs

Code packs can contain implementations for multiple languages from a single recipe artifact:

```
@backstop/http/
  recipe.md              ← language-agnostic requirements
  go/
    server.go
    server_test.go
    go.mod
  ts/
    server.ts
    server.test.ts
    package.json
  backstop.yml
```

Or authors can distribute language-specific packs independently:

```
@backstop/go-http        ← Go only, versions independently
@somedev/go-retry        ← community author, single language
```

Both models coexist. Polyglot packs share one recipe artifact with multiple implementations verified against the same requirements. Language-specific packs version independently with lower maintenance overhead. Author's choice.

### Code pack contents

A code pack published to the registry contains the full provable unit:

```
@backstop/go-http/
  recipe.md              ← the artifact (frontmatter + requirements + claims)
  go/
    server.go            ← the implementation
    server_test.go       ← the tests (mandated by claims)
    go.mod
  backstop.yml           ← gates config (tier, coverage threshold)
```

### Severity overrides are project-only (D-095)

Packs declare rules with default severities. Only the consuming project's `backstop.yml` can override severity. Packs cannot override other packs' severities.

This eliminates cascading severity conflicts entirely. Consider: Pack A declares `SEC-0012` as ERROR. Pack B depends on A and wants it to be WARNING. Pack C depends on B and wants it back at ERROR. Pack F independently overrides B. Without this rule, the consumer inherits a tangled chain of severity overrides with no clear winner.

Instead: all packs see the severity declared at the source. The project author overrides in one place:

```yaml
packs:
  rules:
    - "@backstop/go-security@2.1.0"
    - "@acme/go-relaxed@1.0.0"
  overrides:
    "go:security:SEC-0012":
      severity: warning
```

Pack authors who want to *recommend* different severities for their audience can document it or ship a backstop.yml snippet. But they cannot force severity through the dependency graph.

### Registry-as-publisher for code packs (D-096)

Code packs are submitted to the backstop registry, which runs verification gates and publishes to native package managers on the author's behalf. The registry is the sole publisher and the authoritative source for content hashes.

The developer's workflow:

1. `backstop pack publish` — submit to backstop registry
2. Registry runs gates (tests, coverage, contracts, dependency audit)
3. Registry approves and publishes to npm/PyPI/Go modules
4. Lockfile receives the content hash

This closes the supply chain gap between "what was verified" and "what was distributed." If the registry published it, the checksum is authoritative. If the checksum doesn't match, the artifact isn't legitimate.

### Backstop-controlled native registry scopes (D-097)

Code packs publish under backstop-controlled scopes in native registries, preserving author identity:

| Native registry | Naming pattern |
|----------------|----------------|
| **npm** | `@backstop-registry/acme-auth-lib` |
| **PyPI** | `backstop-registry-acme-auth-lib` |
| **Go modules** | `github.com/backstop-registry/acme-auth-lib` |

The scope (`@backstop-registry`) signals provenance — this came through backstop's verified pipeline. The package name preserves the author's identity. Backstop holds the sole publish token with no risk of revocation.

The native registry name is an implementation detail. Consumers interact with `@acme/auth-lib` in backstop.yml and the CLI resolves it to the native registry name.

Authors with very long namespaces may encounter registry name length limits (npm caps at 214 chars). A soft guardrail warns at registration, but the backstop name is what consumers use day-to-day.

### Independent pack and implementation versioning (D-098)

Pack version and language implementation versions are independent:

| Entity | Version | What changed |
|--------|---------|-------------|
| Recipe pack: `@backstop-recipes/http` | `2.7.1` | TS implementation updated |
| Go impl: `@backstop-registry/backstop-go-http` | `1.1.10` | Unchanged since recipe 2.6.3 |
| TS impl: `@backstop-registry/backstop-ts-auth` | `1.3.1` | Updated in recipe 2.7.1 |
| Py impl: `@backstop-registry/backstop-py-http` | `1.2.0` | Unchanged since recipe 2.5.0 |

The **pack catalog version** revs on any subcomponent change. Each **language implementation** revs only when its actual bits change — code, tests, or documentation. The registry maintains a compatibility matrix mapping implementation versions to pack versions.

This avoids false signals: if Go didn't change, Go doesn't rev. Consumers can see at a glance which language triggered the pack version bump. The recipe version defines the behavioral contract; the recipe-derived test suite is the proof of compliance.

**Versioning rule:** if the bits in the published artifact changed, it revs. Code changes, test coverage improvements, documentation updates — all warrant a patch bump. Unchanged implementations stay at their current version.

### Offline bootstrapping with embedded core pack (D-099)

The CLI binary embeds a baseline rule pack via Go's `embed` directive. `backstop init` works fully offline with bundled rules — baseline enforcement (CWE Top 25, OWASP Top 10) is available immediately without network access.

- `backstop init` → works offline with embedded baseline rules
- `backstop pack sync` → pulls latest versions and additional packs when network is available
- Embedded version is a floor — registry always wins when available
- CLI updates bring updated embedded rules, so permanently offline environments improve over time

No registry dependency for the basic "don't let me ship something stupid" workflow. Packs, customization, and higher compliance tiers require network.

### Rule pack dependencies (D-094)

Rule packs can depend on other rule packs. A downstream pack declares its dependencies with minimum version and required rules:

```yaml
depends_on:
  "@backstop/go-security":
    min_version: "2.0.0"
    requires_rules:
      - SEC-0012
      - SEC-0018
```

Resolution follows Go's minimum version selection: the highest compatible version is used. There is one copy of each rule pack — no diamond dependency problem.

**Immutable rule IDs.** Once `SEC-0012` means "hardcoded credentials check," it always means that. If a fundamentally different check is needed, it gets a new ID.

**Breaking changes use deprecate-and-supersede.** A deprecated rule stays in the pack, still fires, but emits a warning pointing to its successor. The next major version can drop it. This gives downstream authors time to:

1. **Migrate** — adopt the new rule (SEC-0025 replaces SEC-0012)
2. **Fork** — re-implement the rule in their own pack as `@acme/go-fintech:FIN-0012`
3. **Do nothing** — deprecated rule keeps working until the next major version

The registry scans dependent packs when a new major version is published and flags breaking changes: "Dropping SEC-0012 will affect @acme/go-fintech."

### Lockfile

`backstop pack add` writes exact versions to `.backstop/pack-lock.yml`:

```yaml
# Auto-generated — do not edit
rules:
  "@backstop/go-security":
    version: "2.1.0"
    integrity: sha256-abc123...
  "@acme/go-fintech":
    version: "1.3.0"
    integrity: sha256-def456...
    depends_on:
      "@backstop/go-security": ">=2.0.0"
code:
  "@backstop/go-http":
    version: "2.0.0"
    integrity: sha256-ghi789...
    native_package: "github.com/backstop-dev/go-http@v2.0.0"
```

No floating versions. Deterministic across environments. The same lockfile produces the same enforcement regardless of when or where it runs.

### CLI commands

```bash
backstop pack add @backstop/go-security       # install a pack
backstop pack remove @acme/go-fintech         # remove a pack
backstop pack update @backstop/go-security    # update to latest compatible
backstop pack list                            # list installed packs
backstop pack search "go security"            # search the registry
backstop pack publish                         # publish to configured registry
backstop pack vendor                          # pull all packs into local directory (offline/air-gapped)
```

## Consequences

### What this enables
- **Ecosystem growth.** Anyone can author and publish rule packs and code packs. Enforcement is no longer limited to what the core team writes.
- **Private enforcement.** Enterprise teams distribute proprietary rules through private registry tenants without exposing them publicly.
- **Verified distribution.** Every pack in the registry is mechanically verified — backstop's own gates run before publish.
- **SaaS revenue.** Public registry is free (ecosystem). Private tenants are paid (SaaS). Self-hosted is enterprise licensed. Verification gates are compute.
- **Deterministic enforcement.** Lockfile + exact version pinning means the same rules run in dev and CI and production. No drift.
- **Polyglot standard libraries.** One recipe, N language implementations, verified against the same requirements.

### What this requires
- **Registry infrastructure.** The public registry needs to be built and operated — storage, CDN, verification compute, signing infrastructure.
- **Pack authoring tooling.** `backstop pack init`, scaffolding templates, local testing before publish.
- **Namespace governance.** Rules for claiming scopes, handling disputes, preventing squatting.
- **Offline story.** `backstop pack vendor` for air-gapped environments. CLI ships with embedded baseline rules (D-099).

## Open Questions (Resolved)

All open questions from the initial draft have been resolved:

- **Rule pack ordering** → D-095: Severity overrides are project-only. Packs cannot override other packs' severities. Project `backstop.yml` is the sole authority.
- **Code pack trust chain** → D-096 + D-097: Registry-as-publisher model. Backstop runs gates, publishes to native registries under backstop-controlled scopes, and is the authoritative source for content hashes.
- **Recipe/code pack version sync** → D-098: Pack catalog version and implementation versions are independent. Pack revs on any subcomponent change. Implementations rev only on actual changes. Registry maintains compatibility matrix.
- **Offline bootstrapping** → D-099: CLI embeds baseline rule pack via Go embed. `backstop init` works offline. Registry sync upgrades when network is available.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Packs embedded in CLI binary only | No ecosystem growth. Limits enforcement to core team output. Makes private packs impossible. |
| Packs distributed via native package managers only | No central discovery. No unified search. Rule packs (YAML) don't belong in npm. |
| Single pack type (rules and code mixed) | Conflates enforcement with enablement. Different distribution needs — rules are YAML served from our registry, code packs are code served from native package managers. |
| No dependencies between rule packs | Prevents composition and extension. Community can't build on official packs. |
| Floating/range versions | Non-deterministic. Same code could pass Monday, fail Tuesday. Lockfile with exact pins is mandatory. |

## References

- D-089: Two pack types — rules (semgrep YAML) and code packs (recipe-generated code)
- D-090: Manifest structure with explicit rules/code sections
- D-091: Namespaced pack identifiers (@scope/pack-name)
- D-092: Registry tiers — public, team (SaaS), enterprise (self-hosted)
- D-093: Registry as verification service — gates run before publish
- D-094: Rule pack dependencies with immutable IDs and deprecate-and-supersede
- D-095: Severity overrides are project-only — packs cannot override other packs' severities
- D-096: Registry-as-publisher — backstop publishes code packs to native registries on author's behalf
- D-097: Backstop-controlled native registry scopes (@backstop-registry/<author>-<pack>)
- D-098: Independent pack and implementation versioning with compatibility matrix
- D-099: Offline bootstrapping — CLI embeds baseline rule pack via Go embed
- ADR-0006: Standards packs (semgrep-powered enforcement engine)
- ADR-0007: Security standards (tiered compliance enforcement)
- ADR-0013: Standard library model (recipe-to-library pipeline)
