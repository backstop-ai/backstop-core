---
name: completeness-claimed-comment-set
description: A plan asserting "the comment corrections are a set of N, complete this time" is still likely incomplete — sweep yourself with a MULTI-LINE window, since stale sentences wrap
metadata:
  type: project
---

When a plan retracts a documented equivalence (e.g. "`gate --all` == a whole-repo
directory scan") it must correct every comment in the tree asserting it. Plans
that have already corrected their own list twice will still assert the third list
is complete — and still be wrong. Do the sweep yourself, and do it over a
TWO-LINE window: the stale sentence often wraps ("...fall back to scanning
projectRoot. The full-repo scan is" / "reachable only via --all."), so a
single-line grep silently misses it.

Also sweep files in NO task's file scope. On PLAN-ISSUE-091 the two misses were
`cmd/backstop/pack_gate_scope_test.go` (in a task's scope but the sentence
unnamed) and `cmd/backstop/ci_recipes_harness_test.go` (in no task's scope at
all) — the latter independently documented the plan's own "THIRD DIVERGENCE"
mechanism and would have been inverted by the fix.

TWO SWEEP AXES, and plans conflate them. Axis 1 is the RETRACTED EQUIVALENCE
(comments asserting the thing the fix disproves). Axis 2 is the DEFECT-ASSERTED-
AS-LIVE (comments/strings/CI config citing the ISSUE ID as a live hazard, which
the fix resolves). A plan sweeping only axis 1 will miss axis 2 entirely. Run
`grep -rn <ISSUE-ID>` EXCLUDING plans/ specs/ issues/ directives/ — plans run
their existence-in-world grep over artifact dirs ONLY, so source-level citations
are structurally invisible to it. On PLAN-ISSUE-091 round 5 this surfaced
`.github/workflows/ci.yml` (the shipped blocking gate step's comment), a
`workflows_test.go` doc comment, AND a live map-value STRING (not a comment) —
none in any task's scope, and all mandated by a COMPLETED plan (PLAN-ISSUE-020
instructed "cite both in the test comment so the constraint survives its
authors"), making the stale rationale a CLM-009-class consequence to file, not
just text to fix.

AXIS 3, found in round 6: the ENCLOSING COMMENT BLOCK of a site the plan DOES
correct. 091's correction 8 rewrote the ISSUE-091 paragraph of a doc comment and
listed the surrounding sentences to "keep EXACTLY as they are" — one of which
("Diff scope with an explicit base is the only shape that is both complete and
stable") went false from the same fix, two lines below. A preserve-verbatim list
is an ASSERTION that each preserved sentence survives the fix; check each one,
because it is the least-swept place in the whole sweep.

The same round found a THIRD completed-plan claim retraction the plan had not
filed (PLAN-ISSUE-040 CLM-005: "the nil-scope / GateScopeModeAll whole-repo
escape hatch is unchanged … NOT testdata-filtered"). When a plan files two
"completed plan's delivered claim retracted" consequences, grep the completed
plans that OWN the changed code path for their own claim text — there is often a
third.

Also re-tally the plan's own stated total against its own enumeration — 091 said
"SIX in this task, SEVEN across the plan" while enumerating 6 + 2 = 8.

**Why:** the plan's own stated standard is "or the code will contradict itself
and a future reader will trust the stale sentence." That standard applies to the
sites the plan missed exactly as much as to the ones it found.

**How to apply:** for any plan whose tasks include "correct the comment that
says X", grep the WHOLE repo for X's concept pair (not X's exact words) across a
multi-line window, then diff your hit set against the union of every task's
declared corrections. Related: [[verified_enumeration_do_not_rederive]],
[[deleted_symbol_named_in_kept_comment]].
