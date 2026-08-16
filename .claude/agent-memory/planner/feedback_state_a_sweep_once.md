---
name: state-a-sweep-once
description: A grep/sweep command repeated across plan tasks drifts into mismatched variants; state it once byte-identically and pair it with a FINDING test
metadata:
  type: feedback
---

When a plan tells the implementer to run the same sweep in several tasks, state
the command ONCE as canonical and reproduce it BYTE FOR BYTE everywhere else.
Verify with `grep -o "<the command>" <plan> | sort | uniq -c` — the count should
show exactly one unique string.

**Why:** PLAN-ISSUE-122 carried "the signature sweep" in four places with three
different regex alternations (4 alts / 3 alts dropping `Classification` / just
`Classification`). Successive rounds each concluded the sweeps had a coverage
gap and invented a "third mechanism axis" to close it — but the canonical 4-alt
sweep already returned every supposedly-missed hit. The real failure was READING
a long result set carelessly, and the invented axis added a fourth variant, which
is the same drift the sweeps existed to catch.

**How to apply:** Two lessons, both cheap.
1. Before adding a new sweep axis "because the existing ones miss X," RUN the
   existing sweep and read its full output. A missed hit is usually a reading
   failure, not a coverage gap — say so plainly in the plan rather than papering
   over it with another command.
2. A corpus-wide sweep returns dozens of hits, most irrelevant (test names,
   section headings, `consumes:` entries, adjectives). "Any unlisted hit is a
   FINDING" is unusable at that volume. Pair the sweep with an explicit FINDING
   test — what shape of hit actually counts as drift — and a list of near-miss
   hits checked and cleared, so a genuinely new hit reads as new.

Related: [[project_defect_pinned_by_shipped_tests]] (spec prose going stale is
what these sweeps hunt), [[feedback_cite_by_name_in_contended_files]].
