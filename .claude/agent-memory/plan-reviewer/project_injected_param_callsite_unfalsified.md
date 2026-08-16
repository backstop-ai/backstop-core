---
name: injected-param-callsite-unfalsified
description: A claim asserting "every production call site populates parameter X from loaded packs/config" is NOT falsified by walk-level or merge-level tests — demand one test that drives a real command end to end
metadata:
  type: project
---

When a plan de-bakes a literal by turning it into an INJECTED PARAMETER, the claim that
covers the wiring ("every production call site — the gate step, traceability, `artifact
validate`, `doctor` — populates it from the packs already loaded in that command")
routinely ships with zero falsifying tests. The plan's tests prove (a) the merge helper
unions pack declarations, and (b) the walk honors a hand-built set — never that any
production call site actually passes a populated set.

**Why:** an implementer who threads the ZERO VALUE at every production site keeps every
test in the plan green. In backstop-core specifically the regression is invisible
locally — this repo has no `vendor/`/`node_modules/` tree — so it only breaks downstream
consumers. Even an "E2E" task can miss it: driving `loadInstalledPacks +
mergeDependencyDirs + DiscoverArtifacts` from a test proves the merge and the walk, but
still bypasses `artifact_validate.go`'s CLI construction, `gate.go`'s validator/
traceability hops, and `doctor_checks.go`. (PLAN-ISSUE-122 CLM-005, found round 4.)

**How to apply:** whenever a claim names call sites, ask "which test goes RED if all of
them pass the zero value?" If the answer is none, it is a blocker — require one
wiring-level test that invokes a real command (doctor check, `runArtifactValidate`, or
the gate step) over a fixture with a declaring pack and a planted file in the excluded
tree. Also note a documented DEGRADED path (e.g. `ctx.PacksErr != nil` → zero value) is
usually fine to leave test-free on its consequence, but the argument "there is no
alternative branch to test" is unsound — tests pin wiring against regression, and the
plan itself typically forbids an alternative ("do not default the names back in") that
nothing would catch. Same family as [[project_reconciliation_swap_enable_wiring]] and
[[project_dispatch_consumer_edges]].
