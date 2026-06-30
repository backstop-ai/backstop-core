---
name: deletion-file-premise-audit
description: When a deletion plan lists N test files as "all reference deleted symbols / won't compile", grep each one — some couple only to SURVIVING symbols
metadata:
  type: project
---

When a strangler/deletion plan enumerates a fixed set of test files to
"delete-or-migrate" with the rationale "they all reference the deleted symbols
and won't compile" (a Sharp-Edge-9-style premise), do NOT trust the count — grep
each named file for the actual deleted-symbol set.

**Why:** In PLAN-SPEC-038 the spec listed 5 analyzer-coupled test files
(step_contract_test.go, _absence_test.go, _absence_config_test.go,
_noregress_test.go, _parser_absence_test.go). Four referenced deleted go/parser
symbols. The fifth (`step_contract_parser_absence_test.go`) referenced ONLY
`ExtractContractEntries` and `ContractEntry` — both SURVIVING symbols (REQ-012
even EXTENDS ExtractContractEntries). It compiles fine post-deletion and covers
surviving `Absent`-field extraction. The "won't compile, delete-or-migrate"
framing invites an implementer to just delete it, dropping surviving coverage.

**How to apply:** For each file a deletion task lists, `grep -cE "<deleted symbol
union>" <file>`. A zero count means that file is NOT compile-coupled to the
deletion — it must be explicitly classified as MIGRATE (surviving behavior) or
its inclusion challenged, not swept under a blanket "won't compile" rationale.
Cross-check whether the surviving behavior it covers is re-covered by a mandated
test elsewhere in the plan; if not, that's lost coverage even when no spec claim
mandates the old test name. Related: [[whole-file-delete-mandated-test-readsource]].
