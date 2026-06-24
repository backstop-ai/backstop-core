---
number: SPEC-FIX-ABSENCE-PATH
implementation:
  package: pkg/gate
contracts:
  - file: pkg/gate/testdata/contract-absence-present.go
    provides:
      - name: legacyProbeSymbol
        kind: function
        absent: true
        scope: pkg/gate/testdata
---

# Fixture: path-scoped absence contract (TASK-003)

Declares a PATH-scoped absence: the forbidden symbol must not appear anywhere
under the declared path `pkg/gate/testdata`. `scope` is a directory, exercising
the file-OR-path Scope parameter (CLM-010/CLM-040) and the scope-reaches-grep
end-to-end assertion (CLM-041).
