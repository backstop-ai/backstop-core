---
name: project_issue180_testmain_sandbox_family
description: ISSUE-180 confirms/narrows ISSUE-164 for pkg/pack/distribution; pkg/pack/engine still unconfirmed; pkg/gate confirmed clean
metadata:
  type: project
---

ISSUE-180 (`pkg/pack/distribution` missing sandbox-helper `TestMain`) is the same re-exec
collision family as [[project_thin_executor_dogfood]]-adjacent ISSUE-163 (fixed, `cmd/backstop`)
and ISSUE-164 (open `type: question`, named `pkg/pack/distribution` + `pkg/pack/engine` as
at-risk but unconfirmed). ISSUE-180 is the confirmation ISSUE-164 itself said should trigger a
promotion, not a duplicate — filed as a new issue (not a hand-edit of ISSUE-164) because the
issue-author agent's job is filing, and ISSUE-164's own text anticipated "re-file or promote"
as the correct move.

**Narrowing recommendation left un-actioned:** ISSUE-164 should be narrowed to just
`pkg/pack/engine` (still unconfirmed) or closed with a note pointing to ISSUE-180 for the
`pkg/pack/distribution` half. Not done in the ISSUE-180 session — flagged as a recommendation
only. Whoever picks up ISSUE-164 next should actually make that edit.

**pkg/gate confirmed clean, not just "no evidence":** `grep -rl "backstop-core/pkg/packval"
--include="*.go"` across the whole module returns exactly four dirs: `cmd/backstop`,
`pkg/pack/distribution`, `pkg/pack/engine`, `pkg/packval`. `pkg/gate` does not import
`pkg/packval` at all — it cannot be exposed to this defect class by construction. Worth citing
directly next time this comes up rather than re-running the grep.

**The guard's structural blind spot** (`cmd/backstop/sandbox_helper_testmain_guard_test.go`'s
`TestSandboxHelperGate_PresentInEveryPackvalReachingTestMain`) is real: `scanGoPackages` +
`if pkg.testMain == nil { continue }` means a packval-importing package with NO `TestMain` at
all is invisible to the roster by construction — that blind spot is exactly what let
`pkg/pack/distribution` slip through. Generalizing the roster to flag a *missing* `TestMain` (not
just a malformed one) is ISSUE-164's territory, deliberately left out of ISSUE-180 to avoid
scope-creep into an issue that already exists for it.

**Why-of-format:** the founder's live task explicitly asked to decide between folding the
structural generalization into the same issue vs. a follow-on, "using your normal judgment" —
the deciding factor was that ISSUE-164 already existed and named the generalization territory
first; when an existing issue already owns generalization/structural scope, keep the new
confirmed-bug issue narrow and cite the existing one rather than re-scoping it in.
