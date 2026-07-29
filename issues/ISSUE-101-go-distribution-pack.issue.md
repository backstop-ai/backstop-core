---
title: "Go Distribution Pack — Productize the Release Trinity"
schema_version: issue/v1

issue:
  id: ISSUE-101
  title: "Go Distribution Pack — Productize the Release Trinity"
  type: enhancement
  status: open
  created: "2026-07-28"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Go Distribution Pack — Productize the Release Trinity

## Problem

ISSUE-087 delivered a complete, real, tag-triggered release pipeline for backstop-core —
goreleaser config, a tag-driven release workflow, a tag-integrity workflow, version stamping,
and Homebrew formula publication — but it is hand-written by design (per that issue's explicit
scope) and lives only in this one repo. DIR-001's lane-close notes name the intended fate
directly: "The release trinity ISSUE-087 delivered... is hand-written today by design. It is
authored as a future payload of a `go-distribution` pack: founder framing captured 2026-07-27,
'that trinity fits perfectly as a go launch/go distribution pack.'" (`directives/DIR-001-release-workflow.directive.md`,
"Release-pipeline follow-on routings" section). That framing is consistent with this repo's
zero-baked-checks law — hand-baked release infrastructure is a defect to eradicate, never to
extend, and every check/toolchain belongs in a pack, not the CLI or a copy-pasted `.github/`
tree.

Today, any other Go CLI project that wants this exact pipeline (cross-platform goreleaser
builds, tag-integrity gating, Homebrew formula publication, version stamping that survives
both `go install` and a goreleaser-built binary) has no way to consume it except by re-deriving
or hand-copying backstop-core's `.goreleaser.yml` and workflow files — repeating, un-audited,
every hard-won lesson ISSUE-087/PLAN-ISSUE-087 already paid for in real failures on
2026-07-27/28. The founder's ask (2026-07-29) is to close that gap: create the
`backstop-ai/go-distribution` pack so any Go CLI project consumes ONE pack to get a complete,
verified release pipeline. First consumers: backstop-core itself (whose hand-written trinity
the pack's rules then gate — partial retirement of the hand-baked-CI debt class ISSUE-020's
acceptance criterion tracks, see the stale-citation correction below) and the founder's `stash`
CLI (`~/src/projects/stash`, already a backstop consumer with `backstop.yml` and `cmd/stash`).

**Stale-citation correction, recorded so it isn't re-derived incorrectly later.** Both
ISSUE-087 and DIR-001's lane-close notes cite "ISSUE-086" as the owner of the "core `ci.yml`'s
`gate` job hand-bakes Go tooling instead of calling `backstop gate`" gap. That citation predates
a 2026-07-27 founder ruling: ISSUE-086 was narrowed that day, splitting the hand-baked-CI fact
back out — it was "welded into ISSUE-020's acceptance criterion" instead (per ISSUE-086's own
"Split history" section, and ISSUE-020's founder ruling, "do not re-file this as a standalone
issue"). ISSUE-086 today covers a different, narrower defect (the published baseline artifact
generated with zero packs installed). This issue's citations below use ISSUE-020, the correct
current owner — the stale ISSUE-086 references in ISSUE-087/DIR-001 are a pre-existing drift in
those artifacts, not reproduced here, and are worth an independent correction pass.

## Scope

### Pack shape — two capabilities

**A. Recipes** (SPEC-054 machinery — recipe apply + manifest, delivered and E2E-proven).
Starter payloads a consumer applies then adapts, not enforce-as-is:

- `.goreleaser.yml` — the four-platform cross-compile + archives/checksums/Release config,
  including the `brews:` block.
- `.github/workflows/release.yml` — the tag-driven release workflow.
- `.github/workflows/tag-integrity.yml` — the tag-integrity gate workflow.
- A version-resolution Go starter (the `resolveVersion` pattern: ldflags → `ReadBuildInfo`
  module version → `"dev"` fallback, rejecting non-release build-info shapes).

Project-specific strings (binary name, org/repo, main package path) are consumer edits after
applying the recipe — the recipe is a starting point, not a templated black box.

