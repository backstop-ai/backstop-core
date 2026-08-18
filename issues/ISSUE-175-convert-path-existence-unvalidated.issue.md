---
title: "Convert Path Existence Unvalidated"
schema_version: issue/v1

issue:
  id: ISSUE-175
  title: "Convert Path Existence Unvalidated"
  type: technical-debt
  status: open
  created: "2026-08-18"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Convert Path Existence Unvalidated

## Problem

`pkg/pack/engine/testdata/contracts-grep-engine.yml` (manifest name `acme/contracts-grep-pack`)
declares a `grep` engine binding with `command: grep -rn` and `convert: grep/to-sarif.sh` — but
`pkg/pack/engine/testdata/grep/` does not exist on disk at all (verified 2026-08-18: no such
directory, no such file). The `convert:` reference is orphaned.

### Why it has gone unnoticed

The fixture is declaration-level only. Its header comment states it drives
`TestEngine_GrepPackDeclaredNotInDefaultRegistry` — a test that inspects the parsed manifest's
engine binding, never dispatches it. Nothing in the test corpus ever actually runs this binding's
`command` and pipes its output through the named `convert:` script, so the dangling path has never
been exercised and nothing has ever failed because of it.

### How it was found

Surfaced during `PLAN-ISSUE-166`'s investigation of the GNU-grep single-file filename-omission
defect (`ISSUE-166`). That plan's discovery sweep for every convert-bearing `command: grep`
declaration in the repo found this manifest as a real hit — one not identifiable by a
`pack.yml`-only sweep, since the file is not named `pack.yml` — and, checking it against disk,
found the referenced convert script absent. `PLAN-ISSUE-166`'s `TASK-004` gave this binding's
`command` the same `-H -I` flags every other convert-bearing grep declaration received (for
sweep-predicate consistency — the sweep identifies bindings structurally and does not special-case
an orphaned one), but deliberately did not create the missing script: inventing a convert script
for a fixture nothing dispatches would have been scope creep unrelated to that plan's own defect.

## Impact

Low in isolation — the binding is never dispatched, so the dangling reference cannot currently
cause an incorrect gate verdict. But it points at a general validation gap: nothing in this repo's
pack-manifest validation (`pkg/pack`'s manifest parsing, nor `pkg/packval`'s structural/coherence
phases) checks that a declared `convert:` path actually RESOLVES to a file on disk. That is the
same silent-hole shape as the defect `ISSUE-166`/`PLAN-ISSUE-166` fixed — a declared thing that
looks correct until something actually tries to use it — just in a different location (manifest
structural validity, not a convert script's runtime parsing behavior). A REAL pack manifest
shipping a dangling `convert:` path would currently pass `pack check`/`pack test` right up until
the binding is actually dispatched, at which point it would fail at dispatch time with whatever
error the runner produces for a missing script, rather than being caught earlier and more legibly
at authoring time.

## Solution

Not prescribed here. Candidate directions:

- Add a structural check (likely in `pkg/packval`'s phase-1/phase-2 validation, alongside its
  existing manifest-shape checks) that, for every engine binding declaring a non-empty
  `convert:`, the resolved path exists relative to the pack root — surfaced as a validation error
  at `pack check`/`pack test` time rather than left to fail opaquely at dispatch.
- Decide whether this specific fixture (`pkg/pack/engine/testdata/contracts-grep-engine.yml`)
  should instead be updated to NOT declare a `convert:` it doesn't back, if its only purpose is
  declaration-level parsing assertions and the dangling path is incidental rather than
  load-bearing for what it tests.
- Confirm whether `pkg/packval`'s testdata-exclusion conventions (fixtures under `testdata/` are
  not full packs and are not expected to pass `pack check`) already cover this fixture, in which
  case the new structural check would need to explicitly scope past it rather than redding this
  fixture's own tests.

## References

- `pkg/pack/engine/testdata/contracts-grep-engine.yml` — the orphaned declaration, line naming
  `convert: grep/to-sarif.sh`.
- `pkg/pack/engine/contracts_grep_engine_test.go` — carries
  `TestEngine_GrepPackDeclaredNotInDefaultRegistry`, the only test that reads this fixture, and
  does so at the declaration level only.
- `plans/PLAN-ISSUE-166-contracts-grep-convert-singlefile-filename.plan.yml` — notes section 4
  (the sweep that found this as the sixth, not-named-`pack.yml` convert-bearing declaration) and
  `TASK-004`/`TASK-010`'s text (flags added for sweep consistency; script creation explicitly out
  of scope and deferred to this filing).
- `ISSUE-166` (`contracts-pack-phase3-fixtures-fail-on-linux-ci`) — the defect whose investigation
  surfaced this orphaned reference as a byproduct.

### Existence-in-world check

Performed 2026-08-18 before filing: searched `issues/` and `bundles/` for "orphaned", "dangling
convert", "convert path", and "contracts-grep-engine.yml". No open issue or bundle charter already
owns a check for declared-`convert:`-path existence; the general "dangling reference" hits found
(`BUNDLE-005`, `BUNDLE-013`, `BUNDLE-014`, `BUNDLE-003`, `BUNDLE-004`) each cover a different kind
of dangling reference (waiver targets, rule-source paths, traceability `supports` refs, a coverage
knob) and none owns convert-script-path existence specifically.
