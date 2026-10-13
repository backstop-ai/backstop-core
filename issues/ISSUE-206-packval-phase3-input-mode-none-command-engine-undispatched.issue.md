---
title: "pack test phase 3 resolves engine inputs differently from the gate — vacuous passes, false failures and argv mismatches"
schema_version: issue/v1

issue:
  id: ISSUE-206
  title: "pack test phase 3 resolves engine inputs differently from the gate — vacuous passes, false failures and argv mismatches"
  type: bug
  status: open
  created: "2026-09-27"

complexity:
  scope: cross-cutting
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-206: pack test phase 3 resolves engine inputs differently from the gate — vacuous passes, false failures and argv mismatches

## Problem

`backstop pack test` phase 3 (fixtures) decides **whether** to run a rule's engine, **with what
first argument, flag and argv**, using its own logic in `pkg/packval/phase3.go` and
`pkg/packval/executor.go`. The real gate decides the same things in `cmd/backstop/pack_gate.go`
(`gatherEngineInputs`, :557-622, plus the sandbox validator resolution at :1133-1136). The two
implementations diverge, so the same pack gets different verdicts from `pack test` and from
`backstop gate`. Three classes of divergence, each a symptom of the same root cause:

- **Vacuous passes.** `pack test` reports `pass` — including for the negative fixture that should
  require at least one finding — while the engine or validator is **never invoked**.
- **False failures.** `pack test` fails (and `pack add` refuses) a pack the gate runs normally.
- **Argv / config mismatches.** `pack test` runs the tool with arguments the gate never passes, so
  the fixtures exercise a different invocation than production.

`checks` stays at its fixed literal count in every case, so nothing in the CLI's output
distinguishes a real pass from a vacuous one.

### Evidence matrix

Reproduced 2026-10-12 with a branch-built CLI and an instrumented tool that logs its own argv
(the reviewer's matrix — not a full cross-product of `input_mode` × rule input × validator presence, but every cell found to diverge; `input_mode` is one of `config-file`, `rule-flags`, `none`,
`pattern-arg`, `pkg/pack/engine/binding.go:17-45`). "Gate" is the behaviour of
`gatherEngineInputs` / the sandbox path, code-verified; the sandbox row was also confirmed with a
real gate run (exit 2).

| # | Declaration | `pack test` (phase 3) | Gate |
|---|---|---|---|
| **Vacuous passes** | | | |
| A1 | `none` + `command`, rule declares no input | `pass`, tool never invoked | runs the tool, files appended |
| A2 | `config-file` + `input_flag`, no `rule_path` | `pass`, tool never invoked | runs the tool, no flag, no config (`nil, nil`, :572) |
| A3 | `rule-flags`, no `rule_path` | `pass`, tool never invoked | **error** "broken pack ...: rule ... declares no rule_path" (:620-622) |
| A4 | `pattern-arg`, neither `pattern` nor `rule_path` | `pass`, tool never invoked | **error** "broken pack ...: pattern-arg rule ... declares no pattern" (:604-605) |
| A5 | sandbox engine (`none`, `command: ""`), rule with no `validator` | `pass` in every phase, nothing run | **error** "broken pack ...: missing validator" (:1133-1136) |
| A6 | `none` + `command` + `validator`, no rule input | runs **only the validator**; the tool is never invoked (no tool log) | runs the tool and **ignores** the validator (the only `.Validator` read in `cmd/backstop` is `pack_gate.go:1133`, sandbox branch) |
| A7 | non-sandbox `input_mode` (`config-file`, `rule-flags` or `pattern-arg`) with `command: ""` | **correct side:** with a rule input it **fails loud** ("engine binding has no command to run", `executor.go:76`); with none (config-file, no `rule_path`) it is skipped and passes | **passes clean, nothing runs**: `pack_gate.go:298` treats only `none` + empty command as the sandbox branch, so this takes the findings path; `splitCommand("")` yields an empty name, exec fails with "exec: no command", which is not an `*exec.Error`, so `check.NeverStarted` (:876) does not fire and the empty stdout reads as clean |
| **False failures** | | | |
| B1 | `none` (with or without `command`), rule with only `pattern:` — including a sandbox engine with a validator | **fail** (`pattern-arg-input`, phase3.go:65-68); `pack add` refuses | runs the tool, or the sandbox validator, normally |
| B2 | `config-file`, rule with only `pattern:` | **fail**, same error | runs the tool normally |
| B3 | sandbox engine + `validator` + `rule_path`/`file` | **fail**: sends it to `RunEngine` with an empty command — "engine binding has no command to run" | ignores `rule_path`, runs the validator |
| **Argv / config mismatches** | | | |
| C1 | `config-file` + `file: cfg.yml` (alias, no `rule_path`) | injects `--config cfg.yml` | injects nothing — the gate's rule model reads only `rule_path` (`pkg/pack/manifest.go:161`) |
| C2 | `rule-flags` + `file: cfg.yml` | dispatches with `--config cfg.yml` | **error** (no `rule_path`) |
| C3 | `none` + `rule_path: cfg.yml` | passes the rule path as argv[0] | never passes it (`none` injects nothing) |

