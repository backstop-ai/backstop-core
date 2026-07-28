---
name: code-motion-shifts-gate-scope
description: Moving code between files relocates its pre-existing findings and recomputes per-file coverage ratios — the gate reds on inherited debt, so measure HEAD-vs-NOW per file before assuming a regression
metadata:
  type: project
---

A refactor that MOVES a function body from file A to file B makes the gate report
findings and coverage failures that look new but are not. Both gate dimensions are
per-file, so relocation alone changes the verdict:

- **pack_engines**: the semgrep findings travel with the lines. SPEC-055 phase 11
  moved the update+upgrade pipelines into `command.go`; the gate reported 25
  `error-wrapping-required` violations there, but a package-wide HEAD-vs-NOW scan
  showed 37 → 36 total. Net-new was ZERO.
- **coverage_threshold**: the ratio is recomputed over what REMAINS. `add.go` went
  90.1% → 88.5% and crossed the 90% floor without a single newly-uncovered
  statement — 114 well-covered statements left for `command.go`, so the same
  pre-existing uncovered error branches now dominate a smaller denominator.

**Why:** per-file scope plus whole-file diff scope means "the file I touched is
red" says nothing about whether I caused it. Reporting it as a regression wastes
the lead's time; silently inheriting it violates you-touch-it-you-fix-it.

**How to apply:** before fixing or reporting, build the comparison. `git worktree
add <scratch> HEAD` (NEVER `git stash` — see [[feedback_never_stash_shared_tree]]),
run the same pack rule file and the same `-coverprofile` in both trees, and diff
per file. Then: fix what is genuinely yours, fix what a threshold crossing forces
even if the underlying debt is inherited, and report the rest with the measurement
attached. Relates to [[feedback_netnegative_gate_baseline]] and
[[feedback_gostandards_rule_mechanics]].
