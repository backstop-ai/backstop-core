---
name: planning-a-pack-data-fix
description: How to plan a fix that lives in an external pack's declared data — external repo as literal file scope, the dual-declaration lockstep, and the SPEC amendment the flip usually forces
metadata:
  type: project
---

When a defect's fix is a DECLARED VALUE in a pack manifest (not core code), the plan has a
recognizable shape. Learned authoring PLAN-ISSUE-129 (`go-test`
`exempt_from_scope_filter`, 2026-08-15).

**The external repo path is legitimate task file scope.** Packs live outside core by
design, so a `setup` task whose `files:` is
`/Users/bmanson/src/projects/backstop-<name>-pack/pack.yml` is correct, not a workaround.
That task owns: edit + version bump + `pack check`/`pack test` in that repo + commit +
tag + push. Version resolution is git-tag-based, so a bump with no `v<version>` tag will
not resolve via `pack update` downstream — say so in the task.

**There are usually TWO declarations to flip, and the tests read the one you'd overlook.**
Core carries in-repo FIXTURE COPIES of packs under `cmd/backstop/testdata/<pack>/.backstop/packs/...`.
For go-toolchain, `goToolchainManifest`/`goToolchainPackRoot`
(`cmd/backstop/pack_gate_gotoolchain_test.go`) parse the FIXTURE — so the whole SPEC-041
exemption corpus asserts against it, and `pack update`ing the installed pack changes no
test outcome. Flip both, plus any documentary mirror (e.g.
`cmd/backstop/testdata/exempt-matrix-bindings.yml`, whose header comment states the values
in prose and goes stale silently). Nothing asserts the two agree — worth filing as a
follow-on, not absorbing.

**Flipping a declared value usually invalidates an `implemented` spec's claims.** Search
the spec corpus for the value BEFORE planning: SPEC-041 CLM-015/CLM-011/CLM-017 all encoded
"go-test is non-exempt" in claim text AND mandated test names. Those are live gate contracts
(`implemented` specs are what test_verification admits), so the amendment goes SPEC-FIRST as
its own commit via a spec-author dispatch, and the plan is the counterpart carrying the
rename verbatim — see [[extending-a-shipped-plan]] for that convention.

**Preserve the proof, don't delete the failing assertion.** A decoupling/matrix test whose
point survives the flip (SPEC-041's ScopeKind-vs-exempt proof still holds via the engine
that stays non-exempt) must be RE-GROUNDED, never dropped. Spell that out in the task or an
implementer will "fix" it by deleting the assertion.

**"Keep file X, a test reads it by name" names a FILENAME, not a COPY — check the consumer's
ROOT CONSTANT.** ISSUE-157 (2026-08-18) said to keep `sig-mismatch.go`/`sig-kinds-mismatch.go`
in the `backstop-ai/go-contracts` MIRROR because `pkg/pack/engine/contracts_kind_signature_test.go`
reads both by name. It does — but its root is `const durablePackRel = "packs/contracts"`, the
IN-REPO source, so no core consumer reads the mirror's copies at all. A filename grep "confirms"
the premise; only the root constant falsifies it. Same shape as the two-declarations trap above:
in-repo source, installed mirror under `.backstop/packs/`, and testdata fixture copies all carry
identically-named files. Keep the file anyway when it costs nothing (source/mirror parity), but
record the REAL reason — an unverified rationale passed downstream becomes load-bearing.

**Re-measure the mirror's CURRENT version; an issue's snapshot goes stale in days.** ISSUE-157
described the mirror at 1.2.0; a sibling lane (ISSUE-166) had already bumped and pushed it to
1.3.0, carrying this defect along unchanged. Read `pack.yml`'s `version:`, `git ls-remote --tags`,
and the tree's clean/sync state before deriving any diff. Then check whether another plan already
owns the pending `pack update` + relock — if so, your higher bump SUBSUMES its core half, and you
must say so without claiming its issue.

