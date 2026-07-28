---
name: traceability-red-is-corpus-level
description: requirement_traceability is a CORPUS-level gate step, not diff-scoped — a bundle promotion elsewhere reds it for every implementer until the dependent spec's `supports` pins are rebumped
metadata:
  type: project
---

`requirement_traceability` evaluates the whole artifact corpus, so it can go RED
on a change that touches only Go files. The usual cause: someone promotes a
bundle and bumps a requirement's version (e.g. BUNDLE-006 REQ-020 1.0.0 ->
1.1.0), which strands every implemented spec whose `supports:` ref still pins the
old version, and then the spec's plan reports "bundle requirement chain does not
verify". Three violations from one bump.

**Attribution recipe** (do this before touching anything):

```
git show --name-only --format="" <your-commits>       # any artifact files? usually none
git merge-base --is-ancestor <bundle-promote-sha> <your-first-commit>
git status --porcelain <cited spec/plan/bundle>       # blank = you never touched it
```

An affirmative ancestor check plus a clean status on the cited artifacts proves
the red is INHERITED, not activated by you. Report it attributed and leave it.

**Do not fix it yourself**: rebumping a `supports:` pin means hand-editing a
spec, which CLAUDE.md forbids — route it to spec-author. Contrast with
[[project_editing_file_pulls_it_into_gate_scope]], where a file-scoped finding in
a file you touched IS yours to fix outright.

Related: [[feedback_netnegative_gate_baseline]].
