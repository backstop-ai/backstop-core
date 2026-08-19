---
name: ci-evidence-run-is-branch-not-main
description: A close-out's "post-merge CI run" is usually a PR-BRANCH run — check headSha against the merge commit before recording it as main-verified
metadata:
  type: project
---

At close-out, resolve every cited CI run id with
`gh run view <id> --json headSha,createdAt,conclusion` before writing it into an
AS-BUILT banner. A dispatch brief that says "a second CI run post-merge confirmed it
clean" frequently names a run whose `headSha` is a PR-BRANCH commit, not the squash
merge — and the real post-merge `main` run may still be `in_progress`, i.e. unread by
anyone.

**Why:** PLAN-ISSUE-176 (2026-08-18). Both cited evidence runs (32194863181,
32197980951) were on PR #4's branch at `0ee071d` / `a230337`; the merge commit
`14d87ad`'s own main run (32200897995) was still in flight when the banner was
written. The wiring was identical, so the evidence held — but "verified on main" would
have been a false claim, and a squash merge hides `0ee071d`/`a230337` from main's
first-parent history, so the run↔commit mapping is not recoverable later without
recording it.

**How to apply:** in the banner, label each run with its `headSha` and what that commit
is (branch cut / merge / pre-fix main). Cite a PRE-FIX main run as the counterfactual —
that is what turns "the failures are gone" into a measured delta (here: `Total
violations: 4` naming all three ratchet tests at `0943ec4`, vs `1` after). State any
still-in-flight main run explicitly as NOT YET OBSERVED, with the expected output, so
the next reader knows a divergence is a finding. Related:
[[project_closeout_verify_the_fix_landed]], [[project_fetch_the_artifact_the_fix_would_pull]].
