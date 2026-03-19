---
number: ADR-0007
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: D-073
schema_version: adr/v2
---

# ADR-0007: Security Standards Pack — ASVS-Mapped Enforcement

## Context

When an AI agent generates a CRM at 2am and it goes to production, the question isn't "does it work" — it's "is it secure." The agent doesn't know about OWASP. The agent doesn't think about SQL injection unless told to. The agent will use MD5 if it saw MD5 in training data.

Backstop's security pack answers this by mapping enforcement rules to OWASP ASVS (Application Security Verification Standard) — the industry standard for application security requirements. The goal: agent-generated code that is ASVS L2 compliant at the code level by default, without the agent needing to know anything about ASVS.

## Decision

### ASVS L2 as the default

ASVS defines three levels:
- **L1** — Opportunistic. Basic security hygiene. Appropriate for low-risk applications.
- **L2** — Standard. Due diligence for applications handling sensitive data. What the industry considers "good enough" for production.
- **L3** — Advanced. High-security applications, financial systems, healthcare.

The backstop-security pack defaults to **L2** across all categories. This is not the floor — it's the default. Teams can:
- **Downgrade to L1** for internal tools, prototypes, or apps behind VPNs with no PII
- **Upgrade to L3** for specific categories where code-level enforcement is achievable

Downgrading requires explicit opt-out in `backstop.yml`. The default does the right thing.

### ASVS category coverage

| ASVS Category | Pack Rules Cover | Example Checks |
|---------------|-----------------|----------------|
| V2 Authentication | Password hashing, session management, credential storage | No plaintext passwords, bcrypt/argon2 required, session invalidation on logout |
| V4 Access Control | RBAC patterns, authorization checks | No direct object references without authz middleware, role checks before data access |
| V5 Validation | Parameterized queries, output encoding, server-side validation | No string concatenation in SQL, context-aware output encoding, input validation on all endpoints |
| V6 Cryptography | Algorithm strength, key management | No MD5/SHA1 for passwords, no hardcoded keys, proper IV/nonce management |
| V7 Error Handling | Stack trace exposure, log sanitization | No stack traces in HTTP responses, no PII in log output, structured error types |
| V8 Data Protection | PII handling, input sanitization, secrets | No hardcoded credentials, sensitive fields marked, data classification patterns |
| V13 API Security | Endpoint auth, rate limiting, CORS | Auth middleware on all routes, rate limiting patterns, CORS allowlist configuration |

### L3 aspiration categories

Three categories where L3 is both achievable at the code level and high-impact:

- **V2 Authentication L3** — multi-factor patterns, credential rotation, anti-automation controls. Agents generating auth code should produce the best auth code, not the minimum.
- **V5 Validation L3** — comprehensive input validation, server-side validation for everything, context-aware output encoding. Pure code, highly checkable.
- **V6 Cryptography L3** — authenticated encryption, proper IV/nonce management, key derivation functions. No excuse for an agent to use weak crypto.

### What the pack does NOT cover

Not all ASVS requirements are statically checkable. The security pack covers the **code-level slice** (~60-70% of L2):

| Layer | Who Covers It |
|-------|---------------|
| Code patterns (injection, auth, crypto) | **backstop-security pack** |
| Runtime behavior (rate limiting works, sessions expire) | CI integration tests |
| Infrastructure (TLS configured, secrets in vault) | Deployment review, infrastructure-as-code |
| Operational (incident response, penetration testing) | Organizational process |

The claim is precise: "Agent-generated code is ASVS L2 compliant at the code level." Combined with CI and infrastructure review, the full application meets L2.

### Configuration

```yaml
enforcement:
  packs:
    - backstop-security:
        asvs_level: 2              # default
        overrides:
          v2_auth: 3               # upgrade to L3
          v5_validation: 3         # upgrade to L3
          v6_crypto: 3             # upgrade to L3
          v9_communications: 1     # downgrade — internal service, no PII
```

### Interaction with standard library

The standard library amplifies security enforcement. When `useBackstopLibraries: true`:

- `backstop-go/http` uses proper TLS configuration by default
- `backstop-go/auth` hashes with bcrypt/argon2, never MD5
- `backstop-go/db` parameterizes all queries, never concatenates
- `backstop-go/log` uses structured logging with automatic PII redaction

The agent reaches for the library, gets L2 for free, and would have to actively work harder to be insecure. Make the right thing the easy thing.

## Consequences

### What this enables
- **Provably secure agent output.** Not "we hope the agent wrote secure code" — "the code passes ASVS L2 checks mechanically."
- **Configurable security posture.** L1 for prototypes, L2 for production, L3 for high-security categories.
- **Compliance-ready.** ASVS mapping gives auditors a recognized framework to evaluate against.
- **The 2am CRM pitch.** "My agent made me a CRM last night and it's more robust than 80% of enterprise codebases."

### What this requires
- **Rule authoring per ASVS requirement.** Each checkable requirement needs a pack rule, fixtures, and a checker.
- **Per-language implementation.** SQL injection checks in Go look different from Python. Each language pack needs its own security sub-component.
- **Honest scope claim.** The pack covers code-level requirements. Runtime and infrastructure coverage requires additional tooling.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Custom security rules (not mapped to ASVS) | Loses the industry-standard mapping. Auditors can't evaluate against a recognized framework. Reinventing the wheel. |
| L1 as default | L1 is "you tried." Not sufficient for applications handling any sensitive data. L2 is where due diligence starts. |
| L3 everywhere | Many L3 requirements are infrastructure-level (certificate pinning, HSM key storage). Code-level enforcement of those is impossible. Target L3 only where achievable. |
| Security as a separate tool (not a pack) | Breaks the composable pack model. Security should be toggleable and configurable like any other sub-component, not a separate product. |

## References

- D-073: Security standards pack with ASVS L2 default, L1–L3 configurable, per-category overrides
- OWASP ASVS v4.0: https://owasp.org/www-project-application-security-verification-standard/
- ADR-0006: Standards packs (the pack model this extends)
- ADR-0005: backstop.yml manifest (where security pack configuration lives)
- ADR-0013: Standard library model (forthcoming — how libraries amplify security)
