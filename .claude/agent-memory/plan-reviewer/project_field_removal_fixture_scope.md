---
name: field-removal-fixture-scope
description: When a plan removes a struct field / makes a key required (flag-day), it must scope the EXISTING test files and testdata fixtures that set the old field, not just production code
metadata:
  type: project
---

When a plan removes a struct field or flips a key from optional to required under a flag-day (no-grandfather) migration, the breakage surface is not only production `.go` files that read the field — it is every **existing test file** that sets the field literally and every **existing testdata fixture** that declares the old key. Those compile-break (removed field) or fail-at-runtime (old key now a hard ConfigError), and the implementing agent has no authorization to touch them unless a task scopes them.

**Why:** Caught in PLAN-SPEC-031. TASK-006 removed `Rule.Layer` and made empty `engine` a ConfigError, but the plan scoped only the 3 production files (`pack_gate.go`, `manifest.go`, `validate_manifest.go`). Eight existing test files set `.Layer = 1/2/3` directly (`pkg/pack/validate_layer2_test.go`, `validate_layer3_*_test.go`, `validate_manifest_test.go`, `validate_layout_test.go`, `validate_proof_test.go`, `cmd/backstop/pack_gate_test.go`, `pack_gate_helpers_test.go`) and six testdata `pack.yml` fixtures declared `layer:` with zero `engine:` keys. All break; none were in any task's file scope. The final full-gate task (TASK-018) would just fail with no task allowed to migrate them.

**How to apply:** For any field-removal / required-key claim, grep the WHOLE repo (incl. `_test.go` and `testdata/*.yml`) for the old field/key, subtract the files already in task scope, and flag the remainder. New fixtures added by a setup task do NOT cover the obsolescence of the *existing* ones. Related to [[retirement-claim-scope]] and [[project-dispatch-consumer-edges]].
