---
title: "Land Approved Adopt Visitor Page"
schema_version: issue/v1

issue:
  id: ISSUE-196
  title: "Land Approved Adopt Visitor Page"
  type: enhancement
  status: open
  created: "2026-09-01"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Land Approved Adopt Visitor Page

## Problem

`/adopt/` on `main` / live `https://backstop.sh/adopt/` is still the Seed 1
field-guide adoption page. SPEC-072 locked `required_blocks`
`[adoption-paths, install, configure, verify-enforcement]`, ADOPT-INSTALL /
ADOPT-CONFIGURE / ADOPT-ENFORCE command bytes, CLAIM-024 on
`#verify-enforcement`, JLINK-012 from `#install` → `/reference/#configuration`,
and JLINK-013 from `#verify-enforcement` → `/model/#enforcement-loop`.

Visitor review found that Seed 1 does the wrong job. The live page is a
catalog / disposable-repo install dump / "prove a violation blocks" smoke test
/ paste-this-to-the-agent brief. It does not answer the question a visitor
actually has: how to evaluate the complete Backstop model, in sequence, in a
real repository.

The approved page is a sequential evaluation of the complete Backstop model
that a human or an agent with web access can follow directly. No separate
copy/paste agent prompt. No separate human vs agent instructions. The page
itself is the authoritative instruction list.

A local prototype of the approved rewrite exists on
`cursor/evaluate-copy-local-preview-e296` (HEAD `2e20d33`, `draft(docs): restore
locked Adopt pack sentence`). It has not been pushed. This issue records the
approved direction so it can be planned and landed. The prototype is evidence
of intent, not a close and not a substitute for issue → plan.

ISSUE-193 owns `/evaluate/` only and lists Adopt as out of scope. ISSUE-195
owns `/model/` only (filed on `cursor/land-approved-model-visitor-page-2a53`,
tag `backstop/issue/195`) and lists Adopt as out of scope. This is that later
issue.

Do not `/plan` this issue in this filing. Plans for ISSUE-193, ISSUE-195, and
this Adopt issue happen later, together, when Brandon says so.

## Solution

Land the approved `/adopt/` visitor page: copy, paper/ink presentation shared
with Evaluate, Model, and the homepage, and the inventory / topology /
allowance / presentation / page-test / Seed 1 contract lockstep, including
Seed 1 contract literals in SPEC-072 / SPEC-075 / SPEC-076 and related
sitecheck expectations that still name the old Adopt heroes and hops.

Do not reopen BUNDLE-032's visitor-journey charter. Do not mark BUNDLE-032
delivered. Do not `/plan` this issue in this filing. Do not edit `docs/adopt.md`
or any visitor copy as part of filing this issue.

### Approved page contract

Visible structure of `docs/adopt.md`. Lock this structure. Do not rewrite or
restructure. Copy is locked. Quote the approved sentences; do not independently
"improve" them.

Hero (keep):

- `hero_question`: **Try it out.**
- `hero_lede`: **Install Backstop and see how the pieces work together.**
- Same pair in `docs/_data/site-presentation.yml` (`hero_question`) and
  `docs/_data/content-topology.yml`.
- Topology `required_blocks`: `[adoption-paths, install, configure, verify-enforcement]`
- New visible heading IDs also present: `#setup`, `#known-bug`, `#used-the-model`

1. `## 1. Set up Backstop {#setup}`
   - `### Install the binary {#install}`
     - `Pin the released binary in the repository.`
     - Exact command once:
       `GOBIN=./.backstop-bin go install github.com/backstop-ai/backstop-core/cmd/backstop@v0.2.0`
     - Hidden JLINK-012 in `.canonical-note`: Configure Backstop →
       `/reference/#configuration`
   - `### Initialize the repository {#configure}`
     - `Backstop configuration lives with the code.`
     - Exact command once: `backstop init`
     - `#configure` is an H3 under Set up, not a visual peer of the numbered
       sections.

2. `## 2. Start with your existing code {#adoption-paths}`
   - `Install a pack appropriate for the repository's stack. [Choose a pack](/packs/#choose-a-pack).`
   - `Run:` + `<pre><code>backstop baseline</code></pre>`
   - `Review what Backstop finds. Do not fix anything yet.`

