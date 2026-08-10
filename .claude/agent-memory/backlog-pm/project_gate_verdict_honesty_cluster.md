---
name: gate-verdict-honesty-cluster
description: The recurring gate-verdict-honesty issue family (066/067/091/092/093/097/100/106/112/113/114 + the 104-108 severity-contract burst — ELEVEN as of 2026-08-02) — only 066/067/091 and 104/105 still uncited; slot new members into DIR-024 and keep the count current rather than re-litigating the new-directive question
metadata:
  type: project
---

A single failure family keeps arriving one issue at a time: **a gate/verify
surface that reads authoritative and silently isn't.** Seven members as of
2026-07-28: ISSUE-066 (test_verification runs the full package, not a plan's
narrow `-run` filter), ISSUE-067 (go-test engine reports real test failures as
an opaque crash), ISSUE-091 (`gate --all` underreports test-file findings),
ISSUE-092 (`pack test` phase3 fixtures are dead code — a pack whose rule never
fires ships green), ISSUE-093 (`gate --file` false-REDs non-Go files whose
directory holds no Go package, + `--file` silently collapsing repeated flags),
ISSUE-097 (waiver tokens on a dead pack namespace fail open, and harvest is
finding-driven so they are structurally invisible), ISSUE-100 (step tallies
count warnings as violations — **renderer half only**; its sibling verdict half,
the policy layer blocking on a `severity: warning`, was FIXED 2026-07-28 at
`pkg/gate/policy.go:73` under PLAN-ISSUE-020 and must not be re-opened).

**Why:** the cluster's home has been an open founder question since
2026-07-27 and my own recommendation has moved twice as members closed and new
ones landed (4 → 2 → 4 → 5 in about 30 hours). 066/067/091 are still cited by
no directive; 092 and 093 I slotted into DIR-024 "Gate/Engine Quality" under
the standing grant, because DIR-024's charter is explicitly the catch-all for
gate/engine-quality gaps and no second directive has a plausible claim.

**How to apply:** when a new member arrives, slot it into DIR-024 as a clear
fit and say plainly in the INBOX that this does not pre-empt the pending
cluster decision (if a cluster directive is created, all DIR-024-slotted
members move together). Do NOT re-open the new-directive recommendation on my
own initiative — append-to-DIR-024 works and is cheaper. DO restate the
current member count and which members are still uncited, because Brandon's
open ask was framed at four members and the shape keeps changing under it.
Note DIR-027 is taken, so a standalone would need DIR-028.

**The severity-contract sub-family (ISSUE-104 → 108, filed 2026-07-29)** — five
hops on ONE contract, *a pack-declared severity must survive to the verdict*:
104 SARIF parser drops it (FIXED `a42b065`), 105 step verdicts ignore it absent
a policy entry (FIXED `d7d777c`, shipping `gate.StepVerdict` at
`pkg/gate/policy.go:125` as THE single severity predicate), 106 the
substantiveness join discards it, 107 the coverage step reads a warning-only
finding set as `pass` (the INVERTED direction — loud going silent), 108 the
contract carrier cannot represent it. `PLAN-ISSUE-105`'s TASK-006 filed
106/107/108 as its own residuals (commit `21e47ed`); **all three are now slotted
into DIR-024** (three concurrent PM runs, Description items 12/13/14). Still
needing Brandon: **104 and 105 are cited by NO directive and remain
`status: open` though their fixes shipped**, and `PLAN-ISSUE-104` is still
`draft`. ISSUE-108's kind is distinct and worth preserving: elsewhere the value
exists upstream and is mishandled; here the TYPE cannot represent it
(`ContractEngineResult` has no `Severity` member; `VerifyContractVerdict`
hardcodes `"error"` at `contract_verdict.go:77,85,101`), so `contract_signature`
is warning-free by construction. Its fix is coupled to a **self-reporting
premise guard** — `TestStepContractSignature_DeclaredWarningDoesNotFailWithout`
`Policy` (`step_verdict_severity_test.go:163`) asserts today's construction and
flips false the moment the field lands; revise it deliberately, never "fix" it
when it goes red. Also: **DIR-028 now has two claimants** — this cluster and
ISSUE-101's go-distribution pack.

