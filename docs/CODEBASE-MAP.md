# backstop-core — codebase map

Navigational context, auto-loaded via CLAUDE.md. Generated 2026-07-25 by a
full-repo sweep; regenerate (ask a session to re-explore and rewrite this
file) after structural changes. Invariants/behavior live in CLAUDE.md — this
file is only WHERE things are.

> README correction: library code is NOT in `internal/` (empty dir) — it all
> lives under `pkg/`.

## Entry points

- `cmd/backstop/main.go:8` — main; maps `*ExitCodeError` → exit codes
  (0 pass / 1 violations / 2 config).
- `cmd/backstop/root.go:20` `NewRootCommand()`; all subcommands wired at
  `root.go:144`. gate → `gate.go:33`; pack subcommands in `pack_*.go`;
  artifact in `artifact_*.go`; baseline `baseline.go:24`; waiver
  `waiver.go:26`; `commands` (agent-discovery JSON) inline `root.go:126`.

## Package map

- `cmd/backstop/` — CLI layer + gate step wiring (`gate.go`, `pack_gate.go`,
  `pack_gate_provision.go`). Test-heavy (~90 files, mostly tests).
- `pkg/gate/` — CORE: gate kill-chain. `Gate` orchestrator (`gate.go`),
  `StepFunc`/`StepResult`/`GateResult` (`result.go:135,169`), step impls
  (`step_*.go`), scope, baseline compare, policy, traceability/drift.
- `pkg/pack/` — CORE: manifest model + `engines:` parsing (`manifest.go`),
  coordinates.
  - `pkg/pack/distribution/` — pack lifecycle: `add/install/update/upgrade/
    remove/relock.go`, `lockfile.go`, `hash.go`, `provenance.go`,
    `config_merge.go`, `tamper.go`.
  - `pkg/pack/engine/` — engine bindings (`binding.go`), `GateType`
    (`gatetype.go`), pinned tool versions (`allowlist.go` — semgrep 1.96.0).
- `pkg/check/` — findings execution + SARIF (`runner.go`, `parsers.go:42`
  `parseSarif`), coverage channel (`coverage.go`).
- `pkg/waiver/` — `@waiver:` grammar (`waiver.go`), adjudication
  (`adjudicate.go:86` — byte-scans finding line + line above; zero-baked, no
  comment lexer), non-waivable policy (`policy.go`).
- `pkg/packval/` — `pack check`/`pack test` pipeline (phases 1–6: structural,
  coherence, fixtures, archetype, layer, risk-class).
- `pkg/validate/` — per-artifact-type semantic validators (spec/plan/issue/
  adr/bundle/directive/contracts + traceability refs).
- Support: `pkg/config/` (backstop.yml loader), `pkg/schema/` (artifact
  JSON-schema load/resolve), `pkg/artifact/` (frontmatter parse),
  `pkg/scaffold/` (ID/slug alloc, git tag ops), `pkg/baseengines/`
  (embedded base-engines pack).

## Gate flow (`backstop gate --all`)

`cmd/backstop/gate.go:57 runGate` → config + scope (`--all` →
`GateScopeModeAll`, `gate.go:85`) → `buildGateSteps` (`gate.go:617`; loads
installed packs, derives SourceClassifier/TestNameMatcher) → ordered steps
(`gate.go:729-747`, reorder `gate.go:846`):
`pack_lock_verification → artifact_validation → pack_engines →
test_verification → substantiveness → coverage → contracts → status_drift →
requirement_traceability → waiver_resolution → baseline_comparison →
ledger_integrity` (+ toolchain warn step when none declared, `gate.go:605`).
`Gate.Run` (`pkg/gate/gate.go:132`) executes, swaps waiver result
(`gate.go:149`) and baseline (`gate.go:155`), applies policy (`gate.go:168`).
pack_engines: `packValidatorStep` (`gate.go:788`) → `provisionEngines` →
`dispatchPackEngines` (`pack_gate.go:267`) → `runFindingsEngine` → SARIF.

## Pack lifecycle (`pack add <ref>`)

