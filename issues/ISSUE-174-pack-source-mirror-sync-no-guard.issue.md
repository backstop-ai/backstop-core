---
title: "Pack Source Mirror Sync No Guard"
schema_version: issue/v1

issue:
  id: ISSUE-174
  title: "Pack Source Mirror Sync No Guard"
  type: technical-debt
  status: open
  created: "2026-08-18"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Pack Source Mirror Sync No Guard

## Problem

`PLAN-ISSUE-166` (fixing the GNU-grep single-file filename-omission defect in every
grep-to-SARIF convert this repo owns) surfaced, as a byproduct of its own discovery sweep, that
nothing keeps an in-repo pack SOURCE and its released external mirror in sync, and that this is
not a theoretical risk — it is what let the same defect ship in four copies of one script when
the issue that started that plan named only two.

### The measured fact

`PLAN-ISSUE-166`'s notes (section 6) diffed every `grep/to-sarif.sh` in the repo and found FOUR
byte-identical copies of the `jq`-based convert script, not the two `ISSUE-166` named:

- `packs/contracts/grep/to-sarif.sh` — the in-repo pack source.
- `pkg/gate/testdata/traceability-pack/grep/to-sarif.sh` — named in `ISSUE-166`.
- `pkg/gate/testdata/ts-proof-pack/grep/to-sarif.sh` — **not** named in `ISSUE-166`; found only
  by the plan's own discovery sweep, and the exact kind of "silent sibling" a hand-maintained
  convention is supposed to catch and did not.
- `.backstop/packs/backstop-ai/go-contracts/grep/to-sarif.sh` — the installed copy of the
  RELEASED external pack at `/Users/bmanson/src/projects/backstop-go-contracts-pack`, on its own
  release cycle, in its own repo.

Three of the four in-repo/near-repo copies carried the identical `NF >= 3` silent-drop defect;
the fourth (the external mirror) had to be fixed and republished separately (`TASK-008`,
`backstop-ai/go-contracts` bumped 1.2.0 -> 1.3.0) because nothing in this repo's own test suite
or CI reads that repo's source and nothing in that repo's own suite reads this one's. The two
repos were kept aligned by hope, and the hope had already silently failed once (this defect) by
the time anyone looked.

### Why this is general, not specific to go-contracts

The same shape applies wherever backstop-core maintains an in-repo pack source that is ALSO
published as an external, independently-versioned mirror: `backstop-ai/go-contracts`,
`backstop-ai/go-substantiveness`, and the traceability packs are all named in `PLAN-ISSUE-166`'s
notes as carrying the same general exposure. `ISSUE-157`
(`go-contracts-mirror-inverted-fixture-polarity`) and `ISSUE-148`
(`substantiveness-fixture-polarity-inverted`) are each a DIFFERENT concrete defect that surfaced
from this SAME general gap — a mirror silently diverging from its in-repo source of truth — which
is evidence this is a recurring failure mode, not a one-off.

### Related but distinct: ISSUE-137

`ISSUE-137` ("No automated guard keeps the go-toolchain pack fixture in sync with the released
pack") names an adjacent gap for a DIFFERENT surface: a TEST FIXTURE
(`cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`) that mirrors
a released pack's bindings for exemption-matrix testing. This issue's surface is different — the
IN-REPO PACK SOURCE itself (`packs/contracts/**`, and by the same shape `packs/substantiveness/**`
and the traceability packs) drifting from the SOURCE its external repo publishes, independent of
any test fixture. The two issues do not subsume each other; a fix for one would not close the
other, since ISSUE-137's fixture-vs-release check and this issue's source-vs-mirror check compare
different pairs of files.

### Why this plan did not attempt a fix

`PLAN-ISSUE-166` measured and rejected building a cross-repo sync mechanism as part of a contained
bug fix: byte-identity cannot be the enforced convention even within this repo alone (two of the
six grep converts are deliberately divergent, carrying their own `ruleId` and no `jq` dependency,
by design), and no test in this repo can assert equality against a file living in a separate repo
on its own release cycle without inventing new cross-repo tooling. That plan instead pinned the
BEHAVIORAL property (`TestGrepConvert_RefusesUnparseableGrepLineLoudly`, discovering converts
structurally rather than listing them) for the in-repo half, and treated the external mirror as a
separate, founder-gated publish (`TASK-008`) — deliberately leaving this general residual
unabsorbed.

## Impact

Any future change to a pack's grep-based (or, by the same shape, any findings-engine) rule
content, fixture, or convert script can land in the released mirror without a matching in-repo
source update, or vice versa, and nothing will fail loudly on either side — the exact drift class
that let three of four `to-sarif.sh` copies carry an identical silent-false-negative defect for an
unmeasured length of time before a Linux CI run happened to exercise the one code path that
exposed it.

## Solution

Not prescribed here. Candidate directions, none evaluated in depth:

- A CI or `pack check`-adjacent step that, for each in-repo pack source with a known published
  mirror, clones or fetches the mirror's tagged HEAD and diffs the shared surface (engine
  bindings, convert scripts) — flagging divergence as a report, not necessarily a hard block,
  since some divergence (version numbers, deliberately-different scripts) is expected.
- A documented, checked convention for which files in a pack source are expected to be
  byte-identical to their mirror counterpart vs which are allowed to diverge (mirroring the
  distinction `PLAN-ISSUE-166` drew between the `jq`-family converts and the two deliberately
  divergent pure-awk ones).
- Scoping this to the packs actually confirmed to have this exposure today
  (`backstop-ai/go-contracts`, `backstop-ai/go-substantiveness`, the traceability packs) rather
  than inventing a generic N-repo sync framework speculatively.

## References

- `plans/PLAN-ISSUE-166-contracts-grep-convert-singlefile-filename.plan.yml` — notes section 6
  (the four-copy discovery) and the "THE ISSUE'S OPEN 'KEEP THEM BYTE-IDENTICAL?' QUESTION,
  ANSWERED" section (why a byte-identity convention was rejected and this residual deliberately
  deferred, per its own text: "TASK-011 files it as its own issue via a routed `/issue`
  dispatch" — renumbered to `TASK-010` in the committed plan).
- `ISSUE-166` (`contracts-pack-phase3-fixtures-fail-on-linux-ci`) — the defect whose investigation
  surfaced this residual.
- `ISSUE-157` (`go-contracts-mirror-inverted-fixture-polarity`) and `ISSUE-148`
  (`substantiveness-fixture-polarity-inverted`) — two prior, concrete instances of a mirror
  diverging from its in-repo source of truth; cited as evidence this is a recurring shape, not
  proof either issue is a duplicate of this one.
- `ISSUE-137` (`pack-fixture-drift-no-guard`) — an adjacent but distinct gap (test-fixture-vs-
  released-pack parity, not pack-source-vs-external-mirror parity); cross-referenced, not
  subsumed.

### Existence-in-world check

Performed 2026-08-18 before filing: searched `issues/` and `bundles/` for "mirror", "sync",
"drift", "byte-identical", and "source repo". `ISSUE-137` is the closest hit and is a different
surface (documented above, not subsumed). `ISSUE-157`/`ISSUE-148` are concrete instances of the
general gap this issue names, not owners of the general gap itself. `BUNDLE-031`
(`release-currency-versioning-machinery`) covers backstop-core's OWN release-currency signal
(tags vs `HEAD`), not cross-repo pack-source-vs-mirror parity — a different concern. No open issue
or bundle charter already owns this general gap.
