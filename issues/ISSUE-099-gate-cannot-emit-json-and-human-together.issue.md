---
title: "gate Cannot Emit the Human Table and --json Together — No File-Emitting Flag"
schema_version: issue/v1

issue:
  id: ISSUE-099
  title: "gate Cannot Emit the Human Table and --json Together — No File-Emitting Flag"
  type: enhancement
  status: open
  created: "2026-07-28"

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

`.github/workflows/ci.yml`'s "Run the gate" step does exactly this workaround today
(`.github/workflows/ci.yml:126-147`):

```yaml
      - name: Run the gate
        # TWO INVOCATIONS, ONE GATE. The CLI renders EITHER the human table OR
        # --json to stdout, never both, and no flag writes JSON to a file. Both
        # surfaces are load-bearing and neither substitutes for the other: the
        # table is what a person reads in this log, while the JSON carries what no
        # other surface reports - per-file coverage detail, and the in-scope path
        # LIST behind the "in-scope files: N" count.
        #
        # The FIRST invocation is DIAGNOSTIC: its exit is recorded and deliberately
        # not acted on. The SECOND is the one that GATES, and its exit is the job's.
        run: |
          set +e
          ./bin/backstop gate --base "$BASE" --json > gate-report.json
          echo "diagnostic --json invocation exited: $?"
          set -e
          ./bin/backstop gate --base "$BASE"
```

Both invocations run the full kill chain (pack_lock_verification through ledger_integrity,
including `pack_engines` — the expensive step that shells out to every dispatched findings/test/
coverage engine). Measured cost in this workflow: the gate step's wall time roughly doubled, from
~2m40s to ~5m per run, purely from running the same checks twice. The workflow also carries a
two-step diagnostic-then-blocking structure (`set +e` / diagnostic run / `set -e` / blocking run)
that exists only to work around the single-surface limitation — the diagnostic run's exit is
explicitly discarded and only the second run's exit gates the job.

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

**Retirement trigger:** when this ships, `ci.yml`'s diagnostic `--json` invocation and the
`set +e` / `set -e` bracketing around it are dead weight — the diagnostic step's sole reason to
exist is producing `gate-report.json` without disturbing the blocking run's human-readable output.
It should merge into the single blocking invocation and the two-step comment block should be
rewritten accordingly.

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
