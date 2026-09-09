---
title: "Public Site"
number: DIR-035
created: "2026-09-04"
schema_version: directive/v1

directive:
  status: active
  source:
    - "BUNDLE-032"
    - "SPEC-072"
    - "SPEC-073"
    - "SPEC-074"
    - "SPEC-075"
    - "SPEC-076"
    - "ISSUE-190"
    - "ISSUE-191"
    - "ISSUE-192"
    - "ISSUE-193"
    - "ISSUE-195"
    - "ISSUE-196"
    - "ISSUE-197"
    - "ISSUE-198"
    - "ISSUE-199"
    - "ISSUE-200"
---

## Description

Homes the public-site lane: BUNDLE-032 "Website Expansion" and the specs and
issues that delivered — and still owe — backstop.sh as a product surface.

The bulk of the bundle shipped on `main` in late August / early September 2026:
SPEC-072 "Public Product Model", SPEC-073 "Documentation Semantics Integration",
SPEC-074 "Derived Product Truth Pipeline", SPEC-075 "Static Public Site Design
System", and SPEC-076 "End-to-End Website Capabilities" are `implemented` with
`completed` plans. SPEC-071 "Website Expansion" and PLAN-SPEC-071 were
`canceled` in favor of that seed split. Visitor pages (Evaluate, Model, Adopt,
entity reference, Extend, artifact-lifecycle reference) landed in PR #38
(2026-09-03). The canonical homepage restore landed in PR #29 (2026-08-29).

This directive exists because that shipping left the **paperwork and the
residuals** without a live home. BUNDLE-032 stayed `maturity: ready` and was
cited by no `active`/`queued` directive — citation by shipped specs does not
count as a directive home. A 2026-09-04 backlog-pm sweep found the visitor-page
issue files (ISSUE-193, 195, 196, 197, 198, 199) reserved as git tags and
authored on the visitor-docs branch, then omitted from PR #38, which merged
the plans and the pages but not the issues. Those files are restored and
closed via `resolved-by` on `b77d07e`. ISSUE-190 is closed via `resolved-by`
on `cd453e8` (PR #29). ISSUE-194 is a burnt tag with no file and no plan —
it is DIR-034 ledger territory, not a member of this roster.

**Still open, and the reason this directive is `active` rather than `done`:**

- ISSUE-200 — SPEC-072 non-Go `provides`+signature contracts still red
  `contract_signature` on YAML and Mermaid; `ready`, draft plan exists.
- ISSUE-191 — `.cursor/` Cloud Agent environment files sit outside the closed
  Seed 4 sitecheck matrix; `open`, draft plan exists.
- ISSUE-192 — `scripts/websitejourney` production Go files unclassified by
  the architecture pack; `open`; pack-repo release, not a Core CLI change.

Do not mark BUNDLE-032 `delivered` or this directive `done` while ISSUE-200
can still fail the gate. Plan-status closeout (`PLAN-ISSUE-190` and
PLAN-ISSUE-193..199 still `draft` after the pages shipped) is remaining
paperwork, not remaining product scope — tracked in Notes, not as a reason
to keep those issues open.

## Notes

- Created 2026-09-04 from a founder-directed paperwork pass after the public
  site landed. Position in BACKLOG.yml is the session's "website first" call
  relative to DIR-002, whose remaining BUNDLE-003 REQ-022/REQ-024 work is
  blocked. Founder may reorder.
- When this directive was created, BUNDLE-032 was removed from BACKLOG.yml
  `bundles:` (a bundle leaves that section once a live directive cites it).
- Open residuals ISSUE-191, ISSUE-192, ISSUE-200 keep their existing draft
  plans. Next implementation in this lane is ISSUE-200 (live gate defect),
  then ISSUE-191 / ISSUE-192. Do not start those without a validated plan;
  ISSUE-200's plan is still `draft` and wants plan-review before implement.
- PLAN-ISSUE-190 and PLAN-ISSUE-193..199 remaining `draft` after merge is
  recorded, not silently flipped to `completed`. Completing those plans is a
  follow-on closeout (mandated-test honesty) and must not be a vacuous status
  bump.
- ISSUE-194 (`backstop/issue/194` tag, no file, no plan) is not restored.
  Homing burnt IDs is DIR-034 "Artifact Ledger Lifecycle Hardening".

## References

- `bundles/BUNDLE-032-website-expansion.bundle.md`
- `specs/SPEC-072-public-product-model.spec.md` through
  `specs/SPEC-076-end-to-end-website-capabilities.spec.md`
- PR #29, PR #33, PR #38
