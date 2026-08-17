---
name: sandbox-exit71-means-profile-rejected
description: packval sandbox exit 71 is sandbox-exec REFUSING the profile (relative packDir), not the converter failing; same-content-different-path is the isolation move
metadata:
  type: project
---

`sandboxed run (stdout) failed: exit status 71` from packval is **EX_OSERR from
`sandbox-exec` itself** — it could not APPLY the profile — not a failure of the script
being sandboxed. Reading it as "the converter is broken" sends you into the wrong file.

**Why:** measured on PLAN-ISSUE-092 TASK-015 (2026-08-16), filed as **ISSUE-147**.
`darwinSandboxProfile` (`pkg/packval/sandbox_nonlinux.go`) builds the profile from the
packDir argument AS GIVEN, calling `filepath.EvalSymlinks` with no `filepath.Abs` first.
Go's `EvalSymlinks` **preserves relativity**, so `pack test packs/substantiveness` emits a
relative `(subpath "packs/substantiveness")`, which `sandbox-exec` rejects outright. The
same pack via an absolute path converts fine. Compounding it, `platformSandboxedRunStdout`
never captures sandbox-exec's stderr, so the actual diagnostic is discarded and the
operator sees only the bare exit code — the same "information existed, surfacing did not"
family as [[project_information_existed_surfacing_did_not]].

**How to apply:** when a sandboxed step fails and the script runs fine by hand, suspect
**the path argument is itself the data** before suspecting the payload or the profile
contents. The isolation that settled it in minutes: copy the tree somewhere else and run
BOTH — identical bytes (`diff -rq` clean, byte-identical engine output, identical modes)
failing in one location and passing in the other proves the location is the variable, and
absolute-vs-relative is the first thing to vary. Hand-reproducing the sandbox call with the
real profile and real payload is what ruled out the profile and the payload as causes.
Until ISSUE-147 lands, pass **absolute paths** to `pack test` on macOS; a relative one
yields bogus convert failures on every pack that ships a `convert:`. Related:
[[project_pack_copies_and_stale_gate_binary]].
