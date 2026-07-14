---
title: "Bundle Spec Promotion Gate Check"
schema_version: issue/v1

issue:
  id: ISSUE-057
  title: "Bundle Spec Promotion Gate Check"
  type: technical-debt
  status: open
  created: "2026-07-14"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-057: Bundle→Spec Promotion Gate Check

## Problem

Nothing enforces that a spec's parent bundle is promoted (`defined`+) before
specs are authored against it. Verified 2026-07-14: `pkg/validate/spec.go`
only **format-checks** the `supports` reference on a spec requirement — it
confirms the string matches `bundle-name:REQ-NNN` (via `supportsRe`), it never
resolves the referenced bundle and checks its `maturity`:

```go
// pkg/validate/spec.go:487-496
for _, sup := range supItems {
    if strings.TrimSpace(sup) == "" || !supportsRe.MatchString(sup) {
        result.violations = append(result.violations, Violation{
            Rule:     "spec/requirement-supports-format",
            ...
            Message: fmt.Sprintf("requirements[%d] 'supports' value '%s' must "+
                "match format bundle-name:REQ-NNN", i, sup),
            Severity: "error",
        })
    }
}
```

A `grep` for `maturity` across `pkg/validate` confirms it: the only files that
reference bundle maturity at all are `pkg/validate/bundle.go` (a bundle
validating itself) and `pkg/validate/terminal.go`. No spec-side check, and no
`pkg/gate` check, ever resolves a spec's parent bundle and asks whether it has
been promoted. The `.claude/hooks/backstop-agent-guard.sh` hook governs WHO can
write which artifact type by filename extension (`bundle-author` → `*.bundle.md`,
etc.) — it says nothing about ordering BETWEEN artifact types.

This is not hypothetical: it already produced a fossil. A 2026-05-30
auto-dispatch run machine-generated 9 specs (`SPEC-020`..`SPEC-029`) and 9
plans against `BUNDLE-003`, which was — and still is — `exploring` with open
questions at the time. The front half of the bundle→spec track (resolve OQs →
promote → then spec, per `feedback_artifact_tracks` /
`feedback_bundle_workflow`) was skipped entirely and nothing objected. Those
artifacts were never committed and were purged in BUNDLE-003's 0.5.0
from-scratch rewrite (2026-07-13) — see BUNDLE-003 §"Out of Scope /
Dependencies" ("Bundle→spec promotion gate check — an orthogonal
workflow-integrity hole … this bundle's own legacy SPEC-020..029 were
auto-generated against an unpromoted parent") and §Version History 0.5.0. The
mechanism that let that happen is still unfixed.

This is the "prompts are vibes" gap named in CLAUDE.md and
`project_prompts_are_vibes`: the bundle→spec ordering lives only in prose
(CLAUDE.md, agent memory, slash-command instructions) — no enforced code
stops an agent (or an auto-dispatch run) from authoring a spec against an
unpromoted, still-`exploring` bundle.

## Solution

Not committed — left open for the plan. Direction:

1. Add a validation or gate check — dogfooded as a pack rule per
   `feedback_dogfood_rules_as_packs` (never baked directly into the CLI
   binary) where that is a natural fit, or as a `pkg/validate/spec.go` check
   if the artifact-validation layer is the right enforcement point — that
   resolves each spec's `supports` bundle reference to its bundle file and
   confirms `bundle.maturity` is `defined` or later. A spec whose parent
   bundle is still `exploring` (or otherwise unpromoted) is a **workflow
   violation**.
2. Follow the same loud-vs-blocking philosophy the gate already uses
   (`feedback_loud_not_blocking`): this is a broken promise/workflow-ordering
   defect, so it should land on the blocking side, not a warn.
3. Decide where in the pipeline this check belongs — spec `validate`, `gate`,
   or both — and whether it needs a bundle-side complement (e.g. does
   promoting a bundle need to re-check specs authored against it, or is
   spec-side-only sufficient since specs are only ever created after the
   bundle exists).
4. Add a mandated test that authors a spec against a deliberately
   `exploring` bundle fixture and confirms the new check fires.
5. Priority note from the founder (2026-07-14): this is explicitly **LOW /
   non-urgent** — the current working style (agent-driven, one bundle/spec at
   a time, no auto-dispatch in active use) doesn't exercise this gap day to
   day. It's worth formalizing so the fossil-making mechanism is closed, not
   because it's actively burning anyone right now.

## References

- Discovered 2026-07-14 during the BUNDLE-003 OQ-resolution session, while
  auditing the bundle's own "Out of Scope / Dependencies" note about its own
  legacy SPEC-020..029
- BUNDLE-003 §"Out of Scope / Dependencies" — "Bundle→spec promotion gate
  check — an orthogonal workflow-integrity hole: a spec whose parent bundle is
  not promoted should be a violation but currently is not enforced"
- BUNDLE-003 §Version History, 0.5.0 (2026-07-13) — the purge of the
  never-committed SPEC-020..029 / 9 plans, the concrete incident this issue
  formalizes a fix for
- `pkg/validate/spec.go` — `supportsRe` / the `supports`-format check; confirmed
  format-only, no bundle-maturity resolution
- `.claude/hooks/backstop-agent-guard.sh` — confirmed scope: who-writes-what,
  not artifact-to-artifact workflow ordering
- `project_prompts_are_vibes` (agent memory) — foundational: prompts suggest,
  only executed code constrains; this issue is exactly a load-bearing rule
  that currently lives only in prompts
- `feedback_artifact_tracks`, `feedback_bundle_workflow` (agent memory) — the
  issue→plan / bundle→spec→plan→implementation track rules this check would
  make structurally enforced instead of prose-only
- `feedback_dogfood_rules_as_packs`, `feedback_loud_not_blocking` (agent
  memory) — the enforcement-philosophy constraints the plan should follow
