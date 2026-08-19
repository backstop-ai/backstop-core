---
title: "gate Cannot Emit the Human Table and --json Together — No File-Emitting Flag"
schema_version: issue/v1
delivered_by: PLAN-ISSUE-099

issue:
  id: ISSUE-099
  title: "gate Cannot Emit the Human Table and --json Together — No File-Emitting Flag"
  type: enhancement
  status: closed
  created: "2026-07-28"
  closed: "2026-08-19"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# gate Cannot Emit the Human Table and --json Together — No File-Emitting Flag

## Problem

`backstop gate` renders exactly one output surface per run: the human table to stdout, or
(`--json`) the JSON envelope to stdout — never both, and no flag writes the JSON to a file while
the human table still goes to stdout. Verified via `gate --help`:

```
Global Flags:
      --json   Output results as structured JSON
```

`--json` is a plain boolean on the global `jsonFlag` (`cmd/backstop/gate.go:33,170-176`); when set,
`gate.FormatJSON(result)` replaces the human-table render path entirely rather than running
alongside it. There is no third flag that separates "what renders where."

Any consumer that needs both surfaces from the same run — a human reading the table in a CI job
log, plus a machine-readable JSON artifact for downstream tooling — has no single-invocation way
to get them. The only workaround is running the entire gate twice: once for the human table, once
for `--json` piped to a file.

## Impact

`.github/workflows/ci.yml`'s gate job does exactly this workaround today, as two named steps
rather than an in-script `set +e`/`set -e` wrapper — a shape change made deliberately so which
step gates is visible from the step list alone, not from reading shell state
(`.github/workflows/ci.yml:126-152`):

```yaml
      # TWO STEPS, ONE GATE — and the split is structural on purpose. The CLI
      # renders EITHER the human table OR --json to stdout, never both, and no flag
      # writes JSON to a file (ISSUE-099 tracks closing that gap, which would
      # collapse these back into one step). Both surfaces are load-bearing: the
      # table is what a person reads in this log, while the JSON carries what no
      # other surface reports - per-file coverage detail, and the in-scope path LIST
      # behind the "in-scope files: N" count.
      #
      # Splitting them into separate STEPS rather than one script is the lesson of
      # the retired linux-sandbox probe workflow, every step of which ended in
      # `exit 0`: in-script exit handling makes a job's shape misleading to someone
      # skimming it. As two steps, which one gates is visible from the step list
      # alone, with no comment required to explain it.
      - name: Capture the gate report as JSON (diagnostic only - does not gate)
        # Runs BEFORE the blocking step, deliberately: a failing gate aborts the job,
        # so a capture placed after it would be skipped on exactly the runs worth
        # diagnosing. The `|| echo` records the exit instead of acting on it - this
        # step reports, the next step decides.
        run: ./bin/backstop gate --base "$BASE" --json > gate-report.json || echo "diagnostic capture exited $? - the blocking run is the next step"

      - name: Run the gate
        # Diff-scoped with an explicit base. NEVER the all-scope flag (ISSUE-091:
        # it under-reports against diff scope) and never the per-file flag
        # (ISSUE-093: it crashes on non-Go directories and silently drops repeats).
        #
        # THIS is the blocking invocation. Its exit is the job's, unchanged.
        run: ./bin/backstop gate --base "$BASE"
```

Both invocations run the full kill chain (pack_lock_verification through ledger_integrity,
including `pack_engines` — the expensive step that shells out to every dispatched findings/test/
coverage engine). Measured cost in this workflow: the gate step's wall time roughly doubled, from
~2m40s to ~5m per run, purely from running the same checks twice. The two-named-step split
replaced an earlier `set +e` / diagnostic run / `set -e` / blocking run in-script shape, but the
underlying limitation this issue tracks is unchanged: getting both output surfaces still costs
two full gate invocations instead of one.

This was surfaced during CI diagnostics wiring for ISSUE-087 (implementer-020-final,
2026-07-28): the JSON envelope is the only surface that carries per-file coverage detail and the
in-scope path list behind the "in-scope files: N" count (the observability PLAN-ISSUE-020
TASK-031 needed to resolve a 6-vs-5-vs-8 file-count discrepancy — see Notes). Getting that detail
into a CI artifact currently costs a second full gate run.

