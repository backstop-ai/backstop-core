---
title: "Contracts Pack Compiler Func Only Signatures"
schema_version: issue/v1

issue:
  id: ISSUE-036
  title: "Contracts Pack Compiler Func Only Signatures"
  type: bug
  status: closed
  created: "2026-07-05"
  closed: "2026-07-06"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./pkg/pack/engine/... -race"

implementation:
  summary: >
    Rewrote packs/contracts/scripts/compile-signature.sh to be
    declaration-kind-aware: it infers the Go declaration kind from the
    leading token of the signature text (the runtime passes only the
    signature string, never the contract's `kind` field) and emits the
    matching structural ast-grep pattern per kind — func (unchanged),
    method (receiver TYPE now preserved instead of silently dropped),
    type, interface, const (`=` RHS required), and var. Added per-kind
    Go fixtures (packs/contracts/testdata/fixtures/sig-kinds-present.go,
    sig-kinds-mismatch.go) and a new Go test file
    (pkg/pack/engine/contracts_kind_signature_test.go) that shells the
    durable compiler source against real ast-grep. Added five per-kind
    static rules to packs/contracts/pack.yml (type-signature,
    const-signature, var-signature, method-signature,
    interface-signature) so `backstop pack test` itself now exercises
    every non-func kind. Synced the tracked test-harness copy
    (pkg/gate/testdata/traceability-pack/scripts/compile-signature.sh)
    and reinstalled the pack (`pack remove` + `pack add`) so the
    installed copy the gate actually executes
    (.backstop/packs/backstop/contracts/scripts/compile-signature.sh)
    and backstop.lock's content_hash pick up the fix. Delivered on main
    in the squash commit d5efd5b ("feat: eradicate `backstop code
    check` + un-vacuum gate dimensions").
  package: packs/contracts

requirements:
  - id: REQ-001
    text: >
      A `func` signature must continue to compile to the existing
      `func $NAME($$$PARAMS) <ret>` structural pattern — no regression
      to the one kind that already worked.
  - id: REQ-002
    text: >
      The compiler must infer the declaration kind from the leading
      token of the signature text, since the runtime
      (compileContractSignature, cmd/backstop/gate.go) passes only the
      signature string and never the contract's `kind` field.
  - id: REQ-003
    text: >
      A `type` signature (e.g. `type CheckType int`) must compile to a
      `type $NAME $$$` pattern that matches the real type declaration
      and produces zero matches where that name is absent.
  - id: REQ-004
    text: >
      A `const` signature must compile to a `const $NAME = $$$` pattern
      that preserves the `=`/RHS (a bare `const $NAME $$$` is an
      ast-grep ERROR node that matches nothing), matches the real
      constant, and produces zero matches where it is absent.
  - id: REQ-005
    text: >
      A `var` signature must compile to a `var $NAME ...` pattern that
      matches the real variable declaration and produces zero matches
      where it is absent.
  - id: REQ-006
    text: >
      A method-with-receiver signature (e.g. `func (ct CheckType)
      String() string`) must compile to a pattern that PRESERVES the
      receiver TYPE (only the receiver name and params are metavar'd),
      matches the real method, and produces zero matches against both a
      same-named free function and a same-named method on a different
      receiver type — the receiver can no longer be silently dropped.
  - id: REQ-007
    text: >
      An `interface` signature must compile to a `type $NAME interface
      { $$$ }` pattern that matches the real interface declaration and
      produces zero matches where it is absent.
  - id: REQ-008
    text: >
      The contracts pack's own validation suite (`backstop pack test`)
      must itself exercise every non-func kind via static rules and
      positive/negative fixtures in pack.yml, freezing the compiler's
      verified per-kind output as regression-checked rules — the gap
      that let the func-only compiler ship silently in the first place.
  - id: REQ-009
    text: >
      The fix must be durably installed end to end — the durable
      source, the tracked test-harness copy, and the gate's installed
      copy must all be identical after reinstall — and the full gate's
      `contract_signature` dimension must run clean against the
      reinstalled, kind-aware compiler, with any newly-lit contract
      drift the un-vacuumed dimension surfaces explicitly triaged as
      separate follow-up work rather than silently absorbed or papered
      over.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: >
      Compiling a func signature still yields a `func $NAME($$$PARAMS)
      <ret>` pattern that matches the present func fixture and does not
      match the mismatch fixture.
    tests:
      - TestContractCompiler_FuncSignatureUnchanged
  - id: CLM-002
    requirement: REQ-003
    text: >
      Compiling "type CheckType int" yields a `type CheckType ...`
      pattern that matches the real type declaration in the present
      fixture and produces zero matches in the fixture where CheckType
      is absent.
    tests:
      - TestContractCompiler_TypeSignatureMatchesTypeDecl
      - TestContractCompiler_TypeSignatureNoMatchAbsent
  - id: CLM-003
    requirement: REQ-004
    text: >
      Compiling a const signature preserves the name and the `=`,
      matches the real constant declaration, and produces zero matches
      where it is absent.
    tests:
      - TestContractCompiler_ConstSignatureMatches
  - id: CLM-004
    requirement: REQ-005
    text: >
      Compiling a var signature preserves the name, matches the real
      variable declaration, and produces zero matches where it is
      absent.
    tests:
      - TestContractCompiler_VarSignatureMatches
  - id: CLM-005
    requirement: REQ-006
    text: >
      Compiling "func (ct CheckType) String() string" preserves the
      receiver type and matches the real method; the same compiled
      pattern produces zero matches against a same-named free function
      and a same-named method on a different receiver.
    tests:
      - TestContractCompiler_MethodPreservesReceiverType
      - TestContractCompiler_MethodRejectsFreeFunctionSameName
  - id: CLM-006
    requirement: REQ-007
    text: >
      Compiling an interface signature yields a `type $NAME interface {
      $$$ }` pattern that matches the real interface declaration and
      produces zero matches where it is absent.
    tests:
      - TestContractCompiler_InterfaceSignatureMatches
  - id: CLM-007
    requirement: REQ-002
    text: >
      Compiling a type signature never produces a func-wrapped pattern,
      proving the kind is inferred from the signature text rather than
      passed in — the runtime supplies no kind field.
    tests:
      - TestContractCompiler_KindInferredFromSignatureText
  - id: CLM-008
    requirement: REQ-008
    text: >
      The contracts pack's own `backstop pack test` suite runs static
      per-kind rules (type-signature, const-signature, var-signature,
      method-signature, interface-signature) in packs/contracts/pack.yml
      whose patterns equal the compiler's verified per-kind output, each
      passing its positive fixture and rejecting its negative fixture.
    tests:
      - type-signature-go
      - const-signature-go
      - var-signature-go
      - method-signature-go
      - interface-signature-go
  - id: CLM-009
    requirement: REQ-009
    text: >
      After the fix, the durable source
      (packs/contracts/scripts/compile-signature.sh), the tracked
      test-harness copy
      (pkg/gate/testdata/traceability-pack/scripts/compile-signature.sh),
      and the reinstalled gate copy
      (.backstop/packs/backstop/contracts/scripts/compile-signature.sh)
      are byte-identical, and the pack's pre-existing func-only
      regression tests still pass unchanged against the synced script —
      proving the gate actually runs the fixed compiler, not a stale
      copy.
    tests:
      - TestContract_SignaturePresentAstGrepMatchSatisfied
      - TestContract_SignatureMissingAstGrepNoMatchViolation
      - TestContract_PatternCompilerLivesInPackNotBinary
      - TestContract_SignatureStructuralMatchIgnoresParamNames

contracts:
  - file: pkg/gate/step_contract.go
    provides:
      - name: StepContractSignatureScopedFunc
        kind: function
        signature: "func StepContractSignatureScopedFunc(results []ContractEngineResult, scope *GateScope) StepFunc"
        notes: >
          Unchanged by this issue (already declared by SPEC-038); listed
          here because this fix is what makes its verdict real for
          type/const/var/method/interface contracts. Prior to this fix
          it consumed a func-only compiler's output, so every non-func
          contract routed through it was structurally unmatchable and
          silently green whenever diff-scope hid the gap.
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

## Resolution

Delivered on `main` in the squash commit `d5efd5b` ("feat: eradicate
`backstop code check` + un-vacuum gate dimensions"), per
`PLAN-ISSUE-036-contracts-compiler-kind-aware.plan.yml`. `compile-signature.sh`
now infers the declaration kind from the leading token of the signature text
(`func (` → method, `func ` → function, `type ` → type/interface, `const ` →
constant, `var ` → variable — the runtime passes only the signature string,
never the contract's `kind` field) and emits the matching structural ast-grep
pattern per kind, verified against real ast-grep for every kind including the
method-receiver-preservation case the original bug dropped. Per-kind Go
fixtures and the mandated `TestContractCompiler_*` suite
(`pkg/pack/engine/contracts_kind_signature_test.go`) prove each pattern
matches its real declaration and rejects the discriminating decoy. Five
per-kind static rules were added to `packs/contracts/pack.yml` so
`backstop pack test` itself now exercises every non-func kind — the gap in
the pack's own suite that let the func-only compiler ship unnoticed. All
three copies of the compiler (durable source, the tracked test-harness copy,
and the gate's installed copy) were synced and the pack reinstalled
(`pack remove` + `pack add`) so the gate runs the fixed script; `backstop.lock`'s
`backstop/contracts` content_hash reflects the change.

`backstop gate --json` confirms `contract_signature` passes clean (0
violations, 0 new) against the reinstalled compiler. This issue's own scope —
making the compiler declaration-kind-aware — is complete.

**Honest residuals, not papered over — both spawned as separate tracked
issues rather than absorbed here:**

- **ISSUE-037** (iota-member const contracts): the fix's `const $NAME = $$$`
  pattern requires a real `=`/RHS, so a bare `iota`-block const member (no
  assigned value) remains structurally inexpressible under the pattern-arg
  input mode — a gap the fix correctly declined to solve rather than paper
  over with a pattern that would silently never match. The one live instance
  (`CheckTypeFindings`, SPEC-035) was reconciled by retiring that one contract
  as inexpressible and covering its real guarantee behaviorally instead; a
  repo-wide audit in ISSUE-037 found no other latent instance today, but the
  underlying compiler limitation is unresolved and will recur.
- **ISSUE-038** (contract drift exposed by the kind-aware compiler): making
  `contract_signature` non-vacuous for five more kinds exposed pre-existing
  drift between several specs' declared contracts and current code — signature
  mismatches and references to symbols the native-toolchain cutover already
  deleted, all pre-dating this fix and previously hidden by the same
  func-only-compiler / diff-scope gap. This is the dimension doing its job,
  not a regression this fix introduced (`new_violations: 0` on every surfaced
  item confirms it). It is grandfathered via `contract_signature`'s new
  `baseline: true` in `backstop.yml` (refreshed to capture the pre-existing
  drift) rather than left to silently block the gate or force-fixed here; the
  spec-by-spec reconciliation is ISSUE-038's own scope.

## Verification

- `go test ./pkg/pack/engine/... -run TestContractCompiler -v` — all 9
  mandated tests pass: `TestContractCompiler_FuncSignatureUnchanged`,
  `TestContractCompiler_KindInferredFromSignatureText`,
  `TestContractCompiler_TypeSignatureMatchesTypeDecl`,
  `TestContractCompiler_TypeSignatureNoMatchAbsent`,
  `TestContractCompiler_ConstSignatureMatches`,
  `TestContractCompiler_VarSignatureMatches`,
  `TestContractCompiler_MethodPreservesReceiverType`,
  `TestContractCompiler_MethodRejectsFreeFunctionSameName`,
  `TestContractCompiler_InterfaceSignatureMatches`.
- `backstop gate --json` — `contract_signature` step: `pass`, 0 violations, 0
  new violations, confirming the reinstalled kind-aware compiler is what the
  gate actually runs (not a stale copy).
- `diff` confirms the durable source
  (`packs/contracts/scripts/compile-signature.sh`), the tracked test-harness
  copy
  (`pkg/gate/testdata/traceability-pack/scripts/compile-signature.sh`), and
  the installed copy
  (`.backstop/packs/backstop/contracts/scripts/compile-signature.sh`) are
  byte-identical post-reinstall.

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
- ISSUE-037 (contracts-compiler-iota-member-const-support) — SPAWNED by this
  issue's own fix: the residual gap the `const` kind's `=`-required rule
  correctly declined to solve (bare iota-block const members)
- ISSUE-038 (reconcile-contract-drift-exposed-by-kind-aware-compiler) —
  SPAWNED by this issue's own fix: pre-existing spec/code contract drift the
  now-non-vacuous `contract_signature` dimension exposed across the repo,
  grandfathered via `contract_signature`'s new baseline pending spec-by-spec
  reconciliation
- `pkg/pack/engine/contracts_kind_signature_test.go` — the mandated
  `TestContractCompiler_*` suite proving the fix, shelling the durable
  compiler against real ast-grep
- `packs/contracts/testdata/fixtures/sig-kinds-present.go`,
  `sig-kinds-mismatch.go` — per-kind positive/negative Go fixtures added by
  the fix
- `packs/contracts/pack.yml` — five new per-kind static rules
  (type-signature, const-signature, var-signature, method-signature,
  interface-signature) so `backstop pack test` exercises every non-func kind
- `pkg/gate/testdata/traceability-pack/scripts/compile-signature.sh` — the
  tracked test-harness copy synced to the fixed durable source
- `PLAN-ISSUE-036-contracts-compiler-kind-aware.plan.yml` — the delivered
  plan this closure backfills requirements/claims/verification from
- CLAUDE.md — "Loud ≠ blocking" enforcement philosophy and the
  zero-baked-checks / no-vacuous-green first principle this defect violates
- Project memory `feedback_packs_always_external` — the packs-always-external
  policy referenced by the durability constraint above