A7 is the one row where **phase 3 is the correct side**. It is covered by neither ISSUE-206's other
rows nor ISSUE-207: 207 is about an engine that *starts* and then fails, whereas here the process
never starts but `check.NeverStarted` misses it (the error is not an `*exec.Error`). Evidence
(reviewer's repro): `rule-flags` + `command: ""` + `rule_path` — `pack test` fails, the gate passes
with no violations; `config-file` + `command: ""` + no `rule_path` — both pass and nothing runs;
control with a real command — the tool is invoked.

Caveat for A6/B1/B3: the repro's validator script itself failed in the scratch environment (likely a
nested-sandbox effect), so the evidence is the engine-error / `pattern-arg-input` messages and the
absent tool log, not the validator's verdict.

(`rule-flags` with only `pattern:` also errors in phase 3 but for the right reason — the gate
rejects it too — so it is parity, not a divergence.)

**Related, and order-dependent:** ISSUE-207 (the gate runs a pack-relative `command:` script with
the wrong working directory and reads post-start failures as a clean pass). See "Ordering
dependency with ISSUE-207" below — fixing this issue alone is not safe.

### Reproduction (fresh directory, generic names, relative paths from that directory)

```bash
WORK=$(mktemp -d)
cd "$WORK"
backstop pack new --type engine --slug probe-pack --language go
cd probe-pack
```

Replace the sample sandbox engine with a plain command engine and drop the validator:

```yaml
# pack.yml
name: local/probe-pack
version: 0.1.0
language: go
archetype: enforcement
engines:
  probe-pack-engine:
    command: "python3 tool.py"
    input_mode: none
    scope_kind: file-args
    gate_type: findings
content:
  ruleset:
    version: 0.1.0
    rules:
      - id: probe-pack-sample
        engine: probe-pack-engine
        risk_class: correctness
        # no validator:, no rule_path/file:, no pattern:
        claims:
          - id: probe-pack-clm-001
            text: "Sample enforced property."
            fixtures:
              positive:
                - fixtures/valid/example.txt
              negative:
                - fixtures/invalid/example.txt
```

`tool.py` (any interpreter/tool works; this one logs its own invocation and always emits a valid,
empty-results SARIF 2.1.0 document regardless of input, to make non-dispatch unambiguous):

```python
#!/usr/bin/env python3
import sys, json
with open("/tmp/probe-tool-invocations.log", "a") as f:
    f.write("invoked with args: %r\n" % (sys.argv[1:],))
print(json.dumps({"version": "2.1.0", "runs": [{"tool": {"driver": {"name": "probe-tool"}}, "results": []}]}))
```

