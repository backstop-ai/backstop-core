# backstop-core — working invariants

Load-bearing rules. Tags in `[brackets]` name the hook that enforces one; untagged
rules are vibes — honor them. This complements the auto-memory (MEMORY.md), it does
not repeat it. Keep this file short.

## Artifact workflow
- **Two tracks only:** `issue → plan` (reactive) OR `bundle → spec → plan → implementation`
  (feature). Specs are NEVER standalone — always from a bundle. Issues NEVER get specs.
- **Never hand-edit artifacts.** Route all artifact authoring/evolution to the purpose-built
  agents via the slash commands (`/bundle`, `/spec`, `/plan`, `/implement`). Hand-editing
  freeform prose drifts from the schema. [agent-guard + validate-artifact]
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

## Tooling
- CLI: `./bin/backstop`. Validate artifacts: `backstop artifact validate`. Gate: `backstop gate`.
- This Claude Code setup is a stopgap to unblock the opencode runtime — optimize for
  watch/steer and fewer interventions, not for determinism or auditability.
