---
title: "Gate/Engine Quality"
number: DIR-024
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: active
  source:
    - "ISSUE-007"
    - "ISSUE-020"
    - "ISSUE-082"
    - "ISSUE-075"
    - "ISSUE-077"
    - "ISSUE-096"
    - "ISSUE-099"
    - "ISSUE-107"
    - "ISSUE-108"
    - "ISSUE-115"
    - "ISSUE-125"
    - "ISSUE-135"
    - "ISSUE-137"
    - "ISSUE-141"
    - "ISSUE-143"
    - "ISSUE-145"
    - "ISSUE-147"
    - "ISSUE-158"
    - "ISSUE-131"
    - "ISSUE-163"
    - "ISSUE-165"
    - "ISSUE-166"
    - "ISSUE-168"
    - "ISSUE-170"
    - "ISSUE-169"
    - "ISSUE-175"
    - "ISSUE-177"
    - "ISSUE-179"
    - "ISSUE-180"
    - "ISSUE-174"
    - "ISSUE-024"
    - "ISSUE-065"
    - "ISSUE-076"
    - "ISSUE-117"
---

## Description

Seventeen gate/engine-quality gaps that don't fit the other three
newly-added directives' themes:

**Correction (2026-08-17):** the roster grew to NINETEEN with item 19
(ISSUE-131), slotted by directive-author under the standing clear-fit
grant, as item 3's (ISSUE-082) own plan-mandated residual — not a new
theme, and not by any founder roster call. It grew again to TWENTY the
same day with item 20 (ISSUE-163), likewise slotted by backlog-pm under
the standing clear-fit grant as item 1's (ISSUE-020) own delivery
residual — not a new theme, not a founder roster call. It grew again to
TWENTY-ONE on 2026-08-18 with item 21 (ISSUE-165), likewise slotted by
backlog-pm under the standing clear-fit grant as item 1's (ISSUE-020)
SECOND delivery residual — the guard test PLAN-ISSUE-020 itself
authored — not a new theme, not a founder roster call. It grew again to
TWENTY-TWO the same day with item 22 (ISSUE-166), likewise slotted by
backlog-pm under the standing clear-fit grant as another Linux-CI-
viability residual of item 1 (ISSUE-020) — surfaced by the same v0.2.0
release investigation that produced items 18 (ISSUE-158) and 20
(ISSUE-163) — not a new theme, not a founder roster call. It grew again
to TWENTY-THREE the same day with item 23 (ISSUE-168), likewise slotted
by backlog-pm under the standing clear-fit grant as a fourth Linux-CI-
viability residual of item 1 (ISSUE-020) — surfaced by the same v0.2.0
release investigation that produced items 18 (ISSUE-158), 20 (ISSUE-163)
and 22 (ISSUE-166) — not a new theme, not a founder roster call. It grew
again to TWENTY-FOUR on 2026-08-18 with item 24 (ISSUE-170), slotted by
backlog-pm under the standing clear-fit grant as a SECOND-ORDER residual —
item 21's (ISSUE-165) own delivery residual, which makes it item 1's
(ISSUE-020) grand-residual, since ISSUE-165 was itself item 1's residual —
not a new theme, not a founder roster call. It grew again to TWENTY-FIVE
the same day with item 25 (ISSUE-169), slotted by backlog-pm under the
standing clear-fit grant as the DELIVERY RESIDUAL OF ITEM 21 (ISSUE-165) —
mandated by that item's own plan, `PLAN-ISSUE-165` TASK-005 step (5), which
explicitly refuses to absorb it — not a new theme, not a founder roster
call. It grew again to TWENTY-SIX on 2026-08-18 with item 26 (ISSUE-175),
slotted by backlog-pm under the standing clear-fit grant as the DELIVERY
RESIDUAL OF ITEM 22 (ISSUE-166) — mandated by that item's own plan,
`PLAN-ISSUE-166`, whose TASK-004/TASK-010 text explicitly refuses to absorb
it — not a new theme, not a founder roster call. It grew again to
TWENTY-SEVEN the same day with item 27 (ISSUE-177), slotted by backlog-pm
under the standing clear-fit grant as ANOTHER RESIDUAL OF ITEM 22
(ISSUE-166) — that same fix's own affected-test list names the test but
`PLAN-ISSUE-166`'s fix does not clear it, and the asymmetry between it and
its cleared siblings is exactly what the issue exists to investigate — not
a new theme, not a founder roster call. It grew again to TWENTY-EIGHT on
2026-08-19 with item 28 (ISSUE-179), slotted by backlog-pm under the
standing clear-fit grant as a measured no-op in the coverage-reuse speedup
`PLAN-ISSUE-172` shipped — not a new theme, not a founder roster call. It
grew again to TWENTY-NINE the same day with item 29 (ISSUE-180), slotted by
backlog-pm under the standing clear-fit grant as a further Linux-CI-
viability residual of item 1 (ISSUE-020), in the same v0.2.0-release-
investigation family as items 18/20/21/22/23/27 — not a new theme, not a
founder roster call. It grew again to THIRTY-FOUR on 2026-08-19 with items
30-34 (ISSUE-174, ISSUE-024, ISSUE-065, ISSUE-076, ISSUE-117), added by
directive-author in one batch — unlike every prior addition above, NOT a
self-initiated clear-fit slotting under the standing grant, but a
founder-ratified batch of decisions from a completed backlog-pm
investigation sweep, relayed via team-lead. See the Notes section for each
item's individual provenance.

1. **Cross-platform sandbox — Linux is a hard no-op (ISSUE-020).**
   `pkg/packval/sandbox.go`'s `SandboxedRun` / `SandboxedRunStdout` dispatch
   on `runtime.GOOS`: the `darwin` branch wraps pack-supplied convert
   scripts and sandbox-validators under `sandbox-exec` (deny-default,
   deny-network, deny-file-write); the `linux` branch unconditionally
   returns `"sandbox unavailable on linux in this build"` — confirmed still
   the case in the current tree. Since the caller treats any sandbox error
   as a hard failure, no pack convert script or validator can run on Linux
   at all today: the gate is non-functional on Linux, not merely
   unsandboxed. This is security-elevated — the OS sandbox is the only
   trust boundary between arbitrary pack-supplied scripts and the host —
   and must close before CI runs on any Linux host or any third-party pack
   is installed. Candidate mechanisms (bubblewrap, Landlock, seccomp, user
   namespaces) are noted in the issue for the planner, not decided; any
   implementation must hold deny-network/deny-write parity with the macOS
   profile and fail loud rather than silently pass through if the chosen
   mechanism is unavailable on the host kernel.
2. **Build/test pass exclusion mechanism (ISSUE-007).** Originally: no
   `exclude_paths`-style mechanism exists for trees that must not gate
   build/test (e.g. intentionally-uncompilable fixture directories). The
   motivating case in this repo (`prototype/`) was deleted 2026-06-12, and
   the issue itself demoted this to "defer until a real consumer needs it."
   **Flagging for the founder:** this issue's own file still cites
   `buildExecutor`/`testExecutor` in `pkg/check/check.go` as the mechanism
   to extend — a `grep` against the current tree found neither identifier;
   they were removed by the later native-toolchain cutover
   (`project_native_toolchain_cutover`, 2026-07-03) that deleted
   `builtinToolchain`/`realCodeChecker` outright. The underlying concern
   (repos need a way to declare out-of-scope trees for pack-driven
   build/test steps) is still generically valid post-packs-only, but the
   concrete fix site this issue names no longer exists — it needs
   re-scoping to wherever pack-declared build/test steps live today before
   a plan is written against it, not a literal read of the issue's current
   "Fix sketch."
3. **Tool allowlist unreachable entries + overstated guarantee (ISSUE-082).**
   `engine.TrustedToolAllowlist()` (`pkg/pack/engine/allowlist.go`) declares
   eight pinned tool entries, but every call site of `CheckToolAllowed`
   (`cmd/backstop/pack_gate.go:813`, `pkg/pack/manifest.go:547`,
   `pkg/packval/executor.go:63`, `cmd/backstop/pack_gate_provision.go:85`)
   exempts bindings with a nil `Provision`. A sweep of every pack.yml in
   this repo and in `~/src/projects/backstop-packs` found `provision:`
   blocks for only three tools (grep, ast-grep, semgrep) — so `rg`,
   `oxlint`, `bun`, `tsc`, `prettier` are dead entries. `typescript-toolchain`
   (the pack those four were added for) declares no `provision:` at all; it
   shells out via `npx --no-install`.
   Two defects, both cleanup-only: dead entries carrying a
   `// nosemgrep: no-baked-language-token` suppression (the `backstop/self`
   rule fired correctly on a TypeScript tool name in a core Go file, and the
   fix was a suppression rather than not adding the entry — removing the
   entries removes the suppression), and a doc comment claiming a tool
   absent from the map "may not be run by any pack-declared command, no
   matter what a pack declares," which is false given the nil-`Provision`
   exemption. What the map actually governs is the trust floor for tools
   backstop provisions and pins on the user's behalf via the Provision/lock
   path.
   Scope boundary: this is deletion + doc correction only. The governance
   question for arbitrary pack-declared commands outside the Provision path
   belongs to bundles/BUNDLE-021-pack-command-execution-governance.bundle.md
   (exploring, not yet scoped). Do NOT design or implement an enforcement
   mechanism against ISSUE-082.
4. **Smoke fixture vacuous green on a missing mandated test (ISSUE-075).**
   `TestSmoke_GateFailsMissingMandatedTest` (`tests/smoke/smoke_test.go:486`)
   is meant to prove that `test_verification` blocks on a spec's missing
   mandated test, but its `createSpec` helper hardcodes `status: draft` with
   no override — so ISSUE-054's implemented-only mandated-test scoping
   filters the test out before enforcement ever runs, and the scenario
   passes green while proving nothing. The fix is scoped and small: add a
   status override to `specOpts`/`createSpec`, force `SPEC-999` to
   `implemented` for this scenario, and re-run the smoke suite with
   `-count=1` to surface any sibling scenarios cache-masked the same way.
5. **Stale gate binary produces phantom violations (ISSUE-077).** Bare
   `backstop` on `PATH` resolves to whichever copy was last manually
   rebuilt, which can be days stale relative to the working tree; a stale
   binary parsing a pack's updated rule-message format has already produced
   3 phantom violations against a correct diff (the ISSUE-062 incident),
   while `go test` — which always builds from source — passed clean. Two
   complementary fixes are proposed in the issue: make the `PATH` entry a
   shim that rebuilds-then-execs the fresh binary, and a defense-in-depth
   self-check at gate startup that exits 2 (not a phantom violation) if the
   running binary is older than the newest tracked `.go` file. Primarily a
   contributor on-ramp problem for this repo, not consumer-facing — a
   released install has only one binary copy and cannot diverge from itself.
6. **Self-pack rule imprecision forces un-adjudicated escapes in test files
   (ISSUE-096).** The `backstop-ai/backstop-self` pack (v1.1.2, installed at
   `.backstop/packs/backstop-ai/backstop-self`, source repo
   `/Users/bmanson/src/projects/backstop-self-pack`) ships seven rule
   families in `rules/no-baked.yml`. Five carry a `*_test.go` path exclusion
   (lines 73, 110, 144, 185, 230 of that file — families B2/B3/B4/B5/B6).
   ISSUE-096 asks for the same exclusion on Family A (`no-baked-tool-exec`,
   `rules/no-baked.yml:6-29`), whose only exclusion today is
   `tests/smoke/**`. The rationale is B2's own, founder-ratified
   2026-07-27: backstop-core IS the module under test, so its own harness
   tests must NAME the real tool they drive; that is testing the subject,
   not shipping baked routing.
   The issue's site inventory is accurate — independently re-measured, not
   taken on faith. Running `./bin/backstop gate --file` over the named
   files produced exactly 7 live Family A findings across 5 files:
   `cmd/backstop/pack_authoring_loop_test.go` (×1),
   `pkg/gate/self_rule_test.go` (×1),
   `pkg/pack/engine/contracts_grep_engine_test.go` (×1),
   `pkg/pack/engine/import_cycle_test.go` (×1),
   `pkg/validate/spec_hook_test.go` (×3). Each is a legitimate self-test
   naming the tool it drives (the Go toolchain, `bash`, `grep`, `semgrep`).
   The issue's central scope claim is FALSE, and this must be corrected
   before anyone plans against it. ISSUE-096 states Family A "is now the
   **only** family without" the exclusion. It is not. Family B1
   (`no-baked-tool-command`, `rules/no-baked.yml:34-43`) carries NO
   `paths:` block at all — not `*_test.go`, not even `tests/smoke/**`. And
   unlike Family A's dormant sites, B1 has measured live exposure:
   `./bin/backstop gate --file cmd/backstop/gate_wiring_test.go` returns 4
   Family B1 violations on the `Command: "go test ./..."` engine-binding
   literals at `cmd/backstop/gate_wiring_test.go:100,135,160,183` — test
   fixtures constructing engine bindings to assert the gate's own wiring,
   the same self-testing category B2's comment describes. Consequence for
   the fix: a pack bump that patches Family A alone leaves the next
   implementer who touches `gate_wiring_test.go` blocked by the identical
   defect class, one family over. Any plan against this issue should cover
   A and B1 together in one version bump, and should re-derive the family
   inventory from the rules file rather than from the issue's prose.
   Two workarounds already in the tree are what this issue exists to
   retire, both confirmed by measurement: `cmd/backstop/integration_test.go`
   returns zero findings because its `go build` call was rewritten during
   PLAN-ISSUE-020 Phase 4 (2026-07-28) to route through the package's
   parametric `execCommand` helper (`cmd/backstop/root_test.go:360`);
   `cmd/backstop/version_test.go` returns zero findings because of an
   inline
   `// nosemgrep: backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec`
   comment at `:168` (added ISSUE-087 phase 2). The second is the one that
   matters beyond ergonomics: a semgrep-native suppression is consumed
   inside the engine and never reaches backstop's own `@waiver:`
   adjudication (`pkg/waiver/`), so that call site is invisible to the
   `waiver_resolution` gate step and appears in no waiver ledger. Core is
   aware of the channel and drops such findings by design
   (`pkg/check/parsers.go:80,115`).
   Fix location and shape, restated as constraint not design: the change
   lives in the pack repo (`backstop-self-pack/rules/no-baked.yml` + a
   version bump past 1.1.2 + fixture pair + tag), not in core; core's only
   step is picking up the new version (`pack update`, not `pack add` — see
   the ISSUE-095 note below). Per `packs_learn_from_scenarios`/
   `waivers_are_last_resort`, the pack is the durable fix and the two
   in-tree workarounds become removable afterward — with the removal
   itself run as the falsifying check that the pack fix actually subsumes
   them.
   The issue's own verification step is vacuous today, and this is the
   sharpest thing a planner needs to know. ISSUE-096 step 3 prescribes
   re-running `backstop pack test` / `pack check` to confirm the new
   exclusion doesn't blind Family A to a real violation. Per item 6 of
   this directive (ISSUE-092), `pack test` phase 3 executes zero fixture
   checks for any rule declared via `rule_path:` — which is every rule in
   this pack — and returns `phase3-fixtures: pass` regardless. So the
   prescribed falsification cannot falsify while ISSUE-092 stands. Either
   sequence ISSUE-092 first, or substitute an explicit hand-run
   falsification (semgrep or `gate --file` against a known-violating
   non-test fixture) and say so in the plan.
7. **`gate` cannot emit the human table and JSON in one run — a measured
    2x CI cost (ISSUE-099).** **DELIVERED — PLAN-ISSUE-099, commit
    `e20e960`, 2026-08-18 (4 review rounds, signed off CLEAN).** The three
    factual claims below are corrected as of 2026-08-19 (directive-author),
    since PLAN-ISSUE-099 shipped and made them stale the moment it landed;
    left as originally written they'd misdirect a future reader.
    `--json` is a plain boolean on the global `jsonFlag`, registered as a
    ROOT PERSISTENT flag in `cmd/backstop/root.go`
    (`rootCmd.PersistentFlags().BoolVar(&jsonFlag, "json", ...)`, `:42`) —
    corrected citation; `cmd/backstop/gate.go:33` is not the registration
    site, only where the command receives the already-registered flag.
    When set, `gate.FormatJSON(result)` REPLACES the human-table render
    path on stdout rather than running alongside it — this part of the
    original claim still holds, confirmed live at `cmd/backstop/gate.go:
    319-349`. **Corrected as of PLAN-ISSUE-099:** a `--json-out FILE` flag
    now exists — gate-local, registered in `cmd/backstop/gate.go`'s
    `newGateCommand` (`:41,74-78`) — that writes the gate/v1 JSON envelope
    to FILE independently of what `--json` sends to stdout. The claim that
    no such flag existed anywhere in `cmd/backstop/` was true when this
    item was written and is false now.
    Consumer impact is measured, not theoretical: `.github/workflows/
    ci.yml` runs the entire gate TWICE — a diagnostic `--json` capture
    (`:144`) and the blocking human-table run (`:152`). Both execute the
    full kill chain including `pack_engines`, the expensive step that
    shells out to every dispatched engine. Measured cost: the gate step's
    wall time roughly doubled, ~2m40s → ~5m per run.
    Desired shape per the issue: a `--json-out FILE` flag that writes the
    JSON envelope to FILE while the human table still renders to stdout,
    collapsing ci.yml's two invocations into one.
    Accuracy correction the issue file needs, and a planner must not trip
    on: ISSUE-099 quotes a `.github/workflows/ci.yml` block at lines
    126-147 using a `set +e` / diagnostic-run / `set -e` bracketing
    structure, and its "Retirement trigger" paragraph says that bracketing
    should merge away when the flag ships. That shape no longer exists.
    ci.yml has since been rewritten into two separately-named steps
    (`Capture the gate report as JSON (diagnostic only - does not gate)` at
    `:139-144`, then `Run the gate` at `:146-152`) using a `|| echo`
    one-liner instead of `set +e`/`set -e` — and the file already cites
    ISSUE-099 by ID in its own comment. At the time this item was written,
    the deliberate design reason recorded in that comment (separate steps
    make it visible from the step list alone which invocation gates, a
    lesson from the retired linux-sandbox probe workflow whose every step
    ended in `exit 0`) was something a future closer needed to preserve
    rather than collapse blindly. **Corrected as of PLAN-ISSUE-099, commit
    `e20e960`:** that instruction is now retired, not still owed. The two
    steps WERE the retirement trigger's target, and PLAN-ISSUE-099 —
    "whoever closed this" — deliberately collapsed them into the single
    `Run the gate` step (`.github/workflows/ci.yml:144-195`) using
    `--json-out gate-report.json`, per its own commit message and the
    `ci.yml` comment block at `:166` ("ONE INVOCATION, BOTH SURFACES
    (ISSUE-099)"). This was correct, not a regression to flag: the entire
    reason the two-step split existed was that no single invocation could
    produce both the human table on stdout AND a JSON envelope on disk: at
    that time `--json` could only produce one or the other. Once
    `--json-out` exists, the two surfaces come from one `FormatJSON` call
    over one `GateResult` in one gate run, so a single step now serves
    both purposes and the visibility-from-the-step-list rationale no
    longer has anything to protect. Do not read this item as still
    instructing preservation of the split.
    Route the issue-file correction through issue-author; it is NOT this
    directive's scope.
    Second consumer, independent of CI: the client-portal traceability
    feed renders gate JSON, so a file-emitting flag serves it too.
    Classification per the founder's loud-≠-blocking law: this is
    ergonomics/cost debt with a measured cost, NOT a correctness defect —
    nothing here produces a wrong verdict. It must NOT be treated as a
    member of the gate-verdict-honesty cluster.
8. **Warning-only coverage step reads as pass (ISSUE-107).**
    `StepCoverageThreshold`'s own verdict loop (`pkg/gate/step_coverage.go:
    213-219`) initializes `status := "pass"` and sets `"fail"` only when a
    violation has `Severity == "error"`. There is no else branch, so a
    coverage step whose violations are ALL `severity: "warning"` reports
    `"pass"`.
    This is the INVERSE of ISSUE-105's defect: there, non-blocking became
    blocking; here, loud becomes silent. Both violate the same pack
    severity contract on `blocksVerdict`.
    The live instance class exists today: `step_coverage.go:174-181` emits
    a `coverage_exclusion` violation at `Severity: "warning"` whenever an
    in-scope changed file's coverage requirement is suppressed by a
    pack-declared exclusion. backstop-core itself is masked from the
    defect only because its own `backstop.yml` declares a coverage policy
    entry, routing the step through `ApplyPolicy`'s severity-aware
    override. A POLICY-SPARSE consumer with the identical finding set gets
    `"pass"` straight out of the step builder — `ApplyPolicy` only
    overrides steps that have a policy entry.
    Consequence: `"pass"` never increments `GateResult.StepsWarned` and
    never renders as distinguishable from a genuinely clean run, so the
    notice whose entire purpose is visibility becomes invisible to exactly
    the population the warning tier exists to protect.
    Fix shape is small and local: compute the tri-state through the
    EXISTING `gate.StepVerdict` (`pkg/gate/policy.go:125`) rather than
    re-deriving it — `StepVerdict`'s own doc comment says it exists so
    there is exactly one severity predicate and "never a second spelling,"
    and its "WHY WARNING-ONLY IS NOT pass" paragraph already argues this
    issue's case verbatim. Both coverage violation constructors already
    set `Severity` explicitly, so NO upstream severity-plumbing change is
    needed here (unlike ISSUE-106/108).
    Note for the planner: this is a REPORTING change — a step that reads
    `"pass"` today will read `"warning"`. Expected blast radius is zero
    flips on backstop-core's own dogfood run (its policy entry already
    corrects it); the flip appears only on a policy-sparse consumer.
9. **The contract-signature carrier cannot represent a pack-declared
    severity at all (ISSUE-108).** `ContractEngineResult`
    (`pkg/gate/contract_verdict.go:31-36`) is the gate-side carrier of one
    pack engine probe result for one contract entry, and its fields are
    exactly `Entry`, `Matched`, `Scanned`, `Locations` — confirmed in the
    current tree. There is no `Severity` member, so
    `produceContractEngineResults` (`cmd/backstop/gate.go`, in
    `buildContractStep`) has nowhere to put a pack-declared severity even if
    the contracts SARIF dispatch produced one. Consequently
    `VerifyContractVerdict` hardcodes `Severity: "error"` on all three of
    its violation-returning branches — confirmed live at
    `contract_verdict.go:77` (unscanned-scope config error), `:85`
    (forbidden-symbol-present absence violation) and `:101`
    (missing-signature present-contract violation). Every
    `contract_signature` finding therefore blocks unconditionally, and a
    pack that wants a soft-migration signature flagged-but-not-gating has
    no mechanism to say so.
    Why this is a different KIND of defect from its siblings, and the
    reason it is worth its own item: ISSUE-106 and ISSUE-107 are sites where
    a pack-declared value exists upstream and is then discarded or
    misread. Here the value cannot exist — the type cannot represent it.
    `contract_signature` is structurally warning-free BY CONSTRUCTION, not
    by policy. That reclassification was not inferred; it was MEASURED
    during the ISSUE-105 implementation and recorded as evidence in
    `pkg/gate/step_verdict_severity_test.go:145-193`, where the implementer
    wrote that this site "cannot be handed a declared warning at all" and
    reclassified it from the plan's CLASS 1 (a slice carrying a
    pack-resolved severity) to CLASS 3 (structurally error-only).
    The coupled test, which whoever plans this MUST name in the plan
    rather than discover as a red build:
    `TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy`
    (`pkg/gate/step_verdict_severity_test.go:163`, confirmed present) is a
    SELF-REPORTING PREMISE GUARD, not a static assertion. It asserts
    today's construction (`Violations[0].Severity == "error"`) and carries
    an inline warning that a non-`error` result means the CLASS-3 reading
    has gone stale; it also asserts the contingent half directly against
    `StepVerdict` because no constructible input can exercise it today. The
    moment this issue adds a `Severity` field, that premise flips false.
    The test must be revised deliberately as part of this fix into a
    genuine warning-input case — the same shape
    `TestStepArtifactValidation_DeclaredWarningDoesNotFailWithoutPolicy`
    (`step_verdict_severity_test.go:76`) already provides for its own site.
    Its sibling `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
    (`:265`) is the regression lock for the three sites that legitimately
    stay raw-count and must NOT be loosened by this fix.
    One decision the issue deliberately leaves open, restated here as
    constraint so nobody defaults past it: whether ALL THREE violation
    branches become severity-driven or only some. The unscanned-scope
    branch at `:77` is a config/scan-completeness failure, not a
    pack-declared advisory, and ISSUE-105 left `ConfigErr` branches
    hardcoded everywhere else in the codebase on exactly that reasoning
    (locked by `TestDelegateSteps_ConfigErrorStillFailsRegardlessOfSeverity`,
    `step_verdict_severity_test.go:122-143`). The plan must state its
    choice, not thread severity everywhere the field newly exists. Also
    required: an empty or absent severity must default fail-closed to
    `"error"` per the law `blocksVerdict` already documents
    (`pkg/gate/policy.go:63-67`), mirroring the `nonEmptySeverity`-equivalent
    defaulting at `substantiveness_q1_dispatch.go:71,110-113`; and step 2
    of the fix is NOT a trivial pass-through — whether the contracts pack
    engine dispatch carries a severity to that point at all needs verifying
    before the field is assumed populatable.
    Blast-radius bar from the issue, worth preserving: measure
    `contract_signature` verdicts on backstop-core's own dogfood run and at
    least one fixture consumer before and after; today's behavior (every
    contract violation blocks) is the expected floor, and core should show
    zero flips unless it declares a non-default severity on a contract
    entry.
