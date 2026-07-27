---
title: "Contracts Pack Signature Compiler Cannot Bind Grouped Const/Var Block Members"
schema_version: issue/v1

issue:
  id: ISSUE-078
  title: "Contracts Pack Signature Compiler Cannot Bind Grouped Const/Var Block Members"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: exploratory
  risk: safe
---

# ISSUE-078: Contracts Pack Signature Compiler Cannot Bind Grouped Const/Var Block Members

## Problem

This issue has three distinct findings. The first two were discovered while
triaging why SPEC-054 (the first spec to declare `const`-block contracts)
could not get a clean `contract_signature` pass for its `RecipeKind`/`Op.Kind`
consts: one is a currently gate-red broken promise on the closed ISSUE-036;
the other is a real, reproduced compiler capability gap that ISSUE-036 never
actually claimed to close. The third is a sibling capability gap in the same
compiler, verified during PLAN-ISSUE-080 review on 2026-07-26: `kind: type`
contract entries are existence-only, not shape-verifying.

### Finding 1 (currently gate-red): ISSUE-036's CLM-008 mandated tests are unresolvable by construction

`./bin/backstop gate --json` currently reports `artifact_status_drift: pass`
with 7 violations, 5 of which are against the closed, success-terminal
ISSUE-036:

```
artifact ISSUE-036 is closed (success-terminal) but its mandated test
type-signature-go is ABSENT, claim CLM-008 — a broken promise
artifact ISSUE-036 is closed (success-terminal) but its mandated test
const-signature-go is ABSENT, claim CLM-008 — a broken promise
artifact ISSUE-036 is closed (success-terminal) but its mandated test
var-signature-go is ABSENT, claim CLM-008 — a broken promise
artifact ISSUE-036 is closed (success-terminal) but its mandated test
method-signature-go is ABSENT, claim CLM-008 — a broken promise
artifact ISSUE-036 is closed (success-terminal) but its mandated test
interface-signature-go is ABSENT, claim CLM-008 — a broken promise
```

These five names (`type-signature-go`, `const-signature-go`, etc.) are NOT
absent code — they are the `claims[].id` values of five real, passing rules
in `packs/contracts/pack.yml` (`type-signature`, `const-signature`,
`var-signature`, `method-signature`, `interface-signature`, each with a
positive/negative fixture pair). Verified directly:

```
$ ./bin/backstop pack test packs/contracts
status: pass
- phase1-structural: pass
- phase2-coherence: pass
- phase3-fixtures: pass   # <- the 5 per-kind rules + their fixtures all pass
- phase4-archetype: pass
- phase5-layer: pass
- phase6-risk-class: pass
```

The underlying capability CLM-008 describes (`backstop pack test` exercising
every non-func kind) genuinely exists and genuinely passes. The problem is
that a pack.yml `claims[].id` is a kebab-case rule identifier, never a Go
identifier, so it can never satisfy `gate.TestNameMatcher` — which is built
from `Manifest.TestNamePatterns` regexes contributed by *toolchain* packs
(the Go test-function-name convention, e.g. `^Test[A-Z]`) via
`mergeTestNameMatcher` (`cmd/backstop/gate.go:1188`). There is no path by
which a pack-level fixture-claim id can ever register as a "present test" to
`artifact_status_drift` — CLM-008 mandated a test-name shape the gate
structurally cannot resolve, regardless of whether the pack rules exist and
pass. This is a mandated-test-naming defect in how ISSUE-036 was authored,
not a missing capability: the identical claims (per-kind compiler output
matches its declaration) are ALSO proven by `go test`-visible names that
already exist and already pass —
`TestContractCompiler_TypeSignatureMatchesTypeDecl`,
`TestContractCompiler_ConstSignatureMatches`,
`TestContractCompiler_VarSignatureMatches`,
`TestContractCompiler_MethodPreservesReceiverType`,
`TestContractCompiler_InterfaceSignatureMatches` (verified: `go test
./pkg/pack/engine/... -run TestContractCompiler -v` — all 9 pass, 2026-07-25).

### Finding 2 (real capability gap, currently worked around, not gate-red): grouped const/var block members don't compile

