---
name: producer-swap-argv-subcommand
description: A pack `producer:` that replaces the TOOL receives cmdArgs INCLUDING the subcommand (splitCommand drops only argv[0]), so the script must be `go "$@"`, never `go test "$@"`
metadata:
  type: project
---

When a plan makes a pack-declared `producer:` script replace the invoked tool while
keeping core's arg shaping (`runner.RunStdout(ctx, producerPath, cmdArgs...)`), the
producer receives the SUBCOMMAND as `$1`. `splitCommand("go test")` returns
`("go", ["test"])` — only argv[0] is consumed as the tool name. So the script body
must forward the whole argv to the bare tool:

    go "$@" 2>&1        # correct
    go test "$@" 2>&1   # WRONG -> `go test test ./...`

**Why:** measured 2026-08-16 while reviewing PLAN-ISSUE-067. `go test test ./...`
exits 1 even on a fully green tree (`package test is not in std` / `FAIL test
[setup failed]`), and `go build build ./...` likewise. With `crash_guard: true` on
the go-toolchain bindings, that turns EVERY gate run into
`engine "go test" crashed: non-zero exit with no parseable findings`.

**How to apply:** whenever a plan prescribes producer/wrapper script content, hand-run
the script with the argv core actually shapes (`splitCommand(binding.Command)` tail +
inputs + `project_target`) against a GREEN tree, not just a failing one. The
`fixtureRunner` test double returns canned bytes keyed by command name and IGNORES
args, so no unit test in the corpus can catch this class — it first surfaces on a real
gate run, often after a public pack tag has already been pushed. Related:
[[verified_enumeration_do_not_rederive]], [[project_pack_provisioning_integration_gap]].
