---
name: gate-file-scope-nongo-dir-crash
description: "`backstop gate --file <non-Go file in a non-Go directory>` always REDs on a go-test engine crash; pre-existing, reproducible on untouched files, and the diff-scoped gate is unaffected"
metadata:
  type: project
---

Scoping the gate to a non-Go file that lives in a directory containing no Go
package makes `pack_engines` FAIL with:

    dispatching findings engine "go-test" for pack backstop/go-toolchain:
    engine "go test" crashed: non-zero exit with no parseable findings: exit status 1

Cause: go-toolchain's `go-test` engine is `package_scoped: true`, so it derives
package targets from the scoped files. For `.github/workflows/release.yml` it
derives `.github/workflows`, and `go test .github/workflows` exits non-zero
("no required module provides package"), which trips `crash_guard`.

**Why:** measured 2026-07-27 during ISSUE-087 Phase 4. It is NOT caused by the
file's content — the identical failure reproduces on the untouched, long-tracked
`.github/workflows/ci.yml`, while `--file README.md` PASSES (root is a real Go
package, so the derived target resolves). Two `--file` flags also collapse to one
("running against 1 explicit files"): the flag is a string, not a slice.

**How to apply:** never conclude a new non-Go file is dirty from a `--file`-scoped
gate. Run the control first (`gate --file` on a pre-existing file in the same
directory); if it reproduces, the red is inherited. The DEFAULT diff-scoped gate
is unaffected and stays green with those same `.yml` files in scope — verified,
so this does not block a phase whose deliverables are workflows. Related:
[[gate_all_underreports_vs_diff]], [[pack_copies_and_stale_gate_binary]].
