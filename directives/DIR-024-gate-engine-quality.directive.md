---
title: "Gate/Engine Quality"
number: DIR-024
created: "2026-07-15"
schema_version: directive/v1

directive:
  status: queued
  source:
    - "ISSUE-007"
    - "ISSUE-020"
    - "ISSUE-082"
    - "ISSUE-075"
    - "ISSUE-077"
    - "ISSUE-092"
    - "ISSUE-093"
    - "ISSUE-096"
    - "ISSUE-097"
    - "ISSUE-100"
    - "ISSUE-099"
    - "ISSUE-107"
    - "ISSUE-106"
    - "ISSUE-108"
    - "ISSUE-112"
    - "ISSUE-113"
---

## Description

Two gate/engine-quality gaps that don't fit the other three newly-added
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
6. **`pack test` phase3 cannot fail — fixture execution is dead code for
   every real pack (ISSUE-092).** `pkg/packval`'s `Rule` struct
   (`manifest.go:75-88`) reads a rule's source file from YAML key `file:`,
   but every real pack.yml declares it as `rule_path:` — which is what the
   runtime gate parser (`pkg/pack/manifest.go:144`,
   `RulePath string \`yaml:"rule_path"\``) consumes. The authoring-time
   validator and the runtime parser are two independent manifest models
   that have drifted; independently confirmed `rule_path`/`RulePath`
   appears nowhere under `pkg/packval/` (grep clean). Consequence in
   `phase3.go`'s `RunFixtures` (lines 28-113): `rule.File` unmarshals to
   `""` for every real pack, so the `if rule.File != ""` guards at `:31`,
   `:52-58`, `:62` and `:76` are all false — `executor.RunEngine` is never
   invoked for any layer 1-2 semgrep rule declared via `rule_path:`.
   `res.Errors` stays empty and `res.Status` stays `"pass"` (`:217-219`)
   having executed zero fixture checks. Measured, not inferred:
   implementer-087 rewrote a negative (violating) fixture to be fully
   compliant — deleting the very violation it exists to catch — and
   `pack test` still returned `phase3-fixtures: pass`, exit 0; removing a
   rule's `paths: exclude` likewise fails nothing. Independently
   reconfirmed by the issue author against
   `.backstop/packs/backstop/go-standards`. Matters because `pack test` is
   the pack-quality gate for the whole ecosystem, and phase3 is the
   mechanism that enforces the fixtures-from-real-output/must-falsify
   convention (founder law) — a pack whose rule never fires on anything can
   ship green. Two measurement traps the fix must not reintroduce, both
   recorded in the issue: (a) directory-scan vs explicit-file-list
   divergence — semgrep's default ignores skip `*_test.go` on
   directory-target scans, so restored fixture execution must dispatch
   with explicit file targets the way the gate does, or it inherits
   ISSUE-091's undercount; (b) an engine/schema error (e.g.
   `InvalidRuleSchemaError`) currently surfaces as zero results and is
   indistinguishable from a genuine clean run — the fix must make that
   loud and distinct rather than foldable into either pass path. Scope not
   yet verified, explicitly left to whoever plans it: the `tool_config`
   fixture path (`phase3.go:116-143`, keyed on `tc.File`) and the layer-3
   `rule.Validator` path. Any fix must land a regression fixture proving
   phase3 CAN fail (a `rule_path:`-declared rule with a compliant negative
   fixture must turn `pack test` red) — otherwise the fix is itself a
   vacuous-green claim.
