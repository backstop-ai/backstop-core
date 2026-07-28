---
name: no-code-check-command
description: "`backstop code check` does not exist in this CLI — plans and the implementer contract both mandate it; use `backstop gate` (diff-scoped) instead"
metadata:
  type: project
---

Plans' verification tasks (and the implementer agent contract itself) say to run
`backstop code check` as the inner loop. **That subcommand does not exist.**
`backstop --help` lists only: `artifact`, `baseline`, `commands`, `completion`,
`gate`, `help`, `pack`, `version`, `waiver`. Invoking it exits non-zero with
`unknown command "code" for "backstop"`.

**Why:** the name is a holdover from the pre-packs-only era (see ISSUE-018's
mandated `TestCutover_GateNeverWiresStepCodeCheck` — the step was deliberately
cut over and removed). The plan/contract text was never updated.

**How to apply:** for the inner loop, run `go test ./pkg/<pkg>/...` for red/green
evidence, then `backstop gate` (default = diff-scoped) as the real check. Build
the binary fresh first — `backstop` on PATH is a stale /usr/local/bin copy, see
[[pack_copies_and_stale_gate_binary]]. Report the substitution honestly rather
than claiming `code check` passed. Related: [[netnegative_gate_baseline]].
