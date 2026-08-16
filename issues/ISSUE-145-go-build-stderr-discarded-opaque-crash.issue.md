---
title: "go-toolchain go-build engine discards go build's stderr — a real compile error surfaces as an opaque, content-free crash message"
schema_version: issue/v1

issue:
  id: ISSUE-145
  title: "go-toolchain go-build engine discards go build's stderr — a real compile error surfaces as an opaque, content-free crash message"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# go-toolchain go-build engine discards go build's stderr — a real compile error surfaces as an opaque, content-free crash message

## Problem

`go build ./...` writes compiler diagnostics to **stderr**, not stdout. Confirmed directly:

```
$ go build ./... 1>out.txt 2>err.txt
$ cat out.txt   # empty
$ cat err.txt
# gobuildtest
./main.go:4:17: cannot use 5 (untyped int constant) as string value in variable declaration
```

The `backstop-ai/go-toolchain` pack's `go-build` engine binding
(`.backstop/packs/backstop-ai/go-toolchain/pack.yml:70-79`, `command: go build`, `input_mode:
none`, `convert: scripts/build-to-sarif.sh`, `crash_guard: true`) is dispatched through
`runFindingsEngine` (`cmd/backstop/pack_gate.go:727`), which invokes it via
`runner.RunStdout(...)` — `pkg/check/runner.go:52`'s `ExecCommandRunner.RunStdout`, which
captures **only stdout** into an explicit buffer so a tool's stderr banner/progress "cannot
corrupt the SARIF bytes" (REQ-009 / CLM-028, by design for rule-fed engines whose SARIF payload
IS their stdout).

`scripts/build-to-sarif.sh`'s own header comment confirms the assumption that breaks here: it
"reads raw `go build ./...` stdout on stdin." The convert step (`pack_gate.go:784`) feeds it
exactly that stdout buffer. For `go-build`, that buffer is **always empty on a real compile
failure**, because the diagnostic text `go build` produces never reaches stdout at all — it is on
stderr, and `RunStdout` never captures stderr.

Consequence, traced through `pack_gate.go`:
1. `stdout` (captured) is empty; `runErr` is non-nil (`go build` exited 1).
2. `build-to-sarif.sh` receives empty stdin, emits `{"version":"2.1.0","runs":[{"results":[]}]}` —
   valid, empty SARIF.
3. `check.ParsePackFindings` parses it to zero violations.
4. The crash-vs-findings guard (`pack_gate.go:804`, `binding.CrashGuard && runErr != nil &&
   len(checkViolations) == 0`) fires, because it can't distinguish "the tool crashed with no
   output" from "the tool failed and wrote its explanation to a stream we didn't read."
5. The gate reports: `pack backstop-ai/go-toolchain engine "go build" crashed: non-zero exit with
   no parseable findings: exit status 1` — zero compiler diagnostic text, zero file/line, zero
   indication of what actually broke.

This is unconditional, not merely a transient/shared-tree artifact: **any** real `go build`
compile failure under the current `backstop-ai/go-toolchain` pack (confirmed at the installed
version, `pack.yml` header `version: "1.5.0"`) hits this path, because `go build`'s error-reporting
channel is categorically stderr. It was first observed during `PLAN-ISSUE-124` implementation
(2026-08-16) as an inherited, pre-existing gate failure — confirmed present in the pre-edit
baseline gate run, before any of that plan's own changes, and confirmed NOT a genuine break by
running `go build ./...` directly (exit 0).

## Impact

