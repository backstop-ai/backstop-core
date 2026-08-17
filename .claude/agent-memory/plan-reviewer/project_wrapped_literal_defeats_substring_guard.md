---
name: wrapped-literal-defeats-substring-guard
description: A drift guard forbidding a multi-word substring goes vacuous when the offending phrase is LINE-WRAPPED in a Go format string / comment; scaffold the artifact and run the real substring search
metadata:
  type: project
---

When a plan adds a "no shipped byte still says X" drift guard over a multi-word
phrase, GENERATE the artifact and run the exact search the plan prescribes. A
phrase that reads as one string in the Go source is frequently split across a
raw-string line break plus a comment continuation (`... currently always\n      #
passes ...`), so `strings.Contains(b, "always pass")` is FALSE on the emitted
bytes even though the emitted text plainly says it.

**Why:** PLAN-ISSUE-146 (2026-08-17) forbade `always pass` (case-insensitive) in
the written validator, the written `pack.yml`, and `packTypeBlurb`. Measured on a
freshly scaffolded pack: validator TRUE, pack.yml FALSE, blurbs FALSE. Only one of
three arms was red-first, and the plan's implementation task asserted the guard
meant "a partial cleanup will fail the build" — which was false. The easy half of
the fix would have shipped while the emitted pack.yml still advertised the defect.

**How to apply:** For any `must not contain <phrase>` guard — scaffold/render the
real output, run the search whole-file (not `grep`, which is line-based, but the
result matches here), and report which arms are actually red at HEAD. A guard arm
that is green at HEAD is a pure drift guard, not a proof, and any task text
claiming it enforces the fix is a blocker. Demand whitespace/comment-marker
normalization before matching. Related: [[inert_decoy_fixtures_vacuous]],
[[completeness_claimed_comment_set]].
