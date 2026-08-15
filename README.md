# backstop

An AI agent discipline framework. Backstop gates the pipeline before code is written — enforcing standards, verifying claims, and tracing lineage from intent to implementation.

Backstop is not a code quality scanner. It is a framework that assumes AI agents will fabricate, skip steps, and ignore conventions unless mechanically constrained.

## Status

🚧 Under active development. Public, but early — expect rough edges and breaking changes.

## Getting Started

Install the CLI (see the [releases page](https://github.com/backstop-ai/backstop-core/releases) for platform binaries, or build from source with `make build`), then from your project root:

```bash
backstop init
```

This initializes a git repository if one doesn't exist, writes `backstop.yml` declaring your artifact root, creates the artifact-type directories backstop expects, and writes a `.gitignore` for backstop's own generated files. It installs no packs and enforces nothing on its own — packs are what backstop actually checks, and you add them explicitly:

```bash
backstop pack add <org>/<pack>@<version>
```

Prefer a minimal footprint — no bundle/spec/plan artifact layout, just pack-declared checks? Run `backstop init --no-sdlc` instead.

Once you have a project, `backstop doctor` diagnoses common setup problems in one pass — config discovery, git presence, installed packs, the running binary's build identity, whether your packs' declared test/build entrypoints actually execute, and whether your artifacts sit where backstop expects them:

```bash
backstop doctor
```

Each check reports `pass`, `warn`, `fail`, or `skipped` (with the check that owns the skip condition named explicitly, so a chain of unmet prerequisites never reads as an unrelated failure) — never a silent gap.

Run the actual gate against your project with:

```bash
backstop gate
```

## Structure

```
cmd/backstop/    CLI entrypoint
pkg/             Go packages (gate orchestration, pack lifecycle, validation, schema loading)
artifacts/       Primitive schemas, templates, and docs (versioned)
recipes/         Recipe capability primitives
docs/            Internal codebase map (not end-user documentation yet)
```

Packs — the actual checks backstop runs — live in their own repos and install into gitignored `.backstop/packs/`, the same way `node_modules` works for a JS project. See [backstop-ai](https://github.com/backstop-ai) for the published packs.

## License

MIT
