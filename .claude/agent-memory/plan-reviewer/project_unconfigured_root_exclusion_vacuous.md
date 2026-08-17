---
name: unconfigured-root-exclusion-vacuous
description: A filter that excludes "paths under the resolved artifact root" is vacuous-EVERYTHING in backstop-core, because backstop.yml declares no artifact_root and ResolveRoot then returns the PROJECT root
metadata:
  type: project
---

When a plan says "exclude the config-declared artifact root (never a hardcoded
directory list)", check what `artifact.ResolveRoot(projectRoot, declared)`
actually returns for THIS repo before believing the filter is precise.

`backstop.yml` in backstop-core declares NO `artifact_root`. `ResolveRoot` with
an empty `declared` returns `Root{Path: absProject, Configured: false}` — "the
framework exception that keeps a repo-root layout working without configuring
anything" (`pkg/artifact/layout.go`). So "drop every path under
`artifactRoot.Path`" drops EVERY FILE IN THE PROJECT, and any check built on it
reports zero findings forever.

The correct exclusion is the per-KIND directories: `Root.Dir(KindSpec)`,
`Root.Dir(KindPlan)`, `Root.Dir(KindIssue)`, `Root.Dir(KindBundle)`,
`Root.Dir(KindADR)`, `Root.Dir(KindDirective)`, `Root.Dir(KindCapability)` —
`Dir` joins the root to the bare directory name and is "the only sanctioned way
to name an artifact type directory".

**Why:** caught on PLAN-ISSUE-097 (2026-08-17). The plan's unit fixture built a
project with a CONFIGURED artifact root, so its mandated exclusion test would
have passed green while the production path in backstop-core harvested nothing.
Unit-green + production-vacuous is exactly the failure class the plan existed to
close.

**SECOND AXIS — PATH FORM (verified 2026-08-17).** `Root.Path` is
`filepath.Abs(projectRoot)`, so `Root.Dir(kind)` is ABSOLUTE. But
`gate.ComputeGateScope(..., GateScopeModeAll, nil)` → `resolveGateScopeAll`
returns `filepath.Rel(projectRoot, path)` — RELATIVE. A naive
`strings.HasPrefix(scopeFile, root.Dir(kind))` therefore matches NOTHING and the
exclusion silently no-ops. Two independent ways to make the same filter wrong:
`Root.Path` (drops everything) and absolute-vs-relative (drops nothing). Check
both. (`excludeTestdataPaths` is immune — it splits on `/` and compares
segments.)

**How to apply:** any plan that filters, scopes, or excludes by "the artifact
root" — demand the per-kind `Root.Dir(...)` form AND a fixture leg with an
UNCONFIGURED root (root == project root), or it is a blocker. Also confirm the
plan's mandated test would go red on a no-op filter, since the path-form slip is
invisible to reasoning. Related:
[[project_inert_decoy_fixtures_vacuous]], [[project_new_guard_predicate_measure_existing_fixtures]].
