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
    - "ISSUE-137"
---

## Description

Twelve gate/engine-quality gaps that don't fit the other three newly-added
directives' themes:

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
