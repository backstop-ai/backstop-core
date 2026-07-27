---
title: "Release Workflow"
number: DIR-001
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-003"
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
