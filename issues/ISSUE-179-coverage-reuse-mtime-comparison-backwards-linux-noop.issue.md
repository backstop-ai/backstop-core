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

## Investigation (2026-08-19)

Run under `PLAN-ISSUE-179` TASK-001, before any fix was written. All numbers below were
re-derived on 2026-08-19, not quoted from the plan.

### Platform confirmation

`ubuntu-latest` resolves to `ubuntu-24.04` (this repo's own captured runner probe,
`pkg/packval/testdata/ubuntu-runner-probe.txt:13,108`). Confirmed inside `ubuntu:24.04`
rather than assumed:

```
$ docker run --rm ubuntu:24.04 readlink -f /bin/sh
/usr/bin/dash
$ ls -l /bin/sh   ->  /bin/sh -> dash
$ dpkg -l dash    ->  ii  dash  0.5.12-6ubuntu5  amd64  POSIX-compliant shell
```

### The end-to-end falsification, PRE-FIX

The state was arranged by running the **real** `scripts/test-produce.sh` with a `go` shim
that writes `cover.out` and then exits — i.e. the true production chronology including the
real process-teardown gap — and then running the **real** `scripts/coverage-produce.sh`,
observing through the shim whether the suite was re-run.

`ubuntu:24.04`, `/bin/sh` (dash), source pack scripts at v1.7.0:

```
    2026-08-19 18:43:09.724408177 +0000 cover.out
    2026-08-19 18:43:09.729408177 +0000 .backstop/go-coverage-fresh
    CURRENT   (shipped, backwards): NO-REUSE
    CANDIDATE (option b, flipped) : REUSE
  over 50 trial(s): REUSED=0  RE-RAN=50
```

The core fixture's copy of the scripts behaves identically (`REUSED=0 RE-RAN=50`). The gap
is ~5ms, `cover.out` is strictly older every time, and reuse never fires — **0 out of 50**.

darwin 23.6.0, `/bin/sh`, the identical experiment on the identical scripts:

```
    1787165006.911838073 cover.out
    1787165006.916016684 .backstop/go-coverage-fresh
    CURRENT   (shipped, backwards): REUSE
    CANDIDATE (option b, flipped) : REUSE
  over 50 trial(s): REUSED=50  RE-RAN=0
```

Same true ordering (`cover.out` strictly older by ~4.18ms, visible in the raw fractional
epoch fields), opposite verdict — **50 out of 50** reuse. That is the coincidental pass that
made the original darwin verification structurally incapable of catching this.

### The masking is a property of the shell build, not the shell's name

Production chronology, both candidate conditions, every shell on the darwin authoring
machine:

```
  /bin/sh    CURRENT=REUSE     FLIPPED=REUSE   cover.out=1787165011.998066957 stamp=1787165011.998358898
  /bin/dash  CURRENT=REUSE     FLIPPED=REUSE   cover.out=1787165012.040864503 stamp=1787165012.041082833
  /bin/bash  CURRENT=REUSE     FLIPPED=REUSE   cover.out=1787165012.075499438 stamp=1787165012.075708298
  /bin/zsh   CURRENT=NO-REUSE  FLIPPED=REUSE   cover.out=1787165012.116894756 stamp=1787165012.117006963
  /bin/ksh   CURRENT=NO-REUSE  FLIPPED=REUSE   cover.out=1787165012.175830209 stamp=1787165012.175947323
```

"dash is the precise one" is **false as a general statement**: macOS's own `/bin/dash`
truncates to whole seconds exactly like `/bin/sh` and `/bin/bash`. What varies is whether a
given shell's `test` builtin reads the seconds or the nanoseconds field of `stat` *as built
for that platform*. On darwin only `zsh` and `ksh` read nanoseconds. A regression matrix of
`{sh, dash, bash}` is therefore green on darwin with the defect fully live.

### A measurement trap worth recording

