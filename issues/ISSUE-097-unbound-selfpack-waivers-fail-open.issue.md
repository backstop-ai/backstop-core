---
title: "Unbound Selfpack Waivers Fail Open"
schema_version: issue/v1

issue:
  id: ISSUE-097
  title: "Unbound Selfpack Waivers Fail Open"
  type: bug
  status: open
  created: "2026-07-28"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-097: Unbound Selfpack Waivers Fail Open

## Problem

Two `@waiver:` tokens are keyed to a rule ID that no installed pack can ever
produce a finding under, and neither the waiver adjudicator nor
`backstop waiver list` can see that they are dead:

```go
// cmd/backstop/pack_gate.go:888
// @waiver:backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine:false-positive:2027-07-17 legitimate command->argv tokenization (a command line IS whitespace-delimited by shell semantics), not name-from-message extraction (ISSUE-062)

// cmd/backstop/pack_gate_provision.go:119
// @waiver:backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine:false-positive:2027-07-17 legitimate executable-name extraction from a command line (argv[0] is a whitespace-free token by shell semantics), not name-from-message extraction (ISSUE-062)
```

Both key `backstop/self/backstop.packs.backstop.self.rules.no-structural-name-split-on-spine`.
That pack path — `backstop/self` — no longer exists anywhere. Per the
2026-07-27 pack rename (`backstop/self` → `backstop-ai/backstop-self`,
recorded in `backstop.lock`), the installed pack is `backstop-ai/backstop-self`
(`.backstop/packs/backstop-ai/backstop-self/pack.yml:1`) and its findings
carry the namespace `backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.<id>`
— confirmed by a live example already re-keyed correctly elsewhere in this
repo (`cmd/backstop/artifact_validate.go:17`, `pkg/pack/distribution/identity.go:38`,
both post-rename `backstop-ai/go-standards/...` tokens). The rule itself
(`no-structural-name-split-on-spine`) still exists, unchanged, at
`.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml:163-178` — only
its namespaced ID changed under the rename.

Discovered by implementer-020-final during PLAN-ISSUE-020's TASK-020 honesty
pass (2026-07-28). Measured status, independently confirmed here:

- **STALE** — the rule-ID prefix (`backstop/self/...`) references a pack path
  that no longer exists in `backstop.lock` or `.backstop/packs/`.
- ~~**INERT** — the rule currently matches nothing at either waiver site.~~
  **CORRECTED 2026-08-16 — this was false; see "Correction" below.** The
  original `--file`-scoped measurement was confounded: under real directory
  dispatch the rule DOES fire at both sites today.
- **FAIL-OPEN, and not merely hypothetically** — `pkg/waiver.Adjudicate` only
  suppresses on an EXACT rule-ID match (`pkg/waiver/adjudicate.go:124`), and
  the token's pre-rename ID can never equal a real (post-rename) finding's ID
  again. As corrected below, the underlying finding is not a future
  possibility — it is produced RIGHT NOW under directory-scoped gate
  dispatch (`gate --all`, or a bare `gate` on a stale/pre-fix binary), and
  these two waivers do not suppress it. `backstop gate` goes red under that
  dispatch shape with no warning that a waiver "should" have covered it — the
  two comments read as live suppression but are decorative.

### Correction (2026-08-16) — the original INERT measurement was confounded

The original "INERT" claim above was based solely on
`./bin/backstop gate --file cmd/backstop/pack_gate.go` and
`--file cmd/backstop/pack_gate_provision.go` reporting zero `pack_engines`
violations. That measurement is **not evidence the rule doesn't fire** — it
is evidence of a *separate* defect: explicit-file (`--file`) dispatch is
blind to this specific rule because the rule's `paths.include` glob is
directory-prefixed, and explicit-file dispatch does not evaluate
directory-scoped `paths.include` globs the way directory-scoped dispatch
does. This is exactly PLAN-ISSUE-091's own documented "THIRD DIVERGENCE"
defect (explicit-file dispatch missing directory-scoped rules that directory
dispatch catches) — discovered independently during PLAN-ISSUE-091 review on
2026-08-16, which is what surfaced this false premise here.

