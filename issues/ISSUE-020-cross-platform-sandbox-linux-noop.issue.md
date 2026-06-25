---
title: "Linux sandbox is a hard error — pack convert scripts and validators run fully unsandboxed on Linux"
schema_version: issue/v1

issue:
  id: ISSUE-020
  title: "Linux sandbox is a hard error — pack convert scripts and validators run fully unsandboxed on Linux"
  type: bug
  status: open
  created: "2026-06-21"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: critical
---

# ISSUE-020: Linux sandbox is a hard error — pack convert scripts and validators run fully unsandboxed on Linux

## Problem

### The missing implementation

`pkg/packval/sandbox.go` implements two sandbox entry points used by all pack-supplied
arbitrary code:

- `SandboxedRun` (line 62) — used by `DefaultExecutor.RunValidator` in
  `pkg/packval/executor.go:46` to run a pack's exit-code sandbox-validator script.
- `SandboxedRunStdout` (line 93) — the clean-stdout variant used by the convert step
  (engine native output → SARIF, REQ-007/REQ-009/CLM-065 in SPEC-035).

Both functions dispatch on `runtime.GOOS`. The `darwin` branch wraps the command under
`sandbox-exec` with a least-privilege profile (deny-default, deny network, deny
file-write, allow read of packDir + system library paths). The `linux` branch is:

```go
// SandboxedRun, lines 75-77:
case "linux":
    return nil, errors.New("sandbox unavailable on linux in this build")

// SandboxedRunStdout, lines 111-113:
case "linux":
    return nil, errors.New("sandbox unavailable on linux in this build")
```

On Linux both functions return an error immediately without executing the command.
The caller in `DefaultExecutor.RunValidator` (`pkg/packval/executor.go:46`) treats
any error from `SandboxedRun` as a validation failure with `ExitCode: 1`. The caller of
`SandboxedRunStdout` in the convert step likewise propagates the error upstream as a
hard convert failure.

The result on Linux: pack validators always fail and convert scripts never execute —
the gate cannot function on Linux at all once these code paths are reached.

### A second unsandboxed convert path

Beyond `packval`, there are two additional sites in `pkg/gate/` that run pack convert
scripts via raw `exec.Command` with no sandbox wrapper on any platform:

- `pkg/gate/substantiveness_q1_dispatch.go:89-99` — `runConvert` shells convert
  scripts directly via `/bin/sh` (used by the Phase-3 integration test harness and the
  Phase-4 strangler-equivalence path).
- `pkg/gate/contract_equivalence.go:257-267` — `runScriptStdin` shells pack
  scripts via `/bin/sh` directly (used by `convertToLocations`, `astGrepProbe`,
  `grepProbe`).

These two sites bypass `SandboxedRunStdout` entirely on all platforms — they are out of
scope for this issue (which is narrowly the Linux no-op) but confirm that the sandbox
surface is not yet consistently applied across the gate, a related gap to note for the
planner.

### Why this is security-elevated: the pack-engine trust boundary

The trusted-tool allowlist (SPEC-035) gates the *tools* a pack may declare — their
name and pinned version are vetted before execution. But that allowlist covers only the
named engine binary (e.g. `semgrep`, `ast-grep`, `golangci-lint`). It does NOT and
CANNOT cover:

- A pack's **convert script** (e.g. `ast-grep/to-sarif.sh`) — an arbitrary shell
  script that transforms engine output to SARIF. Its content is opaque; only its
  presence in the pack's declared `convert:` binding is known at gate time.
- A pack's **sandbox-validator** — an arbitrary exit-code script that runs as part of
  pack validation (`pkg/packval/executor.go:RunValidator`).

The OS-level sandbox is the **only trust boundary** between these arbitrary pack scripts
and the host system. The macOS `sandbox-exec` profile enforces: deny-default, deny
network, deny file-write, with read access scoped to packDir and necessary system
library paths. This makes a malicious or buggy convert/validator unable to exfiltrate
source code, write to the filesystem, or reach the network.

On Linux, this boundary does not exist. Because the Linux branch is a hard error (not a
pass-through), in practice no convert or validator script runs on Linux at all today —
which means the runtime is entirely non-functional on Linux, not merely unsandboxed.
However, once a Linux sandbox is implemented, it must provide equivalent deny-default /
deny-network / scope-filesystem-writes guarantees. Without parity, a pack installed on a
Linux CI host could run its convert script or validator with full ambient permissions:
full filesystem access, full network access, arbitrary process execution.

### Scope relative to ISSUE-029

ISSUE-029 (closed) addressed the macOS facet: the `sandbox-exec` profile was blocking
interpreter dyld reads, breaking convert scripts on macOS. That fix refined and verified
the macOS sandbox. ISSUE-020 is the distinct Linux facet: there is no sandbox
implementation at all on Linux. The two issues share the same interface
(`SandboxedRun` / `SandboxedRunStdout` in `pkg/packval/sandbox.go`) but address
orthogonal operating-system branches.

### Current real-world exposure

Exposure today is low: the repo is pre-launch, local-only, no CI runs on Linux, and
only first-party packs exist. This defect MUST be closed before:

1. CI runs on any Linux host — convert and validator calls hard-error, making the gate
   non-functional.
2. Any third-party or user-supplied pack is installed — the sandbox is the last line of
   defense against untrusted pack scripts on both platforms.

This is cluster G of the thin-executor eradication backlog.

### Candidate Linux sandboxing mechanisms (for the planner — not a decision)

Linux does not have `sandbox-exec`. Candidate mechanisms at varying complexity and
kernel-version requirements:

- **bubblewrap** (`bwrap`) — unprivileged user-namespace container, widely available in
  CI environments; can drop network and bind-mount a minimal read-only filesystem view.
  Closest operational analog to `sandbox-exec`.
- **Landlock** — in-kernel LSM since Linux 5.13; filesystem access control via syscall,
  no external binary required, no privilege needed.
- **seccomp** — syscall filter; composable with Landlock; effective for blocking
  `connect(2)` / `socket(2)` (deny network) and `open(2)` with `O_WRONLY` (deny
  writes).
- **user namespaces + unshare** — full namespace isolation without root; available on
  most modern distros.

Any implementation must:

1. At minimum deny network and scope filesystem writes to the working directory,
   providing parity with the macOS profile's deny-network + deny-file-write guarantees.
2. Leave the macOS `sandbox-exec` path (`darwinSandboxProfile` in
   `pkg/packval/sandbox.go`) unchanged.
3. Present the same `SandboxedRun` / `SandboxedRunStdout` interface so callers in
   `pkg/packval/executor.go` and the convert pipeline require no changes.
4. Fail loudly (not silently pass through) if the chosen sandbox mechanism is
   unavailable on the host kernel — a silent no-op is exactly this defect; an explicit
   error is at least recoverable and auditable.
