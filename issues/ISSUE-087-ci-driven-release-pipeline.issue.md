---
title: "CI-Driven Release Pipeline — Tag-Triggered Cross-Platform Builds via goreleaser"
schema_version: issue/v1

issue:
  id: ISSUE-087
  title: "CI-Driven Release Pipeline — Tag-Triggered Cross-Platform Builds via goreleaser"
  type: enhancement
  status: closed
  created: "2026-07-27"
  closed: "2026-07-28"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate

delivered_by: PLAN-ISSUE-087
---

# CI-Driven Release Pipeline — Tag-Triggered Cross-Platform Builds via goreleaser

## Resolution

Delivered by PLAN-ISSUE-087 (status: completed) across five forced-order phases: (1) the
module path rename (`github.com/bmanson/backstop-core` → `github.com/backstop-ai/backstop-core`),
(2) version resolution surviving both the goreleaser ldflags path and the `go install @vX.Y.Z`
`ReadBuildInfo` fallback, (3) the goreleaser config — cross-platform builds plus the Homebrew
`brews:` block, (4) the tag-driven release and tag-integrity workflows, (5) lane close (token
export + a derived-env falsifier). Commits `7f7f80c..e9f2061`.

**The pipeline executed for real tonight, not just in dry-run.** Tag `v0.1.0` pushed against a
green commit → tag-integrity checks passed → goreleaser built darwin/amd64, darwin/arm64,
linux/amd64, linux/arm64 archives plus checksums → a GitHub Release was published → the
Homebrew formula was pushed to `backstop-ai/homebrew-tap` via `HOMEBREW_TAP_TOKEN`. This was
`release.yml`'s first-ever execution, and it ran green end to end through formula publication.

**Honest residual, stated plainly rather than swept.** TASK-014's final consumer-side install
legs — `brew install backstop-ai/tap/backstop` and
`go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.1.0` — are pending the
repos' public-visibility flip, the founder-held audit gate on the full commit history (per the
launch plan). The pipeline is proven end-to-end through formula publication; the consumer-side
install proof necessarily lands post-flip, not before — a private tap and a private module path
cannot be fetched by an external `brew`/`go install` regardless of how correctly the pipeline
that populated them ran.

Routed follow-ons that were closure preconditions are done: the Homebrew scope amendment
recorded in this file's "Scope amendment" section (2026-07-28, formula-over-cask ratified,
superseding the original "deferred post-launch" bullet), and the corresponding DIR-001 notes
(linux/arm64 as a shipped fourth build target, the ratified `backstop-ai/homebrew-tap` tap
coordinates superseding the stale `backstop-core/homebrew-backstop` name, and the `brews:`-vs-
`homebrew_casks` migration-trigger note).

## Problem

**DIR-001 (Release Workflow) is the launch-blocking anchor for this issue** — founder tiering,
2026-07-27: "CI-driven releases are LAUNCH-BLOCKING primary distribution, not an optional
convenience layer — this is how backstop is meant to reach any consumer outside this machine."
`go install` stays supported as a convenience path for Go-toolchain-equipped consumers, but it is
not the primary distribution mechanism. Today neither exists in CI-automated form: there is no
tag-driven workflow, no cross-platform build, no GitHub Release, and the binary reports `version
= "dev"` (`cmd/backstop/root.go:16`) regardless of what a consumer actually installed.

This issue scopes the **launch-minimal** slice of DIR-001, per the same 2026-07-27 scoping report
that produced the tiering: goreleaser + a tag-driven workflow, version stamping that survives
`go install @vX.Y.Z`, the module path rename that must precede any push to the new remote, a core
tag-integrity check, and (per the 2026-07-27 scope amendment below) a Homebrew tap. It remains
narrower than DIR-001's full description in one respect — self-gating via `backstop gate` is
explicitly named below as NOT in scope for this issue.

### Stale-memory correction, recorded so it isn't re-derived incorrectly later

No release SPECs have ever existed for this work. An earlier naming scheme (SPEC-020..029 under
BUNDLE-003) was retired in the BUNDLE-003 v0.5.0 purge; the parked branch
`feat/spec-029-binary-distribution` is a relic of that retired numbering and is **deliberately not
resurrected** — goreleaser supersedes whatever that branch was building toward. Anyone picking this
issue up should treat that branch as historical only, not as a starting point.

## Scope

### In scope

1. **goreleaser config** (`.goreleaser.yml` or equivalent) driving cross-platform builds:
   darwin/amd64, darwin/arm64, linux/amd64, linux/arm64. Output: archives, checksums, and a
   GitHub Release, all produced from a single `goreleaser release` invocation.
2. **Tag-driven workflow**: a GitHub Actions workflow triggered on `v*` tag push. Before building,
   the workflow requires CI to already be green on the tagged commit — a tag pushed against a red
   commit does not produce a release.
3. **Version stamping that survives `go install`**: the existing `-ldflags -X` hook
   (`cmd/backstop/root.go:16`, `var version = "dev"`) continues to work for goreleaser-built
   binaries. Additionally, wire a `debug.ReadBuildInfo()` fallback so that a binary built via
   `go install github.com/backstop-ai/backstop-core/cmd/backstop@vX.Y.Z` — which does not run
   goreleaser's ldflags injection — still reports the real tag version rather than falling through
   to `"dev"`. Both paths need to agree at `backstop version`.
