---
title: "Pack Recipe Capability"
number: DIR-019
created: "2026-07-14"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-015"
    - "SPEC-054"
    - "ISSUE-079"
    - "ISSUE-081"
    - "ISSUE-080"
    - "ISSUE-085"
    - "ISSUE-110"
    - "ISSUE-119"
    - "SPEC-067"
    - "PLAN-SPEC-067"
---

## Description

Build the **pack scaffolding-recipe capability**: packs carry scaffolding
recipes (a template plus a pack-declared target path), and core applies them
via a generic, language/platform-blind mechanism — copy template to
pack-declared target, nothing more. No recipe logic, target-path convention,
or platform knowledge is baked into backstop itself; every recipe's shape and
destination is entirely pack-declared. This preserves the thin-executor
invariant (zero baked language/tool knowledge) that `backstop/self` already
enforces — a baked recipe mechanism would be exactly the kind of violation
that invariant exists to catch.

Two consumers of this capability:

1. **Language project scaffolding** — e.g. `pack add ts` bootstraps a starter
   TypeScript project via its recipe; the same mechanism works for any future
   language pack with zero core changes.
2. **CI recipe pack** — per-platform gate-workflow templates (GitHub Actions,
   etc.) that `backstop init` applies by default to wire CI gating out of the
   box.

**This directive is a blocking dependency for DIR-002 (`backstop init`
Command).** BUNDLE-003 resolved (all 7 OQs) that `backstop init` bakes zero
language/platform knowledge itself — it delegates all project scaffolding and
CI gate-workflow templating to packs via this recipe mechanism. `backstop
init`'s spec cannot be written until the recipe capability it depends on
exists, so DIR-019 must land — at least through a spec — before DIR-002's
implementation proceeds.

This directive is sourced from BUNDLE-015 (pack-scaffolding-recipes), which
has since been promoted by the founder to `defined` maturity, and SPEC-054
has been written and implemented against it (see below). The bundle still
carries scope this directive has not yet delivered — the CI recipe pack
consumer in particular (see Notes) — so the directive remains `active`
pending that follow-on work.
**Correction (2026-08-12):** the CI recipe pack consumer is no longer
open scope — it is DELIVERED. SPEC-067 (`status: implemented`) specced it;
`PLAN-SPEC-067` (`status: completed`) implemented it; the pack it specifies
is published and live at `backstop-ai/ci-workflows@v0.1.0` (real GitHub
tag) and is installed into backstop-core itself from that source
(`backstop.yml`, `backstop.lock`) — a real external consumer, not a
fixture. This directive's DIR-002 blocking-dependency claim, two paragraphs
above, is correspondingly CLEARED: the recipe capability DIR-002's spec
depends on now exists with a real CI-gate-workflow consumer proving it end
to end, not just the language-scaffolding mechanism SPEC-054 proved. What
remains open under this directive after this delivery is smaller residual
work — see the Notes corrections below for the current, accurate picture.

### Delivery status

SPEC-054 (`implemented`) shipped the MECHANISM: `pkg/recipe` + the `recipe
apply` CLI, resolving a recipe by `<pack>:<recipe>@<recipe_version>` and
applying its declared ops against a pack-declared manifest. First dogfood of
that mechanism surfaced two defects that must close before it is trustworthy
for the CI-recipe-pack consumer: ISSUE-079 (risk: critical — `recipe apply`
substitutes only payload/fragment CONTENT, never the site fields
(Op.Target/Anchor/Snippet), so it silently writes literal `{{ }}` paths and
still exits 0 — silent wrong output, not a crash), and ISSUE-081 (the
authoring surface itself is underspecified — merge fragment form, `--param`
CLI input, and insert splice semantics are all unpinned). Mechanism-complete,
not yet trustworthy.

## Notes

