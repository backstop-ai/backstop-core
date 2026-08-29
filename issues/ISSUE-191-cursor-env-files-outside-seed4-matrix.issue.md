---
title: ".cursor/ Cloud Agent Environment Files Outside Closed Seed 4 Delivery Matrix"
schema_version: issue/v1

issue:
  id: ISSUE-191
  title: ".cursor/ Cloud Agent Environment Files Outside Closed Seed 4 Delivery Matrix"
  type: policy-violation
  status: open
  created: "2026-08-29"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# .cursor/ Cloud Agent Environment Files Outside Closed Seed 4 Delivery Matrix

## Problem

PR #30 (`cursor/cloud-agent-env-setup-daae`) introduces three Cloud Agent
environment files to the repository:

- `.cursor/Dockerfile`
- `.cursor/environment.json`
- `.cursor/install.sh`

`scripts/sitecheck/inventory.go` enforces a **closed** role/path matrix for
every file that appears in `git diff --name-status <base_commit>...HEAD`
(where `base_commit` is `89f4138aa97e7ba1e8d7c67595cfaee1caefa797`, declared
in `.backstop/seed4-delivery-inventory.yml`). The diff is cumulative from that
fixed historical commit, so any file added to `main` since that commit must be
declared in the inventory.

`expectedRole()` in `scripts/sitecheck/inventory.go` returns `""` for any path
not explicitly matched by its `switch` cases. `validatePathRole()` then returns
`"path is outside the closed Seed 4 matrix"` for those paths, and
`validateInventoryMatchesDiff()` fails with `"inventory differs from git diff"`,
listing the observed-but-not-expected files.

No `.cursor/**` prefix case exists in `expectedRole()`, and no `agent-environment`
(or equivalent) role is defined in `allowedRoles()`. The three `.cursor/*` files
therefore have no legal role assignment and cannot be declared in the inventory.

### Failing check

`./scripts/verify-public-site.sh` (run by both
`.github/workflows/site-verification.yml` on pull requests and
`.github/workflows/pages.yml` on push to `main`) fails at the
`public-site[structure]` step:

```
sitecheck: delivery inventory: inventory differs from git diff
expected:
  ...
observed (extra):
  A	.cursor/Dockerfile
  A	.cursor/environment.json
  A	.cursor/install.sh
```

### Consequence

- PR #30's `Build and verify public site` check is red, blocking merge.
- If forced to `main`, the same check in the Pages deploy workflow would fail on
  every subsequent push to `main` and on every future PR until the files are
  declared, breaking the `pages.yml` deploy pipeline.
- The `Backstop Gate` required check is unaffected; this is purely the
  inventory-contract enforcement.

### Why the current matrix has no path for `.cursor/**`

The closed-matrix philosophy in `expectedRole()` requires every Seed 4 file to
be affirmatively mapped to a role. Infrastructure configuration files outside the
public-site delivery (like Cloud Agent environment definitions) were simply never
contemplated as a Seed 4 delivery surface when the matrix was authored. The
matrix has no catch-all; an absent case is a hard rejection, not a quiet skip.

## Solution

Make Cloud Agent environment files under `.cursor/` first-class rows in the
closed Seed 4 delivery matrix by following the exact precedent established in
commit `cd453e8` (PR #29, "Restore canonical homepage direction"), which solved
the same class of problem for governance artifacts:

1. **Add `agent-environment` to `allowedRoles()`** in
   `scripts/sitecheck/inventory.go`. Using a dedicated role (rather than
   reusing `governance-artifact`) keeps the semantic intent legible:
   agent-environment files are infrastructure configuration, not issue/plan
   governance artifacts.

2. **Add a `.cursor/**` prefix case to `expectedRole()`** returning
   `"agent-environment"` for any `strings.HasPrefix(path, ".cursor/")` path.

3. **Declare the three current files in `.backstop/seed4-delivery-inventory.yml`**:
   - `change: A`, `path: .cursor/Dockerfile`, `role: agent-environment`
   - `change: A`, `path: .cursor/environment.json`, `role: agent-environment`
   - `change: A`, `path: .cursor/install.sh`, `role: agent-environment`

4. **Add a focused test** in `scripts/sitecheck/inventory_test.go` — analogous to
   `TestDeliveryInventory_ISSUE190GovernanceArtifactsAreClosedRows` — asserting
   that representative `.cursor/` paths return `"agent-environment"` from
   `expectedRole()` and that all three inventory entries are present with the
   correct role. This prevents silent regression back to an unrecognized path.

### Scope constraint

The fix is limited to recognition of `.cursor/**` within the closed matrix. It
does not broaden or weaken the closed-matrix philosophy, does not affect any
other role or path mapping, and does not touch the public-site delivery content.

## References

- PR #30: `cursor/cloud-agent-env-setup-daae` — the branch introducing the
  `.cursor/` files that trigger this violation.
- `scripts/sitecheck/inventory.go` — `expectedRole()` switch (no `.cursor/`
  case), `validatePathRole()` closed-matrix rejection, `allowedRoles()` set
  (no `agent-environment` entry).
- `.backstop/seed4-delivery-inventory.yml` — `base_commit:
  89f4138aa97e7ba1e8d7c67595cfaee1caefa797`; no `.cursor/*` entries.
- `.github/workflows/site-verification.yml` and `.github/workflows/pages.yml` —
  both run `./scripts/verify-public-site.sh`, both affected.
- Commit `cd453e8` (PR #29, "Restore canonical homepage direction") — precedent:
  added `governance-artifact` to `allowedRoles()`, mapped two specific paths to
  it in `expectedRole()`, declared them in the inventory, and added
  `TestDeliveryInventory_ISSUE190GovernanceArtifactsAreClosedRows` in
  `scripts/sitecheck/inventory_test.go`.

### Existence-in-world check

Before filing, `issues/` was searched for matches on `inventory`, `cursor`,
`site-verif`, `seed4`, `delivery`, `agent-environment`, and `.cursor/`. No
existing open issue covers this surface. `bundles/` was searched for the same
terms; none of the matching bundles (BUNDLE-003, -008, -009, -013, -014, -032)
own the closed delivery-matrix recognition of `.cursor/**`. The closest existing
issue is ISSUE-190, which solved the identical class of problem for governance
artifacts and supplies the direct fix precedent; it does not cover agent
environment files.
