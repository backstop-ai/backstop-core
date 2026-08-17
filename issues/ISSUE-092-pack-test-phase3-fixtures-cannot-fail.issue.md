---
title: "Pack Test Phase3 Fixtures Cannot Fail"
schema_version: issue/v1

issue:
  id: ISSUE-092
  title: "Pack Test Phase3 Fixtures Cannot Fail"
  type: bug
  status: closed
  created: "2026-07-27"
  closed: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: critical

delivered_by: PLAN-ISSUE-092
---

# Pack Test Phase3 Fixtures Cannot Fail

## Resolution

Delivered by PLAN-ISSUE-092
(`plans/PLAN-ISSUE-092-pack-test-phase3-fixtures-cannot-fail.plan.yml`).

`pkg/packval`'s pack-fixture-execution pipeline (`pack test`/`pack check`) had multiple
compounding defects that together let every phase3 fixture pass regardless of its actual
polarity:

- **CLM-001** — the manifest-model drift diagnosed in Root cause is reconciled: packval's `Rule`
  now reads the canonical `rule_path` key instead of the dead `file` key, matching the
  gate-runtime parser (`pkg/pack/manifest.go`) every real pack.yml was already written for.
- **CLM-002/CLM-003** — phase 3 genuinely dispatches for a `rule_path:`-declared rule, and CAN
  FAIL: a pack whose negative fixture is rewritten to be compliant now turns `pack test` red
  instead of the false `pass` this issue reported.
- **CLM-004** — fixture polarity is corrected on both findings paths, resolving the
  positive/negative vs. valid/invalid inversion flagged as a possible secondary defect above
  (BUNDLE-005 REQ-011).
- **CLM-005** — an engine/schema error is now a distinct, loud phase-3 failure, no longer
  collapsible into either a genuine pass or a genuine fail (the "measurement trap" recorded
  above).
- **CLM-006** — phase 1's rule-file-existence structural check is live for `rule_path:`-declared
  rules.
- **CLM-007** — layer-3 validator dispatch no longer gates on the retired `Layer` field, which had
  made freshly-scaffolded packs' sample validators structurally unreachable.
- **CLM-008** — proven end-to-end through the real CLI, not just unit-level: a pack with a
  genuinely broken negative fixture now correctly fails `pack test` where it previously passed
  vacuously in 17ms.
- **CLM-009** — a drift guard fails the build if the authoring-time and gate-runtime manifest
  models diverge again, closing off silent recurrence of this exact defect class.
- **CLM-010** — the semgrep-rule-id cross-check is conditioned on the resolved binding's declared
  `input_mode`, not on the engine's name, so it does not break for non-semgrep engines.

**Filed, not fixed, as follow-ons — this fix's own real execution exposed three genuinely
pre-existing defects** that were previously invisible behind the dead dispatch paths this fix
corrects. Each requires touching files outside this plan's declared scope, so each was filed
separately rather than folded in:

- ISSUE-146 — `pack new`'s vacuous scaffold validator (this fix's own direct true positive).
- ISSUE-147 — darwin sandbox relative-packDir bug.
- ISSUE-148 — the substantiveness pack's own fixture polarity being backwards.

The final `gate --all` reading was red at close, but every violation was independently attributed
to either other lanes' concurrent in-flight work or these three separately-filed, pre-existing
defects — zero violations cite any file this plan's implementation touched.

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

## Additional evidence

- **Flip-the-switch reproduction (2026-08-11), reviewer, during specs/SPEC-067-ci-recipe-pack.spec.md
  review.** Two-step hands-on demonstration against a copy of `backstop-self-pack`, confirming the
  root cause diagnosed above end-to-end rather than just at the `rule.File` code-read level:
  1. Replaced every violating fixture in the copy with an inert `package x` stub (i.e. a fixture
     that violates nothing) and ran `backstop pack test`. Result: `phase3-fixtures: pass`,
     `status: pass` — a pack whose fixtures no longer demonstrate any violation still reports its
     fixture-execution phase as green. This is the false-green from the Problem section, observed
     directly rather than inferred from code.
  2. On the same copy, added `file: rules/no-baked.yml` alongside the existing `rule_path:` on one
     rule entry, then re-ran `pack test`. Execution switched on immediately: 13 real errors surfaced
     — 7× `[phase3-fixtures/semgrep-positive] positive fixture failed` and 6×
     `[phase3-fixtures/semgrep-negative] negative fixture not triggered`. This directly confirms
     `phase3.go`'s `rule.File != ""` guard is the live/dead switch for fixture dispatch described in
     Root cause: populating the field the runtime parser doesn't even read is what turns the checker
     on, and its absence (the real-world state of every pack.yml in this repo) is what turns it off.

