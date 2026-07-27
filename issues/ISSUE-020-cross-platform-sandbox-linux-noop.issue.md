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
  scope: cross-cutting
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

### Acceptance criteria (raised 2026-07-26)

"A Linux sandbox exists" is NOT a sufficient definition of done. The bar is now:

**`backstop gate` runs green in CI on Linux.**

This deliberately subsumes wiring `.github/workflows/ci.yml` to actually invoke
`backstop gate` on `ubuntu-latest`. That wiring is not separable follow-on work — it
cannot be done until this issue is fixed, because pointing the gate at `ubuntu-latest`
today fails immediately (the Linux branch is a hard `errors.New`, see above). Making CI
invocation the acceptance criterion means two things happen together: the fix cannot be
claimed done on the strength of an isolated unit test over `sandbox.go` — it has to be
proven by a real `backstop gate` run, against real installed packs, going green on a
real Linux CI host — and the same change closes the gap (documented below) that let
this defect stay invisible in the first place. Landing a Linux sandbox implementation
without also flipping CI over to call `backstop gate` does not close this issue.

### Newly-verified scope: the blast radius is near-total, not partial (verified 2026-07-26)

Both `SandboxedRun` (pack validators) and `SandboxedRunStdout` (pack convert scripts)
sit behind `resolveSandboxedRunStdout()`, which wraps pack convert scripts on two
dispatch paths in `cmd/backstop/pack_gate.go`:

- the findings/engine path (`pack_gate.go:693`)
- the coverage path (`pack_gate.go:473`) — which additionally treats a `coverage`
  gate-type engine with no declared `convert:` binding as a broken-pack config error
  ("its native profile is not coverage-records and must be normalized"). A coverage
  engine's convert is mandatory, not optional, which makes the coverage dimension
  strictly convert-dependent too.

Essentially every functional pack ships at least one convert script — counts verified
directly against the packs installed/vendored today:

| Pack | Convert scripts |
|---|---|
| `packs/contracts` | 2 (`ast-grep/to-sarif.sh`, `grep/to-sarif.sh`) |
| `packs/substantiveness` | 1 (`ast-grep/to-sarif.sh`) |
| `packs/base-engines` (embedded in the binary itself) | 1 (`ast-grep/to-sarif.sh`) |
| `.backstop/packs/backstop/go-toolchain` | 3 (`scripts/build-to-sarif.sh`, `scripts/test-to-sarif.sh`, `scripts/coverage-to-records.sh`) |
| `typescript-contracts` (separate `backstop-packs` repo) | 2 |
| `typescript-substantiveness` (separate `backstop-packs` repo) | 1 |
| `typescript-toolchain` (separate `backstop-packs` repo) | 5 |

`base-engines` is the sharpest data point: it is embedded directly in the backstop
binary (`embed.go`, `BaseEnginesFS`), so even a bare install with zero user-added packs
cannot check findings on Linux — this is not an edge case reachable only by exotic
third-party packs.

Consequence, stated plainly: on Linux, `backstop gate` today fails every dimension that
dispatches an engine with a convert script (pack_engines/findings, coverage), while the
dimensions that operate purely on committed documents — artifact validation, status
drift, requirement traceability, waiver resolution, baseline comparison, lockfile
verification — still pass, because they never touch a pack convert script. The gate
checks documents and not software. The one mitigating fact, worth stating because it is
the thing that keeps this from being worse: this is a loud, total failure of the
code-checking half of the gate, not a silent or vacuous green. A pack engine that
cannot run reports an error; it does not report a false pass.

### The related CI defect this issue folds in (deliberately not filed separately)

`.github/workflows/ci.yml` does not invoke `backstop` at all today:

- The `gate` job (`runs-on: ubuntu-latest`) runs raw `go tool golangci-lint run
  ./...`, raw `go test -race -coverprofile=... ./...`, and a hand-rolled shell
  threshold check over `go tool cover -func`. None of this is `backstop gate`.
- The only job that touches the binary is `baseline`, which builds `./backstop` and
  runs `./backstop baseline generate` — commented in the workflow itself as "equivalent
  to `./backstop gate --all --json`" — also on `runs-on: ubuntu-latest`. That is the one
  place CI would hit this issue's wall immediately, the moment it actually ran.

The founder's explicit call: this does not get its own issue. It is a one-file YAML
change with no design decisions behind it — filing a separate issue-and-plan for
pointing a CI step at `backstop gate` instead of raw tool invocations would be ceremony.
It is recorded here, and folded into this issue's acceptance criteria above,
specifically so the Linux-sandbox fix cannot be declared done without also being proven
against the real CI path. Do not re-file this as a standalone issue.

Why it stayed invisible until now, worth recording so it isn't mistaken for a recent
regression: backstop-core's own CI does not dogfood `backstop gate` (see above), and —
independently — the workflow file has never actually executed at all, because the repo
has no remote yet (`git remote -v` returns empty on this clone). This is a latent
failure that has never been observed in practice, not one that was missed.

### Launch relevance (escalated 2026-07-26)