7. **`gate --file` false-REDs non-Go files whose directory holds no Go
   package (ISSUE-093).** `fileModeTestTarget`
   (`cmd/backstop/pack_gate_filemode.go:31-45`) fires whenever the scope is
   `GateScopeModeFile` and the dispatched binding declares
   `PackageScoped: true`. go-toolchain's `go-test` engine declares
   `package_scoped: true`
   (`.backstop/packs/backstop/go-toolchain/pack.yml:86`) and
   `crash_guard: true` (`:83`) — both confirmed present. It unconditionally
   calls `goTestPackageSelector(scope.Files[0])`
   (`pack_gate_filemode.go:44`), which derives a `go test` target from the
   file's directory (`filepath.Dir` → `./` + dir) with NO check that the
   directory contains any `.go` files. For `.github/workflows/ci.yml` the
   target becomes `./.github/workflows`; `go test` there exits non-zero
   with zero parseable findings, and `crash_guard` renders that legitimate
   no-op as an engine CRASH violation. So a per-file verdict on a non-Go
   file is not a property of the file: `--file README.md` PASSES only
   because the repo root happens to hold a Go package. Repo topology, not
   file correctness, decides the verdict. Reproduced on untracked-clean,
   long-tracked, unmodified files by two independent parties
   (implementer-087 during ISSUE-087 phase 4 on 2026-07-27, and the issue
   author). Blast radius bound, and record it explicitly so nobody
   over-scopes the fix: the DEFAULT diff-scoped gate is UNAFFECTED — it
   stays green with those same `.yml` files in scope (measured). This is
   the `--file` path only. Distinct from ISSUE-067, which is also about
   the go-test engine surfacing an opaque crash: there the trigger is a
   REAL test failure the converter cannot parse; here there is no test
   failure at all and no Go code in scope. Same surfacing weakness,
   different trigger — a fix to one does not close the other. The
   secondary, independent defect in the same command surface: `--file` is
   bound as a plain string (`cmd/backstop/gate.go:52`, `StringVar`) and
   read as a string (`gate.go:77`, `cmd.Flags().GetString("file")`), while
   the command's own `Use:` string advertises
   `gate [--all | --file FILE [FILE...]]` (`gate.go:37`). Repeating
   `--file` therefore silently overwrites rather than accumulating —
   verified: `--file README.md --file .github/workflows/ci.yml` reports
   "Gate running against 1 explicit files." A single `--file X` followed
   by bare positional args DOES accumulate (`gate.go:92` appends
   `args...`), so flag-repetition is the only broken shape. No error, no
   warning, no output trace of the discarded value. An agent looping
   `--file` calls believing the flag is repeatable verifies only its last
   file while believing it verified all of them. Why it matters here
   rather than as ergonomics: both defects corrupt the per-file "prove
   each file you touched is clean" verification discipline that exists to
   compensate for shared-tree risk during concurrent multi-agent work.
   Neither is a false GREEN in the gate's own verdict (defect 1 fails
   honest-loud, in the safe direction), but defect 2 IS a false coverage
   claim — the operator believes N files were checked when 1 was.
   Guidance for whoever plans it, kept as constraint not design: the fix
   must preserve SPEC-034 REQ-010 / CLM-035 — file-mode test scoping is
   PRESERVED, not dropped; a whole-module run in file mode is a regression
   asserted by `filemode_scoping_test.go`. So the correct shape is a guard
   on the derived target (or on crash-vs-no-op classification), not
   removal of the file-mode override. And per this directive's own
   zero-baked-language law, any "does this directory hold a Go package"
   guard must not be a baked language check in core — it belongs to the
   pack's declaration or to a generic no-op classification, not to a
   `.go` literal in the CLI.
