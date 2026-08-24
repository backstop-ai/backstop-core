---
title: "Hosted runtimes cannot delegate pack sandboxing to a client-owned external boundary"
schema_version: issue/v1

issue:
  id: ISSUE-185
  title: "Hosted runtimes cannot delegate pack sandboxing to a client-owned external boundary"
  type: enhancement
  status: ready
  created: "2026-08-23"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical

verification:
  level: security
  coverage_threshold: 90
  test_command: "go test ./cmd/backstop/... ./pkg/check/... ./pkg/packval/... ./pkg/gate/..."

implementation:
  summary: >
    Add a gate-local pack-sandbox mode resolved exactly once by the parent before pack loading or
    execution. Native remains the default and retains the existing fail-closed Landlock/seccomp
    and sandbox-exec implementations. An invoking process may explicitly choose external mode by
    --pack-sandbox=external or BACKSTOP_PACK_SANDBOX=external; external mode runs only the
    already-sandboxed convert and validator paths without a nested native sandbox, records that
    native confinement was not applied, and strips passive authorization inheritance from every
    pack-provided child environment through one dependency-neutral environment filter. No
    repository or pack-controlled input may change the running parent's resolved mode.
  package: cmd/backstop, pkg/processenv, pkg/check, pkg/packval, pkg/gate

