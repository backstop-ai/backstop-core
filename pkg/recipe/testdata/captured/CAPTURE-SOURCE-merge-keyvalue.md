# Capture source — merge matrix, key/value formats (TASK-003, CLM-007 / CLM-008)

One fragment per capture task (never a shared appended file). This fragment covers
ONLY the toml + .env half of the merge-format matrix. A reader globs
`CAPTURE-SOURCE-*.md`.

`config.toml` and `dotenv.env` are byte-for-byte copies of real files in real
projects — verified by sha256 against the source at capture time. The `*.fragment.*`
files are NOT captures: a merge fragment is authored by the recipe, so it is
authored here too, sized to make a shallow overwrite falsifiable.

| Fixture | Captured from | Project / commit | sha256 |
|---|---|---|---|
| `merge/config.toml` | `fly.toml` (repo root) | `~/src/projects/slotly` @ `35a7212`; file last modified in `fef1dec` (2025-10-20), clean in the worktree at capture | `facdf19dee01ad36410f1398dba8cb441f83b43d4282f1a687c5f20017ac9ac9` |
| `merge/config.fragment.toml` | authored (recipe-side fragment, not a capture) | — | — |
| `merge/dotenv.env` | `.env.example` (repo root) | `~/src/projects/bclabs-portal` @ `4712f29`; file last modified in `dcb8cc9` (2026-07-22), clean in the worktree at capture | `8084b55b80494f5944f735df32a2d0a87b2479549a16b41ad882da52d999dd28` |
| `merge/dotenv.fragment.env` | authored (recipe-side fragment, not a capture) | — | — |

Captured 2026-07-25.

## Filename neutrality

Same rule as the structured half: the CONTENT is the capture, the NAME is not.
`fly.toml` and `.env.example` are stored under neutral fixture names so the Go test
that names the path does not plant a platform-specific token in core source.

## `config.toml` — what the deep-merge exercises

The captured target carries genuine nested tables (`[http_service.concurrency]`,
`[checks.health]`, `[services.concurrency]`) and arrays of tables
(`[[services]]`, `[[services.ports]]`).

`config.fragment.toml` is built so a shallow overwrite is falsifiable:

- **Nested-table addition** — the fragment sets exactly one key inside an existing
  nested table: `[http_service.concurrency] soft_limit = 24`. Target has
  `{type = "connections", hard_limit = 25, soft_limit = 20}`. A deep merge yields
  `soft_limit = 24` with `type` and `hard_limit` INTACT; a shallow overwrite of the
  nested table drops both, and the test catches it.
- **New top-level keys** — `kill_timeout` (scalar) and `[metrics]` (table), neither
  present in the target.

The fragment touches nothing in `[[services]]`, so array-of-tables handling is
untouched here rather than silently asserted.

## `dotenv.env` — what the merge exercises, and the redaction question

- **Commented lines are pinned**: 18 full-line `#` comments (section banners).
- **Inline trailing comments are present on nearly every assignment**
  (`PORTAL_DEFAULT_BRANCH=main         # [P] default branch`). This is what a real
  `.env` in a real project looks like; it is deliberately NOT sanded off. The merge
  implementation has to take a position on whether the trailing `#` text is part of
  the value or a comment, and the test pins whichever position it takes.
- **Override key**: `PORTAL_DEFAULT_BRANCH` exists in the target with a real
  non-empty value (`main`); the fragment sets it to `release`, so the override is
  observable rather than vacuous.
- **New key**: `PORTAL_INGEST_TIMEOUT_MS` is absent from the target.

**Redaction: none was required, and none was performed.** The captured file is the
committed `.env.example` — the project's environment *surface*, not the gitignored
`.env.local` that holds live values. Every key the file marks `[S]` (secret) is
value-less in the capture (`CRON_SECRET=`, `SUPABASE_SERVICE_ROLE_KEY=`,
`PORTAL_GITHUB_TOKEN=`, …). The only non-empty values are public constants
(`PORTAL_OIDC_ISSUER`, `POSTHOG_HOST`, `PORTAL_DEFAULT_BRANCH=main`). The bytes are
therefore the real file with nothing altered — capture fidelity intact, no secret
carried into the corpus.
