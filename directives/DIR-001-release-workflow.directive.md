---
title: "Release Workflow"
number: DIR-001
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: done
  source:
    - "BUNDLE-003"
    - "ISSUE-087"
    - "ISSUE-090"
    - "ISSUE-101"
    - "ISSUE-109"
    - "ISSUE-111"
---

## Description

Set up GitHub Actions CI/CD to build and release the backstop binary on merge to main. This is the prerequisite for anyone outside this machine to use backstop.

**Launch tiering (founder-decided 2026-07-27):** CI-driven releases are
LAUNCH-BLOCKING primary distribution, not an optional convenience layer —
this is how backstop is meant to reach any consumer outside this machine.
`go install` remains a supported, documented convenience path — kept
working, not deprecated — for a Go-toolchain-equipped consumer, but it is
not the primary distribution mechanism; CI-driven releases are.

- GitHub Actions workflow: build on push to main, run gate, produce binaries
- Cross-platform builds (darwin/amd64, darwin/arm64, linux/amd64)
- GitHub Releases with versioned binary assets
- Homebrew tap (`backstop-core/homebrew-backstop`) for `brew install backstop`
- GoReleaser or equivalent for automated release management
- The release workflow also generates the baseline artifact (BUNDLE-007) post-merge
- ID resolver fix (ISSUE-090): reconcile artifact-ID allocation against
  `max(git tags, local disk scan)` rather than tags alone, so it must land
  **before** the remote is added — see Notes
- Go-distribution pack lane (ISSUE-101, `closed`; ISSUE-109, `open`;
  ISSUE-111, `open`): productizing the hand-written release trinity as the
  `backstop-ai/go-distribution` pack and adopting it into backstop-core's
  own `backstop.yml`/`backstop.lock` — founder-ruled home 2026-08-10, see
  Notes

### Delivery status

Shipped: ISSUE-087 (`closed`) and PLAN-ISSUE-087 (`completed`) delivered the
CI-driven release pipeline — `v0.1.0` and `v0.1.1` are tagged, the remote is
live at `github.com/backstop-ai/backstop-core`, `.github/workflows/`
holds `ci.yml`, `release.yml`, and `tag-integrity.yml`, and both install
paths are proven on a real machine: `brew install` via the published
`backstop-ai/homebrew-tap`, and `go install`.

Not closed by this status change: ISSUE-090 stays cited above and remains
`open`. Its situation has materially inverted since filing — it was written
against this repo's actual state until launch, no configured remote, which
is what made `FetchTags` fail and drove the tag-vs-disk divergence. The
remote now exists, so the tag path is live again and the specific
recurrence window the issue describes is closed for the moment. The
resolver itself is still unfixed, so the latent defect stands the next time
git ops fail (offline, auth failure, a new repo).

Known correctness caveat on this `done` verdict: the Description bullet
"GitHub Actions workflow: build on push to main" was not delivered as
written — ISSUE-087 shipped tag-triggered releases instead, not
push-to-main. `done` here should not be read as "every bullet as literally
worded shipped." Separately, the baseline bullet ("the release workflow
also generates the baseline artifact") is DIR-003's territory and its
producer half did land, but that thread belongs to DIR-003, not detailed
here.

## Notes

**Linux-runner dependency (identified 2026-07-27).** GitHub Actions runners
are Linux, and `SandboxedRun`/`SandboxedRunStdout`
(`pkg/packval/sandbox.go`) are a hard no-op on Linux (ISSUE-020) — every
pack-engine dispatch that runs a convert script or sandbox-validator fails
outright there. Since this directive's own pipeline runs `backstop gate` as
a merge gate, that gate step cannot go green on a Linux CI runner until the
Linux sandbox mechanism lands. Chain: BUNDLE-021 OQ-2 (should packs declare
behavior instead of names, so core derives a sandbox profile — the
enforcement question that routes into OQ-3, and notes the sandbox "dies on
Linux today" per ISSUE-020) and OQ-3 (whether/how the sandbox extends to
engine commands, not just convert scripts) → ISSUE-020 (the Linux sandbox
implementation, scoped under DIR-024) → this directive's self-gating
release pipeline. The non-gating parts of the pipeline (cross-compile,
GitHub Releases, Homebrew tap) do not depend on the sandbox, so status
stays `queued` rather than
formally blocked — but a release workflow that includes `backstop gate`
cannot be proven green end-to-end until ISSUE-020 resolves.

