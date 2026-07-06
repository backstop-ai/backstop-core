---
title: "Contracts Pack Compiler Func Only Signatures"
schema_version: issue/v1

issue:
  id: ISSUE-036
  title: "Contracts Pack Compiler Func Only Signatures"
  type: bug
  status: open
  created: "2026-07-05"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate
---

# ISSUE-036: Contracts Pack Compiler Func Only Signatures

## Problem

The gate's `contract_signature` dimension can only verify `func` contracts. The
contracts pack's signature compiler,
`.backstop/packs/backstop/contracts/scripts/compile-signature.sh` (tracked source
at `packs/contracts/scripts/compile-signature.sh`), unconditionally emits a
`func <name>($$$PARAMS) …` ast-grep pattern regardless of what kind of Go
declaration the contract's `signature` text actually describes. When fed a
`type`, `const`, or method-with-receiver signature it produces a pattern that can
never match real Go source, so the contract fails even when the declared symbol
exists at exactly the file and signature the contract claims.

### Verified reproduction (2026-07-05)

```
$ sh .backstop/packs/backstop/contracts/scripts/compile-signature.sh "type CheckType int"
func type CheckType int($$$PARAMS) type CheckType int

$ sh .backstop/packs/backstop/contracts/scripts/compile-signature.sh 'const CheckTypeFindings = "findings"'
func const CheckTypeFindings = "findings"($$$PARAMS) const CheckTypeFindings = "findings"

$ sh .backstop/packs/backstop/contracts/scripts/compile-signature.sh "func (ct CheckType) String() string"
func ($$$PARAMS) String() string        # receiver silently dropped

$ sh .backstop/packs/backstop/contracts/scripts/compile-signature.sh "func RouteFile(path string, mode int) (string, error)"
func RouteFile($$$PARAMS) (string, error)   # control case: func-shaped input compiles correctly
```

`ast-grep --pattern` against either malformed pattern above returns zero matches
in any real Go source file, no matter how the target symbol is actually
declared — the pattern is syntactically nonsensical Go and can never match.

### Root cause

`compile-signature.sh` (`packs/contracts/scripts/compile-signature.sh`) does one
thing only: it strips a leading `func ` token (if present), splits the remainder
into `<name>(<params>)<rest>`, and re-emits `func <name>($$$PARAMS) <rest>`. It
never inspects what kind of declaration it was handed:

- A `type X …` or `const X …` signature has no `(` at all in the name position
  the script assumes, so the whole non-`func` text (`type CheckType int`, etc.)
  gets treated as the "name," and the script blindly prepends `func ` and
  appends a bogus `($$$PARAMS)` — producing `func type CheckType int($$$PARAMS)
  type CheckType int`, valid neither as a pattern nor as Go.
- A method-with-receiver signature (`func (ct CheckType) String() string`) does
  have parens, but the *first* paren pair the script's naive split finds is the
  **receiver** `(ct CheckType)`, not the parameter list. The receiver is
  discarded and treated as if it were the empty parameter list, so `String`
  becomes the "name" and the pattern loses the receiver type entirely
  (`func ($$$PARAMS) String() string`) — it can no longer distinguish this
  method from a same-named free function or a method on a different receiver
  type.

