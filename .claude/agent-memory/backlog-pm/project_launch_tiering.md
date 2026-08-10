---
name: launch-tiering
description: Founder-set launch razor — FOUR blockers as of 2026-07-27 (recipes, remote pack consumption, Linux/CI viability, CI-driven releases); all four substantively delivered 2026-07-28; everything else including backstop init is tier-2
metadata:
  type: project
---

The launch-blocker list is **four items, not two** (updated 2026-07-27; an
earlier version of this memory said two and was stale for a full day):

1. **pack recipes** — BUNDLE-015 / DIR-019
2. **remote pack consumption** — BUNDLE-006 / DIR-026
3. **Linux+CI viability** — ISSUE-020, riding under DIR-024
4. **CI-driven releases** — DIR-001 / ISSUE-087, tiered up by the founder on
   2026-07-27

Everything else is tier-2, explicitly including `backstop init` (DIR-002).
**All four were delivered and `v0.1.0` SHIPPED 2026-07-29T00:36Z** (release run
`30411560553`: 4 platform archives + checksums + Homebrew formula). The razor's
practical job now is guarding against *re-expansion* and framing post-launch
work, not ranking a pre-launch queue. The remaining founder-held step is the
7-repo visibility flip, gated on the content audit.

**Why:** the razor exists to stop the backlog's breadth from setting the launch
date. Each tier-1 item is load-bearing for the distribution model itself, not a
feature among many.

**How to apply:** BACKLOG.yml's own header comment is the authoritative record
of the current blocker set — **read it before quoting a count from memory**, since
the founder revises the tiering faster than memory gets rewritten. When triaging,
ask whether an issue blocks a tier-1 capability before arguing its own severity: a
`risk: critical` issue inside a tier-2 directive still ranks below tier-1 work. Do
not re-litigate the tiering; it is the founder's call.

**Corollary learned 2026-07-28, the expensive one:** a delivered blocker leaves
its directive's *position* justified by something that no longer exists — DIR-024
held position 4 purely on ISSUE-020, which closed, leaving ten tier-2 items ranked
above DIR-001/DIR-003/DIR-002. After any blocker closes, re-check what its
directive's ranking now rests on. Related: [[mechanism-vs-ecosystem-gap]] (how a
tier-1 item looks done without being launchable) and
[[launch-preconditions-are-not-backlog-items]].
