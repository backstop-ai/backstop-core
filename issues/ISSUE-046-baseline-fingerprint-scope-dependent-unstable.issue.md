---
title: "Baseline Fingerprint Scope Dependent Unstable"
schema_version: issue/v1

issue:
  id: ISSUE-046
  title: "Baseline Fingerprint Scope Dependent Unstable"
  type: bug
  status: open
  created: "2026-07-06"

complexity:
  scope: cross-cutting
  uncertainty: known
  risk: critical
---

# ISSUE-046: Baseline Fingerprint Scope Dependent Unstable

## Problem

The gate's baseline comparison keys every violation on `Identity`/`IdentityHash`
(`pkg/gate/baseline.go:174-183`, `EnrichViolationIdentity`). For the SAME
violation, this key is **not stable** between a full-scope run
(`backstop baseline generate`, `backstop gate --all`) and a diff-scope run
(`backstop gate`). A `baseline generate` immediately followed by `gate` does
not reliably grandfather the violations it just captured.

**Concrete repro (2026-07-06, during ISSUE-045):** an 8-line edit to
`pkg/pack/manifest.go` surfaced 16 pre-existing `backstop/go-standards`
findings (13 error-wrapping + 2 idiomatic-global + 1 in `pack_gate.go`) as
`new_violations` under diff-scope `backstop gate`, immediately after a
`backstop baseline generate` had captured them. For comparison, an
analogous baseline refresh in ISSUE-019 DID grandfather its findings — the
behavior is inconsistent, which is itself the smell: identity should never
depend on how the gate was invoked.

### Root cause: `Identity` folds in an un-normalized, invocation-shape-dependent `File`

`EnrichViolationIdentity` builds the baseline key as:

```go
identity := strings.TrimSpace(v.Rule) + "|" + strings.TrimSpace(v.File)
region := strings.TrimSpace(v.RegionHash)
if region == "" {
    region = hashString(strings.TrimSpace(v.Message) + "|" + strings.TrimSpace(v.Severity) + "|" + strings.TrimSpace(v.SourcePack))
}
v.Identity = identity + "|" + region
```
(`pkg/gate/baseline.go:174-181`)

`File` is folded into `Identity` **raw** — straight from
`pl.ArtifactLocation.URI` in the SARIF result
(`pkg/check/parsers.go:112-116`, `parseSarif`), through
`gate.Violation{File: v.File, ...}` in the engine-findings bridge
(`cmd/backstop/pack_gate.go:722-733`), with **no normalization step**
(no `filepath.Clean`, no `filepath.ToSlash`, no leading-`./` strip) at any
point before it reaches `EnrichViolationIdentity`.

That raw path is exactly the value at risk of differing by scope, because
`runFindingsEngine` (`cmd/backstop/pack_gate.go:591-637`) hands the
findings engine (semgrep, per every rule in
`backstop/go-standards/pack.yml` declaring `engine: semgrep`) a
**structurally different invocation target depending on scope**:

- **Full scope** (`scope == nil || scope.Mode == GateScopeModeAll` —
  `baseline generate`, `gate --all`): appends the single project-root
  directory argument (`cmd/backstop/pack_gate.go:632-634`,
  `cmdArgs = append(cmdArgs, projectRoot)` — `projectRoot` is `"."` for a
  normal in-place run). Semgrep is invoked as `semgrep --config <rule> .`
  and walks the tree itself.
- **Diff scope** (`backstop gate`, no `--all`): appends the explicit,
  repo-relative changed-file list (`cmd/backstop/pack_gate.go:634-636`,
  `cmdArgs = append(cmdArgs, scope.Files...)`). Semgrep is invoked as
  `semgrep --config <rule> pkg/pack/manifest.go cmd/backstop/pack_gate.go
  …` with each file named directly.

A CLI scanner walking a directory argument (`.`) versus being handed
explicit file arguments commonly reports paths in a different textual form
for the identical file (e.g. a `./`-prefixed path from the directory walk
vs. the bare relative path echoed back from an explicit argument — this is
standard behavior for many POSIX-style tools, semgrep included). Whichever
exact byte differs, the code-level defect holds regardless of the engine's
specific quirk: **`runFindingsEngine` is scope-branch-shaped to hand the
engine two different argument forms for the same target set, and nothing
downstream normalizes the path the engine reports before folding it into
the baseline key.** `normalizeScopePath`
(`pkg/gate/scope.go:91-102`, `filepath.Clean` + `filepath.ToSlash` +
absolute-to-relative) already exists in the same package for exactly this
purpose — but it is only ever called from `GateScope.Contains`
(`pkg/gate/scope.go:64-70`, used by the scope *filter*), never from
`EnrichViolationIdentity` or anywhere on the path that builds
`Violation.File` for baseline identity.

Net: `IdentityHash` is a function of an unnormalized, invocation-shape-
dependent string, and the invocation shape is *itself* a function of gate
scope. A violation's baseline identity can therefore differ purely because
of how the gate was run, independent of the violation's actual content —
even though `RegionHash` (the content-based half of the key, sourced from
`sarifFingerprint` in `pkg/check/parsers.go:133-152`, using semgrep's
`partialFingerprints` when present) is designed to be scope-independent.
`RegionHash` being stable cannot rescue the match, because `File`
participates in `Identity` independently and unnormalized.

### Why it is high-severity in both directions

- **False "new" (the observed direction):** pre-existing debt in a touched
  file — or any file whose engine-reported path form shifts for scope
  reasons — surfaces as new, falsely blocking a gate that should be clean.
  It also means a legitimate false-positive (an over-flagged idiomatic
  global) can never be grandfathered even by refreshing the baseline —
  it is stuck red, compounded by ISSUE-017 (`// nosemgrep` inert in the
  SARIF path, so the usual suppression escape-hatch is unavailable too).
