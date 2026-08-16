---
title: "Gate Verdict Honesty"
number: DIR-032
created: "2026-08-10"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-066"
    - "ISSUE-067"
    - "ISSUE-091"
    - "ISSUE-092"
    - "ISSUE-093"
    - "ISSUE-097"
    - "ISSUE-100"
    - "ISSUE-106"
    - "ISSUE-112"
    - "ISSUE-113"
    - "ISSUE-114"
    - "ISSUE-118"
    - "ISSUE-129"
    - "ISSUE-136"
    - "ISSUE-140"
    - "ISSUE-142"
    - "ISSUE-144"
---

## Description

Carved out of DIR-024 "Gate/Engine Quality" per founder ruling (Brandon,
2026-08-10). Sixteen issues share one defect shape: **a gate step computes a
result internally but reports the wrong verdict about it** — silent pass
when it should block, a scoped-clean signal when the unscoped truth is red,
an opaque crash where a legible finding belongs, a dimension that never
fires at all for an entire artifact kind or an entire diff shape, or a
finding that IS computed correctly and then discarded before verdict
computation. This is the "no vacuous green" invariant policing itself, and
it is the exact failure mode the product sells against — worth a dedicated
home rather than diffusion across a catch-all directive.

backlog-pm tracked this cluster growing from two issues (2026-07-26) to
eleven (2026-08-02) inside DIR-024's Notes, escalating the cluster-home
question repeatedly without acting on it unilaterally — see that directive's
Notes for the full history. Eight of the eleven were already cited in
DIR-024's `source:` frontmatter (ISSUE-092, 093, 097, 100, 106, 112, 113,
114) and move here with their Description/Notes prose intact. The remaining
three — ISSUE-066, ISSUE-067, ISSUE-091 — were named repeatedly in DIR-024's
Notes as cluster siblings but were **never added to its `source:` list**;
they had no directive home at all until now. That accounts for the founder's
2026-08-10 ruling, which enumerated eleven members. Items 12 (ISSUE-118), 13
(ISSUE-129) and 14 (ISSUE-136) were **not** part of that ruling — all three
were slotted afterward by backlog-pm under the standing clear-fit grant, on
charter fit against this directive's own description, not by the founder's
original roster. All fourteen members are `status: open` today; as of
2026-08-15, none had an in-flight plan (verified 2026-08-15 for ISSUE-129
specifically: no plan in `plans/` targets it or the go-test scope-filter
exemption it names). **Correction (2026-08-16):** that observation is now
stale, and stale in more than the one place it was first caught —
`plans/PLAN-ISSUE-129-go-test-scope-filter-exemption.plan.yml` exists
(`status: draft`, created 2026-08-15) and its fix is visibly mid-flight in
the working tree (modified `cmd/backstop/testdata/exempt-matrix-bindings.yml`,
the go-toolchain testdata `pack.yml`, `cmd/backstop/pack_gate_exempt_test.go`,
`backstop.lock`). A full sweep of `plans/` against all fourteen roster issue
IDs (independently verified, not just found once) turned up three more:
`plans/PLAN-ISSUE-112-engine-tool-missing-silent-vacuous.plan.yml` (item 9),
`plans/PLAN-ISSUE-113-zero-match-classification-refusal.plan.yml` (item 10),
and `plans/PLAN-ISSUE-118-gate-blind-spot-test-only-diffs.plan.yml` (item
12) — all `status: draft`. Four of the fourteen roster members have
in-flight plans as of 2026-08-16 — items 9, 10, 12, and 13 — and the
remaining ten, including new item 14, are plan-free. See the Notes for why
this is a coordinated drain, not four incidental plans. **Correction (2026-08-16,
later same day):** the paragraph above is now itself stale, in the same direction
as the Notes correction below — all four plans (PLAN-ISSUE-112, -113, -118, -129)
reached `status: completed` and all four issues (ISSUE-112, 113, 118, 129) reached
`status: closed` the same day, per the "overnight P0 batch" closeout. "All fourteen
members are `status: open` today" no longer holds — four are closed, and the
roster itself grew to fifteen with item 15 (ISSUE-140). See the Notes' own
"In-flight execution note" correction for the full picture; preserved here rather
than rewritten, per this directive's own convention. **Correction (2026-08-16,
later still):** the roster grew again, to SIXTEEN, with item 16 (ISSUE-142),
slotted by backlog-pm under the same standing clear-fit grant recorded above for
items 12 (ISSUE-118), 13 (ISSUE-129) and 14 (ISSUE-136) — on charter fit against
this directive's own description, not by the founder's original 2026-08-10
eleven-member roster. ISSUE-142 was mandated by `PLAN-ISSUE-092` (item 4's own
lane, `status: draft`) as one of three named follow-ons its F7 review block
deferred rather than fixed in place; see item 16 below and the Notes for the
full provenance chain, including its two siblings ISSUE-141 (filed 8 seconds
earlier, a hard prerequisite for that same plan's phase 5) and a third,
not-yet-filed follow-on this directive should expect shortly. **Correction
(2026-08-16, later again):** the roster grew once more, to SEVENTEEN, with
item 17 (ISSUE-144), slotted by backlog-pm under the same standing clear-fit
grant recorded above for items 12 (ISSUE-118), 13 (ISSUE-129), 14 (ISSUE-136)
and 16 (ISSUE-142) — on charter fit against this directive's own description,
not by the founder's original 2026-08-10 eleven-member roster. See item 17
below and the Notes for the full provenance and the reasoning that keeps it
here rather than in DIR-024, which took ISSUE-141 (item 17's closest sibling)
on the opposite side of that same line hours earlier.

The cluster's variants, so a planner does not read it as one uniform bug:

- **Items 1-3 (066, 067, 091)** are gate *test-verification* scope defects —
  a narrow `-run` filter silently doubling as the pass/fail bound, a real
  test failure surfacing as an opaque engine crash, and `--all` silently
  under-reporting relative to diff scope. All three surfaced together via
  ISSUE-064's impl-review and compound: any one alone would have hidden the
  same regression a different way.
- **Items 4-10 (092, 093, 097, 100, 106, 112, 113)** mis-report a verdict
  the step DID compute, or compute one from data it misread — dead fixture
  execution, a false-RED on non-Go files, a fail-open waiver keyed to a
  renamed namespace, severity-blind tallies, a severity-discarding join, a
  silent-pass on a missing engine tool, and a mass of fabricated violations
  from an empty classification set. **Item 15 (140) belongs in this bucket
  too, and is not a new variant — it is the SAME defect as item 9
  (silent-pass on a broken engine run) on a second command surface** (`pack
  test`/`pack check` phase3, rather than the gate dispatch path item 9
  fixed). That makes it notable rather than redundant: the bucket now
  contains a member whose existence proves item 9's fix was surface-local,
  not root-caused — the narrow `*exec.Error`-only check item 9 fixed on one
  path was left unwidened on the sibling path it was copied from. **Item 16
  (142) also belongs in this bucket, and also on the `pack test` phase3
  surface, and is also not a new variant.** With item 16 added, that single
  surface now holds THREE independent members: item 4 (the `rule_path:`
  declaration style — packval reads the wrong YAML key on a field that
  exists), item 16 (the `pattern-arg` declaration style — packval has no
  field at all for the key packs actually declare), and item 15 (a
  dispatched engine that never started reading as a clean negative). None of
  the three fixes closes either of the others — together they mean no
  fixture in any real in-repo pack can currently falsify anything the way
  `pack test` phase3 is supposed to guarantee. **Item 17 (144) also belongs
  in this bucket, and also on the `pack test`/`pack check` phase3 surface,
  and is also not a new variant.** With item 17 added, that single surface
  now holds FOUR independent members: item 4 (the `rule_path:` key drift —
  packval reads the wrong YAML key on a field that exists), item 16 (the
  `pattern-arg` declaration style — packval has no field at all for the key
  packs actually declare), item 15 (a dispatched engine that never started
  reading as a clean negative), and item 17 (a dispatched engine whose real
  output lives in a declared file the executor never reads — it parses
  stdout noise instead of the artifact). None of the four fixes closes any
  of the others.
- **Items 11 and 12 (114, 118)** are the *starvation* variant: neither
  mis-reports a verdict it computed (that's items 4-10) — each never
  computes anything at all for its trigger. Item 11 starves on an entire
  ARTIFACT KIND: a plan never produces the delivered-but-open advisory,
  full stop. Item 12 starves on a DIFF SHAPE: an entirely-`_test.go` change
  never gets its suite run to a verdict by any dimension, regardless of
  artifact kind. Item 12 is the higher-blast-radius of the two — item 11
  starves a warn-only advisory, item 12 starves the gate's central blocking
  pass/fail promise. Item 12 also sits with items 1-3 by SURFACE (it is a
  test-verification defect and overlaps item 1's fix directly, per the
  cross-reference in item 12 below) even though its failure MODE groups it
  with item 11 here — a planner needs both facts.
- **Item 13 (129) is the *suppression* variant**, distinct from both of the
  above and not covered by either: the finding is computed CORRECTLY by the
  engine, IS correctly converted to a violation, and is THEN discarded
  post-hoc by a filter downstream of the step that produced it. That is
  neither "mis-reports a verdict it computed" (items 4-10, where the verdict
  itself is derived wrong from data the step has) nor "never computes
  anything" (items 11-12, starvation, where nothing is ever produced): here
  the truth is computed, is right, and is thrown away before status
  computation ever sees it. A planner needs this distinction because the fix
  site is not in the producing step at all — it is in a filter the producing
  step never touches.
- **Item 14 (136) is not a new instance of any of the above — it is the
  COVERAGE/ASSURANCE item for the suppression variant item 13 named.** Item
  13 fixed exactly one mis-declared engine binding; item 14 is the audit that
  bounds how many others are wrong the same way. It carries no verdict defect
  of its own today — its risk is open-ended precisely because nothing
  currently bounds it, which is why it is `risk: moderate` rather than
  `critical`: it is an audit, not a known live defect, though any defect it
  surfaces inherits item 13's `critical` severity class.

1. **Gate test verification runs the full package, not a plan's narrow
   `-run` filter (ISSUE-066).** A spec/plan `test_command` commonly scopes
   to `go test ... -run '<claim-name-pattern>'` to name the tests that prove
   that artifact's claims. The gate's test verification honors that filter,
   so a regression in any test whose name does NOT match the pattern stays
   invisible: the scoped run is green while the full package is red. Two
   distinct concerns are conflated onto one `-run` filter — "which tests
   prove THIS artifact's claims" (legitimately a subset, for claim-mapping)
   and "which tests must pass for the gate to be green" (the full package
   any changed code lives in, never a subset) — and the gate currently
   derives the second from the first. Discovered in ISSUE-064: a real
   regression was broken by a routing change, failed deterministically under
   unfiltered `go test ./cmd/backstop/...`, but matched none of the
   `test_command`'s `-run` patterns — every mechanical check reported green,
   and the regression was only visible via an unfiltered run (and was
   further masked as an opaque engine crash — see item 2). Direction: the
   gate's test step must run the full test package(s) in the change's scope
   independent of the plan's claim-mapping filter; the `-run`/mandated-test
   mapping stays as the claim→test evidence link, but a green gate must
   require the whole package green, not just the mapped subset.

2. **go-test engine reports test failures as an opaque crash, not
   parseable findings (ISSUE-067).** When `go test` exits non-zero because a
   test FAILED (not because the toolchain failed to run), the
   `backstop/go-toolchain` go-test engine surfaces it as an opaque dispatch
   error — `"crashed: non-zero exit with no parseable findings"` — instead
   of a finding naming the failing test(s). A genuine test regression is
   therefore indistinguishable from an environmental tool crash on the gate
   surface, and reads as the latter. Discovered in ISSUE-064: a real test
   regression appeared at the gate ONLY as this "crash," and was dismissed
   by the implementer, the coordinator, and the impl-reviewer as
   known/environmental noise — the exact failure mode this reporting
   invites. Root cause: the converter (`scripts/test-to-sarif.sh`) is not
   extracting `--- FAIL:` failure output into findings before the exit code
   is judged, so the exit code wins and real failures are discarded. `risk:
   critical` — not merely an ergonomics gap, but a trust hole in the gate's
   loud-failure guarantee: it silently converts real test failures into
   dismissible "environmental" noise. Direction: run `go test -json` and
   emit a per-failure finding (test name, package, message); distinguish a
   genuine tool crash (compile error, panic before any test output,
   toolchain missing) from test failures (tests ran, some failed →
   findings, not a crash). Lives in the `backstop/go-toolchain` pack
   (engine binding + converter script), tracked here in backstop-core's
   issues per the pack-fix convention. Fold in with item 1: a full-package
   run that fails must produce legible findings, not an opaque crash.

3. **`gate --all` underreports test file findings relative to diff scope
   (ISSUE-091).** `gate --all` is not a superset of the diff-scoped gate —
   it silently under-reports findings on test files, the opposite direction
   of ISSUE-070 (diff scope leaking project-wide lint; closed). Measured by
   implementer-087 (2026-07-28), comparing two gate runs at the SAME HEAD:
   the diff-scoped gate reported 124 findings `--all` did not; only 22
   findings were shared. Concrete falsifier: `artifact_new_test.go` has five
   confirmed `code, _ :=` sites; diff-scoped reports 4, `--all` reports zero.
   `risk: critical`, and already realized as a consequence: PLAN-ISSUE-087's
   TASK-004 method ("intersect `gate --all` with the swept files") sized a
   founder scope ruling at ~31 violations when the enforcing (diff-scoped)
   truth was 153 rows — a founder made a scope decision on an undercount
   produced by the gate itself, in its own full-scope mode. Root cause,
   confirmed by reading `pkg/pack_gate.go`'s `dispatchPackEngines`: diff
   scope hands the engine an EXPLICIT list of changed files; `--all` hands
   the engine the bare project-root DIRECTORY as its single scan target,
   leaving semgrep to do its own recursive walk and its own
   `paths.include` glob resolution against files it discovered itself —
   two different code paths, unverified to agree, and the falsifier is
   consistent with them not agreeing. Direction (to be weighed by the plan):
   either enumerate the full project file list under `--all` the same way
   diff scope does and pass that explicit list to the engine instead of the
   bare directory, so both scopes exercise the identical semgrep code path;
   or, if a directory target is kept for performance/engine-native-discovery
   reasons, add a reconciliation check that a directory-target run and an
   explicit-file-list run produce the same finding set on a known fixture.
   Either way the fix must be provable against the `artifact_new_test.go`
   falsifier.

4. **`pack test` phase3 cannot fail — fixture execution is dead code for
   every real pack (ISSUE-092).** `pkg/packval`'s `Rule` struct reads a
   rule's source file from YAML key `file:`, but every real pack.yml
   declares it as `rule_path:` — which is what the runtime gate parser
   consumes. The authoring-time validator and the runtime parser are two
   independent manifest models that have drifted; `rule_path`/`RulePath`
   appears nowhere under `pkg/packval/` (grep clean). Consequence in
   `phase3.go`'s `RunFixtures`: `rule.File` unmarshals to `""` for every
   real pack, so its `if rule.File != ""` guards are all false —
   `executor.RunEngine` is never invoked for any layer 1-2 semgrep rule
   declared via `rule_path:`. `res.Status` stays `"pass"` having executed
   zero fixture checks. Measured, not inferred: implementer-087 rewrote a
   negative (violating) fixture to be fully compliant — deleting the very
   violation it exists to catch — and `pack test` still returned
   `phase3-fixtures: pass`, exit 0; removing a rule's `paths: exclude`
   likewise fails nothing. Matters because `pack test` is the pack-quality
   gate for the whole ecosystem, and phase3 is the mechanism that enforces
   the fixtures-from-real-output/must-falsify convention (founder law) — a
   pack whose rule never fires on anything can ship green. Two measurement
   traps the fix must not reintroduce, both recorded in the issue: (a)
   directory-scan vs explicit-file-list divergence — restored fixture
   execution must dispatch with explicit file targets the way the gate
   does, or it inherits ISSUE-091's undercount (item 3, above); (b) an
   engine/schema error currently surfaces as zero results and is
   indistinguishable from a genuine clean run — the fix must make that loud
   and distinct rather than foldable into either pass path. Any fix must
   land a regression fixture proving phase3 CAN fail (a `rule_path:`-
   declared rule with a compliant negative fixture must turn `pack test`
   red) — otherwise the fix is itself a vacuous-green claim. `risk:
   critical` — an active false-green in the tool that gates the entire pack
   ecosystem, live today for every pack in the fleet. Legitimately competes
   with any other item here for first pick.

5. **`gate --file` false-REDs non-Go files whose directory holds no Go
   package (ISSUE-093).** `fileModeTestTarget`
   (`cmd/backstop/pack_gate_filemode.go`) fires whenever the scope is
   `GateScopeModeFile` and the dispatched binding declares
   `PackageScoped: true`. It unconditionally derives a `go test` target from
   the file's directory with NO check that the directory contains any `.go`
   files. For `.github/workflows/ci.yml` the target becomes
   `./.github/workflows`; `go test` there exits non-zero with zero
   parseable findings, and `crash_guard` renders that legitimate no-op as an
   engine CRASH violation. So a per-file verdict on a non-Go file is not a
   property of the file — repo topology, not file correctness, decides the
   verdict. Reproduced independently twice on untracked-clean, unmodified
   files. Blast radius bound: the DEFAULT diff-scoped gate is UNAFFECTED —
   `--file` only. Distinct from item 2 (ISSUE-067): there the trigger is a
   REAL test failure the converter cannot parse; here there is no test
   failure at all and no Go code in scope. A second, independent defect on
   the same command surface: `--file` is bound and read as a plain string,
   so repeating the flag silently OVERWRITES rather than accumulating —
   verified: `--file README.md --file .github/workflows/ci.yml` reports "1
   explicit files." A single `--file X` followed by bare positional args
   DOES accumulate, so flag-repetition is the only broken shape; no error,
   no warning, no output trace of the discarded value. Neither defect is a
   false GREEN in the gate's own verdict (defect one fails honest-loud, in
   the safe direction), but defect two IS a false coverage claim — the
   operator believes N files were checked when 1 was. Constraint for
   whoever plans it: the fix must preserve SPEC-034 REQ-010/CLM-035 —
   file-mode test scoping is PRESERVED, not dropped — so the correct shape
   is a guard on the derived target (or on crash-vs-no-op classification),
   not removal of the file-mode override. Per the zero-baked-language law,
   any "does this directory hold a Go package" guard must not be a baked
   language check in core.

6. **Waiver tokens keyed to a dead pack namespace fail open, and the
   harvest path cannot see them (ISSUE-097).** Two `@waiver:` tokens
   (`cmd/backstop/pack_gate.go:888`, `pack_gate_provision.go:119`) key the
   rule ID `backstop/self/...` — a pack path that exists nowhere;
   `backstop.lock` records `backstop-ai/backstop-self`, fallout of the
   2026-07-27 pack rename. Measured status is three things: STALE (the
   namespace is gone), INERT (the rule matches nothing today), and FAIL-OPEN
   (`waiver.Adjudicate` suppresses only on an EXACT rule-ID match, so if the
   rule ever matches these lines again the tokens will NOT suppress — the
   comments read as adjudicated, and are decorative). The structural half,
   and the reason this is directive-worthy rather than a two-line typo fix:
   **waiver harvest is finding-driven, not tree-driven.** `Adjudicate` only
   reads the association window of each finding it is HANDED; a token on a
   line where no finding lands is never harvested at all, so it does not
   even reach the `Unused` bucket. The same blindness holds on the CLI
   surface (`backstop waiver list`), which inherits the identical
   constraint. Both the `waiver_resolution` gate step and `waiver list`
   therefore report "clean — no active waivers" truthfully-as-implemented
   and falsely-as-read. A rename-orphaned or hand-typo'd rule ID is
   architecturally invisible, not merely unreported. Scope, as two parts:
   (a) re-key or remove the two tokens; (b) give `waiver_resolution`/
   `waiver list` the ability to name a waiver whose rule-ID prefix matches
   no pack namespace in `backstop.lock`, WITHOUT requiring a live finding at
   that location — harvesting tokens from the tree for that cross-check, not
   only from finding-adjacent windows. Per loud-≠-blocking this is a
   WARNING, not a new gate failure. Constraint for (a): the ID prefix must
   be the manifest name `backstop-ai/backstop-self/`
   (`pack.NamespacedRuleID`); do not grep-and-copy from
   `bun_ratchet_flip_test.go:128` or `policy_perpack_test.go:29`, both
   synthetic fixtures with the wrong shape. Bound the scope: the `@waiver:`
   channel only — the parallel un-ledgered `// nosemgrep:` channel is a
   related but out-of-scope observation, no artifact, not folded in without
   a founder call.

7. **Step tallies count warnings as violations — the reporting half remains
   after the verdict half was fixed (ISSUE-100).** `StepResult`'s displayed
   count and `GateResult.total_violations` count every entry in
   `StepResult.Violations` identically, regardless of `Violation.Severity`.
   Two severity-blind call sites, both CONFIRMED LIVE:
   `pkg/gate/result.go:225` and `pkg/gate/output.go:61,80`. Motivating
   measured instance (CI run 30389988184, 2026-07-28): `coverage_threshold`
   rendered `fail (2 violations)` where the JSON proves one `severity:
   error` entry plus one `severity: warning` entry — one blocking problem
   displayed as two. The sibling verdict-level defect is ALREADY FIXED and
   should not be re-opened: `pkg/gate/policy.go:73` is now severity-aware,
   and the SARIF-level→`Violation.Severity`→verdict mapping is locked
   end-to-end by `cmd/backstop/pack_severity_contract_test.go`. Therefore
   ISSUE-100's remaining scope is the renderer/tally half ONLY. Recommended
   fix shape per the issue: keep `Violations` as the single carrier (no
   shape change, no schema bump) and split by `Severity` at the two counting
   sites — `total_violations` counts `severity == "error"` (or gains a
   companion `total_warnings`), and the human line becomes something like
   `(1 blocking, 1 notice)`. A separate `Notices` slice was NOT recommended
   — that's a shape change touching every warning-emitting step, the JSON
   schema, and `pkg/gate/baseline.go:218`. Sequencing: item 7's sibling
   ISSUE-107 (coverage warning-only step reads as pass — not itself a
   cluster member, homed in DIR-024) touches the same surface and should be
   planned together or after this item.

8. **Substantiveness JOIN discards a pack-declared severity (ISSUE-106).**
   `pkg/gate/substantiveness_join.go` throws away a pack's declared severity
   one hop past where ISSUE-104/ISSUE-105 closed the same gap elsewhere. Q1
   dispatch preserves it correctly (`nonEmptySeverity` only defaults an
   EMPTY value to `"error"` and leaves a declared `"warning"` intact); the
   join then overwrites it at two sites, both confirmed hardcoding
   `Severity: "error"`: `HollowFindingsToViolations` on every routed hollow
   finding, and `NoTargetViolation`, the noTarget set-join decision table.
   Consequence: a pack declaring a substantiveness rule at `level: warning`
   — an advisory by the founder-ratified severity contract — blocks the
   gate anyway. The two sites are NOT the same fix: `HollowFindingsToViolations`
   is a 1:1 conversion, so forwarding `v.Severity` is a direct substitution.
   `NoTargetViolation` converts no single input finding — it fires on a
   SET-MEMBERSHIP test over a `map[string]bool` carrying presence only and
   no severity anywhere — so resolving it requires an actual decision:
   either a new channel for the rule to declare the noTarget severity, or a
   ruling that a gate-SYNTHESIZED violation keeps a fixed severity by design
   (a gate-computed defect rather than a pack-tunable advisory). That
   decision must be stated explicitly in whatever plan lands. Test coupling
   the planner must not discover the hard way:
   `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
   (`pkg/gate/step_verdict_severity_test.go`, SITE 2 block) currently LOCKS
   the defect and must be deliberately rewritten to assert preservation once
   the fix lands. Blast-radius discipline: measure substantiveness step
   verdicts on the dogfood run and at least one fixture consumer before and
   after; every flip must fit "was severity-blind-overwriting a declared
   `warning`, should never have blocked." `type: bug`, `scope: contained`,
   `uncertainty: known`, `risk: moderate` — does not displace item 4
   (ISSUE-092, `risk: critical`) for first pick.

9. **Missing engine tool + no CrashGuard yields silently empty SARIF, a
   vacuous `pack_engines` pass, and misleading downstream join violations
   (ISSUE-112).** A findings engine whose tool is ABSENT from `PATH` fails
   in the worst possible way when its binding carries no `CrashGuard`: the
   runner's non-fatal `runErr` is discarded, the empty stdout flows through
   convert, the lenient SARIF parse reads zero findings, and `pack_engines`
   PASSES — while every downstream consumer of that evidence is lied to.
   Provenance is a FIRST-CONSUMER discovery: observed live in
   `bclabs-portal`'s first CI run on a GitHub runner, 2026-07-29 — ast-grep
   absent from `PATH` produced empty SARIF, `pack_engines` went green, the
   `test_substantiveness` join starved, and 397 false "does not call
   package" violations landed on innocent tests. Diagnosis took hours
   because nothing named the missing tool. Two aggravators: the
   assume-present fail-loud in `pack_gate_provision.go` EXEMPTS
   provision-declared tools as "auto-provisioned," but provision is a TRUST
   ALLOWLIST PIN ONLY — no code path installs anything, so provision-pinned
   tools (ast-grep, semgrep) get NEITHER an install NOR a presence check;
   and non-CrashGuard engines treat every non-zero/failed run as
   finding-free, so an exec-not-found error is indistinguishable from a
   clean scan — even though `pkg/packval`'s executor already fails loud on
   exactly this error class, giving the fix an in-tree precedent to copy.
   **Correction (2026-08-16):** that parenthetical is FALSE in the path-ful
   case, and item 15 (ISSUE-140) is exactly its falsification — the packval
   precedent was real only for the BARE-NAME (`*exec.Error`) shape. The
   widened gate predicate this item produced, `runNeverStarted`, documents
   in its own doc comment that it is "deliberately WIDER than packval's
   *exec.Error-only check," and packval's own seed was never widened to
   match. See item 15 for the residual.
   Direction: presence-check provision-pinned tools exactly like
   assume-present ones (fail loud, naming the tool and the install
   expectation, or implement provisioning); make any `*exec.Error`-class
   failure fail loud for EVERY engine regardless of `CrashGuard`. Relationship
   to item 4 (ISSUE-092): both false-green, different layers — ISSUE-092 is
   the PACK-AUTHORING gate going vacuously green, ISSUE-112 is the CONSUMER
   gate going vacuously green. Fixing either does not close the other.

10. **Classification matching zero test files should refuse loudly instead
    of fabricating mass join violations (ISSUE-113).** When a pack's
    classification globs match ZERO test files, the substantiveness join
    silently emits a "does not call package X" violation for EVERY mandated
    test — hundreds of misleading findings whose real cause (empty
    classification) is named nowhere. It is the diagnosability sibling of
    item 9 (ISSUE-112) — the two share one observed signature with two
    different root causes, hit twice in one week by `bclabs-portal`: the
    published `typescript-substantiveness` 1.1.0 shipping harness-baked
    classification globs (397 false violations), and the missing-ast-grep
    case (item 9). Both cost hours; both would have been one line of
    output: "classification matched 0 test files." Direction: extend the
    ISSUE-020 config-error refusal philosophy — when mandated tests exist
    but the classifier matches zero test files (or the substantiveness
    evidence set is empty while mandated tests exist), the step REFUSES with
    a config-error naming its cause instead of emitting per-test violations.
    Sequencing: items 9 and 10 should be read as one arc and planned
    together, or item 10 immediately after item 9 — shipping only item 10
    would convert one misleading failure mode into a different one without
    ever naming the absent binary; shipping only item 9 leaves the
    harness-baked-globs root cause still producing the same silent
    mass-violation signature.

11. **The delivered-but-open drift advisory is structurally unable to fire
    for a plan (ISSUE-114).** `looksDelivered`
    (`pkg/gate/status_drift.go:75-85`) returns false immediately when
    `len(rec.MandatedTests) == 0`. So an artifact with no mandated tests can
    never trip the `ClassNonTerminal` advisory branch — the warn-only "this
    looks delivered, advance or close it" signal. This is NOT a categorical
    exclusion of plans by routing — `ClassifyStatusDrift` iterates records
    with no kind filter at all; a plan reaches the classifier exactly like a
    spec or issue. The gap is in the DATA: for plans, `MandatedTests` comes
    from exactly one source, the OPTIONAL per-task `test_names` field
    (introduced by PLAN-ISSUE-048 itself), and nothing in plan validation
    reads, requires, or acknowledges it — it is documentary at the schema
    level and mechanically optional. Corpus measurement (re-run by
    backlog-pm 2026-08-02): of 98 plans, 47 non-terminal, and of the 26
    files carrying a POPULATED task-level `test_names`, every one is
    terminal — no exception left standing anywhere in the corpus. The
    advisory is structurally unable to fire for any plan in
    `draft`/`ready`/`implementing`, no matter how obviously its code
    shipped. Motivating, close-to-self-referential instance: PLAN-ISSUE-048
    itself was `status: draft` while the machinery it specifies shipped in
    the same commit that authored it; it closed 2026-08-02 and produced NO
    drift signal in either direction, confirming its own predicted blind
    spot. Fix direction, kept as constraint not design: either make
    `test_names` load-bearing at plan-authoring time (schema/planner-agent
    requires it alongside the existing prose "mandated test names (exact)"
    convention, so the two representations cannot drift), or give plans a
    task-claims-derived mandated-test concept structurally parallel to
    spec/issue `claims[].tests`. Either way, the defect to close is that a
    plan's prose-declared mandated tests and its machine-readable
    `MandatedTests` are two independent, unsynchronised representations. A
    planner should also state explicitly whether making the field mandatory
    is retroactive — 48 existing non-terminal plans would need backfill.
    Verification bar: a regression fixture in which a NON-TERMINAL plan
    whose mandated tests are all present DOES produce an
    `artifact_status_drift_advisory` warning; a fixture that is only a
    `completed` plan re-proves the already-working path and nothing else.
    Distinct variant: this item never computes anything for an entire
    artifact kind (silent by starvation), where items 4-10 mis-report a
    verdict they DID compute.

12. **No gate dimension runs the Go suite to a verdict when a diff is
    entirely test files (ISSUE-118).** `backstop gate` reports a full PASS
    while a mandated test genuinely fails, reproducibly, on the same
    tree/commit/binary: `go test ./cmd/backstop/... -race -run
    TestCIRecipes` returns `--- FAIL:
    TestCIRecipes_FleetDeclaresPackAtOneVersionInBothFiles`, exit 1, while
    `./bin/backstop gate` reports `10 passed, 0 failed`, exit 0. Measured
    2026-08-11 by the implementer during
    `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml`, and surfaced rather than
    hand-waved past. Three dimensions touch "did the test pass" and none
    establishes a verdict for this diff shape: `test_verification`
    (`pkg/gate/step_testverify.go`) only checks that a mandated test's NAME
    is present in source (`ExtractMandatedTests` + `ResolveMandatedTestPaths`)
    — I confirmed the file execs nothing, so a test that exists and is named
    correctly satisfies it whether it passes, fails, or panics;
    `test_substantiveness` only checks the test BODY carries real assertions,
    and likewise never executes it, so a genuine assertion that currently
    evaluates false still passes; `coverage_threshold`
    (`pkg/gate/step_coverage.go`) is the ONE dimension that actually invokes
    `go test`, but it exits early at `step_coverage.go:98` with `Status:
    "pass", Reason: "no in-scope files to measure for coverage"` whenever the
    scoped change touches no in-scope production file — verified verbatim in
    the current tree. A 100%-`_test.go` diff hits that skip every time. This
    is the variant of item 11's shape (silent by STARVATION, not misreport)
    but far broader in blast radius: item 11 starves one artifact kind's
    advisory, this starves the gate's central pass/fail promise for an
    entire, common diff shape — and it is exactly the class of change most
    likely to introduce a red test (a plan's final "make the failing test
    pass" step, any test-hardening commit). Fix directions to weigh, kept as
    constraint not design: (a) have `coverage_threshold` still run the
    affected package's suite to a verdict, without scoring coverage, when the
    diff is entirely test files; or (b) a new dimension (or a widened
    `test_verification`) whose job is specifically "run every mandated test
    and read its real exit code," independent of coverage scope. **Overlap a
    planner must resolve before picking either: this item and item 1
    (ISSUE-066) are two halves of one hole** — item 1 is the gate honoring a
    narrow `-run` filter as the pass/fail bound, item 12 is the gate running
    no suite at all; item 1's stated fix ("the gate's test step must run the
    full test package(s) in the change's scope independent of the plan's
    claim-mapping filter") would, if "scope" is read to include test-only
    diffs, subsume item 12, and a fix for item 12 that ignores item 1
    re-derives the same filter question. Plan them together, or item 12
    immediately after item 1. Per the zero-baked-language law, any fix must
    reach `go test` through the pack's declared engine, never a baked Go path
    in core. Verification bar the issue states and that must not be
    softened: a regression fixture in which a tree with a genuinely failing
    mandated test and an entirely-test-file diff turns `backstop gate` RED —
    a fixture that only re-proves the production-file-changed path proves
    nothing. `type: bug`, `scope: cross-cutting`, `uncertainty: known`,
    `risk: critical`.

13. **The go-toolchain pack's `go-test` engine is diff-scope-filtered: a
    diff-scoped gate PASS can coexist with a real, currently-failing,
    out-of-scope Go test (ISSUE-129).** `backstop gate` diff-scoped — the
    DEFAULT invocation, and the only blocking check CI runs on a pull
    request (`.github/workflows/ci.yml:155`, `./bin/backstop gate --base
    "$BASE"`) — can report full PASS while a real, currently-failing Go test
    sits in the tree, whenever the failing test's FILE is outside the diff's
    scope. Holds both when the diff just broke that test and when it was
    ALREADY red on `main` before the diff existed. Measured 2026-08-15
    during `PLAN-SPEC-070` (backstop doctor) implementation: the implementer
    registered the new `doctor` command in `cmd/backstop/root.go` —
    plan-mandated and legitimate. `TestCIRecipes_RegisteredCommandSurfaceUnchanged`
    (`cmd/backstop/ci_recipes_mechanism_test.go`, SPEC-067's CLM-052
    anti-regression pin that enumerates the entire top-level command set by
    name) went red instantly, BY DESIGN. That file was never in
    PLAN-SPEC-070's diff. Every gate run through implementation, including a
    final diff-scoped run against a freshly rebuilt binary, reported PASS.
    The red test was caught only by a hand-run unfiltered `go test ./...`.
    Root cause, confirmed live in the tree — cite the sites: `cmd/backstop/pack_gate.go:730`
    stamps each bridged violation's `ProjectWide` from
    `binding.ExemptFromScopeFilter` (SPEC-041's declared build-exemption
    mechanism, resolved per-violation). `pkg/gate/scope.go:302-326`
    (`filterViolations`) keeps a violation only when `ProjectWide` is true OR
    `scope.Contains(violation.File)`. In
    `.backstop/packs/backstop-ai/go-toolchain/pack.yml` the `go-build` engine
    declares `exempt_from_scope_filter: true` (verified, with an inline
    comment saying exactly why: "so an unchanged-file build break still REDs
    a diff-scoped gate (SPEC-041 CLM-011)") while the `go-test` engine
    declares NO such flag and therefore defaults false (verified — its
    binding is command `go test`, `input_mode: none`, `scope_kind:
    project-wide`, `project_target: "./..."`, `convert:
    scripts/test-to-sarif.sh`, `crash_guard: true`, `gate_type: test`,
    `package_scoped: true`, and no `exempt_from_scope_filter` key). So the
    go-test failure IS computed, IS converted to SARIF by the pack script,
    and is then silently dropped before status computation. The other `go
    test` runner in the gate is structurally blind, so nothing catches the
    miss: `pkg/gate/step_coverage.go`'s built-in `coverage_threshold` step
    also runs `go test -coverprofile=...`, but it measures coverage
    percentage only — a failing test still emits a usable profile, so
    pass/fail never reaches that step's verdict. `--all` is NOT affected —
    `filterViolations` short-circuits and returns every violation unchanged
    when `scope.Mode == GateScopeModeAll` (`pkg/gate/scope.go:303`). But no
    CI job runs an unfiltered suite as a blocking check either:
    `.github/workflows/ci.yml` has exactly two jobs — `gate` (diff-scoped,
    blocking, every push/PR) and `baseline` (`--all`-scoped but gated to
    post-merge pushes on `main`, and `backstop baseline generate`
    (`cmd/backstop/baseline.go:51`) is a snapshot generator with no pass/fail
    exit contract, so it does not fail the workflow on RED). A cross-file
    test break therefore merges to `main` behind a green PR gate with no
    later blocking check to catch it. Relationship to ISSUE-070 (closed),
    stated precisely so nobody re-opens it: ISSUE-070 fixed `pack_engines`
    not applying the scope filter AT ALL. That fix is what makes this the
    residual — the filter now runs correctly, keyed on
    `ProjectWide`/`ExemptFromScopeFilter`, exactly as declared. ISSUE-129 is
    about the DECLARATION, not the filter. Distinctness from item 12
    (ISSUE-118) — this is the most important thing for a planner and must
    not be blurred: both read as "gate reports PASS while a mandated/real
    test genuinely fails," but the mechanisms are opposite and neither fix
    closes the other. Item 12 is STARVATION: for an entirely-`_test.go` diff
    no dimension ever runs the suite to a verdict at all (the
    `coverage_threshold` early skip at `step_coverage.go:98`). Item 13 is
    SUPPRESSION: the suite DOES run, DOES fail, and DOES produce a real
    finding, which is then discarded by file-membership scope filtering —
    and it fires regardless of diff shape (the repro's diff changed only a
    production file, `root.go`). ISSUE-118's own root-cause analysis never
    mentions the go-toolchain `go-test` engine, so these are
    independently-discovered gaps in the same territory. Both stay open
    until EACH has its own proof-of-fix regression fixture. Overlap with
    item 1 (ISSUE-066) worth flagging: item 1 is the gate honoring a narrow
    `-run` filter as the pass/fail bound; item 12 is no suite run at all;
    item 13 is the suite's real failure filtered out by file scope. All
    three are "the gate's pass/fail bound is derived from something other
    than 'is the code green.'" Whoever plans any of them should read all
    three and say which bound they are fixing. Direction, kept as
    constraint not design: (a) decide whether `go-test` should declare
    `exempt_from_scope_filter: true` like `go-build` — the whole-module
    argument is that Go test failures, like build failures, legitimately
    originate from a change to a file the failing test does not live in —
    or whether something narrower is needed (e.g. a baseline compare
    distinguishing "already red before this diff" from "this diff caused
    it," so pre-existing red does not retroactively block unrelated PRs);
    (b) audit EVERY other findings engine lacking the flag, since it is
    declared per-binding and NOT derived from `gate_type`, so any future
    pack engine can silently reintroduce this gap unless something asserts
    intent engine-by-engine; (c) the fix lives in the PACK manifest
    (`backstop-ai/go-toolchain`) if option (a) is chosen — per the
    zero-baked-language law nothing about this may become a baked Go path in
    core. Verification bar, which must not be softened: a regression fixture
    in which a tree with a genuinely failing test in a file OUTSIDE the
    current diff scope turns a diff-scoped `backstop gate` RED. A fixture
    that only re-proves the `--all` path, or one that puts the failing test
    inside the diff, proves nothing and is itself a vacuous-green claim.
    `type: bug`, `scope: cross-cutting`, `uncertainty: known`, `risk:
    critical`.

14. **No audit of findings-engine bindings for missing
    `exempt_from_scope_filter` — ISSUE-129's own Direction item (b) is
    unaddressed (ISSUE-136).** `exempt_from_scope_filter` is a per-binding
    boolean with NO structural derivation — not from `gate_type`, not from
    `scope_kind`, nothing else in the manifest implies it — and nothing
    currently asserts, engine-by-engine, whether each non-declaring engine's
    default-`false` is correct for it or is itself an undetected ISSUE-129
    instance. This is item 13's own Direction item (b), restated verbatim
    above ("audit EVERY other findings engine lacking the flag, since it is
    declared per-binding and NOT derived from `gate_type`, so any future pack
    engine can silently reintroduce this gap unless something asserts intent
    engine-by-engine") — ISSUE-136 is that audit, now owned by an artifact
    instead of living as an unowned bullet inside item 13. Verified directly
    against the live tree (2026-08-16), not inferred: exactly THREE engine
    bindings across all installed non-testdata packs declare
    `exempt_from_scope_filter: true` — `go-build` and `go-test`
    (`.backstop/packs/backstop-ai/go-toolchain/pack.yml:73,93`; go-test's is
    item 13's fix, already landed in the installed pack) and `go-arch-lint`
    (`.backstop/packs/backstop-ai/backstop-core-architecture/pack.yml:17`).
    Every other declared engine — base-engines `semgrep`/`ast-grep`/
    `sandbox`/`config-file`, `go-contracts` (2 bindings),
    `go-substantiveness`, `ci-workflows` `semgrep-ci`, and go-toolchain
    `golangci` — declares no key and defaults false. The load-bearing
    evidence a planner must not miss: `go-arch-lint` is a `gate_type:
    findings` engine that independently arrived at `true`. That is direct
    proof the exemption semantics are NOT tied to `gate_type`, so the
    findings family is not automatically safe just because the two engines
    fixed so far (`go-build`, `go-test`) were `gate_type: build`/`test`.
    Equally, per the existing matrix
    (`cmd/backstop/testdata/exempt-matrix-bindings.yml`), `scope_kind:
    project-wide` does NOT imply exempt either — `golangci` is project-wide
    and deliberately non-exempt (CLM-017). So neither structural signal can
    be used as a shortcut for the audit; it has to be an engine-by-engine
    intent judgment. Direction, kept as constraint not design: (a) for each
    non-declaring binding, judge whether its violations can legitimately
    originate from a file OTHER than the one they are reported against, or
    whether file-scoped filtering is intrinsically correct for it; (b) any
    engine judged mis-declared gets its OWN defect artifact (pack-manifest
    fix + version bump + relock, the ISSUE-129 mechanism) — this issue is the
    audit, never a bundle of fixes; (c) whether `pack check`/`pack test`
    should gain an advisory surfacing a project-wide-scope engine with no
    explicit key is a real design question with real maintenance cost
    (PLAN-ISSUE-129's own words), NOT a free rider — and per the
    zero-baked-language law any such check must be a pack rule, never baked
    core logic. This is NOT a fourteenth instance of a mis-reported verdict —
    it is the coverage/assurance item for the suppression variant item 13
    named (see the variants map above). Item 13 fixed one engine; item 14
    bounds how many others are wrong; its risk is open-ended precisely
    because nothing currently bounds it. `type: technical-debt`, `scope:
    cross-cutting`, `uncertainty: exploratory`, `risk: moderate` — moderate,
    not `critical` like item 13, because this is an audit rather than a known
    live defect, but any defect it surfaces inherits item 13's severity
    class.

15. **`pack test`/`pack check` phase3's never-started check misses path-ful
    engine commands — the same defect item 9 fixed on the gate path, left
    open on the pack-authoring path item 9's own fix assumed was already safe
    (ISSUE-140).** Verified by reading the current tree, not relayed:
    `DefaultExecutor.RunEngine` (`pkg/packval/executor.go`) decides "broken
    run" with `errors.As(runErr, &execErr)` against `*exec.Error` ONLY.
    `*exec.Error` is produced only when `exec.Command` resolves a BARE
    command name via `LookPath` and that lookup fails; a PATH-FUL command
    (`./scripts/checker.sh`) never consults `LookPath` and instead fails at
    fork/exec time as `*fs.PathError{Op: "fork/exec"}`. `buildEngineArgv`
    takes `name` straight from `binding.Command`, pack-declared DATA, with
    nothing preventing or excluding a path-ful command there. On that shape
    the never-started check is false, execution falls through, `stdout` is
    empty, and `check.ParsePackFindings` → `parseSarif` has a deliberate
    lenient case (empty input → `(nil, nil)`, no error) — so `RunEngine`
    returns `ExecutionResult{Passed: false, ExitCode: 0}, nil`, clean and
    error-free. For a NEGATIVE phase3 fixture `Passed: false` IS the success
    condition, so the fixture reads as correctly passing though the engine
    never started. The load-bearing point, and the best single line for this
    item: **the fix that closed item 9 documents this gap in its own source
    and left it open.** `runNeverStarted` (`cmd/backstop/pack_gate.go`)
    matches BOTH shapes, and its doc comment says in so many words that
    `pkg/packval/executor.go` is "the SEED of this shape" and that the gate
    predicate is "deliberately WIDER than packval's *exec.Error-only check"
    — the gate path was widened; the seed it was copied from never was.
    Direction: widen packval's `RunEngine` never-started check to also match
    `*fs.PathError` with `Op == "fork/exec"`, the same two-shape predicate
    `runNeverStarted` already implements. Check whether the predicate can be
    hoisted into a package both `pkg/packval` and `cmd/backstop` can import
    without violating an import-boundary constraint (PLAN-ISSUE-118 hit a
    comparable one where `pkg/gate` could not import `pkg/pack/engine`), or
    whether packval needs its own copy for architectural reasons — either is
    acceptable, but the two copies must not drift in which shapes they
    catch, since that drift is the root cause here. Key on `Op`, never on
    errno (ENOENT/EACCES/ENOEXEC), which would bake OS knowledge into a thin
    executor — that constraint is already recorded on `runNeverStarted`.
    Relationship lines worth stating explicitly, because this item sits at
    the intersection of two existing members: it carries **item 9's
    (ISSUE-112) exact error-shape narrowness** onto **item 4's (ISSUE-092)
    surface** — the pack-AUTHORING gate, `backstop pack test`/`pack check`
    phase3. The directive already distinguishes those two by LAYER (item 4 =
    pack-authoring false-green, item 9 = consumer false-green); item 15 is
    the first member where one member's MECHANISM lands on the other's
    LAYER — neither item 4 nor item 9 closes it. Arguably worse than item 9
    in consequence: `pack test`/`pack check` is the tool a pack author is
    supposed to trust BEFORE a pack ever reaches `backstop gate`. Provenance
    is distinctive and worth recording: **ISSUE-140 was filed BY
    PLAN-ISSUE-112 itself**, one of two named follow-ons in that plan's
    "FOLLOW-ONS, BOTH FILED" close banner (the other is ISSUE-134, `backstop
    doctor`'s toolchain check never probing findings-engine tools — filed,
    still `status: open`, and cited by no directive). The plan's own words:
    "a sibling defect in pkg/packval/executor.go, found by this plan's own
    investigation and filed separately… Different command, different lane;
    not duplicated in this banner." This is the cluster's first member
    discovered by the DELIVERY of another member, not by dogfood or by a
    first consumer. `type: bug`, `scope: contained`, `uncertainty: known`,
    `risk: critical`.

16. **`pack test` phase3 is ALSO dead for `pattern-arg` rules — a second,
    structurally distinct declaration style item 4 does not cover
    (ISSUE-142).** `pkg/packval`'s authoring-time `Rule` struct
    (`pkg/packval/manifest.go`) has NO `Pattern` field at all — grep for
    `Pattern` under `pkg/packval/` returns ZERO hits (verified). The runtime
    gate model does have one: `pkg/pack/manifest.go:166`,
    `Pattern string \`yaml:"pattern"\``, added by SPEC-035 REQ-004, whose own
    doc comment calls it "the inline rule pattern a pattern-arg engine passes
    as a command argument instead of resolving a rule file on disk." So for a
    `pattern-arg` rule the YAML `pattern:` key is silently DISCARDED at
    unmarshal — packval cannot even see the rule source such a rule declares.
    Consequence in `phase3.go`'s `RunFixtures`: `rule.File` is `""`, the
    guards at `phase3.go:31`, `:62` and `:76` are all false,
    `executor.RunEngine` is never invoked for either polarity, `res.Errors`
    stays empty and `res.Status` stays `"pass"`. Measured against the real
    in-repo pack, not inferred: `packs/contracts/pack.yml` declares
    `rule_path` ZERO times and `pattern:` SEVEN times (verified by count) —
    all 7 of its rules, 100% of the pack, can never have a fixture validated.
    A negative fixture rewritten to violate nothing, or a positive fixture
    rewritten to violate the rule, still prints `phase3-fixtures: pass`. A
    fourth dead site the issue itself does not name but `PLAN-ISSUE-092`'s F2
    finding does: `phase1.go:51` carries the same `rule.File != ""` guard, so
    the "the rule file you declared exists" structural check is dead for
    these rules too.

    **This is NOT a duplicate of item 4 (ISSUE-092), and item 4's fix does
    not close it.** Item 4 is the `rule_path:` declaration style — packval
    reads YAML key `file:` where every real pack writes `rule_path:`, a WRONG
    KEY NAME on a field that exists. This item is the `pattern-arg` style —
    neither `file:` nor `rule_path:` is declared, only an inline `pattern:`,
    and the field does not exist in packval's model AT ALL, a MISSING key.
    `PLAN-ISSUE-092` says so itself, in its F7-a finding: `packs/contracts`
    "CAN NEVER DISPATCH, EVEN AFTER THIS LANE… the accessor this plan
    introduces (CLM-001) returns EMPTY for every one of them and the dispatch
    guard stays false. This is a THIRD instance of the same drift family…
    Fixing this is modelling work on top of CLM-001, not a variation of it."
    That plan's F8 ruling (adopted Option 1) states explicitly:
    "`packs/contracts` stays dead (F7-a) — NOT a regression, exactly as
    vacuous as today, and follow-on (i) picks it up later."

    **The fix is a field PLUS a new invocation shape, which is what makes it
    a separate lane rather than a second patch on item 4.** Adding
    `Pattern string \`yaml:"pattern"\`` to `pkg/packval`'s `Rule` and widening
    the dispatch-eligibility guards is necessary but not sufficient: the call
    site today is `executor.RunEngine(packDir, binding, []string{rule.File,
    f.Path})`, an argv built for a FILE-based `input_mode`. A `pattern-arg`
    engine (`packs/contracts/pack.yml` declares `input_mode: pattern-arg` /
    `input_flag: --pattern`) wants the pattern string passed as a command
    argument, not a path. That argv shape has not been traced and is scoped
    to whoever plans this.

    **The root shape is now three-for-three, which is the argument for
    unifying the two `Rule` structs rather than patching a third field.**
    `pkg/pack/manifest.go` (runtime) and `pkg/packval/manifest.go`
    (authoring-time) are two independently-maintained models of the same
    YAML. Item 4 is drift on the rule-file key; this item is drift on
    `pattern`; `PLAN-ISSUE-092`'s F2 finding documents a THIRD — `rule.Layer`,
    retired from the runtime model by SPEC-031 REQ-002, still gates packval's
    layer-3 validator dispatch, so `RunValidator` is dead too. Whoever plans
    either item should weigh one unification against N field-by-field
    patches; recorded here as a question for the planner, not decided.

    **Provenance is the strongest fit signal this slot has.** ISSUE-142 was
    MANDATED by `PLAN-ISSUE-092` (`status: draft`), which is item 4's own
    lane. Its F7 review block found and deferred three deeper defects rather
    than fixing them in place, and its follow-ons list names this one as
    follow-on **(i)**: "packval's Rule model has no Pattern field —
    pattern-arg rules can never dispatch fixtures." Its two siblings from the
    same F7 block: **(ii) = `ISSUE-141`** ("packval's executor never applies
    `binding.Convert`"), filed 8 seconds earlier, which F8's ruling makes a
    HARD PREREQUISITE for that plan's final phase (TASK-015) — the plan
    states it must STOP AND REPORT if ISSUE-141's own fix has not landed by
    then; and **(iii)**, the `semgrep-rule-id` cross-check running
    unconditionally regardless of declared engine — baked semgrep knowledge,
    a thin-executor violation. As of this writing (iii) is no longer a
    separate follow-on to file: F8's ruling folded it directly into
    `PLAN-ISSUE-092` itself (its F9 section, `CLM-010`), conditioned on
    declared `input_mode` rather than a baked engine-name literal; what
    remains outstanding is a narrower, non-blocking requirements question
    about `BUNDLE-005 REQ-012`'s wording, recorded in that plan's own
    follow-ons list rather than as a fourth cluster member here.

    **Sequencing fact a planner needs, and it distinguishes this item from
    its siblings:** ISSUE-142 is the one of the three F7 follow-ons that
    `PLAN-ISSUE-092` does NOT block on — that plan's F8 ruling blocks only on
    (ii)/ISSUE-141, and explicitly defers this one ("follow-on (i) picks it
    up later"). It is genuinely independent work — but it is also the item
    that keeps `packs/contracts` vacuous until it lands, meaning any
    acceptance criterion of the form "`packs/contracts` passes `pack test`"
    stays satisfiable by a vacuous signal even after item 4 ships. `type:
    bug`, `scope: contained`, `uncertainty: known`, `risk: critical`.

17. **`pack test`/`pack check` phase3 never honors a binding's declared
    `stdout_artifact` — a dispatched engine's REAL output can sit unread in a
    file while the executor parses stdout noise instead (ISSUE-144).**
    `DefaultExecutor.RunEngine` (`pkg/packval/executor.go:61-98`) captures a
    run's raw stdout into a buffer and feeds it straight to
    `check.ParsePackFindings`; `binding.StdoutArtifact` appears NOWHERE in the
    file (grep clean). The real gate dispatch path resolves the payload
    BEFORE Convert/parse: `runFindingsEngine`
    (`cmd/backstop/pack_gate.go:759-766`) does `payload := stdout; if
    binding.StdoutArtifact != "" { read filepath.Join(projectRoot,
    binding.StdoutArtifact); a declared-but-missing artifact is a fail-loud
    broken run, never a silent fallback to stdout }`. Per SPEC-048
    (REQ-002/CLM-005..008, DEFECT-2) a `stdout_artifact`-declaring binding
    writes its real machine-readable output to that FILE and prints only a
    human summary to stdout — `RunEngine` has no equivalent step and always
    parses whatever landed in `stdout.Bytes()` regardless of what the binding
    declares.

    **The verdict-direction argument is what places this item here rather
    than beside ISSUE-141 in DIR-024, and it must be stated head-on, because
    DIR-024's ISSUE-141 note put a near-identical packval-vs-gate dispatch
    gap in that directive on the ground that it SHOUTS rather than lies.**
    Verified in the tree 2026-08-16, not relayed: `parseSarif`
    (`pkg/check/parsers.go:130-134`) trims its input and returns `(nil, nil)`
    — no error — on empty/whitespace input. When a `stdout_artifact`-declaring
    binding leaves stdout EMPTY (the normal shape for a tool that writes
    everything to its declared file), `RunEngine` returns
    `ExecutionResult{Passed: false, ExitCode: 0}` with a nil error, and
    `Passed: false` IS the success condition for a NEGATIVE phase3 fixture —
    a clean pass from a run whose real output was never read. That is a lying
    verdict, and it is item 15's (ISSUE-140) exact shape one mechanism over.
    Contrast ISSUE-141 explicitly: there the input is always a non-empty JSON
    ARRAY (ast-grep `--json` emits `[]` even for zero matches), so
    `json.Unmarshal` into the `sarifLog` struct ALWAYS fails and the failure
    is deterministically loud — precisely why DIR-024 owns it and DIR-032
    owns this. When stdout is instead non-empty non-JSON, this item degrades
    to ISSUE-141's loud shape; the mixed direction is why the empty-stdout
    path is the one that decides the home.

    **Ecosystem bound, stated honestly so nobody over-reads this item:**
    verified 2026-08-16 across installed non-testdata packs, exactly ONE
    binding declares `stdout_artifact` today — `go-coverage` in
    `.backstop/packs/backstop-ai/go-toolchain/pack.yml:57`
    (`stdout_artifact: cover.out`, `gate_type: coverage`) — and no rule
    declares that engine, so there is no live in-repo instance right now.
    `pkg/packval/phase3.go` resolves a rule's DECLARED engine to a binding
    with NO `gate_type` filter, so any pack whose rules bind a
    `stdout_artifact`-declaring findings engine hits this immediately. This is
    a live capability gap in the pack-authoring validator with zero current
    in-repo victims, not a currently-firing false green — a planner must not
    size it off a dogfood repro that does not exist yet.

    Direction, kept as constraint not design, taken from the issue: add a
    payload-selection step equivalent to `pack_gate.go`'s, applied BEFORE
    whatever Convert-application fix ISSUE-141 lands (Convert consumes the
    selected payload, not raw stdout). Two things a planner must settle
    rather than assume: (1) the correct base directory — `pack_gate.go`
    resolves against `projectRoot` while `RunEngine` runs with `cmd.Dir =
    packDir`, a different root, and the fix must pick the semantically
    correct base for the fixture-testing context rather than copying
    `projectRoot` verbatim; (2) whether ISSUE-143's proposed shared extraction
    has landed first, in which case this logic belongs in that same shared
    location, not a third independent copy. Verification bar: a fixture with
    a binding declaring `stdout_artifact`, a process writing real SARIF to
    that file while stdout is empty/unrelated, and an assertion that
    `RunEngine` parses the FILE's bytes, not stdout's.

    Sequencing: ISSUE-141 (DIR-024 item 13) is the ordering-relevant
    sibling — this item's stage precedes Convert in the real path's ordering,
    so where the new code goes depends on ISSUE-141 having landed first.
    `pkg/packval/executor.go` is also a live, actively-edited file
    (`PLAN-ISSUE-140` owns the never-started predicate there); do not open a
    lane concurrently with it.

    Provenance: filed 2026-08-16, minutes after ISSUE-143, by the same
    investigation that produced ISSUE-141/143; its author deliberately left it
    unslotted for backlog-pm triage rather than hand-editing a directive, and
    noted it "may fit DIR-032… alongside those siblings." `type: bug`,
    `scope: contained`, `uncertainty: known`, `risk: critical`.

## Notes

Grouping rationale and priority, stated once rather than per-item: four of
the fourteen roster members have in-flight plans as of 2026-08-16 — items 9
(ISSUE-112), 10 (ISSUE-113), 12 (ISSUE-118), and 13 (ISSUE-129), all
`status: draft` (see the paragraph below on why this is a coordinated drain,
not four incidental plans) — and the remaining ten, including new item 14,
are plan-free. This directive still imposes no top-down sequencing beyond
the intra-item notes above (e.g. items 9-10 as one arc, item 7 alongside its
non-member sibling ISSUE-107): position in the roster names committed scope,
not an execution order, and the four in-flight plans were sequenced by
whoever authored and is now running them, not by this directive's list
position. Item 4 (ISSUE-092, `risk: critical`, active false-green in the
tool that gates the entire pack ecosystem), item 3 (ISSUE-091, `risk:
critical`, the gate's own `--all` mode already produced one wrong founder
scope ruling), and item 12 (ISSUE-118, `risk: critical`, a mandated test
genuinely failing on the same tree/commit/binary that `backstop gate`
reported full PASS on) are the three with measured, already-realized
consequences — a planner picking this directive up cold should look at
those three first, not treat the list as strict priority order. Items 1-3
(066/067/091) predate the other eight by roughly two weeks and were never
blocked on anything; they simply had no directive to attach to until now.
**Correction (2026-08-16, later same day):** both the opening sentence and
the shortlist above are now stale. Opening sentence: the roster is now
FIFTEEN, not fourteen (item 15, ISSUE-140), and the four plans named
(PLAN-ISSUE-112, -113, -118, -129) are no longer in flight — all four
reached `status: completed` and their issues (ISSUE-112, 113, 118, 129)
reached `status: closed` the same day, per the "overnight P0 batch"
closeout (see the "In-flight execution note" correction below for detail).
The remaining count is therefore ELEVEN plan-free members, not ten — items
1-3, 5-8, 14, and the new item 15. Shortlist: item 12 (ISSUE-118) is now
DELIVERED and closed and drops off a cold-pickup shortlist by definition —
its consequence was real but is no longer live. Items 3 (ISSUE-091) and 4
(ISSUE-092) remain `status: open` and stay on the shortlist. **Item 15
(ISSUE-140, `risk: critical`) belongs alongside item 4** — both sit on the
same `pack test`/`pack check` surface, and a reader asking "can I trust
`pack test`?" needs both: item 4 is why fixtures can't falsify at all
(phase3 never dispatches for any `rule_path:`-declared rule), item 15 is
why a fixture that DOES dispatch can still read clean when the engine
command never started. The refreshed cold-pickup shortlist is items 3, 4,
and 15.

**Correction (2026-08-16, still later):** the shortlist above is now
incomplete on its own terms, not wrong in what it says — the roster grew to
SIXTEEN with item 16 (ISSUE-142), itself plan-free and new, so the plan-free
count is now TWELVE, not eleven. The substantive point for a cold picker:
**item 4's shortlist entry above is now incomplete on its own.** The
sentence just above already names the limit precisely — "item 4 is why
fixtures can't falsify at all (phase3 never dispatches for any
`rule_path:`-declared rule)" — and that parenthetical IS the boundary: item
16 is the `pattern-arg` half item 4 does not reach. Landing item 4 alone
leaves `packs/contracts` (all 7 rules, `pattern-arg`, 100% of the pack)
exactly as vacuous as today; item 16 is the missing half, not an optional
extra. So the honest answer to "can I trust `pack test`?" now needs THREE
items, not two: item 4 (`rule_path:` style), item 16 (`pattern-arg` style,
ISSUE-142), and item 15 (a dispatched engine that never started reading as a
clean negative). The refreshed cold-pickup shortlist is items 3, 4, 15, and
16.

**Correction (2026-08-16, still later than the above):** "the plan-free count
is now TWELVE, not eleven" in the correction directly above is WRONG — a
measurement error, attributable to the brief that produced it, not a fact
about the tree at the time. A file-by-file enumeration of `plans/` against
every roster member (verified 2026-08-16 ~21:45Z, `status:` read from each
file) finds a plan for every member except one:
`status: draft` (11) — PLAN-ISSUE-066, -067, -091, -092, -093, -097, -100,
-106, -114, -136, -140, i.e. items 1, 2, 3, 4, 5, 6, 7, 8, 11, 14, 15;
`status: completed` (4) — PLAN-ISSUE-112, -113, -118, -129, i.e. items 9, 10,
12, 13, whose issues are all `status: closed`. Plan-free: exactly ONE — item
16 (ISSUE-142). Ten of the eleven drafts are untracked in git and carry
`created: "2026-08-16"` with mtimes clustered around 17:00 local, consistent
with a fresh full-roster planning sweep rather than incremental pickup. The
consequence inverts the shortlist framing above: this directive is no longer
a queue of mostly-unplanned work — it is ONE uncovered member and fifteen
lanes already in various stages. Item 16 is the only thing here nobody has
planned, and it is uncovered BY CONSTRUCTION: `PLAN-ISSUE-092`, the plan for
the item it is the other half of, explicitly declines to cover it
("`packs/contracts` stays dead (F7-a) — that is NOT a regression… follow-on
(i) picks it up later"). The cold-pickup shortlist two corrections above
(items 3, 4, 15, and 16) needs the same qualifier: items 3, 4 and 15 now all
HAVE draft plans, so a cold picker's actual UNCOVERED entry point among the
four is item 16 alone — the other three need a planner's attention only
insofar as their existing drafts need review or their own F8-style blockers
need a ruling, not because nobody has started.

**In-flight execution note (2026-08-16):** the four plans above
(PLAN-ISSUE-112, PLAN-ISSUE-113, PLAN-ISSUE-118, PLAN-ISSUE-129) are not
four incidental plans — they were all authored in ONE commit, `5f28bb1`
("issue+plan: file ISSUE-134/135, author plans for P0 issues
112/113/118/122/129"), and PLAN-ISSUE-113 has since taken review rounds
(`c61ec1d`, "plan(ISSUE-113): 6 review rounds, ready to implement"). This
reads as a deliberate, coordinated P0 drain of this cluster, in flight right
now, not scattered opportunistic pickup — a reader picking this directive up
should treat roughly a third of the roster as already being actively
worked, not sitting queued. Recorded as observation only: this tension is
left for Brandon to resolve, not acted on here — whether an in-flight P0
drain covering four members means this directive's own `status: queued` is
now stale is a status-change judgment call outside backlog-pm's standing
grant, and the `status:` field is deliberately left untouched pending that
call. **Correction (2026-08-16, later same day):** the drain described above
is no longer "in flight right now" — it has LANDED. Verified directly:
`plans/PLAN-ISSUE-112-engine-tool-missing-silent-vacuous.plan.yml`,
`plans/PLAN-ISSUE-113-zero-match-classification-refusal.plan.yml`,
`plans/PLAN-ISSUE-118-gate-blind-spot-test-only-diffs.plan.yml`, and
`plans/PLAN-ISSUE-129-go-test-scope-filter-exemption.plan.yml` are all
`status: completed`; `issues/ISSUE-112`, `ISSUE-113`, `ISSUE-118`, and
`ISSUE-129` are all `status: closed` (closed 2026-08-16), delivered per the
"overnight P0 batch" closeout. The status-change judgment call flagged above
(whether this directive's own `status: queued` should move) is now sharper,
not resolved — a completed four-member drain is a stronger signal than an
in-flight one, and it remains Brandon's call, not backlog-pm's; the
`status:` field stays untouched here regardless. The one-third-of-roster
framing above is preserved as the observation it was at the time; the
current, correct framing is four of fifteen roster members DELIVERED, not
four of fourteen in flight. **Correction (2026-08-16, still later):** this is
now stale by one — the roster grew to SIXTEEN with item 16 (ISSUE-142); the
current, correct framing is four of SIXTEEN roster members DELIVERED.

Cross-directive note: item 6 (ISSUE-097) is rename fallout from DIR-027's
fleet migration (the `backstop/self` → `backstop-ai/backstop-self` rename),
but the durable fix — making an unbound waiver visible to the gate — is
gate/engine mechanism, so it stays homed here rather than DIR-027, which
owns publication/migration/lock-state and explicitly disclaims mechanism
design.

Explicitly NOT in this directive, so a reader does not assume the cluster is
larger than it is: ISSUE-096 (selfpack rule imprecision — related to item 4
but not itself a verdict-honesty defect), ISSUE-099 (gate cannot emit table
+ JSON together — ergonomics/cost, explicitly not a correctness defect),
ISSUE-107 (coverage warning-only step reads as pass — closely related to
item 7 and should be planned alongside it, but the founder's 11-member
enumeration did not include it) and ISSUE-108 (contract carrier drops pack
severity — same severity-contract family as item 8, same exclusion) all
remain homed in DIR-024. If a future founder ruling wants to fold any of
these four into this cluster, that is a fresh decision, not implied by this
carve-out.

ISSUE-129 ("go-test pack engine's failures are diff-scope-filtered") slotted
by backlog-pm 2026-08-15 under the standing clear-fit grant. It is a
POST-carve-out addition, same as item 12 before it: the founder's 2026-08-10
ruling enumerated eleven members, and items 12 and 13 were both slotted
afterward on charter fit, not by that original roster — a reader should not
infer the founder named thirteen. Why DIR-032 and not DIR-024, as charter
reasoning: DIR-024's own recent precedent draws this exact line (ISSUE-125's
slot note there says "DIR-032 is verdict honesty: GO-005 reports exactly the
verdict its regex earns, so the defect is rule precision, not a lying
verdict"). Here the verdict IS a lie — the gate says PASS while holding a
correctly-computed failure it discarded. DIR-027 "Pack Fleet Publication &
Migration" owns publication/migration/lock-state and explicitly disclaims
mechanism design, so the fact that the fix may land in a pack manifest
(`backstop-ai/go-toolchain`) does not move it there — the same reasoning
DIR-024 already recorded for ISSUE-096 and ISSUE-125. Also worth recording:
the ISSUE-129 file itself checked BUNDLE-003's "trustworthy-green guards"
seed (delivered as SPEC-068) and correctly found it does NOT own this — that
seed's REQs are artifact/schema-cohort validation trustworthiness, not
test-execution scope filtering. In-flight coverage is NIL and established
from the corpus, not assumed: no plan in `plans/` targets ISSUE-129 or the
go-test scope-filter exemption; the only artifacts naming
`exempt_from_scope_filter` are SPEC-040/SPEC-041 and their completed plans,
PLAN-ISSUE-070 (the closed sibling fix), PLAN-ISSUE-020, PLAN-ISSUE-027, and
ISSUE-052/070/129 themselves. Standard workaround-and-file shape: the
SPEC-070 implementer hit it, worked around it by hand-running the suite, and
filed. **Correction (2026-08-16):** the "In-flight coverage is NIL" line
above was accurate when written (2026-08-15) and is now stale — a plan
exists, `plans/PLAN-ISSUE-129-go-test-scope-filter-exemption.plan.yml`
(`status: draft`, created 2026-08-15), and its fix is visibly mid-flight in
the working tree. Preserved above as a record of the state at slotting time,
not as a current fact. Priority note, stated at the time as observation and
NOT as a
reorder (backlog-pm has no reorder authority): this is the FOURTH `risk:
critical` member of this directive with a measured, already-realized
consequence, alongside item 3 (ISSUE-091), item 4 (ISSUE-092), and item 12
(ISSUE-118). A separate proposal for Brandon sat in `.backstop/pm/INBOX.md`
at the time this note was written.

**Update (2026-08-15, founder-directed):** the founder acted on that
proposal during a pre-public-launch backlog sweep. This directive moved
from BACKLOG.yml position 5 to position 2 — see BACKLOG.yml's own
"REORDERED 2026-08-15" comment for the full reasoning. The line above ("this
directive's position in BACKLOG.yml is unchanged and must not be touched")
is preserved as a record of backlog-pm's stance at the time it wrote this
note; it no longer describes the current state and should not be read as
still in force.

ISSUE-136 ("no audit of findings-engine bindings for missing
`exempt_from_scope_filter`") slotted by backlog-pm 2026-08-16 under the
standing clear-fit grant. It is a POST-carve-out addition, same as items 12
and 13 before it — not part of the founder's original eleven-member roster.
Why DIR-032 and not elsewhere, as charter reasoning: the charter sentence it
matches is "a finding that IS computed correctly and then discarded before
verdict computation" (this directive's lede) — the audit bounds exactly that
class, engine by engine. DIR-024 "Gate/Engine Quality" was considered and
rejected on the line DIR-024's own ISSUE-125 note already drew ("DIR-032 is
verdict honesty: GO-005 reports exactly the verdict its regex earns, so the
defect is rule precision, not a lying verdict") — here the whole question
IS which verdicts are silently lying, so the same line places it here, not
there. DIR-027 "Pack Fleet Publication & Migration" was considered and
rejected because it owns publication/migration/lock-state and explicitly
disclaims mechanism design, the same reasoning already recorded above for
ISSUE-096 and ISSUE-129, even though any remediation an audit finding
produces would land in pack manifests. The strongest fit signal is direct,
not inferred: PLAN-ISSUE-129's own notes DEFER its Direction §2 explicitly —
"Direction §2 (audit every OTHER findings engine lacking the flag, so no
future pack engine reintroduces the gap silently) is DEFERRED — file it as a
follow-on issue rather than absorbing it here" — and repeat the instruction
in its FOLLOW-ONS-TO-FILE block ("Neither may be silently dropped, and
neither may be quietly absorbed into this plan."). ISSUE-136 is the artifact
that plan required to exist; its in-flight coverage is nil BY CONSTRUCTION
(the deferring plan said so directly), not merely nil by absence of
evidence. Recording the second follow-on from that same PLAN-ISSUE-129 block
so it is not lost: the released-pack / in-repo-fixture divergence risk — the
two `exempt_from_scope_filter` declarations (the released
`backstop-ai/go-toolchain` pack and the in-repo testdata fixture pack) are
maintained by hand with nothing asserting they agree; PLAN-ISSUE-129's
TASK-005 checks this once, manually, and nothing keeps it checked
thereafter. **It IS now filed, as ISSUE-137** ("No automated guard keeps the
go-toolchain pack fixture in sync with the released pack; a parallel
documentary copy is dead code," `type: technical-debt`, `scope: contained`,
`uncertainty: known`, `risk: moderate`, created 2026-08-16 — a diff of the
two files today shows only `name`/`version` differing, fixture
`backstop/go-toolchain` v1.1.0 vs released `backstop-ai/go-toolchain`
v1.4.0, i.e. they are currently in sync; the defect is that nothing KEEPS
them so). ISSUE-137 was homed in **DIR-024 "Gate/Engine Quality"** by
backlog-pm on 2026-08-16 under the standing clear-fit grant — as Description
item 12, with the reasoning recorded in that directive's Notes. (This
sentence previously read "no directive home yet"; that was true when
written, and stopped being true minutes later — the two issues were filed in
one commit and triaged by concurrent PM runs. Corrected in place rather than
left to read as an open question.) The home turned on DIR-032's own charter
boundary: nothing in ISSUE-137 reports a wrong gate verdict — the exemption
tests report exactly the verdict their fixture earns, and the drift risk
lives in backstop-core's own `go test` corpus, not in a gate step. That is
the same test that kept ISSUE-115 and ISSUE-125 in DIR-024. It remains **NOT owned by ISSUE-136** — that
distinction is why this note exists and it still holds: ISSUE-136 is the
engine-by-engine intent audit, ISSUE-137 is the fixture/released-pack sync
guard — sibling follow-ons from the same plan, different surfaces. Priority
note,
stated as observation and explicitly NOT as a reorder (backlog-pm has no
reorder authority): DIR-032 sits at BACKLOG.yml position 2 as of this
writing; this slot does not change its rank.

ISSUE-144 ("pkg/packval/executor.go's RunEngine never honors a binding's
declared StdoutArtifact — Convert/parse runs on the wrong bytes") slotted by
backlog-pm 2026-08-16 under the standing clear-fit grant. It is a
POST-carve-out addition, same as items 12, 13, 14 and 16 before it — not part
of the founder's original 2026-08-10 eleven-member roster; a reader should not
infer the founder named seventeen.

Why DIR-032 and not DIR-024, answered head-on: DIR-024 is the live competing
pull and took ISSUE-141 — a near-identical packval-vs-gate dispatch gap in the
SAME 35-line function, `RunEngine` — on the SHOUT-vs-LIE line only hours
earlier. The discriminator, verified in the tree by backlog-pm 2026-08-16 and
not relayed: `parseSarif` (`pkg/check/parsers.go:130-134`) trims its input and
returns `(nil, nil)` with NO error on empty/whitespace input, so a
`stdout_artifact`-declaring binding that leaves stdout empty — the normal
shape for a tool writing everything to its declared file — yields
`ExecutionResult{Passed: false, ExitCode: 0}` and a nil error, and `Passed:
false` IS the success condition for a NEGATIVE phase3 fixture. That is a clean
pass from a run whose real output was never read: a lying verdict, item 15's
shape one mechanism over. ISSUE-141 has no such path — ast-grep `--json`
emits a non-empty JSON array even for zero matches, so the struct unmarshal
ALWAYS fails and the failure is deterministically loud. This slot does NOT
disturb or contradict the ISSUE-141 ruling recorded in DIR-024's own Notes —
the two are consistent under one test, applied the same way both times: 141
shouts deterministically, 144 has a reachable silent path.

In-flight coverage is NIL, established from the corpus rather than assumed,
with ZERO interviews run and none needed: backlog-pm enumerated `plans/` on
2026-08-16 and no plan targets ISSUE-144 or StdoutArtifact payload selection;
`PLAN-ISSUE-141`'s scope is Convert application only (its own text flags
`binding.StdoutArtifact` as residual R2, explicitly out of scope — "folding
it in would be scope creep on a lane that is blocking"), and
`PLAN-ISSUE-140`'s own scope fence excludes the phase3 executor's payload
selection entirely.

**Correction (2026-08-16):** the roster-framing sentence earlier in these
Notes — "Item 16 is the only thing here nobody has planned" — is now stale by
one. Item 17 (ISSUE-144) is a second plan-free member, uncovered for the same
reason item 16 was: no plan has yet been authored against it. Preserved above
rather than rewritten, per this directive's own convention.

Priority note, stated as observation and explicitly NOT as a reorder
(backlog-pm has no reorder authority): DIR-032 sits at BACKLOG.yml position 2
and this slot does not change its rank; the existing directive-crossing
sequencing observation (DIR-024's item 13 gating DIR-032's item 4) already
sits as a PROPOSAL in `.backstop/pm/INBOX.md` for Brandon and is not
re-proposed here.
