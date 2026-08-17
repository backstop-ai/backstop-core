---
name: trust-gate-fires-at-manifest-parse
description: engine.CheckToolAllowed also runs per-rule inside pack.ParseManifest, so an un-allowlisted provision: never survives loadInstalledPacks — a trust-refusal FIXTURE PROJECT is structurally unreachable
metadata:
  type: project
---

A pack.yml whose rule-bound engine declares a `provision:` naming a tool the
trusted-tool allowlist does not admit **cannot be staged as a fixture project** for any
consumer downstream of `loadInstalledPacks`.

**Why:** the same `engine.CheckToolAllowed` predicate runs per-rule at manifest PARSE
time (`validateEngine`, pkg/pack/manifest.go), not only at the gate/provision walk. So
`pack.ParseManifestFile` returns
`rule "X": tool "Y" is not on the trusted-tool allowlist`, `loadInstalledPacks` fails,
`ctx.PacksErr` is set, `packs-installed` FAILS and owns the condition, and every
downstream check SKIPS naming it. That is correct one-condition-one-owner behavior, not
a defect — but it means the refusal branch in a walk like
`collectRequiredEngineTools` is reachable only from an IN-MEMORY pack set. A version
mismatch fails the same way for the same reason.

Measured 2026-08-16 while building PLAN-ISSUE-134's doctor fixtures: the staged
`engine-tool-refused` project produced `skipped`, never `fail`, and the fixture was
deleted in favor of driving that one case at the function level with a
`pack.Manifest` literal.

**How to apply:** when a plan's setup task asks for a fixture project exercising a
trust-gate refusal, expect it not to work and drive that case at the function level
instead — keep the mandated test name, and record the reason in the test's doc comment
so the next reader does not re-add the fixture. Every OTHER engine fixture should carry
a NIL Provision deliberately (the Layer-0 exemption) so the presence probe is what it
exercises. Related: [[project_hermetic_pack_fixture_recipe]],
[[project_minimal_valid_pack_fixture]].
