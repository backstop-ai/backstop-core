---
name: gostandards-rule-mechanics
description: go-standards semgrep + go-toolchain rule quirks that fire on touched files under gate scope — comma-underscore AND single-underscore ignored-errors, constructor-injection FP, error-type-suffix FP, the errcheck<->no-ignored-errors cleanup-defer pincer, and that go-toolchain analyses are package/repo-level not per-file
metadata:
  type: feedback
---

When your edit pulls a file into `backstop gate` diff scope, EVERY go-standards /
go-toolchain finding in that whole file becomes blocking (not just your new lines,
not just source — test files too), UNLESS the finding's content-based identity is in
`.backstop/baseline.json`. The local baseline is often stale/mismatched (its `git_sha`
!= HEAD), so it won't grandfather pre-existing debt on files you touch. Net effect: to
green pack_engines you must drive the touched file to ZERO in-scope findings.

**Why:** ISSUE-025 (pack install materialize). Cleaning add.go/install.go source was
not enough — editing add_test.go / install_test.go pulled ~37 PRE-EXISTING
`no-ignored-errors` + a `constructor-injection` into blocking scope. Coordinator
confirmed: "a pre-existing finding pulled into diff scope by your edit counts as NEW;
the ratchet requires cleaning the files you touched." Eliminating all findings in the
touched files (source + tests) made pack_engines pass regardless of baseline staleness
(0 in-scope findings -> nothing to compare).

**Rule mechanics (go-standards/rules/core/go-core.yml):**
- `go.core.no-ignored-errors` (severity ERROR) matches `$VAL, _ := $FUNC(...)`,
  `_, _ = $FUNC(...)`, AND a single `_ = $FUNC(...)` — even when the ignored value is
  NOT an error (e.g. `projectDir, _ := setupInstallProject(t)` where the 2nd return is a
  `*Lockfile`). Fix by handling: route optional reads through a `mustX`/`readFileOrNil`
  helper that checks the error, drop the unused return from the signature, or convert
  `_ = f()` to `if err := f(); err != nil { ...handle... }`.
- `go.core.error-wrapping-required` (WARNING) matches a SINGLE-statement body
  `if $ERR != nil { return ..., $ERR }` inside a `func(...) (..., error)`. A body with
  another stmt before the return (e.g. `rollback(); return nil, err`) is NOT matched.
  Fix by wrapping: `return nil, fmt.Errorf("...: %w", err)`. Coverage-neutral.
- `go.core.constructor-injection` (WARNING, GO-005) false-fires on a struct literal
  whose FIELD NAME matches `(?i).*(repo|client|store|db|database|cache|logger|service|
  gateway|adapter|provider).*`. `CachePath: cacheFlag` trips it via "cache". Fix CHEAPLY
  by hoisting that field out of the literal — `opts := X{...}; opts.CachePath = cacheFlag`.
- `go.core.error-type-suffix` (GO-standards) FALSE-FIRES on already-compliant types: it
  flagged `type ExitCodeError struct{...}` (literally ends in `Error`, has `Error()`).
  Treat as a pack FP; do not rename. Report, don't churn.
- go-toolchain errcheck: unchecked `os.RemoveAll/Remove/MkdirAll/WriteFile`. For genuine
  fire-and-forget cleanup use `_ = os.RemoveAll(...)` (and `defer func() { _ = os.RemoveAll(x) }()`
  for defers — the codebase idiom); for the MATERIALIZATION path check + propagate wrapped.

**errcheck <-> no-ignored-errors PINCER on cleanup defers (SPEC-050 finding):**
`defer func() { _ = os.RemoveAll(x) }()` satisfies errcheck but STILL trips go-standards
`no-ignored-errors` (single `_ =` is matched). A bare `defer os.RemoveAll(x)` trips
errcheck instead. NO `_`-based form clears both; genuine handling also fails (Fprintf
logging re-trips errcheck; joining the cleanup error into a named return changes
behavior). The `defer func(){ _ = ... }()` form is still strictly BETTER (1 finding not
2 — clears errcheck). The residual no-ignored-errors on a benign cleanup defer is only
fully clearable with a `// nosemgrep`; if nosemgrep is scoped out, report it as an
unresolvable pincer, do not weaken/waive.

**go-toolchain analyses (errcheck/unused/staticcheck) are PACKAGE/REPO-LEVEL, not
per-file-diff.** Editing ONE file in a package surfaces that whole package's (and often
the whole repo's) pre-existing errcheck/unused/staticcheck debt in pack_engines — SPEC-050
touched pkg/validate + cmd/backstop and the gate reported ~97 findings across UNMODIFIED
files in pkg/check, pkg/scaffold, pkg/packval, pkg/pack/distribution, pkg/gate,
tests/smoke. This is the repo-wide baseline, unfixable without a repo-wide toolchain
cleanup (out of any feature's scope). Prove net-new-0 by showing findings sit on files
NOT in your `git status`; report the baseline — do NOT weaken/waive.

**How to apply:** after editing any tracked .go, run `backstop gate` and grep the
pack_engines output for EACH file you touched (source AND test). Drive every one to
zero where cleanly possible; where a rule-pincer or FP blocks (see above), prove net-new-0
(findings on files outside your `git status`) and report. See
[[editing-file-pulls-it-into-gate-scope]] and [[netnegative_gate_baseline]].
