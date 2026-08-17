---
title: "Issue 148 Stated Fix Menu Falsified"
schema_version: issue/v1

issue:
  id: ISSUE-159
  title: "Issue 148 Stated Fix Menu Falsified"
  type: technical-debt
  status: open
  created: "2026-08-17"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-148 And DIR-032 Item 19 State A Falsified Fix Menu

## Problem

**This is a correction to artifact prose, not a broken promise.** Neither `ISSUE-148` nor
`DIR-032` item 19 declares a mandated test that this issue contradicts — nothing here is
red that was claimed green. It is filed as `PLAN-ISSUE-148` TASK-005 item 2, a mandated
follow-on surfaced while implementing that plan's own fix.

Both artifacts assert the `packs/substantiveness` fixture-polarity defect is purely a
manifest-key filing question, and that either of two stated remedies resolves it. Measured
2026-08-17 at HEAD `c586af3` with real ast-grep 0.43.0: **neither stated remedy actually
works**, because of a second, real property of the pack that neither artifact describes.

### The falsified sentences, quoted verbatim

From `issues/ISSUE-148-substantiveness-fixture-polarity-inverted.issue.md`:

> The two fixture files' CONTENTS are each internally correct examples of what they claim
> to be — the defect is purely which manifest key (`positive`/`negative`) each is filed
> under.

and, in its "Two owners, not one" section, item 1:

> The in-repo copy at `packs/substantiveness/pack.yml` and its fixture files (swap the
> `positive`/`negative` fixture-path assignment under each rule's `claims[].fixtures`, or
> swap the file contents to match their declared slot — either resolves the inversion).

From `directives/DIR-032-gate-verdict-honesty.directive.md` (the item-19 roster note,
~line 128, and again in the item-19 slot body, ~line 1623):

> the defect is purely which manifest key (`positive`/`negative`) each is filed under,
> inverted from backstop's positive=clean/negative=violating convention.

> item 19 is a real, in-repo, currently-installed pair where each fixture is individually
> a correct example of what it claims to be — the defect is purely which declared slot
> (`positive`/`negative`) it is filed under.

### Why it is false: both rules share one sgconfig, and the verdict is per-engine, not per-rule

`pkg/packval/phase3.go` dispatches `rule.RuleSourcePath()`, which for BOTH rules declared
in `packs/substantiveness/pack.yml` resolves to the same file, `ast-grep/sgconfig.yml` —
the project config whose `ruleDirs: [rules]` loads BOTH `hollow-test-go` AND
`referenced-symbol-go` together. So packval's per-fixture verdict answers "did ANY rule in
the combined config fire", not "did THIS rule fire". A declared POSITIVE (clean) fixture
must therefore trigger NEITHER rule, and a declared NEGATIVE fixture need only trigger ONE
rule — any one, not necessarily its own.

**Consequence A — the key swap alone changes nothing.** Measured: swapping the
`positive:`/`negative:` fixture paths in `pack.yml` with the fixture bodies left untouched
still yields the SAME two errors, byte-for-byte unchanged:

```
ERROR [phase3-fixtures/semgrep-positive] positive fixture triggered the rule (false positive)
```

(x2, one per rule.)

**Consequence B — the content-swap alternative inherits the same trap in the other
direction.** A substantive Go test asserts with `t.Fatalf`; `t.Fatalf` is a
`selector_expression` whose operand is the identifier `t`; and `referenced-symbol-go`
matches exactly that shape inside any `^Test`-named function. So a swapped-in "clean"
fixture that still calls `t.Fatalf` to assert STILL fires, and phase3 still fails.

**Consequence C — the part neither artifact describes at all.** Both currently-declared
NEGATIVE fixtures were passing FOR THE WRONG REASON, in two different ways. Measured
per-fixture against the combined ruleset, pre-fix:

