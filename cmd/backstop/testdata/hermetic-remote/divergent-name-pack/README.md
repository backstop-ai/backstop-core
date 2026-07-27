# hermetic/renamed-pack (directory: divergent-name-pack)

The DIVERGENT-NAME fixture (SPEC-056 REQ-010, TASK-002). The manifest declares
`hermetic/renamed-pack`; the directory — and therefore the coordinate a hermetic test
requests — is `hermetic/divergent-name-pack`. The mismatch between those two strings
is the fixture, and it is the shape that produced the real `missing convert script`
failure a consumer hit with a pack legitimately named `backstop/harness-toolchain`
living at `backstop-ai/backstop-harness-toolchain-pack`.

## What depends on the divergence

REQ-003 makes the MANIFEST name the install/runtime identity: the install path under
`.backstop/packs/`, the `backstop.yml` packs key, the lock entry key, and the engine
asset root must all read `hermetic/renamed-pack`, while REQ-004 records
`hermetic/divergent-name-pack` verbatim in `source_coordinate`. CLM-028..031 and
CLM-062..067, CLM-070 all hang off this one pack.

## Two things not to tidy

The difference must stay BYTE-level, not case-level: CLM-066 exists for the case-only
shape and gets its divergence from the caller varying the ref's case, so a case-only
fixture here would silently make the general divergence claims duplicates of it.

The declared rule file must keep existing. CLM-031 resolves it under the manifest-name
asset root after a divergent add; a content-less pack would make that claim
unfalsifiable.

Do not add a `.git` directory here. The harness creates the repository, and it fatals
on a source that arrives carrying one.
