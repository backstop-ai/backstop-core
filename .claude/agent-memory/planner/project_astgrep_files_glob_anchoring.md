---
name: astgrep-files-glob-anchoring
description: ast-grep `files:` resolves against the invoking process's CWD and IS honored under explicit-file dispatch — unlike semgrep `paths.include`; a `**/`-prefixed variant is dark only by accident of hidden-dir skipping
metadata:
  type: project
---

An ast-grep rule's `files:` list resolves against the **invoking process's working
directory**, and it is honored under BOTH directory dispatch and explicit-file dispatch.
Measured 2026-08-17, ast-grep 0.43.0, while planning ISSUE-158.

**Why:** this is the opposite of semgrep's `paths.include`, which goes inert under
explicit-file dispatch — see [[project_path_include_inert_under_file_dispatch]] and
ISSUE-151. Do NOT generalize one tool's path-scoping behavior to the other; they are
different mechanisms and a plan that routes an ast-grep fix through ISSUE-151's
conclusions will be wrong.

Two consequences that decided ISSUE-158's design:

- **Different CWDs are an exploitable seam.** packval phase3 runs the engine with
  `cmd.Dir = packDir`; the gate runs it with `Dir = projectRoot`. So one *root-anchored*
  glob (no leading wildcard segment) can match a pack's own fixture tree under packval
  and match nothing in a consumer workspace — the property a "consumer-dark but
  installable pack" harness needs.
- **A `**/`-prefixed glob is a vacuous-green trap.** `**/testdata/fixtures/.../*.go`
  also passes `pack test` and also looks consumer-dark — but only because ast-grep skips
  hidden directories by default and installed packs live under `.backstop/`. Run the scan
  with `--no-ignore hidden` and the `**/` variants leak findings from the installed pack's
  own fixtures while the root-anchored form stays at zero.

**How to apply:** when a plan turns on a path-scoping glob, make the discriminating test
force hidden directories into the scan (`--no-ignore hidden`), and assert *root-anchoring
structurally* (reject any entry whose first segment contains `*`). Passing `pack test`
alone does not distinguish the correct glob from the trap.