requirements:
  - id: REQ-001
    text: >
      `backstop gate` MUST expose the gate-local option `--pack-sandbox` and read
      `BACKSTOP_PACK_SANDBOX`. The only accepted values on either surface are the exact,
      case-sensitive strings `native` and `external`. If the flag is omitted and the environment
      variable is absent, the selected mode MUST be `native`. An explicitly present empty value,
      leading/trailing whitespace, different case, or any other value MUST be a config error
      (exit 2) before pack loading or execution; values MUST NOT be trimmed or guessed.
  - id: REQ-002
    text: >
      A present CLI flag MUST override the environment variable, including when the environment
      contains an invalid or empty value. If the flag is absent, a present environment variable
      is authoritative and is validated under REQ-001. The parent process MUST resolve this
      precedence exactly once before loading an installed pack or executing pack-provided code,
      construct one immutable sandbox-runner dependency from it, and MUST thread that dependency
      for the remainder of the gate run without mutable global run state. Child environment or
      arguments MUST NOT mutate or cause the running parent to re-resolve that decision. A child
      can deliberately launch a separate Backstop process with a fresh explicit flag or
      environment value; that new process is a new invocation whose principal core cannot
      distinguish, and preventing it is not a promise of this issue.
  - id: REQ-003
    text: >
      Authorization for `external` mode is core/runtime-only and MAY originate only from the
      invoking process's gate-local `--pack-sandbox=external` flag or
      `BACKSTOP_PACK_SANDBOX=external` environment. `backstop.yml`, any other repository file,
      `backstop.lock`, installed pack content, pack manifests, engine declarations, convert or
      validator output, and child-process environment changes MUST NOT select or widen the
      already-running parent's mode. Pack-authored code remains capable of explicitly constructing
      a separate Backstop invocation with its own CLI/environment authorization; repository data
      does not become authoritative merely because that separate process was launched by a pack.
  - id: REQ-004
    text: >
      `native` MUST remain Backstop's default sandbox mode. The existing macOS sandbox-exec and
      Linux Landlock/seccomp confinement semantics, policy/profile, granted and denied rights,
      default selection, and fail-closed behavior MUST remain unchanged. The implementation MAY
      make only the native-path plumbing changes required by this issue: thread the immutable
      runner selected under REQ-002, install the sanitized pack-child environment required by
      REQ-006, and instrument or signal successful native application for the evidence required by
      REQ-007. Those changes MAY alter function signatures, adapters, command environment setup,
      or helper/result plumbing; they MUST NOT weaken confinement, grant new filesystem/network
      capability, skip native setup in native mode, or change failure into fallback. If the native
      mechanism is unavailable, native mode MUST continue to fail closed before pack-supplied code
      executes; it MUST NOT automatically fall back to `external`, an unsandboxed execution path,
      a warning-only result, or any other mode.
  - id: REQ-005
    text: >
      In `external` mode, the already-defined sandboxed pack code paths MUST execute without a
      nested native sandbox-exec or Landlock/seccomp layer, relying on the client-owned external
      isolation boundary that launched Backstop. This mode changes only how the existing
      `SandboxedRun`/`SandboxedRunStdout` boundary is fulfilled. It MUST NOT broaden, narrow, or
      redefine which execution surfaces are sandboxed: the convert-script and sandbox-validator
      paths established by ISSUE-020 remain in scope, while producer, engine, recipe-transform,
      and other execution governance remains exactly where ISSUE-020 and BUNDLE-021 leave it.
  - id: REQ-006
    text: >
      Before every pack-provided child process is started, core MUST remove all
      `BACKSTOP_PACK_SANDBOX` entries from the child environment, whether the child is an existing
      sandboxed convert/validator or an existing unsandboxed producer, engine, or other
      pack-provided execution surface. A dependency-neutral `pkg/processenv` helper MUST perform
      the filtering so `pkg/check` does not import `pkg/packval` (which already imports
      `pkg/check`). `check.ExecCommandRunner` MUST accept an explicitly supplied environment, and
      every production construction used for pack engine/producer dispatch, coverage,
      substantiveness, contracts, recipe transforms, and pack entrypoint probes MUST receive the
      sanitized environment. Direct pack-child `os/exec` construction MUST also set it: the
      packval engine/producer and scaffold-test paths, native/external sandbox commands and Linux
      helper's final exec, and the contracts pack signature compiler. The sanitization MUST
      preserve unrelated entries and apply in external mode. It prevents PASSIVE inheritance: a
      recursive Backstop launched without adding fresh authorization defaults to native. It does
      not claim to stop pack code from explicitly supplying a fresh flag/environment to a separate
      process.
  - id: REQ-007
    text: >
      Human gate output and the additive `gate/v1` JSON result MUST both report the selected
      pack-sandbox mode and whether Backstop's native sandbox was applied. In external mode they
      MUST unambiguously report mode `external` and native-not-applied; in native mode they MUST
      report mode `native`; `native_sandbox_applied` MUST be true only when at least one native
      sandbox invocation reports that confinement was successfully installed before its child
      execution, and false when no sandboxed path ran or native setup failed before application.
      The immutable runner MUST return this evidence per invocation; dispatch/step results MUST
      carry it and the run-level value MUST be the deterministic OR reduction of those results,
      never a mutable package global or an inference from mode alone. Both renderings MUST read
      the same run-level fields. Emit telemetry only through a telemetry surface that already
      exists; because this repository currently has no telemetry exporter, this issue MUST NOT add
      one merely to report the mode.
  - id: REQ-008
    text: >
      External authorization MUST be non-interactive and explicit. The implementation MUST NOT
      add a confirmation prompt, trust dialog, repository acknowledgment, generic
      disable-security/no-sandbox switch, or authorization spelling broader than the exact
      `external` mode on the two surfaces in REQ-001.
  - id: REQ-009
    text: >
      On a host where Landlock is unavailable, an explicitly selected external mode MUST still
      execute the existing sandboxed convert and validator paths without probing or applying the
      native mechanism. Tests MUST independently observe zero Landlock probe calls and zero native
      restriction-application calls on the external leg, not merely infer bypass from command
      success. The same host in default or explicit native mode MUST probe, refuse, and execute no
      pack-supplied sandboxed code. This hosted-runtime compatibility behavior MUST not weaken
      native fail-closed behavior.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: The absent flag and absent environment select native, and exact native/external values are accepted.
    tests:
      - TestPackSandbox_DefaultsToNative
      - TestPackSandbox_AcceptsExactValues
  - id: CLM-002
    requirement: REQ-001
    text: Empty, whitespace-padded, case-variant, and unknown flag or environment values fail with exit 2 before pack work.
    tests:
      - TestPackSandbox_RejectsInvalidAndEmptyValuesBeforePackLoad
  - id: CLM-003
    requirement: REQ-002
    text: A present CLI value overrides the environment, while an omitted CLI value defers to the environment.
    tests:
      - TestPackSandbox_CLIOverridesEnvironment
      - TestPackSandbox_CLIOverridesInvalidOrEmptyEnvironment
  - id: CLM-004
    requirement: REQ-002
    text: The parent resolves the mode once before pack loading, threads one immutable runner without global run state, and is unaffected by child environment or arguments.
    tests:
      - TestPackSandbox_ResolvesExactlyOnceBeforePackLoad
      - TestPackSandbox_RunningParentUnaffectedByChildEnvironmentOrArgs
  - id: CLM-005
    requirement: REQ-003
    text: Repository and pack-controlled data cannot authorize external mode.
    tests:
      - TestPackSandbox_RepositoryAndPackInputsHaveNoAuthority
  - id: CLM-006
    requirement: REQ-004
    text: Native mode preserves the existing real confinement policy and denial semantics and fails closed without fallback when the mechanism is unavailable.
    tests:
      - TestPackSandbox_NativeUnavailableFailsClosedWithoutFallback
      - TestSandboxSecurityDenialsHold
      - TestSandboxUnavailableMechanismIsLoudError
      - TestSandboxLinux_ProductionPathUsesTheRealABIProbe
      - TestLinuxSandbox_DeniesReadOutsideReadableSet
      - TestLinuxSandbox_WriteInsidePackDirIsDenied
      - TestLinuxSandbox_DeniesNetwork
      - TestLinuxSandbox_ConfinementCarriesIntoTheExecdChild
  - id: CLM-007
    requirement: REQ-005
    text: External mode bypasses only nested native confinement on the existing convert and validator sandbox boundary.
    tests:
      - TestExternalSandbox_UsesExistingConvertAndValidatorPathsWithoutNativeLayer
  - id: CLM-008
    requirement: REQ-005
    kind: absence
    text: External mode does not move producers, engines, transforms, or any other execution surface across the ISSUE-020 boundary.
    tests:
      - TestExternalSandbox_DoesNotRedefineSandboxedSurfaces
  - id: CLM-009
    requirement: REQ-006
    text: A neutral helper strips the authorization variable while preserving unrelated entries, and every current runner-based and direct-exec pack-child construction site supplies that sanitized environment without a pkg/check to pkg/packval dependency.
    tests:
      - TestSandboxAuthorization_StrippedFromEveryPackChildEnvironment
      - TestSandboxAuthorization_SanitizerPreservesUnrelatedEnvironment
      - TestSandboxAuthorization_AllCurrentPackChildConstructionSitesUseSanitizedEnvironment
      - TestSandboxAuthorization_DirectExecPathsUseSanitizedEnvironment
      - TestSandboxAuthorization_NoCheckToPackvalDependencyCycle
  - id: CLM-010
    requirement: REQ-006
    text: A recursive Backstop invocation receives no passive external authorization and resolves native by default unless that separate invocation is given fresh explicit authorization.
    tests:
      - TestSandboxAuthorization_RecursiveBackstopDoesNotPassivelyInheritExternalMode
  - id: CLM-011
    requirement: REQ-007
    text: Human and gate/v1 JSON output report the immutable selected mode and the OR-reduced per-invocation native-application evidence, including false when no native sandbox was applied.
    tests:
      - TestPackSandbox_HumanOutputReportsModeAndNativeApplication
      - TestPackSandbox_JSONReportsModeAndNativeApplication
      - TestPackSandbox_NativeApplicationEvidenceIsDeterministicWithoutGlobalState
  - id: CLM-012
    requirement: REQ-007
    kind: absence
    text: The change does not invent a telemetry exporter where no telemetry surface exists.
    tests:
      - TestPackSandbox_DoesNotAddTelemetryExporter
  - id: CLM-013
    requirement: REQ-008
    kind: absence
    text: No interactive confirmation or generic disable-security switch is introduced.
    tests:
      - TestPackSandbox_NoPromptOrGenericDisableSwitch
  - id: CLM-014
    requirement: REQ-009
    text: External mode runs existing sandboxed paths with zero Landlock probe and native-application calls, while native mode on the same unavailable probe result refuses without child execution.
    tests:
      - TestExternalSandbox_UnavailableLandlockSkipsProbeAndApplication
  - id: CLM-015
    requirement: REQ-004
    text: The permitted immutable-runner, environment-sanitization, and application-evidence plumbing leaves native mode on the same profile, rights, probe/application sequence, and fail-closed execution boundary.
    tests:
      - TestPackSandbox_NativeRunnerPlumbingPreservesConfinementPolicy
      - TestPackSandbox_NativeEnvironmentSanitizationPreservesConfinement
      - TestPackSandbox_NativeEvidenceInstrumentationPreservesProbeAndApplicationOrder

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: resolvePackSandboxMode
        kind: function
        signature: "func resolvePackSandboxMode(flagSet bool, flagValue string, envSet bool, envValue string) (packval.SandboxMode, error)"
    consumes:
      - source: pkg/packval/sandbox.go
        name: SandboxRunner
        kind: type
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/processenv/environment.go
    provides:
      - name: Without
        kind: function
        signature: "func Without(environment []string, names ...string) []string"
  - file: pkg/check/runner.go
    provides:
      - name: ExecCommandRunner
        kind: type
        signature: "type ExecCommandRunner struct { Dir string; Env []string }"
  - file: pkg/packval/sandbox.go
    provides:
      - name: PackSandboxEnvVar
        kind: constant
        signature: "const PackSandboxEnvVar = \"BACKSTOP_PACK_SANDBOX\""
      - name: SandboxMode
        kind: type
        signature: "type SandboxMode string"
      - name: SandboxRunResult
        kind: type
        signature: "type SandboxRunResult struct { Output []byte; NativeSandboxApplied bool }"
      - name: SandboxRunner
        kind: interface
        signature: "type SandboxRunner interface { Mode() SandboxMode; Run(cmd string, args []string, packDir string) (SandboxRunResult, error); RunStdout(cmd string, args []string, packDir string, stdin []byte) (SandboxRunResult, error) }"
      - name: NewSandboxRunner
        kind: function
        signature: "func NewSandboxRunner(mode SandboxMode) (SandboxRunner, error)"
    consumes:
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/packval/executor.go
    consumes:
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/packval/sandbox_linux.go
    consumes:
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/packval/sandbox_linux_helper.go
    consumes:
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/packval/sandbox_nonlinux.go
    consumes:
      - source: pkg/processenv/environment.go
        name: Without
        kind: function
  - file: pkg/gate/result.go
    provides:
      - name: GateResult
        kind: type
        signature: "type GateResult struct { PackSandboxMode string `json:\"pack_sandbox_mode\"`; NativeSandboxApplied bool `json:\"native_sandbox_applied\"`; ... }"
      - name: StepResult
        kind: type
        signature: "type StepResult struct { NativeSandboxApplied bool `json:\"-\"`; ... }"