```
hollow-test-go/positive.go       -> fires hollow-test-go (FN=TestHollowExample)          FALSE POSITIVE
hollow-test-go/negative.go       -> fires referenced-symbol-go x3 (PKG=t, PKG=os, PKG=m);
                                     its OWN rule is SILENT                                accidental pass
referenced-symbol-go/positive.go -> fires referenced-symbol-go x2 (PKG=strings, PKG=t)     FALSE POSITIVE
referenced-symbol-go/negative.go -> fires its own rule exactly ONCE, on the t.Fatalf
                                     receiver (PKG=t) — boilerplate, not the package-
                                     qualified call it was written to demonstrate           accidental pass
```

That is why the run reports exactly 2 errors rather than 4 — the two NEGATIVE-slot
fixtures already "pass" phase3, but not for the reason their own file comments describe.
The NEGATIVE half of this pack was **vacuously green all along**, underneath the polarity
inversion the issue names. That makes this its own DIR-032-family vacuous-green finding,
sitting inside the very artifact filed to fix a DIR-032 vacuous-green defect — worth
stating plainly rather than letting the irony pass unremarked.

### What the actual fix was (already delivered — this issue is prose-only, not open work)

`PLAN-ISSUE-148` re-authored all four fixture BODIES so each is clean or violating AGAINST
THE COMBINED RULESET, with `pack.yml` not edited at all — no key swap, no version bump on
the in-repo copy. Two mechanisms are load-bearing:

1. A clean fixture's assertion must be an UNQUALIFIED call whose identifier matches the
   hollow rule's assertion vocabulary (e.g. `mustEqual`), never a `t.X(...)` selector.
2. Helper functions may hold `t.Fatalf` freely, because `referenced-symbol-go` requires
   `inside: {matches: is-test, stopBy: end}` and a non-`^Test`-named `function_declaration`
   is not `is-test`.

Measured: the corrected pack passes all six `pack test` phases, exit 0. The corrective
explanation already exists in two places and needs transcribing into the two source
artifacts, not rediscovering: `PLAN-ISSUE-148`'s own design notes, and the header comment
of `cmd/backstop/substantiveness_fixture_polarity_test.go` (added by that plan's Phase 1).

## Solution

Route the prose correction through each artifact's own proper authoring agent — do not
hand-edit either target:

- **`ISSUE-148`**: issue-author replaces the "purely which manifest key" framing (both
  quoted spots) with the combined-ruleset explanation, and adds the vacuous-green
  negative-half finding (Consequence C above) as a Note.
- **`DIR-032` item 19**: directive-author applies the same correction to the item-19 roster
  note and slot body (both quoted spots, ~line 128 and ~line 1623).

Both corrections are additive/clarifying — they do not change either artifact's status,
requirements, or claims, since the underlying defect both artifacts named was real and the
underlying fix both artifacts pointed at (eventually) landed; only the stated MECHANISM of
the fix was wrong.

## Notes

- ★ Process fence for this issue itself: `ISSUE-159` records the correction only. It must
  not, and does not, hand-edit `ISSUE-148` or
  `directives/DIR-032-gate-verdict-honesty.directive.md` — those edits belong to their own
  owning agents, per the routing above.
- Filed as `PLAN-ISSUE-148` TASK-005 item 2, a mandated follow-on surfaced during that
  plan's own implementation, not an independently discovered defect.
- Existence-in-world check performed 2026-08-17 before filing: searched `issues/` for
  `polarity`, `fixture`, `manifest key`, `combined ruleset`, and `substantiveness`, and
  `bundles/` for the same. `ISSUE-157` ("Go Contracts Mirror Inverted Fixture Polarity") is
  the same defect SHAPE (per-fixture verdict vs. per-rule intent) but a DIFFERENT pack
  (`go-contracts`, not `go-substantiveness`/`packs/substantiveness`) and a different
  artifact (it corrects nothing already stated — it files a fresh mirror-drift defect) — a
  related sibling, not a duplicate. No open issue or bundle charter already owns "correct
  the ISSUE-148/DIR-032 item 19 stated fix-menu prose."
