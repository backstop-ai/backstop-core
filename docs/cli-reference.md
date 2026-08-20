# backstop CLI reference

Every command backstop ships, with its real flags and a working example.

This is the **exhaustive reference** — look things up here. If you want to be walked
through a first setup, read [getting-started.md](./getting-started.md). If you want to
know *why* backstop works the way it does — packs, gates, artifacts, waivers — read
[concepts.md](./concepts.md).

Backstop is a governance enforcement CLI. Every agent, runtime, and workflow interacts
with it by shelling out to these commands.

---

## Conventions

### Exit codes

Backstop uses three exit codes consistently, and a config error always outranks a
violation:

| Code | Meaning |
| ---- | ------- |
| `0`  | Passed / completed successfully |
| `1`  | Violations found (or broken promises) |
| `2`  | Config error — backstop could not run the check at all |

The distinction matters in CI: `1` means backstop ran and found something; `2` means
backstop could not do its job and your pipeline is measuring nothing.

### `--json`

`--json` is a global flag available on every command. It swaps the human-readable table
on stdout for a structured document. Use it whenever something other than a person is
reading the output.

```bash
backstop pack list --json
backstop doctor --json
```

### Discovering commands programmatically

`backstop commands` emits the whole command tree as JSON. That is the supported way for
an agent to learn what this binary can do — see [`commands`](#commands) below.

---

## `init`

**Take a project from nothing to a first gated run**, in one prompt-free invocation.

```
backstop init [flags]
```

A bare `backstop init` runs the full default capability set: initializes a git repository
if there is none, writes the profile-correct `backstop.yml`, scaffolds the artifact
layout, installs the packs you named, writes the canonical `.gitignore`, runs each
installed pack's declared test/build entrypoint once, delegates baseline seeding, and
finally runs the gate once and reports what it noticed.

The capability set is exactly seven names: `git`, `sdlc`, `gitignore`, `packs`,
`toolchain`, `baseline`, `observe`. Generating `backstop.yml` is deliberately *not* among
them — it is unconditional, because an init that does not write it produces nothing you
can use.

**Flags**

| Flag | Purpose |
| ---- | ------- |
| `--pack <ref>` | Install a pack by **portable git reference** (`<org>/<pack>@<version>`). Repeatable. A local filesystem path is refused. |
| `--ci <pack>:<recipe>@<version>` | Wire CI by applying a pinned recipe an installed pack declares. |
| `--scaffold <pack>:<recipe>@<version>` | Scaffold a first source file from a pinned recipe. |
| `--no-<capability>` | Skip one capability (`--no-git`, `--no-sdlc`, `--no-gitignore`, `--no-packs`, `--no-toolchain`, `--no-baseline`, `--no-observe`). |
| `--only <capability>` | Run **only** the named capability. Repeatable. May not be combined with any `--no-` flag. |

**Example**

```bash
backstop init \
  --pack backstop-ai/go-toolchain@1.8.0 \
  --pack backstop-ai/go-standards@1.2.1 \
  --ci backstop-ai/ci-workflows:github-actions-gate@0.1.2
```

**Sharp edges**

- `--pack` takes portable git references only. A local path is refused on purpose: it
  would record a machine-specific path into the `backstop.lock` you commit.
- `--ci` and `--scaffold` each take a **whole pinned recipe reference**, handed to the
  recipe machinery verbatim. Backstop constructs no part of it and holds no default.
  Omitting either is a deliberate, reported no-op — not an error.
- Init never prompts and never reads stdin, so it behaves identically with no TTY.
- Gate findings during init are **observation**, not failure. Pre-existing violations in
  a project you just started governing do not fail init; the exit code carries broken
  promises and nothing else.

---

## `doctor`

**Diagnose a backstop setup — including the conditions that make other commands refuse.**

```
backstop doctor [--check <id>]
```

Doctor is the command you run when everything else refuses. An absent, unparseable, or
gate-fatal `backstop.yml` is reported as a *check result carrying remediation*, not as a
reason to bail out — so a broken setup gets diagnosed rather than merely rejected.

A bare invocation runs every registered check. As of this build the registered check ids
are:

`config-present`, `config-loads`, `git-repository`, `packs-installed`, `build-identity`,
`toolchain-runs`, `engine-tools-present`, `artifact-layout`

**Flags**

| Flag | Purpose |
| ---- | ------- |
| `--check <id>` | Report on one registered check instead of all of them. |

**Example**

```bash
$ backstop doctor --check config-present
backstop doctor
  config-present   pass     backstop.yml is discoverable — found at /path/to/repo/backstop.yml
```

**Sharp edges**

- Exit code is `1` only when at least one check **fails**. Warnings and skips do not fail
  the run.
- An unknown `--check` id is a loud error that names every registered id, rather than an
  empty successful run.

---

## `gate`

**Run the full verification gate** — the reconciliation kill chain that orchestrates
artifact validation, code checking, test verification, test substantiveness, coverage
thresholds, contract signatures, baseline comparison, waiver resolution, and ledger
integrity. This is the primary enforcement checkpoint: if it's green, it ships.

```
backstop gate [--all | --file FILE [--file FILE]... [FILE...] | --base REV]
```

With no scoping flag, gate runs **diff-scoped**: files changed against the merge-base,
plus untracked files. That is the fast inner loop.

**Flags**

| Flag | Purpose |
| ---- | ------- |
| `--all` | Run the full project sweep. |
| `--file <path>` | Scope to explicit files. Repeatable, and trailing positional paths accumulate on top of the flag values. |
| `--base <rev>` | Scope to files changed since the merge-base with `REV`, plus untracked files. |
| `--json-out <file>` | Also write the `gate/v1` JSON envelope to a file. |

**Examples**

```bash
# inner loop — just what you touched
backstop gate

# exit gate — the whole project
backstop gate --all

# two specific files
backstop gate --file pkg/gate/policy.go --file pkg/gate/result.go

# CI, with the table in the log and the report on disk from one run
backstop gate --base "$PR_BASE_SHA" --json-out gate-report.json
```

**Sharp edges**

- **In CI, the default scope checks nothing.** A fresh checkout has a clean working tree,
  so the default diff scope resolves to zero files. Pass `--base` with the pull-request
  base sha or the push before-sha. An unresolvable `REV` is a config error (exit `2`),
  never a silent full sweep.
- `--json-out` is independent of `--json` and combines with it: `--json` decides what
  **stdout** receives, while `--json-out` only *adds* a file destination and never changes
  stdout. That is what lets one CI run produce both the human table and the
  machine-readable report.

---

## `pack`

**Enforcement content commands** — compiling, managing, and distributing enforcement packs
containing rules and code standards.

```
backstop pack <subcommand>
```

Packs are where all of backstop's language and tool knowledge lives. The CLI itself bakes
in none of it.

### `pack add`

Add an enforcement pack to the project: resolve, clone, validate, install, merge config,
and lock it.

```
backstop pack add [pack-ref] [--version <v>]
```

Accepts `org/pack-name@version` for git packs, or a local filesystem path.

```bash
backstop pack add backstop-ai/go-standards@1.2.1
backstop pack add ./packs/my-local-pack
```

`--version` overrides the version in the pack reference if you pass both.

### `pack install`

Restore packs from `backstop.lock` — clones all packs at their locked versions and
verifies content hashes.

```
backstop pack install [--cache <dir>]
```

```bash
backstop pack install
backstop pack install --cache /mnt/pack-mirror   # offline / airgapped
```

Note that install deliberately does **not** run validation or merge `tool_config`. It is
the restore path, not the adoption path.

### `pack update`

Update a pack to the latest compatible minor/patch version. Resolves the latest compatible
version, validates, runs tamper detection, then updates `backstop.yml` and `backstop.lock`
with the new exact pin.

```
backstop pack update [pack-name] [--acknowledge]
```

```bash
backstop pack update backstop-ai/go-standards
```

`--acknowledge` acknowledges tamper-detection findings and proceeds anyway. Read what it
found before you reach for it.

### `pack upgrade`

Upgrade a pack across a major version. Takes an explicit major version target, validates,
scans for new violations, generates a remediation bundle, and rewrites `backstop.yml` and
`backstop.lock`.

```
backstop pack upgrade [pack-ref@version]
```

```bash
backstop pack upgrade backstop-ai/go-standards@2.0.0
```

### `pack remove`

Remove a pack: reverts pack-contributed config settings, deletes pack files, and removes
entries from `backstop.yml`, `backstop.lock`, and provenance.

```
backstop pack remove [pack-name]
```

```bash
backstop pack remove backstop-ai/go-standards
```

### `pack relock`

Refresh a **local** pack's lock entry after editing it in place — re-reads the pack,
recomputes its content hash, and overwrites its `backstop.lock` entry. No remove-then-add.

```
backstop pack relock [path]
```

```bash
backstop pack relock ./packs/my-local-pack
```

Only local-source packs are relockable; git packs move through `pack update` / `pack
upgrade`.

**Sharp edge:** `relock` takes a **filesystem path** where its siblings (`remove`,
`update`, `upgrade`) take a **pack name**. This asymmetry is known and filed —
ISSUE-074 residual, homed under DIR-034. Guessing wrong errors loudly rather than failing
silently, but the inconsistency is real; pass a path.

### `pack list`

List installed packs with name, version, lock status, archetype, rule count, and scaffold
count.

```
backstop pack list [--json]
```

```bash
$ backstop pack list
NAME                           VERSION      LOCK STATUS  ARCHETYPE       RULES    SCAFFOLDS
backstop-ai/go-standards       1.2.1        locked       enforcement     14       0
backstop-ai/go-toolchain       1.8.0        locked       enforcement     4        0
backstop-ai/ci-workflows       0.1.2        locked       recipes         12       0
```

### `pack new`

Scaffold a new pack — a valid `pack.yml` with a declared `engines:` block and a sample rule
that passes `pack check`, `pack test`, and the gate.

```
backstop pack new --slug <name> --type <engine|mechanism|toolchain> [--language <lang>]
```

```bash
backstop pack new --slug rust-standards --type toolchain --language rust
```

`--slug` is kebab-case, 2–64 characters.

### `pack check`

Validate a pack's manifest and constraints. Runs validation phases 1, 2, 4, 5, and 6 —
the manifest and metadata checks.

```
backstop pack check [pack-dir] [--format json|text]
```

```bash
backstop pack check ./packs/my-pack
```

The pack directory defaults to the current directory if omitted.

### `pack test`

Full pack validation: all six phases, **including fixture execution in phase 3**. This is
the one that actually runs your rules against your fixtures.

```
backstop pack test [pack-dir] [--format json|text]
```

```bash
backstop pack test ./packs/my-pack
```

Use `pack check` while iterating on the manifest; use `pack test` before you tag a release.

---

## `artifact`

**Artifact lifecycle commands** — creating and validating backstop artifacts such as
specs, plans, issues, ADRs, bundles, directives, and capabilities.

```
backstop artifact <subcommand>
```

### `artifact new`

Scaffold a new artifact from a template, with an auto-assigned ID. Always create artifacts
this way — the CLI owns ID allocation, so hand-numbering drifts.

```
backstop artifact new [type] --slug <kebab-slug> [--source <ID>]
```

The valid types, and what each produces:

| Type | ID prefix | File |
| ---- | --------- | ---- |
| `spec` | `SPEC-NNN` | `.spec.md` |
| `plan` | `PLAN-NNN` | `.plan.yml` |
| `issue` | `ISSUE-NNN` | `.issue.md` |
| `adr` | `ADR-NNN` | `.adr.md` |
| `directive` | `DIR-NNN` | `.directive.md` |
| `bundle` | `BUNDLE-NNN` | `.bundle.md` |
| `capability` | `CAP-NNN` | `.capability.yml` |

```bash
backstop artifact new issue --slug gate-scope-drops-untracked
backstop artifact new plan --slug fix-gate-scope --source ISSUE-158
```

**Sharp edge:** `--source` (a backing `SPEC-NNN` or `ISSUE-NNN`) is **required** for the
`plan` type and optional elsewhere. A plan with no source has nothing to be a plan *for*.

### `artifact validate`

Validate artifact files against their schema definitions. Discovers artifacts in the
project and validates each against its type-specific schema.

```
backstop artifact validate [--all | --spec <ID> | --plan <ID> | ...]
```

**Flags**

| Flag | Purpose |
| ---- | ------- |
| `--all` | Validate everything. Takes precedence over the type flags. |
| `--spec`, `--plan`, `--issue`, `--adr`, `--bundle`, `--directive` | Validate that type. Each takes an optional ID to narrow to one artifact. |

```bash
backstop artifact validate --all
backstop artifact validate --plan PLAN-ISSUE-158
backstop artifact validate --spec        # every spec
```

Exit codes follow the standard convention: `0` all pass, `1` violations found, `2` config
error.

---

## `recipe`

**Recipe adoption commands.** A recipe is pack-declared *data* — its ops, targets,
payloads, and rewrite rules all live in the pack — so this namespace carries no knowledge
of any language, framework, or CI platform.

```
backstop recipe <subcommand>
```

### `recipe apply`

Apply one pinned recipe from an installed pack into the current project. The recipe is
resolved out of the installed pack corpus, its ops run in the order the recipe declares
them, and the adoption is recorded in the tracked project-root record.

```
backstop recipe apply <pack>:<recipe>@<version> [--param key=value]...
```

```bash
backstop recipe apply backstop-ai/ci-workflows:github-actions-gate@0.1.2 \
  --param backstop_version=v0.1.0 \
  --param default_branch=main
```

**Sharp edges**

- The reference is **always fully pinned**. There is no `latest` form.
- `--param` supplies a value for a param the recipe *declares*; it is repeatable, and any
  param you leave out falls back to the recipe's declared default. A param declared
  required with **no** default can only be supplied this way — without it the apply cannot
  resolve, so it fails rather than writing at an unresolved site.
- A recipe's transform op dispatches to the engine its pack declares, and that engine's
  tool must clear the same trusted-tool allowlist gate every pack-declared enforcement
  command clears. An un-allowlisted or wrongly-pinned tool is refused before any command
  is built.

---

## `baseline`

**Baseline cache and artifact commands** — the baseline is what gate compares against to
distinguish pre-existing findings from new ones.

```
backstop baseline <subcommand>
```

### `baseline generate`

Run gate in full-scope mode (equivalent to `--all`) and write `.backstop/baseline.json` as
`baseline/v1` JSON.

```
backstop baseline generate
```

Intended for CI baseline publication; it does not depend on the local baseline cache TTL.

### `baseline pull`

Fetch the baseline artifact from the latest successful main-branch CI run.

```
backstop baseline pull
```

Artifact lookup uses GitHub Actions runs and artifact-naming semantics, bypasses the local
TTL, and writes `.backstop/baseline.json` **atomically** — a failed pull will not corrupt
the cache you already have.

---

## `waiver`

**Inspect backstop waivers (read-only).**

```
backstop waiver <subcommand>
```

Core backstop never writes or inserts a `@waiver:` token. Authoring and re-certification
belong to the human or the runtime agent — the CLI only ever reports.

### `waiver list`

List the waivers backstop adjudicates over the current scope, grouped into **active**,
**expiring-soon**, and **unused/dangling** sets.

```
backstop waiver list [--json]
```

```bash
backstop waiver list
backstop waiver list --json
```

The dangling group is the useful one during cleanup: a waiver that no longer matches any
finding is a waiver you can delete.

---

## `version`

Print version and schema cohort information: the CLI binary version, the embedded schema
cohort identifier (derived from the set of embedded schema versions), and the Go version
the binary was built with.

```
backstop version [--json]
```

```bash
$ backstop version
backstop version dev
commit: 22c7574...
built: 2026-08-19T22:11:10Z
schema cohort: 49f08ac21d140c2844...
go version: go1.25.3
```

The schema cohort is the field to quote when an artifact validates on one machine and not
another — differing cohorts mean differing embedded schemas.

---

## `commands`

List all available commands for agent discovery. Returns a JSON array describing the full
command tree; each entry carries the command name, full path, description, and available
flags.

```
backstop commands
```

```bash
$ backstop commands | jq -r '.[].path'
artifact
artifact new
artifact validate
baseline
...
```

This is the agent discovery endpoint. An agent that needs to know what this backstop build
can do should read this rather than parse `--help` text or rely on a hardcoded list.

---

## Appendix: full command list

26 commands, exactly as reported by `backstop commands`:

```
artifact              init                  pack test
artifact new          pack                  pack update
artifact validate     pack add              pack upgrade
baseline              pack check            recipe
baseline generate     pack install          recipe apply
baseline pull         pack list             version
commands              pack new              waiver
doctor                pack relock           waiver list
gate                  pack remove
```

`completion` and `help` are Cobra built-ins and are not part of backstop's own surface.
