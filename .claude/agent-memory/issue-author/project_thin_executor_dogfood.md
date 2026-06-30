---
name: project-thin-executor-dogfood
description: ISSUE-024 captures the thin-executor self-enforcing pack rule; blocked on SPEC-035 pattern-arg and BUNDLE-009 OQ-7
metadata:
  type: project
---

ISSUE-024 (thin-executor-absence-rule-dogfood) is filed as `open` technical-debt. It
cannot be planned until two upstream capabilities ship:

1. SPEC-035 `pattern-arg` input mode (REQ-004) — lets the pack supply ast-grep patterns
   inline without a rule-file on disk.
2. BUNDLE-009 OQ-7 resolution — defines absence/forbidden-pattern probe semantics; ISSUE-024
   is a concrete consumer that should inform (not pre-empt) that resolution.

**Why:** The rule must live in the `backstop/go-standards` self-dogfood pack (declared local
in `backstop.yml`), NOT the CLI binary — baking it into the binary is the very sin it catches.
Engine: ast-grep (already wired). Scope: `cmd/backstop/`, `pkg/check/`, `pkg/gate/`,
`pkg/pack/engine/`, `pkg/packval/`. Sanctioned exceptions: `pkg/gate/step_testverify.go`
(own test runner) and `*_test.go` files.

**How to apply:** When the user asks about the thin-executor eradication finish line or sequencing
SPEC-035/BUNDLE-009 work, note that ISSUE-024 is the acceptance-test capstone that closes
the loop — it goes ready once both prerequisites land.
