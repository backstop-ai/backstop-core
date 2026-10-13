---
title: "SARIF result properties with non-string values reject the whole log"
schema_version: issue/v1

issue:
  id: ISSUE-208
  title: "SARIF result properties with non-string values reject the whole log"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# SARIF result properties with non-string values reject the whole log

## Problem

The check SARIF parser declares a result's `properties` as `map[string]string`
(`pkg/check/parsers.go:112`). Its doc comment (`pkg/check/parsers.go:106-111`) states the
intent: "v1 preserves string-valued entries only; a result with no properties yields a nil map
(no error)" — i.e. non-string entries are meant to be dropped. But `encoding/json` does not drop
them: a non-string property value makes `json.Unmarshal` return an `UnmarshalTypeError`
(it keeps decoding the rest of the document, but reports the error). What discards every result
is `parseSarif` returning on any unmarshal error (`pkg/check/parsers.go:136-137`,
`parsing SARIF output: ...`), so one non-string property loses the whole log.

SARIF 2.1.0 section 3.8 defines a property bag as an object whose values may be arbitrary JSON,
and section 3.8.2 reserves `properties.tags` as an array of unique strings. So `map[string]string`
rejects the spec's own reserved property, not just one producer's output. gitleaks' native SARIF
emits `properties.tags` as an array on every result.

Observed adopting `backstop-ai/secrets@1.1.0` in a TypeScript consumer repo where gitleaks found
results. The `pack_engines` step failed (severity error) and the gate exited 2 with:

```
dispatching findings engine "gitleaks" for pack backstop-ai/secrets: pack backstop-ai/secrets engine gitleaks dir . --report-format sarif --report-path .backstop/gitleaks.sarif --no-banner: convert/parse to SARIF failed: parsing SARIF output: json: cannot unmarshal array into Go struct field .runs.results.properties of type string
```

## Impact

- The "secret found" path is the broken one. With zero gitleaks results there are no result
  properties, so the log parses and the pack looks healthy; the first time gitleaks actually
  finds something, the pack cannot report it.
- Fail-closed, not fail-open: the gate does fail (exit 2), so a leaked secret does not slip
  through as a green gate. But the failure is a parse error, not a finding — the content of what
  was found (rule, file, line) is lost, and the operator is pointed at a JSON decoding problem
  instead of at the secret.
- Dispatch behaviour when an engine errors (earlier findings discarded, later engines skipped) is
  tracked separately in ISSUE-209.
- The parser is shared by every SARIF-emitting engine, so any other producer that puts a
  non-string value in a result property bag hits the same failure.

## Root cause

Regression from ISSUE-062. `git log -L` on the `Properties` field shows it was introduced in a
single commit, `efd7e9f` (2026-07-17, "ISSUE-062: structured finding properties +
substantiveness join fix"), whose doc comment promised string-only preservation but whose
`map[string]string` type enforces string-only input instead. The field was tested only against
string-valued properties (the substantiveness `func`/`symbol`/role keys), which never exercised a
foreign producer's array-valued bag.

## Direction

(Direction, not a plan.)

1. Core: decode `properties` leniently — e.g. `map[string]json.RawMessage`, keeping entries whose
   value is a JSON string and ignoring all others — matching the documented v1 intent. Do NOT
   fix this by tolerating `UnmarshalTypeError` and continuing: that would silently keep a
   half-decoded log. `Violation.Properties` stays `map[string]string`, so all consumers (grep
   `.Properties`) keep working unchanged: reads in `pkg/gate/substantiveness_join.go`
   (`:249`, `:276`, `:279`, `:298`), pass-through copies at `pkg/check/parsers.go:175`,
   `cmd/backstop/pack_gate.go:1000` and `pkg/gate/substantiveness_q1_dispatch.go:75`, the
   baseline-identity exclusion documented at `pkg/check/check.go:32` and `pkg/gate/result.go:121`,
   and the `pkg/gate/testdata/self-rule/role-property-routing.go` fixture (reads
   `Properties["substantiveness_role"]`). Add a regression test using a gitleaks-shaped SARIF
   fixture built around array-valued `tags` (per section 3.8.2) plus string properties, proving:
   parse succeeds, violations surface, string properties are preserved, non-string properties
   are ignored.
2. Follow-on, out of scope for this core issue — secrets pack (separate repo,
   `backstop-ai/secrets`): its fixtures should include a planted fake secret so
   `backstop pack test` exercises the found-results path end to end. That gap is the likely
   reason this survived (unverified: the pack repo was not inspected for this issue).

## Notes

- Repro: run gitleaks (native SARIF) over any tree containing a detectable secret and feed the
  log through the findings-engine dispatch with the `backstop-ai/secrets` pack.
