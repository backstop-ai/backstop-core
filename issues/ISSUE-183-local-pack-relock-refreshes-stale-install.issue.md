---
title: "Local Pack Relock Refreshes Stale Install"
schema_version: issue/v1

issue:
  id: ISSUE-183
  title: "Local Pack Relock Refreshes Stale Install"
  type: bug
  status: open
  created: "2026-08-22"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# Local Pack Relock Refreshes Stale Install

## Problem

The local-pack authoring loop can report a successful relock while continuing
to lock and execute a stale installed copy.

Observed sequence while developing `backstop-design-system`:

1. `backstop pack add ../backstop-design-system` installed the local source and
   wrote `local_path: ../backstop-design-system` to `backstop.lock`.
2. The source pack's rule file was edited to add path exclusions.
3. `backstop pack relock ../backstop-design-system` reported success and printed
   the same pre-edit content hash.
4. `backstop gate --all` continued executing the old installed rules and
   reproduced the old violations.
5. `backstop pack remove ...` followed by `backstop pack add
   ../backstop-design-system` produced a different hash and the corrected gate
   result.

The command's help says it "re-reads a locally-installed pack" and is intended
to avoid remove-plus-add. In practice, the supplied/recorded local source path
did not refresh the installed corpus before hashing it. A successful relock was
therefore not evidence that the lock represented the pack the author had just
edited.

Adjacent observation: `backstop pack list --json` reported `version: ""` for
the locally installed pack even though its manifest declared `0.1.0`. That may
be intentional because the lock's version is null for local sources, but it
makes the stale-source ambiguity harder to diagnose.

## Expected Behavior

For a lock entry with `source_type: local`, relock resolves the authoritative
source from the supplied path or recorded `local_path`, validates that source,
refreshes the installed copy atomically, computes the hash from those exact
bytes, and only then updates the lock. If the command intentionally hashes only
the installed copy, it must refuse a source-path argument and state that
contract explicitly.

## Acceptance Evidence

- Add a local pack, edit a governed file in its source, and run relock.
- The resulting hash differs and equals a clean remove-plus-add hash.
- The next gate executes the edited rule without another install command.
- Validation failure leaves both the prior installed copy and lock intact.
- Output identifies the source path and installed destination used.
- `pack list` distinguishes manifest version from lock/source version without
  rendering the former as an unexplained empty string.

## Existence-in-world Check

Searched the current Backstop Core issue/artifact corpus and repository source
for local relock, stale installed packs, and empty local-pack versions before
filing. No existing issue owns this observed sequence.
