# Contributing to backstop

## Development

```bash
# Build the CLI
make build

# Run tests
make test

# Lint
make lint
```

## Standards Packs

Each pack under `standards/` is self-contained and tests itself:

- Checkers are written in the pack's own language (Go pack → Go checkers, TS pack → TS checkers)
- OPA policies live in `policies/`
- Test fixtures live in `testdata/valid/` and `testdata/invalid/`

## Artifacts

Each primitive under `artifacts/` has:

- `schema.json` — JSON Schema (source of truth)
- `README.md` — Human documentation
- `template.md` — Authoring template (where applicable)

Schemas are versioned in subdirectories. The root `schema.json` always resolves to latest stable.