10. **Contract block drift is invisible until a spec closes — validation
    happens at the worst possible moment (ISSUE-115).** `ExtractContractEntries`
    (`pkg/gate/step_testverify.go`) skips any spec where
    `!contractsAreDue(fm.Status)`, and `contractsAreDue` is literally
    `status == "implemented"`. So a spec's `contracts:` block is INERT for its
    entire `draft`/`ready-for-implementation` life and goes live only the
    instant the spec closes. The code the contracts describe keeps evolving
    the whole time with nothing checking it — staleness accumulates unobserved
    and detonates at closure, i.e. someone closing a spec that shipped
    correctly weeks ago gets a red gate for it.
    Evidence, stated precisely and not inflated: in one closure pass on
    2026-08-02, five specs were examined; SPEC-019 was clean (all 8 declared
    signatures verified). Three were stale — SPEC-018 (declared a
    package-level `var gateCmd` that never existed; the real surface,
    `newGateCommand`, was already declared immediately below the stale entry,
    so it was also a duplicate), SPEC-030 (declared
    `(*realCodeChecker).runCheck`, a symbol whose DELETION is what that spec
    delivered — `TestCutover_RealCodeCheckerDeleted` asserts its absence), and
    SPEC-036 (`CapabilityState` declared with four fields; the tree has five).
    Root-cause pattern worth preserving from SPEC-030: that contract entry
    belonged to REQ-002, retired at spec_version 2.0.0 — the contract block
    outlived the requirement that justified it. Retiring a requirement does
    not retire its contracts.
    Two findings widen the fix's target. (a) `consumes:` entries are never
    enforced at all — extraction iterates `Provides` only — so that staleness
    is PERMANENTLY invisible, not merely deferred to closure. Confirmed live:
    SPEC-018 consumes `ComputeChangedFiles` from `pkg/check/scope.go`, which
    exists nowhere in the repo; SPEC-030 carries a whole contract block for
    `cmd/backstop/code_check.go`, a file that does not exist. (b) Not all
    drift is equally dangerous: running the real `go-contracts` compiler
    shows a `kind: type` signature renders field-agnostically (`type
    CapabilityState $$$`), so SPEC-036's four-vs-five-field drift would NOT
    have failed the gate — a documentation defect, not a blocking one —
    while a declared symbol that doesn't exist at all (SPEC-018's var,
    SPEC-030's deleted method) DOES fail. The surface splits into "wrong but
    harmless" and "wrong and blocking," and a fix that only catches the
    blocking kind still leaves the corpus lying.
    Direction, recorded as the issue's recommendation and not a decision:
    validate contract blocks against the tree at closure time, BEFORE the
    status flip to `implemented` is accepted, so a spec cannot reach terminal
    status carrying contracts that don't match. Natural home is wherever
    artifact validation already runs. Left to the planner: whether
    `consumes:` becomes enforced or is dropped as decorative — today it is
    neither.
    Scope boundary, stated so nobody folds it in: a separate defect found in
    the same pass — the `go-contracts` pack's `compile-signature.sh` cannot
    express a grouped `const ( … )` block member (no `const_spec` pattern),
    which is why SPEC-036 could not close — is PACK work and gets no
    backstop-core issue, per the same principle as the ISSUE-103 ruling.
    Referenced here as related context only.
