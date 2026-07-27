# recipes-archetype-pack-no-enforcement — a fixture that must stay RED

This pack is **expected to fail `backstop pack check` with a non-zero exit,
forever**. Do not repair it.

It is the negative half of the ISSUE-085 acceptance pair. Its sibling,
`../recipes-archetype-pack`, is the field-guide shape that must PASS; this one is
byte-for-byte the same idea with the teeth removed: its single recipe declares
`kind: scaffolding` — a kind that regenerates its output and therefore owes a
drift story — and declares no `enforcement:` block at all.

The pack is structurally clean everywhere else on purpose. It must survive
phases 1, 2, 3, 5 and 6 so that it fails in **phase4-archetype**, on check
`recipe-enforcement`, with a message naming the recipe id `scaffold-service`.
A failure anywhere earlier means the fixture drifted and is no longer testing
what it exists to test.

It lives as its own committed tree rather than as a mutation of the green
fixture so that it is greppable and cannot be silently "fixed" by an unrelated
edit — the same convention as `testdata/packgate-broken` and
`testdata/hermetic-remote/fixture-fail-pack`.

If you arrived here because a test went red: the fix belongs in the code under
test, not in this directory.
