# Backstop

An AI agent discipline framework. Backstop gates the pipeline before code is written — enforcing standards, verifying claims, and tracing lineage from intent to implementation.

Backstop is not a code quality scanner. It is a framework that assumes AI agents will fabricate, skip steps, and ignore conventions unless mechanically constrained. It bakes in no language or tool knowledge of its own: every check comes from a **pack** you install explicitly, and backstop only runs what packs declare.

> 🚧 Under active development. Public, but early — expect rough edges and breaking changes.

## Documentation

- **[Getting Started](getting-started.md)** — install the CLI, initialize a project, add your first pack, and run the gate.
- **[Concepts](concepts.md)** — the gate, packs, engines, findings, and why backstop is a thin executor.
- **[CLI Reference](cli-reference.md)** — every command and flag.
- **[Pack Authoring](pack-authoring.md)** — write, validate, test, and publish your own checks.
- **[Artifact Workflow](artifact-workflow.md)** — bundles, specs, plans, and issues, and how lineage is traced from intent to implementation.

## Source

Backstop is open source under the MIT license. The core CLI lives at
[backstop-ai/backstop-core](https://github.com/backstop-ai/backstop-core); published packs live
alongside it under the [backstop-ai](https://github.com/backstop-ai) organization.
