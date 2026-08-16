---
title: "Gate reports full PASS while a mandated test genuinely fails: no gate dimension runs the Go suite to a verdict when only test files change"
schema_version: issue/v1

issue:
  id: ISSUE-118
  title: "Gate reports full PASS while a mandated test genuinely fails: no gate dimension runs the Go suite to a verdict when only test files change"
  type: bug
  status: closed
  created: "2026-08-11"
  closed: "2026-08-16"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical

delivered_by: PLAN-ISSUE-118
---

# Gate reports full PASS while a mandated test genuinely fails

## Resolution

Delivered by PLAN-ISSUE-118
(`plans/PLAN-ISSUE-118-gate-blind-spot-test-only-diffs.plan.yml`), shipped in commit
`4dbf64b` — an overnight batch that also carried ISSUE-112, ISSUE-113 and ISSUE-122.

**This issue's own root-cause analysis was wrong in mechanism, right in conclusion.**
Planning measured it at HEAD: `pkg/gate/step_coverage.go` holds no `go test`
invocation at all (SPEC-042 eradicated the baked Go coverage analyzer), and
`dispatchPackEngines` runs the go-toolchain pack's project-wide `go-test` engine on
*every* diff shape. The suite was always running. The failure was **computed
correctly and then thrown away**: the resulting violation was dropped by
`filterViolations` (`pkg/gate/scope.go`), because `go test` reports failure positions
as a **bare basename** (`widget_test.go`) while the scope's file set holds
canonicalized repo-relative paths (`pkg/widget/widget_test.go`). They can never
match. The reported path was unresolvable, so the verdict was silently filtered —
not starved. That converter behavior is now separately filed as ISSUE-135.

**Fix — a widened `test_verification`, not a new gate dimension:**

- **`gate.Violation.GateType`** (`pkg/gate/result.go`) — every dispatched pack
  finding now carries its producing binding's *declared* `gate_type`, stamped at
  `cmd/backstop/pack_gate.go` beside the existing `ProjectWide` stamp. It is
  `json:"-"` and deliberately **absent from baseline identity**, so no existing
  baseline entry was perturbed.
- **`pkg/gate/test_verdict_join.go`** (new) — `RouteTestVerdictFindings` selects the
  test-verdict subset **by declared gate type alone** (never a pack name, rule name,
  or message sniff — the ISSUE-064 rule), and `MandatedTestFailures` joins it to the
  due mandated tests **by boundary-anchored test name**. The name join is
  path-independent by construction: it is immune to the bare-basename problem, and
  works even for a failure with no reported location at all.
- **`mandated_test_failed` at severity `critical`** — blocking (`blocksVerdict` is
  `!EqualFold(Severity,"warning")`) *and* non-waivable, because `critical` is the
  only route by which a **core-emitted** rule can reach the production waiver
  policy's non-waivable set (`waiver.NewDeclaredPolicy(rules, []string{"critical"})`);
  that set's rule list is harvested from pack manifests only, and there is no
  `non_waivable` key in `backstop.yml` for core rules. No config edit was made or
  needed. The violation is attributed to the **mandated test's own resolved file**
  (falling back to its spec file), never to the finding's unresolvable path.
- **`cmd/backstop/gate.go`** — an *unfiltered* violation collector threaded from
  `packValidatorStep` into the widened `test_verification` step, so the verdict
  stream reaches the join before `FilterViolations` can drop it. `pack_engines`' own
  reported output stays filtered exactly as before.
- **Capability-absent advisory** — when due mandated tests exist but no installed
  pack declares a `gate_type: test` engine, `test_verification` now surfaces a
  distinct **non-blocking** advisory naming what is missing, instead of an
  unqualified pass. Un-adopted capability warns; a broken promise blocks.
- **`pkg/gate/step_coverage.go`** — the "no in-scope files to measure" early return
  stays a non-blocking pass but now **names what it did not do**, so the skip this
  issue misread as the root cause can never again read as a test verdict.

