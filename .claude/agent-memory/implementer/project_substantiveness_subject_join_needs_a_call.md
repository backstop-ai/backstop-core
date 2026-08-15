---
name: substantiveness-subject-join-needs-a-call
description: "test_substantiveness 'does not call package X' is satisfied ONLY by a package-qualified CALL in the test's own body — a selector reference like gate.StepFoo, or a `var _ gate.Iface = v` assertion, does NOT count (and the latter trips two more rules)"
metadata:
  type: project
---

The gate's `test_substantiveness` noTarget join (`pkg/gate/substantiveness_join.go`) keys on the
go-substantiveness pack's Q2 extraction rule (`ast-grep/rules/referenced-symbol-go.yml`). That rule
matches ONLY a `kind: call_expression` whose function is a `selector_expression` with a PACKAGE
operand — a literal `pkg.Func(...)` inside the test declaration.

**Why:** the pack is spec-unaware; it reports the symbols a test REFERENCES via calls, and the gate
alone joins them against the spec's declared subject. A bare selector (`gate.StepCoverageThreshold`)
is not a call_expression, so it extracts nothing and the test still reports "does not call package
gate". A method call on a value of a gate type (`classifier.IsMeasurableSource(...)`) also does not
count — the operand must be the package identifier.

**How to apply:** when a mandated test delegates all its `pkg.` work to a helper and the join goes
red, add a REAL package-qualified call to the test body — one that carries meaning, e.g.
`gate.NewGateResult([]gate.StepResult{res})` to assert the step actually fails the gate it is part
of, or `gate.NewSourceClassifier(nil, nil)` as an explicit negative control. Do NOT reach for
`var _ pkg.Iface = v`: it satisfies nothing (not a call) AND fires two more rules — go-standards
`no-global-mutable-state`, whose pattern is a bare `var $X = ...` and therefore matches
FUNCTION-SCOPE var declarations too, plus staticcheck `QF1011`.

There IS a sanctioned escape for tests that legitimately do not call their subject: the claim-level
`kind: absence` annotation, honored by `NoTargetViolationForTest`. Use it only for genuine absence
proofs, never to quiet a test that really does exercise the package through a helper.

See also [[gate-scope-entry-surfaces-pack-false-positives]].
