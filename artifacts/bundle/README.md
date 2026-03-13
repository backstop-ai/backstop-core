# Context Bundle

A portable knowledge packet that captures decisions, patterns, and constraints for a specific domain. Bundles are versioned and travel with the codebase.

## Schema

- `schema.json` — JSON Schema for bundle validation
- `template.md` — Markdown template for new bundles

## Versioning

Schemas are versioned in subdirectories (e.g., `v1/`). The root `schema.json` always points to the latest stable version.