**ISSUE-087 citation (filed 2026-07-27).** ISSUE-087 (CI-Driven Release
Pipeline — Tag-Triggered Cross-Platform Builds via goreleaser) is the
**launch-minimal slice** of this directive: goreleaser cross-platform
builds (darwin/amd64+arm64, linux/amd64+arm64), a `v*` tag-triggered GitHub
Actions workflow that requires green CI on the tagged commit, version
stamping that survives BOTH goreleaser ldflags AND `go install @vX.Y.Z`
(via a `debug.ReadBuildInfo()` fallback, since `cmd/backstop/root.go:16`
reports `version = "dev"` today), the module path rename
`github.com/bmanson/backstop-core` → `github.com/backstop-ai/backstop-core`
(295 Go files; MUST precede the push to the private remote), and a core
tag-integrity workflow adapted from the pack repos. Two pieces of this
directive's description are explicitly OUT of ISSUE-087's scope and remain
open under DIR-001 after it closes: (a) **self-gating via `backstop gate`**
at release time — deliberately deferred as a fast-follow because it needs a
macOS runner plus, for matrix parity, ISSUE-020's Linux sandbox (this is
the Linux-runner dependency recorded above; ISSUE-087's launch-minimal
scope does not self-gate, so it is NOT blocked by ISSUE-020); and (b) the
**Homebrew tap** — deferred post-launch, cheap to add later via
goreleaser's `brews:` block. ISSUE-087's tag-integrity workflow lands with
a temporary hand-written header comment marking it as a placeholder for a
recipe-sourced successor (DIR-019 / BUNDLE-015 REQ-018, the committed
CI-recipe-pack capability) — so it is not mistaken for a second permanent
hand-baked pipeline of the kind ISSUE-086 tracks. Sizing per the
2026-07-27 founder scoping report: ~1–1.5 days. ISSUE-087 supersedes the
parked branch `feat/spec-029-binary-distribution` (a relic of the retired
BUNDLE-003 v0.5.0 SPEC numbering); it is deliberately not resurrected, and
no release SPECs have ever existed for this work.

**ISSUE-090 citation (filed 2026-07-27).** ISSUE-090 (ID resolver allocates
from git tags only — fallback allocations are never reconciled against
disk) fixes `GitTagResolver.Resolve` (`pkg/scaffold/idresolver.go:73`),
which computes the next artifact ID as `max(existing backstop/<type>/NNN
tag)+1` from git tags alone. `ChainedResolver` silently falls back to
`LocalScanResolver` (a plain filesystem `ReadDir` scan, no tag created) any
time git ops fail — including `FetchTags` — which is this repo's actual
state today since no remote is configured. The two resolvers never
reconcile: a fallback-created ID has no reservation tag, so once a remote
is added and `FetchTags` starts succeeding again, the tag path resumes
computing its next number from `max(tag)+1` alone, unaware of any higher
numbers already consumed on disk by untagged fallback allocations — a
future scaffold can then re-issue an ID that already names a file on disk.
The fix: the resolver must compute the next number from `max(git tag scan,
local filesystem scan)`, not either view alone, making the fallback path
non-lossy going forward.

This belongs to DIR-001 specifically, not as general hardening, because
its trigger condition is exactly this directive's own launch action —
"add the `backstop-ai/backstop-core` remote" (see the ISSUE-087 citation
above). The moment that remote is added, `FetchTags` starts succeeding and
the tag path reactivates; if ISSUE-090 hasn't landed by then, the first
post-remote scaffold can collide with an ID already committed to disk. It
must land **before** the remote is added, not after.

A tag backfill was already executed on 2026-07-27 (`issue/089` and
`adr/001`–`adr/018` retroactively tagged) and closed the *immediate*
collision hazard — verified: every artifact type's max tag now leads its
file count (issue 090 vs 87 files, spec 062 vs 46, bundle 030 vs 21,
directive 031 vs 26, adr 018 vs 18). So ISSUE-090 is the **durable fix**,
not an emergency, and does not on its own block the remote from being
added — but it is cheapest to land before the remote rather than after,
since landing it after would require re-verifying no collision occurred in
the interim.

