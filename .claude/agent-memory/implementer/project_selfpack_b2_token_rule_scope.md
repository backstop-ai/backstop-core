---
name: selfpack-b2-token-rule-scope
description: RESOLVED for B2 (it now excludes *_test.go); Family A no-baked-tool-exec is now the only self-pack family without a test exclusion, and its escape is the in-package execCommand parametric dispatch, not a nosemgrep
metadata:
  type: project
---

STATUS CORRECTED 2026-07-28. `backstop-ai/backstop-self` Family B2
(`no-baked-language-token`, `rules/no-baked.yml`) HAS carried
`paths.exclude: ["*_test.go"]` since the ISSUE-087 TASK-016 precision fix — the
pack comment cites 16 measured rows, all in `*_test.go`, none a routing defect.
The old advice (line-scoped nosemgrep for a test naming `go.mod`) is therefore
obsolete for B2; do not reach for it.

**The family that still lacks a test exclusion is A — `no-baked-tool-exec`.**
It excludes only `tests/smoke/**` (founder-ratified: that IS the binary-building
harness). It matches `exec.Command("$TOOL", ...)` and
`exec.CommandContext($CTX, "$TOOL", ...)` where `$TOOL` is a literal outside
`git|gh|sh|/bin/sh|sandbox-exec` — so ANY test outside tests/smoke that builds
or drives the Go toolchain (`exec.Command("go", "build", ...)`) goes blocking the
moment its file enters diff scope. Several such call sites are dormant today
(`pack_authoring_loop_test.go:18`, `version_test.go:169`).

**Why:** a test that compiles the binary under test is naming its own subject,
not baking routing into the shipped binary — the same rationale the pack wrote
for tests/smoke — but Family A has no path scope to tell the two apart.

**How to apply:** the in-repo escape is the package-level parametric dispatch
helper `execCommand(name string, args ...string) *exec.Cmd`
(`cmd/backstop/root_test.go:360`), already used for a `go build` at
`root_test.go:342` and gate-clean. Route the call through it — the rule's own
message prescribes parametric dispatch. No nosemgrep, and never obfuscate the
literal. The durable fix is a pack precision change (extend Family A's exclude
the way B2 and tests/smoke were extended), which is a cross-repo change in
`backstop-ai/backstop-self` and belongs to that pack's lane, not a consumer's.
Related: [[feedback_gostandards_rule_mechanics]],
[[project_editing_file_pulls_it_into_gate_scope]],
[[project_packless_baseline_fails_at_pack_loading]].
