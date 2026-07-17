---
title: "Resolve By Declaration Not Name — Route Findings and Label Stacks By Declared Role/Language, Not Baked Name Literals"
schema_version: issue/v1

issue:
  id: ISSUE-064
  title: "Resolve By Declaration Not Name — Route Findings and Label Stacks By Declared Role/Language, Not Baked Name Literals"
  type: technical-debt
  status: closed
  created: "2026-07-17"
  closed: "2026-07-17"

delivered_by: PLAN-ISSUE-064

complexity:
  scope: contained
  uncertainty: known
  risk: moderate

verification:
  level: unit
  coverage_threshold: 90
  test_command: "go test ./cmd/backstop/... ./pkg/gate/... -run 'Substantiveness|ToolchainStackLabel|IsToolchainPack|ByDeclaration|SelfRule'"

implementation:
  summary: >
    Two remaining behavioral decisions in cmd/backstop + pkg/gate still key on a baked NAME
    literal where a declared role/property is the correct source, both surfaced by the sweep that
    produced ISSUE-063. (1) Substantiveness finding ROUTING partitions violations by hardcoded
    namespaced rule ids (`hollow-test-go` / `referenced-symbol-go`) instead of a declared role the
    pack stamps in the finding. (2) The cosmetic toolchain stack LABEL derives both membership and
    value from the `-toolchain` name suffix instead of the declared mechanism engine and the
    pack's declared `language:`. Route by declared role and label by declaration in both; delete
    the name-keyed constants and helpers; add a backstop/self rule family (B7) catching rule-id /
    engine-key / finding-role NAME literals used in routing or selection (the sibling B6 misses
    these — it only catches PACK-name keying). A third site — the contracts signature engine keyed
    on the `ast-grep-contracts` engine key — is deferred to ISSUE-065 (it needs a NEW declared
    engine capability; no existing field discriminates the two contracts engines). B7 flags it but
    ships authored-not-activated, so no waiver.
  package: cmd/backstop, pkg/gate, backstop-self-pack

requirements:
  - id: REQ-001
    text: >
      Substantiveness finding routing (`RouteSubstantivenessFindings`, pkg/gate) must
      partition violations by a pack-declared ROLE carried in the finding's structured
      properties channel (`Violation.Properties["substantiveness_role"]`, the ISSUE-062
      channel) — role values `hollow` (a test with no assertion) and `referenced-symbol` (a
      symbol the test references, used for the subject-join). It MUST NOT partition by matching
      a hardcoded namespaced rule id. A pack may name its rules anything (`hollow-test-ts`,
      `hollow-test-go`, org-specific names) as long as each finding declares its role; core
      routes purely on the declared role. The versioned local pack `packs/substantiveness`
      (source_type local) stamps the role in its `ast-grep/to-sarif.sh`, reinstalled so the
      `.backstop/packs` copy the gate runs carries it.
  - id: REQ-002
    text: >
      The rule-name routing constants `substantivenessHollowRuleName = "hollow-test-go"` and
      `substantivenessExtractionRuleName = "referenced-symbol-go"` (cmd/backstop/gate.go) and
      the namespaced-rule-id parameters they feed into `RouteSubstantivenessFindings` are
      DELETED from the routing path. Every existing caller of the old rule-id signature (the
      prod call site plus the pkg/gate roundtrip/migration/strangler tests) is UPDATED IN PLACE
      to the role-property signature — signature migration only — so both packages compile.
      Caller files carrying surviving mandated tests MUST NOT be deleted:
      `substantiveness_migration_test.go` (five SPEC-037 mandated tests) and
      `substantiveness_roundtrip_test.go` (an ISSUE-062-mandated test) are edited, not retired;
      if any individual mandated test is genuinely obsolete, its owning claim is repointed or
      retired in the SAME change (no dangling claims, ISSUE-014). No language-suffixed rule-name
      literal remains in core's finding-routing logic (a rule name may remain only as a
      display/log string, never as the routing key).
  - id: REQ-003
    text: >
      The cosmetic toolchain stack label (`declaredToolchainStackLabel`, cmd/backstop/gate.go)
      must (a) determine toolchain-pack MEMBERSHIP via `declaresToolchainMechanism` — the same
      by-declaration primitive `countToolchainPacks` uses — not the `-toolchain` name suffix;
      and (b) derive each label VALUE from the pack's declared `manifest.Language`, not by
      string-stripping the `-toolchain` suffix off the name. A mechanism pack that omits
      `language:` contributes no token; an empty label set still returns `"unspecified"`. The
      `isToolchainPack` helper and its `-toolchain` name-suffix test (and its B6 waiver) are
      DELETED. The existing name-derived label assertions (`gate_stack_label_test.go`) are
      updated to the declared-language premise, and the shared `spec046ToolchainManifest` stub
      declares `Language`. After this change the label set and the count set are backed by the
      identical membership primitive and cannot disagree.
  - id: REQ-004
    text: >
      `backstop/self` must gain a rule family (B7) flagging routing or dispatch/selection
      logic keyed on a baked NAME literal — a hardcoded rule-id / rule-name string, an engine
      KEY string literal, or a finding-role string literal used to partition, route, or select
      — where a declared role/capability property is the correct source. This is the
      baked-routing-identity class, sibling to B6 (pack-name-keyed capability, ISSUE-063): B6
      catches `cfg.Packs["org/pack"]` and `-toolchain`-style PACK-name keying but does NOT
      catch rule-id or engine-key keying. It ships in BOTH the in-repo self-rule test harness
      (`pkg/gate/testdata/self-rule/` + `pkg/gate/self_rule_test.go`) and the installed
      `backstop/self` pack copy, with a positive fixture (a rule-id-keyed partition) and a
      negative fixture (a role-property partition). Like B5/B6 it is authored and fixture-tested
      but NOT activated into the live `.backstop/packs` gate — activation held for separate
      triage (it would flag the ISSUE-065 contracts-engine site).
  - id: REQ-005
    text: >
      Existing behavior is preserved for the current corpus. With the substantiveness pack
      stamping the declared role, routing partitions exactly as the rule-id match did today
      (same hollow/subject-join verdicts). With the current install, the stack label renders the
      same string (`go-toolchain` declares `language: go` → "go", identical to the old
      suffix-strip). No finding flips hollow<->substantive and no label changes for an
      already-correct install. This is a mechanism swap, not a policy change.

