---
name: relative-packdir-masquerades
description: "`[phase3-fixtures/validator-positive] layer3 positive failed` on darwin is ISSUE-147 (relative packDir) until proven otherwise — RunValidator swallows run errors, so the message cannot distinguish 'wrong verdict' from 'never ran'"
metadata:
  type: project
---

Any darwin `pack test` failure reading
`ERROR [phase3-fixtures/validator-positive] layer3 positive failed` is
**ISSUE-147's relative-packDir sandbox defeat until an absolute-path re-run
says otherwise.** Established 2026-08-17 triaging ISSUE-162, which was filed
arguing explicitly (and wrongly) that it was NOT ISSUE-147.

**The mechanism, verified in tree and by re-running the real CLI:**
- `pack test` with **no argument** defaults `packDir` to `"."`
  (`cmd/backstop/pack_test_cmd.go:29`) — so running it from *inside* a pack,
  the documented default, is the relative case. `TestPackAuthoringLoop_EndToEnd`
  step 3 does exactly this.
- `darwinSandboxProfile` embeds that path verbatim as a `(subpath "...")`
  clause. A relative subpath does **not** blow up the profile — it applies, and
  then silently matches nothing, so the sandboxed validator gets
  `Operation not permitted` reading its own pack. Exit 71 is only the
  **convert**-seam symptom; the **validator** seam gives a plain non-zero exit.
- `RunValidator` (`pkg/packval/executor.go:261-267`) collapses ANY run failure
  into `Passed:false` with a **nil error**, and `phase3.go:141` fires on
  `err != nil || !r.Passed` — so one message covers both "ran, wrong verdict"
  and "never usefully ran". The **negative** branch (`phase3.go:163-168`) DOES
  distinguish. This asymmetry is what makes the duplicate so convincing.

**Why:** ISSUE-147 is filed as convert-step-only, so nobody expects it on the
validator path; and the scaffolded pack it kills is provably *correct* (same
pack + absolute packDir = all six phases pass), which sends triage hunting a
fixture-polarity bug that does not exist. Blast radius is bigger than ISSUE-147
says: `pack test` from inside any validator-bearing pack is red on macOS,
invisible to CI (darwin-skip + Landlock on Linux).

**How to apply.** One command settles it — re-run the same pack with an
**absolute** packDir arg. Free corroboration before running anything:
`PLAN-ISSUE-146` (completed) already carries the rule at TASK-004 ("check the
packDir you passed BEFORE suspecting the fix") and pre-attributes this exact
red at TASK-005 with DO-NOT-skip/waive/absolutize guards — a *completed plan's
verification prose* is a first-class triage source, not just history. Do not
propose promoting DIR-024 (19-item catch-all) to carry ISSUE-147; ask for a
sequencing ack. See [[project_phase3_polarity_and_silent_parse]] for the
findings-seam polarity (different seam, opposite `Passed` meaning) and
[[project_check_the_siblings_plan]] for the general form of this miss.
