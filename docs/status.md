---
title: Status and Boundaries
permalink: /status/
hero_question: "What is supported, limited, planned, or intentionally outside Backstop?"
---

# Status and Boundaries

What is supported, limited, planned, or intentionally outside Backstop?

## Supported and limited {#supported-and-limited}

<!-- backstop-claim: CLAIM-001 -->
Backstop currently validates intent artifacts and runs installed pack engines as a blocking gate.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-002 -->
Backstop cannot force an external harness to stop after a blocking verdict.
<!-- /backstop-claim -->

The first statement names shipped mechanism. The second names the process boundary without turning it into a future commitment.

## Boundary states {#boundary-states}

Public status uses five explicit states: supported, limitation, planned, non-goal, and adjacent guidance. Each state has durable sources and a visitor implication.

<!-- backstop-journey-link: JLINK-007 -->
[See ownership boundaries](/model/#ownership-boundaries)

## Project boundaries {#project-boundaries}

<!-- backstop-claim: CLAIM-003 -->
The website expansion is planned under BUNDLE-032 and is not a shipped v0.2.0 capability.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-004 -->
Backstop does not own an agent runtime or guarantee the behavior of an external toolchain.
<!-- /backstop-claim -->

## Adjacent guidance {#adjacent-guidance}

<!-- backstop-claim: CLAIM-005 -->
Backstop stops at an inspectable verdict because external orchestration and organizational enforcement have different owners.

<!-- backstop-journey-link: JLINK-024 -->
[Continue outside Backstop](/contributing/#external-ownership)

That continuation is guidance, not a guarantee provided by Backstop.
<!-- /backstop-claim -->

## Pack direction {#pack-direction}

<!-- backstop-claim: CLAIM-033 -->
Core stays thin while maintained packs own standards, engines, fixtures, and their release cadence. New coverage belongs in the pack that owns the concern.
<!-- /backstop-claim -->

## Path-filter limitation {#path-filter-limitation}

<!-- backstop-claim: CLAIM-034 -->
Slash-bearing engine path patterns can fail open when a gate supplies explicit changed-file arguments instead of a directory walk; pack validation reports this current limitation as a path-scope advisory rather than treating the ineffective filter as enforcement.
<!-- /backstop-claim -->

## Release history {#release-history}

<!-- backstop-claim: CLAIM-035 -->
Release-specific behavior is identified by immutable version and source provenance. Current adoption instructions target v0.2.0.
<!-- /backstop-claim -->