**The discovery-capability-absent path is deliberately left UNSCOPED.**
`MandatedTest.FilePath` is populated below the test-discovery guard, so above it
`FilePath` is always `""` and `GateScope.Contains("")` is false in diff mode.
Applying the surrounding step's scope guard there would keep a mandated test only
when its *spec file* was in the diff — and in the all-test-file diff this issue is
about, the spec file is not in the diff, so the guard would silently eat the very
defect it sits inside the fix for. The verdict join therefore runs project-wide on
that path regardless of scope mode. The accepted residual is **noise, not silence**:
on a narrow diff this path may report a failing mandated test whose files are
nowhere near the diff, with degraded (spec-file) attribution. That trade is
deliberate and is recorded in the plan's sharp edge 15.

**Verification — the regression fixture this issue demanded (Direction item 3):**

- `TestE2E_GateRedsOnFailingMandatedTestInTestOnlyDiff`
  (`cmd/backstop/gate_test_only_diff_verdict_e2e_test.go`) — the acceptance surface:
  a diff whose changed files are *entirely* test files, with a genuinely failing
  mandated test, drives the assembled gate to a **non-zero exit**. Recorded RED
  before the fix and GREEN after.
- `TestE2E_VerdictCapabilityAbsentIsAdvisoryNotSilentPass` — the paired
  no-verdict-engine case, driven by a committed `test-verdict-absent-e2e` workspace
  fixture declaring no `gate_type: test` binding, so the capability signal is
  genuinely pack-derived rather than seam-injected.
- `TestMandatedTestFailures_BareBasenamePathStillBlocks` and
  `TestMandatedTestFailures_BlocksOnNamedFailingMandatedTest` — the path-independence
  falsifiers, fed **real converter output**: the committed `go test` capture at
  `cmd/backstop/testdata/go-toolchain/fixtures/go-test-failures.txt` piped through
  the committed `test-to-sarif.sh`, not hand-assembled JSON. That capture carries
  both load-bearing shapes — bare-basename positions for a subpackage, and one
  failure (`TestNoPos`) with no location whatsoever.
- `TestMandatedTestFailures_BoundaryAnchoredNameMatch` — both collision directions
  (`TestWidget` vs `TestWidgetFrobnicate`; `TestGadgetSpinner` vs `TestGadgetSpin`)
  plus a positive control, so the anchoring is neither too loose nor too tight.
- `TestMandatedTestFailures_PassingSuiteYieldsNoViolations` and
  `TestTestVerification_NamePresenceBehaviorUnchanged` — the non-vacuity and
  non-regression pins: the join cannot be satisfied by a constant, and existing
  name-presence verification, mandated-test scoping and baseline identity are
  unchanged.
- All **sixteen** mandated test names re-run green at close-out with zero skipped
  subtests; full `go test ./... -race` reported 17 packages green in the delivering
  commit.

**Direction items 1 and 2, as decided.** Item 1 (make `coverage_threshold` run the
suite when the diff is all test files) was **rejected and must not be revisited** —
it is unimplementable without putting a `go test` invocation back inside core, which
is the baked-language-knowledge eradication SPEC-042 completed and CLAUDE.md's first
principle forbids. Item 2 was resolved as **widen, don't add**: `test_verification`
already extracts mandated tests, resolves their paths, scopes them and owns a
capability advisory; it gained one join. Core still runs nothing itself — the verdict
is produced by a pack engine, as the thin-executor principle requires.

**Scope notes — what this close does NOT cover.**
- **ISSUE-129 remains OPEN and untouched.** A failing test whose file is *not* in the
  diff is its surface, deliberately not absorbed here. Its go-toolchain pack-data
  half (`exempt_from_scope_filter`, relock v1.4.0) shipped separately at `2a18148`;
  because this lane's acceptance test seams the dispatch, that flip was structurally
  invisible to it.
- **ISSUE-135** (converter emits bare-basename `File`) is the pack-side correction of
  the mechanism this issue's reproduction tripped over. The fix here was designed to
  work without it.
- **ISSUE-112** (a missing engine tool producing empty SARIF) is a different failure
  mode on a different axis, fixed in the same batch commit under its own plan.
- A **spec-author follow-up already landed in the same commit**: `gate.Violation.GateType`
  falsified fourteen pieces of settled prose — nine spots in SPEC-037 (eight of them
  flat "the field does not exist" assertions, not mere parentheticals) plus five code
  comments across three `pkg/gate` files — and all were corrected via spec-author,
  never hand-edited, along with SPEC-031's provisioning claims. CLM-024's mandated
  test name survived unchanged; ISSUE-064's pre-existing rule-ID-vs-property drift was
  disclosed but not swept in.

