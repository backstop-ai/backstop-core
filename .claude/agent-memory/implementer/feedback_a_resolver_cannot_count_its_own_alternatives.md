---
name: a-resolver-cannot-count-its-own-alternatives
description: "A loop that re-calls a FIRST-MATCH resolver once per candidate can only ever count 0 or 1 — it structurally cannot detect the overlap it claims to test. Assert against the TABLE the resolver reads, not the resolver."
metadata:
  type: feedback
---

Shipped this exact defect in SPEC-068 and an impl-reviewer caught it: an "exclusivity, asserted
structurally" loop ranged over all seven kinds and, for each, re-called the classifier on the SAME
filename, incrementing a counter when the answer equalled that kind. The classifier returns ONE
value, so the counter could only ever be 0 or 1 — the test passed no matter how broken the table
was, and table-level overlap was entirely untested.

**Why:** the resolver returns the FIRST row whose pattern matches. A genuine overlap between two
rows is precisely what the resolver HIDES; asking it repeatedly cannot reveal a second answer it is
designed never to give.

**How to apply:** when the property is "at most one X may claim Y", assert it against the DATA the
resolver consults, not the resolver's answer. In an internal test (same package) iterate the table
and apply each row's own predicate. Then add the whole-class version, which sampling can never
reach: for a suffix-matched table, no row's suffix may END WITH another row's suffix (a bare
markdown suffix shadows every longer one), and none may be empty. A new row with a too-general
pattern passes every sampled case while silently capturing another kind's files.

**Always falsify a rewritten assertion.** Inject a real overlap into the production table (a bogus
row with an over-general suffix), watch BOTH the sampled and whole-class checks go red naming the
colliding pair, then revert. If the mutation does not turn it red, the rewrite is as vacuous as
what it replaced. Do the mutation with Edit on the .go file — a scripted heredoc containing
artifact-shaped literals trips the agent-guard.

Same review round, different symptom: [[write-path-and-read-path-must-share-one-root]].