3. `## 3. Try it on a bug you know {#known-bug}`
   - `Pick a known bug in the repository.`
   - `Use Backstop's lightweight artifact workflow for it:` then the only
     traditional bullet list:
     - Create an issue.
     - Create the corresponding plan.
     - Run the artifact reviewers and deterministic validators.
     - Resolve findings until the artifacts pass.
   - `[Artifact lifecycle](/reference/#artifact-lifecycle-and-closure)`

4. `## 4. Let the agent implement it {#verify-enforcement}`
   - `Assign the approved plan to an implementer agent. The agent executes the plan, runs required gates as it works, and fixes failures before proceeding.`
   - `Stop before merge.`
   - Do **not** show a standalone `backstop gate` smoke-test section. The
     implementer runs gates as required by the workflow. Seed 1 still requires
     the exact command bytes `backstop gate` once under `#verify-enforcement`
     (ADOPT-ENFORCE / CLI-002). The prototype keeps that
     `<pre><code>backstop gate</code></pre>` inside the visually hidden
     `.canonical-note` with CLAIM-024 and JLINK-013. Landing must keep command
     cardinality 1 without restoring a visible "Run the gate" smoke test.

5. `## You've now used the whole model. {#used-the-model}`
   - Own section, visually separated (border-top / extra margin), not the last
     sentence of step 4.
   - `Use the complete workflow, or only the parts you need.` (accent closer)

Intended experience: set it up → inspect an existing repo → use it on work you
already understand → let the agent execute → decide how much of Backstop you
want. The page itself is the authoritative instruction list.

### Rejected — do not restore

- Seed 1 catalog: "What does a first working adoption require?" +
  disposable-repo install dump
- Quick start as the page job (Install → Init → Prove as the story)
- Hero "Put one standard behind the gate." / "Install. Initialize. Prove a
  violation blocks."
- Hero "You don't have to take all of it."
- Hero "Start where it will pay you first." and lede "The pieces compose. Put
  the first one on the failure that already costs you."
- H2 "Start from what's already true." / "The rest can wait." / "Prove it
  blocks."
- H1 "Hand this to your agent."
- Heading "Evaluate Backstop in this repository" / a paste-this agent brief as
  the page job
- Decision matrix / capability fork as the page job
- Inventing a fourth locked command (`backstop pack add`)
- Slide-frame chrome
- Copy/paste agent prompt; separate human vs agent instructions
- Standalone visible "Run the gate" smoke test
- Step 4 copy that tells the reader to manually run `backstop gate`
- "Initialize Backstop" as a visual peer of numbered sections (it is a substep
  of Set up)
- Pack explainer on Adopt ("A pack is a versioned set of standards…") —
  Brandon's "Install a pack assumes a lot of knowledge / we will need to create
  more packs" was a **note to self**, not a copy change. Do not expand Adopt
  into a pack tutorial. Capture that as a known follow-on: catalog coverage is
  a Packs/product gap, not Adopt copy. Locked step-2 sentence stays "Install a
  pack appropriate for the repository's stack. [Choose a pack](/packs/#choose-a-pack)."
- Independent "improvement" of locked copy
- Aphorisms, slogans, manufactured paradoxes

### Presentation (paper/ink, shared with Evaluate, Model, and homepage)

ISSUE-193 / ISSUE-195 said other inner pages stay dark until their own issues.
This is Adopt's issue. Landing must give `/adopt/` the same paper/ink chrome as
Evaluate, Model, and the homepage, without regressing those pages.

Prototype already:

- `page_kind` `adoption` loads `backstop-tokens.css`
- `html:has([data-page-kind="evaluation"], [data-page-kind="model"], [data-page-kind="adoption"])`
  remaps `--ds-*`
- Nested H3s under step 1 are clearly smaller than numbered H2s
- `#used-the-model` has separated closer treatment (border-top / extra margin;
  accent closer)

When editing `site.css` media queries, do not drop
`[data-page-kind="home"] .nav` rules. Playwright `testMatch` stays
`public-site.spec.ts`. Packs stay external.

### Lockstep landing will need

Closed-world allowances in `scripts/verify-public-product-model.sh` for every
non-claim, non-jlink, non-heading block in `docs/adopt.md`. Heading text is
skipped.