claims:
  - id: CLM-001
    requirement: REQ-001
    text: Substantiveness findings are partitioned by the declared substantiveness_role property.
    tests:
      - TestRouteSubstantivenessFindings_PartitionsByRoleProperty
  - id: CLM-002
    requirement: REQ-001
    text: Substantiveness routing ignores the rule id and routes findings whose rule names differ from the Go defaults.
    tests:
      - TestRouteSubstantivenessFindings_RoutesRegardlessOfRuleName
  - id: CLM-003
    requirement: REQ-002
    kind: absence
    text: The hollow-test-go and referenced-symbol-go rule-name routing constants are removed.
    tests:
      - TestSubstantivenessRuleNameConstantsRemoved
  - id: CLM-004
    requirement: REQ-003
    text: The toolchain stack label value is the pack-declared language, not the name-stripped suffix.
    tests:
      - TestToolchainStackLabel_ByDeclaredLanguageNotName
  - id: CLM-005
    requirement: REQ-003
    text: Toolchain-pack membership for the label is by declared mechanism engine, not the -toolchain name suffix.
    tests:
      - TestToolchainStackLabel_MembershipByMechanismDeclaration
  - id: CLM-006
    requirement: REQ-003
    kind: absence
    text: The isToolchainPack name-suffix helper is removed.
    tests:
      - TestIsToolchainPackRemoved
  - id: CLM-007
    requirement: REQ-004
    text: The self rule flags a rule-id-keyed finding partition.
    tests:
      - TestSelfRule_FlagsRuleIDKeyedRouting
  - id: CLM-008
    requirement: REQ-005
    text: With the current install, substantiveness routing and the stack label are unchanged.
    tests:
      - TestByDeclaration_CurrentInstallUnchanged

contracts:
  - file: cmd/backstop/gate.go
    provides:
      - name: substantivenessHollowRuleName
        kind: const
        signature: "const substantivenessHollowRuleName"
        absent: true
      - name: substantivenessExtractionRuleName
        kind: const
        signature: "const substantivenessExtractionRuleName"
        absent: true
      - name: isToolchainPack
        kind: function
        signature: "func isToolchainPack(m *pack.Manifest) bool"
        absent: true
---

# Resolve By Declaration Not Name — Route Findings and Label Stacks By Declared Role/Language, Not Baked Name Literals

## Problem

ISSUE-063 moved traceability-CAPABILITY detection off pack names and onto declared
`gate_type`, and added backstop/self Family B6 to keep pack-name-keyed capability logic out
of core. A follow-up sweep of `cmd/backstop` + `pkg/gate` for the same anti-pattern found
behavioral decisions that B6 does NOT catch — each keys on a baked NAME literal where a
declared role/property is the correct source. Two are fixable now with declared sources that
already exist:

