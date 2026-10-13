---
title: "Typescript Toolchain Fixtures Classified As Source"
schema_version: issue/v1

issue:
  id: ISSUE-222
  title: "Typescript Toolchain Fixtures Classified As Source"
  type: bug
  status: open
  created: "2026-10-13"

complexity:
  scope: contained
  uncertainty: known
  risk: safe
---

# Typescript Toolchain Fixtures Classified As Source

## Problem

Plain JavaScript test fixtures under a `tests/` directory fail the gate twice under
`backstop-ai/typescript-toolchain@1.3.1`:

- they are classified as measurable source, so they need coverage they can never get; and
- the pack's shipped ESLint config lints them with no Node globals and with ESM-only rules.

Observed 2026-10-13 in a TypeScript consumer repository's bare gate (gate built from backstop-core
`dacfd7c`). The repository adds CommonJS fixture scripts that a verifier runs as child processes:
`tests/fixtures/intake/<fixture>/checks/*.cjs` and `tests/fixtures/intake/<fixture>/draft.cjs`. The
gate reported:

- `coverage_unmeasured` for each fixture `.cjs` (`coverage_threshold` failed); and
- 17 `pack_engines` (`ts-lint`) findings on the same files: `no-undef` for `require`, `__dirname`,
  `console`, `process` and `module`, and `@typescript-eslint/no-require-imports` for
  `require(path.join(__dirname, '..', 'draft.cjs'))`.

### (a) Classification (verified in the pack's `pack.yml`)

- `classification.source` includes `"**/*.js"`, `"**/*.mjs"`, `"**/*.cjs"` (and the TS extensions).
- `classification.test`, which the pack defines as "not measurable application source", covers only
  test-named files (`*.test.*`, `*.spec.*`), `**/__tests__/**`, `*.config.*`, `**/testing/**`,
  `**/testsupport/**`, `**/acceptance/**`, `**/e2e/**` and the Next.js route, page and layout
  entrypoints. Nothing matches a non-test-named file under `tests/` or under a `fixtures/` directory.
- Core's `SourceClassifier.IsMeasurableSource` (`pkg/gate/classification.go:43`) is
  `matches source && !matches test`, so `tests/fixtures/**/*.cjs` counts as measurable source.
- These scripts run only in child processes and are never imported in the Vitest process, so v8 never
  measures them. The result is no coverage record and the core `coverage_unmeasured` guard
  (`pkg/gate/step_coverage.go`, ~lines 158-172). This is the same no-record path as ISSUE-221.
- A side effect: a test helper like `tests/helpers.ts` is also classified as source today. It passes
  only because tests happen to load it.

### (b) ESLint config (verified in the pack's `eslint.config.mjs` and the consumer's installed `typescript-eslint` 8.71.1 / ESLint 10.12.0)

- The config is just `ignores`, `js.configs.recommended` and `...tseslint.configs.recommended`. It has
  no `languageOptions.globals` block for any file type.
- `js.configs.recommended` turns on `no-undef`. typescript-eslint's `eslint-recommended` override
  turns `no-undef` off only for `**/*.ts`, `**/*.tsx`, `**/*.mts` and `**/*.cts`
  (`@typescript-eslint/eslint-plugin/dist/configs/eslint-recommended-raw.js`). For `.js`, `.cjs` and
  `.mjs` files, `no-undef` stays on and no Node globals are declared.
- typescript-eslint's `base` config sets `languageOptions.sourceType: 'module'` with no `files` key
  (`dist/configs/flat/base.js:12`). That overrides ESLint's built-in `commonjs` default for `**/*.cjs`,
  so `require`, `module` and `exports` are not defined there either.
- `@typescript-eslint/no-require-imports: 'error'` (`dist/configs/flat/recommended.js:34`) also has no
  `files` scoping, so it fires on `.cjs` files, where `require` is the module system.

## Impact

- A consumer cannot add a plain CommonJS fixture or helper script without red-gating twice. A
  fixture's whole purpose is to run outside the test process, so both failures are false positives.
- The only workarounds are waivers, suppressions or per-file disables, and the backstop rules forbid
  using those to turn a gate green.
- Any `.js`, `.mjs` or `.cjs` Node script the pack lints, such as a tooling script outside `*.config.*`,
  hits the same `no-undef` noise.

## Root cause

1. The pack's `classification.test` bucket has no entry for test-support directories that hold
   non-test-named files (`tests/`, `fixtures/`). go-toolchain has the equivalent `**/testdata/**`.
2. The pack's ESLint config was written for TypeScript, where tsc handles undefined names. It never
   declares a runtime environment for plain JavaScript, and it leaves the typescript-eslint `base`
   `sourceType: 'module'` in force on `.cjs`.

## Direction

(Direction, not a plan.) Every change belongs in the `backstop-ai/typescript-toolchain` pack
repository (`https://github.com/backstop-ai/typescript-toolchain`, currently 1.3.1 at tag `v1.3.1`). No
core change is needed.

1. **Classification.** Add to `classification.test`:
   - `"**/fixtures/**"` and `"**/__fixtures__/**"` (the minimum); and
   - `"**/tests/**"`, which mirrors `**/__tests__/**` and makes test helpers under `tests/` non-source.

   Check before shipping that putting these paths in the `test` bucket does not make the
   test-verification or substantiveness consumers (`IsTestFile`) treat fixture scripts as test files
   that lack assertions.
2. **ESLint.** After the presets in `eslint.config.mjs`, add blocks like these:

   ```js
   // Plain JS runs on Node: declare its globals (tsc covers TS files).
   { files: ['**/*.js', '**/*.jsx', '**/*.mjs', '**/*.cjs'],
     languageOptions: { globals: NODE_GLOBALS } },
   // CommonJS: require/module/exports are the module system.
   { files: ['**/*.cjs'],
     languageOptions: { sourceType: 'commonjs',
       globals: { require: 'readonly', module: 'writable', exports: 'writable',
                  __dirname: 'readonly', __filename: 'readonly' } },
     rules: { '@typescript-eslint/no-require-imports': 'off' } },
   ```

   `NODE_GLOBALS` should be an inline literal (`console`, `process`, `Buffer`, `globalThis`, the
   timers, `URL`, `URLSearchParams`, `structuredClone`, `fetch`, and so on). Do not import the `globals`
   package. The pack's onboarding contract makes consumers install only `eslint`, `@eslint/js`,
   `typescript-eslint` and `typescript`, and under pnpm's strict layout `globals` is not resolvable
   from the consumer root. Adding it to the contract would turn a patch into a breaking change.
3. **Release.** Bump `pack.yml` to `1.3.2`, add `fixtures/` cases for `backstop pack test` (a
   `tests/fixtures/x.cjs` that uses `require`/`__dirname` must produce no lint finding and no coverage
   requirement), then tag `v1.3.2`. Consumers run `backstop pack update` to refresh `backstop.lock`.
   Ship this together with, or before, the ISSUE-221 `--coverage.include` change. Otherwise fixtures
   classified as source turn from `coverage_unmeasured` into 0% `coverage_threshold` failures once
   `include` makes v8 report unloaded files.

## Notes

- The installed pack in the consumer is byte-identical to the 1.3.1 working copy (HEAD `b0cfa10`,
  "v1.3.1: coverage summaries survive a failing suite").
- Related: ISSUE-221 (types-only files, same `coverage_unmeasured` no-record path, same pack
  release); ISSUE-045 (earlier unmeasurable-file false positives).
