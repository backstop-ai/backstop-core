---
title: Plan
layout: default
permalink: /plan/
page_kind: entity
hero_question: "Plan"
hero_lede: "Ordered work an agent can execute. It comes from an issue or a spec."
---

<dl class="entity-meta">
<dt>Kind</dt><dd>Artifact</dd>
<dt>Schema</dt><dd><code>plan/v1</code></dd>
<dt>File</dt><dd><code>plans/PLAN-{SPEC|ISSUE}-NNN-slug.plan.yml</code></dd>
<dt>Format</dt><dd>YAML. Machine-consumed. No markdown body.</dd>
<dt>Created from</dt><dd>An issue or a spec. A plan does not exist on its own.</dd>
</dl>

## Sources

<div class="entity-table entity-table-text">
<table>
<thead>
<tr><th>Work</th><th>Path</th></tr>
</thead>
<tbody>
<tr><td data-label="Work">Feature / substantial</td><td data-label="Path">Bundle → Spec → Plan</td></tr>
<tr><td data-label="Work">Small / reactive</td><td data-label="Path">Issue → Plan</td></tr>
</tbody>
</table>
</div>

## Fields

<div class="entity-table">
<table>
<thead>
<tr><th>Field</th><th>Required</th><th>Form</th></tr>
</thead>
<tbody>
<tr><td data-label="Field"><code>plan_id</code></td><td data-label="Required">yes</td><td data-label="Form"><code>PLAN-SPEC-NNN</code> or <code>PLAN-ISSUE-NNN</code></td></tr>
<tr><td data-label="Field"><code>spec_id</code></td><td data-label="Required">yes</td><td data-label="Form"><code>SPEC-NNN</code> or <code>ISSUE-NNN</code></td></tr>
<tr><td data-label="Field"><code>status</code></td><td data-label="Required">yes</td><td data-label="Form">See Status</td></tr>
<tr><td data-label="Field"><code>created</code></td><td data-label="Required">yes</td><td data-label="Form"><code>YYYY-MM-DD</code></td></tr>
<tr><td data-label="Field"><code>spec_version</code></td><td data-label="Required">no</td><td data-label="Form">string</td></tr>
<tr><td data-label="Field"><code>target_repo</code></td><td data-label="Required">no</td><td data-label="Form">string</td></tr>
<tr><td data-label="Field"><code>target_module</code></td><td data-label="Required">no</td><td data-label="Form">string</td></tr>
<tr><td data-label="Field"><code>test_command</code></td><td data-label="Required">no</td><td data-label="Form">string</td></tr>
<tr><td data-label="Field"><code>coverage_threshold</code></td><td data-label="Required">no</td><td data-label="Form">integer 0–100</td></tr>
<tr><td data-label="Field"><code>notes</code></td><td data-label="Required">no</td><td data-label="Form">string</td></tr>
<tr><td data-label="Field"><code>replaced-by</code></td><td data-label="Required">when <code>replaced</code></td><td data-label="Form">typed artifact id</td></tr>
<tr><td data-label="Field"><code>obsoleted-by</code></td><td data-label="Required">when <code>obsoleted</code></td><td data-label="Form">typed artifact id</td></tr>
</tbody>
</table>
</div>

## Status

Status is a declared field. Authors and implementers set it. It is not inferred from git or CI.

