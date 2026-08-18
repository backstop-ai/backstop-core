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
residual — not a new theme, not a founder roster call.

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
    2x CI cost (ISSUE-099).** `--json` is a plain boolean on the global
    `jsonFlag` (`cmd/backstop/gate.go:33,170-176`); when set,
    `gate.FormatJSON(result)` REPLACES the human-table render path rather
    than running alongside it. Confirmed by grep: no `--json-out` /
    `jsonOut` flag exists anywhere in `cmd/backstop/`. There is no flag
    that separates "what renders where."
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
    ISSUE-099 by ID in its own comment. The deliberate design reason is
    recorded in that comment and should be preserved by whoever closes
    this: separate steps make it visible from the step list alone which
    invocation gates, a lesson from the retired linux-sandbox probe
    workflow whose every step ended in `exit 0`. So the retirement trigger
    is "collapse two steps into one," not "remove `set +e` bracketing."
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