**Release-pipeline follow-on routings (ISSUE-087 lane close, 2026-07-28).**
Four founder-approved decisions surfaced while closing ISSUE-087 and route
here as follow-on scope/notes rather than reopening that issue:

- **go-distribution pack framing.** The release trinity ISSUE-087
  delivered — goreleaser config, the release workflows, and the Homebrew
  formula, all now in scope and shipped by PLAN-ISSUE-087 (the original
  2026-07-27 deferral of the tap was reversed same-day and survives only as
  superseded history in ISSUE-087's scope amendment) — is hand-written
  today by design. It is authored as a future payload of a
  `go-distribution` pack: founder framing captured 2026-07-27, "that
  trinity fits perfectly as a go launch/go distribution pack." This records
  intended eventual pack-ification, consistent with this repo's zero-baked-
  checks law, of infrastructure that should not stay hand-baked forever —
  it is not new scope executed under this directive today.
- **linux/arm64 fourth build target (shipped).** ISSUE-087 delivered a
  fourth cross-platform build target, `linux/arm64`, alongside
  darwin/amd64, darwin/arm64, and linux/amd64 — already present in
  `.goreleaser.yml`. This supersedes the three-platform list in this
  directive's Description bullet above; the fourth platform is live, not
  aspirational.
- **Ratified tap coordinates.** The Homebrew tap — delivered day 1 by
  PLAN-ISSUE-087, not deferred (see the go-distribution bullet above) — is
  ratified at `backstop-ai/homebrew-tap` — convention-over-configuration,
  founder-ruled — superseding the `backstop-core/homebrew-backstop`
  coordinate named in this directive's original Description bullet.
- **Formula-over-cask ruling (founder, 2026-07-28).** Keep goreleaser's
  `brews:` block (a formula) rather than migrating to `homebrew_casks`, to
  retain Linux brew support — casks are macOS-only. Deprecation facts as of
  this ruling: `brews:` went soft-deprecated in goreleaser v2.10 and
  loud-deprecated in v2.16; goreleaser's stated policy removes deprecated
  options only on MAJOR version bumps; no v3 timeline has been announced;
  this repo's release workflow pins the goreleaser-action version to
  `~> v2` (`.github/workflows/release.yml:119`) — the whole v2.x line, not
  a specific patch — so the pin cannot auto-cross into v3. Retirement
  trigger (no silent debt): migrate to `homebrew_casks` — using
  goreleaser's `tap_migrations.json` path — when we deliberately move that
  pin past the v2 line; goreleaser v3 shipping upstream cannot force it on
  its own.

**Go-distribution pack lane — ISSUE-101 home ruling (founder, 2026-08-10).**
The PM inbox escalated ISSUE-101's directive home four times without a
ruling (`.backstop/pm/INBOX.md`, entries at 2026-07-29T01:35Z, 13:22Z,
13:54Z, and 13:59Z) — the delay outlasted the work itself: PLAN-ISSUE-101
went `completed` and the pack was tagged/published to a home nobody had
yet ruled on. The founder has now ruled: **DIR-001 is the home.** This
resolves four artifacts in one line, per the inbox's own framing that they
all move together:

- **ISSUE-101** ("Go Distribution Pack — Productize the Release Trinity",
  now `closed`, delivered by `PLAN-ISSUE-101` `completed`) — productized
  ISSUE-087's hand-written release trinity (goreleaser config, the release
  and tag-integrity workflows, version stamping) as the published
  `backstop-ai/go-distribution` pack (`v0.1.0`), proven against both
  backstop-core itself and a first external consumer (`stash`), with hash
  equality between local and remote install. This is the pack this
  directive's "go-distribution pack framing" bullet (above, in the
  2026-07-28 lane-close notes) anticipated — ISSUE-101 fulfills that
  framing rather than opening new scope.
