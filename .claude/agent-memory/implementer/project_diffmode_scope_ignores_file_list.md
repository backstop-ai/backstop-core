---
name: diffmode-scope-ignores-file-list
description: "ComputeGateScope(root, GateScopeModeDiff, files) IGNORES files — diff scope resolves from git; in a non-git temp dir it silently falls back to a WHOLE-CODEBASE scan"
metadata:
  type: project
---

`gate.ComputeGateScope(projectRoot, gate.GateScopeModeDiff, files)` **discards the `files`
argument entirely**. Diff mode resolves from git (`resolveGateScopeDiff`, `pkg/gate/scope.go`):
merge-base against `origin/main`/`origin/master`, else `git diff --name-only HEAD` plus
`git ls-files --others`. Only `GateScopeModeFile` honors an explicit list.

**Why it bites:** in a `t.TempDir()` that is not a git repo, `resolveGateScopeDiff` falls back to
`resolveGateScopeAll` — a full-codebase walk — and returns a scope whose Mode still says `"diff"`.
So a test that "builds a diff scope of one test file" silently gets every file in the fixture and
looks plausible. Mine was caught only because the test asserted its own premise ("the diff must be
ENTIRELY test files") and tripped on `backstop.lock`. Without that assertion the acceptance test
would have passed while testing the wrong scope.

**How to apply:** to get a genuine one-file diff scope in a test, follow the in-tree precedent
`gitInitCommitAll` / `newDiffScopedPackGateProject` (`cmd/backstop/gate_scope_test.go`):
write the whole fixture, `git init` + `add -A` + `commit`, and THEN plant the changed file so it is
the sole entry `git ls-files --others` appends. Order matters — anything written before the commit
becomes tracked-and-clean and drops out of scope.

Always assert the resulting `scope.Files` explicitly (`len == 1 && Files[0] == want`). A scope
assertion is cheap and is the only thing that distinguishes "diff-scoped" from "silently
whole-repo". `newGateScope` is unexported, so there is no shortcut from outside `pkg/gate`.

Related: [[project_green_gate_by_scope_exit]], [[project_full_cli_gate_fixture_reqs]].
