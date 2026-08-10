---
name: goreleaser-check-preconditions
description: goreleaser check needs a git remote AND rejects the deprecated `brews:` key — both make it exit non-zero on a config that is actually valid
metadata:
  type: project
---

`goreleaser check` on backstop-core exits non-zero for two reasons that have
nothing to do with the config being wrong (measured 2026-07-27, goreleaser
v2.17.1 via `go run github.com/goreleaser/goreleaser/v2@latest`):

1. **No git remote** -> `configuration is invalid: scm releases: no remote
   configured to list refs from`. It never reaches config validation. This repo
   has an empty `git remote -v` until the private remote exists.
2. **`brews:` is DEPRECATED** in favor of `homebrew_casks:` -> `configuration is
   valid, but uses deprecated properties` and still `check failed`, non-zero.
   `release --snapshot` only warns and succeeds.

To validate the config WITHOUT touching the shared repo's git config: copy
`.goreleaser.yml` to a scratch dir, `git init` + `git remote add origin <url>`,
add a stub `go.mod` + `cmd/<binary>/main.go`, then run `check` there. Cite the
sha256 of both copies to prove the file is identical.

**Why:** ISSUE-087 Phase 3 ratifies `brews:` at three sites (CLM-019) while
CLM-012 requires `check` to pass — on current goreleaser you cannot have both,
and the conflict is invisible until you actually run the tool.

**How to apply:** never report `check` as passed/failed without separating these
two causes. The `brews:` -> `homebrew_casks:` rename clears `check` (verified
exit 0) and preserves `repository{owner,name}`, but turns the formula into a
CASK — Homebrew on Linux has no casks, so it silently drops the Linux brew path.
That is a founder decision, not an implementer's. See
[[feedback_agent_guard_testdata]] — `.goreleaser.yml` is non-Go, so it can only
be written via a Bash heredoc.
