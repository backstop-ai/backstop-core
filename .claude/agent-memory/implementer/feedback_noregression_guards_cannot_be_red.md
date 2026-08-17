---
name: noregression-guards-cannot-be-red
description: A plan demanding "ALL N mandated tests must be observed RED first" is wrong for no-regression and no-over-fire guards; report the split, never manufacture a red
metadata:
  type: feedback
---

When a plan says **"ALL of these — with no exceptions — MUST be observed RED"**, re-derive
which of its mandated tests are *capable* of being red. Two classes structurally cannot be,
and forcing them red damages the test:

- **No-regression guards** ("shape X still yields exactly 3"). The claim they encode is
  *unchanged byte-for-byte*, so passing on BOTH sides IS their meaning. A red here would
  mean the baseline moved — the opposite of what is wanted.
- **No-over-fire discriminators** ("hostile input manufactures NO finding"). Before the
  widening lands there is nothing to over-fire, so they pass trivially. They only become
  load-bearing AFTER the implementation task.

**Why:** PLAN-ISSUE-067 mandated 12 red-first tests; 10 went red (reproducing the defect
verbatim), 2 could not. The plan's own escape hatch — "if any passes now, the fixture was
already changed or the test is not on the real path — STOP" — did not apply: both drove the
same real dispatch and real converter that took the other 10 red. Manufacturing reds for
them would have inverted a baseline assertion.

**How to apply:** Write them faithfully, observe the real split, and **report it as a
deviation** with the reasoning. Do NOT silently reinterpret, and do NOT weaken the guard to
make it fail. Separately, check whether a mandated test would be *vacuously green* — in the
same plan, one test fed real captured bytes through a fixture runner that injects as STDOUT
the bytes production only ever produces on STDERR, so it had always passed. The honest fix
was to ADD the reachability half (assert the binding actually declares the producer), which
made it genuinely red pre-fix. See [[project_synthesized_fixture_hides_path_base]] and
[[feedback_choose_compile_red_or_behavioral_red]].

**A third class, and the technique that redeems it: the "guard against a BROKEN FIX".** When
the fix is *derivation of data that today is empty*, every guard asserting "this case yields
NOTHING" is trivially true pre-fix — emptiness is the pre-fix state everywhere. PLAN-ISSUE-114
predicted 6 reds / 4 guards and got 4 / 6 for exactly this reason. Don't stop, and don't fake a
red: **earn the falsification by MUTATION after green.** Delete the guard clause the test
exists to protect (there, the `Class == ClassRetiredTerminal` skip), run only those tests,
confirm they go red with the *predicted* message, restore, and cite the mutation in the report.
That converts "passes on both sides" into proof the guard constrains the shape of the fix —
which is what the plan actually wanted. Cheap: `cp` the green file to /tmp, `perl -0pi -e` the
clause out, test, `cp` back.

Also watch for a red that fires for the WRONG REASON: a plan's `must-not-run` sentinel
command was itself unstartable, so a never-started test passed pre-fix on the COMMAND's
failure rather than the new branch. Point the sentinel at something that genuinely works so
the test can only go red on the surface under test.
