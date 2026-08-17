---
name: go-overlay-control-and-mutation-harness
description: "`go build/test -overlay=<json>` swaps a source file at compile time WITHOUT touching the shared tree — the safe control-vs-treatment and mutation-matrix harness for a repo with live sibling lanes"
metadata:
  type: project
---

Use `go build -overlay=file.json` / `go test -overlay=file.json` to compile a MUTATED or
HISTORICAL version of a file while the on-disk tree stays pristine.

```json
{"Replace":{"/abs/path/in/repo/pkg/x/y.go":"/abs/path/in/scratch/y_variant.go"}}
```

**Why:** this repo is a shared tree with many concurrent lanes, so `git stash`, in-place
edit-and-restore, and `git checkout` are all hazards ([[feedback_never_stash_shared_tree]]).
The overlay needs no tree mutation at all, so there is no window in which a sibling can
observe or collide with a probe. It also beats
[[project_control_vs_treatment_by_preserved_binary]] when the thing under test is a LIBRARY
rather than the CLI: you can build a control binary from `git show HEAD:path` AND run the
package's own tests against arbitrary mutants, from the same mechanism.

**Two proven uses (ISSUE-160, 2026-08-17):**
- CONTROL vs TREATMENT: `git show HEAD:pkg/packval/executor.go > scratch/head.go`, overlay-build
  a control binary, run BOTH binaries against ONE reproduction. Identical output = the red is
  inherited, proven rather than argued.
- MUTATION MATRIX ([[project_mutation_matrix_beats_sequence_red]]): generate one mutant file per
  wrong-variant, one overlay JSON each, run the suite against each and log the catcher.

**How to apply:**
- **Score "NOT CAUGHT" only after checking it compiled.** Deleting a use of a variable makes Go
  fail with `declared and not used` — that is a NO-OP MUTATION, not a weak test. Branch on
  `build failed` in the harness before reporting anything.
- **Source-SCANNING guards are invisible to the overlay.** A test that does
  `os.ReadFile("executor.go")` reads the pristine DISK file regardless. Falsify those with a
  real (briefly) on-disk edit plus an immediate `cp` restore, and `diff` to prove the restore
  was byte-identical.
