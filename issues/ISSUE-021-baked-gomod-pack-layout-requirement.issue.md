---
title: "Baked Gomod Pack Layout Requirement"
schema_version: issue/v1

issue:
  id: ISSUE-021
  title: "Baked Gomod Pack Layout Requirement"
  type: bug
  status: open
  created: "2026-06-21"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# ISSUE-021: Baked Gomod Pack Layout Requirement

## Problem

`pkg/pack/validate_manifest.go`'s `ExpectedLayout(m, base)` unconditionally declares
`go.mod` as part of the expected layout for **every** pack, regardless of the pack's
declared engines:

```go
// validate_manifest.go:59-61
add("pack.yml")
add("go.mod")
add("fixtures/rules/")
```

This is a baked Go-module assumption on a spine that must otherwise be
language-neutral: a pack whose engines are semgrep, ast-grep, a shell script, or any
non-Go tool has no reason to carry a `go.mod`, so listing it as universally "expected"
bakes backstop's own implementation language into pack-layout policy. The
`backstop/self` dogfood pack's `no-baked-language-token` rule
(`.backstop/packs/backstop/self/rules/no-baked.yml`, pattern includes the literal
`go\.mod`) flags this exact line — it is the last open finding in the thin-executor
eradication backlog (DIR-014); every other cluster (A–C, E–G per ISSUE-018/019/020/027)
has landed.

### `ExpectedLayout` is advisory, not enforced — confirmed by callgraph

Before deciding remove-vs-conditional, I traced every caller of `ExpectedLayout`:

```
$ grep -rn "ExpectedLayout" --include="*.go" .
pkg/pack/validate_manifest.go:45   (definition + doc comment)
pkg/pack/validate_manifest.go:48   (func ExpectedLayout(...))
pkg/pack/validate_manifest_test.go:92   (test-only call)
pkg/pack/validate_layout_test.go:20,27,39,52,62,69   (test-only calls)
```

