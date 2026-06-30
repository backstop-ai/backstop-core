---
name: astgrep-pack-convert-script-scope
description: Any pack dispatched through the REAL ast-grep findings path needs its own ast-grep/to-sarif.sh convert script in scope; dispatch hard-errors "broken pack: missing convert script" if absent
metadata:
  type: project
---

Every ast-grep FINDINGS pack that rides the real `dispatchPackEngines` path MUST carry its
own convert script (the `convert:` binding, conventionally `ast-grep/to-sarif.sh`) inside the
pack directory. The dispatch resolves it at `cmd/backstop/pack_gate.go:414-416` relative to
packRoot and HARD-ERRORS `broken pack <name>: missing convert script <path>` if the file is
absent or a dir. Precedent: existing testdata ast-grep packs each ship their own
`ast-grep/to-sarif.sh` (e.g. `cmd/backstop/testdata/astgrep-multirule/.../ast-grep/to-sarif.sh`).
There is NO shared/global converter — it is per-pack.

**Why:** caught reviewing PLAN-SPEC-037 v1.2.1. The plan created the convert script ONLY for
the installable pack B (`packs/substantiveness/ast-grep/to-sarif.sh`) but omitted it from the
two testdata packs (`pkg/gate/testdata/substantiveness-pack/` and `.../ts-proof-pack/`) that
Phase-3 (Q1 real ast-grep, Go + TS) and Phase-4 (strangler real ast-grep) dispatch through the
REAL engine path. Those packs' `pack.yml` would declare a `convert:` binding pointing at a
script no task creates → dispatch fails "broken pack: missing convert script", silently
defeating every real-ast-grep claim that runs over the testdata fixture packs.

**How to apply:** when a plan authors an ast-grep findings/extraction pack AND any task
dispatches it through the real engine (TestQ1_*_RealAstGrep, strangler, E2E), verify a
`to-sarif.sh` (or whatever the manifest's `convert:` names) is in some task's `files` scope for
THAT pack dir. A pack that is "run for real" but missing its convert script is a runtime
blocker, not a unit-test gap. Related: [[project_pack_provisioning_integration_gap]] (the
testdata-as-production / real-over-installed-pack family).
