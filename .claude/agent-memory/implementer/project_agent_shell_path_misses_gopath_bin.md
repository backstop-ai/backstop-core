---
name: agent-shell-path-misses-gopath-bin
description: A pack_engines "required tool not found on PATH" red can be a phantom — the agent Bash shell omits /Users/bmanson/go/bin, where go-arch-lint and other Layer-0 tools actually live.
metadata:
  type: project
---

`./bin/backstop gate` can fail at `pack_engines` with `required tool "go-arch-lint"
not found on PATH: it is an assume-present Layer-0 native tool...` even though the
tool IS installed. `go env GOPATH`/bin (`/Users/bmanson/go/bin`) holds go-arch-lint,
actionlint, ast-metrics, goreleaser — but the agent Bash tool's shell does not
inherit it. Prefix the run: `PATH="/Users/bmanson/go/bin:$PATH" ./bin/backstop gate`.

**Why:** measured 2026-08-09 during ISSUE-116. Without the prefix the gate HALTS at
pack_engines (step 3 of 14) and never reaches test_substantiveness, coverage,
contract_signature or waiver_resolution — so a "green enough" reading is impossible
and a real finding in your own diff stays hidden. With the prefix the same tree ran
all 14 steps and surfaced a genuine `no-global-mutable-state` finding in the new
test file. CI installs the tool at a pin (`go install
github.com/fe3dback/go-arch-lint@v1.16.0`, .github/workflows/ci.yml:56), so this is
purely a shell-environment gap, not a missing dependency.

**How to apply:** before concluding a tool-not-found gate red is real, run
`ls "$(go env GOPATH)/bin"`. If the tool is there, re-run with the PATH prefix
rather than installing anything or reporting an environment blocker. Related:
[[project_pack_copies_and_stale_gate_binary]] (the other phantom-red source — always
rebuild `bin/backstop` first), and [[feedback_zsh_pipestatus_is_one_indexed]] (report
true exit codes with `cmd; echo $?`, never `${PIPESTATUS[0]}`).
