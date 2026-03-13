# Recipe

A reusable implementation pattern for a specific language and domain. Recipes encode proven approaches to common engineering problems and are organized by language.

## Per-Language Structure

Each language under `recipes/<language>/` follows:

```
recipes/
  go/
    recipe-config.yml     # language-level config (toolchain, test runner, etc.)
    <recipe-name>/        # individual recipe
      recipe.yml          # recipe metadata (follows this schema)
      ...                 # recipe-specific files
  typescript/
    recipe-config.yml
    <recipe-name>/
      recipe.yml
      ...
```

## Schema

- `schema.json` — JSON Schema for `recipe.yml` validation

## Versioning

Schemas are versioned in subdirectories (e.g., `v1/`). The root `schema.json` always points to the latest stable version.
