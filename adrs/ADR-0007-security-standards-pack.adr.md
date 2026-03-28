---
number: ADR-0007
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-073, D-087, D-088"
schema_version: adr/v2
---

# ADR-0007: Security Standards — Tiered Compliance Enforcement

## Context

When an AI agent generates a CRM at 2am and it goes to production, the question isn't "does it work" — it's "is it secure." The agent doesn't know about OWASP. The agent doesn't think about SQL injection unless told to. The agent will use MD5 if it saw MD5 in training data.

Meanwhile, companies pay tens of thousands of dollars for a consultant to send them an Excel sheet with a bunch of mean notes about their security posture. What if that entire process — baseline assessment, remediation planning, verification, and audit trail — was mechanical?

Backstop's security enforcement answers this with a three-tier model that scales from "don't get hacked" to "the auditor just left happy." The tiers are additive — each higher tier includes everything below it. Semgrep OSS provides the checker engine (ADR-0006); backstop curates the rule sets and maps them to compliance frameworks.

## Decision

### Three security tiers (D-087)

| Tier | Framework mapping | What it enforces | Who it's for |
|------|------------------|------------------|--------------|
| **baseline** | CWE Top 25 + OWASP Top 10 | Don't ship known vulnerabilities | Side projects, MVPs, early startups — "don't look like a stooge" |
| **standard** | ASVS L2 (code-level slice) | Industry due diligence for production software | Production SaaS, teams that ship to real users |
| **compliance** | SOC 2 trust criteria (code-level) | Audit-ready engineering with machine-verifiable evidence | Enterprise, regulated industries, "the auditor is coming" |

Each tier is a strict superset of the one below. Compliance includes all Standard rules. Standard includes all Baseline rules. There is no skipping.

### Configuration

```yaml
enforcement:
  security:
    tier: standard          # baseline | standard | compliance
    overrides:
      v2_auth: compliance   # upgrade auth to compliance tier
      v6_crypto: compliance # upgrade crypto to compliance tier
```

The default tier is **standard**. This is not the floor — it's the right default for production software. Teams explicitly opt down to baseline (prototypes, internal tools) or up to compliance (regulated, enterprise).

### Tier: Baseline — CWE + OWASP (D-073, revised)

Baseline uses semgrep's built-in rules tagged with CWE and OWASP identifiers. These ship with semgrep — backstop enables them with zero custom rule authoring:

- **~300+ rules per language** — security, correctness, best practices
- **CWE Top 25** — the most dangerous software weaknesses
- **OWASP Top 10** — the most critical web application security risks
- Covers: injection, broken auth, sensitive data exposure, XXE, broken access control, security misconfiguration, XSS, insecure deserialization, known vulnerabilities, insufficient logging

This is the "don't get hacked" tier. If your code fails baseline, you have serious problems.

### Tier: Standard — ASVS L2

Standard adds backstop-authored semgrep rules mapped to ASVS L2 requirements. These are custom rules with explicit ASVS metadata:

```yaml
metadata:
  backstop:
    tier: standard
    asvs_category: v5_validation
    asvs_requirement: "5.3.4"
    enforcement: semgrep    # or: verifier, both
```

ASVS L2 coverage by category:

| ASVS Category | Enforcement mechanism | Example checks |
|---------------|----------------------|----------------|
| V2 Authentication | Semgrep + verifier | bcrypt/argon2 required, session invalidation, no plaintext passwords |
| V4 Access Control | Semgrep | Authorization middleware on routes, no direct object refs without authz |
| V5 Validation | Semgrep | Parameterized queries, server-side validation, context-aware encoding |
| V6 Cryptography | Semgrep | No MD5/SHA1 for passwords, no hardcoded keys, proper IV/nonce |
| V7 Error Handling | Semgrep | No stack traces in responses, no PII in logs, structured error types |
| V8 Data Protection | Semgrep + verifier | No hardcoded credentials, sensitive field annotations, data classification |
| V13 API Security | Semgrep | Auth middleware on all routes, CORS allowlist, rate limiting middleware |

Backstop maintains a CWE→ASVS mapping for semgrep's built-in rules, plus custom rules for ASVS requirements that have no CWE equivalent but are pattern-matchable (approved crypto libraries, salt usage patterns, deprecated API bans).

### Tier: Compliance — SOC 2 Trust Criteria

Compliance extends Standard with rules mapped to SOC 2's five trust service criteria. This is where backstop goes beyond what any static analysis tool offers — enforcing the code-level slice of SOC 2 across all five criteria:

| SOC 2 Criterion | What backstop enforces | Enforcement mechanism |
|-----------------|----------------------|----------------------|
| **Security** | Everything in Standard tier, plus stricter controls | Semgrep + verifier |
| **Availability** | Health check endpoints exist, graceful shutdown handlers, circuit breakers, timeout declarations | Semgrep (pattern presence) |
| **Processing Integrity** | Input validation on all endpoints, idempotency keys on mutations, audit logging on state changes | Semgrep + verifier |
| **Confidentiality** | Data classification annotations, PII field markers, encryption-at-rest patterns, access logging | Semgrep + verifier |
| **Privacy** | PII handling patterns, data retention declarations, consent check middleware | Semgrep (code-level only) |

