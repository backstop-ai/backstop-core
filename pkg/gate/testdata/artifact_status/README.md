# artifact_status resolver fixtures

A tiny cross-type artifact tree used by artifact_status_test.go to exercise the
exported ResolveArtifactStatus resolver + ClassifyArtifactStatus classifier
(ISSUE-042 CLM-001/002/003/013/014). These are FIXTURES, not real artifacts:
they live under testdata (never under the repo's real issues/ specs/ plans/
directives/), so the live `backstop gate` full-sweep never scans them and
`backstop artifact validate` never validates them. Frontmatter is intentionally
minimal — the resolver reads only status + claims[].tests + plan spec_id.

Contents:
- issues/ISSUE-900 (open, delivered) mandates TestFixturePresentAlpha
- issues/ISSUE-901 (closed, success-terminal) mandates TestFixtureAbsentBeta
- issues/ISSUE-902 (open, NO backing plan) — issue->plan empty-linkage case
- issues/ISSUE-903 (replaced, retired-terminal)
- specs/SPEC-900 (implemented, success-terminal) mandates TestFixtureSpecGamma
- specs/SPEC-901 (draft, non-terminal) mandates TestFixtureSpecDelta
- specs/SPEC-902 (replaced, retired-terminal) mandates TestFixtureSpecEpsilon
- plans/PLAN-ISSUE-900 (completed) spec_id ISSUE-900 — the backing plan for ISSUE-900
- directives/DIR-900 (done, success-terminal)
