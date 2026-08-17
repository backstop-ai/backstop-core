---
name: prescribed-pattern-byte-shape
description: When a plan prescribes an awk/regex pattern against real tool output, `od -c` the actual line — prose separators (`\t` vs space) are routinely wrong and produce a silently non-firing guard
metadata:
  type: project
---

A plan task that says "track lines shaped `FOO\tbar\t[baz]`" is asserting BYTES. Run the
real tool and `od -c` the line before accepting it. Prose separators drift from reality,
and the same plan often spells the same shape correctly elsewhere — the inconsistency is
the tell.

**Why:** measured 2026-08-16 reviewing PLAN-ISSUE-067. Its converter-widening task
prescribed tracking `FAIL\t<import-path>\t[build failed]` lines as the floor that keeps a
non-zero run off the crash path. The real `go test` byte shape is `FAIL` + TAB +
`<import-path>` + SPACE + `[build failed]` — ONE tab, not two. The plan spelled it
correctly in six other places including four lines earlier in the SAME task. A literal
two-tab awk pattern never matches, so the floor never fires and the guard is quietly
absent — the "goes quiet" class the plan itself was written to close.

**How to apply:** for any plan prescribing converter/parser/grep patterns, reproduce the
tool output yourself and byte-compare. Also diff every occurrence of the shape WITHIN the
plan: a self-contradiction between two tasks means at least one is wrong, and the fixture
capture (if the plan mandates real captures) is the authority. Related:
[[producer-swap-argv-subcommand]], [[fixtures_from_real_output]].