4. **Module path rename**: `github.com/bmanson/backstop-core` → `github.com/backstop-ai/backstop-core`,
   across all 295 files currently referencing the old path (verified count via
   `grep -rl "github.com/bmanson/backstop-core" --include="*.go"`, 2026-07-27). This is mechanical
   (import path + `go.mod` module directive), but it is an ordering constraint, not an optional
   cleanup: **it MUST precede the push to the private remote** — a release tagged and pushed under
   the old module path would ship an import path no `go install` against the new remote could ever
   resolve. Final verification that `go install @vX.Y.Z` reports a real version against the new
   path is necessarily deferred until the remote exists (see Dependencies).
5. **Core tag-integrity workflow**: adapted from the pattern already proven in the pack repos
   (semver well-formedness + tag-is-ancestor-of-main, gating a tag push before goreleaser runs
   against it). Land it with a **temporary, hand-written header comment** naming this as a
   placeholder for the REQ-018 successor (BUNDLE-015's committed CI-recipe-pack capability) — the
   intent is that this hand-written version gets replaced by a recipe-sourced one once REQ-018
   ships a CI recipe pack, not that it becomes a second permanent hand-baked pipeline alongside the
   Fact-1 gap ISSUE-086 already tracks.
6. **Homebrew tap** (`backstop-ai/homebrew-tap`, DIR-001's `brew install backstop`), via
   goreleaser's `brews:` block — see "Scope amendment" below.

### Scope amendment, recorded so it isn't re-derived incorrectly later

The original scoping (2026-07-27) deferred the Homebrew tap post-launch (see the superseded bullet
this replaces, preserved below). The founder reversed that 2026-07-27, ruling Homebrew + `go
install` both ship day 1, and ratified formula-over-cask 2026-07-28 (casks are macOS-only;
goreleaser's deprecated `brews:` block survives all of goreleaser v2.x by policy, and the pipeline
pins `~> v2` — the v2.x line, not a specific patch, so the `brews:` key cannot disappear without a
deliberate move of the pin past v2). Item 6 above reflects that reversal; the delivered implementation
(PLAN-ISSUE-087) publishes the formula to `backstop-ai/homebrew-tap`.

Superseded bullet (originally under "Explicitly out of scope", struck by the above): *"Homebrew
tap (`backstop-core/homebrew-backstop`, DIR-001's `brew install backstop`). Deferred post-launch.
goreleaser's `brews:` block makes adding a tap cheap once the core pipeline is proven, so deferring
it costs little — it is not on the critical path to a consumer being able to fetch a real
binary."*

### Explicitly out of scope (founder-approved)

- **Self-gating via `backstop gate`.** This pipeline releases without `backstop` gating itself at
  build/release time. The existing lint/test/coverage CI jobs (today hand-baked, per ISSUE-086)
  continue to gate the tagged commit before the tag-driven workflow trusts it — that is sufficient
  for a launch-minimal release pipeline. Self-gating is an explicit fast-follow, not a silent gap:
  it requires a macOS runner (darwin sandbox is real; Linux is not, per ISSUE-020) and a fleet lock
  migration, and — once ISSUE-020's Linux sandbox lands — full parity across the matrix this issue
  builds for. Framing it as fast-follow rather than blocking is a deliberate scope cut, recorded so
  it isn't mistaken for an oversight later.

## Sizing

Approximately 1–1.5 days, per the 2026-07-27 scoping report. The module rename is mechanical and
fast; the goreleaser config, tag-driven workflow, and version-stamping fallback are the parts that
need real verification against an actual tagged build, not just config review.

## Dependencies and cross-references

- **DIR-001** (Release Workflow) — this issue is the launch-minimal slice of DIR-001's scope; DIR-001
  remains open for the deferred piece (self-gating fast-follow) once this issue closes. Homebrew
  tap is no longer a deferred piece — see the "Scope amendment" note above.
- **ISSUE-084** (Published Pack Repos Have No CI) — a fetchable, versioned core binary is what
  unblocks the eleven pack repos' own CI from having something durable to pin against; this issue
  is a prerequisite for that class of work, not a duplicate of it.
- **ISSUE-086** (Core CI Is a Hand-Baked Go Pipeline) — tracks the separate, pre-existing gap that
  `ci.yml`'s `gate` job hand-bakes Go tooling instead of calling `backstop gate`. This issue does
  not fix that gap (see "Explicitly out of scope" above) — it depends on that hand-baked pipeline
  continuing to gate commits in the interim, and the temporary tag-integrity header above is
  written so it doesn't become a second instance of the same category of debt.
- **ISSUE-020** (Linux sandbox hard error) — blocks the self-gating fast-follow specifically (a
  `backstop gate` release step on a Linux runner hits the sandbox no-op); does not block this
  issue's launch-minimal scope, which does not self-gate.
- **`feat/spec-029-binary-distribution`** (parked branch) — historical relic of a retired SPEC
  numbering scheme (BUNDLE-003 v0.5.0 purge); deliberately not resurrected. See "Stale-memory
  correction" above.

## Verification

No requirements/claims are recorded at `open` status (schema-gated — full traceability parity
applies from `ready` onward). Verification criteria recorded here so `ready`-promotion inherits a
concrete target:

- Pushing a well-formed `vX.Y.Z` tag against a commit with green CI produces darwin/amd64,
  darwin/arm64, linux/amd64, and linux/arm64 archives, checksums, and a published GitHub Release —
  via a real tag push against the real workflow, not a dry-run/local-only goreleaser invocation.
- Pushing a tag against a commit where CI is red, or a malformed tag (non-semver, or not an
  ancestor of `main`), does not produce a release.
- `backstop version` on a goreleaser-built binary reports the real tag version.
- `go install github.com/backstop-ai/backstop-core/cmd/backstop@vX.Y.Z` followed by `backstop
  version` reports the real tag version (not `"dev"`) — verification of this specific criterion is
  necessarily deferred until the module rename has landed and the private remote exists.
