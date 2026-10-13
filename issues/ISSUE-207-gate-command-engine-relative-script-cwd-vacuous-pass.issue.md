---
title: "Nothing distinguishes a findings engine that crashed after starting from one that found nothing — a non-CrashGuard engine exiting non-zero with empty output reads as a clean gate pass (e.g. a pack-relative script in command: run with cwd=projectRoot)"
schema_version: issue/v1

issue:
  id: ISSUE-207
  title: "Nothing distinguishes a findings engine that crashed after starting from one that found nothing — a non-CrashGuard engine exiting non-zero with empty output reads as a clean gate pass (e.g. a pack-relative script in command: run with cwd=projectRoot)"
  type: bug
  status: open
  created: "2026-09-27"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: moderate
---

# ISSUE-207: Nothing distinguishes a findings engine that crashed after starting from one that found nothing — a non-CrashGuard engine exiting non-zero with empty output reads as a clean gate pass (e.g. a pack-relative script in command: run with cwd=projectRoot)

## Problem

At real `backstop gate` dispatch, a non-CrashGuard findings engine that **starts**, exits non-zero
and writes **empty stdout** reads as a finding-free clean pass: `pack_engines` reports `pass`, zero
violations. `parseSarif` returns `nil, nil` for empty input (`pkg/check/parsers.go:131-134`),
`check.NeverStarted` (ISSUE-112/140) fires only for a process that never launched, and the
`CrashGuard`-gated crash check (`cmd/backstop/pack_gate.go:940`) is skipped by definition.

**This shape is not always a bug, and core cannot tell.** Core is tool-blind, and legitimate
engines produce exactly this shape:

- `grep`/`rg` exit 1 with empty stdout when nothing matches — a genuinely clean result. A
  `gate_type: findings` engine built on grep plus a `convert:` step is legal (the shipped contracts
  pack's grep engine, `packs/contracts/pack.yml:20-34`, has this shape, though it is typed
  `contracts` and so routed through its dedicated step, `gateTypeHasDedicatedStep`,
  `pack_gate.go:87-94`).
- `stdout_artifact` engines (`pack_gate.go:880-898`) put their payload in a file; stdout is noise,
  so any check must apply to the selected payload, not stdout.
- `convert` engines emit non-SARIF stdout by design, so any check must run after the convert; a
  grep+convert engine can be non-zero with zero findings legitimately.
- Engines that exit non-zero *because* they report findings, which is why rule-fed findings engines
  leave `CrashGuard` false.

A per-engine declaration for the crash case already exists — `CrashGuard` — but a pack that does not
set it gets the fail-open behaviour by default. **The defect is that, by default, nothing
distinguishes "no matches" from "crashed"**: the same silent-pass outcome covers both, and a pack
author gets no signal either way.

The working-directory mismatch below is a **concrete, reproduced trigger** of this: a pack-declared
engine whose `command:` names a script by a path relative to the PACK (e.g. `command: "python3
tool.py"`, with `tool.py` alongside `pack.yml`) is executed with the working directory set to the
**project** root. The interpreter starts, fails to open its script argument, and exits non-zero with
empty stdout. Other post-start failures (bad flag, interpreter or import error) share the shape and
are indistinguishable for the same reason.

**A pack-relative script in `command:` is not the way to declare one.** The documented mechanism
already exists: `producer:` (ISSUE-067; `docs/CODEBASE-MAP.md:170-175`; resolved in
`runFindingsEngine` at `cmd/backstop/pack_gate.go:846-858` via `filepath.Join(packRoot, …)` plus an
`os.Stat` fail-loud). For this trigger, the script placed in `command:` instead of `producer:`
**silently no-ops** rather than failing loud or being detected at `pack check`/`pack test` time.

**This investigation initially set out to check** whether `scope_kind: file-args` selects zero
targets and so never invokes the engine at all. That was not what reproduction showed: the gate's
own `--all` scope computation correctly included both probe files, and the file-args arg-shaping
code correctly appended them to the command's argv. See "What this reproduction ruled out".

**Related:** ISSUE-206 (`pack test` phase 3 resolves engine inputs differently from the gate —
vacuous passes, false failures and argv mismatches). The two have an ordering dependency — see Root cause.

