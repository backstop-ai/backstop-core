---
name: feedback_layer0_tool_missing_masks_gate
description: A Layer-0 assume-present tool missing from the invoking shell's PATH makes `backstop gate` exit 2 at pack_engines and ABORT, so every downstream dimension (test_substantiveness, contract_signature) goes unread — a close-out that stops there records no reading for exactly the dimensions the `implemented` flip exists to activate
metadata:
  type: feedback
---

Found closing out SPEC-037 (2026-08-15). The first post-flip `./bin/backstop
gate` exited 2 with `required tool "go-arch-lint" not found on PATH: it is an
assume-present Layer-0 native tool the project must provide on PATH (backstop
never auto-provisions it)`. The tool was installed the whole time — at
`$(go env GOPATH)/bin/go-arch-lint` — but that directory was not on the
invoking agent shell's PATH.

**Why this is dangerous specifically at close-out.** The gate ABORTS at
`pack_engines`; it does not degrade or skip forward. Everything ordered after
it — `test_verification`, `test_substantiveness`, `coverage_threshold`,
`contract_signature`, `artifact_status_drift`, `requirement_traceability`,
`waiver_resolution` — produces NO result at all. `test_substantiveness` and
`contract_signature` are precisely the two dimensions that only activate at
`status: implemented` (see [[feedback_close_out_must_rerun_gate_after_flip]]),
so an exit-2 here is the one failure mode that silently withholds the only
readings the flip exists to produce. The exit-2 message names a tool, not a
spec, so it reads like an environment problem to shrug at rather than a
verification hole.

**How to apply:** when a close-out (or any gate run whose whole point is a
specific downstream dimension) exits 2 at `pack_engines` for a missing tool,
do NOT record the run, and do NOT report the spec as unverifiable. Check
whether the tool is merely unexposed before concluding anything:
`ls "$(go env GOPATH)/bin"` and `which <tool>`. If it is installed, re-run
with `export PATH="$(go env GOPATH)/bin:$PATH"` — this is a per-invocation
PATH fix, not an environment mutation, and needs no approval. Only if the
tool is genuinely absent is this a real gap worth surfacing. Either way,
never let an exit-2 stand in for a green: read the per-step statuses out of
`gate --json` and confirm the dimensions you care about actually reported.