The compiler's own header comment states its scope: "into a STRUCTURAL ast-grep
pattern ... preserving the function name and the return clause" — it was
written for `func` signatures only and never extended when the contracts
schema's `kind` enum (`function | type | interface | method | constant |
variable`, SPEC-038 / `artifacts/issue/v1/schema.json`) admits the other five
kinds.

### Why it's a silent / latent trap, not an immediately-obvious one

`contract_signature` is diff-scoped: `StepContractSignatureScopedFunc`
(`pkg/gate/step_contract.go:50-54`) filters engine results down to contracts
whose declared file (`r.Entry.File`) is in the current gate scope
(`scope.Contains`) before it will ever surface a violation for that contract. A
`type`/`const`/method contract whose declared file is **not** in the current
diff never gets evaluated at all — it is not verified and passing, it is simply
never asked the question. Every non-`func` contract in the repo has been
sitting in exactly this state since the contracts pack (SPEC-038) went live: it
reads as green in every gate run whose diff doesn't happen to touch its file,
for a reason that has nothing to do with correctness.

This was exposed only because ISSUE-018's deletion work pulled `pkg/check` and
`cmd/backstop` files into diff scope for the first time, surfacing five
contracts — all verified present, on disk, at the declared file and signature —
that the gate nonetheless reports as `contract_signature` violations purely
because the compiler cannot render their kind:

| Symbol | Kind | Spec | File | On-disk confirmation |
|---|---|---|---|---|
| `Result` | type | SPEC-008 | `pkg/check/check.go:40` | `type Result struct` present |
| `ScopeMode` | type | SPEC-008 | `pkg/check/scope.go:14` | `type ScopeMode int` present |
| `CheckTypeFindings` | constant | SPEC-035 | `pkg/check/manifest.go:22` | `CheckTypeFindings` const present |
| `CheckType` | type | SPEC-039 | `pkg/check/manifest.go:10` | `type CheckType int` present |
| `CheckTypeConsumer` | type | SPEC-041 | `cmd/backstop/checktype_catalog.go:9` | `type CheckTypeConsumer struct` present |

All five verify by hand (grep/read of the file at the declared line); all five
fail the gate's `contract_signature` step solely because the compiler cannot
express their `kind` as an ast-grep pattern.

### Repo-wide blast radius

This is not limited to the five contracts above — those are only the ones a
diff happened to surface today. Every `type`/`const`/`interface`/`method`
contract anywhere in the repo is currently unverifiable by construction,
independent of whether its file is in scope right now. A non-exhaustive scan of
`contracts:` blocks with `kind: type` or `kind: constant` across
SPEC-008/013/014/025/035/039/041 (and any future spec) shows the same latent
defect sitting underneath each one — they are silently un-enforced today and
will surface the identical false-positive the moment their file next enters a
diff. The true count of affected contracts across the repo has not been
enumerated exhaustively; it should be treated as "at least the five surfaced by
ISSUE-018, and structurally true of every non-`func` contract in the repo,"
not as "exactly five."

### Why it matters

This is a vacuous-green hole in one of the gate's core enforcement dimensions —
the exact failure mode backstop exists to prevent (see CLAUDE.md's "Loud ≠
blocking" / the broader zero-baked-checks, no-vacuous-green program). An entire
class of API-surface contracts (types, consts, interfaces, methods — five of
the schema's six `kind` values) is not actually being checked; it merely
*looks* checked because diff-scope hides the gap until a file happens to
change. It also currently blocks reaching `contract_signature = 0` cleanly
after ISSUE-018 without either fixing the compiler or wrongly retiring five
live, correct contracts to make the gate pass.

## Solution

Not committed — left open for the plan, per the honest uncertainty below.
Direction to evaluate:

1. **Kind-aware pattern compilation.** Extend
   `packs/contracts/scripts/compile-signature.sh` to detect the declaration
   kind from the signature text (or, if available at call time, from the
   contract's own `kind` field) and emit the matching ast-grep pattern per
   kind instead of always assuming `func`:
   - `func $NAME(...)` — existing, correct, keep as-is.
   - `type $NAME ...` — emit `type $NAME ...` (or the appropriate ast-grep
     `type_declaration` structural pattern), not a `func`-wrapped one.
   - `const $NAME ...` / `var $NAME ...` — likewise, emit the real declaration
     shape.
   - Methods with receivers (`func ($RECV $T) $NAME(...) ...`) — preserve the
     receiver clause verbatim (or metavariable it, e.g. `func ($$$RECV)
     $NAME($$$PARAMS) ...`) instead of discarding it; the receiver type is
     part of what the contract is asserting.
   - Interfaces — confirm whether `interface` contracts route through this
     same compiler today or a different path before assuming they need the
     identical fix.
2. **Verify against ast-grep's actual Go pattern grammar** for each kind before
   committing to a specific emitted string — do not assume the naive
   substitution above is syntactically valid ast-grep without checking.
3. **Add pack fixtures — positive and negative — for every kind** (`type`,
   `const`, `var`, method-with-receiver, and whatever `interface` turns out to
   need) alongside the existing `func` fixtures
   (`packs/contracts/testdata/fixtures/`). The absence of any such fixture is
   why this shipped silently: the pack's own test suite never exercised a
   non-`func` signature, so nothing caught the gap before it reached five live
   contracts.
4. Re-verify all five surfaced contracts (and any others discovered by a
   broader scan) pass `contract_signature` once the compiler is fixed —
   without retiring or rewriting the contracts themselves, since all five are
   already correct.

### Durability constraint (load-bearing)

As of this writing, `backstop/contracts` is declared `local` in `backstop.yml`
and its lock entry (`source_type: local`), meaning its durable source today is
the **tracked** `packs/contracts/` directory inside this repo — NOT the
installed copy at `.backstop/packs/backstop/contracts/`, which is a
gitignored, disposable copy that `pack add`/install regenerates from
`packs/contracts/`. Editing the installed copy directly is non-durable and
will be silently overwritten on the next install/reinstall. The fix must land
in `packs/contracts/scripts/compile-signature.sh` (and
`packs/contracts/testdata/fixtures/` for the new cases), then be re-installed
so the `.backstop/packs/backstop/contracts/` copy and the `backstop.lock`
content hash for `backstop/contracts` pick up the change. Per this project's
packs-always-external policy, the long-term intent is for every pack —
including this one — to live in its own repository rather than as a `local`
in-tree pack; if `backstop/contracts` is externalized before this issue is
picked up, the same constraint applies to that external repo instead. Do not
treat the current in-tree `local` declaration as a violation of that policy in
itself — it is a separate, already-tracked piece of debt, not part of this
issue's scope.

### Interaction with diff-scope (expected, not a surprise)

Once the compiler correctly compiles `type`/`const`/method patterns, every
previously-latent non-`func` contract across the repo starts being evaluated
for real the next time its declared file enters a diff. This may surface
further genuine signature mismatches that were equally hidden by the same gap
(a contract whose declared signature has silently drifted from the real
source, for instance). That is the dimension becoming non-vacuous doing its
job, not a regression — the plan and its review should expect and budget for
follow-on findings rather than treat them as new defects introduced by this
fix.

**Core uncertainty, stated honestly:** the pattern-compilation fix for each
individual kind is believed contained (known shapes, known ast-grep grammar to
verify against), but the true repo-wide blast radius — how many non-`func`
contracts exist today and how many will produce a genuine (not
compiler-artifact) mismatch once actually evaluated — has not been
enumerated. Treat that count as exploratory at plan time, not as a fixed,
small, pre-known number.

## References

- `packs/contracts/scripts/compile-signature.sh` — tracked source of the
  compiler; the sole place the Go-signature → ast-grep-pattern mapping lives
  (per its own header comment)
- `.backstop/packs/backstop/contracts/scripts/compile-signature.sh` — the
  disposable installed copy used for the reproduction above; NOT the fix
  target
- `pkg/gate/step_contract.go:50-54` — `StepContractSignatureScopedFunc`, the
  diff-scope filter (`scope.Contains`) that makes an unevaluated non-`func`
  contract read as silently green rather than as an open question
- `artifacts/issue/v1/schema.json` (`contracts.kind_enum`) — the six declared
  contract kinds (`function | type | interface | method | constant |
  variable`); the compiler currently only serves one of them correctly
- `SPEC-038-traceability-contracts-pack.spec.md` — the contracts pack's
  origin spec (compiler, diff-scoped step, kind enum)
- `SPEC-008-code-check.spec.md` (`Result`, `ScopeMode`) — two of the five
  surfaced live `type` contracts
- `SPEC-035-pack-declared-engines-trusted-allowlist.spec.md`
  (`CheckTypeFindings`) — surfaced live `constant` contract
- `SPEC-039-codecheck-deadcode-prelude.spec.md` (`CheckType`) — surfaced live
  `type` contract
- `SPEC-041-coverage-reimpl-checktype-catalog.spec.md`
  (`CheckTypeConsumer`) — surfaced live `type` contract
- ISSUE-018 — the discovering change; pulled `pkg/check`/`cmd/backstop`
  files into diff scope, first surfacing the five false-positive
  `contract_signature` violations
- ISSUE-034 (gate-coverage-flags-deleted-files), ISSUE-035
  (gate-substantiveness-flags-testmain-absence-tests) — sibling family filed
  the same day: dark/dormant gate-check logic exposed for the first time by
  diff scope suddenly widening under the eradication backlog
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy and the
  zero-baked-checks / no-vacuous-green first principle this defect violates
- Project memory `feedback_packs_always_external` — the packs-always-external
  policy referenced by the durability constraint above
