---
name: cross-seed-pack-location-convert
description: BUNDLE-009 seed plans that ADD rules to a pack a PRIOR seed owns must use the prior seed's actual pack path/layout AND every real-ast-grep/grep pack needs its own ast-grep/to-sarif.sh + grep/to-sarif.sh at pack root
metadata:
  type: project
---

When a later BUNDLE-009 seed plan (e.g. PLAN-SPEC-038/Seed-4) ADDS rules to a shared
pack a PRIOR seed (PLAN-SPEC-037/Seed-3) OWNS, two things must be cross-checked against
the prior seed's PLAN, not just the current spec's prose:

1. **Pack location + layout must match the owner seed.** Seed-3 stands up the shared TS
   proof pack at `pkg/gate/testdata/ts-proof-pack/` with rules under `ast-grep/`. A later
   seed that authors into `.backstop/packs/backstop/ts-proof/` with `rules/` is writing
   into a pack that does not exist and violates the "SAME shared pack" claim (CLM-026).
   Always open the OWNER seed's plan and grep its actual pack file paths.

2. **Every pack dispatched through real ast-grep OR real grep needs its convert script(s)
   at its OWN pack root.** Dispatch resolves convert as `filepath.Join(packRoot, binding.Convert)`
   (cmd/backstop/pack_gate.go ~L413) and hard-errors `broken pack X: missing convert script`.
   The default ast-grep binding hardcodes `Convert: "ast-grep/to-sarif.sh"`
   (pkg/pack/engine/binding.go ~L241). So a pack with an ast-grep rule MUST ship
   `<pack>/ast-grep/to-sarif.sh`; a grep engine MUST ship `<pack>/grep/to-sarif.sh`.

**Why:** PLAN-SPEC-038's first review FAILed on exactly this: the installable `packs/contracts/`
pack correctly shipped both convert scripts, but the TWO TDD packs
(`.backstop/packs/backstop/traceability/` and the `ts-proof` pack) shipped only the grep
convert and NO ast-grep convert — and the ts-proof pack path contradicted Seed-3. This is
the same repeat-offender that FAILed PLAN-SPEC-037's first review. See
[[astgrep-pack-convert-script-scope]].

**How to apply:** For each seed plan that ADDS to a shared/prior pack: (a) grep the OWNER
seed's plan for the real pack path + dir layout and confirm congruence; (b) enumerate
every pack the plan dispatches through real ast-grep/grep and confirm each has BOTH its
ast-grep and/or grep `to-sarif.sh` in some task's file scope. The installable E2E pack
passing this check does NOT mean the TDD packs do.