`fixtures/invalid/example.txt` carries whatever marker/property the (absent) rule content would
have named; it does not matter, because the rule declares nothing to check for.

```bash
rm -f /tmp/probe-tool-invocations.log
backstop pack test . --json
cat /tmp/probe-tool-invocations.log 2>&1 || echo "NO LOG — TOOL NEVER RAN"
```

**Observed** (reproduced directly, 2026-09-27, against the current tree):

```
status: pass
phase3-fixtures: pass   (checks: 5, duration_ms: 0)
```

and `/tmp/probe-tool-invocations.log` is never created — the tool's own instrumentation proves it
was never invoked for either fixture. The negative fixture, which should require ≥1 finding to
pass, was never checked at all.

## Root cause (pinned to file:line)

**One cause: phase 3 has a parallel implementation of engine-input resolution rather than using
the gate's.** The gate's resolution is the single source of truth for what a pack's engine
receives: `gatherEngineInputs` (`cmd/backstop/pack_gate.go:557-572` for the function and its
config-file case, :573-580 rule-flags, :603-606 pattern-arg, :620-622 `resolveRulePath`) plus the
sandbox validator resolution (:1133-1136), operating on `pkg/pack`'s rule model. Phase 3 instead
re-derives eligibility, first argument, flag and argv from a separate rule model and separate code:

- `pkg/packval/phase3.go`, `RunFixtures`:
  - Line 17: `Checks: 5, // semgrep, tool-config, validators, scaffolds, sdk` is a fixed literal,
    never incremented per rule/fixture actually exercised.
  - Line 44: `if ruleSource != "" || rule.Pattern != "" {` is the **dispatch-eligibility gate**.
    Only inside this block does the function resolve the rule's engine binding (`haveBinding`)
    and compute `dispatchInput := rule.RuleDispatchInput(binding.InputMode)` (`phase3.go:56`, via
    `pkg/packval/manifest.go:186-191`). A rule that declares neither a rule source nor a pattern
    never enters the block (A1-A4), whatever the engine's input mode; a comment inside the block
    ("NO SILENT SKIP AT THE WIDENED SEAM") records that fact as if it were a guarantee. Rules
    that do enter it are checked against a *narrower* model than the gate's — a pattern under a
    non-pattern-arg engine is an error (B1, B2, `phase3.go:65-68`), though the gate ignores it.
  - Lines 119 and 147: dispatch is gated `if dispatchInput != "" && haveBinding`, so
    `executor.RunEngine` is never called for the skipped rules; the validator seam
    (`if rule.Validator != ""`, lines 139/162) is also a no-op for a rule with no validator,
    which is why a sandbox engine with no validator (A5) runs nothing and still passes.
  - Line 293 (`if len(res.Errors) > 0 { res.Status = "fail" }`) leaves `res.Status` at its initial
    `"pass"` when no errors were appended.
- **Validator axis.** Phase 3 runs `rule.Validator` for **any** engine (`phase3.go:139/162`),
  whereas the gate reads `Validator` only in the sandbox branch (`pack_gate.go:1133`, the only
  `.Validator` read in `cmd/backstop`) and, for sandbox engines, ignores `rule_path` and `pattern`.
  So a validator on a non-sandbox engine is exercised by `pack test` and never by the gate (A6),
  and a sandbox engine carrying a stray rule input is routed to `RunEngine` or errors in phase 3
  (B1, B3) though the gate just runs the validator.
- `pkg/packval/manifest.go:160-191`: `RuleSourcePath` / `RuleDispatchInput` read the `file`
  back-compat alias as well as `rule_path`, and hand the result to every non-pattern-arg mode as
  the first argument. The gate's model (`pkg/pack/manifest.go:161`) has no `file` key and `none`
  injects nothing (C1-C3).
