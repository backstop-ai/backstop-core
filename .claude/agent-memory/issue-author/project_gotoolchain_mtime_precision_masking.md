---
name: project-gotoolchain-mtime-precision-masking
description: darwin /bin/sh mtime comparisons round to whole seconds, Linux dash rounds to nanoseconds — a backwards or fragile mtime check in a go-toolchain pack script can pass locally (coincidental tie) and be a complete no-op on real Linux CI
metadata:
  type: project
---

ISSUE-179 (filed 2026-08-19): `backstop-ai/go-toolchain` v1.7.0's coverage-reuse mechanism
(`scripts/coverage-produce.sh`, shipped by PLAN-ISSUE-172 to eliminate a redundant `go test ./...`
run) had a backwards mtime comparison (`[ ! cover.out -ot "$stamp" ]` when `cover.out` is always
written BEFORE the stamp is touched — the condition can essentially never be true). It "worked" on
darwin only because macOS's `/bin/sh` `test -ot`/`-nt` compares at whole-SECOND resolution, so a
few-millisecond real gap rounds to a tie that happens to satisfy the backwards check. Ubuntu's
`/bin/sh` (dash) compares at full nanosecond resolution and correctly detected the true ordering
every time — reuse never fired, and CI gate time stayed at ~602500ms instead of collapsing to
~2211ms.

**Why:** the pack's own verification for PLAN-ISSUE-172 was darwin-only (local measurement), and
darwin's coarse clock structurally cannot distinguish a correct mtime comparison from a backwards
one when the real gap is sub-second — which it almost always is for two commands run back-to-back
in the same script. This is a real precedent, not a one-off: this pack repo will keep shipping
shell scripts with mtime-based caching/staleness logic (coverage reuse, provenance timestamps,
etc.), and darwin-side testing alone cannot catch a backwards or coarse-tie-dependent comparison.

**How to apply:** when reviewing or filing issues against `go-toolchain` (or any pack) fixes that
were verified only on darwin and involve file mtime comparisons (`-ot`, `-nt`, or arithmetic on
`stat` timestamps) in shell scripts, treat "confirmed on darwin" as insufficient evidence — ask
whether it was also confirmed on real Linux CI, since that is the only environment that exercises
full-precision comparison. If filing a related follow-on, cite this pattern rather than
re-deriving it. See [[project_thin_executor_dogfood]] for other go-toolchain pack precision fixes
that shipped as version bumps rather than core changes (ISSUE-129, ISSUE-135, ISSUE-145).
