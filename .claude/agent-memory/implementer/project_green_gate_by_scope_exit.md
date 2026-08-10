---
name: green-gate-by-scope-exit
description: after a phase is committed its files LEAVE diff scope, so a gate that was expected RED goes green — the finding is not resolved, it is merely out of scope; prove liveness with a single --file run
metadata:
  type: project
---

PLAN-ISSUE-087 TASK-015 predicted `./bin/backstop gate` would exit NON-ZERO,
"fully accounted for by the grandfathered coverage set." Measured 2026-07-28 it
exited **0, every blocking dimension pass**. Nothing was fixed: the orchestrator
had committed phases 1-4, so the swept files left the diff and the only files in
scope were the current phase's.

Proof the exception was still live, not resolved — a single-file gate on an
UNMODIFIED grandfathered file:
```
./bin/backstop gate --file pkg/validate/directive.go
  coverage_threshold: fail (1)
  file pkg/validate/directive.go coverage 15/23 (statement) below threshold 80%
  exit 1
```

**Why:** a plan's "expect RED" prediction is written against the tree state at
authoring time. Once work lands, the same command means something different. A
green gate then reads as "the carve-out was cleared" when it means "those files
are no longer being looked at" — the vacuous-green failure this project exists to
prevent, arriving through scope rather than through a weakened check.

SHARPENING (ISSUE-098, 2026-07-28): forcing the file back into scope is only
HALF the check. Absence of a finding on an in-scope file is evidence ONLY if the
relevant step reports `pass`. A `skipped` step produces the identical "no
violations" reading while having measured nothing — so quote the step STATUS,
not just the violation count. The decisive line in that lane's report was
"coverage_threshold reports status pass, NOT skipped, with the file as the SOLE
member of scope"; without the status word it would have been another vacuous
green wearing a different hat.

**How to apply:** when a gate comes back greener than the plan predicted, do NOT
report it as the plan's expected end state. Diff the SCOPE first
(`gate --json` → `scope.files`), and confirm any carved-out finding is still live
with a single `--file` run on an unmodified member of the set. Note `--file` is a
STRING not a slice — two flags collapse to the last one, so per-file claims about
several files must come from separate runs or from the diff-scoped gate
([[project_gate_file_scope_nongo_dir_crash]]). Related:
[[project_gate_all_underreports_vs_diff]], [[project_editing_file_pulls_it_into_gate_scope]].
