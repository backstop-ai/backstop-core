---
name: census-through-real-parser
description: Re-derive a plan's token/finding census by running the REAL parser the plan's own design prescribes — a grep census over-counts, because tokens embedded in Go string literals are MALFORMED, not merely unbound
metadata:
  type: project
---

When a plan publishes a measured census ("SIX stale `@waiver` tokens", "27
naive vs 6 filtered"), reproduce it by running the parser the plan's own
pipeline will call, not by grepping the literal.

Concretely for `@waiver:` in backstop-core: `waiver.ParseToken` splits the token
core at the first whitespace and `time.Parse("2006-01-02", expiry)` the third
`:`-segment. A token written INSIDE a Go string literal —
`"x // @waiver:backstop/self/no-baked-language:accepted-risk:2999-01-01"})` —
carries the closing `"})` into the expiry and is rejected as MALFORMED. A
tree-driven harvest that "appends only tokens that parse without error" never
sees it. So every fixture-embedded token (`pkg/rule-a`, `pkg/ghost`,
`pkg/expiring`, `secrets/aws-key`, `go-standards/line-length` in the
`step_waiver_*_test.go` / `adjudicate_*_test.go` families) is dropped by the
malformed-skip, NOT by whatever precision filter the plan credits.

**Why:** PLAN-ISSUE-097 (2026-08-17) claimed six unbound tokens; the real
`ParseToken` yields five, and the plan's own TASK made a count mismatch a
mandatory STOP. Its false-positive breakdown also credited a segment-count
filter for exclusions the malformed-skip was already doing.

**How to apply:** write a throwaway `_test.go` that calls the real parser on the
exact raw lines and `go test -run` it (then delete it) — a two-minute check that
settles the census. Then re-run each of the plan's filters separately and report
what each one actually removes. Related:
[[project_sweep_axis_definition_drift]], [[project_stated_convention_vs_byte_arithmetic]].
