# hermetic/version-drift-pack

The VERSION-DRIFT fixture (SPEC-056 REQ-010, TASK-001). Its manifest declares
`0.1.3`; the tags a test publishes for it stop at `v0.1.1`. That inequality is the
fixture — it is the shape of the real published harness toolchain pack (DIR-027
item 2), and it is what REQ-002's identity gate exists to refuse.

## Build it with the non-rewriting constructor, or it tests nothing

`newHermeticRemote` rewrites `pack.yml`'s version to match each tag it creates. Built
through it, this pack's drift is silently repaired and CLM-016 passes over a fixture
that cannot fail. Use `newHermeticRemoteKeepingManifestVersion` (SPEC-056 TASK-005),
which tags the imported source commit as-is.

`TestHermeticFixture_VersionDriftPackHasDriftingManifest` (CLM-100) asserts all three
values — `v0.1.0`, `v0.1.1`, and `0.1.3` — rather than mere inequality, so a fixture
edited to some other non-matching version still reds.

## Why it executes nothing

Copied from `valid-pack`: the declared rule carries no claims, which is the ordinary
mechanism-rule shape packval permits precisely when the rule's declared engine
resolves, so `phase3-fixtures` performs its rule-file identity check and no execution.
The declared engine carries an empty command so that any future change that DID route
execution here fails loud instead of quietly reaching for a real binary.

Do not add a `.git` directory here. The harness creates the repository, and it fatals
on a source that arrives carrying one.