**Delivered so far:** SPEC-054 covers BUNDLE-015 REQ-001..010/021/023/024 —
13 of 24 REQs. The remaining 11 are uncovered, including REQ-018 (the CI
recipe pack — this directive's own packs-only acceptance test). Zero packs
in the published fleet declare a `recipes:` block in their `pack.yml` yet.
With remote pack consumption (DIR-026) and the fleet migration (DIR-027)
now underway, recipes are the sole remaining tier-1 launch long-pole.
**Update (2026-07-29):** the "zero consumers" claim above is now stale —
`backstop-ai/go-distribution` declares `recipes:` (`pack.yml:56-57`,
`go-release: recipes/go-release`) and is the capability's first real
consumer; see the correction below for what that does and does not close.
**Correction (2026-08-12):** REQ-018 — named above as one of the 11
remaining REQs — is no longer uncovered. It is DELIVERED: SPEC-067
(`status: implemented`) + `PLAN-SPEC-067` (`status: completed`) built the
CI recipe pack, published and live at `backstop-ai/ci-workflows@v0.1.0`,
installed into this repo's own `backstop.yml`/`backstop.lock`. Coverage is
now 14 of 24 REQs; see the corrections further below in these Notes for
what remains of the other 10.

Added to BACKLOG.yml immediately before DIR-002, since DIR-002 depends on
it, but after DIR-001 (Release Workflow) and DIR-003 (Baseline
Implementation) — both are also `backstop init` prerequisites and were
already prioritized ahead of DIR-002. This placement is a defensible default,
not a unilateral reprioritization; the founder should reposition if they
disagree with where this lands relative to other queued work.

The repo's `recipes/` directory currently contains three EMPTY subdirectories
(`go`, `meta`, `typescript` — each just a `.gitkeep`), and no pack in the
installed fleet declares a `recipes:` block in its `pack.yml`. The apply
mechanism is complete (modulo ISSUE-079/081) but has zero consumers:
BUNDLE-015's "CI recipe pack" consumer — the default-CI-wiring story that
`backstop init` depends on — remains unbuilt.
**Correction (2026-07-29):** "has zero consumers" is no longer accurate as
written — `backstop-ai/go-distribution` now declares `recipes:` and is the
capability's first real consumer, so the mechanism is proven outside core.
That pack's payload is a RELEASE workflow, not the CI gate workflow this
paragraph and BUNDLE-015 REQ-018 call for, so REQ-018 itself remains
unbuilt and core's own `recipes/` is still the three empty `.gitkeep` dirs
described above. Also note: the go-distribution pack's own directive home
is UNRULED (escalated to Brandon under ISSUE-101) — it is not claimed as a
deliverable of this directive.
**Correction (2026-08-12):** REQ-018 is no longer unbuilt — it is
DELIVERED. SPEC-067 (`status: implemented`) + `PLAN-SPEC-067`
(`status: completed`) built the CI gate-workflow recipe pack this paragraph
was waiting on; it is published and installed as
`backstop-ai/ci-workflows@v0.1.0`. Note the delivery shape: like
go-distribution, the CI recipe pack lives in its OWN external pack repo,
not inside core's `recipes/` directory — core's `recipes/go`/`meta`/
`typescript` remain the three empty `.gitkeep` placeholders described
above, and that is now understood to be correct/expected rather than a gap,
consistent with this project's "packs live outside core" invariant. Do not
read the empty core `recipes/` dirs as evidence REQ-018 is still open.

ISSUE-080's original two-problem report split on reconciliation (2026-07-26):
the shared `ExitViolations`/`main.go` stderr-suppression root cause (dropped
`manual:` fallback, silenced diagnostics) was fixed generically, repo-wide, by
SPEC-055 (`reportError`/`Explained` in `cmd/backstop/main.go`) — see SPEC-055
for that half's delivery. What remains open under ISSUE-080, and is now cited
by this directive, is the recipe-specific remainder: `recipe apply` silently
clobbers an operator's manually-diverged edit to a recipe-owned file when the
guarding `@waiver:` token carries a malformed reason code — a data-loss defect
in `pkg/recipe/apply.go`'s waiver-divergence adjudication that `reportError`
never touches, since it fails silently at exit 0.
**Correction (2026-08-02):** ISSUE-080 is now `closed` (delivered by
`PLAN-ISSUE-080`, commit `d222975`, 2026-08-02). The data-loss defect
described in this paragraph is fixed: `preserveOrRegenerate`
(`pkg/recipe/apply.go:392-399`) now REFUSES on an uncovered divergence
carrying a malformed waiver diagnostic — it returns an error naming the file
and the unparseable `@waiver` line and writes nothing; the silent-clobber-
at-exit-0 branch this paragraph describes no longer exists in the code. The
error reaches `cmd/backstop/recipe_apply.go:85` and becomes `ExitViolations`,
so exit is 1, not 0. Guard test
`TestRecipeApply_CLI_MalformedWaiverTokenSurfacesDiagnosticOnStderr`
(`cmd/backstop/recipe_apply_divergence_e2e_test.go:169`) drives the built
binary and PASSES (re-run 2026-08-02). This paragraph is left in place as
the historical problem statement the fix closed — see the further
correction below on the "Sequencing update" paragraph for what this means
for this directive's priority ordering.

**Sequencing update (2026-07-27) — ISSUE-079 closed, ISSUE-081 Gaps 1-2
delivered.** The prior sequencing instruction ("ISSUE-079 next, ahead of
ISSUE-081") is spent. ISSUE-079 (risk: critical — untemplated site fields,
silent wrong output) is `closed`; `PLAN-ISSUE-079` completed the fix.
ISSUE-081's Gap 1 (`fragment:` form — DECIDED: path-only, recipe-directory-
relative) and Gap 2 are delivered: `PLAN-ISSUE-081` completed, landing in
`8c80b2c` (fragment path-only contract + repeatable `--param`) and `da9d599`
(SPEC-054 v1.5.0 rewriting CLM-071). ISSUE-081 correctly stays `open` for
Gap 3 alone — insert placement semantics (inline-after-anchor vs.
new-line-after-anchor-line) are unpinned; the issue records founder-ratified
positions for Gaps 1 and 2 with Gap 3 explicitly left open, plus a residual
`Op.Payload` facet. What's genuinely next under this directive is no longer
"ISSUE-079 ahead of ISSUE-081" — it's ISSUE-080's recipe-specific remainder
(above), now this directive's **highest-severity open item** since it's
silent data loss (regenerates a diverged file and exits 0 with no warning),
ahead of ISSUE-081 Gap 3, which is authoring-surface polish by comparison.
**Correction (2026-08-02) — ISSUE-080 delivered; "highest-severity open
item" designation retired.** ISSUE-080 closed 2026-08-02 (`PLAN-ISSUE-080`,
commit `d222975` — code evidence on the ISSUE-080 paragraph above). It is no
longer open and is no longer a candidate for "next up" under this
directive; a reader should not reach for it.

