---
name: zero-baked-violations-have-no-home
description: Zero-baked-language/tool/platform violations have NO standing directive — DIR-014 is done — so home them by the SURFACE they live on, not by the invariant they break
metadata:
  type: project
---

New issues alleging a baked language/tool/**platform** assumption in core have no
invariant-owning directive to fall into: `DIR-014 "Complete Thin Executor Eradication"`
is `status: done` (completed 2026-07-06) and is not in BACKLOG.yml.

**Why:** the eradication arc was declared finished, but the invariant is standing law, so
new instances keep surfacing (ISSUE-120, 2026-08-11: `cmd/backstop/baseline.go`'s
`baseline pull` shells out to `gh auth status` and queries the GitHub Actions API). The
invariant outlived its directive.

**How to apply:** home these by the SURFACE that owns the offending code, not by the law
they break. ISSUE-120 → `DIR-003 "Baseline Implementation"` because DIR-003's own
Description names both "`backstop baseline pull` command" and "GitHub Actions artifact
publishing" — the directive is where the assumption under challenge is *recorded*, which
makes it a clean single-charter fit. Reflexes to resist: `DIR-024 "Gate/Engine Quality"`
looks adjacent to everything but collects gate-**step** defects and is an overloaded
catch-all; `DIR-019 "Pack Recipe Capability"` is usually the *discovery context* (its
platform-plural CI recipes surface these), not the home.

Two things worth carrying into any such triage:
- **In-repo migration precedent exists** — pack distribution already removed a GitHub host
  assumption under SPEC-056 DD-31 (`pkg/pack/distribution/identity.go:216` names it). Hand
  that to the planner instead of letting them re-derive the shape.
- **Grep the whole tree before claiming "the last instance."** A single-file issue is often
  narrower than reality; record the weaker siblings as explicitly OUT of scope so they
  neither get lost nor scope-creep the fix.

If a third or fourth such issue lands with no natural surface owner, that is the signal to
recommend a successor directive to DIR-014 — do not create one unilaterally.

Related: [[project_workaround_and_file_pattern]], [[project_mechanism_vs_ecosystem_gap]],
[[project_concurrent_pm_triage_races]].