8. **Self-pack rule imprecision forces un-adjudicated escapes in test files
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
9. **Waiver tokens keyed to a dead pack namespace fail open, and the
   harvest path cannot see them (ISSUE-097).** Two `@waiver:` tokens —
   `cmd/backstop/pack_gate.go:888` and
   `cmd/backstop/pack_gate_provision.go:119` — key the rule ID
   `backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine`.
   The pack path `backstop/self` exists nowhere: `backstop.lock` records
   `backstop-ai/backstop-self` at 1.1.2 (`source_type: git`), and
   `.backstop/packs/backstop-ai/backstop-self/` is the install. Fallout of
   the 2026-07-27 pack rename.
   Measured status is three things, not one: STALE (the namespace is
   gone), INERT (the rule matches nothing at either site today —
   `gate --file` returns zero, and PLAN-ISSUE-020's honesty pass
   independently agreed via a whole-ruleset semgrep run), and FAIL-OPEN
   (`waiver.Adjudicate` suppresses only on an EXACT rule-ID match,
   `pkg/waiver/adjudicate.go:124`, so if the rule ever matches these lines
   again the tokens will NOT suppress — the comments read as adjudicated,
   and are decorative). Both carry expiry 2027-07-17, so even the expiry
   warning will not surface them for a year.
   The structural half, and the reason this is directive-worthy rather
   than a two-line typo fix: **waiver harvest is finding-driven, not
   tree-driven.** `Adjudicate` (`pkg/waiver/adjudicate.go:94-108`) only
   reads the one-line-above/on-line association window of each finding it
   is HANDED; a token on a line where no finding lands is never harvested
   at all, so it does not even reach the `Unused` bucket (`:126-129`). The
   same blindness holds on the CLI surface: `backstop waiver list` →
   `runWaiverList` → `projectWaiverResult` (`cmd/backstop/waiver.go:78-119`)
   builds its finding list by running the `pack_engines` and
   `test_substantiveness` gate steps and feeding those violations to
   `Adjudicate` — so it inherits the identical constraint. Both the
   `waiver_resolution` gate step and `waiver list` therefore report "clean
   — no active waivers" (`pkg/gate/step_waiver.go:188`) truthfully-as-
   implemented and falsely-as-read. A rename-orphaned or hand-typo'd rule
   ID is architecturally invisible, not merely unreported.
   Scope, as two parts. (a) Re-key or remove the two tokens. (b) Give
   `waiver_resolution` / `waiver list` the ability to name a waiver whose
   rule-ID prefix matches no pack namespace in `backstop.lock`, WITHOUT
   requiring a live finding at that location — which means harvesting
   tokens from the tree for that cross-check, not only from finding-
   adjacent windows. Per this repo's loud-≠-blocking rule this is a
   WARNING, not a new gate failure: the enemy is the silent rot, not the
   noisy staleness.
   Constraint for whoever plans (a), which removes the issue's own open
   caveat: the ID is not a matter of analogy. Core composes it as
   `pack.NamespacedRuleID(manifest.NormalizedName, <engine-emitted rule
   id>)` = `packName + "/" + ruleID` (`pkg/pack/coordinate.go:42-44`,
   called at `cmd/backstop/pack_gate.go:734,868`), so the prefix MUST be
   the manifest name `backstop-ai/backstop-self/`. The dotted tail is
   engine-emitted; the closest LIVE evidence for its current shape is the
   working post-rename semgrep suppression at
   `cmd/backstop/version_test.go:168`
   (`backstop.packs.backstop-ai.backstop-self.rules.no-baked-tool-exec`),
   and the two live post-rename `@waiver:` tokens at
   `cmd/backstop/artifact_validate.go:17` and
   `pkg/pack/distribution/identity.go:38`. **Trap to avoid:**
   `cmd/backstop/bun_ratchet_flip_test.go:128` builds a MIXED id
   (`"backstop-ai/backstop-self/backstop.packs.backstop.self.rules." +
   fragment`) and `pkg/gate/policy_perpack_test.go:29` still holds the
   fully pre-rename form — both synthetic fixtures where any string works,
   so neither is evidence of the real shape. Do not grep-and-copy from
   them.
   Bound the scope explicitly: this is the `@waiver:` channel only. The
   parallel un-ledgered `// nosemgrep:` channel (78 suppressions across
   `cmd/` + `pkg/`, consumed inside the engine and dropped by design at
   `pkg/check/parsers.go:80,115`, two of them —
   `pkg/pack/distribution/contracts_provisioning_test.go:26` and
   `spec015_lineage_test.go:133` — carrying the same dead
   `backstop.packs.backstop.self.*` id) is the same defect class in a
   channel backstop does not adjudicate. It has no artifact and is NOT in
   ISSUE-097's scope; noted in item 8 already. Do not fold it in without a
   founder call.
   Verification bar, so the fix is not itself a vacuous-green claim: a
   fixture with a waiver token bound to a plausible-but-nonexistent pack
   namespace, sitting where no finding lands, that the new check names.
   If the check only fires when a finding is already present, it has
   reimplemented today's blindness.
