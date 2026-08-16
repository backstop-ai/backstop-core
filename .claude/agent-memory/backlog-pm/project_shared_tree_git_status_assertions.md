---
name: shared-tree-git-status-assertions
description: Any test asserting on `git status` of the shared working tree is structurally unattributable now that concurrent implementer lanes run in ONE tree; RUN the test to confirm the red is live
metadata:
  type: project
---

A spec claim enforced via `git status --porcelain -- <path>` cannot distinguish
"this spec's implementation leaked" from "an unrelated concurrent lane is
mid-flight." As of 2026-08 this repo regularly runs **multiple implementer lanes
against one shared working tree** (overnight P0 batches), so the mechanism is
broken by construction, not occasionally unlucky.

First instance: ISSUE-139 (2026-08-16) — `pkg/initialize/sourceset_scan_test.go`,
SPEC-069 CLM-063, enforcing REQ-013 "init changes no file under `pkg/gate`."
Failed live, accusing PLAN-ISSUE-118 and PLAN-ISSUE-113 of init's violation.

The companion failure shape to watch for: a **non-vacuity guard placed after a
`t.Fatalf`**. Go `testing` never reaches it, so the escape hatch is dead code in
exactly the case it was written for — and the claim then has NO steady state in
which it verifies (dirty → false fail; clean → passes checking nothing).

**Why:** these denylist/purity claims are a real and growing pattern in specs
(SPEC-069 declares out-of-scope packages explicitly), so more of them will be
mandated this way.

**How to apply:** when triaging any issue about a test asserting over git state,
(1) RUN the test before ranking — the red is often live, not theoretical, which
changes urgency sharply; (2) home it by the SPEC that mandates the test, not by
the package the assertion *names* — ISSUE-139 names `pkg/gate` throughout but
belongs to DIR-002 because SPEC-069/BUNDLE-003 owns the claim. See
[[project_gate_verdict_honesty_cluster]] for the DIR-032 near-miss: its charter
is a GATE STEP lying about a verdict it computed, never a unit test.
