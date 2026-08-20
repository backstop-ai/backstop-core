---
title: "Sandbox Devnull Write Denied Breaks Idiomatic Scripts"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-168

issue:
  id: ISSUE-168
  title: "Sandbox Devnull Write Denied Breaks Idiomatic Scripts"
  type: bug
  status: closed
  created: "2026-08-18"
  closed: "2026-08-19"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Sandbox Devnull Write Denied Breaks Idiomatic Scripts

## Problem

On real Linux CI (run `32108003542`, `gate-report.json`, `pack_engines` step, confirmed at
commit `970512b`), two `pkg/packval` tests fail because a sandboxed script's write to
`/dev/null` is denied by the Landlock sandbox:

- **`TestLinuxSandbox_RealInterpreterRunsUnderTheFilter`**
  (`pkg/packval/sandbox_linux_exec_test.go:437`) — its convert-script fixture,
  `pkg/packval/testdata/sandbox/convert-jq.sh`, fails outright:
  ```
  a real interpreter could not run a real convert script under the sandbox: sandboxed run
  (stdout) failed: exit status 127: .../convert-jq.sh: 14: cannot create /dev/null: Permission
  denied
  ```
  Line 14 of that fixture is `if command -v jq >/dev/null 2>&1; then` — a completely standard,
  idiomatic POSIX shell existence-check pattern.
- **`TestLinuxSandbox_NetworkAllowedControlLegSucceeds`**
  (`pkg/packval/sandbox_linux_exec_test.go:310`) — its dynamically-generated
  `network-probe.sh` fixture hits `/dev/null: Permission denied` on two lines, visible in its
  captured stderr:
  ```
  .../network-probe.sh: line 4: /dev/null: Permission denied
  TCP_BLOCKED
  .../network-probe.sh: line 5: /dev/null: Permission denied
  UDP_BLOCKED
  PROBE_COMPLETED
  ```
  This test's actual failure mode, distinct from the `/dev/null` denials, is that TCP and UDP
  were both blocked under a capability that PERMITS the network (`TCP_OPEN`/`UDP_OPEN` markers
  absent from the report — the test's `t.Errorf` fires on that condition, not on the `/dev/null`
  lines). The `/dev/null` denials are visible in the captured output and are a real, independent
  instance of this same defect, but they are NOT what makes this particular test fail; that
  distinction matters for whoever fixes the network-blocking failure separately.

### Root cause, fully traced

Both sandbox profiles deny ALL file writes with no exception carved out for `/dev/null`, by
deliberate design, confirmed directly by reading the source:

- `pkg/packval/sandbox_capability.go` — `ConvertValidatorCapability` (used by both tests above)
  sets `WritablePaths: nil` unconditionally for convert/validator work, with the comment: "EMPTY
  for convert/validator work: darwin denies file-write* outright and parity is the spec."
- `pkg/packval/sandbox_nonlinux.go` — the darwin `sandbox-exec` profile literal is:
  ```
  (version 1)(import "bsd.sb")(deny default)(allow process*)(allow file-read*%s)(deny network*)(deny file-write*)
  ```
  a blanket `(deny file-write*)` with no `/dev/null` carve-out, structurally identical in intent
  to Linux's `WritablePaths: nil`.

So this is **not** a Linux-specific regression or a platform asymmetry in what the profiles were
designed to allow — both platforms encode the identical "deny all writes" decision, and neither
profile literal has an explicit `/dev/null` exception. It was simply never exercised by CI
before tonight (`ISSUE-163`'s `TestMain` fix is what let Linux CI reach these tests for the
first time; it did not cause this defect).

### Whether darwin is also, silently, broken the same way — checked directly

`sandbox_linux_exec_test.go:422-423` names
`TestLinuxSandbox_RealInterpreterRunsUnderTheFilter` as "the Linux analogue of darwin's
`TestSandboxConvertWithRealInterpreter`" (`pkg/packval/sandbox_realconvert_test.go:52`) — and
both tests drive the exact same fixture file, `convert-jq.sh`, containing the same
`>/dev/null 2>&1` idiom on line 14. Run directly on this darwin machine:

