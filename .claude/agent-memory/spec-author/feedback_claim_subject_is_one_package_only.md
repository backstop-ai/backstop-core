---
name: claim-subject-is-one-package-only
description: A claim's mandated tests must never span two packages — subject: is one value per claim, TargetPackageName reduces to a single token, so no subject can satisfy a package-straddling claim; the trap stays dormant until closure because the noTarget join only runs on implemented status
metadata:
  type: feedback
---

Found closing SPEC-018 (2026-08-02): CLM-001 mandated two tests, one in
`cmd/backstop` and one in `pkg/gate`. `subject:` overrides (ISSUE-047)
resolved every other claim's package mismatch, but a single subject value
cannot satisfy a claim whose own two tests live in two different packages
— `TargetPackageName` reduces to one token, full stop.

**Why it stays hidden:** `test_substantiveness`'s noTarget join only
activates once a spec is `implemented`/terminal. A claim can carry this
defect for the spec's ENTIRE non-terminal lifetime — sat dormant on
SPEC-018 since spec_version 1.1.0 — and only detonates at closure, the
worst possible moment to discover a modeling defect.

**How to apply:** when authoring or auditing a claim, check whether ALL of
its mandated tests share one package BEFORE relying on a single `subject:`
to cover them. If a claim's own tests straddle packages, split it — one
claim ID per package, each test's real subject, requirement unchanged,
assertion content redistributed faithfully (not narrowed, not dropped).
Claim IDs are `^CLM-\d{3}$` (pkg/validate/spec.go) — no suffix form, so a
split claim gets the next free number, not a lettered variant.

**Detect this WITHOUT a live status flip** (the cheap, safe check): copy
the spec into a scratch dir with status forced to `implemented`, run
`gate.ExtractMandatedTests` over it, and compare each result's `TargetPkg`
against its real test file's directory. A live flip on the working tree
risks landing exactly the violations you're trying to catch. See
[[project_new_file_coverage_floor]] for the sibling "dormant until you
touch it" class.
