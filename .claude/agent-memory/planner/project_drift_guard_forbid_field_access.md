---
name: drift-guard-forbid-field-access
description: A source-text drift guard must forbid the field-ACCESS form (`.Foo`), not a comparison form (`Foo ==` / `Foo !=`) — the comparison spelling is trivially evaded by an inlined normalizing call
metadata:
  type: project
---

When a plan mandates a "no second definition of X" drift guard that scans source
text, the forbidden substring must be the FIELD-ACCESS form with the leading dot
(`.Severity`), never a comparison form (`Severity ==` / `Severity !=`).

**Why:** PLAN-ISSUE-100 round-2 review caught this. The guard's own rationale said
it existed to catch an inlined `strings.EqualFold(strings.TrimSpace(v.Severity),
"warning")` — but that expression contains neither `Severity ==` nor `Severity !=`,
so the specified predicate could not detect the drift it was written for. The claim
text ("contains no direct comparison of its own") was broader than the predicate,
i.e. the claim and its own mandated test disagreed. Worse, the evasive shape was the
natural in-repo idiom: `pkg/gate/baseline.go` already writes
`strings.TrimSpace(v.Severity)`.

**How to apply:**
- Any hand-rolled comparison MUST read the field, so `.Foo` catches every approach
  (naive, EqualFold-inlined, or otherwise) while comparison spellings catch only one.
- Do NOT forbid the bare token without the dot — the struct DECLARES the field and
  comments mention it, so the bare token is present legitimately.
- MEASURE it: grep the exact forbidden substring across the scoped files and record
  in the task that it returns nothing at HEAD (guard genuinely green pre-fix), plus
  WHY it stays green post-fix (e.g. the new code passes whole structs to the single
  authority instead of reading the field).
- Listing the comparison forms ADDITIONALLY is fine for a sharper failure message,
  but the field-access form must be the predicate that decides pass/fail.
- Generalizes beyond severity: any "single authority" guard over a struct field.

See also [[state-a-sweep-once]] — the measured grep belongs in the plan once,
byte-identical.
