---
name: first-consumer-finds-what-dogfood-hides
description: The reference implementation's own CONFIG is a blind spot — dogfooding on a fully-configured repo cannot see defects that only appear for a sparsely-configured consumer; prove new capability on a SECOND, deliberately-plainer project
metadata:
  type: feedback
---

Measured 2026-07-29 across PLAN-ISSUE-101: the go-distribution pack's FIRST real
consumer (stash) exposed TWO core defects that backstop-core's own gate had never
shown, in a codebase that dogfoods itself constantly.

Both had the same root shape — **backstop-core's configuration is richer than a new
consumer's, and the richer config silently supplied what the code failed to**:
- ISSUE-104: every semgrep declared-WARNING rule blocked, because parseSarif read
  only `results[].level` while semgrep states severity on the rule DESCRIPTOR.
  Core never noticed: its own findings were errors anyway.
- ISSUE-105: step status was a raw COUNT, and the severity-aware verdict was only
  reachable through `applyScopedPolicy` — i.e. only for a consumer that DECLARES an
  `enforcement.policy` entry for that dimension. Core's backstop.yml declares one.
  stash does not. Same code, opposite verdict.

**Why:** dogfooding proves the tool works ON THE TOOL. It cannot prove the tool
works for someone who has not yet configured everything, which is every new user.
A capability whose contract says "for ANY pack / ANY consumer" is only tested when
something OTHER than the reference implementation exercises it.

**How to apply:** when shipping capability meant to travel (packs, recipes,
contracts), run the acceptance on a SECOND project chosen for being PLAINER — no
policy table, fewer packs, defaults everywhere — and diff its verdict against the
reference's. Ask specifically: which of these behaviors is supplied by CONFIG rather
than by CODE? Also treat a contract TEST that hand-builds its input as unproven —
both defects survived a green contract test built from synthesized SARIF
([[feedback_fixtures_from_real_output]]). Related:
[[project_sarif_warning_severity_lost]], [[project_go_distribution_pack_shipped]].
