---
title: "Packval Pattern Arg Fixtures Never Dispatch"
schema_version: issue/v1

issue:
  id: ISSUE-142
  title: "Packval Pattern Arg Fixtures Never Dispatch"
  type: bug
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# Packval Pattern Arg Fixtures Never Dispatch

## Problem

`pkg/packval`'s `Rule` struct (`pkg/packval/manifest.go:75-88`) has no `Pattern` field. The
gate-runtime `Rule` model in `pkg/pack/manifest.go` DOES have one (`manifest.go:166`, `Pattern
string `yaml:"pattern"``, added under SPEC-035 REQ-004) — it is how `pattern-arg` type rules
declare their match expression: an inline `pattern:` string, with neither `file:` nor
`rule_path:` set.

Every rule in `packs/contracts/pack.yml` is declared this way. Reproduced directly:

```
$ grep -c rule_path packs/contracts/pack.yml
0
$ grep -c "pattern:" packs/contracts/pack.yml
7
```

All 7 of the pack's rules are `pattern-arg` type, none declare `rule_path:` or `file:`.

`pkg/packval/phase3.go`'s fixture-dispatch loop (`RunFixtures`, `phase3.go:28-113`) gates every
piece of fixture-execution work on `rule.File != ""`:

```go
for _, rule := range pack.Content.Ruleset.Rules {
    var binding engine.EngineBinding
    haveBinding := false
    if rule.File != "" {          // phase3.go:31 — false for every pattern-arg rule
        ...                        // semgrep-rule-id cross-check, engine binding resolution
    }
    for _, claim := range rule.Claims {
        for _, f := range claim.Fixtures.Positive {
            if rule.File != "" && haveBinding {   // phase3.go:62 — false for every pattern-arg rule
                r, err := executor.RunEngine(packDir, binding, []string{rule.File, f.Path})
                ...
            }
            ...
        }
        for _, f := range claim.Fixtures.Negative {
            if rule.File != "" && haveBinding {   // phase3.go:76 — same
                ...
            }
        }
    }
}
```

For a `pattern-arg` rule, `rule.File` unmarshals to `""` — there is no struct field to catch the
YAML `pattern:` key at all, so it is silently dropped during unmarshal. Both `phase3.go:62` and
`phase3.go:76` guards evaluate false, so `executor.RunEngine` is **never invoked** for any
`pattern-arg` rule's fixtures, positive or negative. `res.Errors` stays empty and `res.Status`
stays `"pass"` regardless of whether the rule's fixtures actually demonstrate anything — the same
result `backstop pack test`/`pack check` would report for a rule that fires correctly.

This means `packs/contracts` — all 7 rules, 100% of the pack — can never have its fixtures
validated through `backstop pack test`/`pack check`. A negative fixture could be rewritten to
violate nothing, or a positive fixture rewritten to violate the rule, and `phase3-fixtures` would
still report `pass`.

## Root cause

Sibling defect to ISSUE-092 (`Pack Test Phase3 Fixtures Cannot Fail`), same class, different
field. ISSUE-092 diagnosed that `pkg/packval`'s `Rule` struct reads a rule's source file from
`file:` while every real `rule_path:`-declared pack.yml (semgrep/ast-grep rule-file rules) leaves
that field empty, so `rule.File != ""` never gates true for those rules either — dispatch is dead
for the entire `rule_path:` rule-declaration style.

This issue covers the other structurally distinct rule-declaration style entirely: `pattern-arg`
rules, which declare neither `file:` nor `rule_path:`, only an inline `pattern:` string consumed
by an engine whose `input_mode: pattern-arg` / `input_flag: --pattern` (see
`packs/contracts/pack.yml`) passes that string directly as a command argument rather than reading
it from a rule file on disk. `pkg/packval`'s `Rule` struct has no field for `pattern:` at all —
not a wrong key name (ISSUE-092's shape), but a missing one. `pkg/pack/manifest.go`'s
gate-runtime `Rule` struct already carries this field (`Pattern string `yaml:"pattern"``,
`manifest.go:166`, SPEC-035 REQ-004) — `pkg/packval`'s independently-maintained `Rule` struct was
never updated to match when that field was added to the runtime model, so the same "two Rule
models, one drifts" root shape ISSUE-092 documents recurs here for a different field.

Two consequences, both currently unverified against real dispatch (packval never reaches the code
that would need to consume `Pattern`):

1. `pkg/packval/manifest.go`'s `Rule` struct needs a `Pattern` field (mirroring `manifest.go:166`)
   so the YAML `pattern:` key is captured at all instead of silently discarded.
2. `pkg/packval/phase3.go`'s dispatch-eligibility guards (`rule.File != ""` at `phase3.go:31`,
   `62`, `76`) need to also recognize a populated `Pattern` as dispatchable, and the fixture-run
   call itself needs a path that invokes the engine with the inline pattern string rather than a
   rule-file path — `executor.RunEngine(packDir, binding, []string{rule.File, f.Path})` assumes a
   file-based `input_mode`; a `pattern-arg` engine's invocation shape has not been traced here and
   is scoped to whoever plans the fix.

## Direction (to be specified)

Not scoped here — for the eventual plan to weigh, at minimum:

1. Add `Pattern string `yaml:"pattern"`` to `pkg/packval/manifest.go`'s `Rule` struct.
2. Teach `phase3.go`'s dispatch-eligibility checks to treat a populated `Pattern` as sufficient to
   attempt fixture execution, independent of `File`/`RulePath`.
3. Trace how a `pattern-arg` engine binding actually wants its arguments shaped (per
   `input_mode`/`input_flag` in `packs/contracts/pack.yml` and `pkg/pack/engine`) and route
   `pattern-arg` rules through that shape rather than reusing the `[]string{rule.File, f.Path}`
   call built for file-based engines.
4. Add a regression fixture proving phase3 CAN fail for a `pattern-arg` rule: a
   `packs/contracts`-style rule with a negative fixture rewritten to be compliant must turn
   `pack test` red once dispatch is live. Absent that proof this remains an unverified fix, per
   the fixtures-from-real-output/must-falsify convention.
5. Consider whether this and ISSUE-092 should ultimately land as one unification of the two
   independently-maintained `Rule` structs (`pkg/pack/manifest.go` vs `pkg/packval/manifest.go`)
   rather than two field-by-field patches — left to whoever plans either fix, since a shared model
   would prevent this class of drift recurring for a third field later.

## Notes / references

- Sibling to ISSUE-092 (`Pack Test Phase3 Fixtures Cannot Fail`): same root shape — packval's
  `Rule` model missing a field the gate-runtime `Rule` model (`pkg/pack/manifest.go`) has,
  silently killing phase3 fixture dispatch for one rule-declaration style. ISSUE-092 covers
  `rule_path:`-declared (file/rule-path-based) rules; this issue covers `pattern-arg`-declared
  (inline-pattern-based) rules. Filed separately per the existence-in-world check: structurally
  distinct fix (add a field + a new dispatch/invocation shape) versus ISSUE-092's fix (reconcile
  two models' file-reference key naming) — not a duplicate.
- Found by a planner working PLAN-ISSUE-092, credible lead reported unverified; independently
  reproduced above (`grep` counts against `packs/contracts/pack.yml`, and the `phase3.go`/
  `manifest.go` code trace) before filing.
- Governed by the same BUNDLE-005 (Pack Validation) charter ISSUE-092 cites; this is a bug against
  already-delivered `pack test`/`pack check` behavior for the `packs/contracts` pack specifically,
  not unclaimed charter work.