1. **Substantiveness finding routing keys on rule-id literals.** `RouteSubstantivenessFindings`
   (pkg/gate) partitions violations by matching hardcoded namespaced rule ids built from
   `substantivenessHollowRuleName = "hollow-test-go"` and
   `substantivenessExtractionRuleName = "referenced-symbol-go"` (cmd/backstop/gate.go). A
   consuming pack's rules must therefore be named exactly `hollow-test-go` /
   `referenced-symbol-go` to route — even a TypeScript pack, which is why the TS
   substantiveness pack had to name its rule `hollow-test-go` (a `-go` literal on a TS pack) to
   be recognized at all. The routing key is a Go-shaped rule NAME, not the finding's role.

2. **Toolchain stack label keys on the `-toolchain` name suffix — twice.**
   `declaredToolchainStackLabel` decides membership via `isToolchainPack`
   (`strings.HasSuffix(NormalizedName, "-toolchain")`) and derives each label value by
   string-stripping the `-toolchain` suffix off the name (`backstop/go-toolchain` → "go"). This
   is the exact site ISSUE-063's Resolution flagged as the one remaining B6 hit, waived as
   "cosmetic label, convention is correct here." But the pack already DECLARES the by-declaration
   sources: `declaresToolchainMechanism` (the primitive `countToolchainPacks` uses) for
   membership, and `manifest.Language` (`go-toolchain` declares `language: go`) for the value.
   Leaving it name-keyed makes the label set and the count set derive membership two different
   ways — they can disagree (a `-toolchain`-named pack with no mechanism engine labels but does
   not count; a mechanism pack not named `*-toolchain` counts but does not label).

A third site — `contractSignatureEngine` keyed on the literal engine key `"ast-grep-contracts"`
— is the same family but is **deferred to ISSUE-065**. gate_type is intentionally non-exclusive
(the contracts pack declares `gate_type: contracts` on both its presence and absence engines),
and no existing engine field discriminates them, so a clean fix needs a NEW declared engine
capability rather than a name swap. The B7 rule (REQ-004) flags that site but ships
authored-not-activated, so no waiver is added.

## Root cause

Routing/label identity is keyed on a distribution-or-naming artifact (a rule id, a name suffix)
instead of on what the pack DECLARES about the finding. This is the same conflation ISSUE-063
fixed for pack-name→capability, one level down at the rule / label granularity — and B6, scoped
to PACK-name keying, does not reach it. The declared sources already exist for both fixable
sites: the ISSUE-062 structured-properties channel (`Violation.Properties`) can carry a role;
`manifest.Language` and `declaresToolchainMechanism` already carry the label's membership and
value. Each fix is a mechanism swap, not new plumbing.

## Fix

1. **Route substantiveness findings by declared role (REQ-001/REQ-002).** The versioned local
   `packs/substantiveness` stamps `properties.substantiveness_role` (`hollow` /
   `referenced-symbol`) on each finding via the ISSUE-062 channel (reinstalled into
   `.backstop/packs`); `RouteSubstantivenessFindings` partitions on that property and drops the
   namespaced-rule-id parameters. Delete `substantivenessHollowRuleName` /
   `substantivenessExtractionRuleName`, and update/retire every existing caller of the old
   signature so both packages compile.
2. **Label stacks by declaration (REQ-003).** `declaredToolchainStackLabel` filters by
   `declaresToolchainMechanism` and emits each pack's declared `manifest.Language`; delete
   `isToolchainPack`, its suffix test, and its B6 waiver; update the name-derived label
   assertions and the shared stub. Label membership and count membership now share one primitive.
3. **self rule closes the class (REQ-004).** backstop/self gains Family B7 flagging rule-id /
   engine-key / finding-role NAME literals used in routing or selection — the
   baked-routing-identity class, sibling to B6's baked-distribution-identity — in both the in-repo
   test harness and the installed pack copy, authored-not-activated.
4. **Behavior-preserving (REQ-005).** Current install routes and labels exactly as today; a
   mechanism swap, not a policy change.

## Out of scope

- **The contracts signature engine selector (`contractSignatureEngine` / `ast-grep-contracts`
  key) — ISSUE-065.** Deferred after confirming gate_type is non-exclusive and no existing engine
  field discriminates the two contracts engines; it needs a new declared engine capability, a
  bigger change than these name swaps. B7 flags it but is not activated live.
