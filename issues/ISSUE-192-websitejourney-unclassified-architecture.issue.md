---
title: "scripts/websitejourney Production Go Files Unclassified by Architecture Pack"
schema_version: issue/v1

issue:
  id: ISSUE-192
  title: "scripts/websitejourney Production Go Files Unclassified by Architecture Pack"
  type: policy-violation
  status: open
  created: "2026-08-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# scripts/websitejourney Production Go Files Unclassified by Architecture Pack

## Problem

PLAN-SPEC-076 places Seed 5's integration consumer at `scripts/websitejourney/`
(`artifacts.go`, `main.go`, `map.go`, `types.go`). That is the same repository-local
pattern Seeds 3 and 4 used for `scripts/producttruth`, `scripts/sitecheck`, and
`scripts/render-public-site-contracts`.

`backstop-ai/backstop-core-architecture` v0.1.6 classifies those three script
packages in `architecture/backstop-core.yml` and isolates them from released Core
packages (`anyVendorDeps: true`, no project imports). It has no `websitejourney`
component. `go-arch-lint` therefore emits `unclassified-package` for every
production `.go` file under `scripts/websitejourney/`.

A file-scoped gate of `docs/_data/website-capability-map.yml` (PLAN-SPEC-076
TASK-009) currently fails with four blocking findings:

- `scripts/websitejourney/artifacts.go`
- `scripts/websitejourney/main.go`
- `scripts/websitejourney/map.go`
- `scripts/websitejourney/types.go`

`*_test.go` files are excluded by the pack and are not in this set.

### Why this is not a Core CLI change

Packs are external. The lock file is the durability boundary. Core must not
vendor the architecture pack, bake a websitejourney exception into the CLI, or
move Seed 5 code into `scripts/sitecheck` / `scripts/producttruth` to borrow
those components. Those homes belong to Seeds 4 and 3.

## Solution

Release a new `backstop-ai/backstop-core-architecture` version that adds a
`websitejourney` component:

```yaml
websitejourney: { in: "scripts/websitejourney" }
```

with the same isolation policy as `sitecheck` / `product_truth` /
`site_contracts`:

```yaml
websitejourney: { anyVendorDeps: true }
```

Then update `backstop.lock` in this repository to that pack version and
reinstall. Do not edit gitignored `.backstop/packs/` as a substitute for a
pack release.

### Scope constraint

This issue owns only architecture-pack classification of
`scripts/websitejourney`. It does not own Seed 4 delivery-inventory recognition
of new `capabilities/CAP-00[4-9]*` files, inner-page visual redesign, or
multi-repo Cloud Agent checkout.

## References

- PLAN-SPEC-076 TASK-034 / TASK-035 / TASK-009 file lists
- `.backstop/packs/backstop-ai/backstop-core-architecture/architecture/backstop-core.yml`
  components `product_truth`, `sitecheck`, `site_contracts` (precedent)
- PR #31 `cursor/website-seed5-capabilities-6ced`

### Existence-in-world check

`issues/` has no open issue covering `websitejourney` or `unclassified-package`
for Seed 5. `bundles/BUNDLE-032` owns website capability journeys, not the
architecture pack's component map. Closest precedent is the Seed 3/4 pack
releases that added `product_truth` and `sitecheck` components.
