---
title: "TestPackAuthoringLoop_EndToEnd fails on darwin at phase3-fixtures/validator-positive"
schema_version: issue/v1

issue:
  id: ISSUE-162
  title: "TestPackAuthoringLoop_EndToEnd fails on darwin at phase3-fixtures/validator-positive"
  type: bug
  status: replaced
  created: "2026-08-17"
  closed: "2026-08-17"

replaced-by: ISSUE-147

complexity:
  scope: isolated
  uncertainty: known
  risk: moderate
---

# TestPackAuthoringLoop_EndToEnd fails on darwin at phase3-fixtures/validator-positive

## Problem

`TestPackAuthoringLoop_EndToEnd` (`cmd/backstop/pack_authoring_loop_test.go`) is currently red
on darwin. The test itself already documents that it only runs there
(`runtime.GOOS != "darwin"` → `t.Skip`, line 62-63), so this is **not currently a CI blocker**
(CI runs on `ubuntu-latest`) — but it is a real, currently-red, untracked local-dev-experience
defect: any darwin contributor running the full suite hits a failure with no filed issue to
point at.

Discovered and isolated 2026-08-17 during `ISSUE-158` verification, independently by the
impl-reviewer working that lane. Confirmed by direct re-run at repo HEAD (`722d5f1`):

```
=== RUN   TestPackAuthoringLoop_EndToEnd
    pack_authoring_loop_test.go:96: pack test failed (exit 1): status: fail
        - phase1-structural: pass
        - phase2-coherence: pass
        - phase3-fixtures: fail
        - phase4-archetype: skipped
          reason: phase3-fixtures failed
        - phase5-layer: skipped
          reason: phase3-fixtures failed
        - phase6-risk-class: skipped
          reason: phase3-fixtures failed
        ERROR [phase3-fixtures/validator-positive] layer3 positive failed
--- FAIL: TestPackAuthoringLoop_EndToEnd (1.72s)
```

The test dies at **step 3** (`pack test` against the freshly-scaffolded `loop-pack`), never
reaching `pack add` or the consuming gate run in steps 4-5. The failing check is
`phase3-fixtures/validator-positive`: the scaffolded engine pack's own **positive** fixture is
failing its own sandbox validator — the fixture that is supposed to trigger the rule is not
triggering it (or is triggering the wrong outcome) when run under `sandbox-exec` on darwin.

### Pre-existing, not introduced by ISSUE-158

Git-history bisection confirms this failure is present since `6923dba`
(`feat(ISSUE-032): pack-CLI authoring-loop reboot + ISSUE-030 scaffolder eradication`, the commit
that introduced this test) and is untouched by tonight's `ISSUE-158` work — `ISSUE-158`'s diff
touches only `cmd/backstop/gate_substantiveness_e2e.go` and its own test/plan-review files, none
of which this test's authoring-loop path exercises.

### Distinct from ISSUE-147 — stated explicitly to avoid conflation

`ISSUE-147` ("A relative packDir silently breaks every macOS sandboxed convert step (exit 71,
stderr discarded)") is a **different mechanism**: a relative `packDir` makes `sandbox-exec`
refuse to apply its profile at all, surfacing only an opaque `exit status 71` with the convert
step's stderr discarded. `ISSUE-146`'s own closing Resolution recorded that
`TestPackAuthoringLoop_EndToEnd` stayed red after its fix specifically because of `ISSUE-147`'s
exit-71 failure mode.

The failure reproduced here is **not** that. It fails at `phase3-fixtures/validator-positive` —
a specific, named fixture-level check that the validator ran and returned the wrong verdict for
— not an opaque `exit status 71` with no diagnostic content. Whatever changed between `ISSUE-146`
landing (which shipped the real marker-scanning validator this fixture now exercises) and this
measurement, the current failure signature is this one, not `ISSUE-147`'s. A repo-wide search
confirms no issue anywhere currently tracks a `validator-positive` fixture-level failure — this
is the first filing for it.

### Scope

Isolated to the darwin-only authoring-loop acceptance test and the scaffolded engine pack's
sandbox validator/fixture pair it exercises (`pkg/pack/scaffold.go`, the marker-scanning
validator and fixtures `ISSUE-146` introduced). Does not affect CI (Linux) and does not affect
`pack new`/`pack check`/`pack test` run manually outside this specific E2E's temp-project
harness — root-causing why the sandboxed validator disagrees with the fixture's expected
positive/negative polarity in this harness's invocation shape is this issue's own
plan/implementation lane, not pre-decided here.