<div class="entity-table">
<table>
<thead>
<tr><th>Value</th><th>Kind</th><th>Meaning</th></tr>
</thead>
<tbody>
<tr><td data-label="Value"><code>draft</code></td><td data-label="Kind">live</td><td data-label="Meaning">Being written</td></tr>
<tr><td data-label="Value"><code>ready</code></td><td data-label="Kind">live</td><td data-label="Meaning">Approved to implement</td></tr>
<tr><td data-label="Value"><code>implementing</code></td><td data-label="Kind">live</td><td data-label="Meaning">An implementer is executing it</td></tr>
<tr><td data-label="Value"><code>completed</code></td><td data-label="Kind">live</td><td data-label="Meaning">The work was delivered. Still a full plan.</td></tr>
<tr><td data-label="Value"><code>replaced</code></td><td data-label="Kind">terminal</td><td data-label="Meaning">A named successor took over. Requires <code>replaced-by</code>.</td></tr>
<tr><td data-label="Value"><code>canceled</code></td><td data-label="Kind">terminal</td><td data-label="Meaning">Abandoned. A reason is optional.</td></tr>
<tr><td data-label="Value"><code>obsoleted</code></td><td data-label="Kind">terminal</td><td data-label="Meaning">Shipped, then removed with no 1:1 successor. Requires <code>obsoleted-by</code>.</td></tr>
</tbody>
</table>
</div>

`draft`, `ready`, and `implementing` are the same shape to the validator. Terminal plans are exempt from phase and task completeness.

<div class="entity-illegal">
<p>Illegal</p>
<ul>
<li>Any other status name (<code>deprecated</code>, <code>open</code>, <code>closed</code>, <code>in-progress</code>)</li>
<li><code>replaced</code> without <code>replaced-by</code></li>
<li><code>obsoleted</code> without <code>obsoleted-by</code></li>
<li>A <code>completed</code> plan whose mandated tests are missing — the validator may accept the file; the gate does not</li>
</ul>
</div>

## Phases

A plan is an ordered list of phases. A phase has `id`, `name`, and `tasks`.

Each task has:

<div class="entity-table">
<table>
<thead>
<tr><th>Key</th><th>Required</th><th>Form</th></tr>
</thead>
<tbody>
<tr><td data-label="Key"><code>id</code></td><td data-label="Required">yes</td><td data-label="Form">task id</td></tr>
<tr><td data-label="Key"><code>type</code></td><td data-label="Required">yes</td><td data-label="Form"><code>setup</code> · <code>test</code> · <code>implementation</code> · <code>verification</code> · <code>refactor</code> · <code>documentation</code></td></tr>
<tr><td data-label="Key"><code>title</code></td><td data-label="Required">yes</td><td data-label="Form">string</td></tr>
<tr><td data-label="Key"><code>description</code></td><td data-label="Required">yes</td><td data-label="Form">string</td></tr>
<tr><td data-label="Key"><code>files</code></td><td data-label="Required">yes</td><td data-label="Form">paths this task may touch</td></tr>
<tr><td data-label="Key"><code>claims</code></td><td data-label="Required">yes</td><td data-label="Form">source claims this task covers</td></tr>
<tr><td data-label="Key"><code>depends_on</code></td><td data-label="Required">yes</td><td data-label="Form">task ids; empty list if none</td></tr>
<tr><td data-label="Key"><code>test_names</code></td><td data-label="Required">no</td><td data-label="Form">mandated test function names this task delivers</td></tr>
</tbody>
</table>
</div>

## Constraints

- Tests before implementation
- Verification in any phase that changes code
- Parallel work has disjoint file sets
- Every source claim maps to a task
- Filename starts with `plan_id`
- Do not hand-write a plan into existence

## Validate

<pre><code>backstop artifact validate --plan PLAN-ISSUE-NNN</code></pre>

## Reviewer

`plan-reviewer`. Independent of the planner. Runs after the file validates.

- Reads the source issue or spec and the plan
- Checks claim coverage, file scope, TDD order, and gate cadence
- Findings go back to the planner
- Implementation does not start on a plan that failed review

## Notes

A plan is the bound on implementation. Nobody writes product code until one is approved.

Reactive work uses issue → plan. Feature work uses bundle → spec → plan. Both tracks meet here.

The source artifact defines the work. The plan is the handoff. An implementer executes the approved plan and runs required gates as it works.

<p class="entity-also">
<a href="/issue/">Issue</a>
<a href="/spec/">Spec</a>
<a href="/bundle/">Bundle</a>
<a href="/model/#operating-model">Operating model</a>
<a href="/reference/#artifact-schema-catalog">Artifact schemas</a>
</p>