**Complexity note:** `scope: cross-cutting` because the decided default (Direction #1: declared
success-exit-code set, undeclared = `{0}`) changes behaviour for every non-CrashGuard findings
engine, and adds manifest/engine-binding surface. Uncertainty was `exploratory` while that was an
open three-way decision; the remaining design work (how the set relates to `CrashGuard`, where it
is validated) belongs to the plan. The authoring-time detection in Direction #2 is contained on
its own.

### Reproduction (fresh directories, generic names, relative paths from each)

Build a probe pack (any tool works; `tool.py` here both writes a discoverable stderr line and
would emit a SARIF finding for any `*.txt` file containing a marker, to make non-execution
unambiguous):

```bash
PACK=$(mktemp -d)
cd "$PACK"
backstop pack new --type engine --slug probe-pack --language go
cd probe-pack
rm -rf validators      # keep the scaffold's fixtures/ — phase2-coherence rejects a pack with none
```

`pack.yml`:

```yaml
name: local/probe-pack
version: 0.1.0
description: "probe pack for the relative-script repro"
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
        claims:
          - id: probe-pack-clm-001
            text: "Marker detection for repro."
            fixtures:
              positive:
                - fixtures/valid/example.txt
              negative:
                - fixtures/invalid/example.txt
```

`tool.py`:

```python
#!/usr/bin/env python3
import sys, json
print("PROBE-TOOL-RAN argv=%r" % (sys.argv[1:],), file=sys.stderr)
marker = "BACKSTOP-SAMPLE-VIOLATION"
results = []
for path in sys.argv[1:]:
    try:
        content = open(path).read()
    except OSError:
        continue
    if marker in content:
        results.append({"ruleId": "probe-pack-sample", "level": "error",
                         "message": {"text": "found marker"},
                         "locations": [{"physicalLocation": {"artifactLocation": {"uri": path}}}]})
print(json.dumps({"version": "2.1.0", "runs": [{"tool": {"driver": {"name": "probe-tool"}}, "results": results}]}))
```

(`description:` is required by the gate's pack loader — without it `backstop gate` fails loudly at
`pack_loading` with exit 2 and never reaches `pack_engines`; see "Related observation" below.)

Build a separate consumer repo and add the pack:

```bash
CONSUMER=$(mktemp -d)
cd "$CONSUMER"
git init -q
git config user.email test@example.com && git config user.name test
printf 'project: probe-consumer\n' > backstop.yml
echo "BACKSTOP-SAMPLE-VIOLATION" > bad.txt
echo "clean content" > good.txt
git add -A && git commit -qm initial
backstop pack add "$PACK/probe-pack"
backstop gate --all --json > gate_out.json 2> gate_err.txt
```

**Observed** (reproduced directly, 2026-09-27):

- `gate_out.json`'s `scope.files` correctly lists `bad.txt` and `good.txt` (scope computation is
  not the problem — the file list reaching the dispatch is non-empty and correct).
- The `pack_engines` step: `"status": "pass"`, `"violations": []` — no finding on `bad.txt` despite
  it carrying the marker.
- `gate_err.txt` contains no `PROBE-TOOL-RAN` line — the tool's own stderr instrumentation never
  fired, i.e. the script's Python code never executed at all.
- Running the exact same command manually from the consumer repo's root (the working directory the
  gate itself uses) reproduces the failure directly:
  ```
  $ cd "$CONSUMER" && python3 tool.py bad.txt good.txt
  .../python3: can't open file '.../tool.py': [Errno 2] No such file or directory
  exit=2
  ```
  Empty stdout, non-zero exit, error on stderr — exactly the shape that, for a non-CrashGuard
  findings engine, is indistinguishable from "ran cleanly, found nothing."

### What this reproduction ruled out

- **Not an empty scope.** `scope.Files` for the `--all` run legitimately contained both probe
  files; `pkg/gate/scope.go`'s `resolveGateScopeAll` (walks the whole tree, skipping only
  dot-directories) does not filter by extension or language.
- **Not the file-args arg-shaping branch itself.** `cmd/backstop/pack_gate.go`'s
  `ScopeKindFileArgs` else-branch (`cmdArgs = append(cmdArgs,
  excludeTestdataPaths(scope.Files)...)` at `pack_gate.go:805`) runs unconditionally when `scope.Files` is non-empty and
  does not skip invocation — it always calls `runner.RunStdout`. The gap is upstream of/around this:
  the command itself cannot start correctly regardless of which files get appended to it.