## Problem

`backstop gate`'s core promise is that a green run means the mandated tests actually pass. That
promise is false in a reproducible, general case: when a change touches only `_test.go` files (no
production Go changed), **no gate dimension actually executes the Go test suite to completion and
reads its real pass/fail verdict** — a genuinely failing test can sit in the tree indefinitely with
`backstop gate` reporting green, as long as nobody happens to run `go test` by hand.

## Reproduction

Measured on the same tree, with a freshly-rebuilt `bin/backstop` binary, during implementation of
`plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml` (2026-08-11):

```
$ go test ./cmd/backstop/... -race -run TestCIRecipes
--- FAIL: TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles
FAIL
exit status 1
```

```
$ ./bin/backstop gate
10 passed, 0 failed
exit 0  (full PASS)
```

Same working tree, same commit, same binary. One command says a mandated test is red; the other
says everything is green.

## Root cause — three gate dimensions, none of which run the suite here

- **`test_verification`** (`pkg/gate/step_testverify.go`) only checks that a mandated test's NAME is
  present in source (`ExtractMandatedTests` + `ResolveMandatedTestPaths`). It never invokes `go
  test` and never reads a pass/fail verdict — a test that exists and is named correctly satisfies
  this step whether it passes, fails, or panics.
- **`test_substantiveness`** only checks that a mandated test's body contains real assertions (not
  a stub/vacuous test). It also never executes the test — a test with a genuine assertion that
  currently evaluates false still satisfies this step.
- **`coverage_threshold`** (`pkg/gate/step_coverage.go`) is the ONE gate dimension that actually
  invokes `go test` — but in this specific, general case it never got the chance to: when the
  scoped change touches only `_test.go` files and no in-scope production file, the step exits early
  with `Status: "pass", Reason: "no in-scope files to measure for coverage"`
  (`pkg/gate/step_coverage.go:98`) — it silently SKIPS running the suite rather than running it to a
  verdict. A change that is 100% test-file diff (exactly the shape of the SPEC-067 fix that failed
  to compile/pass) hits this skip path every time.

Net effect: for any diff whose changed files are entirely `_test.go`, the three gate dimensions
that touch "did the test pass" either don't execute the test at all (`test_verification`,
`test_substantiveness`) or explicitly decline to because there's "nothing to measure"
(`coverage_threshold`). Nothing in the kill-chain runs `go test ./...` to completion and asks
"did it pass" for this shape of change.

## Why this matters

This is not cosmetic — it undermines the gate's central claim across this whole project (see
CLAUDE.md: "Verify, don't assert... running the real command... and reading the result"). A test-
only commit (e.g. a plan's final "make the failing test pass" step, or any test-hardening PR) is
exactly the class of change most likely to introduce a genuinely red test, and it is exactly the
class of change this blind spot cannot see. It is also closely related to, but distinct from,
ISSUE-092 (Pack Test Phase3 Fixtures Cannot Fail) — that issue is about `backstop pack test`'s
fixture-execution phase never running; this one is about `backstop gate`'s own Go-suite-verdict
dimensions never running, for a different (but structurally similar) reason in each of the three
steps named above.

## Direction (not scoped here)

At minimum, the eventual plan should weigh:

1. Whether `coverage_threshold`'s early-skip-on-no-production-files behavior should instead still
   run the affected package's test suite to a verdict (without scoring coverage) when the diff is
   entirely test files.
2. Whether a NEW gate dimension (or a widened `test_verification`) should exist whose job is
   specifically "run every mandated test and read its real exit code," independent of coverage
   scope.
3. A regression fixture proving the fix: a tree with a genuinely failing mandated test and an
   entirely-test-file diff must turn `backstop gate` red. Absent that proof, any fix risks becoming
   another vacuous-green claim, per the fixtures-from-real-output/must-falsify convention.

## Notes / references

- Reported by the implementer during `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml`
  (2026-08-11), surfaced correctly rather than hand-waved past.
- Sibling to the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092): a
  validation signal that reads as complete/authoritative and silently isn't. Filed separately per
  the existence-in-world check — this is a `backstop gate` (runtime kill-chain) defect, not a
  `backstop pack test` (authoring-time fixture validator) defect, and no open issue currently names
  it.
