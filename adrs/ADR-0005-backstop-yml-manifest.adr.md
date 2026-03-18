# ADR-0005: Backstop.yml — Namespaced Manifest With Incremental Ceremony

**Number:** ADR-0005
**Created:** 2026-03-18
**Status:** Accepted
**Deciders:** @bmanson
**Decisions:** D-043, D-064, D-063, D-048, D-060, D-070

---

## Context

Every project using backstop needs a single entry point that declares: what enforcement applies, how much ceremony is required, which runtimes are targeted, and (for monorepos) how services differ. This is `backstop.yml` — the manifest.

The manifest must serve two audiences simultaneously:
- **Teams that want the full ceremony** (specs, plans, ledgers, independent review) need knobs for the complete pipeline
- **Teams that just want enforcement** (agent generates code, backstop catches violations) need a minimal config that doesn't force them through the ceremony

The key insight from D-060: the minimum viable path through backstop is code gen with enforcement. Everything else is optional. The manifest must reflect this — enforcement is the floor, ceremony is the dial.

## Decision

### Four namespaced sections

```yaml
version: 2

project:
  name: my-platform
  runtimes:
    - copilot-cli
    - claude-code

enforcement:
  packs:
    - backstop-go
    - backstop-security:
        asvs_level: 2
        overrides:
          v2_auth: 3
  useBackstopLibraries: true
  allow_open_waivers: false        # default: waivers require expiry

workflow:
  require_specs: true
  require_plans: true
  ledger: true
  review_policy: backstop-only     # backstop-only | human-required
  concurrency: serialize-on-overlap

services:                           # optional — monorepo only
  api-gateway:
    path: services/api-gateway
    language: go
    packs:
      - backstop-go
      - backstop-go-api
  web-frontend:
    path: services/web-frontend
    language: typescript
    packs:
      - backstop-typescript
      - backstop-react
    runtimes:
      - claude-code               # override: this team uses Claude
```

### Section responsibilities

**`project:`** — identity and runtime targeting
- `name`: project identifier (used by hosted layer when it exists)
- `runtimes`: list of agent runtimes to generate hooks for. `backstop init` wires up all declared runtimes. Not mutually exclusive — a team can target both Copilot CLI and Claude Code.

**`enforcement:`** — what rules apply
- `packs`: list of standards packs with optional per-pack configuration (ASVS levels, sub-component toggles)
- `useBackstopLibraries`: when true, agents prioritize backstop standard libraries when generating code
- `allow_open_waivers`: whether waivers without expiry dates are permitted (default: false)

**`workflow:`** — how much ceremony is required
- `require_specs`: specs must exist before implementation begins (default: false for enforcement-only)
- `require_plans`: plans must exist before implementation begins (default: false)
- `ledger`: provenance ledger tracking on/off (default: false)
- `review_policy`: `backstop-only` (no human review required) or `human-required`
- `concurrency`: `serialize-on-overlap` (default) or `allow-parallel` for plan-level file conflict handling

**`services:`** — monorepo support (optional)
- Each service declares its path, language, and any overrides to global config
- Services inherit all global settings. Only declare overrides.
- Omit entirely for single-service repos — no penalty, no boilerplate

### The ceremony dial

The `workflow:` section is the "opinion dial." Three profiles:

**Enforcement only** (the floor):
```yaml
workflow:
  require_specs: false
  require_plans: false
  ledger: false
  review_policy: backstop-only
```

**Standard** (recommended):
```yaml
workflow:
  require_specs: true
  require_plans: true
  ledger: true
  review_policy: backstop-only
```

**Full ceremony** (enterprise/regulated):
```yaml
workflow:
  require_specs: true
  require_plans: true
  ledger: true
  review_policy: human-required
```

### Schema evolution

New fields are additive with sensible defaults. Old configs work without them (D-070). Deprecated fields emit warnings with a decommission version. Breaking removals happen only on major version bumps. `backstop migrate` exists as a convenience but is never required.

### Language detection

Languages can be explicitly declared (`language: go`) or inferred from the presence of `go.mod`, `package.json`, `requirements.txt`, etc. Explicit declaration overrides inference. In monorepos, language is typically declared per-service.

## Consequences

### What this enables
- **Incremental adoption.** Start with enforcement-only (3 lines of YAML), graduate to full ceremony as trust builds.
- **Monorepo support.** Per-service overrides with global inheritance. No config duplication.
- **Multi-runtime teams.** Target both Copilot CLI and Claude Code simultaneously.
- **Configurable security.** ASVS levels, sub-component toggles, waiver policies — all in the manifest.

### What this requires
- **`backstop init` must be smart.** It detects language, suggests packs, generates a reasonable default manifest.
- **Validation of the manifest itself.** `backstop validate` checks backstop.yml against its own schema before validating any artifacts.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Flat top-level keys | Doesn't naturally support monorepo. Namespacing enables per-section inheritance and overrides. |
| Minimal manifest with everything inferred | Some settings (review policy, ceremony level) cannot be inferred. Explicit is better than magic. |
| Multiple config files | One file is simpler. Monorepo complexity is handled by the services block, not by file proliferation. |
| No manifest — convention only | Breaks agent-first design. Agents need a machine-readable declaration of what to enforce. |

## References

- D-043: Consuming repos declare backstop.yml manifest
- D-064: Namespaced backstop.yml schema (project, enforcement, workflow, services)
- D-063: compile_target renamed to runtimes (list)
- D-048: useBackstopLibraries flag
- D-060: MVP is enforcement-only — workflow section enables incremental ceremony
- D-070: Backward-compatible schema evolution
- D-075: Waiver files with configurable expiry requirements
- ADR-0004: Validation engine (reads the manifest as its first action)