The founder escalated this issue to the same tier as the two other current launch
blockers: recipes (SPEC-054) and remote pack consumption (ISSUE-073). The functional
argument: a consumer cannot enforce backstop in CI — the only place enforcement
actually matters for a team, as distinct from a single local machine — until this is
fixed. Today the product works on the founder's Mac and nowhere else, which is not
shippable. This also blocks the client-portal traceability feed, which renders gate
JSON that CI is expected to produce: with CI unable to run `backstop gate` on Linux at
all, that feed has no real CI-sourced gate JSON to render.

## Decision (2026-07-27)

Founder packet resolving the open mechanism/scope questions ahead of planning. Recorded
here rather than in a plan because it changes what "done" means for this issue, not just
how the fix is built.

### OQ-2 (capability declaration vs. named profile) — DEFERRED, with a hedge

BUNDLE-021 OQ-2 asks whether packs should declare *behavior* (reads project files, writes
files, network access) rather than backstop matching on a name or platform. That question
is **not resolved by this decision** — BUNDLE-021 remains its owner, and the Linux sandbox
mechanism must not pre-empt it by inventing a second hardcoded profile literal alongside
`darwinSandboxProfile`.

The hedge: the Linux implementation should express darwin-parity semantics (deny-default,
deny-network, scope-writes, scoped reads) against an **internal capability struct** —
something shaped like `{readable paths, writable paths, network bool}` — rather than a
second bespoke profile string. This is not a resolution of OQ-2; it is choosing an internal
representation on the Linux side that a future OQ-2 resolution can slot a pack-declared
value into without a rewrite, instead of one that has to be unwound. The macOS
`sandbox-exec` profile format is unaffected and stays as-is (acceptance item 2, above).

This is deliberately **triple-tracked** so it cannot silently drift out of view:
1. This decision note (bundles-tier position: BUNDLE-021 sits at position 1 in the current
   backlog ordering).
2. A code comment at the capability struct's definition site pointing back to BUNDLE-021
   OQ-2.
3. BUNDLE-021 OQ-2 itself (and its paired OQ-4, consumer trust semantics — the two are
   linked because a capability struct is only as trustworthy as who gets to populate it).

### OQ-3 (sandbox surface: convert+validator only, or wider) — PRINCIPLED, not incidental

