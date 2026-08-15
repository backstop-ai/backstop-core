---
name: gate-verdict-honesty-cluster
description: The recurring gate-verdict-honesty issue family — RESOLVED 2026-08-10 into its own directive DIR-032 (THIRTEEN members as of 2026-08-15, +129 the suppression variant); slot new members into DIR-032 treating its CHARTER not its founder-enumerated roster as the boundary, restate the count, update the variant map, and re-read the directive after the agent returns
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

**HOME RESOLVED 2026-08-10 — the long-open founder question is answered.**
Brandon carved the cluster out of DIR-024 into its own directive,
**`DIR-032 "Gate Verdict Honesty"`** (`status: queued`), enumerating ELEVEN
members and moving them together — including 066/067/091, which until then had
no directive home at all. The old "slot into DIR-024, don't re-litigate"
guidance below is HISTORY; do not act on it. DIR-024 "Gate/Engine Quality"
still exists as the gate/engine catch-all and deliberately RETAINS four
near-members (ISSUE-096, 099, 107, 108) — folding any of those into DIR-032 is
a fresh founder call, never implied.

**How to apply:** slot a new member into DIR-032 under the standing grant.
DIR-024 is the plausible alternative every time (it is the catch-all covering
the same surface), so resolve it on **most-specific-wins** and say so in the
INBOX — that reasoning is worth one line, not silence. Critically: treat
DIR-032's **charter (the defect shape) as the membership boundary, NOT the
founder's eleven-member roster.** Precedent for that reading is ISSUE-115 →
DIR-024, which Brandon confirmed the same day ("standing grant applies, same as
112/113/114"). Flag it as an optional ack; don't stall the slot on it. Always
restate the running count, and preserve the founder's original eleven-member
enumeration as HISTORY in the directive — never rewrite it into "the founder
ruled twelve" (see [[project_corpus_note_supersedes]]).

**Adding a member causes CROSS-REF DRIFT inside the directive — re-read it
after directive-author returns, every time.** DIR-032 carries a "cluster's
variants" map bucketing items by failure mode plus a Notes cold-pickup
shortlist naming the members with measured consequences; both go stale the
moment an item lands and neither is in the agent's default scope. On ISSUE-118
I had to send a second pass for exactly this: the map still called item 11 "the
odd one structurally" after item 12 joined its variant, and the opening
defect-shape sentence enumerated only "an entire artifact kind" when item 12
starves on a *diff shape*. Brief the agent on the map and the shortlist up
front next time.

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

**Member 12 (slotted into DIR-032 2026-08-12, count now TWELVE)** — ISSUE-118:
`backstop gate` reports a full PASS while a mandated test genuinely fails,
because **when a diff is entirely `_test.go` files no dimension runs the Go
suite to a verdict.** Verified in-tree, reusable: `test_verification`
(`step_testverify.go`) only name-matches and execs nothing;
`test_substantiveness` reads the test BODY for assertions and never runs it;
`coverage_threshold` is the ONE dimension that invokes `go test` and it exits
early at `step_coverage.go:98` (`"no in-scope files to measure for coverage"`)
whenever no in-scope PRODUCTION file changed. Same starvation variant as
member 11 but far broader: 11 starves a warn-only advisory for one artifact
kind, 12 starves the gate's central blocking promise for a common diff shape.
**Two things worth carrying forward.** (1) It OVERLAPS member 1 (ISSUE-066) —
066 is the gate running a too-narrow suite, 118 is it running none, and 066's
stated fix subsumes 118 if "scope" includes test-only diffs; record that
coupling in any slot so two planners don't derive it separately. (2) The
practical dogfood consequence, which generalizes: **whenever in-flight work is
producing a test-only diff, a green `backstop gate` on it is not evidence its
tests pass** — check `git status` for the diff shape before repeating any
gate-green claim, and say so in the INBOX when it hits an active lane (it hit
DIR-019/SPEC-067 at BACKLOG position 1 the day it was filed).

**Member 13 (slotted 2026-08-15, count now THIRTEEN) is the THIRD variant** —
ISSUE-129: a diff-scoped gate PASS coexisting with a genuinely failing Go test
whose FILE is outside the diff. Neither mis-report (items 4-10) nor starvation
(11-12): the engine computes the failure CORRECTLY, converts it to a real
SARIF finding, and the finding is then **discarded post-hoc** —
**suppression**. Verified in-tree and reusable: `pack_gate.go:730` stamps
`ProjectWide` from `binding.ExemptFromScopeFilter`; `scope.go:302-326`
(`filterViolations`) keeps a violation only if `ProjectWide` OR in-scope; the
`go-toolchain` pack declares `exempt_from_scope_filter: true` on `go-build`
and **nothing** on `go-test`. `--all` is unaffected (short-circuits on
`GateScopeModeAll`) but CI's only BLOCKING job is diff-scoped, and the
`baseline` job is post-merge-only with no pass/fail exit contract — so this
merges to `main` unseen. **Generalizable triage rule this teaches: a
per-binding flag that is declared rather than derived from `gate_type` is a
silent-regression channel — when a defect turns on one engine's missing flag,
always ask which OTHER engines lack it.** Do NOT let it collapse into item 12
(ISSUE-118): 118 = suite never runs, test-only diffs; 129 = suite runs, fails,
finding thrown away, ANY diff shape.

**Ranking finding from that slot, still awaiting a founder ruling (raised as a
PROPOSAL 2026-08-15, not applied):** DIR-032 sits at BACKLOG position 5 behind
DIR-024, but **DIR-024's own Notes justify its rank "on the strength of
ISSUE-020" — closed and delivered 2026-07-28**, and every source it retains
post-carve-out is tier-2 by that file's own prose. DIR-032's lower rank is
carve-out mechanics (the new directive was appended after its parent), not a
founder priority call. Reusable both as the argument and as the method:
**a directive's position rationale lives in its own Notes and can go stale
silently when the issue that justified it closes — check the rationale, not
just the position.** Recommended 032→3, with a minimal 032↔024 swap offered
as the fallback; DIR-027's state was deliberately NOT re-grounded and that
limit was stated rather than papered over.

A *second*, smaller pattern is now forming with **no live home**: CLI
**arg-shape** defects where the accepted argument shape diverges from the
advertised one — ISSUE-093's `--file` repetition, ISSUE-089 (`artifact
validate` silently discarding a positional path), and the residual half of
ISSUE-074 (`pack relock` takes a path where its siblings take a name).
DIR-017 (Pack CLI Hardening) is `done`. Each is individually too small to
justify a directive; flag the accumulation, don't recommend one yet.

See [[project_orphaned_issue_backlog]], [[feedback_slot_vs_escalate]],
[[project_concurrent_pm_triage_races]], [[project_launch_tiering]].