- **Possible secondary defect — polarity mismatch, unverified.** While reproducing the above, a
  second and less certain issue surfaced: packval's `Fixtures.Positive`/`Fixtures.Negative` fields
  are read by `phase3.go` as "positive fixture MUST fire the rule, negative fixture MUST NOT fire
  the rule" — but the ecosystem's own directory-naming convention reads the opposite way in
  practice. E.g. `backstop-self-pack/pack.yml:26-29` names its non-firing fixtures'
  directory `fixtures/rules/valid/` and its firing (violating) fixtures' directory
  `fixtures/rules/invalid/` — "valid" for what should NOT trigger a rule and "invalid" for what
  SHOULD trigger it reads as the semantically inverted framing from "positive/negative" as those
  words are used elsewhere (positive = present/matches, negative = absent/no-match). This has NOT
  been traced through to confirm whether it is a real functional bug (i.e. packval's check is
  wired backwards against what pack authors intend) or purely a naming-convention mismatch with no
  behavioral consequence (i.e. the `positive:`/`negative:` YAML keys are assigned the right file
  lists regardless of which directory name a pack author chose). Flagging for someone to trace
  through packval's fixture-list assignment against real pack.yml declarations — not asserting a
  defect here.

- **Discovery context, not authority on the defect.** Found while reviewing
  `specs/SPEC-067-ci-recipe-pack.spec.md` (in-progress, not yet implemented as of 2026-08-11) — that
  spec's CLM-058 claims its rules are proven to fire correctly via `backstop pack test`, and that
  claim turned out to be unverifiable against the live tool because of exactly this defect: a
  `pack test` green tells you nothing about whether the rules it claims to validate actually ran.
  Cited here only as the occasion the reviewer rediscovered this issue's already-diagnosed defect,
  not as a second source of truth about the root cause — the flip-the-switch reproduction above is
  the evidence; SPEC-067 is only how the reviewer got there.

- **Second concrete instance — `backstop-ai/ci-workflows` pack, during SPEC-067 implementation
  (2026-08-11).** The new pack's twelve semgrep enforcement rules are proven correct via DIRECT
  semgrep invocation (`scripts/falsify.sh` in the implementation plan) and via the real
  `backstop pack test` pipeline structurally (manifest parses, schema-valid, every recipe/rule/
  engine binding well-formed) — but NOT via `pack test`'s fixture-EXECUTION phase, for the same
  root cause diagnosed above: phase 3 never dispatches because no rule in the pack declares the
  `file:` field packval's `Rule.File != ""` guard keys on (the pack, like every real pack in the
  fleet, declares `rule_path:` instead). SPEC-067's own CLM-058 was deliberately narrowed (spec
  version 1.0.2, then held at 1.0.3) to a STRUCTURAL-only claim for exactly this reason, and its
  mandated test (`TestCIRecipes_InstalledPackClearsRealPackTestStructurally`) asserts nothing about
  rule-firing behavior. The pack's fixtures ARE named to fall under their rule's include glob
  (`backstop-gate-missing-pack-install.yml` and siblings) specifically so they become correct proof
  the day this defect is fixed — recorded as forward-compatible groundwork, not present-day
  protection. Not filed as a separate issue: same root cause, no distinct mechanism — this is
  another concrete instance affected by the already-diagnosed defect above, not a new one. Cites
  `specs/SPEC-067-ci-recipe-pack.spec.md` CLM-058 and its 1.0.2 Version History entry;
  `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml` TASK-031(3) (slug
  `ci-workflows-pack-rules-unproven-under-pack-test`, which directed this annotation rather than a
  new filing). Also flags, for whoever fixes the root defect, that the fixture-POLARITY
  disagreement recorded above (positive/negative vs. valid/invalid directory naming) remains
  unresolved and should be settled before this pack's fixtures are trusted once execution is
  restored.

- **Third concrete instance — `substantiveness` pack, manifest-model drift confirmed at the struct
  level (2026-08-16).** Independent reproduction against `packs/substantiveness/pack.yml`,
  confirming the root cause holds for this pack too and pinning down the exact struct-level
  divergence between the two manifest models named in Root cause:
  - `pkg/pack/manifest.go:139-144` — the GATE-RUNTIME `Rule` struct — declares
    `RulePath string `yaml:"rule_path"`` and has **no `File` field at all**.
  - `pkg/packval/manifest.go:75-78` — the separate, authoring-time-validator `Rule` struct — declares
    `File string `json:"file,omitempty" yaml:"file,omitempty"`` and has **zero `rule_path`
    handling**.
  These are not two views of one model; they are two independently-maintained struct definitions
  that happen to describe the same YAML, and only one of them recognizes the field every real
  pack.yml actually uses. This is the same drift diagnosed in Root cause, now confirmed at the exact
  field-declaration level rather than inferred from behavior.
  - **Live flip-the-switch repro:** broke `substantiveness`'s negative fixture (edited it to no
    longer violate the rule it exists to catch) and ran `./bin/backstop pack test
    packs/substantiveness`. Result: `phase3-fixtures: pass` at `0ms` — zero execution time is itself
    telling; the phase reported success without doing any work. Restored the fixture afterward and
    confirmed `git diff` clean (no residual changes left in the tree).
  This corroborates the reviewer's 2026-08-11 flip-the-switch reproduction above on a different
  pack, and directly answers Direction item 1 in favor of unifying (or at minimum reconciling) the
  two `Rule` structs — the drift is not a naming quirk, it's two independently authored types with
  no shared source of truth for which YAML key means "where is this rule's file."
