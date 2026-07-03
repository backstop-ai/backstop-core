---
name: empirical-collateral-equality
description: For flag-day symbol-removal plans, prove the plan's scoped collateral file set EQUALS the grep-derived set per package, don't spot-check
metadata:
  type: project
---

When reviewing a flag-day removal plan (remove/rename a symbol that compile-breaks
_test.go callers + testdata fixtures), the decisive check is set-EQUALITY, not
spot-coverage: for each affected package, `grep -rln` every removed/renamed token
(the symbol, its `Fn` seam var, every struct field being dropped, the old yaml key,
old field literals) across prod+test, then diff that empirical file set against the
union of files scoped by the relevant impl tasks. PLAN-SPEC-031 passed because the
cmd/backstop empirical set (mergePackRules + runPackValidators + their Fn seams +
.Layer + extraSemgrepConfigs) was EXACTLY the 6 scoped files, and pkg/pack's 7
.Layer test files == the 7 scoped.

**Why:** SPEC-031 failed 3× on the same pattern — a removed symbol breaks a test
file whose breakage is invisible to field-removal alone (e.g. TestRunPackValidators_NoPacks
calls runPackValidators but sets NO .Layer literal, so grepping only `.Layer` misses it;
you must also grep the function name). Set-equality catches the symmetric-twin gap that
per-claim coverage and the validator both miss.

**How to apply:** Grep the DEFINITION-site token AND every call-site token AND every
field/yaml-key being removed, separately; map each test caller to its enclosing
`func Test...` with awk so you can confirm the description NAMES it; then assert the
file-set diff is empty per package. Also confirm symbols defined in an out-of-scope
sibling package with its OWN copy of the struct (pkg/packval's own Rule.Layer, no
pkg/pack import) are correctly NOT flagged. Relates to [[project_field_removal_fixture_scope]],
[[project_retired_feeder_test_collateral]], [[project_symmetric_field_removal_claims]].
