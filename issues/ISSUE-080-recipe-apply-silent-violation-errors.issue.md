---
title: "`recipe apply` Drops a Malformed Waiver Token's Diagnostic and Silently Clobbers a Diverged File"
schema_version: issue/v1

issue:
  id: ISSUE-080
  title: "`recipe apply` Drops a Malformed Waiver Token's Diagnostic and Silently Clobbers a Diverged File"
  type: bug
  status: open
  created: "2026-07-25"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# `recipe apply` Drops a Malformed Waiver Token's Diagnostic and Silently Clobbers a Diverged File

## Delivered by SPEC-055 (2026-07-26) — do not re-open

This issue originally reported two problems under one root cause: `recipe apply`
wrapped every violation-class failure as an `ExitCodeError{Code: ExitViolations}`
with nothing printed first, and `cmd/backstop/main.go`'s old convention silently
suppressed the message for every `ExitViolations` error — discarding the op's
declared `manual:` fallback (SPEC-054 REQ-011) along with every other diagnostic
(missing required param, unresolvable merge fragment, unreachable injection site).

**SPEC-055 (`cmd/backstop/main.go:22-56`, `reportError`/`Explained`) fixed this
generically, repo-wide, and it covers `recipe apply` by name.** The default
inverted to loud: `reportError` prints an `*ExitCodeError`'s message unless
`Explained` is set, and `Explained` is claimed by exactly four sites (`gate`'s
violations verdict, `pack check`, `pack test`, `artifact validate`) that already
render a structured report before failing. `recipe apply` is not one of the four,
so its wrap at `cmd/backstop/recipe_apply.go:70` — unchanged, still
`&ExitCodeError{Code: ExitViolations, Message: err.Error()}` — now prints
unconditionally. `main.go`'s own comment on `reportError` names both silenced
commands this fixed: "ISSUE-074 pack relock, ISSUE-080 recipe apply."

Verified against current code and passing tests (2026-07-26):
- `TestExitSurfacing_RecipeApply_PrintsDiagnostic`
  (`cmd/backstop/exit_surfacing_streams_test.go`) drives the built binary,
  asserts the op's declared `manual:` fragment lands on stderr and stdout stays
  empty — CLM-075's mandated test, closing this issue's original repro by name.
- `TestRecipeApply_CLI_UnreachableTargetRelaysDeclaredManualVerbatim`
  (`cmd/backstop/recipe_apply_e2e_test.go:303`) asserts the same manual text is
  relayed verbatim on the `ExitCodeError.Message` for an unreachable transform
  site.
- Both ran clean: `go test ./cmd/backstop/... -run
  'TestExitSurfacing_RecipeApply_PrintsDiagnostic|TestRecipeApply_CLI_UnreachableTargetRelaysDeclaredManualVerbatim'`
  → PASS.

The fix is structural, not per-op-kind: `runRecipeApply` (`cmd/backstop/recipe_apply.go:60-71`)
wraps whatever error `recipe.Apply` returns identically regardless of which op
produced it, so a `transform` op's `injectionLimit` failure and an `insert` op's
`missingSite` failure (`pkg/recipe/apply.go:576-604`) both flow through the same
`err.Error()` → `ExitCodeError.Message` → `reportError` path already proven above
for the `transform` case. There is no CLI-level regression test naming the
`insert`-kind path specifically; that is a minor test-coverage gap, not a live
defect, since the wrap point both kinds share is the one already under test.

**What remains open, below, is recipe-specific and NOT covered by SPEC-055's
mechanism**: it lives in `pkg/recipe/apply.go`'s waiver-divergence adjudication,
a code path `reportError` never touches because it fails silently at exit 0, not
exit 1.

## Problem

`backstop recipe apply` silently **overwrites an operator's manually-diverged
edit to a recipe-owned file** when the waiver token guarding that divergence has
a malformed reason code — with no error, no warning, and exit 0.

### Repro (live-reproduced 2026-07-26, full recipe-scenario sweep against a real
local pack, applied via the real CLI in a scratch consumer)

A recipe-owned file has been intentionally hand-edited by the operator and marked
with a divergence token carrying an illegal reason code:
`@waiver:some-rule:intentional-fork:2026-12-31` — where `intentional-fork` is not
one of `pkg/waiver`'s closed `ReasonCode` enum (`false-positive`, `accepted-risk`,
`deferred`, `third-party`; `pkg/waiver/waiver.go:26-37`). Re-running `recipe apply`
on that recipe:

- **regenerates the file, overwriting the operator's edit**, and
- **exits 0** printing only `applied recipe <ref>` (`cmd/backstop/recipe_apply.go:73`).

No message names the token, no message says the file was rewritten, no message
distinguishes this from a completely clean re-apply. The operator's edit is gone
with zero indication anything unusual happened.

### Root cause

`pkg/recipe/apply.go:337-356`, `adjudicateDivergence`, is this package's
`WaiverReader` over the real `pkg/waiver` read path:

```go
func adjudicateDivergence(rule string, file string) bool {
    ...
    return len(waiver.Adjudicate(findings, read, nil, time.Now()).Suppressed) > 0
}
```

