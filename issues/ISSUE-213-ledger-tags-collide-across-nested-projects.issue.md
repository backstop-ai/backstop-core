---
title: "Ledger Tags Diverge From File Numbers: Nested Projects and Leaked Reservations"
schema_version: issue/v1

issue:
  id: ISSUE-213
  title: "Ledger Tags Diverge From File Numbers: Nested Projects and Leaked Reservations"
  type: bug
  status: open
  created: "2026-10-12"

complexity:
  scope: contained
  uncertainty: known
  risk: moderate
---

# Ledger Tags Diverge From File Numbers: Nested Projects and Leaked Reservations

## Problem

When one git repository contains two backstop projects (each with its own `backstop.yml` and
artifact root, for example a root project and a nested subdirectory project), `backstop artifact
new` has two ID allocators that scope differently, and they disagree:

- **Tag allocator** (`GitTagResolver.Resolve`, `pkg/scaffold/idresolver.go:83`). Lists
  `backstop/<type>/*` tags with `git tag -l` (`ListTags`, `pkg/scaffold/git_executor_real.go:12`),
  takes `max(numeric suffix)+1`, creates the annotated tag `backstop/<type>/NNN`
  (`idresolver.go:124`) and pushes it. Git tags are repository-wide and the tag name carries no
  project qualifier, so every backstop project in the repository shares one counter per type.
  When this path succeeds, the returned number is used directly as the file number.
- **Local-scan allocator** (`LocalScanResolver.Resolve`, `idresolver.go:171`). Reads the
  resolved artifact root's directory for that type (`opts.Root.Dir(...)`, `idresolver.go:239`;
  root resolved per project at `cmd/backstop/artifact_new.go:89-92`) and takes
  `max(filename number)+1`. This is per project corpus, and it creates no tag.

`ResolveID` (`idresolver.go:216`) runs the tag allocator first and falls back to the local scan on
any `*FallbackError`, including a non-conflict push failure (`idresolver.go:160-161`). The tag is
created locally before the push (`idresolver.go:133`), and nothing deletes it when the push fails
(no `git tag -d` / delete path exists in `pkg/` or `cmd/`). So a fallback allocation both leaves
behind a reserved-looking tag AND takes its file number from a different counter than the tag
used. In a repository with two projects this makes the file numbers and the tag numbers diverge,
and the tag ledger stops describing either project's files.

### Reproduction

Scratch directory, `git init`, no remote (this is the failing mode: `git fetch --tags` succeeds
with no remote configured on git 2.39, so the tag path proceeds, but `git push origin <tag>`
fails with a non-conflict error, which triggers the fallback after the tag already exists
locally):

```
mkdir /tmp/nest-repro && cd /tmp/nest-repro
git init -q && git config user.email a@b.c && git config user.name t
echo "version: 1" > backstop.yml
mkdir sub && cp backstop.yml sub/
git add -A && git commit -qm init
backstop artifact new bundle --slug root-one                       # from the repo root
(cd sub && backstop artifact new bundle --slug nested-one)
(cd sub && backstop artifact new bundle --slug nested-two)
git tag -l 'backstop/*'
```

Observed:

```
Created /tmp/nest-repro/bundles/BUNDLE-001-root-one.bundle.md (ID: 001)
Created /tmp/nest-repro/sub/bundles/BUNDLE-001-nested-one.bundle.md (ID: 001)
Created /tmp/nest-repro/sub/bundles/BUNDLE-002-nested-two.bundle.md (ID: 002)
backstop/bundle/001
backstop/bundle/002
backstop/bundle/003
```

The nested project's first bundle is file `BUNDLE-001` while tag `backstop/bundle/002` was reserved
for it; its second is file `BUNDLE-002` while the tag is `003`. Tags are shared across both
projects; file numbers are per corpus.

With a working remote (a local bare repository added with `git remote add origin
/tmp/nest-remote.git`), the tag path wins and the file number equals the tag number, so the next
bundles in this repository were `BUNDLE-004` (root) and `BUNDLE-005` (nested): tag and file now
agree, but each project's sequence has gaps (`001, 004` and `001, 002, 005`) because the counter
is shared, and the numbers already burned by the earlier fallback allocations are skipped.

## Impact

- File numbers and ledger tags disagree for any second project in a repository. The reservation
  ledger is the evidence that "this number was issued", and it no longer maps to the artifact.
- Mixed modes produce confusing numbering: a project that allocated by local scan (fallback) and
  later allocates through the tag path jumps ahead by however many tags the other project
  consumed; a project that falls back reuses numbers that tags already claim for other
  artifacts, so a tag-based audit ("tag NNN names bundle NNN") is wrong in both directions.
