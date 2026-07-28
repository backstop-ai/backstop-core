---
name: feedback-never-stash-shared-tree
description: Never git stash in this repo's shared working tree; prove a red is inherited by cited-file comparison, per-file gate scoping, or a control-vs-treatment test run from git archive HEAD
metadata:
  type: feedback
---

Never run `git stash` (even path-scoped, e.g. `git stash push -- cmd/backstop`) to
answer "is this red pre-existing?". The working tree is SHARED with concurrently
running sibling agents whose uncommitted work would be swept up with yours.

**Why:** during SPEC-055 phase 3 a path-scoped `git stash push -- cmd/backstop`
silently removed the four `Explained` edits mid-verification; the pop recovered them,
but the repo also had four other agents' untracked/modified files live at that moment
and the stash list already carried two entries labelled "RECOVERED: ... wrongly popped
during phase-4 impl" — this failure mode has cost this repo work before.

**How to apply:** to decide whether a gate red is inherited, compare the violation's
cited FILE against `git status --short` / `git diff --stat` for your own scope — a red
naming a file you never touched is inherited, full stop. If you truly need HEAD's
content, use `git show HEAD:<path> > /tmp/copy` and inspect the copy.

**For TEST reds specifically, the stronger form is a CONTROL-VS-TREATMENT run** — the
team lead endorsed this during ISSUE-081 phase 3 as "the standard going forward."
Build two scratchpad trees: control = `git archive HEAD | tar -x -C ctl` (pristine
HEAD), treatment = a copy of control with ONLY your own changed files copied over from
the working tree. Confirm the isolation with `diff -rq ctl trt` (it must list exactly
your files), run the same `go test ... -race -count=1` in both, then set-diff the
failure NAMES. What this buys you that cited-file comparison cannot: it proves the
NEGATIVE ("newly broken: NONE") rather than only explaining away each red one at a time.

Two mechanics that will bite:
- **Strip the durations before comparing.** `--- FAIL: TestX (0.01s)` vs `(0.02s)`
  makes `comm` report the SAME test as both fixed AND newly-broken. Pipe through
  `sed 's/ (.*//' | sort -u` first.
- **`git archive` omits gitignored content**, so the control tree has no
  `.backstop/packs/` — every dogfood/go-standards/go-toolchain/ratchet test fails in
  BOTH trees as an artifact of the method, not a real red. Compare failure SETS, never
  absolute counts, and expect that inherited block.

Concretely (ISSUE-081 phase 3): the shared tree showed 15 `cmd/backstop` failures, 14
of them ONE sibling's mid-edit `pkg/pack` compile break cascading into every test that
SUBPROCESS-BUILDS the binary. A sibling mid-edit makes the shared tree useless as
evidence for everyone in it; the isolated run still showed 9 fixed / 0 regressions.
Flag such a break to the lead — a full `gate` run is uninterpretable until it clears.

**When a sibling package is MID-REFACTOR** — red in OPPOSITE directions across
snapshots minutes apart (e.g. a call-site/signature arity split that flips side) —
neither cited-file comparison nor control-vs-treatment helps: the package won't
compile either way. Use a DETACHED WORKTREE AT HEAD: `git worktree add --detach
<scratch> HEAD`, copy YOUR new file in, delete any file the sibling committed as a
deliberate TDD red, run the package there, then `git worktree remove --force` +
`git worktree prune`. Never touches the shared tree, unlike stash. Also the right
place to run FALSIFICATION MUTATIONS against production code you do not own
(SPEC-056 phase 2: neutering phase3.go's render loop to prove CLM-080 can red, zero
live-tree risk). Evidence: SPEC-056 phase 2, RunArchetype call sites vs signature
disagreed first one way then the other while validateScaffold's arity split broke
pkg/pack.

Then make the positive case too: run `./bin/backstop gate --file <path>` once per file
YOU touched and show every code dimension green. **`--file` is a single string, not a
repeatable slice** — passing it N times silently keeps only the LAST one ("Gate running
against 1 explicit files"), so loop over the files instead of chaining flags. The
artifact-level dimensions (requirement_traceability, artifact_status_drift) are
whole-repo regardless of `--file`, so they stay red under a scoped run; that itself is
evidence they are not yours. Related: [[project_pack_copies_and_stale_gate_binary]],
[[project_smoke_darkpack_prefailures]].

## Standing RULED-inherited red: SPEC-015's REQ-020/021@1.0.0 pins

An unfiltered `./bin/backstop gate` reports **requirement_traceability FAIL with 3
violations** citing `SPEC-015-pack-distribution` / `PLAN-SPEC-015` pinning
`pack-distribution-lifecycle:REQ-020@1.0.0` and `REQ-021@1.0.0`. Do NOT chase it,
and above all do NOT "fix" it by bumping the pins.

The pins are historically correct BY BUNDLE MANDATE. BUNDLE-006's "Revision Impact
on Existing Artifacts" section says in terms that SPEC-015 "remains historically
pinned to `pack-distribution-lifecycle:REQ-021@1.0.0`. That pin must not be
rewritten: it describes the algorithm the spec evaluated." The prescribed remedy is
NEW DELTA SPECS pinning REQ-020@1.1.0 / REQ-021@1.1.0 (plus 1.1.0 revisions of
REQ-038..042), and those delta specs DO NOT EXIST YET — so the red is structurally
unavoidable until they are written (capability gap being filed as an issue, founder
decision pending). Rewriting the pin to green the gate would destroy the historical
record the bundle explicitly protects.

So when a lane's definition of done includes a full gate run, the bar is "every
OTHER dimension green, and any red attributed to exactly this known set" — not a
green requirement_traceability. State the attribution explicitly in your report
rather than silently passing over it. VERIFY BEFORE CITING: the grounding text is
the "Revision Impact on Existing Artifacts" section of
`bundles/BUNDLE-006-pack-distribution-lifecycle.bundle.md` (~line 1435). This goes
STALE the moment those delta specs land, at which point the red becomes real again.
