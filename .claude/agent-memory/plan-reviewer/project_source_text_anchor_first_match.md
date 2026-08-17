---
name: source-text-anchor-first-match
description: Source-text pin tasks that anchor on a bare symbol name select the WRONG comment block — Go doc convention puts the name in its own comment first; count occurrences and check the preceding field
metadata:
  type: project
---

Any plan task that prescribes "read the source file, isolate the doc comment attached to
symbol X, assert it does NOT contain literals L" must be checked with
`grep -n X <file>` FIRST. In Go, the doc-comment convention opens with the symbol name
(`// ExemptFromScopeFilter, when true, ...`), so the name occurs at least TWICE: once as
prose, once as the declaration. A naive first-match-then-walk-backward anchor lands on the
COMMENT line and collects the PRECEDING declaration's doc block instead.

**Why:** the wrong block is a double failure, not a single one — the NEGATIVE assertion
("names no engine") passes VACUOUSLY on an unrelated block, while the POSITIVE assertion
("still mentions ScopeKind") fails with a message pointing at the wrong target, so the
implementer debugs the wrong thing. Measured on `pkg/pack/engine/binding.go`
(PLAN-ISSUE-136 TASK-011, 2026-08-17): `ExemptFromScopeFilter` at line 251 (comment) and
261 (declaration); first-match selects `FieldContract`'s block, which names no engine and
never says `ScopeKind`.

**How to apply:** when reviewing a source-text pin task, (1) grep the anchor symbol and
count hits; (2) read the PRECEDING declaration's comment block and evaluate BOTH the
negative and positive assertions against it — if the negative passes there, the anchor is
a blocker; (3) prefer an anchor that is unique by construction, e.g. a yaml/json struct
tag literal (`yaml:"exempt_from_scope_filter"`, verified exactly one occurrence) or
"trimmed line starts with the name AND does not start with `//`". Also check the
whole-file alternative: other doc comments legitimately naming the banned literals make a
whole-file scan permanently red, and the obvious "fix" is to gut accurate unrelated prose.

Related: [[verified_enumeration_do_not_rederive]], [[deleted_symbol_named_in_kept_comment]].
