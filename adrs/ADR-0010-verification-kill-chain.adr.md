---
number: ADR-0010
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-009, D-010, D-055, D-056, D-057"
schema_version: adr/v2
---

# ADR-0010: Verification Kill Chain — Spec to Provably Correct Code

## Context

ADR-0001 claims backstop produces provably correct code — not "probably fine" code. That claim requires a mechanical traceability chain from spec to code where every link is verifiable. Without this chain, "green" is an assertion without proof. Agents can write tests that pass but prove nothing. Tests can exist but not exercise the code under test. Claims can be declared but never verified.

The verification kill chain is the mechanism that closes every gap between "the spec says X" and "the code provably does X." Each link in the chain is independently verifiable, and the chain as a whole is what transforms backstop from a linter into a proof system.

## Decision

### The full traceability chain

The kill chain has eight links, each mechanically verifiable:

1. **Spec declares claims[]** — each claim is a testable assertion about system behavior. Claims are the atomic unit of correctness. A spec without claims is a spec without teeth.

2. **Spec declares sharp_edges[]** (D-055) — known risks, edge cases, and gotchas that naive implementations miss. Sharp edges are not optional documentation; they are mandated acknowledgments of complexity that agents must address. Sharp edges force the spec author to think adversarially and the implementing agent to handle the hard cases.

3. **Spec declares test_cases[] with mandated test names** (D-056) — the exact function name the implementing agent must write. Not a description of what to test. Not a suggestion. The literal function name: `TestParseConfig_EmptyInput`, `TestHashChain_TamperedEntry`. The spec author controls the test surface, not the agent.

4. **Plan breaks claims into agent-bounded tasks** — each task has file-level scope and maps to one or more claims. The plan is the bridge between "what" (spec) and "who does what" (agents). Tasks that span too many files are split. Tasks that don't map to claims are rejected.

5. **Agent implements code and tests** — tests MUST use the spec-mandated names. The agent has no discretion over test naming. This constraint is what makes the chain verifiable — if the mandated test name exists and passes, the claim is covered.

6. **`backstop validate` verifies the chain** — test exists (by mandated name), test passes, test body is substantive (D-057). This is the enforcement point. A missing test is a violation. A failing test is a violation. A hollow test is a violation.

7. **Pack AST checkers inspect test bodies** (D-057) — substantiveness is not a vibes check. The AST checker verifies that the test body: calls the function under test, contains meaningful assertions (not just `assert(true)`), includes negative cases where the spec's sharp edges demand them. Hollow tests — tests that exist by name but prove nothing — are caught mechanically.

8. **Ledger records the full chain** — which claim, which test, which agent, pass/fail, timestamp, hash. The ledger (ADR-0011) is the audit trail that makes the chain inspectable after the fact.

### The traceability path

```
Spec → Sharp Edges → Test Cases (mandated names) → Claims → Code → Ledger
```

Each arrow is a mechanical verification. The spec declares what must be true. Sharp edges declare what's hard. Test cases declare how to verify. Claims tie tests to assertions. Code implements. The ledger records.

### What makes a test "substantive"

The AST checker (D-057) applies three checks to every mandated test body:

1. **Calls the function under test** — the test must invoke the production code it claims to verify. A test that only exercises test helpers proves nothing about the system.
2. **Contains meaningful assertions** — `assert(true)`, `expect(1).toBe(1)`, and their equivalents are detected and rejected. Assertions must reference the result of calling production code.
3. **Includes negative cases where required** — when a spec's sharp_edges[] identify an error condition, the corresponding test must verify that the error is handled. The AST checker cross-references sharp edges with test bodies to ensure coverage.

### Mandated test names are non-negotiable

D-056 is the linchpin of the kill chain. If agents could name tests freely, verification would require semantic matching ("does this test probably cover this claim?"). Mandated names make verification syntactic: the name exists or it doesn't. This is a deliberate trade-off — it constrains agent autonomy in exchange for mechanical verifiability.

## Consequences

### What this enables
- **Provably correct is mechanical, not aspirational.** Every claim has a named test. Every test is substantive. Every link is verified. The proof is in the chain, not in a reviewer's judgment.
- **Hollow tests are structurally impossible.** AST checking catches tests that exist by name but prove nothing. Agents cannot game the system with empty test bodies.
- **Sharp edges force adversarial thinking.** Spec authors must identify the hard cases. Agents must handle them. The easy path — ignoring edge cases — is blocked by the spec format itself.
- **The ledger makes the chain auditable.** After the fact, anyone can inspect which claims were tested, by which agent, with what result. The chain is not just enforced; it's recorded.

### What this requires
- **Spec authors must write good claims and sharp edges.** The kill chain verifies that tests match claims, but it cannot verify that claims match intent. Garbage claims produce garbage verification. Spec quality is upstream of everything.
- **AST checkers per language.** Substantiveness checking requires parsing test files. Each supported language needs an AST checker that understands its test framework's assertion patterns.
- **Mandated test names must be valid identifiers.** Spec authors must use names that are legal function names in the target language. The spec schema validates this.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Coverage-only verification (line/branch coverage thresholds) | Coverage measures execution, not correctness. A test that calls every line but asserts nothing has 100% coverage and proves nothing. |
| Semantic test matching (AI determines if a test covers a claim) | Non-deterministic. Two runs could produce different verdicts. Mechanical verification requires syntactic matching. |
| Agent-chosen test names with description matching | Fuzzy matching creates false positives and false negatives. Mandated names are unambiguous. |
| No sharp edges (claims only) | Claims describe the happy path. Sharp edges force coverage of the hard cases — the cases agents are most likely to miss. |

## References

- D-009: Claims as the atomic unit of spec correctness
- D-010: Test verification tied to claims
- D-055: Sharp edges — mandated risk acknowledgment in specs
- D-056: Mandated test names — spec controls the test surface
- D-057: AST-based substantiveness checking for test bodies
- ADR-0001: The thesis — provably correct via mechanical enforcement
- ADR-0002: Artifact primitives — spec format that carries claims, sharp edges, and test cases
- ADR-0004: Validation engine — where AST checking and test verification execute
- ADR-0011: Provenance ledger — where the kill chain is recorded
