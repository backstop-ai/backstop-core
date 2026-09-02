---
title: Reference
layout: default
permalink: /reference/
hero_question: "What exact interface or behavior do I need?"
---

## Compatibility {#compatibility}

<!-- backstop-claim: CLAIM-013 -->
Backstop exposes a repository-oriented CLI, YAML artifacts, JSON-capable output, and process exit codes. A runtime integration must invoke the pinned binary from the intended working directory and propagate blocking exits. That proves operability; artifact order and role separation require additional harness behavior.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-006 -->
A harness can run the binary. That only proves the binary runs. Artifact order, review separation, and stopping on red are extra work the harness has to do.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-021 -->
Invocation proves the process started. A trustworthy integration separately proves the harness kept the order and stopped on failure.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-012 -->
Runtime-neutral does not mean runtime-guaranteed. Backstop cannot make a harness respect a result it chooses to discard.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-009 -->
[Follow compatibility guidance](/status/#adjacent-guidance)

## Artifact schema catalog {#artifact-schema-catalog}

<section data-generated-region data-product-truth-job="artifact-schema-catalog">
<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=artifact-schema-catalog -->
{% include generated/artifact-schema-catalog.md %}
<!-- PRODUCT-TRUTH-INCLUDE:END job=artifact-schema-catalog -->
</section>

<!-- backstop-claim: CLAIM-014 -->
Artifact schemas live under `artifacts/<kind>/v1/schema.json`. The CLI reserves IDs, scaffolds valid records, validates references and state, and excludes terminal historical artifacts where policy declares that behavior.
<!-- /backstop-claim -->

## Configuration {#configuration}

<!-- backstop-claim: CLAIM-026 -->
The repository root contains `backstop.yml`. Its `project`, `packs`, and `enforcement.policy` fields declare versioned standards and policy levels. The lock under `.backstop/` binds resolved pack bytes.
<!-- /backstop-claim -->

## CLI conventions {#cli-conventions}

<!-- backstop-claim: CLAIM-027 -->
Backstop commands return `0` for success, `1` for blocking violations or broken promises, and `2` for configuration failure. The global `--json` flag selects structured output where supported, while `backstop commands` provides machine-readable command discovery.
<!-- /backstop-claim -->

## Troubleshooting {#troubleshooting}

<!-- backstop-claim: CLAIM-028 -->
`backstop doctor` diagnoses configuration discovery and loading, Git repository state, installed packs, build identity, toolchain execution, engine-tool availability, and artifact layout. A failing check returns `1`; an unknown check identifier is a configuration error.
<!-- /backstop-claim -->

## Gate {#gate}

<!-- backstop-claim: CLAIM-015 -->
`backstop gate` evaluates changed files by default. `backstop gate --file <path>` bounds verification explicitly, while `backstop gate --all` runs the repository-wide kill chain. Exit zero means every blocking step passed; a nonzero exit is a verdict to preserve, not suppress.
<!-- /backstop-claim -->

## Pack commands {#pack-commands}

<!-- backstop-claim: CLAIM-016 -->
Use `backstop pack install`, `backstop pack list`, `backstop pack update`, and `backstop pack relock` to manage declared versions and resolved locks. Inspect findings before accepting an update.
<!-- /backstop-claim -->

## Pack artifact {#pack-artifact}

<!-- backstop-claim: CLAIM-029 -->
A pack declares identity, version, claims, engines, fixtures, tools, applicability, and findings. Commands are direct argv contracts; tool pins are trust declarations rather than installers.
<!-- /backstop-claim -->

## Artifact lifecycle and closure {#artifact-lifecycle-and-closure}

Status is a declared field. Authors set it. It is not inferred from git or CI. Per-noun Status tables live on the entity pages. This section is the machine: legal moves, what must be true, what starts being enforced, and what that move enables.

<!-- backstop-claim: CLAIM-030 -->
Bundles progress through `idea`, `exploring`, `defined`, and `ready`; specifications progress from `draft` to `ready-for-implementation`; plans progress from `draft` to `ready` to `implementing` to `completed`. Terminal states distinguish delivered or implemented work from replacement, cancellation, deprecation, and obsolescence, with successor fields required where the schema names one.
<!-- /backstop-claim -->

Feature work uses bundle → spec → plan. Bounded work uses issue → plan. Both tracks meet at a plan. Each spec has exactly one plan. An issue does not get a spec. Product code is not written until a plan is approved.

### Issue

Live: `open` → `ready` → `in-progress`. `blocked` waits on named work.

| State | Before you can enter it | Validator / gate | Enables |
| --- | --- | --- | --- |
| `open` | Scaffolded issue, Problem section | Schema only. Requirements and claims are optional. | Filing. Not a plan yet. |
| `ready` | Requirements, claims, verification, implementation, and contracts | Full spec-parity traceability. | Creating a plan from the issue. |
| `in-progress` | Same rigor as `ready` | Same. | A plan is in flight, or close-out is underway. |
| `blocked` | Same rigor as `ready`, plus `blocked_by` | Same, plus the blocker must be named. | Waiting without pretending the work is moving. |
| `closed` | `## Resolution`, and exactly one close pointer unless the issue carries its own requirements and claims | Close traceability. | The issue is finished. |

<!-- backstop-claim: CLAIM-031 -->
Closing an issue requires a `## Resolution` section and at most one traceability field: `delivered_by` names a completed plan, while `resolved-by` names a direct typed artifact reference, commit SHA, or pull-request URL when no plan lineage applies.
<!-- /backstop-claim -->

`delivered_by` and `resolved-by` on the same close is illegal. `replaced` requires `replaced-by`. `obsoleted` requires `obsoleted-by`. `canceled` is abandoned work.

### Spec

Live: `draft` → `ready-for-implementation` → `implemented`.

A spec comes from a bundle. Live specs carry requirements, claims, verification, implementation, and contracts. Entering `ready-for-implementation` is what makes planning legal. `implemented` means the work was delivered; the file stays a full spec.

Terminals: `replaced`, `canceled`, `deprecated`, `obsoleted`. Retired specs are exempt from live-work completeness.

### Bundle

Live: `idea` → `exploring` → `defined` → `ready`. Success terminal: `delivered`.

The user drives promotion. Do not self-promote.

| State | Before you can enter it | Validator / gate | Enables |
| --- | --- | --- | --- |
| `idea` | Named bundle | Identity and maturity enum. | The work has a name. |
| `exploring` | Real open questions, unresolved | Same. | Exploration. Not specing. |
| `defined` | `problem.summary`, `problem.user_story`, `solution.approach`, `requirements[]`, and sections Draft Requirements, Draft Design Decisions, Spec Seeds, Version History | Maturity gates. Placeholders are illegal. | Approach is clear. |
| `ready` | Everything `defined` requires, plus `problem.success_criteria` and `solution.assumptions` | Same, tighter. | Spec generation. |
| `delivered` | The work shipped. `requirements[]` still required. | Success terminal. | Closure of the bundle. |

Terminals without delivery: `replaced`, `canceled`, `deprecated`. `replaced` requires `replaced-by`.

### Plan

Live: `draft` → `ready` → `implementing` → `completed`.

`draft`, `ready`, and `implementing` are the same shape to the validator: phases, tasks, tests before implementation, every source claim mapped. Terminal plans (`replaced`, `canceled`, `obsoleted`) are exempt from phase and task completeness.

| State | Before you can enter it | Validator / gate | Enables |
| --- | --- | --- | --- |
| `draft` | Source is an issue or a spec. Phases and tasks are present. | Plan schema and completeness. | Authoring. |
| `ready` | Same shape. Reviewer has passed. | Same. | Implementation. An implementer may execute it. |
| `implementing` | An implementer is executing it. | Same. | In-flight execution. |
| `completed` | The work was delivered. Mandated tests exist. | Validator may accept a completed plan whose mandated tests are missing. The gate does not. | Delivery evidence for `delivered_by`. |

`replaced` requires `replaced-by`. `obsoleted` requires `obsoleted-by`.

Directive, ADR, and Capability have their own status vocabularies on [Directive](/directive/), [ADR](/adr/), and [Capability](/capability/). They are not paths into implementation.

## Source traceability {#source-traceability}

<!-- backstop-claim: CLAIM-009 -->
A gate result is useful evidence only when its command, prerequisites, and captured artifact are named.
<!-- /backstop-claim -->

Durable references use repository paths, immutable commits, or published versions. Execution evidence includes the exact command and prerequisites; observation evidence identifies its checked-in artifact and provenance.

<!-- backstop-claim: CLAIM-007 -->
A literal Liquid-shaped `{% raw %}{{ ... }}{% endraw %}` expression is currently interpreted as an undeclared recipe parameter and rejected before any write; ISSUE-182 tracks the missing literal-escape capability.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-008 -->
Linux CI exposed a contracts-pack fixture false negative, and commit `f8b3846fe5d4c2bc6465efc6eb5e4594e1b591da` repaired it with checked-in before-and-after evidence.
<!-- /backstop-claim -->

## CLI command catalog {#cli-command-catalog}

<section data-generated-region data-product-truth-job="cli-command-catalog">
<!-- PRODUCT-TRUTH-INCLUDE:BEGIN job=cli-command-catalog -->
{% include generated/cli-command-catalog.md %}
<!-- PRODUCT-TRUTH-INCLUDE:END job=cli-command-catalog -->
</section>

<!-- backstop-claim: CLAIM-032 -->
The command families cover initialization, diagnosis, gates, packs, artifacts, recipes, baselines, waivers, version reporting, and command discovery. Use `--help` on the pinned release for exact flags and `--json` where a command exposes structured output.
<!-- /backstop-claim -->

<!-- backstop-journey-link: JLINK-023 -->
[Check release history](/status/#release-history)