```
$ go test ./pkg/packval/ -run TestSandboxConvertWithRealInterpreter -v
=== RUN   TestSandboxConvertWithRealInterpreter
--- PASS: TestSandboxConvertWithRealInterpreter (0.04s)
PASS
```

It passes. So the design gap — a blanket "deny all writes" profile literal with no explicit
`/dev/null` exception on either platform — is real and identical on both platforms at the
SOURCE level, but only Linux's Landlock actually enforces it against `/dev/null` writes in
practice; macOS's `sandbox-exec`, despite having no explicit carve-out for it either, evidently
lets writes to `/dev/null` through as an emergent property of how Seatbelt treats the character
device (not something asserted anywhere in the profile text). This was confirmed empirically,
not assumed — state this precisely: the profile-literal gap is symmetric, the OBSERVED breakage
is Linux-only.

## Impact

`command -v foo >/dev/null 2>&1` and equivalents (existence checks, suppressing expected stderr
noise with `2>/dev/null`) are an extremely common, idiomatic pattern in real-world shell
scripts. Any real pack author's convert or validator script that uses this idiom will break
under backstop's Linux sandbox, with a confusing `exit status 127` / "Permission denied"
signature rather than anything that points at the sandbox as the cause. Writing to `/dev/null`
is universally considered safe and harmless — unlike writing to an arbitrary file, granting it
would not weaken the sandbox's actual security properties (no data escapes, no state persists,
no side effect is observable). This is a genuine capability gap worth closing, not a
security-relevant restriction working as intended.

## Resolution

Fixed across two commits, delivered by `PLAN-ISSUE-168` (`status: completed`). Both sandbox
platforms (Linux Landlock, macOS `sandbox-exec`) denied all writes with no `/dev/null` exception,
breaking the extremely common `command -v foo >/dev/null 2>&1` shell existence-check idiom in any
pack's convert script.

- `4f3a810` — the primary fix: a narrow, explicit `/dev/null` write exception on both platforms.
  Landlock via `landlockDevNullRights()` on Linux; `sandbox-exec` via an explicit `(allow
  file-write* (literal "/dev/null"))` rule on darwin.
- `23b08ac` — a tracked, inline-justified coverage-measurement exclusion for
  `pkg/packval/sandbox_nonlinux.go`, which is `//go:build !linux` and structurally unmeasurable
  on CI's Linux-only runners. Explicitly a legitimate declared exclusion, not a waiver.

**Honest verification ceiling.** The Linux-specific mandated test
(`TestLinuxSandbox_DevNullWriteIsPermittedAndOtherWritesAreNot`) is `//go:build linux` and cannot
run on darwin at all — real Linux CI is the only confirmation, and that confirmation exists.

Confirmed genuinely proven, not just locally verified: real Linux CI runs `32314302525` and
`32315586649`, both `conclusion: success` on `main`.

## References

- `pkg/packval/sandbox_linux_exec_test.go:437` — `TestLinuxSandbox_RealInterpreterRunsUnderTheFilter`.
- `pkg/packval/sandbox_linux_exec_test.go:310` — `TestLinuxSandbox_NetworkAllowedControlLegSucceeds`.
- `pkg/packval/testdata/sandbox/convert-jq.sh:14` — `if command -v jq >/dev/null 2>&1; then`,
  the failing line, shared verbatim by both the Linux and darwin real-interpreter tests.
- `pkg/packval/sandbox_capability.go` — `ConvertValidatorCapability`, `WritablePaths: nil`.
- `pkg/packval/sandbox_nonlinux.go` — the darwin `sandbox-exec` profile literal,
  `(deny file-write*)` with no `/dev/null` exception.
