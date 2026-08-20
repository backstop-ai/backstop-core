---
name: narrow-grep-false-missing-citation
description: A plan citation that returns zero hits from a narrow grep is usually MY grep being wrong, not the citation — re-grep the bare symbol before flagging a missing-symbol blocker
metadata:
  type: project
---

When auditing a plan's cited symbols/fields, a zero-hit grep is NOT evidence the citation is
false. Re-run with the bare symbol (no `func ` prefix, no assumed tag spelling) before writing
a blocker.

**Why:** on PLAN-ISSUE-180 round 3 two citations came back empty and both were MINE:
- `grep -n "func fileModeTestTargets" cmd/backstop/pack_gate.go` → nothing, but
  `grep -rn "fileModeTestTarget"` shows it called at exactly the cited `pack_gate.go:658`
  (declared in another file in the package).
- `grep 'json:"scope"' pkg/gate/result.go` → nothing, because the real tag is
  `json:"scope,omitempty"` (`result.go:182`) — the plan's `.scope.files` read path is correct.

Both would have been fabricated blockers in a round the planner had already earned a sign-off on.

**How to apply:** for every "the plan cites X but X doesn't exist" candidate, do a second pass:
strip the qualifier, drop to the bare identifier, and `grep -rn` the whole package. Only after
that returns nothing is it a real finding. Pairs with
[[project_verified_enumeration_do_not_rederive]] — re-derive, but re-derive CORRECTLY.