There is **no production caller** — `ValidateManifest` (the function that actually
gates pack validation and returns `[]ValidationError`) never calls `ExpectedLayout`,
and no CLI command (`pack validate`, `pack add`, `pack new`, the gate's pack-load path)
reads its return value to check an installed pack's directory against it. The only
consumers are `validate_layout_test.go` and `validate_manifest_test.go`, both in the
`pkg/pack` test suite, and one of those tests — `TestExpectedLayout_GoModAlways`
(`validate_layout_test.go:26-30`) — directly asserts the baked behavior:

```go
func TestExpectedLayout_GoModAlways(t *testing.T) {
	layout := pack.ExpectedLayout(makeMinimalManifest(), baseTestRegistry())
	if !containsPath(layout, "go.mod") {
		t.Fatalf("expected go.mod in %#v", layout)
	}
}
```

This means: **removing `go.mod` from the expected set changes no validation behavior
for any existing pack** — nothing downstream rejects a pack for missing or having
`go.mod` today. The only things that must change alongside the fix are this test
(delete or repurpose it) and any doc/scaffold prose that still implies a universal
`go.mod` requirement.

### Do any current packs genuinely need a `go.mod`?

Checked the installed first-party packs (`.backstop/packs/backstop/self`,
plus the go-standards / go-toolchain packs referenced elsewhere in this repo's
history). Packs whose engines shell out to `go build`/`go test`/`go vet` as their
*toolchain* do need a real Go module in the target project being gated — but that
module lives in the **project being checked**, not in the pack's own directory. A
pack's own `go.mod` (inside the pack bundle itself) would only matter if the pack
ships Go source that itself needs modularizing (e.g. a Go-based sandbox-validator
binary) — a pack-specific need, not a universal one. No currently installed pack's
`pack.yml` declares or depends on a go.mod living in its own pack directory; the
literal is vestigial from an earlier all-Go-packs assumption that predates the
packs-only pivot (2026-06-16).

### Are the other `ExpectedLayout` entries also baked?

Audited every other `add(...)` call in the function:

| Entry | Baked? | Why |
|---|---|---|
| `pack.yml` | No | Every pack manifest is named `pack.yml` by the pack format itself — this is backstop's own artifact convention, not a foreign language/tool literal. |
| `go.mod` | **Yes** | Flagged above — a Go-specific project-manifest literal (the same class of literal the `no-baked-language-token` rule's regex also matches for `package.json`, `Cargo.toml`, `requirements.txt`, etc.). |
| `fixtures/rules/` | No | Generic pack-content directory convention, language-neutral. |
| `scaffolds/` (when `Archetype == "code"`) | No | Generic archetype-driven directory, language-neutral. |
| `rules/` (when a rule's resolved `InputMode` is rule-fed) | No | Derived from the rule's resolved engine binding `InputMode`, not a hardcoded engine-name switch (this is the SPEC-035 REQ-006c/CLM-025 pattern already applied correctly here). |
| `validators/` (when `InputMode == none` + `ScopeKind == FileArgs`) | No | Same derivation pattern as `rules/` — resolved from binding data, not a literal. |

So `go.mod` is the sole baked entry; everything else in `ExpectedLayout` already
follows the derive-from-pack-data pattern the fix should extend to this one line.

## Solution

Remove the `add("go.mod")` call from `ExpectedLayout` (`validate_manifest.go:60`).
Since nothing downstream enforces the returned layout against an installed pack's
actual directory contents today, this is a pure deletion with no behavior change to
any existing pack's validation outcome.

If a future pack genuinely needs to ship its own `go.mod` (e.g. a pack that bundles a
Go-based sandbox-validator binary), that expectation should be derived from
pack-declared data at the point the need is real — e.g. keyed off an engine binding
that indicates "this pack ships compiled Go tooling of its own" — rather than
reintroduced as a universal literal. This issue does not need to build that
conditional path preemptively (no pack today has this need); it only needs to remove
the incorrect universal assumption. Flagging the option here so a future issue isn't
tempted to just paste `go.mod` back in wholesale.

**Suggested pass order:**

1. Delete `add("go.mod")` at `validate_manifest.go:60`.
2. Delete or repurpose `TestExpectedLayout_GoModAlways`
   (`validate_layout_test.go:26-30`) — it currently asserts the baked behavior this
   issue removes. If kept, invert it to `TestExpectedLayout_NoUniversalGoMod` asserting
   `go.mod` is **absent** from the default layout.
3. Grep the repo for any doc/scaffold prose (`pack new` scaffolding help text, pack
   authoring docs) that still describes `go.mod` as a required pack file, and correct
   it if found.
4. Run the `backstop/self` dogfood pack against `pkg/pack/validate_manifest.go` and
   confirm the `no-baked-language-token` finding on this file is gone.

## Acceptance

- The `"go.mod"` string literal is gone from `validate_manifest.go`.
- The `backstop/self` `no-baked-language-token` finding on
  `pkg/pack/validate_manifest.go` is cleared (0 findings on this file for this rule).
- `go test ./pkg/pack/...` passes with the updated/repurposed test.
- Pack validation behavior for both Go-toolchain packs and non-Go packs (semgrep,
  ast-grep, shell-script packs) is unchanged, since `ExpectedLayout`'s return value was
  never enforced against a real pack directory to begin with.

### Backlog closure note

This is DIR-014's last open self-finding. Once this lands (and alongside ISSUE-033's
already-suppressed `plan.go` neutral-spine exemption), the `backstop/self` dogfood pack
goes fully GREEN — closing the thin-executor eradication backlog that began with the
2026-06-20 audit.

## References

- `pkg/pack/validate_manifest.go:45-101` — `ExpectedLayout`, the function under fix
- `pkg/pack/validate_manifest.go:60` — the baked `add("go.mod")` literal
- `pkg/pack/validate_layout_test.go:26-30` — `TestExpectedLayout_GoModAlways`, the test
  that currently asserts the baked behavior
- `.backstop/packs/backstop/self/rules/no-baked.yml:43-53` — `no-baked-language-token`
  rule; its regex matches the literal `go\.mod` (and sibling project-manifest tokens
  `package\.json`, `Cargo\.toml`, `requirements\.txt`, `pom\.xml`, `build\.gradle`)
  anywhere in Go source
- DIR-014 — thin-executor eradication backlog (parent); this issue is its last open
  self-finding
- ISSUE-018 — delivered; clusters B/F (vestigial in-process semgrep executor + legacy
  standards validator) of the same backlog
- ISSUE-019 — cluster E (`pkg/packval/` de-Go convergence), sibling eradication issue
- ISSUE-027 — delivered; eradicated the baked `DefaultRegistry()` into an injected base
  engine registry, the same injection pattern `ExpectedLayout` already uses for engine
  resolution
- ISSUE-033 — the sibling suppressed finding (`pkg/validate/plan.go` `fileCategory`
  neutral-spine exemption) that closes alongside this one
- CLAUDE.md — thin-executor first principle: "a baked language/tool branch is a defect
  to eradicate, never to extend"