---

# Hosted runtimes cannot delegate pack sandboxing to a client-owned external boundary

## Problem

Backstop correctly defaults to its own native sandbox for the pack-supplied convert scripts and
sandbox validators covered by ISSUE-020. On Linux, `pkg/packval/sandbox_linux.go` negotiates
Landlock before it constructs the re-exec helper and refuses if the kernel does not provide a
usable ABI. The refusal is deliberate and security-preserving: `resolveLandlockMechanism` and
`newSandboxHelperCommand` execute no pack-supplied command when confinement cannot be installed.
There is no automatic unsandboxed fallback.

That correct default prevents a hosted runtime from using Backstop inside a separate,
client-owned isolation boundary when the inner runtime cannot install Landlock/seccomp. Examples
include constrained containers, remote execution workers, and hosted analysis sandboxes where the
outer service already owns the process/filesystem/network boundary but the nested Backstop
process sees Landlock as unavailable. Today the client has only two outcomes: grant the nested
kernel facilities or receive the native fail-closed error. There is no explicit runtime-only way
for that client to accept responsibility for the boundary.

The missing capability is not evidence that native fail-closed behavior is wrong, and it is not a
request for a repository-controlled security opt-out. The compatibility requirement is a narrow
delegation decision made by the process that launches a single gate run. If repository or pack
content could enable it, an untrusted pack could authorize the removal of its own containment.
Filtering the authorization from pack-child environments prevents passive propagation into a
recursive Backstop invocation. It cannot prove who controls a new process: pack code can
deliberately invoke a separate Backstop with a fresh explicit flag or environment value, and that
separate invocation makes its own decision. The enforceable boundary is the current parent run,
whose resolved mode child environment and arguments cannot mutate.

