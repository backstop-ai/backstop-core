---
title: "Pack Resolution Hardcodes the GitHub Host"
schema_version: issue/v1

issue:
  id: ISSUE-083
  title: "Pack Resolution Hardcodes the GitHub Host"
  type: technical-debt
  status: open
  created: "2026-07-26"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# Pack Resolution Hardcodes the GitHub Host

## Problem

`resolveGitURL` in `pkg/pack/distribution/add.go:95-96` builds every remote pack's clone URL by
string-concatenating a fixed host:

```go
func resolveGitURL(packName string) string {
	return "https://github.com/" + packName + ".git"
}
```

This is the single choke point for remote pack resolution — every call site funnels through it:
`pkg/pack/distribution/command.go:146,418,533,648` (add/update/upgrade/relock-adjacent flows) and
`pkg/pack/distribution/versionresolver.go:70` (tag listing for version resolution). There is no
production code path that builds a git URL any other way.

Surfaced 2026-07-26 by the backlog-pm post-publication review, after ten backstop packs were
published to `github.com/backstop-ai/*` under the `name == coordinate` fleet convention (see
DIR-027) and SPEC-055 delivered the production `ExecGitCloner` this function feeds.

### Why this now matters beyond "it's a TODO"

Before SPEC-055, remote pack installation panicked unconditionally (ISSUE-073) — the hardcoded
host was inert because nothing using it actually worked yet. SPEC-055 made the git-clone path
real, and the fleet convention ratified afterward (DIR-027, ten packs live under
`backstop-ai/*`) made `<org>/<pack-name>` the load-bearing identity format for an installable
pack. Combined, `resolveGitURL`'s fixed host means **a pack name must resolve to a GitHub
`org/repo` pair, full stop** — there is no ref shape, config, or convention that lets a pack
live on GitLab, Bitbucket, a self-hosted git server, or anywhere else. A third party cannot
author an installable backstop pack unless they publish it to GitHub.

This is in tension with two things already on record:
- **BUNDLE-006 DD-31 / OQ-9** explicitly reasoned about this exact host-generality concern for
  the *lock recording* question: "no host-specific case normalization, because
  case-insensitivity is a GitHub property and packs may be hosted anywhere" (bundle line ~1154).
  That resolution covers how a resolved coordinate is *recorded* once known. It does not cover
  how the coordinate is *resolved into a URL* in the first place — `resolveGitURL` still assumes
  the host unconditionally, before any lock entry exists. The "packs may be hosted anywhere"
  premise the bundle already accepted for identity is violated by resolution.
- **The zero-baked-knowledge invariant** (CLAUDE.md: "Thin executor: ZERO baked language/tool
  knowledge") is framed around language/tool checks and toolchains, not infrastructure hosts —
  so this may not be a strict violation of that rule as scoped. But it is the same *shape* of
  defect: an ecosystem assumption (GitHub specifically, not "git") baked into core resolution
  logic rather than left to configuration or an explicit ref. Recorded here as a framing to
  weigh, not a settled verdict — see Solution.
- The project's own ICP (small teams, not all on GitHub — see founder positioning) is narrower
  than what this code path allows a *contributor* to target.

### Ownership gap

Neither existing directive owns this. **DIR-026** delivered remote consumption generally (cited
as the predecessor). **DIR-027** (Pack Fleet Publication & Migration) is scoped explicitly to
*this* fleet's publication under `github.com/backstop-ai` and fleet-internal migration — it does
not address whether *other* hosts can be resolved at all, and its four threads (extract vendored
packs, reconcile harness packs, migrate fleet lock entries, absorb Clone-strip asymmetry) are
silent on host-generality. This issue is unhomed; it needs a backlog-pm pass to decide whether it
folds into a future directive or stands alone.

## Solution

Not decided — recording direction options for a human/PM call, not prescribing one:

1. **Full URL refs alongside the shorthand.** `backstop pack add https://gitlab.com/org/pack@1.0.0`
   (or `git@host:org/pack.git@1.0.0`) accepted as an alternative ref shape to the current
   `org/pack-name@version` shorthand, detected by prefix (`https://`, `git@`, etc.) before falling
   through to the GitHub-shorthand resolver. Keeps the common case (GitHub, `name == coordinate`)
   unchanged.
2. **Host-prefix convention in the ref itself**, e.g. `gitlab:org/pack-name@version`, parsed
   alongside the bare `org/pack-name` (implicitly GitHub) form.
3. **A resolver configured in `backstop.yml`** — e.g. a `pack_host:` or per-org host mapping —
   so a project (or org) can declare its default git host once rather than repeating full URLs
   per pack ref.

Any option needs to reconcile with BUNDLE-006 DD-31 (coordinate recorded verbatim, no
host-specific normalization) and with `versionresolver.go:70`'s tag-listing call, which resolves
the same URL for version discovery before a lock entry exists.

## References

- `pkg/pack/distribution/add.go:95-96` — `resolveGitURL`, the hardcoded-host definition
- `pkg/pack/distribution/command.go:146,418,533,648` — call sites (add/update/upgrade/relock-adjacent)
- `pkg/pack/distribution/versionresolver.go:70` — call site for tag listing / version resolution
- `pkg/pack/distribution/gitcloner.go` — `ExecGitCloner`, the production cloner SPEC-055 wired in,
  which consumes whatever URL `resolveGitURL` hands it
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` — DD-31 / OQ-9 (resolved): source
  coordinate vs. runtime identity, "packs may be hosted anywhere" premise for lock recording
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` — delivered the production
  git-clone mechanism this defect rides on
- `directives/DIR-026-remote-dependency-assembly.directive.md` — predecessor, remote consumption
  generally
- `directives/DIR-027-pack-fleet-publication-migration.directive.md` — explicitly scoped to the
  `backstop-ai` fleet's publication/migration; does not own host-generality
- `issues/ISSUE-073-pack-add-nil-git-cloner-panic.issue.md` — predecessor defect on the same
  `resolveGitURL` function, retired to SPEC-055 (the panic that made this host assumption inert
  until SPEC-055 shipped)
- CLAUDE.md — zero-baked-knowledge invariant (scoped to language/tool checks; this issue records
  the tension without claiming a direct violation)
