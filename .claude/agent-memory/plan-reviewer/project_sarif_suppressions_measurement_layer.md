---
name: sarif-suppressions-measurement-layer
description: Three measurement layers, not two — raw SARIF rows vs parseSarif-active vs waiver-adjudicated gate output; a plan predicting `backstop gate` counts from either of the first two over-counts
metadata:
  type: project
---

A plan that predicts a `backstop gate` reading by counting rows in raw
`semgrep --sarif` output is measuring at the WRONG LAYER and will over-count.

**Why:** semgrep's `--sarif` formatter EMITS `// nosemgrep`-suppressed findings as
results carrying `suppressions: [{"kind":"inSource"}]` rather than dropping them.
(`--json` DOES drop them — the two formatters disagree.) `pkg/check/parsers.go`
`parseSarif` (ISSUE-017) skips any result with a non-empty `Suppressions`, so
backstop's reported violation count is the ACTIVE subset only.

Corollary trap: a bare `// nosemgrep` on a matched line looks non-functional if
you only read `--sarif` rows. It IS functional. Do NOT accept "this annotation
does not suppress, probably because of trailing prose after the rule id" — that
hypothesis is falsified by testing a bare `// nosemgrep`, which behaves identically.

Second corollary: `dir_active` and `explicit_active` are NOT nested sets. Directory
dispatch reports testdata rows explicit dispatch prunes, and path-scoped rules
(`paths.include:` with directory-prefixed globs) fire ONLY under directory dispatch.
So `dir + missed == explicit` is a FALSE identity — do not use it to declare a
counter-measurement "internally inconsistent." Use `|dir ∩ exp| + |exp only| = |exp|`.

**How to apply:** when a plan publishes predicted `gate --all` deltas, re-run the
engine yourself and split every finding by `bool(result.get('suppressions'))`.
Report BOTH layers (raw tool rows vs backstop-reported active rows) and demand the
plan's implementer-facing prediction use the ACTIVE numbers — an implementer told to
expect a row that `parseSarif` will drop reads the fix as broken.

Third corollary: the SUPPRESSED-row count is PER RUN, not a property of the pair.
A plan that writes "N suppressed rows separate raw from active" for a raw pair is
almost always quoting the directory run's N and silently mis-stating the explicit
run's. Compute `raw - active` for EACH run separately and check both. Likewise
"M suppressed across the two" is ambiguous between sum and de-duplicated union —
demand per-run numbers.

Measured 2026-08-16, semgrep 1.156.0, backstop-core `cmd/backstop`:
go-standards alone — raw dir 9 / raw exp 11; ACTIVE dir 6 / ACTIVE exp 7
  (suppressed: dir 3, exp 4 — the 4th is the `hermetic_fixtures_test.go` row).
Four semgrep packs — raw dir 11 / raw exp 14; ACTIVE dir 8 / ACTIVE exp 8 (net zero)
  (suppressed: dir 3, exp 6; union 6, sum 9).

## THERE ARE THREE LAYERS, NOT TWO — the waiver layer is the one plans forget

A plan that equates "parseSarif-active rows" with "what `backstop gate` prints" is
STILL wrong. Gate applies WAIVER ADJUDICATION downstream of parseSarif
(`pkg/gate/gate.go` swaps the waiver result into the run). So:

  raw SARIF rows  ⊃  parseSarif-active  ⊃  gate-printed (waiver-adjudicated)

`pkg/waiver/adjudicate.go` harvests each finding's window as `f.Line-1` plus the
finding line — so a `@waiver:<rule-id>:<reason>:<expiry>` comment on the line
IMMEDIATELY ABOVE a finding suppresses it from gate output entirely.

**How to apply:** for every row a plan predicts will appear in or vanish from
`gate --all`, `sed -n "$((L-1)),${L}p"` the cited file:line and look for a
`@waiver:` token. Check the pack is installed and the expiry is future. A waived
row was never in gate output, so:
  * a predicted "this row disappears" is VACUOUS (it was already absent), and
  * a predicted before/after TOTAL is off by the number of waived rows —
    which silently breaks any "the count should be unchanged / should be N"
    confirmation criterion the plan hands the implementer.

SECOND-ORDER EFFECT plans never name: if the fix stops a waived finding from
being produced at all, its `@waiver:` token becomes ORPHANED, and
`pkg/gate/step_waiver.go` reports "N unused/dangling" in the step summary. That
is a NEW gate-output change the plan created and must account for.

Caught 2026-08-16 on PLAN-ISSUE-091 round 8, after SEVEN prior rounds (two of
which were spent fixing the raw-vs-active layer error one level down). Its two
"THIRD DIVERGENCE" rows — `no-structural-name-split-on-spine` on `splitCommand`
in `cmd/backstop/pack_gate.go` and `engineToolName` in `pack_gate_provision.go` —
both carry live `@waiver:` tokens on the preceding line, so the plan's headline
"NET ZERO: 8 active rows before, 8 after" was a parseSarif-layer number bolted
onto a `./bin/backstop gate --all` command whose real reading is 6 → 8.

Related: [[project_scan_boundary_count_mismatch]],
[[project_verified_enumeration_do_not_rederive]],
[[project_nil_seam_default_needs_reachable_data]].
