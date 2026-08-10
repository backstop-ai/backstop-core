---
name: pack-ships-after-the-core-release
description: A pack whose rules depend on core BEHAVIOR must publish AFTER the core release carrying it — packs cannot declare a minimum core version yet (BUNDLE-020), so the only sequencing mechanism is publication order
metadata:
  type: project
---

Sequencing learned on PLAN-ISSUE-101 (2026-07-29): go-distribution's three WARNING
rules are only non-blocking because of ISSUE-104 + ISSUE-105, which shipped in
backstop-core v0.1.1. Publishing the pack FIRST would have handed every adopter a
pack that REDS their gate for un-adopted capability — the pack would have been
correct and the experience broken, with no diagnostic pointing at the core version.

**There is no version floor to declare.** A pack manifest cannot express "requires
core >= X" — that capability is BUNDLE-020's territory and does not exist yet. So
publication ORDER is the entire mechanism: core release first, pack tag second.

**How to apply:** before tagging a pack, ask what CORE behavior its contract leans
on, and whether that behavior is in a RELEASED core or only on main. If only on
main, the pack is not releasable yet no matter how green its own suite is —
`pack check`, `pack test` and a local falsification harness all pass without ever
touching the core code path in question. The tell is a claim phrased about the
GATE's treatment of a finding (blocking/non-blocking, scoped/exempt, waived) rather
than about the finding itself.

Corollary worth keeping: a pack's own tag-integrity workflow asserts tag == pack.yml
version, so the pack version is cheap to verify but says NOTHING about core
compatibility. Do not mistake a green tag-integrity run for a compatibility check.

Related: [[project_go_distribution_pack_shipped]],
[[feedback_first_consumer_finds_what_dogfood_hides]],
[[project_sarif_warning_severity_lost]].