- `pkg/packval/executor.go:42-52`, `buildEngineArgv`: always appends `binding.InputFlag` when set,
  regardless of whether the gate would emit a flag for that mode and rule set.

**Why the legitimate shapes (A1, A2) must be dispatchable:** `pack_gate.go:298` distinguishes the
sandbox-validator engine (`input_mode: none` **and** `Command == ""`) from a findings engine that
also declares `input_mode: none` but carries a `Command` (anything else with a non-empty `Command`
rides the ordinary findings dispatch, which runs the command with the scoped files appended). A
`config-file` engine with no `rule_path` is legitimate for the explicit reason in
`gatherEngineInputs` ("absence is fine (the tool runs its own rules)", :560-561, `nil, nil` at
:572).

This is the "LIES" variant, not the "SHOUTS" variant, in the terms this repo's gate-verdict-honesty
work uses, for A1-A5: `phase3-fixtures` reports a computed, checked `pass` while having executed
zero checks. B1/B2 are the inverse (a loud failure the gate does not share) and C1-C3 are runs of
the wrong invocation.

### Ordering dependency with ISSUE-207

The two dispatch paths run the engine command in **different working directories**: the packval
executor sets `cmd.Dir = packDir` (`pkg/packval/executor.go:121`), while the real gate's runner is
built with `Dir: projectRoot` (`cmd/backstop/gate.go:1131`, inside `packValidatorStep`,
:1130). A pack-relative script named in `command:` (e.g. `python3 tool.py`) therefore resolves
under `pack test` but not under `backstop gate` (ISSUE-207). If this issue is fixed first, `pack
test` will genuinely run the tool, find its script, and pass — while the gate still never runs it
and reads that as a clean pass. That is a more believable green than today's. Sequence the fixes
so ISSUE-207 (fail loud on the gate side, and/or detection at `pack check`/`pack test`) lands
before or together with this one. (Making phase 3 reproduce the gate's cwd is not an option: phase 3
has no project root, and its fixture args — `f.Path` — are pack-relative and resolve only with
`cmd.Dir = packDir`.)

## Impact

`pack test` is the author's only pre-publish check of whether the pack's engines and fixtures
work, and the two paths disagree:

- A pack author who scaffolds a command-based findings engine (the pattern the scaffold's own
  `pack.yml` comment invites — "Swap this for a semgrep/ast-grep/native-tool engine") and declares a
  rule with no `rule_path`, no `pattern` and no `validator` gets an unconditional, permanent
  `pack test` green with neither fixture ever exercised (A1, A2). Packs that the gate would reject
  loudly (A3, A4, A5) also go green.
- A pack the gate runs fine is refused by `pack test` / `pack add` (B1, B2), steering authors to
  restructure a correct pack.
- Fixtures that do run exercise an invocation the gate never uses (C1-C3), so a pass proves
  nothing about production behaviour.

`checks: 5` never changes, so the CLI's own output cannot distinguish any of these from a real pass.

## Direction (not decided, just the shape)

**Phase 3 should derive eligibility, first argument, flag and argv from the same resolution logic
the gate uses — a shared function or equivalent — not a parallel implementation, and fail loud
wherever the gate fails loud.** Patching shape by shape is what produced this matrix. The
resolver — `gatherEngineInputs`, `resolveRulePath`, and the sandbox validator resolution in
`runSandboxEngineWithEvidence` — currently lives in `cmd/backstop` (package `main`), which cannot be
imported, so it must **move into an importable package**. `pkg/pack` is viable: `pkg/packval`
already depends on it, and `pkg/pack` depends only on `pkg/pack/engine` (checked with `go list
-deps`), so there is no import cycle. The two rule models (`pkg/packval.Rule` with its `file`
alias, `pkg/pack.Rule`) must resolve to one meaning. The scope spans `pkg/pack`, `pkg/packval` and
`cmd/backstop`.

