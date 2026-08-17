---
name: pack-copies-and-stale-gate-binary
description: Editing a local pack means syncing 3-4 copies; and `backstop gate` uses the stale system binary, not your source
metadata:
  type: project
---

Two non-obvious gotchas when a change touches pack content or gate/dispatch logic (hit while implementing ISSUE-062).

**A local pack has multiple on-disk copies that must be kept in sync.** For `backstop/substantiveness` the copies are: the tracked SOURCE (`packs/substantiveness/`), the gitignored INSTALLED copy the gate actually reads (`.backstop/packs/backstop/<pack>/`, dispatched from `filepath.Join(projectRoot,".backstop","packs")`), and per-test copies under `pkg/gate/testdata/{substantiveness-pack,ts-proof-pack}/` that the pkg/gate real-engine tests (`dispatchAstGrepRule` / strangler harness) run against. `backstop/self` source lives in a SEPARATE repo (`~/src/projects/backstop-self-pack/`) with its own installed copy under `.backstop/packs/backstop/self/`. After editing an installed copy, run `backstop pack relock .backstop/packs/backstop/<pack>` or `pack_lock_verification` reds on the content-hash drift. Editing only the source and forgetting the installed/testdata copies makes pkg/gate tests and the gate diverge.

**Why:** the convert script + rule messages are duplicated across all copies; a change to one silently leaves the others on the old behavior.
**How to apply:** when a plan scopes only `packs/<x>/...`, also update the installed `.backstop/packs/` copy (+relock) and any `pkg/gate/testdata/*-pack/` copies, or tests/gate will fail inconsistently.

**`backstop gate` runs the STALE system binary at `/usr/local/bin/backstop`, not your working-tree source.** After changing anything under `cmd/backstop` (or gate/dispatch logic the CLI links), the installed binary lags — a gate run can FALSELY red (e.g. ISSUE-062: I changed pack messages to drop `func=`/`symbol=`; the stale binary's old message-parsing join then read empty func/symbol and raised 3 phantom `noTarget` violations). Verify with a freshly built binary: `go build -o <scratch>/backstop-new ./cmd/backstop && <scratch>/backstop-new gate --all --json`. Overwriting `/usr/local/bin/backstop` is blocked by the permission classifier (out-of-repo system path) — build to scratch and report that the founder must rebuild/reinstall.

**Why:** `go test ./cmd/backstop` compiles from source (correct), but the standalone `backstop` binary is only as new as the last `go build -o /usr/local/bin`.
**How to apply:** never trust a `backstop gate` red until you've re-run it with a source-built binary; a phantom regression that only the installed binary shows is a staleness artifact. See [[netnegative-gate-baseline]].

**PROVE staleness in one command with a string probe — don't infer it from mtimes.** Pick a
string literal your change introduces (an error message is ideal) and grep both binaries:

    strings <scratch>/backstop-new       | grep -c "missing findings producer script"   # 1
    strings /usr/local/bin/backstop      | grep -c "missing findings producer script"   # 0

That is a direct, non-circular answer to "does the binary on PATH contain my change", and it
takes seconds. Measured 2026-08-16 during ISSUE-067: `which -a backstop` resolved to
`/usr/local/bin/backstop -> ../Cellar/backstop/0.1.0/bin/backstop`, built ~3 weeks earlier, so
EVERY lane's `backstop gate` that night was running old code. The staleness is repo-wide and
affects concurrent agents too — worth broadcasting, not just working around locally.

**A second environment gotcha compounds it:** a `go-arch-lint not found on PATH` pack_engines
failure HALTS the gate at step 3 in ~1ms, so nothing downstream runs and the result says
nothing about your change. That is the agent shell missing GOPATH/bin, not a finding — prefix
`PATH=/Users/bmanson/go/bin:$PATH`. See [[agent-shell-path-misses-gopath-bin]].
