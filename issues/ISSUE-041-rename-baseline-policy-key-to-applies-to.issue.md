---
title: "Rename Baseline Policy Key To Applies To"
schema_version: issue/v1

issue:
  id: ISSUE-041
  title: "Rename Baseline Policy Key To Applies To"
  type: technical-debt
  status: open
  created: "2026-07-06"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-041: Rename Baseline Policy Key To Applies To

## Problem

The gate's per-dimension enforcement policy (`enforcement.policy.<dimension>`
in `backstop.yml`) uses a boolean `baseline:` key that misdescribes the
mechanism it controls. Every dimension is always compared to the baseline
file — that comparison is unconditional. What `baseline: true/false` actually
decides is whether *pre-existing* (already-baselined) violations are
**grandfathered** — excluded from the pass/fail verdict — or whether *every*
violation counts regardless of when it was introduced.

Read cold, `baseline: true` sounds like "this dimension participates in the
baseline," which is true of every dimension and therefore says nothing. The
name describes the mechanism's *input* (the baseline file) rather than its
*enforcement effect* (grandfathered vs. zero-tolerance). This has already
caused friction during the 2026-07-06 gate-hardening work: `contract_signature`
was set to `baseline: true` (grandfather pre-existing contract drift) while
`backstop/self`'s scoped override on `pack_engines` was deliberately kept at
`baseline: false` (zero-tolerance on backstop's own zero-baked-checks rule) —
the correct calls, but the key name does not make the *reason* self-evident
the way a well-named enum would.

### Where this lives (verified 2026-07-06)

- **Parser / struct definition** — `pkg/config/config.go:71-75`, the
  `DimensionPolicy` struct:
  ```go
  type DimensionPolicy struct {
      Level    string                     `yaml:"level,omitempty" json:"level,omitempty"`
      Baseline bool                       `yaml:"baseline,omitempty" json:"baseline,omitempty"`
      Sources  map[string]DimensionPolicy `yaml:"sources,omitempty" json:"sources,omitempty"`
  }
  ```
  `Sources` is the nested per-pack/per-rule-source override (SPEC-047
  REQ-007) — it carries its own `Baseline` field, so the rename must cover
  the nested key too, not just the top-level one.
- **Consumer / grandfathering decision** — `pkg/gate/policy.go`:
  - `DimensionPolicy` struct mirrored again at lines 27-31 (this package
    doesn't import `pkg/config`'s type — a second struct with the same shape
    and the same `Baseline bool` field; the rename touches both).
  - `ApplyPolicy` (lines 53-122): `if p.Baseline && baseline != nil` at line
    98 is the unscoped grandfathering branch — when true, `CompareBaseline`
    narrows the counted violations to `cmp.NewViolations` (net-new only);
    when false, every violation in `s.Violations` counts.
  - `applyScopedPolicy` (lines 124-194): `if eff.Baseline && baseline != nil`
    at line 167 is the same decision, resolved per-source (`p.Sources[v.SourcePack]`)
    for the SPEC-047 per-pack override path.
- **Schema gap (found during authoring, worth fixing in the same change)** —
  `artifacts/backstop-yml/v1/schema.json` has **no `policy` property at all**
  under `enforcement` (verified: zero case-insensitive matches for `polic` in
  the file), despite `enforcement` declaring `additionalProperties: false`.
  The committed `backstop.yml`'s `enforcement.policy` block currently loads
  without a schema validation error only because nothing in `pkg/config`'s
  `validateAgainstSchema` path is rejecting it in practice — this is a
  pre-existing schema-fit gap, not something this rename introduces, but
  since the rename touches this exact area, add the `policy` object (with the
  new `applies-to` enum) to the schema in the same change rather than leaving
  the gap to be rediscovered later.
- **The 5 committed policy entries** in `backstop.yml` today, all needing the
  key rename:
  ```yaml
  enforcement:
      policy:
          artifact_validation:
              level: block                    # no baseline key — already all-code equivalent
          contract_signature:
              baseline: true                   # -> applies-to: new-code
              level: block
          coverage_threshold:
              baseline: true                   # -> applies-to: new-code
              level: block
          pack_engines:
              baseline: true                   # -> applies-to: new-code
              level: block
              sources:
                  backstop/self:
                      baseline: false          # -> applies-to: all-code  (zero-baked-checks non-negotiable)
                      level: block
          test_substantiveness:
              baseline: true                   # -> applies-to: new-code
              level: block
          test_verification:
              level: block                     # no baseline key — already all-code equivalent
  ```
  The nested `pack_engines.sources.backstop/self` override is the one
  easiest to miss in a mechanical find/replace — it must land as
  `applies-to: all-code`, matching backstop/self's standing zero-tolerance
  role on its own dogfood dimension (see CLAUDE.md "Zero baked checks" /
  memory `feedback_zero_baked_checks`).

### Why `applies-to: new-code | all-code` (decided direction, 2026-07-06)

This borrows SonarQube's well-established framing for exactly this
distinction — "New Code" vs "Overall Code" quality gates. Codecov's
patch/project coverage split and reviewdog's filter-mode are the same
underlying concept (the general term is "ratchet"); SonarQube's specific
vocabulary was chosen because it is close to impossible to misread cold,
unlike a bare boolean:

- `baseline: true` → `applies-to: new-code` — fail only on violations NOT in
  the baseline (net-new debt counts; pre-existing is tolerated/grandfathered
  — this is the ratchet ISSUE-038's contract-drift backlog and similar
  grandfathered debt rely on).
- `baseline: false` / key absent → `applies-to: all-code` — fail on ANY
  violation in the dimension, baselined or not (zero tolerance) — this is
  `backstop/self`'s stance on its own `pack_engines` override, and the
  default for any dimension with no policy entry at all.

`level` (off/warn/block — what happens when a violation counts) stays a
**separate, orthogonal** key. `applies-to` only decides *which* violations
count; `level` decides *what happens* once they do. The two must not be
merged or conflated — a dimension can be `applies-to: new-code` +
`level: warn` (surface new debt without failing the gate) just as validly as
`applies-to: all-code` + `level: block`.

## Solution

1. **Rename the field** in both `DimensionPolicy` structs
   (`pkg/config/config.go:73`, `pkg/gate/policy.go:29`, and the matching
   field on each struct's nested `Sources` entries) from `Baseline bool` to
   an `AppliesTo string` (or equivalent enum-typed) field with YAML/JSON tags
   `applies-to`. Two enum values: `"new-code"` and `"all-code"`.
2. **Default semantics preserved**: an absent `applies-to` key (or an absent
   policy entry entirely) must resolve to `all-code` — matching today's
   key-absent = block-on-total behavior. This is the safe/strict default and
   must not silently change on this rename.
3. **Update the consumers** (`ApplyPolicy` line 98, `applyScopedPolicy` line
   167) to branch on `AppliesTo == "new-code"` instead of `Baseline == true`.
4. **Add `enforcement.policy` to the `backstop-yml/v1` JSON schema**
   (currently entirely absent — see Problem's schema-gap note), including the
   `applies-to` enum (`new-code`/`all-code`), `level` enum
   (`off`/`warn`/`block`), and the nested `sources` map shape, so the schema
   actually describes what backstop.yml has carried since SPEC-047.
5. **Migration — clean one-time cutover (recommended over a back-compat
   shim)**: rewrite the 5 `baseline:` entries in the committed `backstop.yml`
   to `applies-to:` in the same change (see mapping table above). A
   dual-accepting shim (`baseline:` deprecated-but-still-read alongside
   `applies-to:`) was considered and rejected for now: this repo is the only
   consumer of `enforcement.policy` today (no external `backstop.yml` files
   exist yet per project memory `project_pack_distribution` /
   `project_launch_plan` — pre-public-launch), so a shim buys migration safety
   nobody needs yet and leaves a dead alias + double-parsing branch to
   maintain. If/when packs and consumer projects exist externally with their
   own `backstop.yml` policy blocks, revisit whether a deprecation window is
   warranted at that time — this issue's recommendation is scoped to today's
   single-consumer reality, not a permanent stance against ever supporting a
   transition alias.
6. **Re-run the gate** after the rename to confirm `contract_signature`,
   `coverage_threshold`, `pack_engines` (including the `backstop/self`
   scoped override), and `test_substantiveness` all resolve to the same
   grandfathering behavior as before the rename — this is a pure rename, the
   gate's pass/fail verdict on the current tree must not change.

## Verification

Unit-level: `go test ./pkg/config/... ./pkg/gate/... -run Policy` should cover
both `DimensionPolicy` structs' new `applies-to` field (parsing, default-to
`all-code` when absent, and the nested `sources` override), plus
`ApplyPolicy`/`applyScopedPolicy`'s branch on `AppliesTo == "new-code"`
producing identical grandfathering behavior to the current `Baseline == true`
branch. A full `go run ./cmd/backstop gate` pass after the `backstop.yml`
cutover should show unchanged pass/fail verdicts on every policy-governed
dimension (`contract_signature`, `coverage_threshold`, `pack_engines`
including the `backstop/self` scoped override, `test_substantiveness`) —
this is a pure rename, not a behavior change. Formal `requirements:` /
`claims:` / `verification:` frontmatter blocks are deferred to when this
issue moves to `ready`, per the schema's status-gated requirements.

## References

- `pkg/config/config.go:71-75` — `DimensionPolicy` struct, the `Baseline bool`
  field to rename
- `pkg/gate/policy.go:27-31, 53-122, 124-194` — the second `DimensionPolicy`
  struct, `ApplyPolicy`, and `applyScopedPolicy` — the two grandfathering
  decision sites (`p.Baseline` line 98, `eff.Baseline` line 167)
- `artifacts/backstop-yml/v1/schema.json` — `enforcement.policy` is
  completely absent from this schema despite `additionalProperties: false`
  on `enforcement`; needs the `applies-to`/`level`/`sources` shape added
- `backstop.yml` — the 5 committed policy entries needing the key rename,
  including the `pack_engines.sources.backstop/self` nested override
- ISSUE-038 (reconcile-contract-drift-exposed-by-kind-aware-compiler) — the
  ratchet-down backlog that depends on `contract_signature`'s grandfathering
  (will read as `applies-to: new-code` after this rename)
- CLAUDE.md "Zero baked checks (STANDING RULE)" / memory
  `feedback_zero_baked_checks` — the reason `backstop/self`'s `pack_engines`
  override must map to `applies-to: all-code`, not `new-code`
- SPEC-047 — the spec that introduced the per-pack/per-rule-source `sources`
  scoping this rename must also cover