The existing scope line — the Linux (and macOS) sandbox covers convert scripts and
sandbox-validators, and nothing else — is not an oversight to close in this pass. It is
the same carve-out ISSUE-045 established and verified: **producers** (e.g. the go-toolchain
pack's coverage producer, dispatched unsandboxed via `pack_gate.go:428`) genuinely need
project toolchain access — a deny-all sandbox makes them structurally non-functional, which
is exactly what ISSUE-045 hit and fixed by moving the coverage-normalization logic to the
producer side rather than trying to force it through the sandboxed convert boundary. That
carve-out is preserved verbatim by this decision; it is not reopened here.

What IS required of the Linux mechanism: it must be **graduated-profile-capable**. The
carve-out is a policy choice about which call sites get sandboxed today, not a ceiling on
what the mechanism itself can express — so that if the boundary is later widened (e.g. by an
OQ-2 resolution that lets a producer declare a narrower set of needs than "everything"),
that widening is a policy change, not a rebuild of the Linux sandbox from scratch.

The mechanism contest is narrowed to two candidates from the fuller list already recorded
above: **bubblewrap** vs. **Landlock+seccomp** (the network-denial ingredient paired with
Landlock's filesystem control, since Landlock alone doesn't cover network). A spike matrix
comparing the two against the graduated-profile requirement, CI-environment availability,
and kernel-version floor decides between them — not a document review.

### CI wiring sequencing — may land ahead of the sandbox fix

CI wiring (pointing `.github/workflows/ci.yml`'s `gate` job at `backstop gate` instead of
raw tool invocations) may land **before** the Linux sandbox mechanism is implemented, with
the pack-engine and coverage dimensions expected-failing in that interim state. This is a
founder-approved posture, explicitly not vacuous green: the failure must be loud and
attributed in the CI output to this issue (the Linux sandbox hard-error), not silenced,
skipped, or worked around. Landing CI wiring first de-risks the wiring itself and gives the
sandbox work a real CI target to prove green against once it lands, consistent with the
acceptance criterion above (a real `backstop gate` run on real CI, not an isolated unit
test).

### CI execution target — the private remote, full history

CI executes on the real `backstop-ai/backstop-core` **private** remote, per the full-history
launch plan: no squash-merge of the existing history onto the new remote. The module path
rename (`github.com/bmanson/backstop-core` → `github.com/backstop-ai/backstop-core`, see
ISSUE-087) precedes the push to that remote.

### Sizing and ownership cross-references

Per the scoping report's sizing: the Linux sandbox mechanism itself is roughly 3-4 days
*after* a container test bed exists; the CI wiring is day-scale but carries a fleet-migration
dependency (the packs this issue's acceptance criterion depends on being installable and
runnable in the CI environment, not just locally).

Ownership, to prevent overlap with the sibling CI-tracking issue: **ISSUE-020 owns the
gate-in-CI wiring** (this issue's acceptance criterion is a real `backstop gate` run going
green on Linux CI). **ISSUE-086** keeps the baseline-vacuity and hand-baked-pipeline
tracking — it does not duplicate this issue's Linux-sandbox-plus-CI-wiring scope, and this
issue does not absorb ISSUE-086's baseline-generation concerns.

## Spike results (2026-07-27)

Probe of the OQ-3 mechanism contest (bubblewrap vs. Landlock+seccomp), run against a
Docker container test bed. Recorded here so the eventual plan inherits the findings
rather than re-deriving them.

### Test-bed limit (load-bearing)

Docker Desktop's LinuxKit kernel (6.10.14-linuxkit) compiles in **neither** AppArmor
**nor** Landlock — `CONFIG_LSM="yama,loadpin,safesetid,integrity,bpf"`,
`CONFIG_SECURITY_LANDLOCK` unset. The two decisive questions (Ubuntu's
`apparmor_restrict_unprivileged_userns` behavior; the Landlock ABI level) are therefore
**unmeasurable in containers**, and any Docker-based CI job attempting to validate the
sandbox would false-negative: container seccomp blocks user namespaces in ways a bare
GitHub Actions runner does not, so `bwrap` behavior observed in-container does not
transfer to the host where `ubuntu-latest` actually runs. The container bed is good for
development iteration only — it cannot stand in for sandbox validation.

### Corrected bubblewrap invocation

The scoping report's original `bwrap` form fails two ways: it omits `--unshare-user`
(so the call clones in a privileged style and gets `EPERM`), and `--ro-bind /lib` alone
is insufficient on a merged-`/usr` Ubuntu layout. The corrected invocation, proven
exit-0 both as root and as uid 1001 with seccomp relaxed:

```
bwrap --ro-bind /usr /usr --symlink usr/bin /bin --symlink usr/lib /lib \
  --symlink usr/lib64 /lib64 --unshare-user --unshare-net --dev /dev \
  --proc /proc /bin/true
```

### Falsifiable assertion triple (carry into CI as the test set)

Three checks that together prove denial isn't vacuous — one alone (a bare "exit 0")
doesn't distinguish "sandbox denied correctly" from "sandbox did nothing":

1. Read of an unbound path → `ENOENT`.
2. `getent hosts` under `--unshare-net` → exit 2 (network denied).
3. The **same** command without `--unshare-net` → exit 0 (control leg proving the
   denial in #2 is the sandbox, not an unrelated failure).

### Re-probe list for the real `ubuntu-latest` runner

Once the private `backstop-ai/backstop-core` remote exists (ISSUE-087 sequences it),
re-run on the real GitHub Actions Linux runner, ordered by load-bearingness:

1. `cat /proc/sys/kernel/apparmor_restrict_unprivileged_userns` — decisive for the
   bwrap path; **read it, don't assume the value**.
2. The corrected `bwrap` invocation above, run as the unprivileged runner user.
3. The falsifiable assertion triple.
4. Landlock ABI probe (expect ABI 5+ on a real GHA kernel; zero signal so far — see
   Landlock note below).
5. `--unshare-pid` with `--proc` (the masked-`/proc` failure seen in-container was a
   Docker-specific artifact, not necessarily reproducible on the host).
6. `unshare -Urn` + `aa-status`.
7. `uname -r`, `/sys/kernel/security/lsm`, and whether `bwrap` is preinstalled or needs
   `apt install`.

### Narrowing state

- **bubblewrap** is modestly ahead: it is packaged, its enforcement was demonstrably
  real (the assertion triple passed), and the seccomp blockage observed in-container is
  attributable to the test-bed limit above, not to `bwrap` itself.
- **Landlock** got **zero signal** — `ENOSYS` in-container is a kernel-compile fact
  (Landlock isn't built into the LinuxKit kernel), not ABI or seccomp evidence. Any claim
  about Landlock's viability either way would be fabricated from this probe; it needs a
  real-kernel re-probe (item 4 above) before it can be ranked against bubblewrap.
- **AppArmor remains the open risk** to the bwrap path specifically —
  `apparmor_restrict_unprivileged_userns` is the one variable that could still break
  unprivileged `--unshare-user` on a real Ubuntu GHA runner, and it was unmeasurable in
  this test bed.
- Two negative results worth recording so they aren't re-tried: setting
  `apparmor=unconfined` changed nothing (there was nothing to unconfine in this kernel);
  `--cap-add SYS_ADMIN` is the wrong lever (with it, `bwrap` skips the user-namespace
  path entirely and then lacks `CAP_NET_ADMIN` — relaxing seccomp instead, as done
  above, is the correct and sufficient adjustment in containers).

### Handoff note

Probe scripts (`probe.sh`, `final.sh`, `uall.sh`) existed only in the spike session's
scratchpad and are not part of this record. Porting them to CI means dropping the Docker
wrapper and running the corrected `bwrap` invocation and assertion triple directly on the
runner — the commands captured above are the durable artifact of this spike, not the
scratchpad scripts themselves.
