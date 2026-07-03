---
name: whole-file-delete-shared-types
description: A "delete the whole file" deletion task must verify the file doesn't DEFINE types/helpers used by surviving code (ConfigError/DegradedError lived in semgrep.go but the live path used them)
metadata:
  type: project
---

When a pure-deletion plan says "delete file X in full," grep every type, const, var,
and helper DEFINED in X for non-test callers OUTSIDE the dead cluster. Deleting the
file orphans those definitions and breaks compilation of surviving code — the worst
case in a "no behavior change" deletion.

**Why:** PLAN-ISSUE-018 deleted `pkg/check/semgrep.go` in full to remove the dead
in-process executor, but that file also DEFINED `ConfigError`/`DegradedError` — the
fail-loud exit-2 types used pervasively by the LIVE engine-dispatch path
(parsers.go:34 lookupParser, manifest.go, registry.go, output.go, and
cmd/backstop via check.ConfigError). Deleting the file would break pkg/check +
cmd/backstop compilation and silently drop the fail-loud contract. The plan's
caller-map only checked callers of the DEAD symbols, not co-located live types.

**How to apply:** for every file a deletion task removes in full, run
`grep -rn "<each exported/unexported type/func defined there>" --include="*.go" .`
filtered to non-test, non-cluster files. If any survive, the type must be RELOCATED
(to a surviving file) before the delete, and that relocation needs its own task/scope.
Sibling memory: [[whole-file-delete-mandated-test-readsource]].
