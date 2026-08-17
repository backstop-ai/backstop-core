---
name: stale-sibling-scaffold-evidence
description: A plan's "sibling plan N is an EMPTY SCAFFOLD" collision evidence goes stale within MINUTES on a parallel-authoring night — re-extract every sibling's phases and file scope yourself
metadata:
  type: project
---

Never accept a plan's own cross-lane collision check when it describes siblings as "empty
plan scaffolds" or "`phases: []`, untracked". On a night where several planners run in
parallel, that evidence expires in minutes.

**Why:** PLAN-ISSUE-142 (authored 03:43, 2026-08-17) recorded "PLAN-ISSUE-144
(stdout_artifact) is an EMPTY SCAFFOLD (`phases: []`, untracked)" and "five empty plan
scaffolds (PLAN-ISSUE-142/144/146/148/151)". PLAN-ISSUE-144's mtime was 03:48 — five
minutes later — and by review time all five were fully authored (144 = 655 lines, 148 =
907). The disjointness CONCLUSION still held, but the evidence for it did not, and the
plan had never mentioned PLAN-ISSUE-151 at all, which by then owned `pkg/packval/phase2.go`,
`pkg/packval/pathscope*.go` and `pkg/packval/testdata/path-scope/**` — the same package the
plan under review edits.

**How to apply:** Re-derive it mechanically. `ls -la plans/` shows both size and mtime
(a multi-KB file is not a scaffold), then extract every sibling's real file scope:

```
python3 - plans/PLAN-ISSUE-NNN-*.yml <<'EOF'
import sys,yaml
d=yaml.safe_load(open(sys.argv[1]))
for ph in d.get('phases') or []:
    print("##", ph['id'], ph.get('name'))
    for t in ph.get('tasks') or []:
        print("   ", t['id'], t['type'], "|", t['title'], "|", t.get('files'))
EOF
```

Intersect the file sets across all live sibling plans. Same-PACKAGE-different-file is fine
but must be NAMED in the plan (an implementer told "144 is empty" will not know to leave
`phase2.go` alone). Note that `git status --porcelain plans/` shows sibling plans as `??`
untracked whether they are empty or 900 lines — untracked is NOT evidence of emptiness.

Related: [[sibling-precedent-cited-not-read]], [[lane-enumeration-misses-pending-tasks]],
[[uncommitted-lane-rows-move-pinned-counts]].
