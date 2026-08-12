---
title: "cmd/backstop/baseline.go carries GitHub-Actions-specific knowledge — a candidate zero-baked-platform-knowledge violation"
schema_version: issue/v1

issue:
  id: ISSUE-120
  title: "cmd/backstop/baseline.go carries GitHub-Actions-specific knowledge — a candidate zero-baked-platform-knowledge violation"
  type: technical-debt
  status: open
  created: "2026-08-11"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# `cmd/backstop/baseline.go` bakes GitHub-Actions knowledge into core

## Problem

This project's standing invariant is zero baked language/tool/platform knowledge in core — every
check, toolchain, and platform-specific behavior is supposed to come from a pack, never from a
branch or vocabulary baked into `cmd/backstop`/`pkg/*` (see `backstop/self`'s zero-baked-checks
rule, and CLAUDE.md's "a baked language/tool branch is a defect to eradicate, never to extend").
`cmd/backstop/baseline.go` — specifically its `baseline pull` subcommand — carries GitHub-Actions-
specific knowledge that arguably violates that invariant: it assumes the CI provider is GitHub
Actions, authenticates against GitHub specifically, and queries the GitHub Actions runs/artifacts
API directly, with no pack-level seam.

Concrete citations in `cmd/backstop/baseline.go`:

- **Line ~34** — prose in the `pull` command's `Long` help text: "Artifact lookup uses **GitHub
  Actions** runs and artifact naming semantics..."
- **Lines ~111 and ~179** — the function `ensureGitHubAuth`, whose name and body
  (`exec.Command("gh", "auth", "status")`) are GitHub-CLI-specific.
- **Line ~168** — error string: `"missing origin remote; cannot resolve GitHub repository"`.
- **Line ~183** — error string: `"missing GitHub authentication; run `gh auth login` for baseline
  pull"`.

(There is also a `github\.com` module-path-style regex at line ~171 used to parse the origin
remote into an owner/repo pair for the GitHub API call — that particular literal is a SEPARATE,
narrower, already-resolved concern: see Scope note below.)

## Why this is a candidate violation, not yet a ruling

This is exactly the class of defect the project's `backstop/self` pack and its zero-baked-checks
standing rule exist to catch and eradicate: a platform assumption (GitHub Actions specifically,
plus the `gh` CLI as an implicit dependency) sitting inside core Go source rather than expressed as
pack data. `baseline pull`'s entire mechanism — "find the latest successful main-branch CI run on
GitHub Actions, download its baseline artifact" — has no non-GitHub equivalent path today; a
GitLab/Bitbucket/Jenkins consumer of `backstop baseline pull` gets nothing.

This issue exists to raise the architectural question, not to answer it: whether core should
support ONE hardcoded CI-artifact-retrieval provider (as a documented, accepted exception — baseline
pull needs SOME way to reach a CI provider's API, and that may reasonably not be pack-expressible in
the same way a findings engine is), or whether this needs to become pack-pluggable like everything
else this project has already migrated (see `project_native_toolchain_cutover`,
`project_typescript_packs` in agent memory for the shape that migration usually takes).

## Scope note — this is explicitly NOT a re-litigation of SPEC-067 CLM-050

SPEC-067 (`specs/SPEC-067-ci-recipe-pack.spec.md`) added a claim, CLM-050, that scans
`pkg/recipe/` and `cmd/backstop/` for baked platform literals including the case-sensitive token
`github`. During that spec's implementation the same regex line this issue cites
(`cmd/backstop/baseline.go:171`, `github\.com[:/]...`) was found to conflict with CLM-050's original
literal-prefix reading, and the founder ruled (2026-08-11, spec version 1.0.3) to widen CLM-050's
exemption to cover both the plain `github.com/` and the regex-escaped `github\.com` spelling — both
are legitimate module-path references, not platform knowledge. **CLM-050 is settled and currently
measures GREEN on this tree; do not reopen that question here.**

What the 1.0.3 ruling explicitly did NOT do — and what this issue is actually about — is judge the
FIVE CAPITALIZED mentions in the same file: the "GitHub Actions" comment, `ensureGitHubAuth` (×2),
and the two GitHub-naming error strings. CLM-050's token match is deliberately case-sensitive
(lowercase `github` only), so those five mentions fall outside what that claim measures, on
purpose. This issue is the architectural question the 1.0.3 ruling deliberately left open: whether
core should carry GitHub-Actions-specific knowledge at all — not a request to make CLM-050's scan
stricter.

## Direction (not scoped here)

Not decided by this issue — for whoever plans it to weigh: keep as a documented, narrow exception
(baseline pull is inherently "talk to a CI provider's API," which may not be a findings-engine-
shaped problem a pack can own the same way); or extract a `baseline-pull` seam that a
provider-specific pack or config value supplies, so a non-GitHub-Actions consumer isn't silently
unsupported forever.

## Notes / references

- Discovery context: explicitly named as a follow-on during SPEC-067's 1.0.3 spec amendment
  (CLM-050's exemption widening). The founder ruled this file's broader GitHub-Actions-specific
  knowledge is a SEPARATE concern from CLM-050's narrower module-path-literal claim and directed
  that it get its own issue rather than be silently absorbed into or block SPEC-067.
- Cites `specs/SPEC-067-ci-recipe-pack.spec.md` REQ-008/CLM-050, its Sharp Edge 5 (implementation
  notes, ~line 1376), and the 1.0.3 Version History entry (~line 1577) recording the founder's
  ruling.
- Cites `plans/PLAN-SPEC-067-ci-recipe-pack.plan.yml` TASK-031 (~line 2533), which named this exact
  follow-on (slug `core-baseline-bakes-github-actions-knowledge`) and the scope note above verbatim
  as guidance for whoever authors it.