Separately — and this is what actually forced the SPEC-054 triage — the
kind-aware compiler's `const $NAME = $$$` / `var $NAME = $$$` patterns only
bind a **standalone** declaration (`const NAME = VALUE` with no
surrounding parens). They do not bind a member of an idiomatic **grouped**
`const ( … )` / `var ( … )` block, even when that member has the exact
same name and an explicit `=` value the compiler's own `=`-required rule
demands. Verified directly against the compiler's own emitted pattern:

```
$ sh packs/contracts/scripts/compile-signature.sh 'const KindScaffolding = "scaffolding"'
const KindScaffolding = $$$

$ cat /tmp/grouped.go
package fixture

const (
	KindScaffolding  = "scaffolding"
	KindImplementing = "implementing"
	KindTemplating   = "templating"
)

$ ast-grep run --pattern 'const KindScaffolding = $$$' --lang go /tmp/grouped.go
(no output, exit 1 — zero matches)

$ cat /tmp/standalone.go
package fixture

const KindScaffolding = "scaffolding"

$ ast-grep run --pattern 'const KindScaffolding = $$$' --lang go /tmp/standalone.go
/tmp/standalone.go:3:const KindScaffolding = "scaffolding"   (exit 0, 1 match)
```

Identical name, identical `=` value, identical pattern string — the only
difference is the grouping parens, and that alone flips a real declaration
from matched to unmatched. `KindScaffolding`/`KindImplementing`/
`KindTemplating` and `OpCreate`/`OpMerge`/`OpTransform`/`OpInsert`/`OpStep`
(`pkg/recipe/manifest.go:18-32`) are declared in exactly this idiomatic
grouped form, so SPEC-054 could not give them a real `provides` contract
entry; its notes (`specs/SPEC-054-recipe-apply-and-manifest.spec.md:631,635,914`)
document dropping the entries as inexpressible under the current compiler
rather than declaring a fictional one, and forward-reference this issue by
number. `contract_signature` is currently `pass` (0 violations) precisely
*because* SPEC-054 took that workaround — this finding is not gate-red today,
but it is a real, reproduced gap blocking a real spec from declaring a real
contract on idiomatic Go.

### Finding 3 (real capability gap, currently undetected, not gate-red): `kind: type` contract entries are existence-only, not shape-verifying

Verified 2026-07-26 while reviewing PLAN-ISSUE-080: the kind-aware compiler's
`type` branch (`packs/contracts/scripts/compile-signature.sh:78-89`) collapses
every non-interface type declaration to `printf 'type %s $$$'` — the `$$$`
wildcard matches any underlying type (a different struct entirely, a func
type, a primitive alias, anything). The interface branch is only nominally
stricter: `printf 'type %s interface { $$$ }'` (line 84) still terminates in
a `$$$` wildcard, so it confirms only "declared as `interface`," never which
methods it declares. Verified directly:

```
$ sh packs/contracts/scripts/compile-signature.sh 'type AddOptions struct { ProjectDir string; Version string }'
type AddOptions $$$
```

The static pack rule that exercises this same collapse,
`packs/contracts/pack.yml:70-73` (`type-signature`, `pattern: "type $NAME
$$$"`), matches identically regardless of the type's actual field list,
underlying type, or method set. A `provides` entry of `kind: type` proves
only that a type named `$NAME` exists — never that its shape matches the
spec's declared signature string.

This false premise — "the contracts gate will red if the type's shape drifts
from what's declared" — has now surfaced twice in planning, both caught only
in review, not at authoring time:

- SPEC-055's `AddOptions` contract entry note
  (`specs/SPEC-055-production-remote-dependency-assembly.spec.md:1028`) had
  to explicitly document that the entry "does NOT itself enforce" the
  GitCloner/Validator field removal, because "the contracts pack's signature
  compiler reduces any struct to `type AddOptions $$$` ... and never compares
  field lists" — forcing a dedicated field-absence claim (CLM-030) to carry
  the real enforcement instead of the type-kind entry.