Under directory-scoped dispatch (`gate --all`, or the pre-fix behavior a
bare `gate` on cmd/backstop uses today) the rule **does** fire at both
sites, verified first-hand tonight:

1. A real semgrep 1.156.0 run directly against `cmd/backstop` (directory
   target) produces exactly 2 findings, ruleId
   `backstop.packs.backstop-ai.backstop-self.rules.no-structural-name-split-on-spine`.
   Re-confirmed independently while authoring this correction: isolating the
   rule from `.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml`
   into a standalone config and running semgrep directly —
   `semgrep --config <isolated-rule.yml> cmd/backstop` (directory target) —
   yields exactly the 2 results
   (`cmd/backstop/pack_gate.go:994`, `cmd/backstop/pack_gate_provision.go:188`
   — one line below each `@waiver:` comment, i.e. the code line the comment
   is attached to), while the SAME isolated rule run as
   `semgrep --config <isolated-rule.yml> cmd/backstop/pack_gate.go`
   (explicit single-file target, the `--file`-dispatch shape) yields 0
   results on that same file — reproducing the dispatch-shape divergence
   directly in semgrep itself, independent of backstop's own CLI.
2. `runFindingsEngine` namespaces this via `pack.NamespacedRuleID` using the
   pack's current, post-rename normalized name, producing the real finding
   ID `backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules...`.
3. Both `@waiver:` tokens key the pre-rename string
   `backstop/self/backstop.packs.backstop.self.rules...` — this does not
   match the finding's real ID.
4. `pkg/waiver/adjudicate.go`'s exact-match adjudication therefore suppresses
   NEITHER finding — they surface as live, unwaived violations under
   directory-scoped gate dispatch right now, not merely in some future
   scenario.

The pre-rename/post-rename path-mismatch diagnosis in the rest of this
Problem section (the "STALE" bullet, and the re-keyed-ID construction in
Solution part (a)) was correct and stands unchanged — only the "therefore
it's INERT" conclusion, which rested on the confounded `--file`-only
measurement, was wrong. This also strengthens the acceptance criteria below:
re-keying isn't merely defensive against a hypothetical future match, it
fixes an active fail-open happening under `gate --all` today.

### Root cause of the detection gap — waiver harvest is finding-driven, not tree-driven

This is not just "the two tokens are stale" — it is a structural blind spot in
how waivers are discovered at all. `Adjudicate` never scans the source tree
for `@waiver:` tokens; it only scans the one-line-above/on-line association
window of each **finding it was already given**
(`pkg/waiver/adjudicate.go:94-104`, `harvestWindow`/`windowLines`,
`adjudicate.go:205-230`). A token sitting on a line where its own rule no
longer fires — and where no *other* rule happens to fire in the same
two-line window — is never read into the `tokens` map at all. It doesn't even
reach the `Unused` bucket (`adjudicate.go:126-129`), which only classifies
tokens that WERE harvested. So `backstop waiver list` and the
`waiver_resolution` gate step both report exactly what they see today: clean,
zero active waivers flagged as unused (`pkg/gate/step_waiver.go:188`,
`"clean — no active waivers"`) — not because nothing is wrong, but because
nothing calls `read()` on those two lines outside of a finding that already
landed there. A rename-orphaned or hand-typo'd rule ID is architecturally
invisible to the current adjudication path, not merely unreported by it.

### The class, not just the instance

A waiver keyed to a rule ID that no finding can ever carry is undetectable
dead config: it doesn't warn on `pack install`/`pack update`, doesn't warn on
gate, doesn't show up in `waiver list`, and silently stops doing its one job
(suppression) the moment its target rule would otherwise fire. Per
CLAUDE.md's enforcement philosophy, the enemy here is exactly this kind of
silent config rot, not the current INERT state itself.

