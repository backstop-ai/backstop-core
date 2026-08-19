#!/bin/sh
# go-toolchain TEST PRODUCER (ISSUE-067). The UN-SANDBOXED half of the go-test
# findings engine: the dispatch runs THIS packRoot-resolved script via the runner
# (cwd = project root) IN PLACE of the plain command's tool.
#
# WHY IT EXISTS. `go test` SPLITS its output across two streams: an assertion
# failure lands entirely on stdout, but a COMPILE or VET failure puts the located
# diagnostic on STDERR while stdout carries only `FAIL <pkg> [build failed]`. Core
# captures stdout ONLY, by deliberate design (SPEC-031 REQ-009/CLM-028 made it an
# explicit buffer so stderr cannot interleave into a SARIF-on-stdout engine's
# bytes). So every located compiler diagnostic was discarded BEFORE the convert
# ran, the convert emitted zero results, and crash_guard reported the opaque
# `engine "go test" crashed: non-zero exit with no parseable findings`.
#
# The merge is the PACK's job because the PACK is what knows this tool splits its
# streams. Core learns nothing about Go; it just runs what the pack declares.
#
# THE SUBCOMMAND IS SUPPLIED BY THE CALLER IN "$@" — DO NOT REPEAT IT. Core swaps
# only argv[0]: splitCommand("go test") yields cmdName="go" + cmdArgs=["test"],
# then the scope shaping appends the target, so this script receives
# `test ./...`. Writing `go test "$@"` would run `go test test ./...`, which fails
# with `package test is not in std` even on a GREEN tree — making every gate run
# permanently and opaquely RED.
#
# THE TOOL'S EXIT STATUS MUST SURVIVE VERBATIM. crash_guard and the non-zero-exit
# contract both read it. Do NOT add `|| true` or `exit 0`: the tolerant tail in
# coverage-produce.sh is correct for a PROFILE producer and catastrophic here,
# because it would turn a failing suite into a clean-looking run.
#
# ★ AND THAT IS WHY THE NEXT LINE CAPTURES $? IMMEDIATELY (ISSUE-172). This script
# USED to END on the tool invocation, so the tool's status WAS the script's status by
# construction. It no longer does — the stamp block below runs after it — so the
# status is captured here and re-raised by the final `exit "$status"`. Appending any
# command after the tool without doing this replaces the script's exit status with
# that command's, and EVERY FAILING SUITE WOULD READ GREEN. Guarded executably by
# TestGoToolchainSingleRun_TestProducerPropagatesFailureExit in backstop-core.
go "$@" 2>&1
status=$?

# THE SINGLE-RUN FRESHNESS STAMP (ISSUE-172; ISSUE-068's parked go-toolchain
# follow-on). The engine command now carries -coverprofile=cover.out, so THIS run
# already produced the coverage profile. The stamp tells scripts/coverage-produce.sh —
# a LATER step in the SAME gate invocation — that it may reuse that profile instead of
# running the whole suite a second time. The coverage producer CONSUMES (deletes) the
# stamp, so reuse is structurally impossible without a test run having written it.
#
# ★ THE `./...` CONDITION IS THE WHOLE POINT, NOT A TIDINESS CHECK. go-test declares
# package_scoped: true, so in FILE mode the dispatch narrows the target to the changed
# packages and the resulting profile is PARTIAL. Stamping a partial profile as
# reusable would hand the coverage dimension an incomplete measurement — every
# unmeasured file reading as absent — which is exactly the silent-narrowing class this
# tool exists to prevent. Only the whole-module target may be stamped; every other
# scope falls through unstamped and the coverage producer runs the suite itself.
case " $* " in
  *" ./... "*)
    if [ -f cover.out ]; then
      mkdir -p .backstop
      : > .backstop/go-coverage-fresh
    fi
    ;;
esac

exit "$status"
