---
name: pack-step-provisioning-gap
description: Pack-backed gate steps build in-code manifests with pack-relative rule paths; production resolves them under .backstop/packs but the rules ship only in testdata — step is broken in prod, masked by stub-dispatched/testdata-pointed tests
metadata:
  type: project
---

When a gate STEP is re-implemented to enforce via a pack (replacing a baked analyzer),
the recurring defect is: the step builds an in-code `pack.Manifest` with pack-RELATIVE
rule paths (e.g. `ast-grep/hollow-test-go.yml`), but production `dispatchPackEngines`
resolves the pack root under `.backstop/packs/<normalized-name>/` and `resolveRulePath`
stats the rule file THERE. If the pack rules live only under `pkg/gate/testdata/...`
(not embedded/distributed/provisioned), the real `backstop gate` run hits
"broken pack: missing rule file" and the step config-errors every run.

It passes review-by-green because: (a) real-ast-grep integration tests point
`dispatchAstGrepRule` directly at testdata; (b) wiring spy tests stub
`dispatchPackEnginesFn`. NEITHER drives the step through the REAL (unstubbed)
dispatcher resolving the REAL provisioned pack.

**Why:** Found in SPEC-037 (substantiveness pack, BUNDLE-009 Seed 3) — exactly the
[[feedback_integration_gap]] pattern, and a sibling of [[project_spec035_phase4_gap]]
(pack-engine specs where green tests mask the real seam).

**How to apply:** For any pack-backed gate step, ALWAYS check: (1) do the pack rules
exist on the PRODUCTION pack root (.backstop/packs / embed / distribution / lockfile),
not just testdata? (2) is there ONE test that drives the step builder through the
real dispatcher (dispatchPackEnginesFn NOT stubbed) against a provisioned pack and
asserts the end-to-end verdict? If not, FAIL on integration gap regardless of coverage %.

Separately for STRANGLER specs: when a "prove-equivalence-then-delete" harness ends up
comparing against a HARDCODED captured matrix (not the live analyzer at runtime),
independently re-derive the matrix from the deleted analyzer's source
(`git show HEAD:<file>`) before accepting it — the matrix can be correct-but-unproven,
or fabricated. Verify, don't trust the harness comment.