10. **Step tallies count warnings as violations — the reporting half
    remains after the verdict half was fixed (ISSUE-100).** `StepResult`'s
    displayed count and `GateResult.total_violations` count every entry in
    `StepResult.Violations` identically, regardless of `Violation.Severity`.
    Two severity-blind call sites, both CONFIRMED LIVE in the current tree:
    `pkg/gate/result.go:225` (`r.TotalViolations += len(s.Violations)`, the
    JSON envelope's `total_violations`) and `pkg/gate/output.go:61,80`
    (`violationCount := len(step.Violations)` → `fmt.Sprintf("  (%d
    violations)", violationCount)`, the human table's per-step line).
    Motivating measured instance (implementer-020-final, CI run
    30389988184, 2026-07-28): `coverage_threshold` rendered `fail  (2
    violations)` where the JSON proves one `severity: error` entry (the
    real coverage shortfall) plus one `severity: warning` entry (a
    coverage-exclusion suppression notice). One blocking problem displayed
    as two. Systemic, not coverage-specific: any step emitting a
    warning-severity violation rides the same blind count. The severity
    distinction is populated correctly where violations are created
    (`requirement_traceability.go:290,307,311`; `status_drift.go:99,103`)
    — only the reporting layer discards it.
    The sibling verdict-level defect is ALREADY FIXED, and this must be
    recorded so nobody re-opens it. ISSUE-100's own Amendment section
    describes a second defect where the policy layer blocked on a
    `severity: warning` violation (CI run 30395875188), and states it was
    split into PLAN-ISSUE-020's lane with a placeholder to cite the commit
    "once it lands." Backlog-pm verified it landed: `pkg/gate/policy.go:73`
    now reads `return !strings.EqualFold(strings.TrimSpace(v.Severity),
    "warning")` — the policy layer is severity-aware, and the SARIF-level→
    `Violation.Severity`→verdict mapping is locked end to end by
    `cmd/backstop/pack_severity_contract_test.go` (file confirmed present).
    The founder ratified the governing contract the same day: severity is
    a pack-author contract, `level: warning` from any pack is non-blocking
    by contract. PLAN-ISSUE-020 is `completed`. Therefore ISSUE-100's
    remaining scope is the renderer/tally half ONLY (`result.go:225`,
    `output.go:61,80`); it has no in-flight coverage — the lane that fixed
    the verdict half is closed.
    Recommended fix shape per the issue is (b) render-by-severity: keep
    `Violations` as the single carrier (no shape change, no schema bump)
    and split by `Severity` at the two counting sites — `total_violations`
    counts `severity == "error"` (or gains a companion `total_warnings`),
    and the human line becomes something like `(1 blocking, 1 notice)`.
    Option (a) was NOT recommended, and the reason should stay on record:
    moving warnings into a separate `Notices` slice is a shape change
    touching every warning-emitting step, the JSON schema, and
    `pkg/gate/baseline.go:218`, which currently walks `Violations`
    including warnings for fingerprinting.
    Housekeeping the planner should not have to discover: ISSUE-100's
    Amendment still carries the literal text "*Placeholder: cite the
    PLAN-ISSUE-020 commit that fixes the policy-layer severity blindness
    here once it lands.*" That placeholder is now satisfiable and should be
    filled via issue-author — flagged as an issue-file correction,
    explicitly NOT part of this directive's scope.
    This is the SEVENTH member of the gate-verdict-honesty cluster
    (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092, ISSUE-093, ISSUE-097,
    ISSUE-100) — a surface that reads authoritative and silently isn't.
    Per the established pattern in this directive's Notes, ISSUE-066/067/
    091 remain cited by NO directive and the cluster's home is a founder
    decision still pending; slotting ISSUE-100 here does not pre-empt it.
11. **`gate` cannot emit the human table and JSON in one run — a measured
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
12. **Warning-only coverage step reads as pass (ISSUE-107).**
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
13. **Substantiveness JOIN discards a pack-declared severity (ISSUE-106).**
    The gate-side half of the SPEC-037 substantiveness pack split
    (`pkg/gate/substantiveness_join.go`) throws away a pack's declared
    severity one hop past where ISSUE-104/ISSUE-105 closed the same gap
    elsewhere. Q1 dispatch preserves it correctly —
    `Severity: nonEmptySeverity(v.Severity)` at
    `pkg/gate/substantiveness_q1_dispatch.go:71`, where `nonEmptySeverity`
    (`:110-113`) only defaults an EMPTY value to `"error"` and leaves a
    declared `"warning"` intact (both confirmed present). The join then
    overwrites it at two sites, both confirmed hardcoding
    `Severity: "error"`: `HollowFindingsToViolations`
    (`substantiveness_join.go:184`) on every routed hollow finding, and
    `NoTargetViolation` (`substantiveness_join.go:68`), the noTarget
    set-join decision table. Consequence: a pack declaring a
    substantiveness rule at `level: warning` — an advisory by the
    founder-ratified severity contract — blocks the gate anyway. Same
    "loud ≠ blocking" violation ISSUE-104 (SARIF severity descriptor
    fallback, commit `a42b065`) and ISSUE-105 (step verdicts
    severity-aware without a policy entry, commit `d7d777c`) closed at the
    parse and verdict layers; both commits are confirmed in HEAD's
    lineage, and both fixes BYPASS this converter — the join runs
    downstream of `StepVerdict`'s severity-aware sites and hands it
    violations already flattened to `"error"`.
    The two sites are NOT the same fix, and a plan must not treat them
    uniformly. `HollowFindingsToViolations` is a 1:1 conversion (one
    finding in, one violation out), so forwarding `v.Severity` instead of
    the literal is a direct substitution needing no extra defaulting —
    `nonEmptySeverity` upstream already guarantees non-empty.
    `NoTargetViolation` converts no single input finding at all: it fires
    on a SET-MEMBERSHIP test over `ReferencedSymbolSet`
    (`substantiveness_join.go:26`), a `map[string]bool` carrying presence
    only and no severity anywhere. So there is no severity to forward, and
    resolving it requires an actual decision — either a new channel for
    the rule to declare the noTarget severity, or a ruling that a
    gate-SYNTHESIZED violation keeps a fixed severity by design because it
    is a gate-computed defect rather than a pack-tunable advisory. That
    decision must be stated explicitly in whatever plan lands, not left
    implicit.
    Test coupling the planner must not discover the hard way:
    `TestClass3Sites_ViolationsAreErrorSeverityByConstruction`
    (`pkg/gate/step_verdict_severity_test.go`, SITE 2 block at
    `:282-308`) currently LOCKS the defect — backlog-pm read it directly:
    it feeds `HollowFindingsToViolations` a finding with
    `Severity: "warning"` and asserts the output is `"error"`, with an
    inline comment naming this issue as "SIBLING 1 … Filed, not fixed
    here." When the fix lands that test's premise INVERTS and must be
    deliberately rewritten to assert preservation; a green test suite
    would otherwise mean the old, now-wrong behavior is still locked. The
    same test's SITE 1 and SITE 3 guards (`waiverDiagToViolation`,
    `StepTestVerificationScopedFunc`) are genuinely warning-free by
    construction and are not in scope. Blast-radius discipline per the
    PLAN-ISSUE-020 precedent this family follows: measure substantiveness
    step verdicts on the dogfood run and at least one fixture consumer
    before and after; every flip must fit "was severity-blind-overwriting
    a declared `warning`, should never have blocked."
    Scope/priority: `type: bug`, `scope: contained`, `uncertainty: known`,
    `risk: moderate`. It rides here on thematic fit and does NOT displace
    ISSUE-092 (the `risk: critical` active false-green in the
    pack-ecosystem gate) within this directive.
14. **The contract-signature carrier cannot represent a pack-declared
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
15. **Missing engine tool + no CrashGuard yields silently empty SARIF, a
    vacuous `pack_engines` pass, and misleading downstream join violations
    (ISSUE-112).** A findings engine whose tool is ABSENT from `PATH` fails
    in the worst possible way when its binding carries no `CrashGuard`: the
    runner's non-fatal `runErr` is discarded, the empty stdout flows through
    convert (jq over empty stdin emits nothing), the lenient SARIF parse
    reads zero findings, and `pack_engines` PASSES — while every downstream
    consumer of that evidence is lied to. Two aggravators, both named in the
    issue. First, the assume-present fail-loud in
    `cmd/backstop/pack_gate_provision.go` EXEMPTS provision-declared tools
    as "auto-provisioned," but provision is a TRUST ALLOWLIST PIN ONLY — no
    code path installs anything, so provision-pinned tools (ast-grep,
    semgrep) get NEITHER an install NOR a presence check. Second,
    non-CrashGuard engines treat every non-zero/failed run as finding-free
    (the `runErr` is discarded), so an exec-not-found error is
    indistinguishable from a clean scan — even though `pkg/packval`'s
    executor already fails loud on exactly this error class, giving the fix
    an in-tree precedent to copy rather than invent. Direction:
    presence-check provision-pinned tools exactly like assume-present ones
    (fail loud, naming the tool and the install expectation, or implement
    provisioning); make any `*exec.Error`-class failure (binary could not
    start) fail loud for EVERY engine regardless of `CrashGuard`.
16. **Classification matching zero test files should refuse loudly instead
    of fabricating mass join violations (ISSUE-113).** When a pack's
    classification globs match ZERO test files, the substantiveness join
    silently emits a "does not call package X" violation for EVERY mandated
    test — hundreds of misleading findings whose real cause (empty
    classification) is named nowhere. Direction: extend the ISSUE-020
    config-error refusal philosophy already delivered under this directive —
    when mandated tests exist but the classifier matches zero test files (or
    the substantiveness evidence set is empty while mandated tests exist),
    the step REFUSES with a config-error naming its cause instead of
    emitting per-test violations.

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

ISSUE-092 (backlog-pm slotted, 2026-07-27) does NOT ride along at the
low-urgency tier the paragraphs above describe for ISSUE-007/082/075/077.
It is `risk: critical`, it is an active false-green in the tool that gates
the entire pack ecosystem, and it is live today for every pack in the
fleet. There is also a cross-directive sequencing dependency worth
recording: DIR-027 (position 3 in BACKLOG.yml, ahead of DIR-024 at position
4) has a thread 5 and an acceptance criterion requiring every published
pack repo to run `backstop pack check` + `backstop pack test` in CI on
every push. While ISSUE-092 stands, that criterion is satisfiable with a
vacuous signal — the fleet-wide CI gate DIR-027 is buying would be green by
construction. ISSUE-092 should therefore be sequenced ahead of DIR-027
thread 5's completion, notwithstanding DIR-024's own position 4 — recorded
here as a sequencing note, not as a backlog reorder; the PM has proposed
the coupling to the founder and no reorder is authorized. Within this
directive, ISSUE-092 is the one source that legitimately competes with
ISSUE-020 for first pick, on the strength of the DIR-027 coupling above —
that is a founder call, not decided here. Finally, ISSUE-092 is a sibling
of the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091) —
same failure family, a verdict surface that reads authoritative and
silently isn't — and ISSUE-091 and the rest of that cluster are currently
cited by no directive; that home decision is pending the founder and is
not being slotted here.