**Constraint:** phase 3 must keep `cmd.Dir = packDir` (`executor.go:121`) with pack-relative
fixture paths (`f.Path`); it has no project root, and making it reproduce the gate's cwd is not an
option. Only the *input resolution* is shared, not the working directory. This is the same
ordering point as ISSUE-207 (below): that issue's fail-loud / authoring-time detection should land
before or with this fix so a genuine `pack test` run does not turn into a believable green.

**Acceptance examples** — the shared resolution must satisfy each row of the matrix:

- **A1, A2 — dispatch.** `none` + `Command` and `config-file` without `rule_path` are dispatchable
  with no rule content. `RunEngine`'s calling convention (`[]string{dispatchInput, f.Path}`,
  lines 124/148) assumes a leading rule-content argument, so the call shape must accommodate an
  absent one without a spurious empty-string argument. **The input flag is omitted together with
  the absent config:** `buildEngineArgv` (`executor.go:49-52`) otherwise yields
  `tool --config fixtures/...`, consuming the fixture as the config; the gate emits no flag here.
- **A3, A4 — fail loud.** `rule-flags` with a pathless rule and `pattern-arg` with an empty-pattern
  rule are engine-level phase errors naming the rule, matching `pack_gate.go:603-606` and
  :620-622. Gate semantics are per **engine**: for `config-file` the config comes from the *first*
  rule that declares a `rule_path` (:562-570) and applies to the whole engine run (if sibling A
  declares one and B does not, B runs under A's config, not with none); for `rule-flags` any one
  pathless rule fails the whole engine (:576-580); for `pattern-arg` any one empty pattern does.
  Resolution therefore needs the engine's full rule list, not the single rule being iterated.
- **A5 — fail loud.** A sandbox engine whose rule declares no validator is an error naming the
  rule, matching "missing validator" (:1133-1136).
- **B1, B2, B3 — no false red.** A `pattern:` under `none` or `config-file` (including a sandbox
  engine) is ignored, as the gate ignores it; it must not raise `pattern-arg-input`. **A sandbox
  engine ignores every rule input** (`rule_path`, `file`, `pattern`) and is never sent to
  `RunEngine`; only its validator runs.
- **A7 — empty command on a non-sandbox engine fails loud, in the shared resolution.** The shared
  resolver must **not inherit** the gate's current behaviour here: a `config-file`, `rule-flags` or
  `pattern-arg` engine with `command: ""` is an error naming the engine, in `pack test` and — since
  the resolution is shared — in the gate too, which fixes the gate side (today a silent pass).
- **A6 — validator only where the gate honours it.** A validator on a non-sandbox engine is either
  exercised by the gate too or rejected/flagged by `pack test` (a decision), but phase 3 must not
  report a pass on the strength of a validator alone while the tool is never invoked.
- **C1-C3 — identical argv.** The `file` alias must resolve identically on both sides (either the
  gate reads it too, or phase 3 stops honouring it — a decision, since older testdata packs use
  it); `none` passes no rule path; `config-file` and `rule-flags` pass exactly what the gate would.
- Whether `Checks` should become a real per-rule/per-fixture counter (rather than the fixed
  literal) is a separate, related question worth deciding alongside this fix.

**Acceptance-test notes** (additional cases for the plan, not matrix rows):

- A validator on a non-sandbox engine that *also* has a rule input: phase 3 runs the tool and the
  validator, the gate runs only the tool (the A6 example above already covers the principle).
- An empty `input_flag` with an input under `config-file`, `rule-flags` or `pattern-arg`: the gate
  emits a literal `""` argv element (`pack_gate.go:570`, :590, :607), while phase 3 omits the flag
  (`executor.go:49`). The shared resolution should pick one behaviour (likely fail loud on an
  input without a flag).
- `none` + the `file:` alias puts the path into argv[0] exactly as `rule_path` does (C3).

## Dedupe / existence-in-world check (performed 2026-09-27)

Searched `issues/` and `bundles/` before filing. Related but non-duplicate prior art, all confirmed
distinct by reading each artifact:

- **ISSUE-092** (closed) — fixed phase 3's dispatch guard being keyed off the retired `Layer == 3`
  field instead of `rule.Validator != ""`. Different gating mechanism; already fixed and unrelated
  to the `ruleSource`/`Pattern` eligibility check this issue targets.
- **ISSUE-142** (closed, DIR-032 member 16) — packval's `Rule` type had no `Pattern` field at all, so
  every `pattern-arg` rule was undispatched. Fixed by adding the field and the `rule.Pattern != ""`
  disjunct this issue's `phase3.go:44` condition already contains. This issue generalizes it: the
  `OR` list is one symptom of phase 3 re-deriving the gate's input resolution, not a regression
  of 142's fix.
- **ISSUE-146** (closed) — the scaffold's *sandbox validator* sample (`validator:` + no `command`)
  was a hardcoded always-pass literal. Fixed by making that validator genuinely discriminate. This
  issue's repro deliberately uses the *other* scaffold path (a command engine, no validator at all)
  and is unaffected by that fix — confirmed by reproducing directly, not by comparing prose.
- **ISSUE-160** (closed) — `pkg/packval/executor.go`'s `RunEngine` ignored `CrashGuard`,
  `StrictSarif`, and `Producer`. Different file, different mechanism (fields ignored *inside* a
  dispatched run), and presupposes the rule got dispatched at all — which is exactly what this issue
  says does not happen for this rule shape.
- **ISSUE-151** (closed) — `paths.include`/`exclude` glob patterns invisible under explicit-file
  dispatch; a `pkg/packval/pathscope.go` phase-2 declaration-shape advisory. Unrelated: this issue's
  repro rule declares no `paths.include` at all, and the gap is in phase 3's dispatch eligibility,
  not phase 2's path-scope shape check.
- **DIR-032** ("Gate Verdict Honesty", `status: done`) and **DIR-033** ("Gate Verdict Honesty
  Residual Tail", `status: queued`, 9 named members) — read in full. DIR-032's roster is closed at
  `done` and does not include this defect (its `phase3.go`-adjacent members are 092/142/146/160/151,
  all reviewed above and distinct). DIR-033's nine members (ISSUE-149/150/152/154/155/156/157/159/
  161) are claim-prose retractions, a validator-exemption gap, a fixture-polarity mirror-sync gap,
  and a founder product-tradeoff question — none describe an undispatched command-engine rule.
  Neither directive already owns this surface; recommend a founder/backlog-pm home decision at
  triage time rather than assuming one here.
- `.claude/agent-memory/backlog-pm/project_packval_phase3_family.md` and
  `project_gate_verdict_honesty_cluster.md` (read in full) — name the `pkg/packval` vs
  `cmd/backstop/pack_gate.go` model-drift as a recurring root cause and catalogue prior instances
  (ISSUE-092/140/141/142/143/144/146/151/160 among others); this specific `ruleSource`/`Pattern`
  eligibility gap for an `input_mode: none` command engine is not among them.

No open issue or bundle charter already covers this defect. This is a new filing, not a duplicate.

## Notes

- Reproduced directly against the current working tree (2026-09-27), including instrumenting the
  probe tool to prove non-invocation rather than inferring it from the reported status alone.
- Reframed 2026-10-12 after a review matrix: from a per-shape eligibility gap to the
  shared root cause (phase 3 resolves engine inputs independently of the gate), with the matrix
  as evidence and the per-shape remedies kept as acceptance examples.
- Companion filing from the same investigation session: ISSUE-207 (a different vacuous-pass
  mechanism, at real `backstop gate` dispatch time rather than `pack test`) — and an ordering
  dependency, see above.
- The `config-file` variant was added after independent review (reproduced 2026-10-12 with an
  instrumented tool that logs its own invocation; the log was never created).
