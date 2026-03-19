---
number: ADR-0012
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-059, D-067"
schema_version: adr/v2
---

# ADR-0012: Review Model — Independent Reviewer, "If It's Green It Ships"

## Context

Human code review is the last manual gate in the software delivery pipeline. It exists because we don't trust authors to verify their own work — we need an independent set of eyes. But human review has fundamental problems: it's slow, inconsistent, subject to social pressure, and doesn't scale. A senior engineer reviews differently at 9am Monday than at 4pm Friday. A junior engineer approves what they don't understand. A team lead rubber-stamps PRs from people they trust.

Backstop's thesis (ADR-0001) claims that mechanical enforcement can replace human review entirely. The verification kill chain (ADR-0010) makes "green" meaningful. The provenance ledger (ADR-0011) makes enforcement auditable. This ADR completes the picture: an independent review model that is provably unbiased and a mechanical merge authority that removes humans from the critical path.

## Decision

### Independent reviewer agent

D-067 establishes that the review phase uses a separate agent session from the implementation phase. This is not a suggestion — it is a structural requirement of the review model.

**What the reviewer gets:**
- The spec (claims, sharp edges, mandated test names)
- The code diff (what the implementation agent produced)
- The ledger (what actions were taken, in what order)
- The plan (how the work was decomposed)

**What the reviewer does NOT get:**
- The implementation agent's chain of thought
- Alternative approaches the implementation agent tried and discarded
- Decision context or rationale from the implementation session
- Any conversational history from the implementation phase

This is the software equivalent of double-blind review. The reviewer is provably unbiased because it literally cannot access the implementation context. It evaluates the artifacts on their merits, not on the implementer's reasoning.

**Why a separate session, not just a separate prompt:**

A separate prompt within the same session still has access to the full conversation history. The model's attention mechanism will attend to earlier context regardless of instructions to ignore it. A separate session is a hard boundary — the implementation context does not exist in the reviewer's context window. This is bias prevention by architecture, not by instruction.

### What the reviewer evaluates

The reviewer agent performs a structured review against the spec:

1. **Claim coverage** — does the code implement what the spec claims? Are all claims addressed?
2. **Sharp edge handling** — does the code handle the risks identified in sharp_edges[]? Are edge cases covered, not just happy paths?
3. **Test adequacy** — do the mandated tests exercise the code meaningfully? (The AST checker handles syntactic substantiveness; the reviewer evaluates semantic adequacy.)
4. **Spec conformance** — does the implementation match the spec's intent, not just its letter?
5. **Obvious defects** — bugs, logic errors, security issues visible in the diff

The reviewer records its findings as ledger entries under its own `agent_id`, including approval or rejection with rationale.

### Mechanical merge authority

D-059 establishes that the gate action (ADR-0009) is the merge authority. The gate action evaluates a conjunction of conditions — ALL must be true for the PR to merge:

1. **All validators pass** — schema validation, pack checks, AST analysis (ADR-0004)
2. **All tests pass** — the project's full test suite, including mandated test names
3. **All test bodies are substantive** — AST checker confirms no hollow tests (ADR-0010)
4. **The ledger is intact** — hash chain verified, no gaps, no tampering (ADR-0011)
5. **Coverage thresholds are met** — as configured in backstop.yml
6. **The reviewer agent approved** — an independent review session approved the changes
7. **Every spec claim has a corresponding passing test** — the kill chain is complete

No single condition is sufficient. The conjunction is what makes "green" meaningful. A PR where all tests pass but the reviewer rejected it does not merge. A PR where the reviewer approved but the hash chain is broken does not merge. Every condition must hold.

### "If it's green, it ships"

This phrase — backstop's motto — now has a precise definition:

**Green means:** every claim verified, every test named and passing, every test body substantive, every required section present, every schema satisfied, hash chain intact, coverage above threshold, independent review agent approved.

**Ships means:** the gate action auto-merges the PR. No human approval required. No manual merge button. The enforcement stack IS the reviewer.

This is not reckless automation. It is the opposite — it is more rigorous than human review because every condition is checked mechanically, every time, without fatigue or social pressure. The bar for "green" is higher than any human reviewer would consistently maintain.

### The conversation is preserved

Both the implementation agent and the review agent record their actions in the provenance ledger (ADR-0011) under separate `agent_id` values. The ledger captures:

- What the implementation agent wrote and tested
- What the review agent evaluated and concluded
- The sequence of actions from both agents
- The final verdict (approved/rejected) with the reviewer's rationale

This creates a permanent, tamper-evident record of the entire implementation-and-review cycle. After deployment, anyone can inspect exactly what was reviewed, by whom, and what the outcome was.

### Review policy configuration

The review model is configurable via backstop.yml's `review_policy` field:

- **`backstop-only`** — the enforcement stack is the sole reviewer. Gate auto-merges when green. This is the full thesis.
- **`human-required`** — backstop runs the full enforcement stack AND requires human approval. For teams transitioning to full automation or operating under regulatory requirements that mandate human sign-off.

The enforcement stack runs identically in both modes. The only difference is whether a human approval is additionally required before the gate action merges.

## Consequences

### What this enables
- **Human code review is obsolete for backstop-managed projects.** The independent reviewer, the verification kill chain, and the provenance ledger provide stronger guarantees than human review. Not different guarantees — stronger ones.
- **Merge velocity is machine-speed.** No waiting for reviewer availability. No review backlog. No context-switching cost for reviewers. PRs merge when they're green.
- **Review quality is consistent.** The reviewer agent applies the same rigor to every PR, every time. No Friday afternoon rubber-stamps. No social pressure to approve.
- **The audit trail is complete.** The ledger records both implementation and review as separate, inspectable entries. Regulatory and compliance requirements for review are met — with better evidence than "a human clicked approve."

### What this requires
- **Trust in the enforcement stack.** Teams must trust that the conjunction of validators, tests, AST checkers, ledger verification, and independent review is sufficient. This trust is earned by the mechanical nature of each check, not assumed.
- **Spec quality is paramount.** The reviewer evaluates against the spec. A bad spec produces a bad review. The review model is only as good as the claims and sharp edges it evaluates against.
- **Two agent sessions per body of work.** The independent review requires spinning up a separate agent session. This is a cost in compute and time, but it is the price of provably unbiased review.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Same agent reviews its own work (self-review) | Violates independence. An agent reviewing its own output will rationalize its own decisions. The separate-session boundary is non-negotiable. |
| Human review with backstop as advisor | Preserves the bottleneck backstop exists to eliminate. Backstop-as-advisor is a stepping stone, not the destination. |
| Multiple reviewer agents (panel review) | Over-engineered for v1. One independent reviewer with the full spec context is sufficient. Panel review is future work if single-reviewer proves insufficient. |
| Review agent in the same session with "forget" instructions | LLM context windows don't support selective forgetting. The model attends to all available context regardless of instructions. Separate sessions provide a hard architectural boundary. |

## References

- D-059: "If it's green, it ships" — mechanical merge authority
- D-067: Independent reviewer agent — separate session, no implementation context
- ADR-0001: The thesis — mechanical enforcement replaces human review
- ADR-0004: Validation engine — the validators the gate evaluates
- ADR-0009: CI/CD pipeline — the gate action that implements merge authority
- ADR-0010: Verification kill chain — what makes "green" meaningful
- ADR-0011: Provenance ledger — where implementation and review are recorded