- PLAN-ISSUE-080's Phase 5 reconciliation task
  (`plans/PLAN-ISSUE-080-malformed-waiver-diagnostic-surfacing.plan.yml:732-734`)
  asserted that leaving SPEC-054's declared `WaiverReader` signature unedited
  after widening the interface "reds the gate as contract drift" — treating
  the contracts dimension as a shape-verifying forcing function it
  structurally is not for `kind: type` entries. This was caught and
  corrected during plan review rather than at drafting.

Same root cause as Finding 2 and ISSUE-052: `--pattern`-string matching under
`input_mode: pattern-arg` can express "a node of this shape exists here" but
not "and its named sub-structure matches these values." The relational-rule
`input_mode` ISSUE-052 already proposes (`kind: <node>` + scoped `has:`
clauses) is the same mechanism needed here, generalized to a fourth instance
— `type_declaration`/`interface_type` shape checks — rather than a fourth
parallel fix.

### This is the same root cause ISSUE-052 already tracks, one instance wider

ISSUE-052 (`contracts-engine-relational-rule-input-mode`, open,
technical-debt) already generalizes this exact class of gap — "the compiler
only ever emits a single `--pattern` string, and [the symbol] is not a
standalone Go fragment a `--pattern` string can bind" — across two prior
instances: a bare iota-member const (no `=` at all, absorbed from ISSUE-037)
and a struct field (3 live grandfathered instances from ISSUE-038). A
grouped const/var block member is a third instance of the identical
structural-presence limitation: it has an `=` value (so it isn't the iota
case) and it isn't a struct field, but it is still not a "standalone Go
fragment" — it's a `const_spec`/`var_spec` node nested inside a
parenthesized `const_declaration`/`var_declaration`, and `--pattern`-string
matching does not reach into that grouping. ISSUE-052's proposed fix (a new
relational-rule `input_mode`, e.g. `kind: const_spec` + `has: {field: name,
…}`, scoped to the spec's own name field rather than any same-named
identifier occurrence) generalizes to this case with no new mechanism: the
same rule shape that distinguishes a genuine iota-member declaration from a
same-named reference elsewhere also distinguishes a genuine grouped-block
member, regardless of whether it carries a value.

## Solution

Not committed — three independent, differently-sized fixes, evaluate and
schedule separately rather than as one unit:

1. **Finding 1 (mandated-test repoint, small, no new code).** Edit
   ISSUE-036's CLM-008 `tests` list to cite the existing, passing Go test
   names that already prove the identical per-kind claim
   (`TestContractCompiler_TypeSignatureMatchesTypeDecl`,
   `TestContractCompiler_ConstSignatureMatches`,
   `TestContractCompiler_VarSignatureMatches`,
   `TestContractCompiler_MethodPreservesReceiverType`,
   `TestContractCompiler_InterfaceSignatureMatches`) instead of the pack.yml
   rule ids the gate can never resolve. This clears all 5 `artifact_status_drift`
   violations without touching the pack or the compiler. Whether this edit
   happens directly to ISSUE-036 (a closed artifact — confirm whether
   `resolved-by`/direct-edit is the right mechanism for a closed issue's own
   claim, or whether it needs a small backing plan) is a scoping question for
   the plan, not decided here.
2. **Finding 2 (engine capability, contained but real work).** Fold the
   grouped-const/var-block case into ISSUE-052 as its third confirmed
   instance (updating ISSUE-052's problem statement, live-instance table, and
   fixture set to include `KindScaffolding`-shaped grouped members) rather
   than building a fourth, parallel, narrower fix — the relational-rule
   `input_mode` ISSUE-052 already proposes is the correct mechanism for all
   three symbol classes (bare iota member, struct field, grouped block
   member) and building it three times would fragment one engine capability
   across three issues. Once it lands, SPEC-054's dropped `provides` entries
   for `RecipeManifest.Kind`/`Op.Kind` can be restored as real, verified
   contracts instead of the current inexpressible-entry disposition.
