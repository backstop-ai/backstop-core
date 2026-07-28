---
name: hermetic-pack-fixture-recipe
description: How to build a pack fixture that passes/fails packval with ZERO tool execution (claimless engine-model rule; rule-file id mismatch flips test-only)
metadata:
  type: project
---

Recipe for hermetic pack fixtures (packval runs no external tool):

- **Passes check AND test, executes nothing:** one rule with `engine:` declared in the
  pack's own `engines:` block, a `file:` whose YAML `rules:` list contains the SAME id,
  `risk_class`, and **no claims**. Claimless is legal exactly when the declared engine
  RESOLVES (phase2 + phase5 exemptions), and phase3 only executes per-CLAIM — so adding
  claims is what drags a real engine in. Give the engine `command: ""` so any future
  routing to execution fails loud instead of silently reaching for a binary.
- **Passes check, FAILS test (fixture phase only):** same pack, rule file declares a
  DIFFERENT id → `phase3-fixtures/semgrep-rule-id: pack rule ID "X" not found in rule
  file <path>`. One-bit-apart pair, no tool.
- **Fails check but the MANIFEST LOADER accepts it** (so only the validator can reject
  it — the wiring proof): declare `file:` pointing at a file that does not exist →
  `phase1-structural/file-exists: referenced file not found`. `pkg/pack.ParseManifest`
  never stats rule files and defers unknown engines to gate time, so it passes.
- `pkg/pack.ParseManifest` additionally requires `description`, a non-empty `engine`
  per rule, and a valid `risk_class` — packval's phase1 does not.

**Why:** SPEC-055's E2E revalidates the fixture pack on every `pack add`; a fixture that
executed semgrep/ast-grep would need network + provisioning and would flake.

**How to apply:** reach for this whenever a test needs a pack that must pass or fail
validation deterministically. See [[project_pack_copies_and_stale_gate_binary]].