ISSUE-093 (backlog-pm slotted, 2026-07-28) is `risk: moderate`,
`scope: contained`, `uncertainty: known` — it does NOT carry ISSUE-092's or
ISSUE-020's urgency and must not displace either within this directive. It
rides along on thematic fit. It is the FIFTH member of the
gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092,
ISSUE-093) — the family where a gate verdict surface reads authoritative
and silently isn't. ISSUE-066, ISSUE-067 and ISSUE-091 remain cited by NO
directive; whether that cluster gets its own directive is a founder
decision still pending, and slotting ISSUE-093 here does not pre-empt it —
if the founder creates a cluster directive, 092 and 093 move there
together. A second, smaller pattern worth naming for the founder: the
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

ISSUE-097 (backlog-pm slotted, 2026-07-28) joins the gate-verdict-honesty
cluster named above (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092,
ISSUE-093) — the sixth member sharing the same failure shape: a check
reports clean because of what it cannot see, not because nothing is wrong.
It has no in-flight coverage today, and that is documented rather than
assumed: `plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml` names
both stale tokens under a "PRE-EXISTING, NOT YOURS TO FIX" block and states
"Do NOT fix them in this plan … ISSUE-097 owns them" — because re-keying a
waiver changes which findings it suppresses, and that is a scope change
dressed as a typo fix, not something to absorb incidentally into another
plan. Intra-directive sequencing: part (a) (re-key or remove the two
tokens) is a two-line mechanical fix, but whoever does part (b) (the
harvest-blindness fix) should do (a) in the same change — fixing (a) alone
leaves the blind spot that produced it in place for the next rename.
Neither part competes with ISSUE-020 (the launch-blocking Linux/CI thread)
for priority; nobody should work this directive strictly top-down. Causal
link, so the fix lands in the right place: this is rename fallout from
DIR-027's fleet migration (the 2026-07-27 `backstop/self` →
`backstop-ai/backstop-self` rename), but DIR-027 owns publication,
migration, and lock-state and explicitly disclaims mechanism design — the
durable fix here is making an unbound waiver visible to the gate, which is
gate/engine quality, so it stays homed in this directive rather than
DIR-027.

