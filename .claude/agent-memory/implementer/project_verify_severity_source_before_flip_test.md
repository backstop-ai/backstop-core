---
name: verify-severity-source-before-flip-test
description: A plan's "this site discards a declared severity" classification is a claim about the VIOLATION CONSTRUCTOR, not the call site — grep every Severity assignment reachable from the slice before writing a severity-flip test, or you will be handed a mandated test name no input can honestly satisfy; and stage the fix helper-first, because a flip-test that PASSES before its call site is converted has identified a misclassified site for free
metadata:
  type: project
---

PLAN-ISSUE-105 (2026-07-29, landed d7d777c) classified four raw-count verdict
sites as CLASS 1 ("counts a slice whose entries carry a severity resolved
upstream from a pack or a delegate"). Three were. `pkg/gate/step_contract.go` was
NOT: its violations have exactly ONE source, `VerifyContractVerdict`
(contract_verdict.go), whose three violation returns all HARDCODE
`Severity: "error"`, and `ContractEngineResult{Entry,Matched,Scanned,Locations}`
carries no severity field for a pack to populate at all. So it was structurally
warning-free — the plan's own CLASS 3 — and its mandated test
`TestStepContractSignature_DeclaredWarningDoesNotFailWithoutPolicy` names a flip
that NO input to that function can produce.

**THE STATIC CHECK, before writing any severity-flip test:**

    # every Severity the step's slice can carry, from the CONSTRUCTOR side
    grep -n "Severity:" <the file the violations are BUILT in>

All one hardcoded literal => the site is structurally single-severity and the
"declared warning" test is unwritable as a behavioral flip.

**THE DYNAMIC CHECK — A FREE CLASSIFICATION AUDIT, AND THE ONE THAT ACTUALLY
CAUGHT IT.** Stage the fix in two commits' worth of steps: add the shared helper
FIRST, convert NO call site, and run the whole flip-test set. In that window each
test reports its site's true class with no extra work:

    RED (with the defect's own message) => genuinely CLASS 1, severity flows there
    PASS                                => CLASS 3, the site cannot carry a warning

That is exactly what happened: the two delegate tests went behaviorally red while
the contract test passed, which a class-1 site cannot do. The staging costs
nothing — you need the helper before the call sites anyway — and it converts the
plan's classification from an assumption into a measurement. It also yields the
behavioral red that [[feedback_choose_compile_red_or_behavioral_red]] demands for
an EXISTING defect, since helper-only still leaves the defect live.

**Why:** a mandated test name is a promise the impl-reviewer matches exactly, so
discovering the misclassification AFTER writing the test leaves you choosing
between dropping a mandate and writing a name that lies. Both are bad; the
staged run tells you before you are committed.

**How to apply:** deliver the mandated NAME, but assert the invariant the
conversion actually establishes — `result.Status == StepVerdict(result.Violations)`,
false for a raw count the instant any severity reaches the slice — PLUS an explicit
assertion that the constructor's severity is still the hardcoded literal, so the
test SELF-REPORTS when the premise changes instead of rotting into a tautology.
State the misclassification out loud in the doc comment AND the boundary report,
and route the plan-text reconciliation to the planner (plans/* is not yours). The
conversion itself stays worth doing: behavior-preserving single-authority
maintenance, and the site's real deliverable was the coverage activation
(0.0% -> 100.0%). A site that cannot carry a pack severity is usually its own
filable defect one hop upstream — here the carrier type dropping it — which
became ISSUE-108. Related: [[project_redproof_by_worktree_flip]],
[[project_verdict_decided_after_the_step]],
[[project_lock_the_chain_falsify_per_hop]].