`cmd/backstop/pack_add.go` → `buildAddCommand` (`pack_wiring.go` — assembles
`ExecGitCloner`/`PackvalValidator` via fail-closed positional constructors,
SPEC-055) → `AddCommand.Run` (`add.go`). Free `distribution.Add/Install/
Update/Upgrade` are DELETED; commands exist only via `New*Command`
constructors that error naming any nil dependency (`command.go`).
LOCAL path (`isLocalPath`): read pack.yml in place → `source_type:"local"`,
project-relative `local_path` in lock. REMOTE `org/pack@version`
(`parsePackRef`): `resolveGitURL` → `https://github.com/<name>.git` →
`ExecGitCloner.Clone` (`gitcloner.go`) — real shallow tag clone, root
`.git` STRIPPED before return so remote hashes are reproducible; validation
runs UNCONDITIONALLY through the same `pkg/packval` pipeline `pack check`
uses (`validator.go`). Common tail: copy → `.backstop/packs/<org>/<name>/`
→ `ComputeContentHash` (`hash.go:17`) → config merge/provenance →
`updateBackstopYml` → lockfile → ensureGitignore. Update/upgrade resolve
versions via `TagVersionResolver` (`versionresolver.go`, strict semver over
remote tags); upgrade scans BEFORE any consumer mutation.

### Known gap — `pack relock` arg-shape asymmetry (ISSUE-074, residual)

The SILENT half of ISSUE-074 is FIXED (SPEC-055 REQ-011: relock failures
print their diagnostic to stderr). The residual: `relock` still takes a
filesystem PATH arg (`Use: "relock [path]"`, `pack_relock.go:13`) while
remove/update/upgrade take a pack NAME — guessing wrong now errors loudly
instead of silently, but the asymmetry itself is unresolved (identity/
migration seed territory, where relock gains its migration role).

### Resolved 2026-07-26 — remote pack resolution (was ISSUE-073)

SPEC-055 delivered the production remote path: `ExecGitCloner`
(`gitcloner.go`, real Clone + remote ListTags, root `.git` stripped for
reproducible hashes), unconditional validation (`validator.go`), strict-
semver resolution (`versionresolver.go`), and fail-closed constructors
(`command.go`) that make missing dependencies unconstructable — the old
nil-deref panic class is unrepresentable. Wiring in `pack_wiring.go`;
hermetic remote E2E via `GIT_CONFIG_GLOBAL` insteadOf redirect
(`hermetic_remote_harness_test.go` + `testdata/hermetic-remote/`), incl.
an add→fresh-clone-install round trip asserting hash EQUALITY. Remote
installs are no longer test-mock-only.

## Engines

pack.yml `engines:` → `Manifest.Engines` (`manifest.go:29,68`) →
`EngineBinding` (`manifest.go:296`). Capability presence keyed on declared
`gate_type` (`gate.go:369`), never pack name. Registry = pack bindings over
embedded base-engines defaults (`pkg/baseengines`, root `embed.go`).
InputModes: rule-flags (semgrep `--config` per rule file), config-file,
pattern-arg, none (sandbox validator). Provisioning: backstop-introduced
tools (semgrep/ast-grep) auto-provision at pinned versions
(`allowlist.go:22`); Layer-0 tools (go, golangci-lint) must be on PATH else
exit-2. Coverage is a separate channel: `dispatchPackCoverage`
(`pack_gate.go:339`) → `ParsePackCoverage`, routed on `GateType==coverage`.

## Config & schemas

backstop.yml: schema `artifacts/backstop-yml/v1/schema.json`, loader
`pkg/config/config.go`. backstop.lock: `distribution/lockfile.go`
(`LockEntry{name,version,git_ref,content_hash,source_type,install_date,
local_path}`). Artifact schemas embedded via root `embed.go` (`SchemaFS`,
`//go:embed all:artifacts`); on disk `artifacts/<type>/v<N>/schema.json`.

## Tests

Colocated `*_test.go`; e2e fixture projects under `cmd/backstop/testdata/`
(each with own `.backstop/packs/`), `pkg/gate/testdata/` (waiver-e2e,
substantiveness, traceability, contracts), `pkg/pack/distribution/testdata/`
(tamper fixtures), `pkg/packval/testdata/`. Run: `make test` (race),
`make lint`, `make coverage` (90% threshold), `make ci`.

## Repo-root oddities

- `embed.go` (package backstopcore): go:embed can't reach up, so schemas +
  base-engines pack embed from root (`SchemaFS`, `BaseEnginesFS`).
- `capabilities/`, `directives/` (DIR-001..025), `recipes/`, `standards/`,
  plus `specs/plans/issues/bundles/adrs/` — the repo governing itself with
  its own artifact vocabulary; backstop-core is its own test corpus.
- `packs/` at root: source of the three locally-installed packs the lock
  references; `./backstop` prebuilt binary + `bin/`.
