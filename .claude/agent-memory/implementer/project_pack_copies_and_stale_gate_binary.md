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
