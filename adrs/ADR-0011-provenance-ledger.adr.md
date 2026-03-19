---
number: ADR-0011
created: "2026-03-18"
status: Accepted
deciders: "@bmanson"
decisions: "D-051, D-054"
schema_version: adr/v2
---

# ADR-0011: Provenance Ledger — Hash-Chained Audit Trail Per Directive

## Context

Backstop's thesis requires not just enforcement but proof of enforcement. The validation engine (ADR-0004) can verify that code is correct at a point in time, but it cannot answer: Who did this work? In what order? Was the record altered after the fact? These questions matter when the enforcement stack is the sole authority — when there is no human reviewer to vouch for the process, the process must vouch for itself.

The provenance ledger is the audit trail that makes backstop's enforcement inspectable, tamper-evident, and complete. It records every agent action within a directive's scope, chains entries cryptographically, and provides the foundation for the gate action's ship/no-ship decision.

## Decision

### Scoped per directive, not per spec or branch

D-051 (revised) establishes the directive as the unit of tracked work. A directive is a body of work that may encompass multiple specs, plans, issues, and code changes. The ledger captures everything within that scope.

Why per-directive and not per-spec or per-branch:
- **Per-spec is too narrow.** A single body of work often touches multiple specs. Splitting the audit trail across spec-scoped ledgers fragments the record and makes cross-cutting verification impossible.
- **Per-branch is too broad and too fragile.** Branches are mutable (rebased, renamed, deleted). Tying the audit trail to a branch name creates orphaned or ambiguous records.
- **Per-directive is stable.** Directives have immutable identifiers, defined scope, and clear lifecycle. The ledger lives and dies with the directive.

### File naming

Ledger files follow the directive naming convention (D-072):

```
DIRECTIVE-NNNN-slug.ledger.yml
```

The ledger is committed alongside the directive's other artifacts. It is version-controlled, diff-able, and part of the project's permanent record.

### Hash-chained entries

Every ledger entry includes a cryptographic hash chain (D-054):

```yaml
entries:
  - seq: 1
    timestamp: "2026-03-18T14:23:01Z"
    agent_id: "impl-agent-01"
    action: "write"
    artifact_ref: "specs/SPEC-0042-auth-handler.spec.md"
    claim_ref: null
    result: "created"
    prev_hash: "0000000000000000000000000000000000000000000000000000000000000000"
    hash: "a1b2c3d4e5f6..."

  - seq: 2
    timestamp: "2026-03-18T14:23:47Z"
    agent_id: "impl-agent-01"
    action: "write"
    artifact_ref: "internal/auth/handler.go"
    claim_ref: "SPEC-0042#claim-1"
    result: "created"
    prev_hash: "a1b2c3d4e5f6..."
    hash: "b2c3d4e5f6a1..."

  - seq: 3
    timestamp: "2026-03-18T14:24:12Z"
    agent_id: "impl-agent-01"
    action: "run"
    artifact_ref: "internal/auth/handler_test.go"
    claim_ref: "SPEC-0042#claim-1"
    result: "pass"
    prev_hash: "b2c3d4e5f6a1..."
    hash: "c3d4e5f6a1b2..."
```

Each entry's `hash` is computed over its own content plus `prev_hash`. The first entry's `prev_hash` is the zero hash. This creates a chain where modifying any entry invalidates all subsequent hashes.

Combined with git's own hash chain, this provides dual-layer integrity: the ledger's internal chain proves entries weren't reordered or modified, and git's commit hashes prove the ledger file itself wasn't replaced wholesale.

### Append-only semantics

Ledger entries are strictly appended. No edits, no deletions, no reordering. If an earlier entry was wrong, a new corrective entry is appended — the original remains in the record. This is an audit log, not a mutable document.

### Entry structure

Each entry records:

| Field | Description |
|-------|-------------|
| `seq` | Monotonically increasing sequence number, starting at 1 |
| `timestamp` | ISO 8601 UTC timestamp of the action |
| `agent_id` | Identity of the agent that performed the action |
| `action` | One of: `read`, `write`, `run`, `validate`, `review` |
| `artifact_ref` | Path or identifier of the artifact acted upon |
| `claim_ref` | Optional reference to a specific claim (e.g., `SPEC-0042#claim-1`) |
| `result` | Outcome of the action (e.g., `created`, `modified`, `pass`, `fail`, `approved`, `rejected`) |
| `prev_hash` | Hash of the previous entry (zero hash for seq 1) |
| `hash` | Hash of this entry's content including `prev_hash` |

