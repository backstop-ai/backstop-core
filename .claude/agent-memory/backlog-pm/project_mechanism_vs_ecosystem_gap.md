---
name: mechanism-vs-ecosystem-gap
description: Recurring pattern — a capability lands green in core while zero packs or consuming projects actually use it; verify launch readiness against the fleet (pack repos, portal, stash, harness), never against backstop-core alone
metadata:
  type: project
---

Backstop capabilities repeatedly reach `implemented` in core with a green
gate while **nothing in the fleet consumes them**. Verified 2026-07-26 on
both launch-blocking capabilities at once: SPEC-054 delivered the recipe
apply mechanism, yet no `pack.yml` anywhere declares a `recipes:` block and
core's own `recipes/{go,meta,typescript}` hold only `.gitkeep`; SPEC-055
delivered production remote pack assembly, yet every lock entry across
backstop-core, bclabs-portal, stash, and backstop-harness is
`source_type: local` pointing at a sibling directory.

**Why:** core's gate can only prove core's own claims. A spec's acceptance
tests are hermetic by design (SPEC-055's remote E2E deliberately never
touches the network), so the pack-side and consumer-side halves of a
capability are invisible to it. The bundles know this — BUNDLE-015 REQ-018
names the CI recipe pack as "the packs-only acceptance test" — but that REQ
sits in a later, unbuilt seed, so the acceptance test lands after the thing
it is meant to accept.

**How to apply:** before calling any capability launch-ready, check the
consuming side directly — `grep` the pack repos under `~/src/projects/` for
the declaration the capability needs, and read the `backstop.lock` of every
consuming project (bclabs-portal, stash, backstop-harness) to see which code
path is actually exercised. Report mechanism-complete and ecosystem-complete
as two separate verdicts. Feeds [[launch-tiering]]: a tier-1 capability is
not done at spec-implemented, it is done at first real consumer.

**Verification caveat, learned the hard way (2026-07-26):** check each pack
DIRECTORY for `.git`, never the parent that contains them. I reported that
`~/src/projects/backstop-packs` "is not a git repository, so the TypeScript
suite cannot be published without being split into five repos first" — the
parent indeed has no `.git`, but all five child packs were already
individual repos with `backstop-ai` remotes. The observation was right and
the conclusion was wrong. Iterate over `*/` and test `$d/.git` per pack.

**Resolved 2026-07-26 (same day):** ten packs published under
`github.com/backstop-ai` with `name == coordinate` as the ratified
convention, each proven by a real remote `pack add`. The ecosystem gap for
remote consumption is closed; the recipe half is not — still zero
`recipes:` blocks fleet-wide. DIR-027 owns the residual fleet migration.
