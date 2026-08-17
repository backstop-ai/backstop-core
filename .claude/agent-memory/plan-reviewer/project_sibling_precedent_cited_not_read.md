---
name: sibling-precedent-cited-not-read
description: A plan deferring work by citing a sibling plan's "precedent" often inverts it — read what the sibling plan DOES (its phases/file scope), not the doctrine it states
metadata:
  type: project
---

When a plan defers scope by citing a sibling lane as precedent ("per the ISSUE-NNN
precedent, X and its mirror are two owners, so the mirror is out of scope"), open the
sibling plan and read its PHASES and `files:`, not the doctrine sentence it shares.

**Why:** PLAN-ISSUE-142 (2026-08-17) deferred the external `backstop-ai/go-contracts`
mirror to an unfiled residual, citing "the ISSUE-148 precedent" that an in-repo pack
source and its published mirror are two owners. PLAN-ISSUE-148 states the identical
doctrine and draws the OPPOSITE conclusion — its sharp edge 8 reads "TWO OWNERS… **In-repo
alone is a half fix**" — and fixes its mirror IN-LANE (phase 4 prepares the external
working tree, phase 5 founder-gated publish + relock), plus carries a dedicated task to
FILE its follow-ons rather than absorbing them. The cited precedent contradicted the
decision it was used to justify.

**How to apply:** For any "per the SIBLING precedent, this is out of scope":
1. Extract the sibling's phase list and file scope (`python3 -c` over the YAML) — does it
   actually defer, or does it do the thing in-lane?
2. If it does it in-lane, the deferral needs its OWN justification, not a borrowed one.
   Ask what makes this lane different (in 142's case: `pack install` does NOT validate —
   `NewInstallCommand` takes no validator — so CI's fleet install and the gate are
   unaffected; only `pack add`/`update`/`upgrade` route through
   `RunValidationOnScratchCopy` and would refuse the stale mirror).
3. Check whether the sibling has a FILING task for its residuals. This repo's norm
   (MEMORY: thorough follow-ons) is that surfaced defects get filed first-class; a plan
   that only restates residuals in its own notes has not filed them.
4. State the sharpest consequence explicitly. "The mirror needs the same fix" understates
   "the published pack becomes uninstallable by `pack add` the moment phase 1 lands" — the
   founder needs the second form to prioritize.

Related: [[sibling-lane-exclusivity-fence]], [[lane-enumeration-misses-pending-tasks]],
[[stale-sibling-scaffold-evidence]].
