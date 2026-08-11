---
name: spec066-review2
description: SPEC-066 (BUNDLE-031 CI release auto-tag) re-review 2026-08-10 — 9 of 10 must-fixes landed; FAILED on M9's replacement justification being a second fabrication
metadata:
  type: project
---

SPEC-066 v2.0.0 (BUNDLE-031 Seeds 1+3, CI auto-tagger: `derive-release-delta.sh` +
`tag-from-release-delta.sh` in the external `backstop-ai/go-distribution` pack, plus
`.github/workflows/release-auto-tag.yml`). Re-review verdict **FAIL**, 2026-08-10.

Landed and verified against live code: M1 (delta-level corpus-only gate, CLM-023/024),
M2 (positive checkout `token:` mechanism), M3 (real pipe end-to-end, CLM-104-106),
M4 (positive workflow wiring, CLM-099-101), M5 (`head`↔sha, CLM-084), M6 (jq structural
parse + adversarial fixture, CLM-107), M7 (10-command baseline matches `backstop --help`),
M8 (go-distribution really is only tagged `v0.1.0` and its `scripts/` holds only
`falsify.sh` + `verify-recipe-apply.sh` — the ISSUE-111 subsumption reasoning is sound),
M10 (real cross-file `ci.yml` `name:` join).

**Why:** M9 was NOT resolved — the false noTarget claim was replaced by a second false
mechanical claim (see [[projectroot-is-always-absolute]]), and the "eleven implemented
specs" corpus-consistency figure is wrong (21 implemented specs). Residual: REQ-018 bans
text-scraping on the ACTING half while REQ-002 mandates POSIX-only tooling on the
DERIVATION half, which must scrape YAML frontmatter — no clause bounds those reads to the
frontmatter block and no claim exercises a body line shaped like `type: enhancement`.

**How to apply:** on the next pass, verify only those items; do not relitigate M1-M8/M10.
Validator PASSes, all 18 REQs have claims, 109 unique claims/tests, no surface overlap with
any other spec, integration/80 is the correct floor for `cmd/backstop`.
