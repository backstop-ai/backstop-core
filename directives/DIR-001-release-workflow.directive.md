---
title: "Release Workflow"
number: DIR-001
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-003"
    - "ISSUE-087"
    - "ISSUE-090"
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
