---
title: "Remote (Git) Pack Resolution Is Wired to a Nil GitCloner — `pack add org/pack@version` Panics"
schema_version: issue/v1

issue:
  id: ISSUE-073
  title: "Remote (Git) Pack Resolution Is Wired to a Nil GitCloner — `pack add org/pack@version` Panics"
  type: bug
  status: replaced
  created: "2026-07-25"
  closed: "2026-07-26"

replaced-by: SPEC-055

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical
---

# Remote (Git) Pack Resolution Is Wired to a Nil GitCloner — `pack add org/pack@version` Panics

## Problem

`backstop pack add org/pack-name@version` (any non-local pack ref) panics with a nil-interface
dereference instead of cloning the pack. The entire remote/git pack distribution path — the
publish path for open-sourcing backstop — has no production `GitCloner` wired in and has never
run outside of mocked tests.

Discovered 2026-07-25 while dogfooding backstop from an external consumer project (`stash`) and
its new enforcement pack (`backstop/cobra-cli`, a git-hosted pack). Documented in
`docs/CODEBASE-MAP.md` under a "Known gap" heading.

### Root cause chain (verified against source)

1. `cmd/backstop/pack_add.go:31-34` constructs the options passed to `distribution.Add`:

   ```go
   opts := distribution.AddOptions{
       ProjectDir: ".",
       Version:    versionFlag,
   }
   ```

   No `GitCloner` and no `Validator` are set — both are left as their zero value (`nil`).

2. `pkg/pack/distribution/add.go` `Add()`: for a non-local ref (`org/pack-name@version`, parsed
   by `parsePackRef`, `add.go:292-300`), it builds a URL via `resolveGitURL` (`add.go:318-320`,
   hardcoded to `https://github.com/<name>.git`) and then calls:

   ```go
   gitURL := resolveGitURL(packName)
   if err := opts.GitCloner.Clone(gitURL, "v"+version, tmpDir); err != nil {
   ```

   (`add.go:153`). `opts.GitCloner` is `nil`, so `.Clone(...)` is a nil-interface method call —
   this panics for ANY git pack ref, unconditionally.

3. Even if the panic didn't happen first, validation is silently skipped for the same reason:
   `add.go` only runs `opts.Validator.RunPackCheck`/`RunPackTest` inside `if opts.Validator !=
   nil` — a git pack that somehow got cloned would install unvalidated.

4. `pkg/pack/distribution/install.go:167` at least nil-guards this case
   (`"no git cloner provided for pack %s"`), so `pack install` of a git pack fails cleanly with an
   error rather than panicking — inconsistent with `pack add`, and evidence that the nil-wiring
   gap is known/anticipated in one code path but not closed in the sibling path.

5. `pkg/pack/distribution/update.go:87` and `pkg/pack/distribution/upgrade.go:59` also call
   `opts.GitCloner.Clone(...)` with the same missing-wiring exposure — `pack update` and `pack
   upgrade` of a git pack will panic the same way `pack add` does.

6. The `GitCloner` interface (`add.go:13-16`, methods `Clone` + `ListTags`) has **no production
   implementation anywhere in the repo**. `pkg/scaffold/git_executor_real.go`'s
   `RealGitExecutor` implements `ListTags` (tag operations) but not `Clone`, and is never
   constructed as a `distribution.GitCloner` from any `cmd/backstop/*.go` entry point. Every
   `GitCloner` reference in the codebase outside interface/option definitions is a `mockGitCloner`
   in `*_test.go` files (`add_test.go`, `install_test.go`, `update_test.go`, `upgrade_test.go`,
   `install_materialize_test.go`).

7. All packs installed in real projects to date are `source_type: local`, so the git-clone path
   has never executed in production — only under test mocks.

### Impact

Blocks remote pack distribution entirely. Since remote (git) pack installation is the intended
publish path for open-sourcing backstop, this is a hard blocker on that plan: any external
consumer following the documented `org/pack-name@version` syntax hits a panic, not a config
error.

### Expected

