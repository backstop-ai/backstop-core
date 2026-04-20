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

- GitHub Actions workflow: build on push to main, run gate, produce binaries
- Cross-platform builds (darwin/amd64, darwin/arm64, linux/amd64)
- GitHub Releases with versioned binary assets
- Homebrew tap (`backstop-core/homebrew-backstop`) for `brew install backstop`
- GoReleaser or equivalent for automated release management
- The release workflow also generates the baseline artifact (BUNDLE-007) post-merge
