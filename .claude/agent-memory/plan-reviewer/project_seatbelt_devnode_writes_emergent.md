---
name: seatbelt-devnode-writes-emergent
description: macOS sandbox-exec permits writes to /dev/null AND /dev/zero under a blanket (deny file-write*) — so a "/dev/null carve-out" darwin test is green before the fix and its "and nothing else" is structural only
metadata:
  type: project
---

Measured 2026-08-18 on this darwin host against the real production-shaped profile
(`(version 1)(import "bsd.sb")(deny default)(allow process*)(allow file-read* <subpaths>)(deny network*)(deny file-write*)`):

* `command -v jq >/dev/null 2>&1` and `echo x > /dev/null` SUCCEED under the UNFIXED
  profile. `echo x > /dev/zero` also succeeds. Ordinary file writes (inside or outside
  packDir) are still refused with "Operation not permitted".
* `(allow file-write* (literal "<path>"))` appended AFTER `(deny file-write*)` is real,
  valid SBPL: `touch <literal>` → exit 0, sibling path → refused. Placing it BEFORE the
  deny also works. A malformed operation name exits non-zero with
  `sandbox-exec: unbound variable: ... at <input string>, line 1, column N`.
* Binaries outside the read subpaths (`/bin/sh`, `/usr/bin/touch`) still EXEC fine —
  `bsd.sb` + `(allow process*)` cover it. `pkg/packval/sandbox_security_test.go` already
  relies on this.

**Why:** a plan adding a `/dev/null` write carve-out to the darwin profile is
BEHAVIOURALLY A NO-OP on macOS — the Linux Landlock half is the only half that was ever
broken. Any darwin "the idiom now succeeds" test is a REGRESSION LOCK (green before AND
after), never a red→green; the only darwin RED that flips is the profile-literal pin.

**How to apply:** when reviewing a sandbox-profile plan, reject any framing that presents
the darwin behavioural test as the red phase, and check that a test named
`...AndNothingElse` scopes its claim to the profile TEXT — behaviourally other device
nodes stay writable regardless of the clause. Related: [[project_e2e_fixture_already_loud_at_head]].
