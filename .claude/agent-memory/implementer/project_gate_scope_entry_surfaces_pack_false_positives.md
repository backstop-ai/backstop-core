---
name: gate-scope-entry-surfaces-pack-false-positives
description: "A plan that mandates touching a test file for a MECHANICAL signature repair drags that file's whole pre-existing pack finding set into blocking diff scope — budget for it, and expect some findings to be genuine pack false positives needing a pack release, not code changes"
metadata:
  type: project
---

Measured on PLAN-SPEC-068 (2026-08-14): widening `buildGateSteps` forced a one-argument repair in 13
test files, and that alone surfaced findings that had been sitting dormant for months — 4
`backstop-self/no-baked-tool-command` hits, 3 `test_substantiveness` subject-join hits, 4 errcheck
hits, 3 staticcheck `unused` helpers, and a `staticcheck S1039` that was even present in
`.backstop/baseline.json` and STILL blocked.

**Why:** the diff-scoped gate blocks on a touched file's ENTIRE finding set, and the local
baseline does not suppress pack_engines findings. Meanwhile `gate --all` is NOT a superset — it
reported a different (larger, repo-wide) set and is the wrong ratchet.

**How to apply:**
- Derive the blocking set from `backstop gate --json` (diff-scoped), never from `--all`.
- Separate the three classes before fixing: (a) genuinely mine, (b) pre-existing-but-real (fix it —
  no grandfather on changed files), (c) pack FALSE POSITIVE.
- For (c) the fix is a PACK precision edit + version bump + tag + push + `pack update`, which is a
  cross-repo publish and needs the founder's go. Two arose here: `no-baked-tool-command` firing on
  `config.ToolchainPass{Command: "go test ./..."}` test fixtures (family B1 was the last family
  without the `*_test.go` exclusion B2/B3-B6 already carry), and `constructor-injection` firing on a
  mock literal because the field is named `isRepo` and the rule's regex matches `repo`.
- Prove a pack edit is purely subtractive before proposing it: run semgrep with the OLD and NEW rule
  file over the target file (expect N -> 0) AND over the whole non-test tree (expect identical
  counts). `pack check` / `pack test` passing is necessary but not sufficient.
- `constructor-injection` can be dodged in test scaffolding without a pack change by assigning
  fields after an empty literal instead of composing a keyed one.

See also [[substantiveness-subject-join-needs-a-call]].
