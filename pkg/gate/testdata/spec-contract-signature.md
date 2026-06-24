---
number: SPEC-FIX-SIGNATURE
implementation:
  package: pkg/gate
contracts:
  - file: pkg/gate/testdata/contract-sig-present.go
    provides:
      - name: RouteFile
        kind: function
        signature: "func RouteFile(path string, mode int) (string, error)"
---

# Fixture: present-signature contract (TASK-003)

Declares a present-signature contract so extraction carries a Signature through
UNMODIFIED (CLM-042) — no parse, no AST-walk, no compile.
