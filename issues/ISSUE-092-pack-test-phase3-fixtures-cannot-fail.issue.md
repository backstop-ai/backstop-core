---
title: "Pack Test Phase3 Fixtures Cannot Fail"
schema_version: issue/v1

issue:
  id: ISSUE-092
  title: "Pack Test Phase3 Fixtures Cannot Fail"
  type: bug
  status: open
  created: "2026-07-27"

complexity:
  scope: contained
  uncertainty: known
  risk: critical
---

# Pack Test Phase3 Fixtures Cannot Fail

## Problem

`backstop pack test`'s phase3-fixtures step exists to prove a pack's rules actually FIRE — every
negative fixture (the violating example) must trigger its rule, every positive fixture (the clean
example) must not. That is the fixtures-from-real-output/must-falsify convention (recorded founder
law): a rule with no falsifying fixture pair is not proven to detect anything. Phase3 is the
mechanism that is supposed to enforce that convention on every pack, every time.

It doesn't. Measured by implementer-087 (2026-07-28, during the go-standards rule fix): rewriting a
rule's negative (violating) fixture to be fully compliant — i.e. deleting the violation it exists to
catch — still yields `phase3-fixtures: pass`, exit 0, on `backstop pack test`. Removing a rule's
`paths: exclude` entry also fails nothing. A pack whose rule never fires on anything, ever, passes
`pack test` cleanly. The phase that exists to prove rules work cannot detect a rule that is dead.

Confirmed independently while authoring this issue:
`./bin/backstop pack test .backstop/packs/backstop/go-standards` returns `phase3-fixtures: pass`
today, for a pack whose rules are declared entirely via `rule_path:` (see Root cause below) — i.e.
phase3's fixture-execution path is not merely hard to trigger, it does not run at all for this pack.

Severity is high, not cosmetic: `pack test` is the pack-quality gate for the entire ecosystem. A
pack author (or an agent editing a pack under a rule fix, as implementer-087 was) can ship a rule
that never fires, and the tool whose entire job is to catch that says green.

## Root cause

Traced to `pkg/packval/manifest.go` and `pkg/packval/phase3.go`. `packval`'s `Rule` struct
(`manifest.go:75-88`) reads a rule's source file from a YAML key `file`:

```go
type Rule struct {
    ID     string `json:"id" yaml:"id"`
    Engine string `json:"engine,omitempty" yaml:"engine,omitempty"`
    File   string `json:"file,omitempty" yaml:"file,omitempty"`
    ...
}
```

But no real pack.yml in this repo declares a rule under `file:`. Every one of them —
`packs/substantiveness/pack.yml`, `packs/base-engines/pack.yml`,
`.backstop/packs/backstop/go-standards/pack.yml` — declares the rule source under `rule_path:`
instead, e.g.:

```yaml
- id: go-core-no-global-mutable-state
  rule_path: rules/core/go-core.yml
  engine: semgrep
  claims:
    - id: clm-001
      fixtures:
        positive: [fixtures/rules/valid/go-003-dependency-injection.go]
        negative: [fixtures/rules/invalid/go-003-global-mutable-state.go]
```

`rule_path` is not a stray convention — it is the field the GATE-side manifest parser
(`pkg/pack/manifest.go:144`, `RulePath string `yaml:"rule_path"``) actually consumes at consumption
time. `pkg/packval` (the authoring-time validator behind `pack check`/`pack test`) and `pkg/pack`
(the runtime gate parser) are two independent manifest models, and they have drifted: the runtime
side speaks `rule_path`, the validation side still speaks `file`.

Consequence, traced through `pkg/packval/phase3.go:28-113` (`RunFixtures`, the ruleset-rule loop):
because `rule.File` unmarshals to `""` for every real pack, the guard at `phase3.go:31`
(`if rule.File != ""`) is false, so:

