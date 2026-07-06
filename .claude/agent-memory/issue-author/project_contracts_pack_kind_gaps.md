---
name: project-contracts-pack-kind-gaps
description: ISSUE-036/037 family — contracts compiler kind-aware fixes and the residual iota-member gap; pattern-arg is string-only
metadata:
  type: project
---

The contracts pack's signature compiler (`packs/contracts/scripts/compile-signature.sh`)
went from func-only (bug, ISSUE-036) to kind-aware for func/type/const/var/method. That
fix intentionally still can't verify a bare iota-block const member (no `=` value to
compile) — filed as ISSUE-037 (technical-debt, open, 2026-07-05). The one live instance
(SPEC-035's `CheckTypeFindings`, `pkg/check/manifest.go:22`) was retired as inexpressible
and is covered behaviorally instead (neutral-rename `.String()`/`parseCheckType` tests),
not structurally.

**Why it matters:** the `ast-grep-contracts` engine's `input_mode: pattern-arg`
(`packs/contracts/pack.yml`) only ever supplies a single `--pattern` STRING argument to
ast-grep — never a YAML rule file. A relational YAML rule (`kind: const_spec` + `has:
field: name` scoped match) CAN precisely distinguish a true iota-member declaration from
a mere same-named reference elsewhere (verified directly with `ast-grep scan --rule`) —
so the capability gap is a substrate/input-mode limitation, not an ast-grep
expressibility dead-end. Any future fix needs a new input_mode (rule-arg or similar)
before an iota-aware compiler mode has anywhere to route its output.

**How to apply:** when scoping contracts-pack work, remember `pattern-arg` is
string-only — a rule-file/relational-match capability is a distinct, not-yet-built
input_mode. An audit done for ISSUE-037 (grep `kind: constant` across all specs/issues)
found `CheckTypeFindings` is the ONLY bare-iota-member contract in the repo today — no
other latent instances are waiting to surface, so this is low urgency until a second
instance appears. See [[project_thin_executor_dogfood]] for the sibling
pattern-arg-dependent ISSUE-024.
