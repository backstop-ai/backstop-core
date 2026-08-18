---
name: issue163-sandbox-guard-review
description: ISSUE-163 TestMain sandbox-helper guard — code PASS but lane FAIL on undischarged TASK-005; plus two reusable review facts (cmd/backstop suite is unrunnable in-session, go-arch-lint absence makes gate exit 2)
metadata:
  type: project
---

ISSUE-163 / PLAN-ISSUE-163 review (commit `970512b`, 2026-08-18): the two-file code
change is CORRECT and its two AST tests are genuinely discriminating (8 mutations, all
intended reds fired, incl. the deliberate sharp-edge-8b carveout where a bare `126`
literal reds CLM-001's test but correctly leaves the roster test green). Verdict was
FAIL on LANE COMPLETION, not on code: TASK-005 never landed — no adjacent-hole issue
filed for `pkg/pack/distribution` + `pkg/pack/engine` (both import packval, neither has
a TestMain), no ceiling record in the issue artifact, plan left at `status: draft`.

**Why:** a lane whose whole point is an honest verification ceiling is not finished when
only the code half ships. The roster test only covers packages that ALREADY have a
TestMain, so the unfiled adjacent hole is the exact gap CLM-003 cannot pin.

**How to apply:** on issue→plan lanes, check the documentation/handoff task landed, not
just the code tasks — `git show --stat` the fix commit against the plan's `files:` lists.
Two reusable environment facts learned here:
  * `go test ./cmd/backstop/ -race` is effectively UNRUNNABLE in a review session — it
    blows Go's default 10m alarm at ~604s and ran >60min under load before being killed.
    Don't budget for it. Instead note that ANY successful `-run <subset>` proves TestMain's
    build-and-run path, since TestMain `go build`s binaryPath before `m.Run()`.
  * `go-arch-lint` is NOT on PATH on this machine, so `backstop gate` exits 2 with
    `required tool "go-arch-lint" not found on PATH` and `pack_engines` contributes
    NOTHING. Any gate reading taken here is non-authoritative — say so, don't report it
    as clean. (`golangci-lint` IS present at /usr/local/bin.)
  * `GOOS=linux GOARCH=amd64 go vet ./cmd/backstop/...` is the one linux-side fact
    obtainable on darwin for `//go:build linux`-gated work: it proves the edit COMPILES
    against the linux build. It is not runtime verification — never present it as such.

Related: [[project_issue082_allowlist_review]] (the scratchpad-mutation recipe used here),
[[reference_sandbox_verification]].
