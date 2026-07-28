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
- **INERT** — the rule currently matches nothing at either waiver site.
  `./bin/backstop gate --file cmd/backstop/pack_gate.go` and
  `--file cmd/backstop/pack_gate_provision.go` both PASS with zero
  `pack_engines` violations (re-run during authoring, 2026-07-28) — the same
  conclusion implementer-020-final reached independently via a whole-self-pack-
  ruleset semgrep run, two engines agreeing.
- **FAIL-OPEN** — if a future rule change or code change ever makes
  `no-structural-name-split-on-spine` match this code again, these two waivers
  will NOT suppress the finding: `pkg/waiver.Adjudicate` only suppresses on an
  EXACT rule-ID match (`pkg/waiver/adjudicate.go:124`), and the token's ID can
  never equal a real finding's ID again. `backstop gate` would go red with no
  warning that a waiver "should" have covered it — the two comments read as
  live suppression but are decorative.

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
