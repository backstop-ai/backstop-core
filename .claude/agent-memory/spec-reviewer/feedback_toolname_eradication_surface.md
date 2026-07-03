---
name: toolname-eradication-surface
description: Tool-name eradication specs under-count by ~3 site classes beyond the renamed identifier — grep the literal string AND name-keyed maps, not just the symbol
metadata:
  type: feedback
---

When a spec claims to eradicate a tool name (e.g. CheckTypeSemgrep / "semgrep") so
backstop "knows zero tool names," the enumerated rename footprint reliably under-counts.
Grep the LITERAL string across non-test source, not just the Go identifier, and classify
every hit into: (1) the renamed identifier, (2) tool-name STRING used as a routing/dispatch
discriminator, (3) name-KEYED maps (engine→contract, engine→claim-code), (4) tool-named
CONFIG keys.

**Why:** SPEC-035 re-review (2026-06-21). The corrective commit fixed the CheckTypeSemgrep
identifier count (verified exact vs live grep) AND the String()/parseCheckType string
surface — but still missed: pkg/check/manifest.go `hasSemgrepSignal` (`r.Enforcement ==
"semgrep"` routing discriminator in the SAME file the spec edits), `DefaultFieldContracts`
+ `engineFieldClaim` (engine-name-keyed maps), and the `SemgrepVersion`/`PinnedSemgrepVersion`
config key. Each was neither in spec scope nor delegated to the sibling issue. Builds on
[[rekey-faithfulness]] — same root cause: trusting the spec's enumeration over a live grep.

**How to apply:** For any "knows zero tool names" / rename / eradication spec, run
`grep -rn '"<tool>"' --include=*.go <pkgs> | grep -v _test` and partition EVERY hit against
the spec's REQs. A hit in neither the spec nor a named sibling artifact is a crack. Also
check sibling-artifact LINE COLLISIONS: a rename spec and a deletion issue touching the same
lines (e.g. `delete(opts.Executors, CheckTypeSemgrep)`) need a stated sequencing dependency.
