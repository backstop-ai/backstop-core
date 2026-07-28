---
name: new-file-coverage-floor
description: A plan phase that "opens" a new tiny file (a couple of typed errors) reds coverage_threshold — the 80% floor applies per-file to any file in diff scope, so 1-of-2 statements = 50% = FAIL
metadata:
  type: project
---

`coverage_threshold` is PER FILE at 80%, and a brand-new file in gate diff scope is
measured immediately — so a plan that deliberately opens a file early with only part of
its eventual contents will red the moment one of its few statements is unexercised.

Concrete case (SPEC-055 TASK-024): `pkg/pack/distribution/command.go` was opened in
phase 6 with exactly two typed errors so the resolver's fail-closed constructor had one
to return. Only `MissingDependencyError.Error()` had a caller; the consumers that return
`CapabilityUnavailableError` land in phases 7/9. Result: `command.go coverage 1/2
(statement) below threshold 80%`.

**Why:** the gate does not know the file is mid-construction, and it should not — that is
the anti-vacuous-green property. Waiving would grandfather a brand-new file.

**How to apply:** when a task opens a file whose consumers arrive in a LATER phase, add a
direct test for the orphaned surface in a file your task already owns, and say in a
comment why it lives there. Do NOT create the file the later task declares (e.g.
`command_test.go` belonged to TASK-026) — that collides with a sibling's scope. See
[[gostandards-rule-mechanics]].