11. **go-standards `constructor-injection` rule matches field-name substrings,
    not whole words (ISSUE-125).** The `backstop-ai/go-standards` rule
    `go.core.constructor-injection` (`rule_id: GO-005`, `severity: WARNING`)
    constrains its `$FIELD` metavariable with `regex: (?i).*(repo|client|
    store|db|database|cache|logger|service|gateway|adapter|provider).*` — an
    unanchored, case-insensitive substring match. Any struct field whose NAME
    MERELY CONTAINS one of those tokens fires the rule regardless of whether
    it is a dependency. Verified live in the installed pack at
    `.backstop/packs/backstop-ai/go-standards/rules/core/go-core.yml:49-56`
    (independently re-read, not taken from the issue's prose). Live instance:
    `pkg/scaffold/idresolver_test.go:16` declares `isRepo bool` on
    `mockGitExecutor` — a "is this directory a git repo" flag — and `repo`
    matches as a substring of `isRepo`, firing on all 15 table-driven struct
    literals that set it. Because GO-005 is WARNING severity it does not
    block the gate; classification per the founder's loud-≠-blocking law is
    SIGNAL DEBT, not a correctness defect — no verdict is wrong, but every
    hit is noise that trains reviewers to skim past GO-005 and erodes the
    rule's real signal on genuine direct-wiring violations. It must NOT be
    treated as a member of the gate-verdict-honesty cluster (DIR-032).
    Fix location as constraint, not design: the change lives in the PACK repo
    — source at `/Users/bmanson/src/projects/backstop-go-pack` (`name:
    backstop-ai/go-standards`, `version: "1.2.1"`, working tree clean, tags
    `v1.2.1`/`v1.2.0`) — plus a version bump and tag, then `pack update` in
    this repo; the installed lock entry is `source_type: git`, `git_ref:
    v1.2.1`. Never rename `isRepo` in core to satisfy an overly broad
    pattern. One trap the issue names and a planner must not skip: Go field
    names are camelCase, so a literal `\b` anchor does NOT reliably split
    `isRepo` from `Repo` — any candidate regex must be falsified against
    `isRepo`/`dbTimeout`-shaped names before landing, not merely confirmed to
    still match bare `repo`/`db`.
12. **No automated guard keeps the in-repo `go-toolchain` pack fixture in sync
    with the released pack; a third documentary copy is dead code
    (ISSUE-137).** Two files in this repo describe the SAME external thing —
    the released `backstop-ai/go-toolchain` pack's engine bindings — and
    nothing automated asserts they agree: the released pack consumed via
    `backstop.lock` and installed at
    `.backstop/packs/backstop-ai/go-toolchain/pack.yml`, and the in-repo
    fixture at
    `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`
    that `goToolchainPackRoot()`/`goToolchainManifest()`
    (`cmd/backstop/pack_gate_gotoolchain_test.go:15,87`) actually parse.
    Independently re-measured 2026-08-16, not taken from the issue's prose:
    `diff` between the two returns exactly two hunks — `name`
    (`backstop/go-toolchain` vs `backstop-ai/go-toolchain`) and `version`
    (`1.1.0` vs `"1.4.0"`). Every engine binding is byte-identical today. So
    the fixture IS a maintained mirror, and this has a direct consequence for
    the fix: a byte-equality guard is the WRONG shape, because the two files
    differ on identity fields BY DESIGN (the fixture deliberately keeps the
    pre-rename name). The guard must compare engine bindings — command,
    scope_kind, gate_type, exempt_from_scope_filter, convert — and ignore
    name/version.
    BLAST RADIUS CORRECTION, and a planner needs it: the issue frames the
    fixture as what "every SPEC-041 exemption test reads." That understates
    it. A grep for `goToolchainManifest`/`goToolchainPackRoot` returns ~30
    call sites across roughly fifteen test files in `cmd/backstop` —
    `pack_separation_test.go`, `go_toolchain_engines_test.go`,
    `filemode_scoping_test.go`, `pack_gate_provision_test.go`,
    `golden_equivalence_test.go`, `dispatch_coverage_e2e_test.go`,
    `bridge_test.go`, `gate_no_toolchain_pack_test.go`,
    `pack_gate_scope_test.go`, `gate_transitional_seams_test.go`,
    `coverage_convert_enriched_test.go`, `pack_gate_gotoolchain_test.go`, and
    `spec046_fixtures_test.go` (which copies the entire fixture pack root
    recursively). A stale fixture therefore mis-asserts engine dispatch,
    file-mode scoping, provisioning, coverage conversion and
    golden-equivalence behavior — not only exemption semantics.
    THE SCOPE-OUT IN THE ISSUE IS CORRECT, and for a stronger reason than the
    issue gives — verified rather than assumed, so nobody re-derives it: the
    two other `testdata/**/go-toolchain/pack.yml` fixtures (`classifier-e2e`,
    `spec045-discovery`) must NOT get the same treatment, because they are
    not mirrors at all. Both are `version: 1.0.0` with `archetype: code`
    where the released pack is `enforcement`; `spec045-discovery` declares NO
    `engines:` block whatsoever, and `classifier-e2e` declares exactly one
    deliberately-unbound coverage engine. They are purpose-built reduced
    fixtures for classification and discovery tests. A parity guard pointed
    at them would fail on correct files. The `testdata/go-toolchain/` copy is
    the only mirror in the repo.
    THE MAINTENANCE MODEL IS BEING EXERCISED AS THIS IS WRITTEN, which is the
    sharpest evidence available: uncommitted in the working tree on
    2026-08-16, `backstop.yml` and `backstop.lock` move
    `backstop-ai/go-toolchain` from 1.3.0 to 1.4.0 (`install_date:
    2026-08-16T13:25:09Z`, `git_ref: v1.4.0`), and the SAME uncommitted
    change hand-edits the fixture's `go-test` binding to add the
    `ExemptFromScopeFilter` rationale comment and hand-edits
    `cmd/backstop/testdata/exempt-matrix-bindings.yml`'s matrix table from
    `go-test -> false` to `go-test -> true`. That is the second pack release
    reconciled into the fixture by hand and by memory. It worked; nothing
    made it work, and nothing would have complained if it hadn't.
    The second, lower-severity half:
    `cmd/backstop/testdata/exempt-matrix-bindings.yml` is a THIRD,
    documentary-only copy of a subset of the same bindings, introduced
    alongside SPEC-041/ISSUE-129. Its dead-code status is confirmed, with one
    nuance the issue's "zero referrers" phrasing misses: a repo-wide grep for
    `exempt-matrix` returns exactly one `.go` hit, and it is a COMMENT —
    `cmd/backstop/exempt_test_helpers_test.go:13`, "exemptBinding builds an
    EngineBinding for the exempt-matrix tests" — naming the concept, not the
    file. Every other hit is plan prose (`PLAN-SPEC-041` file scope,
    `PLAN-ISSUE-129` in four places). No code path opens the file and no test
    asserts it matches the in-memory bindings its own header calls the
    source of truth, so its drift is unfalsifiable by construction — yet it
    still costs hand-maintenance, as today's uncommitted edit to it proves.
    Direction per the issue, not decided here: delete it, or make it
    load-bearing with a test that parses it and asserts equality against
    `exempt_test_helpers_test.go`'s bindings. Leaving it unread and
    unverified is the worse option either way.
    Sequencing, stated as a constraint: this goes AFTER `PLAN-ISSUE-129`
    lands. That plan is mid-flight in the working tree right now and edits
    both files this item targets; starting here first collides on them and
    re-derives values that are actively changing.
13. **`pkg/packval`'s fixture executor never applies a binding's declared
    `convert:` script, so any pack with a non-SARIF engine fails `pack
    test`/`pack check` with a parse error instead of a rule verdict
    (ISSUE-141).** `DefaultExecutor.RunEngine` (`pkg/packval/executor.go:
    62-97`) — the command dispatch behind `backstop pack test`/`pack check`
    phase 3 — pipes `stdout.Bytes()` straight into
    `check.ParsePackFindings` at `executor.go:91`, unconditionally.
    `binding.Convert` appears NOWHERE in that file (grep clean). The real
    gate dispatch path does the opposite: `runFindingsEngine`
    (`cmd/backstop/pack_gate.go`, ~lines 760-774) resolves
    `filepath.Join(packRoot, filepath.FromSlash(binding.Convert))` and runs
    it through `resolveSandboxedRunStdout()` BEFORE parsing. Live instance:
    `packs/substantiveness/pack.yml:25` declares `convert:
    ast-grep/to-sarif.sh` on its `ast-grep scan --json` engine precisely
    because ast-grep's native `--json` output is a plain JSON ARRAY of
    match objects, not a SARIF document with a `runs` key — the converter
    script's own header comment says so ("Real ast-grep stdin->SARIF
    converter shipped by the pack (DD-7 / REQ-008, ISSUE-062)"). `parseSarif`
    (`pkg/check/parsers.go`) `json.Unmarshal`s into a `sarifLog` STRUCT,
    which fails on an array.
    STATE THE FAILURE DIRECTION EXPLICITLY — it is the whole reason this
    item is here and not in DIR-032, and a planner must not misread it as a
    vacuous green: the failure is LOUD and in the SAFE direction. `RunEngine`
    returns `ExecutionResult{Passed: false, ExitCode: 1}` together with a
    non-nil error, `engine %q produced no parseable SARIF`
    (`executor.go:92-95`), and `RunFixtures` (`pkg/packval/phase3.go:62-90`)
    turns that non-nil error into a `ValidationError` on BOTH the positive
    (`semgrep-positive`, "positive fixture failed") and negative
    (`semgrep-negative`, "negative fixture run failed") branches. `pack
    test` goes RED. PLAN-ISSUE-092's own finding F7-b puts it best and is
    worth quoting as the framing: "Correct behavior for a genuinely broken
    pipeline; wrong conclusion about the pack." The defect is that a
    Convert-declaring pack can never obtain a genuine pass/fail signal on
    its own rules — a capability gap in the pack-authoring validator — not
    that anything reports a verdict it did not earn.
    Direction, kept as constraint not design, taken from the issue: apply
    the declared Convert script in `RunEngine` (or its phase3 caller) the
    way `runFindingsEngine` does, before `check.ParsePackFindings`. Three
    things whoever plans it must settle rather than assume: (a) check
    `.backstop/packs/backstop-ai/backstop-core-architecture/architecture/
    backstop-core.yml` before assuming the conversion logic can be shared
    verbatim between `cmd/backstop` and `pkg/packval` — those may sit on
    opposite sides of an import-direction constraint, exactly as
    PLAN-ISSUE-118 hit between `pkg/gate` and `pkg/pack/engine`, and as
    PLAN-ISSUE-140 is currently navigating for its own shared predicate (it
    landed the shared never-started predicate in a NEW package,
    `pkg/check/never_started.go`, rather than editing the architecture
    policy file); (b) if a shared helper is not architecturally permitted
    and packval needs its own copy, the two copies must not drift in
    behavior — drift between the gate-side dispatch and its packval seed is
    the ROOT CAUSE of this item's own sibling ISSUE-140 (DIR-032 item 15),
    where the gate predicate was widened and the seed it was copied from
    never was; (c) confirm `resolveSandboxedRunStdout()`'s sandboxing
    guarantee is reusable from `pkg/packval`, or that an equivalent
    guarantee is preserved if reimplemented — note that `pkg/packval/
    sandbox.go` is item 1's (ISSUE-020) surface and is a hard no-op on
    Linux, so a Linux host cannot run a pack convert script at all today; a
    fix here inherits that constraint rather than escaping it.
    DEPENDENCY THE ROSTER MUST CARRY, and it is the sharpest single fact
    about this item: **ISSUE-141 is a declared HARD PREREQUISITE for
    PLAN-ISSUE-092's phase 5.** That plan
    (`plans/PLAN-ISSUE-092-pack-test-phase3-fixtures-cannot-fail.plan.yml`,
    `status: draft`, created 2026-08-16) fixes DIR-032's item 4 —
    phase3 fixture dispatch being dead code for every `rule_path:`-declared
    rule — and its final verification phase depends on a real in-repo pack
    genuinely passing `pack test`. The pack it targets,
    `packs/substantiveness`, cannot pass regardless of ISSUE-092's own fix
    landing correctly, because of this item. The plan says so itself, in
    its FOLLOW-ONS block: follow-on "(ii) packval's executor never applies
    `binding.Convert`… ⚠ PREREQUISITE", and again in F7-b: "The fix belongs
    in `pkg/packval/executor.go`, which this lane does not own." So a
    directive-crossing sequencing edge exists — this item (DIR-024,
    BACKLOG.yml position 5) gates a member of DIR-032 (position 2).
    backlog-pm has NO reorder authority and proposes nothing here; the
    inversion is recorded as a PROPOSAL in `.backstop/pm/INBOX.md` for
    Brandon.
    TWO SIBLINGS FROM THE SAME LANE, named so nobody folds them in:
    PLAN-ISSUE-092's F7 identified THREE independent reasons a real pack
    still cannot pass phase 3, and required all three filed separately
    because they share a root cause (packval's manifest/dispatch model
    predates the engine model) but have three different owning surfaces and
    three different fixes. This item is (ii). The other two: (i)
    `ISSUE-142` — packval's `Rule` struct has no `Pattern` field at all,
    where the runtime `pkg/pack` `Rule` has carried one since SPEC-035
    REQ-004, so every `pattern-arg`-declared rule (all of
    `packs/contracts`) dispatches zero fixtures; filed 2026-08-16 and
    triaged separately (do NOT assert its home in this file — a concurrent
    triage owns that call). (iii) the `semgrep-rule-id` cross-check in
    `RunFixtures` is baked-semgrep and unconditional — it runs
    `semgrepFileContainsRuleID` on a rule's source file regardless of the
    rule's declared engine, so `packs/substantiveness`'s `rule_path:
    ast-grep/sgconfig.yml` (an ast-grep project config whose whole content
    is `ruleDirs: [rules]`) fails it — a live thin-executor violation AND
    bundle-mandated by BUNDLE-005 REQ-012, so correcting it is a
    requirements question, not a local code edit. As of this writing (iii)
    has no issue ID in the tree; if it is still unfiled when someone works
    this item, that is a gap to raise, not to absorb.
14. **Once PLAN-ISSUE-141 lands, the Convert-application step exists as TWO
    independently-maintained implementations instead of one (ISSUE-143).**
    Applying a binding's declared `Convert` script to raw engine output
    before handing it to `check.ParsePackFindings` will then be done twice:
    once in `cmd/backstop/pack_gate.go`'s `runFindingsEngine` (the real gate
    dispatch path, `backstop gate`) and once in `pkg/packval/executor.go`'s
    `RunEngine` (the `pack test`/`pack check` phase3 path). Two
    hand-maintained copies of "apply Convert, then parse" drift the moment
    either call site changes — a new edge case handled in one copy (error
    wrapping, empty-output handling, sandboxing behavior) and not the other
    reproduces exactly the bug class ISSUE-141 was filed to fix, just for a
    different case.
    Why this is NOT a duplicate of item 13: item 13 is the missing stage
    (Convert never runs at all in packval); this item is the two-
    implementation SHAPE that item 13's fix leaves behind. PLAN-ISSUE-141
    deliberately declines consolidation — doing so would require editing
    `pack_gate.go`, which was the active file scope of two other in-flight
    implementers (PLAN-ISSUE-067, PLAN-ISSUE-140) the same night — and
    instead installs a mechanical content-scan drift guard (its CLM-006) as
    an INTERIM TRIPWIRE, not a fix. This item is what retires that tripwire.
    Direction, kept as constraint not design: extract a single shared
    Convert-application step (name TBD by the planner) into `pkg/packval`
    and have `cmd/backstop/pack_gate.go`'s `runFindingsEngine` delegate to
    it — that direction and not the reverse, because `cmd/backstop` already
    has a declared dependency edge onto `pkg/packval` per
    `.backstop/packs/backstop-ai/backstop-core-architecture/architecture/
    backstop-core.yml` and already imports it for
    `packval.SandboxedRunStdout`, so no new architectural edge is needed,
    while `pkg/packval` cannot depend back on a `main` package. Three things
    the planner must confirm before scoping, from the issue: (a) the CURRENT
    shape of both call sites, since PLAN-ISSUE-141 may have landed
    differently than planned and ISSUE-140/ISSUE-142 touch the same file;
    (b) whether CLM-006's content-scan guard is still in place and exactly
    what it checks, since consolidation should retire it rather than leave
    dead weight; (c) whether the two implementations have ALREADY diverged
    in a way that produces wrong output on one path — if so that divergence
    is a live defect, not tech debt, and the item must be re-scoped.
    RELATED SCOPE A PLANNER MUST NOT MISS: ISSUE-144 (filed the same night,
    homed in DIR-032) adds a SECOND stage with the same dual-implementation
    problem — `StdoutArtifact` payload selection, which the gate path
    performs and packval's executor does not. If ISSUE-143's extraction
    lands first, ISSUE-144's payload-selection logic belongs in the same
    shared location rather than becoming a third independent copy; if
    ISSUE-144 lands first, this item's extraction must absorb it. Stated as
    a sequencing coupling, not a merge — they are separate issues with
    separate fixes.
    SEQUENCING: `pkg/packval/executor.go` and `cmd/backstop/pack_gate.go`
    are both live, actively-edited files (PLAN-ISSUE-140, PLAN-ISSUE-067,
    PLAN-ISSUE-092's lane). This item is consolidation work that touches
    both and must be sequenced AFTER the in-flight lanes land, never
    concurrently — opening it early guarantees a collision. It carries no
    verdict defect of its own today.
15. **go-test converter emits a bare basename instead of a repo-relative
    path for nested-package test failures — a finding-data precision
    defect, not a lying verdict (ISSUE-135).** The `backstop-ai/go-toolchain`
    pack's `go-test` SARIF converter (`scripts/test-to-sarif.sh`) copies
    `go test`'s own `--- FAIL:` line verbatim into
    `locations[].physicalLocation.artifactLocation.uri` — which Go's
    `testing` package always basenames via `filepath.Base()` regardless of
    package depth (Go's own behavior, not something backstop's tooling adds
    or could suppress). Reproduced directly against the real converter for a
    two-level-nested package: `uri` reads `a_test.go`, not
    `sub/pkg/a_test.go`. The converter has what it needs to fix this — the
    per-package `FAIL\t<import-path>\t<time>` summary line closing each
    `--- FAIL:` block, resolvable to a repo-relative directory via
    `go.mod`'s module path or `go list -f '{{.Dir}}'` — but currently
    discards it, looking only at the block's own `.go:NN:` line.
    Why this is DIR-024 (finding-data precision) and not DIR-032 (gate
    verdict honesty), stated explicitly because the issue's own References
    section names DIR-032 and a planner would otherwise trip on the
    mismatch: ISSUE-129 already landed `exempt_from_scope_filter: true` on
    the `go-test` binding, so every go-test violation survives
    `filterViolations`'s scope check via the `ProjectWide` OR-branch
    regardless of `File` — the scope-suppression risk this issue's own
    Impact section originally raised is neutralized in the current tree.
    What remains is a violation that IS reported and DOES correctly redden
    the gate, just with an imprecise/ambiguous location string — wrong data
    on a correct verdict, this directive's charter (same test this file has
    drawn for ISSUE-096, ISSUE-115, ISSUE-125, ISSUE-137, ISSUE-141,
    ISSUE-143), not DIR-032's "wrong verdict about a computed result."
    Counter-argument considered and overridden, recorded so a future reader
    isn't left wondering if it was missed: baseline identity includes
    `File` (`pkg/gate` baseline-compare machinery), so two same-named test
    files in different packages (e.g. two `a_test.go`s) could genuinely
    conflate under baseline compare — a real verdict-honesty-adjacent risk,
    since a stale baseline entry for one package's failure could then mask
    a new failure in a different package with the same test filename.
    Founder-ruled (Brandon, 2026-08-16) this still homes in DIR-024: the
    baseline-conflation scenario is a downstream CONSEQUENCE of the
    imprecise `File` value, not an independent defect in a gate step's own
    verdict computation — fixing the converter (this item's own direction)
    removes the conflation risk at its source rather than needing a
    separate DIR-032 fix layered on top.
    Fix lives in the pack repo (`backstop-ai/go-toolchain`, local working
    copy `/Users/bmanson/src/projects/backstop-go-toolchain-pack`), not
    backstop-core — version bump + relock, same shape as ISSUE-129's fix
    and this directive's own ISSUE-096/ISSUE-125 precedents for pack-side
    rule/converter precision fixes.
16. **go-build engine discards `go build`'s stderr — a real compile failure
    surfaces as an opaque, content-free crash message (ISSUE-145).** `go
    build ./...` writes all compiler diagnostics to stderr, confirmed by
    direct repro (`1>out.txt 2>err.txt`: `out.txt` empty, `err.txt` carries
    the real error). The `backstop-ai/go-toolchain` pack's `go-build` engine
    binding (`command: go build`, `input_mode: none`, `convert:
    scripts/build-to-sarif.sh`, `crash_guard: true`) is dispatched through
    `runFindingsEngine` (`cmd/backstop/pack_gate.go:727`) via
    `runner.RunStdout` (`pkg/check/runner.go:52`), which captures stdout
    ONLY — deliberately, per REQ-009/CLM-028, so a rule-fed engine's stderr
    banner can't corrupt its SARIF stdout payload. That design is correct
    for rule-fed engines and wrong for `go-build`, whose diagnostic channel
    IS stderr. Consequence: on a real compile failure the captured stdout
    buffer is always empty, `build-to-sarif.sh` (which documents that it
    reads `go build`'s stdout) emits valid-but-empty SARIF,
    `ParsePackFindings` yields zero violations, and the crash-guard branch
    (`pack_gate.go:804`, `CrashGuard && runErr != nil && len(violations) ==
    0`) fires because it cannot distinguish a genuine crash-with-no-output
    from a failure whose explanation went to an unread stream. The gate
    reports only `pack backstop-ai/go-toolchain engine "go build" crashed:
    non-zero exit with no parseable findings: exit status 1` — no file,
    line, or compiler message.
    This is categorical, not transient: every real `go build` compile
    failure under the installed pack (v1.5.0) hits this path, since `go
    build`'s error channel is always stderr. First observed as a
    pre-existing, inherited gate failure during `PLAN-ISSUE-124`
    implementation (2026-08-16), confirmed not a genuine break by running
    `go build ./...` directly (exit 0 at the time).
    Why DIR-024 and not DIR-032 "Gate Verdict Honesty" — the pull is
    obvious and this file has now drawn the line five times (ISSUE-115,
    ISSUE-125, ISSUE-137, ISSUE-141, ISSUE-135): the gate still correctly
    reds on a real compile break; the crash-guard is doing exactly its
    designed job (SPEC-034 REQ-003/CLM-010) of refusing to read
    zero-findings/non-zero-exit as a silent pass. Nothing here computes or
    reports a wrong verdict — only the diagnostic content is missing. Same
    shape as item 15/ISSUE-135 (go-test's bare-basename `File`): wrong or
    missing DATA on a correct verdict, this directive's charter, not
    DIR-032's "wrong verdict about a computed result."
    Not a duplicate of ISSUE-067 (DIR-032 item 2, go-test's opaque crash):
    that issue's root cause is an extraction bug on data `go test` DOES
    write to stdout; here the diagnostic text is never even offered to the
    converter, because it was written to a channel (`stderr`) `RunStdout`
    discards by design. Checked for siblings: of the three `crash_guard:
    true` bindings installed in this repo, only `go-build` has this defect
    — `go-arch-lint` writes native JSON to stdout by its own
    `--output-type json` contract and doesn't share it.
    Direction, kept as constraint not design, taken from the issue's own
    proposal: use the `producer:` mechanism `pack_gate.go:680-725` already
    supports (an un-sandboxed script that runs in place of the plain
    command and can merge stdout+stderr itself, today used only by the
    coverage engine's `coverage-produce.sh`) to merge `go build`'s two
    streams before `build-to-sarif.sh` sees them — or have
    `build-to-sarif.sh` invoke `go build ./... 2>&1` as its own subprocess,
    a bigger shape change to the `input_mode: none` contract worth weighing
    against the producer option. Whichever shape is chosen, the crash-guard
    message should carry the captured diagnostic text if the crash-guard
    branch remains reachable at all after the fix.
    Fix lives in the pack repo (`backstop-ai/go-toolchain`), not
    backstop-core — version bump + relock, same shape as this directive's
    ISSUE-129/ISSUE-135 precedents; `pack_gate.go`'s producer mechanism the
    fix would use is already shipped core-side and needs no core change to
    adopt.
17. **A relative `packDir` silently breaks every macOS sandboxed convert
    step — `sandbox-exec` rejects a relative `subpath` clause and the real
    diagnostic is discarded, so the caller sees only an opaque `exit status
    71` (ISSUE-147).** `darwinSandboxProfile`
    (`pkg/packval/sandbox_nonlinux.go:72-92`) calls
    `filepath.EvalSymlinks(packDir)` on whatever `packDir` string it is
    handed, with no `filepath.Abs` call first. `EvalSymlinks` preserves
    relativity — given a relative path it returns a relative path back — so
    a relative `packDir` (e.g. `backstop pack test packs/substantiveness`,
    run from the repo root exactly as `pack test`/`pack check` are normally
    invoked) embeds a relative `(subpath "packs/substantiveness")` clause
    into the generated `sandbox-exec` profile. `sandbox-exec` does not
    treat a relative subpath as cwd-relative; it refuses to apply the
    profile at all and exits **71** (`EX_OSERR`) — a sandbox bootstrap
    failure, not a convert-script error, though nothing in the current path
    says so. Reproduced back-to-back and stable by the discovering
    implementer: the identical pack, same bytes, same binary, fails on a
    relative `packDir` and succeeds on an absolute one; `diff -rq` of the
    two pack trees is clean, ruling out a content difference.
    Same file, same profile-construction function ISSUE-029 (closed) fixed
    for a different reason (dyld read denials from an under-scoped
    allowlist) — that fix established `resolved` must be the
    kernel-resolved path for a `sandbox-exec` subpath rule to match at all;
    it did not also establish `resolved` must be absolute, which is the gap
    this issue closes. Darwin-only, confirmed against
    `pkg/packval/sandbox_linux.go`: the Linux path establishes its Landlock
    rules via file descriptors, not a string-matched profile clause, so
    there is no equivalent failure mode there — scoped entirely to
    `pkg/packval/sandbox_nonlinux.go`, the same file as item 1 (ISSUE-020).
    Compounding defect, same file: `platformSandboxedRunStdout`
    (`sandbox_nonlinux.go:133-149`) sets `c.Stdout` to a buffer but never
    sets `c.Stderr`, so `sandbox-exec`'s own diagnostic on this failure is
    discarded before it reaches the operator, leaving only the bare
    `exit status 71`. Direction, kept as constraint not design, from the
    issue: (a) call `filepath.Abs(packDir)` before (or as part of) the
    `EvalSymlinks` call in `darwinSandboxProfile` so the embedded subpath is
    always absolute regardless of caller input; (b) capture `sandbox-exec`'s
    stderr on failure in `platformSandboxedRunStdout` (and audit
    `platformSandboxedRun`'s `CombinedOutput` path for the same gap, though
    it already interleaves stderr and is expected unaffected) and fold it
    into the returned error.
    Why DIR-024 and not DIR-032 "Gate Verdict Honesty" — considered and
    decided, not skipped: `pack test`/`pack check` already reports SOME
    failure here — the exit-71 is loud and blocking, not a silent pass —
    the defect is that the reported diagnostic is wrong/opaque rather than
    naming the actual sandbox-bootstrap cause. Same "loud red, needs a
    legible name" shape as item 16/ISSUE-145 (go-build's discarded stderr)
    and item 15/ISSUE-135 (go-test's bare-basename `File`), both already
    homed here on that exact reasoning, not DIR-032's "computes a result but
    reports the wrong verdict about it."
18. **The zero-match E2E harness's deliberate rule patch makes the copied
    pack unvalidatable, so four substantiveness E2E tests never reach the
    code under test (ISSUE-158).** `(*e2eWorkspace).
    installZeroMatchSubstantivenessPack`
    (`cmd/backstop/gate_substantiveness_e2e.go:297`) copies
    `packs/substantiveness` to a temp dir and APPENDS `files:\n  -
    "harness/fixtures/**/*.go"` to the copy's
    `ast-grep/rules/referenced-symbol-go.yml`. ISSUE-113's intent for that
    patch is legitimate and must be preserved: make the Q2
    `referenced-symbol` rule match ZERO test files in the consumer's
    workspace, producing a healthy-looking pack that yields no Q2 evidence.
    The same glob ALSO takes the pack's OWN fixture tree
    (`testdata/fixtures/rules/**`) out of the rule's scope, so the pack's
    declared negative fixture for `referenced-symbol-go` can no longer
    trigger, and `pack add` — which runs the full packval pipeline
    unconditionally on a scratch copy — REFUSES the patched copy at
    `phase3-fixtures`.
    MEASURED, both numbers (by implementer-issue148, 2026-08-17, repo HEAD
    `c586af3`, real ast-grep 0.43.0 — not re-measured today by backlog-pm):
    applying the harness's exact patch to a scratch copy and running
    `./bin/backstop pack test <abs path>` gives exit 1 / `phase3-fixtures`
    fail with THREE errors on the pre-ISSUE-148 pack (one `semgrep-positive`
    false positive, two `semgrep-negative` not-triggered) and ONE error on
    the ISSUE-148-corrected pack (`semgrep-negative` not triggered). So the
    ISSUE-148 lane takes this copy 3 -> 1 and does NOT clear it; the
    residual is this harness's rule patch colliding with packval, not a
    polarity problem.
    BLAST RADIUS IS FOUR TESTS, NOT THREE — record this correction
    explicitly, because PLAN-ISSUE-148's own notes predicted only three. All
    four fail identically at `pack add` -> `pack test ... failed in
    phase3-fixtures: 1 validation error(s)`, none reaching the code under
    test: `TestE2E_ZeroMatchClassification_RefusesInsteadOfPerTestViolations`
    and `TestE2E_ZeroMatchClassification_RefusalIsNotWaivable`
    (`cmd/backstop/gate_substantiveness_zero_match_e2e_test.go:86,155`),
    plus `TestE2E_HollowEvidenceBlocksZeroMatchRefusal` and
    `TestE2E_HollowEvidenceBlocksRefusal_IsNotVacuous`
    (`cmd/backstop/gate_substantiveness_refusal_boundary_e2e_test.go:52,112`)
    — the last two share the harness and the failure mode and were outside
    the original prediction.
    `TestE2E_ZeroMatchClassification_ControlPackReportsNoViolations` is
    unaffected: it installs the pristine pack via
    `installSubstantivenessLocalPack`.
    SECOND, SMALLER FINDING THE FIX MUST ALSO CORRECT: the comment block
    above `installZeroMatchSubstantivenessPack` headed "WHY THE PATCHED PACK
    STILL PASSES `pack test`" is now stale and asserts the exact opposite of
    the truth. It claims packval never runs these fixtures because
    packval's `Rule` struct reads the YAML key `file` while
    `packs/substantiveness/pack.yml` declares `rule_path:`, and cites
    ISSUE-092 as tracking that hole. ISSUE-092 is CLOSED and its fix is
    committed at HEAD: `pkg/packval/phase3.go:34` now resolves
    `rule.RuleSourcePath()` (`pkg/packval/manifest.go:160`), so phase3 does
    run the fixtures and does notice the patch — which is precisely why
    this defect became visible. (backlog-pm verified both the patch site
    and the committed `RuleSourcePath` resolution in the tree, 2026-08-17.)
    THE DESIGN JUDGMENT THIS ITEM MUST NOT PRE-DECIDE, stated as constraint
    not solution: the fix needs a glob (or another mechanism) that preserves
    ISSUE-113's meaning — "this rule matches nothing in the CONSUMER's
    workspace" — while leaving the PACK's own fixture tree in scope so
    packval can still validate the copy. The choice belongs to this issue's
    own plan lane. Two things are ruled OUT up front by the issue and must
    be recorded as such: do not weaken or delete a fixture, and do not
    `t.Skip` any of the four tests — `requireAstGrepE2E` is a `t.Fatalf` by
    design and a skipped real-engine E2E is silent vacuous green.
    ADJACENCY WITHOUT CONFLATION: note ISSUE-151 (DIR-032 item 20,
    path-scoped pack rules dark under file dispatch) as adjacent but
    explicitly NOT the same defect — ISSUE-151 is the LIVE CONSUMER GATE's
    dispatch shape (directory vs explicit-file) against a real repo; item 18
    is a TEST HARNESS's deliberate rule patch colliding with packval's
    unconditional validate-on-`pack-add` against the PACK'S OWN fixture
    tree. Different mechanism, different surface; neither fix routes through
    the other.
    WHY DIR-024 AND NOT DIR-032 — decided, not skipped, and written in the
    same shape item 17 uses: packval reports a CORRECT verdict here. The
    patched copy genuinely is invalid, `pack add` refuses LOUDLY and
    blockingly, and the four tests go RED — nothing reports a clean pass on
    a scan that did not happen. DIR-032's charter sentence requires a step
    that "computes a result internally but reports the wrong verdict about
    it," and this fails that test the same way items 15 (ISSUE-135), 16
    (ISSUE-145) and 17 (ISSUE-147) do: loud red, wrong or missing legible
    name, homed here. Affirmative precedent as well as elimination: item 12
    already homes test-fixture-hygiene-with-no-guard here, and items 13/14/17
    already home `pkg/packval` machinery and its harnesses here. The
    counter-pull is real and should be named honestly: the harness exists to
    test DIR-032 item 10 (ISSUE-113), and it was surfaced by DIR-032 item
    19's lane (ISSUE-148) — lane adjacency, not charter.
    CROSS-DIRECTIVE COUPLING, state it once so neither directive reads as
    contradicting the other: while item 18 is unfixed, DIR-032 item 10's
    delivered zero-match refusal behavior has NO working E2E coverage — it
    is failing loudly rather than passing vacuously, so this is a coverage
    outage, not a false green. Whoever fixes item 18 must keep ISSUE-113's
    zero-match meaning intact and should sequence after or independently of
    PLAN-ISSUE-148, whose file scope explicitly fences this out.
19. **The allowlist overclaim item 3 (ISSUE-082) fixed survives in three
    sibling files item 3's plan deliberately left out of scope (ISSUE-131).**
    `PLAN-ISSUE-082` (`status: completed`) corrected
    `engine.TrustedToolAllowlist()`'s doc comment
    (`pkg/pack/engine/allowlist.go`) — which had falsely claimed the
    allowlist governs "any pack-declared command" — to state the real,
    narrower guarantee: the allowlist is consulted **only** for engine
    bindings carrying a non-nil `Provision` block; a pack-declared command
    that invokes a tool directly, with no `Provision`, was never covered.
    That is intended design, not a gap.
    The identical false claim survives, unfixed, in three files the plan
    named by file and line as explicitly OUT of its declared scope (plan
    notes lines 69 and 72): `cmd/backstop/pack_gate.go:42-44`
    (`resolveTrustedToolAllowlist`'s doc comment) and
    `cmd/backstop/recipe_apply.go:53-56` (the user-facing `backstop recipe
    apply --help` `Long` text) — the plan's own words call the
    `recipe_apply.go` instance "the sharpest — the only surviving instance a
    user sees." ISSUE-131 names a third file beyond those two. Confirmed
    NOT part of this defect: `checkEngineToolAllowed`
    (`cmd/backstop/pack_gate.go:796-812`) already correctly states the
    `Provision` precondition elsewhere in that same file.
    This is a clear fit, not a founder roster call: it is not a new theme,
    it is item 3's own deliberately-deferred tail, and this directive
    already owns the allowlist surface via item 3.
20. **`cmd/backstop`'s test-binary `TestMain` never checks whether it was
    re-exec'd as a sandbox helper, so every `cmd/backstop` test that
    triggers real sandboxed dispatch dies on Linux before the dispatched
    command runs (ISSUE-163).**
    Mechanism A: `pkg/packval/sandbox_linux.go`'s `newSandboxHelperCommand`
    implements sandboxed dispatch by RE-EXECUTING the currently running
    binary as a helper subprocess — `os.Executable()`, `exec.Command(self)`
    with `Dir` set to the pack directory being validated, and
    `BACKSTOP_SANDBOX_HELPER_SPEC` in the environment to tell that copy it
    is the helper. Under `go test` that "self" is the TEST binary.
    Mechanism B: `cmd/backstop/integration_test.go`'s `TestMain`
    unconditionally runs `execCommand("go","build","-o",binaryPath,".")`
    with `cmd.Dir = "."` as its very first action, with no prior gate of
    any kind. VERIFIED IN TREE by backlog-pm 2026-08-17: the function has
    no `MaybeRunSandboxHelper()` call anywhere, and a repo-wide grep for
    the literal `"failed to build binary"` returns exactly one source, that
    same `TestMain`.
    The collision: for a `cmd/backstop`-resident test that drives real
    sandboxed dispatch (concretely
    `TestSubstantivenessFixtures_RealPackTestPassesPhase3` in
    `cmd/backstop/substantiveness_fixture_polarity_test.go`, which runs
    `packval.NewPipeline(absPackDir, ...).Run()` against
    `packs/substantiveness`), the re-exec'd helper IS that same
    `cmd/backstop` test binary, so it re-enters `TestMain`, tries `go
    build` in the cwd the trampoline set to the PACK directory (no `.go`
    files there), dies with Go's `"no Go files in <dir>"` wrapped as
    `"failed to build binary: %v"` + `os.Exit(1)` — entirely BEFORE
    `runSandboxHelper`'s `BACKSTOP_SANDBOX_HELPER_SPEC` check. The pack's
    real convert script never executes; packval reports it up the stack as
    a generic engine-run failure.
    THE HOME REASONING, and it is the sharpest fact about this item: this
    is item 1's (ISSUE-020) own delivery residual, exactly the shape item
    19 (ISSUE-131) has relative to item 3. `PLAN-ISSUE-020`
    (`status: completed`) explicitly designed this as a WIRING PAIR and
    said so in prose — `pkg/packval/main_test.go`'s own `TestMain` header
    calls itself "the test-side half of a WIRING PAIR whose other half is
    `packval.MaybeRunSandboxHelper()` as the first statement of
    `cmd/backstop`'s `runWith`", and names both failure directions. What
    the plan's model never considered is a THIRD site in the same family:
    a test binary in a DIFFERENT package that also triggers sandboxed
    dispatch. `cmd/backstop` is that third site. Both halves the plan
    built are correctly in place today (verified: `pkg/packval/
    sandbox_linux.go:119` / `sandbox_nonlinux.go:47`, `pkg/packval/
    main_test.go`, and `cmd/backstop/main.go`'s `runWith(stdout, stderr,
    packval.MaybeRunSandboxHelper, NewRootCommand)`); the gap is a site the
    pair was never extended to.
    Blast radius: EVERY test resident in package `cmd/backstop` that
    triggers real sandboxed dispatch against a local pack on Linux, not
    just the one named repro. Per the issue's own investigation (relayed
    from the issue, NOT re-measured by backlog-pm): this accounts for the
    majority of the 62 violations in the `gate-report.json` artifact from
    the failed v0.2.0 release CI run on `ubuntu-latest`, across contracts,
    substantiveness and init dimensions, all surfacing as variants of
    `"pack add ... pack test for ... failed"`.
    Darwin: does NOT reproduce, and the reason is structural rather than
    luck — `sandbox_linux.go`/`sandbox_linux_helper.go` are
    `//go:build linux`-gated and never compile on macOS, and
    `pkg/packval/sandbox_nonlinux.go:47`'s `MaybeRunSandboxHelper()` is a
    bare `return nil` stub (verified in tree). Corollary a planner should
    have: adding the call to `cmd/backstop`'s `TestMain` is a behavioral
    no-op on darwin, so the fix cannot be validated on a Mac — its
    falsifier is Linux CI.
    How the issue's author established it, worth preserving as evidence
    quality: a throwaway debug PR added diagnostic printfs immediately
    before the final `unix.Exec` in `applyRestrictionsAndExec` and was run
    on real GitHub Actions. The printfs appeared for two sandboxed tests
    resident in `pkg/packval` (which has the correct `TestMain`) and NEVER
    for `TestSubstantivenessFixtures_RealPackTestPassesPhase3`, which
    showed the unchanged `"no Go files ... failed to build binary"` —
    proving the helper process dies before reaching any code that is only
    reachable after `runSandboxHelper` has taken over.
    Direction, kept as constraint not design (the issue states it
    explicitly as context-only): add the same first-statement
    `packval.MaybeRunSandboxHelper()` gate to
    `cmd/backstop/integration_test.go`'s `TestMain`, mirroring
    `pkg/packval/main_test.go`'s pattern (check the error, `os.Exit(126)`
    on failure), BEFORE the `go build` step. One thing a planner must
    settle rather than assume: whether `cmd/backstop` is the only
    remaining package with both a `TestMain` and sandbox-triggering tests,
    or whether the pair needs a general guard — the same "two
    hand-maintained copies drift" concern items 13/14 already record for
    the Convert step applies to a wiring pair that is now a wiring triple.
    WHY DIR-024 AND NOT DIR-032/DIR-033 — decided, in the same shape items
    15-18 use: nothing here reports a verdict it did not earn. CI goes RED,
    loudly and blockingly; the defect is that the red is misattributed — a
    harness gate that never runs, surfacing as an opaque build failure and
    then as a generic engine-run failure — so it fails DIR-032's charter
    test ("computes a result internally but reports the wrong verdict about
    it") exactly as items 15-18 do. It is also NOT DIR-033 ("Gate Verdict
    Honesty Residual Tail"): that directive homes follow-ons FILED BY
    DIR-032 member plans, and ISSUE-163 was not filed by one — it came out
    of the v0.2.0 release investigation, the same investigation that
    produced item 18's ISSUE-158, which is already homed here. Affirmative
    precedent, not just elimination: item 18 is a test-harness defect that
    stops `cmd/backstop` E2E tests from reaching the code under test and is
    homed here; this is the same class in the same package, one layer
    deeper.
    Sequencing/urgency, recorded as fact not as a priority claim
    (backlog-pm has no reorder authority): a plan scaffold
    `plans/PLAN-ISSUE-163-cmd-backstop-testmain-sandbox-helper-guard.plan.yml`
    already exists in `draft` with empty `phases:` as of 2026-08-17T22:14
    local — a lane is mid-authoring. Note also that this directive now
    carries TWO items whose issues ask to be the next lane opened (item
    17/ISSUE-147 and this one), and that a BACKLOG.yml `PROPOSAL`/
    sequencing question for Brandon is recorded in `.backstop/pm/
    INBOX.md` — this file makes no position claim.
21. **A `//go:build linux` AST wiring-guard test asserts a property its own
    production code cannot satisfy, so Linux CI reds on correctly-wired
    sandbox dispatch (ISSUE-165).**
    Mechanism, verified in tree:
    `TestSandboxLinux_ProductionPathUsesTheRealABIProbe`
    (`pkg/packval/sandbox_linux_errors_test.go:146`, `//go:build linux`)
    parses `sandbox_linux.go` with `go/parser` and walks every
    `*ast.CallExpr` whose `Fun` is one of three tracked identifiers —
    `linuxSandboxedRunWith`, `linuxSandboxedRunStdoutWith`,
    `newSandboxHelperCommand` — asserting at each that the LAST argument is
    literally the bare identifier `probeLandlockABI`. Four call sites: the
    two OUTER hops, `sandbox_linux.go:214` (`platformSandboxedRun` →
    `linuxSandboxedRunWith(..., probeLandlockABI)`) and `:240`
    (`platformSandboxedRunStdout` → `linuxSandboxedRunStdoutWith(...,
    probeLandlockABI)`), pass; the two INNER hops, `:221` and `:246` (both
    inner functions → `newSandboxHelperCommand(..., probeABI)`), forward the
    enclosing function's OWN PARAMETER, named `probeABI` (declared at `:220`
    and `:245`), and can therefore never match the literal. The guard's
    inner-hop assertion is UNSATISFIABLE BY CONSTRUCTION, not merely
    unsatisfied today.
    Correction to the issue's own text, stated so a planner is not
    surprised: ISSUE-165 quotes a single failure message (the `:221` site).
    The guard uses `t.Errorf`, not `t.Fatalf`, on that branch — so it emits
    TWO errors per run, `:221` and `:246`. Both must be addressed; a fix
    validated against one message alone is incomplete.
    Why the parameter exists and must not simply be deleted:
    `linuxSandboxedRunWith`'s own doc comment (`:216-218`) states it is
    "`platformSandboxedRun`'s body with the ABI prober injected, which is
    what makes the refusal wrap below reachable: on a healthy host the real
    probe always succeeds, so without this seam the wrap could never
    execute." `TestLinuxSandboxedRunWith_WrapsThePrepareFailure` drives it
    with a fake prober. The seam is deliberate and load-bearing for coverage
    of an otherwise-unreachable error path.
    THE SHARPEST THING A PLANNER NEEDS, and it is not in the issue file:
    ISSUE-165 offers as fix option (a) "rename the parameter to literally
    `probeLandlockABI` so it happens to satisfy the naive check." That
    option makes the test GREEN WITHOUT MAKING IT TRUE — the parameter would
    shadow the package-level function inside both bodies, and the inner-hop
    assertion would then pass for ANY caller-supplied prober including a
    fake, because it is asserting a declaration's spelling rather than a
    value's provenance. That is a vacuous green, and it retires the exact
    divergence the guard's own header says it exists to catch ("a seam that
    a test can fill is a seam PRODUCTION can be left holding the wrong thing
    in"). Per the founder's standing never-hack-the-gate-green law, option
    (a) must not be taken without an explicit founder ruling; option (b) —
    trace parameter provenance from the outermost production call, or
    restrict the tracked-identifier set to the two dispatch delegations and
    assert the inner hop by a different means — is the shape that preserves
    the guard's meaning. The plan must state its choice and say what the
    guard still falsifies afterward.
    Second structural hazard in the same test, found by backlog-pm and in
    neither the issue nor the plan corpus: the test closes with `if
    callSites != 4 { t.Fatalf(...) }` — an exact-count assertion whose own
    comment says it exists "so the guard cannot pass vacuously after a
    rename or a deletion." Any fix that changes which identifiers are
    tracked, or that adds/removes a delegation, must update that literal 4
    deliberately and in the same change; leaving it stale converts this from
    a false-positive red into a DIFFERENT false red.
    Why DIR-024 and not DIR-032/DIR-033 — apply this file's own
    repeatedly-drawn test, now for the eighth time (ISSUE-096, ISSUE-115,
    ISSUE-125, ISSUE-137, ISSUE-141, ISSUE-135, ISSUE-145, ISSUE-147,
    ISSUE-158, ISSUE-163): Linux CI goes loudly RED. Nothing reports a
    verdict it did not earn — DIR-032's charter sentence is "a gate step
    computes a result internally but reports the WRONG VERDICT about it,"
    and here the wrongness is a FALSE POSITIVE with a misleading legible
    name ("The sandbox would negotiate its ABI through something other than
    the kernel"), on a production path that is in fact correctly wired. Same
    "loud red, wrong or missing legible name" shape as items 15/16/17/18/20.
    DIR-032 is `done` in any case. DIR-033 "Gate Verdict Honesty Residual
    Tail" homes follow-ons FILED BY DIR-032 MEMBER PLANS; this came out of
    the v0.2.0 Linux-CI release investigation, the same lineage as item 18
    (ISSUE-158) and item 20 (ISSUE-163), both already here. Affirmative
    precedent, not mere elimination: this is item 1's (ISSUE-020, the Linux
    sandbox) own DELIVERY RESIDUAL — PLAN-ISSUE-020 is the lane that
    AUTHORED this guard test (its own text at line 353 and 1706 calls for
    "an AST WIRING GUARD") — the identical shape item 19 (ISSUE-131) has to
    item 3 and item 20 (ISSUE-163) has to item 1, a shape this directive's
    own text already calls "a clear fit, not a founder roster call."
    Exposure and blast radius, stated honestly: impact is CI-signal only.
    The production dispatch chain does pass the real `probeLandlockABI` at
    runtime — verified by reading `:214` and `:240` directly, not by
    trusting the test. Both files are `//go:build linux` and have never
    compiled on the darwin machine this suite is normally authored on, which
    is why this survived undetected; ISSUE-163's `TestMain` fix (`970512b`)
    only EXPOSED it by letting Linux CI reach the test. The cost of leaving
    it is that it trains whoever triages Linux CI to distrust or waive a
    sandbox security guard.
    **Stale-fact correction (backlog-pm, 2026-08-18, superseding this
    item's file/tag claims above, not rewriting them):** the guard
    described above as `TestSandboxLinux_ProductionPathUsesTheRealABIProbe`
    in `pkg/packval/sandbox_linux_errors_test.go` under `//go:build linux`,
    with Linux CI as its only falsifier, has since MOVED. ISSUE-165's own
    fix (commit `8d35706`, "move the ABI-probe wiring guard to an untagged
    file, rewrite it seam-aware") relocated it to
    `pkg/packval/sandbox_wiring_guard_test.go`, which carries NO build tag
    and no `_linux` filename component — deliberately, per a 12-line header
    comment there explaining that the guard parses `sandbox_linux.go` as
    text rather than executing Linux-only code, "so it can, and must, run
    on every platform." Verified in tree 2026-08-18. This item's mechanism
    description, its evidence, and its DIR-024-not-DIR-032/DIR-033 charter
    reasoning all still stand; only the file identity and the darwin-
    invisibility claim are superseded. Consequence recorded for item 24
    below: unlike this item, item 24's guard is untagged and darwin-
    visible, so its planner must not carry over this item's Linux-CI-only
    falsifier ceiling.
22. **`packs/contracts` cannot pass its own `pack add`/`pack test`
    phase3-fixtures self-validation on Linux CI, and its grep-based
    absence probes report zero matches for symbols that are demonstrably
    present (ISSUE-166).** Observed on CI run `32108003542` at commit
    `970512b` — item 20's (ISSUE-163) own `TestMain` fix, which is what
    first let the suite run far enough on GitHub's Linux runner for this
    cluster to surface. Evidence is the downloaded `gate-report.json`
    (`pack_engines` step), read directly by the issue's author;
    backlog-pm did NOT re-measure the CI run.
    TWO SYMPTOM CLUSTERS. (a) Roughly a dozen tests fail identically at
    `pack add <repo>/packs/contracts: ... pack test ... failed: pack
    validation (test) of the validation copy failed in phase3-fixtures:
    14 validation error(s)` — including
    `TestE2E_ContractsInstalledLocalPack_RealGate_MissingSignatureRed`,
    `TestE2E_ContractsUninstalled_NoVacuousGreen`,
    `TestE2E_ContractsRealAstGrepAndGrep_AndSandboxedConvert`,
    `TestNoVacuousGreen_MissingSignatureBlocks`,
    `TestDogfood_BackstopOwnContractSignatureTurnsGreen`,
    `TestInstallContractsLocalPack_InstallsWithSuppliedCommand`. (b) A
    more specific cluster shaped like a grep absence probe finding
    nothing: `TestContract_AbsencePresentSymbolGrepMatchViolation` ("a
    present forbidden symbol must produce a grep match (absence
    VIOLATION), got 0"),
    `TestContract_AbsenceUsesGrepTextPresenceNotAstGrep` ("got 0
    matches"), `TestEngine_GrepConvertScriptEmitsValidSarif`,
    `TestEquivalence_GoAbsencePresentAndAbsentMatchLegacy` ("pack verdict
    (false) != analyzer verdict (true) — equivalence broken"), and
    `TestContractsPack_PatternArgFixturesDispatchAndDiscriminate/the_fixtures_discriminate_through_the_real_engine`
    ("phase3 error: check=semgrep-negative rule=contract-absence
    claim=contract-absence-go message= negative fixture not
    triggered").
    THE MECHANISM IS NOT TRACED, and the issue says so emphatically — do
    not let a planner read a cause into it. Candidates the issue lists
    as explicitly NOT investigated and NOT ruled in: BSD-vs-GNU grep
    behavior, a cross-platform path-resolution difference, something in
    how the sandboxed convert step is invoked on Linux, or something
    else. Ruled OUT by direct inspection: this is not the `/dev/null`
    sandbox-write defect — `packs/contracts/ast-grep/to-sarif.sh` and
    `packs/contracts/grep/to-sarif.sh` contain no `/dev/null` redirects,
    and the `/dev/null` failures on the same CI run are the separate
    `ISSUE-168`.
    A CO-LOCATION FACT BACKLOG-PM VERIFIED IN TREE (2026-08-18), offered
    as an investigation lead and NOT as a cause: cluster (b)'s failing
    tests live in `pkg/pack/engine` (`contracts_grep_engine_test.go`,
    `contracts_go_rules_test.go`), `pkg/pack/distribution`
    (`contracts_local_install_test.go`) and `pkg/gate`
    (`contract_equivalence_test.go`, `contract_pack_paths_test.go`) —
    and NONE of those three packages declares a `func TestMain`.
    `pkg/pack/engine` and `pkg/pack/distribution` are precisely the two
    packages `ISSUE-164` ("Packval Importing Packages Missing TestMain
    Guard", `type: question`, open, filed by item 20's own lane at
    commit `8b3b3d8`) names as invisible to item 20's guard and to its
    regression pin
    `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`.
    ISSUE-164 states its central question is UNMEASURED — whether any
    test in those packages actually drives real sandboxed dispatch on
    Linux. This item is candidate evidence bearing on that question, and
    whoever investigates either one should read the other first. Two
    honest limits on the lead: `cmd/backstop` also appears in cluster
    (a) yet DOES carry the guard today (verified in tree at
    `cmd/backstop/integration_test.go:38`), so the guard gap cannot
    explain the whole cluster; and `pkg/gate`'s contract tests reference
    neither `packval` nor the sandbox directly (verified by grep), so
    `pkg/gate` is not on ISSUE-164's packval-importer inventory at all.
    WHY DIR-024 AND NOT DIR-032 — decided in the same shape items 15-20
    use, and worth stating because cluster (b) invites the opposite
    reading. As observed, every one of these failures is LOUD: `pack
    add` refuses and blocks, the tests go RED, nothing reports a clean
    verdict on a scan that did not happen. That fails DIR-032's charter
    test ("computes a result internally but reports the wrong verdict
    about it"). The affirmative precedent is nearly exact: item 18
    (ISSUE-158) is a pack failing its OWN `phase3-fixtures`
    self-validation and is homed here on this reasoning; item 20
    (ISSUE-163) came out of this same v0.2.0 release investigation and
    is homed here; and Linux/CI viability is item 1's (ISSUE-020)
    surface, this directive's own launch-blocker thread. Not DIR-022
    ("Contracts Engine Hardening", `queued`): that directive's charter
    is capability extension of the `contract_signature` compiler —
    relational-rule input mode and non-Go artifact contracts — and has
    no claim on a platform-behavior defect in a pack's self-validation.
    THE CONDITION UNDER WHICH THAT HOME SHOULD BE REVISITED, recorded
    now so it is not re-derived: if investigation traces cluster (b) to
    the PRODUCTION contracts dispatch path — i.e. a real Linux
    consumer's forbidden-symbol absence check reporting no violation for
    a symbol that IS present — then the underlying defect is a silent
    false-clean on the real gate, which is DIR-032's charter and not
    this directive's, even though today's symptom is a red test.
    backlog-pm has flagged this to Brandon as a conditional re-home, not
    a competing claim.
    Not a duplicate of item 13/ISSUE-141 (`pkg/packval`'s executor never
    applying a binding's `Convert`) nor of the now-closed `ISSUE-142`
    (packval's `Rule` struct lacking `Pattern`, which made every
    `pattern-arg` rule — all of `packs/contracts` — dispatch zero
    fixtures; closed 2026-08-17 via `PLAN-ISSUE-142`). ISSUE-142's fix
    is plausibly WHY these fixtures now dispatch far enough to fail
    rather than silently not running, which makes this a newly-visible
    defect rather than a regression — stated as observation, not
    traced.
23. **Both sandbox profiles deny ALL file writes with no `/dev/null` carve-out,
    so the universal `command -v foo >/dev/null 2>&1` idiom dies under
    Linux's Landlock with a confusing `exit status 127` (ISSUE-168).**
    Two `pkg/packval` tests fail on real Linux CI (run `32108003542`,
    `gate-report.json` at `git_sha: 970512b`, `pack_engines` step):
    `TestLinuxSandbox_RealInterpreterRunsUnderTheFilter`
    (`sandbox_linux_exec_test.go:437`) fails outright — its fixture
    `pkg/packval/testdata/sandbox/convert-jq.sh:14` is `if command -v jq
    >/dev/null 2>&1; then` and yields `exit status 127: cannot create
    /dev/null: Permission denied`; and
    `TestLinuxSandbox_NetworkAllowedControlLegSucceeds`
    (`sandbox_linux_exec_test.go:310`) shows the same `/dev/null:
    Permission denied` denials in its captured stderr.
    State this distinction explicitly, it matters for whoever fixes the
    other half: the `/dev/null` denials are NOT what makes
    `TestLinuxSandbox_NetworkAllowedControlLegSucceeds` fail. That test's
    `t.Errorf` fires because TCP and UDP were both blocked under a
    capability that PERMITS the network (`TCP_OPEN`/`UDP_OPEN` markers
    absent). Its network-blocking failure is a separate, still-unfiled
    defect; only the `/dev/null` denials belong to ISSUE-168.
    Root cause, symmetric by design: `pkg/packval/sandbox_capability.go`
    sets `WritablePaths: nil` unconditionally in
    `ConvertValidatorCapability` (line ~154), with the comment "EMPTY for
    convert/validator work: darwin denies file-write* outright and parity
    is the spec"; the darwin profile literal in
    `pkg/packval/sandbox_nonlinux.go:93` is `(version 1)(import
    "bsd.sb")(deny default)(allow process*)(allow
    file-read*%s)(deny network*)(deny file-write*)` — a blanket deny with
    no `/dev/null` exception either. Both verified by direct read. So this
    is NOT a Linux regression and NOT a designed platform asymmetry.
    Observed behavior IS asymmetric, and that was confirmed empirically
    rather than assumed: the issue author ran darwin's analogue
    `TestSandboxConvertWithRealInterpreter`
    (`pkg/packval/sandbox_realconvert_test.go:52`) — which drives the SAME
    `convert-jq.sh` fixture with the same idiom — locally on darwin, and
    it PASSES. macOS Seatbelt evidently lets writes to the `/dev/null`
    character device through as an emergent property, not because the
    profile text says so. Profile-literal gap symmetric; observed breakage
    Linux-only.
    Why it is worth closing rather than a security restriction working as
    intended: `command -v foo >/dev/null 2>&1` and `2>/dev/null`
    noise-suppression are ubiquitous in real shell scripts, so any
    third-party pack author's convert or validator script using the idiom
    breaks under backstop's Linux sandbox with a signature (`exit status
    127` / "Permission denied") that points nowhere near the sandbox. A
    write to `/dev/null` persists no state, escapes no data, and has no
    observable side effect — granting it narrowly weakens nothing.
    Fix direction (mechanism left to the plan, per the issue): a Landlock
    path rule scoped to `/dev/null` on Linux plus a corresponding `(allow
    file-write* (literal "/dev/null"))` clause in the darwin profile. Note
    for the planner: fix BOTH platforms even though only Linux is
    observably broken — item 1's charter requires deny-write PARITY with
    the macOS profile, and fixing only Linux would make the profiles'
    stated intent diverge from their text again.
    Why it homes here and not elsewhere: it is item 1's (ISSUE-020) own
    surface — the Linux sandbox delivered by `PLAN-ISSUE-020` (`status:
    completed`) — and its delivery residual, the same shape items 22
    (ISSUE-166), 21 (ISSUE-165) and 20 (ISSUE-163) have. NOT DIR-032/
    DIR-033: CI goes loudly RED and no unearned verdict is reported, so it
    fails DIR-032's charter test; and DIR-033 homes follow-ons filed by
    DIR-032 member plans, whereas this came from the same Linux-CI release
    investigation that produced ISSUE-163 (item 20) and ISSUE-158 (item
    18). Already cross-referenced from the neighboring item: item 22
    (ISSUE-166) explicitly ruled ISSUE-168 out as its own cause during its
    own investigation ("this is not the `/dev/null` sandbox-write defect
    ... the `/dev/null` failures on the same CI run are the separate
    `ISSUE-168`"), so that boundary is settled from both sides, not
    asserted unilaterally here.
    Distinct from item 20 (ISSUE-163), verified: ISSUE-163's own text
    records that these two `pkg/packval` tests DO reach the instrumented
    sandbox helper (they have a correctly-behaving `TestMain`) — so
    ISSUE-163's fix cannot resolve this, and this is not a duplicate.
    `PLAN-ISSUE-163` is `status: draft` and scoped to `cmd/backstop`'s
    `TestMain`; `PLAN-ISSUE-020` is completed and never granted any
    `/dev/null` write. No plan in `plans/` covers this.
    Priority note, stated as observation and explicitly NOT a reorder
    (directive-author has no reorder authority): DIR-024 sits at
    BACKLOG.yml position 5 and this slot does not change its rank.
24. **The rewritten prober-wiring guard is silently defeated by a `FuncLit`
    whose OWN parameter shadows the injected prober identifier — a
    deliberately-deferred blind spot the guard documents in its own source
    (ISSUE-170).**
    Mechanism, verified in tree: `proberWiringViolations`
    (`pkg/packval/sandbox_wiring_guard_test.go:81`) attributes every
    tracked call site to its enclosing `*ast.FuncDecl`, and the walk DOES
    descend into `ast.FuncLit` bodies — an advertised strength, stated in
    the comment at `:75-80`. But it resolves WHICH identifier counts as
    "the injected prober" from that same enclosing `*ast.FuncDecl`'s
    parameter list, never from the innermost enclosing scope:
    `proberWiringProberParamNames` (`:311`). A PRECISION THE ISSUE
    UNDERSTATES AND A PLANNER NEEDS: that resolution is BY TYPE, not by
    name — it collects every `*ast.Field` whose type identifier is
    `LandlockABIProbe`, returning every name those fields declare (so the
    grouped spelling `probeABI, decoy LandlockABIProbe` correctly yields
    two). The evasion still lands because a plausible closure parameter
    carries the SAME TYPE, so the outer decl's resolution yields
    `probeABI`, the call site's argument spells `probeABI`, and the names
    match — while the value actually in scope is the closure's own
    binding. Do not let a planner form the mental model "it matches by
    name" and reach for a name-based fix. The rebind scanner
    `proberWiringRebindsOf` (`:356`) cannot cover it: it matches
    `*ast.AssignStmt` and `*ast.ValueSpec` only, and a `FuncLit` parameter
    is neither — it is a new lexical binding, not an assignment. The
    evasion shape is a legitimate refactor, not a contrived attack: a
    retry/timeout wrapper closure that reuses the parameter name. Go warns
    about neither the shadowing nor the outer parameter (still "used" as
    the argument to the closure call).
    Provenance — deferred deliberately, filed as owed, not an oversight:
    this is the THIRD of three silent-green evasions found by the
    ISSUE-165 impl-review; the other two (a dispatch-seam re-bind gap and a
    `var` declaration-form re-bind gap) were CLOSED in commit `fc2b8ce`.
    The guard's own source documents the deferral in place at `:353-355`,
    directly above `proberWiringRebindsOf`: "STILL OUT OF REACH,
    DELIBERATELY: a FuncLit whose own PARAMETER shadows the prober
    identifier. That needs FuncLit-parameter-shadow detection and is
    tracked as a follow-on to ISSUE-165, not silently absorbed."
    `PLAN-ISSUE-165` TASK-005 clause (6) ("FILE ANYTHING ELSE SURFACED, and
    if nothing needed filing, SAY SO EXPLICITLY — silence is not a
    discharge") is the filing mandate. IN-FLIGHT COVERAGE IS THEREFORE NIL
    BY CONSTRUCTION, established from the corpus with ZERO interviews — no
    plan in `plans/` targets ISSUE-170; the only lane on this surface,
    `PLAN-ISSUE-165` (`status: draft`), fences the work out by name in its
    own source comment and in TASK-005.
    THE SHARP EDGE A PLANNER MUST NOT MISS (backlog-pm's addition, in
    neither the issue nor the plan): the guard closes with an exact-count
    assertion at `sandbox_wiring_guard_test.go:434` — `if dispatch != 2 ||
    forward != 2 || dispatch+forward != 4` — whose stated purpose is
    anti-vacuity (a call the walk stops recognising stops being COUNTED, so
    the counts catch selector-form and renamed callees the per-site rules
    cannot see). Any scope-aware fix that changes what is tracked, or that
    introduces a synthetic call site for the closure's own invocation, MUST
    update those literals deliberately in the same change. There is also a
    table-driven falsification harness in the same file asserting
    per-case `wantDispatch`/`wantForward` (`:1131`, `:1136`) — a new
    evasion case must be added there, not merely to the production-file
    walk. Leaving either stale converts a closed blind spot into a NEW
    false red — the exact failure item 21 exists to fix.
    Direction (the issue's own, recorded as the issue's, not decided
    here): for each tracked call site, walk outward through any enclosing
    `ast.FuncLit` boundaries and check whether one declares a parameter of
    type `LandlockABIProbe` before reaching the outer `FuncDecl`; if so,
    the binding in effect is the closure's, and the value passed AT THE
    CLOSURE'S CALL SITE becomes the thing to check. The issue explicitly
    leaves how much scope-resolution machinery is proportionate to the
    plan.
    Why DIR-024 and not DIR-032/DIR-033 — and say plainly that this one
    INVERTS this file's usual polarity, because a reader will notice: every
    recent DIR-024 ruling here (items 15/ISSUE-135, 16/145, 17/147, 18/158,
    20/163, 21/165) turned on "loud red with a wrong or missing legible
    name." Item 24 is the opposite polarity — a SILENT green. It still
    homes here, on a different and equally established line: DIR-032's
    charter requires a GATE STEP reporting a wrong verdict on a product
    surface; a false-green in backstop-core's own `go test` corpus is not
    that. That discriminator was drawn for ISSUE-137 ("product-surface
    false-green → DIR-032; repo test-harness false-green → DIR-024") and
    again for item 18 (ISSUE-158, where the harness EXISTED to test a
    DIR-032 member and still came here — lane adjacency is not charter).
    The AFFIRMATIVE precedent is this file's own item 4, ISSUE-075: a
    fixture that makes a test pass while proving nothing, deliberately left
    behind by the 2026-08-10 DIR-032 carve-out. DIR-032 is `done` in any
    case. DIR-033 "Gate Verdict Honesty Residual Tail" homes follow-ons
    FILED BY DIR-032 MEMBER PLANS; this was filed by `PLAN-ISSUE-165`, a
    DIR-024 item-21 lane, so provenance sends it here. And the
    delivery-residual sub-precedent applies directly: residuals home to
    their parent item's directive as clear fits, not founder roster calls
    — the shape item 19 (ISSUE-131) has to item 3 and item 20 (ISSUE-163)
    has to item 1.
    Exposure and blast radius, stated honestly: ZERO today, and the issue
    says so itself. The guard executes no production code — it parses
    `sandbox_linux.go` as text at test time — and no current code in
    `sandbox_linux.go` uses the evasion shape. The residual risk is narrow
    and forward-looking: a future refactor introducing a same-named
    closure parameter around either injectable seam would pass this guard
    silently while genuinely defeating the prober injection the guard
    exists to protect. Priority-wise this is the LOWEST-urgency member of
    the ISSUE-165 family — record that so a cold picker does not mistake
    "silent green" for "urgent."
    Correction to the parent's ceiling, worth one line here too: unlike
    item 21, whose only falsifier was Linux CI, item 24's guard is untagged
    and runs on darwin — a fix here is locally red-to-green verifiable.
    That is the durable half of ISSUE-165's fix paying off.
    Priority note, stated as observation and explicitly NOT a reorder
    (neither directive-author nor backlog-pm has reorder authority):
    DIR-024 sits at BACKLOG.yml position 5 and this slot does not change
    its rank. It rides on charter fit and displaces nothing — in
    particular it must NOT displace item 1 (ISSUE-020) or the ISSUE-092
    sequencing already recorded in this file.
25. **Stale doc comment argues for the refused option (a) at the ABI-probe
    seam (ISSUE-169).**
    `newSandboxHelperCommand`'s doc comment
    (`pkg/packval/sandbox_linux.go:163-175`) still closes with: "Test
    SandboxLinux_ProductionPathUsesTheRealABIProbe asserts both production
    call sites hand it the real probeLandlockABI, so the seam cannot become
    a place where test and production diverge" — verified byte-present in
    the current tree. Item 21's fix (`8d35706`, then `fc2b8ce`) left
    `sandbox_linux.go` byte-unchanged BY DESIGN — its CLM-006 is asserted
    mechanically by that lane's own TASK-003 — so this sentence was never
    in that lane's scope, and nothing has corrected it since.
    Why it matters: at the two INNER seams this comment governs
    (`newSandboxHelperCommand`'s only two call sites,
    `sandbox_linux.go:221` and `:246`, inside `linuxSandboxedRunWith` /
    `linuxSandboxedRunStdoutWith`) the rewritten guard now requires the
    ENCLOSING FUNCTION'S OWN injected `LandlockABIProbe`-typed parameter
    and flags the literal `probeLandlockABI` there as its own violation.
    The comment therefore states as settled fact the exact shape item 21
    rejected as a vacuous green ("option (a)"). It is precisely the
    sentence a future editor would cite to justify renaming `probeABI` to
    `probeLandlockABI` "so the code matches its own documentation" — a
    rename that shadows the package-level function inside both bodies and
    makes the guard pass for any caller-supplied prober, including a fake.
    CORRECTION TO THE ISSUE'S OWN NARRATIVE, found by backlog-pm and stated
    so a planner does not inherit it: ISSUE-169 frames the sentence as
    having been true until item 21's fix "inverted" it, and its
    Existence-in-world check calls the original ISSUE-020 comment
    "then-accurate." That is too generous. The sentence conflates two
    claims with DIFFERENT histories: (i) "the test asserts this" — was true
    at authoring (the old guard did demand the literal at all four sites)
    and is FALSE now (the rewrite reversed the requirement at these two
    sites); this half is what item 21's fix inverted. (ii) "both production
    call sites hand it the real probeLandlockABI" — was NEVER true. `:221`
    and `:246` have always forwarded `probeABI`, the injected parameter;
    that mismatch is exactly why the old guard went red on Linux. The
    comment asserted compliance the code never had. So the fix must correct
    TWO independent falsehoods, not restore a formerly-accurate sentence. A
    planner who reads only ISSUE-169 will under-scope this.
    An authoritative replacement already exists in the tree, worth naming
    so the fix is a pointer, not a second restatement:
    `pkg/packval/sandbox_wiring_guard_test.go` (lines 397-407, the header
    above `TestSandboxLinux_ProductionPathUsesTheRealABIProbe`) now spells
    out the correct rule at length, including why the inner hop is asserted
    against a typed parameter rather than a spelling and why option (a) is
    refused. The repo currently carries TWO comments describing the same
    seam that contradict each other; the test-file one is correct. Pointing
    `sandbox_linux.go`'s comment at it beats restating the rule in a third
    place that can drift again.
    Scope/risk, stated plainly: documentation only, no behaviour change, no
    gate dimension goes red on it today — the hazard is entirely in what a
    future editor does while trusting it. It is darwin-visible and
    Linux-CI-independent (unlike items 20-23), because the file is untagged
    production source, so this one CAN be verified locally. Because the fix
    is one sentence in production code and this repo's law is no
    implementation without a validated plan, the natural discharge is to
    fold it into whichever lane next legitimately edits
    `pkg/packval/sandbox_linux.go` rather than standing up a plan for a
    comment — but that is a founder sequencing call, recorded here as a
    recommendation, not a decision.
    Provenance and siblings: item 21 (ISSUE-165) is the parent — this is
    its delivery residual, mandated by `PLAN-ISSUE-165` TASK-005 step (5),
    which explicitly refuses to absorb it ("FILE ANYTHING ELSE SURFACED").
    Item 24 (ISSUE-170, the guard's `FuncLit` parameter-shadow blind spot)
    is the SIBLING from the same TASK-005 closeout batch — both were
    surfaced by the same review pass and filed together, but are
    independent fixes: item 24 is a silent-green gap in the guard itself,
    this item is a stale comment one function above it. Neither fix
    subsumes the other.
    Why DIR-024 and not DIR-032/DIR-033 — same test this file has drawn
    repeatedly: nothing here reports a wrong verdict today (no gate step is
    even involved), so it fails DIR-032's charter test outright; the risk
    is entirely forward-looking, in a comment a future editor might trust.
    Affirmative precedent: it is item 21's own delivery residual, the same
    shape item 24 (ISSUE-170), item 20 (ISSUE-163) and item 19 (ISSUE-131)
    already have to their respective parents.
    Priority note, stated as observation and explicitly NOT a reorder
    (backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml
    position 5 and this slot does not change its rank. It rides on charter
    fit and displaces nothing — in particular it must NOT displace item 1
    (ISSUE-020) or the ISSUE-092 sequencing already recorded in this file.

26. **A pack manifest's declared `convert:` path is never checked for
    existence at authoring time (ISSUE-175).**
    THE ANCHOR, verified: `pkg/pack/engine/testdata/contracts-grep-engine.yml`
    (manifest name `acme/contracts-grep-pack`) declares `convert:
    grep/to-sarif.sh` at line 26; `pkg/pack/engine/testdata/grep/` does not
    exist — the directory listing of `pkg/pack/engine/testdata/` is four
    `.yml` files and nothing else. The fixture is declaration-level only,
    read by `TestEngine_GrepPackDeclaredNotInDefaultRegistry`, never
    dispatched.
    ★ MATERIAL CORRECTION TO THE ISSUE'S SCOPE, and it is backlog-pm's, not
    the issue's: this is NOT a singleton. A sweep of every `convert:`
    declaration under `packs/`, `pkg/`, `cmd/` found the SAME dangling shape
    in EVERY declaration-only manifest fixture in the repo —
    `pkg/pack/testdata/pack-pattern-arg.yml` (`acme-query/to-sarif.sh`),
    `pkg/pack/testdata/pack-divergent-flags.yml`
    (`scripts/test-to-sarif.sh`, ×2), `pkg/pack/engine/testdata/
    contracts-astgrep-engine.yml` (`ast-grep/to-sarif.sh`),
    `pkg/pack/engine/testdata/engines-block-valid.yml` (`acme-grep/
    to-sarif.sh`, `scripts/test-to-sarif.sh`),
    `cmd/backstop/testdata/exempt-matrix-bindings.yml` (4 declarations),
    `cmd/backstop/testdata/coverage-routing-bindings.yml` (5 declarations) —
    roughly fifteen dangling references across seven files, none of them
    backed by a script on disk. Consequence for a planner: the issue's
    second candidate direction ("update this specific fixture to not
    declare a `convert:` it doesn't back") is a ~15-site change across seven
    files, not a one-liner; and its third candidate direction (confirm the
    testdata-exclusion conventions cover these) is LOAD-BEARING, not
    optional. The converts that live inside REAL pack directories all
    resolve correctly (`packs/contracts` both, `packs/substantiveness`,
    `pkg/gate/testdata/traceability-pack` both, `pkg/gate/testdata/
    ts-proof-pack`, the three toolchain testdata packs) — the defect class
    is confined to bare manifest fixtures that were never whole packs.
    ★ THE DESIGN CONSTRAINT THAT FALSIFIES THE NAIVE FIX, also backlog-pm's
    and the sharpest thing here: `packs/base-engines/pack.yml:48` — a REAL,
    shipped pack, compiled into every backstop binary via root `embed.go`'s
    `//go:embed all:packs/base-engines` and served as the default engine
    registry by `pkg/baseengines.Registry()` — declares `convert:
    ast-grep/to-sarif.sh`, and `packs/base-engines/` contains ONLY
    `pack.yml`. That is BY CONSTRUCTION, not a defect: base bindings are
    registry DEFAULTS merged into a CONSUMING pack's dispatch, and the
    convert is resolved against the consuming pack's root, not the
    declaring pack's — `cmd/backstop/pack_gate.go:271` computes `packRoot :=
    filepath.Join(packDir, manifest.NormalizedName)` for the manifest being
    dispatched, and `:845` joins the binding's `convert:` onto THAT root.
    `packs/contracts/ast-grep/to-sarif.sh` and
    `packs/substantiveness/ast-grep/to-sarif.sh` both exist at exactly that
    relative path, which is the convention the base binding is asserting.
    So a phase-1 check of the form "the declared `convert:` must exist
    relative to the pack root" would FALSE-RED the base engine pack that
    ships inside every backstop binary. Any implementation must either
    exempt bindings inherited from the base registry, or be reframed as a
    CONSUMER-side resolution check ("every pack that inherits a
    convert-bearing base binding ships the script at the declared relative
    path") — a materially different check from the one ISSUE-175 sketches.
    State this as a constraint a planner must resolve, not as a decided
    design.
    CORRECTION TO THE ISSUE'S IMPACT FRAMING, and it sets the polarity so
    this is not misfiled as a verdict-honesty item: dispatch-time behavior
    is ALREADY LOUD. `pkg/packval/executor.go:218` stats the resolved
    convert path and refuses; `cmd/backstop/pack_gate.go:845-847` does the
    same on the gate path, returning `broken pack %s: missing convert
    script %s` — a named refusal at the resolved path, not "whatever error
    the runner produces". So the gap is EARLIER-AND-MORE-LEGIBLE AUTHORING
    FEEDBACK, i.e. ergonomics, NOT a false verdict — nothing here reports a
    clean scan that did not happen. Read the issue's "same silent-hole
    shape" phrasing as an analogy to ISSUE-166, not as a claim of a silent
    false-clean.
    THE FIX SITE HAS A DIRECT IN-FILE PRECEDENT: `pkg/packval/phase1.go`
    already stats four other declared paths — the rule source at `:52`,
    declared fixtures at `:63`, `rule.Validator` at `:69`, and scaffold
    paths at `:76`. An engine binding's `convert:` is the one path-bearing
    declaration phase 1 does not check. `pkg/packval/phase2.go` carries no
    convert handling at all.
    SIBLING RESIDUAL A PLANNER SHOULD SEE, currently UNFILED:
    `PLAN-ISSUE-142` (`status: completed`) records residual R3 at its lines
    750-752 — "A phase-1 structural check for 'this rule declares no engine
    input at all'. New ecosystem-wide strictness, not a bug fix; would red
    any pack with a claimless placeholder rule. Not filed." That is the
    same family as ISSUE-175: new phase-1 structural strictness applied
    against every pack in the ecosystem, filed by nobody. Recommendation,
    explicitly a recommendation and not a decision: whoever plans ISSUE-175
    should scope R3 into the same lane, since both change the same
    function's strictness contract and each alone would force a second
    ecosystem-wide re-validation pass. (That plan's R2 — unifying
    `pkg/pack.Rule` and `pkg/packval.Rule` — is likewise unfiled but is a
    separate, larger question.)
    IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION, established from the corpus
    with ZERO interviews, but with one live sequencing hazard:
    `PLAN-ISSUE-166` (`status: draft`, the lane that surfaced this) fences
    the work OUT by name — its line 159 records that the sweep fixes
    SCRIPTS that exist and "there is no script here to fix", and its line
    362 states "OUT: creating the missing
    `pkg/pack/engine/testdata/grep/to-sarif.sh`. The orphaned [reference is
    a separate latent defect]". HOWEVER that lane DOES hold
    `pkg/pack/engine/testdata/contracts-grep-engine.yml` in the file scope
    of its TASK-004 and TASK-010 (it adds `-H -I` to that binding's command
    for sweep-predicate consistency). So any fix that edits that fixture
    must sequence AFTER PLAN-ISSUE-166 lands, or coordinate with it — do not
    open a concurrent edit on that file.
    Priority note, stated as observation and explicitly NOT a reorder
    (backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml
    position 5 and this slot does not change its rank. Note also that
    unlike items 18/20/21/22/23, this item is NOT Linux-CI-gated — every
    fact above is verifiable on darwin.
27. **`packs/contracts` still fails its own `pack add`/`pack test`
    phase3-fixtures self-validation on Linux CI for ONE test after item
    22's fix — byte-identical before and after, and the asymmetry itself
    is unexplained (ISSUE-177).**
    `TestInstallContractsLocalPack_InstallsWithSuppliedCommand` fails on
    real Linux CI with `pack validation (test) of the validation copy
    failed in phase3-fixtures: 14 validation error(s)`. Established by a
    direct before/after comparison of two real CI runs' `gate-report.json`:
    run `32172705491` (commit `9aa278e`, immediately pre-fix) and run
    `32179966270` (commit `f8b3846`, item 22's/ISSUE-166's `-H -I` fix) —
    same file, same message, same error count (14) in both.
    This is item 22's (ISSUE-166) own residual, not a fresh symptom: the
    test is named in ISSUE-166's original affected-test list, and roughly a
    dozen structurally similar siblings on the identical `pack add`/`pack
    test` phase3-fixtures path ALL went green from that fix. This one alone
    did not. The asymmetry — same validation path, same pack, same fix
    applied, different outcome for one test — is what the issue exists to
    investigate, not the red itself.
    The 14 errors' actual content is UNKNOWN and unread: `backstop gate`
    truncates phase3-fixtures failures to the summary line, and
    `.github/workflows/ci.yml` has no `go test -v` step for the package.
    Reading them likely needs the throwaway `debug/*` diagnostic-branch
    technique that established ISSUE-166's own root cause — precedent:
    branch `debug/issue166-contracts-grep-repro` (PR #3, closed unmerged).
    CORRECTION TO THE ISSUE AS FILED, verified in tree 2026-08-18 by
    backlog-pm: ISSUE-177's own References section states the failing test
    lives at `pkg/pack/engine/contracts_local_install_test.go`. **That path
    does not exist.** The test is at
    `pkg/pack/distribution/contracts_local_install_test.go:51` (its doc
    comment cites CLM-092); `pkg/pack/engine` contains no such file.
    Recorded so an implementer is not sent to a nonexistent file — the
    issue text itself still needs correcting via issue-author.
    LEAD, verified in tree 2026-08-18 by backlog-pm and worth trying
    before any debug branch: the green/red split correlates exactly with
    `func TestMain` presence. Every sibling ISSUE-177 names as having gone
    green lives in a package that DECLARES `func TestMain` —
    `cmd/backstop` (`integration_test.go:19`) and `pkg/packval`
    (`main_test.go:36`). The one test that stayed red lives in
    `pkg/pack/distribution`, which declares NO `func TestMain` at all
    (confirmed: the package has none). That is precisely the sandbox-helper
    `TestMain` invisibility item 20's (ISSUE-163) guard leaves open, which
    `ISSUE-164` ("Packval Importing Packages Missing TestMain Guard",
    `type: question`, open) asks about — and `pkg/pack/distribution` is
    one of the two packages ISSUE-164 names by file as invisible to that
    guard. State this as a HYPOTHESIS, not a finding: it is a correlation
    over the tests ISSUE-177 itself enumerates, it has NOT been falsified
    on Linux, and a known limit cuts against over-reading it — item 22's
    separate grep cluster also failed in `cmd/backstop`, which DOES carry
    the guard, so the guard gap cannot explain everything on its own. If
    confirmed, ISSUE-177 would be the measurement ISSUE-164 itself asks
    for.
    LIFETIME CAVEAT: `DIR-027`'s thread-1 tier 2 (undelivered) deletes
    `packs/contracts` from `backstop-core` and de-vendors
    `pkg/pack/distribution/contracts_local_install.go` — which would
    plausibly delete this very test. This item is homed here anyway
    because DIR-024 owns live loud reds on the gate/engine path and
    DIR-027 owns the ecosystem/publication side; if tier 2 lands first,
    revisit whether this item is mooted by deletion rather than fixed.
    HOME REASONING: homed here on the same test this directive has now
    drawn repeatedly — a LOUD red whose cause is untraced is DIR-024; only
    "computes a result internally but reports the wrong verdict about it"
    is DIR-032. DIR-032 is `done` and DIR-033 takes only follow-ons filed
    by DIR-032 member plans, so neither competes. Note also that this is
    the THIRD PLAN-ISSUE-166 residual homed here, after item 24-or-later's
    ISSUE-170 lineage and item 26 (ISSUE-175), consistent with that plan's
    own refusal-to-absorb tasks.
    Note the issue's own severity framing: low urgency in isolation, filed
    because an unexplained residual red must not sit as an unexplained
    line in a CI report.
28. **go-toolchain's coverage-reuse freshness comparison is directionally
    backwards — reuse never fires on real Linux CI, so PLAN-ISSUE-172's
    shipped speedup is a complete no-op where it matters (ISSUE-179).**
    `PLAN-ISSUE-172` (closed 2026-08-19) shipped `backstop-ai/go-toolchain`
    v1.7.0 with a coverage-profile-reuse mechanism: `scripts/test-produce.sh`
    runs `go test -coverprofile=cover.out ./...` and THEN writes the stamp
    `.backstop/go-coverage-fresh`; `scripts/coverage-produce.sh` reuses the
    profile instead of re-running the whole suite when it judges it fresh.
    The check at `coverage-produce.sh:38` is `if [ -f "$stamp" ] && [ -f
    cover.out ] && [ ! cover.out -ot "$stamp" ]` — "reuse only if cover.out
    is NOT OLDER than the stamp." Confirmed in the installed pack
    (`.backstop/packs/backstop-ai/go-toolchain/scripts/coverage-produce.sh:
    38`, v1.7.0) and in the in-repo fixture copy
    (`cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/
    go-toolchain/scripts/coverage-produce.sh:38`).
    But `test-produce.sh` writes `cover.out` first (`:37`) and stamps at
    `:58` — so the stamp is always strictly newer, and the condition can
    never be true at full timestamp precision. macOS `/bin/sh` `test -ot`
    compares at whole-second resolution so the few-millisecond gap ties and
    reuse fires by accident; Ubuntu `dash` compares at nanosecond resolution
    and reuse never fires. Measured on real Linux CI (run `32275399064`,
    commit `2f8fa89` on `main`): `coverage_threshold` at ~602500ms vs. the
    ~2211ms the mechanism promises — essentially the pre-fix ~612000ms
    baseline. The pack's own comment block (`coverage-produce.sh:30-31`)
    states the intent as "a profile older than the stamp … falls through" —
    the DOCUMENTED semantics are inverted too, not just the expression; any
    fix must correct the comment, or the next reader re-derives the same
    backwards rule.
    ★ CORRECTION ONE, backlog-pm's not the issue's — ISSUE-179's "Direction"
    says "Fix lives entirely in the external pack repo … not backstop-core."
    THAT IS FALSE, and it is the most important thing a planner needs. Two
    core-side test files pin the defective expression and must change in the
    same lane. `cmd/backstop/gotoolchain_single_run_test.go`'s
    `TestGoToolchainSingleRun_CoverageProducerReusesAFreshProfile` is the
    "executable falsifier" for the reuse half, and it is a VACUOUS GREEN: it
    writes `cover.out`, then writes the stamp, then ages the stamp by one
    second via `os.Chtimes(stamp, now-1s, …)` with the comment "a deliberate
    mtime nudge so cover.out is NOT older than the stamp, which is the
    freshness relation the producer tests." That manufactures the exact
    relation production can never produce, since production writes the
    profile first and stamps after. ★ This is why CI stayed green while the
    mechanism was dead: the falsifier tests a filesystem state the real
    chronology cannot create. The regression test ISSUE-179 asks for is not
    a new test — it is this test's fixture reversed to match production
    chronology (stamp strictly newer than `cover.out`, at sub-second
    precision), and reversing it must be done as part of the fix or the same
    blind spot ships again. The second pinning site, `pkg/pack/engine/
    gotoolchain_installed_pack_singlerun_test.go`'s
    `TestInstalledGoToolchainPack_CarriesSingleRunConvention`, asserts
    `strings.Contains(producer, "-ot")` against the installed producer text,
    plus a `semverGreater` bar over `preFixGoToolchainPackVersion = "1.6.0"`
    — so the version bar needs raising to the fixed version, and the `-ot`
    string assertion red-lines option (a) below outright.
    ★ CORRECTION TWO — ISSUE-179 offers (a) drop the mtime comparison
    entirely, trusting stamp-presence alone, and (b) flip the comparison to
    assert the stamp is not older than `cover.out`. The issue presents (a)
    as the cleaner option and argues it is safe because `rm -f "$stamp"`
    unconditionally consumes the stamp. THAT SAFETY ARGUMENT DOES NOT HOLD,
    because it assumes `coverage-produce.sh` always runs — it does not. A
    concrete falsifying sequence: `gate --all` stamps and is interrupted
    before coverage runs; a later file/diff-scoped `gate` re-runs `go-test`
    with `-coverprofile=cover.out` narrowed to changed packages (the
    `./...` case-guard at `test-produce.sh:47-53,54-61`), overwriting
    `cover.out` with a PARTIAL profile and writing no stamp; under (a) the
    leftover stamp plus the partial profile satisfies the check, and the
    coverage dimension measures a partial profile with every unmeasured
    file reading as absent — precisely the silent-narrowing class
    `test-produce.sh` says the mechanism exists to prevent. Under (b) the
    same sequence is safe, since the leftover stamp is older than the
    freshly-overwritten `cover.out`, so reuse correctly declines. ★ So (b)
    is not "a defensive check if one is still wanted" as the issue frames
    it; the ordering assertion is load-bearing and (b) is the recommended
    shape. Residual even under (b), recorded so a planner does not think
    (b) closes everything: if the aborted run is followed by an invocation
    where `go-test` does not run at all, `cover.out` keeps its old mtime,
    the leftover stamp is still newer, and a STALE whole-module profile
    from an earlier tree state is reused. Closing that needs the stamp
    bound to the specific profile it certifies (e.g. `test-produce.sh`
    writing `cover.out`'s mtime or content hash into the stamp and
    `coverage-produce.sh` requiring the match) rather than any pure mtime
    ordering — weigh that shape against (b) rather than assuming (b) is
    complete.
    Delivery shape: pack repo version bump (`backstop-ai/go-toolchain`,
    local working copy `/Users/bmanson/src/projects/backstop-go-toolchain-
    pack`) + `pack update`/relock in backstop-core, same shape as this
    directive's ISSUE-129/ISSUE-135/ISSUE-145 pack-side precedents — but
    unlike those, this one carries a mandatory core-side edit too (the two
    guard tests above), so it is not a pure pack-side fix. The in-repo
    fixture copy at `cmd/backstop/testdata/go-toolchain/.backstop/packs/
    backstop/go-toolchain/scripts/coverage-produce.sh` carries the identical
    backwards line; whether it is fixed alongside is a planner call, but it
    must be a deliberate one — item 12 of this directive (ISSUE-137, the
    unguarded fixture/pack drift) is the standing reason that copy diverges
    silently.
    IN-FLIGHT NOTE: an empty `PLAN-ISSUE-179` scaffold (`phases: []`, status
    draft, untracked) already existed at triage time — a planner is
    mid-authoring; these corrections are aimed at that lane.
    Why DIR-024 and not DIR-032, following this file's own item 15/16 test:
    nothing here computes or reports a WRONG VERDICT. The gate's pass/fail
    is correct throughout; only the wall-clock cost the fix promised to
    remove is not removed. That is this directive's charter (wrong or
    missing DATA / cost on a correct verdict) and it has direct precedent in
    this file's own item 7 (ISSUE-099, "a measured 2x CI cost") — a pure
    gate-cost item already homed here. The vacuous-green half is a Go unit
    test that cannot falsify, not a gate dimension reporting a false
    verdict, and this directive already carries the test-harness-defect
    class (items 12/21/27). It also sits directly on item 1's (ISSUE-020)
    Linux-CI-viability line: the defect is invisible on darwin and
    deterministic on Linux, and darwin-only verification is exactly what let
    it ship.
    Priority note, stated as observation and explicitly NOT a reorder
    (backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml
    position 5 and this slot does not change its rank.
29. **`pkg/pack/distribution` declares no `TestMain` at all, so the Linux
    sandbox re-exec silently re-runs its whole suite instead of
    intercepting the sandbox-helper spec — the confirmed root cause of
    item 27's (ISSUE-177) unexplained asymmetry (ISSUE-180).**
    `pkg/pack/distribution` declares no `func TestMain` anywhere in the
    package — confirmed: `grep -rn "^func TestMain"` across the module
    returns only `pkg/packval/main_test.go:36` and
    `cmd/backstop/integration_test.go:19`, plus one gate testdata fixture.
    Because `pkg/packval`'s Linux sandbox is a re-exec trampoline that
    spawns `os.Executable()` — under `go test`, the calling package's own
    test binary — with `BACKSTOP_SANDBOX_HELPER_SPEC` set, and the child is
    expected to intercept that env var via `packval.MaybeRunSandboxHelper()`
    as its `TestMain`'s first statement, a package with no `TestMain` gets
    Go's default generated main instead: the child silently RE-RUNS THE
    ENTIRE `pkg/pack/distribution` SUITE FROM SCRATCH in the sandbox's
    scratch-copy cwd, fails fast off any `go.mod` ancestry, and exits 1. Go
    writes that to stdout, so `foldHelperStderrIntoError`
    (`pkg/packval/sandbox_diagnostic.go`) sees empty stderr and reports
    "wrote no diagnostic" while the real output vanishes.
    This is the confirmed root cause of item 27's (ISSUE-177) open anomaly:
    `TestInstallContractsLocalPack_InstallsWithSuppliedCommand`
    (`pkg/pack/distribution/contracts_local_install_test.go`, CLM-092)
    failing `phase3-fixtures: 14 validation error(s)` byte-identically
    before AND after `PLAN-ISSUE-166`'s `-H -I` fix landed. Fourteen errors
    = every fixture in `packs/contracts` (12 ast-grep-dispatched + 2
    grep-dispatched), confirming the mechanism is fixture-shape-agnostic.
    Same collision shape as item 20 (ISSUE-163), different package; the fix
    mirrors ISSUE-163's exactly (add the `MaybeRunSandboxHelper()`-gated
    `TestMain`). Darwin-invisible: `sandbox_nonlinux.go`'s
    `MaybeRunSandboxHelper()` is a bare no-op and the Linux sandbox files
    are `//go:build linux`-gated, so Linux CI is the only falsifier.
    Four boundary facts follow, load-bearing and not to be re-litigated.
    IT CONFIRMS ISSUE-164, DOES NOT DUPLICATE IT, AND ONLY HALF OF IT.
    ISSUE-164 (`type: question`, still open, DIR-024 roster) named
    `pkg/pack/distribution` AND `pkg/pack/engine` as packval-importing
    packages with no `TestMain` and said the right move on confirmation was
    to re-file/promote. ISSUE-180 is that promotion for
    `pkg/pack/distribution` only. `pkg/pack/engine` REMAINS UNCONFIRMED AND
    OPEN under ISSUE-164 — nothing traced a specific test there to real
    sandboxed dispatch.
    `pkg/gate` IS RULED OUT, and this corrects a prior directive/memory
    lead. Earlier notes in this family listed `pkg/gate` as a third
    TestMain-less at-risk package. Verified today: `grep -rl
    "backstop-core/pkg/packval" --include="*.go"` over the whole module
    hits exactly four directories — `cmd/backstop`, `pkg/pack/distribution`,
    `pkg/pack/engine`, `pkg/packval`. `pkg/gate` does not import packval by
    any file, so it cannot become a re-exec target through this mechanism.
    Confirmed absent from the import graph, not merely unobserved.
    THE STRUCTURAL GUARD'S BLIND SPOT IS NOT FIXED BY THIS ISSUE.
    `cmd/backstop/sandbox_helper_testmain_guard_test.go`'s
    `TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain` builds its
    roster from packages that import packval and ALREADY HAVE a `TestMain`
    (`scanGoPackages`, then `if pkg.testMain == nil { continue }`). A
    packval-importing package with no `TestMain` is invisible to it by
    construction — which is exactly this defect. ISSUE-180 adds the missing
    `TestMain`; it does not generalize the roster. That generalization
    stays ISSUE-164's territory. ISSUE-164 must NOT be closed when
    ISSUE-180 lands.
    LIFETIME CAVEAT, same shape as item 27's. DIR-027 thread-1 tier 2
    (undelivered) sits at backlog position 4, AHEAD of DIR-024 at position
    5. It deletes `packs/contracts` and de-vendors
    `pkg/pack/distribution/contracts_local_install.go` — verified still
    present in tree today — which would delete the failing test outright.
    Home here anyway (live loud red on the gate/engine path), but flag:
    revisit as mooted-by-deletion if DIR-027 tier 2 lands first. The
    residual is only partly mooted even then: the missing `TestMain`
    remains a latent trap for any future `pkg/pack/distribution` test
    reaching sandboxed dispatch, even after that specific test is gone.
    HOME REASONING: loud red with a missing legible diagnostic name is
    DIR-024, per the test this directive has now drawn nine times; only
    "computes a result internally but reports the wrong verdict about it"
    is DIR-032 (which is `done` regardless). ISSUE-180 is a further
    Linux-CI-viability residual of item 1 (ISSUE-020), in the same
    v0.2.0-release-investigation family as items 18/20/21/22/23/27. Not a
    new theme, not a founder roster call.

30. **In-repo pack vs external-mirror sync guard (ISSUE-174).** `packs/
    contracts` and `packs/substantiveness` (in-repo) can drift silently
    from their published external mirrors (`backstop-ai/go-contracts`
    v1.4.0, `backstop-ai/go-substantiveness`) with nothing to catch it.
    Same defect class as item 13's (ISSUE-137) fixture/released-pack sync
    guard — this directive's own text already names ISSUE-137 as
    precedent for homing exactly this kind of guard here. DIR-027 tier 2
    (de-vendor/delete, not yet done) would moot this specific pair if it
    lands first, but none of DIR-027's five acceptance criteria require an
    ongoing sync guard, and the defect class recurs for any future
    vendor-then-publish pack regardless.
31. **Thin-executor absence-rule dogfood, real residual (ISSUE-024).** The
    issue's originally-stated blocker (SPEC-035 + BUNDLE-009 OQ-7) is
    moot/already landed — the shipped pack uses ordinary presence-matching,
    no absence primitive — so the blocker language is stale. Real remaining
    work, confirmed via a live `pack check` run: add two missing rule
    families (bare language-name literals, which need real false-positive
    design care; Go-analysis imports `go/parser`/`go/ast`/`go/types`,
    straightforward), and fix that four of the seven currently-active rule
    families are silently dark on ordinary gate runs — slash-bearing
    `paths.include` fails open under semgrep's explicit-file dispatch, the
    same ISSUE-151 mechanism this directive already sources.
32. **Contracts engine selection hardcodes a pack-key preference
    (ISSUE-065).** `cmd/backstop/gate.go:1814-1819` prefers pack key
    `ast-grep-contracts`, else literal `ast-grep`, instead of using declared
    capability. A residual of ISSUE-063 that ISSUE-063's own
    pack-granularity mechanism structurally can't reach: `packsDeclaring
    GateType` breaks after the first matching engine per manifest, so it
    can't distinguish two same-`gate_type` engines within one pack.
33. **Agent-toolchain trust, needs re-scoping down (ISSUE-076).** Most of
    this issue's originally-cited instances are now stale or already
    fixed (`.claude/agents/implementer.md:32` already corrected,
    `PLAN-SPEC-054` is completed/inert prose, and the one other
    non-terminal reference is backed by an obsoleted issue). The only
    genuinely open piece is Solution part 2: a validator that resolves
    every command string named in a plan's verification tasks against the
    real discoverable command surface (`backstop commands`) — confirmed
    unbuilt, no structured verification-command field exists in the plan
    schema today. Whoever plans this should re-scope the issue down to
    that piece first.
34. **Substantiveness `noTarget` violations carry no waivable line
    (ISSUE-117).** `pkg/gate/substantiveness_join.go`'s `NoTargetViolation`
    has no `Line` field, so `pkg/waiver/adjudicate.go`'s `windowLines` has
    nothing to anchor to — these findings are structurally unwaivable. The
    issue deliberately proposes no fix direction; real line vs. synthetic
    anchor is a design call for whoever plans it.

## Notes

Grouped together as the catch-all for gate/engine-quality gaps that aren't
contracts-engine, pack-distribution, or traceability themed. The original
three issues are pre-existing and low-urgency (ISSUE-007 was already
self-demoted by its author to "defer until a real consumer needs it";
ISSUE-020's real-world exposure is explicitly noted as low pre-launch/
local-only/no-Linux-CI-yet, though it is rated `risk: critical` in its own
file once that changes; ISSUE-082 is dead-code + doc cleanup with no
runtime exposure). Positioned last of the four newly-added directives — no
deadline pressure, no bundle depending on any of them landing.

ISSUE-075 and ISSUE-077 (backlog-pm slotted, 2026-07-26) are both scoped
and low-urgency in the same vein — a smoke-fixture gap and a local
dev/contributor on-ramp trap, neither consumer-facing nor blocking other
work — so they ride along here for thematic fit (gate/engine quality)
rather than displacing ISSUE-020's priority within this directive.

Sequencing caveat: DIR-024 holds position 3 in BACKLOG.yml on the strength
of ISSUE-020 — Linux/CI viability is one of the four founder-designated
launch blockers (the other three being recipes/SPEC-054/DIR-019, remote
pack consumption/DIR-026/SPEC-055, and CI-driven releases/DIR-001, tiered
up 2026-07-27). ISSUE-082 is tier-2 by both the founder's
launch razor and its own priority note; it rides along in this directive
for thematic fit only (gate/engine quality), not because it shares
ISSUE-020's urgency. Anyone working this directive top-down should land
ISSUE-020 first — ISSUE-082 (and ISSUE-007) must not be picked up ahead of
it.

A second, smaller pattern worth naming for the founder, carried forward
from the now-moved ISSUE-093 discussion: the
`--file` repetition bug is the third instance of a CLI arg-shape defect
where the accepted argument shape diverges from the advertised one —
alongside ISSUE-089 (`artifact validate` discarding a positional path) and
the residual half of ISSUE-074 (`pack relock` taking a path where its
siblings take a name). DIR-017 (Pack CLI Hardening) is `done`, so that
pattern currently has no live home. Recorded as an observation for the
founder, not a claim on this directive's scope.

ISSUE-096 (backlog-pm slotted, 2026-07-28) was slotted under the standing
clear-fit grant. It is `type: enhancement`, `scope: isolated`,
`uncertainty: known`, `risk: moderate` — it rides along on thematic fit and
must NOT displace ISSUE-020 or ISSUE-092 within this directive. Intra-
directive dependency: ISSUE-096's verification step depends on ISSUE-092
being fixed, so if both are worked, 092 comes first. Why DIR-024 and not
DIR-027 ("Pack Fleet Publication & Migration"): DIR-027 owns which packs
exist, where they are published, and which lock points where — its five
threads and every acceptance criterion are publication/migration/lock-state,
and it explicitly disclaims mechanism design; it has no charter claim on
rule content. This issue is a false-blocking gate verdict on correct code
that forces scattered escapes — gate/engine quality, this directive's theme
— and it happens to be fixed in a pack because this repo dogfoods its rules
as packs. DIR-014 (the directive the self pack was born under) is `done`.
An observation for the founder, not a claim on scope: ~80
`// nosemgrep:` suppressions exist across `cmd/` and `pkg/` today, a
parallel escape channel that never passes through `pkg/waiver` adjudication
and appears in no ledger — ISSUE-096 names one instance; the channel itself
is unowned and unfiled. Two of those suppressions
(`pkg/pack/distribution/contracts_provisioning_test.go:26`,
`pkg/pack/distribution/spec015_lineage_test.go:133`) still name the
pre-rename rule id
`backstop.packs.backstop.self.rules.no-baked-language-token`, which is
neither equal to nor a suffix of the current
`backstop.packs.backstop-ai.backstop-self.rules.no-baked-language-token` —
they are dead, and nothing signalled that when the pack was renamed. Both
sites happen to be masked by B2's new `*_test.go` exclusion, so nothing
fires today; the point is that the rot was silent. One adjacency, stated so
nobody trips on it: the consumer-side pickup step for the new pack version
is `pack update`, so ISSUE-095's `pack add` silent no-op (homed in DIR-026)
does not block it — but an operator who reaches for `pack add` on the
already-installed name will get the false "already installed and up to
date" success line.

ISSUE-099 (backlog-pm slotted, 2026-07-28) was slotted under the standing
clear-fit grant. It is `scope: contained`, `uncertainty: known`, `risk:
safe`. Why DIR-024 and not elsewhere: DIR-001 (Release Workflow) owns
`ci.yml` and is where the measured cost is PAID, but the fix site is a gate
CLI flag in `cmd/backstop/gate.go` — gate/engine surface, this directive's
theme. DIR-001 has no charter claim on gate output flags. The precedent is
this directive's own: ISSUE-082, ISSUE-077 and ISSUE-007 all ride here on
thematic fit as non-correctness gate/engine items. It was originally
discussed alongside its reporting-layer sibling ISSUE-100 ("100 is what the
numbers say, 099 is how many runs it costs to get them") — ISSUE-100 has
since moved to DIR-032 per the founder's 2026-08-10 cluster carve-out (see
the dated note below); fixing either did not and does not close the other.
A structural observation for the founder, now historical: with ISSUE-020
`closed` (delivered 2026-07-28), this directive's lede once read "Two
gate/engine-quality gaps" while it enumerated eleven, and every remaining
source but ISSUE-092 was tier-2. That imbalance is what the 2026-08-10
carve-out (below) resolved — ISSUE-092 and the rest of the
gate-verdict-honesty cluster now live in DIR-032. The stale count itself
went uncorrected for weeks after that carve-out — the lede still read "Two"
against an eleven-item enumeration until backlog-pm caught and fixed it
2026-08-16 while slotting ISSUE-137 (item 12 below).

Status correction (Brandon, 2026-08-02): this directive's status field now
reads `active`, reflecting delivered work under it — the ISSUE-020
Linux-sandbox/CI-gating half and the ISSUE-104/105 severity-contract hops —
while the majority of its 16 cited sources remain open.

ISSUE-107 was slotted by backlog-pm 2026-07-29 as a clear fit. It echoes
the gate-verdict-honesty cluster's shape (a check that reads authoritative
and silently isn't) but the founder's 2026-08-10 carve-out ruling (below)
did not name it among the cluster's eleven members, so it stays homed here
rather than moving to DIR-032. Provenance: ISSUE-107 was filed BY
`PLAN-ISSUE-105` TASK-006 as a deliberately-deferred residual, not
discovered independently. PLAN-ISSUE-105 is `status: completed` and its
CLASS-2 inventory names this exact loop, explaining the deferral: fixing it
introduces a `"warning"` status where consumers currently see `"pass"`,
which is a reporting change that belongs with ISSUE-100's renderer half
rather than inside a verdict fix. THE SEQUENCING CONSEQUENCE, now
cross-directive: ISSUE-107 (here) and ISSUE-100 (moved to DIR-032) touch
the same surface and should be planned together or 107 after 100 —
planning 107 alone means shipping a new `"warning"` state into a renderer
that ISSUE-100 says still miscounts warnings as violations. Its sibling
from the same audit — ISSUE-108 (contract carrier drops pack severity,
stays here too) — is cited by this directive, as is ISSUE-104 and
ISSUE-105 (both closed, cited by no directive). 107 is the only family
member that inverts the failure direction; ISSUE-106, its severity-carrier
sibling, moved to DIR-032 with the rest of the cluster.

ISSUE-108 (backlog-pm slotted, 2026-07-29) came in under the standing
clear-fit grant alongside ISSUE-107, and the two must be read as one arc
rather than two tickets. It is `type: bug`, `scope: contained`,
`uncertainty: known`, `risk: moderate`. It is the last of four residuals
from the pack-severity contract family: ISSUE-104 (SARIF severity lost at
the parser — fixed, `a42b065`), ISSUE-105 (step verdicts ignore severity
absent a policy entry — fixed, `d7d777c`), ISSUE-106 (the substantiveness
join discards a severity that exists upstream — moved to DIR-032 with the
rest of the gate-verdict-honesty cluster, 2026-08-10), ISSUE-107 (the
coverage step's warning-only finding set reads as pass, stays here), and
this one. In-flight coverage is nil and that is established from
artifacts, not assumed: `PLAN-ISSUE-105` is `completed`, and the session
that completed it filed 106/107/108 together at its closure (commit
`21e47ed`) as explicit hand-offs. Two corpus-honesty items ride along and
are NOT this directive's scope, raised to the founder separately:
ISSUE-104 and ISSUE-105 are both still `status: open` despite their fixes
having landed (and `PLAN-ISSUE-104` is still `draft`), and no directive
cites ISSUE-104 or ISSUE-105 today. ISSUE-108 was not among the eleven
issues the founder named in the 2026-08-10 cluster carve-out, so it stays
homed here despite the family-adjacency to ISSUE-106.
Correction in place rather than a stale claim (verified against the corpus
2026-08-02, not asserted): as of this writing ISSUE-104 and ISSUE-105 are
both `status: closed`, and `PLAN-ISSUE-104` and `PLAN-ISSUE-105` are both
`status: completed` (commit `87b12cf`, "close(ISSUE-104,105): severity-
contract hops delivered — plans completed, issues closed"); ISSUE-107 and
ISSUE-108 appear in this directive's own frontmatter `source:` list,
immediately above (ISSUE-106 no longer does — see the note above). The one
fragment of the original sentence that remains true: ISSUE-104 and
ISSUE-105 are still cited by NO directive's frontmatter `source:` — they
are named only in this file's prose.

Dead-ID correction, founder-ruled (Brandon, 2026-08-02): ISSUE-102 and
ISSUE-103 are DEAD. Both were reasoned toward but never filed — no file in
`issues/`, no add-commit anywhere in `git log --all --diff-filter=A`, and
the on-disk ID sequence jumps straight from ISSUE-101 to ISSUE-104. Their
git reservation tags (`backstop/issue/102`, `backstop/issue/103`) DO exist,
though, so the numbers are consumed against artifacts nobody ever wrote.
Ruling: do NOT refile either number under any circumstance; the burnt tags
stay exactly as they are — a burnt-and-unused reservation tag still blocks
re-issuance of that number, which is its whole purpose. This paragraph
exists to head off one specific confusion: a reader who sees the issue
corpus jump 101 → 104, or who follows this item's mention of the
harness-baked-globs defect looking for a ticket, will find reservation tags
with no artifacts behind them. That is intentional, not corruption, and not
evidence of a lost or deleted file. ISSUE-113 (moved to DIR-032 with the
rest of the gate-verdict-honesty cluster, 2026-08-10 — see that directive)
partially recaptures dead ISSUE-102's substance — the harness-baked
classification-globs defect — and is the live home for that concern. Dead
ISSUE-103's substance, `typescript-contracts` rejecting the bare-const
variable-contract idiom as observed in `bclabs-portal`, is NOT recaptured
anywhere in this repo — and that is correct, not a gap: it is a
`typescript-contracts` PACK defect, not a backstop-core concern at all.
Packs live OUTSIDE core by design, so no backstop-core issue should ever be
filed for it, under any ID, fresh or otherwise. Its home is that pack's own
tracking, `backstop-ai/typescript-contracts` (local working copy at
`~/src/projects/backstop-typescript-contracts-pack-local` — verified
present, though its `pack.yml` currently reads the pre-rename
`name: backstop/contracts` at version 1.1.0, so whoever files there should
first confirm they're working against current published repo state, not
this stale mirror). **Correction, founder-ruled (Brandon, 2026-08-02),
superseding this paragraph's earlier framing:** the prior reasoning that
DIR-022 ("Contracts Engine Hardening") is where this belonged was the PM's
original take from 2026-07-29, back when it was believed to be a core-side
contracts-engine concern; that reasoning is superseded, and DIR-022 was
never actually touched on its account.

ISSUE-115 (slotted 2026-08-02 under the standing clear-fit grant; founder-
confirmed the same day — Brandon: "ISSUE-115 → DIR-024: confirmed, standing
grant applies, same as 112/113/114") rides along on thematic fit. It is
`type: bug`, and while it echoes the gate-verdict-honesty cluster's shape —
a check that should catch drift silently doesn't — its root cause is a
different family: contract-block validation TIMING (checked only at spec
closure, and `consumes:` never at all) rather than a severity/data-carrier
defect inside an already-firing step verdict. It stays homed here rather
than moving to DIR-032: the founder's 2026-08-10 carve-out (see below)
named eleven specific cluster members and ISSUE-115 was not among them —
treat it as adjacent to, not a twelfth member of, that cluster. Scope note
carried from the issue itself: the `go-contracts` pack's inability to
express a grouped `const ( ... )` block member (blocking SPEC-036's
closure) is explicitly OUT of this issue's and this directive's scope —
pack work, same principle as the ISSUE-103 ruling recorded above.

**Cluster carve-out (Brandon, 2026-08-10):** the gate-verdict-honesty
cluster — eleven issues sharing the shape "a gate step computes a result
but reports the wrong verdict about it" — has moved to a new dedicated
directive, DIR-032 "Gate Verdict Honesty". Eight of the eleven were cited
here (ISSUE-092, 093, 097, 100, 106, 112, 113, 114) and moved with their
Description/Notes prose; the other three (ISSUE-066, 067, 091) were named
repeatedly in this file's Notes as cluster siblings but were never added to
this directive's `source:` frontmatter, so they had no prior directive home
at all and were added fresh to DIR-032. This directive keeps its
non-cluster sources (ISSUE-007, 020, 082, 075, 077, 096, 099, 107, 108,
115) — the ten items enumerated in the Description above, several of which
are related to the cluster by theme (ISSUE-096, ISSUE-107, ISSUE-108,
ISSUE-115) but were not named among DIR-032's eleven members and were
deliberately left here rather than folded in unasked. See DIR-032's
Description and Notes for the full cluster writeup and the founder ruling's
rationale.

ISSUE-125 slotted by backlog-pm 2026-08-15 under the standing clear-fit
grant. `type: technical-debt`, `scope: isolated`, `uncertainty: known`,
`risk: safe`. It rides here on thematic fit and displaces nothing.
Why DIR-024 and not elsewhere, stated as charter reasoning: this is the
ISSUE-096 precedent exactly — pack rule imprecision producing false findings
on correct code, fixed in the pack repo because this repo dogfoods its rules
as packs. DIR-027 owns which packs exist, where they are published and
which lock points where, and explicitly disclaims mechanism design — it has
no charter claim on rule CONTENT. DIR-005 (Extract go-standards Pack), the
directive under which this pack was born, is `done`. DIR-032 is verdict
honesty: GO-005 reports exactly the verdict its regex earns, so the defect
is rule precision, not a lying verdict.
A live adjacency the founder should rule on, recorded as an observation and
NOT acted on: ISSUE-061 ("go-standards error-type-suffix rule misfires on
non-error structs", GO-021) is the SAME defect class in the SAME pack — and
is homed in DIR-021 (Traceability Hardening & Corpus Drain), where the
directive's own text places it as a deadline-driven exception rather than a
charter fit (DIR-021's four numbered threads are all traceability/corpus;
ISSUE-061 is introduced separately under "Time pressure — carries a hard
deadline"). Verified in tree 2026-08-15: ISSUE-061's defect is STILL LIVE at
the installed v1.2.1 — the whole-file DOTALL `pattern-regex` is unchanged at
`rules/core/go-core.yml:186` — and its inline waiver is still in place at
`cmd/backstop/artifact_validate.go:19`, expiring **2026-10-12**, after which
the gate goes RED on a false positive. Both rules are `severity: WARNING`
and both live in the SAME file of the SAME pack at the SAME version, so ONE
version bump + one `pack update` + one relock fixes both. Consequence for
whoever sequences this: working ISSUE-125 alone means bumping the pack
twice; working them together costs one bump and retires a hard deadline
early. The founder may prefer to co-locate the two (move ISSUE-061 here, or
ISSUE-125 to DIR-021) — that is a re-home call and is deliberately NOT made
here.
One corpus-honesty item riding along, not this directive's scope: ISSUE-061's
file still cites the PRE-RENAME pack path and name
(`.backstop/packs/backstop/go-standards/...`, `backstop/go-standards`). That
path is GONE from the tree — the installed pack is `backstop-ai/go-standards`.
The waiver comment in `artifact_validate.go` was updated to the renamed rule
id but the issue file was not; route any correction through issue-author.

ISSUE-137 slotted by backlog-pm 2026-08-16 under the standing clear-fit
grant. `type: technical-debt`, `scope: contained`, `uncertainty: known`,
`risk: moderate`. It rides here on thematic fit and displaces nothing. Why
DIR-024 and not DIR-032 "Gate Verdict Honesty", which the issue's own text
invokes: DIR-032's charter sentence is specifically "a gate STEP computes a
result internally but reports the wrong verdict about it." Nothing here
reports a wrong verdict — the exemption tests report exactly the verdict
their fixture earns, and every gate step behaves correctly; the risk is that
the fixture stops describing the shipped pack, which is a defect in this
repo's own `go test` corpus, not in a gate verdict. That is the same
boundary test already applied twice in this file, to ISSUE-125 ("GO-005
reports exactly the verdict its regex earns") and to ISSUE-115 (adjacent to
the cluster, deliberately not folded in). The affirmative precedent is this
directive's own item 4, ISSUE-075: a backstop-core test fixture that makes a
test pass while proving nothing, homed here rather than in the cluster.
ISSUE-137 is the same category one layer over. Also considered and rejected:
DIR-027 owns which packs exist, where they are published and which lock
points where, and explicitly disclaims mechanism design — a core-side
test-parity guard is mechanism; DIR-023 is pack-distribution correctness
(provenance, local-path caching), and nothing here is wrong with install or
lock behavior, the lock is merely the natural key for the guard to read;
DIR-021's "corpus drain" means the artifact corpus, not test fixtures.
Provenance is the standard plan-filed-residual shape, so in-flight coverage
is provable from the artifacts rather than assumed: ISSUE-136, ISSUE-137 and
ISSUE-138 were filed together as PLAN-ISSUE-129 fallout in one commit
(`763ecd0`, "issue: annotate ISSUE-053, file ISSUE-136/137/138 from
PLAN-ISSUE-129 fallout"), and no plan in `plans/` targets ISSUE-137 or a
fixture-parity guard. One observation carried to the founder rather than
acted on: `PLAN-ISSUE-129` reads `status: draft` while implementation-shaped
changes for it sit uncommitted in the tree (the fixture flip, the pack bump
to 1.4.0, and a new untracked
`cmd/backstop/pack_gate_issue129_regression_test.go`) — flagged as a
status/lineage observation, not a finding about this issue.

ISSUE-141 slotted by backlog-pm 2026-08-16 under the standing clear-fit
grant. It rides here on charter fit and displaces nothing — in particular
it must NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. Why DIR-024 and not DIR-032 "Gate Verdict Honesty",
which is the obvious pull and must be answered head-on: this issue's fix
site is the SAME 35-line function, `pkg/packval/executor.go`'s `RunEngine`,
as DIR-032's brand-new item 15 (ISSUE-140), and its blocking relationship
is to DIR-032's item 4 (ISSUE-092). Surface adjacency is not the test, and
this file has now drawn that line four times — ISSUE-115, ISSUE-125
("GO-005 reports exactly the verdict its regex earns, so the defect is
rule precision, not a lying verdict"), ISSUE-137, and item 12's own slot
note. DIR-032's charter sentence is "a gate step computes a result
internally but reports the WRONG VERDICT about it." Nothing here reports a
wrong verdict: the parse failure is loud, blocking, and correctly
attributed to a broken pipeline. That is exactly what separates it from
its two neighbours — ISSUE-140 IS a lying verdict (an engine that never
started reads as a clean negative fixture) and ISSUE-092 IS a lying
verdict (`phase3-fixtures: pass` having executed zero checks); ISSUE-141
SHOUTS where those two lie. The one honest concession, recorded rather
than buried: the error phase3 renders blames the FIXTURE ("negative
fixture run failed") rather than naming the unapplied convert step, so the
diagnosis is coarse — but a coarse blocking error is a diagnosability
cost, not a false verdict, and a planner should improve the message while
fixing the cause.
AFFIRMATIVE PRECEDENT, not merely elimination: this directive already
owns `pkg/packval` surface. Item 1 (ISSUE-020) is `pkg/packval/sandbox.go`,
and item 3 (ISSUE-082) enumerates `pkg/packval/executor.go:63` among the
`CheckToolAllowed` call sites — the very function this item fixes, ten
lines above the fix site. So packval-executor work has a home here
already.
BUNDLE-005 "Pack Validation" was considered and rejected as a home: it is
an unhomed bundle awaiting a directive, and BACKLOG.yml's own re-listing
note for it says its validation substance ships today as `pkg/packval` and
that its live residual defect (ISSUE-092) is already carried by DIR-024
and DIR-032 — i.e. that entry marks unhomed intent, not uncarried work. An
open issue needs a directive home, not a bundle.
IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION, established from the artifacts
rather than assumed, and zero interviews were run: PLAN-ISSUE-092 fences
this defect out explicitly by file ownership ("The fix belongs in
`pkg/packval/executor.go`, which this lane does not own") and orders it
FILED, not folded in. `PLAN-ISSUE-140-packval-executor-narrow-neverstarted.
plan.yml` (ready to implement at commit `ef35010`, mid-flight in the
working tree — untracked `pkg/check/never_started.go` present) DOES own
`executor.go`, but only for the never-started predicate; its scope fence
states "THIS LANE DOES NOT TOUCH executor.go [phase3]— deliberate
file-exclusivity split" and it carries no convert-application task.
SEQUENCING CONSEQUENCE a planner must respect: `pkg/packval/executor.go`
is a live, actively-edited file right now. Do not open a lane on this item
concurrently with PLAN-ISSUE-140 — sequence it after that plan lands, or
the two collide on the same function.

ISSUE-143 slotted by backlog-pm 2026-08-16 under the standing clear-fit
grant. It rides here on charter fit and displaces nothing — in particular
it must NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. Why DIR-024 and not DIR-032 "Gate Verdict Honesty",
which the packval-drift family straddles and must be answered head-on:
apply this file's own repeatedly-drawn test — DIR-032's charter sentence is
"a gate step computes a result internally but reports the WRONG VERDICT
about it," and ISSUE-143 reports nothing at all; it is structural
duplication with no verdict claim, the same category as ISSUE-137 (the
fixture/released-pack sync guard, homed here on exactly this ground). One
counter-pull worth disposing of rather than ignoring: DIR-032 does own
ISSUE-136, an audit with no verdict defect of its own — but that audit
bounds WHICH VERDICTS ARE LYING engine-by-engine, whereas ISSUE-143 bounds
nothing and audits nothing; it is a refactor of the fix site for this
directive's OWN item 13. Affirmative precedent, not merely elimination:
this directive already owns the surface — item 1 (ISSUE-020) is
`pkg/packval/sandbox.go`, item 3 (ISSUE-082) names
`pkg/packval/executor.go:63`, and item 13 (ISSUE-141) is the very function
whose fix creates this residual.
IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION, established from the artifacts
rather than assumed, and zero interviews were run: PLAN-ISSUE-141
explicitly declines the consolidation and installs CLM-006 as an interim
tripwire instead of a fix, and no plan in `plans/` targets ISSUE-143
(backlog-pm enumerated `plans/` 2026-08-16).
Priority note, stated as observation and explicitly NOT a reorder
(backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml position
5 and this slot does not change its rank.

ISSUE-135 slotted here 2026-08-16, correcting a routing call ISSUE-135's
own References section had made unilaterally toward DIR-032. Worth stating
precisely, since it is easy to misread as an actual reversal: ISSUE-135 was
never actually added to DIR-032's `source:` frontmatter or Description/
Notes — checked directly against DIR-032's current text, not assumed. The
DIR-032 pull existed only inside the issue's own file, in a References
bullet reading "Fits DIR-032 (Gate Verdict Honesty)," which was never
accepted into that directive. So there was nothing to remove from DIR-032
itself; the only correction needed was in the issue file's own References
section, made in the same pass (see
`issues/ISSUE-135-go-test-converter-bare-basename-file.issue.md`). Founder-
approved rationale for DIR-024 over the issue's own DIR-032 self-citation:
DIR-032's charter is a gate step reporting the WRONG VERDICT about a result
it computed; post-ISSUE-129, this issue's go-test finding IS reported and
DOES correctly redden the gate — the defect is in the finding's own `File`
data, not in verdict computation, matching this directive's ISSUE-096/
ISSUE-125 precedent (pack rule/converter precision fixes) rather than the
gate-verdict-honesty cluster's shape. The baseline-identity counter-
argument (same-named test files across packages could conflate under
baseline compare) was considered and is recorded, not dropped — see item
15 above for the full reasoning and the founder's ruling on why it still
homes here.

ISSUE-145 filed 2026-08-16, first observed as an inherited pre-existing gate
failure during `PLAN-ISSUE-124` implementation. Slotted here under the
standing clear-fit grant; rides on charter fit and displaces nothing — in
particular it must NOT displace item 1 (ISSUE-020) or the ISSUE-092
sequencing already recorded in this file. Why DIR-024 and not DIR-032 "Gate
Verdict Honesty": the issue's own file draws the comparison explicitly and
this directive's Description (item 16) restates it — the gate still
correctly reds on a real compile failure, the crash-guard is doing exactly
its designed job (SPEC-034 REQ-003/CLM-010), and only the diagnostic content
is missing, not the verdict. Same test this file has now applied six times
(ISSUE-096, ISSUE-115, ISSUE-125, ISSUE-137, ISSUE-141, ISSUE-135), and the
closest precedent in shape is item 15/ISSUE-135 (wrong/missing data on a
correct verdict, this directive's charter). Distinct from ISSUE-067 (DIR-032
item 2): that is an extraction bug on data present on stdout; this is data
never offered to the converter because it is on a channel (`stderr`) the
dispatcher discards by design (REQ-009/CLM-028) — confirmed by the issue
author as a different root cause, not merely a different symptom.
Fix lives in the pack repo (`backstop-ai/go-toolchain`) — version bump +
relock, same shape as this directive's ISSUE-129/ISSUE-135 precedents; no
backstop-core code change is required, since the `producer:` mechanism the
fix would use (`pack_gate.go:680-725`) already ships core-side.
Priority note, stated as observation and explicitly NOT a reorder
(directive-author has no reorder authority): DIR-024 sits at BACKLOG.yml
position 5 and this slot does not change its rank.

ISSUE-158 slotted here 2026-08-17 by backlog-pm under the standing
clear-fit grant. Provenance is `PLAN-ISSUE-148` TASK-005 item 1, a MANDATED
follow-on filing ("FILE IT, do not fix it here" — that plan's file scope
lists `cmd/backstop/gate_substantiveness_e2e.go`'s zero-match harness as OUT
at line 254-255), so this was surfaced, not caused, by that lane and
ISSUE-148 is not at fault. IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION,
established from the corpus with zero interviews: no plan in `plans/`
targets ISSUE-158 or the zero-match harness, and PLAN-ISSUE-148 (status:
draft, created 2026-08-17, actively mid-flight in the working tree) forbids
absorbing it. The four reds were NOT independently re-measured by
backlog-pm because the tree is being actively modified by that lane and any
measurement would be unattributable — the patch site, the stale docstring
and the committed `RuleSourcePath` resolution WERE verified in tree.
Priority note, stated as observation and explicitly NOT a reorder
(backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml position
5 and this slot does not change its rank, but the founder should know that
until item 18 is fixed core's own suite carries four red E2E tests that no
other lane will clear.

ISSUE-147 discovered 2026-08-16 by `implementer-issue092` during
PLAN-ISSUE-092 verification. Slotted here under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. Same file as item 1 (ISSUE-020, `pkg/packval/
sandbox_nonlinux.go`'s macOS surface) and shares its sandbox-mechanism
theme, but is a distinct defect: ISSUE-020 is Linux having no sandbox
mechanism at all; ISSUE-147 is the existing macOS mechanism mis-embedding a
relative path into its own profile. This directive already owns
`pkg/packval` sandbox/executor surface — item 1 (ISSUE-020) is
`sandbox.go`, item 3 (ISSUE-082) and item 13 (ISSUE-141) both touch
`executor.go` — so a second `sandbox_nonlinux.go` defect has a clear
existing home.
The issue author's own file flags that its compounding stderr-discard half
could arguably fit DIR-032's "opaque crash where a legible finding belongs"
language. Considered and decided, not left ambiguous: both halves of
ISSUE-147 stay together in DIR-024, because the primary defect (the
relative-path profile bug) is unambiguously a DIR-024 fit — `pack test`/
`pack check` already reports a loud, blocking failure (exit 71), just with
the wrong diagnostic content — the same "loud red, needs a legible name"
shape this directive has already drawn for item 16/ISSUE-145 and item
15/ISSUE-135, not DIR-032's "computes a result but reports the wrong
verdict about it." Splitting the stderr-discard half into a separate
DIR-032 issue would fragment one fix (both changes land in the same
function pair, same commit, same version bump upstream if this were a
pack — but it is not; both fixes are backstop-core-native since
`pkg/packval/sandbox_nonlinux.go` is core code, not a pack) for no
charter gain.
Priority note, stated as observation and explicitly NOT a reorder
(directive-author has no reorder authority): DIR-024 sits at BACKLOG.yml
position 5 and this slot does not change its rank.

ISSUE-131 slotted here 2026-08-17 by directive-author under the standing
clear-fit grant — this is a clear fit, not a founder roster call: it is
item 3's (ISSUE-082) own plan-mandated residual, the three sibling files
`PLAN-ISSUE-082` (`status: completed`) named by file and line as
deliberately out of its declared scope, not a new theme. Rides here on
charter fit and displaces nothing — in particular it must NOT displace item
1 (ISSUE-020) or the ISSUE-092 sequencing already recorded in this file.
Flag for the founder, not a change request and not acted on here:
ISSUE-082 itself is still filed `status: open` while `PLAN-ISSUE-082` is
`status: completed` — a delivered-but-open candidate that backlog-pm
surfaced during the 2026-08-17 full sweep and did not act on. ISSUE-082's
own status is left unchanged by this slotting.
Priority note, stated as observation and explicitly NOT a reorder
(directive-author has no reorder authority): DIR-024 sits at BACKLOG.yml
position 5 and this slot does not change its rank.

ISSUE-165 slotted 2026-08-18 by backlog-pm under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION,
established from the corpus with ZERO interviews: no plan in `plans/`
targets ISSUE-165, and `PLAN-ISSUE-163` — the only actively-authored lane
in this neighborhood — fences `pkg/packval/**` OUT of its file scope by
name at line 137 (`main_test.go`, `sandbox_linux*.go`,
`sandbox_nonlinux.go`), so the lane that exposed this defect is
contractually barred from fixing it. `PLAN-ISSUE-020` is `completed`.
Also record: the fix is a DARWIN-INVISIBLE change — the test carries
`//go:build linux`, so its only falsifier is Linux CI, exactly as item 20
(ISSUE-163) noted for its own fix; a planner must not expect a local
red-to-green on a Mac. Priority note, stated as observation and
explicitly NOT a reorder (backlog-pm has no reorder authority): DIR-024
sits at BACKLOG.yml position 5 and this slot does not change its rank.

ISSUE-170 slotted 2026-08-18 by backlog-pm under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. It is a SECOND-ORDER residual: item 21's
(ISSUE-165) own delivery residual, which makes it item 1's (ISSUE-020)
grand-residual, since ISSUE-165 was itself item 1's residual. IN-FLIGHT
COVERAGE IS NIL BY CONSTRUCTION, established from the corpus with ZERO
interviews: no plan in `plans/` targets ISSUE-170, and `PLAN-ISSUE-165`
(`status: draft`) — the only lane on this surface — fences the work out by
name in its own source comment and in TASK-005. Unlike item 21, this
item's guard is untagged (moved off `//go:build linux` by ISSUE-165's own
fix, commit `8d35706`) and darwin-visible, so a planner should NOT expect
a Linux-CI-only falsifier here. Priority note, stated as observation and
explicitly NOT a reorder (backlog-pm has no reorder authority): DIR-024
sits at BACKLOG.yml position 5 and this slot does not change its rank.

ISSUE-169 slotted 2026-08-18 by backlog-pm under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. It is item 21's (ISSUE-165) own delivery residual,
mandated by that item's own plan, `PLAN-ISSUE-165` TASK-005 step (5),
which explicitly refuses to absorb it — not a new theme, not a founder
roster call. IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION, established from
the corpus with ZERO interviews: no plan in `plans/` targets ISSUE-169,
and `PLAN-ISSUE-165` (`status: draft`) — the only lane on this surface —
names it as an owed follow-on rather than in-scope work. Unlike item 21,
this item's fix site is untagged production source
(`pkg/packval/sandbox_linux.go`), so a planner should NOT expect a
Linux-CI-only falsifier here — it is locally verifiable on darwin.
Priority note, stated as observation and explicitly NOT a reorder
(backlog-pm has no reorder authority): DIR-024 sits at BACKLOG.yml
position 5 and this slot does not change its rank.

ISSUE-175 slotted 2026-08-18 by backlog-pm under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. It is item 22's (ISSUE-166) own delivery residual,
mandated by that item's own plan, `PLAN-ISSUE-166`, whose TASK-004/TASK-010
text holds the one fixture this item targets in scope for an unrelated
edit (`-H -I` sweep-predicate consistency) while its own findings text
records "there is no script here to fix" and explicitly excludes creating
the missing script — the provenance test that decided this item's home is
that same plan's own out-of-scope language, not a founder roster call.
IN-FLIGHT COVERAGE IS NIL BY CONSTRUCTION, established from the corpus
with ZERO interviews: no plan in `plans/` targets ISSUE-175, and
`PLAN-ISSUE-166` (`status: draft`) — the only lane on this surface — fences
the fix out by name while still touching the one fixture this item would
edit, which is why any implementation must sequence after that plan lands
or coordinate with it rather than opening a concurrent edit. Unlike items
18/20/21/22/23, this item is NOT Linux-CI-gated — every fact underpinning
it is verifiable on darwin. Priority note, stated as observation and
explicitly NOT a reorder (backlog-pm has no reorder authority): DIR-024
sits at BACKLOG.yml position 5 and this slot does not change its rank.

ISSUE-177 slotted 2026-08-18 by backlog-pm under the standing clear-fit
grant; rides on charter fit and displaces nothing — in particular it must
NOT displace item 1 (ISSUE-020) or the ISSUE-092 sequencing already
recorded in this file. It is ANOTHER delivery residual of item 22
(ISSUE-166): the failing test was named in ISSUE-166's own original
affected-test list, but `PLAN-ISSUE-166`'s `-H -I` fix left it
byte-identically failing while roughly a dozen structural siblings on the
same phase3-fixtures path cleared — an unexplained asymmetry the fix's own
lane did not absorb, not a new theme, not a founder roster call. IN-FLIGHT
COVERAGE IS NIL BY CONSTRUCTION, established from the corpus with ZERO
interviews: no plan in `plans/` targets ISSUE-177, and `PLAN-ISSUE-166`
(`status: draft`) is the fix whose own affected-test list this issue's
residual test came from, not an owner of this residual. Correction filed
here rather than assumed correct: the issue's own References section
names the failing test's location as
`pkg/pack/engine/contracts_local_install_test.go`, which does not exist —
the test is at
`pkg/pack/distribution/contracts_local_install_test.go:51`; route the
correction through issue-author. Unlike items 18/20/21/22/23, this item's
own investigation is NOT Linux-CI-gated to start — the file-location
correction and the `TestMain`-presence correlation lead are both
verifiable on darwin, though confirming the underlying CI failure itself
requires the same kind of real Linux-CI evidence item 22 used. Priority
note, stated as observation and explicitly NOT a reorder (backlog-pm has
no reorder authority): DIR-024 sits at BACKLOG.yml position 5 and this
slot does not change its rank.

Items 30-34 (ISSUE-174, ISSUE-024, ISSUE-065, ISSUE-076, ISSUE-117) added
2026-08-19 by directive-author, all five per one team-lead-relayed,
founder-approved batch of rulings from a completed backlog-pm investigation
sweep — not self-initiated clear-fit slotting under the standing grant,
the provenance every other addition above carries. Priority note, stated
as observation and explicitly not a reorder (directive-author has no
reorder authority): DIR-024 sits at BACKLOG.yml position 5 and this batch
does not change its rank. Individual notes:

- ISSUE-024: real remaining work confirmed via a live `pack check` run
  (see item 31); its original blocker (SPEC-035 + BUNDLE-009 OQ-7) is
  stale, not load-bearing.
- ISSUE-065: residual of ISSUE-063 (also DIR-024) that ISSUE-063's own
  fix structurally cannot reach — same directive, adjacent mechanism.
- ISSUE-076: needs re-scoping down before planning — most originally-cited
  instances are stale; only Solution part 2 (the plan-verification-command
  validator) is confirmed still open. Flagged, not resolved, here.
- ISSUE-117: no fix direction proposed by the issue itself; real-line vs.
  synthetic-anchor is a design call for whoever plans it.
- ISSUE-174: same defect class as item 13 (ISSUE-137), this directive's
  own precedent for homing fixture/released-pack sync guards. DIR-027
  tier 2, if it lands first, would moot this specific in-repo/mirror pair
  but not the defect class itself.
