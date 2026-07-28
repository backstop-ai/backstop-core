---
name: feedback-colon-not-true-hermetic-fixture
description: Hermetic fixture test_command must be ":" (POSIX special builtin), never "true" — LookPath("true") resolves /usr/bin/true, so builtin-ness and PATH-absence are different properties
metadata:
  type: feedback
---

Any fixture asserting "this command reaches nothing outside the process" via
`exec.LookPath` must use `:` — the POSIX SPECIAL built-in null command — NOT
`true`.

**Why:** `true` is a shell builtin AND a real binary (`/usr/bin/true` exists on
macOS and every coreutils system), so `LookPath("true")` SUCCEEDS and the
assertion fails. `:` has no external counterpart, and a conforming shell resolves
special builtins BEFORE any PATH search. This bit SPEC-056: the plan prescribed
`true` in TASK-003 while TASK-006 mandated that LookPath ERROR — mutually
unsatisfiable. Caught at fixture-authoring time, then proven by falsification in
phase 2 (swapping `:` → `true` reds CLM-103 with "resolves to the executable
/usr/bin/true on PATH"). Plan amended at a7fcd79.

**How to apply:** "is a shell builtin" and "is absent from PATH" are DIFFERENT
properties, and only the second is mechanically checkable — write claims and
assertions against the second. Relevant to any hermetic-fixture work under
SPEC-056 REQ-010 or future sandbox/hermeticity claims. Related:
[[feedback-never-stash-shared-tree]].