`backstop pack add org/pack@version` clones the tagged ref, validates it (`pack check` + `pack
test`), installs, hashes, and locks it — the same tail as local packs. A failure anywhere in that
chain (bad URL, missing tag, network failure, missing production cloner) should be a clean exit-2
config/tool error, never a panic. `pack update` and `pack upgrade` need the same fix since they
share the nil-`GitCloner` exposure.

## References

- `cmd/backstop/pack_add.go:31-34` — constructs `AddOptions` with no `GitCloner`/`Validator`
- `pkg/pack/distribution/add.go:13-16` — `GitCloner` interface definition
- `pkg/pack/distribution/add.go:153` — nil-interface `.Clone()` call, the panic site
- `pkg/pack/distribution/add.go:160` — validator skip guarded only by `!= nil`
- `pkg/pack/distribution/add.go:292-300` — `parsePackRef`
- `pkg/pack/distribution/add.go:318-320` — `resolveGitURL` (hardcoded GitHub URL)
- `pkg/pack/distribution/install.go:167` — sibling path's nil-guard (`"no git cloner provided for
  pack %s"`), the pattern `pack add` is missing
- `pkg/pack/distribution/update.go:87`, `pkg/pack/distribution/upgrade.go:59` — same nil-`GitCloner`
  exposure in the sibling commands
- `pkg/scaffold/git_executor_real.go` — has `ListTags` but no `Clone`; not wired as a
  `distribution.GitCloner` anywhere
- `docs/CODEBASE-MAP.md` — "Known gap" heading documenting this
- Discovered dogfooding from `~/src/projects/stash` (consumer project) attempting to add
  `~/src/projects/backstop-cobra-cli-pack` as a remote pack
- `plans/PLAN-ISSUE-073-production-git-cloner-remote-pack-install.plan.yml` — the reactive-track
  plan this issue produced, itself retired (`status: replaced`, `replaced-by: SPEC-055`)
- `bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` — the feature-track bundle that
  absorbed this surface
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` — the successor spec that owns
  the fix
- `plans/PLAN-SPEC-055-production-remote-dependency-assembly.plan.yml` — the successor's
  implementation plan (52 tasks / 12 phases, commit `856400c`)
- `directives/DIR-026-remote-dependency-assembly.directive.md` — authored in parallel to own this
  workstream in the backlog, citing BUNDLE-006 → SPEC-055

## Resolution

**Track transfer, not abandonment.** This issue was authored on the REACTIVE track
(issue → plan) and reached the point of a fully-worked implementation plan
(`PLAN-ISSUE-073-production-git-cloner-remote-pack-install`). Before that plan was
implemented, the same surface — production remote (git) pack installation — was picked
up on the FEATURE track: `BUNDLE-006` (pack-distribution-lifecycle) resolved its open
questions, promoted to `defined`, and spawned `SPEC-055` (Production Remote Dependency
Assembly) as one of its seeds. Once a feature-track spec owns a surface, the reactive-track
issue is retired to point at the spec rather than being implemented independently in
parallel — running both would fork the fix into two divergent implementations of the same
`GitCloner` wiring.

**Verified successor state at retirement (2026-07-26):**
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` exists, descending from
  `BUNDLE-006`.
- `plans/PLAN-SPEC-055-production-remote-dependency-assembly.plan.yml` exists: 52 tasks /
  12 phases (commit `856400c`).
- Phase 1 (hermetic remote substrate, `cbe0869`) and Phase 2 (reportError seam +
  separated-stream runner, `2a4b5c7`) have already landed, plus a v1.3.0 Clone-strip
  amendment (`4086480`) that also retired the sibling reactive-track plan.
- `plans/PLAN-ISSUE-073-production-git-cloner-remote-pack-install.plan.yml` carries
  `status: replaced`, `replaced-by: SPEC-055`.
- `directives/DIR-026-remote-dependency-assembly.directive.md` is being authored in
  parallel to own this workstream in the backlog, citing `BUNDLE-006 → SPEC-055`.

The defect described above — the nil `GitCloner`, the unconditional panic on any git pack
ref, the missing production `Clone` implementation, and the inconsistent nil-guarding
across `add`/`update`/`upgrade` vs. `install` — is real and is being fixed. It is being
fixed under `SPEC-055`, not here, so this issue is retired rather than closed.
