---
name: substantiveness-extraction-semantics
description: The test_substantiveness Q2 extraction rule matches ONLY package-qualified CALLS (ast-grep call_expression + selector_expression) — type references, method values and helper-mediated calls never enter the join's referenced set, while a local variable receiver like `t.Errorf` DOES; run the rule with ast-grep to settle any "does this test satisfy the join" claim
metadata:
  type: project
---

`.backstop/packs/backstop-ai/go-substantiveness/ast-grep/rules/referenced-symbol-go.yml`
is the Q2 extraction rule feeding `NoTargetViolation`
(`pkg/gate/substantiveness_join.go:63`). Its shape, measured 2026-08-16:

```
rule: kind: call_expression, inside a func whose name matches ^Test,
      has: field: function, kind: selector_expression,
           has: field: operand, pattern: $PKG (constraint: kind: identifier)
```

**Consequences that decide triage arguments:**
- A bare TYPE reference (`[]gate.Violation{}`, `map[string]engine.Binding{}`)
  is a composite literal, NOT a call — it is **never extracted**. Any issue
  claiming "a decorative `pkg.Something` reference would satisfy the checker"
  is wrong; that trap does not exist.
- A call reached through a **same-package helper** (the helper's signature
  names `gate.Violation`, the test only calls the helper) is invisible — the
  rule sees one hop only.
- `t.Errorf(...)` DOES match: the rule cannot distinguish a package qualifier
  from a local variable receiver, so `t` shows up as a "referenced package."
  Over-broad on receivers, under-broad on types/indirection.

**How to apply:** never reason about substantiveness-join membership from
reading a test body — RUN it:
`ast-grep scan --rule <that rule file> <test file>` and read the
`test X references package Y` lines. That is the exact input the join
consumes. `./bin/backstop gate --file ...` is NOT a substitute — it dies at
`pack_engines` (step 3 of 12) whenever `go-arch-lint` is off PATH, so
`test_substantiveness` never runs locally.

**PLAN-ISSUE-113 pins this as intended.** Its refusal fires ONLY when the
extraction partition is empty project-wide; CLM-007 is a non-regression pin
that with evidence present, an unannotated not-in-set test STILL raises. So
"the join fired on my legitimate test" is a CORPUS problem (annotate
`kind: absence`, or make the call real), not a mechanism defect — do not
home such issues at DIR-032.

Applied on ISSUE-138 (2026-08-16): the issue named three dormant tests; the
rule run found **five** (SPEC-041 CLM-018/019 + CLM-004/005/006) and
falsified the issue's central premise about CLM-019 passing by accident.
See [[feedback_verify_the_loss_claim]], [[project_gate_verdict_honesty_cluster]],
[[project_pack_rule_precision_family]].
