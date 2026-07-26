# hermetic/valid-pack

The VALID half of the hermetic remote-dependency fixture substrate (SPEC-055,
TASK-001). Later phases publish this directory as a real tagged git repository at
test time and drive `pack add` / `pack install` / `pack update` against it through
the built CLI (CLM-062, CLM-064, CLM-066).

## The fixture is proven, not asserted

Nothing here is hand-tuned to whatever the validator happened to accept. The tests
that consume this directory run the real `pack check` and `pack test` pipelines
against it, so a change that breaks the pack breaks those tests. Both commands exit
0 on this directory today:

    ./bin/backstop pack check cmd/backstop/testdata/hermetic-remote/valid-pack
    ./bin/backstop pack test  cmd/backstop/testdata/hermetic-remote/valid-pack

## Why it executes nothing

`pack add` validates every clone with the full check + test pipelines, so this pack
is re-validated on every E2E add. Executing a real engine there would mean a network
fetch, a provisioned binary, and a flaky, slow suite. The rule declares no claims —
the ordinary mechanism-rule shape, which packval permits precisely when the rule's
declared engine resolves — so `phase3-fixtures` performs its rule-file identity
check and no execution. The declared engine carries an empty command so that any
future change that DID route execution here fails loud instead of quietly reaching
for a real binary.

## Relationship to its siblings

`fixture-fail-pack` is this pack with exactly one bit flipped: its rule file declares
a different rule id, so it still passes `pack check` and fails `pack test`'s fixture
phase. `invalid-pack` fails `pack check` at phase 1. Together the three make the
validator's verdict falsifiable in both directions and distinguish a check failure
from a fixture failure.

Do not add a `.git` directory here. The harness creates the repository.
