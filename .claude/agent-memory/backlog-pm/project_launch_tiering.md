---
name: launch-tiering
description: Founder-set launch razor (2026-07-25/26) — ONLY pack recipes and remote pack consumption block launch; Linux/CI viability was a third blocker; everything else including backstop init is tier-2
metadata:
  type: project
---

Brandon tiered the launch blockers on 2026-07-25/26 as a short closed list:
**pack recipes** (BUNDLE-015 / DIR-019) and **remote pack consumption**
(BUNDLE-006 / DIR-026), with **Linux/CI viability** (ISSUE-020, riding under
DIR-024) named alongside them in DIR-024/DIR-026's own text. Everything else
is tier-2 — explicitly including `backstop init` (DIR-002), which is wanted
but is not what makes backstop unusable if it slips.

**Why:** the razor exists to stop the backlog's breadth from setting the
launch date. Both tier-1 items are load-bearing for the distribution model
itself, not features among many — without remote consumption "install a pack
from its tap" is advertised-not-delivered, and without recipes `backstop
init` has nothing to delegate scaffolding to.

**How to apply:** when proposing BACKLOG.yml ordering, positions 1-3 are
reserved for DIR-019 / DIR-026 / DIR-024 and the burden of proof is on any
proposal that displaces them. When triaging a new issue, ask whether it
blocks one of the two tier-1 capabilities before arguing its own severity —
a `risk: critical` issue inside a tier-2 directive still ranks below tier-1
work. Do not re-litigate the tiering; it is the founder's call. Related:
[[mechanism-vs-ecosystem-gap]], which is the main way a tier-1 item can look
done without being launchable.
