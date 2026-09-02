---
title: Reference
layout: default
permalink: /reference/
hero_question: "Exact interfaces, schemas, lifecycle rules, and integration behavior."
hero_lede: false
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
<div class="state-index" aria-label="Live states">
<p class="state-index-kicker">Live states</p>
<dl>
<dt>Bundle</dt>
<dd><code>idea</code> → <code>exploring</code> → <code>defined</code> → <code>ready</code></dd>
<dt>Spec</dt>
<dd><code>draft</code> → <code>ready-for-implementation</code> → <code>implemented</code></dd>
<dt>Issue</dt>
<dd><code>open</code> → <code>ready</code> → <code>in-progress</code><br>blocked waits on named work</dd>
<dt>Plan</dt>
<dd><code>draft</code> → <code>ready</code> → <code>implementing</code> → <code>completed</code></dd>
</dl>
<p class="state-index-note">Terminal states distinguish delivered or implemented work from replacement, cancellation, deprecation, and obsolescence. Successor fields are required where the schema names one.</p>
</div>
<!-- /backstop-claim -->

Feature work uses bundle → spec → plan. Bounded work uses issue → plan. Both tracks meet at a plan. Each spec has exactly one plan. An issue does not get a spec. Product code is not written until a plan is approved.

<div class="state-index state-coupling" aria-label="Coupled states">
<p class="state-index-kicker">Coupled states</p>
<dl>
<dt>Feature</dt>
<dd>bundle <code>ready</code> → spec <code>ready-for-implementation</code> → plan <code>ready</code> → plan <code>completed</code> → spec <code>implemented</code><br>bundle <code>delivered</code> is declared separately</dd>
<dt>Bounded</dt>
<dd>issue <code>ready</code> → plan <code>ready</code> → plan <code>completed</code> → issue <code>closed</code> via <code>delivered_by</code></dd>
</dl>
<p class="state-index-note"><code>delivered_by</code> requires a completed <code>PLAN-ISSUE-NNN</code>. Spec <code>implemented</code> is declared; it is not inferred from the plan. A <code>delivered</code> bundle cited by a non-<code>implemented</code> spec fails the gate.</p>
</div>

### Issue

Live: `open` → `ready` → `in-progress`. `blocked` waits on named work.

<div class="tactics-matrix">
<table>
<thead>
<tr><th>State</th><th>Before you can enter it</th><th>Validator / gate</th><th>Enables</th></tr>
</thead>
<tbody>
<tr><td data-label="State"><code>open</code></td><td data-label="Before you can enter it">Scaffolded issue, Problem section</td><td data-label="Validator / gate">Schema only. Requirements and claims are optional.</td><td data-label="Enables">Issue exists but cannot produce a plan</td></tr>
<tr><td data-label="State"><code>ready</code></td><td data-label="Before you can enter it">Requirements, claims, verification, implementation, and contracts</td><td data-label="Validator / gate">Full spec-parity traceability.</td><td data-label="Enables">Plan creation</td></tr>
<tr><td data-label="State"><code>in-progress</code></td><td data-label="Before you can enter it">Same rigor as <code>ready</code></td><td data-label="Validator / gate">Same.</td><td data-label="Enables">Plan implementation or issue close-out</td></tr>
<tr><td data-label="State"><code>blocked</code></td><td data-label="Before you can enter it">Same rigor as <code>ready</code>, plus <code>blocked_by</code></td><td data-label="Validator / gate">Same, plus the blocker must be named.</td><td data-label="Enables">Work remains open with a named blocker</td></tr>
<tr><td data-label="State"><code>closed</code></td><td data-label="Before you can enter it"><code>## Resolution</code>, and exactly one close pointer unless the issue carries its own requirements and claims</td><td data-label="Validator / gate">Close traceability.</td><td data-label="Enables">No further issue work</td></tr>
</tbody>
</table>
</div>

<!-- backstop-claim: CLAIM-031 -->
Closing an issue requires a `## Resolution` section and at most one traceability field: `delivered_by` names a completed plan, while `resolved-by` names a direct typed artifact reference, commit SHA, or pull-request URL when no plan lineage applies.
<!-- /backstop-claim -->

`delivered_by` and `resolved-by` on the same close is illegal. `replaced` requires `replaced-by`. `obsoleted` requires `obsoleted-by`. `canceled` is abandoned work.

### Spec

Live: `draft` → `ready-for-implementation` → `implemented`.

