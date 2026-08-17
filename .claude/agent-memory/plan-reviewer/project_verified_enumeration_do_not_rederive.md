---
name: verified-enumeration-do-not-rederive
description: Plans that publish a "verified against the tree, do NOT re-derive" line-number enumeration must be re-swept by the reviewer — an omission there becomes a DIRECTED miss, not a self-correcting one
metadata:
  type: project
---

When a plan's blast-radius note says "verified against the tree <date> — do NOT
re-derive" and then enumerates specific file:line sites, RE-RUN THE SWEEP
YOURSELF. Grep the whole surface (e.g. `grep -rn "snippet:" --include="*_test.go"`)
and diff the real hit set against the plan's enumeration.

**Why:** PLAN-ISSUE-081-insert-placement-semantics (2026-08-11) enumerated 4
snippet sites in `apply_core_test.go` and 3 in `apply_substitution_test.go`; the
tree actually had 8 and 5. One omission (`apply_core_test.go:756`, the ApplyAll
composition test) carried a real output expectation. Because the plan told the
implementer NOT to re-derive, a normally self-correcting miss became a directed
one. The plan's own final verification task also demanded a grep whose expected
result was "zero hits, and 'in scope' is not a disposition" — unsatisfiable
against the sites the enumeration never named.

**How to apply:** the omitted sites are usually FAILURE-path fixtures (no output
assertion, so they still pass) plus one or two real expectations hiding in a
differently-named test. Check both: the failure-path ones break the plan's own
grep-based acceptance criterion; the expectation ones break the build. Files
being in a task's `files:` list is NOT a fix — the enumeration is what the
implementer executes.

**What a GOOD fix looks like** (that plan's fix pass, re-reviewed 2026-08-11 and
PASSED): the trust marker was downgraded to "RECONCILE against the tree, do not
trust"; the grep command was placed in BOTH the blast-radius note and the owning
task, to be run before editing AND before reporting done; every hit was bucketed
(migrated / declared by another task / explicitly not affected), including four
parse-only hits nobody had named; and the hit COUNT reconciled exactly against
the task split. Reviewer check: run the grep, count, and confirm the arithmetic
closes — 33 hits = 4 parse-only + 2 owned by the crux-test task + 27 owned by
the migration task, in that case.

**Engine-measured enumerations rot the same way, plus one extra vector: inline
suppressions.** PLAN-ISSUE-091 (2026-08-16) published a "MEASURED" 5-row list of
`*_test.go` findings `--all` drops, and its final task told the implementer to
expect "at minimum 5 rows in cmd/backstop alone". Re-running real semgrep gave 4:
`cmd/backstop/hermetic_fixtures_test.go:33` carries `// nosemgrep: <rule>` at HEAD,
so it can never fire under EITHER dispatch shape. The plan's headline counts were
also unreproducible because it never said which pack set it measured (go-standards
alone gave 6/7; all 8 installed packs gave 8/8; the plan claimed 9/11). Reviewer
check: re-run the tool, and demand the plan state the exact rule/pack set used.

Related: [[signature-change-package-fanout]], [[field-removal-fixture-scope]],
[[shared-helper-collapse-blast-radius]].
