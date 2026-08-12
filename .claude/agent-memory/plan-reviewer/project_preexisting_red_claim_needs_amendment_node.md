---
name: preexisting-red-claim-needs-amendment-node
description: A spec claim already RED against the unmodified tree is a founder-decision blocker — demand either a DAG node for the amendment or evidence the spec was already amended, then re-measure the AMENDED predicate yourself
metadata:
  type: project
---

When a plan's own notes report that a mandated claim is RED against the tree BEFORE
any of the plan's work (e.g. PLAN-SPEC-067's CLM-050 vs `cmd/backstop/baseline.go`),
verify the red empirically, then check the plan models the RESOLUTION.

**Why:** a "write it honestly, record the hit list, STOP and report" note diagnoses
correctly but leaves the DAG running past it. The plan then terminates with (a) the
repo's own `go test` RED — you cannot commit a knowingly-failing mandated test — and
(b) the spec unable to honestly flip to `implemented`, which is usually the plan's
whole stated purpose (unblocking a downstream directive). Neither consequence is
named by the plan, so it reads as handled when it is not.

**How to apply:** TWO valid resolutions, and you must tell them apart.
1. **In-DAG node** — a task for "founder picks disposition; spec-author amends; the
   test is written to the amended letter". The plan already knows this shape: it
   makes publication an explicit founder-gate node.
2. **Already amended out-of-band** — the ruling happened DURING planning and the spec
   on disk already carries it (SPEC-067 1.0.3, 2026-08-11: the `github` module-path
   exemption widened to cover BOTH `github.com/` and the regex-escaped `github\.com`,
   staying case-sensitive; case-insensitive and "CI-shaped literals generally" were
   explicitly REJECTED). Then the plan correctly carries NO node, and the right text
   is "expect GREEN from the first run", not a tolerated red.

For (2), do NOT trust the plan's summary of the amendment: read the amended claim
text, then re-measure the AMENDED predicate against the real tree yourself
(occurrence-level, not line-level — one line can hold an exempt and a non-exempt
hit). Confirm the OLD predicate would still have been red, which is what proves the
amendment was load-bearing rather than cosmetic. Also confirm the amendment left no
dangling reference to a decision node that no longer exists, and that any surviving
"founder gate" in the DAG is a DIFFERENT gate (publication) rather than the retired
one. Relates to [[repurposed-test-claim-text-drift]] and
[[coverage-rewrite-predating-spec-drift]] — same family: the claim TEXT, not the test
name, is what strands.
