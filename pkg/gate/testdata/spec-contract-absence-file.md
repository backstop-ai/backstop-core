---
number: SPEC-FIX-ABSENCE-FILE
implementation:
  package: pkg/gate
contracts:
  - file: pkg/gate/testdata/contract-absence-present.go
    provides:
      - name: legacyProbeSymbol
        kind: function
        absent: true
        scope: pkg/gate/testdata/contract-absence-present.go
---

# Fixture: file-scoped absence contract (TASK-003)

Declares a FILE-scoped absence: the forbidden symbol `legacyProbeSymbol` must not
appear in the named file. `scope` is the file itself. Drives
ExtractContractEntries' Scope population (CLM-040) and the file-scoped absence
probe end to end (CLM-041).