A real compile break becomes undebuggable from the gate's own output alone. The gate does still
correctly go RED — this is not a false-pass/vacuous-green defect, and the crash-guard mechanism
(SPEC-034 REQ-003/CLM-010) is doing exactly the job it was built for: refusing to read a
zero-findings non-zero-exit as a silent clean pass. But the message it produces carries none of
the diagnostic content a developer or agent needs to act on it. Anyone hitting this must
independently re-run `go build ./...` by hand to see what actually broke — the gate's own report
is a dead end. This is diagnostic-QUALITY debt on a verdict the gate already computes correctly,
not a wrong-verdict defect: the same "wrong data on a correct verdict" shape DIR-024 already
carries for ISSUE-135 (go-test converter's bare-basename `File`), not the "gate computes/reports
the wrong pass/fail result" shape DIR-032 (Gate Verdict Honesty) charters.

## Direction

Merge or additionally capture stderr for `go-build`'s dispatch so the compiler's real diagnostic
text reaches `build-to-sarif.sh` and, in the crash case, the error message. Two shapes worth
weighing, kept as constraint not design:

- Declare a pack-level `producer` script for `go-build` (the mechanism `pack_gate.go:680-725`
  already supports and documents as existing precisely "because a tool splits its real output
  across streams the runner's deliberate stdout-only capture cannot both see" — currently used
  only by the coverage engine's `coverage-produce.sh`). A producer runs UN-SANDBOXED in place of
  the plain command and can merge `go build`'s stdout+stderr itself before handing bytes onward,
  the same way `runner.Run` (`CombinedOutput`) already does for the build/test executors outside
  the findings-engine path.
- Or have `build-to-sarif.sh` itself invoke `go build ./... 2>&1` internally (as its OWN
  subprocess) rather than relying on the dispatcher to hand it output at all — a bigger shape
  change to the `input_mode: none` contract and worth comparing against the producer option
  before choosing.

Whichever shape is chosen, the crash-guard message (`pack_gate.go:805`) should carry the captured
diagnostic text (or at minimum, the fix should make the SARIF conversion see the real compiler
output so a genuine compile break produces a located finding instead of falling through to the
crash-guard branch at all).

Fix lives in the pack repo (`backstop-ai/go-toolchain`) — version bump + relock, the same shape as
this directive's other go-toolchain pack-side precision fixes (ISSUE-129, ISSUE-135), not a
backstop-core code change; `pack_gate.go`'s producer mechanism the fix would use is already shipped
core-side and requires no core change to adopt.

## Notes

- **Not a duplicate of ISSUE-067** (`go-toolchain go-test Engine Reports Test Failures As an
  Opaque Crash, Not Parseable Findings`, DIR-032 item 2). ISSUE-067 is a different engine
  (`go-test`) with a different root cause: per that issue and DIR-032's own description, `go
  test`'s `--- FAIL:` output IS written to stdout (unlike `go build`'s diagnostics), and the
  reported defect is that `scripts/test-to-sarif.sh` fails to extract it into findings before the
  exit code is judged — an extraction bug, not a stream the dispatcher never captured. This issue
  is a genuinely distinct defect: for `go-build`, the diagnostic text is never even offered to the
  converter, because it was written to a channel (`stderr`) the dispatch layer discards by design
  for every findings engine (REQ-009/CLM-028).
- **Checked for siblings beyond go-build/go-test.** Of the three `crash_guard: true` bindings
  installed in this repo, the third (`go-arch-lint`,
  `.backstop/packs/backstop-ai/backstop-core-architecture/pack.yml:16`) writes its findings as
  native JSON to stdout by the tool's own `--output-type json` contract — no reliance on a stream
  the dispatcher doesn't capture — so it does not share this defect. No other installed engine
  binding declares `crash_guard: true`.
- **Directive fit**: cited under DIR-024 (Gate/Engine Quality), not DIR-032 (Gate Verdict
  Honesty) — checked both directives' current charters before filing. DIR-032's own text is
  explicit that its cluster is "a gate step computes a result internally but reports the wrong
  verdict about it"; this defect's verdict (RED) is correct, only the diagnostic content is
  missing. DIR-024 item 15 (ISSUE-135) draws exactly this line for the same pack ("a violation
  that IS reported and DOES correctly redden the gate, just with an imprecise/ambiguous location
  string — wrong data on a correct verdict, this directive's charter, not DIR-032's 'wrong
  verdict about a computed result'"); this issue's "zero diagnostic content" is the same shape,
  more severe in degree (empty rather than imprecise) but not in kind.
- Confirmed live in the tree 2026-08-16: `go build ./...` writes nothing to stdout and its full
  diagnostic to stderr (direct repro, see Problem section); `pkg/check/runner.go:46-63`'s
  `RunStdout` captures only stdout by explicit design; `scripts/build-to-sarif.sh` documents that
  it reads stdin expecting `go build`'s stdout; `pack_gate.go:804-806` is the crash-guard branch
  that fires and produces the opaque message.
- Repro sketch for whoever picks this up: any tree with a real Go compile error, gated with
  `backstop-ai/go-toolchain`'s `go-build` engine at the current pack version. Expected today:
  `pack backstop-ai/go-toolchain engine "go build" crashed: non-zero exit with no parseable
  findings: exit status 1`, no file/line/message. Expected after fix: a located SARIF finding
  naming the file, line, and compiler message — or, if a genuine crash-guard case remains
  reachable, an error message that includes the captured diagnostic text rather than none.
