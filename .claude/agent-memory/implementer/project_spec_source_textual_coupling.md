---
name: spec-source-textual-coupling
description: SPEC-054's contract signature strings are compared TEXTUALLY against pkg/recipe/apply.go by a Go test, so spec and source must move in the SAME change
metadata:
  type: project
---

`pkg/recipe/contract_signature_test.go` (ISSUE-080 CLM-011) reads SPEC-054's
declared `signature:` strings out of the YAML frontmatter and compares them
TEXTUALLY — after normalizing `;` and whitespace runs — against the type
declarations parsed out of `pkg/recipe/apply.go` with go/parser + go/printer.
It covers ALL SEVEN type entries the spec declares for that ONE file — ApplyMode,
ApplyOptions, TransformDispatch, WaiverReader, DivergenceVerdict, ApplyResult,
PreservedDivergence — plus a COMPLETENESS check that fails if the spec later adds a
type entry for that file which the test does not compare. Field NAMES, field TYPES
and field ORDER are all load-bearing on both sides: a pure field-ORDER swap with
identical names and types is caught (verified against a mutated copy).

**Why:** the contracts gate dimension compiles every `kind: type` entry to the
existence-only ast-grep pattern `type <Name> $$$`, which a STALE and a CORRECTED
signature both satisfy — so no gate dimension can go red on signature drift, and
`artifact validate` passes regardless. This test is the only mechanical catcher.

**How to apply:** editing either side alone fails `go test ./pkg/recipe/...`, not
merely the gate — and a frontmatter-only editor will not discover why. Move the
spec and apply.go in the SAME change. Never resolve a red here by loosening the
comparison to a substring/HasSuffix check; that is the lossy form the test exists
to eliminate. Fix whichever side is actually wrong. Since artifacts are unwritable
to an implementer, the spec half routes to `spec-author` — see
[[agent-guard-testdata]].

**Why only that one file:** the guard stops at the apply source deliberately. The
spec writes its `signature:` strings WITHOUT struct tags, while go/printer emits
them, so the tag-bearing types declared in the manifest and adoption sources
(RecipeManifest, Op, ParamSpec, AdoptionEntry, AdoptionRecord) read as mismatches
that are pure formatting, not drift. Extending the guard past this file first
requires deciding whether the spec's signature convention includes struct tags —
a convention call, not something a test should settle by quietly stripping them.

**DEFERRED, so it is not silently dropped:** the guard's failure mode lands on
SPEC-AUTHORS, not on Go authors — adding a `kind: type` entry for the apply source
to SPEC-054's frontmatter FAILS `go test ./pkg/recipe/` until someone edits
`contractedTypes()`, while `artifact validate` stays green throughout. Today that
instruction lives only in the test's own doc comment, discoverable only by someone
already reading the test. The spec-author-facing form of the warning belongs in
SPEC-054's own Sharp Edges, where a spec-author will actually look. It was
deliberately NOT added on 2026-07-26 to avoid reopening a just-committed artifact
(d222975) for a documentation-only change, and because the guard fails LOUD AND BY
NAME — a tripped editor is told exactly which entry is unguarded, so they are stuck
for seconds, not silently wrong. FOLD IT IN at the next natural SPEC-054 touch;
ISSUE-081 (recipe authoring surface) is the likely one.

