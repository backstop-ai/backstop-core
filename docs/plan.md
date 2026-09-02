---
title: Plan
layout: default
permalink: /plan/
page_kind: entity
hero_question: "Plan."
hero_lede: "Ordered work an agent can execute. It comes from an issue or a spec."
---

## What it is

A plan is the bound on implementation. Nobody writes product code until one is approved.

Reactive work uses issue → plan. Feature work uses bundle → spec → plan. Both tracks meet here.

## Shape

Plans are YAML. The file lives under `plans/` and is named from its source, such as `PLAN-ISSUE-NNN-…` or `PLAN-SPEC-NNN-…`.

A plan has phases. A phase has tasks. Each task names:

- the kind of work (test, implementation, verification, and the rest)
- the files it may touch
- the claims it covers
- what it depends on

Do not hand-write a plan into existence. Create it from the source issue or spec.

## Validator

<pre><code>backstop artifact validate --plan PLAN-ISSUE-NNN</code></pre>

The validator checks the schema and the planning rules: tests before implementation, verification in any phase that changes code, disjoint files for parallel work, and every source claim mapped to a task.

## Reviewer

After it validates, an independent plan reviewer reads the source issue or spec and the plan. It checks claim coverage, file scope, TDD order, and gate cadence.

Findings go back to the planner. Implementation does not start on a plan that failed review.

## Relations

The source artifact defines the work. The plan is the handoff. An implementer executes the approved plan and runs required gates as it works. Implementation review comes after that execution, not instead of plan review.

<p class="entity-also">
<a href="/model/#operating-model">Operating model</a>
<a href="/reference/#artifact-schema-catalog">Artifact schemas</a>
</p>
