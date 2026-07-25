# Capture source — create-op payload + unsupported merge target (TASK-001)

Every fixture below was COPIED BYTE-FOR-BYTE out of a real project. None was
hand-authored, and none was derived from the applier's own output. Fixture
filenames are deliberately NEUTRAL (the captured bytes are the capture, the
name is not): a Go test naming the real captured filename would trip
`backstop/self` Family B2 (`no-baked-language-token`, global + blocking).

Capture origin project: **bclabs-portal** (`~/src/projects/bclabs-portal`,
`bclabs-ai/bclabs-portal`) — the go-live capture named as the standing recipe
fixture in BUNDLE-015 / SPEC-054 References. Repo HEAD at capture time:
`4712f2964c40d0a5d90358cd5c7828ff6d59efa8` (2026-07-25); both source files
were clean (uncommitted-change-free) at that HEAD. Captured 2026-07-25.

| Fixture (this task owns) | Captured AS (real filename) | Source commit | sha256 of the captured bytes |
|---|---|---|---|
| `create/payload.expected.json` | `vercel.json` (repo root) | `0393f42` (2026-07-18) | `978e7bc19e83d15b707743e299bec3356b964ba5990d2110d7d665b60f2d551a` |
| `create/payload.json.tmpl` | `vercel.json` (repo root), + 2 placeholders | `0393f42` (2026-07-18) | (expected-json bytes with the two values below replaced) |
| `merge/unsupported-target.md` | `README.md` (repo root) | `79d005c` (2026-07-10) | `95cc8b748cc0310f2715c0b5b03412aded4c259049889e004078d5dcd99aff57` |

## `create/payload.json.tmpl` — the templated capture

The real Vercel cron-declaration file that the portal's go-live actually
created (documented in `docs/go-live-scaffolding.md` §1: "Vercel Cron: exactly
one hourly job `/api/reconcile`, `0 * * * *`"). Exactly TWO real values were
replaced with `{{ param }}` placeholders; every other byte is the capture:

- `"/api/reconcile"` → `"{{ cron_path }}"`
- `"0 * * * *"` → `"{{ cron_schedule }}"`

## `create/payload.expected.json` — the expected materialization

The UNMODIFIED captured file. It is the capture with the two placeholders
resolved back to their real captured values — NOT a re-derivation of what the
applier emits. A test supplying `cron_path=/api/reconcile` and
`cron_schedule=0 * * * *` must reproduce these bytes exactly; an applier that
skipped substitution leaves `{{ cron_path }}` in the output and fails.

## `merge/unsupported-target.md` — the unstructured merge target

A real, unstructured (non json/yaml/toml/.env) Markdown file, copied verbatim,
used as a `merge` target so CLM-009's fail-loud is driven against a genuine
unstructured file rather than an empty stub. Content is non-secret prose.

## Re-verification

    shasum -a 256 pkg/recipe/testdata/captured/create/payload.expected.json \
                  pkg/recipe/testdata/captured/merge/unsupported-target.md

against the source files at the commits above.
