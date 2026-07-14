---
name: feedback-new-validation-honors-exemptions
description: A new corpus-wide validation pass must mirror the terminal/live-work exemptions the per-artifact validators already apply, or it creates an incoherent split
metadata:
  type: feedback
---

When a spec adds a NEW validation pass (especially a corpus-wide one like supports
resolution), it MUST honor the same exemptions the existing per-artifact validators
already apply — most importantly the terminal-status exemption (`isTerminalStatus`,
pkg/validate/spec.go:175, pkg/validate/issue.go:349, which skip the whole
completeness block including the supports-format check for
replaced/canceled/deprecated/obsoleted artifacts).

**Why:** In SPEC-050 (BUNDLE-014 Seed 1) the resolution pass initially resolved
refs "at ANY citing status." That would have re-checked refs on terminal artifacts
that the format-checker already exempts — a format-exempt-but-resolution-checked
split that pins retired artifacts red forever. It also broke the sibling seed
(SPEC-051) whose corpus cleanup relies on deprecating stale specs to clear their
dangling refs. The spec-reviewer did NOT catch this; a peer spec-author flagged it
as a cross-seam dependency.

**How to apply:** Whenever a spec introduces a validation/resolution/coverage pass,
explicitly ask "which existing exemptions must this mirror?" — terminal status,
draft-vs-live gating, the maturity gates. State the exemption as requirement-level
behavior (not a note), add a claim proving a terminal/exempt artifact passes, and
add a sharp edge naming any downstream seed that depends on it. "Resolve at any
status" almost always means "any LIVE status; terminal is exempt."
