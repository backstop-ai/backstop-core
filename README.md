# backstop

An AI agent discipline framework. Backstop gates the pipeline before code is written — enforcing standards, verifying claims, and tracing lineage from intent to implementation.

Backstop is not a code quality scanner. It is a framework that assumes AI agents will fabricate, skip steps, and ignore conventions unless mechanically constrained.

## Status

🚧 Under active development. Not yet ready for public use.

## Structure

```
cmd/backstop/    CLI entrypoint
internal/        Go packages (validation, schema loading, runtime integration)
artifacts/       Primitive schemas, templates, and docs (versioned)
standards/       Self-contained enforcement packs (rules, checkers, testdata)
recipes/         Implementation patterns organized by language
scripts/         Repo-level gate scripts
docs/            User documentation
```

## License

MIT
