---
name: phase3-polarity-and-silent-parse
description: How to settle SHOUT-vs-LIE for any pkg/packval RunEngine gap — parseSarif's two silent paths plus phase3's inverted fixture polarity; and always check whether the declaring rules have fixtures at all
metadata:
  type: project
---

Two mechanical facts settle the DIR-024-vs-DIR-032 home for every
`pkg/packval/executor.go` `RunEngine` "missing field" issue, and a third
bounds its urgency. Established 2026-08-17 triaging ISSUE-160
(CrashGuard/StrictSarif/Producer); verified in tree, not inferred.

**1. `parseSarif` (`pkg/check/parsers.go`) has TWO silent paths, not one.**
It short-circuits `len(bytes.TrimSpace(out)) == 0` to `(nil, nil)`, AND it
`json.Unmarshal`s into the `sarifLog` **struct** — so any valid JSON
*object* that isn't SARIF unmarshals fine to zero `Runs`, also `(nil, nil)`.
It errors ONLY on input that can't unmarshal into a struct — notably a JSON
**array**. That accident is the entire reason ISSUE-141 (missing `Convert`)
fails loud and went to DIR-024: `packs/substantiveness`'s ast-grep emits an
array. Empty stdout and object-shaped JSON fail SILENT.

**2. phase3 fixture polarity is the INVERSE of the naive reading.** In
`RunFixtures` (`pkg/packval/phase3.go`) a **positive** fixture is CLEAN code
expected NOT to fire — its only error is "positive fixture triggered the rule
(false positive)", raised when `Passed` is *true*. So `Passed: false` with a
nil error **is a positive fixture's success condition**. Any silent-parse
failure therefore reads as a clean positive-fixture pass = vacuous green =
DIR-032. The **negative** path gives the opposite, DIR-024-shaped
consequence: a loud but misattributed "may indicate an engine limitation"
whose FixHint advises *deleting the fixture* — so the same defect can destroy
a good fixture. Name both halves; don't split the issue over them.

**3. Before ranking, check whether ANY rule actually has fixtures.**
phase3 runs an engine only from inside `rule.Claims[].Fixtures`, and dispatch
requires `ruleSource != "" || rule.Pattern != ""`. As of 2026-08-17 every
`crash_guard`/`strict_sarif`/`producer`-declaring engine across all installed
packs (`backstop-ai/go-toolchain`, `backstop-ai/backstop-core-architecture`)
is bound to rules with **zero `claims:`** — so the whole class is LATENT,
zero live victims. `grep -c claims <pack.yml>` is the one-command check.

**Why:** these gaps arrive as batches from PLAN-mandated filings and all look
identical from the issue text; the issue's own severity framing is often
mechanism-true but urgency-false (ISSUE-160 argued Producer was "most
consequential" via go-coverage — which has no fixtures).

**How to apply:** for any `RunEngine`-ignores-field issue, (a) ask what the
failing payload *shape* is → array means shout/DIR-024, empty-or-object means
lie/DIR-032; (b) state the negative-fixture misattribution as a second
consequence in the item; (c) run the fixtures-exist check and report latency
explicitly. Fix surface is NOT charter — see [[project_packval_phase3_family]]
and [[project_gate_verdict_honesty_cluster]].
