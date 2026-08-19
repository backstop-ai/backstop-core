#!/bin/sh
# go-toolchain coverage PRODUCER (ISSUE-045 option (ii)). The UN-SANDBOXED half of the
# coverage engine: the dispatch runs it via the runner (cwd = project root) with FULL
# toolchain + project access, IN PLACE of the plain engine command, so its `go test` /
# `go list` see the project. Its job is to run the Go coverage pass AND fold the two
# pieces of Go/toolchain knowledge the SANDBOXED parse-only convert
# (coverage-to-records.sh) cannot obtain itself — the module path and the package .go
# file list — into cover.out as PLAIN-TEXT comment lines. The convert then PARSES them
# (it runs no `go`). This producer/convert split is what keeps the executor language-
# blind AND the convert toolchain-free.
#
# Output contract: cover.out (the engine's declared stdout_artifact) — the raw Go
# `-coverprofile` with appended `#backstop-module <M>` and `#backstop-gofile
# <import-path>` lines. The producer's own stdout is noise; the dispatch reads the
# FILE. POSIX sh.

# REUSE THE PROFILE THE TEST RUN ALREADY PRODUCED (ISSUE-172; ISSUE-068's parked
# go-toolchain follow-on). go-test's command now carries -coverprofile=cover.out, so
# by the time this producer runs — a later step in the SAME gate invocation — the
# whole-module profile already exists. Re-running the suite here made the gate's two
# dominant steps one workload paid for twice.
#
# THE STAMP IS THE EVIDENCE, AND IT IS CONSUMED ON BOTH BRANCHES. test-produce.sh
# writes it ONLY after a whole-module (`./...`) run, so its presence means "a complete
# profile was produced in THIS invocation". Deleting it here makes reuse structurally
# impossible without a test run having written it: a leftover cover.out from an
# earlier run carries no stamp and is therefore never honoured. A file-mode run's
# profile is PARTIAL and is never stamped at all.
#
# THE DIRECTION OF THE FRESHNESS COMPARISON IS THE WHOLE MECHANISM (ISSUE-179).
# test-produce.sh writes cover.out via `go "$@"` and touches the stamp AFTER it, in the
# same script, so on a genuine same-invocation success the STAMP is the NEWER of the two
# — always, without exception. The check below therefore asks whether the stamp is at
# least as new as the profile. That is false in exactly one situation, and it is the
# situation worth refusing: a LATER run has overwritten cover.out without stamping it,
# which is precisely what a file-scoped (PARTIAL) run does.
#
# The previous form asked the opposite — whether the profile was no older than the stamp
# — which inverts the real write-then-touch order and so demanded a state a successful
# run never produces. Where `-ot` compares at whole-second resolution the few-millisecond
# gap ties and reuse fired by coincidence; where `-ot` reads nanoseconds (the /bin/sh
# Ubuntu ships) it never fired at all, making the mechanism a complete no-op on the one
# platform it was built for.
#
# THE DEGRADED PATH IS SLOW-BUT-CORRECT, NEVER WRONG. No stamp, a missing profile, or a
# stamp older than the profile all fall through to running the tool exactly as this
# producer always did. Losing the reuse costs SPEED only — which is also why the step
# ordering it depends on is pinned by an executable guard in backstop-core
# (TestGateStepOrdering_PackEnginesPrecedesItsDependentSteps) rather than left to a
# comment: without it the reuse could be reordered into a no-op with nothing red.
#
# ONE RESIDUAL, STATED HONESTLY RATHER THAN CLAIMED AWAY. If an invocation aborts between
# the test dispatch and this one its stamp survives (the file is gitignored, so nothing
# surfaces it), and a later invocation that does NOT overwrite cover.out will honour it.
# That covers a build-broken run, where the gate is already failing — but also a `--file`
# scope holding no Go files, where the package-scoped test engine is never dispatched at
# all and the verdict can come back GREEN over a measurement of a tree that has since
# changed. The profile reused is always the COMPLETE one, never a PARTIAL one, which is
# the dangerous case and the case this comparison refuses. Closing the window entirely
# would mean clearing cover.out in test-produce.sh, which changes that script's semantics
# for every consumer.
stamp=".backstop/go-coverage-fresh"
reuse=0
if [ -f "$stamp" ] && [ -f cover.out ] && [ ! "$stamp" -ot cover.out ]; then
  reuse=1
fi
rm -f "$stamp"

# Run the coverage pass. Tolerate a non-zero exit — a failing suite still yields a
# usable profile; the convert + gate decide the verdict, never this producer. (A
# REUSED profile comes from the same failing run, so the behaviour is equivalent.)
if [ "$reuse" -eq 0 ]; then
  go test -coverprofile=cover.out ./... >/dev/null 2>&1 || true
fi

# Nothing to enrich if the profile was not produced at all.
[ -f cover.out ] || exit 0

# Fold in the module path so the convert can strip the "<module>/" import-path prefix
# to yield repo-relative record paths (ISSUE-045 case-2, CLM-005).
module=$(go list -m 2>/dev/null | head -1)
if [ -n "$module" ]; then
  echo "#backstop-module $module" >> cover.out
fi

# Fold in every package's .go files (module-qualified, one per line) so the convert can
# emit a total:0 N/A record for a zero-statement file ABSENT from the profile whose
# package WAS measured (ISSUE-045 case-1, CLM-001). _test.go files are excluded by
# {{.GoFiles}} (test files are TestGoFiles), so only measurable source is listed.
go list -f '{{range .GoFiles}}{{$.ImportPath}}/{{.}}
{{end}}' ./... 2>/dev/null | while IFS= read -r gofile; do
  [ -n "$gofile" ] && echo "#backstop-gofile $gofile"
done >> cover.out

exit 0
