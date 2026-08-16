---
name: feedback_omitted_subject_inherits_wrong_package
description: A claim with no explicit `subject:` inherits the spec-level package by default — if its mandated test actually lives in a different package, the substantiveness join can still pass through an incidental symbol reference or a kind:absence exemption, hiding the mis-subject until the spec flips to implemented
metadata:
  type: feedback
---

Found flipping SPEC-069 to `implemented` (2026-08-15): CLM-046, CLM-047,
CLM-086, CLM-088 and CLM-130 carried no `subject:`, so they inherited the
spec-level default (`pkg/initialize`) — but all five claims' mandated tests
actually live in `cmd/backstop/init_seams_test.go`, package `main`. This sat
green through the spec's entire `draft` lifetime because
`test_substantiveness` only enforces at `implemented` status
([[project_review_state_keystone]]-adjacent: enforcement gated on a status
value, not on the claim being wrong).

**Three non-obvious details that let this hide:**

1. **The substantiveness join is per TEST FUNCTION, not per file.** A file
   header comment asserting "this file references package X, so every test
   in it satisfies X's claims" is false — `pkg/gate/substantiveness_join.go`'s
   `NoTargetViolation` and `cmd/backstop/gate.go`'s
   `testFileColocatedWithTarget` evaluate colocation or symbol-reference
   per function body, not per file. `cmd/backstop/init_seams_test.go` had
   exactly this false comment, left uncorrected during this fix (source
   edit with no plan in flight — filed as an issue instead of hand-edited).

2. **Only package-qualified CALL/selector references count. Composite literals,
   constants and TYPE POSITIONS do NOT.** The extraction rule is
   `referenced-symbol-go.yml` in `backstop-ai/go-substantiveness`: it matches a
   `call_expression` whose function is a `selector_expression` with an identifier
   operand, inside a `Test*` function. So `pkg.Func()` counts; `pkg.Type{…}`,
   `pkg.Const`, and a type-position mention like the closure parameter
   `_ *gate.GateScope` do NOT — a test can genuinely exercise the package and
   still extract an empty set for it. In SPEC-069 the four non-`kind: absence`
   claims' test bodies touched `pkg/initialize` only via `initialize.Options{…}`
   (a composite literal) and `initialize.OutcomeDelivered` (a constant); in
   SPEC-037 (2026-08-15) the same defect recurred across NINE claims whose only
   `gate.` token was a type position. Either way the fallback path that would
   rescue an incidentally-correct mis-subject doesn't fire.

3. **`kind: absence` is EXEMPT from the join entirely**, which looks like a
   pass but is actually a non-check — CLM-088 (an absence claim) "passed"
   this whole time for the wrong reason, the same mis-subject as its four
   siblings, just invisible because the join never ran on it at all.

**The inverse case — a mis-subject that DOES join is fragile, not fine.**
SPEC-037 CLM-035 (fixed 2026-08-15, v1.2.6) was the tenth sibling of the nine
in detail 2, deliberately left alone by that fix because it passed: its test
body happens to make real `gate.`-qualified CALLS (`gate.ClassifyDimension`,
`gate.DimensionSubstantiveness`), so the symbol-set branch fired even though
the test lives in `cmd/backstop` and pins `cmd/backstop/gate.go`'s
`deriveCapabilityState`. That is incidental correctness: the join depends on
one specific call surviving inside the body, and a refactor routing those
assertions through a local helper would silently drop the only branch holding
it up — reintroducing the exact `noTarget` failure, invisible again until the
`implemented` flip. **A wrong `subject:` that currently passes still gets
corrected**, framed as preventive hardening (nothing broken, no live defect)
rather than as a bug fix. Colocation is the durable branch because it depends
on where the file sits, not on what the body calls.

**How to apply:** when authoring or auditing a claim with no explicit
`subject:`, don't trust the spec-level default to be right just because the
claim is siblings with others in the same requirement group — check where
its OWN mandated test(s) actually live. If a sibling claim in the same test
file already declares an explicit `subject:` different from the spec
default (as CLM-087/CLM-110 did here), that's a strong signal the omitted
ones are wrong too, not that the join will sort it out. Detect this the
same way [[feedback_claim_subject_is_one_package_only]] recommends for
straddling claims: force `implemented` on a scratch copy and run the real
join, don't wait for a live status flip to find out.

**Running the scratch simulation (SPEC-037, 2026-08-15):** `rsync -a
--exclude .git/` the repo to scratch (`.backstop/packs/` must come along —
the join needs the installed extraction pack), flip `status:` with the Edit
tool (the agent-guard blocks `sed`/`python` writes to artifact paths even in
scratch), then `./bin/backstop gate --all --json`. If a Layer-0 tool is
missing on PATH the gate exits 2 at `pack_engines` and never reaches
`test_substantiveness` — drop the offending pack from the scratch
`backstop.yml` AND `backstop.lock` and delete its `.backstop/packs/` dir so
lock verification still passes. Diff the step's violation COUNT before and
after the fix, not just the named ones, to prove no new violations appeared.