- **ISSUE-109** ("Goreleaser Derived-Env Cross-File Falsifier", `open`) —
  moves with ISSUE-101 because it is that issue's own filed residual
  (`PLAN-ISSUE-101` TASK-016): the general invariant "every `.Env.<NAME>`
  in `.goreleaser.yml` has a matching export in `release.yml`" is a
  cross-file correlation semgrep (single-file) cannot enforce, so only a
  narrow single-file WARNING rule shipped. Homed here and not at DIR-024
  ("Gate/Engine Quality") because the inbox's own analysis is correct on
  the point that matters for placement: whichever fix direction is chosen
  (a pack-side script engine, or a core-side correlated-file-group input
  mode) is downstream of ruling the pack's home first, and core's own
  release plumbing — the thing actually burned by the failure mode this
  issue tracks (ISSUE-087's half-published-release shape) — is squarely
  DIR-001's charter, not DIR-024's general engine-quality catch-all.
- **ISSUE-111** ("Backstop Core Adopts Go Distribution Pack", `open`) —
  moves with ISSUE-101 for the same reason (also filed by TASK-016): it
  asks whether backstop-core's own committed `backstop.yml`/`backstop.lock`
  durably installs the `go-distribution` pack PLAN-ISSUE-101 proved only in
  discarded worktrees. Adoption was measured verdict-neutral before this
  ruling (zero findings against core's own trinity, other packs'
  finding counts unchanged) — this is bounded, low-risk work, not an open
  question.

**BUNDLE-031 is expected to land here too, but is not yet cited.**
BUNDLE-031 ("Release Currency Versioning Machinery", `exploring`) has its
own open question, OQ-3, asking where its machinery should live, and names
the `go-distribution` pack as one of three candidate homes. This ruling
does not resolve OQ-3 — the founder drives bundle OQ resolution and
maturity promotion, not this directive — but it does make DIR-001 the
*likely* eventual directive home for BUNDLE-031 once/if it promotes past
`exploring` and needs one, on the same reasoning that homed ISSUE-101 here.
BUNDLE-031 itself is deliberately untouched by this ruling: not added to
this directive's `source:` (it is a bundle still in `exploring`, not yet a
citable delivered/committed artifact), and its own file, OQs, and maturity
are unchanged.

**Correction to this directive's own Notes, note-supersedes convention
(2026-08-10).** The "go-distribution pack framing" bullet in the
2026-07-28 lane-close notes above recorded only the founder's *framing*
("that trinity fits perfectly as a go launch/go distribution pack") and
explicitly stated "it is not new scope executed under this directive
today." That framing has since been executed — ISSUE-101 built and shipped
the pack it anticipated — so as of this ruling it is no longer merely
aspirational framing; it is delivered, `source`-cited work. The original
bullet is left intact above as accurate history of what was known
2026-07-28; this paragraph is the dated correction.

### Still-open cited issues are RELEASED, not homed (2026-08-17 founder ruling)

The founder has ruled, extending to issues the identical logic already ruled
for bundles on 2026-08-14 (applied to BUNDLE-004 and BUNDLE-005, see the
rationale comments in `BACKLOG.yml`'s `bundles:` section): an open issue
cited **only** by directives with `status: done` is **released** — it does
not read as "orphaned, needing a directive home," and it may be picked up
standalone via the normal `issue → plan` track without first finding or
opening a directive to carry it.

This directive is `status: done` and, per the 2026-08-02 done-directive
convention, was removed from `BACKLOG.yml` — so its citation of an issue
confers no live ownership over that issue. Three still-open issues cited
above are released by this ruling:

- **ISSUE-090** — the ID resolver must take `max(git tags, disk)`; fallback
  allocations are tag-lossy.
- **ISSUE-109** — the goreleaser derived-env cross-file falsifier.
- **ISSUE-111** — backstop-core adopts the `go-distribution` pack
  (ISSUE-101's residual).

**What deliberately did not change:** all three remain in this directive's
`source:` list above, verbatim — untouched, unreordered. As with the
bundle-side precedent, the citation is preserved as provenance (it records
where the work came from) and is not a claim of live ownership. Nothing in
this section removes or supersedes the ISSUE-090, ISSUE-109, or ISSUE-111
citation notes elsewhere in this file.

One caveat for a reader picking either issue up standalone: ISSUE-111 sits
downstream of BUNDLE-031's still-open OQ-3 (see the BUNDLE-031 paragraph
above), which asks where the `go-distribution` pack's release-currency
machinery should ultimately live and names this pack as only one of three
candidate homes — that question is still unruled. Releasing ISSUE-111 from
this `done` directive does not answer OQ-3; it only stops ISSUE-111 reading
as covered by a directive that no longer carries live work.
