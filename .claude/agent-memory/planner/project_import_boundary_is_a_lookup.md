---
name: import-boundary-is-a-lookup
description: "Whether two packages can share a helper is a LOOKUP in the architecture policy file, never a generalization from a prior plan's wall — read backstop-core.yml's deps: block and check whether the edge already exists"
metadata:
  type: project
---

When an issue asks "can this predicate/helper be shared, or does each package need
its own copy", answer it by READING
`.backstop/packs/backstop-ai/backstop-core-architecture/architecture/backstop-core.yml`
— specifically its `deps:` block — not by reasoning from a previous plan that hit a
boundary. The architecture dimension is live and gate-enforced, so the file is the
authority, and it is per-edge: one blocked edge implies nothing about another.

**Why:** PLAN-ISSUE-118 hit a real wall (`pkg/gate` may NOT import `pkg/pack/engine`),
and ISSUE-140 was filed citing it as precedent for "maybe packval needs its own copy
too." The lookup said the opposite: `packval: { mayDependOn: [..., check, ...] }` and
`cli: { anyProjectDeps: true }`, with `pkg/check` declaring NO project deps at all —
so a helper homed in `pkg/check` adds ZERO new edges, cannot create a cycle, and needs
no policy edit. Generalizing from 118's wall would have planned two copies and shipped
the exact drift the issue exists to eliminate.

**How to apply:** three questions, in order, before proposing a shared home —
1. Is the edge from each consumer to the candidate home ALREADY declared in `deps:`?
   (Better still: already *exercised* — grep the consumer's imports. An already-used
   edge is a zero-risk home.)
2. Does the candidate home declare project deps that could cycle back? A package with
   none (`anyVendorDeps: true` and no `mayDependOn`) is cycle-proof by construction.
3. Does the home make sense on the MERITS, not just legally? Prefer the package that
   already owns the concept (e.g. `pkg/check` owns the command-execution seam, so it
   owns "did the process start"; `pkg/pack/engine` owns DECLARATION — bindings,
   allowlist — so an execution predicate there is a stray).

If the answer requires EDITING backstop-core.yml, the home is probably wrong — write
that into the plan as a stop condition for the implementer, since the file is an
explicit ratchet and editing it to fit a helper inverts the layering it protects.
A brand-new package is the worst option: one new component entry plus N new edges,
for one function.

Also record the rejected alternatives in the plan notes so a reviewer or a later lane
does not re-litigate the placement. Related: [[project_defect_pinned_by_shipped_tests]],
[[feedback_verify_issue_premises]] — an issue's architectural framing is a claim to
check, exactly like its factual ones.
