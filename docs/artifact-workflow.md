# The artifact workflow

**Most teams using backstop will never need this page.** The gate (`backstop gate`) and
packs stand entirely on their own: install a pack, run the gate, get blocked on real
findings. That is the whole product for most adopters.

This page describes the *optional* discipline layer stacked on top: a small set of
numbered, schema-validated markdown/YAML files that record what work was proposed, why,
what it promised, and whether that promise was kept. It exists because AI agents are
fluent enough to produce plausible work that nobody asked for and nothing verifies. The
artifact layer makes "what were we supposed to be building" a machine-checkable fact
rather than something living in a chat transcript.

Adopt it if you want traceability from an idea to the tests that prove it shipped. Skip
it if you want the gate.

For a shorter conceptual tour see [concepts.md](concepts.md); for full command syntax see
[cli-reference.md](cli-reference.md).

---

## Two tracks, and only two

Every piece of work enters through one of two chains. Which one depends on whether you
are *reacting* to something or *proposing* something.

### Reactive: `issue → plan`

A bug, a piece of tech debt, a policy violation, an enhancement someone noticed. You
already know what is wrong. You write it down, then you plan the fix.

```
ISSUE-082  →  PLAN-ISSUE-082
```

**Issues never get specs.** If you find yourself wanting to spec an issue, the work is
bigger than an issue and belongs on the proactive track.

### Proactive: `bundle → spec → plan → implementation`

A feature. You do *not* yet know what you are building, so you explore first.

```
BUNDLE-003  →  SPEC-069  →  PLAN-SPEC-069  →  code + tests
```

- A **bundle** is the exploration space — the problem, the user story, open questions,
  candidate design decisions. It is where disagreement is supposed to happen.
- A **spec** is one implementable slice carved out of a matured bundle: requirements,
  claims, mandated test names, contracts, sharp edges.
- A **plan** turns a spec into ordered TDD tasks with explicit file scope.

**Specs are never standalone.** A spec always names its source bundle in frontmatter
(`source.bundle: BUNDLE-003`). A spec with no bundle behind it is a spec nobody explored,
which in practice means a spec that encodes one agent's first guess.

Both tracks converge on a plan, because a plan is the only artifact that authorizes
touching source code.

---

## Creating artifacts: the CLI reserves the number

Artifacts are ledger-numbered. You never pick the number.

```bash
backstop artifact new issue  --slug tool-allowlist-unreachable-entries
backstop artifact new bundle --slug onboarding-experience
backstop artifact new spec   --slug backstop-init
backstop artifact new plan   --slug backstop-init --source SPEC-069
```

This scaffolds a valid skeleton at the configured artifact root with the next available
ID, its required sections, and a correct `schema_version`. The ID is reserved through an
annotated git tag when git is available (so two people scaffolding at once cannot collide
on the same number), falling back to a local scan of the artifact directory when it is
not — see `pkg/scaffold/idresolver.go`.

Types: `issue`, `bundle`, `spec`, `plan`, `adr`, `directive`, `capability`.

Two mechanics worth knowing:

- **`plan` requires `--source`**, and the plan inherits its source's number rather than
  taking a fresh one. `--source SPEC-069` produces `PLAN-SPEC-069`; `--source ISSUE-082`
  produces `PLAN-ISSUE-082`. The filename prefix (`PLAN-SPEC-` vs `PLAN-ISSUE-`) is how
  the track is legible at a glance.
- **Never hand-number and never suffix.** No `ISSUE-009b`, no "SPEC-041 slice 2". If the
  work split, it is a new artifact with its own reserved ID.

---

## Lifecycle states

Each type carries a status field it moves through. The point is not ceremony — it is that
the gate reads these, so a status is a claim about reality that can be contradicted.

| Type | Field | Live states | Terminal |
|---|---|---|---|
| Bundle | `status.maturity` | `idea` → `exploring` → `defined` → `ready` | `delivered`, `replaced`, `canceled`, `deprecated` |
| Spec | `status` | `draft` → `ready-for-implementation` | `implemented`, `replaced`, `canceled`, `deprecated`, `obsoleted` |
| Plan | `status` | `draft` → `ready` → `implementing` | `completed`, `replaced`, `canceled`, `obsoleted` |
| Issue | `issue.status` | `open` → `ready` → `in-progress` (or `blocked`) | `closed`, `replaced`, `canceled`, `obsoleted` |

Three things this table does not show:

**In practice, a bundle starts at `exploring`, not `idea`.** `idea` is the bare scaffold
state; a bundle worth working on starts life with real open questions to resolve, which is
`exploring`. Promotion past that point — resolving an open question, moving to `defined` —
is driven by the person who owns the decision, not assumed or advanced on their behalf.