- The `org/pack-name` distribution scheme (SPEC-015) and the ISSUE-063 capability-by-declaration
  detectors themselves — unchanged; this issue is strictly the rule/label granularity B6 does not
  reach.
- Any change to WHAT makes a test hollow or how the subject-join decides substantiveness — only
  HOW findings are routed to those checks changes.

## Notes / references

- All targets surfaced by the post-ISSUE-063 sweep of `cmd/backstop` + `pkg/gate` for
  baked-identity patterns. The sweep confirmed the remaining name-shape tests in the area key on
  backstop's OWN artifact/config format (`.spec.md`, `.plan.yml`, `backstop.yml`,
  `.backstop/packs/`) or are structural (rule namespacing), which are language-neutral spine
  conventions and correctly left as-is.
- Sibling to ISSUE-062 (structured properties — the channel REQ-001 reuses) and ISSUE-063
  (capability by declaration — the pattern, and the B6 rule B7 extends). ISSUE-063's Resolution
  explicitly deferred the `isToolchainPack` site handled here by REQ-003.
- `packs/substantiveness` is a LOCAL in-repo pack (`backstop.yml: backstop/substantiveness:
  local`); its source of truth is `packs/substantiveness/ast-grep/to-sarif.sh`, NOT an external
  repo — the `.backstop/packs` copy is a gitignored install artifact and must be re-synced after
  the source edit. (The TS substantiveness pack's `hollow-test-ts` rename is an external
  backstop-packs concern; no TS substantiveness pack is installed in this repo, so it is not a
  core test dependency here.)

## Resolution

Delivered by PLAN-ISSUE-064 (P1→P4), reviewed by plan-reviewer (two rounds) and impl-reviewer.

- **REQ-001/REQ-002 (route by declared role).** `RouteSubstantivenessFindings` (pkg/gate/substantiveness_join.go)
  now partitions on `Violation.Properties["substantiveness_role"]` (`hollow` / `referenced-symbol`,
  the ISSUE-062 channel) and takes only `(violations)` — the `hollowRuleID`/`extractionRuleID`
  parameters are gone. The role is stamped by the versioned LOCAL pack
  `packs/substantiveness/ast-grep/to-sarif.sh` (re-synced into `.backstop/packs`). The constants
  `substantivenessHollowRuleName`/`substantivenessExtractionRuleName` are deleted from
  cmd/backstop/gate.go (contract `absent: true` honored); the prod call site is
  `RouteSubstantivenessFindings(flat)`. All old-signature callers were migrated IN PLACE —
  `substantiveness_migration_test.go` (5 SPEC-037 tests) and `substantiveness_roundtrip_test.go`
  (an ISSUE-062 test) were edited, not deleted.
- **REQ-003 (label by declaration).** `declaredToolchainStackLabel` filters by
  `declaresToolchainMechanism` and emits each pack's declared `manifest.Language`; `isToolchainPack`,
  its `-toolchain` suffix test, and its B6 waiver are deleted (contract `absent: true`). Label and
  count membership now share one primitive.
- **REQ-004 (self rule B7).** `no-rule-id-keyed-routing` ships in both the in-repo harness
  (pkg/gate/testdata/self-rule/no-baked.yml + pkg/gate/self_rule_test.go) and the installed
  `backstop/self` source (../backstop-self-pack), with valid/invalid fixtures. Authored-not-activated
  — NOT synced into `.backstop/packs`, so it does not flag the deferred ISSUE-065
  `ast-grep-contracts` site or red the live gate.
- **REQ-005 (behavior-preserving).** `TestByDeclaration_CurrentInstallUnchanged` proves by-role
  routing partitions identically to the deleted by-rule-id routing and the real install still labels
  "go". All 8 mandated tests pass; `go test ./...` is module-wide green; `backstop gate --all` passes.
- **impl-review catch.** A wiring test (`gate_substantiveness_wiring_test.go`) fed a synthetic
  finding into the routing path without the role property and regressed; it was invisible to the
  scoped `-run` `test_command` and masked at the gate as an opaque go-test crash. Fixed by stamping
  the role on the injected seam finding. The two masking mechanisms are tracked as ISSUE-066
  (gate must run full packages, not a `-run` filter) and ISSUE-067 (go-test engine must surface
  failures as findings, not an opaque crash).
- **Deferred: ISSUE-065.** `contractSignatureEngine` / the `ast-grep-contracts` engine-key literal
  is untouched — gate_type is non-exclusive and no existing engine field discriminates the two
  contracts engines, so it needs a new declared engine capability.
