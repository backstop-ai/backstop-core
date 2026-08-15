---
name: spec037-review
description: SPEC-037 (substantiveness pack) is DELIVERED but NOT promotable — flipping to implemented injects 10 new blocking noTarget violations; contracts are clean
metadata:
  type: project
---

SPEC-037 v1.2.4 (specs/SPEC-037-traceability-substantiveness-pack.spec.md) is DELIVERED
in code but BLOCKED from `status: implemented`. Investigated 2026-08-15 (completeness
audit, not a spec review). It PASSED spec review at v1.2.1 and is cleared to plan — that
part still holds; the blocker is a promotion-time gate consequence.

**Delivery evidence (verified, not asserted):** all 37 mandated tests exist and PASS on a
clean HEAD copy (`go test ./pkg/gate/ ./cmd/backstop/ -race`); the baked analyzer
(`checkSubstantiveness`/`hasAssertions`/`assertionSelectors`/`callsTargetPackage`) is gone
from `pkg/gate/step_testverify.go`; `backstop artifact validate --spec SPEC-037` passes.

**BLOCKER — the Q2 noTarget set-join, not SPEC-036's pack-compiler gap.** Mandated tests
enter the noTarget join ONLY at terminal status (`ContractsAreDue`, `buildTestSubstantivenessStep`
cmd/backstop/gate.go:1177), so the flip is what turns it live. The spec's subject is
`pkg/gate` → target token `gate`; its 26 pkg/gate tests pass via same-package, but 10 of
its 11 `cmd/backstop` tests do NOT reference `gate` in the pack-extracted symbol set →
10 NEW `test_substantiveness` violations. Verified empirically by running the installed
`backstop-ai/go-substantiveness` ast-grep rules over the real test files and by confirming
none of the 10 messages exist in `.backstop/baseline.json`. Policy is
`test_substantiveness: applies-to new-code, level: block`, and new-code grandfathers
against the BASELINE (pkg/gate/policy.go:219) — absent from baseline means it blocks.

**Root cause worth knowing:** the extraction rule counts CALL/selector references, not
type-position ones. `TestWiring_SubstantivenessStepRoutesThroughDispatchSeam` genuinely
exercises pkg/gate but its only `gate.` token is a closure parameter TYPE
(`_ *gate.GateScope`) → not extracted. Cheapest correct fix is a per-claim
`subject: cmd/backstop` on the wiring/capability/provisioning/E2E claims (those claims
really do target the cmd wiring); `kind: absence` and test-body edits are the alternatives.

**Second blocker, recorded by the spec itself (v1.2.4):** CLM-031's mandated test name
`TestProvisioning_SubstantivenessInstalledAsLocalPack_DeclaredAndLocked` contradicts the
amended claim text (which now accepts a `git` source); the live pack is remote
(`git_ref: v1.2.0`). Renaming is test-code work needing its own issue → plan.

**NOT a blocker (checked because SPEC-036 was blocked this way):** all 7 present contract
entries compile through `backstop-ai/go-contracts` `compile-signature.sh` and MATCH live
code — func params are metavar'd (`$$$PARAMS`), so the three drifted signatures
(`RouteSubstantivenessFindings`, `deriveCapabilityState`, `buildTestSubstantivenessStep`)
are DOCUMENTATION drift only. The `absent: true` `checkSubstantiveness` entry probes by
grep on the NAME (Signature ignored) and the deletion comment deliberately never spells
the literal → clean. Coverage is a no-op too: pkg/gate's floor is already 90 (SPEC-044).

**How to apply:** do not promote SPEC-037 until the noTarget exposure is dispositioned.
See [[spec036-ready-to-plan]] (sibling, BUNDLE-009 Seed 1, blocked on the grouped-const
pack gap) and [[feedback_flip_injects_notarget]].
