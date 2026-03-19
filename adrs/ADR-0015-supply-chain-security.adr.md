---
number: ADR-0015
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: D-074
schema_version: adr/v2
---

# ADR-0015: Supply Chain Security — Dependency Audit, SBOM, License Compliance

## Context

ADR-0007 (security standards pack) covers the code you write: no hardcoded secrets, no SQL injection, no insecure crypto. But modern applications are 80%+ third-party code by volume. A project can have pristine first-party code and still be compromised through a vulnerable dependency, a malicious package, or a license that contaminates the entire codebase.

Supply chain attacks are the fastest-growing attack vector in software. The xz utils backdoor, event-stream compromise, and ua-parser-js hijack all exploited the gap between "our code is secure" and "our dependencies are secure." Backstop's thesis (ADR-0001) claims that mechanical enforcement replaces human vigilance — that claim must extend to dependencies, not just source code.

This is a Tier 4 ADR because it requires the pack infrastructure (ADR-0006), the CI pipeline (ADR-0009), and the validation engine (ADR-0004) to be in place. Supply chain checks run as a CI action alongside validate and test, using the same enforcement machinery.

## Decision

### Dependency audit

The supply chain pack runs language-appropriate vulnerability scanners against the project's dependency tree:

- **Go** → `govulncheck` against the Go vulnerability database
- **Node.js** → `npm audit` or `yarn audit` against the npm advisory database
- **Python** → `pip-audit` or `safety` against the PyPI advisory database
- **Rust** → `cargo audit` against the RustSec advisory database

Scanners check the full transitive dependency tree, not just direct dependencies. A vulnerability in a dependency-of-a-dependency is still a vulnerability in your application. Results are normalized into backstop's violation format (severity, rule, message) and reported through the standard validation pipeline.

### Lockfile enforcement

Lockfiles must exist and be committed to version control. The supply chain pack validates:

- **Go** → `go.sum` exists and is committed
- **Node.js** → `package-lock.json` or `yarn.lock` exists and is committed
- **Python** → `requirements.txt` with pinned versions or `poetry.lock` exists and is committed
- **Rust** → `Cargo.lock` exists and is committed

A project without a lockfile has non-deterministic builds — the same `go get` or `npm install` can produce different dependency trees on different days. This is a validation failure, not a warning. Lockfiles are required, not recommended.

### SBOM generation

A Software Bill of Materials is generated at build time as part of the CI pipeline (ADR-0009). The SBOM is a machine-readable inventory of every dependency, its version, its license, and its source. The format follows CycloneDX or SPDX standards.

The SBOM is stored as a build artifact alongside the binary. It answers the question "what exactly is in this build?" with precision — not "whatever was in the lockfile at the time" but a verified, timestamped manifest.

### License compliance

backstop.yml declares allowed dependency licenses:

```yaml
supply_chain:
  allowed_licenses:
    - MIT
    - Apache-2.0
    - BSD-2-Clause
    - BSD-3-Clause
    - ISC
```

Dependencies with licenses not on the allowed list are flagged as violations. The default configuration permits only permissive licenses (MIT, Apache-2.0, BSD). Copyleft licenses (GPL, LGPL, AGPL, MPL) require explicit opt-in — teams must consciously choose to allow copyleft dependencies, not accidentally inherit them.

Unknown or missing licenses are treated as violations. A dependency without a declared license is not "probably fine" — it is legally ambiguous and must be resolved.

### Dependency pinning

All dependencies must be pinned to exact versions. Version ranges (`^1.2.3`, `~1.2.3`, `>=1.0.0`) are not permitted. The supply chain pack inspects manifests and lockfiles to verify exact pinning:

- **Go** → go.mod versions are inherently exact
- **Node.js** → package.json must use exact versions, not ranges
- **Python** → requirements.txt must use `==`, not `>=` or `~=`
- **Rust** → Cargo.toml must use `=` prefix for exact versions

Pinning eliminates the class of bugs where "it worked yesterday" because a dependency auto-updated. Combined with lockfile enforcement, this ensures that every build of a given commit produces identical output.

### Independent toggleability

Per D-016 (revised), the supply chain pack is independently toggleable. Teams can adopt code enforcement (ADR-0007) without supply chain enforcement and add it later. This supports incremental adoption — a team does not need to fix every dependency issue before getting value from code standards.

The supply chain pack is enabled in backstop.yml like any other pack:

```yaml
packs:
  - backstop/security
  - backstop/supply-chain  # independently toggleable
```

## Consequences

### What this enables
- **End-to-end security coverage.** ADR-0007 covers what you write; this ADR covers what you import. Together, they close the gap between "our code is secure" and "our application is secure."
- **License compliance as enforcement, not policy.** License requirements are mechanically checked, not left to developer awareness. A GPL dependency in a proprietary project is caught at CI time, not by a lawyer after release.
- **Reproducible builds.** Lockfile enforcement and dependency pinning guarantee that the same commit always produces the same dependency tree. Non-determinism in builds is a validation failure.
- **SBOM for audit and compliance.** Regulated industries require knowing exactly what is in a deployed artifact. The SBOM provides a machine-readable answer generated automatically at build time.

### What this requires
- **Language-specific scanner maintenance.** Each supported language needs a scanner integration. New languages require new scanner backends.
- **License database accuracy.** License detection depends on package registry metadata and heuristics. Some packages have ambiguous or missing license declarations.
- **Dependency update workflow.** Exact pinning means dependencies do not auto-update. Teams need a deliberate process for updating dependencies — backstop does not solve this, but the audit tooling surfaces when updates are available and whether they introduce vulnerabilities.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Vulnerability scanning only (no license, no pinning) | Addresses only one dimension of supply chain risk. License contamination and non-reproducible builds are equally dangerous. |
| Centralized dependency allowlist | Does not scale. Every project has different dependency needs. The license allowlist approach is more flexible — teams declare what license categories are acceptable. |
| Runtime dependency monitoring (SCA in production) | Complementary but out of scope. Backstop operates at build time. Runtime monitoring is a deployment concern that belongs in the operational stack, not the enforcement stack. |
| Vendoring all dependencies | Solves reproducibility but creates massive repository bloat and makes updates painful. Lockfile enforcement achieves the same reproducibility without the downsides. |

## References

- D-074: Supply chain security — dependency audit, SBOM, license compliance
- ADR-0006: Standards packs — the pack infrastructure supply chain checks plug into
- ADR-0007: Security standards pack — first-party code security (complementary)
- ADR-0009: CI/CD pipeline — where supply chain checks execute
- ADR-0004: Validation engine — the validation pipeline supply chain results flow through
