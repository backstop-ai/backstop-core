---
name: findings-engine-shaping
description: runFindingsEngine appends projectRoot as a scan target and has no scope-kind/file-mode notion; reusing it for go build/test is more than a crash-guard add
metadata:
  type: feedback
---

When a spec routes native build/test through the pack dispatch's `runFindingsEngine`
(cmd/backstop/pack_gate.go), check the function's INVOCATION shaping, not just its
output/parse path. As of 2026-06-18 it: (1) unconditionally appends `projectRoot` as a
trailing scan target (`cmdArgs = append(cmdArgs, projectRoot)`) — correct for
`semgrep <root>` / `ast-grep scan <root>` but WRONG for `go build ./...` (package pattern,
not a dir) and for file-scoped `go test <pkg>`; (2) has NO scope-kind / file-mode notion,
so it cannot carry the `code check --file` hook's package-scoped `go test`; (3) discards
the tool run error (`_ = runErr`), so a crash-vs-findings guard must be ADDED.

**Why:** SPEC-034's revision correctly reused `dispatchPackEngines` (good — answered the
B1 bridge gap) and correctly specified the crash guard as a claim. But it framed the crash
guard as "the ONE behavioral addition the bridge makes to the findings path," which
understates it: the projectRoot-append is a SECOND behavioral change build/test force. The
spec still passed because REQ-010/CLM-034 independently force file-mode scoping to be
preserved-and-tested, so the behavior can't be silently lost — the gap is
implementation-shaping (planner-level), not a missing/unimplementable requirement.

**How to apply:** When a removal/migration spec says native passes run "through
runFindingsEngine," confirm the function can actually express those passes' invocation
shapes (project-pattern vs scan-dir, file-mode scoping). If it appends a fixed scan target
or lacks scope-kind awareness, flag that the reuse requires modifying the dispatcher's
arg-shaping — and that any "only behavioral addition is X" claim is understated. Pairs with
[[parser-locus-seam]] (the engine-model-vs-native-registry seam this bundle keeps testing).