This is therefore an `enhancement`, not a bug in ISSUE-020: the native implementation behaves as
specified, but the runtime lacks an explicit hosted-execution mode for a boundary supplied by the
invoking client.

## Solution

Add exactly two gate-local modes: `native` and `external`. Native is the default. The parent gate
resolves the exact CLI/environment inputs once, before installed packs are loaded, and threads an
immutable runner dependency through the existing sandbox boundary. A valid CLI flag wins over the
environment; without a flag, a present environment value is validated. Explicit empties and all
non-exact values are config errors. No YAML field, lock entry, repository file, pack manifest, or
pack output participates in resolution. There is no package-level current-mode variable, setter,
or callback that can re-read process state after resolution.

In native mode, preserve the current platform implementation and fail-closed behavior. In
external mode, execute the same convert and validator calls with their existing cwd, argv, stdin,
stdout, stderr, and error contracts but without nesting Backstop's native sandbox. The client that
launched Backstop owns the external boundary. This issue does not move any call site into or out
of the sandboxed set and does not answer BUNDLE-021's broader questions about producer, engine,
transform, publisher, pack capability, or consumer trust governance.

Sanitize `BACKSTOP_PACK_SANDBOX` from every pack-provided child environment, not only the two
sandboxed paths. A generic filter lives below both `pkg/check` and `pkg/packval`; runner-based
callers inject its output into `ExecCommandRunner.Env`, and every direct `os/exec` pack-child path
sets the same output explicitly. This avoids the forbidden `pkg/check` -> `pkg/packval` dependency
cycle and covers the paths that bypass the shared runner. The filter prevents passive recursive
inheritance only; it is not a process-principal or anti-spawn control.

