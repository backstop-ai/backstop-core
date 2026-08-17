---
name: packval-phase3-family
description: PLAN-ISSUE-092's F7 named THREE independent reasons a real pack still cannot pass `pack test` phase 3 — (i) ISSUE-142, (ii) ISSUE-141, (iii) the baked semgrep-rule-id check, STILL UNFILED; they split across DIR-032/DIR-024 on shouts-vs-lies, and executor.go is a contended file
metadata:
  type: project
---

`pkg/packval` is the pack-AUTHORING validator (`backstop pack test`/`pack
check`); `cmd/backstop/pack_gate.go` is the runtime gate dispatch. **packval's
manifest/dispatch model predates the engine model, so it has drifted from the
gate path in at least four independent ways** — that shared root cause is why
these arrive in bursts and why each one needs its own home.

`PLAN-ISSUE-092`'s F7 review block (2026-08-16) enumerated THREE, and ordered
all three FILED rather than folded in — different owning surfaces, different
fixes:

1. **(i) ISSUE-142** — packval's `Rule` struct has no `Pattern` field at all;
   `pkg/pack`'s runtime `Rule` has carried one since SPEC-035 REQ-004. Every
   `pattern-arg` rule (all of `packs/contracts`) dispatches zero fixtures →
   vacuous green → **DIR-032 item 16**.
2. **(ii) ISSUE-141** — `RunEngine` never applies `binding.Convert`; raw
   non-SARIF output (ast-grep `--json` is a JSON ARRAY) hits `parseSarif`, which
   unmarshals into a STRUCT and errors. Fails LOUD → **DIR-024 item 13**.
   ⚠ Declared a **hard prerequisite for PLAN-ISSUE-092's phase 5** — that plan's
   verification depends on `packs/substantiveness` genuinely passing, which it
   cannot regardless of 092's own fix.
3. **(iii) STILL UNFILED as of 2026-08-16T21:41Z** — `RunFixtures` runs
   `semgrepFileContainsRuleID` on a rule's source file **regardless of the rule's
   declared engine**, so an ast-grep pack's `rule_path: ast-grep/sgconfig.yml`
   (content: `ruleDirs: [rules]`) fails it. A live **thin-executor violation**
   AND **bundle-mandated by BUNDLE-005 REQ-012**, so fixing it is a requirements
   question, not a code edit. **Check whether this got filed on every sweep.**

Add ISSUE-140 (narrow `*exec.Error`-only never-started check) and ISSUE-092
itself (`rule.File` vs `rule_path:`), and the same function carries FOUR defects.

**How to apply.**
- **`pkg/packval/executor.go` is a CONTENDED file.** PLAN-ISSUE-140 owns it for
  the never-started predicate (it landed the shared predicate in a NEW package,
  `pkg/check/never_started.go`, rather than editing the architecture policy) and
  PLAN-ISSUE-092 owns `phase3.go` — the two plans carry explicit
  file-exclusivity fences at each other. **Never recommend opening a lane on a
  packval defect without checking which lane currently owns the file**; say
  "sequence after X lands," not "fix it."
- **Coverage is nil BY CONSTRUCTION here, never by absence of evidence** — read
  the parent plan's F7/FOLLOW-ONS block, which names each sibling and marks the
  prerequisite. Zero interviews needed. See
  [[project_workaround_and_file_pattern]].
- **Expect a cross-directive dependency edge**, and do not "fix" it with a
  reorder: a DIR-024 (position 5) item now gates a DIR-032 (position 2) member's
  only in-flight plan. Promoting a 13-item catch-all to carry one prerequisite
  drags its eleven tier-2 items along. Propose a **sequencing ack** or a re-home
  ruling instead. See [[project_gate_verdict_honesty_cluster]] for the
  shouts-vs-lies discriminator that split this family across the two directives.
- The plan-reviewer's own memory (`project_packval_real_execution_premises`) says
  it best and is worth citing to a planner: **any plan asserting "pack X passes
  phase 3 once dispatch is restored" must be falsified pack-by-pack by RUNNING
  the declared engine command** — PLAN-ISSUE-092 built a whole final phase on a
  premise all three F7 defects falsify.