**A pending installed-pack pin's version check is a BAR or a PIN, and only one survives your
bump.** ISSUE-166's `TestInstalledGoContractsPack_CarriesFilenameHeaderFix` uses
`semverGreater(entry.Version, "1.2.0")`, so relocking past it to 1.4.0 flips it green as a
byproduct. Had it asserted equality with `1.3.0`, the same relock would have RED'd another lane's
deliberate true-RED test. Read the comparator before promising a byproduct either way.

**A sibling plan's REMAINING scope comes from the COMMITS, not from its task list or its
`status:`.** PLAN-ISSUE-166 still read `status: draft` while four of its tasks had already
shipped in `f8b3846` and `3fdebfd`. I wrote its remaining scope from the plan file and got it
wrong; plan-reviewer caught it before TASK-010 wrote that stale list into ISSUE-157's permanent
Resolution section. Read `git log`/`git show` for the sibling's id before describing what it has
left — and note that anything a close-out task will WRITE INTO an artifact deserves that check
twice, because a wrong fact there outlives the lane.

**Read the commit messages for WHO BLOCKS WHOM before framing an overlap.** I framed this plan's
relock as a byproduct that incidentally helped ISSUE-166. Both commits said the opposite:
"ISSUE-166 stays open by design — core's installed pack still needs `pack update`, blocked on
the unrelated, pre-existing ISSUE-157." Same two facts, inverted causality — and the direction
decides whether your plan is a courtesy or the unblocker. Unblocking still is not closing: the
blocked lane keeps its own issue, close-out, and claims.

**Adding a test beside an existing pin in the same `_test` package: check what that package
already declares.** `package engine_test` already had `installedContractsPackName`,
`preFixContractsPackVersion = "1.2.0"` and `semverGreater`. My new test needed a 1.3.0 bar;
silently reusing the existing 1.2.0 constant would have passed against the still-defective pack —
a weakened check reading green for the wrong reason — and redeclaring it is a compile error.
Name the reusable symbols and the must-not-reuse ones explicitly. ★ Extra-dangerous whenever the
plan frames the new test as "expected to fail when written": a compile error then gets waved
through as the deliberate RED. Say that the expected RED is a RUNNING test with printed
assertion failures, and that a compile error/panic/skip is not it.

**Name the known inherited reds by issue id in every gate step, not just the final one.** "Any
second failure is yours" is too absolute in a repo with open CI-wide reds (here ISSUE-176's
gitignored `baseline.json` and ISSUE-177's phase3 anomaly). Port the prove-inheritance/name-the-
owner clause into the MIDDLE verification tasks too, with the candidates listed.

**Make the founder gate a task boundary, not a paragraph.** Bundling commit + stop-and-ask +
tag/push in one task leaves nothing structurally forcing the handoff. Split it: one task that
commits locally and stops, a separate task whose first line is that it must not start without
explicit authorization. Cheap, and strictly better than prose.

**For fixture/manifest-data fixes, say plainly what RED means instead of force-fitting a Go test.**
The real falsifier is `backstop pack test <clone>`'s own verdict (exit 1, N phase3 errors → exit 0,
six phases green) — measurable in a scratch copy BEFORE authoring, which turns the plan from a
proposal into a reproduction of a verified diff. Then still add a core-side installed-pack test so
the claim outlives the terminal, following the `grep_installed_pack_test.go` three-leg precedent
(tracked-lock leg that never skips + installed-tree legs behind the install guard). That test is
deliberately RED until the relock lands; name the interim RED and its single expected identity in
every gate step before then. Note `pack test | tail` reports the PIPELINE's exit code — read `$?`
unpiped or trust `status:` in the output.

**Two sharp edges worth a notes section every time:** (1) the flip makes the gate stricter
repo-wide the moment the lock updates, so the plan must force an unfiltered suite run and an
honest resolution of anything already red — before the relock task, not after; (2) the
falsification window is order-dependent — the regression test must be observed RED against
the unflipped fixture, and that evidence is unrecoverable once the flip lands.
