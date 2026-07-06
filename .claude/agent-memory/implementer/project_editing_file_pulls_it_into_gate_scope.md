---
name: editing-file-pulls-it-into-gate-scope
description: editing a previously-unmodified tracked .go file adds it to the diff scope, so latent per-file gate landmines (contract-absence prose grep, unmeasured-no-record) fire on it for the first time
metadata:
  type: project
---

`backstop gate` diff scope = `git diff HEAD` + untracked. A tracked file that was
unmodified is OUT of scope, so any per-file dimension entry keyed to it is filtered
by `scope.Contains`. The instant you edit it, it enters scope and latent per-file
checks fire for the FIRST time — even if the trigger predates your edit.

Two landmines seen while doing ISSUE-034 (coverage deleted-file fix in
pkg/gate/step_coverage.go):

1. **Contract-absence prose grep.** The absence probe (Absent=true contracts,
   pkg/gate/contract_verdict.go) is a plain TEXT grep for the symbol name, NOT
   ast-grep. A doc COMMENT mentioning a deleted symbol by its bare identifier token
   (`coverageMeasurablePath is DELETED`) matches -> "expected absent but present
   (forbidden symbol regression)". The symbol was genuinely gone from code; only the
   prose tripped it. Fix = reword the comment to drop the bare identifier token
   (in-scope, weakens nothing — a real `func` reintroduction still contains the token
   and reds). This is the documented prose-gotcha (impl-reviewer
   [[contract-signature-prose-gotcha]]).

2. **Coverage bun-fixture faithfulness.** The ISSUE-034 fix adds an os.Stat existence
   guard in coveragePathsInScope: a not-on-disk in-scope path is treated as deleted
   and excluded. cmd/backstop/bun_fixture_e2e_test.go's seeded-uncovered test set
   ProjectRoot=tmp + scope src/app.ts but never wrote src/app.ts to disk, so under the
   guard it looked deleted and the anti-vacuous-green RED stopped firing. Fix = the
   fixture must materialize the changed source on disk (a genuinely-changed file EXISTS
   in the working tree; only a deleted one doesn't). On-disk existence is the exact
   discriminator, so a faithful "changed but unmeasured" fixture MUST create the file.

**Why:** both read as a regression but are latent conditions your edit merely
surfaced by scope-entry — not defects your change logically introduced.
**How to apply:** when a per-file dimension flips pass->fail after you edit a file,
check whether the file just ENTERED diff scope (was it modified before you touched
it?). If the finding is a text-probe FP on a genuine deletion, or a test fixture that
was never faithful to on-disk reality, fix at the source (reword/materialize) rather
than reverting your change. See [[netnegative_gate_baseline]].
