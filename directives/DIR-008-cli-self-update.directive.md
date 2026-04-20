---
title: "CLI Self-Update"
number: DIR-008
created: "2026-04-19"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "BUNDLE-003"
---

## Description

Implement `backstop upgrade` (or `backstop self-update`) so the CLI can update itself to the latest release. Checks GitHub Releases for newer versions, downloads the appropriate binary for the current platform, replaces itself.

Depends on DIR-001 (release workflow — releases must exist to upgrade to).
