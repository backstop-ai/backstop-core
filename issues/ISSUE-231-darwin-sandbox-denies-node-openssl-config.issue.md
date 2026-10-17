---
title: "Darwin Sandbox Denies Node OpenSSL Config"
schema_version: issue/v1

issue:
  id: ISSUE-231
  title: "Darwin Sandbox Denies Node OpenSSL Config"
  type: bug
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Darwin Sandbox Denies Node OpenSSL Config

## Problem

On macOS, a Node-based pack script (a convert or validator run by `backstop pack test`) can fail to start
inside the pack sandbox because Node cannot read the system OpenSSL configuration. Setting
`OPENSSL_CONF=/dev/null` in the engine's environment is a workaround.

Verified 2026-10-17 by running a Node binary directly under `sandbox-exec` with the profile text from
`darwinSandboxProfile` (read subpaths only; no pack dir needed to reproduce):

```
node: OpenSSL configuration error: ... calling fopen(/usr/local/etc/openssl@3/openssl.cnf, rb)
Operation not permitted
```

With `OPENSSL_CONF=/dev/null` the same command starts normally.

Cause in code: `darwinSandboxProfile` (`pkg/packval/sandbox_nonlinux.go:188-215`) allows file reads only
under the pack directory, `/usr/lib`, `/System/Library`, `/usr/local/lib`, `/usr/local/Cellar`,
`/usr/local/opt`, `/opt/homebrew`, and two dyld cache locations. The Intel Homebrew config directory
`/usr/local/etc` is not listed.

## Impact

Any Node-based convert or validator fails under the darwin sandbox on a machine whose Node reads its
OpenSSL config from `/usr/local/etc`. As with the other darwin sandbox denials, the cause is hard to find
because the sandbox's stderr is not surfaced (see ISSUE-216, ISSUE-218).

## Not confirmed

- The sandbox is used only by packval fixture execution (`pkg/packval/executor.go:223,263`); I found no
  other caller, so this is believed to affect `pack test` / `pack add` validation, not gate engines.
- Only the Intel Homebrew path was reproduced. Apple-Silicon Homebrew config lives under `/opt/homebrew`,
  which is already readable, so it is expected to work; not tested. Other Node installs may read other paths.

## Direction

(Direction, not a plan.) Either add a scoped read grant for the OpenSSL config location (a `literal` or
narrow `subpath`, not a blanket `file-read*`), or document `OPENSSL_CONF=/dev/null` for pack authors.
Either way the trust model stays: deny default, no writes, no network.

## Notes

- Related: ISSUE-224 (framework Python denied: same read allowlist, same file, same fix shape) and
  ISSUE-216 (relative pack dir denied by the same profile). ISSUE-216 and ISSUE-224 are on the unmerged
  `fix/engine-evidence-integrity` branch (not yet on main); the fixes could share one review of
  the allowlist.
