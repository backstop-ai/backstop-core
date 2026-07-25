# Capture source — structured merge fixtures (json + yaml)

Owned by PLAN-SPEC-054 TASK-002. One line per fixture this task owns. Sibling
capture tasks write their own `CAPTURE-SOURCE-<area>.md` fragment; there is no
shared appended file. A reader globs the fragments.

Source project for every capture below: **bclabs-portal**
(`~/src/projects/bclabs-portal`, the BUNDLE-015 standing recipe fixture / go-live
capture), repo HEAD `4712f296` — captured 2026-07-25.

## Captured targets (real files, byte-for-byte)

| Fixture | Captured AS (true original filename) | Origin commit | sha256 |
|---|---|---|---|
| `merge/target.json` | `tsconfig.json` (repo root) | `9915774` (2026-07-22, clean at HEAD) | `a769614822719d87d1039cb529af80191ac5c90374cff8a1bd1e068abe7d802b` |
| `merge/target.yml` | `.github/workflows/backstop-ingest.yml` | `21a5d01` (2026-07-23, clean at HEAD) | `71e8001c7bf0896583b6d2406386ca87bb5b9add4dd53c0273de1fb3bda4e78b` |

Both files were copied verbatim (`cp`) and the sha256 above was verified equal to
the source file's — the BYTES are the capture. Neither was edited, trimmed, or
reformatted to suit a test.

**Why the neutral names.** The true filenames are both tokens matched by
backstop/self rule Family B2 (`no-baked-language-token`), so a Go test that named
either path in source would go RED. The fixtures therefore keep the captured
content under neutral names, and the origin is recorded here instead. (B2 is
`languages: [go]`, so the true filenames are safe to write in this Markdown.)

**Deep-merge falsifiability of each target.**
- `target.json` — nested maps AND arrays: `compilerOptions` is a 14-key nested
  map holding two arrays (`lib`, and `plugins`, whose elements are themselves
  maps), beside the top-level `include` / `exclude` arrays. A shallow overwrite
  of `compilerOptions` destroys 14 siblings including both nested arrays, so
  shallow-vs-deep is falsifiable, and array-valued keys are present at two
  depths.
- `target.yml` — nested maps AND arrays, including maps nested inside array
  elements (`jobs.<id>.steps[].with`, `on.push.branches`, `permissions`,
  `jobs.<id>.env`). Three sibling jobs, so both array handling and deep map
  merging are exercised and falsifiable.
  *Sharp edge preserved by the capture:* the target's top-level `on:` key is the
  YAML 1.1 boolean trap — a YAML 1.1 codec (e.g. PyYAML) reads it as `true`,
  while a YAML 1.2 codec (gopkg.in/yaml.v3) keeps it a string. The fixture is
  kept verbatim so the merged output pins whichever the merger's codec does,
  instead of hiding the question behind a sanitized fixture.

## Authored fragments (NOT captures)

Fragments are recipe declarations, not captured project files — a recipe authors
its fragment. Each is the fragment a real recipe would deep-merge into the target
beside it, and each satisfies the plan's shape: one nested key added inside an
EXISTING object plus one NEW top-level key.

| Fixture | Nested addition (existing object) | New top-level key | Falsifies |
|---|---|---|---|
| `merge/fragment.json` | `compilerOptions.noUncheckedIndexedAccess` | `extends` | shallow merge drops the target's 14 other compiler options, including the `lib` and `plugins` arrays |
| `merge/fragment.yml` | `jobs.gate-and-ingest.timeout-minutes` | `concurrency` | shallow merge drops `runs-on` + `steps` from that job, and the two sibling jobs |

No expected merged result is authored here. Per the plan, expected outputs are
derived later from the two captured/authored inputs — never by copying whatever
the merger happens to emit.
