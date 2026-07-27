---
title: "Core CI Is a Hand-Baked Go Pipeline, Not `backstop gate` — and the Published Baseline Is Generated Packless"
schema_version: issue/v1

issue:
  id: ISSUE-086
  title: "Core CI Is a Hand-Baked Go Pipeline, Not `backstop gate` — and the Published Baseline Is Generated Packless"
  type: technical-debt
  status: open
  created: "2026-07-27"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# Core CI Is a Hand-Baked Go Pipeline, Not `backstop gate` — and the Published Baseline Is Generated Packless

## Problem

This is a **known, founder-acknowledged launch checklist item**, not a fresh discovery. Founder,
2026-07-27, on being shown these two facts: "yes i am aware of that. that'll be a launch checklist
item." This issue is the durable tracker for that checklist item — filed so the gap has an ID,
dependencies, and a resolution path instead of living only in a chat transcript.

backstop-core's own `.github/workflows/ci.yml` — the CI pipeline for the project whose thesis is
**zero baked language/tool knowledge** (`CLAUDE.md`, "What backstop IS") — never invokes
`backstop`. It is two coupled facts, both verified against the current workflow file:

### Fact 1 — the `gate` job hand-bakes the Go toolchain instead of calling `backstop gate`

`ci.yml:13-14` names the job `gate` ("Lint + Test + Coverage"), but nothing in it runs the
`backstop` binary. Every step is a raw, hand-rolled Go invocation:

- `ci.yml:24` — `go tool golangci-lint run ./...`
- `ci.yml:27` — `go test -race -coverprofile=cover.out -covermode=atomic ./...`
- `ci.yml:29-37` — a hand-rolled bash coverage-threshold check: extracts `total:` from
  `go tool cover -func=cover.out`, compares against a hardcoded `THRESHOLD=90`, and fails the
  step with `::error::` if under.

This is exactly the pattern CLAUDE.md calls a defect to eradicate: "a baked Go path AND a baked
TypeScript path are BOTH violations." The job that gates every PR into the project that polices
baked toolchains is itself a baked toolchain, unmediated by any pack.

This is also the root cause of why **ISSUE-020**'s Linux sandbox no-op has stayed invisible in
CI: `pkg/packval/sandbox.go`'s `linux` branch returns `"sandbox unavailable on linux in this
build"` for both `SandboxedRun` and `SandboxedRunStdout`, but since the `gate` job never runs
`backstop gate` (or anything that exercises the pack/sandbox path) on its `ubuntu-latest` runner,
no CI run has ever hit that code path. A real gate-driven CI job on Linux would have surfaced
ISSUE-020 as a red CI run, not left it to be found by inspection.

### Fact 2 — the published baseline artifact is generated packless

`ci.yml:39-64` (the `baseline` job, gated to `push` on `main`, `needs: gate`) builds the CLI
(`ci.yml:51-52`) and runs `./backstop baseline generate` (`ci.yml:54-58`) directly — there is
**no `pack install` (or equivalent) step anywhere in this job**. `.backstop/packs/` is gitignored
(`.gitignore:41`, consistent with the pack-distribution model: packs install like `node_modules`,
`backstop.lock` is the durable record — see CLAUDE.md "Packs"). With no install step, the
directory backstop's pack-engine dispatch reads from is simply absent on the baseline runner.

The result: `backstop-baseline-v1` (published via `actions/upload-artifact@v4` at
`ci.yml:60-64`, path `.backstop/baseline.json`) is generated with **zero pack engines present**.
Every pack-engine-sourced finding (semgrep/ast-grep dims, anything routed through
`dispatchPackEngines`) is structurally absent from the published reference baseline — not because
the codebase is clean on those dims, but because nothing ran them.

