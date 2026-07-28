# backstop-core — working invariants

Load-bearing rules. Tags in `[brackets]` name the hook that enforces one; untagged
rules are vibes — honor them. This complements the auto-memory (MEMORY.md), it does
not repeat it. Keep this file short.

## What backstop IS (first principle — supersedes any conflicting instinct)
- **Thin executor: ZERO baked language/tool knowledge, for ANY language.** Backstop bakes in
  no language- or tool-specific checks, routing, defaults, or toolchain. Every check and every
  language's toolchain comes from a PACK; backstop only runs what packs declare (allowlisted
  commands) and speaks SARIF. There is no native/legacy tier and no "built-in" language — a
  baked Go path AND a baked TypeScript path are BOTH violations. New language = a new pack,
  never new CLI code.
- **A baked language/tool branch is a defect to eradicate, never to extend.** Never scope work
  as "add language X support" (that bakes another language) — scope it as "remove the
  assumption; it comes from a pack." Acceptance: backstop gates a project in ANY language,
  packs-only, going RED when it should — no baked path, no vacuous green. Do not re-litigate
  this or ask whether a legacy/baked check "stays"; only ask how/when to migrate it. (Aspires
  to be gate-enforced — a dogfooded rule failing on baked language/tool literals in the gate
  path; until that exists, it is still law.) See MEMORY: [[feedback_zero_baked_checks]].

## Packs (external by design — do not re-litigate)
- **Packs live OUTSIDE core; the lock file is the durability boundary.** Every pack lives in
  its own repo and installs into gitignored `.backstop/packs/` (like `node_modules`).
  `backstop.lock` (tracked) is the durable record — you reinstall packs as needed. Gitignored
  or absent pack content is **NOT** a durability gap, a "dark gate", or a defect. Never frame
  external/uncommitted packs as a problem, never propose vendoring them into core, and never
  treat "not committed" as "not durable." See MEMORY: [[feedback_packs_always_external]].

## Artifact workflow
- **Two tracks only:** `issue → plan` (reactive) OR `bundle → spec → plan → implementation`
  (feature). Specs are NEVER standalone — always from a bundle. Issues NEVER get specs.
- **Never hand-edit artifacts.** Route all artifact authoring/evolution to the purpose-built
  agents via the slash commands (`/bundle`, `/spec`, `/plan`, `/implement`). Hand-editing
  freeform prose drifts from the schema. [agent-guard + validate-artifact]
- **No implementation without a validated plan.** "Fix ISSUE-NNN" means *plan it first*:
  `/plan` from the issue, validate, then `/implement` against the plan. A session editing
  source with no in-flight plan artifact is off-track — regardless of how small the fix
  looks. (Aspires to be hook-enforced; until then, it is still law.)
- **Bundles start at `exploring` with real open questions.** The user drives OQ resolution
  AND promotion — do NOT pre-resolve OQs or self-promote maturity.
- **Promotion to `defined`+ is structural:** needs Draft Requirements / Draft Design Decisions /
  Spec Seeds / Version History sections + a `requirements[]` array + `solution.approach`.
  Prose alone won't validate. [validate-artifact]

## Behavior
- **Answering a scoping question is NOT a go.** Collect the decision, stop, wait for an
  explicit instruction to start. The user has been burned by agents hauling off prematurely.
- **Verify, don't assert.** Never claim something validates / passes / works without running
  `./bin/backstop` (or the relevant command) and reading the result.
- **No false grounding.** Don't assert time / file / user-state / progress you weren't given — ask.

## Enforcement philosophy (when building checks)
- **Loud ≠ blocking.** Block defects + broken promises; warn-with-guidance for un-adopted
  capability. The enemy is silent/vacuous green, not passing.
- **Dogfood rules as packs.** Never bake check logic into the CLI binary; backstop consumes
  its own rules as a pack like any project.

## Codebase map (auto-loaded — where everything lives)
@docs/CODEBASE-MAP.md

## Tooling
- **Cross-session/agent questions:** any Bash-capable agent may fork-interview any
  session — protocol in `~/.claude/skills/consult/SKILL.md` (identity, context,
  numbered questions, output contract, do-NOT-implement guard). Multi-party
  alignment: the /mediate skill. Answers are claims-with-citations, not facts.
- CLI: `./bin/backstop`. Validate artifacts: `backstop artifact validate`. Gate: `backstop gate`.
- This Claude Code setup is a stopgap to unblock the opencode runtime — optimize for
  watch/steer and fewer interventions, not for determinism or auditability.