- **Not the trusted-tool allowlist.** This engine declares no `provision:` block, so per this
  repo's own verified allowlist behavior (see ISSUE-082, closed, and ISSUE-131, open) it may run
  any tool on PATH — `python3` is not blocked, and no trust-gate error surfaced.
- **Not the never-started/CrashGuard guards.** `check.NeverStarted` (guarding
  `*exec.Error`/fork-exec-class failures — the ISSUE-112/ISSUE-140 shape) does not fire here,
  because the process genuinely starts: `python3` itself launches fine via `LookPath`. It is the
  *interpreter*, after starting, that fails to open its own script argument — a shape neither
  `NeverStarted` nor the `CrashGuard`-gated crash check (`binding.CrashGuard && runErr != nil &&
  len(checkViolations) == 0`, correctly `false` for this engine class) is positioned to catch.

## Root cause

**General fail-open (no default discriminator).** In `runFindingsEngine` (`cmd/backstop/pack_gate.go`),
`stdout, runErr := runner.RunStdout(...)` (:860) is followed by the never-started refusal
(`check.NeverStarted(runErr)`, :876), which fires only for `*exec.Error` and `fork/exec`
`*fs.PathError` (`pkg/check/never_started.go:40-50`). A process that launches and then exits
non-zero falls through. With no `CrashGuard`, the guard at :940 (`binding.CrashGuard && runErr !=
nil && len(checkViolations) == 0`) never applies; `parseSarif` turns empty output into `nil, nil`
(`pkg/check/parsers.go:131-134`); and the step passes. ISSUE-112's own text names this ("Non-
CrashGuard engines treat every non-zero/failed run as finding-free (runErr discarded)", aggravator
2), but its delivered fix was scoped to the never-started class only — see Dedupe.

**Trigger reproduced here: cwd mismatch.** `cmd/backstop/gate.go:1130`'s `packValidatorStep`
constructs the runner used for every generic findings-engine dispatch as
`&check.ExecCommandRunner{Dir: projectRoot, Env: packChildEnvironment()}` (:1131); `RunStdout`
applies that `Dir` (`pkg/check/runner.go:78-80`). `runFindingsEngine` builds the command from
`binding.Command` via `splitCommand` (`pack_gate.go:1185`; called at :691) and runs it with
`runner.RunStdout(..., invoked, cmdArgs...)` (:860). `binding.Command` is a whole command line —
the script is `argv[1]` of an interpreter, not something core can recognise as a path — so it is
used verbatim. `Convert` (:913) and `Producer` (:852), by contrast, are single declared paths and
are explicitly joined against `packRoot` with an `os.Stat` check.

**Working-directory divergence between the two dispatch paths.** The packval executor runs the
same engine with `cmd.Dir = packDir` (`pkg/packval/executor.go:121`), whereas the gate runs it
with `Dir: projectRoot` (`gate.go:1131`). A pack-relative `command:` script therefore resolves
under `backstop pack test` and not under `backstop gate`. Consequence for ordering: ISSUE-206
(`pack test` phase 3 resolves engine inputs differently from the gate, and skips some command engines)
currently masks this by never running the tool;
fixing 206 alone would make `pack test` pass after a real run while the gate never runs the
script — a believable-looking green. Land this fix (or at least the detection) before or with 206.

**Fix risk — do not simply change the gate's cwd.** Moving the findings runner's `Dir` to `packRoot`
would break every engine that receives project-relative arguments: the file-args branch appends
`scope.Files` (project-relative paths, :805) and the `scope == nil` branch appends `projectRoot`
(:803), and tools such as semgrep also resolve rule paths and `paths.include` relative to their
cwd. Any fix must resolve the script path, not relocate the process.

**Not yet verified (stated rather than guessed past):** whether `scope_kind: project-wide`
behaves the same way for a script-relative command. `Dir` is set identically in that branch, so it
should, but it was not run end-to-end.

### Related observation: packval-vs-gate manifest-validation drift