- The leaked tag is a distinct mechanism from ISSUE-090. ISSUE-090 is about fallbacks that
  create NO tag (fetch failure, not a git repository, no git), so tag max and file max drift
  apart. Here the fallback happens AFTER a tag was created (fetch succeeds, push fails), and in
  a single project with no remote that leaked tag is currently what keeps tag max equal to file
  max. Nesting breaks that equality because two corpora share one tag counter.
- Real network-offline use is unaffected by the leak: `git fetch --tags` fails before any tag is
  created. The leak occurs only when fetch succeeds and push fails: no remote configured, push
  auth or permission errors, or server-side hooks.
- A leaked local `backstop/<type>/NNN` that differs from a remote tag of the same name makes
  `git fetch --tags` fail with `would clobber existing tag` (reproduced, exit 1). From then on
  every `artifact new` in that clone hits `FallbackError` at `idresolver.go:88-90` and silently
  allocates by local scan with no reservation, for all artifact types.
- The conflict-retry path leaks the same way: each `TagConflictError` leaves its losing local
  tag behind, so retries accumulate stray local tags.
- Classification gap: `PushTag` (`git_executor_real.go:35`) treats any output containing
  "rejected" as a conflict. Non-conflict rejections (`[remote rejected] (pre-receive hook
  declined)`, protected-tag errors) contain "rejected", so they are retried as conflicts, leak up
  to 4 tags, and end in `RetriesExhaustedError` rather than falling back.
- `git tag -l 'backstop/bundle/*'` matches across `/` (the pattern is a glob, `*` crosses path
  separators), so any namespacing scheme that nests projects under the type prefix, or a type
  whose name is a prefix of another path, must be parsed carefully (`idresolver.go:105-109`
  takes the last `/` segment).

## Root cause

Two allocators with different scopes and no shared key. The tag namespace
(`backstop/<type>/NNN`) is keyed by artifact type only, while the file namespace is keyed by
artifact type AND the project's artifact root. Nothing ties a tag to the project that requested
it, and the fallback path abandons a tag it already created.

## Direction

(Options, not decided.)

1. Namespace the tag by project, e.g. `backstop/<project-key>/<type>/NNN`. The key must be unique
   within the repository: `project:` (`pkg/config/config.go:27`) is optional and nested configs
   are often copies (the repro's `backstop.yml` has none), so either use the project path
   relative to the repository top level, or require a unique non-empty `project:` and fail
   otherwise. This changes the tag format contract in SPEC-007 REQ-003
   (`backstop/<type>/<number>`), so it needs a spec amendment, not only an issue-to-plan track.
   Only `pkg/scaffold/idresolver.go` consumes these tags in code (the rest is documentation).
   Needs a compat story for existing `backstop/<type>/NNN` tags and parser updates
   (`idresolver.go:96-117`).
2. Keep one repository-wide counter and make it authoritative for every project: the tag path
   takes `max(tags, disk)` and the fallback reconciles against the tag max (this is the
   ISSUE-090-style reconciliation). SPEC-007 REQ-003 says gaps from unused reservations are
   acceptable and must not be filled, so per-project gaps under this option are spec-compliant.
3. Refuse more than one backstop project per repository with a clear error from `artifact new`
   (detect an ancestor or descendant `backstop.yml` in the same git repository). Smallest
   change, but removes a layout some consumers use.

Leaked-tag fix, with a hard dependency: deleting the local tag when the tag path falls back is
NOT safe on its own. Simulated: two no-remote bundles (`aa`, `bb`) with their tags deleted, then a
remote added, then `artifact new bundle --slug cc` created `BUNDLE-001-cc` next to
`BUNDLE-001-aa` and pushed `backstop/bundle/001` with exit 0 - a silent duplicate ID that the
file-exists check misses because the slugs differ. So the tag deletion must ship together with
ISSUE-090-style reconciliation (the tag path computes `max(tags, disk)`); otherwise it
regresses. The leak fix and its tests must cover all of: the push-failure fallback, the
conflict-retry path (each losing tag), the fetch-clobber outcome above, and the
"rejected"-classification gap.

Regression test: a git fixture with a root and a nested `backstop.yml`, asserting that for each
allocation the tag number, the file number and the owning project agree (or, under option 3,
that the second project is refused).

## Notes

- Related: ISSUE-090 (tag/disk reconciliation on the fallback path), ISSUE-167 (id collision on
  the fallback path).
- Tag push behavior with no remote: `FetchTags` (`git fetch --tags`) returned success in the
  scratch repo, so the `FetchTags` failure branch (`idresolver.go:88-90`) does not trigger; the
  failure surfaces at `PushTag` (`git push origin <tag>`), after the tag exists locally.