Re-surveyed 2026-08-02 against this directive's actual `source:` list, by
each cited issue's status on disk: ISSUE-079 closed, ISSUE-080 closed,
ISSUE-085 closed. Open: ISSUE-081 (`status: open`, `risk: moderate` —
narrowed to Gap 3 alone, insert placement semantics, plus a residual
`Op.Payload` facet; Gaps 1 and 2 delivered by `PLAN-ISSUE-081`) and
ISSUE-110 (`status: open`, `risk: safe` — recipe substitution has no escape
syntax for foreign `{{ }}` templates, filed 2026-07-29). Between the two,
ISSUE-081 carries the higher declared risk (`moderate` vs. `safe`), so
**ISSUE-081 Gap 3 is now the highest-severity item among this directive's
open cited issues.**

But the largest uncovered scope under this directive is not an issue at
all: **BUNDLE-015 REQ-018 (the CI recipe pack — this directive's own
packs-only acceptance test) remains unbuilt.** Verified 2026-08-02:
`recipes/` in this repo still contains only three empty `.gitkeep`
placeholders (`go/`, `meta/`, `typescript/` — unchanged since the
"Correction (2026-07-29)" note above), and no pack in the fleet ships a CI
gate-workflow recipe — `backstop-ai/go-distribution`'s `recipes:` block is a
RELEASE workflow, not the CI gate workflow REQ-018 calls for (see that same
correction). REQ-018 is what this directive's own Description names as the
blocking dependency for DIR-002 (`backstop init`): "`backstop init`'s spec
cannot be written until the recipe capability it depends on exists." So,
stated plainly: the highest-severity **open issue** under this directive is
ISSUE-081 Gap 3; the largest **open scope gap**, and the one actually
gating DIR-002, is BUNDLE-015 REQ-018.
**Correction (2026-08-12):** the "largest uncovered scope" / "remains
unbuilt" characterization above is stale. BUNDLE-015 REQ-018 is DELIVERED —
SPEC-067 (`status: implemented`), `PLAN-SPEC-067` (`status: completed`),
published pack `backstop-ai/ci-workflows@v0.1.0`, installed into this
repo's own `backstop.yml`/`backstop.lock`. The DIR-002 blocking dependency
this paragraph describes is CLEARED as a result — DIR-002's spec is no
longer waiting on this directive for a proven CI-gate-workflow recipe
consumer. What remains open under this directive is now smaller: ISSUE-081
Gap 3 (`status: open`, plan `ready` but not implemented), ISSUE-110
(`status: open`, plan `draft` but not implemented), and ISSUE-119 (filed
2026-08-11, open, small — SPEC-067's own named follow-on). None of these
is REQ-018-sized; see those issues' own paragraphs below for current
status.

**ISSUE-085 (recipe-pack archetype gap) — delivered 2026-07-27.** A pack
whose whole point is handing out scaffolds + recipes with no ruleset had no
valid archetype: `pkg/packval`'s model recognized only `code` (requires
rules) and `enforcement` (forbids scaffolds), so a recipes-first pack
satisfied neither branch. This was latent behind the pre-SPEC-055 nil-
`Validator` skip (the retired ISSUE-073) and was EXPOSED by SPEC-055 REQ-008
making `pack check`/`pack test` validation unconditional on every `pack
add` — turning a `pack check`-time gap into an install-time hard blocker. It
sat on REQ-018's critical path: BUNDLE-015's CI recipe pack — this
directive's own packs-only acceptance test — is precisely this pack shape
(scaffolds + recipes, no rules) and would have hit the same wall. DIR-026
caused the exposure (REQ-008 shipped there); DIR-019 owned the fix, since
it's this directive's archetype/recipe-capability surface. Founder-decided
2026-07-27: direction 1 — a new `recipes`/scaffolding archetype, WITH TEETH:
every non-templating (`scaffolding`- and `implementing`-kind) recipe in the
pack must declare its own `enforcement.rules`; `templating`-kind recipes are
exempt, since the applier itself is the drift enforcement for the other two
kinds. **Delivered**: `PLAN-ISSUE-085` completed and ISSUE-085 is `closed`.
Commits: `0c729bf` (TDD red, acceptance fixtures), `253c501` (phase 2 —
recipes archetype lands at both parse seams), `419b71a` (phase 3 — clears
the four dormant findings the diff scope activated), `40fc878` (close
delivered).

