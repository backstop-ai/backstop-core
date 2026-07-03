---
name: netnegative-gate-baseline
description: On net-negative/behavior-preserving diffs, gate pack_engines goes RED on PRE-EXISTING findings that surface only because a touched file entered diff scope and no local baseline suppresses them
metadata:
  type: feedback
---

On a NET-NEGATIVE or behavior-preserving diff (deletions/edits that touch a file
but add no new violations), `./bin/backstop gate` can still go RED on
`pack_engines` because semgrep runs over the whole CHANGED FILE, not changed
lines — so PRE-EXISTING findings on untouched lines of a file you edited surface
in scope. Locally there is usually NO baseline (`baseline_comparison` skips:
"missing origin remote / no .backstop/baseline.json"), so nothing suppresses them.

**Why:** baselines are CI-generated post-merge and pulled on demand
([[project_baseline_ci_pull]]); a local dev/agent run has none. The gate then
cannot tell your net-new findings from inherited ones.

**How to apply:**
- Before treating gate `pack_engines` RED as your defect, run a per-file
  HEAD-vs-NOW finding count: `git show HEAD:<f> > /tmp/h.go` then
  `semgrep --config .backstop/packs/backstop/go-standards/rules <f> --json` on
  both and compare counts. If NOW <= HEAD for every changed file, your diff
  introduced ZERO net-new findings — the red is pre-existing/environmental.
- Fix ONLY findings your diff actually introduced (NOW > HEAD). Common
  self-inflicted false-fires from the coarse `var $X = ...`
  (no-global-mutable-state) and `_, _ = $FUNC(...)` (no-ignored-errors) patterns:
  a local `const x = ...`/`var x = ...` in a test, or an ignored os.ReadDir error.
  Dodge the pattern without weakening intent: `const x =` -> `x :=`;
  `var ce error = &T{}` -> `ce := error(&T{})`;
  `_, _ = os.ReadDir(d)` -> `if _, err := os.ReadDir(d); err != nil { ... }`.
- The no-global-mutable-state rule (`var $X = ...`) FALSE-FIRES on a
  `const (...) iota` enum block (semgrep treats const value-specs like var). Same
  family as [[project_dogfood_gate_quirks]] — fix by rule PRECISION, NEVER by
  weakening, and NOT inside an unrelated deletion plan (out of scope).
- Do NOT `baseline generate` to paper over inherited findings (CI-publication-only;
  would bake the false-fire in as "accepted"). Do NOT weaken pack rules. STOP-AND-
  REPORT to the USER (not the coordinator) with the HEAD-vs-NOW proof when the only
  red is inherited.
