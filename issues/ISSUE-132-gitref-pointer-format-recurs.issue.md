---
title: "Gitref Pointer Format Recurs"
schema_version: issue/v1

issue:
  id: ISSUE-132
  title: "Gitref Pointer Format Recurs"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Gitref Pointer Format Recurs

## Problem

Commit `4906704` fixed a diagnostic-message bug in
`cmd/backstop/gate_substantiveness_provisioning_test.go`: a lock-entry `GitRef` check shaped
like

```go
if entry.GitRef == nil || *entry.GitRef != "v1.1.0" {
    t.Errorf("lock entry git_ref = %v, want v1.1.0", entry.GitRef)
}
```

formats `entry.GitRef` — a `*string` — with a bare `%v` in the `t.Errorf`. That's harmless on the
nil branch, but on the branch that actually fires for a real regression (the lock recording the
wrong tag), `%v` on a `*string` prints the pointer's memory address, e.g.
`lock entry git_ref = 0xc00047e8b0, want v1.1.0`, instead of the wrong ref string a developer
needs to diagnose the failure. The fix there split the check into a nil check and a dereferenced
`%q` comparison on the mismatch branch.

That commit's own message flagged three pre-existing siblings of the identical shape, filed here
as this issue. All three are confirmed present, unchanged, and matching the exact pattern (`*string`
compared and dereferenced in the `if`, then the same still-a-pointer variable re-used bare in the
`%v` format string on the mismatch branch):

1. `cmd/backstop/pack_remote_e2e_test.go:75-76`
   ```go
   if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
       t.Errorf("lock entry git_ref = %v, want v1.0.0", entry.GitRef)
   }
   ```
2. `cmd/backstop/init_seams_test.go:112-113`
   ```go
   if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
       t.Fatalf("lock entry git_ref = %v, want v1.0.0", entry.GitRef)
   }
   ```
3. `pkg/pack/distribution/add_test.go:1056-1057`
   ```go
   if entry.GitRef == nil || *entry.GitRef != "v1.0.0" {
       t.Errorf("GitRef = %v, want v1.0.0", entry.GitRef)
   }
   ```

In each case, on the mismatch branch the failure message reports a pointer address instead of the
lock's actual recorded git ref — the one piece of information a developer debugging a real
regression (e.g. a remote add recording the wrong tag) needs to see. This is a test-only
diagnostic-quality defect: the tests already correctly detect the failure, they just report it
uninformatively when they do.

## Solution

Same fix pattern as the already-landed `gate_substantiveness_provisioning_test.go` fix: split the
nil check from the value-mismatch branch, and dereference with `%q` on the mismatch branch. Apply
independently at each of the three sites above (surrounding variable names differ; no single
diff applies to all three).

`pkg/pack/distribution/lockfile_test.go:157-161` already does this correctly and is a good model
for the target shape:

```go
if entry.GitRef == nil {
    t.Fatal("expected non-nil GitRef for git pack")
}
if *entry.GitRef != "v1.2.3" {
    t.Errorf("GitRef = %q, want %q", *entry.GitRef, "v1.2.3")
}
```

No behavior change: these are `t.Errorf`/`t.Fatalf` message strings only, not assertion logic. No
new mandated test is required — this is a diagnostic-output-quality fix, not a coverage gap.

## Notes / references

- Filed at the request of the team lead following commit `4906704` (fix for the same pattern in
  `cmd/backstop/gate_substantiveness_provisioning_test.go`, found during phase-6a's impl-review),
  which named these three sites and reserved this issue number in its commit message.
- Confirmed independently by re-reading all three sites plus the correct-usage model in
  `lockfile_test.go` before filing; no existing issue or bundle covers this surface
  (`grep -rl GitRef issues/ bundles/` finds only `ISSUE-095`, an unrelated pack-add topic).
