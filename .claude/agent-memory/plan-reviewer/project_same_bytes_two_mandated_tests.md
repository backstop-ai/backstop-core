---
name: same-bytes-two-mandated-tests
description: When two mandated tests in one plan are fed the SAME captured payload, run the real converter/parser on those bytes — a "keeps crashing" test and a "always yields a finding" floor claim silently contradict
metadata:
  type: project
---

If a plan mandates two tests whose inputs are the SAME captured bytes (one asserting the
old behavior "MUST KEEP PASSING", one asserting a new floor), evaluate BOTH predicates on
the real bytes. They frequently cannot both hold post-fix, and the validator cannot see it
because each test is individually well-formed.

**Why:** 2026-08-16, PLAN-ISSUE-067. `go-test-build-failure-stdout-only.txt` is exactly
`FAIL\t<pkg> [build failed]` + bare `FAIL`. One mandated test asserted that payload still
produces the opaque crash; CLM-009's floor asserted any output naming a `[build failed]`
package always yields >=1 finding. Post-fix the floor fires on those bytes, so the crash
path is never reached — the two mandated tests contradict on one payload. Resolution
turned on the OTHER claim's exact wording ("no diagnostic output AT ALL"), which genuinely
excluded a payload that names a failed package.

**How to apply:** (1) group a plan's mandated tests by the fixture they are fed; any
fixture feeding two tests gets both predicates evaluated by hand. (2) Pipe the real
captured bytes through the COMMITTED converter/parser to prove red-first empirically —
`sh <pack>/scripts/x-to-sarif.sh < capture.txt` returning `{"results":[]}` is real
red-first evidence, not a narrated one. (3) When two claims collide, read their VERBATIM
text — the narrower wording usually shows one claim never covered the payload, so the
resolution costs no coverage. Verify the losing claim keeps a distinct, non-overlapping
pin elsewhere. Related: [[prescribed-pattern-byte-shape]],
[[repurposed-test-claim-text-drift]].
