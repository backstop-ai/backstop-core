---
name: close-out-must-rerun-gate-after-flip
description: Flipping a spec to `implemented` is what first activates contract_signature and test_substantiveness on it — always re-run the gate AFTER the flip, never before
metadata:
  type: feedback
---

A spec close-out is not a status edit. Flipping `draft` -> `implemented` is the moment two gate
dimensions start reading the artifact for the first time, so the close-out sequence must be:
**flip, THEN run `artifact validate` and the diff-scoped gate, then record what those post-flip
runs actually said.** A pre-flip green says nothing about either dimension.

**Why:** enforcement is status-scoped by design.
- `contract_signature` collects `contracts` blocks only from `implemented` specs.
- `test_substantiveness` enforces only specs at `implemented` status.

So an entire class of defect is structurally invisible for the whole life of the spec — through
authoring, spec review, planning, plan review, implementation, and impl review — and surfaces only
at close-out. Two consecutive close-outs each hit a different member of that class:
- **SPEC-069 1.3.3** — five claims with no `subject:` inherited the spec default and their
  mandated tests sat in another package. The flip reddened the gate with four violations.
- **SPEC-070 1.1.4** — a `provides` entry named `doctorCheckIDs`, a label for an anonymous
  grouped `const (…)` block, i.e. a symbol Go cannot have. The flip reddened
  `contract_signature` with one violation.

Both were spec-only fixes; no source, no test, no waiver. That is the usual shape — the
implementation is typically correct and the DECLARATION is what drifted.

**How to apply:** before flipping, pre-audit the two joins (every claim's `subject:` vs where its
mandated tests actually live; every `provides` symbol's real existence and expressibility). Expect
to fix something. After flipping, re-run and read the result — and if the gate goes red, fix the
spec rather than reverting the flip. Note which one you ran: the repo convention (SPEC-068 1.2.9,
SPEC-069 1.3.4, SPEC-070 1.1.5) is to scope every "gate passes" claim explicitly to DIFF-SCOPED,
because `gate --all` is red for inherited reasons and that red must be proven inherited with a
HEAD control run rather than assumed.

Related: [[grouped-const-contracts-inexpressible]],
[[omitted-subject-inherits-wrong-package]], [[claim-subject-is-one-package-only]].