## Solution

Two parts — the trivial re-key, and the gap that let it go undetected.

### (a) Re-key or remove the two stale tokens

Follow the pack rename migration recipe's waiver step (step 6 of 7,
`.claude/agent-memory/implementer/project_pack_rename_migration_recipe.md`):
re-key both tokens to the current namespace. Note the recipe's normal method
— "get the new ID by reading `rule` from `gate --file <f> --json`, never by
hand-deriving it" — does not directly apply here, because the rule is
currently INERT at both sites and produces no live finding to read the ID
from. The correct new ID must instead be derived from the installed pack's
own rule declaration
(`.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml:163`,
`id: no-structural-name-split-on-spine`) combined with the pack's declared
`name:` (`.backstop/packs/backstop-ai/backstop-self/pack.yml:1`,
`backstop-ai/backstop-self`) and the same dotted-ID convention already
proven live elsewhere in this repo (`cmd/backstop/artifact_validate.go:17`):

```
backstop-ai/backstop-self/backstop.packs.backstop-ai.backstop-self.rules.no-structural-name-split-on-spine
```

Whoever implements this should re-verify that construction against the
pack's own ID-building code (wherever `pkg/pack`/`pkg/check` composes a rule's
namespaced ID from manifest name + rule id) before committing to it — the
string above is not itself a live finding and should not be trusted purely
by analogy.

### (b) Close the detection gap

`waiver_resolution` (or `backstop waiver list`) should be able to name
waivers whose rule-ID prefix matches no installed pack's namespace,
independent of whether any finding currently lands near them. This requires
harvesting `@waiver:` tokens from the tree directly (not only from
finding-adjacent windows) for this cross-check, or at minimum cross-checking
the tokens that a broader, unscoped harvest already turns up against the set
of pack namespaces `backstop.lock` currently records. Per CLAUDE.md's
enforcement philosophy this is loud-not-blocking: a waiver bound to an
unknown pack namespace is a strong signal of exactly this class of rot
(rename fallout, typo, or a pack that was removed outright) and should
surface as a warning, not fail the gate outright — the enemy is silent rot,
not noisy staleness.

### Acceptance

- [ ] Both waiver tokens (`cmd/backstop/pack_gate.go:888`,
      `cmd/backstop/pack_gate_provision.go:119`) are re-keyed to the current
      `backstop-ai/backstop-self` namespace (or removed, if the underlying
      code no longer needs the suppression — re-verify against a live
      `gate --file --json` run once re-keyed, to confirm the new ID actually
      matches were the rule to fire).
  - [ ] Detection: `backstop waiver list` (or the `waiver_resolution` gate
      step) can name a waiver whose rule-ID prefix matches no pack namespace
      currently in `backstop.lock`, without requiring a live finding at that
      location to trigger the check.
- [ ] `./bin/backstop gate --all` is green with no fail-open waivers left
      unbound in the tree.

## Verification

Re-run `./bin/backstop gate --file cmd/backstop/pack_gate.go --json` and
`--file cmd/backstop/pack_gate_provision.go --json` after re-keying and
confirm the tokens' rule-IDs are well-formed against the current
`backstop-ai/backstop-self` namespace (structurally, since the rule remains
INERT at both sites and will not itself fire). For the detection-gap half,
add a fixture: a waiver token bound to a plausible-looking but nonexistent
pack namespace, sitting where no finding currently lands, and confirm the new
check names it — proving the check does NOT rely on a finding being present
to see the token, unlike today's `Adjudicate`/`waiver list` path.

## Additional evidence