<div class="tactics-matrix">
<table>
<thead>
<tr><th>State</th><th>Before you can enter it</th><th>Validator / gate</th><th>Enables</th></tr>
</thead>
<tbody>
<tr><td data-label="State"><code>draft</code></td><td data-label="Before you can enter it">Spec exists from a bundle; required live-work shape</td><td data-label="Validator / gate">Spec schema/completeness</td><td data-label="Enables">Authoring</td></tr>
<tr><td data-label="State"><code>ready-for-implementation</code></td><td data-label="Before you can enter it">Requirements, claims, verification, implementation, contracts complete; review/validation passed</td><td data-label="Validator / gate">Full live-work completeness/traceability</td><td data-label="Enables">Plan creation</td></tr>
<tr><td data-label="State"><code>implemented</code></td><td data-label="Before you can enter it">Work delivered through its plan</td><td data-label="Validator / gate">Delivery/closure requirements</td><td data-label="Enables">No further implementation work</td></tr>
</tbody>
</table>
</div>

Terminals: `replaced`, `canceled`, `deprecated`, `obsoleted`. Retired specs are exempt from live-work completeness.

### Bundle

Live: `idea` → `exploring` → `defined` → `ready`. Success terminal: `delivered`.

The user drives promotion. Do not self-promote.

<div class="tactics-matrix">
<table>
<thead>
<tr><th>State</th><th>Before you can enter it</th><th>Validator / gate</th><th>Enables</th></tr>
</thead>
<tbody>
<tr><td data-label="State"><code>idea</code></td><td data-label="Before you can enter it">Named bundle</td><td data-label="Validator / gate">Identity and maturity enum.</td><td data-label="Enables">Bundle exists but cannot produce a spec</td></tr>
<tr><td data-label="State"><code>exploring</code></td><td data-label="Before you can enter it">Real open questions, unresolved</td><td data-label="Validator / gate">Same.</td><td data-label="Enables">Exploration</td></tr>
<tr><td data-label="State"><code>defined</code></td><td data-label="Before you can enter it"><code>problem.summary</code>, <code>problem.user_story</code>, <code>solution.approach</code>, <code>requirements[]</code>, and sections Draft Requirements, Draft Design Decisions, Spec Seeds, Version History</td><td data-label="Validator / gate">Maturity gates. Placeholders are illegal.</td><td data-label="Enables">Promotion to <code>ready</code></td></tr>
<tr><td data-label="State"><code>ready</code></td><td data-label="Before you can enter it">Everything <code>defined</code> requires, plus <code>problem.success_criteria</code> and <code>solution.assumptions</code></td><td data-label="Validator / gate">Same, tighter.</td><td data-label="Enables">Spec generation</td></tr>
<tr><td data-label="State"><code>delivered</code></td><td data-label="Before you can enter it">The work shipped. <code>requirements[]</code> still required.</td><td data-label="Validator / gate">Success terminal.</td><td data-label="Enables">No further bundle work</td></tr>
</tbody>
</table>
</div>

Terminals without delivery: `replaced`, `canceled`, `deprecated`. `replaced` requires `replaced-by`.

### Plan

Live: `draft` → `ready` → `implementing` → `completed`.

`draft`, `ready`, and `implementing` are the same shape to the validator: phases, tasks, tests before implementation, every source claim mapped. Terminal plans (`replaced`, `canceled`, `obsoleted`) are exempt from phase and task completeness.

<div class="tactics-matrix">
<table>
<thead>
<tr><th>State</th><th>Before you can enter it</th><th>Validator / gate</th><th>Enables</th></tr>
</thead>
<tbody>
<tr><td data-label="State"><code>draft</code></td><td data-label="Before you can enter it">Source is an issue or a spec. Phases and tasks are present.</td><td data-label="Validator / gate">Plan schema and completeness.</td><td data-label="Enables">Authoring</td></tr>
<tr><td data-label="State"><code>ready</code></td><td data-label="Before you can enter it">Same shape. Reviewer has passed.</td><td data-label="Validator / gate">Same.</td><td data-label="Enables">Implementation</td></tr>
<tr><td data-label="State"><code>implementing</code></td><td data-label="Before you can enter it">An implementer is executing it.</td><td data-label="Validator / gate">Same.</td><td data-label="Enables">Plan execution</td></tr>
<tr><td data-label="State"><code>completed</code></td><td data-label="Before you can enter it">The work was delivered. Mandated tests exist.</td><td data-label="Validator / gate">Validator may accept a completed plan whose mandated tests are missing. The gate does not.</td><td data-label="Enables">Source artifact close-out</td></tr>
</tbody>
</table>
</div>

Issue-sourced (`PLAN-ISSUE-NNN`): permits `delivered_by` on that issue. Spec-sourced (`PLAN-SPEC-NNN`): permits setting the spec to `implemented`. Bundle `delivered` is a separate declared status. Spec and bundle status are not inferred from the plan.

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