- `pkg/packval/sandbox_realconvert_test.go:52` — `TestSandboxConvertWithRealInterpreter`, the
  darwin analogue; confirmed passing locally against the same fixture, establishing the
  platform-asymmetric OBSERVED behavior despite symmetric profile design.
- CI run `32108003542`, `gate-report.json` (`git_sha: 970512b`), `pack_engines` step — source of
  both failure messages quoted above.
- `ISSUE-163` / commit `970512b` — the `TestMain` fix that let Linux CI reach these tests for
  the first time; not the cause.

A fix likely needs an explicit, narrow allow-rule for `/dev/null` specifically on both
platforms — a Landlock path rule scoped to `/dev/null` on Linux, and a corresponding
`sandbox-exec` profile addition (e.g. an `(allow file-write* (literal "/dev/null"))` clause) on
darwin, for genuine platform parity between the profiles' stated intent and their actual
behavior. The specific mechanism is left to this issue's plan.

### Existence-in-world check

Performed 2026-08-18 before authoring: `grep -ril` over `issues/` and `bundles/` for "dev/null"
and "devnull" matched no open issue or bundle charter — only this issue's own not-yet-authored
file. No duplicate ownership of this surface exists.

## Verification ceiling — fix landed, awaiting CI confirmation (2026-08-18)

**The fix has landed on `main`, across two commits. This issue stays `open` — do not read this
note as a close.**

- `4f3a810` — the primary fix, darwin and Linux code together: darwin's `sandbox-exec` profile
  literal (`pkg/packval/sandbox_nonlinux.go`) gained `(allow file-write* (literal "/dev/null"))`
  appended after the existing blanket `(deny file-write*)`; Linux (`pkg/packval/sandbox_capability.go`)
  gained one appended Landlock path rule for `/dev/null` with rights
  `READ_FILE|WRITE_FILE|TRUNCATE|IOCTL_DEV`.
- `23b08ac` — a follow-on coverage-exclusion fix, not a behavior change: CI flagged
  `pkg/packval/sandbox_nonlinux.go` as unmeasurable (`//go:build !linux` means it never compiles
  on CI's `ubuntu-latest` runners, so it can structurally never produce a coverage record there),
  and this commit added it to `.backstop/coverage-exclusions` following the exact existing
  precedent already recorded for `sandbox_linux_helper.go`'s mirror-image gap (that file is
  `//go:build linux`-excluded from darwin's own coverage runs for the analogous reason).

### Darwin — verified for real; Linux — compile/link only, kernel behavior awaits CI

The **darwin half was verified behaviorally on a real macOS host**: real `sandbox-exec` runs,
both by the implementer and independently by the impl-reviewer, confirmed the `/dev/null` write
now succeeds under the production profile while an ordinary write to a sibling path in `packDir`
stays denied under the same profile — narrow and non-widening, confirmed directly rather than
inferred.

The **Linux half is only compile+link verified** (`GOOS=linux GOARCH=amd64 go test -c -o /dev/null
./pkg/packval/`) plus the pure-derivation arithmetic (the rights mask and rule-uniqueness tests,
which run natively on darwin since they touch no Landlock syscall). The kernel's actual verdict
on the rule — whether Landlock genuinely accepts and enforces it — is unverifiable on this
machine (no Landlock-capable kernel available locally) and awaits CI.

### Partial real CI confirmation already obtained, as of this writing

CI run `32142326172` (`gate-report.json` downloaded and read directly), for the pre-coverage-fix
commit `4f3a810`, was checked against the specific failure signature this issue documents:
`.scope.files` includes `pkg/packval/sandbox_capability.go`, `sandbox_capability_test.go`,
`sandbox_darwin_test.go`, `sandbox_devnull_test.go`, `sandbox_linux_exec_test.go`, and
`sandbox_nonlinux.go` — so `pkg/packval` was genuinely in scope for this run, not merely expected
to be. No violation in the report's `backstop-ai/go-toolchain/go-test` results mentions
`cannot create /dev/null`, `/dev/null: Permission denied`, or `exit status 127` in the packval
channel — the specific `/dev/null` failure signature is confirmed **GONE** from this run's
violations. (The run's overall conclusion was still `failure`, for unrelated reasons — most
visibly a `coverage_unmeasured` violation on `sandbox_nonlinux.go`, which is exactly the gap the
follow-on commit `23b08ac` was written to close.)

**What this does and does not establish:** this is real, partial confirmation that the `/dev/null`
denial signature is gone from a genuine CI run with the package in scope. It does **not** yet
confirm the coverage-exclusion follow-up commit (`23b08ac`) — that commit has not itself been
through a fresh CI run as of this writing. It also does not, on its own, adjudicate the separate
network-blocking failure in `TestLinuxSandbox_NetworkAllowedControlLegSucceeds`; see the note
below.

State plainly: the primary `/dev/null` fix is real-CI-confirmed working; the coverage-exclusion
follow-up awaits its own CI run. Do not close this issue.

## Open premise correction owed to DIR-024 (not this lane's to hand-edit)

This issue's own "Root cause, fully traced" section above states, as a settled distinction: "the
`/dev/null` denials are visible in the captured output and are a real, independent instance of
this same defect, but they are NOT what makes this particular test fail" (referring to
`TestLinuxSandbox_NetworkAllowedControlLegSucceeds`). Local investigation during this fix's
implementation produced a real, platform-independent measurement (documented in
`plans/PLAN-ISSUE-168-sandbox-devnull-write-allowance.plan.yml` as "M7") that makes this doubtful:
`networkProbeBody` (`pkg/packval/sandbox_linux_exec_test.go`) attaches `2>/dev/null` to the shell
`if` CONDITION itself —

