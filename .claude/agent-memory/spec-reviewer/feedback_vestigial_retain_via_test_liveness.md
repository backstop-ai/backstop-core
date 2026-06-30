---
name: vestigial-retain-via-test-liveness
description: A deletion spec that RETAINS a branch "because tests exercise it" may be preserving vestigial code; demand a PRODUCTION producer, not a test fixture
metadata:
  type: feedback
---

When a removal spec narrows scope to RETAIN part of a dead-code cluster on the grounds
that it is "LIVE" — verify the liveness is PRODUCTION liveness, not test-only liveness.
Tests that author their own fixture in a `t.TempDir()` and then exercise a branch are
NOT evidence the branch is production-reachable; they are evidence the branch has test
callers. Under the zero-baked / no-vestigial principle ([[feedback_zero_baked_checks]]),
a production-dead branch is an eradication target even if tests feed it — delete the
branch AND reconcile its tests, don't keep the branch alive and reshape around it.

**Why:** SPEC-039 review #2. The corrective for review #1 narrowed REQ-010 to DELETE only
the compiled-standards sub-reader and RETAIN the legacy routing-schema `.manifest.json`
arm, reshaping `LoadManifest` to keep decoding it — justified as "LIVE: its zero-routable
ConfigError is a tested fail-loud (TestCodeCheck_LoadManifest_ConfigErrorPropagates...) and
the routing-schema decode is exercised by missingToolchainProject + gate-scope tests." But
on `main`: ZERO production `.manifest.json` producers (only the reader + comments reference
the suffix), NO committed `.manifest.json` outside `pkg/check/testdata/`, NO `.backstop/rules/`
dir in the repo, and EVERY cited "live" test writes its own `.manifest.json` into a tempdir.
The arm is dead-in-production, test-only-live → retaining it preserves vestigial code. The
spec's OWN Sharp Edge 3 stated the rule ("dead-fed in production is true; unreferenced is
NOT — the reader has TEST callers; do not interpret test callers as evidence the reader is
live"), then the narrowing violated it for the legacy arm.

**How to apply:** When a spec retains a branch citing tests, run: (1) grep the non-test tree
for a PRODUCER of the input that branch reads (writer/emitter/compiler/scaffolder/baseline-
gen of the file/suffix); (2) `git ls-files` + `find` for committed instances outside testdata;
(3) check whether the cited "live" tests author their own fixture in `t.TempDir()`. If no
production producer and all fixtures are test-authored → VESTIGIAL, FAIL the retain, corrective
is delete-the-branch-and-reconcile-its-tests (same disposition as the rest of the dead cluster).
A genuine fail-loud guard worth keeping must guard a PRODUCTION-reachable input; guarding a
dead input is itself the vestigial pattern. Sibling to [[feedback_existing_test_coupling]]
(that one: name ALL tests coupled to a deletion; this one: don't let those same test couplings
masquerade as production liveness to justify NOT deleting).
