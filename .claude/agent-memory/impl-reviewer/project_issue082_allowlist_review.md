---
name: issue082-allowlist-review
description: ISSUE-082 allowlist-removal review — 8-mutation scratchpad red-proof technique for deletion+prose specs; deletions need bidirectional + count asserts or they pass against an empty map
metadata:
  type: project
---

ISSUE-082 (remove 5 unreachable `TrustedToolAllowlist()` entries) reviewed clean on
implementation content; only gap was the plan's own `status:` never flipped to
`completed`, which the gate's `artifact_status_drift_advisory` then named.

**Why:** Deletion-shaped work is the easiest place to ship a vacuous green — a bare
"key X is absent" assertion passes against an EMPTY map, and a prose-correction claim
(CLM-003 here) is unfalsifiable unless something reads the file bytes. This plan got
both right: bidirectional presence+absence, an exact `len(al) != 3`, and a
read-the-source-file test asserting absence of BOTH suppression mechanisms this repo
has (`nosem` AND `@waiver:`) with `t.Fatalf` on the read error.

**How to apply:**
- For any leaf package with stdlib-only deps, red-proof falsifiers by copying just the
  source + test file into a scratchpad module with a 3-line `go.mod` and mutating there
  — seconds per mutation, zero risk to a shared tree. For tests that need `repoRoot()`
  (testdata-driven), `rsync -a --exclude '.git'` the whole repo (65M here, ~10s) and
  mutate the fixture. Both beat trusting the implementer's red-proof claim.
- Mutations worth running on a deletion spec: plant each suppression marker; re-add one
  removed key; drop one surviving key; EMPTY the map; restore the exact overclaim
  phrase; add an extra fixture entry to test a count assert. See
  [[issue116-line-carry-pass]] for the revert-based variant.
- A prose-overclaim check is a literal `strings.Contains` on one phrase — it holds the
  exact wording, not the claim. Note that limit rather than treating it as airtight.
- Deletions strand the SAME overclaim in unscoped sibling files. Grep the removed prose
  across the repo (`grep -rn "every pack-declared" --include="*.go"`) — here three sites
  survived (a doc comment, a test-helper comment, and user-facing CLI help). Report as a
  follow-on, not as an implementation failure, when the plan didn't scope them.
- Shared-tree hygiene that worked: build the reviewer's own binary to the scratchpad
  instead of `bin/backstop`, skip `-coverprofile` (another agent owns `cover.out`), and
  poll `ps aux` for a live `backstop gate` before starting your own.
- `backstop gate` needs `PATH=/Users/bmanson/go/bin:$PATH` or it exits 2 on
  `go-arch-lint not found on PATH` — that exit-2 is an environment gap, not a finding.
</content>
</invoke>
