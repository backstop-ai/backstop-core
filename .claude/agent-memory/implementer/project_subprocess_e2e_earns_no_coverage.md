---
name: subprocess-e2e-earns-no-coverage
description: Statements reachable only through the BUILT binary earn zero coverage credit, so adding wiring lines to a thin cmd/backstop file can red coverage_threshold even when the new path is fully e2e-tested
metadata:
  type: project
---

An e2e test that runs the built binary (runBackstop / runBackstopStreams) contributes
NOTHING to the Go coverage profile — the instrumented statements execute in a child
process. So adding a few wiring statements to a thin `cmd/backstop/pack_*.go` command
file can push it under the per-file 80% floor even though the new branch is proven
end-to-end. Measured on SPEC-055 phase 9: inlining a 3-statement `--json` block into
each of the four pack lifecycle commands took `pack_update.go` from 81.2% to 73.7%.

**Why:** the coverage producer runs `go test` in-process; the gate's per-file floor is
blind to subprocess proof. Thin cobra files have so few statements that 3 uncovered
ones move the ratio several points.

**How to apply:** put shared wiring logic in a helper in its OWN file and unit-test the
helper in-process (helper file hits 100%, each command file keeps its statement count
unchanged). Before/after check: `go test ./cmd/backstop/ -coverprofile=… && go tool
cover -func=…` against a `git worktree add … HEAD --detach` copy proves whether a
coverage red is yours or inherited — see [[feedback_never_stash_shared_tree]] and
[[project_editing_file_pulls_it_into_gate_scope]]. `cmd/backstop/pack_upgrade.go` is
capped at 50% at HEAD (see [[project_upgrade_coverage_cap]]): touching it at all pulls
that pre-existing red into blocking diff scope, so keep such an edit statement-neutral.
