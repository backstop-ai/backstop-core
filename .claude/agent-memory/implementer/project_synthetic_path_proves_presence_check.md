---
name: synthetic-path-proves-presence-check
description: Prove a tool-presence check (and its CI blast radius) by running the real gate under a symlink-farm PATH missing exactly one tool — not by reading the workflow file
metadata:
  type: project
---

To prove a presence/provisioning check actually fires — and to prove the CI
blast-radius claim that "if CI dropped tool X the gate goes loud" — build a
symlink farm in the scratchpad holding every tool EXCEPT the one under test, then
run the real gate with `PATH=$D`.

**Why:** reading `.github/workflows/ci.yml` proves the install step exists TODAY;
it does not prove the gate would actually refuse if the step vanished. The
synthetic PATH exercises the real production path over the repo's real pack set
and yields the verbatim refusal message — which is also the acceptance evidence
for "a human learns WHICH tool is missing". Verified on PLAN-ISSUE-112
(2026-08-16): dropping semgrep produced a named, pinned, actionable refusal
naming argv[0] + engine + declaring pack + Provision.Tool + version.

**How to apply:**
- It is CHEAP even when a full gate is slow: provisioning is an early step
  (pack_engines, step 3), so the run fails in ~2s instead of ~5min.
- Build links with an ABSOLUTE resolved path. `ln -sf "$(command -v grep)" $D/grep`
  is a trap — in zsh `command -v` can return the BARE NAME, producing a
  self-referential symlink (`grep -> grep`) and a refusal for a tool you meant to
  provide. That looks exactly like a real false-refusal defect. Use
  `/usr/bin/which`, or a hardcoded absolute path, and verify with `ls -l $D`.
- Cross-check the negative: with the REAL PATH the same step must PASS, or you
  have proved a false refusal rather than a working check.

See [[project_agent_shell_path_misses_gopath_bin]] — the agent shell's own PATH
gap produces the same-shaped phantom red, so establish a known-green control run
before trusting any absence result.
