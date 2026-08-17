---
name: pack-rule-path-scoping-dispatch
description: A pack rule whose paths.include names directory-prefixed globs matches ZERO files under the gate's default explicit-file dispatch — it is a silent no-op, ecosystem-wide; never treat such a rule as live evidence
metadata:
  type: project
---

**A pack rule whose `paths.include` names DIRECTORY-PREFIXED globs
(`pkg/gate/*.go`, `cmd/backstop/pack_gate*.go`) matches ZERO files when
semgrep is handed explicit file targets — which is what the DEFAULT
diff-scoped `backstop gate` has always done. Such a rule is dead on the
everyday gate, in every consuming repo.** Single-segment globs
(`*foo*.go`) are unaffected. Filed as ISSUE-151, homed DIR-032 item 20.

**STATUS 2026-08-17: ISSUE-151 is CLOSED — but only the DETECTOR shipped.**
`PLAN-ISSUE-151` added `pkg/packval/pathscope.go`, a phase-2 declaration-shape
check emitting two non-blocking WARNs: `path-scope-dispatch` (slash-bearing
include/exclude = inert) and `path-scope-fixture-mask` (the pack's fixtures are
matched only by its slash-free "hook" patterns, so `pack test` green proves
nothing). **The inert patterns themselves were NOT remediated** — 26 + 4 in
`backstop-ai/backstop-self`, 11 in `cobra-cli-standards` — and that pack-side
cleanup is explicitly "founder-sequenced; entangled with ISSUE-097" in
ISSUE-151's own Resolution, and **still unfiled as an issue**. So: run
`./bin/backstop pack check .backstop/packs/<pack>` to MEASURE darkness in
seconds (I did, 2026-08-17), and never let a single-rule fix quietly become the
vehicle for that unowned batch.

**Corollary that bites remedy proposals:** "widen `paths.include` to the whole
package" is NOT a fix — a full-package glob still contains a `/` and stays
inert. ISSUE-024's own audit note proposed exactly that; it is falsified.
Measured-dark in backstop-self: `no-baked-repo-layout-classification`,
`no-language-literal-on-neutral-spine`, `no-structural-name-split-on-spine`,
`no-pack-name-keyed-capability` (all four also fixture-masked). The three
repo-wide `no-baked-*` rules (tool-exec, tool-command, language-token) declare
no `include` and DO run.

**Why:** semgrep resolves `paths.include` differently against a directory
target (satisfied) than against an explicit file list (unsatisfied). Not a
core bug — core shapes the args identically. `PLAN-ISSUE-091` collapsed
`--all` onto the same explicit-file dispatch the diff scope already used,
which is what made the pre-existing hole VISIBLE; it did not create it.
Any fix belongs on the pack-authoring side (how `paths.include` is written,
or how packval validates it), never core path-rewriting / an `--include`
flag / a directory-target fallback — those re-introduce the two-code-paths
disease PLAN-ISSUE-091 existed to cure.

**How to apply:**
- **Never accept "this rule enforces X" without checking its `paths.include`
  shape.** A directory-prefixed include means the rule has probably never
  fired on a real gate run. This bites any claim of the form "the gate
  already catches that" and any severity/coverage estimate built on one.
- Corroboration and the best fix seam live in `ciGlobScopingProblems`
  (`cmd/backstop/ci_recipes_harness_test.go`) — it **enforces** the rule
  (flags any include containing `/`), but only over `ci-workflows` recipe
  include sets. **Nothing scans pack rule files the same way.** Read the
  function body, not just its doc comment.
- Repro (semgrep may not be approved in headless runs — say so rather than
  implying you measured): one `--config <abs path>` per `rule_path` the
  pack declares, built as a SHELL ARRAY and expanded `"${CONFIGS[@]}"` —
  zsh word-splitting turns a space-joined string into
  `semgrep: error: File name too long`. Run once per dispatch shape, diff
  the rows carrying no `suppressions` entry (the ACTIVE layer — what
  `parseSarif` keeps and the gate prints).
- Coupling: backstop-self's own `no-structural-name-split-on-spine` is the
  measured instance, and its two dark findings are the only thing keeping
  ISSUE-097's stale `backstop/self/...` waiver tokens in `Adjudicate`'s
  harvest window. Fixing this rule turns both into ACTIVE findings those
  fail-open tokens will not suppress.

See [[project_gate_verdict_honesty_cluster]],
[[project_mechanism_vs_ecosystem_gap]],
[[project_substantiveness_extraction_semantics]].
