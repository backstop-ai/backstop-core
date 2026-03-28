---
number: ADR-0018
created: "2026-03-28"
status: Accepted
deciders: "@bmanson"
decisions: "D-100, D-101, D-102, D-103, D-104"
schema_version: adr/v2
---

# ADR-0018: Workflow State Machine — From Intent to Ship

## Context

Backstop has well-defined artifacts (ADR-0002), enforcement engines (ADR-0006), a verification kill chain (ADR-0010), a provenance ledger (ADR-0011), and a review model (ADR-0012). What it lacks is a codified workflow — the sequence of phases, transitions, and review gates that connect "I have an idea" to "it's in production."

Today, the workflow exists implicitly in the user's head and in the conventions they've developed through practice. An agent dropped into a backstop project can validate artifacts and verify code, but it cannot answer: "What do I do first? What's next? When am I done?"

This ADR formalizes the workflow as a state machine with deterministic transitions, pluggable review strategies per phase, and a clear "getting started" path for new users. It answers the question every new user asks: "This is cool, but what do I actually do?"

## Decision

### The workflow is a state machine (D-100)

Every directive moves through a fixed sequence of phases. Each phase produces artifacts, and each transition requires a review gate to pass. The state machine is:

```
┌──────────┐
│  INTAKE  │  Entry point: bundle, issue, or ADR triggers a directive
└────┬─────┘
     │
     ▼
┌──────────┐
│   SPEC   │  Produce spec(s) from the directive's source artifacts
└────┬─────┘
     │ ◄── review/fix loop
     ▼
┌──────────┐
│   PLAN   │  Produce plan(s) from approved spec(s)
└────┬─────┘
     │ ◄── review/fix loop
     ▼
┌──────────┐
│  IMPL    │  Execute plan(s), produce code and tests
└────┬─────┘
     │ ◄── review/fix loop
     ▼
┌──────────┐
│   GATE   │  Verification kill chain: all validators, all engines, full chain
└────┬─────┘
     │
     ▼
┌──────────┐
│   SHIP   │  Green. Merge. Deploy.
└──────────┘
```

The directive's current phase is recorded in its frontmatter and in the ledger. There is no skipping phases. There is no going from SPEC directly to IMPL. The state machine enforces the discipline.

### Phase definitions

**INTAKE** — The directive is created. Its source is identified: a bundle (new feature area), an issue (bug or small feature), or an ADR (architectural change). The directive captures scope, success criteria, and links to source artifacts. This is the "what and why."

Entry point for the user: "I want to do X." The agent (or the user) creates a directive that references the upstream artifact. The directive is the commitment — everything downstream is authorized by it.

**SPEC** — One or more specs are authored against the directive. Each spec contains requirements, claims with mandated test names, sharp edges, contracts, and verification config. Specs are the "what to build, precisely."

The review gate at SPEC compares specs against the directive's source artifacts. The question is: "Do these specs fully cover the directive's scope? Are there gaps between what the directive asks for and what the specs promise?"

**PLAN** — One or more plans are authored against approved specs. Each plan breaks a spec into ordered, file-scoped tasks. Plans are the "how to build it."

The review gate at PLAN performs 1:1 analysis of each plan against its parent spec. The question is: "Does this plan cover every claim in the spec? Are the tasks correctly scoped? Is the ordering sound?"

**IMPL** — The implementation agent executes the plans. Code is written, tests are written (using mandated names), contracts are fulfilled. The ledger records every action.

The review gate at IMPL is the independent reviewer (ADR-0012). Separate session, no access to implementation context. The question is: "Does the code satisfy the spec's claims? Do the tests prove what they claim to prove?"

**GATE** — The verification kill chain (ADR-0010) runs end to end. All validators pass. Semgrep rules pass. Coverage thresholds met. Contracts verified. Ledger intact. This is not a review — it's a mechanical pass/fail.

**SHIP** — All gates green. The directive is complete. Merge and deploy.

### Every phase has a review/fix loop (D-101)

The inner loop at each phase is identical in structure:

```
produce artifact(s) → review → pass? → advance to next phase
                        │
                        ▼ fail
                      fix → review → pass? → advance
                               │
                               ▼ fail
                             fix → review → ...
```

The loop is bounded — not infinite. If a review fails N times (configurable, default 3), the directive escalates to the user. The agent does not spin forever on a problem it cannot solve.

What differs per phase is the **review strategy** — what the reviewer checks, what evidence it examines, what constitutes a pass:

| Phase | Reviewer checks | Evidence examined | Pass criteria |
|-------|----------------|-------------------|---------------|
| SPEC | Gaps between source and specs | Directive source + spec frontmatter + requirements | Every directive requirement addressable by at least one spec claim |
| PLAN | Completeness and ordering | Spec claims + plan tasks + file scope | Every spec claim mapped to at least one task; no circular dependencies |
| IMPL | Correctness and substance | Spec + code diff + test results + ledger | All mandated tests exist, pass, and are substantive; contracts fulfilled |
| GATE | Full kill chain | Everything | All validators green, all engines green, coverage met, ledger intact |

### The review is a separate agent session (D-102)

