# Capture source — transform before/after fixture (TASK-004)

Covers `pkg/recipe/testdata/captured/transform/` (CLM-010, CLM-026).

## Files

| File | Origin |
| --- | --- |
| `transform/rewrite-rule.yml` | Hand-authored ast-grep rewrite rule (pack DATA). Modeled on `cmd/backstop/testdata/astgrep-multirule/.backstop/packs/test-org/multirule-pack/ast-grep/rules/rule-one.yml` — same `id` / `language` / `rule.pattern` / `message` shape, extended with a `fix:` so the rule REWRITES rather than only reports. |
| `transform/before.go` | Hand-authored Go source snippet — the input state the rule's pattern matches. Two call sites of `legacyHelper(...)` plus both helper declarations. Deliberately free of any `exec.Command` literal and of any foreign-language / manifest token, because `testdata/*.go` is in scope for the GLOBAL `backstop/self` families A / B1 / B2. |
| `transform/after.go` | **CAPTURED — the verbatim output of running the real `ast-grep` binary** over a scratch copy of `before.go`. Not typed by hand; the file is byte-for-byte what the tool wrote back. |

## Tool

```
$ ast-grep --version
ast-grep 0.43.0
```

`/usr/local/bin/ast-grep`. 0.43.0 is the version pinned for `ast-grep` in
`pkg/pack/engine/allowlist.go` (`TrustedToolAllowlist`), so the capture was
taken with the exact build the gate provisions.

## Exact capture procedure

Run outside the repository (a scratch working copy) so the committed
`before.go` is never mutated in place:

```
$ cp pkg/recipe/testdata/captured/transform/before.go        <scratch>/work/before.go
$ cp pkg/recipe/testdata/captured/transform/rewrite-rule.yml <scratch>/rewrite-rule.yml
$ cd <scratch>
$ ast-grep scan --rule rewrite-rule.yml --update-all work/before.go
Applied 2 changes
$ cp work/before.go pkg/recipe/testdata/captured/transform/after.go
```

`--update-all` applies the rule's `fix:` in place and prints the change count;
`work/before.go` after that command IS `after.go`.

## Why the `.go` extension is load-bearing

`ast-grep` picks its parser from the FILE EXTENSION. A neutral extension
(`.src`, `.txt`) parses as nothing, the rewrite silently no-ops, and
`before == after` would make the whole transform suite vacuous. The extension
must therefore be real and must match the rule's declared `language: go`.

`.go` specifically: it matches the existing `astgrep-multirule` precedent, and
`backstop/self` Family B2 (`no-baked-language-token`) deliberately OMITS `.go`
from its token regex, so naming this path in Go test source is not a
baked-token violation. Family B3, which does cover `.go`, is path-scoped to
the neutral gate spine and excludes `_test.go` — it does not reach
`pkg/recipe`.

## Verification — before and after DIFFER

```
$ diff -u before.go after.go
--- before.go
+++ after.go
@@ -2,12 +2,12 @@

 // Greeting builds the message shown to a newly onboarded member.
 func Greeting(name string) string {
-	return legacyHelper(name, "welcome")
+	return modernHelper(name, "welcome")
 }

 // Farewell builds the message shown when a member signs out.
 func Farewell(name string) string {
-	return legacyHelper(name, "goodbye")
+	return modernHelper(name, "goodbye")
 }

 func legacyHelper(name string, verb string) string {
```

Two call sites rewritten (matching the tool's own `Applied 2 changes`). The
`legacyHelper` / `modernHelper` DECLARATIONS are untouched: the rule's pattern
`legacyHelper($$$ARGS)` is a call expression, so a function declaration of the
same name does not match. That asymmetry is itself part of what the fixture
pins — a transform executor that rewrote the declarations too would fail
against this captured output.