**B. Rules** (semgrep supports YAML). The invariants that must survive consumer adaptation —
i.e., things the recipe payload establishes that a rule then re-checks so drift doesn't
silently reintroduce a known failure mode. Calibrated per this repo's loud-≠-blocking law:
block broken promises (e.g., a release workflow with no tag-integrity gate ahead of it), warn
on un-adopted capability (e.g., a consumer that hasn't wired Homebrew publication yet).

### Hard-won requirements to capture

Each of the following was paid for in real tokens/failures during ISSUE-087/PLAN-ISSUE-087
(2026-07-27/28). They are recorded here as the acceptance bar for the eventual spec/plan, not
as already-decided implementation, so the planner does not have to re-derive any of them from
the closed issue.

1. **Formula over cask (founder-ratified).** Use goreleaser's `brews:` block despite its
   deprecation, not `homebrew_casks` — casks are macOS-only and this pipeline must keep Linux
   `brew` working. Deprecation facts as ratified: soft-deprecated in goreleaser v2.10,
   loud-deprecated in v2.16, removed only on a MAJOR version bump (no v3 timeline announced).
   Pin the `goreleaser-action` to the `~> v2` line, never `@latest` — that pin is what makes
   removal require a deliberate move, not an upstream release. Retirement trigger to encode in
   the pack's docs/recipe comments: migrate to `homebrew_casks` via goreleaser's
   `tap_migrations.json` path if/when that pin is deliberately moved past v2.
   Rule: assert the workflow pins the goreleaser-action to the v2 line (not `@latest`).
2. **Tag integrity as a gate before goreleaser runs.** Three checks: (a) the tag is a
   well-formed semver `vX.Y.Z`; (b) the tag's commit is an ancestor of `main`; (c) CI is GREEN
   on the tagged commit (query the CI run for that commit; refuse if it isn't green — don't just
   assume). The release job MUST declare `needs:` on both the tag-integrity job and the
   CI-check job. Rule: `release.yml`'s goreleaser job carries those `needs`; the workflow
   triggers ONLY on `v*` tag push (never branch push or PR); permissions are `contents: write`
   + `actions: read` (nothing broader).
3. **Version stamping.** ldflags `-X main.version={{.Version}}` for goreleaser-built binaries.
   The consumer also carries a `resolveVersion` starter with precedence: ldflags value → module
   version from `debug.ReadBuildInfo()` → `"dev"`. It must REJECT non-release build-info shapes
   — `(devel)`, `+dirty`, and pseudo-versions — rather than reporting them as if they were a
   real tag. The trap this guards against: a plain local `go build` already stamps a pseudo-
   version via `ReadBuildInfo`, discovered only by capturing real output during ISSUE-087. Both
   distribution paths were proven live on ISSUE-087's launch day: a Homebrew-installed binary
   reported the stamped `0.1.0`; `go install ...@v0.1.0` reported `v0.1.0` via module version.
   Recipe payload: the `version.go` starter plus its table test (covering all three precedence
   branches and the rejected shapes).
4. **Four platforms.** darwin/amd64, darwin/arm64, linux/amd64, linux/arm64 — linux/arm64 is
   founder-ratified as first-class (DIR-001 lane-close), not an optional fourth target.
5. **`HOMEBREW_TAP_TOKEN` and the derived-env falsifier class.** The workflow must export
   EVERY env var goreleaser's config templates reference via `{{ .Env.X }}` — goreleaser
   resolves those strictly, and a missing export produces a half-published release (binaries
   and the GitHub Release go out; the formula-publish step silently dies). The token is a
   fine-grained PAT with Contents R/W, scoped to the tap repo only — not the whole org. One
   org tap can host many formulas: `backstop-ai/homebrew-tap` already serves `backstop` and can
   serve `stash`.
6. **CI droppings.** Any file a workflow step writes into the workspace before a gate/diff-
   scoped step runs enters that step's diff scope as an untracked file — ISSUE-087's own CI
   learned this the hard way when `ast-grep.zip` and `gate-report.json` were being gated as if
   they were source changes. Every starter that writes files into the workspace (goreleaser's
   `dist/`, any diagnostic capture) must ship the matching `.gitignore` entries as part of its
   recipe payload, not leave the consumer to discover the gating false-positive themselves.
7. **`goreleaser check` limitation.** `goreleaser check` requires a configured git remote and,
   with a `brews:` key present, exits 2 ("valid but deprecated") even on a config that is
   otherwise correct — this is expected, not a defect to chase. Consumer-facing verification
   guidance should point at `goreleaser release --snapshot --clean` (a real four-platform build
   needing neither a remote nor a tag) rather than bare `check`. Per PLAN-ISSUE-087's CLM-012
   carve-out: tolerate EXACTLY the `brews:` deprecation exit and nothing else — any other
   `check` failure is real and must fail the verification.
8. **actionlint-clean.** Every workflow the pack's recipes ship must pass `actionlint` with no
   findings — this was true of ISSUE-087's delivered workflows and is a floor, not a stretch
   goal, for the pack's starters.
9. **Pack-craft constraints from the fleet's own law.** These aren't specific to release
   tooling but bind this pack like any other: every claim needs both a positive and a negative
   fixture (`ParseManifestFile` enforces the pair); fixtures must be CAPTURED from real tool
   output, never fabricated, and must actually be capable of FALSIFYING the claim they support;
   SARIF severity is the pack-author's blocking contract — `warning` is non-blocking,
   `error`/absent blocks (founder-ratified 2026-07-28); and rules must assert invariants that
   SURVIVE consumer adaptation (e.g., "the goreleaser-action pin is on the v2 line"), never
   literal strings tied to one consumer (e.g., a specific binary name or org/repo).

### Open question (does not block authoring)

Where `stash` publishes — its own org/repo, and whether it shares `backstop-ai/homebrew-tap` or
gets its own tap — is founder input still pending. The pack itself must not encode an answer
either way (tap coordinates, org, and binary name are all consumer-configured through the
recipe's project-specific edits); this only affects how `stash` applies the pack once it exists,
not the pack's design.

## Dependencies and cross-references

- **DIR-001** (Release Workflow) — the framing directive; its "Release-pipeline follow-on
  routings" section is the source of the `go-distribution` pack framing and the ratified tap
  coordinates (`backstop-ai/homebrew-tap`), the linux/arm64 fourth platform, and the
  formula-over-cask ruling.
- **ISSUE-087 / PLAN-ISSUE-087** (CI-Driven Release Pipeline) — the hand-written trinity this
  issue productizes; PLAN-ISSUE-087's CLM-012 carve-out (goreleaser `check`'s expected
  `brews:`-deprecation exit) and its close-out phases (module rename, version resolution,
  goreleaser config, release + tag-integrity workflows, the token-export/derived-env falsifier)
  are the primary source of the hard-won requirements captured above.
- **BUNDLE-015 / SPEC-054** (Pack Scaffolding & Recipes) — the recipe-apply-and-manifest
  machinery this pack's payload (A) is built on; delivered and E2E-proven, so this issue
  consumes it rather than extending it.
- **ISSUE-084** (Published Pack Repos Have No CI) — this pack's own repo
  (`backstop-ai/go-distribution`) needs its own CI eventually, per that issue's class of gap;
  not a blocker for authoring the pack itself.
- **ISSUE-020** (Linux Sandbox Is a Hard Error) — its acceptance criterion carries the
  hand-baked-CI fact ("core `ci.yml`'s `gate` job hand-bakes Go tooling instead of calling
  `backstop gate`", welded there by 2026-07-27 founder ruling per the stale-citation correction
  above). This pack's rules gating backstop-core's own trinity is a partial retirement of that
  debt class once backstop-core adopts the pack — but the full fix (the `gate` job itself
  calling `backstop gate`) remains blocked on ISSUE-020's Linux sandbox work, not on this pack.

## Verification

No requirements/claims are recorded at `open` status (schema-gated — full traceability parity
applies from `ready` onward). Recorded here so `ready`-promotion inherits a concrete target:

- The pack passes `pack test` across all phases (structural, coherence, fixtures, archetype,
  layer, risk-class) with no decoy padding.
- Every rule fixture pair (positive + negative) is captured from real tool output and each
  negative fixture actually fails the rule it targets.
- A fresh Go CLI project consuming the pack via `pack add backstop-ai/go-distribution` and
  applying its recipes produces a `.goreleaser.yml` + release/tag-integrity workflows that pass
  `goreleaser release --snapshot --clean` (real four-platform build) and `actionlint`.
- backstop-core itself adopts the pack and its rules gate the existing hand-written trinity —
  proving the rules assert invariants generically, not against backstop-core's own literals.
- The pack is published at `v0.1.0` in its own repo (`backstop-ai/go-distribution`, local
  `~/src/projects/backstop-go-distribution-pack`) and consumable via `pack add`.

## References

- `directives/DIR-001-release-workflow.directive.md` — "Release-pipeline follow-on routings"
  section (go-distribution pack framing, ratified tap coordinates, linux/arm64, formula-over-
  cask ruling)
- `issues/ISSUE-087-ci-driven-release-pipeline.issue.md` — the delivered hand-written trinity
- `plans/PLAN-ISSUE-087-ci-driven-release-pipeline.plan.yml` — CLM-012 carve-out (line ~616),
  the derived-env falsifier (lane close), CI-droppings gating lesson
- `issues/ISSUE-085-recipe-pack-archetype-gap.issue.md` — the recipes/scaffolding archetype
  this pack's recipe payload (capability A) depends on
- `issues/ISSUE-084-published-packs-ungated-at-tag-time.issue.md` — the pack-repo CI gap this
  pack's own repo will eventually need to close
- `issues/ISSUE-020-cross-platform-sandbox-linux-noop.issue.md` — current owner of the
  hand-baked-CI-pipeline fact (acceptance criterion), correcting the stale ISSUE-086 citation
  in ISSUE-087/DIR-001
- `issues/ISSUE-086-published-baseline-generated-packless.issue.md` — the narrower defect
  ISSUE-086 covers today, and the "Split history" section documenting the 2026-07-27 narrowing
  that moved the hand-baked-CI fact to ISSUE-020