- **False "grandfathered" (the dangerous direction — silent green):** the
  same instability that produces a false "new" match can, by the identical
  mechanism, produce a false match in the other direction — a genuinely
  NEW violation whose reported path happens to normalize to a stale
  baseline entry's raw (unnormalized) form is silently forgiven. This is
  precisely the vacuous-green hole the baseline/ratchet exists to prevent
  (`CLAUDE.md`, "Loud ≠ blocking" / enforcement philosophy).
- Net: the baseline/ratchet is only trustworthy if a violation has ONE
  stable identity regardless of the scope or invocation shape used to
  observe it. Right now it does not.

### Scope of the instability

Confirmed for `pack_engines`/`backstop/go-standards` (SARIF/semgrep path,
the concrete repro above). The traceability dimensions
(`test_verification`, `test_substantiveness`, `contract_signature`,
`coverage_threshold`) build their own `Violation.File` values through
different code paths (not through `runFindingsEngine`'s scope-branch), so
they are not automatically implicated — but they are not yet verified
clean either, since none of them normalize `File` before
`EnrichViolationIdentity` runs. Any dimension whose `Violation.File`
construction differs by scope carries the same class of risk. Auditing
those paths is in scope for the fix.

## Solution (fix direction to evaluate — the planner details it)

Per the approved (not yet built) baseline design
(`project_baseline_design` memory): make the fingerprint **content-based,
position-independent, and language-blind**, derived from the SARIF
finding's stable content (rule id + a normalized, content-derived region
hash + a normalized repo-relative path) — never from a raw engine-reported
path string or scope-dependent invocation shape — so a violation has ONE
identity whether computed full-scope or diff-scope.

Concretely:

1. **Normalize `File` before it ever reaches identity computation.**
   Route every `Violation.File` through the same normalization
   `normalizeScopePath` already performs (`filepath.Clean` +
   `filepath.ToSlash` + absolute-to-relative against `projectRoot`) at the
   point it is set (`pkg/check/parsers.go` `parseSarif`, and/or
   `cmd/backstop/pack_gate.go:722-733`'s `gate.Violation{...}` construction)
   — not just at the scope-filter call site. Consider promoting
   `normalizeScopePath` (or an equivalent) to a shared helper both call
   sites use, so there is exactly one normalization implementation, not
   two that can drift.
2. **Do not change engine invocation shape as the fix.** The full-scope
   vs. diff-scope argument-shaping difference in `runFindingsEngine`
   (directory arg vs. explicit file list) is legitimate and load-bearing
   (diff-scoping keeps rule-fed engines from scanning the whole repo,
   ISSUE-010) — normalize the OUTPUT path, don't try to force identical
   engine inputs.
3. **Re-verify `RegionHash`/`sarifFingerprint` is actually engine-stable in
   practice**, not just by design intent — confirm semgrep's
   `partialFingerprints` (or the snippet fallback) is identical across a
   full-scope and a diff-scope invocation of the SAME rule against the SAME
   file, empirically, not just by reading the comment in
   `pkg/check/check.go:21-26`.
4. **Audit every non-`pack_engines` dimension's `Violation.File`
   construction** for the same class of scope-dependent raw path risk
   (see "Scope of the instability" above), and normalize those too if
   affected.

### Tests to add

- A finding's fingerprint (`Identity`/`IdentityHash`), computed once via the
  full-scope path and once via the diff-scope path for the identical
  violation (same rule, same file, same code), is **byte-identical** in
  both directions.
- A `baseline generate` (full scope) immediately followed by a `gate` (diff
  scope) against the SAME tree produces **zero** `new_violations` for
  content that was present in the just-generated baseline — the direct
  regression guard for the ISSUE-045 repro.
- A finding whose surrounding file shifts (an unrelated edit changes line
  numbers or file-list ordering) but whose own rule+file+content is
  unchanged **keeps** its fingerprint (not classified as new).
- A finding whose rule+file+content genuinely changed (new violation) gets
  a **new**, distinct fingerprint (not silently absorbed into a stale
  baseline entry) — the regression guard for the false-grandfather /
  silent-green direction.

### Constraints / non-negotiables

- The fix must stay language-blind (`CLAUDE.md` thin-executor first
  principle / `feedback_zero_baked_checks`): normalization operates on
  path strings and SARIF content generically, with no language-specific
  parsing.
- Do not weaken `RegionHash`'s existing content-based design intent
  (`pkg/check/parsers.go:133-152`) — the fix targets the unnormalized
  `File` component and verifies the content component, not a rewrite of
  the whole scheme, unless investigation shows `sarifFingerprint` itself
  is also unstable in practice.

## References

- **ISSUE-045** (Coverage Flags Unmeasurable And Root Package Files) — the
  change during which this instability was discovered; also the source of
  the repo-relative-vs-module-qualified path normalization pattern this fix
  should mirror.
- **ISSUE-017** (`// nosemgrep` suppressions silently ignored) — compounds
  the false-"new"/stuck-red direction: a legitimate false-positive can
  neither be suppressed inline nor grandfathered via baseline refresh.
- **ISSUE-019** — prior baseline refresh that DID grandfather its findings;
  the inconsistency between that outcome and the ISSUE-045 repro is the
  evidence that identity is scope-dependent rather than reliably broken or
  reliably working.
- **DIR-003** (Baseline Implementation) — this is baseline-mechanism
  territory; the approved `project_baseline_design` memory ("fix coarse
  fingerprint (RegionHash from SARIF content, language-blind)") already
  anticipates this class of fix.
- **DIR-015** (Gate Checker Hardening) — pulled forward as a blocker: gate
  hardening work is untrustworthy while the same tree can gate red or green
  purely as a function of full-scope vs. diff-scope invocation.
