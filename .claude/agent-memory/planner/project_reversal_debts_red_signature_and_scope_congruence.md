---
name: reversal-debts-red-signature-and-scope-congruence
description: After reversing a plan decision, re-derive the RED signature per-assertion (a superset signature is satisfied by the buggy implementation) and retract the source issue's text that disclaims the new scope
metadata:
  type: project
---

Reversing a decision mid-plan (narrow→generalize, defer→deliver) leaves downstream debts that
validators do not catch. Two recur, both found in PLAN-ISSUE-180 review round 2:

★ **1. RE-DERIVE THE PRE-FIX RED PER ASSERTION — DO NOT SAY "these all fail."** A stated RED
signature that is a SUPERSET of the true one is worse than vague: it is usually the signature
the BUGGY implementation produces, so the implementer's wrong fix matches your words better
than the right one does, and it still goes green at GREEN-time — shipping the claim unfulfilled.

Measured instance: I wrote "STEP 2's floor AND STEP 3a both red pre-fix." Run for real, STEP 2
was **GREEN** — the derived set is built independently of `TestMain`, so the target package was
already a member. Only STEP 3a red. And the both-fail signature is exactly what you get if the
implementer keeps the blind-spot `if pkg.testMain == nil { continue }` skip and merely widens
the hardcoded floor. Remedy: name the ONE assertion that reds, and state the other's pre-fix
GREEN **as a tell** — "if the floor also reds, the skip was left in; stop and fix it."

★ **2. THE SOURCE ISSUE'S OWN TEXT MAY DISCLAIM THE WORK YOU JUST ADDED.** An issue often
carries an explicit out-of-scope section ("that's ISSUE-NNN's territory, folding it in would
scope-creep this issue"). When the plan reverses and delivers exactly that, and the issue closes
via `delivered_by: PLAN-ISSUE-NNN`, the issue ships a body contradicting its own plan's claim —
a scope-fence congruence failure no schema check sees. Route a retraction, and make it **IN
PLACE and PARTIAL**: the section's factual *description* is usually still accurate and stays;
only the *disposition* is retracted, dated and attributed to a founder-visible decision rather
than rewritten as if the issue always said so. (See [[project_issue_self_retraction]].)

★ **3. A CORRECTION ROUTED BY SECTION NAME FIXES ONE SITE.** A false premise propagates — mine
appeared at ~8 sites across two issues, including the *methodology* line that produced it. Hand
the routed author an explicit line-by-line site list with quoted text (line numbers move), and
mark which corrections are FALSE vs merely STALE.

**How to apply:** after any decision reversal in a plan, re-walk (a) every stated pre-fix/post-fix
verdict, (b) the source artifact's scope fences, (c) every occurrence of any premise that changed.
Related: [[project_run_the_generalized_predicate_never_grep_imports]],
[[project_phase3_loop_polarity_stated_backwards]].
