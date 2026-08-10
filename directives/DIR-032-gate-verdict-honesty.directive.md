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
---

## Description

Carved out of DIR-024 "Gate/Engine Quality" per founder ruling (Brandon,
2026-08-10). Eleven issues share one defect shape: **a gate step computes a
result internally but reports the wrong verdict about it** — silent pass
when it should block, a scoped-clean signal when the unscoped truth is red,
an opaque crash where a legible finding belongs, or a dimension that never
fires at all for an entire artifact kind. This is the "no vacuous green"
invariant policing itself, and it is the exact failure mode the product
sells against — worth a dedicated home rather than diffusion across a
catch-all directive.

backlog-pm tracked this cluster growing from two issues (2026-07-26) to
eleven (2026-08-02) inside DIR-024's Notes, escalating the cluster-home
question repeatedly without acting on it unilaterally — see that directive's
Notes for the full history. Eight of the eleven were already cited in
DIR-024's `source:` frontmatter (ISSUE-092, 093, 097, 100, 106, 112, 113,
114) and move here with their Description/Notes prose intact. The remaining
three — ISSUE-066, ISSUE-067, ISSUE-091 — were named repeatedly in DIR-024's
Notes as cluster siblings but were **never added to its `source:` list**;
they had no directive home at all until now. All eleven are `status: open`
today; none has an in-flight plan.

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
  from an empty classification set.
- **Item 11 (114)** is the odd one structurally: it never computes anything
  for an entire artifact kind (plans), so it is silent by *starvation*
  rather than misreport.

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

## Notes

Grouping rationale and priority, stated once rather than per-item: none of
these eleven has an in-flight plan, so there is no top-down sequencing
imposed by this directive beyond the intra-item notes above (e.g. items 9-10
as one arc, item 7 alongside its non-member sibling ISSUE-107). Item 4
(ISSUE-092, `risk: critical`, active false-green in the tool that gates the
entire pack ecosystem) and item 3 (ISSUE-091, `risk: critical`, the gate's
own `--all` mode already produced one wrong founder scope ruling) are the
two with measured, already-realized consequences — a planner picking this
directive up cold should look at those two first, not treat the list as
strict priority order. Items 1-3 (066/067/091) predate the other eight by
roughly two weeks and were never blocked on anything; they simply had no
directive to attach to until now.

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