- 2026-08-16: found independently while verifying PLAN-ISSUE-112's `gate --all` run. Both
  orphaned tokens are still present today, though at drifted line numbers (unrelated edits in
  both files since this issue was filed): `cmd/backstop/pack_gate.go:981` (was `:888`) and
  `cmd/backstop/pack_gate_provision.go:187` (was `:119`) — line numbers here are likely to drift
  further; match by rule-ID string, not line. New angle: diff-scoped and `--file`-scoped gate
  runs over the exact files carrying these stale tokens both report clean — the orphaned-waiver
  visibility gap only surfaces via an UNSCOPED `gate --all` run, because the underlying rule
  wasn't in scope for the narrower runs. Given this project's stated primary use case is
  diff-scoped local gates (CI treated as basically ceremony per founder framing), a team relying
  on diff-scoped runs day-to-day could carry a silently-orphaned waiver indefinitely without ever
  seeing it flagged in the runs they actually watch. Worth folding into whatever "loud, not
  blocking" detection fix is eventually built for solution part (b): the fix should surface
  regardless of gate scope, not just on a full sweep.

- 2026-08-16: **Correction** to the entry above and to the original "INERT" claim in the Problem
  section — see "Correction (2026-08-16)" under Problem for the full mechanism. Short version:
  the original `gate --file` measurement was confounded by PLAN-ISSUE-091's THIRD DIVERGENCE
  (explicit-file dispatch blind to this rule's directory-prefixed `paths.include`), not evidence
  the rule doesn't fire. Under directory-scoped dispatch the rule fires at both sites and neither
  waiver suppresses it — verified via a direct semgrep run plus tracing `runFindingsEngine`'s
  namespacing through to the exact (non-matching) token strings.

## References

- `cmd/backstop/pack_gate.go:888` — first stale waiver token
- `cmd/backstop/pack_gate_provision.go:119` — second stale waiver token
- `.backstop/packs/backstop-ai/backstop-self/pack.yml:1` — current pack name
  (`backstop-ai/backstop-self`)
- `.backstop/packs/backstop-ai/backstop-self/rules/no-baked.yml:163-178` —
  the still-live `no-structural-name-split-on-spine` rule definition
- `pkg/waiver/adjudicate.go:94-104,124,205-230` — `Adjudicate`'s
  finding-driven harvest (`harvestWindow`/`windowLines`) and the exact
  rule-ID match that makes a re-namespaced token fail open; `Unused`
  classification only covers tokens that were harvested in the first place
- `pkg/gate/step_waiver.go:188` — `"clean — no active waivers"`, the message
  a scan like this one currently produces despite the two dead tokens
- `cmd/backstop/waiver.go` — `backstop waiver list`, the read-only inspection
  surface that should be able to name these
- `cmd/backstop/artifact_validate.go:17`, `pkg/pack/distribution/identity.go:38`
  — live post-rename waiver tokens proving the current
  `backstop-ai/<pack>/backstop.packs.backstop-ai.<pack>.rules.<id>` ID
  convention (ISSUE-061)
- `.claude/agent-memory/implementer/project_pack_rename_migration_recipe.md`
  — the 7-step pack-rename migration recipe; step 6 documents that a rename
  silently unbinds waivers ("An unbound waiver does not warn; the finding
  just reappears") — this issue is that same failure mode discovered from
  the opposite direction (the finding does NOT reappear because the rule
  itself is currently inert, so the unbinding stayed invisible even longer)
- `backstop.lock` — records the current `backstop-ai/backstop-self` pack
  coordinate and version (`1.1.2`)
- ISSUE-061 — prior waiver re-keying precedent from the same rename
- ISSUE-062 — the `no-structural-name-split-on-spine` rule's own origin
  (structured finding properties vs. parsed prose), referenced in both stale
  tokens' notes
- PLAN-ISSUE-020 (`plans/PLAN-ISSUE-020-linux-sandbox-gate-in-ci.plan.yml`),
  TASK-020 — the honesty pass that surfaced this defect
- PLAN-ISSUE-091 — source of the "THIRD DIVERGENCE" defect (explicit-file
  dispatch blind to directory-scoped `paths.include` rules) that explains why
  the original `gate --file`-based INERT measurement in this issue was
  confounded and false; discovered during its review on 2026-08-16