`waiver.Adjudicate` already classifies an unparseable reason code as a
`DiagnosticMalformed` entry in `Result.Diagnostics`
(`pkg/waiver/adjudicate.go:39,120`) — the signal this needs already exists in the
value `adjudicateDivergence` receives. But the `WaiverReader` type
(`pkg/recipe/apply.go:63`, `func(rule string, file string) (covered bool)`) is
closed to a single `bool`, so `adjudicateDivergence` reads only `.Suppressed` and
the `Diagnostics` slice — including the malformed-token entry — is dropped on the
floor before it can return.

`coveredDivergence` (`pkg/recipe/apply.go:304-321`) then sees "not covered" from
that `bool` and `preserveOrRegenerate` (`pkg/recipe/apply.go:266-303`) proceeds to
regenerate — **the exact same branch it takes when no token was present at all.**
A malformed token and an absent token are indistinguishable by the time
`preserveOrRegenerate` decides, even though `waiver.Adjudicate` told the caller
which one actually happened.

### Compounding gap — the success path never reports what it did

`cmd/backstop/recipe_apply.go:73` prints exactly one static line —
`applied recipe <ref>` — regardless of outcome. `ApplyResult` already carries the
distinction: `Preserved []PreservedDivergence` (`pkg/recipe/apply.go:96`, appended
at `apply.go:242` whenever a divergence is preserved) versus files actually
written. Neither is surfaced to the operator. So even once the malformed-token
diagnostic above is wired through, a VALID waiver's clean preservation and an
INVALID waiver's silent override will still print the identical line unless the
success output is also extended to name preserved vs. (re)written files.

### Why tests missed it

No test drives a diverged file through `recipe.Apply` with a token whose reason
code is outside the closed `ReasonCode` enum and asserts on what the operator is
told. `adjudicateDivergence`'s narrowing of `waiver.Result` to a single `bool` is
untested on its `Diagnostics` side entirely — `pkg/waiver/adjudicate_malformed_test.go`
exercises `waiver.Adjudicate` directly and proves `DiagnosticMalformed` is
produced, but nothing carries that proof through the recipe-apply seam to the CLI.

### Expected

`adjudicateDivergence` (or the `WaiverReader` shape it implements) surfaces
`waiver.Result.Diagnostics` rather than only `.Suppressed`, so a malformed token
is distinguishable from an absent one by the time `preserveOrRegenerate` decides,
and a malformed-token divergence produces a diagnostic naming the file and the bad
reason code — e.g. "waiver token on `<file>` has unknown reason code `%q`" —
instead of a silent, zero-feedback regeneration. Separately, `recipe apply`'s
success output should name every file it (re)wrote versus preserved, so a
diverged-and-regenerated notice is visibly different from a clean re-apply even
independent of the malformed-token message itself.

A process-level test — invoking the built `recipe apply` binary against a
recipe-owned file carrying a malformed-reason-code token and asserting the
operator's edit survives, or that the failure/warning is visible on an observable
stream — should guard the fix.

## References

- `pkg/recipe/apply.go:337-356` (`adjudicateDivergence`) — narrows
  `waiver.Adjudicate`'s `Result` to a single `bool` (`.Suppressed` only), dropping
  `.Diagnostics` on the floor; a malformed reason code produces a
  `DiagnosticMalformed` entry that never reaches the caller
- `pkg/recipe/apply.go:63` (`WaiverReader`) — the closed
  `func(rule string, file string) (covered bool)` signature `adjudicateDivergence`
  implements; changing what recipe apply can report requires widening this shape
- `pkg/recipe/apply.go:304-321` (`coveredDivergence`) — treats "not covered"
  (which includes "token was malformed") identically to "no token present" and
  proceeds to regenerate either way
- `pkg/recipe/apply.go:266-303` (`preserveOrRegenerate`) — the caller that
  actually overwrites the file when `coveredDivergence` reports "not covered"
- `pkg/waiver/waiver.go:22-46` (`ReasonCode`, `validReason`) — the closed
  four-member enum (`false-positive`/`accepted-risk`/`deferred`/`third-party`);
  any other value, e.g. `intentional-fork`, is malformed
- `pkg/waiver/adjudicate.go:39,120` (`DiagnosticMalformed`) — the malformed-token
  diagnostic kind `Adjudicate` already produces, unused by the recipe-apply path
- `pkg/recipe/apply.go:96,242` (`ApplyResult.Preserved`) — already tracks
  preserved divergences; never surfaced by the CLI
- `cmd/backstop/recipe_apply.go:73` — success path prints only `applied recipe
  <ref>`; never reports `ApplyResult.Preserved` or which files were (re)written,
  so a silently-overridden divergence is indistinguishable from a clean apply
- `specs/SPEC-055-production-remote-dependency-assembly.spec.md` — CLM-074/075,
  the `reportError`/`Explained` mechanism that delivered the ExitViolations-silence
  half of this issue's original scope; see "Delivered by SPEC-055" above
- `cmd/backstop/main.go:22-56` (`reportError`) — the seam SPEC-055 introduced;
  its own doc comment names this issue as one of the two defects it closes
- `pkg/waiver/adjudicate_malformed_test.go` — proves `waiver.Adjudicate` produces
  `DiagnosticMalformed`, but at the `pkg/waiver` layer only; nothing carries that
  proof through `pkg/recipe`'s divergence adjudication to the CLI
- `issues/ISSUE-074-pack-relock-silent-failure.issue.md` — the sibling defect
  SPEC-055 also closed (same `ExitViolations`-suppresses-stderr convention);
  cite as precedent, not as still-open
