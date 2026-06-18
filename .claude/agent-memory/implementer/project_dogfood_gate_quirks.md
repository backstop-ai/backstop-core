---
name: dogfood-gate-quirks
description: Two gate/dogfood-rule limitations surfaced building SPEC-031 engine dispatch; how they were fixed without weakening
metadata:
  type: project
---

Building SPEC-031 (pluggable engine dispatch) surfaced two genuine gate/dogfood issues,
both fixed by *increasing precision*, never weakening enforcement.

1. **Contract checker collapsed defined types.** `pkg/gate/step_contract.go findType`
   rendered `type InputMode string` as bare `type InputMode`, so any spec contract
   pinning a defined type over a primitive/composite (string/int/map/slice) was
   unsatisfiable. Fixed by adding `underlyingTypeString` to render the underlying type
   (ident/selector/star/slice/map); unrenderable shapes (func/chan/fixed-array) fall
   back to the bare name. Now `type X string` matches AND a wrong underlying fails.

2. **`no-global-mutable-state` dogfood rule false-positived on `const`.** The go-standards
   pack rule `var $X = ...` matches typed `const Name T = value` specs in semgrep's Go
   grammar (~100 hits across the repo's existing immutable constants). Added
   `pattern-not: const $X = ...` to `.backstop/packs/backstop/go-standards/rules/core/
   go-core.yml` so the rule matches its stated intent (mutable *variable*) and stops
   flagging immutable constants. Verified it still catches real `var $X = ...` in the
   pack's own invalid fixture.

**Why:** the SPEC-031 EngineBinding leaf package needs typed string/int enums (InputMode,
ScopeKind) and the spec contract pins `type InputMode string`. Both limitations would have
forced gaming the gate or weakening; precision fixes were the correct path.

**How to apply:** when a dogfood rule or gate check false-fires on legitimate code,
prefer a precision fix (narrow the match / improve the renderer) over a suppression or
threshold change. Function-local `var x T = ...` in tests also trips the rule — rewrite
the test to avoid the form rather than broadening the rule's scope.
