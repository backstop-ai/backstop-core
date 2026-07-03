---
name: spec048-dispatch-fix-pass
description: SPEC-048 findings-engine dispatch fix (self-target + stdout_artifact) reviewed PASS; the real-runner e2e finally closed the recurring pack-provisioning integration gap
metadata:
  type: project
---

SPEC-048 (BUNDLE-012 follow-on) reviewed PASS 2026-06-30. Two proven `runFindingsEngine` fixes in cmd/backstop/pack_gate.go: DEFECT-1 project-wide empty-ProjectTarget self-targets (append nothing, not projectRoot); DEFECT-2 honor `stdout_artifact` FILE as payload for BOTH shape-guard and convert, fail-loud %w on missing. Plus REQ-004a coverage %v->%w and REQ-004b SPEC-040 phantom-contract realign.

**Why notable:** this is the FIRST spec to actually discharge [[project_pack_provisioning_integration_gap]] the RIGHT way — a REAL `check.ExecCommandRunner{Dir: projectRoot}` over a committed POSIX fake-engine.sh (arg-sensitive: suppresses its finding if any arg is bolted on; writes SARIF to the artifact FILE, noise to stdout; records argc.txt). I independently confirmed the e2e FAILS pre-fix AND depends on each fix independently (reverting either alone reds the seeded test). That arg-sensitive + argc-recording fake is the exemplar to demand on future dispatch e2es.

**How to apply:** when reviewing pack-dispatch work, require a real-runner-over-committed-fake e2e that fails if you mentally revert either fix; a canned-stdout fixtureRunner masks both defect classes. Verify the trust gate isn't bypassed — here checkEngineToolAllowed no-ops legitimately because the fixture declares no `provision:` block (not a hack).

Gate was RED but proven net-negative (HEAD 152 -> 144, net-new 0). The lone pack_gate.go "new" finding is the missing-origin/main-baseline artifact on the PRE-EXISTING intentional unwrapped `*check.ConfigError` passthrough (return nil, gateErr — must stay unwrapped for the exit-2 type assertion; identical at HEAD lines 363/544/714). test_verification's 4 fails are pre-existing SPEC-040 phantom mandated tests (new=0), out of scope.
