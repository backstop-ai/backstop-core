---
name: projectroot-is-always-absolute
description: Specs justifying a subject choice by "root colocation is invocation-directory-dependent" are fabricating — projectRoot is always absolute, so a root test file's leaf is the repo dir basename
metadata:
  type: feedback
---

`runGate` derives `projectRoot = filepath.Dir(cfgPath)` where `cfgPath` comes from
`config.DiscoverConfigPath` → `DiscoverConfigPathFrom`, which does `filepath.Abs(startDir)`
before joining `backstop.yml` (`pkg/config/config.go:120-160`). **The result is ALWAYS an
absolute path**, identical whether the gate is invoked from the repo root or any
subdirectory. Verified empirically 2026-08-10.

Consequences a spec must not get wrong:
- Mandated-test paths from `collectTestFuncNamesScoped`'s `filepath.Walk(codeDir, …)`
  (`pkg/gate/step_testverify.go:460`) are ABSOLUTE.
- A repo-ROOT test file's `filepath.Base(filepath.Dir(path))` is the **repository directory
  basename** (`backstop-core`) — never `"."`, never `".."`.
- So `implementation.subject: "."` does NOT colocate a root-package test
  (`TargetPackageName(".")` = `"."` ≠ `"backstop-core"`), and there is NO
  invocation-directory dependence to appeal to. The real (and better) fragility argument for
  avoiding a root subject is that it would have to be the repo DIRECTORY NAME, which a clone
  into a differently-named directory breaks.

**Why:** SPEC-066 v2.0.0 was a fix pass correcting a fabricated mechanical claim about root
colocation, and replaced it with a second fabricated claim carrying cited line numbers. Both
were false. Reviewers must run the derivation, not read the prose.

**How to apply:** any spec justifying `implementation.subject` (or a claim about noTarget /
colocation) with reasoning about `"."`, `".."`, or where the gate was invoked from is wrong
on its face — check `DiscoverConfigPathFrom` before accepting it. See
[[project_spec066_review2]].