This reinforces ADR-0012 and extends it to all phases, not just implementation review. At every review gate:

- The reviewer is a **separate agent session** — no access to the producer's context
- The reviewer receives only the **artifacts and evidence** defined in the table above
- The reviewer produces a **structured verdict**: pass, fail (with specific issues), or escalate

This is double-blind review at every phase, not just implementation. The spec reviewer doesn't know what the spec author was thinking. The plan reviewer doesn't know what alternatives were considered. Bias prevention by architecture.

### The directive is the state container (D-103)

The directive artifact owns the workflow state. Its frontmatter tracks:

```yaml
phase: spec          # current phase in the state machine
attempt: 2           # current review attempt within the phase
created: "2026-03-28T20:00:00Z"
source:
  type: bundle
  ref: "BUNDLE-0042-user-auth"
specs:
  - "SPEC-0089-auth-handler"
  - "SPEC-0090-token-refresh"
plans:
  - "PLAN-0089-auth-handler"
  - "PLAN-0090-token-refresh"
```

Phase transitions are recorded in the ledger as first-class entries:

```yaml
- seq: 14
  action: "phase_transition"
  from_phase: "spec"
  to_phase: "plan"
  review_verdict: "pass"
  attempt: 2
  timestamp: "2026-03-28T21:15:00Z"
  hash: "..."
```

The directive's phase and the ledger's entries must agree. If they don't, the gate fails. No tampering.

### Getting started (D-104)

The new user workflow answers "what do I actually do?"

**Day one — bootstrap:**

```bash
backstop init
```

That's it. One command. `backstop init` drops the user into an interactive agent session that:

1. Scaffolds `backstop.yml` and `.backstop/` directory
2. Applies embedded baseline rules (D-099)
3. Immediately asks: "What's the first thing you want to build?"
4. Guides the user through creating their first bundle — exploring the problem space, structuring requirements, capturing domain knowledge
5. When the bundle is ready, asks: "Ready to commit to building this?"
6. Creates the directive and begins the state machine

The user never needs to know that bundles, directives, or specs exist as separate commands. The agent session is the interface. The artifacts are produced as a side effect of the conversation.

**The ongoing loop:**

Once initialized, starting new work is the same experience:

```bash
backstop new
```

The agent asks what they want to do — new feature, bug fix, architectural change — and routes to the right intake (bundle, issue, or ADR). The state machine takes over from there.

**For power users:**

Direct artifact creation is available for users who know the primitives:

```bash
backstop new bundle              # skip the routing, go straight to bundle
backstop new issue               # skip the routing, go straight to issue
backstop new directive --source BUNDLE-0042
```

But the default path assumes the user knows nothing except `backstop init`. The framework teaches itself through the conversation.

Backstop doesn't eliminate thinking — it eliminates everything after thinking.

## Consequences

### What this enables
- **Deterministic workflow.** Agents don't improvise. The state machine tells them what phase they're in, what to produce, and what the review gate expects.
- **Pluggable runtime.** The state machine is the contract. The execution layer (Copilot SDK agents, Claude Code, custom runtime) is the implementation. Any runtime that can drive the state machine is a valid backstop executor.
- **Onboarding in one command.** `backstop init` drops into an interactive session that guides the user from zero to first directive. No manual required.
- **Bounded iteration.** Review/fix loops have configurable attempt limits. Agents don't spin forever.
- **Full auditability.** Phase transitions are ledger entries. The entire journey from intake to ship is recorded and hash-chained.

### What this requires
- **Directive schema and validator.** The directive needs a first-class schema with phase tracking, source references, and artifact linkage.
- **Review strategy implementations.** Each phase needs a concrete reviewer that knows what to check and what evidence to examine.
- **Runtime integration.** The state machine needs to be drivable by external runtimes (Copilot SDK, Claude Code) via CLI commands or SDK calls.
- **Escalation UX.** When the review/fix loop exhausts its attempts, the user needs a clear signal and a way to intervene.

## Alternatives Considered

| Approach | Why Rejected |
|----------|-------------|
| Freeform workflow (agent decides) | Non-deterministic. Different runs produce different paths. No guarantees about what was reviewed or when. |
| Linear pipeline (no review loops) | First-pass code is rarely correct. Without review/fix iteration, quality depends entirely on initial generation. |
| Human-driven phase transitions | Defeats the purpose. The user becomes the bottleneck. Agents wait for humans to click "approve" at every stage. |
| Single review strategy for all phases | What makes a good spec review is different from what makes a good implementation review. One-size-fits-all reviews are shallow reviews. |

## References

- D-100: Workflow as a fixed-phase state machine (intake → spec → plan → impl → gate → ship)
- D-101: Review/fix loop at every phase with configurable attempt bounds
- D-102: Review is always a separate agent session (extends ADR-0012 to all phases)
- D-103: Directive as the state container with phase tracking and ledger integration
- D-104: Getting started workflow (backstop init → backstop new directive → agent takes over)
- ADR-0002: Six canonical artifact primitives (includes directive)
- ADR-0010: Verification kill chain (the GATE phase)
- ADR-0011: Provenance ledger (records phase transitions)
- ADR-0012: Review model (independent reviewer, "if it's green it ships")
