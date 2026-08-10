---
name: packless-baseline-fails-at-pack-loading
description: With .backstop/packs empty the gate collapses to a single pack_loading config error (exit 2, ONE violation naming the first declared pack) and baseline generate refuses to write — pack_lock_verification and its missing_pack findings never run
metadata:
  type: project
---

Measured 2026-07-28 in a detached worktree at ba9cc49, all six lock entries
`source_type: git`, `.backstop/packs/` absent:

    ./bin/backstop gate --all        exit 2, 1 violation:
      [pack_loading] declared pack backstop-ai/backstop-self is missing from <root>/.backstop/packs
    ./bin/backstop baseline generate exit 2, NO baseline written:
      "gate reported a configuration error (exit 2); refusing to write a baseline
       from a gate that produced no steps"

`loadInstalledPacks` (`cmd/backstop/pack_gate.go:164`) returns on the FIRST
missing declared pack, and `buildGateSteps` (`cmd/backstop/gate.go:654`)
collapses the entire step set to one `pack_loading` step with `ConfigErr: true`.

**Why:** it is easy to predict "N missing packs -> N `missing_pack` failures"
from `VerifyLock` (`pkg/pack/distribution/verify.go:46-60`), because that step
does iterate every git entry. PLAN-ISSUE-020 predicted exactly six. It is
unreachable: `pack_lock_verification` sits DOWNSTREAM of a gate that is never
built. The count is 1, not N, and the failure class is config-error, not
violations.

**How to apply:** when reasoning about a packless CI job, the failure is a loud
exit-2 that publishes NOTHING — it cannot silently ratchet a degraded baseline.
Populated (after `pack install`, six packs) the same command exits 0 and writes
`pack_lock_verification: 0`. Do not quote a `missing_pack` count for a packless
run without running it. Related: [[project_pack_copies_and_stale_gate_binary]],
[[project_full_cli_gate_fixture_reqs]].
