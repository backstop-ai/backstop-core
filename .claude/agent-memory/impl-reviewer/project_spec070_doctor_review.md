---
name: spec070-doctor-review
description: SPEC-070 backstop doctor — PASS after one fix round; the defect worth remembering is that a nil XErr is not proof a conditional load ran
metadata:
  type: project
---

SPEC-070 (`backstop doctor`, BUNDLE-003 Seed 3) reviewed 2026-08-15. FAIL on round 1
(one substantive defect + two smaller), **PASS/CLEAN on round 2** — all three fixes
verified by reproduction, gate exit 0, whole-repo `go test ./...` clean. The test
corpus is the strongest reviewed on this project so far.

**The defect worth remembering as a PATTERN:** `checkToolchainRuns` keys its
"pack set could not be gathered" skip on `ctx.PacksErr != nil` alone. But
`gatherDoctorContext` only CALLS `loadInstalledPacks` when the config both discovered
and loaded — so on a no-config or unloadable-config project the pack set is
NEVER GATHERED and `PacksErr` is nil, indistinguishable from "gathered, declares
none". Result: doctor asserts "no installed pack declares a test or build
entrypoint" on a project whose packs were never consulted. `checkPacksInstalled`
does NOT have the bug because it reads `ConfigPathErr`/`ConfigErr` explicitly first.

**Why:** a nil error is NOT proof a load succeeded when the load is conditional.
Whenever a context struct carries `X`/`XErr` pairs populated behind a guard, the
"was it gathered" signal needs its own representation, not `XErr == nil`.

**How to apply:** on any spec with a gather-once context + per-check skip rules,
enumerate the gather guards and check every consumer's skip predicate against ALL
of them. Also: the spec's own **Review Questions** section caught this and the
mandated-test set did not — RQ9 stated the exact expected matrix
("`toolchain-runs` must be `skipped`" on a no-`backstop.yml` directory) with no
claim or mandated test behind it. Always evaluate Review Questions by RUNNING the
scenario, not by reading code.

Secondary findings: init's `doctorGuidanceForSteps` prints the identical guidance
line once per failing toolchain StepReport (N packs failing = N identical lines);
CLM-051's stack-policy source scan does `strings.Contains(literal, "lts")`, which
false-positives on ordinary words like "results"/"defaults" — latent, not yet red.

Reviewer-side technique that paid off here: four implementer-claimed falsifications
were all reproduced in an rsync'd scratch copy and all held EXACTLY as described
(skip-branch removal, layout containment reduction reddening 6/13 with 7 green,
RunStdout capture, storage-time indentation). See [[project_issue116_line_carry_pass]].

Environmental trap on this machine: a CONCURRENT agent's gitignored
`.claude/worktrees/agent-<id>/` full-repo copy makes `go test ./...` red
(`TestModulePath_NoLegacyReferencesInLiveTree`) and floods `backstop gate` with
hundreds of ungated-artifact + pack_engines violations that look like new
regressions. Attribute by path before believing them. Also `go-arch-lint` lives in
`~/go/bin`, which is NOT on the default PATH — without it the gate exits 2 early.
