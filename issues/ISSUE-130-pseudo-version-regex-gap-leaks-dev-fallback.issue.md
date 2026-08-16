---
title: "Pseudo Version Regex Gap Leaks Dev Fallback"
schema_version: issue/v1

issue:
  id: ISSUE-130
  title: "Pseudo Version Regex Gap Leaks Dev Fallback"
  type: bug
  status: open
  created: "2026-08-15"

complexity:
  scope: isolated
  uncertainty: known
  risk: safe
---

# Pseudo Version Regex Gap Leaks Dev Fallback

## Problem

A plain local build with no injected ldflags (`go build -o bin/backstop ./cmd/backstop/` — the
exact `make build` target, `Makefile:1-2`, confirmed to carry no `-ldflags`) reports a
VCS-derived pseudo-version instead of the honest `"dev"` fallback the CLI's own doc comments say
it should report. This is what `README.md:13` now tells contributors to run ("build from source
with `make build`") and what `CONTRIBUTING.md:7` documents as the build step.

## Reproduction

```
$ go build -o /tmp/test-backstop ./cmd/backstop && /tmp/test-backstop version
backstop version v0.1.3-0.20260815231324-cf17746b5df5
```

Confirmed by running the repo's own mandated regression test, which currently fails:

```
$ go test ./cmd/backstop/ -run TestVersion_LdflagsInjectionReachesBuiltCLI -v
--- FAIL: TestVersion_LdflagsInjectionReachesBuiltCLI (2.36s)
    version_test.go:208: plain build reports "v0.1.3-0.20260815231324-cf17746b5df5", want "dev"
        — a VCS-derived pseudo-version is leaking through the fallback
```

`TestVersion_LdflagsInjectionReachesBuiltCLI` (`cmd/backstop/version_test.go:197-213`) is not a
new test written for this issue — it already exists, already documents the intended behavior
("plain build reports dev, want dev — a VCS-derived pseudo-version is leaking through the
fallback" is literally its own failure message), and is already wired into the suite `make test`/
`make ci` run. It is failing right now on `main`.

## Root cause

This is not an unguarded gap — `cmd/backstop/version.go` already has anti-pseudo-version logic,
added specifically to reject exactly this class of value (see the doc comments on
`pseudoVersionSuffix`, `isReleasedModuleVersion`, and CLM-007's test coverage in
`version_test.go`). The guard has a regex bug that makes it miss the pseudo-version shape Go
actually produces in this repository.

`isReleasedModuleVersion` (`version.go:59-70`) rejects a build-info module version as
non-released if it matches `pseudoVersionSuffix`:

```go
var pseudoVersionSuffix = regexp.MustCompile(`-\d{14}-[0-9a-f]{12}`)
```

This pattern only matches Go's pseudo-version format for a repo with **no preceding version
tag** — `vX.0.0-yyyymmddhhmmss-abcdefabcdef`, where the 14-digit timestamp follows the dash
immediately. But per Go's own pseudo-version spec
(https://go.dev/ref/mod#pseudo-versions), when the most recent versioned commit **is** a tag
(the actual state of this repo — `v0.1.0`, `v0.1.1`, etc. are real tags per ISSUE-087's release
pipeline), the toolchain instead produces:

```
vX.Y.(Z+1)-0.yyyymmddhhmmss-abcdefabcdef
```

— note the `0.` counter segment inserted between the dash and the timestamp. The captured
reproduction value is exactly this shape: `v0.1.3-0.20260815231324-cf17746b5df5`. Verified
directly against the two regexes in isolation:

```go
pseudoVersionSuffix := regexp.MustCompile(`-\d{14}-[0-9a-f]{12}`)
releasedVersion := regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)
v := "v0.1.3-0.20260815231324-cf17746b5df5"
pseudoVersionSuffix.MatchString(v)  // false — the "0." prefix breaks the \d{14} match
releasedVersion.MatchString(v)      // true  — matches the general X.Y.Z(-prerelease) shape
```

`pseudoVersionSuffix` fails to match (the character right after the dash is `0`, then `.`, not
14 consecutive digits), so `isReleasedModuleVersion` never reaches its intended rejection.
`releasedVersion`'s pre-release group (`[0-9A-Za-z.-]+`) is permissive enough to accept the
dashes, digits, and hex letters in a pseudo-version's suffix, so the value passes as if it were a
real release tag and `resolveVersion` (`version.go:37-48`) returns it unchanged.

The same gap applies to the third Go pseudo-version shape (`vX.Y.Z-pre.0.yyyymmddhhmmss-hash`,
used when the preceding tag itself carries a pre-release segment) — it also inserts a
non-digit-prefixed counter before the timestamp, for the same reason.

`TestResolveVersion_PseudoVersionFallsBackToDev` (`version_test.go:65-73`) and
`TestResolveVersion_DirtyBuildFallsBackToDev` (`version_test.go:75-94`) only exercise the
no-preceding-tag pseudo-version shape (`v0.0.0-20260727014125-1ccb2a60b2f7`, no `0.` prefix)
and the `+dirty` metadata case — neither covers the tagged-repo pseudo-version shape this repo
actually produces, which is why the gap shipped without a table-test catching it; only the slow,
real-build integration test (`TestVersion_LdflagsInjectionReachesBuiltCLI`) exercises a real
tagged checkout and catches it.

## Scope

Confirmed NOT affecting real distributed releases: `.goreleaser.yml` explicitly sets
`ldflags: -s -w -X main.version={{.Version}}` for every release build (what ships via Homebrew
and the GitHub releases page) — `resolveVersion`'s injected-value precedence (step 1) returns
that value unconditionally and never reaches the pseudo-version check. This bug is scoped to
local ad-hoc builds with no explicit `-ldflags`: `make build`, or a bare `go build
./cmd/backstop`, run from within this git repository (any checkout with a real tag history,
which is every clone of this repo).

Practical impact: anyone building from source per `README.md`/`CONTRIBUTING.md` sees a
confusing, real-looking, but meaningless version string wherever the CLI surfaces its version —
`backstop version`, `backstop version --json`, and `backstop doctor`'s build-identity check all
read the same `effectiveBuildIdentity()` (`version.go:155-158`), so all three are affected
identically.

## Direction (not scoped here)

The fix is a regex correction, not a new mechanism: `pseudoVersionSuffix` needs to also match the
`0.`-prefixed (and `pre.0.`-prefixed) counter segment Go inserts before the timestamp when a
preceding tag exists, per the pseudo-version formats documented at
https://go.dev/ref/mod#pseudo-versions. Whoever picks this up should re-derive the intended
pattern from that spec rather than special-casing the one captured reproduction string, since all
three pseudo-version shapes need to be rejected and only one is currently exercised by a passing
table test.

`TestVersion_LdflagsInjectionReachesBuiltCLI` is the existing regression guard — no new mandated
test is required to prove the fix, but the currently-uncovered tagged-repo pseudo-version shape
(`vX.Y.(Z+1)-0.yyyymmddhhmmss-hash`) should probably also get its own fast table-test case
alongside `TestResolveVersion_PseudoVersionFallsBackToDev`, so a future regression is caught by
the sub-millisecond unit test rather than only by the two-real-builds integration test.

## Notes / references

- Reported by the team lead during this session, 2026-08-15; reproduced and root-caused directly
  against `cmd/backstop/version.go` and `cmd/backstop/version_test.go` on `main` before filing.
- `TestVersion_LdflagsInjectionReachesBuiltCLI` (`cmd/backstop/version_test.go:197-213`) is the
  pre-existing mandated test that already fails and documents the intended behavior.
- `Makefile:1-2` (`build:` target) confirmed to carry no `-ldflags`.
- `.goreleaser.yml` ldflags block confirmed to inject `-X main.version={{.Version}}` on every
  release build, which is why real releases are unaffected.