- The semgrep-rule-id cross-check (`phase3.go:31-48`, verifying the rule ID inside the rule file
  matches the pack's declared rule ID) never runs.
- `haveBinding` is never set (`phase3.go:52-58` is skipped).
- The per-claim fixture loop's guard `if rule.File != "" && haveBinding` (`phase3.go:62`, `76`) is
  false for both positive and negative fixtures, so `executor.RunEngine` — the call that actually
  invokes the engine and parses SARIF findings — is **never invoked** for any layer 1-2 semgrep rule
  declared via `rule_path:`. Not "rarely fails" — the dispatch path is dead code for every pack in
  this repo's ecosystem today.

`res.Errors` therefore stays empty, `res.Status` stays `"pass"` (`phase3.go:217-219`), and phase3
reports success having executed zero fixture checks. This fully explains both symptoms
implementer-087 reported: a rewritten negative fixture is never run against the engine at all, and
`paths: exclude` is likewise never exercised because the rule file + fixtures never reach the
engine invocation.

The `tool_config` fixture-execution path (`phase3.go:116-143`, keyed on `tc.File` / yaml `file`) and
the layer-3 validator path (`rule.Validator`, independent of `rule.File`) are separate code paths
not covered by this root cause and were not verified either way — scoped to whoever plans the fix.

## Measurement traps for the fix

Two additional traps surfaced by implementer-087 while working around this defect (invoking semgrep
directly on explicit file targets instead of trusting `pack test`), worth recording so the fix
doesn't reintroduce either as a false-green:

- **Directory-scan vs explicit-file-list divergence.** Semgrep's default ignore rules skip
  `*_test.go` files on a directory-target scan; a probe that hands semgrep a directory reports zero
  test-file findings, while the same rule dispatched with explicit file targets (the gate's own
  diff-scope arg shape) reports dozens. This is the same failure family as ISSUE-091 (`gate --all`
  underreporting test-file findings via directory-target dispatch) — whichever fix restores real
  fixture execution here should invoke the engine the same way the gate does (explicit file
  targets), not via a directory scan, or it will inherit ISSUE-091's undercount.
- **Silent schema-break false-pass.** A `sed`-blanked line in a rule's YAML rule list (e.g. an
  accidentally emptied list item) produces an `InvalidRuleSchemaError` from the engine, which
  currently surfaces as zero results — read at a glance as a passing falsification (fixture "didn't
  trigger" and "engine errored" are not currently distinguished in a way that's obviously different
  from "ran clean"). The fix should make an engine/schema error loud and distinct from a genuine
  zero-finding result, not foldable into either the positive-pass or negative-pass path.

## Direction (to be specified)

Not scoped here — for the eventual plan to weigh, at minimum:

1. Reconcile `pkg/packval`'s `Rule.File`/`rule.file` with `pkg/pack`'s `Rule.RulePath`/`rule_path` —
   either packval should read `rule_path` (matching the runtime parser and every real pack), or the
   two manifest models should be unified so this class of drift can't recur silently.
2. Once the dispatch path is live again, decide how phase3 should react to an engine/schema error
   (see measurement trap above) versus a genuine non-firing fixture — they must not collapse to the
   same "pass."
3. Add a regression fixture that proves phase3 CAN fail: a pack with a `rule_path:`-declared rule
   and a negative fixture rewritten to be compliant (mirroring implementer-087's repro) must turn
   `pack test` red. Absent that proof, any fix here risks becoming another vacuous-green claim.
4. Confirm whether the `tool_config` and layer-3 validator paths are similarly affected or were
   already sound — they were not verified as part of this issue.

## Notes / references

- Reported by team-lead via implementer-087's evidence from the go-standards rule fix session
  (2026-07-28).
- Governed by BUNDLE-005 (Pack Validation, `ready`), whose own REQ-002 and Phase 3 design ("100%
  pass rate... every negative fixture MUST trigger the rule") is the requirement this defect
  violates in the shipped implementation — this is a bug against already-delivered behavior, not
  unclaimed charter work.
- Sibling to the gate-verdict-honesty cluster (ISSUE-066, ISSUE-067, ISSUE-091): a validation signal
  that reads as complete/authoritative and silently isn't. Different defect from ISSUE-091 (that one
  is the `backstop gate --all` runtime path underreporting test-file findings via directory-scan
  dispatch; this one is `backstop pack test`'s authoring-time fixture-execution path never running
  at all for `rule_path`-declared rules) — filed separately per the existence-in-world check, not as
  a duplicate.
- Fixtures-from-real-output/must-falsify convention referenced above: recorded founder law, see
  agent memory `feedback_fixtures_from_real_output`.
