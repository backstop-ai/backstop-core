---
title: "macOS sandbox profile denies dyld reads — every convert script using a dynamically-linked interpreter SIGABRTs"
schema_version: issue/v1

issue:
  id: ISSUE-029
  title: "macOS sandbox profile denies dyld reads — every convert script using a dynamically-linked interpreter SIGABRTs"
  type: bug
  status: ready
  created: "2026-06-23"

complexity:
  scope: contained
  uncertainty: known
  risk: critical

verification:
  level: security
  coverage_threshold: 90
  test_command: "go test ./pkg/packval/... -run TestSandbox"

implementation:
  summary: >
    Relax the macOS sandbox-exec profile in SandboxedRun and SandboxedRunStdout
    to allow read-only access to system and dyld library paths (dyld shared cache,
    /usr/lib, /System/Library, /usr/local/lib, /opt/homebrew/lib) while preserving
    all deny rules: deny file-write*, deny network*, deny reads of non-pack
    non-system paths. Both variants share the profile string so the fix is a
    single-point change.
  package: pkg/packval

requirements:
  - id: REQ-001
    text: >
      The macOS sandbox profile MUST allow read-only access to the system and
      runtime library paths that dynamically-linked interpreters (e.g. jq)
      require at startup: the dyld shared cache, /usr/lib, /System/Library
      (framework and dyld paths), /usr/local/lib (Intel Homebrew), and
      /opt/homebrew/lib (Apple Silicon Homebrew).
  - id: REQ-002
    text: >
      The relaxed profile MUST still deny all file-write operations, all network
      operations, and all reads of paths that are neither (a) under packDir nor
      (b) under the allowed system/runtime library paths. A convert script MUST
      NOT be able to read project source files, write any file, or open a network
      connection under the fixed profile.
  - id: REQ-003
    text: >
      A real, un-stubbed integration test MUST run a convert script that invokes
      a dynamically-linked interpreter (jq) through the production
      SandboxedRunStdout path (i.e. via real sandbox-exec) and the command MUST
      succeed, producing correct output. The /bin/sh stub pattern (stubSandboxedRunStdout)
      MUST NOT be used to satisfy this requirement — the stub is exactly what hid
      this defect.
  - id: REQ-004
    text: >
      A real, un-stubbed security test MUST confirm that under the relaxed profile:
      (a) a convert script attempting to READ a file outside packDir and outside the
      allowed system/library paths is DENIED; (b) a convert script attempting to
      WRITE any file is DENIED; (c) a convert script attempting to open a NETWORK
      connection is DENIED. All three denials are required — relaxing the profile
      MUST NOT silently open any of these holes.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      SandboxedRun and SandboxedRunStdout both use an updated macOS sandbox
      profile that includes (allow file-read*) subpath rules for /usr/lib,
      /System/Library, /usr/local/lib, /opt/homebrew/lib, and the dyld shared
      cache path in addition to packDir.
    tests:
      - TestSandboxProfileAllowsDyldLibraries
  - id: CLM-002
    requirement: REQ-002
    text: >
      The updated profile retains (deny file-write*), (deny network*), and deny
      default, so a converter cannot write files, reach the network, or read
      arbitrary project paths not covered by the allowlist.
    tests:
      - TestSandboxDeniesProjectFileRead
      - TestSandboxDeniesFileWrite
      - TestSandboxDeniesNetwork
  - id: CLM-003
    requirement: REQ-003
    text: >
      TestSandboxConvertWithRealInterpreter runs a convert script that pipes
      stdin through real jq under the real SandboxedRunStdout/sandbox-exec
      path (no stub) and asserts the output is the expected transformed JSON.
      The test is skipped when sandbox-exec or jq is unavailable (non-darwin or
      missing binary) but is not accepted as green when skipped on darwin with
      jq present.
    tests:
      - TestSandboxConvertWithRealInterpreter
  - id: CLM-004
    requirement: REQ-004
    text: >
      TestSandboxSecurityDenialsHold exercises three deny assertions under the
      relaxed profile via real sandbox-exec: (a) a shell one-liner reading a
      file above packDir fails with a non-zero exit, (b) a shell one-liner
      writing a temp file fails, (c) a shell one-liner opening a TCP connection
      fails. All three sub-cases must fail (exit non-zero) for the test to pass.
    tests:
      - TestSandboxSecurityDenialsHold

contracts:
  - file: pkg/packval/sandbox.go
    provides:
      - name: SandboxedRun
        kind: function
        signature: "func SandboxedRun(cmd string, args []string, packDir string) ([]byte, error)"
      - name: SandboxedRunStdout
        kind: function
        signature: "func SandboxedRunStdout(cmd string, args []string, packDir string, stdin []byte) ([]byte, error)"
---

# ISSUE-029: macOS sandbox profile denies dyld reads — every convert script using a dynamically-linked interpreter SIGABRTs

## Problem

### The broken profile

`pkg/packval/sandbox.go`'s `SandboxedRun` (line 14) and `SandboxedRunStdout`
(line 45) construct identical macOS `sandbox-exec` profiles:

```
(version 1)(deny default)(allow process*)(allow file-read* (subpath "<packDir>"))(deny network*)(deny file-write*)
```

The only `file-read*` allowance is `(subpath "<packDir>")`.

### What breaks

A pack's `convert` script (e.g. ast-grep's `to-sarif.sh`) pipes native engine
output through a dynamically-linked interpreter such as `jq`
(`/usr/local/bin/jq` on Intel Homebrew, `/opt/homebrew/bin/jq` on Apple
Silicon). At process launch, `jq`'s dynamic linker (`dyld`) must load shared
libraries — `/usr/local/lib/libjq.1.dylib`, the dyld shared cache,
`/System/Library` frameworks — before the process enters `main()`. Every one of
those library paths is outside `<packDir>`. The profile denies those reads.
`dyld` aborts the process with `SIGABRT` ("abort trap: 6") before a single line
of `jq` logic executes.

