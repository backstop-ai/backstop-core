---
title: "No automated guard keeps the go-toolchain pack fixture in sync with the released pack; a parallel documentary copy is dead code"
schema_version: issue/v1

issue:
  id: ISSUE-137
  title: "No automated guard keeps the go-toolchain pack fixture in sync with the released pack; a parallel documentary copy is dead code"
  type: technical-debt
  status: open
  created: "2026-08-16"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# No automated guard keeps the go-toolchain pack fixture in sync with the released pack; a parallel documentary copy is dead code

## Problem

Two artifacts in this repo are meant to describe the SAME external thing — the released
`backstop-ai/go-toolchain` pack's engine bindings — and nothing automated asserts they agree:

1. The RELEASED pack, consumed via `backstop.lock` and installed at
   `.backstop/packs/backstop-ai/go-toolchain/pack.yml`.
2. The IN-REPO FIXTURE COPY at
   `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml`, which
   `goToolchainManifest()`/`goToolchainPackRoot()`
   (`cmd/backstop/pack_gate_gotoolchain_test.go:87,114`) actually parse. Every SPEC-041
   exemption test reads THIS fixture, not the installed pack — flipping only the released pack
   leaves the test corpus asserting the OLD behavior; flipping only the fixture ships nothing.

This is not hypothetical drift risk — it is the CONFIRMED maintenance model today. During
`PLAN-ISSUE-129` (the go-test `exempt_from_scope_filter` fix), the two files were compared
BY HAND, once, and found in sync only after a manual review pass (the plan's own task text says
so explicitly: "verified — sweeping every `pack.yml` and every reference to
`scripts/test-to-sarif.sh` in the worktree"). Re-verified directly for this issue
(2026-08-16): a diff of the two files today shows only `name`/`version` differ (fixture:
`backstop/go-toolchain` v1.1.0; released: `backstop-ai/go-toolchain` v1.4.0) — every engine
binding is currently byte-identical. But nothing CHECKS this automatically on every future pack
release; the next hand-made release-side change (a new engine, a flag flip, a command change)
can land without a matching fixture update, and the test corpus would keep silently asserting
stale behavior against a fixture that no longer reflects what the real pack ships.

A second, related gap: `cmd/backstop/testdata/exempt-matrix-bindings.yml` is a THIRD,
documentary-only copy of a subset of the same engine bindings (`go-build`, `golangci`,
`go-test`, plus a generic `findings` row), introduced alongside SPEC-041/ISSUE-129 to describe
the `exempt_from_scope_filter` matrix in prose-adjacent YAML. Its own header comment says the
in-memory test helper (`exempt_test_helpers_test.go`) is the actual source of truth the matrix
tests build against, and that this file only "documents the matrix." Confirmed by grep
(2026-08-16): zero `.go` files in the repository reference
`testdata/exempt-matrix-bindings.yml` or `exempt-matrix-bindings` at all. It is pure
documentation with no code path reading it and no test asserting it matches the bindings it
claims to mirror — a second instance of the same failure class as the fixture/release pair
above: a human-maintained mirror of ground truth that can drift silently, except this one has no
consumer at all, so drift here is unfalsifiable by construction (nothing would ever fail).

## Why this matters

Both are the same underlying gap: a hand-maintained copy of external or generative truth, with
no automated equality/consistency check, in a codebase whose own stated conventions (fixtures
captured from real output, must-falsify) exist precisely to prevent this class of drift from
becoming invisible. A future pack release changing `backstop-ai/go-toolchain`'s engine bindings
without a matching fixture update would make the SPEC-041 exemption test corpus pass while
testing behavior the real, installed pack no longer has — a false-green risk in the same
gate-verdict-honesty family DIR-032 exists to close, even though the mechanism here (test
fixture staleness) is different from DIR-032's scope-filtering defects. The dead documentary
file is lower-severity (nothing consumes it, so it cannot cause a false pass/fail) but is still
maintenance debt: a reader can be misled into trusting it as ground truth, and it will drift the
moment the real bindings change without anyone noticing, because nothing ever will.

## Solution

Not prescribed here. Two independent directions, since the two gaps have different fixes:

1. **Fixture/release parity.** Add an automated check — a test, a `pack check` rule, or a CI
   step — that asserts the in-repo fixture's engine bindings (or at least the fields the
   SPEC-041 test corpus depends on: command, scope_kind, gate_type, exempt_from_scope_filter,
   convert) match the currently-locked released pack's bindings, keyed off `backstop.lock`'s
   recorded version/hash for `backstop-ai/go-toolchain`. Scope this to the SPECIFIC fixture at
   `cmd/backstop/testdata/go-toolchain/...` confirmed load-bearing by PLAN-ISSUE-129 — do not
   assume every other `testdata/**/go-toolchain/pack.yml` fixture in the repo (e.g.
   `classifier-e2e`, `spec045-discovery`) needs the same treatment; those are older/simpler
   fixtures serving different tests (classification, discovery) and were not established as
   load-bearing for exemption semantics.
2. **Dead documentary file.** Either delete `cmd/backstop/testdata/exempt-matrix-bindings.yml`
   (its own header says it is not the source of truth and nothing reads it), or make it
   load-bearing by adding a test that parses it and asserts equality against
   `exempt_test_helpers_test.go`'s in-memory bindings — whichever the next person picks up
   judges correct. Leaving it as an unread, unverified prose-adjacent file is the worse of the
   two options either way.

## References

- `plans/PLAN-ISSUE-129-go-test-scope-filter-exemption.plan.yml:80-96` — the plan text that
  named both the fixture/release pair and the exempt-matrix-bindings.yml documentary copy,
  confirming the hand-verification model and the "must be updated or it becomes a lie in the
  repo" framing for the documentary file.
- `cmd/backstop/pack_gate_gotoolchain_test.go:87,114` — `goToolchainManifest()`/
  `goToolchainPackRoot()`, the actual consumers of the in-repo fixture.
- `cmd/backstop/testdata/go-toolchain/.backstop/packs/backstop/go-toolchain/pack.yml` — the
  load-bearing fixture.
- `.backstop/packs/backstop-ai/go-toolchain/pack.yml` — the currently-installed released pack.
- `cmd/backstop/testdata/exempt-matrix-bindings.yml` — the dead documentary copy; confirmed via
  repo-wide grep to have zero `.go` referrers.
- Surfaced as a follow-on while investigating ISSUE-129/ISSUE-136 fallout (2026-08-16); not a
  duplicate of either — searched `issues/` and `bundles/` for prior coverage of pack-fixture/
  release drift or dead test-documentation files and found none.