**The recurring design question inside that sub-family**, worth naming in any
future member's slot: *what severity does a SYNTHESIZED, non-1:1 violation
carry?* Sites that convert one finding 1:1 just forward `v.Severity` (ISSUE-106's
`HollowFindingsToViolations`, `substantiveness_join.go:184` — a direct
substitution). Sites that SYNTHESIZE have nothing to forward and need a ruling,
not a patch: `NoTargetViolation` (`:68`) fires on set-membership over a
presence-only `map[string]bool`; 108's carrier has no field at all. Either the
rule gains a declaration channel, or synthesized violations stay fixed-severity
by design as gate-computed defects. Answering it once, generally, is plausibly
cheaper than three site-local substitutions — a founder/planner call; record it,
don't decide it.

**How to apply to this sub-family:** these are plan-filed residuals, so
coverage is provably nil from the artifacts — read the parent plan's AS-BUILT
+ CLASS-2 sections instead of interviewing. And check the **renderer coupling**
before recommending sequence: 107 emits a new `"warning"` status into the same
renderer ISSUE-100 says miscounts warnings, so 107 must be planned with or
after 100. 106 and 108 are severity-plumbing (a value is lost upstream of the
verdict); 107 is verdict-only. Don't let "same family" collapse that difference
in a plan recommendation.

**Members 9 and 10 (slotted 2026-08-02, count now TEN)** — and they broke the
family's provenance pattern: ISSUE-112 (a findings engine whose tool is absent
from PATH passes vacuously — empty stdout, jq emits nothing, lenient SARIF
parse reads zero findings, `pack_engines` green) and ISSUE-113 (a
classification matching zero test files emits per-test violations instead of
refusing). Every prior member was a DOGFOOD discovery; these two were found by
the **first external consumer** (bclabs-portal's first GitHub-runner CI run,
397 false violations, hours to diagnose). Both were filed 2026-07-29 and sat
**untriaged and uncited for four days** — the pm-trigger hook never logged
them. So: on a sweep, never assume `pending.log` is complete; diff
`issues/*.md` against the log.

112's load-bearing finding, worth reusing: `cmd/backstop/pack_gate_provision.go`
exempts provision-declared tools from the assume-present fail-loud as
"auto-provisioned", but provision is a **trust-allowlist pin only — no code
path installs anything**, so ast-grep/semgrep get neither install nor presence
check. `pkg/packval`'s executor already fails loud on exec-not-found while
gate dispatch does not — an in-tree precedent, not a new mechanism. Keep the
layer distinction vs ISSUE-092 in any slot: 092 is pack-AUTHORING false-green,
112 is CONSUMER false-green; neither closes the other.

**Member 11 (slotted 2026-08-02, count now ELEVEN) is a NEW VARIANT worth
naming** — ISSUE-114: the `artifact_status_drift_advisory` cannot fire for a
plan, ever. The other ten mis-report a verdict they DID compute; this one never
computes anything for an entire artifact KIND — **silent by starvation**.
Mechanism: `looksDelivered` (`status_drift.go:75-85`) bails on
`len(MandatedTests)==0`, and plans' only mandated-test source is the OPTIONAL
per-task `test_names` key, which `pkg/validate/` never reads (grep: zero hits).
Measured 2026-08-02: 98 plans, 48 non-terminal, **zero** with a populated
`test_names`; the 28 files containing the string are 25 completed / 1 obsoleted
/ 1 replaced / 1 draft (the draft is PLAN-ISSUE-048, prose-only). **The reusable
triage lesson: an OPTIONAL schema field that no validator reads is an
empirically-dead data channel — check whether a check's input is ever populated
before accepting "the mechanism is wired correctly."** Home reasoning, so it
need not be re-derived: DIR-021 rejected (requirement-traceability substrate +
corpus drain; its ISSUE-048 thread is per-artifact reconciliation, not
mechanism), DIR-016 is `done`, and sibling ISSUE-098 is cited by no directive.
Provenance is self-referential — it is the mechanism behind my own 15:20Z
delivered-but-open flag on PLAN-ISSUE-048.

A *second*, smaller pattern is now forming with **no live home**: CLI
**arg-shape** defects where the accepted argument shape diverges from the
advertised one — ISSUE-093's `--file` repetition, ISSUE-089 (`artifact
validate` silently discarding a positional path), and the residual half of
ISSUE-074 (`pack relock` takes a path where its siblings take a name).
DIR-017 (Pack CLI Hardening) is `done`. Each is individually too small to
justify a directive; flag the accumulation, don't recommend one yet.

See [[project_orphaned_issue_backlog]], [[feedback_slot_vs_escalate]],
[[project_concurrent_pm_triage_races]], [[project_launch_tiering]].
