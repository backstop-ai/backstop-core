---
name: new-guard-predicate-measure-existing-fixtures
description: A plan adding a short-circuit/refusal predicate to a gate step must be MEASURED against every existing e2e fixture's real finding counts — reasoned precision arguments miss legitimate inputs that trip the guard
metadata:
  type: project
---

When a plan adds a new guard/refusal/short-circuit predicate in front of an existing
gate step loop, RUN the real engine over every EXISTING fixture that drives that step
and compute the predicate's inputs. Do not accept the plan's precision argument.

**Why:** PLAN-ISSUE-113 (2026-08-16) proposed refusing when `eligible >= 1 && extraction == 0`,
arguing it "fires ONLY when the step is about to emit a violation it cannot justify."
Running real ast-grep over the pre-existing `newE2EWorkspace` fixture
(`func TestE2EHollowSubject(t *testing.T) { doSubject() }`) measured hollow=1,
extraction=0, eligible=1 — the guard fires and DISCARDS a true hollow finding, reddening
two SPEC-037-mandated tests. Generalized: a diff-scoped run over one changed hollow test
file would exit 2 blaming pack configuration. The plan's own non-regression claim asserted
the opposite. No amount of reading the plan surfaced this; only measuring did.

**How to apply:** For any plan adding a step-level guard, (1) list every existing fixture
workspace feeding that step, (2) run the real engine over each and record the actual
partition counts, (3) evaluate the proposed predicate on those counts. Any fixture where it
fires is either a false-positive design flaw or an unscoped test-update task — both blockers.
Also check whether the newly-added fixture deliberately avoids the risky combination
(here: hollow==0 by construction), which means the dangerous case is untested.

Related: [[project_deletion_file_premise_audit]], [[project_captured_fixture_source_must_exist]],
[[project_coverage_rewrite_predating_spec_drift]].
