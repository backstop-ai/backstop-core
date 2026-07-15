---
title: "go-standards error-type-suffix rule misfires on non-error structs"
schema_version: issue/v1

issue:
  id: ISSUE-061
  title: "go-standards error-type-suffix rule misfires on non-error structs"
  type: bug
  status: open
  created: "2026-07-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# ISSUE-061: go-standards error-type-suffix rule misfires on non-error structs

## Problem

The `backstop/go-standards` pack's `go.core.error-type-suffix` rule
(`GO-021`, `.backstop/packs/backstop/go-standards/rules/core/go-core.yml:152-162`)
fires on `type ValidateConfig struct` at
`cmd/backstop/artifact_validate.go:18` — a plain config struct with no
`Error()` method and no reason to be named `*Error` — while it does NOT
flag anything about the file's one actual error type, `ExitCodeError`
(`cmd/backstop/artifact_validate.go:35`, correctly suffixed, implements
`Error()` at line 40). The finding is currently suppressed only by an
inline waiver directly above the struct:

```go
// @waiver:backstop/go-standards/backstop.packs.backstop.go-standards.rules.core.go.core.error-type-suffix:false-positive:2026-10-12 pack rule fix pending — ValidateConfig is not an error type; rule misfires on a non-error struct
type ValidateConfig struct {
```

**The waiver expires 2026-10-12.** On expiry the finding returns unsuppressed
and `backstop gate` goes red on a false positive, with no upstream fix in
flight today.

### Root cause

The rule's pattern is a raw regex, not a scoped AST match, and it binds no
relationship between the struct it flags and the `Error()` method it treats
as evidence:

```
(?s)type\s+\w+(?<!Error)\s+struct\s*\{.*?\}\s*func\s*\([^\)]*\)\s*Error\s*\(
```

`(?s)` (DOTALL) plus the non-greedy `.*?` mean this matches across the
**entire file**, not one declaration: starting at any `type X struct {`
where `X` doesn't end in `Error`, it scans forward — through any number of
intervening declarations — for the *first* `func (...) Error(` anywhere
later in the file, and calls that a match. There is no capture/backreference
tying the receiver type of that `func (...) Error(` back to `X`. So the
rule doesn't actually check "does this struct itself implement `Error()`
without the suffix" — it checks "is there a non-`Error`-suffixed struct
declaration somewhere, followed eventually by *any* type's `Error()` method
anywhere else in the file."

In `artifact_validate.go` that's exactly what happens: `ValidateConfig`
(line 18, no `Error()` method of its own) is followed — 22 lines later, past
the unrelated `ValidateResult` struct — by `ExitCodeError`'s `Error()`
method (line 40). The non-greedy `.*?` bridges that gap, and the match
anchors the finding on `ValidateConfig` even though the `Error()` method it
matched belongs to a completely different, correctly-suffixed type.
Meanwhile `ExitCodeError` itself is exempted from the match by the
`(?<!Error)` lookbehind on its own type name, which is correct but
coincidental — the rule was never evaluating whether `ExitCodeError`
implements `Error()` correctly; it just can't be the `\w+(?<!Error)` anchor
because its name ends in `Error`.

## Solution

Fix belongs in the `backstop/go-standards` **pack repo**, not here — packs
are external and installed into gitignored `.backstop/packs/`
(CLAUDE.md); editing the installed copy is not durable and would be lost on
next `pack install`/`pack update`.

1. Rewrite `go.core.error-type-suffix` so it only matches a struct against
   its **own** `Error()` method — e.g. an ast-grep/semgrep relational rule
   scoped per type declaration (matching the receiver type name against the
   struct's own name), not a whole-file DOTALL regex that treats "any struct
   declaration" and "any later `Error()` method" as related evidence.
2. Add a regression fixture pair, per pack convention
   (`fixtures/rules/valid/` and `fixtures/rules/invalid/`, wired into
   `pack.yml`'s `claims.fixtures.positive`/`negative`), that reproduces this
   exact shape: a non-error struct declared immediately before an unrelated
   type that has a genuine, correctly-suffixed `Error()` method (must NOT
   fire), alongside the existing genuinely-missuffixed-error-type fixture
   (must still fire). `backstop pack test` is the harness that exercises
   these fixtures against the ruleset.
3. Cut a new `backstop/go-standards` version with the fix, update this
   repo's `backstop.lock` to it, and run `backstop pack install`.
4. Remove the inline waiver at `cmd/backstop/artifact_validate.go:17` once
   the fixed rule no longer fires on `ValidateConfig`.

### Acceptance

- [ ] `backstop/go-standards` ships a fixed `go.core.error-type-suffix` that
      does not fire on a non-error struct merely because an unrelated,
      correctly-suffixed error type appears later in the same file
      (`ValidateConfig`-shaped fixture), while still firing on a genuinely
      missuffixed error type (existing `go-021-error-type-no-suffix.go`
      shape) — both proven by fixtures in the pack's own repo.
- [ ] This repo's `backstop.lock` points at the fixed pack version and
      `backstop pack install` has materialized it into `.backstop/packs/`.
- [ ] The inline waiver on `cmd/backstop/artifact_validate.go:17` is
      removed.
- [ ] `backstop gate --all` is green with **zero**
      `error-type-suffix` waivers anywhere in the tree.

## Verification

Run `backstop pack test <path-to-go-standards-pack-checkout>` to prove both
fixture directions (false-positive shape stays clean, genuine violation
still fires) before cutting the pack release. After `pack install` +
waiver removal, run `./bin/backstop gate --all` and confirm no
`error-type-suffix` findings and no `error-type-suffix` waivers remain.

## References

- `.backstop/packs/backstop/go-standards/rules/core/go-core.yml:152-162` —
  `go.core.error-type-suffix` (`GO-021`), the misfiring whole-file DOTALL
  regex
- `.backstop/packs/backstop/go-standards/fixtures/rules/valid/go-021-error-type-suffix.go`,
  `.backstop/packs/backstop/go-standards/fixtures/rules/invalid/go-021-error-type-no-suffix.go`
  — existing single-type fixtures; neither exercises the multi-declaration
  shape that trips this bug
- `cmd/backstop/artifact_validate.go:17` — the inline waiver suppressing the
  false positive, expires 2026-10-12
- `cmd/backstop/artifact_validate.go:18` — `ValidateConfig`, the struct
  incorrectly flagged
- `cmd/backstop/artifact_validate.go:35,40` — `ExitCodeError` and its
  `Error()` method; the actual error type in the file, not flagged, and not
  the type the rule's match was ever really evaluating
- `backstop.lock` — durability boundary for the pack version pin; must be
  updated once the fix ships
- CLAUDE.md — "packs live OUTSIDE core... editing the installed copy is
  non-durable"; "waivers are last resort, fix > waive"