Each immutable `SandboxRunner` call returns its output plus native-application evidence. External
calls always return false and never enter the Landlock probe/application or sandbox-exec
construction. Native calls return true only after native confinement was installed for that
child. Dispatch carries the evidence into its step result, and `GateResult` OR-reduces all step
evidence. This makes `native_sandbox_applied` deterministic even when no sandboxed path ran or
native setup failed, with no mutable global run state. The adapter, sanitized command environment,
and application signal are permitted native-path plumbing; they may change signatures and helper
result flow but may not change the platform profile, rights, setup order, or fail-closed boundary.
The single gate result feeds both human and JSON formatters. No interactive prompt, broad
no-security flag, or new telemetry subsystem is part of the change.

## Verification

The mandated security tests cover the positive compatibility path and its adversarial boundaries:

- exact accepted values, explicit empty/invalid values, and CLI-over-environment precedence;
- resolution exactly once before pack load, an immutable runner, and parent state unaffected by child env/args;
- attempted authorization through config, repository files, installed packs, and manifests;
- passive authorization stripping across every runner/direct-exec pack-child path and recursive invocation;
- external mode with independently observed zero Landlock probe/application calls, paired with native refusal and no fallback;
- the existing real macOS/Linux confinement, denial, ABI-probe, and fail-closed tests, plus targeted tests proving the permitted plumbing leaves those policy semantics intact;
- matching human and `gate/v1` JSON reporting of selected mode and native-not-applied; and
- absence of prompts, generic disable-security controls, and a newly invented telemetry exporter.

The unavailable-Landlock test MUST use separate controlled counters for Landlock probing, native
restriction application, and child execution. The external leg requires probe=0, application=0,
and child>0 through the existing convert/validator boundary. The native leg requires probe>0,
application=0, and child=0. A test that merely observes an exit code or command success is
insufficient. The full package test command deliberately has no `-run` filter: existing
`TestSandbox...` and `TestLinuxSandbox...` confinement tests are mandatory regression evidence,
not incidental tests excluded by a new-name regex.

## References

- `pkg/packval/sandbox.go`: platform-neutral `SandboxedRun` and `SandboxedRunStdout` boundary.
- `pkg/packval/sandbox_linux.go`: Landlock negotiation, re-exec helper, and current fail-closed path.
- `pkg/packval/sandbox_linux_helper.go`: inherited helper-environment filtering before `execve`.
- `pkg/check/runner.go`: shared engine/producer command runner; currently inherits the process environment implicitly.
- `pkg/packval/executor.go`: direct engine/producer and scaffold-test exec paths in pack validation.
- `cmd/backstop/gate.go`: gate-local flags, pack loading in `buildGateSteps`, result stamping, and output selection.
- `cmd/backstop/pack_gate.go`: the current sandbox runner seams and convert/validator call sites.
- `cmd/backstop/recipe_apply.go`: pack-declared transform execution through `ExecCommandRunner`.
- `pkg/gate/result.go` and `pkg/gate/output.go`: the single `gate/v1` result and its JSON/human renderers.
- ISSUE-020: native Linux Landlock/seccomp implementation, fail-closed requirement, and the existing sandboxed surface.
- BUNDLE-021: broader pack-command execution governance; its open questions remain out of scope and unresolved.
