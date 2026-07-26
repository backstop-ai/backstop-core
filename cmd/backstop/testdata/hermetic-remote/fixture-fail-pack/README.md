# hermetic/fixture-fail-pack

The pack that PASSES `pack check` and FAILS `pack test` (SPEC-055, TASK-002). It
exists so a consumer can tell the two validator entry points apart: without it, a
fixture-phase failure would be indistinguishable from the check failure that
`invalid-pack` already provides.

## The named failure

    ./bin/backstop pack check cmd/backstop/testdata/hermetic-remote/fixture-fail-pack
    status: pass

    ./bin/backstop pack test cmd/backstop/testdata/hermetic-remote/fixture-fail-pack
    status: fail
    - phase3-fixtures: fail
    ERROR [phase3-fixtures/semgrep-rule-id] pack rule ID "forbidden-marker" not found in rule file rules/forbidden-marker.yml

## One bit apart from the valid pack

This pack is `valid-pack` with the rule file's declared id changed. Same manifest
shape, same declared engine, same phases — one differing datum, opposite verdicts on
`pack test`. That is what makes the pair falsifiable: if the fixture phase ever stops
running, this pack starts passing and its test fails.

The failure needs no tool: it is a manifest-versus-rule-file identity mismatch that
the fixture phase detects before any execution, so this pack is as hermetic as its
passing sibling.

Do not reconcile the ids, and do not add claims — claims would make the fixture phase
attempt real engine execution.
