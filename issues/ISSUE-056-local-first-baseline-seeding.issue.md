---
title: "Local First Baseline Seeding"
schema_version: issue/v1

issue:
  id: ISSUE-056
  title: "Local First Baseline Seeding"
  type: technical-debt
  status: open
  created: "2026-07-14"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-056: Local-First Baseline Seeding

## Problem

On a fresh repo with no `origin` remote, `baseline_comparison` is dark and
suggests a fix that cannot work for the exact user hitting it.

`pkg/gate/gate.go:182-188` (`computeBaselineResult`) skips the step with:

```go
reason := "baseline unavailable: no cached baseline found at .backstop/baseline.json; " +
    "run CI baseline publication or backstop baseline pull"
```

But `backstop baseline pull` (`cmd/backstop/baseline.go:147-158`,
`resolveRepositoryFromOrigin`) itself requires an `origin` remote to resolve a
GitHub repository, and fails loud when there isn't one:

```go
cmd := exec.Command("git", "remote", "get-url", "origin")
...
return "", fmt.Errorf("missing origin remote; cannot resolve GitHub repository")
```

So the skip message tells a remoteless user to run the one command that fails
for the identical missing-remote reason — self-contradictory, dead-end
guidance.

BUNDLE-007 settled that the baseline is a **CI-generated post-merge artifact**
(DD-10: cached locally at `.backstop/baseline.json`, gitignored, pulled by the
gate, ratchet-only). That design assumes a remote + CI exists. But the primary
user is **solo and local-first**, and is frequently remoteless in practice —
`backstop-core` itself has no `origin` remote yet (per `project_launch_plan`:
"no remote yet by design; cleanup → tree audit → big-bang squash to public
remote"). The result: the ratchet — a core value proposition of the whole
gate/baseline architecture (`project_gate_hardening_arc`,
`project_baseline_design`) — is dark for exactly the person who is using the
tool the most.

## Solution

Not committed — left open for the plan. Direction settled by BUNDLE-003 OQ-3
(RESOLVED, 2026-07-13):

1. `backstop init` seeds a **GITIGNORED LOCAL baseline** day-zero so the
   ratchet is live locally from the first `gate` run, with no remote required.
2. The **CI-generated** baseline (BUNDLE-007 / DIR-003) remains the LATER team
   upgrade — it adds tamper-resistance and avoids concurrent-write stomping
   that a purely local baseline can't provide, but it is an upgrade, not a
   prerequisite.
3. The local seed is a **regenerable, gitignored cache** — never committed,
   never hand-edited — exactly preserving what BUNDLE-007 established for the
   CI-generated baseline (DD-10), just with a second, local-first source.
4. Two baseline sources going forward: **local-seed** (solo / day-0, generated
   by `init` or an equivalent local command) and **CI-generated** (team /
   durable, per BUNDLE-007). The gate should prefer the CI-generated baseline
   when one is available, falling back to the local seed otherwise — this
   preference rule needs to be nailed down by the plan.
5. Fix the self-contradictory remoteless message in
   `pkg/gate/gate.go:184` — it should not recommend `backstop baseline pull`
   when there is no `origin` remote to pull against; it should point at local
   seeding instead (or at generating one) once that capability exists.
6. This issue's fix reflects back into BUNDLE-007 / DIR-003 as a new
   day-zero-local baseline mode — it is not a fresh baseline design, it is an
   additional source feeding the same `CompareBaseline` chokepoint
   (`project_ratchet_default_on`).

## References

- BUNDLE-003 OQ-3 (RESOLVED, 2026-07-13) — the source of this issue's
  direction: gitignored local baseline seeded at init; CI-generated baseline
  is the later team upgrade; fix the contradictory remoteless message
- `pkg/gate/gate.go:182-188` (`computeBaselineResult`) — the self-contradictory
  skip message this issue fixes
- `cmd/backstop/baseline.go:147-158` (`resolveRepositoryFromOrigin`) — the
  `backstop baseline pull` remote requirement that makes the current message
  dead-end guidance
- BUNDLE-007 (baseline) — DD-10, the CI-generated / gitignored / ratchet-only
  baseline design this issue extends with a local-first source, not replaces
- DIR-003 (Baseline CI-gen + pull) — the CI half of the baseline mechanism;
  `project_baseline_ci_pull` in agent memory
- `project_baseline_design` (agent memory) — fingerprint built, DIR-018
  ratchet-honesty delivered; CLEAN baseline + CI-pull still designed-not-built;
  this issue adds the local-seed source to that still-open design
- `project_ratchet_default_on` (agent memory) — ISSUE-050's `CompareBaseline`
  chokepoint; the local-seed source should feed the same chokepoint, not a
  parallel mechanism
- `project_launch_plan` (agent memory) — "no remote yet by design"; the
  concrete reason `backstop-core` itself is a live instance of this gap