This bears directly on **DIR-003**'s pull model: DIR-003 (Baseline Implementation) presumes CI
generates a baseline that gate can trust and pull from; a packless baseline is not that — it's a
partial baseline masquerading as the full one. It also bears on the pending
coverage-baseline-refresh sequencing noted in project memory
([[project_baseline_design]]/[[project_gate_dogfood_mostly_dark]]): refreshing the tracked
baseline against this packless artifact would ratchet the project against a vacuum on every
pack-engine dimension, silently declaring those dims "clean" when they were never evaluated.

## Dependencies and path to resolution

Fixing Fact 1 properly — wiring `backstop gate` into the `gate` job so CI runs the same kill
chain as a local gate — is **blocked by ISSUE-020**: the sandbox hard-errors on Linux, and
`ubuntu-latest` is the current runner (`ci.yml:15`), so a naive swap would turn every convert-step
or sandboxed-validator-bearing pack red for the wrong reason (sandbox unavailability, not a real
finding). ISSUE-020's own resolution path routes through **BUNDLE-021**'s OQ-2 ("should packs
declare behavior instead of names, so core can derive a sandbox profile") and OQ-3 ("should the
sandbox extend to engine commands, not just convert scripts") — both open as of this writing
(`bundles/BUNDLE-021-pack-command-execution-governance.bundle.md:338,359`).

Interim options exist and don't require waiting on BUNDLE-021 to resolve:

- Switch the `gate` job to a `macos-latest` runner, where `sandbox-exec` is available today
  (`pkg/packval/sandbox.go`'s `darwin` branch), and run `backstop gate` for real.
- Stay on Linux, add a `pack install` step, and run `backstop gate` with sandbox-dependent
  dimensions explicitly warned/skipped rather than silently absent — loud, not vacuous, per
  [[feedback_loud_not_blocking]].

Either interim path also fixes Fact 2 as a side effect: once the `gate`/`baseline` jobs install
packs, `baseline generate` runs with engines present and the published artifact stops being
packless.

There is a REQ-018 recursion worth naming explicitly: whichever pack ends up providing the CI
recipe/workflow for `backstop gate` in Actions, backstop-core's own `ci.yml` is that pack's most
visible first consumer — this repo dogfoods its own CI-integration story before any external
adopter does.

## Cross-references

- **ISSUE-020** — Linux sandbox hard error; blocks the proper fix for Fact 1, and this issue's
  Fact 1 is the reason ISSUE-020 stayed invisible in CI for as long as it did.
- **ISSUE-084** — published pack *repos* have no CI (`pack check`/`pack test` on push); a sibling
  gap at the pack-repo layer to this issue's gap at the core-repo layer. Different surface, same
  category of thesis violation (CI that doesn't exercise backstop's own verification).
- **DIR-001** (Release Workflow) — a self-gating release pipeline depends on CI actually running
  `backstop gate`, not a hand-baked substitute; this issue blocks that DIR-001 dependency being
  true.
- **DIR-003** (Baseline Implementation) — depends on the published baseline being a trustworthy,
  full-scope artifact; Fact 2 breaks that assumption today.
- **BUNDLE-021** (Pack Command Execution Governance) — OQ-2/OQ-3 are the upstream open questions
  that ISSUE-020's proper fix routes through, which in turn gates this issue's proper fix for
  Fact 1.

## Verification

No code change accompanies this filing — it is the checklist tracker. Verification criteria for
the eventual fix (recorded here so `ready`-promotion inherits a target, not a blank page):

- `.github/workflows/ci.yml`'s `gate` job invokes `backstop gate` (or equivalent) rather than raw
  `golangci-lint`/`go test`/hand-rolled coverage bash, on a runner where the sandbox is not a
  no-op for the packs in use.
- The `baseline` job installs packs before `backstop baseline generate` runs, so
  `backstop-baseline-v1` reflects pack-engine findings rather than an empty engine set.
- A subsequent gate run against the refreshed baseline does not silently ratchet-clean any
  pack-engine dimension that was previously unevaluated — any newly-surfaced findings from
  now-present engines are visible, not swallowed by the baseline swap.
