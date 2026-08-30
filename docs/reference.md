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

<!-- backstop-claim: CLAIM-030 -->
Bundles progress through `idea`, `exploring`, `defined`, and `ready`; specifications progress from `draft` to `ready-for-implementation`; plans progress from `draft` to `approved` and `in-progress`. Terminal states distinguish delivered or implemented work from replacement, cancellation, deprecation, and obsolescence, with successor fields required where the schema names one.
<!-- /backstop-claim -->

<!-- backstop-claim: CLAIM-031 -->
Closing an issue requires a `## Resolution` section and at most one traceability field: `delivered_by` names a completed plan, while `resolved-by` names a direct typed artifact reference, commit SHA, or pull-request URL when no plan lineage applies.
<!-- /backstop-claim -->

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
