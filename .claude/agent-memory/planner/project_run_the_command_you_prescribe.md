---
name: run-the-command-you-prescribe
description: Before shipping a plan, execute each verification command verbatim and read every existing helper the mandated tests will reuse — both hide silent-pass traps that survive review of the prose
metadata:
  type: project
---

A plan's verification commands and its reused test helpers are CODE. Author them, then run
or read them — do not ship either on inspection.

**Why:** PLAN-ISSUE-176 review round 1 (2026-08-18). Three must-fixes, all of this shape:

* A prescribed `go test . -run "TestMakefile_DeclaresOptIn" -count=1` named the ROOT package
  while the test had been scoped into `cmd/backstop`. `go test -run` with a non-matching
  pattern exits **0** printing `ok <pkg> [no tests to run]` — a verification task that checks
  nothing and reports success. Running it once would have caught it. Every scoped test command
  in a plan now carries `-v` plus "count the `=== RUN` lines; zero executed is a FAILED
  verification."
* A fence phrased "after the blocking gate step" reused `workflows_test.go`'s existing
  `stepIndex`, which returns the **first** match — and `ci.yml` invokes `backstop gate` twice
  (indices 7 and 8). The fence anchored on 7, so a forging step inserted between them sat at 8
  and passed. Fix: make **uniqueness** the primary fence (exactly one step contains
  `baseline pull`), not ordering; resolve any ordering anchor by last match with the match
  count asserted.
* A sweep predicate ("the file contains no bare `os.ReadFile`") was unsatisfiable, because the
  fix's own helper lives in that file and calls `os.ReadFile`, plus an unrelated pre-existing
  call. An unsatisfiable predicate gets weakened at GREEN time by whoever hits it. State the
  exact discriminating shape in the plan — here, "no single line contains both `os.ReadFile`
  and `baseline.json`" — and state the design constraint that makes it satisfiable (the helper
  takes the path as a parameter).

**Round 2 found the same traps in fresh locations, which is the real lesson — fixing an
instance is not fixing the class:**

* THE INVERTED SWEEP RECURRED in a sibling leg ("the file contains no `baseline pull`"):
  zero occurrences today, so green at RED time, and the fix's own error-message text would
  have flipped it red *after* the GREEN task. Generalized rule: **a sweep leg that forbids a
  literal the fix itself must introduce is inverted. Forbid the MECHANISM, not the string** —
  imports and call sites distinguish executing a command from naming one, and a string in an
  error message trips neither.
* **A claim whose forbidden alternative also passes is not pinned.** The plan claimed
  "step-level env, not job-level" while the test accepted the token "from the step's env OR
  the job's env" — so the rejected spelling passed everything. Every narrowing claim needs a
  NEGATIVE leg naming what must be absent.
* **`if: always()` is rarely the guard you want.** On CI it fires on runs where the thing being
  diagnosed never executed, so the diagnostic misattributes an unrelated upstream failure to
  your lane. Guard on the anchor step's conclusion (`!= 'skipped'`), and assert the anchor's
  `id` actually exists — a guard naming a nonexistent id silently evaluates to a permanent skip.
* **Counts stated in a plan get checked.** Four were wrong (16 tests not 14; 6 call sites
  across 3 tests, not "fourteen tests"; 6 `TestRatchet_` matches not 5). Count with a command
  and paste it, or don't state a number.

**Round 4 — the JUSTIFICATION prose is a claim too, not just the headline measurements.** The
plan opened with "every number below was produced by running the command, not inferred," then
justified a guard with "this job dies at tool install/build/pack install often enough that it is
the common case." Measured after the fact: of the last 78 CI runs (40 failures), **38 died at
"Run the gate" and ZERO before it**. The guard was still right to keep — one expression, cheap,
and its *mechanism* was separately confirmed on real history (run `30398137055`: "Generate
baseline" → `failure`, next step "Publish baseline" → `skipped`) — but it is insurance against
an unobserved case, not a fix for a common one, and saying otherwise inside an
evidence-disciplined document is the corrosive kind of wrong. Restate as measured, and cite the
run that confirms the mechanism separately from the frequency.
Also from this round: **a stop-the-lane precondition must key on EVIDENCE, not on a step's own
verdict**, when the plan itself offers a mode where that step passes on the failure condition —
otherwise the precondition is literally satisfiable while proving nothing.

**Round 5 — a retraction or count fix is a SWEEP, not an edit, and the highest-stakes copy is
the one the plan tells someone to WRITE INTO THE REPO.** Both corrections from round 4 survived
in spots the first pass missed: the retracted "it has done so" was still in the source comment
TASK-002 prescribes for `.github/workflows/ci.yml` — permanent in-repo commentary that would
have outlived the plan's own notes and become the last surviving copy of a falsified claim — and
the "three tests" count was still in TASK-001's description, which is what an implementing agent
reads first, even though notes prose had been fixed. **When you retract a claim or correct a
count: grep every instance, and check prescribed file CONTENT (comments, error strings, docs)
before plan prose — a claim you tell someone to ship outranks a claim you merely wrote.** Add a
"do not tighten this back" note beside a deliberately hedged shipped comment, or the next editor
restores the overstatement. Same round: an enumeration claiming exhaustiveness listed 39 of 40 —
recount after every edit to it ([[feedback_enumerations_assert_exhaustiveness]]).

**How to apply:** before declaring a plan done — run every verification command verbatim
(including the frontmatter `test_command`, re-checked by actually running the regex against the
full mandated-name list whenever a test is renamed) and read the exit code AND the executed-test
count; grep-read every existing helper a mandated test will call, checking first-vs-last match
and any implicit "exactly one" assumption; apply each source-sweep predicate to the file as it
will look AFTER the fix; and for every "X, not Y" claim, find the leg that fails Y. Prefer a
uniqueness/existence assertion over a positional one — index comparisons have holes wherever
the anchor is ambiguous. See [[feedback_enumerations_assert_exhaustiveness]] and
[[project_fetch_the_artifact_the_fix_would_pull]].