While fixing the repro: a `pack.yml` without `description:` is **accepted** by `backstop pack
test` (reported `pass`) but **rejected** by the gate's pack loader (`pkg/pack/manifest.go:293`,
`description is required`; `backstop gate` reports `pack_loading` fail, exit 2). Same family as
the packval-vs-gate model drift behind ISSUE-206 and the earlier phase-3 issues; not fixed here.

## Impact

Any non-CrashGuard findings engine that fails after starting — bad flag, interpreter or import
error, and in particular a script sitting in the pack directory and invoked by a pack-relative
`command:` (rather than declared as a `producer:` or a PATH-resolved binary like
semgrep/ast-grep/go) — silently never runs its real logic at gate time. Because the failure mode is a non-zero exit
with empty stdout, it is structurally indistinguishable from "the engine ran and found nothing" —
the exact "no vacuous green" invariant this repo's gate-verdict-honesty work exists to protect,
here triggered by a plausible pack-authoring mistake (script in `command:`, not `producer:`) or
any post-start failure, rather than a missing PATH tool.

## Direction

1. **How the default distinguishes "no matches" from "crashed" — DECIDED (founder, 2026-10-12):
   a per-engine declared success-exit-code set, defaulting to `{0}` when undeclared.**
   - A findings engine may declare e.g. `success_exit_codes: [0, 1]` (grep/rg "no matches").
   - When the tool's **raw output** — the declared `stdout_artifact` file if the engine has one,
     otherwise stdout — is **empty, checked BEFORE any `convert:` step**, and the exit status is
     **not** in the engine's success set, the engine has failed: a loud error naming
     pack, engine and exit status, and telling the author to declare `success_exit_codes` if that
     status is a legitimate clean result.
   - A **non-empty** payload is parsed as findings regardless of exit status, so tools that exit
     non-zero *because* they report findings are unaffected.
   - Undeclared = `{0}` is a deliberate fail-safe default: an engine that legitimately exits
     non-zero with nothing to say fails loudly (with the fix in the message) rather than any
     crash passing silently. Ship as a minor version with a changelog note. `CrashGuard` engines
     already behave at least this strictly; the plan decides how the two declarations relate.
   - A local survey (2026-10-12) of the findings engines in the packs available found none that
     would break: four are `CrashGuard`, and the non-`CrashGuard` semgrep engine emits SARIF even
     with no findings. Third-party packs are unknown, hence the loud, self-explaining failure.

   The options considered were:
   - **Flip the default:** treat non-zero + no parseable findings as a crash unless the engine opts
     out (the inverse of today's opt-in `CrashGuard`). Safest, but breaks grep-style engines until
     they declare the opt-out.
   - **Extend `CrashGuard`:** keep it opt-in but widen what it covers/what packs are steered to
     declare, e.g. surface it in `pack check` for engines without it.
   - **Declare a success-exit-code set per engine** (e.g. `[0, 1]` for grep): any other exit status
     is an engine failure, any listed status is not. Most precise; new manifest surface.
   **Where the emptiness check runs — DECIDED (founder, 2026-10-12): before `convert:`.** A
   `convert:` step can wrap a crashed tool's empty output in a valid empty SARIF envelope, so
   checking after convert would leave every convert engine unable to detect a crash. The check
   therefore reads the raw tool output (the `stdout_artifact` file when declared, else stdout)
   before conversion. grep-style engines whose clean result is "exit 1, no output" declare
   `success_exit_codes: [0, 1]` — exactly what the field is for. The failure must name pack,
   engine and exit status.
2. **Detection at authoring time (independent of #1):** `pack check` / `pack test` should flag a
   `command:` whose argv names a file that exists in the pack directory (or does not resolve from
   the project root), steering the author to `producer:`. Do **not** attempt to "resolve Command
   against packRoot": `Command` is a whole command line and the script is argv[1] of an
   interpreter, so there is no single path to join.
3. **Keep the cwd:** per the fix risk above, do not relocate the gate runner's `Dir`.
4. Close the visibility gap shared with ISSUE-112's family: an engine that never meaningfully ran
   should never report the same `pack_engines: pass` as one that ran and found nothing.
5. Sequence against ISSUE-206 (see Root cause).

## Dedupe / existence-in-world check (performed 2026-09-27)

Searched `issues/` and `bundles/` before filing.

- **ISSUE-112** (closed) — a PATH-absent tool + no CrashGuard = vacuous pass. Same *family*, and
  its aggravator 2 ("non-CrashGuard engines treat every non-zero/failed run as finding-free") is
  the general started-then-failed fail-open this issue tracks. 112's delivered fix is scoped to the
  never-started class only (`exec.LookPath`/`*exec.Error` failing to launch the process;
  `check.NeverStarted`). This issue covers the remainder: a process that DOES start and then fails
  (bad flag, interpreter crash, and, as the reproduced trigger, a pack-relative script that the
  interpreter cannot open because of a cwd/packRoot mismatch). `check.NeverStarted` structurally
  cannot catch that shape.
- **ISSUE-131** (open) — allowlist-overclaim doc debt (the `Provision`-only scope of
  `checkEngineToolAllowed`). Cited above only to rule the allowlist out as a cause; it does **not**
  track the started-then-failed hole, and no open issue does. ISSUE-112's aggravator 2 describes
  the general hole but its closed fix covers only never-started, so this issue is the first to
  track the remainder.
- **ISSUE-140** (closed) — widened the never-started check to cover a path-ful command's
  `*fs.PathError` (a binary named by path, not found by `LookPath`). Still a fork/exec-level
  failure of the COMMAND itself; this issue's command (`python3`) starts successfully — the failure
  is one level down, inside the started interpreter failing to open its own script file.
- **ISSUE-151** (closed) — `paths.include`/`exclude` glob patterns invisible to semgrep under
  explicit-file dispatch (a phase-2 declaration-shape advisory in `pkg/packval/pathscope.go`).
  Unrelated mechanism (semgrep's own path-matching semantics vs. dispatch cwd for a pack's own
  script) and does not touch `cmd/backstop/pack_gate.go`'s command-invocation code at all.
- **DIR-032** ("Gate Verdict Honesty", `status: done`, closed roster) and **DIR-033** ("Gate Verdict
  Honesty Residual Tail", `status: queued`, 9 named members, read in full) — neither's roster names
  a `Command`-resolution/cwd gap. DIR-032's members closest in shape (112, 140) are reviewed above
  and are distinct triggers; DIR-033's nine members are claim-prose retractions, a validator
  exemption gap, a fixture-polarity mirror-sync gap, and a founder product-tradeoff question — none
  describe this mechanism.
- `.claude/agent-memory/backlog-pm/project_gate_verdict_honesty_cluster.md` (read in full) —
  catalogues the family's known triggers (missing PATH tool, narrow `*exec.Error`-only check,
  StdoutArtifact ignored, Convert ignored, paths.include asymmetry, etc.). A `Command`-vs-`packRoot`
  cwd mismatch for a pack-relative script is not among the entries recorded there.
- **Trusted-tool allowlist** (verified behavior, not a new finding): confirmed during this
  investigation that the allowlist gate (`checkEngineToolAllowed`) applies only to engines
  declaring a `provision:` block; a pack that omits `provision` may run any tool on PATH, per the
  already-filed ISSUE-082 (closed) and ISSUE-131 (open). Not filing a new allowlist issue — noted
  here only because it was checked and ruled out as a contributing cause of this defect.

No open issue or bundle charter already covers this defect. This is a new filing, not a duplicate.

## Notes

- Reproduced directly against the current working tree (2026-09-27), including a direct manual
  re-run of the failing command from the exact working directory the gate uses, which reproduced
  the underlying OS-level error (`can't open file '.../tool.py'`) independent of the CLI.
- Revised 2026-10-12 after independent review (two rounds): widened from the cwd trigger to the
  general fail-open, then reframed honestly (non-zero + empty stdout is legitimate for grep-style,
  stdout_artifact and convert engines; decision options laid out, none chosen); reframed against the existing `producer:` seam; repro corrected (`description:` added);
  line citations re-pinned against the current tree.
- Companion filing from the same investigation session: ISSUE-206 (`pack test` phase 3 resolves
  engine inputs differently from the gate — vacuous passes, false failures and argv mismatches —
  rather than a gate-dispatch fail-open).
- The originally suspected mechanism for this defect (`scope_kind: file-args` selecting zero
  targets) was investigated and not confirmed — see "What this reproduction ruled out" above. If a
  zero-target-selection gap exists independently of the cwd/script issue found here, it was not
  reproduced in this session and would need its own separate repro.
