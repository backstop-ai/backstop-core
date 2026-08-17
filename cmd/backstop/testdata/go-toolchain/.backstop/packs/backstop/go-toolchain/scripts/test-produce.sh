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
go "$@" 2>&1
