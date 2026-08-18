---
name: seatbelt-permits-device-nodes
description: darwin Seatbelt already permits writes to /dev/null and /dev/zero under a blanket (deny file-write*), so a darwin behavioural test for a device-node carve-out is a REGRESSION LOCK, not a red->green
metadata:
  type: project
---

macOS Seatbelt permits writes to DEVICE NODES as a class even under a blanket
`(deny file-write*)`. Measured 2026-08-18 through real `sandbox-exec` on the
production-shaped packval profile: `command -v jq >/dev/null 2>&1` and
`echo probe > /dev/null` both succeed with NO carve-out clause present, while
`touch <file in packDir>` under the same profile is refused ("Operation not
permitted"). `/dev/zero` behaves identically. The profile TEXT never stated this.

**Why:** Linux Landlock enforces the same stated "no writes" intent LITERALLY, so
the identical profile broke `>/dev/null` on Linux (exit 127, `cannot create
/dev/null: Permission denied`) while darwin silently worked. That asymmetry WAS
ISSUE-168 — the platforms disagreed because one enforced the promise and the other
had an accident that happened to match what scripts needed.

**How to apply:** When a plan asks for a darwin behavioural test of a device-node
sandbox carve-out, that test PASSES BEFORE THE FIX. Classify it honestly as a
regression lock (it goes red only if a future macOS tightens Seatbelt, or if
someone deletes the clause as decorative). The genuine darwin red is the
PROFILE-LITERAL pin. Reporting the behavioural test as the red phase is wrong and
gets corrected in review. See [[falsify-sandbox-allow-by-retargeting]] for how to
prove such a clause is live anyway.
