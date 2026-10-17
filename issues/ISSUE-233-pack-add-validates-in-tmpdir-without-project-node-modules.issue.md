---
title: "Pack Add Validates In TMPDIR Without Project Node Modules"
schema_version: issue/v1

issue:
  id: ISSUE-233
  title: "Pack Add Validates In TMPDIR Without Project Node Modules"
  type: bug
  status: open
  created: "2026-10-17"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Pack Add Validates In TMPDIR Without Project Node Modules

## Problem

`backstop pack add` validates the incoming pack in a scratch directory under the system temp dir, away from
the consuming project. A pack whose fixture engine runs a tool that is installed in the consumer project
(for example resolved through the project's `node_modules/.bin`) can fail validation during add because
that tool does not exist in the scratch location.

Verified at origin/main:

- `RunValidationOnScratchCopy` (`pkg/pack/distribution/command.go:40-62`) does
  `os.MkdirTemp("", scratchValidationDirPattern)`, copies only the pack directory into it
  (`copyDirRecursive(packDir, scratch)`), then runs the full `RunPackCheck` and `RunPackTest` on it.
- `Add` calls it for both git and local sources (`command.go:~247-259`). `update` and `upgrade` use the
  same function (`command.go:~680, ~827`).
- The scratch copy exists on purpose (fixture execution must not mutate the operator's tree or the
  hashed pack contents), so the fix is not to validate in place.

## Not confirmed

I found no code in core that resolves `node_modules/.bin`; resolution is by PATH and by the engine's working
directory, which for fixtures is the pack directory (here, the scratch copy). So the mechanism by which a
project-local tool would be found during `pack test` run from inside the project, but not during `add`,
is inferred from the scratch-directory design, not reproduced. A reproduction needs a pack whose fixture
engine invokes a project-installed tool.

## Impact

A pack that passes `pack test` in the project can be refused by `pack add` (or `update`/`upgrade`), with no
indication that the environment, not the pack, is the cause.

## Direction

(Direction, not a plan; uncertain.) Options: run the scratch validation with the project's tool locations
visible (extra PATH entries, or a link to the project's dependency directory inside the scratch dir),
or let a pack declare that its fixtures need project tools and skip the fixture phase at add time with a
clear message. Any option must keep the scratch copy so validation cannot alter the hashed pack content.

## Notes

- Related: ISSUE-231 (sandbox environment differences that make `pack test` behave differently from a
  real run).