ISSUE-100 and ISSUE-099 (backlog-pm slotted, 2026-07-28) were both slotted
under the standing clear-fit grant. Both are `scope: contained`,
`uncertainty: known`, `risk: safe` — the lowest-risk pair this directive
holds. Neither may displace ISSUE-092 (the `risk: critical` active
false-green in the pack-ecosystem gate) within this directive. Why DIR-024
and not elsewhere, for ISSUE-099 specifically: DIR-001 (Release Workflow)
owns `ci.yml` and is where the measured cost is PAID, but the fix site is a
gate CLI flag in `cmd/backstop/gate.go` — gate/engine surface, this
directive's theme. DIR-001 has no charter claim on gate output flags. The
precedent is this directive's own: ISSUE-082, ISSUE-077 and ISSUE-007 all
ride here on thematic fit as non-correctness gate/engine items. The two are
siblings in the same reporting-layer neighborhood but independent defects:
100 is what the numbers say, 099 is how many runs it costs to get them.
Fixing either does not close the other.
A structural observation for the founder, stated as observation and NOT as
a claim on scope: with ISSUE-020 now `closed` (delivered 2026-07-28), this
directive's lede still says "Two gate/engine-quality gaps" while it
enumerates ELEVEN, and the one source that justified its BACKLOG.yml
position — the Linux/CI launch blocker — is delivered. Every remaining
source is tier-2 except ISSUE-092. Whether DIR-024 should now be split,
re-scoped, or repositioned is a founder call that backlog-pm has raised
separately and has NOT acted on.

ISSUE-107 was slotted by backlog-pm 2026-07-29 as a clear fit; this does
NOT pre-empt the still-pending founder decision on whether the
gate-verdict-honesty cluster gets its own directive. If that directive is
created, all DIR-024-slotted cluster members move together.
Provenance: ISSUE-107 was filed BY `PLAN-ISSUE-105` TASK-006 as a
deliberately-deferred residual, not discovered independently.
PLAN-ISSUE-105 is `status: completed` and its CLASS-2 inventory names this
exact loop, explaining the deferral: fixing it introduces a `"warning"`
status where consumers currently see `"pass"`, which is a reporting change
that belongs with ISSUE-100's renderer half rather than inside a verdict
fix. THE SEQUENCING CONSEQUENCE: ISSUE-107 and ISSUE-100 (already cited
here) touch the same surface and should be planned together or 107 after
100 — planning 107 alone means shipping a new `"warning"` state into a
renderer that ISSUE-100 says still miscounts warnings as violations.
Its sibling from the same audit — ISSUE-108 (contract carrier drops pack
severity) — remains cited by NO directive as of this writing, as are
ISSUE-104 and ISSUE-105 (both closed). Correction in place rather than a
stale claim: this paragraph originally also named ISSUE-106 as cited by no
directive; ISSUE-106 is now slotted into this same directive, immediately
below — see that paragraph for detail. 107 is the only family member that
inverts the failure direction.

