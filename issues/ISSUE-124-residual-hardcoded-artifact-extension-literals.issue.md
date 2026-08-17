---
title: "Residual Hardcoded Artifact Extension Literals"
schema_version: issue/v1

issue:
  id: ISSUE-124
  title: "Residual Hardcoded Artifact Extension Literals"
  type: technical-debt
  status: closed
  created: "2026-08-14"
  closed: "2026-08-16"

delivered_by: PLAN-ISSUE-124

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: safe
---

# ISSUE-124: Residual Hardcoded Artifact Extension Literals

## Resolution

Removed all 15 hand-typed artifact-extension/directory literals named in the Problem
section (13 extension literals, including the 4 composed forms like `.epic.bundle.md`, and
2 directory literals), across six files, each now routing through `pkg/artifact`'s shared
layout authority (`LayoutFor`/`ClassifyFilename`) instead of an inline string:

- `pkg/gate/step_testverify.go` — 4 sites (`ExtractMandatedTests`, `CountTerminalSpecs`,
  `ExtractSpecVerifications`, `ExtractContractEntries`), converted to
  `artifact.ClassifyFilename`-based guards rather than an error-returning accessor — the
  right choice here since the classifier's bool already answers the question with no
  impossible state to error on.
- `pkg/validate/spec.go`, `pkg/validate/adr.go`, `pkg/validate/bundle.go`,
  `pkg/validate/supports_resolution.go` — converted to `LayoutFor`/`ClassifyFilename`.
- `cmd/backstop/gate_substantiveness_e2e.go` — a new shared `e2eSpecLayout(tmp)` helper
  used by both test-fixture builders, replacing the hardcoded `"specs"` directory literal.

A falsifying scan, `pkg/artifact/layout_consumer_scan_test.go`, went from RED (15 hits) to
GREEN (0). Verified via 6 targeted mutations, each producing its predicted red, all reverted
cleanly. All 7 mandated tests pass.

While fixing this, two pre-existing go-standards `error-wrapping-required` violations were
found and fixed in `gate_substantiveness_e2e.go` as a byproduct of the same edits. A
remaining coverage-floor red on that same file was investigated and proven **inherited, not
caused by this fix**: a control-worktree measurement showed the file was already under the
80% coverage floor (74.4%) at clean HEAD, before this lane touched it. The real fix is
downstream of ISSUE-148 (substantiveness pack fixture polarity), which is what's currently
suppressing the tests that exercise this file. Accepted as inherited debt rather than waived
or hacked green — `waiver_resolution` reports clean.

This closes DIR-002's last open roster member (noted here for context only; DIR-002 itself
is directive-author territory and is not edited by this closeout).

Delivered by `PLAN-ISSUE-124` (status: `completed`, committed as `5121645`).

## Problem

PLAN-SPEC-068 (trustworthy-green guards, implemented against
`specs/SPEC-068-trustworthy-green-guards.spec.md`) established `pkg/artifact`'s
`LayoutFor`/`Root.Dir` as the ONE shared authority for artifact-type-directory and
type-extension mappings, and de-duplicated six hardcoded copies of this table across the
codebase — most recently `pkg/validate/delivered_by.go`, fixed as an impl-reviewer finding
during that plan's implementation. While fixing `delivered_by.go`, the implementer swept the
codebase for the same defect class and found MORE instances that were deliberately left
out of PLAN-SPEC-068's scope (the fix request was `delivered_by.go` only). Each of these is a
candidate "seventh hardcoding" in the same family, but each needs its own verification —
confirm it is actually a project-root-relative artifact-type reference and not incidental
string matching — before being touched, which is why the sweep did not fix them inline.

Verified at HEAD (2026-08-14), the residual hits are:

- `pkg/gate/step_testverify.go:120,225,520,564` — four separate
  `strings.HasSuffix(entry.Name(), ".spec.md")` checks, each gating whether a directory entry
  is treated as a spec file, inside `ExtractMandatedTests`, `CountTerminalSpecs`, and
  `ExtractSpecVerifications` (two occurrences).
- `pkg/validate/spec.go:751` — `suffix := ".spec.md"` used to derive a spec's slug from its
  filename.
- `pkg/validate/adr.go:159` — `strings.HasSuffix(rest, ".adr.md")`.
- `pkg/validate/bundle.go:275-278` — `strings.HasSuffix(stem, ".epic.bundle.md")` /
  `strings.HasSuffix(stem, ".bundle.md")`.
- `pkg/validate/supports_resolution.go:252` — `strings.HasSuffix(art.Filename, ".issue.md")`.
- `cmd/backstop/gate_substantiveness_e2e.go:44` — `specDir := filepath.Join(tmp, "specs")`,
  a hardcoded directory name inside an e2e test harness rather than production code, but the
  same literal-vs-authority pattern.

Each of these hardcodes an artifact type's file-extension or directory-name literal
independently of `pkg/artifact.LayoutFor`, which is exactly the silent-break-under-a-
`.backstop/`-rooted-project (or any non-default artifact root) failure mode PLAN-SPEC-068's
guards seed exists to eliminate: a project that configures its artifact root or per-type
directory naming away from the historical default would drift out from under production
validation and gate logic silently, with the mismatch visible only as false-negative misses
(a spec/adr/bundle/issue file present on disk but never discovered/validated by the code
that's supposed to see it) rather than a loud failure.

**Explicitly NOT in this class:** `pkg/scaffold/scaffold.go`'s `FileExtension` entries are
deliberately sanctioned as scaffold's own local concern per PLAN-SPEC-068's TASK-019 — do not
lump those in with this issue's scope.

## Solution

For each hit above: confirm it is genuinely a project-root-relative artifact-type-extension
reference (not, e.g., matching a filename literal for an unrelated reason), then replace the
inline `.spec.md`/`.adr.md`/`.bundle.md`/`.issue.md`/`"specs"` literal with a call through
`pkg/artifact`'s shared layout authority (`LayoutFor`, or whatever accessor exposes a given
artifact type's extension/directory to a non-`pkg/artifact` package after PLAN-SPEC-068).
Where a caller only needs the extension (not the full directory), confirm the authority
exposes that granularity before doing a mechanical string swap — if it doesn't, that gap
belongs in this issue's fix too, since a partial authority invites exactly this drift back in.

## References

- `specs/SPEC-068-trustworthy-green-guards.spec.md` — originating context: the guards seed
  that established the one artifact-layout authority and closed the first six hardcoded
  copies of this table
- `plans/PLAN-SPEC-068-trustworthy-green-guards.plan.yml` — TASK-019, which explicitly
  sanctions `pkg/scaffold/scaffold.go`'s `FileExtension` as out of scope; verify current
  status before treating either artifact as closed
- `pkg/gate/step_testverify.go:120,225,520,564`
- `pkg/validate/spec.go:751`
- `pkg/validate/adr.go:159`
- `pkg/validate/bundle.go:275-278`
- `pkg/validate/supports_resolution.go:252`
- `cmd/backstop/gate_substantiveness_e2e.go:44`
- `pkg/validate/delivered_by.go` — the sixth hardcoded copy, fixed during PLAN-SPEC-068's
  implementation as an impl-reviewer finding; the sweep that found this issue's residual hits
  happened while fixing that file
