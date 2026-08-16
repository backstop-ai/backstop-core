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

**Two sharp edges worth a notes section every time:** (1) the flip makes the gate stricter
repo-wide the moment the lock updates, so the plan must force an unfiltered suite run and an
honest resolution of anything already red — before the relock task, not after; (2) the
falsification window is order-dependent — the regression test must be observed RED against
the unflipped fixture, and that evidence is unrecoverable once the flip lands.