Hero / presentation matrix: `docs/_data/site-presentation.yml`,
`docs/_data/content-topology.yml`, `scripts/verify-public-product-model.sh`
heroes list, `scripts/tests/public-product-model/verify-content-topology.sh`,
`scripts/sitecheck/presentation_test.go`.

Seed 1 still names old Adopt heroes/hops in SPEC-072 / SPEC-075 / SPEC-076
(ADOPT-INSTALL / ADOPT-CONFIGURE / ADOPT-ENFORCE owner anchors, JLINK-012 /
JLINK-013, `required_blocks`). Landing must update those contract literals the
same way ISSUE-193 / ISSUE-195 called out for Evaluate / Model. Do not update
SPECs in this filing — only record that landing will.

Public-site completeness requires `#adoption-paths`, `#install`, `#configure`,
`#verify-enforcement` visible with ≥80 chars of sibling text. `#configure` as
an H3 with short copy is tight against that 80-char rule; landing must keep
Brandon's sentence `Backstop configuration lives with the code.` and adjust
the completeness check if needed — do not pad the sentence.

### Scope constraint

This issue owns `/adopt/` only, plus the smallest corresponding inventory,
topology, allowance, presentation, page-test, and Seed 1 contract updates
needed to land that page without silent divergence from SPEC-072 / SPEC-075 /
SPEC-076.

Out of scope — later page-by-page issues, do not absorb:

- Homepage (ISSUE-190; canonical homepage stays)
- Evaluate (ISSUE-193)
- Model (ISSUE-195)
- Use Cases, Packs, Extend, Reference, Status, Contributing, and other
  inner-page copy
- Creating more packs / pack-catalog coverage (product gap; Brandon's note to
  self)
- Logo / hamburger / mobile nav for remaining inner pages
- Marking BUNDLE-032 delivered
- Vendoring packs
- Pushing the local prototype without a plan
- `/plan` of this issue, ISSUE-193, or ISSUE-195 in this filing

## References

- Local prototype: branch `cursor/evaluate-copy-local-preview-e296`, HEAD
  `2e20d33`. Primary source `docs/adopt.md`; CSS `docs/assets/css/site.css`;
  layout `docs/_layouts/default.html`.
- SPEC-072 Seed 1 Adopt `required_blocks` `[adoption-paths, install, configure,
  verify-enforcement]`, ADOPT-INSTALL / ADOPT-CONFIGURE / ADOPT-ENFORCE
  instructions, CLAIM-024, JLINK-012 (`#install` → `/reference/#configuration`),
  JLINK-013 (`#verify-enforcement` → `/model/#enforcement-loop`).
- BUNDLE-032 REQ-010: the bundle does not contain final copy; Seed 1 owned it.
  This issue revises Adopt's Seed 1 copy after visitor review. It is not a
  competing charter for the website expansion.
- ISSUE-190: homepage fence. This issue must not redesign `/`.
- ISSUE-192: architecture-pack classification of `scripts/websitejourney`;
  explicitly does not own inner-page visual redesign.
- ISSUE-193: Evaluate rewrite; lists Adopt as out of scope.
- ISSUE-195: Model rewrite (branch `cursor/land-approved-model-visitor-page-2a53`,
  tag `backstop/issue/195`); lists Adopt as out of scope. This issue is the
  Adopt sibling.
- Live site still old: `https://backstop.sh/adopt/`.

### Existence-in-world check

Before filing, `issues/` and `bundles/` were searched for Adopt visitor copy,
`/adopt/` rewrite, paper/ink Adopt restyle, and "Try it out."

- BUNDLE-032 / SPEC-072 own visitor-journey IA and Seed 1 Adopt. This issue
  does not reopen that charter. It records the approved rewrite of that one
  page after visitor review.
- ISSUE-190 owns canonical homepage restoration only. Must not redesign `/`.
- ISSUE-192 owns `scripts/websitejourney` classification; excludes inner-page
  visual redesign.
- ISSUE-193 owns `/evaluate/` only; lists Adopt as out of scope.
- ISSUE-195 owns `/model/` only (filed on
  `cursor/land-approved-model-visitor-page-2a53`, tag `backstop/issue/195`).
  Lists Adopt as out of scope.
- No open issue in this working tree owns the Adopt rewrite.