A *synthetic* zero-gap arrangement (`printf > cover.out; : > stamp` back to back) is **not**
a faithful model of production. Under dash on the container filesystem it ties 191 times out
of 200, so the backwards check reports REUSE 95.5% of the time and the defect appears not to
reproduce. The production gap spans a whole `go` process exit (~5ms), which exceeds the
filesystem's ~1ms timestamp granularity, and the reproduction is then deterministic. Any
future probe of this mechanism must arrange the state through `test-produce.sh`, not by hand.

### Option (a) is refuted by a reachable state

Presence-of-stamp-plus-`cover.out` is not sufficient, because the `rm -f "$stamp"` consumption
only runs *if the producer runs at all*:

1. Gate invocation A runs project-wide; `test-produce.sh` writes a complete `cover.out` and
   touches the stamp.
2. Invocation A terminates between the test dispatch and the coverage dispatch — Ctrl-C, a CI
   job timeout, an OOM kill. The stamp survives. `.backstop/go-coverage-fresh` is gitignored,
   so nothing surfaces it in `git status`.
3. Invocation B runs `./bin/backstop gate --file <path>` **explicitly** (not the bare
   diff-scoped default — this issue already rules out diff mode as the narrowing path).
   `go-test` declares `package_scoped: true`, so the dispatch narrows to the changed packages
   and overwrites `cover.out` with a **partial** profile. `test-produce.sh`'s `./...` guard
   does not match, so invocation B writes no stamp; A's leftover is still the only stamp.
4. `coverage-produce.sh` now sees a stamp and a `cover.out`.

Under option (a) step 4 reuses a **partial** profile: every unmeasured file reads as absent and
the coverage dimension returns an incomplete measurement with nothing red — precisely the
silent-narrowing outcome `test-produce.sh`'s own `./...` guard exists to prevent. Option (b)
refuses this state correctly (stamp older than profile → no reuse).

Option (a) would also require deleting a shipped guard: `pkg/pack/engine/gotoolchain_installed_pack_singlerun_test.go`
asserts the installed producer carries an `-ot` freshness comparison. Dropping the comparison
reds that assertion, and the only way to green it is to delete it.

### The residual option (b) does not close

If invocation A leaves both a stamp and a **complete** `cover.out`, and invocation B does not
overwrite `cover.out`, option (b) reuses A's profile. Two ways to reach it:

- **Build-broken**: B's `go test` fails to compile, so no `-coverprofile` output is written.
  The gate is already failing.
- **Green-verdict, and this one has no red anywhere** — confirmed in code, not derived from the
  manifest. `gate --file <non-Go paths>`: `fileModeTestTargets` returns state (C)
  `fileModeClaimsNothing` and `runFindingsEngine` **returns early before `runner.RunStdout`**
  (`cmd/backstop/pack_gate.go:660-676`), so `go-test` is genuinely not dispatched and `cover.out`
  is untouched. The skip is reported via `dispatchAdvisory`, whose `Severity` is `"warning"`
  (`pack_gate.go:977-986`) — **non-blocking** by the pack severity contract. Meanwhile
  `go-coverage` is `scope_kind: project-wide` and *not* `package_scoped`, so it dispatches
  regardless, finds A's surviving stamp and A's untouched complete profile, and reports a
  **green** verdict over a measurement of a tree that may since have changed.

What still bounds it: the reused profile is always **complete**, never partial (option (b)
refuses the partial case above); and it requires a prior invocation to have aborted abnormally
between two dispatches. Closing it entirely would mean an unconditional `rm -f cover.out` at the
head of `test-produce.sh`, which changes that script's semantics for every consumer and belongs
in its own issue. It is recorded here as a known, bounded edge case, not fixed.

### Version reasoning

The bump is a **minor**, `v1.8.0`. The precedent is a shipped comment in
`pkg/pack/engine/gotoolchain_installed_pack_singlerun_test.go`: a fix that "changes what a
shipped engine executes ... is a behaviour change in a released rule path, not a patch." This
fix changes what a shipped engine executes by strictly more than ISSUE-172's did — after it, the
coverage engine stops running a whole test suite it currently always runs.

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
