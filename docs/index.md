---
title: Backstop
layout: default
permalink: /
hero_question: "What failure does Backstop prevent?"
---

## Why Backstop {#why-backstop}

<!-- backstop-claim: CLAIM-017 -->
AI can produce code faster than a team can establish whether it satisfies the team's intent. Backstop makes that mismatch inspectable: define the work, bind execution, and run deterministic standards before drift becomes review debt.
<!-- /backstop-claim -->

<p class="home-lede">The problem is not generating more code. The problem is knowing whether the code that arrived is the code you actually asked for, whether it still follows the standards you intended to preserve, and whether a green result means the same thing tomorrow that it means today.</p>

<div class="home-capabilities" data-home-capabilities>
  <article>
    <span>01</span>
    <h3>Define the work</h3>
    <p>Use explicit artifacts to make intent inspectable before implementation begins. Larger work moves from bundle to spec to plan; smaller reactive work can move from issue to plan. Both paths create a concrete acceptance boundary instead of asking an agent to infer one.</p>
  </article>
  <article>
    <span>02</span>
    <h3>Enforce your standards</h3>
    <p>Put mechanical engineering decisions in versioned packs and run them while the agent works and again at integration. Architecture boundaries, test expectations, dependency policy, and implementation patterns stop being prompt suggestions and become blocking rules.</p>
  </article>
  <article>
    <span>03</span>
    <h3>Detect drift</h3>
    <p>Compare standards against an explicit baseline and requirements against implementation evidence. Existing debt can remain visible without becoming permanent permission for new debt, while completion claims stay tied to the tests and evidence that make them true.</p>
  </article>
</div>

<div class="home-gate-proof" data-home-gate-proof aria-label="Example Backstop gate result">
  <div class="home-gate-command"><span aria-hidden="true">$</span> backstop gate</div>
  <div class="home-gate-results">
    <span>✓ pack integrity</span>
    <span>✓ artifacts</span>
    <span>✓ engineering standards</span>
    <span>✓ tests</span>
    <span>✓ requirements</span>
    <strong>PASS · exit 0</strong>
  </div>
</div>

Backstop does not make probabilistic systems deterministic. It makes the important boundaries around them deterministic: what work was promised, which standards apply, what evidence must exist, and what conditions are allowed to produce a passing verdict.

<!-- backstop-journey-link: JLINK-001 -->
[Evaluate the failure fit](/evaluate/#failure-fit)

## Choose your path {#choose-your-path}

You do not have to adopt the entire framework at once. Start with the failure you need to control, then add the pieces that make that boundary enforceable.

<div class="home-paths" data-home-paths>
  <article>
    <span>Evaluate</span>
    <h3>Decide whether Backstop fits</h3>
    <p>Start with the failure class, guarantees, limits, compatibility boundary, and evidence model before changing a repository.</p>
    <a href="/evaluate/">Evaluate the control surface →</a>
  </article>
  <article>
    <span>Understand</span>
    <h3>See how the pieces reinforce one another</h3>
    <p>Follow the artifact lifecycle, enforcement loop, ownership boundaries, and the point where deterministic checks replace agent judgment.</p>
    <a href="/model/">Read the operating model →</a>
  </article>
  <article>
    <span>Adopt</span>
    <h3>Put one real standard behind the gate</h3>
    <p>Install Backstop, initialize the repository, choose a maintained pack, and prove that a violation actually blocks before expanding the surface.</p>
    <a href="/adopt/">Start a working adoption →</a>
  </article>
</div>

The framework is composable: artifacts can make delivery intent traceable without policy packs; packs and the gate can enforce standards without the full artifact chain; recipes can provide deterministic scaffolding without owning enforcement. The pieces are stronger together, but each should earn its place by controlling a real failure.

> If it has to be right, it must be deterministic. If it's green, it ships.
