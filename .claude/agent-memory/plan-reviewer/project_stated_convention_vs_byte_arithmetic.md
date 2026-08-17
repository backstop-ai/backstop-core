---
name: stated-convention-vs-byte-arithmetic
description: A plan's stated measurement convention (absolute vs repo-relative paths, pruned vs unpruned) can be falsified by dividing the quoted byte total by the quoted file count — do the division, planners mislabel which knob moved the number
metadata:
  type: project
---

When a plan quotes a size/headroom figure as `<N files> / <B bytes>` AND names the
convention that produced it, DIVIDE `B/N` and sanity-check the average against the
repo. Planners routinely attribute a delta to the wrong knob.

**Why:** PLAN-ISSUE-091 (2026-08-16) reconciled two argv-headroom readings by saying
the older ~12x figure came from an "absolute-path, unpruned count (1,615 files /
82,362 bytes)". 82,362/1,615 = 51 bytes/path, but `/Users/bmanson/src/projects/
backstop-core` is 41 chars — an absolute path here averages ~90 bytes and the real
absolute total is ~150 KB (~7x headroom), reconciling to NEITHER quoted reading. The
figure was repo-RELATIVE and UNPRUNED; the entire ~12x→~21x delta was testdata
pruning. The paragraph existed solely to make two readings reconcilable and, as
written, made them irreconcilable.

**How to apply:** for any plan quoting an argv/scope size, (1) do the `B/N` division;
(2) recompute yourself under the ship-path convention — for backstop that is
`resolveGateScopeAll`'s `filepath.Rel(projectRoot, path)` walk, which skips dot-dirs,
does NOT honor `.gitignore`, and yields every file (not just source), then
`excludeTestdataPaths` segment-prunes; (3) measure in a CLEAN worktree at HEAD, since
untracked artifacts from concurrent agents shift the count by tens of files. Clean
HEAD 17fac05 reference: 1,634 files / 79,494 bytes relative-unpruned; 1,152 /
48,765 relative-pruned (21.5x under ARG_MAX 1,048,576).

Related: [[project_scan_boundary_count_mismatch]],
[[project_sarif_suppressions_measurement_layer]],
[[project_verified_enumeration_do_not_rederive]].
