---
name: self-pack-backlog-closed
description: DIR-014 eradication backlog closed 2026-07-06 (ISSUE-021) — backstop/self is whole-repo GREEN, zero active findings
metadata:
  type: project
---

ISSUE-021 removed the last baked-language literal flagged by `backstop/self`:
`ExpectedLayout` in `pkg/pack/validate_manifest.go` unconditionally added
`"go.mod"` to every pack's expected layout. That function is ADVISORY / test-only
(no production caller enforces its return value), so the fix was a pure one-line
deletion — NOT a `Language=="go"` conditional (which would just relocate the
literal and re-trip the rule).

**Why:** it was DIR-014's last open self-finding; closing it makes the
thin-executor eradication backlog whole-repo GREEN on the self-pack dimension.

**How to apply:** after ISSUE-021, `backstop gate --all --json` shows **0 active
`backstop/self` violations across the whole repo** (the 5 go.mod findings on
validate_manifest.go moved to the FIXED array). If a NEW active `backstop/self`
finding appears, it is a genuine regression — investigate, do not assume it is
pre-existing. The overall gate still exits RED, but purely on non-self dimensions
(go-standards no-global-mutable-state, go-toolchain/staticcheck, contract_signature,
test_verification/substantiveness, coverage) — the known pre-existing quirks in
[[project_gate_residual_reds_issue027]] / [[project_struct_contract_compiler_gap]],
none of them self-pack. Relates to [[project_defaultregistry_eradication]].