ISSUE-106 (backlog-pm slotted, 2026-07-29) was slotted under the standing
clear-fit grant. It is the EIGHTH member of the gate-verdict-honesty
cluster (ISSUE-066, ISSUE-067, ISSUE-091, ISSUE-092, ISSUE-093, ISSUE-097,
ISSUE-100, ISSUE-106) — the family where a verdict surface reads
authoritative and silently isn't. ISSUE-066/067/091 remain cited by NO
directive and the cluster's home is a founder decision still pending;
slotting ISSUE-106 here does not pre-empt it — if a cluster directive is
created, every DIR-024-slotted member moves together.
Within that cluster a tighter SUB-FAMILY has now formed, and it is the
useful unit for sequencing: the pack-severity contract chain, filed hop by
hop as each fix exposed the next carrier. Hop 1 ISSUE-104 (parse: SARIF
severity falls back to the rule descriptor, `a42b065`), hop 2 ISSUE-105
(verdict: step verdicts read severity without a policy entry, `d7d777c`) —
both landed. Hop 3 ISSUE-106 is this item. Two further residuals from the
same implementer-105 audit (2026-07-29) were filed alongside this one and
are now ALSO cited by this directive, slotted by concurrent backlog-pm
passes the same day: ISSUE-107 "Coverage Warning-Only Step Reads As Pass"
(item 12) and ISSUE-108 "Contract Carrier Drops Pack Severity" (item 14).
Recorded so nobody plans ISSUE-106 believing it is the last hop: whoever
fixes the join should expect the same question ("what severity does a
SYNTHESIZED, non-1:1 violation carry?") to recur in the coverage and
contracts carriers, and answering it once, generally, may be cheaper than
three site-local substitutions. That generalization is a founder/planner
call, not decided here.
Relationship to ISSUE-100 (item 10 above), so the two are not conflated:
100 is the RENDERER half (tallies count warnings as violations) and 106 is
a CARRIER half (severity destroyed before the renderer or the verdict ever
sees it). They are independent — fixing either does not close the other —
and 106 sits upstream of 100 in the data flow.

ISSUE-108 (backlog-pm slotted, 2026-07-29) came in under the standing
clear-fit grant alongside ISSUE-107, and the two must be read as one arc
rather than two tickets. It is `type: bug`, `scope: contained`,
`uncertainty: known`, `risk: moderate`; it does not compete with ISSUE-092
(the `risk: critical` active false-green in the tool that gates the whole
pack ecosystem) for first pick within this directive. It is the last of
four residuals from the pack-severity contract family: ISSUE-104 (SARIF
severity lost at the parser — fixed, `a42b065`), ISSUE-105 (step verdicts
ignore severity absent a policy entry — fixed, `d7d777c`), ISSUE-106 (the
substantiveness join discards a severity that exists upstream), ISSUE-107
(the coverage step's warning-only finding set reads as pass), and this
one. In-flight coverage is nil and that is established from artifacts, not
assumed: `PLAN-ISSUE-105` is `completed`, and the session that completed it
filed 106/107/108 together at its closure (commit `21e47ed`) as explicit
hand-offs. Two corpus-honesty items ride along and are NOT this
directive's scope, raised to the founder separately: ISSUE-104 and
ISSUE-105 are both still `status: open` despite their fixes having landed
(and `PLAN-ISSUE-104` is still `draft`), and no directive cites ISSUE-104,
ISSUE-105, ISSUE-106 or ISSUE-108 today. Finally, per the standing pattern
in the paragraphs above: slotting ISSUE-108 here does NOT pre-empt the
founder's pending decision on whether the gate-verdict-honesty cluster gets
its own directive — if it does, this issue and the other DIR-024-slotted
members move there together.
Correction in place rather than a stale claim (verified against the corpus
2026-08-02, not asserted): as of this writing ISSUE-104 and ISSUE-105 are
both `status: closed`, and `PLAN-ISSUE-104` and `PLAN-ISSUE-105` are both
`status: completed` (commit `87b12cf`, "close(ISSUE-104,105): severity-
contract hops delivered — plans completed, issues closed"); ISSUE-106,
ISSUE-107 and ISSUE-108 all now appear in this directive's own frontmatter
`source:` list, immediately above. The one fragment of the original
sentence that remains true: ISSUE-104 and ISSUE-105 are still cited by NO
directive's frontmatter `source:` — they are named only in this file's
prose.

ISSUE-112 (slotted under the standing clear-fit grant, 2026-08-02) is a NEW
member of the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091,
ISSUE-092, ISSUE-093, ISSUE-097, ISSUE-100, ISSUE-106) — the family where a
verdict surface reads authoritative and silently isn't. With ISSUE-112 and
ISSUE-113 the cluster now numbers TEN members. As with every prior slotting
in this lineage, this does NOT pre-empt the still-pending founder decision
on whether the cluster gets its own directive — if one is created, every
DIR-024-slotted member moves together.
Provenance is a FIRST-CONSUMER discovery, not a dogfood one: observed live
in `bclabs-portal`'s first CI run on a GitHub runner, 2026-07-29 — ast-grep
absent from `PATH` produced empty SARIF, `pack_engines` went green, the
`test_substantiveness` join starved, and 397 false "does not call package"
violations landed on innocent tests. Diagnosis took hours because nothing
named the missing tool. A portal-side workaround shipped (explicit gitleaks
+ ast-grep installs in its CI workflow), so the filer is the blocked
consumer working around the defect — in-flight coverage is nil by
construction.
The sharpest fact for whoever plans it, and it is a fleet-level asymmetry
rather than one bug: the assume-present fail-loud in
`cmd/backstop/pack_gate_provision.go` EXEMPTS provision-declared tools as
"auto-provisioned," but provision is a TRUST ALLOWLIST PIN ONLY — no code
path installs anything. So provision-pinned tools (ast-grep, semgrep)
receive NEITHER an install NOR a presence check. The second half matters
too: `pkg/packval`'s executor already fails loud on an exec-not-found
error, but the gate dispatch path does not — the two paths disagree about
what a missing binary means, so the fix has an existing in-tree precedent
to copy rather than invent.
Relationship to ISSUE-092 (`pack test` phase-3 fixtures cannot fail), which
this directive already cites as its `risk: critical` first pick: both are
false-green, but they sit at different layers — ISSUE-092 is the
PACK-AUTHORING gate going vacuously green, ISSUE-112 is the CONSUMER gate
going vacuously green. Fixing either does not close the other.

ISSUE-113 (slotted under the standing clear-fit grant, 2026-08-02) carries
an explicit founder ack recorded in the issue body: "Founder-ack'd (Brandon,
2026-07-28) for slotting per PM flow (DIR-024 recommended)." This slot
therefore executes a ruling rather than proposing one.
It is the diagnosability sibling of ISSUE-112, and the two share one
observed signature with two different root causes — hit twice in one week
by `bclabs-portal`: (1) the published `typescript-substantiveness` 1.1.0
shipping harness-baked classification globs, 397 false violations — staged
in PM records as "ISSUE-102" but note that no `ISSUE-102` artifact exists
in `issues/` as of this writing (verified: absent from the working tree and
from git history alike; the PM inbox records that slot as decided but never
applied; this flag is now RESOLVED, not outstanding — see the dead-ID
paragraph immediately below for the founder ruling) — and (2) the
missing-ast-grep case, which is ISSUE-112. Both cost
hours; both would have been one line of output: "classification matched 0
test files".
Its direction explicitly EXTENDS the ISSUE-020 config-error refusal
philosophy already delivered under this directive — when mandated tests
exist but the classifier matches zero test files, the step REFUSES with a
config-error naming its cause instead of emitting per-test violations. That
makes it a continuation of shipped work in this directive, not a new
mechanism.
Sequencing note for a planner: ISSUE-112 and ISSUE-113 should be read as one
arc and planned together, or ISSUE-113 immediately after ISSUE-112.
ISSUE-112 makes the missing tool NAME ITSELF at the point of failure;
ISSUE-113 makes the resulting empty evidence set REFUSE instead of
fabricating per-test violations. Shipping only ISSUE-113 would convert one
misleading failure mode into a different one without ever naming the absent
binary; shipping only ISSUE-112 leaves the harness-baked-globs root cause
(the never-filed, founder-ruled-dead "ISSUE-102" — see the dead-ID
paragraph below) still producing the same silent mass-violation signature.

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
evidence of a lost or deleted file. ISSUE-113 (this same item, cited above)
partially recaptures dead ISSUE-102's substance — the harness-baked
classification-globs defect — and is the live home for that concern. Dead
ISSUE-103's substance, `typescript-contracts` rejecting the bare-const
variable-contract idiom as observed in `bclabs-portal`, is NOT recaptured
anywhere and is NOT this directive's scope; if it resurfaces it needs a
fresh ID (never 103), and DIR-022 ("Contracts Engine Hardening") is where it
was reasoned to belong.