```
if (exec 3<>/dev/tcp/127.0.0.1/%d) 2>/dev/null; then echo TCP_OPEN; else echo TCP_BLOCKED; fi
```

— and a real-bash probe run locally against an OPEN loopback port showed that an unopenable
redirect target on an `if` condition makes the shell take the ELSE branch WITHOUT ever attempting
the command inside — reproducing the CI output's exact shape (`TCP_BLOCKED`/`UDP_BLOCKED` with no
socket ever attempted). If that holds, both failing tests may share the single `/dev/null` root
cause this issue fixes, rather than the network-blocking test hiding a second, independent
defect.

**This was left, correctly, as an unconfirmed hypothesis pending a real CI read — not asserted as
fact.** It is recorded here rather than resolved because a real CI run against the fix is what
adjudicates it: control leg (`TestLinuxSandbox_NetworkAllowedControlLegSucceeds`) GREEN once the
`/dev/null` fix lands means this issue accounted for both failures and nothing further needs
filing for the network; control leg still RED with the `/dev/null` stderr lines gone means a
genuine, previously-masked network-permission defect exists and needs its own issue, filed with
the newly-clean evidence.

**`directives/DIR-024-gate-engine-quality.directive.md`, lines ~1241-1247 (item 23), currently
states the ORIGINAL, now-uncertain claim verbatim as settled fact** — the same sentence this
issue's own "Root cause" section carries above. DIR-024 is the committed backlog layer a future
session reads FIRST, before any individual issue, so leaving it stale there defeats the purpose
of correcting it here: a future reader would still be sent after a defect that may not exist,
just through a different door.

**DIR-024 is a directive, not an issue, and it is currently being actively modified in the
working tree by the backlog-pm lane** (`git status` at authoring time showed
`M directives/DIR-024-gate-engine-quality.directive.md`, from that lane's ongoing auto-triage of
other same-night filings). Per this repo's routing rules, directives are never hand-edited by an
issue-authoring lane. This note exists so the obligation is not silently lost: **once a real CI
run confirms or refutes the M7 hypothesis above, DIR-024 lines ~1241-1247 need the same
correction this issue's own text would need if it turned out wrong** — a backlog-pm-owned
correction to route through that lane (or whatever mechanism supersedes it), not one for a future
session to hand-edit directly.
