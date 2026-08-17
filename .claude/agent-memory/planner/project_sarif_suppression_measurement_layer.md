---
name: sarif-suppression-measurement-layer
description: A semgrep finding count is ambiguous until you name the LAYER — raw SARIF rows overstate gate output because --sarif emits nosemgrep-suppressed results that parseSarif then drops
metadata:
  type: project
---

When a plan predicts what `backstop gate` will report, RAW SARIF row counts are the
wrong number and will produce a false prediction. State the layer with every count.

**The mechanism (verified at HEAD, semgrep 1.156.0):**
- semgrep's `--sarif` formatter EMITS `nosemgrep`-suppressed findings as `results[]`
  entries carrying `suppressions: [{"kind":"inSource"}]` rather than dropping them.
  `--json` DROPS them. **The two formatters disagree**, and backstop uses `--sarif`.
- `parseSarif` in `pkg/check/parsers.go` then SKIPS any result with non-empty
  `Suppressions` (the shipped ISSUE-017 mechanism). This is the exact parser
  `runFindingsEngine` feeds via `check.ParsePackFindings`.
- So: raw rows − live `nosemgrep` annotations in scope = what the gate prints.

**Why:** PLAN-ISSUE-091 review round 2 "corrected" round 1's active-layer numbers to
raw-layer numbers and called the originals wrong. They weren't. The bad correction
also produced a *false finding* — that a `// nosemgrep` annotation was broken because
"prose trailing the rule id" defeated it. Falsified: trailing em-dash prose and a bare
`// nosemgrep` suppress identically; the row was suppressed all along, just visible in
raw SARIF. A whole review round was spent on it.

**How to apply:** label every count in a plan with BOTH its pack set AND its layer
(RAW vs POST-SUPPRESSION-FILTER ACTIVE). Predicted gate readings may only be quoted in
the active layer. When a count is unchanged, check whether gains and losses cancel —
an unchanged TOTAL is not an unchanged SET, so diff (File, Rule, Line) tuples rather
than comparing counts. Counts taken from `dispatchPackEngines` return values are
inherently active (downstream of `parseSarif`); counts from a raw `semgrep --sarif`
invocation are not. See [[feedback_verify_issue_premises]] and
[[feedback_state_a_sweep_once]].