## Desired behavior

A flag — e.g. `--json-out FILE` — that writes the JSON envelope to `FILE` while the human table
still renders to stdout, in one gate invocation. This collapses `ci.yml`'s two-invocation block to
a single line:

```
./bin/backstop gate --base "$BASE" --json-out gate-report.json
```

**Retirement trigger:** when this ships, `ci.yml`'s diagnostic step (`Capture the gate report as
JSON (diagnostic only - does not gate)`) is dead weight — its sole reason to exist is producing
`gate-report.json` without disturbing the blocking run's human-readable output. It should merge
into the single blocking `Run the gate` step, and the comment block explaining the two-step split
should be rewritten accordingly.

## Notes / references

- Found by implementer-020-final while wiring CI diagnostics for ISSUE-087, 2026-07-28.
- Cross-reference: PLAN-ISSUE-020 TASK-031 — the diagnostic need this gap blocks (per-file
  coverage in JSON + scope-list observability) was driven by a 6-vs-5-vs-8 in-scope file-count
  discrepancy that could only be resolved by reading the JSON's scope list, which has no
  file-emitting path today.
- Cross-reference: the client-portal traceability feed direction (memory
  `project_client_portal_traceability_feed.md` — portal renders gate JSON) is a second consumer a
  file-emitting flag would serve, independent of CI.
- Cross-reference: ISSUE-091 (`gate --all` underreports test-file findings) already establishes
  deriving blocking sets from the JSON envelope as recorded practice — this issue is about how
  many gate runs it costs to get that JSON alongside the human table, not about the JSON's
  content.
- Loud-not-blocking framing: this is ergonomics/cost debt with a measured, cited cost (roughly 2x
  wall time on the CI gate step), not a correctness defect — nothing here produces a wrong
  verdict.

## Update (2026-08-18) — fresh evidence and founder priority

**Existence-in-world check performed tonight:** a fresh issue (ISSUE-171) was scaffolded to track
this exact defect, then correctly abandoned once this issue was found to already own the same
surface. This section adds tonight's evidence and priority signal to the existing issue rather
than forking a duplicate.

**Real, measured cost from an actual CI run.** Run `32151610956`'s `gate-report.json` was
downloaded and inspected directly. Within a SINGLE gate invocation, step timings were wildly
skewed: `pack_engines` took 629797ms (~10.5 minutes) and `coverage_threshold` took 612148ms
(~10.2 minutes) — every other step combined was under 10 seconds. Because `.github/workflows/
ci.yml`'s blocking job runs the ENTIRE gate twice per push — once with `--json` for the
diagnostic artifact, once without for the actual blocking check, same scope, same base, genuinely
duplicate work — this duplication alone adds roughly 20+ minutes to every single CI run. This is
more concrete and current than the ~2m40s→~5m estimate already cited above, and confirms the cost
scales with whatever `pack_engines`/`coverage_threshold` cost on a given run, not a fixed offset.

**Direct code verification that a single invocation could serve both purposes.**
`cmd/backstop/gate.go`'s `runGate` computes `exitCode` at line 244 (`result, exitCode :=
g.Run(context.Background())`) — before any branch on `--json` exists in the function. The
`jsonFlag` branch (`gate.FormatJSON(result)` vs. `gate.FormatHuman(result, noColor)`) spans lines
267–292 and is nothing more than two alternate FORMATTERS over the same already-decided verdict;
neither branch touches `exitCode`. The final check, `if exitCode != 0 { return &ExitCodeError{...}
}`, is at line 294 and is unconditional — it does not care which formatter ran above it. This
directly confirms the mechanism this issue's `--json-out FILE` proposal already assumes: one
`g.Run` call is sufficient to produce both surfaces, so a flag that writes the JSON to a file
alongside the existing stdout render (rather than replacing it) is a formatting-layer change, not
a re-run of the kill chain. Reinforces the existing proposed fix direction — this update does not
redesign it.

**This is now a founder-flagged priority, not just a nice-to-have.** In tonight's conversation the
founder directly flagged CI's runtime as a real scalability problem — "we cannot have two gate
runs on the entire project every single CI run... not scalable" — and asked directly how to fix
it. In the same conversation, a separate-but-related finding was filed as ISSUE-172 (gate's
internal steps also run sequentially rather than in parallel, independent of this issue's
duplicate-invocation waste).

**Cross-reference ISSUE-172 — the two combine multiplicatively.** Fixing this issue (ISSUE-099)
alone eliminates the duplicate full-gate run, roughly halving worst-case CI time (~42min →
~21min, using tonight's measured ~21min-per-invocation figure). Fixing both issues takes the
critical path down further, to roughly ~11min — the max of `pack_engines`'s and
`coverage_threshold`'s individual durations — since the two steps are structurally independent of
each other and could run concurrently once this issue's duplicate-run waste is also eliminated.
Neither fix alone reaches that floor; both together do.

## Update (2026-08-19) — fix shipped, closure pending plan completion

**Shipped to main at commit `e20e960` (`e20e9609fa27bcfce58bff3e2adbef84ae0ec179`, 2026-08-18
22:53:29 -0400), via `PLAN-ISSUE-099` (4 review rounds, signed off CLEAN).** The commit adds a
gate-local `--json-out FILE` flag to `backstop gate` that writes the gate/v1 JSON envelope to
`FILE` while the human table still renders to stdout, in one invocation — this is, near-verbatim,
what the "Desired behavior" section above already specified. `.github/workflows/ci.yml`'s
"Capture the gate report as JSON (diagnostic only)" step and its "TWO STEPS, ONE GATE" comment
block are deleted; the gate job now runs
`./bin/backstop gate --base "$BASE" --json-out gate-report.json` as a single step. The "Retirement
trigger" paragraph under "Desired behavior" above — which said the diagnostic step "should merge
into the single blocking `Run the gate` step, and the comment block explaining the two-step split
should be rewritten accordingly" — is now fulfilled exactly as described.

**Verification performed:**
- 9 new `TestGateJSONOut_*` tests pass (both-surfaces-from-one-run, flag interaction, refusal
  classes, verdict independence, the kill-chain-runs-once regression lock).
- 21 `TestCIWorkflow_*` tests pass, including the ci.yml-shape assertions that pin the single-
  invocation step count and the retirement of the diagnostic capture's `> gate-report.json ||
  echo` signature.
- A production smoke test on a real failing gate run confirmed both surfaces land correctly in one
  invocation: the human table rendered to the terminal AND a 182KB valid `gate/v1` JSON envelope
  was written, with the exit code correctly reflecting the verdict.
- `gate --all` was compared against a control worktree at pre-fix HEAD: 120 violations in both
  runs, byte-identically inherited, zero attributable to this change.

**Outstanding obligation — NOT yet confirmed, do not read as claimed.** The plan's own
"POST-MERGE OBLIGATION" section calls for reading the *next real CI run* after this landed and
confirming, from that run and not from local testing: exactly one "Run the gate" step in the job's
step list, the gate step's duration roughly halved against a recent pre-fix run, and the
`gate-report` artifact still uploaded and non-empty. That post-merge CI confirmation has **not**
been read yet. This is recorded here as an open obligation, not as a win — the plan explicitly
warns not to claim the win before reading it.

**Closure status: left `open`, not closed.** `PLAN-ISSUE-099`
(`plans/PLAN-ISSUE-099-gate-json-out-flag.plan.yml`) is still `status: draft` as of this update —
the implementer who executed the plan had no artifact-write path to flip it to `completed`. Per
`pkg/validate/delivered_by.go`, a `delivered_by` pointer is only satisfied once the plan it names
validates clean in its own right — pointing `delivered_by` at a plan still sitting at `status:
draft` would be getting ahead of the plan's own state. This issue's status is being left as `open`
rather than flipped to `closed` until (a) the plan's status is formally moved to `completed`
(planner territory, out of scope for this update) and (b) the post-merge CI confirmation above is
read. Both are separate, biteable follow-ons for the founder to dispatch.

## Resolution

`PLAN-ISSUE-099` was flipped to `status: completed` (its own AS-BUILT banner, 2026-08-19), so both
of the preconditions the update above named for closing this issue are now met. Closing here with
`delivered_by: PLAN-ISSUE-099` rather than re-authoring requirements/claims onto this issue.

**What shipped (summarized from the 2026-08-19 update above — not duplicated verbatim).**
`backstop gate` gained a gate-local `--json-out FILE` flag: one invocation now writes the human
table to stdout AND the `gate/v1` JSON envelope to `FILE`. `.github/workflows/ci.yml`'s "Capture
the gate report as JSON (diagnostic only)" step and its "TWO STEPS, ONE GATE" comment block are
gone; the job runs `./bin/backstop gate --base "$BASE" --json-out gate-report.json` as its single
gate step. Verified pre-close via 9 `TestGateJSONOut_*` tests, 21 `TestCIWorkflow_*` tests, a
production smoke test on a real failing gate run (182KB valid envelope written alongside the
rendered table, exit code matching the verdict), and a `gate --all` byte-identical comparison
against a pre-fix control worktree (120 violations both sides, zero attributable to this change).
Shipped at commit `e20e960` (`e20e9609fa27bcfce58bff3e2adbef84ae0ec179`).

**Post-merge CI confirmation — completed for real, not carried from the plan's partial read.** The
plan's close-out banner had checked run `32214168504` mid-flight and confirmed only check 1 of 3.
That run has since finished; re-read directly at this issue's own close-out (2026-08-19, via
`gh run view 32214168504` and `gh api .../artifacts`):

- **Check 1 (one gate invocation) — CONFIRMED**, and reconfirmed on the completed run: the
  "Backstop Gate" job's step list contains exactly one gate step, `Run the gate` (step 9), and no
  diagnostic-capture step.
- **Check 2 (duration roughly halved) — CONFIRMED.** Post-fix run `32214168504`'s single `Run the
  gate` step took ~20m55s (04:01:08Z → 04:22:03Z). Two pre-fix anchor runs on `main`
  (`32201643315`, `32200897995`) each ran the diagnostic-capture step (~20-21min) followed by the
  blocking `Run the gate` step (~20min), for a combined gate-work total of ~41-42min per run. ~41min
  → ~21min is roughly a halving, consistent with eliminating exactly one duplicate kill-chain run.
- **Check 3 (gate-report artifact uploaded and non-empty) — CONFIRMED.** `gh api
  repos/backstop-ai/backstop-core/actions/runs/32214168504/artifacts` lists one artifact,
  `gate-report`, 83151 bytes, not expired. The `Upload the gate JSON report` step succeeded
  (`if: always()`), including on this run whose gate step itself concluded `failure` (a normal
  violations-found red, not a defect in this fix) — which is itself further live evidence for
  CLM-004 (the file is written before the process decides its exit).

All three post-merge obligations from the plan are now read and confirmed, not merely predicted.
This closes the loop the plan's banner left open.

**Not resolved by this close — recorded, not absorbed.** Two items surfaced during implementation
that this issue's fix did not need to resolve and that this close does not paper over:

1. `.github/workflows/ci.yml`'s diff-scope paragraph on the surviving gate step still frames the
   all-scope ban as "pending a founder decision," while `TestCIWorkflow_BlockingJobNeverUsesAllOrFileScope`
   records a standing founder ruling dated 2026-08-17 (ISSUE-152 / ISSUE-156) that it has already
   been decided. This inconsistency predates this lane and was deliberately left verbatim rather
   than "while I'm here"-fixed; reconciling the prose with the ruling is a separate follow-on.
2. Whether the confirmation step's `|| echo` pattern (`./bin/backstop baseline pull || echo
   "bare baseline pull exited $?"`) should now join SPEC-067 CLM-028's `|| echo` swallow-denylist
   exemption is a spec-amendment question for the spec-author agent and the founder, not something
   this issue's fix needed to decide, and it is not decided here.
