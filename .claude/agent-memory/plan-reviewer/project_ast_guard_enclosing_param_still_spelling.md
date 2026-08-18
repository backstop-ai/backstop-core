---
name: ast-guard-enclosing-param-still-spelling
description: "Rewriting an AST wiring guard to 'assert the last arg is the ENCLOSING function's last parameter' is still SPELLING, not provenance — a trailing decoy param of the same type and a local re-bind both green it"
metadata:
  type: project
---

When a plan fixes a false-positive AST "wiring guard" by classifying call sites by
ENCLOSING `*ast.FuncDecl` and requiring the last argument to name that decl's LAST
PARAMETER, run the proposed checker yourself over hand-built fixtures before believing
the plan's "this is provenance, not spelling" framing. Measured on PLAN-ISSUE-165
(2026-08-18), a faithful prototype of the prescribed rules returned ZERO violations for:

* **Trailing decoy param** — `func inner(..., probeABI LandlockABIProbe, decoy LandlockABIProbe)`
  forwarding `decoy`. "Last parameter" is POSITIONAL, so the decoy IS the last param.
  Same hole with a grouped field `probeABI, decoy LandlockABIProbe` (plan's own
  "take the last Name of the last Field" rule selects `decoy`).
* **Local re-bind** — `probeABI = func() {...fake...}` before the call. Name-equality
  still matches; only `go/types` could see it.
* **Selector call form** (`x.newSandboxHelperCommand(...)`) is skipped entirely
  (`call.Fun` is not `*ast.Ident`) — only the per-bucket COUNT assertion catches it,
  which is why exact-count assertions in these guards are load-bearing, not scenery.

**THE REMEDY THAT WORKS — verified end to end on PLAN-ISSUE-165 round 2.** Prescribe all
three, and require the fixture table to carry one case per hole:
1. Resolve the prober param BY TYPE — collect every `Name` across every `*ast.Field`
   whose type ident matches, require EXACTLY ONE, violate on 0 or 2+. Kills the decoy in
   both the separate-field and grouped-field spellings (`probeABI, decoy T` yields two).
2. Evaluate the option-(a) refusal FIRST and UNCONDITIONALLY (param must not be named
   after the package-level func; arg must not be that literal), with the name-equality
   rule ALSO always evaluated. Never as `else`.
3. Scan the enclosing body for an `*ast.AssignStmt` whose LHS is that identifier; catches
   both `=` and an inner-scope `:=`. Frame it as a MITIGATION, not proof of provenance.
Re-measured after the fix: real production file still 0 violations; decoy 1; re-bind 1;
full option-(a) 4 under the specified ordering and **0 under the `else` mis-ordering** —
which is what gives the fixture case its teeth.

**Why:** the whole defect class being fixed is a guard that overclaimed what it checked;
a replacement that overclaims "provenance" in its claim text repeats it one layer up.
**How to apply:** prototype the checker in the scratchpad (~60 lines, `go/parser` +
`go/ast`), run it on the REAL production file (expect the positive control to pass) plus
4-6 adversarial miniatures. Also demand the precedence between "literal X here is a
violation" and "must equal the enclosing param" be stated — if the param were renamed to
X, the two rules collide and the wrong ordering silently restores the vacuous green.
Related: [[project_wrapped_literal_defeats_substring_guard]],
[[project_single_authority_refactor_unfalsifiable]].