**ISSUE-110 (recipe substitution has no escape syntax) — filed 2026-07-29,
open.** `pkg/recipe/Substitute` (`pkg/recipe/substitute.go:31-62`) reads
EVERY `{{ ... }}` span as a param NAME, with no escape form. An undeclared
name returns an error and NO string — the apply hard-fails rather than
emitting partial output. That is deliberate (the doc comment's
"deliberately NOT Turing-complete" framing, REQ-002), but it leaves any
payload that must emit a FOREIGN `{{ }}` template verbatim — GitHub Actions
`${{ }}`, goreleaser `{{ .X }}`, Helm, Jinja — with no first-class way to do
it. The shipped workaround, proven in production by the
`backstop-ai/go-distribution` pack: declare self-emitting pass-through
params, one per foreign span, where the param NAME is the inner text and
its DEFAULT is that same text re-wrapped in delimiters (e.g. name
`.Version`, default `{{ .Version }}`). Eight such params exist in that
pack. It works only because substituted values are never rescanned
(`substitute.go:22-23`). It carries a live hazard: a caller passing
`--param .Version=anything` rewrites the emitted foreign template instead
of leaving it alone. NEW MEASURED FINDING, not previously recorded anywhere
in this directive: the substituter is not comment-aware. It byte-scans the
raw template for `{{`/`}}` regardless of `#` YAML comments or any other
context, so a payload COMMENT that merely mentions a template in prose
hard-fails the apply with the same "unresolvable placeholder" error as a
real unresolved substitution. This cost a real debugging cycle; the
in-production workaround is at
`recipes/go-release/payload/release.yml:15-25` in the go-distribution pack,
which writes the reference delimiter-less in prose with a note explaining
why. Fix directions the issue records for a future spec/plan, none chosen:
(1) a first-class escape form (triple-brace or backslash), (2) comment-
aware skipping (narrower, and language-syntax-aware in a way the
substituter deliberately is not), (3) do nothing and sanction the
pass-through-param idiom as documented authoring surface.
SEQUENCING/CONSOLIDATION NOTE: ISSUE-110 is adjacent to ISSUE-081 but
distinct. ISSUE-081's remaining scope is Gap 3 alone (insert placement
semantics) plus the residual `Op.Payload` facet; ISSUE-110 is a
language-level gap in the substitution GRAMMAR with its own shipped
workaround. The issue itself flags that the founder may prefer to
consolidate it as ISSUE-081 "Gap 4" — record that as an OPEN founder call,
do not decide it. Either way the work sits under this directive. PRIORITY
within this directive, stated plainly: ISSUE-110 is authoring-surface
polish, ranking BELOW ISSUE-080's recipe-specific remainder (this
directive's highest-severity open item — silent data loss at exit 0), and
roughly peer to ISSUE-081 Gap 3.
**Correction (2026-08-02):** the "ranking BELOW ISSUE-080's recipe-specific
remainder (this directive's highest-severity open item...)" clause above is
stale — ISSUE-080 closed 2026-08-02 (`PLAN-ISSUE-080`, commit `d222975`;
see the corrections on the ISSUE-080 paragraph and the "Sequencing update"
paragraph above for the evidence). With ISSUE-080 out of the picture, this
paragraph's own relative ordering still holds: ISSUE-081 Gap 3
(`risk: moderate`) ranks above ISSUE-110 (`risk: safe`), and ISSUE-081 Gap 3
is accordingly the highest-severity item among this directive's open cited
issues. The larger open item overall is BUNDLE-015 REQ-018 (the CI recipe
pack, unbuilt) — see the "Sequencing update" paragraph's correction above.
**Correction (2026-08-12):** the "larger open item overall is BUNDLE-015
REQ-018 ... unbuilt" clause immediately above is itself now stale —
REQ-018 is DELIVERED (SPEC-067 `implemented`, `PLAN-SPEC-067` `completed`,
published pack `backstop-ai/ci-workflows@v0.1.0`; see the Description
correction and the "Sequencing update" paragraph's 2026-08-12 correction
above). With REQ-018 delivered, this paragraph's ISSUE-081-Gap-3-vs-
ISSUE-110 ranking is unaffected and still holds — ISSUE-081 Gap 3 remains
the highest-severity item among this directive's open cited issues; there
is just no larger REQ-018-sized item still standing above it.

**ISSUE-119 (recipe brownfield adoption silently wires no CI gate) — filed
2026-08-11, open, type: enhancement, scope contained / uncertainty
exploratory / risk moderate.** Surfaced during implementation of SPEC-067
(the CI recipe pack — BUNDLE-015 REQ-018, at the time this directive's
largest open item; SPEC-067 has since shipped — `status: implemented`,
`PLAN-SPEC-067` `status: completed`, published pack
`backstop-ai/ci-workflows@v0.1.0` — see the Description correction above),
which named it as a deliberate follow-on rather than improvising a
fix mid-implementation: Sharp Edge 2 of
`specs/SPEC-067-ci-recipe-pack.spec.md`, and the Scope Fences + TASK-031 of
`plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml`. The defect: three of the CI
pack's four platform targets — `.gitlab-ci.yml`, `bitbucket-pipelines.yml`,
`Jenkinsfile` — are each their platform's ONLY conventional CI entry point,
so any brownfield consumer that already has one hits `create`'s
never-clobber rule. The apply reports `preserved … (the consumer's own
file)`, and the consumer walks away with an adoption record and NO backstop
gate in CI. Correct non-destructive behavior, genuinely bad outcome. Init's
greenfield case is unaffected.
MEASURED CORRECTION to the issue's own framing — record this plainly, it
changes what the eventual spec/plan should do. The issue is titled "Recipe
payloads have no merge/insert op" and its fix menu item 1 proposes adding "a
`merge` or `insert` op kind". That premise is FALSE at the core level.
`pkg/recipe` already ships a CLOSED five-member op allowlist — `create`,
`merge`, `transform`, `insert`, `step` (`pkg/recipe/manifest.go:26-31`) —
and `Apply` dispatches four of them for real (`pkg/recipe/apply.go:174-187`;
`applyCreate` at :265, `applyInsert` at :639, `applyMerge` at :877; only
`step` is reserved-not-executed, BUNDLE-019's). `merge` is fragment-based
and additive by contract (see the `Op.Fragment` doc note and `ApplyAll`'s
"merge is additive, so a co-write is composition rather than a conflict").
So the real gap is NOT a missing core capability — it is that SPEC-067's
REQ-003 deliberately PROHIBITED reaching for those ops in the initial cut
("a gate workflow is a whole file the recipe owns, and an op family that
edits a consumer's existing file would put a recipe-owned promise inside
consumer-owned bytes"). The work is a PACK-authoring + policy decision on
top of shipped core mechanism, not a core op-kind build. Scope it that way.
Dependency this directive should sequence on: a brownfield merge/insert
variant lands squarely on ISSUE-081 Gap 3 (insert placement semantics
unpinned) — `applyInsert` splices at the byte offset immediately after the
anchor's last matched character with no line-boundary consideration, so an
insert of a CI job/step under an anchor currently lands INLINE. Gap 3 now
has a `ready` plan
(`plans/PLAN-ISSUE-081-insert-placement-semantics.plan.yml`, created
2026-08-11). ISSUE-081 Gap 3 should land BEFORE ISSUE-119 is specced, or
ISSUE-119 will re-derive the same placement question. Also relevant and
already recorded in ISSUE-081: SDLC-mediated mode is CLI-unreachable
(`cmd/backstop/recipe_apply.go:142` hardcodes `Mode: recipe.ModeDirect`),
so every insert/transform site through the bare CLI is the recipe's own
declared `target`/`anchor` — a brownfield variant driven from `backstop
init` inherits that constraint.
Second facet, independent of the op question and cheaper: the issue's item
2 asks whether `create`'s never-clobber `preserved` outcome should surface
LOUDER — a distinct warning that no gate was actually wired. Note that core
already declines to record an adoption when nothing was materialized
(`Apply`'s `own.materialized` guard, `pkg/recipe/apply.go`), so the honesty
signal partly exists in the data; what is missing is the operator-facing
surfacing. This facet is separable and could ship without the op work.
Record it as such, take no position on which ships first.
Third facet, an OPEN founder call — do NOT decide it: the issue's item 3
asks whether this belongs to the recipe-capability layer generically (ANY
`scaffolding`-kind recipe targeting a platform's singular conventional
entry point has this problem) or is CI-recipe-pack-specific. Record it as
an open question for whoever picks it up; note that the generic reading
keeps it here under DIR-019 either way, so the home is not in question —
only the scope is.
PRIORITY within this directive, stated plainly: ISSUE-119 ranks BELOW
BUNDLE-015 REQ-018 / SPEC-067 itself (it is that work's own follow-on and
is meaningless before it lands) and BELOW ISSUE-081 Gap 3 (its
dependency). It is roughly peer to ISSUE-110 (authoring-surface polish) but
carries higher consequence, since the failure mode is a consumer who
believes CI enforcement is live when it is not.
**Correction (2026-08-12):** SPEC-067 has now landed (`status: implemented`,
`PLAN-SPEC-067` `status: completed`, published pack
`backstop-ai/ci-workflows@v0.1.0` — see the Description correction above),
so the "meaningless before it lands" qualifier no longer applies — ISSUE-119
is now actionable in principle. Its ranking below ISSUE-081 Gap 3 still
holds, since Gap 3 (insert placement semantics) is this issue's own recorded
dependency and remains unimplemented (plan `ready`, not yet built).

**Status assessment (2026-08-12).** With REQ-018 delivered, this
directive's largest scope gap is closed and its DIR-002 blocking
dependency is cleared (see the Description correction). That is not,
however, grounds to self-close this directive to `done`: three real,
unimplemented items remain under its `source:` list — ISSUE-081 Gap 3
(insert placement semantics; plan `ready`, i.e. planned and reviewed, but
NOT implemented), ISSUE-110 (foreign-template escape; plan `draft`, i.e.
through two review-fix rounds but never promoted past draft, and NOT
implemented), and ISSUE-119 (brownfield-merge-gap; filed 2026-08-11, open,
small, and itself blocked on ISSUE-081 Gap 3 landing first). All three are
in-flight or queued work, not a backlog of ideas — that is squarely what
`active` means. `queued` would understate progress already shipped
(SPEC-054, SPEC-067, ISSUE-079/080/085 all delivered); `specced` would
misdescribe a directive whose specs are already implemented and whose
remaining work is issue-level cleanup, not a not-yet-implemented spec; and
`done` would be a self-close this correction is explicitly not authorized
to make while ISSUE-081/110 have real, unimplemented plans. **Status
remains `active`** — this is a characterization correction, not a status
transition.