## References

- `cmd/backstop/pack_authoring_loop_test.go:58` — `TestPackAuthoringLoop_EndToEnd`, the failing
  acceptance test (darwin-only via `t.Skip` at line 62-63).
- `ISSUE-146` — "backstop pack new ships a sample validator that always exits 0" — closed; its
  fix is what made the scaffolded validator/fixture pair genuinely discriminating, and its own
  Resolution first recorded this test staying red on darwin (there, attributed to `ISSUE-147`).
- `ISSUE-147` — "A relative packDir silently breaks every macOS sandboxed convert step (exit 71,
  stderr discarded)" — open; **adjacent but not the same defect**. That mechanism is an opaque
  `exit status 71` from a relative `packDir` at the sandboxed convert step; this issue's failure
  is a named `phase3-fixtures/validator-positive` fixture-level mismatch. Do not merge or route
  one's fix through the other.
- `ISSUE-158` — the lane whose verification surfaced this residual (not itself at fault; its own
  diff does not touch this test or the scaffolded-pack code path).
- `pkg/pack/scaffold.go` — the scaffolded engine pack's sample validator/fixture pair
  (`ISSUE-146`'s marker-scanning validator).

## Resolution

**Retracted as a duplicate: this is ISSUE-147, not a distinct defect.** The "Distinct from
ISSUE-147" section above was wrong. It reasoned from the *visible message*
(`phase3-fixtures/validator-positive` vs. `exit status 71`) rather than from the *mechanism*,
and the two seams genuinely produce different-looking output for the same root cause — which is
exactly what made the wrong conclusion so plausible at filing time.

**How this was caught and settled (2026-08-17):**

1. The repo's own backlog-pm auto-triage hook flagged the filing and wrote
   `.claude/agent-memory/backlog-pm/project_relative_packdir_masquerades.md`, tracing the
   mechanism in tree: `pack test` with no argument (exactly what
   `TestPackAuthoringLoop_EndToEnd` step 3 does via `cmd.Dir = packDir` + no path arg) defaults
   `packDir` to `"."` (`cmd/backstop/pack_test_cmd.go:29`); `darwinSandboxProfile` embeds that
   relative path verbatim into the `sandbox-exec` profile; the profile still *applies* (no
   exit-71) but its subpath clause silently matches nothing, so the sandboxed validator gets
   `Operation not permitted` reading its own pack. `RunValidator`
   (`pkg/packval/executor.go:261-267`) then collapses that run failure into `Passed:false` with a
   **nil error**, and `phase3.go:141` fires the same generic `layer3 positive failed` message
   whether the validator ran and returned the wrong verdict or never usefully ran at all — one
   message covers both cases, which is why the validator seam can't self-report "this is a
   sandbox access failure" the way the convert seam's `exit status 71` at least hints at one.
2. Independently, the reporter (not the triage hook) reproduced this directly: built the exact
   scaffolded pack `TestPackAuthoringLoop_EndToEnd` creates, ran `pack test` from inside it with
   a relative/implicit packDir (cwd = packDir, no path argument — matching the test's
   `runBackstop` helper exactly) and got the identical `validator-positive` failure; then ran
   `pack test <same pack, absolute path>` from a different cwd and got a clean pass, all six
   phases. Same pack, same bytes — only the packDir's relative-vs-absolute-ness changed. That is
   ISSUE-147's mechanism by direct, controlled reproduction, not inference.

**Why the two lines of evidence corroborate rather than merely agree:** the memory file
establishes the mechanism from reading the code path; the direct relative-vs-absolute rerun
establishes it empirically, independently, against the real CLI. Both land on the same cause.

**Disposition:** `ISSUE-147` ("A relative packDir silently breaks every macOS sandboxed convert
step") is the owning issue — retitled in spirit, if not in name, to also cover the validator
seam it did not originally describe. `ISSUE-147` has been amended with a new "Validator-seam
manifestation" section documenting this test as a concrete instance and citing this issue's
retraction. No fix work happens under `ISSUE-162`; do not re-file this failure signature again —
see the backlog-pm memory file for the standing triage rule
(`.claude/agent-memory/backlog-pm/project_relative_packdir_masquerades.md`): any darwin
`phase3-fixtures/validator-positive` failure is ISSUE-147's mechanism until an absolute-packDir
re-run says otherwise.
