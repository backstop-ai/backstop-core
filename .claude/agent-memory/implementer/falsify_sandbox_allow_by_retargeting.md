---
name: falsify-sandbox-allow-by-retargeting
description: prove a scoped sandbox-exec allow clause is LIVE by retargeting the same clause shape at a regular file, where the platform does enforce -- you cannot falsify it on the permissive path itself
metadata:
  type: feedback
---

To prove a scoped `(allow file-write* (literal "<path>"))` clause actually
overrides a preceding `(deny file-write*)` — rather than being decorative syntax
the sandbox ignores — RETARGET THE SAME CLAUSE SHAPE at a regular file and run it.

Measured 2026-08-18, production-shaped profile, real `sandbox-exec`:
- clause -> `<tmp>/allowed.txt`: `touch` exits 0, file created
- same profile, `touch <tmp>/sibling.txt`: "Operation not permitted", not created
- typo'd operation (`file-writ*`): exit **65**, `unbound variable ... column 250`

**Why:** the clause's real target (`/dev/null`) sits on a path the platform permits
ANYWAY (see [[seatbelt-permits-device-nodes]]), so running it there can never
distinguish "the clause worked" from "the clause did nothing". Only a target the
platform genuinely denies can falsify it. The typo leg matters separately: it
proves a malformed clause fails LOUDLY and cannot silently widen or silently no-op.

**How to apply:** any time a sandbox/permission grant lands on a path that already
happens to work, do not report the passing run as evidence the grant functions.
Move the grant to an enforced path, show allow-vs-sibling, and show the malformed
form is loud. Three cheap commands, and they turn "tests pass" into a mechanism
proof.