**Honest scope claim:** Backstop enforces the code-level controls for SOC 2 — estimated at 50-60% of what an auditor evaluates. The remaining 40% is organizational: access reviews, employee onboarding, incident response plans, vendor assessments, physical security. Backstop doesn't write your HR policies. But it handles the engineering half with a machine-verifiable audit trail instead of screenshots and hope.

### The compliance workflow (D-088)

For existing codebases adopting compliance tier:

1. **`backstop baseline --tier compliance`** — scan codebase, catalog all violations against compliance rules
2. **Backstop generates issues** — each violation becomes a tracked issue with full traceability (REQ → CLM → tests)
3. **Agents burn down issues** — mechanical verification at every step, progressive enforcement from ready onward
4. **`backstop validate` goes green** — auditable trail of what was fixed, when, by whom, with proof it works
5. **Hand the auditor the ledger** — not an Excel sheet

For greenfield projects: set `tier: compliance` on day one. Every line of code ships compliant because the gates won't let non-compliant code through. When the auditor shows up, you've been compliant since commit one.

### L3 aspiration categories

Within the standard tier, three ASVS categories support L3 upgrade via overrides — areas where L3 is both achievable at the code level and high-impact:

- **V2 Authentication L3** — multi-factor patterns, credential rotation, anti-automation controls
- **V5 Validation L3** — comprehensive input validation, server-side validation for everything
- **V6 Cryptography L3** — authenticated encryption, proper IV/nonce management, key derivation functions

### Interaction with standard library

The standard library amplifies security enforcement. When `useBackstopLibraries: true`:

- `backstop-go/http` uses proper TLS configuration by default
- `backstop-go/auth` hashes with bcrypt/argon2, never MD5
- `backstop-go/db` parameterizes all queries, never concatenates
- `backstop-go/log` uses structured logging with automatic PII redaction

The agent reaches for the library, gets the configured tier for free, and would have to actively work harder to be non-compliant. Make the right thing the easy thing.

## Consequences

### What this enables
- **Provably secure agent output.** Not "we hope the agent wrote secure code" — "the code passes mechanical compliance checks."
- **Progressive security posture.** Start at baseline, ratchet to standard when you ship to users, upgrade to compliance when the auditor calls.
- **The 2am CRM pitch.** "My agent made me a CRM last night and it's more robust than 80% of enterprise codebases."
- **The founder pitch.** "I've been SOC 2 compliant at the code level since day one. Here's the ledger."
- **The enterprise pitch.** "We baselined 800 violations, burned them down to 20 in two days, and have a machine-verified audit trail for every fix."
- **Compliance-ready.** ASVS and SOC 2 mappings give auditors recognized frameworks to evaluate against.

### What this requires
- **Rule authoring per tier per framework requirement.** Custom semgrep rules for ASVS and SOC 2 requirements not covered by built-in rules.
- **Per-language rule variants.** SQL injection patterns in Go look different from Python. Each language pack's security category needs its own rules — but they're YAML, not compiled checkers.
- **CWE→ASVS mapping maintenance.** A lookup table mapping semgrep's built-in CWE-tagged rules to ASVS categories and tiers.
- **Honest scope claim.** Code-level controls only. Runtime, infrastructure, and organizational controls require additional tooling and process.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| ASVS only (no SOC 2 mapping) | Limits the compliance story to security practitioners. SOC 2 is what executives and auditors speak. |
| L1 as default | L1 is "you tried." Not sufficient for applications handling any sensitive data. Standard (L2) is where due diligence starts. |
| L3 everywhere | Many L3 requirements are infrastructure-level (certificate pinning, HSM key storage). Code-level enforcement of those is impossible. Target L3 only where achievable via overrides. |
| Security as a separate tool (not a pack) | Breaks the composable pack model. Security should be toggleable and configurable like any other category, not a separate product. |
| Semgrep Pro for cross-file taint analysis | Adds commercial dependency. OSS covers pattern matching; backstop verifier handles cross-file. |

## References

- D-073: Security standards with tiered enforcement (revised from ASVS-only)
- D-087: Three security tiers — baseline, standard, compliance
- D-088: Compliance workflow — baseline scan → issue generation → burn down → audit trail
- OWASP ASVS v4.0: https://owasp.org/www-project-application-security-verification-standard/
- CWE Top 25: https://cwe.mitre.org/top25/
- SOC 2 Trust Services Criteria: https://www.aicpa.org/soc2
- ADR-0006: Standards packs (semgrep-powered enforcement engine)
- ADR-0005: backstop.yml manifest (where security configuration lives)
- ADR-0013: Standard library model (how libraries amplify security)