### Entries tag their artifacts

Every entry includes an `artifact_ref` that identifies what was acted upon. This enables two query patterns:

1. **Aggregate status** — "show me all entries for this directive" gives the full timeline
2. **Per-artifact drill-down** — "show me all entries touching this file" gives the file's provenance

### Implementation and review are separate identities

When the review model (ADR-0012) runs, the implementation agent and review agent record entries under different `agent_id` values. The ledger captures both perspectives:

```yaml
  - seq: 7
    agent_id: "impl-agent-01"
    action: "write"
    artifact_ref: "internal/auth/handler.go"
    result: "modified"
    # ...

  - seq: 8
    agent_id: "review-agent-01"
    action: "review"
    artifact_ref: "internal/auth/handler.go"
    result: "approved"
    # ...
```

This separation is what makes the review model auditable — you can verify that the reviewer was a different agent than the implementer.

### Integrity verification

`backstop validate` includes a ledger verification pass that:

1. **Walks the hash chain backwards** from the last entry to the first
2. **Recomputes each hash** from the entry content and `prev_hash`
3. **Confirms the chain is unbroken** — if any entry was modified, its hash won't match and all subsequent hashes will also fail
4. **Checks sequence continuity** — no gaps, no duplicates in `seq` numbers
5. **Verifies completeness** — all spec claims have corresponding `run` entries with `pass` results

### Directives as the default work unit

Directives are promoted from optional to the default way tracked work is initiated. When `workflow.ledger` is enabled in backstop.yml, every body of work begins with a directive, and every directive gets a ledger.

Enforcement-only users who want backstop's validation without the provenance tracking can set `workflow.ledger: false` in backstop.yml. This disables directives and ledgers entirely — backstop functions as a pure enforcement tool.

## Consequences

### What this enables
- **"If it's green, it ships" has proof.** The gate action reads the ledger and mechanically verifies that every claim was tested, every test passed, and every review completed. The ledger is the evidence the gate action evaluates.
- **Tamper evidence is mathematical.** Modifying a ledger entry invalidates the hash chain. Combined with git's commit hashes, the dual-layer integrity makes undetected tampering computationally infeasible.
- **Full audit trail.** After deployment, the ledger answers: who implemented this, who reviewed it, what tests were run, what was the result, in what order. Every question has a mechanically verifiable answer.
- **Agent accountability.** Different agents record under different identities. The ledger can show that the reviewer was independent of the implementer.

### What this requires
- **Agents must record entries.** Every action that modifies or verifies an artifact must append a ledger entry. This is overhead, but it's the cost of provability.
- **Hash computation must be deterministic.** The hash algorithm, field ordering, and serialization format must be specified precisely so any implementation can independently verify the chain.
- **Directive lifecycle management.** Directives must be created before work begins and closed when work is complete. The ledger's scope depends on the directive's scope.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Git log as the audit trail | Git records file-level changes but not claim-level verification. It doesn't know which test covers which claim, or whether a review was independent. The ledger carries semantic provenance that git cannot. |
| Per-spec ledgers | Fragments the audit trail across specs. A directive that touches three specs would have three ledgers with no unified view. Per-directive scoping keeps the trail coherent. |
| Database-backed ledger (not file-based) | Adds infrastructure dependency. A YAML file committed to git is portable, version-controlled, and requires no external services. |
| No hash chain (rely on git integrity alone) | Git proves the file wasn't replaced, but not that individual entries weren't reordered or modified between commits. The internal hash chain provides entry-level integrity. |

## References

- D-051: Ledger scoped per directive (revised from per-spec)
- D-054: Hash-chained ledger entries for tamper evidence
- D-072: Directive naming convention
- ADR-0001: The thesis — mechanical enforcement replaces human review
- ADR-0004: Validation engine — ledger verification as a validation pass
- ADR-0009: CI/CD pipeline — `backstop/ledger-verify` action
- ADR-0010: Verification kill chain — the chain the ledger records
- ADR-0012: Review model — implementation and review as separate agent identities
