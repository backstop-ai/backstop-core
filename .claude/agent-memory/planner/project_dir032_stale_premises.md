---
name: dir032-stale-premises
description: DIR-032 gate-verdict-honesty members were filed over months; several premises are already dissolved by later delivered work — falsify empirically in a worktree before authoring
metadata:
  type: project
---

DIR-032's older members (ISSUE-066/067/091/093 era, filed ~2026-07-17) describe gate
defects against a codebase that has since been rebuilt underneath them. Falsify the
premise against current code AND with a real gate run before authoring a plan.

**Why:** ISSUE-066 ("gate test verification honors a plan's narrow `-run` filter, so a
non-matching regression stays invisible") was measured non-reproducible on 2026-08-17.
Three separate later deliveries dissolved it, none of which names ISSUE-066:
- SPEC-034/040/042 (native toolchain cutover) demoted `verification.test_command` to
  inert metadata. `ExtractSpecVerifications` still requires the field to be NON-EMPTY
  for a spec to enter coverage extraction, but its VALUE is never read or executed —
  the only consumers are a struct-field declaration and an assignment. Core runs no
  `-run` filter anywhere.
- ISSUE-070 made the diff-scope filter apply to dispatched pack violations at all.
- ISSUE-129 declared `exempt_from_scope_filter: true` on the go-toolchain `go-test`
  binding, so an out-of-diff-scope test failure still REDs a diff-scoped gate.

The bound on "what must pass" is now the go-test engine's `project_target: "./..."` —
the whole module — not any mapped subset. `fileModeTestTarget` narrows a
`package_scoped` engine only under `GateScopeModeFile` (explicit `gate --file`), never
under the default diff scope.

**How to apply:** before planning any DIR-032 member, (1) grep for the mechanism the
issue names in current non-test source, and (2) run the real repro in a throwaway
`git worktree` — inject a production defect that breaks tests in UNCHANGED files whose
names match no claim-mapping pattern, then run a freshly built `./bin/backstop gate`
diff-scoped. Pick a defect with wide blast radius and a distinctive literal (changing
the pinned semgrep version in `pkg/pack/engine/allowlist.go` yielded 108 findings all
quoting the injected string, making attribution unambiguous without a control run).
See [[closeout_real_gate_in_worktree]] for worktree setup (copy `.backstop/packs` in —
and note `.backstop/coverage-exclusions` is TRACKED, so `cp -R .backstop dest/.backstop`
nests instead of creating).