The result: `SandboxedRunStdout` returns an error containing "abort trap" and
produces no SARIF output. The convert step fails completely.

This is not a `jq`-specific defect. Any dynamically-linked interpreter invoked
from a convert script — `python3`, `node`, `ruby`, custom binaries — requires
the same dyld bootstrap reads. The profile blocks every one of them.

### Consequence

NO convert script that uses a dynamically-linked interpreter can run under the
current macOS sandbox. The convert step (engine native output → SARIF) cannot
execute for any such pack. This directly blocks:

- SPEC-037 Seed 3 (substantiveness pack, ast-grep → SARIF convert via `jq`)
- SPEC-037 Seed 4 (contracts pack, same shape)
- Any future pack whose convert script calls a system binary

### Why it was hidden

Every existing convert test stubs the sandbox. `stubSandboxedRunStdout` in the
test suite shells the convert script directly via `/bin/sh`, bypassing
`sandbox-exec` entirely. The same pattern is used in the go-toolchain convert
tests. No existing test routes a real convert through real `sandbox-exec`. The
stub is operationally correct for unit speed but structurally incorrect as proof
of sandbox compatibility — it is precisely the pattern that allowed this defect
to survive undetected.

### Why it is security-relevant

This sandbox is SPEC-035's trust boundary for arbitrary pack `convert` scripts
and sandbox `validator`s — the one piece of pack-supplied code that cannot be
allowlisted as a named tool. The sandbox is the only enforcement layer standing
between a malicious or buggy convert script and the host filesystem / network.
A broken sandbox that causes abort-on-launch is a hard operational defect; a
sandbox silently relaxed too far is a security defect. Both halves of the fix
(relaxation + denial tests) are required.

### Repro

1. Write a convert script that pipes stdin through real `jq` (any transformation).
2. Call `packval.SandboxedRunStdout(<path-to-jq>, args, packDir, input)` directly
   (no stub) on macOS.
3. Observe: exit with "abort trap: 6" / `signal: abort trap`. No output.

## Solution

Extend the macOS sandbox profile to allow **read-only** access to the system and
runtime library paths that `dyld` requires:

```
(version 1)
(deny default)
(allow process*)
(allow file-read*
  (subpath "<packDir>")
  (subpath "/usr/lib")
  (subpath "/usr/local/lib")
  (subpath "/opt/homebrew/lib")
  (subpath "/System/Library")
  (literal "/private/var/db/dyld")
  (subpath "/private/var/db/dyld"))
(deny network*)
(deny file-write*)
```

The security model is preserved: read-only access to **system libraries** (not
project source) does not let a malicious convert script exfiltrate project code,
write files, or reach the network — those denials remain hard. The dyld shared
cache path (`/private/var/db/dyld`) on older macOS versions and the standard
framework paths under `/System/Library` cover the OS linker bootstrap.

Both `SandboxedRun` and `SandboxedRunStdout` share the same profile string;
the fix is a single-point change. The profile string should be extracted to a
package-level constant or constructor to eliminate the current duplication.

**Scope fence:** This issue covers the macOS profile relaxation only. The
sibling facet — the Linux sandbox is a no-op (`errors.New("sandbox unavailable
on linux")`, lines 25 and 61), meaning convert and validator scripts run fully
unsandboxed on Linux — is tracked as ISSUE-020 (cross-platform Linux sandbox,
security-elevated, separate/deferred). Do NOT fold the Linux fix here.

## Verification

Acceptance requires **both** halves:

**(a) Integration test — real interpreter through real sandbox succeeds.**
A test (`TestSandboxConvertWithRealInterpreter`) must run a convert script using
a real dynamically-linked interpreter (`jq`) through the production
`SandboxedRunStdout` / real `sandbox-exec` path and assert the output is the
correctly transformed result. The `stubSandboxedRunStdout` / `/bin/sh` bypass
pattern MUST NOT be used to satisfy this — the stub is what hid the defect.

**(b) Security test — deny properties still hold under the relaxed profile.**
A test (`TestSandboxSecurityDenialsHold`) must confirm via real `sandbox-exec`
that under the relaxed profile: (i) reading a project file outside `packDir`
and outside allowed system paths is denied, (ii) writing any file is denied,
(iii) opening a network connection is denied. All three denials must hold or
the test must fail — relaxing the profile must not silently open a hole.

Tests tagged for darwin may skip on linux (sandbox-exec unavailable) but must
not be accepted as green when skipped on a darwin host with `jq` present.

## References

- `pkg/packval/sandbox.go:14,45` — the broken profile string (identical in both
  `SandboxedRun` and `SandboxedRunStdout`)
- `pkg/packval/sandbox.go:25,61` — Linux no-op (ISSUE-020 facet; do not fold in)
- SPEC-035 — defines the sandbox as the trust boundary for pack convert scripts
  and sandbox validators; this defect undermines that boundary operationally
- SPEC-037 — the consuming spec (Seed 3 ast-grep→SARIF convert) that exposed
  the defect; directly blocked until this is fixed
- ISSUE-020 — sibling Linux facet: convert/validators run fully unsandboxed on
  Linux; security-elevated, deferred separately
- ISSUE-028 — sibling substrate blocker (multi-rule ast-grep dispatch); closed
- ISSUE-027 — sibling substrate blocker (default-pack shipping)