**Bundle promotion is structural, not editorial.** Moving a bundle to `defined` requires
real content — a `requirements[]` array, a `solution.approach`, Draft Design Decisions —
not prose that reads as if it were defined. `requirements[]` is deliberately *omitted*
below `defined`; a bundle that has not been explored should not look like one that has.

**The terminal states are not interchangeable.** `replaced` means a named successor took
over (and requires `replaced-by`). `canceled` means the work was abandoned. `obsoleted`
means the capability shipped and was later removed with no 1:1 successor (and requires
`obsoleted-by`). Retiring an artifact into the wrong terminal state strands the specs and
plans that pointed at it.

---

## Closing an issue: `delivered_by` vs `resolved-by`

An issue does not close just because someone typed `status: closed`. Closing requires
traceability — evidence the fix exists — plus a `## Resolution` section. There are two
weights of evidence, and at most one may be present.

**`delivered_by: PLAN-ISSUE-NNN`** — the full track. The issue was fixed through a plan
with mandated tests, and that plan reached `status: completed`. The close traces to the
completed plan instead of re-authoring requirements and claims onto the issue itself. Use
this whenever the fix went through `/plan` → `/implement`.

Real example — `issues/ISSUE-082-tool-allowlist-unreachable-entries.issue.md`:

```yaml
delivered_by: PLAN-ISSUE-082

issue:
  id: ISSUE-082
  status: closed
  closed: "2026-08-19"
```

Its `## Resolution` section names the delivering commit and the falsifying test that was
added; `plans/PLAN-ISSUE-082-tool-allowlist-unreachable-entries.plan.yml` sits at
`status: completed` with the claim-by-claim evidence.

**`resolved-by: <ref>`** — the lighter path, for something fixed *directly* with no plan
lineage. The value is either a typed artifact ref (`BUNDLE`/`SPEC`/`ISSUE`/`PLAN`/`DIR-NNN`)
or a commit SHA / PR URL. A valid `resolved-by` plus a Resolution section satisfies the
close without the issue's own requirements, without a mandated test, and without a backing
plan. `issues/ISSUE-011-artifact-new-issue-wrong-extension.issue.md` closes this way with
`resolved-by: c59c951`.

Reach for `resolved-by` when the fix was a genuine one-liner that no test lineage would
meaningfully cover, or when the work was absorbed by some other artifact. Reach for
`delivered_by` for anything that earned a plan. Forcing plan ceremony onto a trivial fix
just to make the close look formal is the failure mode `resolved-by` exists to prevent.

---

## Validation is the enforcement

An artifact is not real until it validates.

```bash
backstop artifact validate --issue ISSUE-082   # one artifact
backstop artifact validate --spec              # all specs
backstop artifact validate --all               # everything
```

Exit `0` all pass, `1` violations, `2` config error. A clean run reports the schema each
file was checked against and the schema cohort hash:

```
✓ All checks passed
artifacts asserted: 1
scanned root: /path/to/repo
  /path/to/repo/issues/ISSUE-082-....issue.md [issue] issue/v1@d5fb0b75...
```

Schemas live at `artifacts/<type>/v<N>/schema.json` (embedded in the binary). A minimal
issue, for instance, needs a `title` and `schema_version: issue/v1` at the top level, an
`issue:` block carrying `id`/`title`/`type`/`status`/`created`, and a `## Problem` section
— everything else (`Solution`, `Verification`, `Resolution`, `References`) is optional
until a close requires it.

**Hand-editing artifacts is how this layer rots.** The schemas are strict and the
frontmatter is interlocked across artifacts; freeform prose drifts from it quickly. Route
authoring through the purpose-built agents (`/bundle`, `/spec`, `/plan`, `/implement`) and
let `artifact validate` be the check, not the author.

### The gate reads the artifacts too

Once artifacts exist, `backstop gate` gains dimensions that have nothing to do with your
source code's lint status:

- `artifact_validation` — the schema check above, run as part of the gate.
- `artifact_status_drift` — catches the broken promise: a spec marked `implemented` whose
  mandated test does not exist. It also warns in the other direction (delivered-but-open).
- `requirement_traceability` — requirements and claims actually trace back to their source.
- `ledger_integrity` — the numbering ledger is internally consistent.

That is the payoff for the whole layer: marking something done becomes a falsifiable
assertion the gate can contradict, rather than a status field nobody re-reads.

---

## The one rule that matters

**No implementation without a validated plan.** "Fix ISSUE-NNN" means plan it first, then
implement against the plan. A session editing source with no in-flight plan artifact is
off-track no matter how small the fix looks — that is precisely the state where an agent's
work becomes unreviewable.

If that rule feels heavier than your team wants, use the gate and skip this page. The two
layers are genuinely independent.
