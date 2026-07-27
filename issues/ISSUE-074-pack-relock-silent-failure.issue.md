---
title: "`pack relock` on an Edited Local Pack Exits 1 With Zero Output and Leaves the Lock Unchanged"
schema_version: issue/v1

issue:
  id: ISSUE-074
  title: "`pack relock` on an Edited Local Pack Exits 1 With Zero Output and Leaves the Lock Unchanged"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `pack relock` on an Edited Local Pack Exits 1 With Zero Output and Leaves the Lock Unchanged

## Problem

`backstop pack relock <pack-name>` — the documented workflow for refreshing a local pack's lock
entry after editing it in place — fails silently on exactly that use case: exit code 1, no
stdout, no stderr, `backstop.lock` unchanged. Documented in `docs/CODEBASE-MAP.md` under a "Known
gap" heading.

### Repro (performed 2026-07-25)

From the `stash` consumer project, which has `backstop/cobra-cli` installed as a local pack via
relative path `../backstop-cobra-cli-pack`:

1. Edit the installed local pack in place at its source (fixtures added, `pack.yml` version
   bumped 0.1.0 → 0.1.1; `backstop pack check` and `pack test` both pass in the pack repo).
2. In the consumer project, run `backstop pack relock backstop/cobra-cli`.
3. Observed: exit code 1, **no stdout, no stderr**, `backstop.lock` `content_hash` unchanged.

The command's own `--help` text says it exists to "Refresh a local pack's lock entry after
editing it in place" — this is its exact intended use case, and it fails on it.

Workaround that succeeded: `backstop pack remove backstop/cobra-cli` then `backstop pack add
../backstop-cobra-cli-pack` (lock hash refreshed `d45dd469…` → `62f6456d…`).

### Root cause — two defects in one

**(a) `relock` takes a filesystem path, not the pack name every sibling command uses.**
`cmd/backstop/pack_relock.go` declares `Use: "relock [path]"` and passes `args[0]` straight
through to `distribution.Relock(".", args[0])`. Inside `pkg/pack/distribution/relock.go`,
`readPackName(packPath)` (relock.go:72) does `os.ReadFile(filepath.Join(packDir, "pack.yml"))` —
it expects `args[0]` to be a directory containing `pack.yml`. But `pack remove <name>`, `pack
update <name>`, and `pack upgrade <name>` all take the pack's declared NAME (e.g.
`backstop/cobra-cli`, as it appears in `backstop.yml`/`backstop.lock`), not a path. A user
reaching for `relock` naturally supplies the same identifier they use everywhere else — the pack
name — which is not a valid directory relative to cwd, so `readPackName` fails with `"reading
pack manifest at backstop/cobra-cli: ...: no such file or directory"`.

**(b) That error is swallowed — never printed.** `cmd/backstop/pack_relock.go` wraps any error
from `distribution.Relock` as `&ExitCodeError{Code: ExitViolations, Message: err.Error()}` (exit
code 1). `cmd/backstop/main.go`'s error handler only prints `Error: <message>` to stderr `if
exitErr.Code != ExitViolations` — the comment there explains the `ExitViolations` (exit 1) branch
is meant for commands like `gate`/`pack check` that already printed structured findings before
returning the error, so re-printing would be redundant. `relock` never prints anything before
returning its error, so this exit-1-means-already-explained convention doesn't hold for it: the
diagnostic is generated (`err.Error()`) and then discarded, leaving a bare nonzero exit with zero
output.

### Expected

`backstop pack relock <pack-name>` (mirroring `remove`/`update`/`upgrade`'s pack-name argument —
or, if the path-based signature is intentional, clearly documented as such and made to fail loud
when a bare name is given) recomputes the content hash from the pack's local source, updates the
lock entry (and version field), and prints what changed. On failure for any reason, a diagnostic
must reach stderr — `relock` should not route non-diagnostic errors through the
`ExitViolations` "already explained" convention, or it must explain itself before returning.

## References

- `cmd/backstop/pack_relock.go` — `Use: "relock [path]"`; wraps errors as
  `ExitCodeError{Code: ExitViolations, ...}` with nothing printed beforehand
- `pkg/pack/distribution/relock.go:29` — `Relock(projectDir, packPath string)` signature (path,
  not name)
- `pkg/pack/distribution/relock.go:72-86` — `readPackName`, the failure site when given a bare
  pack name instead of a directory
- `cmd/backstop/main.go:17-24` — the `ExitViolations`-suppresses-stderr convention that swallows
  the diagnostic here
- `cmd/backstop/exitcode.go:13,16` — `ExitViolations = 1`, `ExitConfigError = 2`
- Contrast: `cmd/backstop/pack_remove.go`, `pack_update.go`, `pack_upgrade.go` all take the pack
  name as declared in `backstop.yml`/`backstop.lock`
- `docs/CODEBASE-MAP.md` — "Known gap" heading documenting this
- Discovered dogfooding from `~/src/projects/stash` (consumer project) with
  `backstop/cobra-cli` installed from `~/src/projects/backstop-cobra-cli-pack` as a local pack
