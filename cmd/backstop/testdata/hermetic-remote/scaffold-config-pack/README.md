# hermetic/scaffold-config-pack

The SCAFFOLD-CONFIG fixture (SPEC-056 REQ-010, TASK-003), and the corpus's first
`tier: complete` scaffold declaration — every other scaffold in this repository is
`tier: skeleton`, so nothing before this could provoke the mutation REQ-008 exists to
contain.

## What it is for

packval phase3 renders each complete scaffold's declared `sample_config` entries into
`<packDir>/<scaffold.path>/<relPath>` before running the scaffold's test command. `pack
add`, `pack update` and `pack upgrade` all validate a tree in place and then copy and
hash that same tree, so for a pack shaped like this one the hash recorded at add time
cannot be reproduced by a fresh clone. This fixture is what makes that reproducible on
demand, and CLM-090 is the round trip that proves the scratch-copy fix.

## Three things that must not be tidied

`sample-settings.yml` must stay UNAUTHORED. A rendered file is only detectable because
no authored file sits at that path (CLM-102). A `pack test` run against this directory
WILL create it — that file is the mutation under study, not a missing fixture. Do not
commit it.

`test_command` must stay `:`. `DefaultExecutor.RunScaffoldTest` runs `sh -c`
unconditionally for a complete scaffold and `PackvalValidator` supplies no executor, so
a shell subprocess is unavoidable here. The hermetic property this fixture claims is
network-free and toolchain-free, NOT process-free.

`:` is the POSIX SPECIAL built-in null command: a conforming shell resolves it before
any PATH search, and it has no external counterpart. `true` would NOT do — it is a
builtin too, but `/usr/bin/true` also exists, so `exec.LookPath("true")` succeeds and
CLM-103's assertion (the declared command resolves to no executable on PATH) fails
against it. That assertion is the only mechanical statement of "reaches nothing outside
the process", so the command has to be one PATH genuinely cannot supply.

The archetype must stay `code`. packval phase4 rejects an enforcement pack that
declares any scaffold, and a code pack requires each rule to declare `pairs_with` and
each scaffold's `pairs_with.rules` to resolve — which is why the rule and the scaffold
here point at each other.

## Proven, not asserted

Both commands exit 0 on this directory:

    ./bin/backstop pack check cmd/backstop/testdata/hermetic-remote/scaffold-config-pack
    ./bin/backstop pack test  cmd/backstop/testdata/hermetic-remote/scaffold-config-pack

`TestHermeticFixture_NewFixturesPassPackCheckAndPackTest` (CLM-104) drives the built
binary to keep it that way. Because `pack test` renders into the tree, prefer copying
this fixture into a temp directory and validating the copy — which is, not
coincidentally, the same move REQ-008 makes.

Do not add a `.git` directory here. The harness creates the repository, and it fatals
on a source that arrives carrying one.
