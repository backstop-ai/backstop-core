---
name: ci-confirmation-string-must-exist-in-the-log
description: When a plan hands verification to a CI run, every string in the confirmation signature must be grepped out of a REAL log first — a string derived from the source that emits it may never reach CI output
metadata:
  type: project
---

When a plan defers its real verification to a CI run and names a literal-string signature as
the confirmation criterion, VERIFY EACH STRING APPEARS IN AN ACTUAL CI LOG before writing it
into the plan. Grep a pre-fix run's FULL log (`gh run view <id> --log-failed` is not enough —
pull the whole log) and record the per-string hit counts.

**Why:** PLAN-ISSUE-180 (2026-08-19) named a THREE-string confirmation signature for its
`ubuntu-latest` gate run. Two were real. The third, `the sandboxed command wrote no
diagnostic`, appeared ZERO times in the pre-fix run's full 158,712-byte log. The plan derived
it from `pkg/packval/sandbox_diagnostic.go`, which is where the string LIVES — but CI's
verbosity folds that inner diagnostic away before the gate reports, so it never reaches the
log the criterion would be read from. Measured counts:

    TestInstallContractsLocalPack_InstallsWithSuppliedCommand: 1
    in phase3-fixtures: 14 validation error(s): 1
    the sandboxed command wrote no diagnostic: 0

The criterion was written as a CONJUNCTION — "all three present = not fixed" — which is
UNSATISFIABLE as stated. A later reader applying it literally would conclude the defect was
fixed on a run where nothing had changed. The implementer caught it and corrected it to the
two usable strings.

**How to apply:**
- A confirmation criterion must be measured against **the artifact it will be read from**,
  never against the source that emits it. Source-presence proves the string can be produced,
  not that it survives to the log/report.
- State criteria as a conjunction only over strings you have counted. If a string is
  unverifiable, drop it from the criterion and say why in the plan, rather than leaving a
  plausible-looking third clause.
- The same discipline applies to the structured report: read `gate-report.json`'s
  `.scope.files` to discharge "the package was actually in scope" — do not infer scope from
  the diff or from the run's conclusion. (On PLAN-ISSUE-180 the close-out downloaded the
  `gate-report` artifact and read all four scope entries verbatim.)
- Related: [[project_run_the_command_you_prescribe]] (run what you prescribe),
  [[project_fetch_the_artifact_the_fix_would_pull]] (hash the CI artifact, don't assume),
  [[project_ci_evidence_run_is_branch_not_main]] (resolve every cited run's `headSha`).
