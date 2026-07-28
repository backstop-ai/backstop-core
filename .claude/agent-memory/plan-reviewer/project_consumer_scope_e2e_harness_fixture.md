---
name: consumer-scope-e2e-harness-fixture
description: Consumer-level status-scoping blast radius must include non-_test.go E2E harness SOURCE files that embed draft spec fixtures; distinguish vacuous-pass from red
metadata:
  type: project
---

When a plan scopes a gate dimension to `implemented` specs by filtering at the
CONSUMER (e.g. buildTestSubstantivenessStep / StepTestVerificationScopedFunc drop
mandated tests whose spec != implemented), the fixture blast radius is NOT just
`*_test.go` files.

**Why:** cmd/backstop has E2E harness helpers that are plain `package main`
`.go` files (e.g. `gate_substantiveness_e2e.go`, `gate_substantiveness_e2e.go`'s
`newE2EWorkspace`) which embed a `status: draft` spec string and drive the real
consumer. A curated "grep the test files" list misses these. Grep every
`status: draft` / statusless `.spec.md`-shaped string across BOTH `*_test.go` AND
non-test `.go` sources in the consuming package.

**How to apply:** For each hit, read the CONSUMING test's assertion and classify:
- POSITIVE assertion (expects a Q2 noTarget "does not call package" violation to
  FIRE) → dropping the draft mandated test makes it go RED. (PLAN-ISSUE-054 missed
  `gate_substantiveness_e2e.go` → `TestE2E_SubstantivenessMultiRuleDispatch_And
  SandboxedConvert` breaks red.)
- NEGATIVE assertion (expects NO noTarget violation, e.g. colocation short-circuit)
  → dropping the draft mandated test makes it pass VACUOUSLY (hides the real
  behavior). Flip to `implemented` to keep it honest.
- Q1 hollow findings are pack-produced over test files independent of the mandated
  list, so status-scoping does NOT affect them (the hollow-red E2E stays green).

Also: exporting an unexported predicate for a cross-package consumer — prefer an
ADDITIVE exported wrapper over renaming, because the rename ripples into
already-landed sibling test files (contract_implemented_scope_test.go references
lowercase `contractsAreDue`) that are outside the plan's task file scope. Relates
to [[project_signature_change_package_fanout]].
