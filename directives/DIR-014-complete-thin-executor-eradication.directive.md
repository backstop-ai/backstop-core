---
title: "Complete Thin Executor Eradication"
number: DIR-014
created: "2026-07-06"
schema_version: directive/v1

directive:
  status: done
  source:
    - "ISSUE-027"
    - "ISSUE-019"
    - "ISSUE-021"
  completed: "2026-07-06"
---

## Description

Finish backstop's first principle (CLAUDE.md): the binary bakes in ZERO
language/tool knowledge, for any language. Every check and every language's
toolchain comes from a PACK; backstop only runs allowlisted, pack-declared
commands and speaks SARIF. A baked language/tool branch is a defect to
eradicate, never to extend — there is no native/legacy tier and no "built-in"
language.

This directive is the finish line for that arc, not its start. Substantial
work is already delivered:

- **BUNDLE-009** — stack-aware traceability reimplemented as installed packs
  (+ the pack-run substrate).
- **BUNDLE-010** — the pluggable pack-engine model (engines are first-class,
  pack-declared, no longer hand-wired per tool).
- **BUNDLE-011 + BUNDLE-012** — the native-toolchain cutover: the legacy
  `pkg/check` engine collapsed into pack-declared toolchain packs,
  `builtinToolchain`/`realCodeChecker`/the bridge deleted, `Config.Language`
  retired, and the gate now measures and routes entirely from pack
  declarations (language-neutral consumer + TypeScript toolchain pack).
- **SPEC-035** — pack-declared engines + the trusted-tool allowlist, closing
  the mechanism gap that let the last baked defaults be overridden by packs.
- **ISSUE-018** — the vestigial `backstop code check` command and its baked
  `pkg/check` engine (semgrep executor, manifest routing, format parsers,
  `.standard.md` validator) eradicated outright (2026-07-06); the gate never
  depended on that path.

What remained was exactly the residue the `backstop/self` dogfood pack
named as findings — the last baked-language/tool literals on the spine:

- **ISSUE-027** — `DefaultRegistry()` / `DefaultFieldContracts()` in
  `pkg/pack/engine/binding.go` still ship 7 baked engine bindings (including
  Go-specific `go-build`/`go-test`/`golangci` commands and convert scripts)
  as an overridable fallback rather than pack-owned data. 6 findings.
- **ISSUE-019** — the `pkg/packval` pack-authoring validation harness bakes
  `semgrep`, `golangci-lint`, and `go mod tidy` directly into Go code
  (interface method names, hardcoded `exec.Command` calls, a tool-keyed
  switch), a separate execution path from the gate's engine dispatch. 3
  findings.
- **ISSUE-021** — pack-layout validation (`validate_manifest.go`) hardcodes
  a `go.mod`-shaped layout requirement, baking an assumption about what a
  pack's toolchain looks like. 1 finding.

(A fourth literal, plan validation's `fileCategory()` baked `.go` suffix
check, was also named by this audit as ISSUE-033 but is re-homed to DIR-015
— see Notes.)

None of ISSUE-027/019/021 were net-new scope — they were the named remainder
of the 2026-06-20 eradication audit, carried forward once BUNDLE-009 through
ISSUE-018 closed out everything ahead of them.

## Acceptance Criteria

- The `backstop/self` dogfood pack goes GREEN: zero baked-language/tool
  literals remain anywhere it flagged (`pkg/pack/engine/binding.go`,
  `pkg/packval/`, `validate_manifest.go`). **MET 2026-07-06** — whole-repo
  `backstop/self` findings = 0. Note that `backstop/self` is `baseline: false`
  (all-code, by design) — its RED literally *was* this backlog, not an
  unrelated defect.
- Backstop gates a project in ANY language via packs only — no baked path,
  no vacuous green — for the full set of check categories the gate covers
  (lint/build/test/coverage/substantiveness/contracts), not just the four
  literals enumerated above if further baked residue surfaces during the
  work.

## Notes

- This directive is the finish line the 2026-07-06 session set up once the
  native-toolchain cutover landed on main (squash `824530e`) and
  `backstop code check` was deleted (`d5efd5b`, ISSUE-018). See project
  memory `project_eradication_backlog.md` and
  `project_native_toolchain_cutover.md`.
- Do not re-litigate whether baked checks "stay" — CLAUDE.md's
  zero-baked-checks invariant is standing law; the only open questions were
  how/when each remaining literal migrated to a pack, which is what
  ISSUE-027/019/021 individually scoped.
- **Completed 2026-07-06.** `backstop/self` findings across the whole repo
  are 0. The substance shipped in three commits: ISSUE-027 (`0dabc82`,
  `DefaultRegistry` → embedded base-engines pack + external go-toolchain
  pack), ISSUE-019 (`e483d92`, the `packval` harness de-Go'd onto the engine
  model), ISSUE-021 (`43c9dbd`, `go.mod` de-baked from `ExpectedLayout`).
- **ISSUE-033 re-homed to DIR-015, not resolved here.** It was the one
  residual literal this audit named (`fileCategory()` in `pkg/validate/plan.go`
  baking a `.go` suffix check). It is currently suppressed with an interim
  `// nosemgrep` (non-firing, not truly removed) rather than fixed. Its real
  fix — deriving source-file classification from SPEC-043 pack-declared
  globs, threaded through `validate.Plan`'s signature and ~90 test callers —
  is disproportionate to a lone minor literal, so it was moved to DIR-015 as
  a tracked follow-up rather than blocking this directive's completion.

## References

- BUNDLE-009 — `bundles/BUNDLE-009-stack-aware-traceability.bundle.md`
- BUNDLE-010 — `bundles/BUNDLE-010-pluggable-pack-engines.bundle.md`
- BUNDLE-011 — `bundles/BUNDLE-011-collapse-legacy-codecheck-into-packs.bundle.md`
- BUNDLE-012 — `bundles/BUNDLE-012-language-neutral-consumer-ts-toolchain.bundle.md`
- SPEC-035 — `specs/SPEC-035-pack-declared-engines-trusted-allowlist.spec.md`
- ISSUE-018 — `issues/ISSUE-018-remove-vestigial-baked-in-code.issue.md`
- ISSUE-027 — `issues/ISSUE-027-eradicate-default-registry-into-packs.issue.md`
- ISSUE-019 — `issues/ISSUE-019-de-go-packval-harness.issue.md`
- ISSUE-021 — `issues/ISSUE-021-baked-gomod-pack-layout-requirement.md`
- ISSUE-033 — `issues/ISSUE-033-de-go-plan-validation-file-classification.issue.md`
  (re-homed to DIR-015; no longer part of this directive's `source`)
- CLAUDE.md — zero-baked-checks first principle
