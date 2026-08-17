---
name: fixture-joins-the-corpus-it-measures
description: When you build a whole-tree scanner, your own test file's literal fixtures become real instances in the corpus — assemble the marker at runtime or the repo-invariant test is permanently red
metadata:
  type: project
---

A test fixture containing the literal string a WHOLE-TREE scanner looks for is not a
fixture — it is a real instance, because the scanner walks the test file too.

Measured on ISSUE-097 (2026-08-17). The lane built `HarvestTokens`, which byte-scans
every file in the repo for the literal `@waiver:` marker, plus a repo-invariant test
asserting zero unbound tokens. The plan's pre-measured census was FIVE. The first census
run reported **SIX** — the extra was `pkg/waiver/unbound_test.go:161`, my own fixture
line `"// two on one line: @waiver:a/b/first:... @waiver:a/b/second:..."`. Three
`/`-segments, well-formed, keyed to a pack no lock records: a genuine unbound token that
would have kept the invariant red forever.

**Why:** the scanner has no notion of "this file is a test". Fixtures live in real files
at real paths, and a whole-tree walk is exactly the thing that cannot tell them apart.
Excluding `*_test.go` would be worse — the five real tokens this lane fixed include three
in `tests/smoke/smoke_test.go`.

**How to apply:** assemble the marker at runtime so the SOURCE carries no complete
literal — `func mark(body string) string { return "@" + "waiver:" + body }` — and build
fixture lines as `"// " + mark("a/b/first:accepted-risk:2999-01-01")`. The runtime value
still contains the marker (so the parser under test sees it); the file on disk does not.
Do this from the first fixture, not after the census surprises you.

**Two corroborating details worth keeping:**

- **Only ONE of the two tokens on that line was flagged.** The second (`a/b/second`) sat
  before the Go string literal's closing `",`, which trapped `"` into the expiry field and
  made it MALFORMED, so `HarvestTokens` dropped it. Whether a literal fixture escapes the
  corpus is an accident of where the closing quote falls — never a property you can
  eyeball. Same mechanism as [[project_hermetic_pack_fixture_recipe]]'s parse-vs-read gap.
- **Interpolated fixtures escaped for a different accidental reason.**
  `"// @waiver:" + staleRuleID + ":false-positive:..."` parses to a rule-id of literally
  `" + staleRuleID + "`, which has no `/` and is therefore dropped by the extractable-name
  filter. It escaped the census by luck, not design. `mark()` makes it deliberate.

Related: [[project_synthesized_fixture_hides_path_base]] — the sibling failure where a
fixture is too synthetic to falsify anything. This is the opposite pole: a fixture that is
not synthetic ENOUGH and contaminates the thing it measures.
