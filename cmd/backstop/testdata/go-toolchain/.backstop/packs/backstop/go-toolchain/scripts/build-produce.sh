#!/bin/sh
# go-toolchain BUILD PRODUCER (ISSUE-067). The go-build analogue of
# test-produce.sh, and the more severe case of the same defect.
#
# WHY IT EXISTS. `go build` writes NOTHING to stdout — EVERY compiler diagnostic
# goes to stderr. Since core captures stdout only (SPEC-031 CLM-028),
# build-to-sarif.sh received an EMPTY payload on every real run and has never once
# emitted a finding in production: every compile error in production Go code
# surfaced as `engine "go build" crashed`, never as a located finding. The
# converter was always correct; the bytes never arrived.
#
# THE SUBCOMMAND IS SUPPLIED BY THE CALLER IN "$@" — DO NOT REPEAT IT.
# splitCommand("go build") yields cmdName="go" + cmdArgs=["build"], so this script
# receives `build ./...`. `go build "$@"` would double the subcommand.
#
# THE TOOL'S EXIT STATUS MUST SURVIVE VERBATIM — see test-produce.sh.
#
# The body is BYTE-IDENTICAL to test-produce.sh's, and that is correct rather than
# a smell: the subcommand rides in "$@", so neither script names a subcommand.
# They are separate files so the pack can declare a distinct producer: per binding
# and so the harness alias can key canned output back to the right command.
go "$@" 2>&1
