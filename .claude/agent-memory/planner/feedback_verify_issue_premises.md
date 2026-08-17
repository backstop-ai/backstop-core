---
name: verify-issue-premises
description: "Verify an issue's factual premises (especially 'no test asserts on this') against source before planning — they are claims, not facts"
metadata:
  type: feedback
---

**An issue's stated premises are claims to verify, not facts to inherit.** Before
planning a removal or a change an issue calls "inert," check the premise against
source yourself.

**Why:** ISSUE-082 (2026-08-15) asserted in its Acceptance Criteria that "no test
currently asserts on the five removed keys" and that the fix was confined to one
file. Both were false. Four tests read the map and asserted membership directly,
bypassing the code path the issue had analyzed — and three of the five keys were
load-bearing for MANDATED test names of spec claims, one on a spec with status
`implemented`. A code-only removal would have failed the suite AND tripped the
`artifact_status_drift` gate dimension as a broken promise. The issue was not
careless; it analyzed the runtime path correctly and simply never grepped for
direct-membership assertions. Founder approved widening scope to amend both specs
(SPEC-038 v1.2.2, SPEC-047 v1.2.1) before the cleanup could proceed.

**How to apply:** when an issue proposes deleting a symbol, map entry, or config
key, grep for every reader of it before scoping the plan — not just the runtime
call path the issue describes. Then check whether any test touching it is a
mandated test name of a spec, and what that spec's `status` is: `implemented`
means `status_drift` will treat a deleted mandated test as a broken promise, so
the spec must be amended first via spec-author. Deleting or weakening a mandated
test to make a plan executable is never the answer.

Corollary that also paid off on the same issue: an issue's own verification
command can be too narrow. ISSUE-082's sweep glob (`packs/*/pack.yml`) missed
every testdata manifest; sweeping wider confirmed the conclusion but the issue's
command could not have proven it.

**An issue's ROOT-CAUSE HYPOTHESIS is a separate claim from its SYMPTOM — verify
both, because the fix's correct owner depends on the cause, not the symptom.**
ISSUE-091 (verified 2026-08-16) reported `gate --all` under-reporting findings on
`*_test.go` and blamed semgrep's `paths.include` glob resolution differing between
directory-walk and explicit-file dispatch. The symptom reproduced exactly. The
cause did not: a rule with NO `paths:` block at all showed the same 0-vs-1 split,
which falsifies the glob theory in one command. The real cause was semgrep's
built-in default `.semgrepignore` (which excludes test files) being applied ONLY
during self-discovery — `semgrep --verbose` prints "Skipped by .semgrepignore"
and names the files. This changed ownership: a glob bug is the PACK's to fix,
an engine-default-ignore asymmetry is CORE's, because no pack can opt out of a
directory target core hands it.

Two things fell out of probing rather than reading, and neither was in the issue:
an equal-and-opposite OVER-report (`--all` reported `testdata/` findings the diff
scope suppresses), and the issue's named falsifier having gone inert since filing
(the cited rule had gained a `*_test.go` exclusion, so it now fires on neither
scope). **Re-derive a live falsifier before planning against a stale one** — and
when the probe surfaces a second divergence sharing the same cause, cure both or
you are knowingly leaving two code paths where one belongs.

**How to apply:** for any issue whose fix location depends on WHY, build the
smallest probe that separates the stated cause from rival causes before writing
tasks — for a tool-behavior claim that is usually one command with the tool's own
verbose/diagnostic flag, which tends to name the mechanism outright. Also grep for
shipped tests asserting the CURRENT (defective) behavior: ISSUE-091's fix had to
invert a subtest inside `TestPackEngines_AllScope_RestoresWholeRepoScan`, a name a
`completed` plan had promised, so the body was rewritten and the name preserved.

**When an issue names N failing tests, VERIFY EACH ONE'S FAILURE MESSAGE — they can
be red for different causes, and only some are yours.** ISSUE-146 (verified
2026-08-17) named two red tests as evidence of one defect. Running them
individually split them: `TestPackNew_ScaffoldPassesCheckAndTest` failed with
`[validator-negative] layer3 negative unexpectedly passed` — genuinely ISSUE-146 —
while `TestPackAuthoringLoop_EndToEnd` failed with `[validator-positive] layer3
positive failed`, a DIFFERENT check, because it invokes `pack test` with a relative
packDir and so trips ISSUE-147 (sandbox-exec refuses a relative profile subpath).
The difference was one word in the message. Proven by building the FIXED artifact
in a scratch tree and running it both ways: relative still failed, absolute passed
— so the second test provably could not go green from this lane, and the plan had
to name it as an inherited blocker instead of promising it.

**How to apply:** run each named test SEPARATELY and read its actual message
before writing an acceptance criterion around it; two tests failing "for the same
issue" is a hypothesis. Then, for any test you claim your fix turns green, build
the fixed artifact in a scratch tree and run the real command against it — that
counterfactual is what separates "my fix turns this green" from "this is red for
someone else's reason." When one stays red, fence it OUT explicitly and forbid the
tempting local workaround by name (here: passing an absolute path in the E2E),
because that workaround masks a filed defect and is the vacuous-green move the
directive exists to kill.

Related: [[code-check-command-removed]], [[defect-pinned-by-shipped-tests]],
[[dir032-stale-premises]]
