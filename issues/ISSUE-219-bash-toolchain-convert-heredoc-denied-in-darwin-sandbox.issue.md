---
title: "Bash Toolchain Convert Heredoc Denied In Darwin Sandbox"
schema_version: issue/v1

issue:
  id: ISSUE-219
  title: "Bash Toolchain Convert Heredoc Denied In Darwin Sandbox"
  type: bug
  status: closed
  created: "2026-10-12"
  closed: "2026-10-13"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
resolved-by: e7d9169fe7d31386e97f9b2a803c6baeee3fc77b
---

# Bash Toolchain Convert Heredoc Denied In Darwin Sandbox

## Problem

On macOS, when a bash-test wrapped by the `backstop-ai/bash-toolchain` pack fails, the gate hides the
verifier's real failure message behind an opaque convert error.

Mechanism, verified in the installed pack (`.backstop/packs/backstop-ai/bash-toolchain`):

- The convert script `scripts/test-to-sarif.sh` (~line 21) turns a non-zero verifier result into SARIF
  with `python3 - "$message" <<'PY' ... PY`, a here-document.
- That branch runs only when the wrapped command (for example
  `scripts/verify-public-product-model.sh`) exits non-zero. The success path prints a fixed SARIF
  string and never reaches the here-document.
- Bash implements a here-document by writing a temporary file. The darwin convert sandbox denies file
  writes, so bash fails with `cannot create temp file for here document: Operation not permitted`.
- The gate then reports only
  `convert step (scripts/test-to-sarif.sh) failed: sandboxed run (stdout) failed: exit status 1`. The
  verifier's actual message, which the convert was meant to carry into a finding, is lost.

Observed 2026-10-12, when an unlinked docs paragraph made the public-product-model verifier fail.

### Reproduction

macOS only, in a repo using the bash-toolchain pack: make the wrapped bash verifier exit non-zero.
The gate output shows the convert-step failure above instead of the verifier's message.

## Impact

- On darwin, every bash-test failure is undiagnosable from gate output. Only green runs work.
- Evidence integrity: the real failure cause is dropped and replaced by a sandbox artifact, the same
  class of loss as ISSUE-209.
- Linux CI (Landlock) may not exhibit this, so the defect is invisible there.

## Root cause

The convert script depends on a capability (creating temp files for a here-document) that the darwin
convert sandbox withholds. Two parts:

1. Pack side: `test-to-sarif.sh` uses a here-document on its failure branch.
2. Core side: the convert sandbox offers no private writable temp directory, so any convert that needs
   scratch space fails opaquely, and the sandbox error is flattened to `exit status 1`.

## Direction

(Direction, not a plan.)

Owning repo for the primary fix: `backstop-ai/bash-toolchain` (separate repo, not this one).

1. Pack (primary): remove the here-document. Pass the Python program via `python3 -c '...' "$message"`
   or ship a `.py` file and invoke it. Add a pack test that runs the failing-verifier path under the
   sandbox. Alternatively document that convert scripts must not require temp files.
2. Core (optional, tracked here): give the convert sandbox a private writable `TMPDIR` so convert
   scripts that legitimately need scratch space work, and surface the sandboxed run's stderr in the
   convert-failure message so a denial like this is diagnosable.

Keep each change small and contained.

## Notes

- Related: ISSUE-216 (darwin sandbox denies a relative pack dir and drops the real cause); ISSUE-209
  (pack engine dispatch error discards findings). All three are darwin or evidence-loss defects where
  the real error is hidden.

## Resolution

Fixed on the pack side in `backstop-ai/bash-toolchain` v0.1.3 (commit
`e7d9169fe7d31386e97f9b2a803c6baeee3fc77b`, 2026-10-13). `scripts/test-to-sarif.sh` now passes the
Python program with `python3 -c '...' "$message"` instead of a here-document, so the failure branch
needs no temp file. Output is byte-identical across quoting, control-character and non-ASCII cases.
A new pack test, `TestBashToolchainConverterFailureBranchSurvivesWriteDenial`, runs the failure
branch under a write-denying `sandbox-exec` profile on darwin. The earlier test missed the defect
because bash 3.2 falls back to `/tmp` when `TMPDIR` does not exist. backstop-core adopted 0.1.3 the
same day.

Not done here: the optional core item (a private writable `TMPDIR` for converts and surfacing the
sandbox's stderr). It moves to ISSUE-224, along with a second darwin denial found while verifying
this fix (framework Python under `/Library/Frameworks`).