3. **Finding 3 (engine capability, contained but real work).** Fold the
   `kind: type` shape-verification gap into ISSUE-052 as a fourth confirmed
   instance (updating ISSUE-052's problem statement and live-instance table
   to include type/interface shape checks, alongside the bare iota member,
   struct field, and grouped block member instances) rather than building a
   fifth, parallel, narrower fix. Once it lands, `kind: type` `provides`
   entries (e.g. SPEC-055's `AddOptions`, SPEC-054's `WaiverReader`) stop
   being existence-only and the CLM-030-style dedicated-claim workaround
   becomes unnecessary going forward.
4. Whichever order these ship in, Finding 1 does not block or depend on
   Finding 2 or Finding 3 — the mandated-test repoint is independently
   actionable today.

## References

- `packs/contracts/scripts/compile-signature.sh` — the kind-aware compiler;
  its `const $NAME = $$$` / `var $NAME = $$$` emission is correct for a
  standalone declaration and does not (and structurally cannot, under
  `pattern-arg`) bind a grouped block member; its `type` branch
  (lines 78-89) is Finding 3's source — both the plain and `interface`
  cases terminate in a `$$$` wildcard, so a `kind: type` entry proves only
  that the named type exists, never its shape
- `packs/contracts/pack.yml` — the five per-kind static rules
  (`type-signature`, `const-signature`, `var-signature`,
  `method-signature`, `interface-signature`) whose `claims[].id` values are
  ISSUE-036's unresolvable CLM-008 mandated-test names; `input_mode:
  pattern-arg` is the substrate limitation behind Findings 2 and 3;
  `type-signature`'s `pattern: "type $NAME $$$"` (lines 70-73) is the static
  rule that exercises Finding 3's existence-only match
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md:1028` — the
  `AddOptions` contract entry note documenting that the compiler "reduces
  any struct to `type AddOptions $$$` ... and never compares field lists,"
  forcing the dedicated CLM-030 field-absence claim to carry real
  enforcement; Finding 3's first precedent
- `plans/PLAN-ISSUE-080-malformed-waiver-diagnostic-surfacing.plan.yml:732-734`
  — the Phase 5 reconciliation task's now-corrected premise that leaving
  SPEC-054's `WaiverReader` signature unedited "reds the gate as contract
  drift"; Finding 3's second precedent, caught and corrected in plan review
- `pkg/pack/engine/contracts_kind_signature_test.go` — the
  `TestContractCompiler_*` suite; the 9 tests that already prove ISSUE-036's
  per-kind claims under names the gate CAN resolve
- `cmd/backstop/gate.go:1179-1198` (`mergeTestNameMatcher`) — builds
  `gate.TestNameMatcher` from toolchain-pack `TestNamePatterns` only; the
  reason a pack-level (enforcement-pack) claim id can never register as a
  present test
- `pkg/recipe/manifest.go:18-32` — `KindScaffolding`/`KindImplementing`/
  `KindTemplating` and `OpCreate`/`OpMerge`/`OpTransform`/`OpInsert`/`OpStep`,
  the live grouped-const-block instances that forced this triage
- `specs/SPEC-054-recipe-apply-and-manifest.spec.md:631,635,672-675,914` —
  the spec's notes documenting the dropped-entry workaround and
  forward-referencing this issue by number
- ISSUE-036 (`contracts-pack-compiler-func-only-signatures`) — closed,
  success-terminal; the source of Finding 1's currently-red CLM-008
- ISSUE-037 (`contracts-compiler-iota-member-const-support`) — replaced by
  ISSUE-052; the first instance of the structural-presence gap this issue's
  Finding 2 is a third instance of
- ISSUE-038 (`reconcile-contract-drift-exposed-by-kind-aware-compiler`) —
  found the struct-field variant (second instance) with 3 live grandfathered
  contracts
- ISSUE-052 (`contracts-engine-relational-rule-input-mode`) — open,
  technical-debt; the general fix (relational-rule `input_mode`) this
  issue's Finding 2 and Finding 3 both recommend folding into rather than
  duplicating
- CLAUDE.md — "Loud ≠ blocking" and the zero-baked-checks / no-vacuous-green
  first principle; Finding 1 is a gate false-red (loud on a real capability),
  Finding 2 is a real, honestly-worked-around gap rather than a silently
  fabricated contract
