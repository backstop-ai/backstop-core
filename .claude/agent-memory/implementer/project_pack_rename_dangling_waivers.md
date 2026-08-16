---
name: pack-rename-dangling-waivers
description: A renamed pack silently unbinds every @waiver: token naming its old id — findings resurface only under gate --all, and waiver_resolution reports them as "unused/dangling" while still PASSING
metadata:
  type: project
---

When a pack is renamed (`backstop/self` -> `backstop-ai/backstop-self`), every
in-source `@waiver:` token still naming the OLD id stops binding. The finding's
rule id now carries the new pack prefix, so the waiver matches nothing.

**Why this is hard to notice:**
- `waiver_resolution` still reports **PASS** — it lists the orphans as
  "N unused/dangling" in its summary line rather than failing. A dangling waiver
  is not itself a violation.
- The unwaived findings surfaced only under `gate --all`; the diff-scoped and
  `--file` runs over the very same files reported `pack_engines pass`. Consistent
  with [[project_gate_all_underreports_vs_diff]] — the two scopes are not
  supersets of each other in EITHER direction.

Observed 2026-08-16 during PLAN-ISSUE-112 close-out: exactly 2 dangling tokens,
both `no-structural-name-split-on-spine` on `strings.Fields(command)` in
`cmd/backstop/pack_gate.go` and `cmd/backstop/pack_gate_provision.go`. Both
constructs are byte-identical at HEAD, so the findings were inherited, not new.

**How to apply:**
- Read the `waiver_resolution` summary line for "unused/dangling" on EVERY gate
  run — it is the cheapest detector, and it sits inside a PASSING step.
- After any `pack add`/rename, grep the repo for `@waiver:<old-pack-id>/` and
  migrate the tokens. See [[project_pack_rename_migration_recipe]] for the other
  six silent effects of a rename.
- When attributing an unscoped red, diff the offending CONSTRUCT against HEAD
  (`git show HEAD:<file> | grep ...`) before owning it. A finding on a file your
  lane edited is not automatically a finding your lane caused.
