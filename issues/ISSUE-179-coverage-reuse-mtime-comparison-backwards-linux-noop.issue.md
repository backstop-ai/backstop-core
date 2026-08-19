---
title: "go-toolchain coverage-reuse mtime comparison is backwards — reuse never fires on real Linux CI, silently defeating PLAN-ISSUE-172's fix"
schema_version: issue/v1

issue:
  id: ISSUE-179
  title: "go-toolchain coverage-reuse mtime comparison is backwards — reuse never fires on real Linux CI, silently defeating PLAN-ISSUE-172's fix"
  type: bug
  status: open
  created: "2026-08-19"

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# go-toolchain coverage-reuse mtime comparison is backwards — reuse never fires on real Linux CI, silently defeating PLAN-ISSUE-172's fix

## Problem

`PLAN-ISSUE-172` shipped `backstop-ai/go-toolchain` v1.7.0 (local clone
`/Users/bmanson/src/projects/backstop-go-toolchain-pack`, commit `fb2b947`) to eliminate a
redundant full `go test ./...` run during backstop's own gate, via a coverage-profile-reuse
mechanism: `scripts/test-produce.sh` writes `cover.out` and touches a stamp file; a later
`coverage-produce.sh` invocation is supposed to reuse that same `cover.out` instead of re-running
the full suite, if the stamp shows it's still fresh.

The reuse check in `scripts/coverage-produce.sh` (line ~38) is:

```sh
if [ -f "$stamp" ] && [ -f cover.out ] && [ ! cover.out -ot "$stamp" ]; then
  reuse=1
fi
```

i.e. "reuse only if `cover.out` is NOT OLDER THAN the stamp." But `scripts/test-produce.sh` writes
`cover.out` FIRST (via `go "$@" 2>&1`), then touches the stamp file a moment LATER, in the same
script. By construction, `cover.out`'s mtime is always strictly earlier than the stamp's — the
condition as written can essentially never be true at full timestamp precision. **The comparison
is directionally backwards**: it should be asking whether the stamp is fresh relative to
`cover.out`, not the reverse.

**Confirmed on real Linux CI**, not inferred: CI run `32275399064` (commit `2f8fa89` on `main`)
measured `coverage_threshold` at 602500ms — essentially unchanged from the pre-fix baseline of
~612000ms — instead of the ~2211ms (~133x collapse) PLAN-ISSUE-172 measured and verified locally
on darwin. The mechanism is a complete no-op on real Linux CI.

### Why it "worked" locally and hid the bug

macOS's `/bin/sh` `test -ot`/`-nt` compares mtimes at whole-**second** resolution. The true gap
between `test-produce.sh` writing `cover.out` and touching the stamp is only a few milliseconds,
so on darwin both timestamps round to the same second and the comparison ties — which happens to
satisfy the (backwards) condition by accident, making reuse fire. Ubuntu's `/bin/sh` (`dash`)
compares at full **nanosecond** resolution, correctly detects the true ordering every time, and
reuse never fires.

### Root cause, precisely diagnosed with real evidence

Diagnosed on a throwaway debug branch `debug/issue172-linux-mtime-repro` against real Linux CI
(run `32278614259`, since cleaned up — findings recorded here as the durable record):

- A synthetic write-then-touch test on Linux showed a ~98µs gap, correctly detected as `NO-REUSE`.
- The actual end-to-end `test-produce.sh` → `coverage-produce.sh` sequence on Linux showed a
  ~13ms gap, also correctly detected as `NO-REUSE`.
- The identical experiment run on darwin showed the same true ordering (~3.9ms gap), but macOS's
  second-resolution `test -ot` reported `REUSE` anyway — the coincidental pass that masked the bug
  during PLAN-ISSUE-172's local verification.

**Alternative hypotheses ruled out with real evidence** (cited for completeness — all falsified):

- *Working-directory/process isolation between the two producer scripts* — both construct their
  runner with the identical `projectRoot`, and a live filesystem-state test confirmed the file
  persists across the two invocations within one job.
- *Diff-scope narrowing the target away from `./...` on CI's `gate --base` invocation* —
  code-confirmed `GateScopeModeDiff` never triggers the file-mode narrowing path, corroborated
  live: CI's actual scope showed `scope.Mode=diff files=46`, never file mode.
- *A step-ordering race or a root/permission issue on the Linux runner* — no evidence found; the
  failure is deterministic every run and fully explained by the shell-level comparison, not a race
  or permission denial.

## Impact

The entire point of PLAN-ISSUE-172's shipped fix — collapsing gate wall-clock time by reusing an
already-computed coverage profile instead of re-running `go test ./...` a second time — does not
happen where it matters most: real Linux CI, the platform the fix was built for. CI gate runs pay
the full pre-fix cost every time (~602500ms vs. the ~2211ms the mechanism is supposed to deliver),
silently, with no error or warning — the gate still passes, just slow. This is a correctness-shaped
defect in a performance mechanism: it doesn't produce a wrong verdict, but it produces none of the
benefit PLAN-ISSUE-172 shipped it to deliver, and the darwin-only verification method that shipped
it structurally could not have caught it (coarse local clock ties in exactly the direction that
hides a backwards comparison).

## Direction

Fix lives entirely in the external pack repo
(`/Users/bmanson/src/projects/backstop-go-toolchain-pack`), not backstop-core — needs a pack
version bump (e.g. v1.7.1), then `pack update`/relock in backstop-core. In `coverage-produce.sh`,
either:

- **(a)** Drop the mtime comparison entirely and trust presence-of-stamp-plus-`cover.out` as
  sufficient. The stamp is unconditionally consumed via `rm -f` right after the check regardless
  of branch outcome, so a stale leftover stamp cannot survive across separate gate invocations —
  only a same-run write can produce one. This removes the fragile cross-platform mtime-precision
  dependency entirely.
- **(b)** If a defensive check is still wanted, flip the comparison direction to match the real
  write-then-touch chronology: assert the **stamp** is not older than `cover.out`
  (`[ ! "$stamp" -ot cover.out ]`) instead of the current backwards check.

Whichever shape is chosen, the fix needs a regression test that specifically exercises
full-precision mtime ordering (e.g. a deliberately sub-second write-then-touch gap asserted via
`stat` nanosecond fields, or run under a shell/tool that doesn't round to whole seconds) — not one
that could tie on a coarse clock the way this defect's original verification did, since that is
exactly the gap that let this ship.

## Notes

- Origin context: `PLAN-ISSUE-172` (closed 2026-08-19, delivered via `backstop-ai/go-toolchain`
  v1.7.0 as approach (C), "adopt the single-run convention" — discharging `ISSUE-068`'s parked
  follow-on) and `ISSUE-172` ("Gate Steps Run Sequentially Not Parallel", closed, `delivered_by:
  PLAN-ISSUE-172`). This issue is a follow-on defect in that shipped mechanism, discovered during
  the plan's own post-merge CI verification — not a defect in `pkg/gate/gate.go` or
  `.github/workflows/ci.yml`, both of which PLAN-ISSUE-172 explicitly left untouched.
- Existence-in-world checked before filing: no open issue or bundle in `backstop-core` covers this
  mtime-comparison defect specifically. Related-but-distinct go-toolchain issues (`ISSUE-145`
  go-build stderr discarded, `ISSUE-155` gotoolchain-exempt-decisions-pending) are different
  mechanisms and do not overlap.
