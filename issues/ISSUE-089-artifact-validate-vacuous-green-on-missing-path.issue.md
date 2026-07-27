---
title: "`artifact validate <path>` Ignores The Path Entirely — Vacuous Green On Missing/Typo'd Files"
schema_version: issue/v1

issue:
  id: ISSUE-089
  title: "`artifact validate <path>` Ignores The Path Entirely — Vacuous Green On Missing/Typo'd Files"
  type: bug
  status: open
  created: "2026-07-27"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `artifact validate <path>` Ignores The Path Entirely — Vacuous Green On Missing/Typo'd Files

## Problem

`backstop artifact validate <path>` does not validate `<path>`. It has no positional-argument
handling at all, so a nonexistent, renamed, or typo'd path is silently dropped and the command
falls back to validating the **entire discovered corpus** (or whichever type is scoped in via
`--spec`/`--issue`/etc.) — the result reported has nothing to do with the file named on the
command line.

### Reproduction (verified live, 2026-07-27)

```
$ ./bin/backstop artifact validate issues/DOES-NOT-EXIST.issue.md
```

Exit code and output depend entirely on the ambient state of the rest of the repo's artifact
corpus, not on whether `issues/DOES-NOT-EXIST.issue.md` exists. Ran once against a repo state
where one real issue happened to have a violation: exit 1, reporting a violation on a
*different, real* file — never mentioning the missing path at all. If every artifact in the
corpus is clean, the same nonexistent-path invocation exits 0 with "✓ All checks passed" —
a pure vacuous green, since the named file was never opened, let alone checked.

### Root cause (cited)

`cmd/backstop/artifact_validate.go`, `NewArtifactValidateCommand()`:

- The `cobra.Command` literal sets no `Args` validator, so cobra's default (nil `Args` = no
  positional-arg validation) silently accepts any stray positional argument.
- `RunE: func(cmd *cobra.Command, _ []string) error` — the positional-args slice is discarded
  outright; nothing in the function body ever reads it.
- `cmd.Flags()` only defines `--spec`, `--plan`, `--adr`, `--bundle`, `--issue`, `--directive`,
  `--all` — all *type*-scoped ID filters, not a path. `--help` confirms the command's own
  contract: `Usage: backstop artifact validate [flags]` — no path argument exists in the
  command's design.
- `ValidateArtifacts(cfg ValidateConfig)` (same file) always calls
  `DiscoverArtifacts(cfg.ProjectRoot, typeFilters)` — a `cwd`-rooted, corpus-wide discovery scoped
  only by `cfg.TypeFilters` (which come from `--spec`/`--issue`/etc., never from a path) — and
  never once reads, stats, or references the string a caller passed as a bare argument.

So this is not "path-collection logic yields an empty artifact set for a missing path" (there is
no path-collection logic to begin with) — it's that the CLI accepts a positional path argument
syntactically, silently reinterprets the invocation as "validate the whole project" (or whole
type, if `--issue`/`--spec`/etc. was also given), and reports a verdict completely decoupled
from the name on the command line.

### Impact

This is the exact vacuous-green class backstop exists to kill, surfacing in backstop's own
tooling:

1. Orchestration/CI that validates a specific artifact by path gets a result that is a function
   of the rest of the repo, not of that artifact — a typo'd or renamed path can report either a
   false pass (rest of corpus clean) or a false/unrelated failure (rest of corpus dirty), never
   the truth about the named file. This bit the orchestrator directly: a validate call against a
   wrong ISSUE-081 filename returned green and temporarily masked whether an agent's edits
   against that artifact existed at all.
2. Any workflow gating on "`artifact validate <path>` exits 0" for a specific file is
   unfalsifiable against path drift. Artifact filenames do change in this repo (slugs vs. later
   shorthand, scaffold renames), and the command gives no signal when the target it was told to
   check isn't the target it actually checked.

### Expected behavior

Either:
- Add real positional-path support: resolve `<path>`, error loudly (config-class, exit 2) if it
  does not exist or does not parse as a known artifact type, and validate *only* that artifact;
  or
- If path-scoped validation is out of scope by design, reject the stray positional argument
  outright (`Args: cobra.NoArgs` or `cobra.ArbitraryArgs` replaced with an explicit "unknown
  argument" error) rather than silently accepting and discarding it — so a caller who thinks
  they scoped the check gets a loud, immediate signal instead of a corpus-wide result standing
  in for the file they named.

Either fix restores the invariant this tool exists to enforce elsewhere: a check that can't run
against its named target should never report as if it did.

## References

- `cmd/backstop/artifact_validate.go` — `NewArtifactValidateCommand()` (no `Args` validator, no
  path flag/positional wiring; `RunE`'s `_ []string` discards positional args); `ValidateArtifacts`
  (always corpus-wide `DiscoverArtifacts(cfg.ProjectRoot, typeFilters)`, never path-aware)
- `./bin/backstop artifact validate --help` — confirmed usage contract is `[flags]` only, no path
  argument
- `cmd/backstop/main.go` — `*ExitCodeError` → exit code mapping (0 pass / 1 violations / 2 config),
  the convention a loud missing-path error should follow
